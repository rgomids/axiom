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
