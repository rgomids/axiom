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

// Exchange faults are fixed engine codes recorded as the attempt diagnostic.
// No byte of a candidate or delivery reaches them.
const (
	// The attempt's own send area is unsafe or flooded: persistent, so routing
	// stops for the remaining attempt and nothing it contains is accepted.
	faultSendArea = "message_send_area_refused"
	// A candidate could not be committed (store failure or lost fence).
	faultIngress = "message_ingress_failed"
	// The canonical messages could not be read to deliver them.
	faultDeliveryUnconfirmed = "message_delivery_unconfirmed"
	// An accepted message addressed to the attempt could not be delivered.
	faultDelivery = "message_delivery_failed"
)

// messagePump routes one running attempt's candidates into the canonical
// Execution and delivers accepted messages addressed to its agent. Messages
// are data: no pump step changes an attempt, a gate or readiness.
type messagePump struct {
	exchange Exchange
	request  LaunchRequest
	load     func() (workflow.Execution, error)
	update   func(func(*workflow.Execution) error) (workflow.Execution, error)
	now      func() time.Time
	// submitted maps a candidate ID to the exact bytes already committed or
	// refused, so an unchanged replay is not resubmitted while a changed one
	// still is (and is then refused as a conflicting duplicate by the
	// aggregate). A candidate whose commit failed is retried by the next step.
	submitted map[string]string
	delivered map[string]bool
	// refused is the persistent fault that stopped routing, if any.
	refused string
}

// step routes once and returns the fault that left work undone. Failed
// commits, reads and deliveries are retried by the next step, so only a fault
// still present at the final drain decides the attempt; a stop of the
// periodic poll is not a fault.
func (p *messagePump) step(ctx context.Context) string {
	if p.refused != "" {
		return p.refused
	}
	candidates, err := p.exchange.Poll(ctx, p.request)
	if err != nil {
		if ctx.Err() != nil {
			return ""
		}
		p.refused = faultSendArea
		return p.refused
	}
	identity := ingress(p.request)
	for _, c := range candidates {
		wire, err := json.Marshal(c)
		if err != nil || p.submitted[c.ID] == string(wire) {
			continue
		}
		if _, err := p.update(func(x *workflow.Execution) error { return x.AcceptMessage(identity, c, p.now()) }); err != nil && !messageRefusal(err) {
			// Nothing was committed; the principal records the store failure.
			return faultIngress
		}
		p.submitted[c.ID] = string(wire)
	}
	x, err := p.load()
	if err != nil {
		return faultDeliveryUnconfirmed
	}
	fault := ""
	for _, m := range x.Messages {
		if m.Recipient != p.request.Ref.AgentID || p.delivered[m.ID] {
			continue
		}
		if err := p.exchange.Deliver(ctx, p.request, m); err != nil {
			if ctx.Err() == nil {
				fault = faultDelivery
			}
			continue
		}
		p.delivered[m.ID] = true
	}
	return fault
}

// startMessages runs the pump once the attempt's process identity is durable
// (the attempt is running). The returned function stops it and returns the
// exchange fault; with drain set it performs one final step, so a candidate
// written just before a clean exit is still ingested before the terminal
// commit. Afterwards the aggregate refuses any candidate from the terminal
// attempt.
func (e Engine) startMessages(ctx context.Context, request LaunchRequest, update func(func(*workflow.Execution) error) (workflow.Execution, error)) func(drain bool) string {
	exchange, ok := e.Launcher.(Exchange)
	if !ok {
		return func(bool) string { return "" }
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
	return func(drain bool) string {
		stop()
		<-finished
		if drain {
			return p.step(context.WithoutCancel(ctx))
		}
		return p.refused
	}
}
