package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/cli"
	"github.com/rgomids/axiom/internal/local"
	"github.com/rgomids/axiom/internal/projectapp"
	"github.com/rgomids/axiom/internal/workitem"
)

// Issue #230 I230-T06 black-box Evidence for the Work Item lifecycle verbs.

func seedWorkItemLink(t *testing.T, state, projectID string) {
	t.Helper()
	store, err := local.NewWorkItemStore(state)
	if err != nil {
		t.Fatal(err)
	}
	link := workitem.Link{ProjectID: projectID, RepositoryKey: "main", Provider: "github", Resource: "owner/repo", ExternalID: "1", URL: "https://github.com/owner/repo/issues/1", State: "OPEN"}
	if err := store.Save(context.Background(), link); err != nil {
		t.Fatal(err)
	}
}

// lifecycleOperations runs every Provider-using Work Item entrypoint, each in
// its preview and in its authorized form.
func lifecycleOperations(service lifecycleService) map[string]cli.Result {
	ctx := context.Background()
	item := cli.WorkItemInput{Project: "guarded", Repository: "main", ProviderRepository: "owner/repo", Number: 1, Title: "New title", Message: "x", Intent: "x", Problem: "x", DesiredOutcome: "x", Context: "x", Scope: "x", Constraints: "x", NonGoals: "x", Acceptance: "x"}
	authorized := item
	authorized.PreviewDigest, authorized.AuthorizeExternal, authorized.AuthorizeLocal = strings.Repeat("0", 64), true, true
	return map[string]cli.Result{
		"update preview":   service.WorkItemUpdate(ctx, item),
		"update":           service.WorkItemUpdate(ctx, authorized),
		"close preview":    service.WorkItemClose(ctx, item),
		"close":            service.WorkItemClose(ctx, authorized),
		"reopen preview":   service.WorkItemReopen(ctx, item),
		"reopen":           service.WorkItemReopen(ctx, authorized),
		"comment preview":  service.WorkItemComment(ctx, item),
		"comment":          service.WorkItemComment(ctx, authorized),
		"complete preview": service.WorkItemComplete(ctx, item),
		"complete":         service.WorkItemComplete(ctx, authorized),
		"create":           service.WorkItemCreate(ctx, authorized),
		"select":           service.WorkItemSelect(ctx, authorized),
	}
}

func assertLocalInspectionAllowed(t *testing.T, service lifecycleService) {
	t.Helper()
	listed := service.WorkItemList(context.Background(), cli.WorkItemInput{Project: "guarded"})
	if listed.Admission != nil || listed.Category != "work_items_listed" || len(listed.WorkItems) != 1 || listed.WorkItems[0].ExternalID != "1" {
		t.Fatalf("list = %+v", listed)
	}
	shown := service.WorkItemShow(context.Background(), cli.WorkItemInput{Project: "guarded", Repository: "main", Number: 1})
	if shown.Admission != nil || shown.Category != "work_item_loaded" {
		t.Fatalf("show = %+v", shown)
	}
}

func TestArchivedProjectAllowsWorkItemInspectionAndDeniesEveryEffect(t *testing.T) {
	env := newAdmissionEnv(t)
	seedWorkItemLink(t, env.state, env.projectID)
	env.apply(t, projectapp.ArchiveProject, "")
	service := env.service.(lifecycleService)
	before := snapshotTrees(t, env.root, env.state, env.workspace)
	for name, result := range lifecycleOperations(service) {
		if result.Category != projectapp.AdmissionProjectArchived || result.Admission == nil || result.Admission.Allowed || result.WorkItemChange != nil {
			t.Fatalf("%s: %+v", name, result)
		}
	}
	assertLocalInspectionAllowed(t, service)
	var output bytes.Buffer
	args := []string{"work-item", "close", "--project", "guarded", "--repository", "main", "--number", "1", "--authorize-external"}
	if code := cli.Run(context.Background(), args, env.service, currentProvenance(), &output); code != cli.ExitFailure || !strings.Contains(output.String(), `"code":"project_archived"`) {
		t.Fatalf("CLI close on archived Project: %d %s", code, output.String())
	}
	if after := snapshotTrees(t, env.root, env.state, env.workspace); !bytes.Equal(before, after) {
		t.Fatal("denied Work Item operation changed portable, local or Repository state")
	}
	if env.providerCalled() {
		t.Fatal("archived Project reached the Provider")
	}
}

