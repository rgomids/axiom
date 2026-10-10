package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strconv"

	"github.com/rgomids/axiom/internal/local"
	"github.com/rgomids/axiom/internal/project"
	"github.com/rgomids/axiom/internal/projectapp"
	"github.com/rgomids/axiom/internal/workflow"
	"github.com/rgomids/axiom/internal/workflowdefinition"
)

type workflowDefinitions struct {
	installation local.InstallationStore
	portable     local.PortableStore
}

func readStageResult(path string, value *workflow.StageResult) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	b, err := io.ReadAll(io.LimitReader(f, 64<<10+1))
	if err != nil || len(b) > 64<<10 {
		return workflowdefinition.ErrInvalid
	}
	return workflowdefinition.StrictJSON(b, value)
}

func (r workflowDefinitions) ResolveDefinition(ctx context.Context, selector string) (workflow.DefinitionObservation, string) {
	resolved := r.installation.Resolve(ctx, selector)
	if resolved.Status != local.ResolutionFound {
		return workflow.DefinitionObservation{}, resolved.Category
	}
	// Use a store rooted at the recorded source's parent, including imported
	// Projects. InspectWorkflows supplies locked content/index observations.
	store, err := local.NewPortableStore(filepath.Dir(resolved.Project.Source))
	if err != nil || filepath.Base(resolved.Project.Source) != resolved.Project.Slug {
		return workflow.DefinitionObservation{}, "unsupported_portable_source"
	}
	snapshot, catalog, err := store.InspectWorkflows(ctx, resolved.Project.Slug)
	if err != nil || len(catalog.Unindexed) != 0 || snapshot.Project().State().ID != resolved.Project.ID {
		return workflow.DefinitionObservation{}, "recovery_required"
	}
	if snapshot.Revision() != resolved.Project.PortableRevision {
		return workflow.DefinitionObservation{}, "project_configuration_drift"
	}
	selected, ok := snapshot.Project().State().WorkflowSelection.Value()
	if !ok {
		return workflow.DefinitionObservation{}, "workflow_selection_required"
	}
	ref := workflowdefinition.Ref{WorkflowID: selected.WorkflowID, Revision: selected.Revision, Digest: selected.Digest, Source: selected.Source}
	var doc workflowdefinition.Document
	if ref.Source == "builtin" {
		doc = workflowdefinition.Builtin()
	} else {
		published := false
		for _, entry := range catalog.Index.Revisions {
			if entry.Ref() == ref && entry.State == "published" {
				published = true
			}
		}
		if !published {
			return workflow.DefinitionObservation{}, "recovery_required"
		}
		doc = catalog.Documents[ref.WorkflowID+"/"+strconv.Itoa(ref.Revision)+".json"]
	}
	if doc.Ref(ref.Source) != ref {
		return workflow.DefinitionObservation{}, "recovery_required"
	}
	contexts := map[string]workflow.ContextObservation{}
	for _, stage := range doc.Definition.Stages {
		for _, input := range stage.Inputs {
			if input.Kind != "project-context" {
				continue
			}
			if _, ok := contexts[input.Source]; ok {
				continue
			}
			content, err := readWorkflowContext(ctx, resolved.Project, snapshot.Project().State(), input.Source)
			if err != nil {
				if input.Required {
					return workflow.DefinitionObservation{}, "stage_prerequisite_missing"
				}
				continue
			}
			hash := sha256.Sum256(content)
			contexts[input.Source] = workflow.ContextObservation{Source: input.Source, Digest: hex.EncodeToString(hash[:]), Content: content}
		}
	}
	portableRevision, _ := snapshot.Revision().Digest()
	localWire, _ := json.Marshal(resolved.Project.Local)
	localRevision := sha256.Sum256(localWire)
	wire, _ := json.Marshal(struct {
		Project    local.ResolvedProject
		Revision   [32]byte
		Definition workflowdefinition.Ref
		Contexts   map[string]workflow.ContextObservation
	}{resolved.Project, portableRevision, ref, contexts})
	hash := sha256.Sum256(wire)
	return workflow.DefinitionObservation{ProjectID: resolved.Project.ID, Document: doc, Source: ref.Source, Digest: hex.EncodeToString(hash[:]), Contexts: contexts, ProjectRevision: hex.EncodeToString(portableRevision[:]), LocalRevision: hex.EncodeToString(localRevision[:])}, ""
}

func readWorkflowContext(ctx context.Context, resolved local.ResolvedProject, state project.State, key string) ([]byte, error) {
	if key == "business-context" {
		business, ok := state.BusinessContext.Value()
		if !ok {
			return nil, workflowdefinition.ErrInvalid
		}
		text, _ := business.Text.Value()
		documents, _ := business.Documents.Value()
		sources, _ := business.SourceRefs.Value()
		glossary, _ := business.Glossary.Value()
		return json.Marshal(struct {
			Text                  string
			Documents, SourceRefs []string
			Glossary              []project.GlossaryEntry
		}{text, documents, sources, glossary})
	}
	sources, _ := state.DocumentationSources.Value()
	for _, source := range sources {
		if source.Key == key {
			request := projectapp.DocumentationRequest{Source: source}
			var rootPath, relative string
			if source.Kind == project.RepositorySource {
				repository, _ := source.RepositoryRef.Value()
				relative, _ = source.Path.Value()
				for _, r := range resolved.Repositories {
					if r.Key == repository {
						request.RepositoryPath = r.Path
						rootPath = r.Path
					}
				}
			} else {
				for _, b := range resolved.Local.Documentation {
					if b.SourceKey == key {
						binding := b
						request.Binding = &binding
						rootPath = filepath.Dir(b.ExplicitPath)
						relative = filepath.Base(b.ExplicitPath)
					}
				}
			}
			if (local.DocumentationResolver{}).ResolveDocumentation(ctx, request) != projectapp.DocumentationAvailable {
				return nil, workflowdefinition.ErrInvalid
			}
			root, err := os.OpenRoot(rootPath)
			if err != nil {
				return nil, err
			}
			defer root.Close()
			f, err := root.Open(relative)
			if err != nil {
				return nil, err
			}
			defer f.Close()
			b, err := io.ReadAll(io.LimitReader(f, workflowdefinition.MaxBytes+1))
			if err != nil || len(b) > workflowdefinition.MaxBytes {
				return nil, workflowdefinition.ErrInvalid
			}
			if (local.DocumentationResolver{}).ResolveDocumentation(ctx, request) != projectapp.DocumentationAvailable {
				return nil, workflowdefinition.ErrInvalid
			}
			return b, nil
		}
	}
	return nil, workflowdefinition.ErrInvalid
}
