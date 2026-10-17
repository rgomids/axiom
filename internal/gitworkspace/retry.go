package gitworkspace

import (
	"context"
	"errors"

	"github.com/rgomids/axiom/internal/executiongraph"
)

// ErrRetryResetFencingUnavailable is a fail-closed blocker, not permission to
// retry a Git command. No reset effects have occurred when it is returned.
var ErrRetryResetFencingUnavailable = errors.New("retry_reset_authority_fencing_unavailable")

// InspectRetryWorkspace verifies the exact managed worktree while preserving
// its modifications. Clean-tree validation would silently forbid preserve.
func (m *Manager) InspectRetryWorkspace(ctx context.Context, child executiongraph.ChildExecution) error {
	_, err := m.inspect(ctx, child, false)
	return err
}

// ResetWorkspace restores the exact owned worktree to its base revision and
// then requires it to be clean. guard is the caller's fencing: it must fail
// once the run lease, the pending retry receipt or the persisted state that
// authorized the reset no longer holds. It runs, together with a fresh
// ownership and confinement inspection, immediately before every destructive
// effect and once after the last one.
func (m *Manager) ResetWorkspace(ctx context.Context, child executiongraph.ChildExecution, guard func(context.Context) error) error {
	if m == nil || guard == nil {
		return ErrRetryResetFencingUnavailable
	}
	effects := [][]string{{"reset", "--hard", "--quiet", m.baseRevision}, {"clean", "-ffdxq"}}
	for _, args := range effects {
		if err := m.fenced(ctx, child, guard); err != nil {
			return err
		}
		if _, err := m.gitOutput(ctx, child.Envelope.Workspace, nil, nil, args...); err != nil {
			return errors.Join(ErrWorkspaceInvalid, err)
		}
	}
	if err := guard(ctx); err != nil {
		return err
	}
	_, err := m.inspect(ctx, child, true)
	return err
}

func (m *Manager) fenced(ctx context.Context, child executiongraph.ChildExecution, guard func(context.Context) error) error {
	if err := guard(ctx); err != nil {
		return err
	}
	if _, err := m.inspect(ctx, child, false); err != nil {
		return err
	}
	return m.rejectExecutableConfiguration(ctx, child.Envelope.Workspace)
}
