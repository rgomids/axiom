package stageexecution

import (
	"context"
	"errors"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"reflect"
	"time"

	"github.com/rgomids/axiom/internal/coordination"
	"github.com/rgomids/axiom/internal/executiongraph"
	"github.com/rgomids/axiom/internal/graphapplication"
	"github.com/rgomids/axiom/internal/workflow"
)

// SingleRequest is the only identity that crosses the hidden single-stage
// supervisor entrypoint: exact logical keys and one receipt digest.
type SingleRequest struct{ ProjectID, ExecutionID, ScopeDigest, AuthorityRef string }

// SinglePublication is one single-stage ledger generation, its receipt and the
// Execution3 state the receipt binds. Execution3 is never written by it.
type SinglePublication struct {
	Run       executiongraph.SingleStageRun
	Authority executiongraph.SingleStageAuthority
	Execution workflow.State
}

func (p SinglePublication) Request() SingleRequest {
	return SingleRequest{ProjectID: p.Run.Scope.ProjectID, ExecutionID: p.Run.Scope.ExecutionID, ScopeDigest: p.Run.ScopeDigest, AuthorityRef: p.Run.Authorities[len(p.Run.Authorities)-1]}
}

// SingleStore is the protected single-stage persistence and admission port.
// Contract reloads the current Execution3 and Runtime configuration.
type SingleStore interface {
	LoadSingle(context.Context, SingleRequest) (executiongraph.SingleStageRun, executiongraph.SingleStageAuthority, error)
	SingleContract(context.Context, SingleRequest) (Contract, error)
	AcquireSingle(context.Context, executiongraph.SingleStageRun) (SupervisorLease, error)
	SaveSingle(context.Context, executiongraph.SingleStageRun, SupervisorLease) (executiongraph.SingleStageRun, error)
}

// SinglePreparation revalidates the frozen tuple, policy, capability, effort
// sources, authentication and workspace, then builds the attempt inbox and
// task packet. It never resolves a replacement Profile.
type SinglePreparation interface {
	PrepareSingle(context.Context, executiongraph.SingleStageRun, string, uint64) (executiongraph.Invocation, error)
	DrainSingle(context.Context, executiongraph.SingleStageRun, executiongraph.SuccessorAttempt, bool) ([]executiongraph.AttemptRecord, error)
}

type SingleSupervisor struct {
	Store        SingleStore
	Preparation  SinglePreparation
	Runner       SupervisorRunner
	AllocateID   func() (string, error)
	Now          func() time.Time
	PollInterval time.Duration
}

func verifySingle(r SingleRequest, run executiongraph.SingleStageRun, a executiongraph.SingleStageAuthority) (uint32, error) {
	ref, err := executiongraph.SingleStageAuthorityDigest(a)
	if err != nil || ref != r.AuthorityRef || run.Scope.ProjectID != r.ProjectID || run.Scope.ExecutionID != r.ExecutionID || run.ScopeDigest != r.ScopeDigest {
		return 0, ErrSupervisorAuthority
	}
	n, err := executiongraph.CheckSingleStageDispatch(run, a)
	if err != nil {
		return 0, ErrSupervisorAuthority
	}
	return n, nil
}

// admitSingle requires the bound Execution3 generation, revision and stage to
// be current: a concurrent workflow change can never authorize a child effect.
func admitSingle(ctx context.Context, store SingleStore, r SingleRequest, run executiongraph.SingleStageRun) error {
	c, err := store.SingleContract(ctx, r)
	if err != nil {
		return err
	}
	e := c.Execution
	if e.ExecutionID != run.Scope.ExecutionID || e.ProjectID != run.Scope.ProjectID || e.Revision != run.Binding.WorkflowRevision || string(e.Stage) != run.Scope.StageID || e.StageOrdinal != run.Scope.StageOrdinal || fmt.Sprintf("%x", e.StorageRevision) != run.Binding.ExecutionGeneration {
		return ErrStalePreview
	}
	c.Graph = nil
	_, err = Admit(StageRun, c)
	return err
}

