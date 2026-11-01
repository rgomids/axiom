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
	"github.com/rgomids/axiom/internal/projectapp"
	"github.com/rgomids/axiom/internal/workflowdefinition"
)

const authoredPolicyManifest = `schemaVersion: 2
project:
  id: 123e4567-e89b-42d3-a456-426614174000
  slug: authored
  name: Authored
repositories:
  - key: main
providers:
  - key: work-items
    id: github
integrations:
  - key: work-items
    providerRef: work-items
    capabilities:
      - work-item
runtimes:
  - id: claude
  - id: codex
modelProfiles:
  - key: careful
    runtimeRef: claude
    model: approved-model
  - key: worker
    runtimeRef: codex
    model: approved-model
runtimePreferences:
  - role: implementation
    complexity: high
    modelProfileRef: worker
`

// An operator-authored Runtime/Profile policy reaches production only through
// install with an exact binding for every declared Repository.
func TestProjectInstallBindsEveryDeclaredRepository(t *testing.T) {
	env := newWorkItemSourceEnvironment(t)
	source := filepath.Join(env.elsewhere, "authored")
	repository := filepath.Join(t.TempDir(), "repository")
	for _, directory := range []string{source, repository} {
		if err := os.Mkdir(directory, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(source, "axiom.yaml"), []byte(authoredPolicyManifest), 0o600); err != nil {
		t.Fatal(err)
	}
	before := snapshotTrees(t, env.root, env.elsewhere)
	for _, bindings := range [][]string{
		nil,
		{"--repository", "main=relative"},
		{"--repository", "other=" + repository},
		{"--repository", "main=" + filepath.Join(t.TempDir(), "missing")},
		{"--repository", "main=" + repository, "--repository", "main=" + repository},
		{"--repository", "main=" + repository, "--repository", "other=" + repository},
	} {
		var output bytes.Buffer
		args := append([]string{"--json", "project", "install", "--source", source}, bindings...)
		if code := cli.Run(context.Background(), args, env.service, currentProvenance(), &output); code == cli.ExitSuccess || strings.Contains(output.String(), repository) {
			t.Fatalf("%v: code=%d output=%s", bindings, code, &output)
		}
	}
	if after := snapshotTrees(t, env.root, env.elsewhere); !bytes.Equal(before, after) {
		t.Fatal("rejected install changed portable state")
	}
	if records, _ := filepath.Glob(filepath.Join(env.state, "projects", "*", "installation.json")); len(records) != 0 {
		t.Fatalf("rejected install recorded local state: %v", records)
	}
	runCLI(t, env.service, []string{"project", "install", "--source", source, "--repository", "main=" + repository}, cli.ExitSuccess, "installed")
	runCanonicalCLI(t, env.service, []string{"project", "show", "--selector", "authored"}, cli.ExitSuccess, "success", "Project resolved")
	runCLI(t, env.service, []string{"project", "install", "--source", source, "--repository", "main=" + repository}, cli.ExitSuccess, "already_installed")
}

func selectProfileBuiltinForTest(t *testing.T, s lifecycleService, selector string) {
	t.Helper()
	resolved := s.installation.Resolve(context.Background(), selector)
	if resolved.Status != local.ResolutionFound {
		t.Fatal("current builtin fixture Project unresolved")
	}
	record, category := s.installation.Inspect(context.Background(), resolved.Project.ID)
	if category != "" {
		t.Fatal(category)
	}
	// Fixture setup explicitly owns the external source for this scoped authoring
	// call; the tested service keeps its original source roots afterward.
	s.projectsRoot = filepath.Dir(record.Record.State().SourceLocation)
	var err error
	s.portable, err = local.NewPortableStore(s.projectsRoot)
	if err != nil {
		t.Fatal(err)
	}

	d, _ := workflowdefinition.BuiltinRevision(2)
	ref := d.Ref("builtin")
	r := cli.ProjectWorkflowInput{WorkflowRequest: projectapp.WorkflowRequest{Operation: "select", Project: selector, Ref: ref}}
	p := s.ProjectWorkflow(context.Background(), r)
	if p.Category == "unchanged" {
		return
	}
	if p.WorkflowAuthoring == nil || p.Category != "previewed" {
		t.Fatalf("current builtin selection preview: %+v", p)
	}
	r.ExpectedRevision = p.WorkflowAuthoring.ProjectRevision
	r.PreviewDigest = p.WorkflowAuthoring.PreviewDigest
	r.AuthorizeLocal = true
	if applied := s.ProjectWorkflow(context.Background(), r); applied.Category != "applied" {
		t.Fatalf("current builtin selection apply: %+v", applied)
	}
}
