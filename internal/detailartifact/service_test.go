package detailartifact

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/rgomids/axiom/internal/completion"
)

type recordingCreator struct {
	calls int
	value Artifact
	err   error
}

func (c *recordingCreator) Create(context.Context, Draft) (Artifact, error) {
	c.calls++
	return c.value, c.err
}

func TestMaterializeCreatesExactlyOneForcedArtifactAndNoCompactArtifact(t *testing.T) {
	artifact, err := New("123e4567-e89b-42d3-a456-426614174001", time.Unix(1, 0), validDraft(t))
	if err != nil {
		t.Fatal(err)
	}
	creator := &recordingCreator{value: artifact}
	compact := Materialize(context.Background(), creator, false, false, completion.Facts{Completed: true}, Draft{})
	if creator.calls != 0 || compact.Details != "" {
		t.Fatal("compact outcome created artifact")
	}
	forced := Materialize(context.Background(), creator, true, true, completion.Facts{Completed: true}, validDraft(t))
	if creator.calls != 1 || forced.Details != artifact.Reference() || forced.Err != nil {
		t.Fatalf("forced outcome = %#v", forced)
	}
}

func TestMaterializePreservesPrimaryTruthAcrossArtifactFailure(t *testing.T) {
	failure := errors.New("capacity")
	for _, test := range []struct {
		name                         string
		required                     bool
		primary                      completion.Facts
		want                         completion.Status
		wantRequestedEffectConfirmed bool
	}{
		{"required before primary", true, completion.Facts{}, completion.Failure, false},
		{"required after completed without requested effect", true, completion.Facts{Completed: true}, completion.Failure, false},
		{"required after primary", true, completion.Facts{RequestedEffectConfirmed: true}, completion.Partial, true},
		{"optional after primary", false, completion.Facts{Completed: true}, completion.Success, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			result := Materialize(context.Background(), &recordingCreator{err: failure}, true, test.required, test.primary, validDraft(t))
			status, err := completion.Classify(result.Facts)
			if err != nil || status != test.want || !errors.Is(result.Err, failure) {
				t.Fatalf("materialization = %#v status=%s err=%v", result, status, err)
			}
			if result.Facts.RequestedEffectConfirmed != test.wantRequestedEffectConfirmed {
				t.Fatalf("RequestedEffectConfirmed = %t, want %t", result.Facts.RequestedEffectConfirmed, test.wantRequestedEffectConfirmed)
			}
		})
	}
}
