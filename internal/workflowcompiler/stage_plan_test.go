package workflowcompiler

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/rgomids/axiom/internal/executiongraph"
	"github.com/rgomids/axiom/internal/workflowdefinition"
)

func stagePlanRequest(t *testing.T, compile Request, gates []GateRef) StagePlanRequest {
	t.Helper()
	return StagePlanRequest{Execution: ExecutionRef{ExecutionID: "execution-1", Revision: 4}, Gates: gates, PlanDocumentDigest: digestForTest("plan-document"), Compile: compile}
}

func TestPlanStageSingleAgentHasNoGraphAndIsDeterministic(t *testing.T) {
	doc := workflowdefinition.Builtin()
	compile := requestFor(t, doc, "intake", []InputBinding{{InputID: "source", Reference: "target", Digest: digestForTest("work-item")}})
	compiler := New(testRuntimes{}, capabilityOK{})
	first, err := compiler.PlanStage(context.Background(), stagePlanRequest(t, compile, nil))
	if err != nil {
		t.Fatal(err)
	}
	if first.ExecutionKind != "single" || first.GraphProposal != nil || first.Compilation.Proposal != nil || len(first.Resolutions) != 1 || len(first.Compilation.Agents) != 1 {
		t.Fatalf("single-agent stage fabricated a graph or agent: %+v", first)
	}
	if first.Execution != (ExecutionRef{ExecutionID: "execution-1", Revision: 4}) || first.Workflow != compile.Workflow || first.StageID != "intake" || first.Plan.Digest != compile.Plan.PlanDigest || len(first.Blockers) != 0 {
		t.Fatalf("plan lost execution, workflow or Plan identity: %+v", first)
	}
	resolution := first.Resolutions[0]
	if resolution.RuntimeID != "codex" || resolution.EffectiveEffort != "default" || resolution.RequestedEffort.Mode != "runtime-default" || resolution.CapabilityEvidenceRef != "runtime-observation:codex:"+digestForTest("observation") {
		t.Fatalf("unexpected resolution: %+v", resolution)
	}
	second, err := compiler.PlanStage(context.Background(), stagePlanRequest(t, compile, nil))
	if err != nil || second.Digest != first.Digest || !reflect.DeepEqual(first, second) {
		t.Fatal("same inputs produced a different plan")
	}
	changed := stagePlanRequest(t, compile, nil)
	changed.Execution.Revision = 5
	if other, err := compiler.PlanStage(context.Background(), changed); err != nil || other.Digest == first.Digest {
		t.Fatal("Execution revision is not bound by the plan digest")
	}
	changed = stagePlanRequest(t, compile, nil)
	changed.PlanDocumentDigest = digestForTest("other-document")
	if other, err := compiler.PlanStage(context.Background(), changed); err != nil || other.Digest == first.Digest {
		t.Fatal("Plan document is not bound by the plan digest")
	}
	if other, err := New(testRuntimes{revision: 9}, capabilityOK{}).PlanStage(context.Background(), stagePlanRequest(t, compile, nil)); err != nil || other.Digest == first.Digest || other.Compilation.Digest == first.Compilation.Digest {
		t.Fatal("Runtime configuration drift did not invalidate the plan")
	}
	for _, invalid := range []StagePlanRequest{{Compile: compile, PlanDocumentDigest: digestForTest("x")}, {Execution: ExecutionRef{ExecutionID: "e"}, Compile: compile, PlanDocumentDigest: digestForTest("x")}, {Execution: ExecutionRef{ExecutionID: "e", Revision: 1}, Compile: compile, PlanDocumentDigest: "short"}} {
		if _, err := compiler.PlanStage(context.Background(), invalid); !errors.Is(err, ErrInvalidRequest) {
			t.Fatalf("incomplete ExecutionRef/Plan accepted: %v", err)
		}
	}
}