// Run consumes exactly the pending receipt under one lease. It never creates
// another receipt, retries, reconciles or integrates.
func (s SingleSupervisor) Run(ctx context.Context, r SingleRequest) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s.Store == nil || s.Preparation == nil || s.Runner == nil || s.AllocateID == nil {
		return ErrSupervisorAuthority
	}
	run, a, err := s.Store.LoadSingle(ctx, r)
	if err != nil {
		return err
	}
	if _, err = verifySingle(r, run, a); err != nil {
		return err
	}
	if err = admitSingle(ctx, s.Store, r, run); err != nil {
		return err
	}
	lease, err := s.Store.AcquireSingle(ctx, run)
	if err != nil {
		return err
	}
	if lease == nil || lease.Epoch() == 0 {
		return ErrSupervisorRecovery
	}
	defer lease.Close()
	reload := func() (executiongraph.SingleStageRun, uint32, error) {
		fresh, receipt, err := s.Store.LoadSingle(ctx, r)
		if err != nil {
			return fresh, 0, err
		}
		n, err := verifySingle(r, fresh, receipt)
		if err != nil {
			return fresh, 0, err
		}
		return fresh, n, admitSingle(ctx, s.Store, r, fresh)
	}
	run, number, err := reload()
	if err != nil {
		return err
	}
	attemptID, err := s.AllocateID()
	if err != nil {
		return err
	}
	inv, err := s.Preparation.PrepareSingle(ctx, run, attemptID, lease.Epoch())
	if err != nil {
		return err
	}
	fresh, again, err := reload()
	if err != nil {
		return err
	}
	if fresh.StorageRevision != run.StorageRevision || again != number {
		return ErrSupervisorRecovery
	}
	if len(inv.Stdin) == 0 || len(inv.Stdin) > executiongraph.MaxStdinBytes {
		return executiongraph.ErrInvalidInvocation
	}
	if err = lease.Validate(); err != nil {
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
	start := now().UTC()
	digest := sha256.Sum256(inv.Stdin)
	at := executiongraph.SuccessorAttempt{AttemptID: attemptID, Number: number, Status: executiongraph.AttemptRunning, StartedAt: &start, LeaseEpoch: lease.Epoch(), PacketDigest: hex.EncodeToString(digest[:]), ReconciliationFacts: []executiongraph.ReconciliationFact{}}
	return s.attempt(ctx, r, run, at, inv, lease, now, interval)
}

type singleEvent struct {
	identity *executiongraph.ProcessCorrelation
	result   *executiongraph.ProcessResult
	ack      chan error
}

func (s SingleSupervisor) attempt(ctx context.Context, r SingleRequest, run executiongraph.SingleStageRun, at executiongraph.SuccessorAttempt, inv executiongraph.Invocation, lease SupervisorLease, now func() time.Time, interval time.Duration) error {
	workCtx, stop := context.WithCancel(ctx)
	defer stop()
	events := make(chan singleEvent, 2)
	go func() {
		childCtx, cancel := context.WithTimeout(workCtx, run.Binding.Controls.Timeout)
		defer cancel()
		result := s.Runner.RunObserved(childCtx, inv, func(identity *executiongraph.ProcessCorrelation) error {
			ack := make(chan error, 1)
			events <- singleEvent{identity: identity, ack: ack}
			return <-ack
		})
		events <- singleEvent{result: &result}
	}()
	started := false
	var firstErr error
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	done := ctx.Done()
	for {
		select {
		case ev := <-events:
			if ev.ack != nil {
				err := s.saveStart(ctx, r, &run, &at, ev.identity, lease)
				ev.ack <- err
				if err != nil {
					firstErr = err
					stop()
				} else {
					started = true
				}
				continue
			}
			if !started {
				if firstErr == nil {
					firstErr = ErrSupervisorRecovery
				}
				return firstErr
			}
			if err := s.saveFinish(ctx, r, &run, at, *ev.result, lease, now); err != nil {
				return err
			}
			return firstErr
		case <-done:
			stop()
			done = nil
			if firstErr == nil {
				firstErr = ctx.Err()
			}
		case <-ticker.C:
			if !started {
				continue
			}
			fresh, _, err := s.Store.LoadSingle(ctx, r)
			if err == nil && fresh.CancelRequest != nil {
				run = fresh
				stop()
			} else if err == nil && !sameSingleExceptCancel(run, fresh) {
				err = ErrSupervisorRecovery
			}
			if err == nil {
				err = lease.Validate()
			}
			if err == nil {
				_, err = s.Preparation.DrainSingle(ctx, run, at, false)
			}
			if err != nil && firstErr == nil {
				firstErr = err
				stop()
			}
		}
	}
}

