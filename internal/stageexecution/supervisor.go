package stageexecution

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"reflect"
	"time"

	"github.com/rgomids/axiom/internal/executiongraph"
)

var ErrSupervisorAuthority = errors.New("supervisor_authority_required")
var ErrSupervisorRecovery = errors.New("supervisor_recovery_required")

// Only logical, exact-bound identities cross the hidden entrypoint.
type SupervisorRequest struct{ ProjectID, ParentID, PreviewDigest, AuthorityRef string }

type SupervisorLease interface {
	Epoch() uint64
	Validate() error
	Close() error
}
type SupervisorStore interface {
	Contract(context.Context, SupervisorRequest) (Contract, error)
	Load(context.Context, SupervisorRequest) (executiongraph.SuccessorGraph, executiongraph.DispatchAuthority, error)
	Acquire(context.Context, executiongraph.SuccessorGraph) (SupervisorLease, error)
	Save(context.Context, executiongraph.SuccessorGraph, SupervisorLease) (executiongraph.SuccessorGraph, error)
	Records(context.Context, executiongraph.SuccessorGraph) ([]executiongraph.AttemptRecord, error)
}

// Prepare must revalidate the frozen tuple, policy, capability, effort sources,
// subscription auth and workspace using the existing dispatch adapters. It
// builds the attempt inbox and task packet, without resolving a replacement.
type SupervisorPreparation interface {
	Prepare(context.Context, executiongraph.SuccessorGraph, executiongraph.SuccessorChild, string, uint64) (executiongraph.Invocation, error)
	Drain(context.Context, executiongraph.SuccessorGraph, executiongraph.SuccessorChild, executiongraph.SuccessorAttempt, bool) ([]executiongraph.AttemptRecord, error)
}

// Started is called before waiting for the process. Failure must stop the
// containment; an identity/save failure is never treated as successful work.
type SupervisorRunner interface {
	RunObserved(context.Context, executiongraph.Invocation, func(*executiongraph.ProcessCorrelation) error) executiongraph.ProcessResult
}
type Supervisor struct {
	Store        SupervisorStore
	Preparation  SupervisorPreparation
	Runner       SupervisorRunner
	AllocateID   func() (string, error)
	Now          func() time.Time
	PollInterval time.Duration
}

func admitSupervisor(ctx context.Context, store SupervisorStore, r SupervisorRequest, g executiongraph.SuccessorGraph) error {
	c, e := store.Contract(ctx, r)
	if e != nil {
		return e
	}
	// The authoritative loaded graph, never a caller-supplied version claim.
	c.Graph = &executiongraph.GraphDocument{Successor: &g}
	_, e = Admit(StageRun, c)
	return e
}

func verifySupervisor(r SupervisorRequest, g executiongraph.SuccessorGraph, a executiongraph.DispatchAuthority) error {
	if g.Parent.ExecutionID != r.ParentID || len(g.Children) == 0 || g.Children[0].Envelope.Scope.ProjectID != r.ProjectID || g.DispatchAuthorityRef != r.AuthorityRef || executiongraph.ValidateGraphDispatchAuthority(g, a, r.PreviewDigest) != nil {
		return ErrSupervisorAuthority
	}
	return nil
}

