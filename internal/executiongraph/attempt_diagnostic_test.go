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
	r, err := DecodeSingleStageRun(mustRead(t, "testdata/axm36/single-run-v2-unknown.json"))
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
	for name, mutate := range map[string]func(*SingleStageRun){
		"certainty":       func(r *SingleStageRun) { r.Attempts[0].Diagnostic.EffectCertainty = "none" },
		"known":           func(r *SingleStageRun) { r.Attempts[0].Diagnostic.EffectCertainty = "known" },
		"wrong attempt":   func(r *SingleStageRun) { r.Attempts[0].Diagnostic.AttemptID = r.Scope.ExecutionID },
		"wrong operation": func(r *SingleStageRun) { r.Attempts[0].Diagnostic.Operation = "stage-preview" },
		"running": func(r *SingleStageRun) {
			r.Attempts[0].Status = AttemptRunning
			r.Attempts[0].FinishedAt = nil
			r.Attempts[0].ClassificationReason = ""
			r.Attempts[0].AmbiguousEffect = false
		},
		"success": func(r *SingleStageRun) {
			r.Attempts[0].Status = AttemptSucceeded
			r.Attempts[0].ExitCode = 0
			r.Attempts[0].AmbiguousEffect = false
			r.Attempts[0].ClassificationReason = ResultRecorded
			r.Attempts[0].QualifyingRecordRef = strings.Repeat("a", 64)
		},
	} {
		t.Run(name, func(t *testing.T) {
			r, _ := DecodeSingleStageRun(wire)
			mutate(&r)
			if _, err := EncodeSingleStageRun(r); err == nil {
				t.Fatal("diagnostic contradicts attempt")
			}
		})
	}
	r, _ := DecodeSingleStageRun(wire)
	next, _ := DecodeSingleStageRun(wire)
	next.Attempts[0].Diagnostic.Cause = "permission_denied"
	next.Attempts[0].Diagnostic.CauseConfidence = "confirmed"
	if ValidateSingleStageRun(next) != nil || ValidateSingleStageUpdate(r, next) == nil {
		t.Fatal("terminal diagnostic rewritten")
	}
	next, _ = DecodeSingleStageRun(wire)
	next.Attempts[0].Diagnostic = nil
	if ValidateSingleStageUpdate(r, next) == nil {
		t.Fatal("terminal diagnostic removed")
	}
	plain, _ := DecodeSingleStageRun(mustRead(t, "testdata/axm36/single-run-v2.json"))
	next = plain
	next.Attempts = append([]SuccessorAttempt(nil), plain.Attempts...)
	d := diagnosticFixture(t)
	next.Attempts[0].Diagnostic = &d
	if ValidateSingleStageUpdate(plain, next) == nil {
		t.Fatal("summary backfilled on a terminal attempt")
	}
	encoded, err := EncodeSingleStageRun(r)
	if err != nil || !bytes.Equal(wire, encoded) {
		t.Fatal("refused mutations changed original record")
	}
}

func TestOperationalDiagnosticUncertaintyRequiresCanonicalUnknown(t *testing.T) {
	for _, certainty := range []string{"unknown", "partial"} {
		t.Run(certainty, func(t *testing.T) {
			r, err := DecodeSingleStageRun(mustRead(t, "testdata/axm36/single-run-v2.json"))
			if err != nil {
				t.Fatal(err)
			}
			d := diagnosticFixture(t)
			d.EffectCertainty = certainty
			r.Attempts[0].Diagnostic = &d
			if ValidateSingleStageRun(r) == nil {
				t.Fatal("diagnostic uncertainty could coexist with retry-eligible failed state")
			}
			g := operationalGraphVector(t, "process_failed")
			g.Children[0].Attempts[0].Diagnostic.EffectCertainty = certainty
			if ValidateSuccessorGraph(g) == nil {
				t.Fatal("graph diagnostic uncertainty could coexist with failed state")
			}
		})
	}
}

func TestDiagnoseAttemptSeparatesCauseAndEffects(t *testing.T) {
	for _, tc := range []struct{ name, output, code, cause, confidence string }{
		{"option", "error: unexpected argument '--secret-token'", "invocation_incompatible", "invocation_incompatible", "hypothesis"},
		{"auth", "error: authentication failed: private-account@example.invalid", "authentication_unavailable", "authentication_unavailable", "hypothesis"},
		{"permission", "error: permission denied: /private/secret", "permission_denied", "permission_denied", "hypothesis"},
		{"model", "error: unsupported model", "model_unsupported", "model_unsupported", "hypothesis"},
		{"effort", "error: unsupported effort", "effort_unsupported", "effort_unsupported", "hypothesis"},
		{"unknown", "secret-token permission denied in a transcript", "subprocess_failed", "unknown", "unknown"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := SuccessorAttempt{AttemptID: "00000000-0000-4000-8000-000000000001", Number: 2, Status: AttemptFailed, ClassificationReason: ProcessFailed, OutputDigest: strings.Repeat("a", 64)}
			p := ProcessResult{ExitCode: 7, Output: []byte(tc.output)}
			d := DiagnoseAttempt(a, p)
			if ValidateAttemptDiagnostic(*d) != nil || d.Code != tc.code || d.Cause != tc.cause || d.CauseConfidence != tc.confidence || d.EffectCertainty != "known" || d.Operation != "stage-retry" {
				t.Fatalf("%+v", d)
			}
			wire, _ := json.Marshal(d)
			for _, secret := range []string{"secret-token", "private-account", "/private/secret"} {
				if bytes.Contains(wire, []byte(secret)) {
					t.Fatal("raw process output retained")
				}
			}
			a.Status, a.ClassificationReason = AttemptUnknown, ProcessOutcomeUnknown
			d = DiagnoseAttempt(a, p)
			if d.Code != "effects_uncertain" || d.EffectCertainty != "unknown" || d.NextAction != "observe-and-reconcile" || d.RequiredAuthority != "reconciliation" {
				t.Fatalf("%+v", d)
			}
		})
	}
}

func TestDiagnoseAttemptNoSpawnAndResultFailures(t *testing.T) {
	a := SuccessorAttempt{AttemptID: "00000000-0000-4000-8000-000000000001", Number: 1, Status: AttemptFailed, ClassificationReason: ProcessFailed}
	for _, code := range []string{"spawn_failed", "permission_denied", "executable_unavailable"} {
		d := DiagnoseAttempt(a, ProcessResult{SpawnFailed: true, FailureCode: code})
		if ValidateAttemptDiagnostic(*d) != nil || d.Code != code || d.CauseConfidence != "confirmed" || d.EffectCertainty != "none" {
			t.Fatalf("%+v", d)
		}
	}
	for reason, code := range map[ClassificationReason]string{ResultMissing: "result_missing", ResultNotSuccessful: "result_invalid", TimeoutStopped: "attempt_timeout", CancelConfirmed: "attempt_cancelled"} {
		a.ClassificationReason = reason
		d := DiagnoseAttempt(a, ProcessResult{})
		if d.Code != code || d.Cause != "unknown" {
			t.Fatalf("%+v", d)
		}
	}
	a.Status = AttemptSucceeded
	if DiagnoseAttempt(a, ProcessResult{}) != nil {
		t.Fatal("success acquired a failure diagnostic")
	}
}