func TestPlanStageGraphExposesMixedRuntimesDependenciesAndGates(t *testing.T) {
	gates := []GateRef{{Kind: "implementation_authority", Timing: "before"}, {Kind: "human_acceptance", Timing: "after"}}
	compiler := New(testRuntimes{}, capabilityOK{})
	plan, err := compiler.PlanStage(context.Background(), stagePlanRequest(t, multiRequest(t), gates))
	if err != nil {
		t.Fatal(err)
	}
	if plan.ExecutionKind != "graph" || plan.GraphProposal == nil || plan.GraphProposal.Digest != plan.Compilation.Proposal.Digest || len(plan.Compilation.Proposal.Nodes) != 3 {
		t.Fatalf("graph proposal not inspectable: %+v", plan)
	}
	runtimes, efforts := map[string]string{}, map[string]string{}
	for _, resolution := range plan.Resolutions {
		runtimes[resolution.AgentID], efforts[resolution.AgentID] = resolution.RuntimeID, resolution.EffectiveEffort
	}
	if !reflect.DeepEqual(runtimes, map[string]string{"implementer": "codex", "reviewer": "claude", "integrator": "codex"}) || !reflect.DeepEqual(efforts, map[string]string{"implementer": "medium", "reviewer": "high", "integrator": "default"}) {
		t.Fatalf("resolutions lost Runtime or effort: %v %v", runtimes, efforts)
	}
	for _, node := range plan.Compilation.Proposal.Nodes {
		if node.Key == "implementation:integrator" && (!node.IntegrationOwner || !reflect.DeepEqual(node.Dependencies, []string{"implementation:implementer", "implementation:reviewer"})) {
			t.Fatalf("integration owner lost its dependencies: %+v", node)
		}
	}
	if len(plan.Compilation.Ordering) != 2 || len(plan.Compilation.Ordering[0]) > plan.Compilation.Concurrency {
		t.Fatalf("ordering exceeds stage concurrency: %v", plan.Compilation.Ordering)
	}
	if !reflect.DeepEqual(plan.Blockers, []string{"human_gate_pending:implementation_authority"}) || len(plan.GateRefs) != 2 || len(plan.ValidatorRefs) == 0 {
		t.Fatalf("gates/validators not reported as read-only criteria: %+v %+v", plan.Blockers, plan.GateRefs)
	}
	gates[0].Satisfied = true
	satisfied, err := compiler.PlanStage(context.Background(), stagePlanRequest(t, multiRequest(t), gates))
	if err != nil || len(satisfied.Blockers) != 0 || satisfied.Digest == plan.Digest || satisfied.Compilation.Digest != plan.Compilation.Digest {
		t.Fatal("recorded human fact did not change only the gate view")
	}
	if !reflect.DeepEqual(satisfied.StageInput, plan.StageInput) || plan.StageInput.ID != "stage-input-set-"+plan.StageInput.Digest {
		t.Fatal("stage input set reference is not stable")
	}
}

