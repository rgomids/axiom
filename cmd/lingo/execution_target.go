package main

import (
	"context"
	"github.com/rgomids/axiom/internal/cli"
)

// WorkflowStart cannot create a legacy execution or predictive policy preview.
// New starts use the canonical one-action workflow run operation.
func (s lifecycleService) WorkflowStart(ctx context.Context, input cli.WorkflowInput) cli.Result {
	return s.retiredWorkflowMutation()
}
