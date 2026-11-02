package stageexecution

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/rgomids/axiom/internal/executiongraph"
	"github.com/rgomids/axiom/internal/workflow"
)

// Engine is the one fenced run loop for new Executions (ADR-0037 §3). One or
// many agents use the same loop, store and codec; there is no Graph/Single
// branch, Stage Plan, Validator Policy, preview/apply or admission catalogue.
//
// For each eligible task it resolves the invocation, persists the prepared
// attempt under the runner fence, spawns only after that commit, publishes the
// observed process identity before releasing stdin, and commits the observed
// terminal result. A crash anywhere between preparation and terminal commit
// leaves the attempt to the next runner, which records it unknown and never
// respawns it.
type Engine struct {
	Store    ExecutionStore
	Runner   ObservedRunner
	Launcher Launcher
	// Resolve returns the frozen invocation and exact authority for a task.
	// Its refusal (missing/conflicting selection, denied Runtime) prevents
	// preparation; nothing is recorded or spawned.
	Resolve func(context.Context, workflow.Execution, workflow.TaskRef) (workflow.Invocation, workflow.AttemptAuthority, error)
	// Admit, when set, rechecks the current Project operational admission and
	// Repository binding immediately before the spawn of a prepared attempt, so
	// revocation after Resolve still prevents the process. Its refusal records
	// the prepared attempt as spawn-failed (nothing ran) and stops the run.
	Admit func(context.Context, workflow.Execution) error
	NewID func() (string, error)
	Now   func() time.Time
	// CancelPoll bounds how quickly a persisted cancel intent reaches a
	// running attempt; zero uses 50ms.
	CancelPoll time.Duration
}

// ExecutionStore is the single Execution owner (local.ExecutionStore):
// lease-free status reads, the exclusive runner fence and fenced revision-CAS
// updates. A nil fence is allowed only for changes that keep the generation.
type ExecutionStore interface {
	Load(ctx context.Context, projectID, executionID string) (workflow.Execution, error)
	AcquireFence(ctx context.Context, projectID, executionID string) (workflow.RunnerFence, error)
	UpdateFenced(ctx context.Context, projectID, executionID string, fence workflow.RunnerFence, mutate func(*workflow.Execution) error) (workflow.Execution, error)
}

// ObservedRunner is the ADR-0028 process boundary (OSProcessRunner): own
// process group, PID/start/group identity, confirmed stop, honest unknowns.
type ObservedRunner interface {
	RunObserved(context.Context, executiongraph.Invocation, func(*executiongraph.ProcessCorrelation) error) executiongraph.ProcessResult
}

type LaunchRequest struct {
	PrepareCommand func(string) error
	StartCommand   func(string, workflow.ProcessIdentity) error
	FinishCommand  func(string, int) (workflow.CommandResult, error)
	Execution      workflow.Execution
	Ref            workflow.TaskRef
	Stage          workflow.StageSpec
	Agent          workflow.AgentSpec
	Attempt        workflow.Attempt
}

// Launcher maps a frozen attempt to its concrete process and reads its
// declared outputs. Real Runtime adapters must apply
// runtimeadapter.CheckFrozenSelectors in Build; Build errors are observed
// spawn failures, not eligibility predictions.
// Build and Collect may run concurrently for distinct attempts; implementations
// must scope mutable state by attempt. Command callbacks return their mutations
// to the principal writer.
type Launcher interface {
	Build(context.Context, LaunchRequest) (executiongraph.Invocation, error)
	Collect(context.Context, LaunchRequest, executiongraph.ProcessResult) (Outcome, error)
}

type Outcome struct {
	Commands   []workflow.CommandResult
	Artifacts  []workflow.ArtifactRef
	Effects    []workflow.EffectFact
	Diagnostic string
}

var ErrRunnerActive = workflow.ErrRunnerHeld

// ErrOrphanLiveness refuses an abandonment while a recorded process of the
// Execution is alive or its liveness cannot be observed; the operator must
// stop it outside Axiom first. Nothing is written or signalled.
var ErrOrphanLiveness = errors.New("stop_orphan_then_abandon")

// terminalCommitBound bounds the fenced commits of an attempt. They run
// detached from the caller's cancellation so an interrupted runner still
// records its observed stop instead of leaving a phantom running attempt.
const terminalCommitBound = 10 * time.Second

