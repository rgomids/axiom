package workflow

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"os"
	"testing"
)

func TestI369LegacyBytes(t *testing.T) {
	for name, digest := range map[string]string{
		"active":   "95151550d70e9582beeab7db681939b8a294d698bf23ef942e118e988edd67c6",
		"terminal": "aaf99b86fdaec6ffaa3a34c0d75ab5b67d2209b03b8cacb86d356e0000ef8943",
	} {
		t.Run(name, func(t *testing.T) {
			wire, err := os.ReadFile("testdata/i369/legacy-" + name + ".json")
			if err != nil {
				t.Fatal(err)
			}
			if fmt.Sprintf("%x", sha256.Sum256(wire)) != digest {
				t.Fatal("frozen historical bytes changed")
			}
			e, err := DecodeExecution(wire)
			if err != nil {
				t.Fatal(err)
			}
			back, err := EncodeExecution(e)
			if err != nil || !bytes.Equal(back, wire) {
				t.Fatal("historical round trip changed", err)
			}
		})
	}
}

func TestI369LegacyBudget(t *testing.T) {
	for _, limit := range []int{1, 2, 10} {
		t.Run(fmt.Sprint(limit), func(t *testing.T) {
			e, err := NewExecution(NewExecutionInput{ExecutionID: testExecutionID, ProjectID: "proj", WorkItem: WorkItemLink{"linear", "AXM-42", "app"}, Repositories: []RepositoryBinding{{"app", "example/app"}}, WorkflowID: "minimal", Revision: "r1", Now: t0, Stages: []StageSpec{{ID: "work", TimeoutSeconds: 60, Agents: []AgentSpec{{ID: "builder", MaximumAttempts: limit}}}}})
			if err != nil {
				t.Fatal(err)
			}
			if err = e.ClaimRunner(1, t0); err != nil {
				t.Fatal(err)
			}
			for n := 0; n <= limit; n++ {
				err = e.PrepareAttempt(PrepareInput{Ref: TaskRef{"work", "builder"}, AttemptID: fmt.Sprintf("a%d", n), Generation: 1, Invocation: testInvocation("do"), Authority: AttemptAuthority{"app", []string{}, "user:local"}, Now: t0, Retry: n > 0})
				if n == limit {
					if err == nil {
						t.Fatal("historical refusal refunded")
					}
					break
				}
				if err != nil {
					t.Fatal(err)
				}
				if err = e.MarkSpawnFailed(fmt.Sprintf("a%d", n), 1, "launch_refused", t0); err != nil {
					t.Fatal(err)
				}
			}
			if len(e.Stages[0].Tasks[0].Attempts) != limit {
				t.Fatal("budget changed")
			}
		})
	}
}

func TestI369FailureCodec(t *testing.T) {
	e := twoStageExecution(t, false)
	prepare(t, &e, TaskRef{"work", "builder"}, "a1")
	if err := e.MarkLaunchFailed("a1", 1, LaunchFailure{Reason: "selector_conflict", Phase: "selector_inspection", Source: "user"}, t0); err != nil {
		t.Fatal(err)
	}
	wire, err := EncodeExecution(e)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct{ old, new string }{
		{`"selector_conflict"`, `"unknown"`}, {`"selector_inspection"`, `"unknown"`}, {`"user"`, `"private-path"`},
		{`"launchFailure":{`, `"launchFailure":{"extra":true,`},
	} {
		if _, err := DecodeExecution(bytes.Replace(wire, []byte(c.old), []byte(c.new), 1)); err == nil {
			t.Fatal("unknown failure accepted")
		}
	}
	back, err := DecodeExecution(wire)
	if err != nil || *back.Stages[0].Tasks[0].Attempts[0].LaunchFailure != *e.Stages[0].Tasks[0].Attempts[0].LaunchFailure {
		t.Fatal("failure not retained", err)
	}
}
