package install

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/compatibility"
	"github.com/rgomids/axiom/internal/testfs"
)

// Issue #153 / I153-T02: the RecognizedPOC transition preserve -> clean
// rebuild -> supported reconfiguration, orchestrated by the owned upgrade.

// pocInstallation is an owned 1.0.0 installation over the historical POC
// state, with the Axiom-owned archive namespace beside its receipts.
func pocInstallation(t *testing.T) installation {
	t.Helper()
	installed := install(t, newBundle("1.0.0", []byte("old-binary\n")))
	installed.withPOCState(t)
	installed.target.Archive = filepath.Join(filepath.Dir(installed.target.ReceiptDir), "archive")
	return installed
}

func (i installation) archivePath(t *testing.T) string {
	t.Helper()
	entries, err := os.ReadDir(i.target.Archive)
	if err != nil || len(entries) != 1 || !compatibility.ValidArchiveName(entries[0].Name()) {
		t.Fatalf("archive namespace entries=%v err=%v", entries, err)
	}
	return filepath.Join(i.target.Archive, entries[0].Name())
}

// fixtureInventory independently lists the POC fixture's state and portable
// objects with category, relative path, digest and bytes, without using the
// compatibility package.
func fixtureInventory(t *testing.T) map[string]string {
	t.Helper()
	inventory := map[string]string{}
	for _, category := range []string{"state", "projects"} {
		root := filepath.Join(pocFixture, category)
		err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
			if err != nil || entry.IsDir() {
				return err
			}
			wire, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			relative, _ := filepath.Rel(root, path)
			sum := sha256.Sum256(wire)
			inventory[category+"/"+filepath.ToSlash(relative)] = hex.EncodeToString(sum[:]) + ":" + itoa(len(wire))
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	return inventory
}

func itoa(value int) string { return strconv.Itoa(value) }

func readManifest(t *testing.T, archive string) compatibility.PreservationManifest {
	t.Helper()
	var document compatibility.PreservationManifest
	if err := json.Unmarshal([]byte(read(t, filepath.Join(archive, "manifest.json"))), &document); err != nil {
		t.Fatal(err)
	}
	return document
}

func applyPreview(t *testing.T, service Service, preview Preview) (Result, error) {
	t.Helper()
	authority, err := Authorize(preview, preview.Digest)
	if err != nil {
		t.Fatalf("authorize: %v (%+v)", err, preview)
	}
	return service.Apply(context.Background(), preview, authority)
}

func effectKinds(effects []Effect) []string {
	kinds := []string{}
	for _, effect := range effects {
		kinds = append(kinds, effect.Kind)
	}
	return kinds
}

func countKind(effects []Effect, kind string) int {
	count := 0
	for _, effect := range effects {
		if effect.Kind == kind {
			count++
		}
	}
	return count
}

// assertRebuilt proves the rebuilt active state: only validated Project
// intent and associations stay, byte-identical; no POC workflow or Work Item
// material is active; the root resolves direct as canonical v1.
func assertRebuilt(t *testing.T, installed installation) {
	t.Helper()
	active := []string{}
	_ = filepath.WalkDir(installed.target.State.State, func(path string, entry fs.DirEntry, err error) error {
		if err == nil && !entry.IsDir() {
			relative, _ := filepath.Rel(installed.target.State.State, path)
			active = append(active, filepath.ToSlash(relative))
		}
		return nil
	})
	if want := []string{"projects/" + pocProjectID + "/installation.json"}; strings.Join(active, ",") != strings.Join(want, ",") {
		t.Fatalf("active state = %v, want %v", active, want)
	}
	for active, fixture := range map[string]string{
		filepath.Join(installed.target.State.State, "projects", pocProjectID, "installation.json"): "state/projects/" + pocProjectID + "/installation.json",
		filepath.Join(installed.target.State.Projects, "poc-fixture", "axiom.yaml"):                "projects/poc-fixture/axiom.yaml",
	} {
		if read(t, active) != read(t, filepath.Join(pocFixture, fixture)) {
			t.Fatalf("kept %s changed", fixture)
		}
	}
	for _, retired := range []string{"workflows", "work-items"} {
		if _, err := os.Lstat(filepath.Join(installed.target.State.State, retired)); !os.IsNotExist(err) {
			t.Fatalf("historical %s still active: %v", retired, err)
		}
	}
	report, err := compatibility.Inspect(context.Background(), installed.target.State)
	if err != nil || report.Classification != compatibility.ValidV1 || compatibility.Resolve(compatibility.Owned, report).Strategy != compatibility.StrategyDirect {
		t.Fatalf("rebuilt state = %+v err=%v", report, err)
	}
}

// assertPreserved independently compares the archive's final manifest with
// the fixture inventory and every archived object's bytes.
func assertPreserved(t *testing.T, installed installation) {
	t.Helper()
	archive := installed.archivePath(t)
	document := readManifest(t, archive)
	expected := fixtureInventory(t)
	observed := map[string]string{}
	for _, object := range document.Objects {
		observed[object.Category+"/"+object.Relative] = object.Digest + ":" + itoa(int(object.Bytes))
		wire, err := os.ReadFile(filepath.Join(archive, "objects", object.Digest))
		sum := sha256.Sum256(wire)
		if err != nil || hex.EncodeToString(sum[:]) != object.Digest || int64(len(wire)) != object.Bytes {
			t.Fatalf("archived %s does not verify: %v", object.Relative, err)
		}
	}
	if len(observed) != len(expected) {
		t.Fatalf("manifest objects %v, inventory %v", observed, expected)
	}
	for key, value := range expected {
		if observed[key] != value {
			t.Fatalf("manifest %s = %q, inventory %q", key, observed[key], value)
		}
	}
	if document.Policy != compatibility.PreservationPolicy || document.POCTag != compatibility.HistoricalPOCTag || len(document.Retired) != 2 || len(document.Kept) != 2 {
		t.Fatalf("manifest = %+v", document)
	}
	// Machine-local, owner-only, and never a source of portable content.
	_ = filepath.WalkDir(archive, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			t.Fatal(err)
		}
		if entry.IsDir() {
			assertMode(t, path, 0o700)
		} else {
			assertMode(t, path, 0o600)
		}
		return nil
	})
	if strings.Contains(read(t, filepath.Join(archive, "manifest.json")), string(filepath.Separator)+"axiom-poc-fixture") {
		t.Fatal("manifest carries record content")
	}
}