// interruptActor records a runner interrupted by its operator (SIGINT,
// SIGTERM, SIGHUP) as the cancel intent it observed and acted on.
const interruptActor = "runner:interrupted"

// RunOptions selects an explicit retry of one task; zero means "progress
// eligible work".
type RunOptions struct {
	Retry *workflow.TaskRef
	// NewSelection is the explicit authorization to freeze the freshly
	// resolved invocation for the retry instead of the previous tuple. Without
	// it a retry reuses the frozen invocation and its provenance.
	NewSelection         bool
	WorkspaceDisposition string
}

func (e Engine) now() time.Time {
	if e.Now != nil {
		return e.Now().UTC()
	}
	return time.Now().UTC()
}

// Claim acquires the runner lease and records its generation. Attempts left
// prepared or running by an older runner have no correlated terminal
// evidence: they become unknown, are never adopted and never respawned.
func (e Engine) Claim(ctx context.Context, projectID, executionID string) (workflow.RunnerFence, workflow.Execution, error) {
	lease, err := e.Store.AcquireFence(ctx, projectID, executionID)
	if err != nil {
		return nil, workflow.Execution{}, err
	}
	now := e.now()
	gen := lease.Generation()
	current, err := e.Store.UpdateFenced(ctx, projectID, executionID, lease, func(x *workflow.Execution) error {
		if err := x.ClaimRunner(gen, now); err != nil {
			return err
		}
		for _, id := range orphanedAttempts(*x, gen) {
			if err := x.MarkUnknown(id, gen, "runner_lost_before_terminal_evidence", now); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		lease.Close()
		return nil, workflow.Execution{}, err
	}
	return lease, current, nil
}

func orphanedAttempts(x workflow.Execution, generation uint64) []string {
	var ids []string
	for _, s := range x.Stages {
		for _, t := range s.Tasks {
			for _, a := range t.Attempts {
				if (a.Status == workflow.AttemptPrepared || a.Status == workflow.AttemptRunning) && a.RunnerGeneration < generation {
					ids = append(ids, a.AttemptID)
				}
			}
		}
	}
	return ids
}

// Run progresses eligible work through one principal writer. Independent ready
// tasks share the same attempt lifecycle and codec; only process observation and
// output collection run concurrently. Every worker commit returns to this loop.
func (e Engine) Run(ctx context.Context, projectID, executionID string, opts RunOptions) (workflow.Execution, error) {
	lease, current, err := e.Claim(ctx, projectID, executionID)
	if err != nil {
		return current, err
	}
	defer lease.Close()
	return e.runReady(ctx, lease, current, opts)
}

func (e Engine) prepareAttempt(ctx context.Context, lease workflow.RunnerFence, current workflow.Execution, ref workflow.TaskRef, isRetry, newSelection bool, disposition string) (LaunchRequest, error) {
	if !isRetry {
		disposition = ""
	}
	projectID, executionID, gen := current.ProjectID, current.ExecutionID, lease.Generation()
	invocation, authority, err := e.Resolve(ctx, current, ref)
	if err != nil {
		return LaunchRequest{}, err
	}
	if isRetry {
		// Resolve still rechecks current target/effect restrictions. An ordinary
		// retry does not authorize selecting changed local defaults, however.
		// Freeze the previous tuple again, including its original provenance.
		// An explicit new selection keeps the fresh tuple as a new attempt; no
		// retry may retarget the repository.
		for _, stage := range current.Stages {
			for _, task := range stage.Tasks {
				if stage.StageID == ref.StageID && task.AgentID == ref.AgentID && len(task.Attempts) != 0 {
					previous := task.Attempts[len(task.Attempts)-1]
					if authority.RepositoryKey != previous.Authority.RepositoryKey {
						return LaunchRequest{}, workflow.ErrTransition
					}
					if newSelection {
						continue
					}
					if invocation.Runtime != previous.Invocation.Runtime || invocation.Executable != previous.Invocation.Executable {
						return LaunchRequest{}, workflow.ErrTransition
					}
					invocation = previous.Invocation
				}
			}
		}
	}
	attemptID, err := e.NewID()
	if err != nil {
		return LaunchRequest{}, err
	}
	update := e.commit(ctx, projectID, executionID, lease)
	prepared, err := update(func(x *workflow.Execution) error {
		return x.PrepareAttempt(workflow.PrepareInput{Ref: ref, AttemptID: attemptID, Generation: gen, Invocation: invocation, Authority: authority, Now: e.now(), Retry: isRetry, NewSelection: newSelection, WorkspaceDisposition: disposition})
	})
	if err != nil {
		return LaunchRequest{}, err
	}
	return e.request(prepared, ref, attemptID), nil
}

// commit keeps caller correlation and the fence, but detaches cancellation so
// an interrupted runner can still retain its observed terminal facts.
func (e Engine) commit(ctx context.Context, projectID, executionID string, lease workflow.RunnerFence) func(func(*workflow.Execution) error) (workflow.Execution, error) {
	return func(mutate func(*workflow.Execution) error) (workflow.Execution, error) {
		commitCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), terminalCommitBound)
		defer cancel()
		return e.Store.UpdateFenced(commitCtx, projectID, executionID, lease, mutate)
	}
}

