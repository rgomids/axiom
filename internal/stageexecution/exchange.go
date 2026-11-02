package stageexecution

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/rgomids/axiom/internal/workflow"
)

// Exchange is the optional message contract shared by the Codex and Claude
// adapters (ADR-0037 §6). It moves bytes only: the engine, never the child,
// binds each candidate to the running attempt's launcher-issued identity and
// commits it through the principal writer. A Launcher that does not implement
// Exchange runs attempts without collaboration messages.
type Exchange interface {
	// Poll returns the untrusted candidates the child of r has published so far,
	// read only from r's own launcher-issued area. Repeated candidates are
	// expected; the engine deduplicates them.
	Poll(context.Context, LaunchRequest) ([]workflow.MessageCandidate, error)
	// Deliver hands an accepted message addressed to r's agent to the running
	// child of r. It must be idempotent per message ID.
	Deliver(context.Context, LaunchRequest, workflow.Message) error
}

// ingress is the engine-issued identity of one running attempt. The child
// cannot present another: it is derived from the persisted attempt.
func ingress(r LaunchRequest) workflow.IngressContext {
	return workflow.IngressContext{ExecutionID: r.Execution.ExecutionID, AttemptID: r.Attempt.AttemptID, AgentID: r.Ref.AgentID, Runtime: r.Attempt.Invocation.Runtime, Generation: r.Attempt.RunnerGeneration}
}

// messageRefusal reports a per-candidate disposition that changed nothing.
// It never stops the run or the attempt.
func messageRefusal(err error) bool {
	return errors.Is(err, workflow.ErrMessageRejected) || errors.Is(err, workflow.ErrMessageDuplicate)
}

// messagePump routes one running attempt's candidates into the canonical
// Execution and delivers accepted messages addressed to its agent. Messages
// are data: no pump step changes an attempt, a gate or readiness.
type messagePump struct {
	exchange Exchange
	request  LaunchRequest
	load     func() (workflow.Execution, error)
	update   func(func(*workflow.Execution) error) (workflow.Execution, error)
	now      func() time.Time
	// submitted maps a candidate ID to the exact bytes already submitted, so an
	// unchanged replay is not resubmitted while a changed one still is (and
	// is then refused as a conflicting duplicate by the aggregate).
	submitted map[string]string
	delivered map[string]bool
	stopped   bool
}

func (p *messagePump) step(ctx context.Context) {
	if p.stopped {
		return
	}
	candidates, err := p.exchange.Poll(ctx, p.request)
	if err != nil {
		// An unsafe or unreadable area fails closed for the remaining attempt;
		// nothing it contains is accepted. Stopping the periodic poll is not a
		// failure and still allows the final drain.
		p.stopped = ctx.Err() == nil
		return
	}
	identity := ingress(p.request)
	for _, c := range candidates {
		wire, err := json.Marshal(c)
		if err != nil || p.submitted[c.ID] == string(wire) {
			continue
		}
		p.submitted[c.ID] = string(wire)
		if _, err := p.update(func(x *workflow.Execution) error { return x.AcceptMessage(identity, c, p.now()) }); err != nil && !messageRefusal(err) {
			// A lost fence or store failure stops routing; the principal records it.
			p.stopped = true
			return
		}
	}
	x, err := p.load()
	if err != nil {
		return
	}
	for _, m := range x.Messages {
		if m.Recipient != p.request.Ref.AgentID || p.delivered[m.ID] {
			continue
		}
		if p.exchange.Deliver(ctx, p.request, m) != nil {
			return
		}
		p.delivered[m.ID] = true
	}
}

// startMessages runs the pump once the attempt's process identity is durable
// (the attempt is running). The returned function stops it; with drain set it
// performs one final step, so a candidate written just before a clean exit is
// still ingested before the terminal commit. Afterwards the aggregate refuses
// any candidate from the terminal attempt.
func (e Engine) startMessages(ctx context.Context, request LaunchRequest, update func(func(*workflow.Execution) error) (workflow.Execution, error)) func(drain bool) {
	exchange, ok := e.Launcher.(Exchange)
	if !ok {
		return func(bool) {}
	}
	p := &messagePump{exchange: exchange, request: request, update: update, now: e.now, submitted: map[string]string{}, delivered: map[string]bool{},
		load: func() (workflow.Execution, error) {
			return e.Store.Load(context.WithoutCancel(ctx), request.Execution.ProjectID, request.Execution.ExecutionID)
		}}
	poll := e.CancelPoll
	if poll == 0 {
		poll = 50 * time.Millisecond
	}
	pumpCtx, stop := context.WithCancel(ctx)
	finished := make(chan struct{})
	go func() {
		defer close(finished)
		ticker := time.NewTicker(poll)
		defer ticker.Stop()
		for {
			select {
			case <-pumpCtx.Done():
				return
			case <-ticker.C:
				p.step(pumpCtx)
			}
		}
	}()
	return func(drain bool) {
		stop()
		<-finished
		if drain {
			p.step(context.WithoutCancel(ctx))
		}
	}
}
