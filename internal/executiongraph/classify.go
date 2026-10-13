package executiongraph

// ClassificationReason describes a terminal outcome without changing graph wire formats.
type ClassificationReason string

const (
	CancelConfirmed       ClassificationReason = "cancel_confirmed"
	ProcessOutcomeUnknown ClassificationReason = "process_outcome_unknown"
	TimeoutStopped        ClassificationReason = "timeout_stopped"
	BlockerPublished      ClassificationReason = "blocker_published"
	ProcessFailed         ClassificationReason = "process_failed"
	ResultMissing         ClassificationReason = "result_missing"
	ResultNotSuccessful   ClassificationReason = "result_not_successful"
	ResultRecorded        ClassificationReason = "result_recorded"
	IntegrationApplied    ClassificationReason = "integration_applied"
	IntegrationFailed     ClassificationReason = "integration_failed"
	IntegrationUnknown    ClassificationReason = "integration_unknown"
)

// AttemptRecord is a projection of an accepted canonical coordination record.
// Rejected or quarantined candidates must never be projected as Accepted.
// coordination owns validation and constructs this projection to avoid an import cycle.
type AttemptRecord struct {
	ParentID, ChildID, AttemptID string
	GraphRevision, Revision      uint64
	Kind, Outcome, Digest        string
	Accepted                     bool
}

type AttemptCorrelation struct {
	ParentID, ChildID, AttemptID string
	GraphRevision                uint64
}

// LatestAttemptRecord selects the highest canonical revision. Digest is a
// deterministic tie-breaker for malformed duplicate revisions; canonical streams
// have unique revisions. The input is never modified.
func LatestAttemptRecord(correlation AttemptCorrelation, records []AttemptRecord) (AttemptRecord, bool) {
	var latest AttemptRecord
	found := false
	for _, record := range records {
		if !record.Accepted || record.Revision == 0 || record.Digest == "" || record.ParentID != correlation.ParentID || record.ChildID != correlation.ChildID || record.AttemptID != correlation.AttemptID || record.GraphRevision != correlation.GraphRevision || (record.Kind != "blocker" && record.Kind != "result") {
			continue
		}
		if !found || record.Revision > latest.Revision || record.Revision == latest.Revision && record.Digest > latest.Digest {
			latest, found = record, true
		}
	}
	return latest, found
}

// ClassifyAttempt implements Plan §3.5 in first-match order. Process uncertainty
// and ambiguous effects override coordination, which cannot prove process stop.
func ClassifyAttempt(process ProcessResult, cancelRequested bool, correlation AttemptCorrelation, records []AttemptRecord) (AttemptStatus, ClassificationReason, string) {
	if cancelRequested && process.Stopped && !process.EffectAmbiguous {
		return AttemptCancelled, CancelConfirmed, ""
	}
	if process.EffectAmbiguous || process.OutcomeUnknown && !(process.TimedOut && process.Stopped) || process.TimedOut && !process.Stopped || cancelRequested && !process.Stopped {
		return AttemptUnknown, ProcessOutcomeUnknown, ""
	}
	if process.TimedOut && process.Stopped {
		return AttemptFailed, TimeoutStopped, ""
	}
	latest, found := LatestAttemptRecord(correlation, records)
	if found && latest.Kind == "blocker" {
		return AttemptBlocked, BlockerPublished, latest.Digest
	}
	if process.ExitCode != 0 {
		return AttemptFailed, ProcessFailed, ""
	}
	if !found {
		return AttemptFailed, ResultMissing, ""
	}
	if latest.Outcome != "succeeded" {
		return AttemptFailed, ResultNotSuccessful, latest.Digest
	}
	return AttemptSucceeded, ResultRecorded, latest.Digest
}

// ClassifyIntegrationAttempt retains integration's process-only semantics.
func ClassifyIntegrationAttempt(process ProcessResult) (AttemptStatus, ClassificationReason) {
	if process.OutcomeUnknown || process.EffectAmbiguous || process.TimedOut && !process.Stopped {
		return AttemptUnknown, IntegrationUnknown
	}
	if process.ExitCode != 0 || process.TimedOut {
		return AttemptFailed, IntegrationFailed
	}
	return AttemptSucceeded, IntegrationApplied
}