func (e Engine) runAttempt(ctx context.Context, lease workflow.RunnerFence, request LaunchRequest) (workflow.Execution, error) {
	prepared, attemptID := request.Execution, request.Attempt.AttemptID
	projectID, executionID, gen := prepared.ProjectID, prepared.ExecutionID, lease.Generation()
	update := e.commit(ctx, projectID, executionID, lease)

	cancelBeforeSpawn := func() (workflow.Execution, error) {
		return update(func(x *workflow.Execution) error {
			if err := recordInterrupt(ctx, x, e.now()); err != nil {
				return err
			}
			return x.MarkCancelled(attemptID, gen, "cancelled_before_spawn", e.now())
		})
	}
	// Only the runner knows nothing was spawned yet; a cancel arriving after
	// this check is caught by the watcher and stops the owned group.
	if ctx.Err() != nil || e.cancelPersisted(ctx, projectID, executionID) {
		return cancelBeforeSpawn()
	}
	built, err := e.Launcher.Build(ctx, request)
	if err != nil && ctx.Err() == nil {
		return update(func(x *workflow.Execution) error {
			return x.MarkSpawnFailed(attemptID, gen, "launch_refused: check Runtime configuration and frozen selectors", e.now())
		})
	}
	// Build may read configuration or prepare local output paths. Cancel intent
	// arriving during that work still prevents any process from being spawned.
	if ctx.Err() != nil || e.cancelPersisted(ctx, projectID, executionID) {
		return cancelBeforeSpawn()
	}
	if e.Admit != nil {
		if err := e.Admit(ctx, prepared); err != nil {
			refused, commitErr := update(func(x *workflow.Execution) error {
				return x.MarkSpawnFailed(attemptID, gen, "admission_revoked_before_spawn", e.now())
			})
			if commitErr != nil {
				return refused, commitErr
			}
			return refused, err
		}
	}
	stage := request.Stage
	timeout := stage.TimeoutSeconds
	if request.Agent.TimeoutSeconds != 0 {
		timeout = request.Agent.TimeoutSeconds
	}
	runCtx, stop := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer stop()
	cancelled := e.watchCancel(runCtx, stop, projectID, executionID)
	startedCommitted := false
	result := e.Runner.RunObserved(runCtx, built, func(pc *executiongraph.ProcessCorrelation) error {
		_, err := update(func(x *workflow.Execution) error {
			return x.MarkStarted(attemptID, gen, workflow.ProcessIdentity{PID: pc.PID, StartIdentity: pc.StartIdentity, GroupIdentity: pc.GroupIdentity}, e.now())
		})
		startedCommitted = err == nil
		return err
	})
	stop()
	now := e.now()
	// An operator interrupt cancels ctx; the runner already stopped its own
	// group through RunObserved and records that stop like a cancel.
	cancelRequested := cancelled() || ctx.Err() != nil || e.cancelPersisted(ctx, projectID, executionID)
	// Collection may run explicitly authorized checks. Keep it outside the
	// aggregate lock so cancellation can be persisted while a check runs.
	var outcome Outcome
	var collectErr error
	if startedCommitted && !cancelRequested && !result.OutcomeUnknown && !result.EffectAmbiguous && !result.TimedOut {
		collectCtx, done := context.WithCancel(ctx)
		checkCancelled := e.watchCancel(collectCtx, done, projectID, executionID)
		request.PrepareCommand = func(id string) error {
			_, err := update(func(x *workflow.Execution) error { return x.PrepareCommand(attemptID, gen, id, e.now()) })
			return err
		}
		request.StartCommand = func(id string, p workflow.ProcessIdentity) error {
			_, err := update(func(x *workflow.Execution) error { return x.StartCommand(attemptID, gen, id, p, e.now()) })
			return err
		}
		request.FinishCommand = func(id string, code int) (workflow.CommandResult, error) {
			x, err := update(func(x *workflow.Execution) error { return x.FinishCommand(attemptID, gen, id, code, e.now()) })
			if err != nil {
				return workflow.CommandResult{}, err
			}
			a, _, ok := x.FindAttempt(attemptID)
			if !ok || len(a.Commands) == 0 {
				return workflow.CommandResult{}, workflow.ErrTransition
			}
			return a.Commands[len(a.Commands)-1], nil
		}
		outcome, collectErr = e.Launcher.Collect(collectCtx, request, result)
		done()
		cancelRequested = checkCancelled() || ctx.Err() != nil || e.cancelPersisted(ctx, projectID, executionID)
	}
	return update(func(x *workflow.Execution) error {
		switch {
		case result.SpawnFailed && !startedCommitted:
			if result.FailureCode == "attempt_cancelled" && (cancelRequested || x.Cancel != nil) {
				// Interrupted before the start: nothing ran.
				if err := recordInterrupt(ctx, x, now); err != nil {
					return err
				}
				return x.MarkCancelled(attemptID, gen, "cancelled_before_spawn", now)
			}
			return x.MarkSpawnFailed(attemptID, gen, "spawn_failed: "+result.FailureCode, now)
		case !startedCommitted:
			// Spawned but identity was never durably published; effects unknown.
			return x.MarkUnknown(attemptID, gen, "process_identity_not_committed", now)
		// The reloaded aggregate is authoritative: a cancel persisted after the
		// last poll still refuses the result instead of leaving it running.
		case cancelRequested || x.Cancel != nil:
			if err := recordInterrupt(ctx, x, now); err != nil {
				return err
			}
			// The agent result says nothing about a validation command that was
			// started during collection; its stop must be confirmed separately.
			if commandEndUnconfirmed(*x, attemptID, collectErr) {
				return x.MarkUnknown(attemptID, gen, "cancel_command_stop_unconfirmed", now)
			}
			if result.Stopped && !result.OutcomeUnknown && !result.EffectAmbiguous {
				return x.MarkCancelled(attemptID, gen, "group_stop_confirmed", now)
			}
			if !result.OutcomeUnknown && !result.EffectAmbiguous && !result.Stopped {
				// The process exited by itself before the stop; its late result is
				// refused and recorded as cancelled, never as success.
				return x.MarkCancelled(attemptID, gen, "exited_after_cancel_result_refused", now)
			}
			return x.MarkUnknown(attemptID, gen, "cancel_stop_unconfirmed", now)
		case result.OutcomeUnknown || result.EffectAmbiguous:
			return x.MarkUnknown(attemptID, gen, "process_outcome_unknown", now)
		case result.TimedOut:
			return x.CompleteAttempt(attemptID, gen, workflow.AttemptResult{ExitCode: -1, Diagnostic: "attempt_timeout", Now: now})
		}
		if collectErr != nil {
			return x.MarkUnknown(attemptID, gen, "output_collection_failed", now)
		}
		err = x.CompleteAttempt(attemptID, gen, workflow.AttemptResult{ExitCode: result.ExitCode, Diagnostic: outcome.Diagnostic, Effects: outcome.Effects, Artifacts: outcome.Artifacts, Commands: outcome.Commands, Now: now})
		if errors.Is(err, workflow.ErrUnauthorizedEffect) {
			return x.MarkUnknown(attemptID, gen, "unauthorized_effect_reported", now)
		}
		return err
	})
}

