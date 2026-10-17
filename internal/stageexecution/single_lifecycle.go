package stageexecution

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/rgomids/axiom/internal/executiongraph"
	"github.com/rgomids/axiom/internal/workflow"
	"github.com/rgomids/axiom/internal/workflowcompiler"
)

var ErrStageKindMismatch = errors.New("single_stage_operation_unsupported")

// SingleLedger is the protected port of one resolved single-stage target. The
// snapshot is the admission contract plus the current ledger; writes are the
// store's CAS operations and never change attempt classification.
type SingleLedger interface {
	SingleSnapshot(context.Context) (Contract, executiongraph.SingleStageRun, error)
	RequestSingleCancel(context.Context, executiongraph.SingleStageRun, executiongraph.CancelRequest) (executiongraph.SingleStageRun, error)
	AppendSingleReconciliation(context.Context, executiongraph.SingleStageRun, executiongraph.ReconciliationFact) (executiongraph.SingleStageRun, error)
	CommitSingleRetry(context.Context, executiongraph.SingleStageRun, executiongraph.SingleStageAuthority, workflow.State) error
}

// SingleObserver gathers the same bounded digest-only observations as the
// graph observer; the store repeats it under the run lease at apply.
type SingleObserver interface {
	ObserveSingle(context.Context, executiongraph.SingleStageRun, string) (executiongraph.ReconciliationObservations, error)
}

// SingleDisposer applies an explicit, authorized workspace disposition after
// the retry receipt is committed and before its supervisor starts.
type SingleDisposer interface {
	DisposeSingle(context.Context, executiongraph.SingleStageRun, executiongraph.SingleStageAuthority) error
}

type SingleLifecycle struct {
	Ledger     SingleLedger
	Observer   SingleObserver
	Dispatch   PreviewService
	Disposer   SingleDisposer
	Store      SingleStore
	Detached   DetachedStart
	AllocateID func() (string, error)
	Now        func() time.Time
}

func (s SingleLifecycle) now() time.Time {
	if s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}

func (s SingleLifecycle) snapshot(ctx context.Context, operation Operation) (Contract, executiongraph.SingleStageRun, error) {
	if s.Ledger == nil {
		return Contract{}, executiongraph.SingleStageRun{}, ErrPreview
	}
	c, run, err := s.Ledger.SingleSnapshot(ctx)
	if err != nil {
		return c, run, err
	}
	c.Graph = nil
	if _, err = Admit(operation, c); err != nil {
		return c, run, err
	}
	if c.Execution.ExecutionID != run.Scope.ExecutionID || c.Execution.ProjectID != run.Scope.ProjectID {
		return c, run, ErrAdmission
	}
	return c, run, nil
}

// SingleStatusResult renders persisted ledger fields plus non-authoritative
// liveness observations; it never claims a stop or a success.
type SingleStatusResult struct {
	Run    executiongraph.SingleStageRun           `json:"single"`
	Health map[string]executiongraph.AttemptHealth `json:"health"`
	Next   string                                  `json:"next,omitempty"`
}

func (s SingleLifecycle) Status(ctx context.Context, observer StatusObservation) (SingleStatusResult, error) {
	_, run, err := s.snapshot(ctx, StageStatus)
	if err != nil {
		return SingleStatusResult{}, err
	}
	result := SingleStatusResult{Run: run, Health: map[string]executiongraph.AttemptHealth{}}
	pending, blocker := executiongraph.SingleStageReadiness(run)
	switch {
	case pending:
		result.Next = "resume_with_stage_run"
	case blocker == string(executiongraph.AttemptUnknown):
		result.Next = "reconcile_required"
	case blocker == string(executiongraph.AttemptSucceeded):
		result.Next = "advance_execution"
	default:
		result.Next = blocker
	}
	if ok, _ := executiongraph.SingleStageRetryEligible(run); ok {
		result.Next = "retry_available"
	}
	request := StatusRequest{ProjectID: run.Scope.ProjectID, ParentID: run.Scope.ExecutionID}
	for _, at := range run.Attempts {
		if at.Status != executiongraph.AttemptRunning {
			continue
		}
		if observer == nil {
			result.Health[at.AttemptID] = executiongraph.AttemptHealth{LivenessUnknown: true}
			continue
		}
		held, err := observer.LeaseHeld(ctx, request)
		if err != nil {
			result.Health[at.AttemptID] = executiongraph.AttemptHealth{LivenessUnknown: true}
			continue
		}
		var observed *executiongraph.ProcessCorrelation
		var observationErr error
		if !held && at.Process != nil {
			observed, observationErr = observer.Process(ctx, *at.Process)
		}
		h := executiongraph.ObserveAttemptHealth(at, held, observed, observationErr)
		result.Health[at.AttemptID] = h
		if h.Stale {
			result.Next = "reconcile_required"
		}
		if h.Orphan {
			result.Next = "stop_orphan_then_reconcile"
		}
	}
	return result, nil
}

