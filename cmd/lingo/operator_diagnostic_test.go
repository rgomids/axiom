package main

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/rgomids/axiom/internal/cli"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/completion"
	"github.com/rgomids/axiom/internal/coordination"
	"github.com/rgomids/axiom/internal/local"
	"github.com/rgomids/axiom/internal/projectapp"
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

		{coordination.ErrChildContext, "child_context_required", completion.DeniedAuthority},
		{local.ErrConflict, "temporarily_unavailable", completion.RetryableFailure},
		{local.ErrRecoveryRequired, "recovery_required", completion.Failure},
	} {
		t.Run(tc.category, func(t *testing.T) {
			var payload any = map[string]string{"persisted": "unchanged"}
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

func TestOperatorPartialReferencesAreNeverSynthesized(t *testing.T) {
	ref := strings.Repeat("c", 64)
	for _, payload := range []any{nil, map[string]string{"AuthorityRef": ref}, map[string]string{"ReceiptRef": "hostile instruction"}} {
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
		"execution-run": "workflow run", "status": "workflow stage status",
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