// recordInterrupt persists the runner's own cancel intent when the operator
// interrupted it and no intent exists yet, so the observed stop is recorded
// as cancelled (or unknown when unconfirmed) and nothing is scheduled after it.
func recordInterrupt(ctx context.Context, x *workflow.Execution, now time.Time) error {
	if ctx.Err() == nil || x.Cancel != nil {
		return nil
	}
	return x.RequestCancel(interruptActor, now)
}

// commandEndUnconfirmed reports a started validation command without a
// committed terminal result whose end the launcher did not confirm.
func commandEndUnconfirmed(x workflow.Execution, attemptID string, collectErr error) bool {
	a, _, ok := x.FindAttempt(attemptID)
	if !ok {
		return true
	}
	for _, c := range a.Commands {
		if c.Status == "running" && !errors.Is(collectErr, ErrCommandTerminated) {
			return true
		}
	}
	return false
}

func (e Engine) request(x workflow.Execution, ref workflow.TaskRef, attemptID string) LaunchRequest {
	r := LaunchRequest{Execution: x, Ref: ref}
	for _, s := range x.Workflow.Stages {
		if s.ID != ref.StageID {
			continue
		}
		r.Stage = s
		for _, a := range s.Agents {
			if a.ID == ref.AgentID {
				r.Agent = a
			}
		}
	}
	if a, _, ok := x.FindAttempt(attemptID); ok {
		r.Attempt = *a
	}
	return r
}

