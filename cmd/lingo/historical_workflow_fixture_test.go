package main

import (
	"context"
	"testing"

	"github.com/rgomids/axiom/internal/cli"
	"github.com/rgomids/axiom/internal/local"
	"github.com/rgomids/axiom/internal/workflow"
)

// Seed an already-existing historical record through its canonical protected
// store. This is fixture preparation, never public authorization for a new start.
func seedHistoricalWorkflow(t *testing.T, service lifecycleService, input cli.WorkflowInput) *cli.WorkflowView {
	t.Helper()
	returnService := historicalWorkflowService(t, service)
	target := workflowTarget(input)
	target.RuntimePreview = input.RuntimePreview
	result := returnService.Start(context.Background(), target)
	if result.Status != workflow.Succeeded {
		t.Fatalf("historical fixture seed %s", result.Category)
	}
	return workflowResult(result, service.provenance).Workflow
}

func historicalWorkflowService(t *testing.T, service lifecycleService) workflow.Service {
	t.Helper()
	store, err := local.NewWorkflowStore(service.stateRoot)
	if err != nil {
		t.Fatal(err)
	}
	artifacts, err := local.NewArtifactStore(service.stateRoot)
	if err != nil {
		t.Fatal(err)
	}
	refs := local.NewWorkflowReferenceValidator(artifacts)
	historical := workflow.New(workflowResolver{service.installation}, workflowWorkItems{service.workItems}, store, nil, refs, service.provenance, nil, nil).WithDefinitions(workflowDefinitions{service.installation, service.portable}, refs)
	return historical
}