func TestPlanStageFailsClosedWithSpecificationCategories(t *testing.T) {
	overlap := multiRequest(t)
	rebindStage(t, &overlap, func(s *workflowdefinition.Stage) {
		s.Agents[1].EffectCeilings = append(s.Agents[1].EffectCeilings, "repository-write")
	})
	write := executiongraph.Effect{Kind: "repository-write", Target: "src/shared.go"}
	overlap.ParentAuthority = []executiongraph.Effect{write}
	for index := 0; index < 2; index++ {
		overlap.Plan.Work[index].Scope.Paths = []string{"src/shared.go"}
		overlap.Plan.Work[index].Effects = []executiongraph.Effect{write}
	}
	excess := multiRequest(t)
	excess.Plan.Work[0].Effects = []executiongraph.Effect{{Kind: "repository-write", Target: "src/one.go"}}
	ceiling := multiRequest(t)
	ceiling.ParentAuthority = []executiongraph.Effect{{Kind: "integration", Target: "src/one.go"}}
	for index := range ceiling.Plan.Work {
		if ceiling.Plan.Work[index].Key == "reviewer" {
			ceiling.Plan.Work[index].Effects = []executiongraph.Effect{{Kind: "integration", Target: "src/one.go"}}
			ceiling.Plan.Work[index].Scope.Paths = []string{"src"}
		}
	}
	escapeScope := multiRequest(t)
	escapeScope.Plan.Work[0].Scope.Paths = []string{".."}
	escapeWrite := multiRequest(t)
	parent := executiongraph.Effect{Kind: "repository-write", Target: ".."}
	escapeWrite.ParentAuthority = []executiongraph.Effect{parent}
	escapeWrite.Plan.Work[0].Scope.Paths = []string{".."}
	escapeWrite.Plan.Work[0].Effects = []executiongraph.Effect{parent}
	backslash := multiRequest(t)
	backslash.Plan.Work[0].Scope.Paths = []string{`..\outside`}
	unapproved := multiRequest(t)
	unapproved.Plan.Approved = false
	for _, tc := range []struct {
		name     string
		compiler Compiler
		request  Request
		category string
		denied   bool
	}{
		{"unsupported effort", New(testRuntimes{unsupportedEffort: true}, capabilityOK{}), multiRequest(t), "unsupported_effort", false},
		{"no match", New(testRuntimes{noMatch: true}, capabilityOK{}), multiRequest(t), "runtime_unresolvable", false},
		{"missing capability", New(testRuntimes{}, capabilityDenied{}), multiRequest(t), "runtime_unresolvable", false},
		{"effect outside parent ceiling", New(testRuntimes{}, capabilityOK{}), excess, "authority_denied", true},
		{"effect outside agent ceiling", New(testRuntimes{}, capabilityOK{}), ceiling, "authority_denied", true},
		{"unapproved plan", New(testRuntimes{}, capabilityOK{}), unapproved, "invalid_stage_plan", false},
		{"parent scope path", New(testRuntimes{}, capabilityOK{}), escapeScope, "authority_denied", true},
		{"parent write target within matching ceiling", New(testRuntimes{}, capabilityOK{}), escapeWrite, "authority_denied", true},
		{"backslash scope path", New(testRuntimes{}, capabilityOK{}), backslash, "authority_denied", true},
		{"unsafe parallel overlap", New(testRuntimes{}, capabilityOK{}), overlap, "invalid_stage_topology", false},
	} {
		_, err := tc.compiler.PlanStage(context.Background(), stagePlanRequest(t, tc.request, nil))
		failure := ClassifyFailure(err)
		if err == nil || failure.Category != tc.category || failure.AuthorityDenied != tc.denied {
			t.Fatalf("%s: err=%v failure=%+v", tc.name, err, failure)
		}
	}
	for err, category := range map[error]string{executiongraph.ErrCycle: "invalid_stage_topology", executiongraph.ErrUnsafeOverlap: "invalid_stage_topology", executiongraph.ErrMissingOwner: "invalid_stage_topology", executiongraph.ErrAuthoritySubset: "authority_denied", executiongraph.ErrUnresolvable: "runtime_unresolvable", ErrInvalidRequest: "invalid_stage_plan"} {
		if ClassifyFailure(err).Category != category {
			t.Fatalf("%v classified as %+v", err, ClassifyFailure(err))
		}
	}
}

func TestDecodePlanDocumentIsStrictAndFormattingIndependent(t *testing.T) {
	compile := multiRequest(t)
	wire, err := json.Marshal(PlanDocument{FormatVersion: 1, Plan: compile.Plan, AuthorityCeiling: []executiongraph.Effect{{Kind: "read", Target: "docs"}}})
	if err != nil {
		t.Fatal(err)
	}
	document, digest, err := DecodePlanDocument(wire)
	if err != nil || !reflect.DeepEqual(document.Plan, compile.Plan) || !workflowdefinition.ValidDigest(digest) {
		t.Fatalf("valid document rejected: %v", err)
	}
	var indented []byte
	var generic any
	_ = json.Unmarshal(wire, &generic)
	indented, _ = json.MarshalIndent(generic, "", "  ")
	if _, again, err := DecodePlanDocument(indented); err != nil || again != digest {
		t.Fatal("formatting changed the Plan document digest")
	}
	var object map[string]any
	_ = json.Unmarshal(wire, &object)
	object["unexpected"] = true
	unknown, _ := json.Marshal(object)
	object["formatVersion"] = 2
	delete(object, "unexpected")
	future, _ := json.Marshal(object)
	for _, invalid := range [][]byte{nil, []byte("{"), unknown, future, []byte(`{"formatVersion":1,"formatVersion":1}`), make([]byte, MaxPlanDocumentBytes+1)} {
		if _, _, err := DecodePlanDocument(invalid); !errors.Is(err, ErrInvalidPlanDocument) {
			t.Fatalf("invalid document accepted: %.40q", invalid)
		}
	}
}
