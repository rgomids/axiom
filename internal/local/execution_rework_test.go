package local

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/rgomids/axiom/internal/workflow"
)

func rejectedStoreFixture(t *testing.T) (ExecutionStore, workflow.Execution) {
	t.Helper()
	s, _ := newTestExecutionStore(t)
	ctx := context.Background()
	x := storeTestExecution(t)
	if err := s.Create(ctx, x); err != nil {
		t.Fatal(err)
	}
	lease := claim(t, s)
	x, err := s.Update(ctx, x.ProjectID, x.ExecutionID, lease, func(e *workflow.Execution) error {
		if err := e.PrepareAttempt(workflow.PrepareInput{Ref: workflow.TaskRef{StageID: "work", AgentID: "builder"}, AttemptID: "a1", Generation: lease.Epoch, Invocation: storeInvocation(), Authority: workflow.AttemptAuthority{RepositoryKey: "app", Actor: "user:old"}, Now: storeT0}); err != nil {
			return err
		}
		if err := e.MarkStarted("a1", lease.Epoch, workflow.ProcessIdentity{PID: 100, StartIdentity: "s", GroupIdentity: "100"}, storeT0); err != nil {
			return err
		}
		return e.CompleteAttempt("a1", lease.Epoch, workflow.AttemptResult{ExitCode: 0, Now: storeT0})
	})
	lease.Close()
	if err != nil {
		t.Fatal(err)
	}
	digest, _ := x.StageResultDigest("work")
	x, err = s.Update(ctx, x.ProjectID, x.ExecutionID, nil, func(e *workflow.Execution) error {
		return e.RejectResult("work", "human:operator", digest, workflow.ArtifactRef{Name: "human-review", RetainedID: "evidence-1", RepositoryKey: "app", Digest: workflow.DigestText([]byte("review")), Size: 1}, storeT0)
	})
	if err != nil {
		t.Fatal(err)
	}
	return s, x
}

func linkedFresh(t *testing.T, source workflow.Execution) workflow.Execution {
	t.Helper()
	x, err := workflow.NewExecution(workflow.NewExecutionInput{ExecutionID: "00000000-0000-4000-8000-000000000099", ProjectID: source.ProjectID, WorkItem: source.WorkItem, Repositories: source.Repositories, WorkflowID: source.Workflow.ID, Revision: source.Workflow.Revision, Stages: source.Workflow.Stages, Now: storeT0})
	if err != nil {
		t.Fatal(err)
	}
	link, err := source.RejectedSource()
	if err != nil {
		t.Fatal(err)
	}
	x.ReworkOf = &link
	return x
}

func TestExecutionStoreRejectedHistoryAndLinkAreImmutable(t *testing.T) {
	s, source := rejectedStoreFixture(t)
	ctx := context.Background()
	for _, mode := range []string{"erase", "actor", "evidence", "result", "attempt", "erase-results"} {
		wire, _ := workflow.EncodeExecution(source)
		next, _ := workflow.DecodeExecution(wire)
		switch mode {
		case "erase":
			next.Rejection = nil
			next.Status = workflow.RunCompleted
		case "actor":
			next.Rejection.Actor = "human:other"
		case "evidence":
			next.Rejection.Evidence.RetainedID = "other-evidence"
		case "result":
			next.Stages[0].Tasks[0].Attempts[0].Diagnostic = "rewritten"
			next.Rejection.ResultDigest, _ = next.StageResultDigest("work")
		case "attempt":
			next.Stages[0].Tasks[0].Attempts[0].Authority.Actor = "user:other"
			next.Rejection.ResultDigest, _ = next.StageResultDigest("work")
		case "erase-results":
			next.Stages[0].Tasks[0].Attempts = nil
			next.Stages[0].Status = workflow.StagePending
			next.Status = workflow.RunReady
			next.Rejection = nil
		default:
			t.Fatalf("unhandled mode %q", mode)
		}
		next.Revision++
		if err := next.Validate(); err != nil {
			t.Fatalf("%s invalid fixture: %v", mode, err)
		}
		if err := s.Save(ctx, next, source.Revision, nil); !errors.Is(err, ErrConflict) {
			t.Fatalf("%s rewrite: %v", mode, err)
		}
	}
	next := linkedFresh(t, source)
	if err := s.Create(ctx, next); err != nil {
		t.Fatal(err)
	}
	loaded, err := s.Load(ctx, next.ProjectID, next.ExecutionID)
	if err != nil || loaded.ReworkOf == nil || len(loaded.Stages[0].Tasks[0].Attempts) != 0 || len(loaded.Deliveries) != 0 || loaded.RunnerGeneration != 0 {
		t.Fatal("authority/state inherited", err)
	}
	changed := loaded
	changed.ReworkOf = nil
	changed.Revision++
	if err := s.Save(ctx, changed, loaded.Revision, nil); !errors.Is(err, ErrConflict) {
		t.Fatal("link erased", err)
	}
	got, _ := s.Load(ctx, source.ProjectID, source.ExecutionID)
	if !reflect.DeepEqual(got, source) {
		t.Fatal("source changed")
	}
}

