package main

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/rgomids/axiom/internal/capabilityproof"
	"github.com/rgomids/axiom/internal/cli"
	"github.com/rgomids/axiom/internal/local"
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

func TestCapabilityProbeCommandGatesAndFakeRun(t *testing.T) {
	for _, runtime := range []string{"codex", "claude"} {
		t.Run(runtime, func(t *testing.T) {
			f, service, target, e := newPreparedProbeFixture(t, runtime, capabilityproof.RepositoryWriteVariant, "success")
			s := lifecycleService{provenance: authProvenance(t)}
			in := cli.RuntimeCapabilityInput{Operation: "probe", RuntimeID: runtime, ProfileID: target.ProfileID, Variant: target.Variant, Process: true}
			before := probeTree(t, f.base)
			r := s.capabilityOperation(context.Background(), service, in)
			if r.Result.Category != "capability_preview_ready" {
				t.Fatal(r)
			}
			in.PreviewDigest = r.Payload.(map[string]any)["previewDigest"].(string)
			for _, flags := range [][2]bool{{false, false}, {true, false}, {false, true}} {
				in.AuthorizeLocal, in.AuthorizeExternal = flags[0], flags[1]
				if r = s.capabilityOperation(context.Background(), service, in); r.Result.Category != "probe_authority_required" {
					t.Fatal(r)
				}
			}
			in.AuthorizeLocal, in.AuthorizeExternal = true, true
			digest := in.PreviewDigest
			in.PreviewDigest = strings.Repeat("0", 64)
			if r = s.capabilityOperation(context.Background(), service, in); r.Result.Category != "stale_preview" {
				t.Fatal(r)
			}
			if !reflect.DeepEqual(before, probeTree(t, f.base)) {
				t.Fatal("preview or refusal wrote state")
			}
			in.PreviewDigest = digest
			if r = s.capabilityOperation(context.Background(), service, in); r.Result.Category != "capability_probe_inconclusive" {
				t.Fatal(r)
			}
			out := r.Payload.(capabilityproof.ProbeOutcome)
			for _, result := range out.Evidence.Capabilities {
				if result.State != capabilityproof.Unknown {
					t.Fatal("child data promoted capability", out)
				}
			}
			if _, err := local.OpenPreparedEnvelope(local.EnvelopePolicy{StateRoot: f.state, Variant: target.Variant}, e.Manifest().ProbeID, e.ManifestDigest()); err == nil {
				t.Fatal("completed preparation reused")
			}
		})
	}
}

func TestCapabilityPreparationCommand(t *testing.T) {
	f, _, target, _ := newPreparedProbeFixture(t, "codex", capabilityproof.RepositoryWriteVariant, "success")
	s := lifecycleService{stateRoot: f.state, provenance: authProvenance(t)}
	in := cli.RuntimeCapabilityInput{Operation: "prepare", RuntimeID: "codex", ProfileID: target.ProfileID, Variant: target.Variant, ProbeID: "separate-preparation", RepositoryRoot: f.roots.Project, WorkspaceRoot: f.stage.Workspace}
	ctx := context.Background()
	before := probeTree(t, f.base)
	r := s.RuntimeCapability(ctx, in)
	if r.Result.Category != "capability_preparation_preview" {
		t.Fatal(r)
	}
	in.PreviewDigest = r.Payload.(map[string]any)["previewDigest"].(string)
	in.AuthorizeLocal, in.AuthorizeExternal = true, true
	if r = s.RuntimeCapability(ctx, in); r.Result.Category != "preparation_authority_required" {
		t.Fatal("standard probe authority authorized preparation", r)
	}
	if !reflect.DeepEqual(before, probeTree(t, f.base)) {
		t.Fatal("preview or refusal wrote state")
	}
	in.AuthorizePrepare = true
	if r = s.RuntimeCapability(ctx, in); r.Result.Category != "capability_envelope_prepared" {
		t.Fatal(r)
	}
	manifest := r.Payload.(map[string]any)["manifestDigest"].(string)
	policy, err := s.capabilityEnvelopePolicy(in)
	if err != nil {
		t.Fatal(err)
	}
	e, err := local.OpenPreparedEnvelope(policy, in.ProbeID, manifest)
	if err != nil || e.InspectReady() != nil {
		t.Fatal("separate prepared envelope not readable", err)
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
