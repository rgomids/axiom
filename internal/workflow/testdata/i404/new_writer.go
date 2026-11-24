// Produce new-member bytes for the pinned old-reader refusal probe only.
package main

import (
	"fmt"
	"github.com/rgomids/axiom/internal/workflow"
	"os"
	"path/filepath"
	"time"
)

func main() {
	for _, n := range []int{1, 2} {
		e, err := workflow.NewExecution(workflow.NewExecutionInput{ExecutionID: "00000000-0000-4000-8000-000000000404", ProjectID: "synthetic", WorkItem: workflow.WorkItemLink{Provider: "local", ID: "i404", RepositoryKey: "app"}, Repositories: []workflow.RepositoryBinding{{Key: "app", Identity: "example/app"}}, WorkflowID: "i404", Revision: "r1", Now: time.Unix(1, 0), Stages: []workflow.StageSpec{{ID: "work", TimeoutSeconds: 60, Concurrency: &n, Agents: []workflow.AgentSpec{{ID: "a"}, {ID: "b"}}}}})
		if err != nil {
			panic(err)
		}
		wire, err := workflow.EncodeExecution(e)
		if err != nil {
			panic(err)
		}
		if err = os.WriteFile(filepath.Join(os.Args[1], fmt.Sprintf("explicit-%d.json", n)), wire, 0600); err != nil {
			panic(err)
		}
	}
}