type SingleOperationPreview struct {
	Operation       Operation                                  `json:"operation"`
	ScopeDigest     string                                     `json:"scopeDigest"`
	StorageRevision string                                     `json:"storageRevision"`
	AttemptID       string                                     `json:"attemptId,omitempty"`
	Status          executiongraph.AttemptStatus               `json:"status,omitempty"`
	Observations    *executiongraph.ReconciliationObservations `json:"observations,omitempty"`
	Classification  string                                     `json:"classification,omitempty"`
	AttemptNumber   uint32                                     `json:"attemptNumber,omitempty"`
	Remaining       uint32                                     `json:"remaining,omitempty"`
	Disposition     bool                                       `json:"dispositionRequired,omitempty"`
	DispatchDigest  string                                     `json:"dispatchDigest,omitempty"`
	Effects         []executiongraph.Effect                    `json:"effects,omitempty"`
	Next            string                                     `json:"next,omitempty"`
	Digest          string                                     `json:"digest"`
}

func singleDigest(p SingleOperationPreview, run executiongraph.SingleStageRun) (string, error) {
	p.Digest = ""
	return canonicalDigest(struct {
		Preview SingleOperationPreview
		Run     executiongraph.SingleStageRun
	}{p, run})
}

func newSinglePreview(operation Operation, run executiongraph.SingleStageRun) SingleOperationPreview {
	return SingleOperationPreview{Operation: operation, ScopeDigest: run.ScopeDigest, StorageRevision: fmt.Sprintf("%x", run.StorageRevision)}
}

func (s SingleLifecycle) CancelPreview(ctx context.Context) (SingleOperationPreview, error) {
	_, run, err := s.snapshot(ctx, StageCancel)
	if err != nil {
		return SingleOperationPreview{}, err
	}
	p := newSinglePreview(StageCancel, run)
	p.Digest, err = singleDigest(p, run)
	return p, err
}

// Cancel publishes only the write-once control message. Observing it and
// confirming a stop belongs to the supervisor holding the run lease.
func (s SingleLifecycle) Cancel(ctx context.Context, reviewed string, authorizeLocal bool, actor string) (*executiongraph.CancelRequest, error) {
	if !authorizeLocal || actor == "" {
		return nil, ErrDispatchAuthority
	}
	fresh, err := s.CancelPreview(ctx)
	if err != nil {
		return nil, err
	}
	if fresh.Digest != reviewed {
		return nil, ErrStalePreview
	}
	_, run, err := s.snapshot(ctx, StageCancel)
	if err != nil {
		return nil, err
	}
	if run.CancelRequest != nil {
		request := *run.CancelRequest
		return &request, nil
	}
	if s.AllocateID == nil {
		return nil, ErrPreview
	}
	id, err := s.AllocateID()
	if err != nil {
		return nil, err
	}
	request := executiongraph.CancelRequest{RequestID: id, ParentID: run.Scope.ExecutionID, GraphRevision: run.Binding.WorkflowRevision, ActorKind: "human-operator", Actor: actor, RequestedAt: s.now()}
	if request.RequestedAt.Before(run.CreatedAt) {
		request.RequestedAt = run.CreatedAt
	}
	request.AuthorityRef, err = canonicalDigest(struct {
		Operation Operation
		Preview   string
		Request   executiongraph.CancelRequest
	}{StageCancel, reviewed, request})
	if err != nil {
		return nil, err
	}
	saved, err := s.Ledger.RequestSingleCancel(ctx, run, request)
	if err != nil {
		return nil, err
	}
	if saved.CancelRequest == nil || *saved.CancelRequest != request {
		return nil, ErrSupervisorRecovery
	}
	return &request, nil
}

