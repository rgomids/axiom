package workflowcompiler

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/rgomids/axiom/internal/executiongraph"
	"github.com/rgomids/axiom/internal/runtimeapplication"
	"github.com/rgomids/axiom/internal/runtimeprofile"
	"github.com/rgomids/axiom/internal/workflowdefinition"
)

type testRuntimes struct {
	unsupportedEffort bool
	noMatch           bool
	revision          uint64
}

func (r testRuntimes) Preview(_ context.Context, projectID string, request runtimeapplication.Request) (runtimeapplication.Preview, error) {
	if r.noMatch {
		return runtimeapplication.Preview{Blocker: &runtimeprofile.Blocker{Code: "no_allowed_match"}}, runtimeapplication.ErrBlocked
	}
	runtimeID := "codex"
	if request.Role == "reviewer" {
		runtimeID = "claude"
	}
	if request.RuntimeID != "" && request.RuntimeID != runtimeID {
		return runtimeapplication.Preview{Blocker: &runtimeprofile.Blocker{Code: "no_allowed_match"}}, runtimeapplication.ErrBlocked
	}
	for _, capability := range request.Capabilities {
		if strings.HasPrefix(capability, "reasoning-effort-") && r.unsupportedEffort {
			return runtimeapplication.Preview{Blocker: &runtimeprofile.Blocker{Code: "no_allowed_match"}}, runtimeapplication.ErrBlocked
		}
	}
	profileID := request.ModelProfileID
	if profileID == "" {
		profileID = "profile-" + runtimeID
	} else if profileID != "profile-"+runtimeID {
		return runtimeapplication.Preview{Blocker: &runtimeprofile.Blocker{Code: "no_allowed_match"}}, runtimeapplication.ErrBlocked
	}
	revision := r.revision
	if revision == 0 {
		revision = 3
	}
	return runtimeapplication.Preview{ProjectID: projectID, Request: request, ConfigurationRevision: revision, ProjectDigest: digestForTest("project"), ConfigurationDigest: digestForTest("configuration"), ObservationDigest: digestForTest("observation"), Choice: &runtimeprofile.Choice{RuntimeID: runtimeID, Adapter: runtimeID, ModelProfileID: profileID, Model: "local-profile-model", Capabilities: append([]string(nil), request.Capabilities...), RuntimeVersion: "test-1.0", ExecutableDigest: digestForTest("executable"), ConfigurationRevision: revision, ObservationRevision: revision + 2}}, nil
}

type capabilityOK struct{}

func (capabilityOK) ValidateCapability(context.Context, executiongraph.CapabilityRequest) error {
	return nil
}

