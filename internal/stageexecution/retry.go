package stageexecution

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/rgomids/axiom/internal/executiongraph"
	"github.com/rgomids/axiom/internal/workflowcompiler"
)

var (
	ErrRetryIneligible     = errors.New("retry_not_eligible")
	ErrRetryExhausted      = errors.New("retry_attempts_exhausted")
	ErrDispositionRequired = errors.New("workspace_disposition_required")
)

// RetrySupervisorStore resolves only a protected retry receipt by digest. The
// receipt is single-use: it binds the graph revision that its attempt changes.
type RetrySupervisorStore interface {
	SupervisorStore
	LoadRetry(context.Context, SupervisorRequest, string) (executiongraph.RetryAuthority, error)
}

// RetryStore commits the retry receipt for a detached supervisor. It cannot
// create attempts, change status or rewrite first-attempt authority.
type RetryStore interface {
	StatusSource
	CommitRetry(context.Context, executiongraph.SuccessorGraph, executiongraph.RetryAuthority) (string, error)
}

// WorkspaceDisposer applies an explicit, authorized disposition. "preserve"
// must leave the workspace untouched; there is no implicit reset.
type WorkspaceDisposer interface {
	Dispose(context.Context, executiongraph.SuccessorGraph, executiongraph.SuccessorChild, string) error
}
type RetryStarter interface {
	StartRetry(context.Context, SupervisorRequest, string) (executiongraph.ProcessCorrelation, error)
}

type RetryRequest struct {
	Target    StatusRequest
	Execution workflowcompiler.ExecutionRef
	StageID   string
	ChildID   string
}
type RetryPreview struct {
	Request             RetryRequest
	Revision            uint64
	StorageRevision     [32]byte
	PriorAttemptID      string
	PriorStatus         executiongraph.AttemptStatus
	AttemptNumber       uint32
	Remaining           uint32
	DispositionRequired bool
	// DispatchDigest is a fresh dispatch preview: policy, frozen tuple and
	// capability evidence are re-observed, never reused from the first run.
	DispatchDigest string
	Effects        []executiongraph.Effect
	Digest         string
}
type RetryApply struct {
	Preview           RetryPreview
	AuthorizeLocal    bool
	AuthorizeExternal bool
	Actor             string
	Disposition       string
}
type RetryResult struct {
	Status, Recovery string
	Authority        *executiongraph.RetryAuthority
	AuthorityRef     string
	Process          *executiongraph.ProcessCorrelation
}
type RetryService struct {
	Store     RetryStore
	Dispatch  PreviewService
	Disposer  WorkspaceDisposer
	Starter   RetryStarter
	Now       func() time.Time
	Authority func(context.Context, StatusRequest) (SupervisorRequest, error)
}

func (s RetryService) snapshot(ctx context.Context, r RetryRequest) (executiongraph.SuccessorGraph, executiongraph.SuccessorChild, error) {
	if s.Store == nil {
		return executiongraph.SuccessorGraph{}, executiongraph.SuccessorChild{}, ErrPreview
	}
	c, err := s.Store.StageSnapshot(ctx, r.Target)
	if err != nil {
		return executiongraph.SuccessorGraph{}, executiongraph.SuccessorChild{}, err
	}
	if _, err = Admit(StageRetry, c); err != nil {
		return executiongraph.SuccessorGraph{}, executiongraph.SuccessorChild{}, err
	}
	if c.Graph == nil || c.Graph.Successor == nil {
		return executiongraph.SuccessorGraph{}, executiongraph.SuccessorChild{}, executiongraph.ErrUnsupportedGraphFormat
	}
	g := *c.Graph.Successor
	if g.Parent.ExecutionID != r.Target.ParentID || g.Children[0].Envelope.Scope.ProjectID != r.Target.ProjectID || r.Execution.ExecutionID != r.Target.ParentID {
		return g, executiongraph.SuccessorChild{}, ErrAdmission
	}
	for _, child := range g.Children {
		if child.ExecutionID == r.ChildID {
			return g, child, nil
		}
	}
	return g, executiongraph.SuccessorChild{}, ErrAdmission
}