func TestRecognizedPOCUpgradePreservesRebuildsAndInstalls(t *testing.T) {
	installed := pocInstallation(t)
	candidate := installed.candidate(t, newBundle("1.1.0", []byte("new-binary\n")))
	preview, err := NewService().Preview(context.Background(), installed.target, candidate)
	if err != nil {
		t.Fatalf("preview=%+v err=%v", preview, err)
	}
	kinds := effectKinds(preview.Effects)
	if preview.Transition.Strategy != compatibility.StrategyPreserveRebuildReconfigure || countKind(preview.Effects, "preserve") != 4 || countKind(preview.Effects, "preservation_manifest") != 1 || countKind(preview.Effects, "retire") != 2 || kinds[len(kinds)-2] != "binary" || kinds[len(kinds)-1] != "receipt" {
		t.Fatalf("effects=%v transition=%+v", kinds, preview.Transition)
	}
	retire := []Effect{}
	for _, effect := range preview.Effects {
		if effect.Kind == "retire" {
			retire = append(retire, effect)
		}
	}
	if !strings.Contains(retire[0].Name, "work-items/") || !strings.Contains(retire[1].Name, "workflows/") {
		t.Fatalf("retirement order = %+v", retire)
	}
	result, err := applyPreview(t, NewService(), preview)
	if err != nil || result.Status != "success" || result.Preservation == "" || len(result.Ledger) != len(preview.Effects) {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	assertPreserved(t, installed)
	assertRebuilt(t, installed)
	if read(t, filepath.Join(installed.target.BinaryDir, binaryName)) != "new-binary\n" {
		t.Fatal("binary not upgraded")
	}
	if _, err := os.Lstat(filepath.Join(installed.target.ReceiptDir, markerName)); !os.IsNotExist(err) {
		t.Fatal("marker left behind")
	}
	// An equivalent retry is a no-op and never touches the archive.
	before := snapshot(t, filepath.Dir(installed.target.BinaryDir))
	again, err := NewService().Preview(context.Background(), installed.target, candidate)
	if err != nil || len(again.Effects) != 0 || again.Transition.Strategy != compatibility.StrategyDirect {
		t.Fatalf("retry preview=%+v err=%v", again, err)
	}
	if _, err := Authorize(again, again.Digest); err == nil {
		t.Fatal("no-op retry authorized a mutation")
	}
	if after := snapshot(t, filepath.Dir(installed.target.BinaryDir)); after != before {
		t.Fatal("retry changed installation, state or archive")
	}
}

// The same transition runs for a same-release reinstall over POC state.
func TestRecognizedPOCTransitionRunsWithoutBinaryChange(t *testing.T) {
	installed := pocInstallation(t)
	candidate := installed.candidate(t, installed.current)
	preview, err := NewService().Preview(context.Background(), installed.target, candidate)
	if err != nil || countKind(preview.Effects, "binary") != 0 || countKind(preview.Effects, "retire") != 2 {
		t.Fatalf("preview=%+v err=%v", preview, err)
	}
	if result, err := applyPreview(t, NewService(), preview); err != nil || result.Status != "success" {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	assertRebuilt(t, installed)
}

// Interruption at every transition boundary is recoverable under fresh exact
// authority, never duplicates preservation, and never reports success early.
func TestRecognizedPOCInterruptionAtEachBoundaryResumes(t *testing.T) {
	boundaries := []struct {
		name          string
		stop          func(string) bool
		sourceIntact  bool
		resumeEffects map[string]int
	}{
		{"during copy", func(label string) bool { return strings.HasPrefix(label, "preserve:") }, true, map[string]int{"preserve": 3, "preservation_manifest": 1, "retire": 2, "binary": 1, "receipt": 1}},
		{"after preservation before activation", func(label string) bool { return label == "preservation_manifest" }, true, map[string]int{"preserve": 0, "preservation_manifest": 0, "retire": 2, "binary": 1, "receipt": 1}},
		{"during retirement", func(label string) bool { return strings.HasPrefix(label, "retire:state/work-items/") }, true, map[string]int{"preserve": 0, "retire": 1, "binary": 1, "receipt": 1}},
		{"after activation before install", func(label string) bool { return strings.HasPrefix(label, "retire:state/workflows/") }, false, map[string]int{"preserve": 0, "retire": 0, "binary": 1, "receipt": 1}},
		{"after binary", func(label string) bool { return label == "binary" }, false, map[string]int{"preserve": 0, "retire": 0, "binary": 0, "receipt": 1}},
	}
	for _, boundary := range boundaries {
		t.Run(boundary.name, func(t *testing.T) {
			installed := pocInstallation(t)
			candidate := installed.candidate(t, newBundle("1.1.0", []byte("new-binary\n")))
			preview, err := NewService().Preview(context.Background(), installed.target, candidate)
			if err != nil {
				t.Fatal(err)
			}
			stopped := false
			interrupted := Service{afterEffect: func(label string) error {
				if !stopped && boundary.stop(label) {
					stopped = true
					return errors.New("injected interruption")
				}
				return nil
			}}
			result, err := applyPreview(t, interrupted, preview)
			if err == nil || result.Status != "partial" {
				t.Fatalf("interrupted result=%+v err=%v", result, err)
			}
			if _, err := os.Stat(installed.pocWorkflow()); (err == nil) != boundary.sourceIntact {
				t.Fatalf("source workflow present=%v, want %v", err == nil, boundary.sourceIntact)
			}
			if !strings.Contains(read(t, filepath.Join(installed.target.ReceiptDir, markerName)), "transitionArchive=recognized-poc-") {
				t.Fatal("marker does not bind the transition archive")
			}
			resume, err := NewService().Preview(context.Background(), installed.target, candidate)
			if err != nil || !resume.Resume {
				t.Fatalf("resume preview=%+v err=%v", resume, err)
			}
			for kind, want := range boundary.resumeEffects {
				if got := countKind(resume.Effects, kind); got != want {
					t.Fatalf("resume %s effects = %d, want %d (%v)", kind, got, want, effectKinds(resume.Effects))
				}
			}
			if result, err := applyPreview(t, NewService(), resume); err != nil || result.Status != "success" {
				t.Fatalf("resumed result=%+v err=%v", result, err)
			}
			assertPreserved(t, installed)
			assertRebuilt(t, installed)
			if again, err := NewService().Preview(context.Background(), installed.target, candidate); err != nil || len(again.Effects) != 0 || again.Resume {
				t.Fatalf("after resume preview=%+v err=%v", again, err)
			}
		})
	}
}

// An interrupted stage inside the archive is recovered only under the
// operation marker; without it the occupied archive is refused untouched.
func TestRecognizedPOCArchiveStageLeftovers(t *testing.T) {
	installed := pocInstallation(t)
	candidate := installed.candidate(t, newBundle("1.1.0", []byte("new-binary\n")))
	preview, _ := NewService().Preview(context.Background(), installed.target, candidate)
	stopped := false
	interrupted := Service{afterEffect: func(label string) error {
		if !stopped && strings.HasPrefix(label, "preserve:") {
			stopped = true
			writeFile(t, filepath.Join(installed.archivePath(t), "objects", ".axiom-archive-stage-crashed"), []byte("partial"), 0o600)
			return errors.New("injected interruption")
		}
		return nil
	}}
	if _, err := applyPreview(t, interrupted, preview); err == nil {
		t.Fatal("interruption not reported")
	}
	resume, err := NewService().Preview(context.Background(), installed.target, candidate)
	if err != nil || len(resume.Leftovers) != 1 || !strings.HasSuffix(resume.Leftovers[0], ".axiom-archive-stage-crashed") {
		t.Fatalf("resume=%+v err=%v", resume, err)
	}
	if result, err := applyPreview(t, NewService(), resume); err != nil || result.Status != "success" {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	assertPreserved(t, installed)

	fresh := pocInstallation(t)
	occupied := filepath.Join(privateDirectory(t, privateDirectory(t, filepath.Dir(fresh.target.Archive), "archive"), preview.Preservation[len(filepath.Dir(preview.Preservation))+1:]), "objects")
	writeFile(t, filepath.Join(privateDirectory(t, filepath.Dir(occupied), "objects"), ".axiom-archive-stage-unknown"), []byte("x"), 0o600)
	before := snapshot(t, filepath.Dir(fresh.target.BinaryDir))
	if _, err := NewService().Preview(context.Background(), fresh.target, fresh.candidate(t, newBundle("1.1.0", []byte("new-binary\n")))); category(err) != "preservation_conflict" {
		t.Fatalf("unmarked stage leftover: %v", err)
	}
	if after := snapshot(t, filepath.Dir(fresh.target.BinaryDir)); after != before {
		t.Fatal("refusal changed state")
	}
}

// After the final manifest, missing, unexpected, changed and unverifiable
// objects each block retirement, including when every copied object's digest
// still passes; the source stays intact and active.
func TestRecognizedPOCCorrespondenceBlocksRetirement(t *testing.T) {
	cases := map[string]func(t *testing.T, archive string){
		"missing archived object": func(t *testing.T, archive string) {
			document := readManifest(t, archive)
			if err := os.Remove(filepath.Join(archive, "objects", document.Objects[0].Digest)); err != nil {
				t.Fatal(err)
			}
		},
		"unexpected archived object": func(t *testing.T, archive string) {
			writeFile(t, filepath.Join(archive, "objects", strings.Repeat("a", 64)), []byte("extra\n"), 0o600)
		},
		"changed archived object": func(t *testing.T, archive string) {
			document := readManifest(t, archive)
			path := filepath.Join(archive, "objects", document.Objects[0].Digest)
			writeFile(t, path, append([]byte(read(t, path)), ' '), 0o600)
		},
		"unverifiable archived object": func(t *testing.T, archive string) {
			document := readManifest(t, archive)
			if err := testfs.SharedMode(filepath.Join(archive, "objects", document.Objects[0].Digest), 0o644); err != nil {
				t.Fatal(err)
			}
		},
		"manifest misses an inventory object although copies pass": func(t *testing.T, archive string) {
			document := readManifest(t, archive)
			removed := document.Objects[0]
			document.Objects = document.Objects[1:]
			rewriteManifest(t, archive, document)
			if err := os.Remove(filepath.Join(archive, "objects", removed.Digest)); err != nil {
				t.Fatal(err)
			}
		},
		"manifest lists an object the inventory lacks": func(t *testing.T, archive string) {
			document := readManifest(t, archive)
			extra := document.Objects[0]
			extra.Relative = "projects/" + pocProjectID + "/other.json"
			document.Objects = append(document.Objects, extra)
			rewriteManifest(t, archive, document)
		},
		"manifest changes an object's bytes": func(t *testing.T, archive string) {
			document := readManifest(t, archive)
			document.Objects[0].Bytes++
			rewriteManifest(t, archive, document)
		},
	}
	for name, tamper := range cases {
		t.Run(name, func(t *testing.T) {
			if strings.Contains(name, "unverifiable") {
				testfs.POSIXModes(t)
			}
			installed := pocInstallation(t)
			candidate := installed.candidate(t, newBundle("1.1.0", []byte("new-binary\n")))
			preview, _ := NewService().Preview(context.Background(), installed.target, candidate)
			interrupted := Service{afterEffect: func(label string) error {
				if label == "preservation_manifest" {
					return errors.New("injected interruption")
				}
				return nil
			}}
			if _, err := applyPreview(t, interrupted, preview); err == nil {
				t.Fatal("interruption not reported")
			}
			tamper(t, installed.archivePath(t))
			before := snapshot(t, installed.target.State.State)
			resume, err := NewService().Preview(context.Background(), installed.target, candidate)
			if err == nil {
				// A plan that still forms must fail its correspondence proof
				// before the first retirement.
				result, applyErr := applyPreview(t, NewService(), resume)
				if applyErr == nil || countConfirmed(result, "retire") != 0 {
					t.Fatalf("tampered archive authorized retirement: %+v %v", result, applyErr)
				}
			} else if c := category(err); c != "preservation_conflict" && c != "state_changed" {
				t.Fatalf("resume error=%v", err)
			}
			if after := snapshot(t, installed.target.State.State); after != before {
				t.Fatal("legacy state retired or changed without complete correspondence")
			}
			if _, err := os.Stat(installed.pocWorkflow()); err != nil {
				t.Fatal("POC workflow retired")
			}
		})
	}
}

func rewriteManifest(t *testing.T, archive string, document compatibility.PreservationManifest) {
	t.Helper()
	wire, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(archive, "manifest.json"), append(wire, '\n'), 0o600)
}

func countConfirmed(result Result, kind string) int {
	count := 0
	for _, entry := range result.Ledger {
		if entry.Kind == kind && entry.Confirmed {
			count++
		}
	}
	return count
}

// Source drift after preservation (a new or changed active object not in
// the manifest) blocks retirement; stale authority and target drift before
// apply are refused with zero effects.
func TestRecognizedPOCDriftAndStaleAuthority(t *testing.T) {
	t.Run("new source object after manifest", func(t *testing.T) {
		installed := pocInstallation(t)
		candidate := installed.candidate(t, newBundle("1.1.0", []byte("new-binary\n")))
		preview, _ := NewService().Preview(context.Background(), installed.target, candidate)
		interrupted := Service{afterEffect: func(label string) error {
			if label == "preservation_manifest" {
				return errors.New("injected interruption")
			}
			return nil
		}}
		if _, err := applyPreview(t, interrupted, preview); err == nil {
			t.Fatal("interruption not reported")
		}
		wire := read(t, installed.pocWorkflow())
		writeFile(t, filepath.Join(filepath.Dir(installed.pocWorkflow()), "main-8.json"), []byte(strings.Replace(wire, `"workItem":7`, `"workItem":8`, 1)), 0o600)
		if _, err := NewService().Preview(context.Background(), installed.target, candidate); category(err) != "state_changed" && category(err) != "preservation_conflict" {
			t.Fatalf("drifted source: %v", err)
		}
		if _, err := os.Stat(installed.pocWorkflow()); err != nil {
			t.Fatal("source retired after drift")
		}
	})
	for name, drift := range map[string]func(t *testing.T, installed installation){
		"source leaf replaced": func(t *testing.T, installed installation) {
			replaceIn(t, installed.workItem(), `"state":"OPEN"`, `"state":"CLOSED"`)
		},
		"source ancestor replaced": func(t *testing.T, installed installation) {
			workflows := filepath.Join(installed.target.State.State, "workflows")
			if err := os.Rename(workflows, workflows+".old"); err != nil {
				t.Fatal(err)
			}
			if err := os.Mkdir(workflows, 0o700); err != nil {
				t.Fatal(err)
			}
		},
	} {
		t.Run(name, func(t *testing.T) {
			installed := pocInstallation(t)
			candidate := installed.candidate(t, newBundle("1.1.0", []byte("new-binary\n")))
			preview, err := NewService().Preview(context.Background(), installed.target, candidate)
			if err != nil {
				t.Fatal(err)
			}
			authority, _ := Authorize(preview, preview.Digest)
			drift(t, installed)
			before := snapshot(t, filepath.Dir(installed.target.BinaryDir))
			if result, err := NewService().Apply(context.Background(), preview, authority); category(err) != "authority_denied" || len(result.Ledger) != 0 {
				t.Fatalf("result=%+v err=%v", result, err)
			}
			if after := snapshot(t, filepath.Dir(installed.target.BinaryDir)); after != before {
				t.Fatal("stale authority changed something")
			}
		})
	}
}

// Unsafe, overlapping, occupied or foreign archive destinations are refused
// before any effect; foreign content in the source fails closed.
func TestRecognizedPOCUnsafeDestinationsAndSourcesHaveZeroEffects(t *testing.T) {
	cases := map[string]struct {
		setup func(t *testing.T, installed *installation)
		want  string
	}{
		"archive inside the state root": {func(t *testing.T, installed *installation) {
			installed.target.Archive = filepath.Join(installed.target.State.State, "archive")
		}, "preservation_target_unsafe"},
		"archive containing the projects root": {func(t *testing.T, installed *installation) {
			installed.target.Archive = filepath.Dir(installed.target.State.Projects)
		}, "preservation_target_unsafe"},
		"relative archive": {func(t *testing.T, installed *installation) {
			installed.target.Archive = "archive"
		}, "preservation_target_unsafe"},
		"symlinked archive root": {func(t *testing.T, installed *installation) {
			if err := testfs.Symlink(t, t.TempDir(), installed.target.Archive); err != nil {
				t.Skip(err)
			}
		}, "preservation_target_unsafe"},
		"permissive archive root": {func(t *testing.T, installed *installation) {
			testfs.POSIXModes(t)
			if err := os.Mkdir(installed.target.Archive, 0o700); err != nil {
				t.Fatal(err)
			}
			if err := testfs.SharedMode(installed.target.Archive, 0o777); err != nil {
				t.Fatal(err)
			}
		}, "preservation_target_unsafe"},
		"occupied archive with foreign content": {func(t *testing.T, installed *installation) {
			report, err := compatibility.Inspect(context.Background(), installed.target.State)
			if err != nil {
				t.Fatal(err)
			}
			archive := privateDirectory(t, privateDirectory(t, filepath.Dir(installed.target.Archive), "archive"), compatibility.ArchiveName(report.Digest))
			writeFile(t, filepath.Join(archive, "notes.txt"), []byte("foreign\n"), 0o600)
		}, "preservation_conflict"},
		"archived object of another source": {func(t *testing.T, installed *installation) {
			report, err := compatibility.Inspect(context.Background(), installed.target.State)
			if err != nil {
				t.Fatal(err)
			}
			archive := privateDirectory(t, privateDirectory(t, filepath.Dir(installed.target.Archive), "archive"), compatibility.ArchiveName(report.Digest))
			writeFile(t, filepath.Join(privateDirectory(t, archive, "objects"), strings.Repeat("b", 64)), []byte("x"), 0o600)
		}, "preservation_conflict"},
		"foreign entry in the POC root": {func(t *testing.T, installed *installation) {
			writeFile(t, filepath.Join(installed.target.State.State, "notes.txt"), []byte("foreign\n"), 0o600)
		}, "state_unsafe"},
		"hard-linked POC record": {func(t *testing.T, installed *installation) {
			if err := os.Link(installed.pocWorkflow(), filepath.Join(t.TempDir(), "link")); err != nil {
				t.Skip(err)
			}
		}, "state_unsafe"},
		"symlinked POC record": {func(t *testing.T, installed *installation) {
			if err := os.Remove(installed.workItem()); err != nil {
				t.Fatal(err)
			}
			if err := testfs.Symlink(t, installed.pocWorkflow(), installed.workItem()); err != nil {
				t.Skip(err)
			}
		}, "state_unsafe"},
		"no archive namespace": {func(t *testing.T, installed *installation) {
			installed.target.Archive = ""
		}, "state_transition_unavailable"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			installed := pocInstallation(t)
			tc.setup(t, &installed)
			before := snapshot(t, filepath.Dir(installed.target.BinaryDir))
			preview, err := NewService().Preview(context.Background(), installed.target, installed.candidate(t, newBundle("1.1.0", []byte("new-binary\n"))))
			if category(err) != tc.want || preview.Digest != "" {
				t.Fatalf("preview=%+v err=%v", preview, err)
			}
			if after := snapshot(t, filepath.Dir(installed.target.BinaryDir)); after != before {
				t.Fatal("refusal changed installation, state or archive")
			}
		})
	}
}

// Free space for the archive is checked before any effect.
func TestRecognizedPOCInsufficientArchiveSpaceHasZeroEffects(t *testing.T) {
	installed := pocInstallation(t)
	full := Service{availableSpace: func(path string) (uint64, error) {
		if path == installed.target.BinaryDir {
			return 1 << 40, nil
		}
		return 10, nil
	}}
	before := snapshot(t, filepath.Dir(installed.target.BinaryDir))
	if _, err := full.Preview(context.Background(), installed.target, installed.candidate(t, newBundle("1.1.0", []byte("new-binary\n")))); category(err) != "insufficient_space" {
		t.Fatalf("space error=%v", err)
	}
	if after := snapshot(t, filepath.Dir(installed.target.BinaryDir)); after != before {
		t.Fatal("refusal changed something")
	}
}

// Cross-filesystem preservation uses the same copy -> verify -> manifest ->
// correspondence protocol. It runs when a second writable filesystem is
// available (Linux /dev/shm); otherwise the case is reported unverified.
func TestRecognizedPOCCrossFilesystemArchive(t *testing.T) {
	other := os.Getenv("AXIOM_TEST_OTHER_FILESYSTEM")
	if other == "" {
		if info, err := os.Stat("/dev/shm"); err == nil && info.IsDir() {
			other = "/dev/shm"
		}
	}
	if other == "" || sameDevice(t, other, t.TempDir()) {
		t.Skip("unverified: no second writable filesystem (set AXIOM_TEST_OTHER_FILESYSTEM)")
	}
	base, err := os.MkdirTemp(other, "axiom-archive-")
	if err != nil {
		t.Skip("unverified: ", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(base) })
	if err := os.Chmod(base, 0o700); err != nil {
		t.Fatal(err)
	}
	installed := pocInstallation(t)
	installed.target.Archive = filepath.Join(base, "archive")
	preview, err := NewService().Preview(context.Background(), installed.target, installed.candidate(t, newBundle("1.1.0", []byte("new-binary\n"))))
	if err != nil {
		t.Fatalf("preview err=%v", err)
	}
	if result, err := applyPreview(t, NewService(), preview); err != nil || result.Status != "success" {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	assertPreserved(t, installed)
	assertRebuilt(t, installed)
	t.Logf("cross-filesystem archive verified on %s", other)
}

// After activation the rebuilt root is ordinary v1 state: if the upgrade is
// interrupted before the binary changes, the still-installed release may
// write v1 state, and the resume only re-proves the recorded archive itself.
func TestRecognizedPOCResumeAfterActivationVerifiesOnlyTheArchive(t *testing.T) {
	for name, between := range map[string]func(t *testing.T, installed installation){
		"v1 state written meanwhile": func(t *testing.T, installed installation) {
			createV1Artifact(t, installed.target.State.State)
		},
		"archive tampered meanwhile": func(t *testing.T, installed installation) {
			archive := installed.archivePath(t)
			document := readManifest(t, archive)
			path := filepath.Join(archive, "objects", document.Objects[0].Digest)
			writeFile(t, path, append([]byte(read(t, path)), ' '), 0o600)
		},
	} {
		t.Run(name, func(t *testing.T) {
			installed := pocInstallation(t)
			candidate := installed.candidate(t, newBundle("1.1.0", []byte("new-binary\n")))
			preview, _ := NewService().Preview(context.Background(), installed.target, candidate)
			interrupted := Service{afterEffect: func(label string) error {
				if strings.HasPrefix(label, "retire:state/workflows/") {
					return errors.New("injected interruption")
				}
				return nil
			}}
			if _, err := applyPreview(t, interrupted, preview); err == nil {
				t.Fatal("interruption not reported")
			}
			between(t, installed)
			resume, err := NewService().Preview(context.Background(), installed.target, candidate)
			if strings.Contains(name, "tampered") {
				if category(err) != "recovery_required" {
					t.Fatalf("tampered archive resume: %v", err)
				}
				return
			}
			if err != nil || resume.Preservation == "" || countKind(resume.Effects, "binary") != 1 {
				t.Fatalf("resume=%+v err=%v", resume, err)
			}
			if result, err := applyPreview(t, NewService(), resume); err != nil || result.Status != "success" || result.Preservation == "" {
				t.Fatalf("result=%+v err=%v", result, err)
			}
		})
	}
}

// Confirmed retirement is positive marker truth, not inferred from absence.
func TestRecognizedPOCResumeRequiresExactActiveInventory(t *testing.T) {
	for _, name := range []string{"portable absent", "installation absent", "unconfirmed absent", "confirmed absent", "new object", "modified object"} {
		t.Run(name, func(t *testing.T) {
			installed := pocInstallation(t)
			candidate := installed.candidate(t, newBundle("1.1.0", []byte("new-binary\n")))
			preview, err := NewService().Preview(context.Background(), installed.target, candidate)
			if err != nil {
				t.Fatal(err)
			}
			service := Service{afterEffect: func(label string) error {
				if strings.HasPrefix(label, "retire:state/work-items/") {
					return errors.New("stop after confirmed retirement")
				}
				return nil
			}}
			result, err := applyPreview(t, service, preview)
			if err == nil || result.Status != "partial" {
				t.Fatalf("result=%+v err=%v", result, err)
			}
			portable := filepath.Join(installed.target.State.Projects, "poc-fixture", "axiom.yaml")
			record := filepath.Join(installed.target.State.State, "projects", pocProjectID, "installation.json")
			switch name {
			case "portable absent":
				err = os.Remove(portable)
			case "installation absent":
				err = os.Remove(record)
			case "unconfirmed absent":
				err = os.Remove(installed.pocWorkflow())
			case "confirmed absent":
				for _, effect := range preview.Effects {
					if effect.Kind == "retire" && strings.HasPrefix(effect.Name, "state/work-items/") {
						if _, e := os.Stat(effect.Target); !os.IsNotExist(e) {
							t.Fatalf("confirmed object still present: %v", e)
						}
					}
				}
			case "new object":
				createV1Artifact(t, installed.target.State.State)
			case "modified object":
				writeFile(t, portable, append([]byte(read(t, portable)), '\n'), 0o600)
			}
			if err != nil && name != "confirmed absent" && name != "new object" && name != "modified object" {
				t.Fatal(err)
			}
			before := snapshot(t, filepath.Dir(installed.target.ReceiptDir))
			resume, resumeErr := NewService().Preview(context.Background(), installed.target, candidate)
			if name == "confirmed absent" {
				if resumeErr != nil {
					t.Fatal(resumeErr)
				}
				if result, err := applyPreview(t, NewService(), resume); err != nil || result.Status != "success" {
					t.Fatalf("result=%+v err=%v", result, err)
				}
			} else {
				if resumeErr == nil || resume.Digest != "" {
					t.Fatalf("unproved state accepted: %+v %v", resume, resumeErr)
				}
				if snapshot(t, filepath.Dir(installed.target.ReceiptDir)) != before {
					t.Fatal("refused resume mutated installation")
				}
			}
		})
	}
}

func TestRecognizedPOCResumeRejectsSemanticManifestTampering(t *testing.T) {
	mutations := map[string]func(*compatibility.PreservationManifest){
		"pocTag":      func(m *compatibility.PreservationManifest) { m.POCTag = "v0.1.0-poc.2" },
		"pocRevision": func(m *compatibility.PreservationManifest) { m.POCRevision = strings.Repeat("a", 40) },
		"retired":     func(m *compatibility.PreservationManifest) { m.Retired = []string{} },
		"kept":        func(m *compatibility.PreservationManifest) { m.Kept = []string{} },
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			installed := pocInstallation(t)
			candidate := installed.candidate(t, newBundle("1.1.0", []byte("new-binary\n")))
			preview, err := NewService().Preview(context.Background(), installed.target, candidate)
			if err != nil {
				t.Fatal(err)
			}
			interrupted := Service{afterEffect: func(label string) error {
				if label == "preservation_manifest" {
					return errors.New("stop before first retirement")
				}
				return nil
			}}
			if _, err := applyPreview(t, interrupted, preview); err == nil {
				t.Fatal("interruption missing")
			}
			archive := installed.archivePath(t)
			document := readManifest(t, archive)
			mutate(&document)
			rewriteManifest(t, archive, document)
			resume, err := NewService().Preview(context.Background(), installed.target, candidate)
			if err == nil || resume.Digest != "" {
				t.Fatalf("tamper accepted: %+v %v", resume, err)
			}
			for _, effect := range preview.Effects {
				if effect.Kind == "retire" {
					if _, err := os.Stat(effect.Target); err != nil {
						t.Fatalf("retired despite tampering: %v", err)
					}
				}
			}
		})
	}
}

