package compatibility

import (
	"path/filepath"
	"reflect"
	"testing"

	"github.com/rgomids/axiom/internal/codexruntime"
)

func policyReport(classification Classification, reason string) Report {
	report := Report{Classification: classification, Reason: reason, Digest: "digest", SkillSet: codexruntime.Inventory{State: codexruntime.SkillSetAbsent}}
	if classification == RecognizedPOC {
		report.POCTag, report.POCRevision = HistoricalPOCTag, HistoricalPOCRevision
	}
	return report
}

func TestTransitionPolicyMatrix(t *testing.T) {
	poc := func(mutate func(*Report)) Report {
		report := policyReport(RecognizedPOC, "complete_poc_workflow_signature")
		mutate(&report)
		return report
	}
	tests := []struct {
		name      string
		ownership Ownership
		report    Report
		strategy  Strategy
		outcome   Outcome
		reason    string
	}{
		{"owned absent state", Owned, policyReport(AbsentV1, "no_owned_state"), StrategyDirect, OutcomeCompatible, "no_persisted_state"},
		{"owned current v1", Owned, policyReport(ValidV1, "v1_readable_state"), StrategyDirect, OutcomeCompatible, "stable_format_v1"},
		{"owned recognized POC", Owned, policyReport(RecognizedPOC, "complete_poc_workflow_signature"), StrategyPreserveRebuildReconfigure, OutcomeAutomaticTransition, "historical_poc_transition"},
		{"owned recognized POC with upgradable skills", Owned, poc(func(r *Report) { r.SkillSet.State = codexruntime.SkillSetUpgradable }), StrategyPreserveRebuildReconfigure, OutcomeAutomaticTransition, "historical_poc_transition"},
		{"POC wrong revision", Owned, poc(func(r *Report) { r.POCRevision = "0000000000000000000000000000000000000000" }), StrategyRefuse, OutcomeUnsafe, "historical_poc_preconditions_unmet"},
		{"POC wrong tag", Owned, poc(func(r *Report) { r.POCTag = "v0.1.0-poc.2" }), StrategyRefuse, OutcomeUnsafe, "historical_poc_preconditions_unmet"},
		{"POC without complete signature", Owned, poc(func(r *Report) { r.Reason = "partial" }), StrategyRefuse, OutcomeUnsafe, "historical_poc_preconditions_unmet"},
		{"POC foreign skill set", Owned, poc(func(r *Report) { r.SkillSet.State = codexruntime.SkillSetForeign }), StrategyRefuse, OutcomeUnsafe, "historical_poc_preconditions_unmet"},
		{"POC interrupted skill set", Owned, poc(func(r *Report) { r.SkillSet.State = codexruntime.SkillSetInterrupted }), StrategyRefuse, OutcomeUnsafe, "historical_poc_preconditions_unmet"},
		{"POC unverifiable inspection", Owned, poc(func(r *Report) { r.Digest = "" }), StrategyRefuse, OutcomeUnsafe, "unverifiable_inspection"},
		{"foreign installation with current v1", Foreign, policyReport(ValidV1, "v1_readable_state"), StrategyRefuse, OutcomeUnsafe, "installation_not_owned"},
		{"ambiguous installation with absent state", Ambiguous, policyReport(AbsentV1, "no_owned_state"), StrategyRefuse, OutcomeUnsafe, "installation_not_owned"},
		{"foreign installation with recognized POC", Foreign, policyReport(RecognizedPOC, "complete_poc_workflow_signature"), StrategyRefuse, OutcomeUnsafe, "installation_not_owned"},
		{"unset ownership", "", policyReport(ValidV1, "v1_readable_state"), StrategyRefuse, OutcomeUnsafe, "installation_not_owned"},
		{"modified or corrupt state", Owned, policyReport(Malformed, "unrecognized_or_unsafe_content"), StrategyRefuse, OutcomeUnsafe, "unrecognized_ambiguous_or_unsafe_state"},
		{"v1 with preserved POC history", Owned, policyReport(ValidV1, "v1_state_with_preserved_poc_history"), StrategyDirect, OutcomeCompatible, "stable_format_v1"},
		{"bounded inspection", Owned, policyReport(Malformed, "entry_bound_exceeded"), StrategyRefuse, OutcomeUnsafe, "unrecognized_ambiguous_or_unsafe_state"},
		{"unsupported newer", Owned, policyReport(UnsupportedNewer, "newer_format_version"), StrategyRefuse, OutcomeUnsupported, "newer_state_format"},
		{"out of window older", Owned, policyReport(UnsupportedOlder, "older_format_version"), StrategyRefuse, OutcomeUnsupported, "state_format_outside_window"},
		{"interrupted protocol state", Owned, policyReport(RecoveryRequired, "interrupted_protocol_state"), StrategyRefuse, OutcomeRecoveryRequired, "interrupted_operation"},
		{"future unclassified format", Owned, policyReport("valid_v2", "v2_readable_state"), StrategyRefuse, OutcomeUnsupported, "unclassified_state"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			decision := Resolve(test.ownership, test.report)
			if decision.Strategy != test.strategy || decision.Outcome != test.outcome || decision.Reason != test.reason {
				t.Fatalf("decision=%+v want %s/%s/%s", decision, test.strategy, test.outcome, test.reason)
			}
			if decision.Classification != test.report.Classification || decision.StateDigest != test.report.Digest {
				t.Fatalf("decision is not bound to the inspected report: %+v", decision)
			}
			if again := Resolve(test.ownership, test.report); again != decision {
				t.Fatalf("resolution is not deterministic: %+v then %+v", decision, again)
			}
		})
	}
}

// The window is a publication-time declaration: widening it, or adding a
// stable format, must change this test deliberately.
func TestStableWindowIsExactlyV1(t *testing.T) {
	window := StableWindow()
	if !reflect.DeepEqual(window, map[Format]Strategy{FormatV1: StrategyDirect}) {
		t.Fatalf("stable window=%v want exactly {v1: direct}", window)
	}
	for format, strategy := range window {
		switch strategy {
		case StrategyDirect, StrategyMigrate, StrategyPreserveRebuildReconfigure:
		default:
			t.Fatalf("format %s declares non-forward strategy %q", format, strategy)
		}
	}
	window[FormatV1] = StrategyRefuse
	if StableWindow()[FormatV1] != StrategyDirect {
		t.Fatal("callers can mutate the declared window")
	}
	for _, classification := range []Classification{AbsentV1, RecognizedPOC, Malformed, UnsupportedOlder, UnsupportedNewer, RecoveryRequired, "valid_v2"} {
		if _, ok := stableFormat(classification); ok {
			t.Fatalf("%s entered the stable window", classification)
		}
	}
}

func TestTransitionPolicyResolvesInspectedFixtures(t *testing.T) {
	base := t.TempDir()
	absent := Roots{Projects: filepath.Join(base, "p"), State: filepath.Join(base, "s")}
	v1 := pocRoots(t)
	removeAll(t, filepath.Join(v1.State, "workflows"))
	for _, test := range []struct {
		name     string
		roots    Roots
		strategy Strategy
	}{
		{"absent", absent, StrategyDirect},
		{"valid v1", v1, StrategyDirect},
		{"historical POC fixture", pocRoots(t), StrategyPreserveRebuildReconfigure},
	} {
		t.Run(test.name, func(t *testing.T) {
			if decision := Resolve(Owned, inspect(t, test.roots)); decision.Strategy != test.strategy {
				t.Fatalf("decision=%+v want %s", decision, test.strategy)
			}
		})
	}
}