// watchCancel cancels the attempt context when a persisted cancel intent
// appears. Reads take no lease.
func (e Engine) watchCancel(ctx context.Context, stop context.CancelFunc, projectID, executionID string) func() bool {
	poll := e.CancelPoll
	if poll == 0 {
		poll = 50 * time.Millisecond
	}
	seen, finished := make(chan struct{}), make(chan struct{})
	go func() {
		defer close(finished)
		ticker := time.NewTicker(poll)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if e.cancelPersisted(ctx, projectID, executionID) {
					close(seen)
					stop()
					return
				}
			}
		}
	}()
	// The check stops and joins the poller so no store read outlives the
	// attempt (stop is idempotent; callers have already cancelled).
	return func() bool {
		stop()
		<-finished
		select {
		case <-seen:
			return true
		default:
			return false
		}
	}
}

func (e Engine) cancelPersisted(ctx context.Context, projectID, executionID string) bool {
	x, err := e.Store.Load(context.WithoutCancel(ctx), projectID, executionID)
	return err == nil && x.Cancel != nil
}

// Cancel persists the intent without a runner lease; the active runner stops
// only its verified own process group and records the observed outcome.
func (e Engine) Cancel(ctx context.Context, projectID, executionID, actor string) (workflow.Execution, error) {
	now := e.now()
	return e.Store.UpdateFenced(ctx, projectID, executionID, nil, func(x *workflow.Execution) error {
		return x.RequestCancel(actor, now)
	})
}

// ApproveGate records a real human gate decision; it spawns, signals or
// replays nothing and needs no runner lease.
func (e Engine) ApproveGate(ctx context.Context, projectID, executionID, stageID, actor string) (workflow.Execution, error) {
	now := e.now()
	return e.Store.UpdateFenced(ctx, projectID, executionID, nil, func(x *workflow.Execution) error {
		return x.ApproveGate(stageID, actor, now)
	})
}

// Abandon records the human abandonment of an unknown attempt, which is what
// permits an explicit retry. Abandonment must never let a retry run beside a
// surviving writer: it is refused while any runner lease is held, and every
// recorded attempt and unfinished command identity of the Execution is
// re-observed first. A live process or a failed observation refuses with
// ErrOrphanLiveness and writes nothing; a gone process, a reused PID or no
// recorded identity is reported and allowed. Nothing is ever signalled.
func (e Engine) Abandon(ctx context.Context, projectID, executionID, attemptID, actor string) (workflow.Execution, []ReconcileFact, error) {
	// The lease is only held, never claimed: no generation is recorded, and no
	// runner can start between the observation and the disposition.
	lease, err := e.Store.AcquireFence(ctx, projectID, executionID)
	if err != nil {
		return workflow.Execution{}, nil, err
	}
	defer lease.Close()
	current, err := e.Store.Load(ctx, projectID, executionID)
	if err != nil {
		return current, nil, err
	}
	if a, _, ok := current.FindAttempt(attemptID); !ok || a.Status != workflow.AttemptUnknown || a.Resolution != nil {
		return current, nil, workflow.ErrTransition
	}
	facts := observeLiveness(current)
	for _, f := range facts {
		// Command facts carry a "command:<id>:" prefix.
		if strings.HasSuffix(f.Observation, "process_alive_effects_unknown") || strings.HasSuffix(f.Observation, "observation_failed") {
			return current, facts, ErrOrphanLiveness
		}
	}
	now := e.now()
	// Lease-free CAS on the observed revision: any concurrent change refuses.
	next, err := e.Store.UpdateFenced(ctx, projectID, executionID, nil, func(x *workflow.Execution) error {
		if x.Revision != current.Revision {
			return workflow.ErrTransition
		}
		return x.ResolveUnknown(attemptID, actor, now)
	})
	return next, facts, err
}

