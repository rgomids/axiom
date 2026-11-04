package workflow

import "time"

const SpawnBoundaryPolicy = "spawn-boundary-v1"
const (
	AccountingPending = "pending"
	AccountingCharged = "charged"
	AccountingRefusal = "deterministic_refusal"
)

// AttemptCounts derives both lifetime budgets from the bounded canonical history.
func AttemptCounts(attempts []Attempt) (charged, refusals int) {
	for _, a := range attempts {
		if a.Accounting == AccountingRefusal {
			refusals++
		} else {
			charged++
		}
	}
	return
}
func (e Execution) attemptBudgetAvailable(task TaskRun, agent AgentSpec) bool {
	if e.AttemptPolicy == "" {
		return len(task.Attempts) < agent.AttemptLimit()
	}
	charged, refusals := AttemptCounts(task.Attempts)
	return charged < agent.AttemptLimit() && refusals < 2
}
func (e *Execution) ReserveLaunch(id string, generation uint64, now time.Time) error {
	a, err := e.fencedAttempt(id, generation, AttemptPrepared)
	if err != nil {
		return err
	}
	if e.AttemptPolicy != SpawnBoundaryPolicy || a.Accounting != AccountingPending || e.Cancel != nil {
		return ErrTransition
	}
	a.Accounting = AccountingCharged
	e.touch(now)
	return e.Validate()
}

// MarkLaunchRefused requires the engine's fenced pre-Runner assertion. Absence
// of process identity alone is never evidence that a launch did not occur.
func (e *Execution) MarkLaunchRefused(id string, generation uint64, failure LaunchFailure, materialization string, now time.Time) error {
	a, err := e.fencedAttempt(id, generation, AttemptPrepared)
	if err != nil {
		return err
	}
	if e.AttemptPolicy != SpawnBoundaryPolicy || a.Accounting != AccountingPending || !failure.Valid() || failure.Reason == "builder_failed" || failure.Phase == "spawn" || failure.Phase == "runtime" || e.Cancel != nil || (materialization != "none" && materialization != "prepared") {
		return ErrTransition
	}
	at := now.UTC()
	a.Accounting = AccountingRefusal
	a.NoSpawn = true
	a.Materialization = materialization
	a.LaunchFailure = &failure
	a.Status = AttemptFailed
	a.EndedAt = &at
	a.Diagnostic = "launch_refused: " + failure.Reason
	e.touch(now)
	return e.Validate()
}
func (e Execution) validateAccounting(task TaskRun, agent AgentSpec) error {
	if e.AttemptPolicy == "" {
		if len(task.Attempts) > agent.AttemptLimit() {
			return invalid("legacy attempt bound")
		}
		for _, a := range task.Attempts {
			if a.Accounting != "" || a.ParentAttemptID != "" || a.NoSpawn || a.Materialization != "" {
				return invalid("legacy accounting metadata")
			}
		}
		return nil
	}
	charged, refusals := AttemptCounts(task.Attempts)
	if charged > agent.AttemptLimit() || refusals > 2 || len(task.Attempts) > agent.AttemptLimit()+2 {
		return invalid("attempt accounting bound")
	}
	for n, a := range task.Attempts {
		priorCharged, priorRefusals := AttemptCounts(task.Attempts[:n])
		if priorCharged >= agent.AttemptLimit() || priorRefusals >= 2 {
			return invalid("attempt after exhausted lifetime budget")
		}
		if a.Accounting != AccountingPending && a.Accounting != AccountingCharged && a.Accounting != AccountingRefusal {
			return invalid("attempt accounting")
		}
		if n == 0 && a.ParentAttemptID != "" || n > 0 && (a.ParentAttemptID != task.Attempts[n-1].AttemptID || a.WorkspaceDisposition != "preserve") {
			return invalid("retry lineage")
		}
		if a.Accounting == AccountingPending && a.Status != AttemptPrepared && a.Status != AttemptUnknown {
			return invalid("pending attempt")
		}
		if a.Accounting == AccountingRefusal {
			if !a.NoSpawn || a.Materialization != "none" && a.Materialization != "prepared" || a.Status != AttemptFailed || a.Process != nil || a.StartedAt != nil || a.EndedAt == nil || a.LaunchFailure == nil || a.LaunchFailure.Reason == "builder_failed" || a.LaunchFailure.Phase == "spawn" || a.LaunchFailure.Phase == "runtime" {
				return invalid("deterministic refusal evidence")
			}
		} else if a.NoSpawn || a.Materialization != "" {
			return invalid("unproved no-spawn")
		}
	}
	return nil
}
