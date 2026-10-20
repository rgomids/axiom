package runtimeadapter

import (
	"bytes"
	"context"
	"encoding/json"
	"path/filepath"
	"reflect"
	"slices"
	"strings"

	"github.com/rgomids/axiom/internal/capabilityproof"
	"github.com/rgomids/axiom/internal/executiongraph"
	"github.com/rgomids/axiom/internal/runtimeprofile"
)

// StageEffortRoots resolves the effort-source anchor roots of one Stage
// operation (Specification 007 AXM-22 PM-13). Roots are stable host and
// repository locations, never persisted: user is the HOME of the effective
// child environment, managed the fixed platform directory of the Runtime,
// project the registered repository root. Only repository-write children are
// admitted; read-only children stay blocked until Runtime-version evidence
// establishes how their working root resolves project configuration.
type StageEffortRoots struct {
	Repository string
}

func (r StageEffortRoots) Resolve(ctx context.Context, t capabilityproof.Target, environment []string) (EffortSourceRoots, error) {
	var none EffortSourceRoots
	if ctx.Err() != nil || t.Variant != capabilityproof.RepositoryWriteVariant || !stagePath(r.Repository) || r.Repository == string(filepath.Separator) {
		return none, ErrEffortConfigurationUnproven
	}
	home, ok := effectiveHome(environment)
	if !ok {
		return none, ErrEffortConfigurationUnproven
	}
	managed, ok := managedEffortRoot(t.RuntimeID)
	if !ok {
		return none, ErrEffortConfigurationUnproven
	}
	return EffortSourceRoots{User: home, Project: r.Repository, Managed: managed}, nil
}

// effectiveHome is the child's own HOME, exactly once and clean. A vendor
// configuration-directory override relocates the user source to a path the
// declaration does not name, so its presence refuses instead of being
// reinterpreted.
func effectiveHome(environment []string) (string, bool) {
	home, homes := "", 0
	for _, entry := range environment {
		key, value, _ := strings.Cut(entry, "=")
		switch {
		case strings.EqualFold(key, "HOME"):
			home, homes = value, homes+1
			if key != "HOME" {
				return "", false
			}
		case strings.EqualFold(key, "CODEX_HOME"), strings.EqualFold(key, "CLAUDE_CONFIG_DIR"):
			return "", false
		}
	}
	return home, homes == 1 && stagePath(home) && home != string(filepath.Separator)
}

// managedEffortRoot is the single directory of the Runtime's existing managed
// configuration locations (DefaultManagedConfiguration); no other location is
// inferred.
func managedEffortRoot(runtimeID string) (string, bool) {
	c := DefaultManagedConfiguration()
	paths := map[string][]string{"codex": c.CodexConfig, "claude": c.ClaudeSettings}[runtimeID]
	root := ""
	for _, p := range paths {
		dir := filepath.Dir(p)
		if root != "" && dir != root {
			return "", false
		}
		root = dir
	}
	return root, stagePath(root)
}

