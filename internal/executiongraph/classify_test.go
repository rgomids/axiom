package executiongraph

import "testing"

func TestClassifyAttemptRules(t *testing.T) {
	correlation := AttemptCorrelation{"parent", "child", "attempt", 2}
	result := AttemptRecord{"parent", "child", "attempt", 2, 1, "result", "succeeded", "result-digest", true}
	blocker := result
	blocker.Kind, blocker.Digest = "blocker", "blocker-digest"
	laterResult := result
	laterResult.Revision = 2
	laterBlocker := blocker
	laterBlocker.Revision = 2
	failed := result
	failed.Outcome = "failed"
	otherAttempt := result
	otherAttempt.AttemptID = "other"
	otherRevision := result
	otherRevision.GraphRevision = 1
	otherParent := result
	otherParent.ParentID = "other"
	rejected := result
	rejected.Accepted = false
	tests := []struct {
		name    string
		process ProcessResult
		cancel  bool
		records []AttemptRecord
		status  AttemptStatus
		reason  ClassificationReason
		ref     string
	}{
		{"rule0", ProcessResult{Stopped: true}, true, []AttemptRecord{blocker}, AttemptCancelled, CancelConfirmed, ""},
		{"rule1", ProcessResult{OutcomeUnknown: true}, false, []AttemptRecord{result}, AttemptUnknown, ProcessOutcomeUnknown, ""},
		{"timeout_unconfirmed", ProcessResult{TimedOut: true}, false, nil, AttemptUnknown, ProcessOutcomeUnknown, ""},
		{"ambiguous_cancel", ProcessResult{Stopped: true, EffectAmbiguous: true}, true, nil, AttemptUnknown, ProcessOutcomeUnknown, ""},
		{"cancel_unconfirmed", ProcessResult{}, true, nil, AttemptUnknown, ProcessOutcomeUnknown, ""},
		{"rule1b", ProcessResult{TimedOut: true, Stopped: true, OutcomeUnknown: true}, false, []AttemptRecord{blocker}, AttemptFailed, TimeoutStopped, ""},
		{"rule2", ProcessResult{ExitCode: 1}, false, []AttemptRecord{blocker}, AttemptBlocked, BlockerPublished, blocker.Digest},
		{"rule3", ProcessResult{ExitCode: 1}, false, []AttemptRecord{result}, AttemptFailed, ProcessFailed, ""},
		{"rule4", ProcessResult{}, false, nil, AttemptFailed, ResultMissing, ""},
		{"rule5", ProcessResult{}, false, []AttemptRecord{failed}, AttemptFailed, ResultNotSuccessful, failed.Digest},
		{"rule6", ProcessResult{}, false, []AttemptRecord{result}, AttemptSucceeded, ResultRecorded, result.Digest},
		{"blocker_then_result", ProcessResult{}, false, []AttemptRecord{laterResult, blocker}, AttemptSucceeded, ResultRecorded, result.Digest},
		{"result_then_blocker", ProcessResult{}, false, []AttemptRecord{laterBlocker, result}, AttemptBlocked, BlockerPublished, blocker.Digest},
		{"uncorrelated_rejected", ProcessResult{}, false, []AttemptRecord{otherAttempt, otherRevision, otherParent, rejected}, AttemptFailed, ResultMissing, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, reason, ref := ClassifyAttempt(tt.process, tt.cancel, correlation, tt.records)
			if status != tt.status || reason != tt.reason || ref != tt.ref {
				t.Fatalf("got %s/%s/%s", status, reason, ref)
			}
		})
	}
}

func TestLatestAttemptRecordTotalOrdering(t *testing.T) {
	c := AttemptCorrelation{"p", "c", "a", 1}
	a := AttemptRecord{"p", "c", "a", 1, 2, "result", "succeeded", "a", true}
	b := a
	b.Digest = "b"
	b.Kind = "blocker"
	for _, records := range [][]AttemptRecord{{a, b}, {b, a}} {
		latest, ok := LatestAttemptRecord(c, records)
		if !ok || latest != b {
			t.Fatalf("latest=%+v", latest)
		}
	}
}

func TestClassifyIntegrationReasons(t *testing.T) {
	for _, tt := range []struct {
		process ProcessResult
		status  AttemptStatus
		reason  ClassificationReason
	}{
		{ProcessResult{}, AttemptSucceeded, IntegrationApplied},
		{ProcessResult{ExitCode: 1}, AttemptFailed, IntegrationFailed},
		{ProcessResult{OutcomeUnknown: true}, AttemptUnknown, IntegrationUnknown},
		{ProcessResult{TimedOut: true, Stopped: true}, AttemptFailed, IntegrationFailed},
	} {
		status, reason := ClassifyIntegrationAttempt(tt.process)
		if status != tt.status || reason != tt.reason {
			t.Fatalf("%s/%s", status, reason)
		}
	}
}
