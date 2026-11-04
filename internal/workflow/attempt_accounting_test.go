package workflow

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"testing"
)

func accountingExecution(t *testing.T, limit int) Execution {
	t.Helper()
	e, err := NewExecution(NewExecutionInput{ExecutionID: testExecutionID, ProjectID: "proj", WorkItem: WorkItemLink{"linear", "AXM-42", "app"}, Repositories: []RepositoryBinding{{"app", "example/app"}}, WorkflowID: "minimal", Revision: "r1", Now: t0, Stages: []StageSpec{{ID: "work", TimeoutSeconds: 60, Agents: []AgentSpec{{ID: "builder", MaximumAttempts: limit}}}}})
	if err != nil {
		t.Fatal(err)
	}
	e.AttemptPolicy = SpawnBoundaryPolicy
	if err = e.ClaimRunner(1, t0); err != nil {
		t.Fatal(err)
	}
	return e
}
func accountPrepare(e *Execution, n int) error {
	return e.PrepareAttempt(PrepareInput{Ref: TaskRef{"work", "builder"}, AttemptID: fmt.Sprintf("a%d", n), Generation: e.RunnerGeneration, Invocation: testInvocation("do"), Authority: AttemptAuthority{"app", []string{}, "user:local"}, Now: t0, Retry: n > 1, WorkspaceDisposition: map[bool]string{true: "preserve"}[n > 1]})
}
func TestI369AccountingBudgets(t *testing.T) {
	for _, limit := range []int{1, 2, 10} {
		for refusals := 0; refusals <= 2; refusals++ {
			t.Run(fmt.Sprintf("%d/%d", limit, refusals), func(t *testing.T) {
				e := accountingExecution(t, limit)
				n := 1
				for r := 0; r < refusals; r++ {
					if err := accountPrepare(&e, n); err != nil {
						t.Fatal(err)
					}
					if err := e.MarkLaunchRefused(fmt.Sprintf("a%d", n), 1, LaunchFailure{Reason: "selector_conflict", Phase: "selector_inspection"}, "none", t0); err != nil {
						t.Fatal(err)
					}
					n++
				}
				if refusals == 2 {
					if err := accountPrepare(&e, n); err == nil {
						t.Fatal("third record allowed")
					}
					return
				}
				for c := 0; c < limit; c++ {
					if err := accountPrepare(&e, n); err != nil {
						t.Fatal(err)
					}
					charged, free := AttemptCounts(e.Stages[0].Tasks[0].Attempts)
					if charged != c+1 || free != refusals {
						t.Fatal("pending refunded")
					}
					if err := e.ReserveLaunch(fmt.Sprintf("a%d", n), 1, t0); err != nil {
						t.Fatal(err)
					}
					if err := e.MarkLaunchRefused(fmt.Sprintf("a%d", n), 1, LaunchFailure{Reason: "selector_conflict", Phase: "selector_inspection"}, "none", t0); err == nil {
						t.Fatal("charged refunded")
					}
					if err := e.MarkSpawnFailed(fmt.Sprintf("a%d", n), 1, "spawn_failed", t0); err != nil {
						t.Fatal(err)
					}
					n++
				}
				if err := accountPrepare(&e, n); err == nil {
					t.Fatal("charged budget exceeded")
				}
				wire, err := EncodeExecution(e)
				if err != nil {
					t.Fatal(err)
				}
				if _, err = DecodeExecution(wire); err != nil {
					t.Fatal(err)
				}
			})
		}
	}
}
func TestI369AccountingStrictness(t *testing.T) {
	e := accountingExecution(t, 2)
	if err := accountPrepare(&e, 1); err != nil {
		t.Fatal(err)
	}
	if err := e.MarkStarted("a1", 1, ProcessIdentity{PID: 1, StartIdentity: "s", GroupIdentity: "1"}, t0); err == nil {
		t.Fatal("unreserved start")
	}
	wire, err := EncodeExecution(e)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct{ old, new string }{
		{`"spawn-boundary-v1"`, `"unknown"`}, {`"accounting":"pending",`, ``}, {`"pending"`, `"unknown"`}, {`"attemptPolicy":`, `"extra":true,"attemptPolicy":`},
	} {
		if _, err := DecodeExecution(bytes.Replace(wire, []byte(c.old), []byte(c.new), 1)); err == nil {
			t.Fatal("unknown/missing policy metadata")
		}
	}
	if err := e.MarkUnknown("a1", 1, "runner_lost", t0); err != nil {
		t.Fatal(err)
	}
	if c, r := AttemptCounts(e.Stages[0].Tasks[0].Attempts); c != 1 || r != 0 {
		t.Fatal("unknown refunded")
	}
}

