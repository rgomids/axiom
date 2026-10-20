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
	c.RuntimeVersion = ""
	c.FrozenProfile = &runtimeprofile.ResolvedProfileV4{ProfileID: c.ModelProfileID, RuntimeID: c.RuntimeID, Model: c.Model, Effort: r.effort, ProfileDigest: digestForTest("profile"), ConfigurationRevision: c.ConfigurationRevision, ConfigurationDigest: preview.ConfigurationDigest, ExecutableDigest: c.ExecutableDigest, InvocationMode: "noninteractive", ObservationRevision: c.ObservationRevision, ObservationDigest: preview.ObservationDigest, EffortSourceDigest: digestForTest("effort-source")}
	if r.unknown {
		c.FrozenProfile.EffortSourceDigest = ""
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
		if result.FormatVersion != OperationalResultVersion || agent.Controls.ReasoningEffort != expected || calls != 1 || agent.Input.ProfileEffort == nil || *agent.Input.ProfileEffort != effort || agent.Input.EffortSource != "local-model-profile" {
			t.Fatalf("wrong projection %+v calls=%d", result, calls)
		}
		wire, _ := json.Marshal(result)
		if strings.Contains(string(wire), "requestedEffort") {
			t.Fatal("v2 relabelled legacy effort")
		}
		if agent.Input.FrozenProfile == nil || agent.Input.ResolvedProfile != nil || !executiongraph.ValidFrozenProfileControls(agent.Choice.FrozenProfile, agent.Controls, agent.Capability) {
			t.Fatal("invalid controls")
		}
		stale := req
		stale.Plan.Work[0].Controls.ReasoningEffort = "other"
		if _, err = compiler.CompileV2(context.Background(), stale); err == nil {
			t.Fatal("accepted work override")
		}
		stale.Plan.Work[0].Controls.ReasoningEffort = ""
		plan, err := compiler.PlanStageV2(context.Background(), StagePlanRequest{Execution: ExecutionRef{ExecutionID: "execution", Revision: 1}, PlanDocumentDigest: digestForTest("plan"), Compile: req})
		if err != nil || plan.FormatVersion != OperationalStagePlanVersion || plan.Resolutions[0].ProfileEffort == nil || plan.Resolutions[0].FrozenProfile == nil || plan.Resolutions[0].CapabilityEvidenceRef != "" {
			t.Fatal("stage2", err)
		}
	}
	if _, err := New(v2Runtimes{unknown: true, effort: runtimeprofile.Effort{Mode: "explicit", Value: "high"}}, capabilityOK{}).CompileV2(context.Background(), req); err == nil {
		t.Fatal("accepted an unobserved effort source")
	}
	req.Definition = workflowdefinition.Builtin()
	req.Workflow = req.Definition.Ref("builtin")
	if _, err := New(testRuntimes{}, capabilityOK{}).CompileV2(context.Background(), req); !errors.Is(err, ErrWorkflowMigrationRequired) {
		t.Fatal(err)
	}
}
