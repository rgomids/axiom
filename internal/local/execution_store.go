package local

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"

	"github.com/rgomids/axiom/internal/workflow"
)

// ExecutionStore is the single owner of new-run truth (ADR-0037):
// executions/v2/<project>/<execution>.json, one Execution format 4 record per
// run. It never reads executions/v1, graphs/v1 or single-stage-runs/v1.
//
// Lock order: the runner lease (run-leases/v1, owned by RunLeaseStore) is
// acquired before the aggregate directory locks. Status reads take no lease.
type ExecutionStore struct {
	root   string
	hooks  publicationHooks
	leases RunLeaseStore
}

func NewExecutionStore(root string) (ExecutionStore, error) {
	if !filepath.IsAbs(root) || filepath.Clean(root) == string(filepath.Separator) {
		return ExecutionStore{}, ErrUnsafe
	}
	canonical, err := trustedCanonical(root)
	if err != nil {
		return ExecutionStore{}, err
	}
	return ExecutionStore{root: canonical, leases: RunLeaseStore{root: canonical}}, nil
}

func validRunAddress(projectID, executionID string) bool {
	return workflow.ValidToken(projectID) && platformEntryName(projectID) && workflow.ValidExecutionID(executionID)
}

type executionRoots struct{ chain []*os.Root }

func (r executionRoots) project() *os.Root { return r.chain[len(r.chain)-1] }
func (r executionRoots) Close() {
	for i := len(r.chain) - 1; i >= 0; i-- {
		r.chain[i].Close()
	}
}

func (s ExecutionStore) open(projectID string, create bool) (executionRoots, error) {
	openRoot, openChild := existingPrivateRoot, existingPrivateChild
	if create {
		openRoot, openChild = privateRoot, privateChild
	}
	root, err := openRoot(s.root)
	if err != nil {
		return executionRoots{}, err
	}
	roots := executionRoots{chain: []*os.Root{root}}
	for _, name := range []string{"executions", "v2", projectID} {
		child, err := openChild(roots.project(), name)
		if err != nil {
			roots.Close()
			return executionRoots{}, err
		}
		roots.chain = append(roots.chain, child)
	}
	return roots, nil
}

func runRecordName(executionID string) string { return executionID + ".json" }

// Create publishes the first revision; an existing record is a conflict.
func (s ExecutionStore) Create(ctx context.Context, e workflow.Execution) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if !validRunAddress(e.ProjectID, e.ExecutionID) || e.Revision != 1 || e.RunnerGeneration != 0 {
		return ErrUnsafe
	}
	wire, err := workflow.EncodeExecution(e)
	if err != nil {
		return err
	}
	roots, err := s.open(e.ProjectID, true)
	if err != nil {
		return err
	}
	defer roots.Close()
	locks, err := lockRoots(true, roots.chain...)
	if err != nil {
		return err
	}
	defer closeFiles(locks)
	if e.ReworkOf != nil {
		source, _, err := s.read(roots.project(), e.ProjectID, e.ReworkOf.ExecutionID)
		if err != nil {
			return err
		}
		if !workflow.ValidReworkStart(source, e) {
			return ErrUnsafe
		}
	}
	return publishFile(ctx, roots.project(), runRecordName(e.ExecutionID), nil, wire, true, s.withLimit(nil))
}

func (s ExecutionStore) withLimit(beforeCommit func() error) publicationHooks {
	hooks := s.hooks
	hooks.contentLimit = workflow.MaxExecutionBytes
	if beforeCommit != nil {
		prior := hooks.beforeCommit
		hooks.beforeCommit = func() error {
			if prior != nil {
				if err := prior(); err != nil {
					return err
				}
			}
			return beforeCommit()
		}
	}
	return hooks
}

func (s ExecutionStore) read(project *os.Root, projectID, executionID string) (workflow.Execution, []byte, error) {
	wire, err := readPublishedFile(project, runRecordName(executionID))
	if errors.Is(err, os.ErrNotExist) {
		return workflow.Execution{}, nil, ErrNotFound
	}
	if err != nil {
		return workflow.Execution{}, nil, err
	}
	e, err := workflow.DecodeExecution(wire)
	if err != nil {
		return workflow.Execution{}, nil, err
	}
	if e.ProjectID != projectID || e.ExecutionID != executionID {
		return workflow.Execution{}, nil, ErrUnsafe
	}
	return e, wire, nil
}