func (s RetryService) preview(ctx context.Context, r RetryRequest) (RetryPreview, executiongraph.SuccessorGraph, error) {
	g, child, err := s.snapshot(ctx, r)
	if err != nil {
		return RetryPreview{}, g, err
	}
	if child.Envelope.IntegrationOwner || len(child.Attempts) == 0 {
		return RetryPreview{}, g, ErrRetryIneligible
	}
	used := uint32(len(child.Attempts))
	if used >= child.Envelope.Controls.MaximumAttempts {
		return RetryPreview{}, g, ErrRetryExhausted
	}
	eligible, err := executiongraph.SuccessorRetryEligible(g, child.ExecutionID, true)
	if err != nil {
		return RetryPreview{}, g, err
	}
	if !eligible {
		return RetryPreview{}, g, ErrRetryIneligible
	}
	dispatch, err := s.Dispatch.Preview(ctx, r.Execution, r.StageID)
	if err != nil {
		return RetryPreview{}, g, errors.Join(ErrStalePreview, err)
	}
	prior := child.Attempts[len(child.Attempts)-1]
	p := RetryPreview{Request: r, Revision: g.Parent.GraphRevision, StorageRevision: g.StorageRevision, PriorAttemptID: prior.AttemptID, PriorStatus: prior.Status, AttemptNumber: used + 1, Remaining: child.Envelope.Controls.MaximumAttempts - used, DispositionRequired: executiongraph.RetryDispositionRequired(child), DispatchDigest: dispatch.Digest, Effects: child.Envelope.AllowedEffects}
	p.Digest, err = canonicalDigest(struct {
		Operation Operation
		Preview   RetryPreview
		Graph     executiongraph.SuccessorGraph
	}{StageRetry, p, g})
	return p, g, err
}

func (s RetryService) Preview(ctx context.Context, r RetryRequest) (RetryPreview, error) {
	p, _, err := s.preview(ctx, r)
	return p, err
}

// Apply needs local and external authority (PD-12) and, after possible
// effects, an explicit disposition (PD-8). It never resets attempt counters.
func (s RetryService) Apply(ctx context.Context, r RetryApply) (RetryResult, error) {
	result := RetryResult{Status: "denied_authority"}
	if !r.AuthorizeLocal || !r.AuthorizeExternal || r.Actor == "" {
		return result, ErrDispatchAuthority
	}
	if s.Disposer == nil || s.Starter == nil || s.Authority == nil {
		return result, ErrPreview
	}
	fresh, g, err := s.preview(ctx, r.Preview.Request)
	if err != nil {
		return result, err
	}
	if fresh.Digest != r.Preview.Digest {
		return result, ErrStalePreview
	}
	disposition := r.Disposition
	switch {
	case fresh.DispositionRequired && disposition != executiongraph.DispositionPreserve && disposition != executiongraph.DispositionReset:
		return result, ErrDispositionRequired
	case !fresh.DispositionRequired && disposition != "" && disposition != executiongraph.DispositionNotRequired:
		return result, ErrDispositionRequired
	case !fresh.DispositionRequired:
		disposition = executiongraph.DispositionNotRequired
	}
	supervisor, err := s.Authority(ctx, r.Preview.Request.Target)
	if err != nil {
		return result, err
	}
	if supervisor.ParentID != g.Parent.ExecutionID || supervisor.AuthorityRef != g.DispatchAuthorityRef {
		return result, ErrSupervisorAuthority
	}
	now := time.Now
	if s.Now != nil {
		now = s.Now
	}
	a := executiongraph.RetryAuthority{FormatVersion: 1, ProjectID: r.Preview.Request.Target.ProjectID, ParentID: g.Parent.ExecutionID, Revision: g.Parent.GraphRevision, GraphDigest: fmt.Sprintf("%x", g.StorageRevision), PreviewDigest: fresh.Digest, DispatchDigest: fresh.DispatchDigest, ChildID: fresh.Request.ChildID, PriorAttemptID: fresh.PriorAttemptID, AttemptNumber: fresh.AttemptNumber, Effects: fresh.Effects, Disposition: disposition, ActorKind: "human-operator", Actor: r.Actor, RecordedAt: now().UTC()}
	if _, err = executiongraph.CheckRetryDispatch(g, a); err != nil {
		return result, err
	}
	result.Status = "failure"
	ref, err := s.Store.CommitRetry(ctx, g, a)
	if err != nil {
		return result, err
	}
	digest, err := executiongraph.RetryAuthorityDigest(a)
	if err != nil || ref != digest {
		result.Status, result.Recovery = "recovery_required", "confirm the protected retry receipt"
		return result, errors.Join(ErrSupervisorRecovery, err)
	}
	result.Authority, result.AuthorityRef = &a, ref
	// The receipt is committed: later failures are recoverable, never success.
	if disposition != executiongraph.DispositionNotRequired {
		child := executiongraph.SuccessorChild{}
		for _, c := range g.Children {
			if c.ExecutionID == a.ChildID {
				child = c
			}
		}
		if err = s.Disposer.Dispose(ctx, g, child, disposition); err != nil {
			result.Status, result.Recovery = "recovery_required", "apply the recorded workspace disposition before resuming the retry"
			return result, err
		}
	}
	process, err := s.Starter.StartRetry(ctx, supervisor, ref)
	if err != nil {
		result.Status, result.Recovery = "recovery_required", "start the committed retry supervisor"
		return result, err
	}
	if process.PID <= 0 || process.StartIdentity == "" || process.GroupIdentity == "" {
		result.Status, result.Recovery = "recovery_required", "observe supervisor identity before resuming"
		return result, ErrSupervisorRecovery
	}
	result.Status, result.Recovery, result.Process = "success", "", &process
	return result, nil
}

