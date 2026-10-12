package main

import (
	"context"

	"github.com/rgomids/axiom/internal/cli"
	"github.com/rgomids/axiom/internal/completion"
	"github.com/rgomids/axiom/internal/local"
	"github.com/rgomids/axiom/internal/runtimeprofile"
)

var _ cli.RuntimeProfileService = lifecycleService{}

func (s lifecycleService) RuntimeProfileValidate(ctx context.Context) cli.Result {
	failure := func() cli.Result {
		return canonicalCompletion(completion.Facts{ValidationFailed: true}, "Runtime profile configuration is missing or invalid", nil, "Use axiom-runtime to inspect state and prepare explicit local setup, migration or scoped recovery", s.provenance)
	}
	store, err := local.NewRuntimeProfileStore(s.stateRoot)
	if err != nil {
		return failure()
	}
	cfg, err := store.Load(ctx)
	if err != nil {
		return failure()
	}
	if err := runtimeprofile.Validate(cfg); err != nil {
		return failure()
	}
	next := ""
	if cfg.FormatVersion == 1 {
		next = "Historical configuration is structurally valid; use axiom-runtime for explicit effort migration before current admission"
	}
	return canonicalCompletion(completion.Facts{Completed: true}, "Runtime profile configuration is valid", nil, next, s.provenance)
}