func TestRecognizedPOCResumeRejectsAmbiguousMarkers(t *testing.T) {
	for _, name := range []string{"old transition format", "negative progress", "oversized progress", "missing progress", "unknown field", "duplicate empty field", "activation without manifest"} {
		t.Run(name, func(t *testing.T) {
			installed := pocInstallation(t)
			candidate := installed.candidate(t, newBundle("1.1.0", []byte("new-binary\n")))
			preview, err := NewService().Preview(context.Background(), installed.target, candidate)
			if err != nil {
				t.Fatal(err)
			}
			stop := Service{afterEffect: func(label string) error {
				if label == "preservation_manifest" {
					return errors.New("stop")
				}
				return nil
			}}
			if _, err := applyPreview(t, stop, preview); err == nil {
				t.Fatal("interruption missing")
			}
			path := filepath.Join(installed.target.ReceiptDir, markerName)
			wire := read(t, path)
			switch name {
			case "old transition format":
				wire = strings.Replace(wire, "formatVersion=2", "formatVersion=1", 1)
			case "negative progress":
				wire = strings.Replace(wire, "transitionRetired=0", "transitionRetired=-1", 1)
			case "oversized progress":
				wire = strings.Replace(wire, "transitionRetired=0", "transitionRetired=999999", 1)
			case "missing progress":
				wire = strings.Replace(wire, "transitionRetired=0\n", "", 1)
			case "unknown field":
				wire += "unknown=value\n"
			case "duplicate empty field":
				wire += "transitionManifest=\ntransitionManifest=\n"
			case "activation without manifest":
				wire = strings.Replace(wire, "transitionActivated=false", "transitionActivated=true", 1)
			}
			writeFile(t, path, []byte(wire), 0o600)
			resume, err := NewService().Preview(context.Background(), installed.target, candidate)
			if err == nil || resume.Digest != "" {
				t.Fatalf("ambiguous marker accepted: %+v %v", resume, err)
			}
		})
	}
}

func TestRecognizedPOCActivatedResumeRequiresCompleteRetirementProgress(t *testing.T) {
	for _, count := range []string{"0", "1", "999"} {
		t.Run(count, func(t *testing.T) {
			installed := pocInstallation(t)
			candidate := installed.candidate(t, newBundle("1.1.0", []byte("new-binary\n")))
			preview, err := NewService().Preview(context.Background(), installed.target, candidate)
			if err != nil {
				t.Fatal(err)
			}
			interrupted := Service{afterEffect: func(label string) error {
				if strings.HasPrefix(label, "retire:state/workflows/") {
					return errors.New("stop after activation")
				}
				return nil
			}}
			if _, err := applyPreview(t, interrupted, preview); err == nil {
				t.Fatal("interruption missing")
			}
			path := filepath.Join(installed.target.ReceiptDir, markerName)
			wire := strings.Replace(read(t, path), "transitionRetired=2", "transitionRetired="+count, 1)
			writeFile(t, path, []byte(wire), 0o600)
			resume, err := NewService().Preview(context.Background(), installed.target, candidate)
			if category(err) != "recovery_required" || resume.Digest != "" {
				t.Fatalf("inconsistent activation accepted: %+v %v", resume, err)
			}
		})
	}
}
