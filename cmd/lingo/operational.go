package main

import (
	"context"

	"github.com/rgomids/axiom/internal/cli"
	"github.com/rgomids/axiom/internal/completion"
	"github.com/rgomids/axiom/internal/projectapp"
)

// Issue #230 composition of machine-local operational-state requests. The
// caller resolves the exact Project and validates any Integration key against
// portable intent; ApplyOperational owns preview, no-op, authority and CAS.

// applyOperational previews or applies one reviewed local transition and
// renders the canonical result. noun names the subject in user-facing text.
func (s lifecycleService) applyOperational(ctx context.Context, request projectapp.OperationalRequest, reviewedDigest string, authorizeLocal bool, noun string) cli.Result {
	outcome := projectapp.ApplyOperational(ctx, s.operational, request, reviewedDigest, authorizeLocal)
	references := []string{"project:" + request.ProjectID}
	if request.Integration != "" {
		references = append(references, "integration:"+request.Integration)
	}
	var result cli.Result
	switch outcome.Status {
	case projectapp.OperationalPreviewed:
		result = canonicalCompletion(completion.Facts{Completed: true}, noun+" preview ready", references, "Review the local effect, then repeat with --preview-digest <digest> and --authorize-local", s.provenance)
	case projectapp.OperationalUnchanged:
		result = canonicalCompletion(completion.Facts{Completed: true}, noun+" already in the requested state", references, "No local change is required", s.provenance)
	case projectapp.OperationalDenied:
		result = canonicalCompletion(completion.Facts{AuthorityDenied: true}, noun+" authority is missing or stale", references, "Review the current preview and authorize its exact digest with --authorize-local", s.provenance)
	case projectapp.OperationalCommitted:
		if outcome.Category == projectapp.OperationalAppliedRecovery {
			result = canonicalCompletion(completion.Facts{RequestedEffectConfirmed: true, SecondaryFailure: true}, noun+" committed; local publication cleanup requires recovery", references, "Run recovery inspect and apply the reviewed plan", s.provenance)
		} else {
			result = canonicalCompletion(completion.Facts{Completed: true}, noun+" applied on this machine", references, "Portable intent, other machines, credentials and Providers are unchanged", s.provenance)
		}
	default:
		facts, message, next := completion.Facts{Failed: true}, noun+" failed", "Inspect local Project state before retrying"
		switch outcome.Category {
		case projectapp.OperationalInvalidInput:
			facts, message, next = completion.Facts{ValidationFailed: true}, noun+" input is invalid", "Provide an exact Project selector and a declared Integration key"
		case projectapp.OperationalNotInstalled:
			facts, message, next = completion.Facts{ValidationFailed: true}, "Project is not installed on this machine", "Select an installed Project by UUID or slug"
		case projectapp.OperationalStateInvalid:
			message, next = "Local Project operational state is invalid", "Inspect preserved local state; it is never treated as absent or overwritten"
		case projectapp.OperationalRecoveryRequired:
			message, next = "Local Project state requires recovery", "Run recovery inspect and apply the reviewed plan, then retry"
		case projectapp.OperationalConflict:
			facts, message, next = completion.Facts{AuthorityDenied: true}, "Local Project operational state changed concurrently", "Review a fresh preview and authorize its exact digest"
		case projectapp.OperationalCancelled:
			facts, message, next = completion.Facts{WasInterrupted: true}, noun+" was cancelled", "Retry the request"
		}
		result = canonicalCompletion(facts, message, references, next, s.provenance)
	}
	result.Category = outcome.Category
	result.Operational = outcome.Preview
	return result
}
