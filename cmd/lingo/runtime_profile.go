package main

import (
	"context"

	"github.com/rgomids/axiom/internal/cli"
	"github.com/rgomids/axiom/internal/completion"
	"github.com/rgomids/axiom/internal/local"
	"github.com/rgomids/axiom/internal/runtimeapplication"
)

var _ cli.RuntimeProfileService = lifecycleService{}

// RuntimeProfileValidate accepts only an active, valid canonical format-4
// record (runtimeProfileValidity). Historical records are refused with
// explicit retirement guidance; nothing is migrated.
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
	switch code := runtimeProfileValidity(snapshot, err); code {
	case "":
	case "unsupported_configuration_format":
		r := failure("Historical local configuration is not used and is never migrated; retire it explicitly with axiom-runtime (binding remove --runtime <id> --cascade), then create canonical configuration")
		r.Category = code
		return r
	default:
		r := failure(setup)
		r.Category = code
		return r
	}
	return canonicalCompletion(completion.Facts{Completed: true}, "Runtime profile configuration is valid", nil, "", s.provenance)
}

func runtimeProfileValidity(snapshot local.RuntimeProfileSnapshot, err error) string {
	if err != nil {
		if snapshot.State == "pending" {
			return "configuration_recovery_required"
		}
		return "configuration_unsafe"
	}
	_, code := runtimeapplication.ConsumerFromRecord(snapshot.State, snapshot.V4)
	return code
}
