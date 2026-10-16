package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/rgomids/axiom/internal/capabilityproof"
	"github.com/rgomids/axiom/internal/cli"
	"github.com/rgomids/axiom/internal/local"
)

func (s lifecycleService) capabilityProbe(ctx context.Context, service capabilityproof.Service, in cli.RuntimeCapabilityInput, target capabilityproof.Target) cli.RuntimeCapabilityResponse {
	if in.PreviewDigest != "" && (!in.AuthorizeLocal || !in.AuthorizeExternal) {
		return s.capabilityFailure("probe_authority_required", nil)
	}
	p, err := service.PreviewProbe(ctx, target)
	if err != nil {
		return s.capabilityFailure("capability_unknown", nil)
	}
	digest, err := capabilityPreviewDigest(p)
	if err != nil {
		return s.capabilityFailure("capability_unknown", nil)
	}
	if in.PreviewDigest == "" {
		return s.capabilitySuccess("capability_preview_ready", map[string]any{"previewDigest": digest, "preview": p})
	}
	if in.PreviewDigest != digest {
		return s.capabilityFailure("stale_preview", nil)
	}
	out, err := service.Probe(ctx, p, capabilityproof.ProbeAuthority{Actor: stageActor(), Local: in.AuthorizeLocal, External: in.AuthorizeExternal, Preview: p})
	if err != nil {
		category := "probe_recovery_required"
		if errors.Is(err, capabilityproof.ErrAuthentication) {
			category = "authentication_blocked"
		} else if errors.Is(err, capabilityproof.ErrStalePreview) {
			category = "stale_preview"
		}
		return s.capabilityFailure(category, out)
	}
	category := "capability_probe_recorded"
	if out.Inconclusive {
		category = "capability_probe_inconclusive"
	}
	return s.capabilitySuccess(category, out)
}

func (s lifecycleService) capabilityEnvelopePolicy(in cli.RuntimeCapabilityInput) (local.EnvelopePolicy, error) {
	home, err := os.UserHomeDir()
	if err != nil || !filepath.IsAbs(in.RepositoryRoot) || !filepath.IsAbs(in.WorkspaceRoot) {
		return local.EnvelopePolicy{}, capabilityproof.ErrBlocked
	}
	vendor := []string{filepath.Join(home, ".codex"), filepath.Join(home, ".claude")}
	for _, key := range []string{"CODEX_HOME", "CLAUDE_CONFIG_DIR"} {
		if v := os.Getenv(key); v != "" {
			vendor = append(vendor, v)
		}
	}
	return local.EnvelopePolicy{StateRoot: s.stateRoot, Variant: in.Variant, RepositoryRoots: []string{in.RepositoryRoot, in.WorkspaceRoot}, VendorHomes: vendor}, nil
}

func (s lifecycleService) capabilityPrepare(ctx context.Context, in cli.RuntimeCapabilityInput) cli.RuntimeCapabilityResponse {
	if in.PreviewDigest != "" && !in.AuthorizePrepare {
		return s.capabilityFailure("preparation_authority_required", nil)
	}
	policy, err := s.capabilityEnvelopePolicy(in)
	if err != nil || in.ProbeID == "" {
		return s.capabilityFailure("preparation_blocked", nil)
	}
	limits := local.EnvelopeLimits{Timeout: 30 * time.Second, MaxFiles: 64, MaxBytes: 1 << 20}
	identity, err := local.InspectEnvelopePreparation(policy, in.ProbeID, limits)
	if err != nil {
		return s.capabilityFailure("preparation_blocked", nil)
	}
	p := struct {
		Operation string
		Identity  string
		Policy    local.EnvelopePolicy
		ProbeID   string
		Limits    local.EnvelopeLimits
	}{"capability-prepare-v1", identity, policy, in.ProbeID, limits}
	digest, err := capabilityPreviewDigest(p)
	if err != nil {
		return s.capabilityFailure("preparation_blocked", nil)
	}
	if in.PreviewDigest == "" {
		return s.capabilitySuccess("capability_preparation_preview", map[string]any{"previewDigest": digest, "preview": p})
	}
	if in.PreviewDigest != digest {
		return s.capabilityFailure("stale_preview", nil)
	}
	e, err := prepareCapabilityEnvelope(ctx, probePreparationAuthority{Actor: stageActor(), Local: in.AuthorizePrepare, Policy: policy, ProbeID: in.ProbeID, Limits: limits})
	if err != nil {
		return s.capabilityFailure("preparation_recovery_required", nil)
	}
	return s.capabilitySuccess("capability_envelope_prepared", map[string]any{"probeID": in.ProbeID, "manifestDigest": e.ManifestDigest(), "manifest": e.Manifest()})
}
