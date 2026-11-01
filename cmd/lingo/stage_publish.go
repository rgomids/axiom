package main

import (
	"context"
	"github.com/rgomids/axiom/internal/cli"
	"github.com/rgomids/axiom/internal/coordination"
)

// No environment-provided Graph/Single identity can publish after cutover.
// S1's private ingress is validated by Engine.Ingest against the running
// attempt/fence; the public child transport is part of S2.
func (s lifecycleService) stagePublish(ctx context.Context, in cli.WorkflowStageInput) cli.WorkflowStageResponse {
	return s.stageError(coordination.ErrChildContext, nil)
}
