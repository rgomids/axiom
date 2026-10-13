package stageexecution

import (
	"context"
	"errors"
	"reflect"
	"time"

	"github.com/rgomids/axiom/internal/executiongraph"
	"github.com/rgomids/axiom/internal/workflow"
	"github.com/rgomids/axiom/internal/workflowcompiler"
)

var ErrDispatchAuthority = errors.New("local_and_external_dispatch_authority_required")

// Publication holds exactly one graph or single ledger candidate. Commit must
// use T07's protected multi-object protocol, CAS and (for graphs) a run lease.
type Publication struct {
	Graph     *executiongraph.SuccessorGraph
	Single    *workflow.SingleDispatch
	Authority executiongraph.DispatchAuthority
}
type DispatchPublisher interface {
	Build(context.Context, DispatchPreview, string, time.Time) (Publication, error)
	Commit(context.Context, Publication) error
	ReadBack(context.Context, Publication) (Publication, error)
}
type DispatchWorkspaces interface {
	Prepare(context.Context, Publication) error
}
type DispatchStarter interface {
	Start(context.Context, Publication) (executiongraph.ProcessCorrelation, error)
}
type DispatchProjection interface {
	Project(context.Context, Publication) error
}
type RunService struct {
	ResumeStore SupervisorStore
	Preview     PreviewService
	Publisher   DispatchPublisher
	Workspaces  DispatchWorkspaces
	Starter     DispatchStarter
	Projection  DispatchProjection
	Now         func() time.Time
}
type RunRequest struct {
	Committed                         *SupervisorRequest
	Execution                         workflowcompiler.ExecutionRef
	StageID                           string
	PreviewDigest                     string
	ExpectedRevision                  uint64
	AuthorizeLocal, AuthorizeExternal bool
	Actor                             string
}
type RunResult struct {
	Status      string
	Recovery    string
	Publication Publication
	Process     *executiongraph.ProcessCorrelation
}

func (s RunService) Apply(ctx context.Context, r RunRequest) (RunResult, error) {
	if r.Committed != nil {
		return s.resume(ctx, r)
	}
	result := RunResult{Status: "denied_authority"}
	if !r.AuthorizeLocal || !r.AuthorizeExternal || r.Actor == "" {
		return result, ErrDispatchAuthority
	}
	if r.ExpectedRevision != r.Execution.Revision {
		return result, ErrStalePreview
	}
	if s.Publisher == nil || s.Workspaces == nil || s.Starter == nil {
		return result, ErrPreview
	}
	fresh, e := s.Preview.Preview(ctx, r.Execution, r.StageID)
	if e != nil {
		return result, errors.Join(ErrStalePreview, e)
	}
	if fresh.Digest != r.PreviewDigest {
		return result, ErrStalePreview
	}
	now := time.Now
	if s.Now != nil {
		now = s.Now
	}
	publication, e := s.Publisher.Build(ctx, fresh, r.Actor, now().UTC())
	if e != nil {
		return result, e
	}
	// Build/allocation must not open a drift window for policy or proof changes.
	check, e := s.Preview.Preview(ctx, r.Execution, r.StageID)
	if e != nil {
		return result, errors.Join(ErrStalePreview, e)
	}
	if check.Digest != fresh.Digest {
		return result, ErrStalePreview
	}
	result.Status = "failure"
	if e = s.Publisher.Commit(ctx, publication); e != nil {
		var committed interface{ EffectCommitted() bool }
		if errors.As(e, &committed) && committed.EffectCommitted() {
			result.Status = "partial"
			result.Publication = publication
		}
		result.Recovery = "read publication and authority before any recovery dispatch"
		return result, e
	}
	result.Publication = publication
	read, e := s.Publisher.ReadBack(ctx, publication)
	if e != nil || !samePublication(publication, read) {
		result.Status = "recovery_required"
		result.Recovery = "confirm the protected publication"
		if e == nil {
			e = ErrSupervisorRecovery
		}
		return result, e
	}
	// Projection failure is a confirmed partial publication, never success.
	if s.Projection != nil {
		if e = s.Projection.Project(ctx, read); e != nil {
			result.Status = "partial"
			result.Recovery = "repair projection of the committed publication"
			return result, e
		}
	}
	if e = s.Workspaces.Prepare(ctx, read); e != nil {
		result.Status = "recovery_required"
		result.Recovery = "prepare workspaces for committed publication"
		return result, e
	}
	process, e := s.Starter.Start(ctx, read)
	if e != nil {
		result.Status = "recovery_required"
		result.Recovery = "resume the committed first-attempt authority after supervisor observation"
		return result, e
	}
	if process.PID <= 0 || process.StartIdentity == "" || process.GroupIdentity == "" {
		result.Status = "recovery_required"
		result.Recovery = "observe supervisor identity before resuming"
		return result, ErrSupervisorRecovery
	}
	result.Process = &process
	result.Status = "success"
	return result, nil
}
func samePublication(a, b Publication) bool {
	if a.Graph != nil {
		a.Graph = cloneGraphWithoutStorage(a.Graph)
	}
	if b.Graph != nil {
		b.Graph = cloneGraphWithoutStorage(b.Graph)
	}
	if a.Single != nil {
		c := *a.Single
		c.State.StorageRevision = [32]byte{}
		a.Single = &c
	}
	if b.Single != nil {
		c := *b.Single
		c.State.StorageRevision = [32]byte{}
		b.Single = &c
	}
	return reflect.DeepEqual(a, b)
}
func cloneGraphWithoutStorage(g *executiongraph.SuccessorGraph) *executiongraph.SuccessorGraph {
	c := *g
	c.StorageRevision = [32]byte{}
	return &c
}
