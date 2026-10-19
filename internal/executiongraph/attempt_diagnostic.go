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

func validateOperationalDiagnostic(a OperationalAttempt) error {
	if a.Diagnostic == nil {
		return nil
	}
	d := *a.Diagnostic
	if ValidateAttemptDiagnostic(d) != nil || a.Status == AttemptRunning || a.Status == AttemptSucceeded || d.AttemptID != a.AttemptID || d.Operation != "stage-run" && d.Operation != "stage-retry" {
		return ErrInvalidGraph
	}
	if a.Status == AttemptUnknown && d.EffectCertainty != "unknown" && d.EffectCertainty != "partial" || a.AmbiguousEffect && d.EffectCertainty != "unknown" && d.EffectCertainty != "partial" {
		return ErrInvalidGraph
	}
	return nil
}
