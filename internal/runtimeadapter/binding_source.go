package runtimeadapter

import (
	"context"
	"errors"
	"path/filepath"
	"slices"

	"github.com/rgomids/axiom/internal/capabilityproof"
	"github.com/rgomids/axiom/internal/runtimeprofile"
)

// StageContext is the caller's effective dispatch context for one Target: the
// child's workspace, inbox and parent environment. It is required, never
// defaulted; ModeDigest records only the path roles.
type StageContext struct {
	Workspace, Inbox  string
	ParentEnvironment []string
}

// BindingSource is the production capabilityproof.BindingSource
// (Specification 007 AXM-22 BI-1/BI-2/BI-2a). Every Current observes afresh the
// version-dispatched configuration, Runtime executable and version, pinned
// Axiom executable, effective stage mode and effort sources, then asks the
// AXM-21 Selector for the unique Evidence equal to that identity. No identity
// is read from Evidence, ProfileProof or a ResolvedProfile, and nothing here
// writes, probes, runs a Runtime or proves a capability.
type BindingSource struct {
	// Configuration reads local configuration through the approved
	// version-aware path (runtimeprofile.DecodeVersioned).
	Configuration func(context.Context) (runtimeprofile.VersionedConfiguration, error)
	// LookPath resolves a configured Runtime adapter to its absolute executable.
	LookPath func(string) (string, error)
	// Version acquires the Runtime version for an observed executable identity.
	Version func(ctx context.Context, runtimeID, executableDigest string) (string, error)
	// AxiomPath is the pinned Axiom executable the stage permits.
	AxiomPath string
	// Stage supplies the effective dispatch context for the Target.
	Stage func(context.Context, capabilityproof.Target) (StageContext, error)
	// EffortRoots supplies the trusted current effort-source anchor roots.
	EffortRoots     func(context.Context) (EffortSourceRoots, error)
	EffortConsented bool
	Selector        capabilityproof.Selector
}

var _ capabilityproof.BindingSource = BindingSource{}

func (s BindingSource) Current(ctx context.Context, t capabilityproof.Target) (capabilityproof.Current, error) {
	b, m, err := s.observe(ctx, t)
	if err != nil {
		return capabilityproof.Current{}, err
	}
	ref, _, err := s.Selector.Select(ctx, b, &m)
	if err != nil {
		return capabilityproof.Current{}, err
	}
	return capabilityproof.Current{Binding: b, Model: &m, EvidenceRef: ref, ProfileActive: true}, nil
}

