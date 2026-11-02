package main

import (
	"context"
	"errors"

	"github.com/rgomids/axiom/internal/cli"
)

var (
	errChildContext   = errors.New("running child context required")
	errRecordNotFound = errors.New("execution message not found")
)

// No environment-provided Graph/Single identity can publish after cutover.
// A running Codex or Claude child publishes only into its own attempt's
// message area; the engine binds it to the attempt and fence (S2.2). This CLI
// leaf has no attempt context and therefore always refuses.
func (s lifecycleService) stagePublish(ctx context.Context, in cli.WorkflowStageInput) cli.WorkflowStageResponse {
	return s.stageError(errChildContext, nil)
}
