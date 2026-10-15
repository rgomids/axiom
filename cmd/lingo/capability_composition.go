package main

import (
	"context"
	"os"
	"path/filepath"

	"github.com/rgomids/axiom/internal/capabilityproof"
	"github.com/rgomids/axiom/internal/local"
	"github.com/rgomids/axiom/internal/runtimeadapter"
	"github.com/rgomids/axiom/internal/runtimebootstrap"
	"github.com/rgomids/axiom/internal/runtimeprofile"
	"github.com/rgomids/axiom/internal/stageexecution"
)

// capabilityDependencies are the inputs of the production capability
// composition (Specification 007 AXM-22). Every field is a current host or
// local-state observation; none is read from Evidence, a declaration or a
// ResolvedProfile.
type capabilityDependencies struct {
	stateRoot string
	lookPath  runtimebootstrap.LookPath
	// version is the passive Runtime version observer; production uses only
	// the embedded registry (PM-5a).
	version         func(context.Context, string, string) (string, error)
	axiomPath       string
	stage           func(context.Context, capabilityproof.Target) (runtimeadapter.StageContext, error)
	effortRoots     func(context.Context, capabilityproof.Target, []string) (runtimeadapter.EffortSourceRoots, error)
	effortConsented bool
	envelope        *local.Envelope
}

// stageOperation scopes capability observation to one authorized Stage
// operation (PM-13): its registered repository root, its workspace root and
// whether the operator gave explicit read-only effort-inspection consent with
// --authorize-local for this operation. Consent is not dispatch or external
// authority and proves no capability.
type stageOperation struct {
	repository, workspaceRoot string
	consented                 bool
	probe                     *local.Envelope
}

// productionCapabilityService composes the canonical Evidence store with the
// production runtimeadapter.BindingSource. A present Source enables fresh
// identity observation and AXM-21 Evidence selection only; it proves nothing.
// Probe ports require a live, independently prepared T13 handle. Composition
// itself launches no effects and reads only the embedded version registry.
func productionCapabilityService(stateRoot string, lookPath runtimebootstrap.LookPath, op stageOperation) (capabilityproof.Service, error) {
	registry, err := runtimeadapter.EmbeddedVersionObserver()
	if err != nil {
		return capabilityproof.Service{}, err
	}
	axiom, err := os.Executable()
	if err != nil {
		return capabilityproof.Service{}, err
	}
	return composeCapabilityService(capabilityDependencies{stateRoot: stateRoot, lookPath: lookPath, version: registry.Version, axiomPath: axiom, stage: op.context, effortRoots: runtimeadapter.StageEffortRoots{Repository: op.repository}.Resolve, effortConsented: op.consented, envelope: op.probe})
}

// context is the operation's effective dispatch context. ModeDigest and the
// effort projection use only the path roles of the workspace and inbox
// (BI-1, BI-2b), so the operation's own namespaces identify every child of a
// shared Target; the parent environment is the one the detached supervisor
// inherits and filters for each attempt.
func (op stageOperation) context(ctx context.Context, _ capabilityproof.Target) (runtimeadapter.StageContext, error) {
	if err := ctx.Err(); err != nil {
		return runtimeadapter.StageContext{}, err
	}
	if !filepath.IsAbs(op.workspaceRoot) || filepath.Clean(op.workspaceRoot) != op.workspaceRoot {
		return runtimeadapter.StageContext{}, capabilityproof.ErrBlocked
	}
	return runtimeadapter.StageContext{Workspace: filepath.Join(op.workspaceRoot, "worktrees"), Inbox: filepath.Join(op.workspaceRoot, "inboxes"), ParentEnvironment: os.Environ()}, nil
}

func composeCapabilityService(d capabilityDependencies) (capabilityproof.Service, error) {
	store, err := local.NewCapabilityStore(d.stateRoot)
	if err != nil {
		return capabilityproof.Service{}, err
	}
	profiles, err := local.NewRuntimeProfileStore(d.stateRoot)
	if err != nil {
		return capabilityproof.Service{}, err
	}
	if d.lookPath == nil || d.version == nil || !filepath.IsAbs(d.axiomPath) || d.stage == nil || d.effortRoots == nil {
		return capabilityproof.Service{}, capabilityproof.ErrBlocked
	}
	source := runtimeadapter.BindingSource{
		Configuration:   canonicalV3Configuration(profiles),
		LookPath:        d.lookPath,
		Version:         d.version,
		AxiomPath:       d.axiomPath,
		Stage:           d.stage,
		EffortRoots:     d.effortRoots,
		EffortConsented: d.effortConsented,
		Selector:        capabilityproof.Selector{Evidence: store},
	}
	adapter := preparedProbeAdapter{source: source, envelope: d.envelope}
	auth, err := runtimeadapter.NewAuthPreflight(runtimeadapter.OSStatusRunner{}, runtimeadapter.DefaultManagedConfiguration())
	if err != nil {
		return capabilityproof.Service{}, err
	}
	return capabilityproof.Service{Store: store, Source: source, ProbeAdapter: adapter, ProbeEnvelopes: local.PreparedProbeEnvelopes{Envelope: d.envelope}, ProbeWriter: store, ProbeAuth: preparedProbeAuthentication{adapter: adapter, preflight: auth}}, nil
}

// productionEffortInspector is the supervisor's dispatch-time PE-05a
// inspector (PM-13). Its consent is the committed Stage dispatch authority;
// repository-write children are those the reviewed preview grants
// repository-write.
func productionEffortInspector(stateRoot, repository string, children []stageexecution.ChildRights) (runtimeadapter.StageEffortInspector, error) {
	profiles, err := local.NewRuntimeProfileStore(stateRoot)
	if err != nil {
		return runtimeadapter.StageEffortInspector{}, err
	}
	registry, err := runtimeadapter.EmbeddedVersionObserver()
	if err != nil {
		return runtimeadapter.StageEffortInspector{}, err
	}
	inspector := runtimeadapter.StageEffortInspector{Configuration: canonicalV3Configuration(profiles), Version: registry.Version, Roots: runtimeadapter.StageEffortRoots{Repository: repository}.Resolve, Consented: true}
	for _, child := range children {
		for _, effect := range child.Effects {
			if effect.Kind == "repository-write" {
				inspector.Workspaces = append(inspector.Workspaces, child.Workspace)
			}
		}
	}
	return inspector, nil
}

// canonicalV3Configuration reads local configuration through the approved
// zero-write store reader on every observation. Only an active canonical v3
// record is observable; fresh, retired, pre-stable, future, corrupt, unsafe
// and pending states are blocked, with no compatibility fallback or migration.
func canonicalV3Configuration(store local.RuntimeProfileStore) func(context.Context) (runtimeprofile.VersionedConfiguration, error) {
	return func(ctx context.Context) (runtimeprofile.VersionedConfiguration, error) {
		snapshot, err := store.Inspect(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return runtimeprofile.VersionedConfiguration{}, ctx.Err()
			}
			return runtimeprofile.VersionedConfiguration{}, capabilityproof.ErrBlocked
		}
		if snapshot.State != "active-v3" || snapshot.V3 == nil {
			return runtimeprofile.VersionedConfiguration{}, capabilityproof.ErrBlocked
		}
		v3 := *snapshot.V3
		return runtimeprofile.VersionedConfiguration{FormatVersion: runtimeprofile.FormatVersionV3, V3: &v3}, nil
	}
}