// Load is a read-only status read: shared locks, no runner lease. Interrupted
// publication reports recovery_required, never a guessed state.
func (s ExecutionStore) Load(ctx context.Context, projectID, executionID string) (workflow.Execution, error) {
	if err := ctx.Err(); err != nil {
		return workflow.Execution{}, err
	}
	if !validRunAddress(projectID, executionID) {
		return workflow.Execution{}, ErrUnsafe
	}
	roots, err := s.open(projectID, false)
	if err != nil {
		return workflow.Execution{}, err
	}
	defer roots.Close()
	locks, err := lockRoots(false, roots.chain...)
	if err != nil {
		return workflow.Execution{}, err
	}
	defer closeFiles(locks)
	e, _, err := s.read(roots.project(), projectID, executionID)
	return e, err
}

// AcquireRunner takes the exclusive non-blocking runner lease. Its epoch is
// strictly greater than any generation already recorded in the Execution;
// the caller must commit it with Execution.ClaimRunner before dispatching.
func (s ExecutionStore) AcquireRunner(ctx context.Context, projectID, executionID string) (*RunLease, error) {
	current, err := s.Load(ctx, projectID, executionID)
	if err != nil {
		return nil, err
	}
	return s.leases.Acquire(ctx, projectID, executionID, current.RunnerGeneration)
}

// Generation exposes the lease epoch as a workflow.RunnerFence.
func (l *RunLease) Generation() uint64 { return l.Epoch }

// AcquireFence is AcquireRunner behind the workflow.RunnerFence boundary; a
// held lease reports workflow.ErrRunnerHeld.
func (s ExecutionStore) AcquireFence(ctx context.Context, projectID, executionID string) (workflow.RunnerFence, error) {
	lease, err := s.AcquireRunner(ctx, projectID, executionID)
	if errors.Is(err, ErrConflict) {
		return nil, errors.Join(workflow.ErrRunnerHeld, err)
	}
	if err != nil {
		return nil, err
	}
	return lease, nil
}

// UpdateFenced is Update for a fence obtained from AcquireFence (or nil).
func (s ExecutionStore) UpdateFenced(ctx context.Context, projectID, executionID string, fence workflow.RunnerFence, mutate func(*workflow.Execution) error) (workflow.Execution, error) {
	var lease *RunLease
	if fence != nil {
		l, ok := fence.(*RunLease)
		if !ok || l == nil {
			return workflow.Execution{}, ErrUnsafe
		}
		lease = l
	}
	return s.Update(ctx, projectID, executionID, lease, mutate)
}

// Save is the revision CAS: it commits next only if the stored revision is
// still expected. With a lease, the lease is revalidated at the commit point
// and must equal next's runner generation; without one (human cancel, gate,
// disposition) the runner generation cannot change.
func (s ExecutionStore) Save(ctx context.Context, next workflow.Execution, expected uint64, lease *RunLease) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if !validRunAddress(next.ProjectID, next.ExecutionID) || next.Revision <= expected {
		return ErrConflict
	}
	wire, err := workflow.EncodeExecution(next)
	if err != nil {
		return err
	}
	roots, err := s.open(next.ProjectID, false)
	if err != nil {
		return err
	}
	defer roots.Close()
	locks, err := lockRoots(true, roots.chain...)
	if err != nil {
		return err
	}
	defer closeFiles(locks)
	current, prior, err := s.read(roots.project(), next.ProjectID, next.ExecutionID)
	if err != nil {
		return err
	}
	if current.Revision != expected || next.RunnerGeneration < current.RunnerGeneration {
		return ErrConflict
	}
	if !preservesExecutionHistory(current, next) {
		return ErrConflict
	}
	if lease == nil && !reflect.DeepEqual(current.Deliveries, next.Deliveries) {
		return workflow.ErrStaleRunner
	}
	var fence func() error
	if lease == nil {
		if next.RunnerGeneration != current.RunnerGeneration {
			return ErrConflict
		}
	} else {
		// The fence belongs to the state root that issued it. Both roots are the
		// trustedCanonical form, so exact equality (never a prefix) binds the lease
		// to this store; a same-epoch lease from another root owns nothing here.
		if lease.stateRoot != s.root || lease.Epoch != next.RunnerGeneration || lease.projectID != next.ProjectID || lease.parentID != next.ExecutionID {
			return workflow.ErrStaleRunner
		}
		// Pin the lease through the commit point so it cannot be released
		// between fence validation and rename.
		lease.mu.Lock()
		defer lease.mu.Unlock()
		fence = func() error {
			if err := lease.validateLocked(); err != nil {
				return workflow.ErrStaleRunner
			}
			return nil
		}
	}
	return publishFile(ctx, roots.project(), runRecordName(next.ExecutionID), prior, wire, false, s.withLimit(fence))
}

