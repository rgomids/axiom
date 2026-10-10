package main

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/rgomids/axiom/internal/local"
	"github.com/rgomids/axiom/internal/manifest"
	"github.com/rgomids/axiom/internal/project"
	"github.com/rgomids/axiom/internal/projectapp"
	"github.com/rgomids/axiom/internal/workitem"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/cli"
	"github.com/rgomids/axiom/internal/runtimeadapter"
	"github.com/rgomids/axiom/internal/runtimeapplication"
	"github.com/rgomids/axiom/internal/workflowdefinition"
)

func TestWorkflowStartValidatesTargetBeforeRuntimePreview(t *testing.T) {
	env := newWorkItemSourceEnvironment(t)
	env.installOutsideRoot(t, portableManifest(t, workItemSourceProjectID, "external", "External", true))
	installTestRuntimePolicy(t, env.state, workItemSourceProjectID, "codex")
	service, _ := testRuntime(t, env.service.(lifecycleService), "codex")
	before := snapshotTrees(t, env.root, env.state, env.elsewhere)
	for _, number := range []int{0, -1, 7} {
		input := cli.WorkflowInput{Project: "external", Repository: "main", Number: number, Role: "implementation", Complexity: "high", Capabilities: []string{runtimeadapter.IntegrationCapability}}
		got := service.WorkflowStart(context.Background(), input)
		if got.Completion == nil || got.Completion.Status() != "validation_failure" || got.RuntimeResolution != nil || got.Workflow != nil {
			t.Errorf("invalid/unlinked %d reached Runtime preview: %+v", number, got)
		}
	}
	if !bytes.Equal(before, snapshotTrees(t, env.root, env.state, env.elsewhere)) {
		t.Fatal("invalid target changed state")
	}
}

// A real protected installation/link fixture; no Provider or Runtime process is run.
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

func TestWorkflowStartReviewBindsCanonicalTargetAndContext(t *testing.T) {
	env := newWorkItemSourceEnvironment(t)
	installLinkedTarget(t, env)
	installTestRuntimePolicy(t, env.state, workItemSourceProjectID, "codex")
	service, _ := testRuntime(t, env.service.(lifecycleService), "codex")
	ctx := context.Background()
	input := cli.WorkflowInput{Project: "external", Repository: "main", Number: 7, Role: "implementation", Complexity: "high", Capabilities: []string{runtimeadapter.IntegrationCapability}}
	preview := service.WorkflowStart(ctx, input)
	if preview.ExecutionTarget == nil || preview.ExecutionTarget.ProjectID != workItemSourceProjectID || preview.ExecutionTarget.ProjectSource != "explicit" || preview.ExecutionTarget.WorkItem != "github:owner/repo#7" {
		t.Fatalf("preview=%+v", preview)
	}
	if preview.PreviewDigest == preview.RuntimeResolution.Digest() {
		t.Fatal("workflow review reused policy-only digest")
	}
	before := snapshotTrees(t, env.root, env.state, env.elsewhere)
	for name, change := range map[string]func(*cli.WorkflowInput){
		"item":               func(v *cli.WorkflowInput) { v.Number = 8 },
		"old policy token":   func(v *cli.WorkflowInput) { v.RuntimePreview = preview.RuntimeResolution.Digest() },
		"conflicting fields": func(v *cli.WorkflowInput) { v.WorkItem = "github:owner/repo#7" },
		"execution selector": func(v *cli.WorkflowInput) { v.Execution = "different" },
	} {
		t.Run(name, func(t *testing.T) {
			v := input
			v.RuntimePreview = preview.PreviewDigest
			change(&v)
			got := service.WorkflowStart(ctx, v)
			if got.Completion.Status() != "validation_failure" || got.Workflow != nil {
				t.Fatalf("accepted changed target: %+v", got)
			}
		})
	}
	if !bytes.Equal(before, snapshotTrees(t, env.root, env.state, env.elsewhere)) {
		t.Fatal("refusal mutated state")
	}
	// Slug and UUID identify the same explicit target.
	input.Project = workItemSourceProjectID
	input.RuntimePreview = preview.PreviewDigest
	started := service.WorkflowStart(ctx, input)
	if started.Workflow == nil || started.Workflow.WorkItem.ProjectID != workItemSourceProjectID || started.Workflow.WorkItem.ExternalID != "7" {
		t.Fatalf("start=%+v", started)
	}
	before = snapshotTrees(t, env.root, env.state, env.elsewhere)
	repeated := service.WorkflowStart(ctx, input)
	if repeated.Category != "execution_already_started" || repeated.Workflow.ExecutionID != started.Workflow.ExecutionID || !bytes.Equal(before, snapshotTrees(t, env.root, env.state, env.elsewhere)) {
		t.Fatal("equivalent start did not converge")
	}
}

