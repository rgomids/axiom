package main

import (
	"context"
	"github.com/rgomids/axiom/internal/cli"
	"github.com/rgomids/axiom/internal/completion"
)

// The former Profile preview predicted eligibility from capabilities and
// complexity. Canonical runs resolve explicit per-agent inputs instead.
func (s lifecycleService) RuntimeProfilePreview(context.Context, cli.RuntimeProfilePreviewInput) cli.Result {
	result := canonicalCompletion(completion.Facts{ValidationFailed: true}, "Predictive Runtime Profile preview is retired", nil, "Use workflow run --select <agent>.runtime=<id> --select <agent>.model=<model> --select <agent>.effort=runtime-default; an existing Profile is optional", s.provenance)
	result.Category = "runtime_policy_preview_retired"
	return result
}