// VerifyChildEquivalence proves that a repository-write child reads the same
// effective configuration that the repository root inspection observed. Both
// views pass the canonical inspection independently; their projections must
// agree in every fact except the location binding of project-anchored
// sources, and every project-anchored source must have the same presence and
// exact bytes. User and managed anchors are the same roots in both views. Any
// difference or uncertainty is effort_configuration_unproven (or the existing
// effort_override_conflict), never a merge.
func VerifyChildEquivalence(ctx context.Context, r EffortSourceRequest, child string) error {
	if !stagePath(child) || child == r.Roots.Project || child == r.Roots.User || child == r.Roots.Managed {
		return ErrEffortConfigurationUnproven
	}
	repoWire, repoDeclaration, err := inspectEffortDeclaration(ctx, r)
	if err != nil {
		return err
	}
	childRequest := r
	childRequest.Roots.Project = child
	childWire, childDeclaration, err := inspectEffortDeclaration(ctx, childRequest)
	if err != nil {
		return err
	}
	if len(repoDeclaration.Sources) != len(childDeclaration.Sources) {
		return ErrEffortConfigurationUnproven
	}
	projectScopes := map[string]bool{}
	for i, source := range repoDeclaration.Sources {
		mirror := childDeclaration.Sources[i]
		if source.Root != r.Roots.Project {
			if !reflect.DeepEqual(source, mirror) {
				return ErrEffortConfigurationUnproven
			}
			continue
		}
		// Project-anchored drop-in directories are not a recognized source.
		if source.Directory || mirror.Root != child || mirror.Scope != source.Scope || mirror.Relative != source.Relative {
			return ErrEffortConfigurationUnproven
		}
		projectScopes[source.Scope] = true
		repoBytes, repoPresent, _, repoErr := readEffortSource(source)
		childBytes, childPresent, _, childErr := readEffortSource(mirror)
		if repoErr != nil || childErr != nil || repoPresent != childPresent || !bytes.Equal(repoBytes, childBytes) {
			return ErrEffortConfigurationUnproven
		}
	}
	repoFacts, err := locationIndependentFacts(repoWire, projectScopes)
	if err != nil {
		return err
	}
	childFacts, err := locationIndependentFacts(childWire, projectScopes)
	if err != nil || !reflect.DeepEqual(repoFacts, childFacts) {
		return ErrEffortConfigurationUnproven
	}
	if ctx.Err() != nil {
		return ErrEffortConfigurationUnproven
	}
	return nil
}

// locationIndependentFacts removes only the location binding of
// project-anchored source facts; the canonical projection itself is unchanged.
func locationIndependentFacts(wire []byte, projectScopes map[string]bool) ([]effortSourceFact, error) {
	var projection struct {
		Proof string             `json:"proof"`
		Facts []effortSourceFact `json:"facts"`
	}
	if json.Unmarshal(wire, &projection) != nil || projection.Proof == "" {
		return nil, ErrEffortConfigurationUnproven
	}
	for i := range projection.Facts {
		if projectScopes[projection.Facts[i].Scope] {
			projection.Facts[i].Binding = ""
		}
	}
	return append([]effortSourceFact{{Scope: "proof", Key: projection.Proof}}, projection.Facts...), nil
}

// StageEffortInspector is the dispatch-time effort-source inspector of the
// Stage supervisor (AXM-36 D4). Immediately before each attempt it re-reads
// canonical configuration 4 and the actual executable, and returns the
// canonical EffortSourceDigest of the repository-root view for comparison
// with the frozen tuple. A reviewed repository-write child must read the same
// project sources as the repository root; a read-only child's actual working
// root is inspected and must hold no competing or unsafe source. Consented
// records the committed Stage dispatch authority; it grants nothing else.
type StageEffortInspector struct {
	Configuration func(context.Context) (runtimeprofile.ConfigurationV4, error)
	Repository    string
	// Workspaces are the reviewed repository-write child workspaces.
	Workspaces []string
	Consented  bool
}

func (i StageEffortInspector) CheckEffortSources(ctx context.Context, p runtimeprofile.ResolvedProfileV4, inv executiongraph.Invocation) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if !i.Consented || i.Configuration == nil || runtimeprofile.ValidateResolvedProfileV4(p) != nil || len(inv.Argv) == 0 || inv.RuntimeID != p.RuntimeID {
		return "", ErrEffortConfigurationUnsafe
	}
	digest, err := ExecutableIdentity(inv.Argv[0])
	if err != nil {
		return "", ErrExecutableUnavailable
	}
	cfg, err := i.Configuration(ctx)
	if err != nil {
		return "", ErrEffortConfigurationUnsafe
	}
	roots, err := (StageEffortRoots{Repository: i.Repository}).ResolveOperational(ctx, inv, true)
	if err != nil {
		return "", err
	}
	r := EffortSourceRequestV4{Consented: true, Configuration: cfg, ProfileID: p.ProfileID, ExecutableDigest: digest, InvocationMode: p.InvocationMode, Roots: roots, Invocation: inv}
	if slices.Contains(i.Workspaces, inv.CWD) {
		err = VerifyChildEquivalenceV4(ctx, r, inv.CWD)
	} else {
		actual := r
		actual.Roots.Project = inv.CWD
		_, err = InspectEffortSourcesV4(ctx, actual)
	}
	if err != nil {
		return "", err
	}
	return InspectEffortSourcesV4(ctx, r)
}
