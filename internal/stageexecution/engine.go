package stageexecution

import (
	"context"
	"errors"
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
	NewID   func() (string, error)
	Now     func() time.Time
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
	Execution workflow.Execution
	Ref       workflow.TaskRef
	Stage     workflow.StageSpec
	Agent     workflow.AgentSpec
	Attempt   workflow.Attempt
}

// Launcher maps a frozen attempt to its concrete process and reads its
// declared outputs. Real Runtime adapters must apply
// runtimeadapter.CheckFrozenSelectors in Build; Build errors are observed
// spawn failures, not eligibility predictions.
type Launcher interface {
	Build(context.Context, LaunchRequest) (executiongraph.Invocation, error)
	Collect(context.Context, LaunchRequest, executiongraph.ProcessResult) (Outcome, error)
}

type Outcome struct {
	Artifacts  []workflow.ArtifactRef
	Effects    []workflow.EffectFact
	Diagnostic string
}

var ErrRunnerActive = workflow.ErrRunnerHeld

// RunOptions selects an explicit retry of one task; zero means "progress
// eligible work".
type RunOptions struct {
	Retry *workflow.TaskRef
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

// Run progresses eligible sequential work until nothing is eligible: done,
// failed, cancelled, awaiting a real human gate or blocked by an unknown
// effect. It returns the last committed state.
func (e Engine) Run(ctx context.Context, projectID, executionID string, opts RunOptions) (workflow.Execution, error) {
	lease, current, err := e.Claim(ctx, projectID, executionID)
	if err != nil {
		return current, err
	}
	defer lease.Close()
	retry := opts.Retry
	for {
		if err := ctx.Err(); err != nil {
			return current, err
		}
		var ref workflow.TaskRef
		if retry != nil {
			ref = *retry
		} else {
			ready := current.Ready()
			if len(ready) == 0 {
				return current, nil
			}
			ref = ready[0]
		}
		next, err := e.attempt(ctx, lease, current, ref, retry != nil)
		retry = nil
		if err != nil {
			return next, err
		}
		current = next
	}
}

func (e Engine) attempt(ctx context.Context, lease workflow.RunnerFence, current workflow.Execution, ref workflow.TaskRef, isRetry bool) (workflow.Execution, error) {
	projectID, executionID, gen := current.ProjectID, current.ExecutionID, lease.Generation()
	invocation, authority, err := e.Resolve(ctx, current, ref)
	if err != nil {
		return current, err
	}
	attemptID, err := e.NewID()
	if err != nil {
		return current, err
	}
	update := func(mutate func(*workflow.Execution) error) (workflow.Execution, error) {
		return e.Store.UpdateFenced(ctx, projectID, executionID, lease, mutate)
	}
	prepared, err := update(func(x *workflow.Execution) error {
		return x.PrepareAttempt(workflow.PrepareInput{Ref: ref, AttemptID: attemptID, Generation: gen, Invocation: invocation, Authority: authority, Now: e.now(), Retry: isRetry})
	})
	if err != nil {
		return current, err
	}
	request := e.request(prepared, ref, attemptID)
	// Only the runner knows nothing was spawned yet; a cancel arriving after
	// this check is caught by the watcher and stops the owned group.
	if e.cancelPersisted(ctx, projectID, executionID) {
		return update(func(x *workflow.Execution) error {
			return x.MarkCancelled(attemptID, gen, "cancelled_before_spawn", e.now())
		})
	}
	built, err := e.Launcher.Build(ctx, request)
	if err != nil {
		return update(func(x *workflow.Execution) error {
			return x.MarkSpawnFailed(attemptID, gen, "launch_refused: "+err.Error(), e.now())
		})
	}
	stage := request.Stage
	runCtx, stop := context.WithTimeout(ctx, time.Duration(stage.TimeoutSeconds)*time.Second)
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
	cancelRequested := cancelled() || e.cancelPersisted(ctx, projectID, executionID)
	return update(func(x *workflow.Execution) error {
		switch {
		case result.SpawnFailed && !startedCommitted:
			return x.MarkSpawnFailed(attemptID, gen, "spawn_failed: "+result.FailureCode, now)
		case !startedCommitted:
			// Spawned but identity was never durably published; effects unknown.
			return x.MarkUnknown(attemptID, gen, "process_identity_not_committed", now)
		case cancelRequested:
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
		outcome, err := e.Launcher.Collect(ctx, request, result)
		if err != nil {
			return x.MarkUnknown(attemptID, gen, "output_collection_failed", now)
		}
		err = x.CompleteAttempt(attemptID, gen, workflow.AttemptResult{ExitCode: result.ExitCode, Diagnostic: outcome.Diagnostic, Effects: outcome.Effects, Artifacts: outcome.Artifacts, Now: now})
		if errors.Is(err, workflow.ErrUnauthorizedEffect) {
			return x.MarkUnknown(attemptID, gen, "unauthorized_effect_reported", now)
		}
		return err
	})
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
	seen := make(chan struct{})
	go func() {
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
	return func() bool {
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
	}
	return after, facts, nil
}

func observeRecorded(p *workflow.ProcessIdentity) string {
	if p == nil {
		return "no_process_identity_recorded"
	}
	current, err := executiongraph.ObserveRecordedProcess(executiongraph.ProcessCorrelation{PID: p.PID, StartIdentity: p.StartIdentity, GroupIdentity: p.GroupIdentity})
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
