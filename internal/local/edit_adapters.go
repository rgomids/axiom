package local

import (
	"context"
	"path/filepath"

	"github.com/rgomids/axiom/internal/projectapp"
)

// RecordCodec validates and encodes a complete application local candidate
// with the exact installation-record wire contract. It performs no I/O.
type RecordCodec struct{}

func (RecordCodec) EncodeLocal(state projectapp.LocalRecordState) ([]byte, []projectapp.Issue) {
	record, issues := NewRecord(RecordState(state))
	if len(issues) != 0 {
		return nil, invalidLocalCandidate()
	}
	wire, issues := EncodeRecord(record)
	if len(issues) != 0 {
		return nil, invalidLocalCandidate()
	}
	return wire, nil
}

// ApplicationRecord exposes a sealed record's complete state to the
// application layer without its wire format.
func ApplicationRecord(record Record) projectapp.LocalRecordState {
	return projectapp.LocalRecordState(record.State())
}

// DirectoryObserver observes only the explicit directory it is given, using the
// same non-following identity rule as initial Project setup.
type DirectoryObserver struct{}

func (DirectoryObserver) ObserveCheckout(ctx context.Context, request projectapp.CheckoutRequest) (projectapp.CheckoutFacts, []projectapp.Issue) {
	if ctx.Err() != nil {
		return projectapp.CheckoutFacts{}, []projectapp.Issue{{Phase: projectapp.LocalPhase, Field: projectapp.InstallationField, Code: projectapp.CancelledOperation}}
	}
	identity, err := DirectoryIdentity(filepath.Clean(request.ExplicitPath))
	if err != nil {
		return projectapp.CheckoutFacts{}, []projectapp.Issue{{Phase: projectapp.LocalPhase, Field: projectapp.InstallationField, Code: projectapp.InvalidSnapshot}}
	}
	return projectapp.CheckoutFacts{CanonicalIdentity: identity, Observation: projectapp.Observation{Availability: projectapp.Unverified, Basis: projectapp.NotChecked}}, nil
}

func invalidLocalCandidate() []projectapp.Issue {
	return []projectapp.Issue{{Phase: projectapp.LocalPhase, Field: projectapp.InstallationField, Code: projectapp.InvalidPreview}}
}
