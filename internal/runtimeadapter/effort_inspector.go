package runtimeadapter

import (
	"context"
	"os"
	"path/filepath"

	"github.com/rgomids/axiom/internal/executiongraph"
	"github.com/rgomids/axiom/internal/runtimeprofile"
)

// EffortSourceRoots are the trusted current local roots of the closed
// configuration anchors (Specification 007 AXM-22 BI-2a D6). They are host
// observations supplied by the caller, never read from persisted configuration.
type EffortSourceRoots struct {
	User, Project, Managed string
}

// EffortSourceRequest carries only independently supplied facts: a validated
// version-dispatched configuration, the Model Profile whose effort intent is
// inspected, the observed Runtime/version/executable/mode, trusted anchor
// roots and the effective invocation. It requires no ProfileProof, capability
// Evidence or BindingSource output (BI-2 item 4).
type EffortSourceRequest struct {
	Consented     bool
	Configuration runtimeprofile.VersionedConfiguration
	ProfileID     string
	Observed      runtimeprofile.EffortApplicability
	Roots         EffortSourceRoots
	Invocation    executiongraph.Invocation
}

// InspectEffortSources returns the canonical EffortSourceDigest for the one
// configured declaration applicable to the observed identity. It reuses the
// PE-05a projection and digest exactly; the declaration is configuration
// identity, never capability proof. Every refusal is
// effort_configuration_unproven or the existing effort_override_conflict.
func InspectEffortSources(ctx context.Context, r EffortSourceRequest) (string, error) {
	wire, err := inspectEffortProjection(ctx, r)
	if err != nil {
		return "", err
	}
	return effortHash(wire), nil
}
func inspectEffortProjection(ctx context.Context, r EffortSourceRequest) ([]byte, error) {
	if ctx.Err() != nil || !r.Consented {
		return nil, ErrEffortConfigurationUnproven
	}
	// Selection validates the whole v3 configuration (version, uniqueness,
	// scopes, anchors, precedence) and matches only the exact observation.
	configured, err := runtimeprofile.ApplicableEffortDeclaration(r.Configuration, r.Observed)
	if err != nil {
		return nil, ErrEffortConfigurationUnproven
	}
	profile, ok := effortProfile(r.Configuration.V3, r.ProfileID, r.Observed.RuntimeID)
	if !ok {
		return nil, ErrEffortConfigurationUnproven
	}
	subject := effortSubject{r.Observed.RuntimeID, r.Observed.RuntimeVersion, r.Observed.InvocationMode, r.Observed.ExecutableDigest, profile.Model, profile.Effort}
	if !validEffortSubject(subject) {
		return nil, ErrEffortConfigurationUnproven
	}
	declaration := EffortDeclaration{RuntimeID: subject.RuntimeID, RuntimeVersion: configured.RuntimeVersion, InvocationMode: configured.InvocationMode, ExecutableDigest: configured.ExecutableDigest, ProofReference: configured.ProofReference(), PrecedenceProven: configured.PrecedenceEstablished, JSONKeys: configured.JSONKeys, EnvironmentKeys: configured.EnvironmentKeys}
	roots := map[string]string{runtimeprofile.EffortAnchorUser: r.Roots.User, runtimeprofile.EffortAnchorProject: r.Roots.Project, runtimeprofile.EffortAnchorManaged: r.Roots.Managed}
	bindings := map[string]string{}
	for _, s := range configured.Sources {
		root, ok := roots[s.Anchor]
		if !ok || !runtimeprofile.ValidEffortSourcePath(s.Path) {
			return nil, ErrEffortConfigurationUnproven
		}
		if _, seen := bindings[s.Anchor]; !seen {
			binding, err := trustedEffortRoot(root)
			if err != nil {
				return nil, ErrEffortConfigurationUnproven
			}
			bindings[s.Anchor] = binding
		}
		declaration.Sources = append(declaration.Sources, EffortSource{Scope: s.Scope, Root: root, Relative: filepath.FromSlash(s.Path), Format: s.Format, Directory: s.Directory})
	}
	wire, err := effortSourceProjection(ctx, declaration, subject, r.Invocation)
	if err != nil {
		return nil, err
	}
	// A root replaced or re-permissioned during inspection is not the root the
	// digest observed.
	for anchor, before := range bindings {
		after, err := trustedEffortRoot(roots[anchor])
		if err != nil || after != before {
			return nil, ErrEffortConfigurationUnproven
		}
	}
	if ctx.Err() != nil {
		return nil, ErrEffortConfigurationUnproven
	}
	return wire, nil
}

func effortProfile(c *runtimeprofile.ConfigurationV3, id, runtimeID string) (runtimeprofile.ModelProfile, bool) {
	if c == nil {
		return runtimeprofile.ModelProfile{}, false
	}
	for _, p := range c.ModelProfiles {
		if p.ID == id {
			return p, p.RuntimeID == runtimeID
		}
	}
	return runtimeprofile.ModelProfile{}, false
}

// validEffortSubject is the subset of ValidateResolvedProfile that constrains
// the digest subject; proof-derived fields are intentionally absent.
func validEffortSubject(s effortSubject) bool {
	return runtimeprofile.SafeIdentifier(s.RuntimeID) && runtimeprofile.SafeIdentifier(s.Model) && runtimeprofile.ValidEffort(s.Effort) && runtimeprofile.SafeRuntimeVersion(s.RuntimeVersion) && s.InvocationMode == "noninteractive"
}

// trustedEffortRoot requires an existing, canonical, non-symlinked anchor
// directory whose path and identity bind through the PE-05a boundary, owned
// by the current user or root without group/other write, below ancestors that
// no other principal can replace. Nothing is created or repaired.
func trustedEffortRoot(root string) (string, error) {
	if !filepath.IsAbs(root) || filepath.Clean(root) != root || root == string(filepath.Separator) {
		return "", ErrEffortConfigurationUnproven
	}
	binding, present, err := effortPathBinding(root)
	if err != nil || !present {
		return "", ErrEffortConfigurationUnproven
	}
	info, err := os.Lstat(root)
	if err != nil || !info.IsDir() || !effortRootSafe(info) {
		return "", ErrEffortConfigurationUnproven
	}
	for current := filepath.Dir(root); ; current = filepath.Dir(current) {
		ancestor, err := os.Lstat(current)
		if err != nil || !ancestor.IsDir() || !effortAncestorSafe(ancestor) {
			return "", ErrEffortConfigurationUnproven
		}
		if current == filepath.Dir(current) {
			return binding, nil
		}
	}
}