// observeLiveness re-observes every attempt whose process may still exist
// (prepared, running or unknown) and every unfinished command of each.
func observeLiveness(x workflow.Execution) []ReconcileFact {
	var facts []ReconcileFact
	for _, s := range x.Stages {
		for _, t := range s.Tasks {
			for _, a := range t.Attempts {
				if a.Status != workflow.AttemptPrepared && a.Status != workflow.AttemptRunning && a.Status != workflow.AttemptUnknown {
					continue
				}
				facts = append(facts, ReconcileFact{AttemptID: a.AttemptID, Observation: observeRecorded(a.Process)})
				for _, command := range a.Commands {
					if command.Status != "completed" {
						facts = append(facts, ReconcileFact{AttemptID: a.AttemptID, Observation: "command:" + command.ID + ":" + observeRecorded(command.Process)})
					}
				}
			}
		}
	}
	return facts
}

// Ingest accepts one candidate message for a running attempt through its
// launcher-issued context. Messages never change authority or status.
func (e Engine) Ingest(ctx context.Context, projectID string, ingress workflow.IngressContext, candidate workflow.MessageCandidate) (workflow.Execution, error) {
	now := e.now()
	return e.Store.UpdateFenced(ctx, projectID, ingress.ExecutionID, nil, func(x *workflow.Execution) error {
		return x.AcceptMessage(ingress, candidate, now)
	})
}

// ReconcileFact reports what is observable about an attempt left by a lost
// runner. Reconcile never signals a process and never promotes success.
type ReconcileFact struct {
	AttemptID   string
	Observation string
}

// Reconcile claims the runner (recording orphaned attempts unknown) and
// observes recorded process identities without signalling anything. A
// reused PID is reported, never touched.
func (e Engine) Reconcile(ctx context.Context, projectID, executionID string) (workflow.Execution, []ReconcileFact, error) {
	before, err := e.Store.Load(ctx, projectID, executionID)
	if err != nil {
		return before, nil, err
	}
	lease, after, err := e.Claim(ctx, projectID, executionID)
	if err != nil {
		return before, nil, err
	}
	defer lease.Close()
	var facts []ReconcileFact
	for _, id := range orphanedAttempts(before, lease.Generation()) {
		a, _, _ := before.FindAttempt(id)
		facts = append(facts, ReconcileFact{AttemptID: id, Observation: observeRecorded(a.Process)})
		for _, command := range a.Commands {
			if command.Status != "completed" {
				facts = append(facts, ReconcileFact{AttemptID: id, Observation: "command:" + command.ID + ":" + observeRecorded(command.Process)})
			}
		}
	}
	return after, facts, nil
}

// observeRecordedProcess is the read-only platform observation; tests replace
// it only to inject an observation failure.
var observeRecordedProcess = executiongraph.ObserveRecordedProcess

func observeRecorded(p *workflow.ProcessIdentity) string {
	if p == nil {
		return "no_process_identity_recorded"
	}
	current, err := observeRecordedProcess(executiongraph.ProcessCorrelation{PID: p.PID, StartIdentity: p.StartIdentity, GroupIdentity: p.GroupIdentity})
	switch {
	case err != nil:
		return "observation_failed"
	case current == nil:
		return "process_gone_effects_unknown"
	case current.StartIdentity != p.StartIdentity || current.GroupIdentity != p.GroupIdentity:
		return "pid_reused_not_signalled"
	}
	return "process_alive_effects_unknown"
}
