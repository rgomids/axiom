package workflow

import (
	"fmt"
	"time"
)

// Every mutation below is fenced by the runner generation, increments the
// revision and leaves Status derived from facts; callers persist the result
// through the store's revision CAS. None of them spawns or signals anything.

// DispositionAbandoned is the only human disposition of an unknown attempt:
// its effects stay recorded as uncertain and are never replayed; it only
// permits an explicit new attempt.
const DispositionAbandoned = "abandoned"

// DispositionApproved records a real human gate decision.
const DispositionApproved = "approved"

func (e *Execution) touch(now time.Time) {
	e.Revision++
	e.UpdatedAt = now.UTC()
	if e.UpdatedAt.Before(e.CreatedAt) {
		e.UpdatedAt = e.CreatedAt
	}
	for i := range e.Stages {
		e.Stages[i].Status = e.derivedStageStatus(i)
	}
	e.Status = e.derivedStatus()
}

// ClaimRunner records a newly acquired, strictly newer runner lease
// generation. Older runners can no longer commit anything.
func (e *Execution) ClaimRunner(generation uint64, now time.Time) error {
	if generation <= e.RunnerGeneration {
		return ErrStaleRunner
	}
	e.RunnerGeneration = generation
	e.touch(now)
	return nil
}

func (e *Execution) fence(generation uint64) error {
	if generation == 0 || generation != e.RunnerGeneration {
		return ErrStaleRunner
	}
	return nil
}

func (e *Execution) stageIndex(id string) int {
	for i, s := range e.Workflow.Stages {
		if s.ID == id {
			return i
		}
	}
	return -1
}

func (e *Execution) task(ref TaskRef) (*TaskRun, AgentSpec, StageSpec, error) {
	i := e.stageIndex(ref.StageID)
	if i < 0 {
		return nil, AgentSpec{}, StageSpec{}, ErrTransition
	}
	for j := range e.Stages[i].Tasks {
		if e.Stages[i].Tasks[j].AgentID == ref.AgentID {
			return &e.Stages[i].Tasks[j], e.Workflow.Stages[i].Agents[j], e.Workflow.Stages[i], nil
		}
	}
	return nil, AgentSpec{}, StageSpec{}, ErrTransition
}

func lastAttempt(t TaskRun) *Attempt {
	if len(t.Attempts) == 0 {
		return nil
	}
	return &t.Attempts[len(t.Attempts)-1]
}

// FindAttempt returns the attempt and its task reference.
func (e *Execution) FindAttempt(id string) (*Attempt, TaskRef, bool) {
	for i := range e.Stages {
		for j := range e.Stages[i].Tasks {
			for k := range e.Stages[i].Tasks[j].Attempts {
				if e.Stages[i].Tasks[j].Attempts[k].AttemptID == id {
					return &e.Stages[i].Tasks[j].Attempts[k], TaskRef{e.Stages[i].StageID, e.Stages[i].Tasks[j].AgentID}, true
				}
			}
		}
	}
	return nil, TaskRef{}, false
}

func unresolvedUnknown(a Attempt) bool { return a.Status == AttemptUnknown && a.Resolution == nil }

// effectiveStatus treats an abandoned unknown attempt as failed for
// progression, while its stored status and uncertain effects stay unchanged.
func effectiveStatus(a Attempt) AttemptStatus {
	if a.Status == AttemptUnknown && a.Resolution != nil {
		return AttemptFailed
	}
	return a.Status
}

func (e *Execution) derivedStageStatus(i int) StageStatus {
	run := e.Stages[i]
	succeeded, started := 0, false
	failed, cancelled, active := false, false, false
	for _, t := range run.Tasks {
		last := lastAttempt(t)
		if last == nil {
			continue
		}
		started = true
		switch effectiveStatus(*last) {
		case AttemptSucceeded:
			succeeded++
		case AttemptPrepared, AttemptRunning, AttemptUnknown:
			active = true
		case AttemptFailed:
			failed = true
		case AttemptCancelled:
			cancelled = true
		}
	}
	switch {
	case active:
		return StageRunning
	case succeeded == len(run.Tasks):
		if e.Workflow.Stages[i].HumanGate && run.Gate == nil {
			return StageAwaitingGate
		}
		return StageCompleted
	case failed:
		return StageFailed
	case cancelled:
		return StageCancelled
	case started:
		return StageRunning
	}
	return StagePending
}

