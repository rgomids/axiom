package workflow

import "context"

// ProjectionRouter selects a ProjectionCapability by Work Item provider.
type ProjectionRouter struct {
	ByProvider map[string]ProjectionCapability
}

func (r ProjectionRouter) Inspect(ctx context.Context, item WorkItem, desiredLabel, projectionKey string) (ProjectionObservation, error) {
	capability := r.ByProvider[item.Provider]
	if capability == nil {
		return ProjectionObservation{}, &ProjectionError{Kind: ProjectionInvalidResponse, EffectNotCommitted: true}
	}
	return capability.Inspect(ctx, item, desiredLabel, projectionKey)
}

func (r ProjectionRouter) Apply(ctx context.Context, item WorkItem, effect ProjectionEffect) error {
	capability := r.ByProvider[item.Provider]
	if capability == nil {
		return &ProjectionError{Kind: ProjectionInvalidResponse, EffectNotCommitted: true}
	}
	return capability.Apply(ctx, item, effect)
}
