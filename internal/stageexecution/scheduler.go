package stageexecution

import (
	"context"
	"sync"
	"time"

	"github.com/rgomids/axiom/internal/workflow"
)

// attemptStore sends mutations back to the principal Run loop. Worker goroutines
// may observe state, but never call the durable writer themselves. Cancellation
// of the run does not close the queue: observed terminal facts still need commits.
type attemptStore struct {
	ExecutionStore
	commits chan<- attemptCommit
}

type attemptCommit struct {
	ctx                context.Context
	project, execution string
	fence              workflow.RunnerFence
	mutate             func(*workflow.Execution) error
	reply              chan attemptCommitResult
}

type attemptCommitResult struct {
	x   workflow.Execution
	err error
}

func (s attemptStore) UpdateFenced(ctx context.Context, project, execution string, fence workflow.RunnerFence, mutate func(*workflow.Execution) error) (workflow.Execution, error) {
	reply := make(chan attemptCommitResult, 1)
	select {
	case s.commits <- attemptCommit{ctx, project, execution, fence, mutate, reply}:
	case <-ctx.Done():
		return workflow.Execution{}, ctx.Err()
	}
	// Once submitted, join the commit even if its context expires. The principal
	// observes that context and returns the actual commit result, never a guessed
	// outcome while a write remains pending.
	r := <-reply
	return r.x, r.err
}

type attemptFinished struct {
	ref workflow.TaskRef
	err error
}

// executionSnapshot uses the canonical codec for independent read-only worker
// snapshots, avoiding slice aliases even with a custom ExecutionStore.
func executionSnapshot(x workflow.Execution) (workflow.Execution, error) {
	wire, err := workflow.EncodeExecution(x)
	if err != nil {
		return workflow.Execution{}, err
	}
	return workflow.DecodeExecution(wire)
}

func (e Engine) runReady(ctx context.Context, lease workflow.RunnerFence, current workflow.Execution, opts RunOptions) (workflow.Execution, error) {
	project, execution := current.ProjectID, current.ExecutionID
	commits := make(chan attemptCommit)
	finished := make(chan attemptFinished)
	workers := e
	workers.Store = attemptStore{e.Store, commits}
	// Resolution and identifier allocation stay on the principal. Serialize the
	// remaining injected callbacks, preserving their original serial contract.
	var callbacks sync.Mutex
	if e.Admit != nil {
		workers.Admit = func(ctx context.Context, x workflow.Execution) error {
			callbacks.Lock()
			defer callbacks.Unlock()
			return e.Admit(ctx, x)
		}
	}
	if e.Now != nil {
		workers.Now = func() time.Time {
			callbacks.Lock()
			defer callbacks.Unlock()
			return e.Now()
		}
	}
	principal := e
	principal.Admit = workers.Admit
	principal.Now = workers.Now
	principal.Resolve = func(ctx context.Context, x workflow.Execution, ref workflow.TaskRef) (workflow.Invocation, workflow.AttemptAuthority, error) {
		callbacks.Lock()
		defer callbacks.Unlock()
		return e.Resolve(ctx, x, ref)
	}
	principal.NewID = func() (string, error) {
		callbacks.Lock()
		defer callbacks.Unlock()
		return e.NewID()
	}
	inflight := map[workflow.TaskRef]bool{}
	retry := opts.Retry
	var firstErr error
	if len(current.Enrollments) != 0 {
		firstErr = principal.preflightWorkspaces(ctx, current)
	}
	for {
		// External cancellation/gate changes are authoritative. Refresh before
		// computing eligibility; PrepareAttempt checks it again at the commit.
		latest, err := e.Store.Load(context.WithoutCancel(ctx), project, execution)
		if err == nil {
			current = latest
		} else if firstErr == nil {
			firstErr = err
		}
		if firstErr == nil && ctx.Err() == nil && len(inflight) == 0 {
			current, firstErr = principal.integrateReady(ctx, lease, current)
		}
		if firstErr == nil && ctx.Err() == nil {
			ready := current.Ready()
			isRetry := retry != nil
			if isRetry {
				ready = []workflow.TaskRef{*retry}
				retry = nil
			}
			for _, ref := range ready {
				if inflight[ref] {
					continue
				}
				request, err := principal.prepareAttempt(ctx, lease, current, ref, isRetry, isRetry && opts.NewSelection, opts.WorkspaceDisposition)
				if err != nil {
					if request.Execution.ExecutionID != "" {
						current = request.Execution
					}
					firstErr = err
					break
				}
				current = request.Execution
				// The worker owns an independent canonical snapshot. Neither a
				// later principal update nor a peer can alter its input slices.
				snapshot, err := executionSnapshot(request.Execution)
				if err != nil {
					firstErr = err
					break
				}
				request = e.request(snapshot, ref, request.Attempt.AttemptID)
				inflight[ref] = true
				go func() {
					_, err := workers.runAttempt(ctx, lease, request)
					finished <- attemptFinished{ref, err}
				}()
			}
		}
		if len(inflight) == 0 {
			if firstErr != nil {
				return current, firstErr
			}
			return current, ctx.Err()
		}
		// Continue servicing already-running attempts after a refusal or operator
		// cancellation. Returning early would strand process/terminal callbacks.
		// A worker error stops new scheduling, without mislabelling it as an
		// operator cancel or discarding independent results already in flight.
		select {
		case c := <-commits:
			x, err := e.Store.UpdateFenced(c.ctx, c.project, c.execution, c.fence, c.mutate)
			// A refused or duplicate message changed nothing; it is the child's
			// disposition, never a reason to stop scheduling independent work.
			if err != nil && firstErr == nil && !messageRefusal(err) {
				firstErr = err
			}
			c.reply <- attemptCommitResult{x, err}
		case f := <-finished:
			delete(inflight, f.ref)
			if f.err != nil && firstErr == nil {
				firstErr = f.err
			}
		}
	}
}
