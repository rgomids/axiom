package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rgomids/axiom/internal/capabilityproof"
	"github.com/rgomids/axiom/internal/cli"
)

func TestCapabilityInvalidationRoundTrip(t *testing.T) {
	f := newCompositionFixture(t)
	target := compositionTarget(f.cfg.ModelProfiles[0].ID, capabilityproof.RepositoryWriteVariant, false)
	ref := f.record(t, target, time.Now().UTC(), "a", capabilityproof.Read)
	service := f.service(t)
	s := lifecycleService{provenance: authProvenance(t)}
	in := cli.RuntimeCapabilityInput{RuntimeID: target.RuntimeID, ProfileID: target.ProfileID, Variant: target.Variant, Operation: "show"}
	ctx := context.Background()
	r := s.capabilityOperation(ctx, service, in)
	if r.Result.Category != "capability_observed" || r.Payload.(capabilityproof.View).States[capabilityproof.Read] != capabilityproof.Proven {
		t.Fatalf("show: %+v", r)
	}
	in.Operation = "invalidate"
	p := s.capabilityOperation(ctx, service, in)
	if p.Result.Category != "capability_preview_ready" {
		t.Fatalf("preview: %+v", p)
	}
	in.PreviewDigest = p.Payload.(map[string]any)["previewDigest"].(string)
	if r = s.capabilityOperation(ctx, service, in); r.Result.Category != "local_authority_required" {
		t.Fatal(r)
	}
	in.AuthorizeLocal = true
	reviewed := in.PreviewDigest
	in.PreviewDigest = strings.Repeat("0", 64)
	if r = s.capabilityOperation(ctx, service, in); r.Result.Category != "stale_preview" {
		t.Fatal(r)
	}
	stored, err := f.evidence.Load(ctx, ref)
	if err != nil || stored.InvalidatedReason != "" {
		t.Fatal("refusal changed evidence", err)
	}
	in.PreviewDigest = reviewed
	if r = s.capabilityOperation(ctx, service, in); r.Result.Category != "capability_invalidated" {
		t.Fatal(r)
	}
	in.Operation = "show"
	if r = s.capabilityOperation(ctx, service, in); r.Payload.(capabilityproof.View).States[capabilityproof.Read] != capabilityproof.Unknown {
		t.Fatal("invalidation promoted capability", r)
	}
}

func TestCapabilityPublicFreshStateIsReadOnly(t *testing.T) {
	state := filepath.Join(t.TempDir(), "state")
	s := lifecycleService{stateRoot: state, provenance: authProvenance(t)}
	for _, op := range []string{"show", "invalidate"} {
		r := s.RuntimeCapability(context.Background(), cli.RuntimeCapabilityInput{Operation: op, RuntimeID: "codex", ProfileID: "p", Variant: capabilityproof.ReadOnlyVariant})
		if r.Result.Category != "capability_unknown" {
			t.Fatal(r)
		}
	}
	if _, err := os.Lstat(state); !os.IsNotExist(err) {
		t.Fatal("refusal wrote state", err)
	}
}
