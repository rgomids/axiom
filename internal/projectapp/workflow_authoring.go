package projectapp

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/rgomids/axiom/internal/project"
	"github.com/rgomids/axiom/internal/workflowdefinition"
)

// Workflow authoring is a Project application operation. Adapters own I/O;
// this layer owns revision assignment, reference validation and exact authority.
type WorkflowRequest struct {
	Operation        string                 `json:"operation"`
	Project          string                 `json:"project"`
	Definition       []byte                 `json:"-"`
	FromDefault      bool                   `json:"fromDefault"`
	WorkflowID       string                 `json:"workflowId"`
	Ref              workflowdefinition.Ref `json:"ref"`
	Prior            workflowdefinition.Ref `json:"prior"`
	ExpectedRevision string                 `json:"expectedRevision"`
	PreviewDigest    string                 `json:"previewDigest"`
	AuthorizeLocal   bool                   `json:"authorizeLocal"`
}
type WorkflowObservation struct {
	Selection EditSelection
	Catalog   workflowdefinition.Catalog
	// ReferencesChecked is false whenever retention inventory is uncertain.
	ReferencesChecked bool
	Referenced        []workflowdefinition.Ref
}
type WorkflowAuthoringPort interface {
	ObserveWorkflows(context.Context, string) (WorkflowObservation, string)
	PublishWorkflowRevision(context.Context, WorkflowObservation, *workflowdefinition.Document, workflowdefinition.Index) string
	PublishWorkflowSelection(context.Context, WorkflowObservation, project.Project) string
}
type WorkflowReport struct {
	ProjectID       string                          `json:"projectId,omitempty"`
	ProjectRevision string                          `json:"projectRevision,omitempty"`
	Selection       *workflowdefinition.Ref         `json:"selection,omitempty"`
	Revisions       []workflowdefinition.Ref        `json:"revisions,omitempty"`
	Retired         []workflowdefinition.Ref        `json:"retired,omitempty"`
	Definition      *workflowdefinition.Definition  `json:"definition,omitempty"`
	Reference       *workflowdefinition.Ref         `json:"reference,omitempty"`
	Diagnostics     []workflowdefinition.Diagnostic `json:"diagnostics,omitempty"`
	Prerequisites   []string                        `json:"prerequisites,omitempty"`
	Effects         []string                        `json:"effects,omitempty"`
	PreviewDigest   string                          `json:"previewDigest,omitempty"`
}
type WorkflowAuthoringResult struct {
	Category string
	Report   WorkflowReport
}

