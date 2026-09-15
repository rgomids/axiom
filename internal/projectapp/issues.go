package projectapp

import "sort"

// Diagnostic dimensions use fixed vocabularies; no rejected scalar, arbitrary
// path, key, OS error or parser excerpt can be stored in an Issue.
type Phase uint8

const (
	ValidationPhase Phase = iota
	InspectionPhase
	ApprovalPhase
	PersistencePhase
	LocalPhase
)

type Field uint8

const (
	ProjectField Field = iota
	TextField
	AuthorityField
	RevisionField
	ArtifactsField
	InstallationField
)

type Code uint8

const (
	InvalidSnapshot Code = iota
	InvalidPreview
	MissingAuthority
	RevokedAuthority
	StaleAuthority
	RevisionMismatch
	WrongOperation
	CancelledOperation
	StoreFailure
	ConcurrentMutation
	RecoveryNeeded
	ScannerUnavailable
	ScannerFinding
	ScannerFailed
)

func (c Code) String() string {
	names := [...]string{"invalid_snapshot", "invalid_preview", "missing_authority", "revoked_authority", "stale_authority", "revision_mismatch", "wrong_operation", "cancelled", "store_failure", "concurrent_mutation", "recovery_required", "scanner_unavailable", "scanner_finding", "scanner_failed"}
	if int(c) >= len(names) {
		return "invalid_issue"
	}
	return names[c]
}
func (f Field) String() string {
	names := [...]string{"project", "text", "authority", "revision", "artifacts", "installation"}
	if int(f) >= len(names) {
		return "project"
	}
	return names[f]
}

type Issue struct {
	Phase Phase
	Field Field
	Code  Code
	Index uint32
}

func (i Issue) Severity() string {
	if i.Code >= ScannerUnavailable && i.Code <= ScannerFailed {
		return "warning"
	}
	return "error"
}
func (i Issue) Category() string {
	switch i.Code {
	case MissingAuthority, RevokedAuthority, StaleAuthority, WrongOperation:
		return "authority"
	case RevisionMismatch, ConcurrentMutation:
		return "conflict"
	case StoreFailure:
		return "persistence"
	case RecoveryNeeded:
		return "recovery"
	case ScannerUnavailable, ScannerFinding, ScannerFailed:
		return "inspection"
	}
	return "validation"
}
func (i Issue) Remedy() string {
	switch i.Category() {
	case "authority", "conflict":
		return "refresh_preview_and_confirm"
	case "persistence", "recovery":
		return "preserve_state_and_inspect_outcome"
	case "inspection":
		return "review_and_sanitize_text"
	}
	return "correct_input"
}
func OrderedIssues(input []Issue) []Issue {
	result := append([]Issue(nil), input...)
	sort.Slice(result, func(i, j int) bool {
		a, b := result[i], result[j]
		if a.Phase != b.Phase {
			return a.Phase < b.Phase
		}
		if a.Field.String() != b.Field.String() {
			return a.Field.String() < b.Field.String()
		}
		if a.Field != b.Field {
			return a.Field < b.Field
		}
		if a.Code.String() != b.Code.String() {
			return a.Code.String() < b.Code.String()
		}
		if a.Code != b.Code {
			return a.Code < b.Code
		}
		return a.Index < b.Index
	})
	return result
}
func problem(code Code) []Issue {
	if code == InvalidSnapshot {
		return []Issue{{Phase: ValidationPhase, Field: ArtifactsField, Code: code}}
	}
	if code == RevisionMismatch {
		return []Issue{{Phase: ApprovalPhase, Field: RevisionField, Code: code}}
	}
	return []Issue{{Phase: ApprovalPhase, Field: AuthorityField, Code: code}}
}
