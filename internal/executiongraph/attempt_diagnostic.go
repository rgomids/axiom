package executiongraph

import "strings"

type AttemptDiagnostic struct {
	Operation         string   `json:"operation"`
	AttemptID         string   `json:"attemptId,omitempty"`
	Code              string   `json:"code"`
	Cause             string   `json:"cause"`
	CauseConfidence   string   `json:"causeConfidence"`
	EffectCertainty   string   `json:"effectCertainty"`
	EvidenceRefs      []string `json:"evidenceRefs"`
	NextAction        string   `json:"nextAction"`
	RequiredAuthority string   `json:"requiredAuthority"`
}

func ValidateAttemptDiagnostic(d AttemptDiagnostic) error {
	switch d.Operation {
	case "workflow-preview", "workflow-start", "stage-plan", "stage-preview", "stage-run", "stage-retry":
	default:
		return ErrInvalidGraph
	}
	if d.AttemptID != "" && !validOpaqueID(d.AttemptID) || !validDiagnosticCode(d.Code) || len(d.EvidenceRefs) > 16 {
		return ErrInvalidGraph
	}
	if d.Cause == "unknown" {
		if d.CauseConfidence != "unknown" {
			return ErrInvalidGraph
		}
	} else if !validDiagnosticCode(d.Cause) || d.CauseConfidence != "confirmed" && d.CauseConfidence != "hypothesis" {
		return ErrInvalidGraph
	}
	switch d.EffectCertainty {
	case "none", "known", "partial", "unknown":
	default:
		return ErrInvalidGraph
	}
	if d.Code == "effects_uncertain" && d.EffectCertainty != "partial" && d.EffectCertainty != "unknown" {
		return ErrInvalidGraph
	}

	switch d.NextAction {
	case "inspect-executable-and-preview", "inspect-adapter-and-preview", "restore-auth-and-preview", "inspect-configuration-and-preview", "select-profile-and-preview", "observe-and-reconcile", "inspect-result-and-reconcile":
	default:
		return ErrInvalidGraph
	}
	switch d.RequiredAuthority {
	case "inspection", "configuration", "reconciliation":
	default:
		return ErrInvalidGraph
	}
	seen := map[string]bool{}
	for _, ref := range d.EvidenceRefs {
		if !strings.HasPrefix(ref, "evidence:") || !successorDigest(strings.TrimPrefix(ref, "evidence:")) || seen[ref] {
			return ErrInvalidGraph
		}
		seen[ref] = true
	}
	return nil
}

func validDiagnosticCode(code string) bool {
	switch code {
	case "executable_unavailable", "spawn_failed", "invocation_incompatible", "authentication_unavailable", "configuration_unsafe", "configuration_invalid", "permission_denied", "effort_override_conflict", "model_unsupported", "effort_unsupported", "attempt_timeout", "attempt_cancelled", "subprocess_failed", "result_missing", "result_invalid", "effects_uncertain":
		return true
	}
	return false
}

func validateDiagnostic(a SuccessorAttempt) error {
	d := *a.Diagnostic
	if ValidateAttemptDiagnostic(d) != nil || a.Status == AttemptRunning || a.Status == AttemptSucceeded || d.AttemptID != a.AttemptID || d.Operation != "stage-run" && d.Operation != "stage-retry" {
		return ErrInvalidGraph
	}
	if a.Status == AttemptUnknown && d.EffectCertainty != "unknown" && d.EffectCertainty != "partial" || a.AmbiguousEffect && d.EffectCertainty != "unknown" && d.EffectCertainty != "partial" {
		return ErrInvalidGraph
	}
	if (d.EffectCertainty == "unknown" || d.EffectCertainty == "partial") && a.Status != AttemptUnknown {
		return ErrInvalidGraph
	}
	return nil
}

