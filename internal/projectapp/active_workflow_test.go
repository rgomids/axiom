package projectapp_test

import (
	"github.com/rgomids/axiom/internal/project"
	"github.com/rgomids/axiom/internal/projectapp"
	"github.com/rgomids/axiom/internal/workflowdefinition"
	"strings"
	"testing"
)

func TestActiveWorkflowExactResolution(t *testing.T) {
	port, custom := workflowFixture(t)
	builtin := workflowdefinition.Builtin()
	cases := []struct {
		name             string
		ref              workflowdefinition.Ref
		status, category string
		change           func(*workflowdefinition.Catalog)
	}{
		{name: "none", status: "none"},
		{name: "builtin", ref: builtin.Ref("builtin"), status: "selected"},
		{name: "project", ref: custom.Ref("project"), status: "selected"},
		{name: "builtin digest", ref: workflowdefinition.Ref{WorkflowID: builtin.Definition.WorkflowID, Revision: 1, Digest: strings.Repeat("a", 64), Source: "builtin"}, status: "unresolvable", category: "recovery_required"},
		{name: "builtin revision", ref: workflowdefinition.Ref{WorkflowID: builtin.Definition.WorkflowID, Revision: 2, Digest: builtin.Digest, Source: "builtin"}, status: "unresolvable", category: "recovery_required"},
		{name: "missing document", ref: custom.Ref("project"), status: "unresolvable", category: "recovery_required", change: func(c *workflowdefinition.Catalog) { c.Documents = nil }},
		{name: "wrong identity", ref: custom.Ref("project"), status: "unresolvable", category: "recovery_required", change: func(c *workflowdefinition.Catalog) {
			c.Documents = map[string]workflowdefinition.Document{"custom/1.json": builtin}
		}},
		{name: "content digest", ref: custom.Ref("project"), status: "unresolvable", category: "recovery_required", change: func(c *workflowdefinition.Catalog) {
			changed := custom
			changed.Definition.Name = "Changed"
			c.Documents = map[string]workflowdefinition.Document{"custom/1.json": changed}
		}},
		{name: "index digest", ref: custom.Ref("project"), status: "unresolvable", category: "recovery_required", change: func(c *workflowdefinition.Catalog) {
			c.Index.Revisions = []workflowdefinition.Entry{{WorkflowID: "custom", Revision: 1, Digest: strings.Repeat("a", 64), State: "published"}}
		}},
		{name: "retired index", ref: custom.Ref("project"), status: "unresolvable", category: "recovery_required", change: func(c *workflowdefinition.Catalog) {
			c.Index.Revisions = []workflowdefinition.Entry{{WorkflowID: "custom", Revision: 1, Digest: custom.Digest, State: "retired"}}
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			state := port.observed.Selection.Portable.Project().State()
			if tc.ref.WorkflowID != "" {
				state.WorkflowSelection = project.Configured(project.WorkflowSelection{WorkflowID: tc.ref.WorkflowID, Revision: tc.ref.Revision, Digest: tc.ref.Digest, Source: tc.ref.Source})
			}
			catalog := port.observed.Catalog
			if tc.change != nil {
				tc.change(&catalog)
			}
			got := projectapp.ResolveActiveWorkflow(state, catalog, "")
			if got.Status != tc.status || got.Category != tc.category {
				t.Fatalf("%+v", got)
			}
			if tc.status == "selected" && got.Name == "" {
				t.Fatal("missing resolved name")
			}
			if tc.status == "unresolvable" && (got.Name != "" || got.WorkflowID != tc.ref.WorkflowID || got.Digest != tc.ref.Digest || got.Revision != tc.ref.Revision || got.Source != tc.ref.Source) {
				t.Fatalf("reference changed: %+v", got)
			}
		})
	}
	state := port.observed.Selection.Portable.Project().State()
	state.WorkflowSelection = project.Configured(project.WorkflowSelection{WorkflowID: "custom", Revision: 1, Digest: custom.Digest, Source: "project"})
	got := projectapp.ResolveActiveWorkflow(state, port.observed.Catalog, "project_configuration_drift")
	if got.Status != "unresolvable" || got.Category != "project_configuration_drift" || got.Name != "" || got.Digest != custom.Digest {
		t.Fatalf("%+v", got)
	}
	if port.writes != 0 {
		t.Fatal("inspection wrote state")
	}
}
