package projectapp_test

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/project"
	"github.com/rgomids/axiom/internal/projectapp"
	"github.com/rgomids/axiom/internal/workflowdefinition"
)

func selectedPolicy(t *testing.T, p project.Project, doc workflowdefinition.Document, source string) project.Project {
	t.Helper()
	ref := doc.Ref(source)
	next, issues := p.SelectWorkflow(project.WorkflowSelection{WorkflowID: ref.WorkflowID, Revision: ref.Revision, Digest: ref.Digest, Source: ref.Source})
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	return next
}

func workflowPolicyState(t *testing.T, p project.Project, edit func(*project.State)) project.Project {
	t.Helper()
	state := p.State()
	edit(&state)
	next, issues := project.New(state)
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	return next
}

func TestPolicyWorkflowExactDefaultAndExplicitReferences(t *testing.T) {
	for _, schema := range []int{1, 2} {
		definition := workflowdefinition.Builtin().Definition
		definition.SchemaVersion = schema
		definition.WorkflowID = "policy-test"
		for i := range definition.Stages {
			agent := &definition.Stages[i].Agents[0]
			agent.ProfileRef = "review"
			if schema == 2 {
				agent.Effort = workflowdefinition.Effort{}
			}
		}
		doc, issues := workflowdefinition.Encode(definition)
		if len(issues) != 0 {
			t.Fatal(issues)
		}
		catalog := workflowdefinition.Catalog{Index: workflowdefinition.Index{SchemaVersion: 1, Revisions: []workflowdefinition.Entry{{WorkflowID: doc.Definition.WorkflowID, Revision: 1, Digest: doc.Digest, State: "published"}}}, Documents: map[string]workflowdefinition.Document{"policy-test/1.json": doc}}
		original := selectedPolicy(t, policyProject(t, 3), doc, "project")
		next := workflowPolicyState(t, original, func(s *project.State) {
			s.RuntimePreferences = project.Configured([]project.RuntimePreference{{Role: "engineer", Complexity: "medium", ModelProfileRef: "coding"}})
		})
		report := projectapp.CheckPolicyWorkflow(original, next, catalog, "")
		if !report.Compatible || report.Workflow.Digest != doc.Digest || report.SchemaVersion != schema || len(report.Impact) != len(definition.Stages) {
			t.Fatalf("bad exact report %+v", report)
		}
		for _, row := range report.Impact {
			if len(row.After.Profiles) != 1 || row.After.Profiles[0].Profile != "review" || row.After.Profiles[0].Runtime != "codex" {
				t.Fatalf("explicit Profile overridden: %+v", row)
			}
		}
		removed := workflowPolicyState(t, next, func(s *project.State) { ps, _ := s.ModelProfiles.Value(); s.ModelProfiles = project.Configured(ps[:1]) })
		report = projectapp.CheckPolicyWorkflow(next, removed, catalog, "")
		if report.Compatible || report.Reason != "workflow_policy_incompatible" {
			t.Fatal("removed explicit Profile accepted")
		}
		for _, row := range report.Impact {
			if row.Change != "introduced_incompatibility" || row.After.Reason != "profile_missing" || row.After.Profiles[0].Profile != "review" {
				t.Fatalf("lost exact removed reference %+v", row)
			}
		}
		report = projectapp.CheckPolicyWorkflow(removed, removed, catalog, "")
		if report.Compatible {
			t.Fatal("retained incompatibility accepted")
		}
		for _, row := range report.Impact {
			if row.Change != "unresolved_incompatibility" {
				t.Fatal(row)
			}
		}
		report = projectapp.CheckPolicyWorkflow(removed, next, catalog, "")
		if !report.Compatible {
			t.Fatal("explicit repair rejected")
		}
	}
}

func TestPolicyWorkflowHistoricalPreferencePrecedence(t *testing.T) {
	builtin := workflowdefinition.Builtin()
	original := selectedPolicy(t, policyProject(t, 2), builtin, "builtin")
	report := projectapp.CheckPolicyWorkflow(original, original, workflowdefinition.Catalog{}, "")
	if !report.Compatible || report.Inventory != "unknown_uninspected" || len(report.Scopes) != 4 {
		t.Fatalf("invalid default impact %+v", report)
	}
	for _, row := range report.Impact {
		if len(row.After.Profiles) != 2 {
			t.Fatal("multiple admissible Profiles silently selected")
		}
	}
	next := workflowPolicyState(t, original, func(s *project.State) {
		s.RuntimePreferences = project.Configured([]project.RuntimePreference{{Role: "engineer", Complexity: "medium", ModelProfileRef: "review"}})
	})
	report = projectapp.CheckPolicyWorkflow(original, next, workflowdefinition.Catalog{}, "")
	if !report.Compatible {
		t.Fatal(report.Reason)
	}
	for _, row := range report.Impact {
		if len(row.After.Profiles) != 1 || row.After.Profiles[0].Profile != "review" {
			t.Fatal("exact preference ignored")
		}
	}
	constrained := workflowPolicyState(t, next, func(s *project.State) {
		rs, _ := s.Runtimes.Value()
		s.Runtimes = project.Configured(append(rs, project.Runtime{ID: "other"}))
		ps, _ := s.ModelProfiles.Value()
		ps[1].RuntimeRef = project.Configured("other")
		s.ModelProfiles = project.Configured(ps)
	})
	report = projectapp.CheckPolicyWorkflow(next, constrained, workflowdefinition.Catalog{}, "")
	if report.Compatible {
		t.Fatal("disallowed preference fell back")
	}
	for _, row := range report.Impact {
		if row.After.Reason != "profile_runtime_mismatch" || row.After.Profiles[0].Runtime != "other" {
			t.Fatal(row)
		}
	}
	unconfigured := workflowPolicyState(t, original, func(s *project.State) {
		s.ModelProfiles = project.Unconfigured[[]project.ModelProfile]()
		s.RuntimePreferences = project.Declaration[[]project.RuntimePreference]{}
	})
	report = projectapp.CheckPolicyWorkflow(original, unconfigured, workflowdefinition.Catalog{}, "")
	if report.Compatible || report.Impact[0].After.Reason != "portable_candidates_empty" {
		t.Fatal("empty admissible set accepted")
	}
	v2, _ := workflowdefinition.BuiltinRevision(2)
	explicitDefault := selectedPolicy(t, policyProject(t, 3), v2, "builtin")
	report = projectapp.CheckPolicyWorkflow(explicitDefault, explicitDefault, workflowdefinition.Catalog{}, "")
	if report.Compatible || report.Impact[0].After.Reason != "profile_missing" || report.Impact[0].After.Profiles[0].Profile != "policy-default" {
		t.Fatal("schema-2 explicit Profile reinterpreted as historical default")
	}
}

