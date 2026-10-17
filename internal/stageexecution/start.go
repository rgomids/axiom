package stageexecution

import (
	"context"
	"github.com/rgomids/axiom/internal/executiongraph"
)

// CommittedStarter launches the detached supervisor of a committed graph or
// single-stage publication. Both receive only the committed receipt identity.
type CommittedStarter struct {
	Store    SupervisorStore
	Single   SingleStore
	Detached DetachedStart
}

func (s CommittedStarter) Start(ctx context.Context, p Publication) (executiongraph.ProcessCorrelation, error) {
	if p.Graph != nil && p.Single == nil {
		return Launch(ctx, SupervisorRequest{ProjectID: p.Authority.ProjectID, ParentID: p.Authority.ParentID, PreviewDigest: p.Authority.PreviewDigest, AuthorityRef: p.Graph.DispatchAuthorityRef}, s.Store, s.Detached)
	}
	if p.Single != nil && p.Graph == nil {
		return LaunchSingle(ctx, p.Single.Request(), s.Single, s.Detached)
	}
	return executiongraph.ProcessCorrelation{}, ErrPreview
}

// StartRetry launches the detached supervisor for one committed retry receipt.
func (s CommittedStarter) StartRetry(ctx context.Context, r SupervisorRequest, retryRef string) (executiongraph.ProcessCorrelation, error) {
	store, ok := s.Store.(RetrySupervisorStore)
	if !ok {
		return executiongraph.ProcessCorrelation{}, ErrSupervisorAuthority
	}
	return LaunchRetry(ctx, r, retryRef, store, s.Detached)
}