// RunRetry dispatches exactly the attempt named by a committed retry receipt
// under the run lease, repeating all preparation checks. First-attempt
// authority still binds the graph; it is never used to authorize the retry.
func (s Supervisor) RunRetry(ctx context.Context, r SupervisorRequest, retryRef string) error {
	if e := ctx.Err(); e != nil {
		return e
	}
	store, ok := s.Store.(RetrySupervisorStore)
	if !ok || s.Preparation == nil || s.Runner == nil || s.AllocateID == nil {
		return ErrSupervisorAuthority
	}
	load := func() (executiongraph.SuccessorGraph, executiongraph.RetryAuthority, error) {
		g, a, err := store.Load(ctx, r)
		if err != nil {
			return g, executiongraph.RetryAuthority{}, err
		}
		if err = verifySupervisor(r, g, a); err != nil {
			return g, executiongraph.RetryAuthority{}, err
		}
		c, err := store.Contract(ctx, r)
		if err != nil {
			return g, executiongraph.RetryAuthority{}, err
		}
		c.Graph = &executiongraph.GraphDocument{Successor: &g}
		if _, err = Admit(StageRetry, c); err != nil {
			return g, executiongraph.RetryAuthority{}, err
		}
		retry, err := store.LoadRetry(ctx, r, retryRef)
		if err != nil {
			return g, retry, err
		}
		digest, err := executiongraph.RetryAuthorityDigest(retry)
		if err != nil || digest != retryRef {
			return g, retry, ErrSupervisorAuthority
		}
		if _, err = executiongraph.CheckRetryDispatch(g, retry); err != nil {
			return g, retry, ErrSupervisorAuthority
		}
		return g, retry, nil
	}
	g, _, err := load()
	if err != nil {
		return err
	}
	lease, err := s.Store.Acquire(ctx, g)
	if err != nil {
		return err
	}
	if lease == nil {
		return ErrSupervisorRecovery
	}
	defer lease.Close()
	if lease.Epoch() == 0 {
		return ErrSupervisorRecovery
	}
	g, retry, err := load()
	if err != nil {
		return err
	}
	if executiongraph.ValidateLeaseEpoch(g, lease.Epoch()) != nil {
		return ErrSupervisorRecovery
	}
	for _, c := range g.Children {
		if len(c.Attempts) > 0 && c.Attempts[len(c.Attempts)-1].Status == executiongraph.AttemptRunning {
			return ErrSupervisorRecovery
		}
	}
	records, err := s.Store.Records(ctx, g)
	if err != nil {
		return err
	}
	consumer, err := executiongraph.LoadSuccessorConsumer(executiongraph.GraphDocument{Successor: &g}, records)
	if err != nil {
		return err
	}
	if err = consumer.DispatchBatch([]string{retry.ChildID}, map[string]bool{retry.ChildID: true}); err != nil {
		return err
	}
	now := s.Now
	if now == nil {
		now = time.Now
	}
	interval := s.PollInterval
	if interval <= 0 {
		interval = 100 * time.Millisecond
	}
	check := func(g executiongraph.SuccessorGraph, id string) (uint32, error) {
		if id != retry.ChildID {
			return 0, ErrSupervisorAuthority
		}
		return executiongraph.CheckRetryDispatch(g, retry)
	}
	return s.runBatch(ctx, r, &g, check, lease, []string{retry.ChildID}, now, interval)
}