func TestPolicyWorkflowMissingTamperedAndReselected(t *testing.T) {
	doc := workflowdefinition.Builtin()
	original := selectedPolicy(t, policyProject(t, 3), doc, "project")
	catalog := workflowdefinition.Catalog{Index: workflowdefinition.Index{SchemaVersion: 1, Revisions: []workflowdefinition.Entry{{WorkflowID: doc.Definition.WorkflowID, Revision: 1, Digest: doc.Digest, State: "published"}}}, Documents: map[string]workflowdefinition.Document{doc.Definition.WorkflowID + "/1.json": doc}}
	for _, mutate := range []func(*workflowdefinition.Catalog){
		func(c *workflowdefinition.Catalog) { c.Documents = nil },
		func(c *workflowdefinition.Catalog) { c.Index.Revisions = nil },
		func(c *workflowdefinition.Catalog) { c.Index.Revisions[0].Digest = strings.Repeat("a", 64) },
		func(c *workflowdefinition.Catalog) { c.Index.Revisions[0].State = "retired" },
		func(c *workflowdefinition.Catalog) {
			d := doc
			d.Definition.Name = "tampered"
			c.Documents = map[string]workflowdefinition.Document{doc.Definition.WorkflowID + "/1.json": d}
		},
		func(c *workflowdefinition.Catalog) { c.Unindexed = []workflowdefinition.Ref{doc.Ref("project")} },
	} {
		broken := catalog
		broken.Index.Revisions = append([]workflowdefinition.Entry(nil), catalog.Index.Revisions...)
		mutate(&broken)
		report := projectapp.CheckPolicyWorkflow(original, original, broken, "")
		if report.Compatible || report.Reason != "workflow_unresolvable" || len(report.Impact) != 0 {
			t.Fatalf("tampered catalog accepted %+v", report)
		}
	}
	report := projectapp.CheckPolicyWorkflow(original, original, catalog, "sentinel-machine-error")
	b, _ := json.Marshal(report)
	if report.Compatible || strings.Contains(string(b), "sentinel-machine-error") {
		t.Fatal("unreadable selection accepted or raw error leaked")
	}
	reselection := selectedPolicy(t, policyProject(t, 3), doc, "builtin")
	report = projectapp.CheckPolicyWorkflow(original, reselection, catalog, "")
	if report.Compatible || report.Reason != "workflow_selection_changed" {
		t.Fatal("selection rewrite accepted")
	}
	for _, version := range []int{2, 3} {
		p := policyProject(t, version)
		report = projectapp.CheckPolicyWorkflow(p, p, workflowdefinition.Catalog{}, "")
		if !report.Compatible || report.Workflow.Status != "none" {
			t.Fatal("none selection fabricated")
		}
	}
}

func TestPolicyWorkflowBoundedCompleteImpact(t *testing.T) {
	doc := workflowdefinition.Builtin()
	p := selectedPolicy(t, policyProject(t, 3), doc, "builtin")
	p = workflowPolicyState(t, p, func(s *project.State) {
		profiles := []project.ModelProfile{}
		for i := 0; i < 32; i++ {
			profiles = append(profiles, project.ModelProfile{Key: fmt.Sprintf("profile-%d-", i) + strings.Repeat("x", 1500), RuntimeRef: project.Configured("codex"), Model: project.Configured("model")})
		}
		s.ModelProfiles = project.Configured(profiles)
		s.RuntimePreferences = project.Declaration[[]project.RuntimePreference]{}
	})
	state := p.State()
	report := projectapp.CheckPolicyWorkflow(p, p, workflowdefinition.Catalog{}, "")
	if report.Compatible || report.Reason != "workflow_impact_limit" || len(report.Impact) != 0 {
		t.Fatal("oversized impact truncated into accepted compatibility")
	}
	wire, _ := json.Marshal(report)
	if len(wire) > projectapp.MaxPolicyDraftBytes {
		t.Fatal("overflow error unbounded")
	}
	if !reflect.DeepEqual(p.State(), state) {
		t.Fatal("analysis changed Project")
	}
}
