package workflow

import (
	"errors"
	"os"
	"reflect"
	"testing"
)

func i404Admission(t *testing.T, limit *int) Execution {
	t.Helper()
	wire, err := os.ReadFile("testdata/i404/dag.json")
	if err != nil {
		t.Fatal(err)
	}
	x, err := DecodeExecution(wire)
	if err != nil {
		t.Fatal(err)
	}
	x.Workflow.Stages[0].Concurrency = limit
	x.Workflow.Stages[0].Agents[2].DependsOn = nil
	x.Workflow.Digest, err = workflowDigest(x.Workflow)
	if err != nil {
		t.Fatal(err)
	}
	if err = x.ClaimRunner(1, t0); err != nil {
		t.Fatal(err)
	}
	return x
}

func i404Prepare(x *Execution, agent, attempt string, retry bool, inflight ...InflightAttempt) error {
	return x.PrepareAttempt(PrepareInput{Ref: TaskRef{"work", agent}, AttemptID: attempt, Generation: x.RunnerGeneration, Invocation: testInvocation("do"), Authority: AttemptAuthority{"app", []string{}, "user:local"}, Now: t0, Retry: retry, Inflight: inflight})
}

func TestI404AdmissionUnionAndCachedReady(t *testing.T) {
	two := 2
	x := i404Admission(t, &two)
	cached := x.Ready()
	if len(cached) != 2 {
		t.Fatal(cached)
	}
	for i, ref := range cached {
		if err := i404Prepare(&x, ref.AgentID, []string{"a1", "b1"}[i], false); err != nil {
			t.Fatal(err)
		}
	}
	before := x.Revision
	if err := i404Prepare(&x, "join", "c1", false); !errors.Is(err, ErrStageCapacity) || x.Revision != before {
		t.Fatal("cached readiness overprepared", err)
	}
	if len(x.Ready()) != 0 {
		t.Fatal("saturated Ready", x.Ready())
	}
	// Duplicate active IDs count once, including their durable representation.
	workers := []InflightAttempt{{TaskRef{"work", "a"}, "a1"}, {TaskRef{"work", "a"}, "a1"}, {TaskRef{"work", "b"}, "b1"}}
	if remaining, err := x.stageCapacity("work", workers); err != nil || remaining != 0 {
		t.Fatal(remaining, err)
	}
	if err := x.MarkStarted("a1", 1, ProcessIdentity{PID: 404, StartIdentity: "s", GroupIdentity: "404"}, t0); err != nil {
		t.Fatal(err)
	}
	if err := x.CompleteAttempt("a1", 1, AttemptResult{Now: t0}); err != nil {
		t.Fatal(err)
	}
	if len(x.Ready()) != 1 {
		t.Fatal("durable slot did not return")
	}
	if err := i404Prepare(&x, "join", "c1", false, workers...); !errors.Is(err, ErrStageCapacity) {
		t.Fatal("terminal inflight freed slot", err)
	}
	for _, bad := range []InflightAttempt{{TaskRef{"work", "b"}, "a1"}, {TaskRef{"missing", "a"}, "a1"}, {TaskRef{"work", "a"}, "missing"}} {
		if err := i404Prepare(&x, "join", "c1", false, bad); !errors.Is(err, ErrTransition) {
			t.Fatal("corrupt correlation accepted", err)
		}
	}
	if err := i404Prepare(&x, "join", "c1", false, InflightAttempt{TaskRef{"work", "b"}, "b1"}); err != nil {
		t.Fatal("genuine returned slot blocked", err)
	}
}

func TestI404AbsentAdmissionAndHistoricalMultiActive(t *testing.T) {
	x := i404Admission(t, nil)
	frozen := cloneStages(x.Workflow.Stages)
	if len(x.Ready()) != 1 {
		t.Fatal("absence must narrow dispatch")
	}
	if err := i404Prepare(&x, "a", "a1", false); err != nil {
		t.Fatal(err)
	}
	if err := i404Prepare(&x, "b", "b1", false); !errors.Is(err, ErrStageCapacity) {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(frozen, x.Workflow.Stages) {
		t.Fatal("fallback persisted")
	}
	wire, err := os.ReadFile("testdata/i404/multi-active.json")
	if err != nil {
		t.Fatal(err)
	}
	x, err = DecodeExecution(wire)
	if err != nil {
		t.Fatal(err)
	}
	if len(x.Ready()) != 0 {
		t.Fatal("old active occupancy ignored")
	}
	// Keep the join independent solely to test admission, not its dependency gate.
	x.Workflow.Stages[0].Agents[2].DependsOn = nil
	x.Workflow.Digest, _ = workflowDigest(x.Workflow)
	if err := i404Prepare(&x, "join", "c1", false); !errors.Is(err, ErrStageCapacity) {
		t.Fatal(err)
	}
	if err := x.MarkUnknown("attempt-a", 1, "synthetic", t0); err != nil {
		t.Fatal(err)
	}
	if err := i404Prepare(&x, "join", "c1", false); !errors.Is(err, ErrUnknownOutstanding) {
		t.Fatal("unknown became permission", err)
	}
}

func TestI404RetryCapacityAndControlPrecedence(t *testing.T) {
	two := 2
	x := i404Admission(t, &two)
	if err := i404Prepare(&x, "a", "a1", false); err != nil {
		t.Fatal(err)
	}
	if err := i404Prepare(&x, "b", "b1", false); err != nil {
		t.Fatal(err)
	}
	if err := x.MarkSpawnFailed("a1", 1, "synthetic", t0); err != nil {
		t.Fatal(err)
	}
	worker := InflightAttempt{TaskRef{"work", "a"}, "a1"}
	before := x.Revision
	if err := i404Prepare(&x, "a", "a2", true, worker); !errors.Is(err, ErrStageCapacity) || x.Revision != before || len(x.Stages[0].Tasks[0].Attempts) != 1 {
		t.Fatal("saturation consumed retry", err)
	}
	if err := i404Prepare(&x, "a", "a2", true); err != nil {
		t.Fatal("retry slot did not return", err)
	}
	if err := x.RequestCancel("user:local", t0); err != nil {
		t.Fatal(err)
	}
	if err := i404Prepare(&x, "join", "c1", false); !errors.Is(err, ErrCancelled) {
		t.Fatal("capacity outranked cancel", err)
	}
	if err := x.PrepareAttempt(PrepareInput{Generation: 2}); !errors.Is(err, ErrStaleRunner) {
		t.Fatal("capacity outranked fence", err)
	}
}
