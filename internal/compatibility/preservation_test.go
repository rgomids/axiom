package compatibility

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

// preservationRoots is the POC fixture (projects, state and the POC skill
// root) with an Axiom-owned archive namespace beside it.
func preservationRoots(t *testing.T) (Roots, string) {
	t.Helper()
	roots := pocRoots(t)
	return roots, filepath.Join(filepath.Dir(roots.State), "archive")
}

func planFor(t *testing.T, roots Roots, archive, resumeArchive, resumeManifest string) TransitionPlan {
	t.Helper()
	plan, err := PlanPOCTransition(context.Background(), roots, inspect(t, roots), archive, resumeArchive, resumeManifest)
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	return plan
}

func preserveAll(t *testing.T, roots Roots, plan TransitionPlan) {
	t.Helper()
	for _, object := range plan.Copy {
		if err := PreserveObject(context.Background(), roots, plan, object); err != nil {
			t.Fatalf("preserve %s: %v", object.Relative, err)
		}
	}
}

func TestPreservationPolicyCoversStateAndProjectsOnly(t *testing.T) {
	roots, archive := preservationRoots(t)
	plan := planFor(t, roots, archive, "", "")
	categories := map[string]int{}
	for _, object := range plan.Preserve {
		categories[object.Category]++
	}
	if categories[CategoryState] != 3 || categories[CategoryProjects] != 1 || categories[CategorySkills] != 0 || len(plan.Retire) != 2 {
		t.Fatalf("preserve=%v retire=%v", plan.Preserve, plan.Retire)
	}
	before := treeHash(t, filepath.Dir(roots.State))
	if _, err := os.Lstat(archive); !os.IsNotExist(err) {
		t.Fatal("planning created the archive")
	}
	if after := treeHash(t, filepath.Dir(roots.State)); after != before {
		t.Fatal("planning changed state")
	}
}

// ENOSPC and EDQUOT during a copy fail the preservation with the source
// untouched and the partial archive non-canonical; a later retry completes
// without duplicating confirmed copies.
func TestPreservationSpaceErrorsKeepSourceAndResume(t *testing.T) {
	for name, fault := range map[string]error{"ENOSPC": syscall.ENOSPC, "EDQUOT": edquot} {
		t.Run(name, func(t *testing.T) {
			if fault == nil {
				t.Skip("unverified: EDQUOT is not defined on this platform")
			}
			roots, archive := preservationRoots(t)
			plan := planFor(t, roots, archive, "", "")
			source := treeHash(t, roots.State)
			writes := 0
			beforePreservationWrite = func(Object) error {
				writes++
				if writes == 2 {
					return fault
				}
				return nil
			}
			t.Cleanup(func() { beforePreservationWrite = nil })
			var err error
			for _, object := range plan.Copy {
				if err = PreserveObject(context.Background(), roots, plan, object); err != nil {
					break
				}
			}
			if !errors.Is(err, fault) {
				t.Fatalf("copy error = %v", err)
			}
			if treeHash(t, roots.State) != source {
				t.Fatal("source changed by a failed preservation")
			}
			if _, err := os.Lstat(filepath.Join(plan.ArchivePath(), preservationManifest)); !os.IsNotExist(err) {
				t.Fatal("manifest written after a failed copy")
			}
			beforePreservationWrite = nil
			resumed := planFor(t, roots, archive, plan.Archive, "")
			if len(resumed.Copy) != len(plan.Copy)-1 {
				t.Fatalf("resume copies %d of %d", len(resumed.Copy), len(plan.Copy))
			}
			preserveAll(t, roots, resumed)
			if err := CompletePreservation(context.Background(), roots, resumed); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestRetirementIsConfinedToHistoricalStateObjects(t *testing.T) {
	roots, archive := preservationRoots(t)
	plan := planFor(t, roots, archive, "", "")
	preserveAll(t, roots, plan)
	if err := CompletePreservation(context.Background(), roots, plan); err != nil {
		t.Fatal(err)
	}
	escape := plan.Retire[0]
	for name, object := range map[string]Object{
		"traversal":         {Category: CategoryState, Relative: "../projects/poc-fixture/axiom.yaml", Kind: escape.Kind, Digest: escape.Digest, Bytes: escape.Bytes},
		"absolute":          {Category: CategoryState, Relative: filepath.Join(roots.State, escape.Relative), Kind: escape.Kind, Digest: escape.Digest, Bytes: escape.Bytes},
		"top-level":         {Category: CategoryState, Relative: "main-7.json", Kind: escape.Kind, Digest: escape.Digest, Bytes: escape.Bytes},
		"kept kind":         {Category: CategoryState, Relative: escape.Relative, Kind: "installation_record", Digest: escape.Digest, Bytes: escape.Bytes},
		"portable category": {Category: CategoryProjects, Relative: escape.Relative, Kind: escape.Kind, Digest: escape.Digest, Bytes: escape.Bytes},
		"changed digest":    {Category: CategoryState, Relative: escape.Relative, Kind: escape.Kind, Digest: plan.Retire[1].Digest, Bytes: escape.Bytes},
	} {
		t.Run(name, func(t *testing.T) {
			before := treeHash(t, filepath.Dir(roots.State))
			if err := RetireObject(context.Background(), roots, object); err == nil {
				t.Fatal("unsafe retirement accepted")
			}
			if after := treeHash(t, filepath.Dir(roots.State)); after != before {
				t.Fatal("refused retirement changed state")
			}
		})
	}
	for _, object := range plan.Retire {
		if err := RetireObject(context.Background(), roots, object); err != nil {
			t.Fatal(err)
		}
	}
	if report := inspect(t, roots); report.Classification != ValidV1 {
		t.Fatalf("rebuilt root = %s", report.Classification)
	}
}

// After the first retirement only the manifest recorded by the operation is
// accepted: replacing it (even with one whose copies verify) is a conflict.
func TestResumeAfterRetirementRequiresTheRecordedManifest(t *testing.T) {
	roots, archive := preservationRoots(t)
	plan := planFor(t, roots, archive, "", "")
	preserveAll(t, roots, plan)
	if err := CompletePreservation(context.Background(), roots, plan); err != nil {
		t.Fatal(err)
	}
	if err := RetireObject(context.Background(), roots, plan.Retire[0]); err != nil {
		t.Fatal(err)
	}
	if _, err := PlanPOCTransition(context.Background(), roots, inspect(t, roots), archive, plan.Archive, ""); !errors.Is(err, ErrPreservationConflict) {
		t.Fatalf("resume without recorded manifest: %v", err)
	}
	resumed := planFor(t, roots, archive, plan.Archive, plan.ManifestDigest())
	if len(resumed.Retire) != 1 || len(resumed.Copy) != 0 || !resumed.ManifestPresent {
		t.Fatalf("resumed plan = %+v", resumed)
	}
	manifest := filepath.Join(plan.ArchivePath(), preservationManifest)
	writePrivate(t, manifest, append([]byte(read(t, manifest)), '\n'))
	if _, err := PlanPOCTransition(context.Background(), roots, inspect(t, roots), archive, plan.Archive, plan.ManifestDigest()); !errors.Is(err, ErrPreservationConflict) {
		t.Fatalf("replaced manifest: %v", err)
	}
}
