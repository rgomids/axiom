package projectapp

import (
	"context"
	"strconv"

	"github.com/rgomids/axiom/internal/project"
	"github.com/rgomids/axiom/internal/workflowdefinition"
)

// ActiveWorkflow is the bounded R10 read model. A recorded reference is never
// replaced by a default, and an unresolved definition contributes no name.
type ActiveWorkflow struct {
	Status     string `json:"status"`
	WorkflowID string `json:"workflowId,omitempty"`
	Revision   int    `json:"revision,omitempty"`
	Digest     string `json:"digest,omitempty"`
	Source     string `json:"source,omitempty"`
	Name       string `json:"name,omitempty"`
	Category   string `json:"category,omitempty"`
}

type ActiveWorkflowReader interface {
	ReadActiveWorkflow(context.Context, string) ActiveWorkflow
}

// ResolveActiveWorkflow consumes a strict portable/catalog observation only.
// Categories match the binding boundary; this inspection grants no authority.
func ResolveActiveWorkflow(state project.State, catalog workflowdefinition.Catalog, category string) ActiveWorkflow {
	out := ActiveWorkflow{Status: "none"}
	selection, recorded := state.WorkflowSelection.Value()
	if recorded {
		out = ActiveWorkflow{Status: "unresolvable", WorkflowID: selection.WorkflowID, Revision: selection.Revision, Digest: selection.Digest, Source: selection.Source}
	}
	if category != "" {
		out.Status, out.Category = "unresolvable", category
		return out
	}
	if !recorded {
		return out
	}
	out.Category = "recovery_required"
	ref := workflowdefinition.Ref{WorkflowID: selection.WorkflowID, Revision: selection.Revision, Digest: selection.Digest, Source: selection.Source}
	if !ref.Valid() || len(catalog.Unindexed) != 0 {
		return out
	}
	doc := workflowdefinition.Builtin()
	if ref.Source == "project" {
		found := false
		for _, entry := range catalog.Index.Revisions {
			if entry.Ref() == ref && entry.State == "published" {
				found = true
			}
		}
		if !found {
			return out
		}
		var ok bool
		doc, ok = catalog.Documents[ref.WorkflowID+"/"+strconv.Itoa(ref.Revision)+".json"]
		if !ok {
			return out
		}
	}
	// Recompute the digest from the complete definition as well as checking
	// document identity; a read model never trusts a detached digest field.
	canonical, issues := workflowdefinition.Encode(doc.Definition)
	if len(issues) != 0 || canonical.Digest != doc.Digest || doc.Ref(ref.Source) != ref {
		return out
	}
	out.Status, out.Name, out.Category = "selected", doc.Definition.Name, ""
	return out
}