func ApplyWorkflow(ctx context.Context, port WorkflowAuthoringPort, r WorkflowRequest) WorkflowAuthoringResult {
	out := WorkflowAuthoringResult{Category: "invalid_input"}
	fail := func(category string) WorkflowAuthoringResult { out.Category = category; return out }
	read := r.Operation == "list" || r.Operation == "show" || r.Operation == "validate"
	if r.Project == "" || (read && (r.AuthorizeLocal || r.PreviewDigest != "" || r.ExpectedRevision != "")) {
		return out
	}
	observed, category := port.ObserveWorkflows(ctx, r.Project)
	if category != "" {
		return fail(category)
	}
	p := observed.Selection.Portable.Project()
	state := p.State()
	catalog := observed.Catalog
	out.Report.ProjectID = state.ID
	out.Report.ProjectRevision = observed.Selection.PortableRevision
	if selection, ok := state.WorkflowSelection.Value(); ok {
		ref := workflowdefinition.Ref{WorkflowID: selection.WorkflowID, Revision: selection.Revision, Digest: selection.Digest, Source: selection.Source}
		out.Report.Selection = &ref
	}
	builtin := workflowdefinition.Builtin()
	for _, entry := range catalog.Index.Revisions {
		if entry.State == "published" {
			out.Report.Revisions = append(out.Report.Revisions, entry.Ref())
		} else {
			out.Report.Retired = append(out.Report.Retired, entry.Ref())
		}
	}
	out.Report.Revisions = append([]workflowdefinition.Ref{builtin.Ref("builtin")}, out.Report.Revisions...)
	if len(catalog.Unindexed) > 0 && r.Operation != "recover" {
		out.Report.Revisions = catalog.Unindexed
		return fail("recovery_required")
	}
	lookup := func(ref workflowdefinition.Ref) (workflowdefinition.Document, bool) {
		if !ref.Valid() {
			return workflowdefinition.Document{}, false
		}
		if ref.Source == "builtin" {
			return builtin, ref == builtin.Ref("builtin")
		}
		for _, e := range catalog.Index.Revisions {
			if e.Ref() == ref && (e.State == "published" || r.Operation == "show") {
				d, ok := catalog.Documents[fmt.Sprintf("%s/%d.json", ref.WorkflowID, ref.Revision)]
				return d, ok && d.Digest == ref.Digest
			}
		}
		return workflowdefinition.Document{}, false
	}
	if r.Operation == "list" {
		return fail("listed")
	}
	var doc workflowdefinition.Document
	var newIndex workflowdefinition.Index
	effects := []string{}
	switch r.Operation {
	case "show", "select", "remove":
		var ok bool
		doc, ok = lookup(r.Ref)
		if !ok {
			return fail("workflow_not_found")
		}
	case "create", "edit", "validate", "recover":
		var issues []workflowdefinition.Diagnostic
		if r.FromDefault {
			if !workflowdefinition.ValidKey(r.WorkflowID) || len(r.Definition) != 0 || r.Operation != "create" {
				return out
			}
			d := builtin.Definition
			d.WorkflowID = r.WorkflowID
			d.Name = r.WorkflowID
			d.Revision = 1
			for _, e := range catalog.Index.Revisions {
				if e.WorkflowID == d.WorkflowID && e.Revision >= d.Revision {
					d.Revision = e.Revision + 1
				}
			}
			doc, issues = workflowdefinition.Encode(d)
		} else {
			doc, issues = workflowdefinition.Decode(r.Definition)
		}
		if len(issues) > 0 {
			out.Report.Diagnostics = issues
			return fail("invalid_definition")
		}
	default:
		return out
	}
	out.Report.Definition = &doc.Definition
	source := "project"
	if r.Ref.Source == "builtin" {
		source = "builtin"
	}
	ref := doc.Ref(source)
	out.Report.Reference = &ref
	out.Report.Diagnostics = workflowProjectDiagnostics(state, doc.Definition)
	out.Report.Prerequisites = []string{"runtime_profile_and_effort_readiness_checked_before_execution", "validator_availability_checked_before_execution", "artifact_context_digests_resolved_before_execution"}
	if len(out.Report.Diagnostics) > 0 && r.Operation != "show" && r.Operation != "remove" {
		return fail("unresolved_references")
	}
	if r.Operation == "show" {
		return fail("inspected")
	}
	if r.Operation == "validate" {
		return fail("validated")
	}
	newIndex = catalog.Index
	newIndex.Revisions = append([]workflowdefinition.Entry(nil), catalog.Index.Revisions...)
	switch r.Operation {
	case "create", "edit", "recover":
		maxRevision := 0
		exists := false
		for _, e := range catalog.Index.Revisions {
			if e.WorkflowID != doc.Definition.WorkflowID {
				continue
			}
			if e.Revision > maxRevision {
				maxRevision = e.Revision
			}
			if e.Revision == doc.Definition.Revision {
				if e.Digest != doc.Digest || e.State != "published" {
					return fail("revision_conflict")
				}
				exists = true
			}
		}
		if r.Operation == "edit" {
			if _, ok := lookup(r.Prior); !ok || r.Prior.Source != "project" || r.Prior.WorkflowID != ref.WorkflowID {
				return fail("prior_revision_required")
			}
		}
		if exists {
			return fail("unchanged")
		}
		if ref.Revision <= maxRevision {
			return fail("revision_conflict")
		}
		if r.Operation == "recover" {
			if len(catalog.Unindexed) != 1 || catalog.Unindexed[0] != ref {
				return fail("recovery_required")
			}
		}
		if len(newIndex.Revisions) >= 1024 {
			return fail("revision_capacity")
		}
		newIndex.Revisions = append(newIndex.Revisions, workflowdefinition.Entry{WorkflowID: ref.WorkflowID, Revision: ref.Revision, Digest: ref.Digest, State: "published"})
		effects = []string{"publish_immutable_definition", "append_revision_index"}
	case "select":
		if out.Report.Selection != nil && *out.Report.Selection == ref {
			return fail("unchanged")
		}
		effects = []string{"publish_project_schema4_selection", "refresh_local_portable_observation"}
	case "remove":
		if ref.Source == "builtin" {
			return fail("builtin_read_only")
		}
		if out.Report.Selection != nil && *out.Report.Selection == ref {
			return fail("revision_in_use")
		}
		if !observed.ReferencesChecked {
			return fail("reference_inventory_unknown")
		}
		for _, used := range observed.Referenced {
			if used == ref {
				return fail("revision_in_use")
			}
		}
		for i, e := range newIndex.Revisions {
			if e.Ref() == ref {
				newIndex.Revisions[i].State = "retired"
			}
		}
		// Retain content conservatively: retirement makes it unselectable without
		// discarding the inspectable immutable historical definition.
		effects = []string{"retire_revision_index_preserve_content"}
	}
	out.Report.Effects = effects
	envelope := struct {
		Operation, ProjectID, Portable, Local string
		Selection                             *workflowdefinition.Ref
		Reference                             workflowdefinition.Ref
		Prior                                 workflowdefinition.Ref
		Index                                 workflowdefinition.Index
		Effects                               []string
	}{r.Operation, state.ID, observed.Selection.PortableRevision, observed.Selection.LocalRevision, out.Report.Selection, ref, r.Prior, catalog.Index, effects}
	b, _ := json.Marshal(envelope)
	digest := sha256.Sum256(b)
	out.Report.PreviewDigest = hex.EncodeToString(digest[:])
	if !r.AuthorizeLocal && r.PreviewDigest == "" && r.ExpectedRevision == "" {
		return fail("previewed")
	}
	if !r.AuthorizeLocal || r.PreviewDigest != out.Report.PreviewDigest || r.ExpectedRevision != observed.Selection.PortableRevision {
		return fail("stale_authority")
	}
	if ctx.Err() != nil {
		return fail("cancelled")
	}
	if r.Operation == "select" {
		next, issues := p.SelectWorkflow(project.WorkflowSelection{WorkflowID: ref.WorkflowID, Revision: ref.Revision, Digest: ref.Digest, Source: ref.Source})
		if len(issues) > 0 {
			return fail("invalid_selection")
		}
		return fail(port.PublishWorkflowSelection(ctx, observed, next))
	}
	if r.Operation == "remove" {
		return fail(port.PublishWorkflowRevision(ctx, observed, nil, newIndex))
	}
	return fail(port.PublishWorkflowRevision(ctx, observed, &doc, newIndex))
}

