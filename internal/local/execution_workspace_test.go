package local

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/workflow"
)

func storeEnrolledExecution(t *testing.T) workflow.Execution {
	t.Helper()
	e, err := workflow.NewExecution(workflow.NewExecutionInput{ExecutionID: storeTestExecutionID, ProjectID: "proj", WorkItem: workflow.WorkItemLink{Provider: "linear", ID: "AXM-43", RepositoryKey: "app"}, Repositories: []workflow.RepositoryBinding{{Key: "app", Identity: "github.com/example/app"}}, WorkflowID: "minimal", Revision: "r1", Now: storeT0, EnrollIntegration: true, Stages: []workflow.StageSpec{{ID: "work", Agents: []workflow.AgentSpec{{ID: "a", Effects: []string{"workspace_write"}}, {ID: "b", DependsOn: []string{"a"}, Effects: []string{"workspace_write"}}}, TimeoutSeconds: 60}}})
	if err != nil {
		t.Fatal(err)
	}
	return e
}
func storeWorkspaceInput() workflow.PrepareInput {
	return workflow.PrepareInput{Ref: workflow.TaskRef{StageID: "work", AgentID: "a"}, AttemptID: "attempt-a", Invocation: workflow.Invocation{Runtime: "stub", Executable: "stub-bin", Model: "m1", Effort: "runtime-default", Instructions: "do", InstructionsDigest: workflow.DigestText([]byte("do")), Provenance: []workflow.FieldProvenance{{Field: "runtime", Source: "local_default"}, {Field: "model", Source: "workflow_agent"}, {Field: "effort", Source: "run_override"}, {Field: "instructions", Source: "workflow_agent"}}}, Authority: workflow.AttemptAuthority{RepositoryKey: "app", Effects: []string{"workspace_write"}, Actor: "user:local"}, Workspace: &workflow.AttemptWorkspace{RepositoryKey: "app", RepositoryIdentity: "github.com/example/app", BaseSHA: strings.Repeat("a", 40), InitialTargetHead: strings.Repeat("a", 40), SourceBranch: "refs/heads/axiom/a", TargetRef: "refs/heads/axiom/target", Worktree: "attempt-a"}, Now: storeT0}
}
func TestExecutionWorkspaceRequiresFenceAndIsImmutable(t *testing.T) {
	s, _ := newTestExecutionStore(t)
	ctx := context.Background()
	if err := s.Create(ctx, storeEnrolledExecution(t)); err != nil {
		t.Fatal(err)
	}
	lease := claim(t, s)
	defer lease.Close()
	in := storeWorkspaceInput()
	in.Generation = lease.Epoch
	current, err := s.Load(ctx, "proj", storeTestExecutionID)
	if err != nil {
		t.Fatal(err)
	}
	wire, _ := workflow.EncodeExecution(current)
	next, _ := workflow.DecodeExecution(wire)
	if err := next.PrepareAttempt(in); err != nil {
		t.Fatal(err)
	}
	if err := s.Save(ctx, next, current.Revision, nil); !errors.Is(err, workflow.ErrStaleRunner) {
		t.Fatal("unfenced workspace accepted", err)
	}
	current, err = s.Update(ctx, "proj", storeTestExecutionID, lease, func(e *workflow.Execution) error { return e.PrepareAttempt(in) })
	if err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"enrollment", "base", "source", "worktree", "target", "initial"} {
		wire, _ = workflow.EncodeExecution(current)
		next, _ = workflow.DecodeExecution(wire)
		a, _, _ := next.FindAttempt(in.AttemptID)
		switch mode {
		case "enrollment":
			next.Enrollments = nil
		case "base":
			a.Workspace.BaseSHA = strings.Repeat("b", 40)
		case "source":
			a.Workspace.SourceBranch = "refs/heads/other"
		case "worktree":
			a.Workspace.Worktree = "other"
		case "target":
			a.Workspace.TargetRef = "refs/heads/other"
		case "initial":
			a.Workspace.InitialTargetHead = strings.Repeat("b", 40)
		}
		next.Revision++
		if err := s.Save(ctx, next, current.Revision, lease); err == nil {
			t.Fatal("workspace rewrite accepted", mode)
		}
	}
}
func TestExecutionWorkspaceUpdateSnapshotIsolation(t *testing.T) {
	s, _ := newTestExecutionStore(t)
	ctx := context.Background()
	if err := s.Create(ctx, storeEnrolledExecution(t)); err != nil {
		t.Fatal(err)
	}
	lease := claim(t, s)
	defer lease.Close()
	in := storeWorkspaceInput()
	in.Generation = lease.Epoch
	current, err := s.Update(ctx, "proj", storeTestExecutionID, lease, func(e *workflow.Execution) error { return e.PrepareAttempt(in) })
	if err != nil {
		t.Fatal(err)
	}
	before, _ := workflow.EncodeExecution(current)
	refusal := errors.New("refused")
	got, err := s.Update(ctx, "proj", storeTestExecutionID, lease, func(e *workflow.Execution) error {
		a, _, _ := e.FindAttempt(in.AttemptID)
		a.Workspace.Worktree = "mutated"
		return refusal
	})
	if !errors.Is(err, refusal) {
		t.Fatal(err)
	}
	after, _ := workflow.EncodeExecution(got)
	if string(before) != string(after) {
		t.Fatal("failed mutation changed returned snapshot")
	}
}

func TestExecutionWorkspaceWriterRecognizedByInventory(t *testing.T) {
	s, root := newTestExecutionStore(t)
	ctx := context.Background()
	e := storeEnrolledExecution(t)
	e.ProjectID = "87654321-4321-4abc-bdef-123456789abc"
	if err := s.Create(ctx, e); err != nil {
		t.Fatal(err)
	}
	lease, err := s.AcquireRunner(ctx, e.ProjectID, e.ExecutionID)
	if err != nil {
		t.Fatal(err)
	}
	defer lease.Close()
	if _, err = s.Update(ctx, e.ProjectID, e.ExecutionID, lease, func(x *workflow.Execution) error {
		if err := x.ClaimRunner(lease.Epoch, storeT0); err != nil {
			return err
		}
		in := storeWorkspaceInput()
		in.Generation = lease.Epoch
		return x.PrepareAttempt(in)
	}); err != nil {
		t.Fatal(err)
	}
	inventory, err := InspectStateInventory(ctx, root)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, entry := range inventory.Entries {
		found = found || ExecutionReferenceNamespace(root, entry)
	}
	if !found {
		t.Fatalf("enrolled writer unrecognized: %+v", inventory.Entries)
	}
	loaded, err := s.List(ctx, e.ProjectID)
	if err != nil || len(loaded) != 1 || loaded[0].Stages[0].Tasks[0].Attempts[0].Workspace == nil {
		t.Fatalf("canonical enrolled writer cannot load: %+v %v", loaded, err)
	}
}
