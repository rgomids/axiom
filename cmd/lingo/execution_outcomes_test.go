//go:build linux || darwin

package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/rgomids/axiom/internal/cli"
	"github.com/rgomids/axiom/internal/local"
	"github.com/rgomids/axiom/internal/workflow"
)

func TestRepositoryReadinessReobservesLocalOutputsWithoutPR(t *testing.T) {
	f := newExecutionRunFixture(t)
	done := f.run(t, cli.ExitSuccess, minimalRun("--authorize-external")...)
	if len(done.Stage.RepositoryOutcomes) != 1 || !done.Stage.RepositoryOutcomes[0].Ready || !done.Stage.RepositoryOutcomes[0].ReadinessObserved {
		t.Fatalf("missing observed outcomes: %+v", done.Stage)
	}
	a := done.Stage.Stages[0].Tasks[0].Attempts[0]
	path := filepath.Join(f.env.state, "executions", "v2-attempt-outputs", done.Stage.ProjectID, done.Stage.ExecutionID, a.AttemptID, "result")
	if err := os.WriteFile(path, []byte("forged success"), 0600); err != nil {
		t.Fatal(err)
	}
	s := f.env.service.(lifecycleService)
	got := s.WorkflowStage(context.Background(), cli.WorkflowStageInput{Operation: "delivery-packet", Project: "external", Execution: done.Stage.ExecutionID})
	p := got.Payload.(map[string]any)
	outcomes := p["repositories"].([]workflow.RepositoryOutcome)
	if p["ready"] != false || len(outcomes) != 1 || outcomes[0].Ready {
		t.Fatalf("stale bytes ready: %+v", p)
	}
	status := f.run(t, cli.ExitSuccess, "workflow", "stage", "status", "--project", "external", "--execution", done.Stage.ExecutionID)
	if status.Stage.Status != workflow.RunCompleted || status.Stage.RepositoryOutcomes[0].Ready {
		t.Fatal("history rewritten or stale readiness accepted")
	}
}

func TestRepositoryReadinessRefusesUnsafeOrReboundPath(t *testing.T) {
	f := newExecutionRunFixture(t)
	done := f.run(t, cli.ExitSuccess, minimalRun("--authorize-external")...)
	service := f.env.service.(lifecycleService)
	resolved := service.installation.Resolve(context.Background(), done.Stage.ProjectID)
	store, err := local.NewExecutionStore(f.env.state)
	if err != nil {
		t.Fatal(err)
	}
	x, err := store.Load(context.Background(), done.Stage.ProjectID, done.Stage.ExecutionID)
	if err != nil {
		t.Fatal(err)
	}
	c := executionComposition{s: service, project: resolved.Project, store: store}
	link := filepath.Join(t.TempDir(), "repository-link")
	if err := os.Symlink(c.project.Repositories[0].Path, link); err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"missing", "symlink", "identity"} {
		t.Run(mode, func(t *testing.T) {
			candidate := c
			candidate.project.Repositories = append([]local.ResolvedRepository(nil), c.project.Repositories...)
			switch mode {
			case "missing":
				candidate.project.Repositories[0].Path = filepath.Join(t.TempDir(), "missing")
			case "symlink":
				candidate.project.Repositories[0].Path = link
			case "identity":
				candidate.project.Repositories[0].CanonicalIdentity = "github.com/example/wrong"
			}
			for _, o := range candidate.repositoryOutcomes(context.Background(), x) {
				if o.Ready {
					t.Fatal("wrong repository path ready")
				}
			}
		})
	}
}
