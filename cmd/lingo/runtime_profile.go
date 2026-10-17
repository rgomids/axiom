package main

import (
	"context"

	"github.com/rgomids/axiom/internal/cli"
	"github.com/rgomids/axiom/internal/completion"
	"github.com/rgomids/axiom/internal/local"
)

var _ cli.RuntimeProfileService = lifecycleService{}

// RuntimeProfileValidate accepts only an active, valid canonical v3 record
// (runtimeProfileValidityV3). Pre-stable records are refused with explicit
// retirement guidance; nothing is migrated.
func (s lifecycleService) RuntimeProfileValidate(ctx context.Context) cli.Result {
	failure := func(next string) cli.Result {
		return canonicalCompletion(completion.Facts{ValidationFailed: true}, "Runtime profile configuration is missing or invalid", nil, next, s.provenance)
	}
	setup := "Use axiom-runtime to inspect state and prepare explicit local setup or scoped recovery"
	store, err := local.NewRuntimeProfileStore(s.stateRoot)
	if err != nil {
		return failure(setup)
	}
	snapshot, err := store.Inspect(ctx)
	switch code := runtimeProfileValidityV3(snapshot, err); code {
	case "":
	case "unsupported_configuration_format":
		r := failure("Pre-stable local configuration is not used and is never migrated; retire it explicitly with axiom-runtime (binding remove --runtime <id> --cascade), then create canonical configuration")
		r.Category = code
		return r
	default:
		r := failure(setup)
		r.Category = code
		return r
	}
	return canonicalCompletion(completion.Facts{Completed: true}, "Runtime profile configuration is valid", nil, "", s.provenance)
}
