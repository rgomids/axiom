package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/rgomids/axiom/internal/capabilityproof"
	"github.com/rgomids/axiom/internal/cli"
	"github.com/rgomids/axiom/internal/executiongraph"
	"github.com/rgomids/axiom/internal/local"
	"github.com/rgomids/axiom/internal/runtimeadapter"
)

func (s lifecycleService) capabilityExtended(ctx context.Context, service capabilityproof.Service, in cli.RuntimeCapabilityInput, target capabilityproof.Target) cli.RuntimeCapabilityResponse {
	if in.PreviewDigest != "" && (!in.AuthorizeLocal || !in.AuthorizeExternal) {
		return s.capabilityFailure("probe_authority_required", nil)
	}
	if in.PreviewDigest != "" && !in.AuthorizeExtended {
		return s.capabilityFailure("extended_authority_required", nil)
	}
	observe := func() (extendedCapabilityObservation, capabilityproof.ExtendedService, string, string) {
		probe, err := service.PreviewProbe(ctx, target)
		if err != nil {
			return extendedCapabilityObservation{}, capabilityproof.ExtendedService{}, "", "capability_unknown"
		}
		if probe.Preparation.Manifest.ProbeID != in.ProbeID {
			return extendedCapabilityObservation{}, capabilityproof.ExtendedService{}, "", "stale_preview"
		}
		extended, roots, err := extendedCapabilityPreview(ctx, in, probe)
		if err != nil {
			return extendedCapabilityObservation{}, capabilityproof.ExtendedService{}, "", "extended_root_refused"
		}
		digest, err := capabilityPreviewDigest(extended)
		if err != nil {
			return extendedCapabilityObservation{}, capabilityproof.ExtendedService{}, "", "capability_unknown"
		}
		return extended, roots, digest, ""
	}
	extended, roots, digest, category := observe()
	if category != "" {
		return s.capabilityFailure(category, nil)
	}
	if in.PreviewDigest == "" {
		return s.capabilitySuccess("capability_extended_preview", map[string]any{"previewDigest": digest, "preview": extended})
	}
	if in.PreviewDigest != digest {
		return s.capabilityFailure("stale_preview", nil)
	}
	if service.ProbeAuth == nil || service.ProbeEnvelopes == nil || service.ProbeAdapter == nil {
		return s.capabilityFailure("capability_unknown", nil)
	}
	if ok, err := service.ProbeAuth.Proven(ctx, target); err != nil || !ok {
		return s.capabilityFailure("authentication_blocked", nil)
	}
	extended, roots, fresh, category := observe()
	if category != "" || fresh != digest {
		return s.capabilityFailure("stale_preview", nil)
	}
	envelope, err := service.ProbeEnvelopes.Open(ctx, extended.Probe.Preparation)
	if err != nil || envelope == nil {
		return s.capabilityFailure("probe_recovery_required", nil)
	}
	if err := local.BindExtendedRootPermit(ctx, envelope, extended.Roots); err != nil {
		finish, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*extended.Roots.Timeout)
		defer cancel()
		quarantined := envelope.Quarantine(finish) == nil
		return s.capabilityFailure("probe_recovery_required", map[string]any{"previewDigest": digest, "quarantined": quarantined})
	}
	observer := roots.Inspector.(local.ExtendedRootObserver)
	roots.Effects = &local.ExtendedRootEffects{ExtendedRootObserver: observer, Run: func(ctx context.Context, root capabilityproof.ExtendedRoot, name string) error {
		inv, err := extendedCanaryInvocation(extended.Probe.Preparation, filepath.Join(root.Path, name))
		if err != nil {
			return err
		}
		r := service.ProbeAdapter.Run(ctx, inv)
		if r.ExitCode != 0 || r.TimedOut || r.OutcomeUnknown || r.EffectAmbiguous || r.Stopped {
			return capabilityproof.ErrBlocked
		}
		return nil
	}}
	results, err := roots.Apply(ctx, extended.Roots, capabilityproof.ExtendedAuthority{Actor: stageActor(), Local: true, External: true, Preview: extended.Roots})
	finish, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*extended.Roots.Timeout)
	defer cancel()
	payload := map[string]any{"previewDigest": digest, "results": results, "denial": "unattested; only a host-observed file proves a writable root"}
	if err != nil {
		if quarantineErr := envelope.Quarantine(finish); quarantineErr != nil {
			err = errors.Join(err, quarantineErr)
		}
		category := "probe_recovery_required"
		if errors.Is(err, capabilityproof.ErrProbeResidue) {
			category = "extended_residue_detected"
		} else if errors.Is(err, capabilityproof.ErrStalePreview) {
			category = "stale_preview"
		}
		return s.capabilityFailure(category, payload)
	}
	if err := envelope.Cleanup(finish); err != nil {
		return s.capabilityFailure("probe_recovery_required", payload)
	}
	return s.capabilitySuccess("capability_extended_measured", payload)
}

type extendedCapabilityObservation struct {
	Operation string
	Probe     capabilityproof.ProbePreview
	Roots     capabilityproof.ExtendedPreview
	Canaries  map[string]string
	Cost      string
}

func extendedCanaryInvocation(p capabilityproof.ProbePreparation, path string) (executiongraph.Invocation, error) {
	var packet runtimeadapter.TaskPacket
	if err := json.Unmarshal(p.Invocation.Stdin, &packet); err != nil {
		return executiongraph.Invocation{}, err
	}
	m := p.Manifest
	instructions := fmt.Sprintf("Run coordination.axiomPath with arguments __probe-canary-extended %s %s %s from the effective working directory, exactly once. Do nothing else and do not claim capabilities.", m.ProbeID, p.ManifestDigest, path)
	input := runtimeadapter.PacketInput{AgentID: packet.AgentID, Role: packet.Role, Responsibilities: "Perform only the bounded extended temp-root canary", Purpose: "Measure temp-root writability independently of Runtime claims", Instructions: instructions, Inputs: packet.Inputs, Scope: executiongraph.Scope{Paths: packet.AllowedPaths}, Effects: []executiongraph.Effect{{Kind: "process", Target: "probe-canary-extended"}}, Controls: executiongraph.ExecutionControls{Timeout: m.Limits.Timeout, MaximumAttempts: 1}}
	inv, _, err := runtimeadapter.WithTaskPacket(p.Invocation, input, packet.Coordination)
	return inv, err
}

func extendedCapabilityPreview(ctx context.Context, in cli.RuntimeCapabilityInput, probe capabilityproof.ProbePreview) (extendedCapabilityObservation, capabilityproof.ExtendedService, error) {
	service, err := local.ExtendedProbeService(in.RootKinds)
	if err != nil {
		return extendedCapabilityObservation{}, capabilityproof.ExtendedService{}, err
	}
	p, err := service.Preview(ctx, in.ProbeID, in.RootKinds)
	if err != nil {
		return extendedCapabilityObservation{}, service, err
	}
	o := extendedCapabilityObservation{Operation: "capability-extended-preview-v2", Probe: probe, Roots: p, Canaries: map[string]string{}, Cost: fmt.Sprintf("one vendor inference invocation per root (%d); no retries", len(p.Roots))}
	for _, root := range p.Roots {
		inv, err := extendedCanaryInvocation(probe.Preparation, filepath.Join(root.Path, p.Filename))
		if err != nil {
			return extendedCapabilityObservation{}, service, err
		}
		sum := sha256.Sum256(inv.Stdin)
		o.Canaries[root.Kind] = hex.EncodeToString(sum[:])
	}
	return o, service, nil
}