// DiagnoseAttempt describes the observed terminal boundary without retaining
// process output or treating a diagnostic hypothesis as effect certainty.
func DiagnoseAttempt(a SuccessorAttempt, p ProcessResult) *AttemptDiagnostic {
	if a.Status == AttemptSucceeded {
		return nil
	}
	d := &AttemptDiagnostic{Operation: "stage-run", AttemptID: a.AttemptID, Code: "subprocess_failed", Cause: "unknown", CauseConfidence: "unknown", EffectCertainty: "known", EvidenceRefs: []string{}, NextAction: "inspect-adapter-and-preview", RequiredAuthority: "inspection"}
	if a.Number > 1 {
		d.Operation = "stage-retry"
	}
	if successorDigest(a.OutputDigest) {
		d.EvidenceRefs = append(d.EvidenceRefs, "evidence:"+a.OutputDigest)
	}
	switch a.ClassificationReason {
	case CancelConfirmed:
		d.Code = "attempt_cancelled"
	case TimeoutStopped:
		d.Code = "attempt_timeout"
	case ResultMissing:
		d.Code = "result_missing"
	case ResultNotSuccessful:
		d.Code = "result_invalid"
	case ProcessOutcomeUnknown:
		d.Code = "effects_uncertain"
	}
	if p.TimedOut {
		d.Code, d.Cause, d.CauseConfidence = "attempt_timeout", "attempt_timeout", "confirmed"
	}
	if p.SpawnFailed {
		d.Code, d.EffectCertainty = "spawn_failed", "none"
		switch p.FailureCode {
		case "permission_denied", "executable_unavailable", "spawn_failed", "attempt_timeout", "attempt_cancelled":
			d.Code = p.FailureCode
		}
		d.Cause, d.CauseConfidence = d.Code, "confirmed"
	} else if p.ExitCode != 0 {
		if code := recognizedFailure(p.Output); code != "" && d.CauseConfidence != "confirmed" {
			d.Cause, d.CauseConfidence = code, "hypothesis"
			if a.ClassificationReason == ProcessFailed {
				d.Code = code
			}
		}
	}
	switch d.Code {
	case "executable_unavailable", "spawn_failed":
		d.NextAction = "inspect-executable-and-preview"
	case "authentication_unavailable":
		d.NextAction, d.RequiredAuthority = "restore-auth-and-preview", "configuration"
	case "configuration_unsafe", "configuration_invalid", "effort_override_conflict", "permission_denied":
		d.NextAction, d.RequiredAuthority = "inspect-configuration-and-preview", "configuration"
	case "model_unsupported", "effort_unsupported":
		d.NextAction, d.RequiredAuthority = "select-profile-and-preview", "configuration"
	case "result_missing", "result_invalid":
		d.NextAction = "inspect-result-and-reconcile"
	}
	if a.Status == AttemptUnknown || a.AmbiguousEffect {
		d.EffectCertainty, d.NextAction, d.RequiredAuthority = "unknown", "observe-and-reconcile", "reconciliation"
	}
	return d
}

func recognizedFailure(output []byte) string {
	if len(output) > 4096 {
		return ""
	}
	lines := strings.Split(strings.ToLower(string(output)), "\n")
	if len(lines) > 32 {
		return ""
	}
	for _, line := range lines {
		if len(line) > 512 {
			continue
		}
		line = strings.TrimSpace(line)
		for _, prefix := range []string{"error: unexpected argument ", "error: unknown option ", "error: unrecognized option ", "error: unknown flag "} {
			if strings.HasPrefix(line, prefix) {
				return "invocation_incompatible"
			}
		}
		for _, pair := range [][2]string{{"error: authentication failed", "authentication_unavailable"}, {"error: not logged in", "authentication_unavailable"}, {"error: permission denied", "permission_denied"}, {"error: model not found", "model_unsupported"}, {"error: unsupported model", "model_unsupported"}, {"error: unsupported effort", "effort_unsupported"}, {"error: invalid effort", "effort_unsupported"}} {
			if line == pair[0] || strings.HasPrefix(line, pair[0]+":") {
				return pair[1]
			}
		}
	}
	return ""
}
