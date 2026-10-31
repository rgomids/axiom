package runtimeapplication

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/rgomids/axiom/internal/workflow"
)

var profileA = &ProfileInput{ID: "fast-codex", Digest: workflow.DigestText([]byte("profile-a")), Runtime: "codex", Model: "gpt-a", Effort: "low", Instructions: "profile instructions"}

func baseRequest() InvocationRequest {
	return InvocationRequest{
		LocalDefault:  InvocationLayer{Assignments: []Assignment{{FieldRuntime, "codex"}, {FieldModel, "local-model"}, {FieldEffort, "runtime-default"}}},
		WorkflowAgent: InvocationLayer{Assignments: []Assignment{{FieldInstructions, "build the feature"}}},
		Executables:   map[string]string{"codex": "codex-binding", "claude": "claude-binding"},
	}
}

func provenanceOf(inv workflow.Invocation, field string) workflow.FieldProvenance {
	for _, p := range inv.Provenance {
		if p.Field == field {
			return p
		}
	}
	return workflow.FieldProvenance{}
}

func TestResolveInvocationPrecedenceTable(t *testing.T) {
	// Local default once, explicit run effort, no Profile.
	r := baseRequest()
	r.RunOverride.Assignments = []Assignment{{FieldEffort, "high"}}
	inv, err := ResolveInvocation(r)
	if err != nil {
		t.Fatal(err)
	}
	if inv.Runtime != "codex" || inv.Model != "local-model" || inv.Effort != "high" || inv.Profile != "" {
		t.Fatalf("tuple %+v", inv)
	}
	if provenanceOf(inv, FieldEffort).Source != "run_override" || provenanceOf(inv, FieldRuntime).Source != "local_default" || provenanceOf(inv, FieldInstructions).Source != "workflow_agent" {
		t.Fatalf("provenance %+v", inv.Provenance)
	}

	// Profile at the Workflow level with its effort replaced for this run.
	r = baseRequest()
	r.WorkflowAgent.Profile = profileA
	r.RunOverride.Assignments = []Assignment{{FieldEffort, "high"}}
	inv, err = ResolveInvocation(r)
	if err != nil {
		t.Fatal(err)
	}
	if inv.Model != "gpt-a" || inv.Effort != "high" || inv.Instructions != "build the feature" || inv.Profile != "fast-codex" || inv.ProfileDigest != profileA.Digest {
		t.Fatalf("profile override %+v", inv)
	}
	if p := provenanceOf(inv, FieldEffort); p.Source != "run_override" || p.Profile != "" {
		t.Fatalf("overridden effort presented as Profile-owned: %+v", p)
	}
	if p := provenanceOf(inv, FieldModel); p.Source != "workflow_agent" || p.Profile != "fast-codex" {
		t.Fatalf("profile field provenance %+v", p)
	}

	// Profile selection without override keeps the whole Profile tuple except
	// same-level explicit instructions, which replace the Profile default.
	r = baseRequest()
	r.WorkflowAgent.Profile = profileA
	inv, err = ResolveInvocation(r)
	if err != nil || inv.Effort != "low" || inv.Instructions != "build the feature" {
		t.Fatalf("profile selection: %v %+v", err, inv)
	}

	// runtime-default is an explicit token, not a claimed vendor level.
	inv, err = ResolveInvocation(baseRequest())
	if err != nil || inv.Effort != "runtime-default" {
		t.Fatalf("runtime-default: %v %+v", err, inv)
	}
}

func TestResolveInvocationRefusesMissingConflictsAndDeniedTargets(t *testing.T) {
	r := baseRequest()
	r.LocalDefault.Assignments = r.LocalDefault.Assignments[:2]
	if _, err := ResolveInvocation(r); !errors.Is(err, ErrInvocationMissing) || !strings.Contains(err.Error(), "effort") {
		t.Fatalf("missing effort became implicit default: %v", err)
	}
	r = baseRequest()
	r.RunOverride.Assignments = []Assignment{{FieldModel, "a"}, {FieldModel, "b"}}
	if _, err := ResolveInvocation(r); !errors.Is(err, ErrInvocationConflict) {
		t.Fatalf("same-level conflict: %v", err)
	}
	r = baseRequest()
	r.RunOverride.Assignments = []Assignment{{FieldModel, "a"}, {FieldModel, "a"}}
	if _, err := ResolveInvocation(r); err != nil {
		t.Fatalf("identical same-level value is not a conflict: %v", err)
	}
	r = baseRequest()
	r.RunOverride.Assignments = []Assignment{{"sandbox", "off"}}
	if _, err := ResolveInvocation(r); !errors.Is(err, ErrInvocationConflict) {
		t.Fatalf("unknown selector field accepted: %v", err)
	}
	r = baseRequest()
	r.PermittedRuntimes = []string{"claude"}
	if _, err := ResolveInvocation(r); !errors.Is(err, ErrRuntimeNotPermitted) {
		t.Fatalf("denied runtime: %v", err)
	}
	r = baseRequest()
	r.RunOverride.Assignments = []Assignment{{FieldRuntime, "gemini"}}
	if _, err := ResolveInvocation(r); !errors.Is(err, ErrRuntimeNotConfigured) {
		t.Fatalf("unconfigured runtime was substituted: %v", err)
	}
	r = baseRequest()
	r.WorkflowAgent.Profile = &ProfileInput{ID: "no-digest", Model: "m"}
	if _, err := ResolveInvocation(r); !errors.Is(err, ErrInvocationConflict) {
		t.Fatalf("profile without identity: %v", err)
	}
	r = baseRequest()
	r.RunOverride.Assignments = []Assignment{{FieldEffort, "high; rm -rf"}}
	if _, err := ResolveInvocation(r); !errors.Is(err, ErrInvocationConflict) {
		t.Fatalf("unsafe effort token: %v", err)
	}
}

func TestResolvedInvocationIsAValidFrozenSnapshot(t *testing.T) {
	r := baseRequest()
	r.WorkflowAgent.Profile = profileA
	inv, err := ResolveInvocation(r)
	if err != nil {
		t.Fatal(err)
	}
	e, err := workflow.NewExecution(workflow.NewExecutionInput{ExecutionID: "00000000-0000-4000-8000-000000000042", ProjectID: "p", WorkItem: workflow.WorkItemLink{Provider: "linear", ID: "W-1", RepositoryKey: "app"},
		Repositories: []workflow.RepositoryBinding{{Key: "app", Identity: "id"}}, WorkflowID: "w", Revision: "r", Now: time.Date(2026, 10, 31, 0, 0, 0, 0, time.UTC), Stages: []workflow.StageSpec{{ID: "s", Agents: []workflow.AgentSpec{{ID: "a"}}, TimeoutSeconds: 1}}})
	if err != nil {
		t.Fatal(err)
	}
	_ = e.ClaimRunner(1, e.CreatedAt)
	if err := e.PrepareAttempt(workflow.PrepareInput{Ref: workflow.TaskRef{StageID: "s", AgentID: "a"}, AttemptID: "a1", Generation: 1, Invocation: inv, Authority: workflow.AttemptAuthority{RepositoryKey: "app", Effects: []string{}, Actor: "user:local"}, Now: e.CreatedAt}); err != nil {
		t.Fatalf("resolver output rejected by Execution codec: %v", err)
	}
}
