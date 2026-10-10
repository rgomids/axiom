package workflowcompiler

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/rgomids/axiom/internal/executiongraph"
	"github.com/rgomids/axiom/internal/project"
	"github.com/rgomids/axiom/internal/runtimeadapter"
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

type capabilityDenied struct{}

func (capabilityDenied) ValidateCapability(context.Context, executiongraph.CapabilityRequest) error {
	return errors.New("capability unavailable")
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
	hash := sha256.Sum256([]byte(value))
	return hex.EncodeToString(hash[:])
}

type transformedRuntime struct {
	change func(*runtimeapplication.Preview)
}

func (r transformedRuntime) Preview(ctx context.Context, projectID string, request runtimeapplication.Request) (runtimeapplication.Preview, error) {
	preview, err := (testRuntimes{}).Preview(ctx, projectID, request)
	if err == nil {
		r.change(&preview)
	}
	return preview, err
}

func multiRequest(t *testing.T) Request {
	t.Helper()
	doc := loadWorkflow(t, "../../docs/specifications/007-configurable-workflows/examples/custom-r1.json")
	return requestFor(t, doc, "implementation", []InputBinding{{InputID: "source", Reference: "tasks/result", Digest: digestForTest("prior-stage")}, {InputID: "business-context", Reference: "business-context", Digest: digestForTest("context")}})
}