func TestCompileSingleAgentUsesNoGraphAndBindsStageInputs(t *testing.T) {
	doc := workflowdefinition.Builtin()
	stage := findStage(t, doc, "intake")
	request := requestFor(t, doc, stage.ID, []InputBinding{{InputID: "source", Reference: "target", Digest: digestForTest("work-item")}})
	result, err := New(testRuntimes{}, capabilityOK{}).Compile(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if result.ExecutionKind != "single" || result.Proposal != nil || len(result.Agents) != 1 || result.Digest == "" {
		t.Fatalf("unexpected single-stage result: %+v", result)
	}
	compiled := result.Agents[0]
	if compiled.Input.Workflow != request.Workflow || compiled.Input.StageID != stage.ID || compiled.Input.Instructions != stage.Instructions || len(compiled.Input.Inputs) != 1 || compiled.Input.Inputs[0].Digest != digestForTest("work-item") {
		t.Fatalf("stage input lost lineage or binding: %+v", compiled.Input)
	}
}

func TestCompileMultiAgentGraphPreservesRuntimesAndOrdering(t *testing.T) {
	doc := loadWorkflow(t, "../../docs/specifications/007-configurable-workflows/examples/custom-r1.json")
	stage := findStage(t, doc, "implementation")
	request := requestFor(t, doc, stage.ID, []InputBinding{{InputID: "source", Reference: "tasks/result", Digest: digestForTest("prior-stage")}, {InputID: "business-context", Reference: "business-context", Digest: digestForTest("context")}})
	compiler := New(testRuntimes{}, capabilityOK{})
	first, err := compiler.Compile(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if first.ExecutionKind != "graph" || first.Proposal == nil || len(first.Proposal.Nodes) != len(stage.Agents) || first.Proposal.Digest == "" {
		t.Fatalf("unexpected graph result: %+v", first)
	}
	runtimes := map[string]string{}
	for _, agent := range first.Agents {
		runtimes[agent.AgentID] = agent.Choice.RuntimeID
	}
	if runtimes["implementer"] != "codex" || runtimes["reviewer"] != "claude" || runtimes["integrator"] != "codex" {
		t.Fatalf("runtime constraints not honored: %v", runtimes)
	}
	if len(first.Ordering) == 0 || len(first.Ordering[0]) > stage.Concurrency {
		t.Fatalf("concurrency bound not represented: %v", first.Ordering)
	}
	var integrator ResolvedAgent
	for _, agent := range first.Agents {
		if agent.AgentID == "integrator" {
			integrator = agent
		}
	}
	if len(integrator.Input.DependencyOutputs) != 2 || len(integrator.Input.DependencyOutputs[0].Outputs) == 0 || len(integrator.Input.DependencyOutputs[1].Outputs) == 0 {
		t.Fatalf("integration inputs lack declared predecessor outputs: %+v", integrator.Input.DependencyOutputs)
	}
	request.Plan.Work[0], request.Plan.Work[2] = request.Plan.Work[2], request.Plan.Work[0]
	request.InputBindings[0], request.InputBindings[1] = request.InputBindings[1], request.InputBindings[0]
	second, err := compiler.Compile(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if first.Digest != second.Digest || first.Proposal.Digest != second.Proposal.Digest {
		t.Fatalf("output depends on input order: %s != %s", first.Digest, second.Digest)
	}
	changed, err := New(testRuntimes{revision: 4}, capabilityOK{}).Compile(context.Background(), request)
	if err != nil || first.Digest == changed.Digest {
		t.Fatalf("runtime policy revision did not invalidate proposal: changed=%+v err=%v", changed, err)
	}
}

func TestCompileFailsClosedForEffortAndUnboundedAuthority(t *testing.T) {
	doc := loadWorkflow(t, "../../docs/specifications/007-configurable-workflows/examples/custom-r1.json")
	stage := findStage(t, doc, "implementation")
	request := requestFor(t, doc, stage.ID, []InputBinding{{InputID: "source", Reference: "tasks/result", Digest: digestForTest("prior-stage")}, {InputID: "business-context", Reference: "business-context", Digest: digestForTest("context")}})
	_, err := New(testRuntimes{unsupportedEffort: true}, capabilityOK{}).Compile(context.Background(), request)
	if !errors.Is(err, ErrUnsupportedEffort) {
		t.Fatalf("unsupported effort err=%v", err)
	}
	request = requestFor(t, doc, stage.ID, []InputBinding{{InputID: "source", Reference: "tasks/result", Digest: digestForTest("prior-stage")}, {InputID: "business-context", Reference: "business-context", Digest: digestForTest("context")}})
	request.ParentAuthority = nil
	request.Plan.Work[0].Effects = []executiongraph.Effect{{Kind: "repository-write", Target: "src/main.go"}}
	_, err = New(testRuntimes{}, capabilityOK{}).Compile(context.Background(), request)
	if !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("authority expansion err=%v", err)
	}
	request = requestFor(t, doc, stage.ID, []InputBinding{{InputID: "source", Reference: "tasks/result", Digest: digestForTest("prior-stage")}, {InputID: "business-context", Reference: "business-context", Digest: digestForTest("context")}})
	write := executiongraph.Effect{Kind: "repository-write", Target: "src/main.go"}
	request.ParentAuthority = []executiongraph.Effect{write}
	request.Plan.Work[0].Effects = []executiongraph.Effect{write}
	_, err = New(testRuntimes{}, capabilityOK{}).Compile(context.Background(), request)
	if !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("out-of-scope repository effect err=%v", err)
	}
}

func TestCompileDoesNotFallbackWhenRuntimeOrProfileIsUnavailable(t *testing.T) {
	doc := workflowdefinition.Builtin()
	stage := findStage(t, doc, "intake")
	request := requestFor(t, doc, stage.ID, []InputBinding{{InputID: "source", Reference: "target", Digest: digestForTest("work-item")}})
	if _, err := New(testRuntimes{noMatch: true}, capabilityOK{}).Compile(context.Background(), request); !errors.Is(err, ErrRuntimeBlocked) {
		t.Fatalf("unavailable runtime err=%v", err)
	}
	doc = loadWorkflow(t, "../../docs/specifications/007-configurable-workflows/examples/custom-r1.json")
	stage = findStage(t, doc, "implementation")
	request = requestFor(t, doc, stage.ID, []InputBinding{{InputID: "source", Reference: "tasks/result", Digest: digestForTest("prior-stage")}, {InputID: "business-context", Reference: "business-context", Digest: digestForTest("context")}})
	definition := request.Definition.Definition
	for index := range definition.Stages {
		if definition.Stages[index].ID == stage.ID {
			definition.Stages[index].Agents[0].ProfileRef = "missing-profile"
		}
	}
	doc, diagnostics := workflowdefinition.Encode(definition)
	if len(diagnostics) != 0 {
		t.Fatalf("test definition invalid: %+v", diagnostics)
	}
	request.Definition = doc
	request.Workflow = doc.Ref("project")
	if _, err := New(testRuntimes{}, capabilityOK{}).Compile(context.Background(), request); !errors.Is(err, ErrRuntimeBlocked) {
		t.Fatalf("implicit profile fallback occurred: err=%v", err)
	}
}

func TestCompileRejectsForgedDefinitionAndMissingReferences(t *testing.T) {
	doc := workflowdefinition.Builtin()
	stage := findStage(t, doc, "intake")
	request := requestFor(t, doc, stage.ID, []InputBinding{{InputID: "source", Reference: "target", Digest: digestForTest("work-item")}})
	request.Definition.Definition.Name = "changed"
	if _, err := New(testRuntimes{}, capabilityOK{}).Compile(context.Background(), request); !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("forged immutable definition accepted: %v", err)
	}
	request = requestFor(t, doc, stage.ID, nil)
	if _, err := New(testRuntimes{}, capabilityOK{}).Compile(context.Background(), request); !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("missing required input accepted: %v", err)
	}
	doc = loadWorkflow(t, "../../docs/specifications/007-configurable-workflows/examples/custom-r1.json")
	stage = findStage(t, doc, "intake")
	request = requestFor(t, doc, stage.ID, []InputBinding{{InputID: "source", Reference: "target", Digest: digestForTest("work-item")}})
	request.Workflow = request.Definition.Ref("builtin")
	if _, err := New(testRuntimes{}, capabilityOK{}).Compile(context.Background(), request); !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("custom content relabeled as built-in accepted: %v", err)
	}
}

