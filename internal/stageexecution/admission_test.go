package stageexecution

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/executiongraph"
	"github.com/rgomids/axiom/internal/runtimeprofile"
	"github.com/rgomids/axiom/internal/workflow"
	"github.com/rgomids/axiom/internal/workflowcompiler"
	"github.com/rgomids/axiom/internal/workflowdefinition"
)

func graphFixture(t *testing.T) executiongraph.SuccessorGraph {
	t.Helper()
	wire, e := os.ReadFile("../executiongraph/testdata/axm36/graph-v4.json")
	if e != nil {
		t.Fatal(e)
	}
	g, e := executiongraph.DecodeSuccessorGraph(wire)
	if e != nil {
		t.Fatal(e)
	}
	var proposal executiongraph.Proposal
	wire, e = os.ReadFile("../executiongraph/testdata/axm36/proposal-v3.json")
	if e != nil {
		t.Fatal(e)
	}
	if e = json.Unmarshal(wire, &proposal); e != nil {
		t.Fatal(e)
	}
	g.Parent.ProposalDigest = proposal.Digest
	return g
}

func currentContract(t *testing.T) Contract {
	g := graphFixture(t)
	p := g.Children[0].Envelope.Resolution.FrozenProfile
	cfg := runtimeprofile.ConfigurationV4{FormatVersion: runtimeprofile.FormatVersionV4, Revision: 4,
		Runtimes: []runtimeprofile.RuntimeV4{{ID: p.RuntimeID, Adapter: p.RuntimeID, Enabled: true, AllowlistedProfileIDs: []string{p.ProfileID, "other-profile"}}},
		ModelProfiles: []runtimeprofile.ModelProfile{
			{ID: p.ProfileID, RuntimeID: p.RuntimeID, Model: p.Model, Effort: p.Effort, Capabilities: []string{"read"}},
			{ID: "other-profile", RuntimeID: p.RuntimeID, Model: "other-model", Effort: runtimeprofile.Effort{Mode: runtimeprofile.EffortRuntimeDefault, Value: "default"}, Capabilities: []string{"read"}},
		}}
	if err := runtimeprofile.ValidateV4(cfg); err != nil {
		t.Fatal(err)
	}
	return Contract{Definition: func() workflowdefinition.Document { d, _ := workflowdefinition.BuiltinRevision(2); return d }(), Execution: workflow.State{FormatVersion: workflow.OperationalConfiguredFormatVersion, Binding: &workflow.WorkflowBinding{SchemaVersion: 2, FrozenProfile: p}}, Record: "active-v4", Configuration: &cfg, Graph: &executiongraph.GraphDocument{Successor: &g}, WorkflowSelected: true, PolicyAllowed: true}
}

// withoutFrozenProfile removes the frozen Profile from the current allowlist.
func withoutFrozenProfile(c *Contract) {
	cfg := *c.Configuration
	cfg.Runtimes = append([]runtimeprofile.RuntimeV4(nil), cfg.Runtimes...)
	cfg.Runtimes[0].AllowlistedProfileIDs = []string{"other-profile"}
	c.Configuration = &cfg
}