func TestCompileBindsRuntimeSnapshotsAndStageKeys(t *testing.T) {
	request := multiRequest(t)
	first, err := New(testRuntimes{}, capabilityOK{}).Compile(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	changed, err := New(transformedRuntime{change: func(p *runtimeapplication.Preview) { p.ObservationDigest = digestForTest("new-observation") }}, capabilityOK{}).Compile(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if first.Digest == changed.Digest || first.Proposal.Digest == changed.Proposal.Digest {
		t.Fatal("changed observation bytes did not invalidate both review digests")
	}
	for _, node := range first.Proposal.Nodes {
		if !strings.HasPrefix(node.Key, "implementation:") {
			t.Fatalf("node lost stage correlation: %s", node.Key)
		}
		found := false
		for _, input := range node.Inputs {
			found = found || strings.HasPrefix(input, "stage-input-")
		}
		if !found {
			t.Fatalf("node %s has no digest-bound stage input", node.Key)
		}
		for _, agent := range first.Agents {
			if agent.Input.NodeKey != node.Key {
				continue
			}
			for _, dependency := range agent.Input.DependencyOutputs {
				for _, output := range dependency.Outputs {
					token := "output-implementation-" + dependency.AgentID + "-" + output
					if !contains(node.Inputs, token) {
						t.Fatalf("dependency output %s is absent from consumer inputs", token)
					}
					for _, producer := range first.Proposal.Nodes {
						if producer.Key == "implementation:"+dependency.AgentID && !contains(producer.Outputs, token) {
							t.Fatalf("consumer input %s does not match producer outputs", token)
						}
					}
				}
			}
		}
	}
}

func TestCompileReportsRequiredEffectsSeparatelyFromAuthorityCeiling(t *testing.T) {
	r := multiRequest(t)
	r.ParentAuthority = []executiongraph.Effect{{Kind: "read", Target: "approved-context"}}
	result, err := New(testRuntimes{}, capabilityOK{}).Compile(context.Background(), r)
	if err != nil || len(result.Authority) != 0 || len(result.AuthorityCeiling) != 1 {
		t.Fatalf("required authority was expanded to unused ceiling: %+v err=%v", result, err)
	}
}

func TestCompileRejectsPolicyDriftAcrossAgents(t *testing.T) {
	resolver := transformedRuntime{change: func(p *runtimeapplication.Preview) {
		if p.Request.Role == "reviewer" {
			p.ProjectDigest = digestForTest("changed-project")
		}
	}}
	if _, err := New(resolver, capabilityOK{}).Compile(context.Background(), multiRequest(t)); !errors.Is(err, ErrRuntimeBlocked) {
		t.Fatalf("mixed Project snapshots accepted: %v", err)
	}
	resolver.change = func(p *runtimeapplication.Preview) {
		if p.Request.Role == "integrator" {
			p.ObservationDigest = digestForTest("changed-Runtime")
		}
	}
	if _, err := New(resolver, capabilityOK{}).Compile(context.Background(), multiRequest(t)); !errors.Is(err, ErrRuntimeBlocked) {
		t.Fatalf("mixed Runtime snapshots accepted: %v", err)
	}
}

func TestCompileDoesNotMutateApprovedScope(t *testing.T) {
	request := multiRequest(t)
	request.Plan.Work[0].Scope.Paths = []string{"z/file.go", "a/file.go"}
	before, _ := json.Marshal(request.Plan)
	if _, err := New(testRuntimes{}, capabilityOK{}).Compile(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	after, _ := json.Marshal(request.Plan)
	if string(before) != string(after) {
		t.Fatal("compiler mutated the approved input plan")
	}
}

func TestCompileRejectsInvalidTopologyControlsAndReferences(t *testing.T) {
	tests := []struct {
		name string
		edit func(*Request)
	}{
		{"unapproved plan", func(r *Request) { r.Plan.Approved = false }},
		{"excess nodes", func(r *Request) { r.Plan.MaximumNodes = 2 }},
		{"unbounded node limit", func(r *Request) { r.Plan.MaximumNodes = 33 }},
		{"zero attempts", func(r *Request) { r.Plan.Work[0].Controls.MaximumAttempts = 0 }},
		{"excess attempts", func(r *Request) { r.Plan.Work[0].Controls.MaximumAttempts = 3 }},
		{"excess timeout", func(r *Request) { r.Plan.Work[0].Controls.Timeout = 25 * time.Hour }},
		{"zero timeout", func(r *Request) { r.Plan.Work[0].Controls.Timeout = 0 }},
		{"duplicate work", func(r *Request) { r.Plan.Work[1].Key = r.Plan.Work[0].Key }},
		{"wrong Project", func(r *Request) { r.Plan.Work[0].Scope.ProjectID = "other" }},
		{"wrong Repository", func(r *Request) { r.Plan.Work[0].Scope.RepositoryKey = "other" }},
		{"scope traversal", func(r *Request) { r.Plan.Work[0].Scope.Paths = []string{"../private"} }},
		{"stale revision", func(r *Request) { r.Workflow.Revision++ }},
		{"host path input", func(r *Request) { r.InputBindings[0].Reference = "/private/credential" }},
		{"credential-bearing effect", func(r *Request) {
			effect := executiongraph.Effect{Kind: "process", Target: "api_key=unsafe-fixture"}
			r.Plan.Work[0].Effects, r.ParentAuthority = []executiongraph.Effect{effect}, []executiongraph.Effect{effect}
		}},
		{"dangling dependency", func(r *Request) {
			r.Definition.Definition.Stages[5].Agents[0].DependsOn = []string{"missing"}
		}},
		{"cycle", func(r *Request) {
			r.Definition.Definition.Stages[5].Agents[0].DependsOn = []string{"integrator"}
		}},
		{"unsafe instructions", func(r *Request) {
			r.Definition.Definition.Stages[5].Instructions = "api_key=unsafe-fixture"
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			r := multiRequest(t)
			test.edit(&r)
			if _, err := New(testRuntimes{}, capabilityOK{}).Compile(context.Background(), r); err == nil {
				t.Fatal("unsafe request accepted")
			}
		})
	}
	if _, err := New(testRuntimes{}, capabilityDenied{}).Compile(context.Background(), multiRequest(t)); !errors.Is(err, executiongraph.ErrUnresolvable) {
		t.Fatalf("missing capability err=%v", err)
	}
}

func rebindStage(t *testing.T, r *Request, edit func(*workflowdefinition.Stage)) {
	t.Helper()
	for index := range r.Definition.Definition.Stages {
		if r.Definition.Definition.Stages[index].ID == r.StageID {
			edit(&r.Definition.Definition.Stages[index])
		}
	}
	doc, diagnostics := workflowdefinition.Encode(r.Definition.Definition)
	if len(diagnostics) != 0 {
		t.Fatalf("invalid test stage: %+v", diagnostics)
	}
	r.Definition, r.Workflow = doc, doc.Ref("project")
}

func TestCompileSequentialConcurrencyAndUnsafeOverlap(t *testing.T) {
	r := multiRequest(t)
	rebindStage(t, &r, func(s *workflowdefinition.Stage) {
		s.Mode, s.Concurrency = "sequential", 1
		s.Agents[1].DependsOn = []string{s.Agents[0].ID}
	})
	sequential, err := New(testRuntimes{}, capabilityOK{}).Compile(context.Background(), r)
	if err != nil || len(sequential.Ordering) != 3 {
		t.Fatalf("sequential ordering=%v err=%v", sequential.Ordering, err)
	}
	for _, layer := range sequential.Ordering {
		if len(layer) != 1 {
			t.Fatal("sequential graph permits parallel execution")
		}
	}
	r = multiRequest(t)
	rebindStage(t, &r, func(s *workflowdefinition.Stage) { s.Concurrency = 1 })
	limited, err := New(testRuntimes{}, capabilityOK{}).Compile(context.Background(), r)
	if err != nil || len(limited.Proposal.Nodes[2].Dependencies) != 1 {
		t.Fatalf("concurrency limit missing dependency: %+v err=%v", limited.Proposal, err)
	}
	r = multiRequest(t)
	rebindStage(t, &r, func(s *workflowdefinition.Stage) {
		s.Agents[1].EffectCeilings = append(s.Agents[1].EffectCeilings, "repository-write")
	})
	write := executiongraph.Effect{Kind: "repository-write", Target: "src/shared.go"}
	r.ParentAuthority = []executiongraph.Effect{write}
	for index := 0; index < 2; index++ {
		r.Plan.Work[index].Scope.Paths = []string{"src/shared.go"}
		r.Plan.Work[index].Effects = []executiongraph.Effect{write}
	}
	if _, err := New(testRuntimes{}, capabilityOK{}).Compile(context.Background(), r); !errors.Is(err, executiongraph.ErrUnsafeOverlap) {
		t.Fatalf("unsafe parallel overlap err=%v", err)
	}
}

type memoryGraph struct{ graph executiongraph.Graph }

func (s *memoryGraph) Create(_ context.Context, g executiongraph.Graph) error {
	s.graph = g
	return nil
}
func (s *memoryGraph) Load(context.Context, string, string) (executiongraph.Graph, error) {
	return s.graph, nil
}

func TestCompiledInputsSurviveExistingEnvelopePublication(t *testing.T) {
	r := multiRequest(t)
	compiled, err := New(testRuntimes{}, capabilityOK{}).Compile(context.Background(), r)
	if err != nil {
		t.Fatal(err)
	}
	publication := executiongraph.PublicationRequest{Proposal: *compiled.Proposal, ExpectedDigest: compiled.Proposal.Digest, GraphRevision: 1, ParentAuthority: r.ParentAuthority, ChildAuthorities: map[string][]executiongraph.Effect{}, Resolutions: map[string]executiongraph.Resolution{}, Workspaces: map[string]string{}, AuthorityReferences: map[string]string{}}
	for _, agent := range compiled.Agents {
		key := agent.Input.NodeKey
		publication.ChildAuthorities[key] = agent.Effects
		publication.Resolutions[key] = executiongraph.Resolution{RuntimeID: agent.Choice.RuntimeID, ModelProfileID: agent.Choice.ModelProfileID, ConfigurationRevision: agent.Choice.ConfigurationRevision, ObservationRevision: agent.Choice.ObservationRevision}
		publication.Workspaces[key] = "isolated/" + agent.AgentID
		publication.AuthorityReferences[key] = "reviewed-authority-" + agent.AgentID
		ref, err := stageInputReference(agent.Input)
		if err != nil || ref != agent.InputReference {
			t.Fatal("stage input digest does not bind canonical bytes")
		}
	}
	next := 0
	graph, err := executiongraph.NewGraphService(&memoryGraph{}, func() (string, error) { next++; return fmt.Sprintf("00000000-0000-4000-8000-%012d", next), nil }, func() time.Time { return time.Unix(10, 0).UTC() }).Publish(context.Background(), publication)
	if err != nil {
		t.Fatal(err)
	}
	for _, child := range graph.Children {
		if child.ParentID != graph.Parent.ExecutionID {
			t.Fatal("child lost parent lineage")
		}
		for _, agent := range compiled.Agents {
			if agent.Input.NodeKey == child.NodeKey && (!contains(child.Envelope.Inputs, agent.InputReference.ID) || child.Envelope.Controls != agent.Controls) {
				t.Fatal("envelope lost stage reference or controls")
			}
		}
	}
}

type fixedPolicySource struct{ snapshot runtimeapplication.Snapshot }

func (s fixedPolicySource) Load(context.Context, string) (runtimeapplication.Snapshot, error) {
	return s.snapshot, nil
}

func TestCompileWithRealProjectPolicyAndModelSpecificObservations(t *testing.T) {
	const projectID = "123e4567-e89b-42d3-a456-426614174000"
	state := project.State{SchemaVersion: 2, ID: projectID, Slug: "example", Name: "Example"}
	var runtimes []project.Runtime
	var profiles []project.ModelProfile
	configuration := runtimeprofile.Configuration{FormatVersion: 1, Revision: 7}
	var observations []runtimeprofile.Observation
	for _, id := range []string{"codex", "claude"} {
		model, profileID := "local-"+id+"-model", "profile-"+id
		runtimes = append(runtimes, project.Runtime{ID: id})
		profiles = append(profiles, project.ModelProfile{Key: profileID, RuntimeRef: project.Configured(id), Model: project.Configured(model)})
		configuration.Runtimes = append(configuration.Runtimes, runtimeprofile.Runtime{ID: id, Adapter: id, Enabled: true, AllowlistedProfileIDs: []string{profileID}, CredentialReference: "env:PRIVATE_REFERENCE_ONLY"})
		configuration.ModelProfiles = append(configuration.ModelProfiles, runtimeprofile.ModelProfile{ID: profileID, RuntimeID: id, Model: model, Capabilities: []string{"read", "repository-write", "reasoning-effort-high", "reasoning-effort-medium"}, Complexities: []string{"medium"}})
		observations = append(observations, runtimeprofile.Observation{RuntimeID: id, Adapter: id, Installed: true, Available: true, Version: "test-1.0", ExecutableDigest: digestForTest(id), Revision: 7, ObservedAt: time.Unix(1, 0).UTC(), CapabilityStatus: map[string]runtimeprofile.CapabilityStatus{"read": runtimeprofile.CapabilityProven, "repository-write": runtimeprofile.CapabilityProven, "reasoning-effort-high": runtimeprofile.CapabilityProven, "reasoning-effort-medium": runtimeprofile.CapabilityProven}, NonInteractiveModelCapabilities: map[string]map[string]runtimeprofile.CapabilityStatus{model: {"reasoning-effort-high": runtimeprofile.CapabilityProven, "reasoning-effort-medium": runtimeprofile.CapabilityProven}}})
	}
	state.Runtimes, state.ModelProfiles = project.Configured(runtimes), project.Configured(profiles)
	state.RuntimePreferences = project.Configured([]project.RuntimePreference{{Role: "integrator", Complexity: "medium", ModelProfileRef: "profile-codex"}})
	configured, diagnostics := project.New(state)
	if len(diagnostics) != 0 {
		t.Fatal(diagnostics)
	}
	inventory, err := runtimeadapter.NewInventory(observations)
	if err != nil {
		t.Fatal(err)
	}
	service := runtimeapplication.New(fixedPolicySource{snapshot: runtimeapplication.Snapshot{Project: configured, Configuration: configuration, Observer: inventory}})
	r := multiRequest(t)
	r.ProjectID = projectID
	for index := range r.Plan.Work {
		r.Plan.Work[index].Scope.ProjectID = projectID
	}
	compiled, err := New(service, capabilityOK{}).Compile(context.Background(), r)
	if err != nil {
		t.Fatal(err)
	}
	for _, agent := range compiled.Agents {
		if _, err := service.Check(context.Background(), agent.Input.RuntimePreview); err != nil {
			t.Fatalf("compiled Runtime preview cannot be revalidated: %v", err)
		}
	}
	wire, _ := json.Marshal(compiled)
	if strings.Contains(string(wire), "PRIVATE_REFERENCE_ONLY") {
		t.Fatal("compiled proposal leaked credential reference")
	}
	observations[0].NonInteractiveModelCapabilities = nil
	inventory, err = runtimeadapter.NewInventory(observations)
	if err != nil {
		t.Fatal(err)
	}
	service = runtimeapplication.New(fixedPolicySource{snapshot: runtimeapplication.Snapshot{Project: configured, Configuration: configuration, Observer: inventory}})
	if _, err := New(service, capabilityOK{}).Compile(context.Background(), r); !errors.Is(err, ErrUnsupportedEffort) {
		t.Fatalf("Runtime-wide effort proof admitted: %v", err)
	}
}

func workflowSource(doc workflowdefinition.Document) string {
	if doc.Definition.WorkflowID == "default-sdd" {
		return "builtin"
	}
	return "project"
}