// Run owns one lease and only first attempts. Interrupted saves leave persisted
// running state for explicit recovery; no retry, reconciliation or integration
// port exists here. The caller's context is local to the detached process.
func (s Supervisor) Run(ctx context.Context, r SupervisorRequest) error {
	if e := ctx.Err(); e != nil {
		return e
	}
	if s.Store == nil || s.Preparation == nil || s.Runner == nil || s.AllocateID == nil {
		return ErrSupervisorAuthority
	}
	g, a, err := s.Store.Load(ctx, r)
	if err != nil {
		return err
	}
	if err = verifySupervisor(r, g, a); err != nil {
		return err
	}
	if err = admitSupervisor(ctx, s.Store, r, g); err != nil {
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
	// Reload under the lease, before dispatch or any inbox creation.
	g, a, err = s.Store.Load(ctx, r)
	if err != nil {
		return err
	}
	if err = verifySupervisor(r, g, a); err != nil {
		return err
	}
	if err = admitSupervisor(ctx, s.Store, r, g); err != nil {
		return err
	}
	for _, c := range g.Children {
		for _, at := range c.Attempts {
			if at.Status == executiongraph.AttemptRunning {
				return ErrSupervisorRecovery
			}
		}
	}
	now := s.Now
	if now == nil {
		now = time.Now
	}
	interval := s.PollInterval
	if interval <= 0 {
		interval = 100 * time.Millisecond
	}
	for {
		if err = ctx.Err(); err != nil {
			return err
		}
		if err = lease.Validate(); err != nil {
			return err
		}
		if g.CancelRequest != nil {
			return nil
		}
		records, e := s.Store.Records(ctx, g)
		if e != nil {
			return e
		}
		consumer, e := executiongraph.LoadSuccessorConsumer(executiongraph.GraphDocument{Successor: &g}, records)
		if e != nil {
			return e
		}
		ready, _, e := consumer.Ready(nil)
		if e != nil {
			return e
		}
		var batch []string
		for _, id := range ready {
			if executiongraph.CheckFirstDispatch(g, a, r.PreviewDigest, id, 1) != nil {
				continue
			}
			proposed := append(append([]string(nil), batch...), id)
			if consumer.DispatchBatch(proposed, nil) == nil {
				batch = proposed
			}
		}
		if len(batch) == 0 {
			return nil
		}
		if err = s.runBatch(ctx, r, &g, a, lease, batch, now, interval); err != nil {
			return err
		}
		// Refresh cancellation and CAS state. Never overwrite a concurrent request.
		g, a, err = s.Store.Load(ctx, r)
		if err != nil {
			return err
		}
		if err = verifySupervisor(r, g, a); err != nil {
			return err
		}
	}
}

type supervisorEvent struct {
	index    int
	attempt  executiongraph.SuccessorAttempt
	process  executiongraph.ProcessResult
	identity *executiongraph.ProcessCorrelation
	ack      chan error
}

func (s Supervisor) runBatch(ctx context.Context, r SupervisorRequest, g *executiongraph.SuccessorGraph, a executiongraph.DispatchAuthority, lease SupervisorLease, batch []string, now func() time.Time, interval time.Duration) error {
	workCtx, stop := context.WithCancel(ctx)
	defer stop()
	events := make(chan supervisorEvent, len(batch)*2)
	active := map[int]executiongraph.SuccessorAttempt{}
	pending := 0
	var firstErr error
	// Start each child after its own fresh preparation, not a batch-wide check.
	for _, id := range batch {
		if firstErr != nil {
			break
		}
		fresh, receipt, e := s.Store.Load(ctx, r)
		if e == nil {
			e = verifySupervisor(r, fresh, receipt)
		}
		if e != nil {
			firstErr = e
			break
		}
		if fresh.CancelRequest != nil {
			*g = fresh
			stop()
			break
		}
		if fresh.StorageRevision != g.StorageRevision {
			firstErr = ErrSupervisorRecovery
			break
		}
		index := -1
		for i, c := range g.Children {
			if c.ExecutionID == id {
				index = i
				break
			}
		}
		if index < 0 || executiongraph.CheckFirstDispatch(*g, a, r.PreviewDigest, id, 1) != nil {
			firstErr = ErrSupervisorAuthority
			break
		}
		attemptID, e := s.AllocateID()
		if e != nil {
			firstErr = e
			break
		}
		inv, e := s.Preparation.Prepare(ctx, *g, g.Children[index], attemptID, lease.Epoch())
		if e != nil {
			firstErr = e
			break
		}
		// Preparation can observe drift or a cancellation published while auth
		// was checked. Reload immediately before spawn, without replacing tuples.
		fresh, receipt, e = s.Store.Load(ctx, r)
		if e == nil {
			e = verifySupervisor(r, fresh, receipt)
		}
		if e != nil {
			firstErr = e
			break
		}
		if fresh.CancelRequest != nil {
			*g = fresh
			stop()
			break
		}
		if fresh.StorageRevision != g.StorageRevision {
			firstErr = ErrSupervisorRecovery
			break
		}
		if len(inv.Stdin) == 0 || len(inv.Stdin) > executiongraph.MaxStdinBytes {
			firstErr = executiongraph.ErrInvalidInvocation
			break
		}
		if e = lease.Validate(); e != nil {
			firstErr = e
			break
		}
		start := now().UTC()
		digest := sha256.Sum256(inv.Stdin)
		at := executiongraph.SuccessorAttempt{AttemptID: attemptID, Number: 1, Status: executiongraph.AttemptRunning, StartedAt: &start, LeaseEpoch: lease.Epoch(), PacketDigest: hex.EncodeToString(digest[:])}
		child := g.Children[index]
		pending++
		go func(index int, at executiongraph.SuccessorAttempt, inv executiongraph.Invocation) {
			childCtx, cancel := context.WithTimeout(workCtx, child.Envelope.Controls.Timeout)
			defer cancel()
			result := s.Runner.RunObserved(childCtx, inv, func(identity *executiongraph.ProcessCorrelation) error {
				ack := make(chan error, 1)
				events <- supervisorEvent{index: index, attempt: at, identity: identity, ack: ack}
				return <-ack
			})
			events <- supervisorEvent{index: index, attempt: at, process: result}
		}(index, at, inv)
		// Serialize the initial save before preparing the next child, while already
		// started children run concurrently. Terminal events may arrive first.
		waiting := true
		for waiting {
			ev := <-events
			if ev.index == index {
				waiting = false
			}
			if e = s.handleEvent(ctx, r, g, lease, ev, active, now); e != nil && firstErr == nil {
				firstErr = e
				stop()
			}
			if ev.ack == nil {
				pending--
			}
		}
	}
	if firstErr != nil {
		stop()
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	done := ctx.Done()
	for pending > 0 {
		select {
		case ev := <-events:
			if e := s.handleEvent(ctx, r, g, lease, ev, active, now); e != nil && firstErr == nil {
				firstErr = e
				stop()
			}
			if ev.ack == nil {
				pending--
			}
		case <-done:
			stop()
			done = nil
			if firstErr == nil {
				firstErr = ctx.Err()
			}
		case <-ticker.C:
			fresh, receipt, e := s.Store.Load(ctx, r)
			if e == nil {
				e = verifySupervisor(r, fresh, receipt)
			}
			if e == nil && fresh.CancelRequest != nil {
				*g = fresh
				stop()
			} else if e == nil && fresh.StorageRevision != g.StorageRevision {
				e = ErrSupervisorRecovery
			}
			if e == nil {
				e = lease.Validate()
			}
			if e == nil {
				for index, at := range active {
					_, e = s.Preparation.Drain(ctx, *g, g.Children[index], at, false)
					if e != nil {
						break
					}
				}
			}
			if e != nil && firstErr == nil {
				firstErr = e
				stop()
			}
		}
	}
	return firstErr
}

func (s Supervisor) handleEvent(ctx context.Context, r SupervisorRequest, g *executiongraph.SuccessorGraph, lease SupervisorLease, ev supervisorEvent, active map[int]executiongraph.SuccessorAttempt, now func() time.Time) error {
	// Observe cancellation even when completion wins the race with the poll.
	// Only the append-only control message may change under our lease.
	fresh, receipt, err := s.Store.Load(ctx, r)
	if err == nil {
		err = verifySupervisor(r, fresh, receipt)
	}
	if err == nil {
		before, after := *g, fresh
		before.CancelRequest, after.CancelRequest = nil, nil
		before.StorageRevision, after.StorageRevision = [32]byte{}, [32]byte{}
		if !reflect.DeepEqual(before, after) {
			err = ErrSupervisorRecovery
		}
	}
	if err != nil {
		if ev.ack != nil {
			ev.ack <- err
		}
		return err
	}
	*g = fresh
	if ev.ack != nil {
		at := ev.attempt
		at.Process = ev.identity
		if ev.identity == nil {
			ev.ack <- ErrSupervisorRecovery
			return ErrSupervisorRecovery
		}
		candidate := cloneSupervisorGraph(*g)
		candidate.Children[ev.index].Attempts = append(candidate.Children[ev.index].Attempts, at)
		saved, e := s.Store.Save(ctx, candidate, lease)
		if e == nil {
			*g = saved
			active[ev.index] = at
		}
		ev.ack <- e
		return e
	}
	at, ok := active[ev.index]
	if !ok {
		return ErrSupervisorRecovery
	}
	records, e := s.Preparation.Drain(ctx, *g, g.Children[ev.index], at, true)
	if e != nil {
		return e
	}
	cancelled := g.CancelRequest != nil
	status, reason, ref := executiongraph.ClassifyAttempt(ev.process, cancelled, executiongraph.AttemptCorrelation{ParentID: g.Parent.ExecutionID, ChildID: g.Children[ev.index].ExecutionID, AttemptID: at.AttemptID, GraphRevision: g.Parent.GraphRevision}, records)
	finish := now().UTC()
	at.Status = status
	at.ClassificationReason = reason
	at.QualifyingRecordRef = ref
	at.FinishedAt = &finish
	at.ExitCode = ev.process.ExitCode
	at.AmbiguousEffect = ev.process.EffectAmbiguous
	at.CancellationSeen = cancelled
	digest := sha256.Sum256(ev.process.Output)
	at.OutputDigest = hex.EncodeToString(digest[:])
	at.ResultReference = at.OutputDigest
	candidate := cloneSupervisorGraph(*g)
	candidate.Children[ev.index].Attempts[len(candidate.Children[ev.index].Attempts)-1] = at
	saved, e := s.Store.Save(ctx, candidate, lease)
	if e != nil {
		return e
	}
	*g = saved
	delete(active, ev.index)
	return nil
}
func cloneSupervisorGraph(g executiongraph.SuccessorGraph) executiongraph.SuccessorGraph {
	g.Children = append([]executiongraph.SuccessorChild(nil), g.Children...)
	for i := range g.Children {
		g.Children[i].Attempts = append([]executiongraph.SuccessorAttempt(nil), g.Children[i].Attempts...)
	}
	return g
}
