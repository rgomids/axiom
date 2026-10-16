package main

import (
	"context"
	"os"
	"path/filepath"

	"github.com/rgomids/axiom/internal/capabilityproof"
	"github.com/rgomids/axiom/internal/cli"
	"github.com/rgomids/axiom/internal/local"
)

func (s lifecycleService) capabilityExtended(ctx context.Context, service capabilityproof.Service, in cli.RuntimeCapabilityInput, target capabilityproof.Target) cli.RuntimeCapabilityResponse {
	if in.PreviewDigest != "" && (!in.AuthorizeLocal || !in.AuthorizeExternal) {
		return s.capabilityFailure("probe_authority_required", nil)
	}
	if in.PreviewDigest != "" && !in.AuthorizeExtended {
		return s.capabilityFailure("extended_authority_required", nil)
	}
	probe, err := service.PreviewProbe(ctx, target)
	if err != nil {
		return s.capabilityFailure("capability_unknown", nil)
	}
	if probe.Preparation.Manifest.ProbeID != in.ProbeID {
		return s.capabilityFailure("stale_preview", nil)
	}
	extended, err := extendedCapabilityPreview(ctx, in, probe)
	if err != nil {
		return s.capabilityFailure("extended_root_refused", nil)
	}
	digest, err := capabilityPreviewDigest(extended)
	if err != nil {
		return s.capabilityFailure("capability_unknown", nil)
	}
	if in.PreviewDigest == "" {
		return s.capabilitySuccess("capability_extended_preview", map[string]any{"previewDigest": digest, "preview": extended})
	}
	if in.PreviewDigest != digest {
		return s.capabilityFailure("stale_preview", nil)
	}
	return s.capabilityFailure("extended_measurement_unavailable", map[string]any{"state": "unknown", "previewDigest": digest, "effects": "none; no independently observed canary or owned-file cleanup backend"})
}

type extendedCapabilityObservation struct {
	Operation string
	Probe     capabilityproof.ProbePreview
	Roots     capabilityproof.ExtendedPreview
}

func extendedCapabilityPreview(ctx context.Context, in cli.RuntimeCapabilityInput, probe capabilityproof.ProbePreview) (extendedCapabilityObservation, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return extendedCapabilityObservation{}, err
	}
	home, err = filepath.EvalSymlinks(home)
	if err != nil {
		return extendedCapabilityObservation{}, err
	}
	temp, err := filepath.EvalSymlinks(os.TempDir())
	if err != nil {
		return extendedCapabilityObservation{}, err
	}
	system, err := filepath.EvalSymlinks("/tmp")
	if err != nil {
		return extendedCapabilityObservation{}, err
	}
	policy := capabilityproof.ExtendedPolicy{TempDir: temp, SystemTemp: system, Home: home, VendorHomes: []string{filepath.Join(home, ".codex"), filepath.Join(home, ".claude")}}
	for _, key := range []string{"CODEX_HOME", "CLAUDE_CONFIG_DIR"} {
		if path := os.Getenv(key); path != "" {
			policy.VendorHomes = append(policy.VendorHomes, path)
		}
	}
	for i, path := range policy.VendorHomes {
		if physical, err := filepath.EvalSymlinks(path); err == nil {
			policy.VendorHomes[i] = physical
		} else if !os.IsNotExist(err) {
			return extendedCapabilityObservation{}, err
		}
	}
	observer := local.ExtendedRootObserver{}
	for _, kind := range in.RootKinds {
		switch kind {
		case "tmpdir":
			observer.TempDir = temp
		case "system-temp":
			observer.SystemTemp = system
		}
	}
	p, err := (capabilityproof.ExtendedService{Policy: policy, Inspector: observer}).Preview(ctx, in.ProbeID, in.RootKinds)
	return extendedCapabilityObservation{Operation: "capability-extended-preview-v1", Probe: probe, Roots: p}, err
}
