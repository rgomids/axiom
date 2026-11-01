package main

import (
	"context"
	"github.com/rgomids/axiom/internal/local"
	"github.com/rgomids/axiom/internal/manifest"
	"github.com/rgomids/axiom/internal/project"
	"github.com/rgomids/axiom/internal/workitem"
	"os"
	"path/filepath"
	"testing"

	"github.com/rgomids/axiom/internal/cli"
	"github.com/rgomids/axiom/internal/workflowdefinition"
)

func installLinkedTarget(t *testing.T, env workItemSourceEnvironment) {
	t.Helper()
	source := filepath.Join(env.elsewhere, "external")
	repository := filepath.Join(t.TempDir(), "repository")
	for _, path := range []string{source, repository} {
		if err := os.Mkdir(path, 0700); err != nil {
			t.Fatal(err)
		}
	}
	value, issues := manifest.Decode(portableManifest(t, workItemSourceProjectID, "external", "External", true))
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	state := value.State()
	state.Repositories = project.Configured([]project.Repository{{Key: "main"}})
	value, issues = project.New(state)
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	ref := workflowdefinition.Builtin().Ref("builtin")
	value, issues = value.SelectWorkflow(project.WorkflowSelection{WorkflowID: ref.WorkflowID, Revision: ref.Revision, Digest: ref.Digest, Source: ref.Source})
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	wire, issues := manifest.Encode(value)
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	if err := os.WriteFile(filepath.Join(source, "axiom.yaml"), wire, 0600); err != nil {
		t.Fatal(err)
	}
	runCLI(t, env.service, []string{"project", "install", "--source", source, "--repository", "main=" + repository}, cli.ExitSuccess, "installed")
	saveTargetLink(t, env.state, workItemSourceProjectID, "7", "owner/repo")
	saveTargetLink(t, env.state, workItemSourceProjectID, "8", "owner/repo")
}

func saveTargetLink(t *testing.T, root, projectID, id, resource string) {
	t.Helper()
	store, err := local.NewWorkItemStore(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Save(context.Background(), workitem.Link{ProjectID: projectID, RepositoryKey: "main", Provider: "github", Resource: resource, ExternalID: id, URL: "https://github.com/" + resource + "/issues/" + id, State: "OPEN"}); err != nil {
		t.Fatal(err)
	}
}

func currentProfileStartFixture(t *testing.T, service lifecycleService, root string) lifecycleService {
	t.Helper()
	store, err := local.NewRuntimeProfileStore(root)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot, err := store.Inspect(context.Background()); err != nil || snapshot.State != "active-v4" {
		t.Fatalf("current fixtures need canonical v4 local configuration: %s %v", snapshot.State, err)
	}
	if record, category := service.installation.Inspect(context.Background(), workItemSourceProjectID); category == "" && record.Record.State().ProjectID == workItemSourceProjectID {
		selectProfileBuiltinForTest(t, service, workItemSourceProjectID)
	}
	return service
}