func TestWorkflowStartEffectiveProjectSourcesAndDisclosure(t *testing.T) {
	env := newWorkItemSourceEnvironment(t)
	installLinkedTarget(t, env)
	installTestRuntimePolicy(t, env.state, workItemSourceProjectID, "codex")
	service, _ := testRuntime(t, env.service.(lifecycleService), "codex")
	ctx := projectapp.WithProjectSession(context.Background(), "target-session")
	input := cli.WorkflowInput{Repository: "main", WorkItem: "github:owner/repo#7", Role: "implementation", Complexity: "high", Capabilities: []string{runtimeadapter.IntegrationCapability}}
	if got := service.WorkflowStart(ctx, input); got.Category != "project_context_unresolved" {
		t.Fatal(got)
	}
	if category := service.projectContext().Set(ctx, "", "external"); category != "" {
		t.Fatal(category)
	}
	for _, source := range []string{"default", "session", "explicit"} {
		if source == "session" {
			if category := service.projectContext().Set(ctx, "target-session", "external"); category != "" {
				t.Fatal(category)
			}
		}
		if source == "explicit" {
			input.Project = "external"
		}
		preview := service.WorkflowStart(ctx, input)
		if preview.ExecutionTarget == nil || preview.ExecutionTarget.ProjectSource != source {
			t.Fatalf("source=%s preview=%+v", source, preview)
		}
		args := []string{"--session", "target-session", "workflow", "start", "--repository", "main", "--work-item", input.WorkItem, "--role", input.Role, "--complexity", input.Complexity, "--capabilities", runtimeadapter.IntegrationCapability}
		if input.Project != "" {
			args = append(args, "--project", input.Project)
		}
		for _, jsonMode := range []bool{false, true} {
			call := append([]string(nil), args...)
			if jsonMode {
				call = append([]string{"--json"}, call...)
			}
			var out bytes.Buffer
			if code := cli.RunInteractive(ctx, call, service, currentProvenance(), nil, &out, nil); code != 0 {
				t.Fatalf("code=%d output=%s", code, &out)
			}
			projectSource := "- **projectSource:** `" + source + "`"
			if jsonMode {
				projectSource = `"projectSource":"` + source + `"`
			}
			for _, want := range []string{"executionTarget", workItemSourceProjectID, projectSource, "github:owner/repo#7"} {
				if !strings.Contains(out.String(), want) {
					t.Fatalf("missing %s: %s", want, &out)
				}
			}
			if strings.Contains(out.String(), env.state) || strings.Contains(out.String(), env.elsewhere) {
				t.Fatal("target leaked local paths")
			}
		}
		// Confirm through the CLI too: parser-injected UUID must retain its
		// original session/default source for the exact reviewed digest.
		var confirmed bytes.Buffer
		approvedArgs := append(append([]string{"--json"}, args...), "--runtime-preview", preview.PreviewDigest)
		if code := cli.RunInteractive(ctx, approvedArgs, service, currentProvenance(), nil, &confirmed, nil); code != 0 {
			t.Fatalf("confirmed %s code=%d output=%s", source, code, &confirmed)
		}
		var started struct {
			Workflow *cli.WorkflowView `json:"workflow"`
		}
		if err := json.Unmarshal(confirmed.Bytes(), &started); err != nil || started.Workflow == nil || started.Workflow.WorkItem.ProjectID != workItemSourceProjectID || started.Workflow.WorkItem.ExternalID != "7" {
			t.Fatalf("confirmed source %s lost canonical target: %s", source, &confirmed)
		}
	}
	// A stale winning session never falls back to the valid default; explicit wins.
	stale := "123e4567-e89b-42d3-a456-426614174999"
	if err := service.installation.WriteProjectContext(ctx, "target-session", stale); err != nil {
		t.Fatal(err)
	}
	input.Project = ""
	if got := service.WorkflowStart(ctx, input); got.ExecutionTarget != nil || got.Completion.Status() != "validation_failure" {
		t.Fatal(got)
	}
	input.Project = "external"
	if got := service.WorkflowStart(ctx, input); got.ExecutionTarget == nil {
		t.Fatal(got)
	}
}