func (s SingleLifecycle) reconcileTarget(ctx context.Context, attemptID string) (executiongraph.SingleStageRun, executiongraph.SuccessorAttempt, error) {
	_, run, err := s.snapshot(ctx, StageReconcile)
	if err != nil {
		return run, executiongraph.SuccessorAttempt{}, err
	}
	for i, a := range run.Attempts {
		if a.AttemptID != attemptID {
			continue
		}
		if i != len(run.Attempts)-1 || a.Status != executiongraph.AttemptRunning && a.Status != executiongraph.AttemptUnknown && !a.AmbiguousEffect {
			return run, a, ErrReconcileIneligible
		}
		return run, a, nil
	}
	return run, executiongraph.SuccessorAttempt{}, ErrAdmission
}

func (s SingleLifecycle) ReconcilePreview(ctx context.Context, attemptID string) (SingleOperationPreview, error) {
	p, _, err := s.reconcilePreview(ctx, attemptID)
	return p, err
}

func (s SingleLifecycle) reconcilePreview(ctx context.Context, attemptID string) (SingleOperationPreview, executiongraph.SingleStageRun, error) {
	if s.Observer == nil {
		return SingleOperationPreview{}, executiongraph.SingleStageRun{}, ErrPreview
	}
	run, a, err := s.reconcileTarget(ctx, attemptID)
	if err != nil {
		return SingleOperationPreview{}, run, err
	}
	o, err := s.Observer.ObserveSingle(ctx, run, attemptID)
	if err != nil {
		return SingleOperationPreview{}, run, err
	}
	if o.ParentID != run.Scope.ExecutionID || o.ChildID != run.Scope.ExecutionID || o.AttemptID != attemptID || o.GraphRevision != run.Binding.WorkflowRevision || o.LeaseEpoch != a.LeaseEpoch {
		return SingleOperationPreview{}, run, ErrStalePreview
	}
	p := newSinglePreview(StageReconcile, run)
	p.AttemptID, p.Status, p.Observations, p.Classification = attemptID, a.Status, &o, "recovery_required"
	switch {
	case o.LeaseHeld:
		p.Next = ErrReconcileLeaseHeld.Error()
	case o.ProcessAlive:
		p.Next = ErrReconcileOrphanAlive.Error()
	case o.ProcessUnknown:
		p.Next = "process_liveness_unknown"
	default:
		p.Classification, p.Next = "retry_safe", "explicit_retry_required"
	}
	p.Digest, err = singleDigest(p, run)
	return p, run, err
}