func sameSingleExceptCancel(a, b executiongraph.SingleStageRun) bool {
	a.CancelRequest, b.CancelRequest = nil, nil
	a.StorageRevision, b.StorageRevision = [32]byte{}, [32]byte{}
	return reflect.DeepEqual(a, b)
}

// loadCurrent accepts only a cancellation published under our lease; any
// other concurrent change requires recovery, never an overwrite.
func (s SingleSupervisor) loadCurrent(ctx context.Context, r SingleRequest, run executiongraph.SingleStageRun) (executiongraph.SingleStageRun, error) {
	fresh, _, err := s.Store.LoadSingle(ctx, r)
	if err != nil {
		return run, err
	}
	if !sameSingleExceptCancel(run, fresh) {
		return run, ErrSupervisorRecovery
	}
	return fresh, nil
}

func (s SingleSupervisor) saveStart(ctx context.Context, r SingleRequest, run *executiongraph.SingleStageRun, at *executiongraph.SuccessorAttempt, identity *executiongraph.ProcessCorrelation, lease SupervisorLease) error {
	if identity == nil {
		return ErrSupervisorRecovery
	}
	fresh, err := s.loadCurrent(ctx, r, *run)
	if err != nil {
		return err
	}
	if fresh.CancelRequest != nil {
		return ErrSupervisorAuthority
	}
	at.Process = identity
	candidate := executiongraph.CloneSingleStageRun(fresh)
	candidate.Attempts = append(candidate.Attempts, *at)
	saved, err := s.Store.SaveSingle(ctx, candidate, lease)
	if err != nil {
		return err
	}
	*run = saved
	return nil
}

func (s SingleSupervisor) saveFinish(ctx context.Context, r SingleRequest, run *executiongraph.SingleStageRun, at executiongraph.SuccessorAttempt, process executiongraph.ProcessResult, lease SupervisorLease, now func() time.Time) error {
	fresh, err := s.loadCurrent(ctx, r, *run)
	if err != nil {
		return err
	}
	records, err := s.Preparation.DrainSingle(ctx, fresh, at, true)
	if err != nil {
		return err
	}
	cancelled := fresh.CancelRequest != nil
	id := fresh.Scope.ExecutionID
	status, reason, ref := executiongraph.ClassifyAttempt(process, cancelled, executiongraph.AttemptCorrelation{ParentID: id, ChildID: id, AttemptID: at.AttemptID, GraphRevision: fresh.Binding.WorkflowRevision}, records)
	finish := now().UTC()
	at.Status, at.ClassificationReason, at.QualifyingRecordRef = status, reason, ref
	at.FinishedAt = &finish
	at.ExitCode = process.ExitCode
	at.AmbiguousEffect = process.EffectAmbiguous
	at.CancellationSeen = cancelled
	digest := sha256.Sum256(process.Output)
	at.OutputDigest = hex.EncodeToString(digest[:])
	at.ResultReference = at.OutputDigest
	candidate := executiongraph.CloneSingleStageRun(fresh)
	candidate.Attempts[len(candidate.Attempts)-1] = at
	saved, err := s.Store.SaveSingle(ctx, candidate, lease)
	if err != nil {
		return err
	}
	*run = saved
	return nil
}

