package detailartifact

import (
	"context"

	"github.com/rgomids/axiom/internal/completion"
)

type Creator interface {
	Create(context.Context, Draft) (Artifact, error)
}

type Materialization struct {
	Facts   completion.Facts
	Details string
	Err     error
}

// Materialize keeps primary-effect truth separate from optional/required detail
// publication. Compact outcomes do not touch the artifact store.
func Materialize(ctx context.Context, store Creator, needed, required bool, primary completion.Facts, draft Draft) Materialization {
	if !needed {
		return Materialization{Facts: primary}
	}
	artifact, err := store.Create(ctx, draft)
	if err == nil {
		return Materialization{Facts: primary, Details: artifact.Reference()}
	}
	if !required {
		return Materialization{Facts: primary, Err: err}
	}
	if primary.RequestedEffectConfirmed {
		return Materialization{Facts: completion.Facts{RequestedEffectConfirmed: true, SecondaryFailure: true}, Err: err}
	}
	return Materialization{Facts: completion.Facts{Failed: true}, Err: err}
}