// Reconcile appends one fact; it never upgrades status or authorizes a retry.
func (s SingleLifecycle) Reconcile(ctx context.Context, attemptID, reviewed string, authorizeLocal bool, actor string) (*executiongraph.ReconciliationFact, error) {
	if !authorizeLocal || actor == "" {
		return nil, ErrDispatchAuthority
	}
	fresh, run, err := s.reconcilePreview(ctx, attemptID)
	if err != nil {
		return nil, err
	}
	if fresh.Digest != reviewed {
		return nil, ErrStalePreview
	}
	o := *fresh.Observations
	if o.LeaseHeld {
		return nil, ErrReconcileLeaseHeld
	}
	if o.ProcessAlive {
		return nil, ErrReconcileOrphanAlive
	}
	if s.AllocateID == nil {
		return nil, ErrPreview
	}
	id, err := s.AllocateID()
	if err != nil {
		return nil, err
	}
	digest, err := o.Digest()
	if err != nil {
		return nil, err
	}
	f := executiongraph.ReconciliationFact{FactID: id, ParentID: run.Scope.ExecutionID, ChildID: run.Scope.ExecutionID, GraphRevision: run.Binding.WorkflowRevision, AttemptID: attemptID, ObservationsDigest: digest, Classification: fresh.Classification, ActorKind: "human-operator", Actor: actor, RecordedAt: s.now()}
	f.AuthorityRef, err = canonicalDigest(struct {
		Operation Operation
		Preview   string
		Fact      executiongraph.ReconciliationFact
	}{StageReconcile, reviewed, f})
	if err != nil {
		return nil, err
	}
	if _, err = executiongraph.AppendSingleStageReconciliation(run, f, o); err != nil {
		return nil, err
	}
	saved, err := s.Ledger.AppendSingleReconciliation(ctx, run, f)
	if err != nil {
		return nil, err
	}
	for _, a := range saved.Attempts {
		for _, got := range a.ReconciliationFacts {
			if got.FactID == f.FactID && got.ObservationsDigest == f.ObservationsDigest {
				return &f, nil
			}
		}
	}
	return nil, ErrSupervisorRecovery
}

func (s SingleLifecycle) retryPreview(ctx context.Context, ref workflowcompiler.ExecutionRef, stage string) (SingleOperationPreview, Contract, executiongraph.SingleStageRun, error) {
	c, run, err := s.snapshot(ctx, StageRetry)
	if err != nil {
		return SingleOperationPreview{}, c, run, err
	}
	if len(run.Attempts) == 0 || len(run.Authorities) != len(run.Attempts) {
		return SingleOperationPreview{}, c, run, ErrRetryIneligible
	}
	used := uint32(len(run.Attempts))
	if used >= run.Binding.Controls.MaximumAttempts {
		return SingleOperationPreview{}, c, run, ErrRetryExhausted
	}
	eligible, err := executiongraph.SingleStageRetryEligible(run)
	if err != nil {
		return SingleOperationPreview{}, c, run, err
	}
	if !eligible {
		return SingleOperationPreview{}, c, run, ErrRetryIneligible
	}
	dispatch, err := s.Dispatch.Preview(ctx, ref, stage)
	if err != nil {
		return SingleOperationPreview{}, c, run, errors.Join(ErrStalePreview, err)
	}
	if dispatch.Publication != nil || dispatch.Plan.Digest != run.Binding.ProposalDigest || len(dispatch.Plan.Compilation.Agents) != 1 {
		return SingleOperationPreview{}, c, run, ErrStalePreview
	}
	frozen := dispatch.Plan.Compilation.Agents[0].Input.ResolvedProfile
	if frozen == nil || *frozen != *run.Binding.Resolution.ResolvedProfile {
		return SingleOperationPreview{}, c, run, ErrStalePreview
	}
	prior := run.Attempts[len(run.Attempts)-1]
	p := newSinglePreview(StageRetry, run)
	p.AttemptID, p.Status, p.AttemptNumber, p.Remaining = prior.AttemptID, prior.Status, used+1, run.Binding.Controls.MaximumAttempts-used
	p.Disposition, p.DispatchDigest, p.Effects = executiongraph.SingleStageRetryDispositionRequired(run), dispatch.Digest, run.Binding.Effects
	p.Digest, err = singleDigest(p, run)
	return p, c, run, err
}

func (s SingleLifecycle) RetryPreview(ctx context.Context, ref workflowcompiler.ExecutionRef, stage string) (SingleOperationPreview, error) {
	p, _, _, err := s.retryPreview(ctx, ref, stage)
	return p, err
}

type SingleRetryResult struct {
	Status, Recovery string
	Authority        *executiongraph.SingleStageAuthority
	AuthorityRef     string
	Process          *executiongraph.ProcessCorrelation
}