// LaunchSingle verifies the committed pending receipt before spawning the
// same binary. The hidden process receives identities and one digest only.
func LaunchSingle(ctx context.Context, r SingleRequest, store SingleStore, start DetachedStart) (executiongraph.ProcessCorrelation, error) {
	if err := ctx.Err(); err != nil {
		return executiongraph.ProcessCorrelation{}, err
	}
	if store == nil || start == nil {
		return executiongraph.ProcessCorrelation{}, ErrSupervisorAuthority
	}
	run, a, err := store.LoadSingle(ctx, r)
	if err != nil {
		return executiongraph.ProcessCorrelation{}, err
	}
	if _, err = verifySingle(r, run, a); err != nil {
		return executiongraph.ProcessCorrelation{}, err
	}
	if err = admitSingle(ctx, store, r, run); err != nil {
		return executiongraph.ProcessCorrelation{}, err
	}
	return start.Start(ctx, []string{"__stage-supervisor", "single", r.ProjectID, r.ExecutionID, r.ScopeDigest, r.AuthorityRef})
}

// SingleAttemptLoader supplies trusted material for one attempt under the held
// lease: workspace validation and inbox creation belong to it.
type SingleAttemptLoader interface {
	LoadSingleAttempt(context.Context, executiongraph.SingleStageRun, string, uint64) (AttemptMaterial, error)
}

// CompiledSinglePreparation composes the same T21/T19/T20/T23 boundary as the
// graph preparation, bound to the immutable single-stage ledger binding.
type CompiledSinglePreparation struct {
	Loader   SingleAttemptLoader
	Checker  graphapplication.StageBindingChecker
	Sources  graphapplication.EffortSourceInspector
	Auth     graphapplication.SubscriptionAuthenticator
	attempts map[string]AttemptMaterial
}

func (p *CompiledSinglePreparation) PrepareSingle(ctx context.Context, run executiongraph.SingleStageRun, id string, epoch uint64) (executiongraph.Invocation, error) {
	if p == nil || p.Loader == nil || run.Binding.Resolution.ResolvedProfile == nil {
		return executiongraph.Invocation{}, ErrAdmission
	}
	m, err := p.Loader.LoadSingleAttempt(ctx, run, id, epoch)
	if err != nil {
		return executiongraph.Invocation{}, err
	}
	frozen := *run.Binding.Resolution.ResolvedProfile
	inputDigest, err := canonicalDigest(m.Input)
	if err != nil || inputDigest != run.Binding.InputDigest {
		return executiongraph.Invocation{}, ErrStalePreview
	}
	b, scope := run.Binding, run.Scope
	if m.Input.ResolvedProfile == nil || *m.Input.ResolvedProfile != frozen || m.Input.ProjectID != scope.ProjectID || m.Input.RepositoryKey != scope.RepositoryKey || m.Input.StageID != scope.StageID || !reflect.DeepEqual(m.Input.Effects, b.Effects) || m.Input.Controls != b.Controls || m.Invocation.Workspace != b.Workspace || m.Context.Workspace != b.Workspace || m.Context.Inbox != m.Invocation.Inbox || m.Packet.Inbox != m.Context.Inbox || m.Packet.AxiomPath != m.Invocation.AxiomPath || m.Packet.AxiomDigest != b.AxiomDigest || m.Context.ParentID != scope.ExecutionID || m.Context.ChildID != scope.ExecutionID || m.Context.GraphRevision != b.WorkflowRevision || m.Context.AttemptID != id || m.Packet.AttemptID != id || m.Context.Epoch != epoch || !coordination.ValidChildContext(m.Context) {
		return executiongraph.Invocation{}, ErrAdmission
	}
	m.Invocation.Process, m.Invocation.RepositoryWrite = false, false
	for _, effect := range b.Effects {
		switch effect.Kind {
		case "process":
			m.Invocation.Process = true
		case "repository-write":
			m.Invocation.RepositoryWrite = true
		}
	}
	inv, _, err := graphapplication.PrepareFrozenStageInvocation(ctx, m.Invocation, frozen, m.Input.RuntimePreview, p.Checker, p.Sources, p.Auth)
	if err != nil {
		return executiongraph.Invocation{}, err
	}
	inv, _, err = graphapplication.AttachCompiledTaskPacket(inv, m.Input, m.Packet)
	if err != nil {
		return executiongraph.Invocation{}, err
	}
	if p.attempts == nil {
		p.attempts = map[string]AttemptMaterial{}
	}
	if _, exists := p.attempts[id]; exists {
		return executiongraph.Invocation{}, ErrSupervisorRecovery
	}
	p.attempts[id] = m
	return inv, nil
}

