package workflow

import (
	"bytes"
	"errors"
	"reflect"
	"testing"
)

func enrolledExecution(t *testing.T, gate bool) Execution {
	t.Helper()
	effects := []string{"workspace_write", "validation_command"}
	e, err := NewExecution(NewExecutionInput{ExecutionID: testExecutionID, ProjectID: "proj", WorkItem: WorkItemLink{"linear", "AXM-43", "app"}, Repositories: []RepositoryBinding{{"app", "github.com/example/app"}, {"lib", "other"}}, WorkflowID: "w", Revision: "r", Now: t0, EnrollIntegration: true, Stages: []StageSpec{
		{ID: "work", HumanGate: gate, TimeoutSeconds: 60, Agents: []AgentSpec{{ID: "a", Effects: effects}, {ID: "b", DependsOn: []string{"a"}, Effects: effects}, {ID: "reader", DependsOn: []string{"a"}}, {ID: "other", Repository: "lib"}}, Commands: []CommandCheck{{ID: "check", Argv: []string{"/bin/true"}, TimeoutSeconds: 10, Effects: []string{"validation_command"}}}},
		{ID: "review", DependsOn: []string{"work"}, TimeoutSeconds: 60, Agents: []AgentSpec{{ID: "read"}}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if err = e.ClaimRunner(1, t0); err != nil {
		t.Fatal(err)
	}
	return e
}
func workspace(id string) *AttemptWorkspace {
	return &AttemptWorkspace{RepositoryKey: "app", RepositoryIdentity: "github.com/example/app", BaseSHA: sha("b"), SourceBranch: "refs/heads/axiom/agent-" + id, Worktree: "agent-" + id, TargetRef: "refs/heads/axiom/integration", InitialTargetHead: sha("b")}
}
func prepareEnrolled(t *testing.T, e *Execution, id string) {
	t.Helper()
	if err := e.PrepareAttempt(PrepareInput{Ref: TaskRef{"work", id}, AttemptID: "att-" + id, Generation: 1, Invocation: testInvocation("do"), Authority: AttemptAuthority{"app", []string{"workspace_write", "validation_command"}, "user:local"}, Workspace: workspace("att-" + id), Now: t0}); err != nil {
		t.Fatal(err)
	}
}
func succeedEnrolled(t *testing.T, e *Execution, id string) {
	t.Helper()
	prepareEnrolled(t, e, id)
	attempt := "att-" + id
	for _, err := range []error{e.MarkStarted(attempt, 1, ProcessIdentity{PID: 4242, StartIdentity: "s", GroupIdentity: "g"}, t0), e.PrepareCommand(attempt, 1, "check", t0), e.StartCommand(attempt, 1, "check", ProcessIdentity{PID: 4243, StartIdentity: "s2", GroupIdentity: "g2"}, t0), e.FinishCommand(attempt, 1, "check", 0, t0)} {
		if err != nil {
			t.Fatal(err)
		}
	}
	a, _, _ := e.FindAttempt(attempt)
	if err := e.CompleteAttempt(attempt, 1, AttemptResult{Commands: a.Commands, Now: t0}); err != nil {
		t.Fatal(err)
	}
}

func TestWorkspaceEnrollmentCountsDependentWritersAndBindsReaders(t *testing.T) {
	e := enrolledExecution(t, false)
	if !reflect.DeepEqual(e.Enrollments, []IntegrationEnrollment{{"work", "app"}}) {
		t.Fatal(e.Enrollments)
	}
	for _, ref := range []TaskRef{{"work", "a"}, {"work", "b"}, {"work", "reader"}, {"review", "read"}} {
		if !e.IntegrationEnrolled(ref) {
			t.Fatal(ref)
		}
	}
	if e.IntegrationEnrolled(TaskRef{"work", "other"}) {
		t.Fatal("other repository enrolled")
	}
	if err := e.PrepareAttempt(PrepareInput{Ref: TaskRef{"work", "a"}, AttemptID: "att-a", Generation: 1, Invocation: testInvocation("do"), Authority: AttemptAuthority{"app", []string{"workspace_write"}, "user:local"}, Now: t0}); !errors.Is(err, ErrExecutionInvalid) {
		t.Fatal("missing workspace accepted", err)
	}
}
func TestWorkspaceIntegrationGatesDependentsAndHumanApproval(t *testing.T) {
	for _, gate := range []bool{false, true} {
		t.Run(map[bool]string{false: "ordinary", true: "gate"}[gate], func(t *testing.T) {
			e := enrolledExecution(t, gate)
			succeedEnrolled(t, &e, "a")
			for _, r := range e.Ready() {
				if r.AgentID == "b" || r.AgentID == "reader" {
					t.Fatal("success unblocked dependent")
				}
			}
			if gate {
				if e.Status != RunRunning {
					t.Fatal("after-stage gate moved before all results", e.Status)
				}
				if err := e.PrepareIntegration(intent(e, "int-a", "att-a")); err == nil {
					t.Fatal("gate bypassed")
				}
				if err := e.ApproveGate("work", "human:rafael", t0); !errors.Is(err, ErrTransition) {
					t.Fatal("early after-stage approval accepted", err)
				}
				if err := e.MarkWorkspaceGateWait("att-a", 1, t0); !errors.Is(err, ErrTransition) {
					t.Fatal("partial stage recorded safe gate wait", err)
				}
				return
			}
			if err := e.PrepareIntegration(intent(e, "int-a", "att-a")); err != nil {
				t.Fatal(err)
			}
			if err := e.FinishIntegration("int-a", 1, IntegrationResult{Status: IntegrationIntegrated, MergeSHA: sha("c"), FinalSHA: sha("c"), ObservedTargetHead: sha("c"), Checks: passed(), Now: t0}); err != nil {
				t.Fatal(err)
			}
			found := false
			for _, r := range e.Ready() {
				found = found || r.AgentID == "b"
			}
			if !found {
				t.Fatal("integrated dependent not ready")
			}
		})
	}
}
func TestWorkspaceBindingCodecAndTampering(t *testing.T) {
	e := enrolledExecution(t, false)
	prepareEnrolled(t, &e, "a")
	wire, err := EncodeExecution(e)
	if err != nil {
		t.Fatal(err)
	}
	back, err := DecodeExecution(wire)
	if err != nil {
		t.Fatal(err)
	}
	again, _ := EncodeExecution(back)
	if !bytes.Equal(wire, again) {
		t.Fatal("codec changed bytes")
	}
	for _, change := range []func(*Execution){func(x *Execution) { x.Enrollments = append(x.Enrollments, x.Enrollments[0]) }, func(x *Execution) { x.Enrollments[0].RepositoryKey = "lib" }, func(x *Execution) { a, _, _ := x.FindAttempt("att-a"); a.Workspace.Worktree = "../escape" }, func(x *Execution) { a, _, _ := x.FindAttempt("att-a"); a.Workspace.RepositoryIdentity = "other" }} {
		probe, _ := DecodeExecution(wire)
		change(&probe)
		if err := probe.Validate(); err == nil {
			t.Fatal("forged binding accepted")
		}
	}
	plain := integrationExecution(t, false)
	old, _ := EncodeExecution(plain)
	if bytes.Contains(old, []byte("enrollments")) || bytes.Contains(old, []byte("workspace\":")) {
		t.Fatal("unenrolled encoding changed")
	}
}

func TestWorkspaceLostPrincipalPreservesValidatedResultAndStops(t *testing.T) {
	e := enrolledExecution(t, false)
	succeedEnrolled(t, &e, "a")
	a, _, _ := e.FindAttempt("att-a")
	known := *a
	if err := e.MarkWorkspaceUnknown("att-a", 1, t0); !errors.Is(err, ErrTransition) {
		t.Fatal("same runner marked lost", err)
	}
	if err := e.ClaimRunner(2, t0); err != nil {
		t.Fatal(err)
	}
	if err := e.MarkWorkspaceUnknown("att-a", 2, t0); err != nil {
		t.Fatal(err)
	}
	a, _, _ = e.FindAttempt("att-a")
	if a.Status != known.Status || !reflect.DeepEqual(a.Artifacts, known.Artifacts) || !reflect.DeepEqual(a.Commands, known.Commands) {
		t.Fatal("known terminal evidence rewritten")
	}
	if e.Status != RunRecoveryRequired || len(e.Ready()) != 0 {
		t.Fatal("uncertain integration unblocked work")
	}
	in := intent(e, "int-a", "att-a")
	if err := e.PrepareIntegration(in); !errors.Is(err, ErrUnknownOutstanding) {
		t.Fatal("uncertainty replayed", err)
	}
	wire, err := EncodeExecution(e)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := DecodeExecution(wire); err != nil {
		t.Fatal(err)
	}
}

func TestWorkspaceGateWaitIsExplicitSafeStop(t *testing.T) {
	e := enrolledExecution(t, false)
	stages := cloneStages(e.Workflow.Stages[:1])
	stages[0].HumanGate = true
	stages[0].Agents = stages[0].Agents[:2]
	stages[0].Agents[1].DependsOn = nil
	var err error
	e, err = NewExecution(NewExecutionInput{ExecutionID: e.ExecutionID, ProjectID: e.ProjectID, WorkItem: e.WorkItem, Repositories: e.Repositories, WorkflowID: e.Workflow.ID, Revision: e.Workflow.Revision, Stages: stages, EnrollIntegration: true, Now: t0})
	if err != nil {
		t.Fatal(err)
	}
	if err = e.ClaimRunner(1, t0); err != nil {
		t.Fatal(err)
	}
	succeedEnrolled(t, &e, "a")
	if e.Status != RunRunning {
		t.Fatal("partial stage reached after-stage gate", e.Status)
	}
	if err := e.MarkWorkspaceGateWait("att-a", 1, t0); !errors.Is(err, ErrTransition) {
		t.Fatal("partial stage safe-wait accepted", err)
	}
	succeedEnrolled(t, &e, "b")
	if e.Status != RunAwaitingGate {
		t.Fatal("completed results missed gate", e.Status)
	}
	if err := e.MarkWorkspaceGateWait("att-a", 1, t0); err != nil {
		t.Fatal(err)
	}
	if len(e.Ready()) != 0 {
		t.Fatal("gate wait released dependent", e.Ready())
	}
	a, _, _ := e.FindAttempt("att-a")
	if !a.WorkspaceGateWait || a.WorkspaceUncertain {
		t.Fatal("known wait uncertain")
	}
	if err := e.ClaimRunner(2, t0); err != nil {
		t.Fatal(err)
	}
	if err := e.MarkWorkspaceGateWait("att-a", 2, t0); !errors.Is(err, ErrTransition) {
		t.Fatal("new runner rewrote observed wait", err)
	}
	if err := e.ApproveGate("work", "human:rafael", t0); err != nil {
		t.Fatal(err)
	}
	if err := e.PrepareIntegration(intent(e, "int-a", "att-a")); err != nil {
		t.Fatal(err)
	}
	if err := e.FinishIntegration("int-a", 2, IntegrationResult{Status: IntegrationNotIntegrated, ObservedTargetHead: sha("b"), Diagnostic: "merge_conflict", Now: t0}); err != nil {
		t.Fatal(err)
	}
	if e.Status != RunFailed || len(e.Ready()) != 0 {
		t.Fatal("failed integration released work", e.Status, e.Ready())
	}
	if err := e.PrepareIntegration(intent(e, "repeat", "att-a")); !errors.Is(err, ErrTransition) {
		t.Fatal("failed integration replayed", err)
	}
}
