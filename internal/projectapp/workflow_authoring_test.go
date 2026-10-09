package projectapp_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/manifest"
	"github.com/rgomids/axiom/internal/project"
	"github.com/rgomids/axiom/internal/projectapp"
	"github.com/rgomids/axiom/internal/workflowdefinition"
)

type workflowPort struct {
	observed projectapp.WorkflowObservation
	writes   int
}

func (p *workflowPort) ObserveWorkflows(context.Context, string) (projectapp.WorkflowObservation, string) {
	return p.observed, ""
}
func (p *workflowPort) PublishWorkflowRevision(context.Context, projectapp.WorkflowObservation, *workflowdefinition.Document, workflowdefinition.Index) string {
	p.writes++
	return "applied"
}
func (p *workflowPort) PublishWorkflowSelection(context.Context, projectapp.WorkflowObservation, project.Project) string {
	p.writes++
	return "applied"
}
func workflowFixture(t *testing.T) (*workflowPort, workflowdefinition.Document) {
	t.Helper()
	p, issues := project.New(project.State{SchemaVersion: 3, ID: "12345678-1234-4abc-8def-123456789abc", Slug: "sample", Name: "Sample"})
	if len(issues) > 0 {
		t.Fatal(issues)
	}
	wire, issues := manifest.Encode(p)
	if len(issues) > 0 {
		t.Fatal(issues)
	}
	snapshot, problems := projectapp.ReadSnapshot(manifest.Codec{}, wire, nil)
	if len(problems) > 0 {
		t.Fatal(problems)
	}
	d := workflowdefinition.Builtin().Definition
	d.WorkflowID = "custom"
	doc, diags := workflowdefinition.Encode(d)
	if len(diags) > 0 {
		t.Fatal(diags)
	}
	catalog := workflowdefinition.Catalog{Index: workflowdefinition.Index{SchemaVersion: 1, Revisions: []workflowdefinition.Entry{{WorkflowID: "custom", Revision: 1, Digest: doc.Digest, State: "published"}}}, Documents: map[string]workflowdefinition.Document{"custom/1.json": doc}}
	return &workflowPort{observed: projectapp.WorkflowObservation{Selection: projectapp.EditSelection{Portable: snapshot, PortableRevision: strings.Repeat("a", 64), LocalRevision: strings.Repeat("b", 64)}, Catalog: catalog, ReferencesChecked: true}}, doc
}
func TestWorkflowRetirementRequiresCompleteReferenceInventory(t *testing.T) {
	ctx := context.Background()
	p, doc := workflowFixture(t)
	r := projectapp.WorkflowRequest{Operation: "remove", Project: "sample", Ref: doc.Ref("project")}
	p.observed.Referenced = []workflowdefinition.Ref{r.Ref}
	if out := projectapp.ApplyWorkflow(ctx, p, r); out.Category != "revision_in_use" || p.writes != 0 {
		t.Fatalf("referenced retirement: %+v", out)
	}
	p.observed.Referenced = nil
	p.observed.ReferencesChecked = false
	if out := projectapp.ApplyWorkflow(ctx, p, r); out.Category != "reference_inventory_unknown" || p.writes != 0 {
		t.Fatal(out)
	}
	p.observed.ReferencesChecked = true
	preview := projectapp.ApplyWorkflow(ctx, p, r)
	if preview.Category != "previewed" || p.writes != 0 {
		t.Fatal(preview)
	}
	r.ExpectedRevision = p.observed.Selection.PortableRevision
	r.PreviewDigest = preview.Report.PreviewDigest
	r.AuthorizeLocal = true
	if out := projectapp.ApplyWorkflow(ctx, p, r); out.Category != "applied" || p.writes != 1 {
		t.Fatal(out)
	}
}
func TestWorkflowPreviewBindsIndexContentAndLocalRevision(t *testing.T) {
	ctx := context.Background()
	p, doc := workflowFixture(t)
	d := doc.Definition
	d.Revision = 2
	next, issues := workflowdefinition.Encode(d)
	if len(issues) > 0 {
		t.Fatal(issues)
	}
	r := projectapp.WorkflowRequest{Operation: "edit", Project: "sample", Definition: next.Canonical, Prior: doc.Ref("project")}
	preview := projectapp.ApplyWorkflow(ctx, p, r)
	if preview.Category != "previewed" {
		t.Fatal(preview)
	}
	r.ExpectedRevision = p.observed.Selection.PortableRevision
	r.PreviewDigest = preview.Report.PreviewDigest
	r.AuthorizeLocal = true
	p.observed.Selection.LocalRevision = strings.Repeat("c", 64)
	if out := projectapp.ApplyWorkflow(ctx, p, r); out.Category != "stale_authority" || p.writes != 0 {
		t.Fatal(out)
	}
	p.observed.Selection.LocalRevision = strings.Repeat("b", 64)
	p.observed.Catalog.Index.Revisions = append(p.observed.Catalog.Index.Revisions, workflowdefinition.Entry{WorkflowID: "another", Revision: 1, Digest: strings.Repeat("d", 64), State: "published"})
	if out := projectapp.ApplyWorkflow(ctx, p, r); out.Category != "stale_authority" || p.writes != 0 {
		t.Fatal(out)
	}
}
func TestWorkflowProjectReferencesAndCapabilitiesFailBeforeApply(t *testing.T) {
	for name, change := range map[string]func(*workflowdefinition.Definition){
		"context": func(d *workflowdefinition.Definition) {
			d.Stages[0].Inputs[0].Kind = "project-context"
			d.Stages[0].Inputs[0].Source = "missing"
		},
		"profile":    func(d *workflowdefinition.Definition) { d.Stages[0].Agents[0].ProfileRef = "missing" },
		"validator":  func(d *workflowdefinition.Definition) { d.Stages[0].Validators[0].PolicyRef = "missing" },
		"capability": func(d *workflowdefinition.Definition) { d.Stages[0].Agents[0].Capabilities = []string{"unsupported"} },
	} {
		t.Run(name, func(t *testing.T) {
			p, doc := workflowFixture(t)
			d := doc.Definition
			d.Revision = 2
			change(&d)
			wire, _ := json.Marshal(d)
			r := projectapp.WorkflowRequest{Operation: "create", Project: "sample", Definition: wire, AuthorizeLocal: true}
			out := projectapp.ApplyWorkflow(context.Background(), p, r)
			if out.Category != "unresolved_references" || p.writes != 0 || len(out.Report.Diagnostics) == 0 {
				t.Fatalf("invalid refs: %+v", out)
			}
		})
	}
}
