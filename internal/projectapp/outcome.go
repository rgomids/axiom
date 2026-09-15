package projectapp

// Commit status describes the Project/installation mutation, never a Git commit.
// Zero is deliberately uncertain; an omitted adapter result cannot imply rollback.
type CommitStatus uint8

const (
	CommitUnknown CommitStatus = iota
	NotCommitted
	Committed
)

type MutationStatus uint8

const (
	Failed MutationStatus = iota
	Applied
	Unchanged
	Denied
	Conflict
	Cancelled
	RecoveryRequired
)

type MutationResult struct {
	Commit CommitStatus
	Status MutationStatus
	Issues []Issue
}

func (r MutationResult) Normalized() MutationResult {
	r.Issues = OrderedIssues(r.Issues)
	if r.Commit > Committed {
		r.Commit = CommitUnknown
	}
	if r.Commit == CommitUnknown {
		r.Status = RecoveryRequired
		return r
	}
	if r.Status > RecoveryRequired || (r.Commit == NotCommitted && r.Status == Applied) {
		r.Status = RecoveryRequired
	}
	// Preserve known committed state for every later failure/cancellation. The
	// result must direct recovery, never describe a denied/rolled-back mutation.
	if r.Commit == Committed && r.Status != Applied {
		r.Status = RecoveryRequired
	}
	return r
}

type Readiness uint8

const (
	ReadinessUnverified Readiness = iota
	UnresolvedDependencies
)

type Classification uint8

const (
	PreCommitFailure Classification = iota
	MutationCommitted
	CommittedWithLocalFailure
	RecoveryNeededOutcome
	NoMutation
)

// Outcome keeps separate results; no optional Git result or global ready flag.
// LocalAttempted distinguishes an absent local step from an unknown local commit.
type Outcome struct {
	Portable       MutationResult
	Local          MutationResult
	LocalAttempted bool
	Readiness      Readiness
}

func (o Outcome) Classify() Classification {
	p := o.Portable.Normalized()
	if p.Commit == CommitUnknown || p.Status == RecoveryRequired {
		return RecoveryNeededOutcome
	}
	if p.Commit == Committed {
		if o.LocalAttempted && o.Local.Normalized().Status != Applied && o.Local.Normalized().Status != Unchanged {
			return CommittedWithLocalFailure
		}
		return MutationCommitted
	}
	if p.Status != Unchanged {
		return PreCommitFailure
	}
	if o.LocalAttempted {
		local := o.Local.Normalized()
		if local.Commit == CommitUnknown || local.Status == RecoveryRequired {
			return RecoveryNeededOutcome
		}
		if local.Commit == Committed {
			return MutationCommitted
		}
		if local.Status != Unchanged {
			return PreCommitFailure
		}
	}
	return NoMutation
}
