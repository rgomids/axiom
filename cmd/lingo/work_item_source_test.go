package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/cli"
	"github.com/rgomids/axiom/internal/local"
	"github.com/rgomids/axiom/internal/manifest"
	"github.com/rgomids/axiom/internal/project"
)

// Issue #147 Evidence: Work Item resolution reads an installed Project's
// portable configuration only from the SourceLocation its protected local
// record names, and fails closed unless that configuration is exactly the one
// the installation validated (Project ID, observed slug, portable revision).

const workItemSourceProjectID = "123e4567-e89b-42d3-a456-426614174147"

type workItemSourceEnvironment struct {
	root, state, elsewhere string
	service                cli.Service
	installation           local.InstallationStore
	portable               local.PortableStore
	resolver               workItemResolver
}

func newWorkItemSourceEnvironment(t *testing.T) workItemSourceEnvironment {
	t.Helper()
	env := workItemSourceEnvironment{
		root:      filepath.Join(t.TempDir(), "projects"),
		state:     filepath.Join(t.TempDir(), "state"),
		elsewhere: filepath.Join(t.TempDir(), "elsewhere"),
	}
	// Both portable roots exist so every resolution can prove it wrote nothing.
	for _, path := range []string{env.root, env.elsewhere} {
		if err := os.Mkdir(path, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("LINGO_PROJECTS_ROOT", env.root)
	t.Setenv("LINGO_STATE_ROOT", env.state)
	// An absolute Provider binary keeps the GitHub capability composed without
	// depending on the host PATH; Work Item preview never executes it.
	t.Setenv("AXIOM_GH_BIN", filepath.Join(t.TempDir(), "gh-never-executed"))
	env.service = compose()
	var err error
	if env.installation, err = local.NewInstallationStore(env.state); err != nil {
		t.Fatal(err)
	}
	if env.portable, err = local.NewPortableStore(env.root); err != nil {
		t.Fatal(err)
	}
	env.resolver = workItemResolver{installation: env.installation, portable: env.portable}
	return env
}

// portableManifest encodes a canonical portable Project with or without the
// GitHub Work Item capability and no Repository associations.
func portableManifest(t *testing.T, id, slug, name string, workItems bool) []byte {
	t.Helper()
	state := project.State{SchemaVersion: 1, ID: id, Slug: slug, Name: name}
	if workItems {
		state.Providers = project.Configured([]project.Provider{{Key: "work-items", ID: "github"}})
		state.Integrations = project.Configured([]project.Integration{{Key: "work-items", ProviderRef: project.Configured("work-items"), Capabilities: project.Configured([]string{"work-item"})}})
	}
	value, issues := project.New(state)
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	wire, encodeIssues := manifest.Codec{}.Encode(value)
	if len(encodeIssues) != 0 {
		t.Fatal(encodeIssues)
	}
	return wire
}

// installOutsideRoot writes a portable Project outside <projects-root> and
// installs it with project install --source, so its recorded SourceLocation is
// not <projects-root>/<slug>.
func (env workItemSourceEnvironment) installOutsideRoot(t *testing.T, wire []byte) string {
	t.Helper()
	source := filepath.Join(env.elsewhere, "external")
	if err := os.Mkdir(source, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "axiom.yaml"), wire, 0o600); err != nil {
		t.Fatal(err)
	}
	runCLI(t, env.service, []string{"project", "install", "--source", source}, cli.ExitSuccess, "installed")
	if _, err := os.Lstat(filepath.Join(env.root, "external")); !os.IsNotExist(err) {
		t.Fatalf("outside-root install created <projects-root>/<slug>: %v", err)
	}
	return source
}

// resolve runs the composed Work Item resolver and proves it wrote nothing to
// the portable root, the protected state root or the recorded source root.
func (env workItemSourceEnvironment) resolve(t *testing.T, ctx context.Context, selector string) (string, int) {
	t.Helper()
	before := snapshotTrees(t, env.root, env.state, env.elsewhere)
	resolved, category := env.resolver.Resolve(ctx, selector)
	if after := snapshotTrees(t, env.root, env.state, env.elsewhere); !bytes.Equal(before, after) {
		t.Fatalf("Work Item resolution of %q wrote state", selector)
	}
	if category != "" && (resolved.ID != "" || resolved.Provider != "" || resolved.Repositories != nil) {
		t.Fatalf("failed resolution leaked a Project: %+v", resolved)
	}
	return category, len(resolved.Repositories)
}

func (env workItemSourceEnvironment) previewCreate(t *testing.T, selector string) string {
	t.Helper()
	before := snapshotTrees(t, env.root, env.state, env.elsewhere)
	result := env.service.WorkItemCreate(context.Background(), cli.WorkItemInput{Project: selector, Repository: "main", ProviderRepository: "owner/repo", Intent: "Recorded source"})
	if after := snapshotTrees(t, env.root, env.state, env.elsewhere); !bytes.Equal(before, after) {
		t.Fatal("Work Item preview wrote state")
	}
	return result.Category
}

func (env workItemSourceEnvironment) recordPath(id string) string {
	return filepath.Join(env.state, "projects", id, "installation.json")
}

func rewriteRecord(t *testing.T, path, old, replacement string) {
	t.Helper()
	wire, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	changed := strings.Replace(string(wire), old, replacement, 1)
	if changed == string(wire) {
		t.Fatalf("record rewrite of %q had no effect:\n%s", old, wire)
	}
	if err := os.WriteFile(path, []byte(changed), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestWorkItemResolverReadsRecordedSourceOutsideProjectsRoot(t *testing.T) {
	env := newWorkItemSourceEnvironment(t)
	env.installOutsideRoot(t, portableManifest(t, workItemSourceProjectID, "external", "External", true))
	for _, selector := range []string{"external", workItemSourceProjectID} {
		resolved, category := env.resolver.Resolve(context.Background(), selector)
		if category != "" || resolved.ID != workItemSourceProjectID || resolved.Provider != "github" || len(resolved.Repositories) != 0 {
			t.Fatalf("selector %q: project=%+v category=%q", selector, resolved, category)
		}
		env.resolve(t, context.Background(), selector)
	}
	// Capability evaluation passes on the recorded source; with no Repository
	// bindings (out of scope for #147) the service stops at the Repository.
	if category := env.previewCreate(t, "external"); category != "repository_not_configured" {
		t.Fatalf("outside-root Work Item preview = %q", category)
	}
}

func TestWorkItemResolverNeverAcceptsShadowCopyAtProjectsRoot(t *testing.T) {
	for name, recordedCapability := range map[string]bool{
		"shadow adds capability":    false,
		"shadow removes capability": true,
	} {
		t.Run(name, func(t *testing.T) {
			env := newWorkItemSourceEnvironment(t)
			env.installOutsideRoot(t, portableManifest(t, workItemSourceProjectID, "external", "External", recordedCapability))
			// An unrecorded same-ID, same-slug manifest at <projects-root>/<slug>.
			shadow := portableManifest(t, workItemSourceProjectID, "external", "External", !recordedCapability)
			if err := env.portable.Create(context.Background(), "external", shadow); err != nil {
				t.Fatal(err)
			}
			category, _ := env.resolve(t, context.Background(), "external")
			preview := env.previewCreate(t, "external")
			if recordedCapability {
				if category != "" || preview != "repository_not_configured" {
					t.Fatalf("recorded capability judged by shadow copy: resolve=%q preview=%q", category, preview)
				}
				return
			}
			if category != "work_item_capability_unavailable" || preview != "work_item_capability_unavailable" {
				t.Fatalf("shadow copy replaced recorded source: resolve=%q preview=%q", category, preview)
			}
		})
	}
}

func TestWorkItemResolverFailsClosedOnRecordedSourceDivergence(t *testing.T) {
	const otherID = "123e4567-e89b-42d3-a456-426614174999"
	for name, test := range map[string]struct {
		selector string
		mutate   func(t *testing.T, env workItemSourceEnvironment, source string)
	}{
		"portable project id changed after install": {"external", func(t *testing.T, _ workItemSourceEnvironment, source string) {
			rewriteManifest(t, source, func(wire string) string { return strings.Replace(wire, workItemSourceProjectID, otherID, 1) })
		}},
		"portable slug changed after install": {"external", func(t *testing.T, _ workItemSourceEnvironment, source string) {
			rewriteManifest(t, source, func(wire string) string { return strings.Replace(wire, "slug: external", "slug: other", 1) })
		}},
		"portable content changed after install": {"external", func(t *testing.T, _ workItemSourceEnvironment, source string) {
			rewriteManifest(t, source, func(wire string) string { return strings.Replace(wire, "name: External", "name: Changed", 1) })
		}},
		// The record-side divergences below keep the portable bytes (and so
		// the observed revision) unchanged, isolating each coherence check.
		"recorded project id differs only": {"external", func(t *testing.T, env workItemSourceEnvironment, _ string) {
			rewriteRecord(t, env.recordPath(workItemSourceProjectID), workItemSourceProjectID, otherID)
			if err := os.Rename(filepath.Dir(env.recordPath(workItemSourceProjectID)), filepath.Dir(env.recordPath(otherID))); err != nil {
				t.Fatal(err)
			}
		}},
		"recorded observed slug differs only": {workItemSourceProjectID, func(t *testing.T, env workItemSourceEnvironment, _ string) {
			rewriteRecord(t, env.recordPath(workItemSourceProjectID), `"external"`, `"other"`)
		}},
		"recorded portable revision differs only": {"external", func(t *testing.T, env workItemSourceEnvironment, source string) {
			observed, err := env.portable.InspectRecordedSource(context.Background(), source, "external")
			if err != nil || !observed.Exists {
				t.Fatalf("observed=%+v err=%v", observed, err)
			}
			rewriteRecord(t, env.recordPath(workItemSourceProjectID), observed.Revision, strings.Repeat("ab", 32))
		}},
	} {
		t.Run(name, func(t *testing.T) {
			env := newWorkItemSourceEnvironment(t)
			source := env.installOutsideRoot(t, portableManifest(t, workItemSourceProjectID, "external", "External", true))
			test.mutate(t, env, source)
			// The protected record itself remains selectable; only the
			// portable/local relationship is incoherent.
			if resolved := env.installation.Resolve(context.Background(), test.selector); resolved.Status != local.ResolutionFound {
				t.Fatalf("precondition: installation resolution = %+v", resolved)
			}
			if category, _ := env.resolve(t, context.Background(), test.selector); category != "invalid_project_capability_state" {
				t.Fatalf("divergent recorded source = %q", category)
			}
		})
	}
}

func TestWorkItemResolverFailsClosedOnUnavailableOrUnsafeRecordedSource(t *testing.T) {
	for name, test := range map[string]struct {
		want   string
		mutate func(t *testing.T, env workItemSourceEnvironment, source string)
	}{
		"missing source": {"project_source_unavailable", func(t *testing.T, _ workItemSourceEnvironment, source string) {
			if err := os.RemoveAll(source); err != nil {
				t.Fatal(err)
			}
		}},
		// An existing source directory without its manifest is incoherent
		// rather than absent, matching the seam's locked default-root read.
		"source without manifest": {"invalid_project_capability_state", func(t *testing.T, _ workItemSourceEnvironment, source string) {
			if err := os.Remove(filepath.Join(source, "axiom.yaml")); err != nil {
				t.Fatal(err)
			}
		}},
		"symlinked source": {"project_source_unavailable", func(t *testing.T, _ workItemSourceEnvironment, source string) {
			if err := os.Rename(source, source+"-real"); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(source+"-real", source); err != nil {
				t.Fatal(err)
			}
		}},
		"shared source permissions": {"invalid_project_capability_state", func(t *testing.T, _ workItemSourceEnvironment, source string) {
			if err := os.Chmod(source, 0o755); err != nil {
				t.Fatal(err)
			}
		}},
		"unexpected source content": {"invalid_project_capability_state", func(t *testing.T, _ workItemSourceEnvironment, source string) {
			if err := os.WriteFile(filepath.Join(source, "notes.txt"), []byte("unexpected"), 0o600); err != nil {
				t.Fatal(err)
			}
		}},
		"shadow copy remains after source removal": {"project_source_unavailable", func(t *testing.T, env workItemSourceEnvironment, source string) {
			if err := env.portable.Create(context.Background(), "external", portableManifest(t, workItemSourceProjectID, "external", "External", true)); err != nil {
				t.Fatal(err)
			}
			if err := os.RemoveAll(source); err != nil {
				t.Fatal(err)
			}
		}},
	} {
		t.Run(name, func(t *testing.T) {
			env := newWorkItemSourceEnvironment(t)
			source := env.installOutsideRoot(t, portableManifest(t, workItemSourceProjectID, "external", "External", true))
			test.mutate(t, env, source)
			if category, _ := env.resolve(t, context.Background(), "external"); category != test.want {
				t.Fatalf("category = %q, want %q", category, test.want)
			}
		})
	}
	t.Run("cancelled", func(t *testing.T) {
		env := newWorkItemSourceEnvironment(t)
		env.installOutsideRoot(t, portableManifest(t, workItemSourceProjectID, "external", "External", true))
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if category, _ := env.resolve(t, ctx, "external"); category != "cancelled" {
			t.Fatalf("cancelled resolution = %q", category)
		}
	})
}

func TestWorkItemResolverKeepsDefaultRootSourceBehavior(t *testing.T) {
	env := newWorkItemSourceEnvironment(t)
	repository := filepath.Join(t.TempDir(), "repository")
	if err := os.Mkdir(repository, 0o700); err != nil {
		t.Fatal(err)
	}
	configureProject(t, env.service, "capability", "Capability", "main="+repository, "github")

	resolved, category := env.resolver.Resolve(context.Background(), "capability")
	if category != "" || resolved.Provider != "github" || len(resolved.Repositories) != 1 || resolved.Repositories[0].Key != "main" || resolved.Repositories[0].Path != repository {
		t.Fatalf("default-root Project = %+v %q", resolved, category)
	}
	installed := env.installation.Resolve(context.Background(), "capability")
	if installed.Status != local.ResolutionFound || installed.Project.Source != filepath.Join(env.root, "capability") || resolved.ID != installed.Project.ID {
		t.Fatalf("recorded default-root source = %+v", installed)
	}

	t.Run("recovery detection", func(t *testing.T) {
		stage := filepath.Join(env.root, "capability", ".lingo-manifest-interrupted")
		if err := os.WriteFile(stage, nil, 0o600); err != nil {
			t.Fatal(err)
		}
		defer os.Remove(stage)
		if category, _ := env.resolve(t, context.Background(), "capability"); category != "recovery_required" {
			t.Fatalf("interrupted default-root source = %q", category)
		}
	})
	t.Run("stale recorded revision", func(t *testing.T) {
		// The legacy update changes only the portable manifest, so the
		// installation no longer validated the configuration it would read.
		runCLI(t, env.service, []string{"project", "update", "--slug", "capability", "--name", "Changed"}, cli.ExitSuccess, "applied")
		if category, _ := env.resolve(t, context.Background(), "capability"); category != "invalid_project_capability_state" {
			t.Fatalf("stale default-root revision = %q", category)
		}
	})
}
