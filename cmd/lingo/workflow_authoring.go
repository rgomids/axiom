package main

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strconv"

	"github.com/rgomids/axiom/internal/cli"
	"github.com/rgomids/axiom/internal/completion"
	"github.com/rgomids/axiom/internal/local"
	"github.com/rgomids/axiom/internal/manifest"
	"github.com/rgomids/axiom/internal/project"
	"github.com/rgomids/axiom/internal/projectapp"
	"github.com/rgomids/axiom/internal/provenance"
	"github.com/rgomids/axiom/internal/workflowdefinition"
)

func (s lifecycleService) ProjectWorkflow(ctx context.Context, input cli.ProjectWorkflowInput) cli.Result {
	if denied := s.gate(ctx, input.Project, projectapp.AdmitProjectEdit); denied != nil {
		return *denied
	}
	request := input.WorkflowRequest
	if input.File != "" {
		f, e := os.Open(input.File)
		if e != nil {
			return canonicalCompletion(completion.Facts{ValidationFailed: true}, "Workflow input is unreadable", nil, "Provide a complete UTF-8 JSON definition", s.provenance)
		}
		defer f.Close()
		b, e := io.ReadAll(io.LimitReader(f, workflowdefinition.MaxBytes+1))
		if e != nil || len(b) > workflowdefinition.MaxBytes {
			return canonicalCompletion(completion.Facts{ValidationFailed: true}, "Workflow input exceeds bounds", nil, "Provide a definition no larger than 1 MiB", s.provenance)
		}
		request.Definition = b
	}
	outcome := projectapp.ApplyWorkflow(ctx, workflowAuthoringAdapter{s}, request)
	return workflowAuthoringCompletion(outcome, s.provenance)
}

func workflowAuthoringCompletion(outcome projectapp.WorkflowAuthoringResult, source provenance.Value) cli.Result {
	var references []string
	if ref := outcome.Report.Reference; ref != nil && ref.Valid() {
		references = []string{"workflow:" + ref.WorkflowID + "/" + strconv.Itoa(ref.Revision) + ":" + ref.Digest}
	}
	facts := completion.Facts{ValidationFailed: true}
	switch outcome.Category {
	case "previewed", "listed", "inspected", "validated", "applied", "unchanged":
		facts = completion.Facts{Completed: true}
	case "stale_authority":
		facts = completion.Facts{AuthorityDenied: true}
	case "partial":
		facts = completion.Facts{RequestedEffectConfirmed: true, SecondaryFailure: true}
	case "recovery_required":
		facts = completion.Facts{Failed: true}
	case "cancelled":
		facts = completion.Facts{WasInterrupted: true}
	}
	next := "Inspect Project workflows; review the exact revision and digest before applying"
	if outcome.Category == "recovery_required" || outcome.Category == "partial" {
		next = "Run recovery inspect/apply for preserved file protocols; then project workflow recover with the exact orphan definition, review its preview and authorize it"
	}
	result := canonicalCompletion(facts, "Project workflow "+outcome.Category, references, next, source)
	result.Category = outcome.Category
	result.WorkflowAuthoring = &outcome.Report
	return result
}

type workflowAuthoringAdapter struct{ s lifecycleService }

