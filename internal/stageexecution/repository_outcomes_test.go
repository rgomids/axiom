//go:build linux || darwin

package stageexecution

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/rgomids/axiom/internal/executiongraph"
	"github.com/rgomids/axiom/internal/gitworkspace"
	"github.com/rgomids/axiom/internal/workflow"
)

type repositoryRunner map[string]*integrationFixture

func (r repositoryRunner) RunObserved(ctx context.Context, inv executiongraph.Invocation, started func(*executiongraph.ProcessCorrelation) error) executiongraph.ProcessResult {
	if inv.Argv[0] == commandGateShell {
		return (executiongraph.OSProcessRunner{}).RunObserved(ctx, inv, started)
	}
	return (gitRunner{r[inv.Argv[1]]}).RunObserved(ctx, inv, started)
}

type repositoryLauncher struct {
	gitLauncher
	repositories map[string]*integrationFixture
}

func (l repositoryLauncher) Collect(ctx context.Context, r LaunchRequest, p executiongraph.ProcessResult) (Outcome, error) {
	out, err := l.gitLauncher.Collect(ctx, r, p)
	if err != nil {
		return out, err
	}
	data, err := os.ReadFile(filepath.Join(l.repositories[r.Ref.AgentID].root, "agent-"+r.Ref.AgentID, "result.txt"))
	if err != nil {
		return out, err
	}
	out.Artifacts = []workflow.ArtifactRef{{Name: "result", RepositoryKey: r.Attempt.Authority.RepositoryKey, Digest: workflow.DigestText(data), Size: int64(len(data))}}
	return out, nil
}

func TestRepositoryOutcomesTwoIndependentBindingsOneExecution(t *testing.T) {
	ctx := context.Background()
	a := newIntegrationFixture(t, map[string]edit{"a": commitFile("result.txt", "app result")})
	b := newIntegrationFixture(t, map[string]edit{"b": commitFile("result.txt", "lib result")})
	fixtures := map[string]*integrationFixture{"a": a, "b": b}
	keys := map[string]string{"a": "app", "b": "lib"}
	effects := []string{"workspace_write", "validation_command"}
	for agent, f := range fixtures {
		if _, err := f.git.PrepareAgentWorktree(ctx, "agent-"+agent, "refs/heads/axiom/result", f.base); err != nil {
			t.Fatal(err)
		}
	}
	x, err := workflow.NewExecution(workflow.NewExecutionInput{ExecutionID: secondExecutionID, ProjectID: "proj", WorkItem: workflow.WorkItemLink{Provider: "linear", ID: "AXM-42", RepositoryKey: "app"}, Repositories: []workflow.RepositoryBinding{{Key: "app", Identity: integrationIdentity}, {Key: "lib", Identity: "github.com/example/lib"}}, WorkflowID: "two-repos", Revision: "r1", Now: time.Now(), Stages: []workflow.StageSpec{{ID: "work", Agents: []workflow.AgentSpec{{ID: "a", Effects: effects}, {ID: "b", Effects: effects}}, Outputs: []string{"result"}, Commands: a.checks, TimeoutSeconds: 30}}})
	if err != nil {
		t.Fatal(err)
	}
	if err := a.store.Create(ctx, x); err != nil {
		t.Fatal(err)
	}
	engine := a.engine
	engine.Runner = repositoryRunner(fixtures)
	engine.Launcher = repositoryLauncher{repositories: fixtures}
	engine.Resolve = func(_ context.Context, _ workflow.Execution, r workflow.TaskRef) (workflow.Invocation, workflow.AttemptAuthority, error) {
		return stubInvocation(), workflow.AttemptAuthority{RepositoryKey: keys[r.AgentID], Effects: effects, Actor: "user:local"}, nil
	}
	x, err = engine.Run(ctx, "proj", x.ExecutionID, RunOptions{})
	if err != nil {
		t.Fatal(err)
	}
	for _, agent := range []string{"a", "b"} {
		f := fixtures[agent]
		identity := integrationIdentity
		if agent == "b" {
			identity = "github.com/example/lib"
		}
		req := IntegrationRequest{IntegrationID: "integrate-" + agent, SourceAttemptID: attemptFor(t, x, agent).AttemptID, RepositoryKey: keys[agent], RepositoryIdentity: identity, Worktree: "agent-" + agent, SourceBranch: "refs/heads/axiom/result", BaseSHA: f.base, TargetRef: integrationTarget, ExpectedTargetHead: f.base}
		x, err = engine.Integrate(ctx, "proj", x.ExecutionID, f.git, req)
		if err != nil {
			t.Fatal(err)
		}
	}
	outcomes := x.RepositoryOutcomes()
	if len(outcomes) != 2 || len(outcomes[0].Blockers) != 0 || len(outcomes[1].Blockers) != 0 {
		t.Fatalf("outcomes: %+v", outcomes)
	}
	for k, agent := range []string{"a", "b"} {
		o := outcomes[k]
		f := fixtures[agent]
		if len(o.Results) != 1 || len(o.Integrations) != 1 || o.Results[0].AttemptID != o.Integrations[0].SourceAttemptID || o.Integrations[0].FinalSHA != f.target() || o.Results[0].Artifacts[0].RepositoryKey != keys[agent] {
			t.Fatalf("mixed lineage: %+v", o)
		}
		if err := gitworkspace.ObserveIntegration(ctx, f.repo, o.Integrations[0], true); err != nil {
			t.Fatal(err)
		}
	}
	if outcomes[0].Results[0].Artifacts[0].Digest == outcomes[1].Results[0].Artifacts[0].Digest || outcomes[0].Integrations[0].FinalSHA == outcomes[1].Integrations[0].FinalSHA {
		t.Fatal("mixed repository results")
	}
	stored, err := a.store.Load(ctx, "proj", x.ExecutionID)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(outcomes, stored.RepositoryOutcomes()) {
		t.Fatal("store lost provenance")
	}
	wire, _ := workflow.EncodeExecution(stored)
	back, err := workflow.DecodeExecution(wire)
	if err != nil || !reflect.DeepEqual(outcomes, back.RepositoryOutcomes()) {
		t.Fatal("codec lost provenance")
	}
}
