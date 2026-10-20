//go:build unix

package runtimeadapter

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rgomids/axiom/internal/executiongraph"
	"github.com/rgomids/axiom/internal/runtimeprofile"
)

// Specification 007 AXM-22 PM-13: stable host/repository effort roots with
// per-attempt child equivalence. Synthetic files only; nothing is executed,
// created outside the test root or repaired.

func TestStageEffortRootsResolve(t *testing.T) {
	ctx := context.Background()
	repo := filepath.Join(t.TempDir(), "repository")
	invocation := func(runtimeID, cwd string, env ...string) executiongraph.Invocation {
		return executiongraph.WithExactEnvironment(executiongraph.Invocation{RuntimeID: runtimeID, CWD: cwd}, env)
	}
	child := filepath.Join(repo, "child")
	inbox := filepath.Join(repo, "inbox")
	managed := filepath.Dir(DefaultManagedConfiguration().CodexConfig[0])
	roots, err := StageEffortRoots{Repository: repo}.ResolveOperational(ctx, invocation("codex", child, "HOME=/synthetic/home", "PATH=/usr/bin"), true)
	if err != nil || roots != (EffortSourceRoots{User: "/synthetic/home", Project: repo, Managed: managed}) {
		t.Fatalf("roots %+v %v", roots, err)
	}
	if roots, err := (StageEffortRoots{Repository: repo}).ResolveOperational(ctx, invocation("claude", child, "HOME=/synthetic/home"), true); err != nil || roots.Managed != filepath.Dir(DefaultManagedConfiguration().ClaudeSettings[0]) {
		t.Fatalf("claude managed root %+v %v", roots, err)
	}
	if roots, err := (StageEffortRoots{Repository: repo}).ResolveOperational(ctx, invocation("codex", inbox, "HOME=/h"), false); err != nil || roots.Project != inbox {
		t.Fatalf("read-only child roots %+v %v", roots, err)
	}
	refusals := map[string]struct {
		inv  executiongraph.Invocation
		repo string
	}{
		"missing-home":      {invocation("codex", child, "PATH=/usr/bin"), repo},
		"relative-home":     {invocation("codex", child, "HOME=h"), repo},
		"unclean-home":      {invocation("codex", child, "HOME=/h/../x"), repo},
		"root-home":         {invocation("codex", child, "HOME=/"), repo},
		"duplicate-home":    {invocation("codex", child, "HOME=/h", "HOME=/i"), repo},
		"case-alias-home":   {invocation("codex", child, "home=/h"), repo},
		"codex-home":        {invocation("codex", child, "HOME=/h", "CODEX_HOME=/h/.codex"), repo},
		"claude-config-dir": {invocation("claude", child, "HOME=/h", "claude_config_dir=/h/.claude"), repo},
		"unknown-runtime":   {invocation("other", child, "HOME=/h"), repo},
		"relative-cwd":      {invocation("codex", "child", "HOME=/h"), repo},
		"root-cwd":          {invocation("codex", "/", "HOME=/h"), repo},
		"relative-repo":     {invocation("codex", child, "HOME=/h"), "repository"},
		"root-repo":         {invocation("codex", child, "HOME=/h"), "/"},
	}
	for name, c := range refusals {
		if _, err := (StageEffortRoots{Repository: c.repo}).ResolveOperational(ctx, c.inv, true); !errors.Is(err, ErrEffortConfigurationUnsafe) {
			t.Fatalf("%s: %v", name, err)
		}
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := (StageEffortRoots{Repository: repo}).ResolveOperational(canceled, invocation("codex", child, "HOME=/h"), true); err == nil {
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
			present, err := InspectEffortSourcesV4(ctx, f.request)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.RemoveAll(f.request.Roots.Managed); err != nil {
				t.Fatal(err)
			}
			absent, err := InspectEffortSourcesV4(ctx, f.request)
			if err != nil || absent == present {
				t.Fatalf("absent managed root: %q %v", absent, err)
			}
			if again, err := InspectEffortSourcesV4(ctx, f.request); err != nil || again != absent {
				t.Fatalf("absence not stable: %v", err)
			}
			if _, err := os.Lstat(f.request.Roots.Managed); !errors.Is(err, os.ErrNotExist) {
				t.Fatal("inspection created the managed root")
			}
			// A missing intermediate ancestor is still absence.
			f.request.Roots.Managed = filepath.Join(f.base, "opt", "vendor", "managed")
			if _, err := InspectEffortSourcesV4(ctx, f.request); err != nil {
				t.Fatalf("missing intermediate ancestor: %v", err)
			}
			if _, err := os.Lstat(filepath.Join(f.base, "opt")); !errors.Is(err, os.ErrNotExist) {
				t.Fatal("inspection created an ancestor")
			}
			// The root appearing later is another identity.
			f.request.Roots.Managed = filepath.Join(f.base, "managed")
			if err := os.Mkdir(f.request.Roots.Managed, 0o700); err != nil {
				t.Fatal(err)
			}
			if appeared, err := InspectEffortSourcesV4(ctx, f.request); err != nil || appeared == absent {
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
			f.request.Roots.Managed = filepath.Join(parent, "managed")
		},
		"group-writable-parent": func(t *testing.T, f *effortInspectorFixture) {
			parent := filepath.Join(f.base, "group")
			if err := os.Mkdir(parent, 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.Chmod(parent, 0o770); err != nil {
				t.Fatal(err)
			}
			f.request.Roots.Managed = filepath.Join(parent, "managed")
		},
		"symlinked-parent": func(t *testing.T, f *effortInspectorFixture) {
			real := filepath.Join(f.base, "real")
			if err := os.Mkdir(real, 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(real, filepath.Join(f.base, "link")); err != nil {
				t.Fatal(err)
			}
			f.request.Roots.Managed = filepath.Join(f.base, "link", "managed")
		},
		"missing-user": func(t *testing.T, f *effortInspectorFixture) {
			os.RemoveAll(f.request.Roots.User)
		},
		"missing-project": func(t *testing.T, f *effortInspectorFixture) {
			os.RemoveAll(f.request.Roots.Project)
		},
	}
	for name, mutate := range unsafe {
		t.Run(name, func(t *testing.T) {
			f := newEffortInspectorFixture(t, "codex")
			mutate(t, f)
			if _, err := InspectEffortSourcesV4(ctx, f.request); !errors.Is(err, ErrEffortConfigurationUnsafe) {
				t.Fatalf("%s accepted: %v", name, err)
			}
		})
	}
}

// equivalentChild mirrors the fixture's project sources into the fixture's
// child worktree, which is the working directory of its invocation.
func equivalentChild(t *testing.T, f *effortInspectorFixture) string {
	t.Helper()
	for _, s := range f.rules().Sources {
		if s.Root != f.request.Roots.Project || s.Directory {
			continue
		}
		wire, err := os.ReadFile(filepath.Join(s.Root, s.Relative))
		if err != nil {
			t.Fatal(err)
		}
		writeFixtureFile(t, filepath.Join(f.child, s.Relative), string(wire))
	}
	return f.child
}

func (f *effortInspectorFixture) projectSource(t *testing.T, root string) string {
	t.Helper()
	for _, s := range f.rules().Sources {
		if s.Scope == "project" {
			return filepath.Join(root, s.Relative)
		}
	}
	t.Fatal("no project source")
	return ""
}

func TestVerifyChildEquivalence(t *testing.T) {
	ctx := context.Background()
	for _, runtime := range []string{"codex", "claude"} {
		t.Run(runtime, func(t *testing.T) {
			f := newEffortInspectorFixture(t, runtime)
			child := equivalentChild(t, f)
			if err := VerifyChildEquivalenceV4(ctx, f.request, child); err != nil {
				t.Fatalf("equivalent child: %v", err)
			}
			// Equivalence reads only; the repository-root digest is unchanged.
			before, _ := InspectEffortSourcesV4(ctx, f.request)
			if err := VerifyChildEquivalenceV4(ctx, f.request, child); err != nil {
				t.Fatal(err)
			}
			if after, _ := InspectEffortSourcesV4(ctx, f.request); after != before {
				t.Fatal("equivalence changed the canonical digest")
			}
		})
	}
	cases := map[string]struct {
		runtime string
		mutate  func(*testing.T, *effortInspectorFixture, string)
		want    error
	}{
		"bytes-differ": {"codex", func(t *testing.T, f *effortInspectorFixture, child string) {
			writeFixtureFile(t, f.projectSource(t, child), "# other non-selector content\n")
		}, ErrEffortConfigurationUnsafe},
		"missing-in-child": {"codex", func(t *testing.T, f *effortInspectorFixture, child string) { os.Remove(f.projectSource(t, child)) }, ErrEffortConfigurationUnsafe},
		"missing-in-repository": {"codex", func(t *testing.T, f *effortInspectorFixture, child string) {
			os.Remove(f.projectSource(t, f.request.Roots.Project))
		}, ErrEffortConfigurationUnsafe},
		"child-selector": {"codex", func(t *testing.T, f *effortInspectorFixture, child string) {
			writeFixtureFile(t, f.projectSource(t, child), "model_reasoning_effort = \"low\"\n")
		}, ErrEffortOverrideConflict},
		"child-symlink": {"codex", func(t *testing.T, f *effortInspectorFixture, child string) {
			path := f.projectSource(t, child)
			os.Remove(path)
			os.Symlink(f.projectSource(t, f.request.Roots.Project), path)
		}, ErrEffortConfigurationUnsafe},
		"child-unsafe-root": {"codex", func(t *testing.T, f *effortInspectorFixture, child string) { os.Chmod(child, 0o777) }, ErrEffortConfigurationUnsafe},
		"child-missing":     {"codex", func(t *testing.T, f *effortInspectorFixture, child string) { os.RemoveAll(child) }, ErrEffortConfigurationUnsafe},
		"project-local-only-in-repository": {"claude", func(t *testing.T, f *effortInspectorFixture, child string) {
			os.Remove(filepath.Join(child, ".claude", "settings.local.json"))
		}, ErrEffortConfigurationUnsafe},
		"managed-drift-revalidated": {"claude", func(t *testing.T, f *effortInspectorFixture, child string) {
			writeFixtureFile(t, filepath.Join(f.request.Roots.Managed, "managed-settings.d", "20-other.json"), `{"effortLevel":"high"}`)
		}, ErrEffortOverrideConflict},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			f := newEffortInspectorFixture(t, c.runtime)
			child := equivalentChild(t, f)
			c.mutate(t, f, child)
			if err := VerifyChildEquivalenceV4(ctx, f.request, child); !errors.Is(err, c.want) {
				t.Fatalf("%s: %v, want %v", name, err, c.want)
			}
		})
	}
	f := newEffortInspectorFixture(t, "codex")
	for name, child := range map[string]string{"repository-itself": f.request.Roots.Project, "user-root": f.request.Roots.User, "managed-root": f.request.Roots.Managed, "relative": "worktree", "unclean": f.base + "/x/../child", "not-the-working-directory": f.base} {
		if err := VerifyChildEquivalenceV4(ctx, f.request, child); !errors.Is(err, ErrEffortConfigurationUnsafe) {
			t.Fatalf("%s: %v", name, err)
		}
	}
	f.request.Consented = false
	if err := VerifyChildEquivalenceV4(ctx, f.request, equivalentChild(t, f)); !errors.Is(err, ErrEffortConfigurationUnsafe) {
		t.Fatalf("unconsented equivalence: %v", err)
	}
}

type stageInspectorFixture struct {
	*effortInspectorFixture
	inspector StageEffortInspector
	frozen    runtimeprofile.ResolvedProfileV4
}

func newStageInspectorFixture(t *testing.T, runtime string) *stageInspectorFixture {
	t.Helper()
	f := &stageInspectorFixture{effortInspectorFixture: newEffortInspectorFixture(t, runtime)}
	equivalentChild(t, f.effortInspectorFixture)
	roots, err := (StageEffortRoots{Repository: f.request.Roots.Project}).ResolveOperational(context.Background(), f.request.Invocation, true)
	if err != nil {
		t.Fatal(err)
	}
	f.request.Roots = roots
	f.frozen = f.operationalEffortFixture.frozen(t)
	f.inspector = StageEffortInspector{
		Configuration: func(context.Context) (runtimeprofile.ConfigurationV4, error) { return f.request.Configuration, nil },
		Repository:    f.request.Roots.Project,
		Workspaces:    []string{f.child},
		Consented:     true,
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
			want, err := InspectEffortSourcesV4(ctx, f.request)
			if err != nil {
				t.Fatal(err)
			}
			got, err := f.inspector.CheckEffortSources(ctx, f.frozen, f.request.Invocation)
			if err != nil || got != want {
				t.Fatalf("digest %q %v, want %q", got, err, want)
			}
			// Drift in the child between attempts is caught on the next call.
			writeFixtureFile(t, f.projectSource(t, f.child), "# drift\n")
			if _, err := f.inspector.CheckEffortSources(ctx, f.frozen, f.request.Invocation); err == nil {
				t.Fatal("child drift admitted")
			}
		})
	}
	cases := map[string]func(*testing.T, *stageInspectorFixture, *executiongraph.Invocation){
		"not-consented": func(_ *testing.T, f *stageInspectorFixture, _ *executiongraph.Invocation) {
			f.inspector.Consented = false
		},
		"no-configuration": func(_ *testing.T, f *stageInspectorFixture, _ *executiongraph.Invocation) {
			f.inspector.Configuration = nil
		},
		"invalid-profile": func(_ *testing.T, f *stageInspectorFixture, _ *executiongraph.Invocation) { f.frozen.RuntimeID = "" },
		"unallowed-profile": func(_ *testing.T, f *stageInspectorFixture, _ *executiongraph.Invocation) {
			f.frozen.ProfileID = "other-profile"
		},
		"runtime-mismatch": func(_ *testing.T, f *stageInspectorFixture, inv *executiongraph.Invocation) { inv.RuntimeID = "claude" },
		"empty-argv":       func(_ *testing.T, f *stageInspectorFixture, inv *executiongraph.Invocation) { inv.Argv = nil },
		"home-missing": func(_ *testing.T, f *stageInspectorFixture, inv *executiongraph.Invocation) {
			*inv = executiongraph.WithExactEnvironment(*inv, []string{"PATH=fixture"})
		},
		"home-relocated": func(_ *testing.T, f *stageInspectorFixture, inv *executiongraph.Invocation) {
			*inv = executiongraph.WithExactEnvironment(*inv, []string{"HOME=" + f.request.Roots.User, "CODEX_HOME=" + f.request.Roots.User})
		},
		"configuration-unavailable": func(_ *testing.T, f *stageInspectorFixture, _ *executiongraph.Invocation) {
			f.inspector.Configuration = func(context.Context) (runtimeprofile.ConfigurationV4, error) {
				return runtimeprofile.ConfigurationV4{}, errors.New("unavailable")
			}
		},
		"configuration-empty": func(_ *testing.T, f *stageInspectorFixture, _ *executiongraph.Invocation) {
			f.inspector.Configuration = func(context.Context) (runtimeprofile.ConfigurationV4, error) {
				return runtimeprofile.ConfigurationV4{}, nil
			}
		},
		"child-not-equivalent": func(t *testing.T, f *stageInspectorFixture, _ *executiongraph.Invocation) {
			os.Remove(f.projectSource(t, f.child))
		},
		"child-selector": func(t *testing.T, f *stageInspectorFixture, _ *executiongraph.Invocation) {
			writeFixtureFile(t, f.projectSource(t, f.child), "model_reasoning_effort = \"low\"\n")
		},
		"read-only-selector": func(t *testing.T, f *stageInspectorFixture, _ *executiongraph.Invocation) {
			f.inspector.Workspaces = nil
			writeFixtureFile(t, f.projectSource(t, f.child), "model_reasoning_effort = \"low\"\n")
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			f := newStageInspectorFixture(t, "codex")
			inv := f.request.Invocation
			mutate(t, f, &inv)
			if got, err := f.inspector.CheckEffortSources(ctx, f.frozen, inv); err == nil || got != "" {
				t.Fatalf("%s admitted: %q", name, got)
			}
		})
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	f := newStageInspectorFixture(t, "codex")
	if _, err := f.inspector.CheckEffortSources(canceled, f.frozen, f.request.Invocation); err == nil {
		t.Fatal("canceled inspection")
	}
}

