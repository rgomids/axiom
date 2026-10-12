package workflowcompiler

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/executiongraph"
	"github.com/rgomids/axiom/internal/runtimeapplication"
	"github.com/rgomids/axiom/internal/runtimeprofile"
	"github.com/rgomids/axiom/internal/workflowdefinition"
)

type v2Runtimes struct {
	effort  runtimeprofile.Effort
	unknown bool
	calls   *int
}

func (r v2Runtimes) Preview(ctx context.Context, id string, request runtimeapplication.Request) (runtimeapplication.Preview, error) {
	if r.calls != nil {
		*r.calls++
	}
	preview, err := (testRuntimes{}).Preview(ctx, id, request)
	if err != nil {
		return preview, err
	}
	c := preview.Choice
	c.ResolvedProfile = &runtimeprofile.ResolvedProfile{ProfileID: c.ModelProfileID, RuntimeID: c.RuntimeID, Model: c.Model, Effort: r.effort, ProfileDigest: digestForTest("profile"), ConfigurationRevision: c.ConfigurationRevision, ConfigurationDigest: preview.ConfigurationDigest, ExecutableDigest: c.ExecutableDigest, RuntimeVersion: c.RuntimeVersion, InvocationMode: "noninteractive", ObservationRevision: c.ObservationRevision, ObservationDigest: preview.ObservationDigest, EffortSourceDigest: digestForTest("effort-source"), CapabilityProofReference: "injected-proof"}
	if r.unknown {
		c.ResolvedProfile.EffortSourceDigest = ""
	}
	return preview, nil
}
func TestV2CompilerDerivesOnlyProfileEffort(t *testing.T) {
	doc, _ := workflowdefinition.BuiltinRevision(2)
	req := requestFor(t, doc, "intake", []InputBinding{{InputID: "source", Reference: "target", Digest: digestForTest("item")}})
	for _, effort := range []runtimeprofile.Effort{{Mode: "explicit", Value: "opaque-high"}, {Mode: "runtime-default", Value: "default"}} {
		calls := 0
		compiler := New(v2Runtimes{effort: effort, calls: &calls}, capabilityOK{})
		result, err := compiler.CompileV2(context.Background(), req)
		if err != nil {
			t.Fatal(err)
		}
		expected := ""
		if effort.Mode == "explicit" {
			expected = effort.Value
		}
		agent := result.Agents[0]
		if result.FormatVersion != 2 || agent.Controls.ReasoningEffort != expected || calls != 1 || agent.Input.ProfileEffort == nil || *agent.Input.ProfileEffort != effort || agent.Input.EffortSource != "local-model-profile" {
			t.Fatalf("wrong projection %+v calls=%d", result, calls)
		}
		wire, _ := json.Marshal(result)
		if strings.Contains(string(wire), "requestedEffort") {
			t.Fatal("v2 relabelled legacy effort")
		}
		if !executiongraph.ValidProfileControls(agent.Choice.ResolvedProfile, agent.Controls, agent.Capability) {
			t.Fatal("invalid controls")
		}
		stale := req
		stale.Plan.Work[0].Controls.ReasoningEffort = "other"
		if _, err = compiler.CompileV2(context.Background(), stale); err == nil {
			t.Fatal("accepted work override")
		}
		stale.Plan.Work[0].Controls.ReasoningEffort = ""
		plan, err := compiler.PlanStageV2(context.Background(), StagePlanRequest{Execution: ExecutionRef{ExecutionID: "execution", Revision: 1}, PlanDocumentDigest: digestForTest("plan"), Compile: req})
		if err != nil || plan.FormatVersion != 2 || plan.Resolutions[0].ProfileEffort == nil {
			t.Fatal("stage2", err)
		}
	}
	if _, err := New(v2Runtimes{unknown: true, effort: runtimeprofile.Effort{Mode: "explicit", Value: "high"}}, capabilityOK{}).CompileV2(context.Background(), req); err == nil {
		t.Fatal("accepted unknown source proof")
	}
	req.Definition = workflowdefinition.Builtin()
	req.Workflow = req.Definition.Ref("builtin")
	if _, err := New(testRuntimes{}, capabilityOK{}).CompileV2(context.Background(), req); !errors.Is(err, ErrWorkflowMigrationRequired) {
		t.Fatal(err)
	}
}