func (e Execution) derivedStatus() RunStatus {
	all := true
	var running, failed, cancelled, gate bool
	for i, s := range e.Stages {
		for _, t := range s.Tasks {
			for _, a := range t.Attempts {
				if unresolvedUnknown(a) {
					return RunRecoveryRequired
				}
			}
		}
		switch e.derivedStageStatus(i) {
		case StageRunning:
			running = true
		case StageFailed:
			failed = true
		case StageCancelled:
			cancelled = true
		case StageAwaitingGate:
			gate = true
		}
		if e.derivedStageStatus(i) != StageCompleted {
			all = false
		}
	}
	switch {
	case running && hasActiveAttempt(e):
		return RunRunning
	case e.Cancel != nil || cancelled:
		return RunCancelled
	case failed:
		return RunFailed
	case all:
		return RunCompleted
	case gate:
		return RunAwaitingGate
	case running:
		return RunRunning
	}
	return RunReady
}

func hasActiveAttempt(e Execution) bool {
	for _, s := range e.Stages {
		for _, t := range s.Tasks {
			if last := lastAttempt(t); last != nil && (last.Status == AttemptPrepared || last.Status == AttemptRunning) {
				return true
			}
		}
	}
	return false
}

// Ready lists never-attempted tasks whose dependencies completed (including
// any human gate), in Workflow order. Cancellation, failure or any unresolved
// unknown effect yields nothing: successors stop.
func (e Execution) Ready() []TaskRef {
	if e.Cancel != nil || e.Status == RunRecoveryRequired || e.Status == RunFailed || e.Status == RunCancelled {
		return nil
	}
	var refs []TaskRef
	for i, spec := range e.Workflow.Stages {
		ready := true
		for _, dep := range spec.DependsOn {
			if e.Stages[e.stageIndex(dep)].Status != StageCompleted {
				ready = false
			}
		}
		if !ready {
			continue
		}
		for _, t := range e.Stages[i].Tasks {
			if len(t.Attempts) == 0 {
				refs = append(refs, TaskRef{spec.ID, t.AgentID})
			}
		}
	}
	return refs
}

type PrepareInput struct {
	Ref        TaskRef
	AttemptID  string
	Generation uint64
	Invocation Invocation
	Authority  AttemptAuthority
	Now        time.Time
	// Retry permits a new attempt after a failed, cancelled or abandoned
	// unknown attempt; it never applies to an unresolved unknown.
	Retry bool
}

// PrepareAttempt records the frozen invocation and authority before any
// spawn. The authority must stay within the agent's Workflow effect bound and
// the bound repositories.
func (e *Execution) PrepareAttempt(in PrepareInput) error {
	if err := e.fence(in.Generation); err != nil {
		return err
	}
	if e.Cancel != nil {
		return ErrCancelled
	}
	if e.Status == RunRecoveryRequired {
		return ErrUnknownOutstanding
	}
	task, agent, _, err := e.task(in.Ref)
	if err != nil {
		return err
	}
	last := lastAttempt(*task)
	switch {
	case last == nil && in.Retry:
		return fmt.Errorf("%w: nothing to retry", ErrTransition)
	case last == nil:
		eligible := false
		for _, r := range e.Ready() {
			eligible = eligible || r == in.Ref
		}
		if !eligible {
			return fmt.Errorf("%w: task is not eligible", ErrTransition)
		}
	case !in.Retry:
		return fmt.Errorf("%w: task already attempted", ErrTransition)
	default:
		if s := effectiveStatus(*last); s != AttemptFailed && s != AttemptCancelled {
			return fmt.Errorf("%w: last attempt is %s", ErrTransition, last.Status)
		}
		if len(task.Attempts) >= MaxTaskAttempts {
			return fmt.Errorf("%w: attempt limit", ErrTransition)
		}
	}
	allowed := map[string]bool{}
	for _, k := range agent.Effects {
		allowed[k] = true
	}
	for _, k := range in.Authority.Effects {
		if !allowed[k] {
			return fmt.Errorf("%w: %s exceeds the Workflow agent effect bound", ErrUnauthorizedEffect, k)
		}
	}
	if _, _, dup := e.FindAttempt(in.AttemptID); dup {
		return ErrTransition
	}
	task.Attempts = append(task.Attempts, Attempt{
		AttemptID: in.AttemptID, Number: len(task.Attempts) + 1, RunnerGeneration: in.Generation,
		Invocation: in.Invocation, Authority: in.Authority, Status: AttemptPrepared, PreparedAt: in.Now.UTC(),
	})
	e.touch(in.Now)
	return e.Validate()
}