func historicalGraph(t *testing.T, version int) executiongraph.GraphDocument {
	t.Helper()
	wire, err := os.ReadFile("../executiongraph/testdata/successor/valid/publication.json")
	if err != nil {
		t.Fatal(err)
	}
	g, err := executiongraph.DecodeSuccessorGraph(wire)
	if err != nil {
		t.Fatal(err)
	}
	base := executiongraph.Graph{FormatVersion: version, Parent: g.Parent}
	for _, c := range g.Children {
		if version == 1 {
			c.Envelope.Resolution.ResolvedProfile = nil
		}
		base.Children = append(base.Children, executiongraph.ChildExecution{ExecutionID: c.ExecutionID, ParentID: c.ParentID, GraphRevision: c.GraphRevision, NodeKey: c.NodeKey, Envelope: c.Envelope})
	}
	wire, err = executiongraph.EncodeGraph(base)
	if err != nil {
		t.Fatal(err)
	}
	d, err := executiongraph.DecodeGraphDocument(wire)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func TestAdmissionHistoricalMatrix(t *testing.T) {
	cases := []struct {
		name   string
		op     Operation
		change func(*Contract)
		want   error
		next   string
	}{
		{"row01-schema1-default", StagePlan, func(c *Contract) { c.Definition, _ = workflowdefinition.BuiltinRevision(1) }, workflowcompiler.ErrWorkflowMigrationRequired, ""},
		{"row02-schema1-explicit", StagePlan, func(c *Contract) {
			c.Definition, _ = workflowdefinition.BuiltinRevision(1)
			c.Definition.Definition.Stages[0].Agents[0].Effort = workflowdefinition.Effort{Mode: "explicit", Value: "high"}
		}, workflowcompiler.ErrWorkflowMigrationRequired, ""},
		{"row03-legacy-run", StageRun, func(c *Contract) { c.Execution.FormatVersion = 2 }, ErrLegacyExecution, ""},
		{"row04-legacy-retry", StageRetry, func(c *Contract) { c.Execution.FormatVersion = 1 }, ErrLegacyExecution, ""},
		// Row05 belongs to #274; #276 inspection must not alter its admission.
		{"row05-sequential-legacy-inspection", StageStatus, func(c *Contract) { c.Execution.FormatVersion = 2 }, nil, "legacy_execution_contract"},
		{"row06-legacy-coordination", CoordinationRead, func(c *Contract) { c.Execution.FormatVersion = 1 }, nil, "legacy_execution_contract"},
		{"row07-graph1", StageRun, func(c *Contract) { d := historicalGraph(t, 1); c.Graph = &d }, executiongraph.ErrUnsupportedGraphFormat, ""},
		{"row08-local1", StageRun, func(c *Contract) { c.Record = "active-v1" }, ErrProfileEffortUnconfigured, ""},
		{"row08-local2", StageRun, func(c *Contract) { c.Record = "active-v2" }, ErrProfileEffortUnconfigured, ""},
		{"local-missing", StageRun, func(c *Contract) {
			c.Record, c.Configuration = "fresh", nil
		}, ErrProfileEffortUnconfigured, ""},
		{"local-retired", StageRun, func(c *Contract) {
			c.Record, c.Configuration = "retired", nil
		}, ErrProfileEffortUnconfigured, ""},
		{"row09-stage-plan", StagePlan, func(c *Contract) { c.Graph = nil }, nil, ""},
		{"row10-first-attempt-next-gates", StageRun, nil, nil, ""},
		{"row11-retry-next-gates", StageRetry, nil, nil, ""},
		{"row12-resume-next-gates", StageResume, nil, nil, ""},
		{"row13-migration-not-selected", StageRun, func(c *Contract) { c.WorkflowSelected = false }, workflowcompiler.ErrWorkflowMigrationRequired, ""},
		{"row13-missing-profile", StageRun, withoutFrozenProfile, ErrProfileEffortUnconfigured, ""},
		{"row13-policy", StageRun, func(c *Contract) { c.PolicyAllowed = false }, ErrPolicyUpdateRequired, ""},
		{"row14-removed-profile", StageRetry, withoutFrozenProfile, ErrProfileEffortUnconfigured, ""},
		{"row15-default-r1", StagePlan, func(c *Contract) { c.Definition, _ = workflowdefinition.BuiltinRevision(1) }, workflowcompiler.ErrWorkflowMigrationRequired, ""},
		{"successor-delta-graph2", StageRun, func(c *Contract) { d := historicalGraph(t, 2); c.Graph = &d }, executiongraph.ErrUnsupportedGraphFormat, ""},
		{"unknown-successor", StageRun, func(c *Contract) { c.Graph.Successor.FormatVersion = 99 }, executiongraph.ErrUnsupportedGraphFormat, ""},
		{"missing-successor-version", StageRun, func(c *Contract) { c.Graph.Successor.FormatVersion = 0 }, executiongraph.ErrUnsupportedGraphFormat, ""},
		{"missing-classification", StageRun, func(c *Contract) { c.Graph.Successor.ClassificationPolicy = "" }, executiongraph.ErrUnsupportedGraphFormat, ""},
		{"missing-frozen-tuple", StageRun, func(c *Contract) { c.Execution.Binding.FrozenProfile = nil }, ErrLegacyExecution, ""},
		{"unknown-operation", Operation("stage.arbitrary"), nil, ErrAdmission, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := currentContract(t)
			if tc.change != nil {
				tc.change(&c)
			}
			got, e := Admit(tc.op, c)
			if !errors.Is(e, tc.want) || got.Next != tc.next {
				t.Fatalf("admission=%+v err=%v want=%v next=%s", got, e, tc.want, tc.next)
			}
		})
	}
}
func TestAllHistoricalGraphMutationsRefusedAndInspectionPreservesBytes(t *testing.T) {
	for _, v := range []int{1, 2} {
		d := historicalGraph(t, v)
		before, e := executiongraph.EncodeGraphDocument(d)
		if e != nil {
			t.Fatal(e)
		}
		digest := sha256.Sum256(before)
		for _, op := range []Operation{StageRun, StageRetry, StageResume, StageCancel, StageReconcile, StageIntegrate} {
			c := currentContract(t)
			c.Graph = &d
			if _, e := Admit(op, c); !errors.Is(e, executiongraph.ErrUnsupportedGraphFormat) {
				t.Fatal(v, op, e)
			}
		}
		for _, op := range []Operation{StageStatus, CoordinationRead} {
			c := currentContract(t)
			c.Graph = &d
			if a, e := Admit(op, c); e != nil || a.Next != "unsupported_graph_format" {
				t.Fatal(a, e)
			}
		}
		after, _ := executiongraph.EncodeGraphDocument(d)
		if !bytes.Equal(before, after) || sha256.Sum256(after) != digest {
			t.Fatal("historical bytes changed")
		}
	}
}
func TestUnknownWireNeverFallsBack(t *testing.T) {
	g := graphFixture(t)
	wire, _ := executiongraph.EncodeSuccessorGraph(g)
	for _, bad := range []string{strings.Replace(string(wire), `"formatVersion":4`, `"formatVersion":99`, 1), strings.Replace(string(wire), `"formatVersion":4,`, "", 1), strings.Replace(string(wire), `"classificationPolicy":"stage-attempt-v1",`, "", 1)} {
		if d, e := executiongraph.DecodeGraphDocument([]byte(bad)); e == nil || d.Historical != nil || d.Successor != nil {
			t.Fatal("fallback", e)
		}
	}
}
