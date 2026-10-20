package runtimeadapter

import (
	"context"
	"errors"
	"path/filepath"
	"time"

	"github.com/rgomids/axiom/internal/executiongraph"
	"github.com/rgomids/axiom/internal/runtimeprofile"
)

var ErrInspectionConsentRequired = runtimeprofile.ErrInspectionConsentRequired

const (
	effortRoleWorkspace = "/axiom-role/workspace"
	effortRoleInbox     = "/axiom-role/inbox"
	effortRoleAxiom     = "/axiom-role/axiom"
)

// OperationalSource observes the current operational facts of one Runtime for
// policy resolution: the executable resolved for its adapter (path and bytes)
// and, with explicit inspection consent, the effort sources of the
// repository-root view. It never runs a Runtime, reads credentials, consults
// stored Evidence or asserts vendor support.
type OperationalSource struct {
	Configuration runtimeprofile.ConfigurationV4
	LookPath      func(string) (string, error)
	Repository    string
	Environment   []string
	Consented     bool
	ObservedAt    time.Time
}

func (s OperationalSource) ObserveExecutable(ctx context.Context, runtimeID string) (runtimeprofile.OperationalObservation, error) {
	o := runtimeprofile.OperationalObservation{RuntimeID: runtimeID, Adapter: runtimeID, Revision: s.Configuration.Revision, ObservedAt: s.ObservedAt.UTC(), InvocationMode: "noninteractive"}
	if err := ctx.Err(); err != nil {
		return o, err
	}
	if runtimeID != "codex" && runtimeID != "claude" || s.LookPath == nil || s.ObservedAt.IsZero() {
		return o, ErrInvalidAdapterConfiguration
	}
	path, err := s.LookPath(runtimeID)
	if err != nil || !filepath.IsAbs(path) {
		return o, ErrExecutableUnavailable
	}
	identity, err := ExecutableIdentity(path)
	if err != nil {
		return o, ErrExecutableUnavailable
	}
	o.Installed, o.Available, o.ExecutableDigest = true, true, identity
	return o, nil
}

// ObserveOperational adds the Profile's canonical effort-source digest to the
// executable identity. The digest binds the adapter rule, Profile intent,
// executable and sanitized source facts; never a Runtime version or proof.
// Workspace, inbox and Axiom paths of the inspected invocation are role
// placeholders that the projection excludes, so the digest equals the
// repository-root view observed before each real attempt.
func (s OperationalSource) ObserveOperational(ctx context.Context, runtimeID, profileID string) (runtimeprofile.OperationalObservation, error) {
	o, err := s.ObserveExecutable(ctx, runtimeID)
	if err != nil {
		return o, err
	}
	if !s.Consented {
		return o, ErrInspectionConsentRequired
	}
	consumer, err := runtimeprofile.NewConsumerV4(s.Configuration)
	if err != nil {
		return o, ErrEffortConfigurationUnsafe
	}
	profile, err := consumer.AllowedProfile(runtimeID, profileID)
	if err != nil {
		return o, err
	}
	path, err := s.LookPath(runtimeID)
	if err != nil {
		return o, ErrExecutableUnavailable
	}
	inv, err := effortInvocation(runtimeID, path, profile, s.Environment)
	if err != nil {
		return o, err
	}
	roots, err := (StageEffortRoots{Repository: s.Repository}).ResolveOperational(ctx, inv, true)
	if err != nil {
		return o, err
	}
	digest, err := InspectEffortSourcesV4(ctx, EffortSourceRequestV4{Consented: true, Configuration: s.Configuration, ProfileID: profileID, ExecutableDigest: o.ExecutableDigest, InvocationMode: o.InvocationMode, Roots: roots, Invocation: inv})
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return o, ctxErr
		}
		if errors.Is(err, ErrExecutableUnavailable) {
			return o, err
		}
		return o, operationalEffortError(err)
	}
	o.EffortSourceDigest = digest
	return o, nil
}

func effortInvocation(runtimeID, executable string, profile runtimeprofile.ModelProfile, environment []string) (executiongraph.Invocation, error) {
	inv, err := BuildOperationalStageInvocation(StageInvocationRequest{Profile: CommandProfile{RuntimeID: runtimeID, ModelProfileID: profile.ID, Executable: executable, Model: profile.Model}, Workspace: effortRoleWorkspace, Inbox: effortRoleInbox, AxiomPath: effortRoleAxiom, RepositoryWrite: true, ParentEnvironment: environment})
	if err != nil {
		return executiongraph.Invocation{}, err
	}
	return withStageEffort(inv, profile.Effort)
}
