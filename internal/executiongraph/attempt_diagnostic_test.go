package executiongraph

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/strictjson"
)

func diagnosticFixture(t *testing.T) AttemptDiagnostic {
	t.Helper()
	r, err := DecodeOperationalSingleStageRun(mustRead(t, "testdata/axm36/single-run-v2-unknown.json"))
	if err != nil {
		t.Fatal(err)
	}
	return *r.Attempts[0].Diagnostic
}

func TestAttemptDiagnosticClosedCategories(t *testing.T) {
	for _, code := range []string{"executable_unavailable", "spawn_failed", "invocation_incompatible", "authentication_unavailable", "configuration_unsafe", "configuration_invalid", "permission_denied", "effort_override_conflict", "model_unsupported", "effort_unsupported", "attempt_timeout", "attempt_cancelled", "subprocess_failed", "result_missing", "result_invalid", "effects_uncertain"} {
		for _, confidence := range []string{"confirmed", "hypothesis", "unknown"} {
			for _, certainty := range []string{"none", "known", "partial", "unknown"} {
				d := diagnosticFixture(t)
				d.Code, d.Cause, d.CauseConfidence, d.EffectCertainty = code, code, confidence, certainty
				if confidence == "unknown" {
					d.Cause = "unknown"
				}
				wantInvalid := code == "effects_uncertain" && certainty != "partial" && certainty != "unknown"
				if (ValidateAttemptDiagnostic(d) != nil) != wantInvalid {
					t.Fatalf("diagnostic validity: %s/%s/%s", code, confidence, certainty)
				}
			}
		}
	}
	for name, mutate := range map[string]func(*AttemptDiagnostic){
		"unknown code":       func(d *AttemptDiagnostic) { d.Code = "vendor_guessed" },
		"unlabelled cause":   func(d *AttemptDiagnostic) { d.Cause = "permission_denied" },
		"false confirmation": func(d *AttemptDiagnostic) { d.CauseConfidence = "confirmed" },
		"uncertainty":        func(d *AttemptDiagnostic) { d.EffectCertainty = "success" },
		"authority":          func(d *AttemptDiagnostic) { d.RequiredAuthority = "automatic-repair" },
		"repair":             func(d *AttemptDiagnostic) { d.NextAction = "chmod-and-retry" },
		"duplicate ref":      func(d *AttemptDiagnostic) { d.EvidenceRefs = append(d.EvidenceRefs, d.EvidenceRefs[0]) },
		"oversize":           func(d *AttemptDiagnostic) { d.EvidenceRefs = make([]string, 17) },
	} {
		t.Run(name, func(t *testing.T) {
			d := diagnosticFixture(t)
			mutate(&d)
			if ValidateAttemptDiagnostic(d) == nil {
				t.Fatal("unsafe diagnostic admitted")
			}
		})
	}
}

func TestAttemptDiagnosticRejectsRawPayloads(t *testing.T) {
	seed := diagnosticFixture(t)
	wire, _ := json.Marshal(seed)
	var fields map[string]any
	if json.Unmarshal(wire, &fields) != nil {
		t.Fatal("bad seed")
	}
	for _, payload := range []string{"Bearer synthetic-secret", "/home/synthetic/private", "account@example.invalid", "raw transcript\npermission denied", strings.Repeat("x", 4096)} {
		for field := range fields {
			t.Run(field+"/"+payload[:4], func(t *testing.T) {
				bad := map[string]any{}
				for k, v := range fields {
					bad[k] = v
				}
				if field == "evidenceRefs" {
					bad[field] = []string{payload}
				} else {
					bad[field] = payload
				}
				b, _ := json.Marshal(bad)
				var d AttemptDiagnostic
				if strictjson.Decode(b, &d, 8192) == nil && ValidateAttemptDiagnostic(d) == nil {
					t.Fatal("raw payload admitted for persistence")
				}
			})
		}
	}
	for _, extra := range []string{`"stderr":"synthetic-secret",`, `"path":"/synthetic/private",`, `"Code":"permission_denied",`} {
		var d AttemptDiagnostic
		bad := append([]byte("{"+extra), wire[1:]...)
		if strictjson.Decode(bad, &d, 8192) == nil {
			t.Fatal("unknown or alias field admitted")
		}
	}
}