func (s BindingSource) observe(ctx context.Context, t capabilityproof.Target) (capabilityproof.Binding, capabilityproof.ModelBinding, error) {
	var none capabilityproof.Binding
	var noModel capabilityproof.ModelBinding
	if err := ctx.Err(); err != nil {
		return none, noModel, err
	}
	if s.Configuration == nil || s.LookPath == nil || s.Version == nil || s.Stage == nil || s.EffortRoots == nil || t.ProfileID == "" || t.Variant != capabilityproof.ReadOnlyVariant && t.Variant != capabilityproof.RepositoryWriteVariant {
		return none, noModel, capabilityproof.ErrBlocked
	}
	cfg, err := s.Configuration(ctx)
	if err != nil {
		return none, noModel, blocked(ctx, err)
	}
	// Only format 3 carries an EffortDeclaration; formats 1/2 stay readable
	// elsewhere but their effort configuration is unproven (D3).
	if cfg.FormatVersion != runtimeprofile.FormatVersionV3 || cfg.V3 == nil {
		return none, noModel, ErrEffortConfigurationUnproven
	}
	if runtimeprofile.ValidateV3(*cfg.V3) != nil {
		return none, noModel, capabilityproof.ErrBlocked
	}
	runtime, profile, ok := activeProfile(*cfg.V3, t)
	if !ok {
		return none, noModel, capabilityproof.ErrBlocked
	}
	profileDigest, err := runtimeprofile.ProfileDigest(profile)
	if err != nil {
		return none, noModel, capabilityproof.ErrBlocked
	}
	executable, err := s.LookPath(runtime.Adapter)
	if err != nil || !filepath.IsAbs(executable) {
		return none, noModel, capabilityproof.ErrBlocked
	}
	executableDigest, err := ExecutableIdentity(executable)
	if err != nil {
		return none, noModel, capabilityproof.ErrBlocked
	}
	version, err := s.Version(ctx, t.RuntimeID, executableDigest)
	if err != nil || version == "" {
		return none, noModel, blocked(ctx, err)
	}
	axiomDigest, err := ExecutableIdentity(s.AxiomPath)
	if err != nil {
		return none, noModel, capabilityproof.ErrBlocked
	}
	stage, err := s.Stage(ctx, t)
	if err != nil {
		return none, noModel, blocked(ctx, err)
	}
	request := StageInvocationRequest{Profile: CommandProfile{RuntimeID: t.RuntimeID, ModelProfileID: profile.ID, Executable: executable, Model: profile.Model}, Version: version, Workspace: stage.Workspace, Inbox: stage.Inbox, AxiomPath: s.AxiomPath, Process: t.Process, RepositoryWrite: t.Variant == capabilityproof.RepositoryWriteVariant, ParentEnvironment: stage.ParentEnvironment}
	// The same effective request feeds the canonical mode identity and, with
	// the Profile's effort encoding, the effort-source inspection.
	mode, err := ModeDigest(request)
	if err != nil {
		return none, noModel, capabilityproof.ErrBlocked
	}
	adapter, err := AdapterVersion(t.RuntimeID)
	if err != nil {
		return none, noModel, capabilityproof.ErrBlocked
	}
	base, err := BuildStageInvocation(request)
	if err != nil {
		return none, noModel, capabilityproof.ErrBlocked
	}
	inv, err := withStageEffort(base, profile.Effort)
	if err != nil {
		return none, noModel, capabilityproof.ErrBlocked
	}
	roots, err := s.EffortRoots(ctx)
	if err != nil {
		return none, noModel, blocked(ctx, err)
	}
	effortDigest, err := InspectEffortSources(ctx, EffortSourceRequest{Consented: s.EffortConsented, Configuration: cfg, ProfileID: profile.ID, Observed: runtimeprofile.EffortApplicability{RuntimeID: t.RuntimeID, RuntimeVersion: version, ExecutableDigest: executableDigest, InvocationMode: "noninteractive"}, Roots: roots, Invocation: inv})
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return none, noModel, ctxErr
		}
		return none, noModel, err
	}
	// Executables replaced during observation are not the ones observed.
	if after, err := ExecutableIdentity(executable); err != nil || after != executableDigest {
		return none, noModel, capabilityproof.ErrBlocked
	}
	if after, err := ExecutableIdentity(s.AxiomPath); err != nil || after != axiomDigest {
		return none, noModel, capabilityproof.ErrBlocked
	}
	b := capabilityproof.Binding{RuntimeID: t.RuntimeID, ExecutableDigest: executableDigest, Version: version, ModeDigest: mode, Variant: t.Variant, AdapterVersion: adapter, AxiomDigest: axiomDigest}
	m := capabilityproof.ModelBinding{ProfileID: profile.ID, ProfileDigest: profileDigest, Model: profile.Model, EffortMode: profile.Effort.Mode, EffortToken: profile.Effort.Value, ConfigurationRevision: cfg.V3.Revision, EffortSourceDigest: effortDigest}
	if !capabilityproof.ValidBinding(b) {
		return none, noModel, capabilityproof.ErrBlocked
	}
	return b, m, nil
}

// activeProfile requires an enabled Runtime that allowlists exactly the
// Target's Profile, and that Profile bound to the same Runtime.
func activeProfile(c runtimeprofile.ConfigurationV3, t capabilityproof.Target) (runtimeprofile.RuntimeV3, runtimeprofile.ModelProfile, bool) {
	for _, r := range c.Runtimes {
		if r.ID != t.RuntimeID || r.Adapter != t.RuntimeID || !r.Enabled || !slices.Contains(r.AllowlistedProfileIDs, t.ProfileID) {
			continue
		}
		for _, p := range c.ModelProfiles {
			if p.ID == t.ProfileID && p.RuntimeID == t.RuntimeID {
				return r, p, true
			}
		}
	}
	return runtimeprofile.RuntimeV3{}, runtimeprofile.ModelProfile{}, false
}

// blocked keeps cancellation and deadline identities and hides every other
// dependency error behind the fail-closed sentinel.
func blocked(ctx context.Context, err error) error {
	if ctxErr := ctx.Err(); ctxErr != nil {
		return ctxErr
	}
	if errors.Is(err, context.Canceled) {
		return context.Canceled
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return context.DeadlineExceeded
	}
	return capabilityproof.ErrBlocked
}
