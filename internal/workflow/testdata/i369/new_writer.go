package main

import (
	"github.com/rgomids/axiom/internal/workflow"
	"os"
	"time"
)

func main() {
	e, err := workflow.NewExecution(workflow.NewExecutionInput{ExecutionID: "00000000-0000-4000-8000-000000000042", ProjectID: "proj", WorkItem: workflow.WorkItemLink{Provider: "local", ID: "synthetic", RepositoryKey: "app"}, Repositories: []workflow.RepositoryBinding{{Key: "app", Identity: "example/app"}}, WorkflowID: "synthetic", Revision: "r1", Stages: []workflow.StageSpec{{ID: "work", TimeoutSeconds: 60, Agents: []workflow.AgentSpec{{ID: "builder", MaximumAttempts: 2}}}}, Now: time.Unix(1, 0)})
	if err != nil {
		panic(err)
	}
	e.AttemptPolicy = workflow.SpawnBoundaryPolicy
	wire, err := workflow.EncodeExecution(e)
	if err != nil {
		panic(err)
	}
	if err = os.WriteFile("/tmp/i369-new-policy.json", wire, 0600); err != nil {
		panic(err)
	}
	if err = e.ClaimRunner(1, time.Unix(2, 0)); err != nil {
		panic(err)
	}
	inv := workflow.Invocation{Runtime: "stub", Executable: "stub-bin", Model: "synthetic", Effort: "runtime-default", Instructions: "work", InstructionsDigest: workflow.DigestText([]byte("work")), Provenance: []workflow.FieldProvenance{{Field: "runtime", Source: "local_default"}, {Field: "model", Source: "local_default"}, {Field: "effort", Source: "local_default"}, {Field: "instructions", Source: "workflow_agent"}}}
	if err = e.PrepareAttempt(workflow.PrepareInput{Ref: workflow.TaskRef{StageID: "work", AgentID: "builder"}, AttemptID: "a1", Generation: 1, Invocation: inv, Authority: workflow.AttemptAuthority{RepositoryKey: "app", Actor: "user:local", Effects: []string{}}, Now: time.Unix(3, 0)}); err != nil {
		panic(err)
	}
	if err = e.MarkLaunchRefused("a1", 1, workflow.LaunchFailure{Reason: "selector_conflict", Phase: "selector_inspection", Source: "user"}, "none", time.Unix(4, 0)); err != nil {
		panic(err)
	}
	wire, err = workflow.EncodeExecution(e)
	if err != nil {
		panic(err)
	}
	if err = os.WriteFile("/tmp/i369-new-refusal.json", wire, 0600); err != nil {
		panic(err)
	}
}
