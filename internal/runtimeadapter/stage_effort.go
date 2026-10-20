package runtimeadapter

import (
	"context"
	"path/filepath"
	"slices"
	"strings"

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
