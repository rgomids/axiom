// Run only inside an archive of 15d0f87911b758636a922cd9ab9777ed5d05046f.
// This pre-change generator must never be used to update expected bytes.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/rgomids/axiom/internal/workflow"
	"os"
	"path/filepath"
	"time"
)

func must(err error) {
	if err != nil {
		panic(err)
	}
}
func main() {
	out := os.Args[1]
	now := time.Date(2026, 11, 24, 12, 0, 0, 0, time.UTC)
	for _, name := range []string{"single", "dag", "terminal", "prepared", "running", "multi-active"} {
		agents := []workflow.AgentSpec{{ID: "a", MaximumAttempts: 3}}
		if name != "single" {
			agents = append(agents, workflow.AgentSpec{ID: "b", MaximumAttempts: 3}, workflow.AgentSpec{ID: "join", DependsOn: []string{"a", "b"}, MaximumAttempts: 3})
		}
		e, err := workflow.NewExecution(workflow.NewExecutionInput{ExecutionID: "00000000-0000-4000-8000-000000000404", ProjectID: "synthetic", WorkItem: workflow.WorkItemLink{Provider: "local", ID: "i404", RepositoryKey: "app"}, Repositories: []workflow.RepositoryBinding{{Key: "app", Identity: "example/app"}}, WorkflowID: "i404", Revision: "r1", Stages: []workflow.StageSpec{{ID: "work", TimeoutSeconds: 60, Agents: agents}}, Now: now})
		must(err)
		if name != "single" && name != "dag" {
			must(e.ClaimRunner(1, now))
			ids := []string{"a"}
			if name == "multi-active" {
				ids = append(ids, "b")
			}
			for i, id := range ids {
				attempt := "attempt-" + id
				inv := workflow.Invocation{Runtime: "stub", Executable: "stub-bin", Model: "synthetic", Effort: "runtime-default", Instructions: "synthetic fixture", InstructionsDigest: workflow.DigestText([]byte("synthetic fixture")), Provenance: []workflow.FieldProvenance{{Field: "runtime", Source: "local_default"}, {Field: "model", Source: "workflow_agent"}, {Field: "effort", Source: "run_override"}, {Field: "instructions", Source: "workflow_agent"}}}
				must(e.PrepareAttempt(workflow.PrepareInput{Ref: workflow.TaskRef{StageID: "work", AgentID: id}, AttemptID: attempt, Generation: 1, Invocation: inv, Authority: workflow.AttemptAuthority{RepositoryKey: "app", Effects: []string{}, Actor: "user:local"}, Now: now}))
				if name != "prepared" {
					must(e.MarkStarted(attempt, 1, workflow.ProcessIdentity{PID: int64(404 + i), StartIdentity: "synthetic-start-" + id, GroupIdentity: fmt.Sprint(404 + i)}, now))
				}
				if name == "terminal" {
					must(e.CompleteAttempt(attempt, 1, workflow.AttemptResult{ExitCode: 0, Now: now}))
				}
			}
		}
		wire, err := workflow.EncodeExecution(e)
		must(err)
		back, err := workflow.DecodeExecution(wire)
		must(err)
		encoded, err := workflow.EncodeExecution(back)
		must(err)
		if !bytes.Equal(wire, encoded) {
			panic("baseline roundtrip changed")
		}
		must(os.WriteFile(filepath.Join(out, name+".json"), wire, 0600))
		digestInput, err := json.Marshal(struct {
			ID       string               `json:"id"`
			Revision string               `json:"revision"`
			Stages   []workflow.StageSpec `json:"stages"`
		}{e.Workflow.ID, e.Workflow.Revision, e.Workflow.Stages})
		must(err)
		must(os.WriteFile(filepath.Join(out, name+".digest-input.json"), digestInput, 0600))
		fmt.Println(name, workflow.DigestText(wire), e.Workflow.Digest)
	}
}
