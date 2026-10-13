package stageexecution

import (
	"context"
	"errors"

	"github.com/rgomids/axiom/internal/executiongraph"
)

// resume is the committed-publication branch of stage run, never a new public
// operation or receipt. The detached supervisor repeats all frozen checks under
// its lease, including capability, auth, effort sources and workspace checks.
func (s RunService) resume(ctx context.Context, r RunRequest) (RunResult, error) {
	result := RunResult{Status: "denied_authority"}
	if !r.AuthorizeLocal || !r.AuthorizeExternal || r.Actor == "" {
		return result, ErrDispatchAuthority
	}
	if s.ResumeStore == nil || s.Starter == nil || s.Workspaces == nil || r.ExpectedRevision != r.Execution.Revision {
		return result, ErrStalePreview
	}
	request := *r.Committed
	if request.ParentID != r.Execution.ExecutionID || request.PreviewDigest != r.PreviewDigest {
		return result, ErrSupervisorAuthority
	}
	g, a, err := s.ResumeStore.Load(ctx, request)
	if err != nil {
		return result, err
	}
	if err = verifySupervisor(request, g, a); err != nil {
		return result, err
	}
	if err = admitSupervisor(ctx, s.ResumeStore, request, g); err != nil {
		return result, err
	}
	if g.CancelRequest != nil {
		return result, ErrSupervisorAuthority
	}
	remaining := false
	for _, c := range g.Children {
		for _, at := range c.Attempts {
			if at.Status == executiongraph.AttemptRunning || at.Status == executiongraph.AttemptUnknown || at.AmbiguousEffect {
				result.Status, result.Recovery = "recovery_required", "reconcile_required"
				return result, ErrSupervisorRecovery
			}
		}
		if executiongraph.CheckFirstDispatch(g, a, request.PreviewDigest, c.ExecutionID, 1) == nil {
			remaining = true
		}
	}
	if !remaining {
		return result, ErrSupervisorAuthority
	}
	fresh, err := s.Preview.Preview(ctx, r.Execution, r.StageID)
	if err != nil || fresh.Digest != a.PreviewDigest {
		return result, errors.Join(ErrStalePreview, err)
	}
	p := Publication{Graph: &g, Authority: a}
	result.Publication = p
	result.Status, result.Recovery = "recovery_required", "resume committed first attempts"
	if err = s.Workspaces.Prepare(ctx, p); err != nil {
		return result, err
	}
	// Workspace preparation must not open a policy/proof drift window.
	fresh, err = s.Preview.Preview(ctx, r.Execution, r.StageID)
	if err != nil || fresh.Digest != a.PreviewDigest {
		return result, errors.Join(ErrStalePreview, err)
	}
	process, err := s.Starter.Start(ctx, p)
	if err != nil {
		return result, err
	}
	if process.PID <= 0 || process.StartIdentity == "" || process.GroupIdentity == "" {
		return result, ErrSupervisorRecovery
	}
	result.Status, result.Recovery, result.Process = "success", "", &process
	return result, nil
}
