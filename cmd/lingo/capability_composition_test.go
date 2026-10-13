package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/rgomids/axiom/internal/capabilityproof"
	"github.com/rgomids/axiom/internal/local"
)

func TestProductionCapabilityCompositionFailsClosed(t *testing.T) {
	root := filepath.Join(t.TempDir(), "state")
	s, err := productionCapabilityService(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := s.Store.(local.CapabilityStore); !ok {
		t.Fatal("production composition must use the canonical local store")
	}
	if s.Source != nil || s.ProbeAdapter != nil || s.ProbeEnvelopes != nil || s.ProbeWriter != nil || s.ProbeAuth != nil {
		t.Fatal("unavailable production source and effect dependencies must remain absent")
	}
	ctx := context.Background()
	for _, variant := range []string{capabilityproof.ReadOnlyVariant, capabilityproof.RepositoryWriteVariant} {
		target := capabilityproof.Target{RuntimeID: "codex", ProfileID: "synthetic", Variant: variant}
		view, err := s.Show(ctx, target)
		if !errors.Is(err, capabilityproof.ErrBlocked) || view.EvidenceRef != "" {
			t.Fatalf("missing source/evidence: %+v, %v", view, err)
		}
		for cap, state := range view.States {
			if state != capabilityproof.Unknown || view.Blockers[cap] != "capability_unknown" {
				t.Fatalf("%s unexpectedly proven: %+v", cap, view)
			}
		}
		for _, expectedRef := range []string{"", "synthetic-reviewed-reference"} {
			ref, code, err := s.Check(ctx, target, nil, variant == capabilityproof.RepositoryWriteVariant, true, expectedRef)
			if ref != "" || code != "capability_unknown" || !errors.Is(err, capabilityproof.ErrBlocked) {
				t.Fatalf("check = %q %q %v", ref, code, err)
			}
		}
		if _, err := s.PreviewProbe(ctx, target); !errors.Is(err, capabilityproof.ErrBlocked) {
			t.Fatalf("unconfigured probe = %v", err)
		}
	}
	// Neither composition nor blocked reads/checks create even local state.
	// No Runtime or Provider port is installed in this composition.
	if _, err := os.Lstat(root); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("blocked composition created state: %v", err)
	}
}

func TestProductionCapabilityCompositionRejectsUnsafeRoot(t *testing.T) {
	if _, err := productionCapabilityService("relative-state"); !errors.Is(err, local.ErrUnsafe) {
		t.Fatalf("unsafe root = %v", err)
	}
}