func TestWorkflowStartStaleContextReviewAndPersistedIdentity(t *testing.T) {
	env := newWorkItemSourceEnvironment(t)
	installLinkedTarget(t, env)
	installTestRuntimePolicy(t, env.state, workItemSourceProjectID, "codex")
	service, _ := testRuntime(t, env.service.(lifecycleService), "codex")
	ctx := projectapp.WithProjectSession(context.Background(), "one")
	for _, session := range []string{"", "one"} {
		if c := service.projectContext().Set(ctx, session, "external"); c != "" {
			t.Fatal(c)
		}
	}
	input := cli.WorkflowInput{Repository: "main", Number: 7, Role: "implementation", Complexity: "high", Capabilities: []string{runtimeadapter.IntegrationCapability}}
	preview := service.WorkflowStart(ctx, input)
	if preview.ExecutionTarget == nil {
		t.Fatal(preview)
	}
	// Identical UUID, changed winning source: review must be renewed.
	if c := service.projectContext().Clear(ctx, "one"); c != "" {
		t.Fatal(c)
	}
	before := snapshotTrees(t, env.root, env.state, env.elsewhere)
	input.RuntimePreview = preview.PreviewDigest
	refused := service.WorkflowStart(ctx, input)
	if refused.RuntimeResolution == nil || refused.RuntimeResolution.Blocker.Code != "stale_preview" || !bytes.Equal(before, snapshotTrees(t, env.root, env.state, env.elsewhere)) {
		t.Fatal("changed source reused review")
	}
	input.RuntimePreview = ""
	preview = service.WorkflowStart(ctx, input)
	input.RuntimePreview = preview.PreviewDigest
	started := service.WorkflowStart(ctx, input)
	if started.Workflow == nil {
		t.Fatal(started)
	}
	for _, session := range []string{"", "one"} {
		if err := service.installation.WriteProjectContext(ctx, session, "123e4567-e89b-42d3-a456-426614174999"); err != nil {
			t.Fatal(err)
		}
	}
	before = snapshotTrees(t, env.root, env.state, env.elsewhere)
	input.Project = workItemSourceProjectID
	input.Execution = started.Workflow.ExecutionID
	status := service.WorkflowStatus(ctx, input)
	if status.Workflow == nil || status.Workflow.ExecutionID != started.Workflow.ExecutionID || status.Workflow.WorkItem.ProjectID != workItemSourceProjectID || status.Workflow.RuntimeID != "codex" || !bytes.Equal(before, snapshotTrees(t, env.root, env.state, env.elsewhere)) {
		t.Fatal("context rebound Execution")
	}
}