func (e *Execution) fencedAttempt(id string, generation uint64, want AttemptStatus) (*Attempt, error) {
	if err := e.fence(generation); err != nil {
		return nil, err
	}
	a, _, ok := e.FindAttempt(id)
	if !ok || a.RunnerGeneration != generation || a.Status != want {
		return nil, ErrTransition
	}
	return a, nil
}

// MarkStarted records the observed process identity right after spawn.
func (e *Execution) MarkStarted(id string, generation uint64, process ProcessIdentity, now time.Time) error {
	a, err := e.fencedAttempt(id, generation, AttemptPrepared)
	if err != nil {
		return err
	}
	at := now.UTC()
	a.Status, a.StartedAt, a.Process = AttemptRunning, &at, &process
	e.touch(now)
	return e.Validate()
}

// MarkSpawnFailed records an observed spawn failure (missing executable,
// vendor rejection before start). Nothing ran, so no effect is uncertain.
func (e *Execution) MarkSpawnFailed(id string, generation uint64, diagnostic string, now time.Time) error {
	a, err := e.fencedAttempt(id, generation, AttemptPrepared)
	if err != nil {
		return err
	}
	at := now.UTC()
	a.Status, a.EndedAt, a.Diagnostic = AttemptFailed, &at, clip(diagnostic)
	e.touch(now)
	return e.Validate()
}

type AttemptResult struct {
	ExitCode   int
	Diagnostic string
	Effects    []EffectFact
	Artifacts  []ArtifactRef
	Now        time.Time
}

// CompleteAttempt commits an observed terminal result. A post-cancel result
// is refused; an effect outside authority is refused; exit zero without every
// declared output is a failure, never success.
func (e *Execution) CompleteAttempt(id string, generation uint64, r AttemptResult) error {
	a, err := e.fencedAttempt(id, generation, AttemptRunning)
	if err != nil {
		return err
	}
	if e.Cancel != nil {
		return ErrCancelled
	}
	allowed := map[string]bool{}
	for _, k := range a.Authority.Effects {
		allowed[k] = true
	}
	for _, f := range r.Effects {
		if !allowed[f.Kind] {
			return fmt.Errorf("%w: %s", ErrUnauthorizedEffect, f.Kind)
		}
	}
	_, ref, _ := e.FindAttempt(id)
	spec := e.Workflow.Stages[e.stageIndex(ref.StageID)]
	at := r.Now.UTC()
	code := r.ExitCode
	a.EndedAt, a.ExitCode, a.Effects, a.Artifacts = &at, &code, r.Effects, r.Artifacts
	a.Diagnostic = clip(r.Diagnostic)
	a.Status = AttemptFailed
	if code == 0 {
		a.Status = AttemptSucceeded
		have := map[string]bool{}
		for _, ref := range r.Artifacts {
			have[ref.Name] = true
		}
		for _, out := range spec.Outputs {
			if !have[out] {
				a.Status, a.Diagnostic = AttemptFailed, "missing_output: "+out
				break
			}
		}
	}
	for _, f := range r.Effects {
		if f.State == EffectUnknown {
			a.Status = AttemptUnknown
		}
	}
	e.touch(r.Now)
	return e.Validate()
}