func TestExecutionStoreRefusesForgedReworkAndInheritedState(t *testing.T) {
	s, source := rejectedStoreFixture(t)
	ctx := context.Background()
	for _, mode := range []string{"source", "stage", "attempt", "repository", "identity", "project", "work-item", "attempt-and-invocation", "rejection-state", "cancellation", "timestamp", "rejection-digest", "inherited-before-gate"} {
		next := linkedFresh(t, source)
		switch mode {
		case "source":
			next.ReworkOf.ExecutionID = "00000000-0000-4000-8000-000000000088"
		case "stage":
			next.ReworkOf.StageID = "other"
		case "attempt":
			next.ReworkOf.ResultDigest = workflow.DigestText([]byte("wrong attempt"))
		case "repository":
			next.WorkItem.RepositoryKey = "other"
			next.Repositories = []workflow.RepositoryBinding{{Key: "other", Identity: "github.com/example/app"}}
			next.ReworkOf.RepositoryKey = "other"
		case "identity":
			next.Repositories = []workflow.RepositoryBinding{{Key: "app", Identity: "github.com/example/other"}}
		case "project":
			next.ProjectID = "other"
		case "work-item":
			next.WorkItem.ID = "AXM-99"
		case "attempt-and-invocation":
			next.Stages = source.Stages
			next.Status = workflow.RunCompleted
		case "rejection-state":
			next.Rejection = source.Rejection
			next.Stages = source.Stages
			next.Status = workflow.RunRejected
		case "cancellation":
			next.Cancel = &workflow.CancelIntent{Actor: "user:old", RequestedAt: storeT0}
			next.Status = workflow.RunCancelled
		case "timestamp":
			next.UpdatedAt = storeT0.Add(1)
		case "rejection-digest":
			next.ReworkOf.RejectionDigest = workflow.DigestText([]byte("old grant"))
		case "inherited-before-gate":
			link := next.ReworkOf
			next, _ = workflow.NewExecution(workflow.NewExecutionInput{ExecutionID: next.ExecutionID, ProjectID: next.ProjectID, WorkItem: next.WorkItem, Repositories: next.Repositories, WorkflowID: "selected", Revision: "r1", Stages: []workflow.StageSpec{{ID: "work", BeforeGate: true, TimeoutSeconds: 60, Agents: []workflow.AgentSpec{{ID: "builder"}}}}, Now: storeT0})
			next.ReworkOf = link
			if err := next.ApproveGate("work", "human:old", storeT0); err != nil {
				t.Fatal(err)
			}
			next.Revision = 1
			if err := next.Validate(); err != nil {
				t.Fatalf("valid-shaped inherited gate: %v", err)
			}
		default:
			t.Fatalf("unhandled mode %q", mode)
		}
		if err := s.Create(ctx, next); err == nil {
			t.Fatalf("%s forged rework accepted", mode)
		}
	}
}