func TestWorkflowStartAmbiguousLinkAndProjectFailClosed(t *testing.T) {
	env := newWorkItemSourceEnvironment(t)
	installLinkedTarget(t, env)
	installTestRuntimePolicy(t, env.state, workItemSourceProjectID, "codex")
	service, _ := testRuntime(t, env.service.(lifecycleService), "codex")
	input := cli.WorkflowInput{Project: "external", Repository: "main", Number: 7, Role: "implementation", Complexity: "high", Capabilities: []string{runtimeadapter.IntegrationCapability}}
	saveTargetLink(t, env.state, workItemSourceProjectID, "7", "other/repo")
	before := snapshotTrees(t, env.root, env.state, env.elsewhere)
	if got := service.WorkflowStart(context.Background(), input); got.Category != "work_item_ambiguous" || got.RuntimeResolution != nil {
		t.Fatal(got)
	}
	input.Number = 0
	input.WorkItem = "github:owner/repo#7"
	if got := service.WorkflowStart(context.Background(), input); got.ExecutionTarget == nil {
		t.Fatal("exact selector failed to disambiguate", got)
	}
	if !bytes.Equal(before, snapshotTrees(t, env.root, env.state, env.elsewhere)) {
		t.Fatal("ambiguous linkage mutated state")
	}
	// Duplicate observed slug is representable; UUID remains the exact escape.
	other := "123e4567-e89b-42d3-a456-426614174001"
	record, issues := local.NewRecord(local.RecordState{ProjectID: other, ObservedSlug: "external", SourceLocation: filepath.Join(env.elsewhere, "external"), PortableRevision: projectapp.RecordedPortableRevision([32]byte{1}), ArtifactDigests: []projectapp.ArtifactDigest{{Name: "axiom.yaml", Digest: [32]byte{1}}}})
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	wire, issues := local.EncodeRecord(record)
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	directory := filepath.Join(env.state, "projects", other)
	if err := os.Mkdir(directory, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "installation.json"), wire, 0600); err != nil {
		t.Fatal(err)
	}
	before = snapshotTrees(t, env.root, env.state, env.elsewhere)
	if got := service.WorkflowStart(context.Background(), input); got.Category != "project_ambiguous" || got.RuntimeResolution != nil {
		t.Fatal("ambiguous project accepted", got)
	}
	input.Project = workItemSourceProjectID
	if got := service.WorkflowStart(context.Background(), input); got.ExecutionTarget == nil {
		t.Fatal("UUID escape failed", got)
	}
	if !bytes.Equal(before, snapshotTrees(t, env.root, env.state, env.elsewhere)) {
		t.Fatal("ambiguous Project mutated state")
	}
}

func TestWorkflowStartNeverSearchesAnotherProjectForWorkItem(t *testing.T) {
	env := newWorkItemSourceEnvironment(t)
	installLinkedTarget(t, env)
	installTestRuntimePolicy(t, env.state, workItemSourceProjectID, "codex")
	service, _ := testRuntime(t, env.service.(lifecycleService), "codex")
	other := "123e4567-e89b-42d3-a456-426614174001"
	source, repository := filepath.Join(env.elsewhere, "other"), filepath.Join(t.TempDir(), "repository")
	for _, path := range []string{source, repository} {
		if err := os.Mkdir(path, 0700); err != nil {
			t.Fatal(err)
		}
	}
	value, issues := manifest.Decode(portableManifest(t, other, "other", "Other", true))
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	state := value.State()
	state.Repositories = project.Configured([]project.Repository{{Key: "main"}})
	value, issues = project.New(state)
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
	input := cli.WorkflowInput{Project: other, Repository: "main", WorkItem: "github:owner/repo#7", Role: "implementation", Complexity: "high", Capabilities: []string{runtimeadapter.IntegrationCapability}}
	before := snapshotTrees(t, env.root, env.state, env.elsewhere)
	if got := service.WorkflowStart(context.Background(), input); got.Category != "work_item_not_linked" || got.RuntimeResolution != nil || got.Workflow != nil {
		t.Fatal(got)
	}
	if !bytes.Equal(before, snapshotTrees(t, env.root, env.state, env.elsewhere)) {
		t.Fatal("cross-project refusal mutated state")
	}
}

type targetDriftPolicySource struct {
	source runtimeapplication.Source
	loads  int
	drift  func()
}

func (s *targetDriftPolicySource) Load(ctx context.Context, id string) (runtimeapplication.Snapshot, error) {
	snapshot, err := s.source.Load(ctx, id)
	s.loads++
	if err == nil && s.loads == 2 {
		s.drift()
	}
	return snapshot, err
}

