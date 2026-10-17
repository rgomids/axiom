package main

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/rgomids/axiom/internal/capabilityproof"
	"github.com/rgomids/axiom/internal/executiongraph"
	"github.com/rgomids/axiom/internal/local"
	"github.com/rgomids/axiom/internal/runtimeadapter"
	"github.com/rgomids/axiom/internal/runtimeprofile"
)

type probePreparationAuthority struct {
	Actor   string
	Local   bool
	Policy  local.EnvelopePolicy
	ProbeID string
	Limits  local.EnvelopeLimits
}

func runtimeadapterApplicability(p capabilityproof.ProbePreparation) runtimeprofile.EffortApplicability {
	return runtimeprofile.EffortApplicability{RuntimeID: p.Current.Binding.RuntimeID, RuntimeVersion: p.Current.Binding.Version, ExecutableDigest: p.Current.Binding.ExecutableDigest, InvocationMode: "noninteractive"}
}

func prepareCapabilityEnvelope(ctx context.Context, a probePreparationAuthority) (*local.Envelope, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !a.Local || strings.TrimSpace(a.Actor) == "" || strings.ContainsAny(a.Actor, "\x00\r\n") {
		return nil, capabilityproof.ErrAuthority
	}
	return local.CreateEnvelope(a.Policy, a.ProbeID, a.Limits)
}

type preparedProbeAdapter struct {
	source   runtimeadapter.BindingSource
	envelope *local.Envelope
}

func (a preparedProbeAdapter) Prepare(ctx context.Context, t capabilityproof.Target) (capabilityproof.ProbePreparation, error) {
	if a.envelope == nil {
		return capabilityproof.ProbePreparation{}, capabilityproof.ErrBlocked
	}
	if err := a.envelope.InspectReady(); err != nil {
		return capabilityproof.ProbePreparation{}, err
	}
	e := a.envelope
	m := e.Manifest()
	if t.Variant != m.Variant {
		return capabilityproof.ProbePreparation{}, capabilityproof.ErrBlocked
	}
	source := a.source
	original := source.Stage
	source.Stage = func(ctx context.Context, t capabilityproof.Target) (runtimeadapter.StageContext, error) {
		stage, err := original(ctx, t)
		if err != nil {
			return stage, err
		}
		stage.Workspace, stage.Inbox = filepath.Join(e.Path(), "workspace"), filepath.Join(e.Path(), "inbox")
		return stage, nil
	}
	p, err := source.PrepareProbe(ctx, t)
	if err != nil {
		return p, err
	}
	if t.Variant == capabilityproof.RepositoryWriteVariant {
		cfg, err := source.Configuration(ctx)
		if err != nil {
			return capabilityproof.ProbePreparation{}, err
		}
		roots, err := source.EffortRoots(ctx, t, executiongraph.EffectiveInvocationEnvironment(p.Invocation))
		if err != nil {
			return capabilityproof.ProbePreparation{}, err
		}
		request := runtimeadapter.EffortSourceRequest{Consented: source.EffortConsented, Configuration: cfg, ProfileID: t.ProfileID, Observed: runtimeadapterApplicability(p), Roots: roots, Invocation: p.Invocation}
		if err := runtimeadapter.VerifyChildEquivalence(ctx, request, filepath.Join(e.Path(), "workspace")); err != nil {
			return capabilityproof.ProbePreparation{}, err
		}
	}
	p.Manifest, p.ManifestDigest = m, e.ManifestDigest()
	instructions := "Read ../workspace/nonce and copy its exact bytes into ../inbox/read-nonce. "
	if m.Variant == capabilityproof.RepositoryWriteVariant {
		instructions += "Create ../workspace/probe-work with created: followed by the nonce, then edit it to append ;edited: followed by the nonce. "
	}
	instructions += fmt.Sprintf("Run coordination.axiomPath with arguments __probe-canary %s %s from the effective working directory. The deterministic canary alone may attempt the manifest negative targets. Publish only into this attempt inbox. Do not claim capabilities.", m.ProbeID, p.ManifestDigest)
	input := runtimeadapter.PacketInput{AgentID: m.ProbeID, Role: "capability-probe", Responsibilities: "Perform only the bounded manifest probe", Purpose: "Measure effects independently of Runtime claims", Instructions: instructions, Inputs: []runtimeadapter.PacketInputBinding{{InputID: "envelope", Reference: "probe-envelope", Digest: p.ManifestDigest}}, Scope: executiongraph.Scope{Paths: m.AllowedRoots}, Effects: []executiongraph.Effect{{Kind: "read", Target: "workspace/nonce"}, {Kind: "write", Target: "inbox"}, {Kind: "process", Target: "probe-canary"}}, Controls: executiongraph.ExecutionControls{Timeout: m.Limits.Timeout, MaximumAttempts: 1}}
	if m.Variant == capabilityproof.RepositoryWriteVariant {
		input.Effects = append(input.Effects, executiongraph.Effect{Kind: "repository-write", Target: "workspace/probe-work"})
	}
	binding := runtimeadapter.PacketContext{AttemptID: e.ProbeAttemptID(), Inbox: filepath.Join(e.Path(), "inbox", "attempt"), AxiomPath: source.AxiomPath, AxiomDigest: p.Current.Binding.AxiomDigest}
	p.Invocation, _, err = runtimeadapter.WithTaskPacket(p.Invocation, input, binding)
	if err != nil {
		return capabilityproof.ProbePreparation{}, err
	}
	if err := e.InspectReady(); err != nil {
		return capabilityproof.ProbePreparation{}, err
	}
	return p, nil
}

func (a preparedProbeAdapter) Run(ctx context.Context, inv executiongraph.Invocation) executiongraph.ProcessResult {
	return (executiongraph.OSProcessRunner{}).Run(ctx, inv)
}

type preparedProbeAuthentication struct {
	adapter   preparedProbeAdapter
	preflight runtimeadapter.AuthPreflight
}

func (a preparedProbeAuthentication) Proven(ctx context.Context, t capabilityproof.Target) (bool, error) {
	p, err := a.adapter.Prepare(ctx, t)
	if err != nil {
		return false, err
	}
	report := a.preflight.Check(ctx, runtimeadapter.AuthTarget{RuntimeID: t.RuntimeID, Executable: p.Invocation.Argv[0], ExpectedDigest: p.Current.Binding.ExecutableDigest, WorkingDirectory: p.Invocation.CWD, Environment: executiongraph.EffectiveInvocationEnvironment(p.Invocation)})
	return report.DispatchAllowed() && report.ExecutableDigest == p.Current.Binding.ExecutableDigest && report.Version == p.Current.Binding.Version, nil
}
