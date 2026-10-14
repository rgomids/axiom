package runtimeapplication

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/runtimeprofile"
)

type unsupportedFormatStore struct{ writes int }

func (s *unsupportedFormatStore) Inspect(context.Context) (LocalSnapshot, error) {
	return LocalSnapshot{State: "unsafe"}, fmt.Errorf("unsafe: %w", runtimeprofile.ErrUnsupportedConfigurationFormat)
}
func (s *unsupportedFormatStore) Publish(context.Context, string, []byte) (LocalSnapshot, error) {
	s.writes++
	return LocalSnapshot{}, errors.New("unexpected publication")
}

// Specification 007 AXM-22 D8: v2-only authoring refuses format 3 with an
// actionable next step, inside the existing public category, and never writes.
func TestAuthoringRefusesV3WithoutMigrationOrWrite(t *testing.T) {
	for _, req := range []AuthoringRequest{createRequest(), {Kind: "profile", Operation: "list"}, {Kind: "profile", Operation: "migrate"}} {
		store := &unsupportedFormatStore{}
		got, err := Authoring{Store: store}.Run(context.Background(), req)
		if err == nil || got.Category != "configuration_unavailable" || !strings.Contains(got.Next, "no downgrade or migration") || store.writes != 0 {
			t.Fatalf("%s: %+v %v", req.Operation, got, err)
		}
	}
}