func TestWorkflowStartRejectsTargetDriftDuringPolicyCheck(t *testing.T) {
	for _, drift := range []string{"linkage", "context"} {
		t.Run(drift, func(t *testing.T) {
			env := newWorkItemSourceEnvironment(t)
			installLinkedTarget(t, env)
			installTestRuntimePolicy(t, env.state, workItemSourceProjectID, "codex")
			service, _ := testRuntime(t, env.service.(lifecycleService), "codex")
			ctx := projectapp.WithProjectSession(context.Background(), "one")
			for _, session := range []string{"", "one"} {
				if c := service.projectContext().Set(ctx, session, "external"); c != "" {
					t.Fatal(c)
				}
			}
			input := cli.WorkflowInput{Repository: "main", Number: 7, Role: "implementation", Complexity: "high", Capabilities: []string{runtimeadapter.IntegrationCapability}}
			preview := service.WorkflowStart(ctx, input)
			input.RuntimePreview = preview.PreviewDigest
			var afterDrift []byte
			source := &targetDriftPolicySource{source: runtimePolicySource{installation: service.installation, portable: service.portable, stateRoot: env.state, runtimes: service.runtimeObservationSources()}}
			source.drift = func() {
				if drift == "context" {
					if c := service.projectContext().Clear(ctx, "one"); c != "" {
						t.Fatal(c)
					}
				} else {
					store, err := local.NewWorkItemStore(env.state)
					if err != nil {
						t.Fatal(err)
					}
					link, err := store.Load(ctx, workItemSourceProjectID, "main", "github", "owner/repo", "7")
					if err != nil {
						t.Fatal(err)
					}
					link.State = "CLOSED"
					if err := store.Save(ctx, link); err != nil {
						t.Fatal(err)
					}
				}
				afterDrift = snapshotTrees(t, env.root, env.state, env.elsewhere)
			}
			service.RuntimePolicySource = source
			got := service.WorkflowStart(ctx, input)
			if source.loads != 2 || got.RuntimeResolution == nil || got.RuntimeResolution.Blocker == nil || got.RuntimeResolution.Blocker.Code != "stale_preview" || got.Workflow != nil {
				t.Fatalf("drift accepted: %+v loads=%d", got, source.loads)
			}
			if !bytes.Equal(afterDrift, snapshotTrees(t, env.root, env.state, env.elsewhere)) {
				t.Fatal("stale reviewed start mutated state")
			}
		})
	}
}

func TestWorkflowStartRechecksAdmissionAfterPolicyCheck(t *testing.T) {
	env := newWorkItemSourceEnvironment(t)
	installLinkedTarget(t, env)
	installTestRuntimePolicy(t, env.state, workItemSourceProjectID, "codex")
	service, _ := testRuntime(t, env.service.(lifecycleService), "codex")
	ctx := context.Background()
	input := cli.WorkflowInput{Project: "external", Repository: "main", Number: 7, Role: "implementation", Complexity: "high", Capabilities: []string{runtimeadapter.IntegrationCapability}}
	preview := service.WorkflowStart(ctx, input)
	if preview.ExecutionTarget == nil || preview.PreviewDigest == "" {
		t.Fatalf("preview = %+v", preview)
	}
	input.RuntimePreview = preview.PreviewDigest
	var archived []byte
	source := &targetDriftPolicySource{source: runtimePolicySource{installation: service.installation, portable: service.portable, stateRoot: env.state, runtimes: service.runtimeObservationSources()}}
	source.drift = func() {
		store, err := local.NewOperationalStore(env.state)
		if err != nil {
			t.Fatal(err)
		}
		request := projectapp.OperationalRequest{ProjectID: workItemSourceProjectID, Operation: projectapp.ArchiveProject}
		preview := projectapp.ApplyOperational(ctx, store, request, "", false)
		if preview.Preview == nil {
			t.Fatalf("archive preview = %+v", preview)
		}
		if result := projectapp.ApplyOperational(ctx, store, request, preview.Preview.Digest, true); result.Status != projectapp.OperationalCommitted {
			t.Fatalf("archive = %+v", result)
		}
		archived = snapshotTrees(t, env.root, env.state, env.elsewhere)
	}
	service.RuntimePolicySource = source
	got := service.WorkflowStart(ctx, input)
	if source.loads != 2 || got.Category != projectapp.AdmissionProjectArchived || got.Admission == nil || got.Workflow != nil {
		t.Fatalf("archived Project reached execution: %+v, loads=%d", got, source.loads)
	}
	if !bytes.Equal(archived, snapshotTrees(t, env.root, env.state, env.elsewhere)) {
		t.Fatal("denied start changed state")
	}
}
