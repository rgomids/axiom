package local

import (
	"context"
	"errors"
	"os"
	"path/filepath"

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
	var fence func() error
	if lease == nil {
		if next.RunnerGeneration != current.RunnerGeneration {
			return ErrConflict
		}
	} else {
		if lease.Epoch != next.RunnerGeneration || lease.projectID != next.ProjectID || lease.parentID != next.ExecutionID {
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
		next.Messages = append([]workflow.Message(nil), current.Messages...)
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
		}
	}
	return out
}