// Retry commits a separate single-use receipt for exactly the next attempt,
// applies the explicit disposition, then launches its supervisor. Counters
// are never reset and no retry is automatic.
func (s SingleLifecycle) Retry(ctx context.Context, ref workflowcompiler.ExecutionRef, stage, reviewed string, authorizeLocal, authorizeExternal bool, actor, disposition string) (SingleRetryResult, error) {
	result := SingleRetryResult{Status: "denied_authority"}
	if !authorizeLocal || !authorizeExternal || actor == "" {
		return result, ErrDispatchAuthority
	}
	if s.Disposer == nil || s.Store == nil || s.Detached == nil {
		return result, ErrPreview
	}
	fresh, c, run, err := s.retryPreview(ctx, ref, stage)
	if err != nil {
		return result, err
	}
	if fresh.Digest != reviewed {
		return result, ErrStalePreview
	}
	switch {
	case fresh.Disposition && disposition != executiongraph.DispositionPreserve && disposition != executiongraph.DispositionReset:
		return result, ErrDispositionRequired
	case !fresh.Disposition && disposition != "" && disposition != executiongraph.DispositionNotRequired:
		return result, ErrDispositionRequired
	case !fresh.Disposition:
		disposition = executiongraph.DispositionNotRequired
	}
	b := run.Binding
	a := executiongraph.SingleStageAuthority{FormatVersion: executiongraph.SingleStageAuthorityVersion, Scope: executiongraph.SingleStageAuthorityScope, ProjectID: run.Scope.ProjectID, ExecutionID: run.Scope.ExecutionID, StageID: run.Scope.StageID, StageOrdinal: run.Scope.StageOrdinal, ScopeDigest: run.ScopeDigest, ExecutionGeneration: b.ExecutionGeneration, WorkflowRevision: b.WorkflowRevision, LedgerGeneration: fmt.Sprintf("%x", run.StorageRevision), PreviewDigest: b.PreviewDigest, DispatchDigest: fresh.DispatchDigest, ProposalDigest: b.ProposalDigest, AttemptNumber: fresh.AttemptNumber, PriorAttemptID: fresh.AttemptID, Effects: b.Effects, Disposition: disposition, ActorKind: "human-operator", Actor: actor, RecordedAt: s.now()}
	if err = executiongraph.CheckSingleStageRetryAuthority(run, a); err != nil {
		return result, err
	}
	authorityRef, err := executiongraph.SingleStageAuthorityDigest(a)
	if err != nil {
		return result, err
	}
	next := executiongraph.CloneSingleStageRun(run)
	next.Authorities = append(next.Authorities, authorityRef)
	result.Status = "failure"
	if err = s.Ledger.CommitSingleRetry(ctx, next, a, c.Execution); err != nil {
		var committed interface{ EffectCommitted() bool }
		if errors.As(err, &committed) && committed.EffectCommitted() {
			result.Status, result.Recovery = "recovery_required", "confirm the protected retry receipt"
		}
		return result, err
	}
	result.Authority, result.AuthorityRef = &a, authorityRef
	if disposition != executiongraph.DispositionNotRequired {
		if err = s.Disposer.DisposeSingle(ctx, next, a); err != nil {
			result.Status, result.Recovery = "recovery_required", "inspect the committed retry authority and disposition failure before any retry start"
			return result, err
		}
	}
	process, err := LaunchSingle(ctx, SingleRequest{ProjectID: run.Scope.ProjectID, ExecutionID: run.Scope.ExecutionID, ScopeDigest: run.ScopeDigest, AuthorityRef: authorityRef}, s.Store, s.Detached)
	if err != nil {
		result.Status, result.Recovery = "recovery_required", "resume the committed retry receipt with stage run"
		return result, err
	}
	if process.PID <= 0 || process.StartIdentity == "" || process.GroupIdentity == "" {
		result.Status, result.Recovery = "recovery_required", "observe supervisor identity before resuming"
		return result, ErrSupervisorRecovery
	}
	result.Status, result.Recovery, result.Process = "success", "", &process
	return result, nil
}
