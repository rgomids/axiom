package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/rgomids/axiom/internal/capabilityproof"
	"github.com/rgomids/axiom/internal/cli"
)

type extendedFixture struct {
	f       *compositionFixture
	s       lifecycleService
	service capabilityproof.Service
	in      cli.RuntimeCapabilityInput
	root    string
}

func newExtendedFixture(t *testing.T, mode string) extendedFixture {
	t.Helper()
	f, service, target, envelope := newPreparedProbeFixture(t, "codex", capabilityproof.RepositoryWriteVariant, mode)
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("TMPDIR", root)
	f.stage.ParentEnvironment = append(f.stage.ParentEnvironment, "TMPDIR="+root)
	in := cli.RuntimeCapabilityInput{Operation: "probe-extended", RuntimeID: target.RuntimeID, ProfileID: target.ProfileID, Variant: target.Variant, ProbeID: envelope.Manifest().ProbeID, RootKinds: []string{"tmpdir"}, Process: true}
	return extendedFixture{f: f, s: lifecycleService{provenance: authProvenance(t)}, service: service, in: in, root: root}
}

func (x *extendedFixture) preview(t *testing.T) {
	t.Helper()
	r := x.s.capabilityOperation(context.Background(), x.service, x.in)
	if r.Result.Category != "capability_extended_preview" {
		t.Fatal(r)
	}
	x.in.PreviewDigest = r.Payload.(map[string]any)["previewDigest"].(string)
	x.in.AuthorizeLocal, x.in.AuthorizeExternal, x.in.AuthorizeExtended = true, true, true
}

func (x extendedFixture) apply(t *testing.T) (string, []capabilityproof.ExtendedResult) {
	t.Helper()
	r := x.s.capabilityOperation(context.Background(), x.service, x.in)
	payload, _ := r.Payload.(map[string]any)
	results, _ := payload["results"].([]capabilityproof.ExtendedResult)
	return r.Result.Category, results
}

func TestCapabilityExtendedPreviewRequiresItsOwnAuthority(t *testing.T) {
	x := newExtendedFixture(t, "success")
	before := probeTree(t, x.f.base)
	x.preview(t)
	x.in.AuthorizeExtended = false
	if category, _ := x.apply(t); category != "extended_authority_required" {
		t.Fatal("standard probe authority authorized extended operation", category)
	}
	x.in.AuthorizeExtended, x.in.AuthorizeExternal = true, false
	if category, _ := x.apply(t); category != "probe_authority_required" {
		t.Fatal(category)
	}
	if !reflect.DeepEqual(before, probeTree(t, x.f.base)) {
		t.Fatal("preview/refusal invoked auth, child, claim or Evidence write")
	}
	if entries, err := os.ReadDir(x.root); err != nil || len(entries) != 0 {
		t.Fatal("extended refusal changed the fake root", err)
	}
	x.in.AuthorizeExternal = true
	if err := os.Chmod(x.root, 0500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(x.root, 0700) })
	if category, _ := x.apply(t); category != "stale_preview" {
		t.Fatal("root mode drift accepted", category)
	}
}

func TestCapabilityExtendedMeasuresOnlyHostObservedEffects(t *testing.T) {
	t.Run("child-created file is advisory and cleaned", func(t *testing.T) {
		x := newExtendedFixture(t, "success")
		x.preview(t)
		category, results := x.apply(t)
		if category != "capability_extended_measured" || len(results) != 1 || results[0].Kind != "tmpdir" || results[0].State != "advisory" || !results[0].Measurement.Canary.Created {
			t.Fatal(category, results)
		}
		if entries, err := os.ReadDir(x.root); err != nil || len(entries) != 0 {
			t.Fatal("measurement left residue", entries, err)
		}
		if category, _ := x.apply(t); category == "capability_extended_measured" {
			t.Fatal("consumed envelope measured twice")
		}
	})
	t.Run("absent file stays unknown", func(t *testing.T) {
		x := newExtendedFixture(t, "extended-skip")
		x.preview(t)
		category, results := x.apply(t)
		if category != "capability_extended_measured" || len(results) != 1 || results[0].State != "unknown" {
			t.Fatal(category, results)
		}
	})
	t.Run("foreign residue is preserved and quarantined", func(t *testing.T) {
		x := newExtendedFixture(t, "extended-residue")
		x.preview(t)
		if category, _ := x.apply(t); category != "extended_residue_detected" {
			t.Fatal(category)
		}
		wire, err := os.ReadFile(filepath.Join(x.root, ".axiom-probe-"+x.in.ProbeID))
		if err != nil || string(wire) != "not empty" {
			t.Fatal("non-owned residue removed", err)
		}
	})
	t.Run("authentication blocks before effects", func(t *testing.T) {
		x := newExtendedFixture(t, "success")
		settings, _ := json.Marshal(probeFakeSettings{Runtime: "codex", Mode: "success", Auth: false})
		if err := os.WriteFile(filepath.Join(filepath.Dir(x.f.exe), "fake-probe-runtime.json"), settings, 0600); err != nil {
			t.Fatal(err)
		}
		x.preview(t)
		before := probeTree(t, x.f.base)
		if category, _ := x.apply(t); category != "authentication_blocked" {
			t.Fatal(category)
		}
		if !reflect.DeepEqual(before, probeTree(t, x.f.base)) {
			t.Fatal("blocked authentication claimed the envelope")
		}
		if entries, _ := os.ReadDir(x.root); len(entries) != 0 {
			t.Fatal("blocked authentication touched the root")
		}
	})
}