func TestDisabledWorkItemsIntegrationDeniesLifecycleEffectsButNotInspection(t *testing.T) {
	env := newAdmissionEnv(t)
	seedWorkItemLink(t, env.state, env.projectID)
	env.apply(t, projectapp.DisableIntegration, "work-items")
	service := env.service.(lifecycleService)
	before := snapshotTrees(t, env.root, env.state, env.workspace)
	for name, result := range lifecycleOperations(service) {
		if result.Category != projectapp.AdmissionIntegrationDisabled || result.Admission == nil || result.Admission.Integration != "work-items" {
			t.Fatalf("%s: %+v", name, result)
		}
	}
	assertLocalInspectionAllowed(t, service)
	if after := snapshotTrees(t, env.root, env.state, env.workspace); !bytes.Equal(before, after) {
		t.Fatal("denied Work Item operation changed state")
	}
	if env.providerCalled() {
		t.Fatal("disabled Integration reached the Provider")
	}
}

func TestWorkItemDeleteIsNotACommand(t *testing.T) {
	env := newAdmissionEnv(t)
	seedWorkItemLink(t, env.state, env.projectID)
	before := snapshotTrees(t, env.root, env.state, env.workspace)
	for _, args := range [][]string{
		{"work-item", "delete", "--project", "guarded", "--repository", "main", "--number", "1"},
		{"work-item", "delete", "--project", "guarded", "--repository", "main", "--number", "1", "--preview-digest", strings.Repeat("0", 64), "--authorize-external"},
	} {
		var output bytes.Buffer
		if code := cli.Run(context.Background(), args, env.service, currentProvenance(), &output); code != cli.ExitFailure || !strings.Contains(output.String(), `"category":"invalid_command"`) {
			t.Fatalf("%v: %d %s", args, code, output.String())
		}
	}
	if after := snapshotTrees(t, env.root, env.state, env.workspace); !bytes.Equal(before, after) || env.providerCalled() {
		t.Fatal("work-item delete had an effect")
	}
}

type workItemLifecycleEvent struct {
	Status   string `json:"status"`
	Result   string `json:"result"`
	Category string `json:"category"`
	Change   *struct {
		Digest   string                   `json:"digest"`
		Effects  []string                 `json:"effects"`
		Document *workitem.DocumentChange `json:"document"`
	} `json:"change"`
	Selection *struct {
		Digest string `json:"digest"`
	} `json:"selection"`
	WorkItems *[]cli.WorkItemView `json:"workItems"`
}