func (p *CompiledSinglePreparation) DrainSingle(ctx context.Context, run executiongraph.SingleStageRun, a executiongraph.SuccessorAttempt, final bool) ([]executiongraph.AttemptRecord, error) {
	m, ok := p.attempts[a.AttemptID]
	if !ok || m.Context.ParentID != run.Scope.ExecutionID || m.Context.Epoch != a.LeaseEpoch {
		return nil, coordination.ErrChildContext
	}
	records, err := m.Ingestor.Drain(ctx, m.Context, final)
	if err == nil && final {
		delete(p.attempts, a.AttemptID)
	}
	return records, err
}

// resumeSingle is the committed-ledger branch of stage run. It starts a
// supervisor only for the pending receipt, after the same fresh re-checks; a
// running, unknown or ambiguous latest attempt needs reconciliation first.
func (s RunService) resumeSingle(ctx context.Context, r RunRequest) (RunResult, error) {
	result := RunResult{Status: "denied_authority"}
	if !r.AuthorizeLocal || !r.AuthorizeExternal || r.Actor == "" {
		return result, ErrDispatchAuthority
	}
	if s.SingleStore == nil || s.Starter == nil || s.Workspaces == nil || r.ExpectedRevision != r.Execution.Revision {
		return result, ErrStalePreview
	}
	request := *r.CommittedSingle
	run, a, err := s.SingleStore.LoadSingle(ctx, request)
	if err != nil {
		return result, err
	}
	if run.CancelRequest != nil {
		return result, ErrSupervisorAuthority
	}
	for _, at := range run.Attempts {
		if at.Status == executiongraph.AttemptRunning || at.Status == executiongraph.AttemptUnknown && len(at.ReconciliationFacts) == 0 {
			result.Status, result.Recovery = "recovery_required", "reconcile_required"
			return result, ErrSupervisorRecovery
		}
	}
	if _, err = verifySingle(request, run, a); err != nil {
		return result, err
	}
	if a.DispatchDigest != r.PreviewDigest {
		return result, ErrSupervisorAuthority
	}
	if err = admitSingle(ctx, s.SingleStore, request, run); err != nil {
		return result, err
	}
	check := func() error {
		fresh, err := s.Preview.Preview(ctx, r.Execution, r.StageID)
		if err != nil {
			return errors.Join(ErrStalePreview, err)
		}
		if fresh.Digest != a.DispatchDigest || fresh.Publication != nil {
			return ErrStalePreview
		}
		return nil
	}
	if err = check(); err != nil {
		return result, err
	}
	p := Publication{Single: &SinglePublication{Run: run, Authority: a}}
	result.Publication = p
	result.Status, result.Recovery = "recovery_required", "resume the committed single-stage receipt"
	if err = s.Workspaces.Prepare(ctx, p); err != nil {
		return result, err
	}
	if err = check(); err != nil {
		return result, err
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