func TestStageEffortInspectorExecutableIdentity(t *testing.T) {
	ctx := context.Background()
	f := newStageInspectorFixture(t, "codex")
	before, err := f.inspector.CheckEffortSources(ctx, f.frozen, f.request.Invocation)
	if err != nil {
		t.Fatal(err)
	}
	executable := f.request.Invocation.Argv[0]
	if err := os.WriteFile(executable, []byte("replaced; never execute"), 0o700); err != nil {
		t.Fatal(err)
	}
	after, err := f.inspector.CheckEffortSources(ctx, f.frozen, f.request.Invocation)
	if err != nil || after == before {
		t.Fatal("executable drift is not bound into the digest", err)
	}
	if err := os.Remove(executable); err != nil {
		t.Fatal(err)
	}
	if _, err := f.inspector.CheckEffortSources(ctx, f.frozen, f.request.Invocation); !errors.Is(err, ErrExecutableUnavailable) {
		t.Fatal("missing executable admitted", err)
	}
}

func TestStageEffortInspectorReadOnlyChildInspectsActualRoot(t *testing.T) {
	ctx := context.Background()
	f := newStageInspectorFixture(t, "codex")
	f.inspector.Workspaces = nil
	want, err := InspectEffortSourcesV4(ctx, f.request)
	if err != nil {
		t.Fatal(err)
	}
	if got, err := f.inspector.CheckEffortSources(ctx, f.frozen, f.request.Invocation); err != nil || got != want {
		t.Fatalf("read-only child digest %q %v, want %q", got, err, want)
	}
}

// Preview (OperationalSource over the repository root) and dispatch (inspector
// over an equivalent child worktree) observe the same EffortSourceDigest.
func TestStageEffortPreviewDispatchAgreement(t *testing.T) {
	ctx := context.Background()
	for _, runtime := range []string{"codex", "claude"} {
		t.Run(runtime, func(t *testing.T) {
			f := newStageInspectorFixture(t, runtime)
			executable := f.request.Invocation.Argv[0]
			source := OperationalSource{Configuration: f.request.Configuration, LookPath: func(string) (string, error) { return executable, nil }, Repository: f.request.Roots.Project, Environment: f.stage.ParentEnvironment, Consented: true, ObservedAt: time.Unix(1, 0)}
			observed, err := source.ObserveOperational(ctx, runtime, f.request.ProfileID)
			if err != nil {
				t.Fatal(err)
			}
			got, err := f.inspector.CheckEffortSources(ctx, f.frozen, f.request.Invocation)
			if err != nil || got != observed.EffortSourceDigest {
				t.Fatalf("dispatch digest %q %v, preview %q", got, err, observed.EffortSourceDigest)
			}
		})
	}
}