func TestWorkItemLifecycleThroughTheCLIWithAStatefulProvider(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Provider stub is a POSIX script")
	}
	env := newBootstrapEnv(t)
	providerDir := t.TempDir()
	gh := filepath.Join(providerDir, "gh")
	script := `#!/bin/sh
dir="$AXIOM_TEST_PROVIDER"
state=$(cat "$dir/state" 2>/dev/null || echo open)
case "$*" in
  *PATCH*repos/owner/repo/issues/7*)
    input=$(cat)
    printf '%s\n' "$input" >> "$dir/patches"
    case "$input" in
      *'"state":"closed"'*) state=closed ;;
      *'"state":"open"'*) state=open ;;
    esac
    echo "$state" > "$dir/state" ;;
  *repos/owner/repo/issues/7*) ;;
  *) exit 1 ;;
esac
printf '{"number":7,"html_url":"https://github.com/owner/repo/issues/7","state":"%s","title":"Current title","body":"Current body"}\n' "$state"
`
	if err := os.WriteFile(gh, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AXIOM_GH_BIN", gh)
	t.Setenv("AXIOM_TEST_PROVIDER", providerDir)
	env.service = compose()
	repository := writeTree(t, filepath.Join(env.workspace, "main"), map[string]string{"README.md": "x"})
	configureProject(t, env.service, "life", "Life", "main="+repository, "github")
	run := func(wantCode int, args ...string) workItemLifecycleEvent {
		t.Helper()
		var output bytes.Buffer
		code := cli.Run(context.Background(), args, env.service, currentProvenance(), &output)
		var event workItemLifecycleEvent
		if err := json.Unmarshal(output.Bytes(), &event); err != nil || code != wantCode {
			t.Fatalf("%v: code=%d err=%v output=%s", args, code, err, output.String())
		}
		return event
	}
	patches := func() string {
		wire, _ := os.ReadFile(filepath.Join(providerDir, "patches"))
		return string(wire)
	}
	selector := []string{"--project", "life", "--repository", "main", "--work-item", "github:owner/repo#7"}
	selection := run(cli.ExitSuccess, append([]string{"work-item", "select"}, selector...)...)
	run(cli.ExitSuccess, append(append([]string{"work-item", "select"}, selector...), "--preview-digest", selection.Selection.Digest, "--authorize-local")...)

	update := append(append([]string{"work-item", "update"}, selector...), "--title", "Clearer title")
	preview := run(cli.ExitSuccess, update...)
	if preview.Category != "work_item_update_ready" || preview.Change == nil || preview.Change.Document == nil || preview.Change.Document.Title != "Clearer title" || preview.Change.Document.Body != "" {
		t.Fatalf("update preview = %+v", preview)
	}
	if denied := run(cli.ExitFailure, append(update, "--authorize-external")...); denied.Category != "external_authority_denied" || denied.Change == nil || patches() != "" {
		t.Fatalf("single-step update = %+v patches=%q", denied, patches())
	}
	if updated := run(cli.ExitSuccess, append(update, "--preview-digest", preview.Change.Digest, "--authorize-external")...); updated.Category != "work_item_updated" || patches() != `{"title":"Clearer title"}`+"\n" {
		t.Fatalf("update = %+v patches=%q", updated, patches())
	}
	sections := append(append([]string{"work-item", "update"}, selector...), "--scope", "Narrower scope")
	if unrecognized := run(cli.ExitFailure, sections...); unrecognized.Result != "Current Issue body does not have the Axiom section structure" || patches() != `{"title":"Clearer title"}`+"\n" {
		t.Fatalf("section update of a free-form body = %+v", unrecognized)
	}

	closeArgs := append([]string{"work-item", "close"}, selector...)
	closePreview := run(cli.ExitSuccess, closeArgs...)
	if closePreview.Category != "work_item_close_ready" || len(closePreview.Change.Effects) != 2 {
		t.Fatalf("close preview = %+v", closePreview)
	}
	if closed := run(cli.ExitSuccess, append(closeArgs, "--preview-digest", closePreview.Change.Digest, "--authorize-external")...); closed.Category != "work_item_closed" {
		t.Fatalf("close = %+v", closed)
	}
	listed := run(cli.ExitSuccess, "work-item", "list", "--project", "life", "--repository", "main")
	if listed.WorkItems == nil || len(*listed.WorkItems) != 1 || (*listed.WorkItems)[0].State != "CLOSED" {
		t.Fatalf("list after close = %+v", listed.WorkItems)
	}
	before := patches()
	if again := run(cli.ExitSuccess, append(closeArgs, "--preview-digest", closePreview.Change.Digest, "--authorize-external")...); again.Category != "work_item_already_closed" {
		t.Fatalf("close when closed = %+v", again)
	}
	if complete := run(cli.ExitSuccess, append([]string{"work-item", "complete"}, selector...)...); complete.Category != "work_item_already_closed" || patches() != before {
		t.Fatalf("complete when closed = %+v patches=%q", complete, patches())
	}
	reopenArgs := append([]string{"work-item", "reopen"}, selector...)
	reopenPreview := run(cli.ExitSuccess, reopenArgs...)
	if reopened := run(cli.ExitSuccess, append(reopenArgs, "--preview-digest", reopenPreview.Change.Digest, "--authorize-external")...); reopened.Category != "work_item_reopened" {
		t.Fatalf("reopen = %+v", reopened)
	}
	if again := run(cli.ExitSuccess, reopenArgs...); again.Category != "work_item_already_open" {
		t.Fatalf("reopen when open = %+v", again)
	}
	if want := `{"title":"Clearer title"}` + "\n" + `{"state":"closed"}` + "\n" + `{"state":"open"}` + "\n"; patches() != want {
		t.Fatalf("Provider PATCH requests = %q", patches())
	}
	listed = run(cli.ExitSuccess, "work-item", "list", "--project", "life")
	if listed.WorkItems == nil || len(*listed.WorkItems) != 1 || (*listed.WorkItems)[0].State != "OPEN" {
		t.Fatalf("list after reopen = %+v", listed.WorkItems)
	}
}
