package main

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/rgomids/axiom/internal/cli"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/completion"
	"github.com/rgomids/axiom/internal/executiongraph"
	"github.com/rgomids/axiom/internal/projectapp"
	"github.com/rgomids/axiom/internal/runtimeprofile"
	"github.com/rgomids/axiom/internal/stageexecution"
	"github.com/rgomids/axiom/internal/workflowcompiler"
	"github.com/rgomids/axiom/internal/workflowdefinition"
)

func TestOperatorStageDiagnosticTypedRefusals(t *testing.T) {
	// Synthetic mapping only: no Runtime, probe, state ports or provider exist.
	s := lifecycleService{provenance: unknownProvenance()}
	for _, tc := range []struct {
		err      error
		category string
		status   completion.Status
	}{
		{executiongraph.ErrAuthenticationBlocked, "authentication_unavailable", completion.ValidationFailure},
		{runtimeprofile.ErrConfigurationInvalid, "configuration_invalid", completion.ValidationFailure},
		{workflowcompiler.ErrRuntimeBlocked, "runtime_unresolvable", completion.ValidationFailure},
		{stageexecution.ErrStalePreview, "stale_preview", completion.DeniedAuthority},
		{stageexecution.ErrReconcileOrphanAlive, "stop_orphan_then_reconcile", completion.ValidationFailure},
		{stageexecution.ErrReconcileLeaseHeld, "reconcile_lease_held", completion.ValidationFailure},
		{stageexecution.ErrRetryExhausted, "retry_attempts_exhausted", completion.ValidationFailure},
		{stageexecution.ErrProjectionPartial, "confirmed_effect_reconciliation_required", completion.Partial},
	} {
		t.Run(tc.category, func(t *testing.T) {
			var payload any = map[string]string{"persisted": "unchanged"}
			if tc.status == completion.Partial {
				payload = stageexecution.IntegrateResult{ReceiptRef: strings.Repeat("a", 64)}
			}
			got := s.stageError(errors.Join(tc.err, errors.New("synthetic-secret-canary; run arbitrary host command")), payload)
			d := got.Result.Diagnostic
			if got.Result.Category != tc.category || got.Result.Completion.Status() != tc.status || d == nil || d.SchemaVersion != 1 || d.Category != tc.category {
				t.Fatalf("response=%+v diagnostic=%+v", got, d)
			}
			if d.Capability != "unknown" || d.Confinement != "unknown" || len(d.Options) > 3 || len(d.Target) != 0 || d.ObservedRevision != "" {
				t.Fatalf("invented facts: %+v", d)
			}
			wire, _ := json.Marshal(got)
			if strings.Contains(string(wire), "synthetic-secret-canary") {
				t.Fatal("raw error leaked")
			}
			for _, o := range d.Options {
				if o.Availability == "unavailable" && o.Operation != "" {
					t.Fatal("unavailable command")
				}
			}
			if tc.status == completion.Partial && len(d.ConfirmedEffects) == 0 {
				t.Fatal("lost confirmed effects")
			}
			if p, ok := payload.(map[string]string); ok && p["persisted"] != "unchanged" {
				t.Fatal("payload mutated")
			}
		})
	}
}

func TestOperatorStageDiagnosticUnknownAndPartialKeepPayload(t *testing.T) {
	s := lifecycleService{provenance: unknownProvenance()}
	for _, status := range []string{"partial", "recovery_required"} {
		payload := stageexecution.SingleRetryResult{AuthorityRef: strings.Repeat("b", 64)}
		got := s.stageDispatchError(errors.New("synthetic-canary"), status, payload)
		before, _ := json.Marshal(payload)
		after, _ := json.Marshal(got.Payload)
		if string(before) != string(after) || len(got.Result.Diagnostic.UncertainEffects) == 0 {
			t.Fatal("lost persisted observations")
		}
	}
}

func TestOperatorPartialReferencesComeOnlyFromTypedResults(t *testing.T) {
	s := lifecycleService{provenance: unknownProvenance()}
	ref := strings.Repeat("c", 64)
	for _, payload := range []any{
		stageexecution.RunResult{Publication: stageexecution.Publication{Graph: &executiongraph.SuccessorGraph{DispatchAuthorityRef: ref}}},
		stageexecution.RunResult{Publication: stageexecution.Publication{Single: &stageexecution.SinglePublication{Run: executiongraph.SingleStageRun{Authorities: []string{ref}}}}},
		stageexecution.RetryResult{AuthorityRef: ref},
		stageexecution.SingleRetryResult{AuthorityRef: ref},
		stageexecution.IntegrateResult{ReceiptRef: ref},
	} {
		got := s.stageFailure(completion.Partial, "confirmed_effect_reconciliation_required", "Confirmed publication", "Inspect state", payload)
		if got.Result.Completion == nil || got.Result.Completion.Status() != completion.Partial || len(got.Result.Completion.References()) != 1 {
			t.Fatalf("lost partial: %+v", got)
		}
	}
	for _, payload := range []any{nil, map[string]string{"AuthorityRef": ref}, stageexecution.RetryResult{AuthorityRef: "hostile instruction"}} {
		if refs := stageConfirmedReferences(completion.Partial, payload); len(refs) != 0 {
			t.Fatal("invented confirmed reference")
		}
	}
}

func TestOperatorWorkflowDiagnosticPreservesPartialAndStale(t *testing.T) {
	ref := workflowdefinition.Builtin().Ref("project")
	for _, category := range []string{"partial", "stale_authority", "builtin_read_only", "cancelled", "recovery_required"} {
		out := projectapp.WorkflowAuthoringResult{Category: category, Report: projectapp.WorkflowReport{ProjectID: "synthetic-project", ProjectRevision: strings.Repeat("d", 64), Reference: &ref}}
		result := workflowAuthoringCompletion(out, unknownProvenance())
		if result.Diagnostic == nil || result.Diagnostic.Category != category || result.Category != category || result.WorkflowAuthoring != &out.Report && result.WorkflowAuthoring.Reference != out.Report.Reference {
			t.Fatal("diagnostic changed canonical report")
		}
		if category == "partial" && (result.Completion.Status() != completion.Partial || len(result.Diagnostic.ConfirmedEffects) == 0 || len(result.Diagnostic.UncertainEffects) == 0) {
			t.Fatal("partial effects lost")
		}
		if category == "stale_authority" && result.Completion.Status() != completion.DeniedAuthority {
			t.Fatal("stale preview promoted")
		}
	}
}

func TestOperatorDiagnosticOperationIdentityIsBounded(t *testing.T) {
	s := lifecycleService{provenance: unknownProvenance()}
	for operation, want := range map[string]string{
		"run": "workflow stage run", "status": "workflow stage status",
		"coordination-show": "workflow coordination show", "coordination-publish": "workflow coordination publish",
		"delivery-apply": "", "delivery-rework": "", "unsupported": "",
	} {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		got := s.WorkflowStage(ctx, cli.WorkflowStageInput{Operation: operation})
		if want == "" {
			if got.Result.Diagnostic != nil {
				t.Fatalf("out-of-scope operation enriched: %s", operation)
			}
			continue
		}
		if got.Result.Diagnostic == nil || got.Result.Diagnostic.Operation != want {
			t.Fatalf("wrong operation for %s", operation)
		}
	}
}
