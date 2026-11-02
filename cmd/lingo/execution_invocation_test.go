//go:build linux || darwin

package main

import (
	"context"
	"testing"

	"github.com/rgomids/axiom/internal/cli"
	"github.com/rgomids/axiom/internal/local"
	"github.com/rgomids/axiom/internal/runtimeprofile"
	"github.com/rgomids/axiom/internal/workflow"
	"github.com/rgomids/axiom/internal/workflowdefinition"
)

// DMH-05 exercises the portable schema through the public runner and reads the
// frozen attempt back, rather than testing a manually assembled resolver input.
func TestPublicWorkflowAgentInvocationPrecedence(t *testing.T) {
	for _, tc := range []struct {
		name, profile, defaults, model, effort string
		direct                                 *workflowdefinition.AgentInvocation
		selections                             []string
		modelSource, effortSource              string
	}{
		{name: "direct without profile", direct: &workflowdefinition.AgentInvocation{Runtime: "codex", Model: "workflow-model", Effort: "high"}, model: "workflow-model", effort: "high", modelSource: "workflow_agent", effortSource: "workflow_agent"},
		{name: "workflow profile defaults", profile: "shared", model: "shared-model", effort: "medium", modelSource: "workflow_agent", effortSource: "workflow_agent"},
		{name: "direct replaces workflow profile", profile: "shared", direct: &workflowdefinition.AgentInvocation{Model: "workflow-model", Effort: "low"}, model: "workflow-model", effort: "low", modelSource: "workflow_agent", effortSource: "workflow_agent"},
		{name: "run replaces workflow", direct: &workflowdefinition.AgentInvocation{Runtime: "codex", Model: "workflow-model", Effort: "high"}, selections: []string{"--select", "owner.model=run-model", "--select", "owner.effort=low"}, model: "run-model", effort: "low", modelSource: "run_override", effortSource: "run_override"},
		{name: "run profile above workflow direct", direct: &workflowdefinition.AgentInvocation{Runtime: "codex", Model: "workflow-model", Effort: "high"}, selections: []string{"--select", "owner.profile=shared"}, model: "shared-model", effort: "medium", modelSource: "run_override", effortSource: "run_override"},
		{name: "local profile below workflow direct", defaults: `["profile=shared"]`, direct: &workflowdefinition.AgentInvocation{Model: "workflow-model"}, model: "workflow-model", effort: "medium", modelSource: "workflow_agent", effortSource: "local_default"},
		{name: "local explicit replaces local profile", defaults: `["profile=shared","model=local-model","effort=runtime-default"]`, model: "local-model", effort: "runtime-default", modelSource: "local_default", effortSource: "local_default"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newExecutionRunFixture(t)
			installRuntimeImpactPolicy(t, f.env.state, workItemSourceProjectID)
			config := singleRuntimeV3("codex", "shared", "shared-model", []string{"read"}, []string{"medium"})
			config.ModelProfiles[0].Effort = runtimeprofile.Effort{Mode: runtimeprofile.EffortExplicit, Value: "medium"}
			publishRuntimeProfileV3(t, f.env.state, config)
			builtin, _ := workflowdefinition.BuiltinRevision(2)
			d := builtin.Definition
			d.Stages = d.Stages[:1]
			s := &d.Stages[0]
			s.Checkpoint = true
			s.Validators, s.CompletionCriteria, s.HumanGates = []workflowdefinition.Validator{}, []workflowdefinition.Criterion{}, []workflowdefinition.Gate{}
			s.Agents[0].ProfileRef = tc.profile
			s.Agents[0].Invocation = tc.direct
			document, issues := workflowdefinition.Encode(d)
			if len(issues) != 0 {
				t.Fatal(issues)
			}
			decoded, issues := workflowdefinition.Decode(document.Canonical)
			if len(issues) != 0 || decoded.Digest != document.Digest {
				t.Fatalf("roundtrip: %v", issues)
			}
			f.installWorkflowDefinition(t, decoded.Definition)
			if tc.defaults != "" {
				t.Setenv("LINGO_INVOCATION_DEFAULTS", tc.defaults)
			}
			args := []string{"workflow", "run", "--project", "external", "--repository", "main", "--work-item", "github:owner/repo#7", "--authorize-external"}
			got := f.run(t, cli.ExitSuccess, append(args, tc.selections...)...)
			store, err := local.NewExecutionStore(f.env.state)
			if err != nil {
				t.Fatal(err)
			}
			x, err := store.Load(context.Background(), got.Stage.ProjectID, got.Stage.ExecutionID)
			if err != nil {
				t.Fatal(err)
			}
			inv := x.Stages[0].Tasks[0].Attempts[0].Invocation
			if x.Status != workflow.RunCompleted || inv.Runtime != "codex" || inv.Model != tc.model || inv.Effort != tc.effort {
				t.Fatalf("tuple: %+v", inv)
			}
			for _, p := range inv.Provenance {
				if p.Field == "model" && p.Source != tc.modelSource || p.Field == "effort" && p.Source != tc.effortSource {
					t.Fatalf("provenance %+v", inv.Provenance)
				}
			}
		})
	}
}

func TestPublicWorkflowAgentInvocationRequiresEffort(t *testing.T) {
	f := newExecutionRunFixture(t)
	builtin, _ := workflowdefinition.BuiltinRevision(2)
	d := builtin.Definition
	d.Stages = d.Stages[:1]
	s := &d.Stages[0]
	s.Checkpoint = true
	s.Validators, s.CompletionCriteria, s.HumanGates = []workflowdefinition.Validator{}, []workflowdefinition.Criterion{}, []workflowdefinition.Gate{}
	s.Agents[0].ProfileRef = ""
	s.Agents[0].Invocation = &workflowdefinition.AgentInvocation{Runtime: "codex", Model: "workflow-model"}
	f.installWorkflowDefinition(t, d)
	got := f.run(t, cli.ExitFailure, "workflow", "run", "--project", "external", "--repository", "main", "--work-item", "github:owner/repo#7", "--authorize-external")
	if got.Category != "invocation_selection_missing" || len(f.executions(t)) != 0 {
		t.Fatal(got.Raw)
	}
}