func TestOperationalDiagnosticCannotRewriteAttempt(t *testing.T) {
	wire := mustRead(t, "testdata/axm36/single-run-v2-unknown.json")
	for name, mutate := range map[string]func(*OperationalSingleStageRun){
		"certainty":       func(r *OperationalSingleStageRun) { r.Attempts[0].Diagnostic.EffectCertainty = "none" },
		"known":           func(r *OperationalSingleStageRun) { r.Attempts[0].Diagnostic.EffectCertainty = "known" },
		"wrong attempt":   func(r *OperationalSingleStageRun) { r.Attempts[0].Diagnostic.AttemptID = r.Scope.ExecutionID },
		"wrong operation": func(r *OperationalSingleStageRun) { r.Attempts[0].Diagnostic.Operation = "stage-preview" },
		"running": func(r *OperationalSingleStageRun) {
			r.Attempts[0].Status = AttemptRunning
			r.Attempts[0].FinishedAt = nil
			r.Attempts[0].ClassificationReason = ""
			r.Attempts[0].AmbiguousEffect = false
		},
		"success": func(r *OperationalSingleStageRun) {
			r.Attempts[0].Status = AttemptSucceeded
			r.Attempts[0].ExitCode = 0
			r.Attempts[0].AmbiguousEffect = false
			r.Attempts[0].ClassificationReason = ResultRecorded
			r.Attempts[0].QualifyingRecordRef = strings.Repeat("a", 64)
		},
	} {
		t.Run(name, func(t *testing.T) {
			r, _ := DecodeOperationalSingleStageRun(wire)
			mutate(&r)
			if _, err := EncodeOperationalSingleStageRun(r); err == nil {
				t.Fatal("diagnostic contradicts attempt")
			}
		})
	}
	r, _ := DecodeOperationalSingleStageRun(wire)
	next, _ := DecodeOperationalSingleStageRun(wire)
	next.Attempts[0].Diagnostic.Cause = "permission_denied"
	next.Attempts[0].Diagnostic.CauseConfidence = "confirmed"
	if ValidateOperationalSingleStageRun(next) != nil || ValidateOperationalSingleStageUpdate(r, next) == nil {
		t.Fatal("terminal diagnostic rewritten")
	}
	next, _ = DecodeOperationalSingleStageRun(wire)
	next.Attempts[0].Diagnostic = nil
	if ValidateOperationalSingleStageUpdate(r, next) == nil {
		t.Fatal("terminal diagnostic removed")
	}
	plain, _ := DecodeOperationalSingleStageRun(mustRead(t, "testdata/axm36/single-run-v2.json"))
	next = plain
	next.Attempts = append([]OperationalAttempt(nil), plain.Attempts...)
	d := diagnosticFixture(t)
	next.Attempts[0].Diagnostic = &d
	if ValidateOperationalSingleStageUpdate(plain, next) == nil {
		t.Fatal("summary backfilled on a terminal attempt")
	}
	encoded, err := EncodeOperationalSingleStageRun(r)
	if err != nil || !bytes.Equal(wire, encoded) {
		t.Fatal("refused mutations changed original record")
	}
}

func TestOperationalDiagnosticUncertaintyRequiresCanonicalUnknown(t *testing.T) {
	for _, certainty := range []string{"unknown", "partial"} {
		t.Run(certainty, func(t *testing.T) {
			r, err := DecodeOperationalSingleStageRun(mustRead(t, "testdata/axm36/single-run-v2.json"))
			if err != nil {
				t.Fatal(err)
			}
			d := diagnosticFixture(t)
			d.EffectCertainty = certainty
			r.Attempts[0].Diagnostic = &d
			if ValidateOperationalSingleStageRun(r) == nil {
				t.Fatal("diagnostic uncertainty could coexist with retry-eligible failed state")
			}
			g := operationalGraphVector(t, "process_failed")
			g.Children[0].Attempts[0].Diagnostic.EffectCertainty = certainty
			if ValidateOperationalGraph(g) == nil {
				t.Fatal("graph diagnostic uncertainty could coexist with failed state")
			}
		})
	}
}
