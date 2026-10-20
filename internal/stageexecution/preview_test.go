package stageexecution

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/rgomids/axiom/internal/executiongraph"
	"github.com/rgomids/axiom/internal/workflowcompiler"
	"github.com/rgomids/axiom/internal/workflowdefinition"
)

type fakePreviewSource struct {
	o      PreviewObservation
	calls  int
	change func(int, *PreviewObservation)
}

func (f *fakePreviewSource) Observe(_ context.Context, _ workflowcompiler.ExecutionRef, _ string) (PreviewObservation, error) {
	f.calls++
	if f.change != nil {
		f.change(f.calls, &f.o)
	}
	return f.o, nil
}

type fakePlanCompiler struct {
	plan  workflowcompiler.StagePlan
	calls int
}

func (f *fakePlanCompiler) PlanStageV2(_ context.Context, r workflowcompiler.StagePlanRequest) (workflowcompiler.StagePlan, error) {
	f.calls++
	if r.Execution != f.plan.Execution {
		return workflowcompiler.StagePlan{}, ErrStalePreview
	}
	return f.plan, nil
}

func previewFixture(t *testing.T, single bool) (PreviewService, *fakePreviewSource, *fakePlanCompiler) {
	t.Helper()
	g := graphFixture(t)
	c := currentContract(t)
	c.Graph = nil
	// The graph parent is an allocated identity, distinct from the Execution.
	ref := workflowcompiler.ExecutionRef{ExecutionID: "00000000-0000-4000-8000-0000000005e1", Revision: 1}
	plan := workflowcompiler.StagePlan{FormatVersion: 3, Execution: ref, ProjectID: g.Children[0].Envelope.Scope.ProjectID, RepositoryKey: "main", Workflow: c.Definition.Ref("builtin"), StageID: "implementation", ExecutionKind: "graph", PlanDocumentDigest: strings.Repeat("a", 64), Digest: strings.Repeat("b", 64), Compilation: workflowcompiler.Result{FormatVersion: 3, Concurrency: 1, ExecutionKind: "graph", AuthorityCeiling: g.Parent.AuthorityEffects}}
	o := PreviewObservation{Contract: c, PolicyDigest: strings.Repeat("c", 64), ValidatorPolicyDigest: strings.Repeat("d", 64), AxiomDigest: strings.Repeat("e", 64), Workspaces: map[string]string{}, Inboxes: map[string]string{}}
	wire, e := os.ReadFile("testdata/stage-plan-v3.json")
	if e != nil {
		t.Fatal(e)
	}
	var compiled workflowcompiler.StagePlan
	if e = json.Unmarshal(wire, &compiled); e != nil {
		t.Fatal(e)
	}
	plan.Compilation = compiled.Compilation
	plan.Resolutions = compiled.Resolutions
	plan.GraphProposal = compiled.GraphProposal
	plan.Workflow = compiled.Workflow
	definitionWire, e := os.ReadFile("../workflowcompiler/testdata/issue306/workflow-v2.json")
	if e != nil {
		t.Fatal(e)
	}
	definition, issues := workflowdefinition.Decode(definitionWire)
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	c.Definition = definition
	c.Execution.ExecutionID = ref.ExecutionID
	c.Execution.Revision = ref.Revision
	c.Execution.Stage = "implementation"
	c.Execution.Binding.Definition = plan.Workflow
	o.Contract = c
	for i, ch := range g.Children {
		if single && i > 0 {
			break
		}
		o.Workspaces[ch.NodeKey] = ch.Envelope.Workspace
		o.Inboxes[ch.NodeKey] = "/synthetic/inbox/" + ch.NodeKey
	}
	if single {
		plan.ExecutionKind = "single"
		plan.Compilation.ExecutionKind = "single"
		plan.Compilation.Agents = plan.Compilation.Agents[:1]
		plan.Compilation.Proposal = nil
		plan.Resolutions = plan.Resolutions[:1]
		plan.GraphProposal = nil
	}
	o.Request = workflowcompiler.StagePlanRequest{Execution: ref, PlanDocumentDigest: plan.PlanDocumentDigest, Compile: workflowcompiler.Request{Definition: c.Definition, Workflow: plan.Workflow, StageID: plan.StageID}}
	o.Reviewed = plan
	source := &fakePreviewSource{o: o}
	compiler := &fakePlanCompiler{plan: plan}
	return PreviewService{Source: source, Compiler: compiler}, source, compiler
}
func TestDispatchPreviewGraphAndSingle(t *testing.T) {
	for _, single := range []bool{false, true} {
		t.Run(map[bool]string{false: "graph", true: "single"}[single], func(t *testing.T) {
			s, source, compiler := previewFixture(t, single)
			a, e := s.Preview(context.Background(), compiler.plan.Execution, compiler.plan.StageID)
			if e != nil {
				t.Fatal(e)
			}
			b, e := s.Preview(context.Background(), compiler.plan.Execution, compiler.plan.StageID)
			if e != nil || !reflect.DeepEqual(a, b) {
				t.Fatal("non-deterministic preview", e)
			}
			if (a.Publication == nil) != single || source.calls != 2 || compiler.calls != 2 {
				t.Fatal("wrong path", a)
			}
			digest, _ := previewDigest(a)
			if a.Digest != digest {
				t.Fatal("incorrect digest")
			}
		})
	}
}
func TestDispatchPreviewRightsGolden(t *testing.T) {
	s, _, compiler := previewFixture(t, false)
	p, e := s.Preview(context.Background(), compiler.plan.Execution, compiler.plan.StageID)
	if e != nil {
		t.Fatal(e)
	}
	type surface struct {
		Confinement, Reads, Commands string
		Prohibitions                 map[string]string
		NotTested                    []string
	}
	got := map[string]surface{}
	for i, c := range p.Children {
		got[p.Plan.Compilation.Agents[i].Input.FrozenProfile.RuntimeID] = surface{c.Confinement, c.Reads, c.Commands, c.Prohibitions, c.NotTested}
	}
	wire, _ := json.MarshalIndent(got, "", "  ")
	wire = append(wire, '\n')
	want, e := os.ReadFile("testdata/dispatch-rights.json")
	if e != nil {
		t.Fatal(e)
	}
	if string(wire) != string(want) {
		t.Fatalf("rights differ:\n%s", wire)
	}
}
func TestDispatchPreviewRefusals(t *testing.T) {
	cases := []struct {
		name   string
		change func(*fakePreviewSource, *fakePlanCompiler)
		want   error
	}{
		{"plan digest mismatch", func(s *fakePreviewSource, c *fakePlanCompiler) {
			c.plan.Digest = strings.Repeat("f", 64)
		}, ErrStalePreview},
		{"plan document mismatch", func(s *fakePreviewSource, c *fakePlanCompiler) {
			c.plan.PlanDocumentDigest = strings.Repeat("f", 64)
		}, ErrStalePreview},
		{"missing frozen tuple", func(s *fakePreviewSource, c *fakePlanCompiler) {
			c.plan.Compilation.Agents[0].Input.FrozenProfile = nil
			s.o.Reviewed = c.plan
		}, ErrPreview},
		{"legacy execution", func(s *fakePreviewSource, _ *fakePlanCompiler) {
			s.o.Contract.Execution.FormatVersion = 2
		}, ErrLegacyExecution},
		{"legacy workflow", func(s *fakePreviewSource, _ *fakePlanCompiler) {
			s.o.Contract.Definition.Definition.SchemaVersion = 1
		}, workflowcompiler.ErrWorkflowMigrationRequired},
		{"policy refused", func(s *fakePreviewSource, _ *fakePlanCompiler) {
			s.o.Contract.PolicyAllowed = false
		}, ErrPolicyUpdateRequired},
		{"33 nodes", func(s *fakePreviewSource, c *fakePlanCompiler) {
			for len(c.plan.Compilation.Agents) < 33 {
				c.plan.Compilation.Agents = append(c.plan.Compilation.Agents, c.plan.Compilation.Agents[0])
			}
			s.o.Reviewed = c.plan
		}, ErrPreview},
		{"33-node proposal", func(s *fakePreviewSource, c *fakePlanCompiler) {
			for len(c.plan.Compilation.Proposal.Nodes) < 33 {
				c.plan.Compilation.Proposal.Nodes = append(c.plan.Compilation.Proposal.Nodes, c.plan.Compilation.Proposal.Nodes[0])
			}
			s.o.Reviewed = c.plan
		}, ErrPreview},
		{"historical graph", func(s *fakePreviewSource, _ *fakePlanCompiler) {
			d := historicalGraph(t, 1)
			s.o.Contract.Graph = &d
		}, executiongraph.ErrUnsupportedGraphFormat},
		{"zero attempts", func(s *fakePreviewSource, c *fakePlanCompiler) {
			c.plan.Compilation.Agents[0].Controls.MaximumAttempts = 0
			s.o.Reviewed = c.plan
		}, ErrPreview},
		{"11 attempts", func(s *fakePreviewSource, c *fakePlanCompiler) {
			c.plan.Compilation.Agents[0].Controls.MaximumAttempts = 11
			s.o.Reviewed = c.plan
		}, ErrPreview},
		{"over 24h", func(s *fakePreviewSource, c *fakePlanCompiler) {
			c.plan.Compilation.Agents[0].Controls.Timeout = 24*time.Hour + time.Nanosecond
			s.o.Reviewed = c.plan
		}, ErrPreview},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s, source, compiler := previewFixture(t, false)
			tc.change(source, compiler)
			_, e := s.Preview(context.Background(), compiler.plan.Execution, compiler.plan.StageID)
			if !errors.Is(e, tc.want) {
				t.Fatalf("got %v want %v", e, tc.want)
			}
		})
	}
}
func TestPreviewBindsPolicyAndDisclosure(t *testing.T) {
	s, source, compiler := previewFixture(t, false)
	a, e := s.Preview(context.Background(), compiler.plan.Execution, compiler.plan.StageID)
	if e != nil {
		t.Fatal(e)
	}
	source.o.PolicyDigest = strings.Repeat("f", 64)
	b, e := s.Preview(context.Background(), compiler.plan.Execution, compiler.plan.StageID)
	if e != nil || a.Digest == b.Digest {
		t.Fatal("policy drift unbound", e)
	}
	advisory := false
	for _, c := range b.Children {
		if c.Confinement == "unknown" {
			advisory = true
			for _, v := range c.Prohibitions {
				if v != "policy-only" {
					t.Fatal("advisory enforcement overstated")
				}
			}
		}
	}
	if !advisory {
		t.Fatal("advisory child omitted")
	}
}
func TestDispatchPreviewBindsProcessMode(t *testing.T) {
	s, source, compiler := previewFixture(t, true)
	before, err := s.Preview(context.Background(), compiler.plan.Execution, compiler.plan.StageID)
	if err != nil {
		t.Fatal(err)
	}
	agent := &compiler.plan.Compilation.Agents[0]
	agent.Effects = append(agent.Effects, executiongraph.Effect{Kind: "process", Target: "worktree"})
	source.o.Reviewed = compiler.plan
	after, err := s.Preview(context.Background(), compiler.plan.Execution, compiler.plan.StageID)
	if err != nil || before.Digest == after.Digest || !reflect.DeepEqual(after.Children[0].Effects, agent.Effects) {
		t.Fatalf("process authority not bound: %+v %v", after, err)
	}
}
