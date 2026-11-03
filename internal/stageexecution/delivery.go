package stageexecution

import (
	"context"
	"github.com/rgomids/axiom/internal/workflow"
)

// Deliver owns a single current-phase push OR PR. Check is the trusted
// application's live readiness/binding/authority check, required before any
// remote read and again immediately before the effect. Effects are never
// executed in a CAS callback and are never retried.
func (e Engine) Deliver(ctx context.Context, r workflow.DeliveryRequest, grant workflow.DeliveryGrant, effects workflow.DeliveryEffects, check func(context.Context, workflow.Execution) error) (workflow.Execution, error) {
	if !grant.Covers(r) || effects == nil || check == nil {
		return workflow.Execution{}, workflow.ErrUnauthorizedEffect
	}
	current, err := e.Store.Load(ctx, r.ProjectID, r.ExecutionID)
	if err != nil {
		return current, err
	}
	if current.Revision != r.Revision {
		return current, workflow.ErrUnauthorizedEffect
	}
	if err = check(ctx, current); err != nil {
		return current, err
	}
	lease, current, err := e.Claim(ctx, r.ProjectID, r.ExecutionID)
	if err != nil {
		return current, err
	}
	defer lease.Close()
	// Validate the transition on an isolated copy before any Provider call.
	probe := current
	probe.Deliveries = append([]workflow.ExecutionDelivery(nil), current.Deliveries...)
	if err = probe.PrepareDelivery(r, grant, lease.Generation(), e.now()); err != nil {
		return current, err
	}
	if err = check(ctx, current); err != nil {
		return current, err
	}
	o, err := effects.ObserveDelivery(ctx, r.Preview())
	if err != nil || !r.MatchesObservation(o, false) {
		return current, workflow.ErrUnauthorizedEffect
	}
	update := e.commit(ctx, r.ProjectID, r.ExecutionID, lease)
	current, err = update(func(x *workflow.Execution) error { return x.PrepareDelivery(r, grant, lease.Generation(), e.now()) })
	if err != nil {
		return current, err
	}
	index := len(current.Deliveries) - 1
	// Terminal writes outlive caller cancellation; loss of the write leaves a
	// prepared intent which Claim converts to unknown without a second effect.
	finish := func(status string, observation *workflow.DeliveryObservation) (workflow.Execution, error) {
		x, err := update(func(x *workflow.Execution) error {
			return x.FinishDelivery(index, lease.Generation(), status, observation, e.now())
		})
		if err != nil {
			return x, err
		}
		if status == "unknown" {
			return x, workflow.ErrUnknownOutstanding
		}
		return x, nil
	}
	latest, err := e.Store.Load(ctx, r.ProjectID, r.ExecutionID)
	if err != nil || latest.Cancel != nil || latest.RunnerGeneration != lease.Generation() || ctx.Err() != nil {
		return finish("unknown", nil)
	}
	if err = check(ctx, latest); err != nil {
		return finish("unknown", nil)
	}
	// Re-observe after intent publication. A drift never reaches the effect.
	o, err = effects.ObserveDelivery(ctx, r.Preview())
	if err != nil || !r.MatchesObservation(o, false) {
		return finish("unknown", &o)
	}
	if e.cancelPersisted(ctx, r.ProjectID, r.ExecutionID) || ctx.Err() != nil {
		return finish("unknown", nil)
	}
	latest, err = e.Store.Load(ctx, r.ProjectID, r.ExecutionID)
	if err != nil || latest.Cancel != nil || latest.RunnerGeneration != lease.Generation() {
		return finish("unknown", nil)
	}
	if err = check(ctx, latest); err != nil {
		return finish("unknown", nil)
	}
	p := r.Preview()
	p.Observation = o
	if r.Target.Push {
		err = effects.PushDelivery(ctx, p)
	} else {
		err = effects.CreateDeliveryPR(ctx, p)
	}
	if err != nil {
		return finish("unknown", nil)
	}
	o, err = effects.ObserveDelivery(ctx, p)
	if err != nil || !r.MatchesObservation(o, true) {
		return finish("unknown", &o)
	}
	return finish("confirmed", &o)
}

// InspectDelivery appends only an observation under the original exact grant.
// Unknown remains unknown until the separately scoped human disposition path.
func (e Engine) InspectDelivery(ctx context.Context, projectID, executionID string, index int, grant workflow.DeliveryGrant, effects workflow.DeliveryEffects, check func(context.Context, workflow.Execution) error) (workflow.Execution, error) {
	x, err := e.Store.Load(ctx, projectID, executionID)
	if err != nil {
		return x, err
	}
	if index < 0 || index >= len(x.Deliveries) || !grant.Covers(x.Deliveries[index].Request) || grant != x.Deliveries[index].Grant || effects == nil || check == nil {
		return x, workflow.ErrUnauthorizedEffect
	}
	if err = check(ctx, x); err != nil {
		return x, err
	}
	lease, x, err := e.Claim(ctx, projectID, executionID)
	if err != nil {
		return x, err
	}
	defer lease.Close()
	if x.Deliveries[index].Status != "unknown" {
		return x, workflow.ErrTransition
	}
	if err = check(ctx, x); err != nil {
		return x, err
	}
	o, observeErr := effects.ObserveDelivery(ctx, x.Deliveries[index].Request.Preview())
	if observeErr != nil {
		o = workflow.DeliveryObservation{}
	}
	return e.commit(ctx, projectID, executionID, lease)(func(x *workflow.Execution) error {
		return x.ObserveDelivery(index, lease.Generation(), grant, o, e.now())
	})
}
