package main

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/rgomids/axiom/internal/capabilityproof"
	"github.com/rgomids/axiom/internal/cli"
)

func TestCapabilityExtendedPreviewAndFailClosedApply(t *testing.T) {
	f, service, target, envelope := newPreparedProbeFixture(t, "codex", capabilityproof.RepositoryWriteVariant, "success")
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("TMPDIR", root)
	s := lifecycleService{provenance: authProvenance(t)}
	in := cli.RuntimeCapabilityInput{Operation: "probe-extended", RuntimeID: target.RuntimeID, ProfileID: target.ProfileID, Variant: target.Variant, ProbeID: envelope.Manifest().ProbeID, RootKinds: []string{"tmpdir"}, Process: true}
	before := probeTree(t, f.base)
	r := s.capabilityOperation(context.Background(), service, in)
	if r.Result.Category != "capability_extended_preview" {
		t.Fatal(r)
	}
	in.PreviewDigest = r.Payload.(map[string]any)["previewDigest"].(string)
	in.AuthorizeLocal, in.AuthorizeExternal = true, true
	if r = s.capabilityOperation(context.Background(), service, in); r.Result.Category != "extended_authority_required" {
		t.Fatal("standard probe authority authorized extended operation", r)
	}
	in.AuthorizeExtended = true
	if r = s.capabilityOperation(context.Background(), service, in); r.Result.Category != "extended_measurement_unavailable" {
		t.Fatal("unobserved measurement accepted", r)
	}
	if !reflect.DeepEqual(before, probeTree(t, f.base)) {
		t.Fatal("preview/refusal invoked auth, child, claim or Evidence write")
	}
	entries, err := os.ReadDir(root)
	if err != nil || len(entries) != 0 {
		t.Fatal("extended refusal changed the fake root", err)
	}
	if err := os.Chmod(root, 0500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(root, 0700) })
	if r = s.capabilityOperation(context.Background(), service, in); r.Result.Category != "stale_preview" {
		t.Fatal("root mode drift accepted", r)
	}
}
