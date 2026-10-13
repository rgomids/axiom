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

// ResetWorkspace is disabled until persisted retry authority and lease fencing
// can be held across each destructive effect. Workspace inspection alone cannot
// establish that guarantee, even when a caller claims a committed receipt.
func (m *Manager) ResetWorkspace(ctx context.Context, child executiongraph.ChildExecution) error {
	return applyWorkspaceReset(ctx, child.Envelope.Workspace, "", nil, nil)
}

// Keep the intercepted effect boundary fail-closed as well. There is no
// validation/effect window: neither callback is invoked, regardless of inputs.
func applyWorkspaceReset(ctx context.Context, workspace, base string, validate func(bool) error, run func(context.Context, string, ...string) error) error {
	return ErrRetryResetFencingUnavailable
}