func TestI369FrozenAccountingOracle(t *testing.T) {
	wire, err := os.ReadFile("testdata/i369/budget-oracle.json")
	if err != nil {
		t.Fatal(err)
	}
	var oracle struct {
		Examples []struct {
			MaximumAttempts int  `json:"maximumAttempts"`
			Charged         int  `json:"chargedIncludingPending"`
			Refusals        int  `json:"lifetimeRefusals"`
			Prepare         bool `json:"prepare"`
		} `json:"examples"`
	}
	if err = json.Unmarshal(wire, &oracle); err != nil {
		t.Fatal(err)
	}
	for _, c := range oracle.Examples {
		task := TaskRun{}
		for n := 0; n < c.Charged; n++ {
			task.Attempts = append(task.Attempts, Attempt{Accounting: AccountingPending})
		}
		for n := 0; n < c.Refusals; n++ {
			task.Attempts = append(task.Attempts, Attempt{Accounting: AccountingRefusal})
		}
		e := Execution{AttemptPolicy: SpawnBoundaryPolicy}
		if e.attemptBudgetAvailable(task, AgentSpec{MaximumAttempts: c.MaximumAttempts}) != c.Prepare {
			t.Fatal("approved budget oracle mismatch", c)
		}
	}
}
func TestI369RefusalAllowanceSurvivesLaunchAndRestart(t *testing.T) {
	e := accountingExecution(t, 2)
	if err := accountPrepare(&e, 1); err != nil {
		t.Fatal(err)
	}
	failure := LaunchFailure{Reason: "selector_conflict", Phase: "selector_inspection"}
	if err := e.MarkLaunchRefused("a1", 1, failure, "none", t0); err != nil {
		t.Fatal(err)
	}
	if err := accountPrepare(&e, 2); err != nil {
		t.Fatal(err)
	}
	if err := e.ReserveLaunch("a2", 1, t0); err != nil {
		t.Fatal(err)
	}
	if err := e.MarkStarted("a2", 1, ProcessIdentity{PID: 42, StartIdentity: "s", GroupIdentity: "42"}, t0); err != nil {
		t.Fatal(err)
	}
	if err := e.CompleteAttempt("a2", 1, AttemptResult{ExitCode: 3, Now: t0}); err != nil {
		t.Fatal(err)
	}
	wire, err := EncodeExecution(e)
	if err != nil {
		t.Fatal(err)
	}
	e, err = DecodeExecution(wire)
	if err != nil {
		t.Fatal(err)
	}
	if err = e.ClaimRunner(2, t0); err != nil {
		t.Fatal(err)
	}
	if err = accountPrepare(&e, 3); err != nil {
		t.Fatal(err)
	}
	if err = e.MarkLaunchRefused("a3", 2, failure, "prepared", t0); err != nil {
		t.Fatal(err)
	}
	if err = accountPrepare(&e, 4); err == nil {
		t.Fatal("refusal allowance reset after launch/restart")
	}
	if charged, refusals := AttemptCounts(e.Stages[0].Tasks[0].Attempts); charged != 1 || refusals != 2 {
		t.Fatal("lifetime accounting changed")
	}
}
