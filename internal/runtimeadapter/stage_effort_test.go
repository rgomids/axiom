//go:build unix

package runtimeadapter

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/capabilityproof"
	"github.com/rgomids/axiom/internal/executiongraph"
	"github.com/rgomids/axiom/internal/runtimeprofile"
)

// Specification 007 AXM-22 PM-13: stable host/repository effort roots with
// per-attempt child equivalence. Synthetic files only; nothing is executed,
// created outside the test root or repaired.

func TestStageEffortRootsResolve(t *testing.T) {
	ctx := context.Background()
	repo := filepath.Join(t.TempDir(), "repository")
	write := capabilityproof.Target{RuntimeID: "codex", ProfileID: "p", Variant: capabilityproof.RepositoryWriteVariant}
	managed := filepath.Dir(DefaultManagedConfiguration().CodexConfig[0])
	roots, err := StageEffortRoots{Repository: repo}.Resolve(ctx, write, []string{"HOME=/synthetic/home", "PATH=/usr/bin"})
	if err != nil || roots != (EffortSourceRoots{User: "/synthetic/home", Project: repo, Managed: managed}) {
		t.Fatalf("roots %+v %v", roots, err)
	}
	claude := write
	claude.RuntimeID = "claude"
	if roots, err := (StageEffortRoots{Repository: repo}).Resolve(ctx, claude, []string{"HOME=/synthetic/home"}); err != nil || roots.Managed != filepath.Dir(DefaultManagedConfiguration().ClaudeSettings[0]) {
		t.Fatalf("claude managed root %+v %v", roots, err)
	}
	refusals := map[string]struct {
		target capabilityproof.Target
		repo   string
		env    []string
	}{
		"read-only-child":   {capabilityproof.Target{RuntimeID: "codex", ProfileID: "p", Variant: capabilityproof.ReadOnlyVariant}, repo, []string{"HOME=/h"}},
		"missing-home":      {write, repo, []string{"PATH=/usr/bin"}},
		"relative-home":     {write, repo, []string{"HOME=h"}},
		"unclean-home":      {write, repo, []string{"HOME=/h/../x"}},
		"root-home":         {write, repo, []string{"HOME=/"}},
		"duplicate-home":    {write, repo, []string{"HOME=/h", "HOME=/i"}},
		"case-alias-home":   {write, repo, []string{"home=/h"}},
		"codex-home":        {write, repo, []string{"HOME=/h", "CODEX_HOME=/h/.codex"}},
		"claude-config-dir": {claude, repo, []string{"HOME=/h", "claude_config_dir=/h/.claude"}},
		"unknown-runtime":   {capabilityproof.Target{RuntimeID: "other", ProfileID: "p", Variant: capabilityproof.RepositoryWriteVariant}, repo, []string{"HOME=/h"}},
		"relative-repo":     {write, "repository", []string{"HOME=/h"}},
		"root-repo":         {write, "/", []string{"HOME=/h"}},
	}
	for name, c := range refusals {
		if _, err := (StageEffortRoots{Repository: c.repo}).Resolve(ctx, c.target, c.env); !errors.Is(err, ErrEffortConfigurationUnproven) {
			t.Fatalf("%s: %v", name, err)
		}
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := (StageEffortRoots{Repository: repo}).Resolve(canceled, write, []string{"HOME=/h"}); err == nil {
		t.Fatal("canceled resolution")
	}
}

// A missing managed root is an absence fact only under a trusted nearest
// existing ancestor; absence is part of the identity, revalidated, and never
// repaired. User and project roots must exist.
func TestTrustedEffortAnchorAbsentManaged(t *testing.T) {
	ctx := context.Background()
	for _, runtime := range []string{"codex", "claude"} {
		t.Run(runtime, func(t *testing.T) {
			f := newEffortInspectorFixture(t, runtime)
			present, err := InspectEffortSources(ctx, f.req)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.RemoveAll(f.req.Roots.Managed); err != nil {
				t.Fatal(err)
			}
			absent, err := InspectEffortSources(ctx, f.req)
			if err != nil || absent == present {
				t.Fatalf("absent managed root: %q %v", absent, err)
			}
			if again, err := InspectEffortSources(ctx, f.req); err != nil || again != absent {
				t.Fatalf("absence not stable: %v", err)
			}
			if _, err := os.Lstat(f.req.Roots.Managed); !errors.Is(err, os.ErrNotExist) {
				t.Fatal("inspection created the managed root")
			}
			// A missing intermediate ancestor is still absence.
			f.req.Roots.Managed = filepath.Join(f.base, "opt", "vendor", "managed")
			if _, err := InspectEffortSources(ctx, f.req); err != nil {
				t.Fatalf("missing intermediate ancestor: %v", err)
			}
			if _, err := os.Lstat(filepath.Join(f.base, "opt")); !errors.Is(err, os.ErrNotExist) {
				t.Fatal("inspection created an ancestor")
			}
			// The root appearing later is another identity.
			f.req.Roots.Managed = filepath.Join(f.base, "managed")
			if err := os.Mkdir(f.req.Roots.Managed, 0o700); err != nil {
				t.Fatal(err)
			}
			if appeared, err := InspectEffortSources(ctx, f.req); err != nil || appeared == absent {
				t.Fatalf("appeared root reused the absent identity: %v", err)
			}
		})
	}
	unsafe := map[string]func(*testing.T, *effortInspectorFixture){
		// Sticky does not excuse a writable nearest ancestor of a missing root.
		"sticky-writable-parent": func(t *testing.T, f *effortInspectorFixture) {
			parent := filepath.Join(f.base, "shared")
			if err := os.Mkdir(parent, 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.Chmod(parent, 0o1777); err != nil {
				t.Fatal(err)
			}
			f.req.Roots.Managed = filepath.Join(parent, "managed")
		},
		"group-writable-parent": func(t *testing.T, f *effortInspectorFixture) {
			parent := filepath.Join(f.base, "group")
			if err := os.Mkdir(parent, 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.Chmod(parent, 0o770); err != nil {
				t.Fatal(err)
			}
			f.req.Roots.Managed = filepath.Join(parent, "managed")
		},
		"symlinked-parent": func(t *testing.T, f *effortInspectorFixture) {
			real := filepath.Join(f.base, "real")
			if err := os.Mkdir(real, 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(real, filepath.Join(f.base, "link")); err != nil {
				t.Fatal(err)
			}
			f.req.Roots.Managed = filepath.Join(f.base, "link", "managed")
		},
		"missing-user": func(t *testing.T, f *effortInspectorFixture) {
			os.RemoveAll(f.req.Roots.User)
		},
		"missing-project": func(t *testing.T, f *effortInspectorFixture) {
			os.RemoveAll(f.req.Roots.Project)
		},
	}
	for name, mutate := range unsafe {
		t.Run(name, func(t *testing.T) {
			f := newEffortInspectorFixture(t, "codex")
			mutate(t, f)
			if _, err := InspectEffortSources(ctx, f.req); !errors.Is(err, ErrEffortConfigurationUnproven) {
				t.Fatalf("%s accepted: %v", name, err)
			}
		})
	}
}

// child mirrors the fixture's project sources into a separate worktree.
func equivalentChild(t *testing.T, f *effortInspectorFixture) string {
	t.Helper()
	child := filepath.Join(f.base, "worktree")
	if err := os.Mkdir(child, 0o700); err != nil {
		t.Fatal(err)
	}
	for _, s := range f.declaration().Sources {
		if s.Anchor != runtimeprofile.EffortAnchorProject {
			continue
		}
		wire, err := os.ReadFile(filepath.Join(f.req.Roots.Project, filepath.FromSlash(s.Path)))
		if err != nil {
			t.Fatal(err)
		}
		writeFixtureFile(t, filepath.Join(child, filepath.FromSlash(s.Path)), string(wire))
	}
	return child
}

func TestVerifyChildEquivalence(t *testing.T) {
	ctx := context.Background()
	for _, runtime := range []string{"codex", "claude"} {
		t.Run(runtime, func(t *testing.T) {
			f := newEffortInspectorFixture(t, runtime)
			child := equivalentChild(t, f)
			if err := VerifyChildEquivalence(ctx, f.req, child); err != nil {
				t.Fatalf("equivalent child: %v", err)
			}
			// Equivalence reads only; the repository-root digest is unchanged.
			before, _ := InspectEffortSources(ctx, f.req)
			if err := VerifyChildEquivalence(ctx, f.req, child); err != nil {
				t.Fatal(err)
			}
			if after, _ := InspectEffortSources(ctx, f.req); after != before {
				t.Fatal("equivalence changed the canonical digest")
			}
		})
	}
	project := func(f *effortInspectorFixture, root string) string {
		for _, s := range f.declaration().Sources {
			if s.Scope == "project" {
				return filepath.Join(root, filepath.FromSlash(s.Path))
			}
		}
		return ""
	}
	cases := map[string]struct {
		runtime string
		mutate  func(*testing.T, *effortInspectorFixture, string)
		want    error
	}{
		"bytes-differ": {"codex", func(t *testing.T, f *effortInspectorFixture, child string) {
			writeFixtureFile(t, project(f, child), "# other non-selector content\n")
		}, ErrEffortConfigurationUnproven},
		"missing-in-child": {"codex", func(t *testing.T, f *effortInspectorFixture, child string) { os.Remove(project(f, child)) }, ErrEffortConfigurationUnproven},
		"missing-in-repository": {"codex", func(t *testing.T, f *effortInspectorFixture, child string) {
			os.Remove(project(f, f.req.Roots.Project))
		}, ErrEffortConfigurationUnproven},
		"child-selector": {"codex", func(t *testing.T, f *effortInspectorFixture, child string) {
			writeFixtureFile(t, project(f, child), "model_reasoning_effort = \"low\"\n")
		}, ErrEffortOverrideConflict},
		"child-symlink": {"codex", func(t *testing.T, f *effortInspectorFixture, child string) {
			path := project(f, child)
			os.Remove(path)
			os.Symlink(project(f, f.req.Roots.Project), path)
		}, ErrEffortConfigurationUnproven},
		"child-unsafe-root": {"codex", func(t *testing.T, f *effortInspectorFixture, child string) { os.Chmod(child, 0o777) }, ErrEffortConfigurationUnproven},
		"child-missing":     {"codex", func(t *testing.T, f *effortInspectorFixture, child string) { os.RemoveAll(child) }, ErrEffortConfigurationUnproven},
		"project-local-only-in-repository": {"claude", func(t *testing.T, f *effortInspectorFixture, child string) {
			os.Remove(filepath.Join(child, ".claude", "settings.local.json"))
		}, ErrEffortConfigurationUnproven},
		"managed-drift-revalidated": {"claude", func(t *testing.T, f *effortInspectorFixture, child string) {
			writeFixtureFile(t, filepath.Join(f.req.Roots.Managed, "managed-settings.d", "20-other.json"), `{"fixtureEffortKey":"high"}`)
		}, ErrEffortOverrideConflict},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			f := newEffortInspectorFixture(t, c.runtime)
			child := equivalentChild(t, f)
			c.mutate(t, f, child)
			if err := VerifyChildEquivalence(ctx, f.req, child); !errors.Is(err, c.want) {
				t.Fatalf("%s: %v, want %v", name, err, c.want)
			}
		})
	}
	f := newEffortInspectorFixture(t, "codex")
	for name, child := range map[string]string{"repository-itself": f.req.Roots.Project, "user-root": f.req.Roots.User, "relative": "worktree", "unclean": f.base + "/x/../worktree"} {
		if err := VerifyChildEquivalence(ctx, f.req, child); !errors.Is(err, ErrEffortConfigurationUnproven) {
			t.Fatalf("%s: %v", name, err)
		}
	}
	f.req.Consented = false
	if err := VerifyChildEquivalence(ctx, f.req, equivalentChild(t, f)); !errors.Is(err, ErrEffortConfigurationUnproven) {
		t.Fatalf("unconsented equivalence: %v", err)
	}
}

type stageInspectorFixture struct {
	*effortInspectorFixture
	child     string
	inspector StageEffortInspector
	profile   runtimeprofile.ResolvedProfile
	inv       executiongraph.Invocation
	env       []string
}

func newStageInspectorFixture(t *testing.T, runtime string) *stageInspectorFixture {
	t.Helper()
	f := &stageInspectorFixture{effortInspectorFixture: newEffortInspectorFixture(t, runtime)}
	f.child = equivalentChild(t, f.effortInspectorFixture)
	_, f.profile = f.legacy()
	f.inv = f.req.Invocation
	f.inv.CWD = f.child
	f.inv.Env = []string{"HOME=" + f.req.Roots.User, "PATH=fixture"}
	f.inspector = StageEffortInspector{
		Configuration: func(context.Context) (runtimeprofile.VersionedConfiguration, error) { return f.req.Configuration, nil },
		Version: func(_ context.Context, id, digest string) (string, error) {
			if id != runtime || digest != f.req.Observed.ExecutableDigest {
				return "", capabilityproof.ErrBlocked
			}
			return f.req.Observed.RuntimeVersion, nil
		},
		Roots: func(_ context.Context, target capabilityproof.Target, env []string) (EffortSourceRoots, error) {
			f.env = env
			if target.Variant != capabilityproof.RepositoryWriteVariant || target.RuntimeID != runtime {
				return EffortSourceRoots{}, ErrEffortConfigurationUnproven
			}
			return f.req.Roots, nil
		},
		Workspaces: []string{f.child},
		Consented:  true,
	}
	return f
}

// The dispatch-time inspector returns the repository-root canonical digest for
// an equivalent repository-write child, recomputed on every call.
func TestStageEffortInspector(t *testing.T) {
	ctx := context.Background()
	for _, runtime := range []string{"codex", "claude"} {
		t.Run(runtime, func(t *testing.T) {
			f := newStageInspectorFixture(t, runtime)
			want, err := InspectEffortSources(ctx, f.req)
			if err != nil {
				t.Fatal(err)
			}
			got, err := f.inspector.CheckEffortSources(ctx, f.profile, f.inv)
			if err != nil || got != want {
				t.Fatalf("digest %q %v, want %q", got, err, want)
			}
			if strings.Join(f.env, ",") != strings.Join(executiongraph.EffectiveInvocationEnvironment(f.inv), ",") {
				t.Fatal("roots not resolved from the attempt's effective environment")
			}
			// Drift in the child between attempts is caught on the next call.
			writeFixtureFile(t, filepath.Join(f.child, filepath.FromSlash(f.declaration().Sources[1].Path)), "# drift\n")
			if _, err := f.inspector.CheckEffortSources(ctx, f.profile, f.inv); err == nil {
				t.Fatal("child drift admitted")
			}
		})
	}
	cases := map[string]func(*testing.T, *stageInspectorFixture){
		"not-consented":    func(_ *testing.T, f *stageInspectorFixture) { f.inspector.Consented = false },
		"read-only-child":  func(_ *testing.T, f *stageInspectorFixture) { f.inspector.Workspaces = nil },
		"unreviewed-cwd":   func(_ *testing.T, f *stageInspectorFixture) { f.inv.CWD = f.base },
		"no-roots":         func(_ *testing.T, f *stageInspectorFixture) { f.inspector.Roots = nil },
		"invalid-profile":  func(_ *testing.T, f *stageInspectorFixture) { f.profile.RuntimeID = "" },
		"runtime-mismatch": func(_ *testing.T, f *stageInspectorFixture) { f.inv.RuntimeID = "claude" },
		"executable-drift": func(t *testing.T, f *stageInspectorFixture) {
			os.Remove(f.exe)
			writeFixtureFile(t, f.exe, "replaced; never execute\n")
		},
		"version-mismatch": func(_ *testing.T, f *stageInspectorFixture) { f.profile.RuntimeVersion = "synthetic-codex-2" },
		"configuration-unavailable": func(_ *testing.T, f *stageInspectorFixture) {
			f.inspector.Configuration = func(context.Context) (runtimeprofile.VersionedConfiguration, error) {
				return runtimeprofile.VersionedConfiguration{}, capabilityproof.ErrBlocked
			}
		},
		"roots-refused": func(_ *testing.T, f *stageInspectorFixture) {
			f.inspector.Roots = func(context.Context, capabilityproof.Target, []string) (EffortSourceRoots, error) {
				return EffortSourceRoots{}, ErrEffortConfigurationUnproven
			}
		},
		"child-not-equivalent": func(t *testing.T, f *stageInspectorFixture) {
			os.Remove(filepath.Join(f.child, filepath.FromSlash(f.declaration().Sources[1].Path)))
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			f := newStageInspectorFixture(t, "codex")
			mutate(t, f)
			if got, err := f.inspector.CheckEffortSources(ctx, f.profile, f.inv); err == nil || got != "" {
				t.Fatalf("%s admitted: %q", name, got)
			}
		})
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	f := newStageInspectorFixture(t, "codex")
	if _, err := f.inspector.CheckEffortSources(canceled, f.profile, f.inv); err == nil {
		t.Fatal("canceled inspection")
	}
}

// Preview (BindingSource over the repository root) and dispatch (inspector
// over an equivalent child worktree) observe the same EffortSourceDigest.
func TestStageEffortPreviewDispatchAgreement(t *testing.T) {
	ctx := context.Background()
	for _, runtime := range []string{"codex", "claude"} {
		t.Run(runtime, func(t *testing.T) {
			f := newBindingFixture(t, runtime)
			target := f.target(capabilityproof.RepositoryWriteVariant, false)
			f.seed(t, target)
			c, err := f.source.Current(ctx, target)
			if err != nil {
				t.Fatal(err)
			}
			child := filepath.Join(f.base, "child-worktree")
			for _, s := range f.cfg.Runtimes[0].EffortDeclarations[0].Sources {
				if s.Anchor != runtimeprofile.EffortAnchorProject {
					continue
				}
				wire, err := os.ReadFile(filepath.Join(f.roots.Project, filepath.FromSlash(s.Path)))
				if err != nil {
					t.Fatal(err)
				}
				writeFixtureFile(t, filepath.Join(child, filepath.FromSlash(s.Path)), string(wire))
			}
			os.Chmod(child, 0o700)
			profile := f.cfg.ModelProfiles[0]
			hash := strings.Repeat("a", 64)
			resolved := runtimeprofile.ResolvedProfile{ProfileID: profile.ID, RuntimeID: runtime, Model: profile.Model, Effort: profile.Effort, ProfileDigest: hash, ConfigurationRevision: f.cfg.Revision, ConfigurationDigest: hash, ExecutableDigest: c.Binding.ExecutableDigest, RuntimeVersion: f.version, InvocationMode: "noninteractive", ObservationRevision: 1, ObservationDigest: hash, EffortSourceDigest: c.Model.EffortSourceDigest, CapabilityProofReference: "synthetic-proof"}
			request := StageInvocationRequest{Profile: CommandProfile{RuntimeID: runtime, ModelProfileID: profile.ID, Executable: f.exe, Model: profile.Model}, Version: f.version, Workspace: child, Inbox: filepath.Join(f.base, "inbox"), AxiomPath: f.axiom, RepositoryWrite: true, ParentEnvironment: []string{"PATH=fixture"}}
			inv, err := BuildFrozenStageInvocation(request, resolved)
			if err != nil {
				t.Fatal(err)
			}
			cfg := decodeV3Fixture(t, f.cfg)
			inspector := StageEffortInspector{
				Configuration: func(context.Context) (runtimeprofile.VersionedConfiguration, error) { return cfg, nil },
				Version:       func(context.Context, string, string) (string, error) { return f.version, nil },
				Roots: func(context.Context, capabilityproof.Target, []string) (EffortSourceRoots, error) {
					return f.roots, nil
				},
				Workspaces: []string{child},
				Consented:  true,
			}
			got, err := inspector.CheckEffortSources(ctx, resolved, inv)
			if err != nil || got != c.Model.EffortSourceDigest {
				t.Fatalf("dispatch digest %q %v, preview %q", got, err, c.Model.EffortSourceDigest)
			}
		})
	}
}
