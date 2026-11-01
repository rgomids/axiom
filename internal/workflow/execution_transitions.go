package workflow

import (
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
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
	e.refreshStageStatuses()
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

// Refresh in dependency order; definitions need not list predecessors first.
func (e *Execution) refreshStageStatuses() {
	visited := map[int]bool{}
	var visit func(int)
	visit = func(i int) {
		if visited[i] {
			return
		}
		visited[i] = true
		for _, dep := range e.Workflow.Stages[i].DependsOn {
			visit(e.stageIndex(dep))
		}
		e.Stages[i].Status = e.derivedStageStatus(i)
	}
	for i := range e.Stages {
		visit(i)
	}
}

func (e *Execution) derivedStageStatus(i int) StageStatus {
	run := e.Stages[i]
	if e.Workflow.Stages[i].BeforeGate && run.BeforeApproval == nil {
		for _, dep := range e.Workflow.Stages[i].DependsOn {
			if e.Stages[e.stageIndex(dep)].Status != StageCompleted {
				return StagePending
			}
		}
		return StageAwaitingGate
	}
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
		if e.Workflow.Stages[i].HumanGate && run.Gate == nil || e.Workflow.Stages[i].ReviewRequired && run.Review == nil {
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
		if spec.BeforeGate && e.Stages[i].BeforeApproval == nil {
			ready = false
		}
		if !ready {
			continue
		}
		for _, t := range e.Stages[i].Tasks {
			agentReady := true
			for _, agent := range spec.Agents {
				if agent.ID == t.AgentID {
					for _, dep := range agent.DependsOn {
						for _, dependency := range e.Stages[i].Tasks {
							if dependency.AgentID == dep {
								last := lastAttempt(dependency)
								if last == nil || last.Status != AttemptSucceeded {
									agentReady = false
								}
							}
						}
					}
				}
			}
			if len(t.Attempts) == 0 && agentReady {
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
	// NewSelection marks an explicitly authorized new invocation selection
	// for a retry. Only with it may the frozen tuple differ from the previous
	// attempt; the previous attempt and its provenance stay unchanged.
	NewSelection         bool
	WorkspaceDisposition string
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
	case in.NewSelection && !in.Retry:
		return fmt.Errorf("%w: a new selection applies only to an explicit retry", ErrTransition)
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
		// Ordinary retry cannot pick up changed machine-local defaults. A new
		// selection requires a separate explicit authorization contract; Retry
		// alone grants no permission to replace the frozen invocation.
		if !in.NewSelection && !reflect.DeepEqual(in.Invocation, last.Invocation) {
			return fmt.Errorf("%w: retry requires the frozen invocation", ErrTransition)
		}
		if len(task.Attempts) >= agent.AttemptLimit() {
			return fmt.Errorf("%w: attempt limit", ErrTransition)
		}
	}
	if in.WorkspaceDisposition != "" && (!in.Retry || in.WorkspaceDisposition != "preserve") {
		return ErrTransition
	}
	if in.Retry && last != nil && slices.Contains(last.Authority.Effects, "workspace_write") && in.WorkspaceDisposition != "preserve" {
		return ErrWorkspaceDispositionRequired
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
	if len(agent.Runtimes) != 0 && !slices.Contains(agent.Runtimes, in.Invocation.Runtime) {
		return fmt.Errorf("%w: Runtime %s is outside the Workflow agent restriction", ErrUnauthorizedEffect, in.Invocation.Runtime)
	}
	if _, _, dup := e.FindAttempt(in.AttemptID); dup {
		return ErrTransition
	}
	task.Attempts = append(task.Attempts, Attempt{
		AttemptID: in.AttemptID, Number: len(task.Attempts) + 1, RunnerGeneration: in.Generation,
		Invocation: in.Invocation, Authority: in.Authority, WorkspaceDisposition: in.WorkspaceDisposition, Status: AttemptPrepared, PreparedAt: in.Now.UTC(),
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
	Commands   []CommandResult
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
	_, agent, spec, _ := e.task(ref)
	at := r.Now.UTC()
	code := r.ExitCode
	a.EndedAt, a.ExitCode, a.Effects, a.Artifacts = &at, &code, r.Effects, r.Artifacts
	a.Diagnostic = clip(r.Diagnostic)
	if !reflect.DeepEqual(a.Commands, r.Commands) {
		return ErrTransition
	}
	a.Status = AttemptFailed
	if code == 0 {
		a.Status = AttemptSucceeded
		if len(a.Commands) != len(spec.Commands) {
			a.Status = AttemptFailed
			a.Diagnostic = "missing_command_validation"
		}
		for i, result := range a.Commands {
			if i >= len(spec.Commands) {
				return ErrTransition
			}
			wire, _ := json.Marshal(spec.Commands[i])
			if result.ID != spec.Commands[i].ID || result.ArgvDigest != DigestText(wire) || (result.ExitCode != 0 || result.Status != "completed") {
				a.Status = AttemptFailed
				a.Diagnostic = "command_validation_failed"
			}
		}
		have := map[string]bool{}
		for _, ref := range r.Artifacts {
			have[ref.Name] = true
		}
		required := spec.Outputs
		if agent.OutputScope {
			required = nil
			for _, name := range agent.Outputs {
				if slices.Contains(spec.Outputs, name) {
					required = append(required, name)
				}
			}
		}
		for _, out := range required {
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

// RequestCancel persists cancel intent; it fences any further scheduling and
// needs no runner lease. It never rewrites attempts: only the runner knows
// whether a prepared attempt was spawned, so the runner records
// cancelled-before-spawn, a confirmed stop, or unknown; a lost runner's
// attempts become unknown at the next claim.
func (e *Execution) RequestCancel(actor string, now time.Time) error {
	if !ValidToken(actor) {
		return ErrTransition
	}
	if e.Cancel != nil {
		return nil
	}
	e.Cancel = &CancelIntent{Actor: actor, RequestedAt: now.UTC()}
	e.touch(now)
	return e.Validate()
}

// ApproveGate records a real human gate decision for a stage awaiting one.
// The actor must be a human identity; agent output never reaches this path.
func (e *Execution) ApproveGate(stageID, actor string, now time.Time) error {
	i := e.stageIndex(stageID)
	if e.Cancel != nil || e.Status == RunRecoveryRequired || i < 0 || e.Stages[i].Status != StageAwaitingGate || !ValidToken(actor) || len(actor) < 7 || actor[:6] != "human:" {
		return ErrTransition
	}
	d := &Disposition{Actor: actor, Decision: DispositionApproved, RecordedAt: now.UTC()}
	if e.Workflow.Stages[i].BeforeGate && e.Stages[i].BeforeApproval == nil {
		e.Stages[i].BeforeApproval = d
	} else if e.Workflow.Stages[i].HumanGate && e.Stages[i].Gate == nil {
		e.Stages[i].Gate = d
	} else {
		return ErrTransition
	}
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

func humanActor(actor string) bool {
	return ValidToken(actor) && len(actor) > 6 && actor[:6] == "human:"
}
func humanDisposition(d Disposition) bool {
	return humanActor(d.Actor) && d.Decision == DispositionApproved && !d.RecordedAt.IsZero()
}

// StageResultDigest binds review to the complete latest successful result set.
// Gate decisions cannot substitute for this result-bound review.
func (e Execution) StageResultDigest(stageID string) (string, error) {
	i := e.stageIndex(stageID)
	if i < 0 {
		return "", ErrTransition
	}
	type result struct {
		Agent   string
		Attempt Attempt
	}
	var results []result
	for _, task := range e.Stages[i].Tasks {
		a := lastAttempt(task)
		if a == nil || a.Status != AttemptSucceeded {
			return "", ErrTransition
		}
		results = append(results, result{task.AgentID, *a})
	}
	wire, err := json.Marshal(struct {
		Execution, Definition, Stage string
		Results                      []result
	}{e.ExecutionID, e.Workflow.Digest, stageID, results})
	if err != nil {
		return "", ErrTransition
	}
	return DigestText(wire), nil
}

// ReviewStage accepts only an explicit human result approval with retained Evidence.
func (e *Execution) ReviewStage(stageID, actor, resultDigest string, evidence ArtifactRef, now time.Time) error {
	i := e.stageIndex(stageID)
	if i < 0 || e.Cancel != nil || e.Status == RunRecoveryRequired || !e.Workflow.Stages[i].ReviewRequired || e.Stages[i].Review != nil || !humanActor(actor) {
		return ErrTransition
	}
	digest, err := e.StageResultDigest(stageID)
	if err != nil || digest != resultDigest || evidence.RetainedID == "" || evidence.RepositoryKey != e.WorkItem.RepositoryKey || !validPrefixedDigest(evidence.Digest) {
		return ErrTransition
	}
	e.Stages[i].Review = &ResultReview{Actor: actor, ResultDigest: digest, Evidence: evidence, RecordedAt: now.UTC()}
	e.touch(now)
	return e.Validate()
}

// PrepareCommand records intent before the command subprocess can exist.
func (e *Execution) PrepareCommand(attemptID string, generation uint64, id string, now time.Time) error {
	a, err := e.fencedAttempt(attemptID, generation, AttemptRunning)
	if err != nil {
		return err
	}
	if e.Cancel != nil {
		return ErrCancelled
	}
	_, ref, _ := e.FindAttempt(attemptID)
	_, _, spec, _ := e.task(ref)
	index := len(a.Commands)
	if index >= len(spec.Commands) || spec.Commands[index].ID != id {
		return ErrTransition
	}
	for _, effect := range spec.Commands[index].Effects {
		if !slices.Contains(a.Authority.Effects, effect) {
			return ErrUnauthorizedEffect
		}
	}
	if index > 0 && (a.Commands[index-1].Status != "completed" || a.Commands[index-1].ExitCode != 0) {
		return ErrTransition
	}
	wire, _ := json.Marshal(spec.Commands[index])
	a.Commands = append(a.Commands, CommandResult{ID: id, ArgvDigest: DigestText(wire), Status: "prepared", PreparedAt: now.UTC()})
	e.touch(now)
	return e.Validate()
}
func (e *Execution) StartCommand(attemptID string, generation uint64, id string, process ProcessIdentity, now time.Time) error {
	a, err := e.fencedAttempt(attemptID, generation, AttemptRunning)
	if err != nil {
		return err
	}
	if e.Cancel != nil {
		return ErrCancelled
	}
	if len(a.Commands) == 0 {
		return ErrTransition
	}
	c := &a.Commands[len(a.Commands)-1]
	if c.ID != id || c.Status != "prepared" {
		return ErrTransition
	}
	at := now.UTC()
	c.Process = &process
	c.StartedAt = &at
	c.Status = "running"
	e.touch(now)
	return e.Validate()
}
func (e *Execution) FinishCommand(attemptID string, generation uint64, id string, code int, now time.Time) error {
	a, err := e.fencedAttempt(attemptID, generation, AttemptRunning)
	if err != nil {
		return err
	}
	if e.Cancel != nil {
		return ErrCancelled
	}
	if len(a.Commands) == 0 {
		return ErrTransition
	}
	c := &a.Commands[len(a.Commands)-1]
	if c.ID != id || c.Status != "running" {
		return ErrTransition
	}
	at := now.UTC()
	c.EndedAt = &at
	c.ExitCode = code
	c.Status = "completed"
	e.touch(now)
	return e.Validate()
}