func workflowProjectDiagnostics(p project.State, d workflowdefinition.Definition) []workflowdefinition.Diagnostic {
	var issues []workflowdefinition.Diagnostic
	profiles, _ := p.ModelProfiles.Value()
	docs, _ := p.DocumentationSources.Value()
	policies, _ := p.Policies.Value()
	for i, s := range d.Stages {
		field := fmt.Sprintf("stages[%d]", i)
		for _, input := range s.Inputs {
			if input.Kind == "project-context" {
				found := input.Source == "business-context"
				for _, doc := range docs {
					found = found || doc.Key == input.Source
				}
				if !found {
					issues = append(issues, workflowdefinition.Diagnostic{Field: field + ".inputs", Code: "unknown_project_context"})
				}
			}
		}
		for _, validator := range s.Validators {
			found := validator.PolicyRef == "builtin-sdd-v1"
			for _, policy := range policies {
				found = found || policy == validator.PolicyRef
			}
			if !found {
				issues = append(issues, workflowdefinition.Diagnostic{Field: field + ".validators", Code: "unknown_validator_policy"})
			}
		}
		for _, agent := range s.Agents {
			if agent.ProfileRef != "policy-default" {
				found := false
				for _, profile := range profiles {
					found = found || profile.Key == agent.ProfileRef
				}
				if !found {
					issues = append(issues, workflowdefinition.Diagnostic{Field: field + ".agents", Code: "unknown_profile"})
				}
			}
			for _, capability := range agent.Capabilities {
				switch capability {
				case "read", "repository-write", "process", "artifact-publish", "integration":
				default:
					issues = append(issues, workflowdefinition.Diagnostic{Field: field + ".agents", Code: "unsupported_capability"})
				}
			}
		}
	}
	return issues
}
