package stageexecution

import (
	"context"
	"errors"

	"github.com/rgomids/axiom/internal/workflow"
)

// observeSourceCapture never mutates Git. The captured path must equal the
// frozen path, so a changed machine-local workspace root fails closed.
func (e Engine) observeSourceCapture(ctx context.Context, lease workflow.RunnerFence, x workflow.Execution, git IntegrationGit, id string) (workflow.Execution, error) {
	a, _, ok := x.FindAttempt(id)
	if !ok || a.Capture == nil {
		return x, workflow.ErrTransition
	}
	if a.Capture.Recovery != nil {
		return x, nil
	}
	w, c := a.Workspace, a.Capture
	observeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), terminalCommitBound)
	defer cancel()
	o, observeErr := git.ObserveCapture(observeCtx, w.Worktree, w.SourceBranch, w.BaseSHA, c.Path, c.CandidateSHA)
	if observeErr != nil {
		o.Outcome = "unknown"
	}
	o.ObservedAt = e.now()
	updated, err := e.commit(ctx, x.ProjectID, x.ExecutionID, lease)(func(current *workflow.Execution) error { return current.ObserveCapture(id, lease.Generation(), o) })
	return updated, errors.Join(observeErr, err)
}

func (e Engine) recoverSourceCaptures(ctx context.Context, lease workflow.RunnerFence, x workflow.Execution) (workflow.Execution, error) {
	for _, stage := range x.Stages {
		for _, task := range stage.Tasks {
			for _, a := range task.Attempts {
				if a.Capture == nil || a.Capture.Recovery != nil || x.AttemptIntegrated(a.AttemptID) || a.Capture.RunnerGeneration >= lease.Generation() {
					continue
				}
				// The usual runner-loss handling has already blocked this workspace. If a
				// local Git binding is unavailable, persist unknown without inventing facts.
				var git WorkspaceGit
				var err error
				if e.Workspaces != nil {
					git, err = e.Workspaces(ctx, x, a.Workspace.RepositoryKey)
				}
				if git == nil || err != nil {
					x, err = e.commit(ctx, x.ProjectID, x.ExecutionID, lease)(func(current *workflow.Execution) error {
						return current.ObserveCapture(a.AttemptID, lease.Generation(), workflow.CaptureObservation{Outcome: "unknown", ObservedAt: e.now()})
					})
				} else {
					x, err = e.observeSourceCapture(ctx, lease, x, git, a.AttemptID)
				}
				if err != nil {
					return x, err
				}
			}
		}
	}
	return x, nil
}
