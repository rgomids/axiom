package stageexecution

import (
	"context"
	"github.com/rgomids/axiom/internal/executiongraph"
	"github.com/rgomids/axiom/internal/workflow"
)

// CommittedStarter delegates graph supervision to T27 and single supervision to
// the extended sequential executor. Both receive the same committed receipt.
type CommittedStarter struct {
	Store      SupervisorStore
	Detached   DetachedStart
	Sequential workflow.SequentialStageSupervisor
}

func (s CommittedStarter) Start(ctx context.Context, p Publication) (executiongraph.ProcessCorrelation, error) {
	if p.Graph != nil && p.Single == nil {
		return Launch(ctx, SupervisorRequest{ProjectID: p.Authority.ProjectID, ParentID: p.Authority.ParentID, PreviewDigest: p.Authority.PreviewDigest, AuthorityRef: p.Graph.DispatchAuthorityRef}, s.Store, s.Detached)
	}
	if p.Single != nil && p.Graph == nil {
		return workflow.StartSingleStage(ctx, *p.Single, p.Authority, s.Sequential)
	}
	return executiongraph.ProcessCorrelation{}, ErrPreview
}