func requestFor(t *testing.T, doc workflowdefinition.Document, stageID string, bindings []InputBinding) Request {
	t.Helper()
	stage := findStage(t, doc, stageID)
	work := make([]executiongraph.WorkUnit, 0, len(stage.Agents))
	var effects []executiongraph.Effect
	for index, agent := range stage.Agents {
		path := []string{"src/one.go", "docs/two.md", "pkg/three.go"}[index%3]
		work = append(work, executiongraph.WorkUnit{Key: agent.ID, Scope: executiongraph.Scope{ProjectID: "project-1", RepositoryKey: "main", Paths: []string{path}}, Controls: executiongraph.ExecutionControls{Timeout: 5 * time.Minute, MaximumAttempts: 1}})
	}
	return Request{ProjectID: "project-1", RepositoryKey: "main", Workflow: doc.Ref(workflowSource(doc)), Definition: doc, StageID: stageID, Plan: executiongraph.ApprovedPlan{Approved: true, PlanRevision: "plan-1", PlanDigest: digestForTest("plan"), MaximumNodes: len(work), Work: work}, InputBindings: bindings, ParentAuthority: effects}
}

func loadWorkflow(t *testing.T, path string) workflowdefinition.Document {
	t.Helper()
	wire, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	doc, diagnostics := workflowdefinition.Decode(wire)
	if len(diagnostics) != 0 {
		t.Fatalf("invalid workflow: %+v", diagnostics)
	}
	return doc
}

func findStage(t *testing.T, doc workflowdefinition.Document, id string) workflowdefinition.Stage {
	t.Helper()
	for _, stage := range doc.Definition.Stages {
		if stage.ID == id {
			return stage
		}
	}
	t.Fatalf("missing stage %q", id)
	return workflowdefinition.Stage{}
}

func digestForTest(value string) string {
	return strings.Repeat("a", 64)
}

func workflowSource(doc workflowdefinition.Document) string {
	if doc.Definition.WorkflowID == "default-sdd" {
		return "builtin"
	}
	return "project"
}