func (a workflowAuthoringAdapter) ObserveWorkflows(ctx context.Context, selector string) (projectapp.WorkflowObservation, string) {
	s := a.s
	selected := s.installation.Select(ctx, selector)
	if selected.Status != local.ResolutionFound {
		return projectapp.WorkflowObservation{}, selected.Category
	}
	record, category := s.installation.Inspect(ctx, selected.Project.ID)
	if category != "" {
		return projectapp.WorkflowObservation{}, category
	}
	state := record.Record.State()
	if state.SourceLocation != filepath.Join(s.projectsRoot, state.ObservedSlug) {
		return projectapp.WorkflowObservation{}, "unsupported_portable_source"
	}
	snapshot, catalog, e := s.portable.InspectWorkflows(ctx, state.ObservedSlug)
	if e != nil {
		return projectapp.WorkflowObservation{}, workflowStorageCategory(e)
	}
	digest, _ := snapshot.Revision().Digest()
	observation := projectapp.WorkflowObservation{Selection: projectapp.EditSelection{Portable: snapshot, Local: local.ApplicationRecord(record.Record), LocalWire: record.Wire, PortableDestination: state.SourceLocation, LocalDestination: filepath.Join(s.stateRoot, "projects", state.ProjectID), PortableRevision: fmtDigest(digest), LocalRevision: record.Revision}, Catalog: catalog}
	inventory, e := local.InspectStateInventory(ctx, s.stateRoot)
	observation.ReferencesChecked = e == nil
	for _, entry := range inventory.Entries {
		if !entry.Kind.Supported() {
			observation.ReferencesChecked = false
		}
		// Legacy v1 executions have no configurable binding. Future formats or
		// unknown recovery/reference records deny retirement rather than guessing.
	}
	if observation.ReferencesChecked {
		store, err := local.NewWorkflowStore(s.stateRoot)
		if err != nil {
			observation.ReferencesChecked = false
		} else {
			states, err := store.List(ctx, state.ProjectID)
			if err != nil {
				observation.ReferencesChecked = false
			} else {
				for _, execution := range states {
					if execution.Binding != nil {
						observation.Referenced = append(observation.Referenced, execution.Binding.Definition)
					}
				}
			}
		}
	}
	return observation, ""
}
func fmtDigest(d [32]byte) string {
	const hex = "0123456789abcdef"
	b := make([]byte, 64)
	for i, v := range d {
		b[i*2] = hex[v>>4]
		b[i*2+1] = hex[v&15]
	}
	return string(b)
}
func workflowStorageCategory(e error) string {
	switch {
	case errors.Is(e, context.Canceled), errors.Is(e, context.DeadlineExceeded):
		return "cancelled"
	case errors.Is(e, local.ErrConflict):
		return "revision_conflict"
	case errors.Is(e, local.ErrRecoveryRequired), errors.Is(e, local.ErrSimulatedInterruption):
		return "recovery_required"
	default:
		return "unsafe_workflow_storage"
	}
}
func (a workflowAuthoringAdapter) PublishWorkflowRevision(ctx context.Context, o projectapp.WorkflowObservation, d *workflowdefinition.Document, index workflowdefinition.Index) string {
	wire, e := local.EncodeWorkflowIndex(index)
	if e != nil {
		return "invalid_revision_index"
	}
	e = a.s.installation.PublishWorkflow(ctx, a.s.portable, o.Selection.Local.ProjectID, o.Selection.Local.ObservedSlug, o.Selection.LocalWire, o.Selection.Portable.Manifest(), o.Catalog.Wire, d, wire)
	if e != nil {
		var publication *local.PublicationError
		if errors.As(e, &publication) && publication.EffectCommitted() {
			return "partial"
		}
		return workflowStorageCategory(e)
	}
	return "applied"
}
func (a workflowAuthoringAdapter) PublishWorkflowSelection(ctx context.Context, o projectapp.WorkflowObservation, next project.Project) string {
	wire, issues := manifest.Encode(next)
	if len(issues) > 0 {
		return "invalid_selection"
	}
	snapshot, problems := projectapp.ReadSnapshot(manifest.Codec{}, wire, nil)
	if len(problems) > 0 {
		return "invalid_selection"
	}
	state := o.Selection.Local
	state.PortableRevision = snapshot.Revision()
	state.ArtifactDigests = snapshot.Digests()
	localWire, problems := (local.RecordCodec{}).EncodeLocal(state)
	if len(problems) > 0 {
		return "invalid_selection"
	}
	result := (local.ProjectEditPublisher{Installation: a.s.installation, Portable: a.s.portable.WithWorkflowIndex(o.Catalog.Wire)}).PublishEdit(ctx, projectapp.EditPublicationRequest{ProjectID: state.ProjectID, Slug: state.ObservedSlug, PortableDestination: o.Selection.PortableDestination, LocalDestination: o.Selection.LocalDestination, PortableExpected: o.Selection.Portable.Manifest(), PortableNext: wire, LocalExpected: o.Selection.LocalWire, LocalNext: localWire})
	return string(result.Status)
}