// CAS alone cannot protect immutable provenance: a current writer must also
// retain the selected definition, attempt identity, frozen invocation and
// terminal observations. Human resolution may append a disposition to an
// unresolved unknown, but never rewrite the uncertain effects themselves.
func preservesExecutionHistory(prior, next workflow.Execution) bool {
	if !reflect.DeepEqual(prior.ReworkOf, next.ReworkOf) {
		return false
	}
	if prior.Rejection != nil {
		frozen := next
		frozen.Revision, frozen.RunnerGeneration, frozen.UpdatedAt = prior.Revision, prior.RunnerGeneration, prior.UpdatedAt
		if !reflect.DeepEqual(prior, frozen) {
			return false
		}
	}
	if prior.ExecutionID != next.ExecutionID || prior.ProjectID != next.ProjectID || prior.WorkItem != next.WorkItem || !prior.CreatedAt.Equal(next.CreatedAt) || !reflect.DeepEqual(prior.Repositories, next.Repositories) || !reflect.DeepEqual(prior.Workflow, next.Workflow) || !reflect.DeepEqual(prior.Selections, next.Selections) || len(prior.Stages) != len(next.Stages) {
		return false
	}
	if prior.Cancel != nil && !reflect.DeepEqual(prior.Cancel, next.Cancel) {
		return false
	}
	if !workflow.PreservesDeliveryHistory(prior.Deliveries, next.Deliveries) {
		return false
	}
	if len(next.Integrations) < len(prior.Integrations) {
		return false
	}
	for _, integration := range next.Integrations[len(prior.Integrations):] {
		// New intent must be committed before any terminal observation.
		if integration.Status != workflow.IntegrationPrepared {
			return false
		}
	}
	for k, integration := range prior.Integrations {
		changed := next.Integrations[k]
		if integration.Status == workflow.IntegrationPrepared && changed.Status != workflow.IntegrationPrepared {
			// Only the observed outcome may be filled in; intent stays immutable.
			changed.Status = integration.Status
			changed.MergeSHA, changed.FinalSHA, changed.ObservedTargetHead = integration.MergeSHA, integration.FinalSHA, integration.ObservedTargetHead
			changed.Checks, changed.Diagnostic, changed.EndedAt = integration.Checks, integration.Diagnostic, integration.EndedAt
		} else if integration.Status == workflow.IntegrationUnknown && integration.Resolution == nil {
			// Validate permits only an explicit human abandonment here.
			changed.Resolution = nil
		}
		if !reflect.DeepEqual(integration, changed) {
			return false
		}
	}
	// Accepted messages are append-only facts.
	if len(next.Messages) < len(prior.Messages) {
		return false
	}
	for i, m := range prior.Messages {
		n := next.Messages[i]
		if !m.ReceivedAt.Equal(n.ReceivedAt) {
			return false
		}
		n.ReceivedAt = m.ReceivedAt
		if m != n {
			return false
		}
	}
	for i, stage := range prior.Stages {
		if stage.StageID != next.Stages[i].StageID || len(stage.Tasks) != len(next.Stages[i].Tasks) || (stage.Gate != nil && !reflect.DeepEqual(stage.Gate, next.Stages[i].Gate)) || (stage.BeforeApproval != nil && !reflect.DeepEqual(stage.BeforeApproval, next.Stages[i].BeforeApproval)) || (stage.Review != nil && !reflect.DeepEqual(stage.Review, next.Stages[i].Review)) {
			return false
		}
		for j, task := range stage.Tasks {
			updated := next.Stages[i].Tasks[j]
			if task.AgentID != updated.AgentID || len(updated.Attempts) < len(task.Attempts) {
				return false
			}
			for k, attempt := range task.Attempts {
				changed := updated.Attempts[k]
				if attempt.AttemptID != changed.AttemptID || attempt.Number != changed.Number || attempt.RunnerGeneration != changed.RunnerGeneration || !attempt.PreparedAt.Equal(changed.PreparedAt) || attempt.WorkspaceDisposition != changed.WorkspaceDisposition || !reflect.DeepEqual(attempt.Invocation, changed.Invocation) || !reflect.DeepEqual(attempt.Authority, changed.Authority) {
					return false
				}
				if attempt.LaunchFailure != nil && !reflect.DeepEqual(attempt.LaunchFailure, changed.LaunchFailure) {
					return false
				}
				if len(changed.Commands) < len(attempt.Commands) {
					return false
				}
				for n, command := range attempt.Commands {
					nextCommand := changed.Commands[n]
					if command.ID != nextCommand.ID || command.ArgvDigest != nextCommand.ArgvDigest || !command.PreparedAt.Equal(nextCommand.PreparedAt) || command.Process != nil && !reflect.DeepEqual(command.Process, nextCommand.Process) || command.StartedAt != nil && !reflect.DeepEqual(command.StartedAt, nextCommand.StartedAt) || command.EndedAt != nil && !reflect.DeepEqual(command, nextCommand) {
						return false
					}
				}
				if attempt.Status == workflow.AttemptPrepared || attempt.Status == workflow.AttemptRunning {
					if attempt.Process != nil && !reflect.DeepEqual(attempt.Process, changed.Process) || attempt.StartedAt != nil && !reflect.DeepEqual(attempt.StartedAt, changed.StartedAt) {
						return false
					}
					continue
				}
				if attempt.Status == workflow.AttemptUnknown && attempt.Resolution == nil {
					changed.Resolution = nil
				}
				if !reflect.DeepEqual(attempt, changed) {
					return false
				}
			}
		}
	}
	return true
}

