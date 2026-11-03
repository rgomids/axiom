//go:build linux || darwin

package main

import (
	"context"
	"os"
	"path/filepath"
	"slices"
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

// A Project Repository without any Work Item, attempt or integration fact has
// no outcome to observe; it must neither block nor be path-checked, while the
// Work Item Repository and any Repository with real claims stay enforced.
func TestRepositoryReadinessIgnoresUnclaimedProjectRepository(t *testing.T) {
	f, paths := newExecutionRunFixtureRepositories(t, "main", "docs")
	done := f.run(t, cli.ExitSuccess, minimalRun("--authorize-external")...)
	service := f.env.service.(lifecycleService)
	packet := func() (map[string]any, map[string]workflow.RepositoryOutcome) {
		got := service.WorkflowStage(context.Background(), cli.WorkflowStageInput{Operation: "delivery-packet", Project: "external", Execution: done.Stage.ExecutionID})
		p, ok := got.Payload.(map[string]any)
		if !ok {
			t.Fatalf("packet refused: %+v", got.Result)
		}
		byKey := map[string]workflow.RepositoryOutcome{}
		for _, o := range p["repositories"].([]workflow.RepositoryOutcome) {
			byKey[o.RepositoryKey] = o
		}
		return p, byKey
	}
	p, outcomes := packet()
	if p["ready"] != true || len(outcomes) != 2 || !outcomes["main"].Ready || len(outcomes["main"].Results) != 1 || !outcomes["docs"].Ready || len(outcomes["docs"].Results) != 0 || len(outcomes["docs"].Blockers) != 0 {
		t.Fatalf("unclaimed repository blocked readiness: %+v", p)
	}

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
	observe := func(c executionComposition, x workflow.Execution) map[string]workflow.RepositoryOutcome {
		byKey := map[string]workflow.RepositoryOutcome{}
		for _, o := range c.repositoryOutcomes(context.Background(), x) {
			byKey[o.RepositoryKey] = o
		}
		return byKey
	}
	clone := func() workflow.Execution {
		wire, err := workflow.EncodeExecution(x)
		if err != nil {
			t.Fatal(err)
		}
		y, err := workflow.DecodeExecution(wire)
		if err != nil {
			t.Fatal(err)
		}
		return y
	}
	// Project admission refuses unavailable Repositories before readiness, so
	// the unclaimed path is removed only for the readiness observation itself.
	if err := os.Remove(paths["docs"]); err != nil {
		t.Fatal(err)
	}
	if o := observe(c, x); !o["main"].Ready || !o["docs"].Ready || len(o["docs"].Results) != 0 {
		t.Fatalf("unclaimed repository path observed: %+v", o)
	}
	for _, mode := range []string{"path", "identity", "digest"} {
		t.Run("work-item-"+mode, func(t *testing.T) {
			candidate, y := c, clone()
			candidate.project.Repositories = append([]local.ResolvedRepository(nil), c.project.Repositories...)
			for k, r := range candidate.project.Repositories {
				switch {
				case r.Key != "main":
				case mode == "path":
					candidate.project.Repositories[k].Path = filepath.Join(t.TempDir(), "missing")
				case mode == "identity":
					candidate.project.Repositories[k].CanonicalIdentity = "github.com/example/wrong"
				}
			}
			if mode == "digest" {
				y.Stages[0].Tasks[0].Attempts[0].Artifacts[0].Digest = workflow.DigestText([]byte("other"))
			}
			if o := observe(candidate, y); o["main"].Ready {
				t.Fatalf("wrong Work Item repository evidence ready: %+v", o)
			}
		})
	}
	t.Run("claimed-second-repository", func(t *testing.T) {
		y := clone()
		a := &y.Stages[0].Tasks[0].Attempts[0]
		a.Authority.RepositoryKey = "docs"
		for k := range a.Artifacts {
			a.Artifacts[k].RepositoryKey = "docs"
		}
		if err := y.Validate(); err != nil {
			t.Fatal(err)
		}
		o := observe(c, y)
		if o["docs"].Ready || len(o["docs"].Results) != 1 || !slices.Contains(o["docs"].Blockers, "repository_binding_mismatch") || o["main"].Ready {
			t.Fatalf("claimed second repository ignored: %+v", o)
		}
	})

	if err := os.Mkdir(paths["docs"], 0700); err != nil {
		t.Fatal(err)
	}
	a := done.Stage.Stages[0].Tasks[0].Attempts[0]
	output := filepath.Join(f.env.state, "executions", "v2-attempt-outputs", done.Stage.ProjectID, done.Stage.ExecutionID, a.AttemptID, "result")
	if err := os.WriteFile(output, []byte("forged success"), 0600); err != nil {
		t.Fatal(err)
	}
	if p, _ := packet(); p["ready"] != false {
		t.Fatal("tampered Work Item output ready")
	}
	if err := os.Remove(output); err != nil {
		t.Fatal(err)
	}
	if p, outcomes := packet(); p["ready"] != false || outcomes["main"].Ready || !outcomes["docs"].Ready {
		t.Fatalf("deleted Work Item output ready: %+v", p)
	}
}
