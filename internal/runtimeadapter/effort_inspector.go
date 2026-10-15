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
	wire, _, err := inspectEffortDeclaration(ctx, r)
	return wire, err
}

// inspectEffortDeclaration also returns the resolved declaration so the child
// equivalence check (PM-13) reads exactly the inspected sources.
func inspectEffortDeclaration(ctx context.Context, r EffortSourceRequest) ([]byte, EffortDeclaration, error) {
	var none EffortDeclaration
	if ctx.Err() != nil || !r.Consented {
		return nil, none, ErrEffortConfigurationUnproven
	}
	// Selection validates the whole v3 configuration (version, uniqueness,
	// scopes, anchors, precedence) and matches only the exact observation.
	configured, err := runtimeprofile.ApplicableEffortDeclaration(r.Configuration, r.Observed)
	if err != nil {
		return nil, none, ErrEffortConfigurationUnproven
	}
	profile, ok := effortProfile(r.Configuration.V3, r.ProfileID, r.Observed.RuntimeID)
	if !ok {
		return nil, none, ErrEffortConfigurationUnproven
	}
	subject := effortSubject{r.Observed.RuntimeID, r.Observed.RuntimeVersion, r.Observed.InvocationMode, r.Observed.ExecutableDigest, profile.Model, profile.Effort}
	if !validEffortSubject(subject) {
		return nil, none, ErrEffortConfigurationUnproven
	}
	declaration := EffortDeclaration{RuntimeID: subject.RuntimeID, RuntimeVersion: configured.RuntimeVersion, InvocationMode: configured.InvocationMode, ExecutableDigest: configured.ExecutableDigest, ProofReference: configured.ProofReference(), PrecedenceProven: configured.PrecedenceEstablished, JSONKeys: configured.JSONKeys, EnvironmentKeys: configured.EnvironmentKeys}
	roots := map[string]string{runtimeprofile.EffortAnchorUser: r.Roots.User, runtimeprofile.EffortAnchorProject: r.Roots.Project, runtimeprofile.EffortAnchorManaged: r.Roots.Managed}
	bindings := map[string]string{}
	for _, s := range configured.Sources {
		root, ok := roots[s.Anchor]
		if !ok || !runtimeprofile.ValidEffortSourcePath(s.Path) {
			return nil, none, ErrEffortConfigurationUnproven
		}
		if _, seen := bindings[s.Anchor]; !seen {
			binding, err := trustedEffortAnchor(s.Anchor, root)
			if err != nil {
				return nil, none, ErrEffortConfigurationUnproven
			}
			bindings[s.Anchor] = binding
		}
		declaration.Sources = append(declaration.Sources, EffortSource{Scope: s.Scope, Root: root, Relative: filepath.FromSlash(s.Path), Format: s.Format, Directory: s.Directory})
	}
	wire, err := effortSourceProjection(ctx, declaration, subject, r.Invocation)
	if err != nil {
		return nil, none, err
	}
	// A root replaced or re-permissioned during inspection is not the root the
	// digest observed.
	for anchor, before := range bindings {
		after, err := trustedEffortAnchor(anchor, roots[anchor])
		if err != nil || after != before {
			return nil, none, ErrEffortConfigurationUnproven
		}
	}
	if ctx.Err() != nil {
		return nil, none, ErrEffortConfigurationUnproven
	}
	return wire, declaration, nil
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

// trustedEffortAnchor applies trustedEffortRoot, except that a missing
// managed root is an absence fact (Specification 007 PM-13): its nearest
// existing ancestor must be a safe anchor-grade directory no other principal
// can create entries in, every existing ancestor above it must be safe, and
// its absent binding is revalidated like a present one. Nothing is created.
// User and project roots must exist.
func trustedEffortAnchor(anchor, root string) (string, error) {
	if anchor != runtimeprofile.EffortAnchorManaged {
		return trustedEffortRoot(root)
	}
	if _, err := os.Lstat(root); !os.IsNotExist(err) {
		return trustedEffortRoot(root)
	}
	if !filepath.IsAbs(root) || filepath.Clean(root) != root || root == string(filepath.Separator) {
		return "", ErrEffortConfigurationUnproven
	}
	binding, present, err := effortPathBinding(root)
	if err != nil || present {
		return "", ErrEffortConfigurationUnproven
	}
	nearest := true
	for current := filepath.Dir(root); ; current = filepath.Dir(current) {
		info, err := os.Lstat(current)
		switch {
		case os.IsNotExist(err) && nearest:
		case err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0:
			return "", ErrEffortConfigurationUnproven
		case nearest && !effortRootSafe(info), !nearest && !effortAncestorSafe(info):
			return "", ErrEffortConfigurationUnproven
		default:
			nearest = false
		}
		if current == filepath.Dir(current) {
			if nearest {
				return "", ErrEffortConfigurationUnproven
			}
			return "absent\x00" + binding, nil
		}
	}
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