// Update reloads, applies mutate and saves under CAS, retrying a bounded
// number of times when another writer won. mutate must be a pure state
// transition; it is never used to spawn or signal processes.
func (s ExecutionStore) Update(ctx context.Context, projectID, executionID string, lease *RunLease, mutate func(*workflow.Execution) error) (workflow.Execution, error) {
	for range 8 {
		current, err := s.Load(ctx, projectID, executionID)
		if err != nil {
			return workflow.Execution{}, err
		}
		next := current
		next.Stages = cloneExecutionStages(current.Stages)
		next.Deliveries = append([]workflow.ExecutionDelivery(nil), current.Deliveries...)
		for i := range next.Deliveries {
			next.Deliveries[i].Observations = append([]workflow.DeliveryObservation(nil), current.Deliveries[i].Observations...)
		}
		next.Messages = append([]workflow.Message(nil), current.Messages...)
		next.Integrations = append([]workflow.Integration(nil), current.Integrations...)
		for i := range next.Integrations {
			integration := &next.Integrations[i]
			integration.Checks = append([]workflow.IntegrationCheck(nil), integration.Checks...)
			if integration.EndedAt != nil {
				at := *integration.EndedAt
				integration.EndedAt = &at
			}
			if integration.Resolution != nil {
				resolution := *integration.Resolution
				integration.Resolution = &resolution
			}
		}
		if err := mutate(&next); err != nil {
			return current, err
		}
		err = s.Save(ctx, next, current.Revision, lease)
		if errors.Is(err, ErrConflict) {
			continue
		}
		if err != nil {
			return current, err
		}
		return next, nil
	}
	return workflow.Execution{}, ErrConflict
}

func cloneExecutionStages(stages []workflow.StageRun) []workflow.StageRun {
	out := make([]workflow.StageRun, len(stages))
	for i, s := range stages {
		out[i] = s
		out[i].Tasks = make([]workflow.TaskRun, len(s.Tasks))
		for j, t := range s.Tasks {
			out[i].Tasks[j] = t
			out[i].Tasks[j].Attempts = append([]workflow.Attempt(nil), t.Attempts...)
			for k, a := range t.Attempts {
				if a.LaunchFailure != nil {
					f := *a.LaunchFailure
					out[i].Tasks[j].Attempts[k].LaunchFailure = &f
				}
			}
		}
	}
	return out
}