// MarkUnknown records that the effects of a prepared or running attempt
// cannot be established. It is never followed by an automatic respawn.
func (e *Execution) MarkUnknown(id string, generation uint64, diagnostic string, now time.Time) error {
	if err := e.fence(generation); err != nil {
		return err
	}
	a, _, ok := e.FindAttempt(id)
	if !ok || (a.Status != AttemptPrepared && a.Status != AttemptRunning) {
		return ErrTransition
	}
	at := now.UTC()
	a.Status, a.EndedAt, a.Diagnostic = AttemptUnknown, &at, clip(diagnostic)
	e.touch(now)
	return e.Validate()
}

// MarkCancelled records an observed confirmed stop of a cancelled attempt.
// An unconfirmed stop must use MarkUnknown instead.
func (e *Execution) MarkCancelled(id string, generation uint64, diagnostic string, now time.Time) error {
	if err := e.fence(generation); err != nil {
		return err
	}
	a, _, ok := e.FindAttempt(id)
	if !ok || e.Cancel == nil || (a.Status != AttemptPrepared && a.Status != AttemptRunning) {
		return ErrTransition
	}
	at := now.UTC()
	a.Status, a.EndedAt, a.Diagnostic = AttemptCancelled, &at, clip(diagnostic)
	e.touch(now)
	return e.Validate()
}

// RequestCancel persists cancel intent; it fences any further scheduling.
// Prepared-but-unspawned attempts are cancelled immediately; running ones are
// left for the runner to stop and observe. It needs no runner lease.
func (e *Execution) RequestCancel(actor string, now time.Time) error {
	if !ValidToken(actor) {
		return ErrTransition
	}
	if e.Cancel != nil {
		return nil
	}
	e.Cancel = &CancelIntent{Actor: actor, RequestedAt: now.UTC()}
	at := now.UTC()
	for i := range e.Stages {
		for j := range e.Stages[i].Tasks {
			if last := lastAttempt(e.Stages[i].Tasks[j]); last != nil && last.Status == AttemptPrepared {
				last.Status, last.EndedAt, last.Diagnostic = AttemptCancelled, &at, "cancelled_before_spawn"
			}
		}
	}
	e.touch(now)
	return e.Validate()
}

// ApproveGate records a real human gate decision for a stage awaiting one.
// The actor must be a human identity; agent output never reaches this path.
func (e *Execution) ApproveGate(stageID, actor string, now time.Time) error {
	i := e.stageIndex(stageID)
	if i < 0 || e.Stages[i].Status != StageAwaitingGate || !ValidToken(actor) || len(actor) < 7 || actor[:6] != "human:" {
		return ErrTransition
	}
	e.Stages[i].Gate = &Disposition{Actor: actor, Decision: DispositionApproved, RecordedAt: now.UTC()}
	e.touch(now)
	return e.Validate()
}

// ResolveUnknown records the explicit human abandonment of an unknown
// attempt. Its uncertain effects remain recorded; nothing is replayed.
func (e *Execution) ResolveUnknown(id, actor string, now time.Time) error {
	a, _, ok := e.FindAttempt(id)
	if !ok || !unresolvedUnknown(*a) || !ValidToken(actor) || len(actor) < 7 || actor[:6] != "human:" {
		return ErrTransition
	}
	a.Resolution = &Disposition{Actor: actor, Decision: DispositionAbandoned, RecordedAt: now.UTC()}
	e.touch(now)
	return e.Validate()
}

func clip(s string) string {
	if len(s) <= MaxDiagnosticBytes {
		return s
	}
	cut := MaxDiagnosticBytes
	for cut > 0 && s[cut]&0xC0 == 0x80 {
		cut--
	}
	return s[:cut]
}
