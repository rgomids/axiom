package local

import (
	"context"
	"crypto/sha256"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/rgomids/axiom/internal/capabilityproof"
)

func storeEvidence() capabilityproof.Evidence {
	d := strings.Repeat("a", 64)
	return capabilityproof.Evidence{SchemaVersion: 1, Binding: capabilityproof.Binding{RuntimeID: "codex", ExecutableDigest: d, Version: "1", ModeDigest: d, Variant: "read-only", AdapterVersion: "1", AxiomDigest: d}, ObservedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), ProbePacketDigest: d, Capabilities: map[string]capabilityproof.Result{"read": {State: capabilityproof.Proven, Facts: capabilityproof.ProbeFacts{AuthenticationProven: true, BehavioralProbe: true, Verified: true}}}, Confinement: capabilityproof.Confinement{Level: "unknown", NotTested: []string{"home", "vendor-home", "tmpdir", "system-temp", "external-network"}}}
}
func TestCapabilityStoreRoundTripInvalidate(t *testing.T) {
	ctx := context.Background()
	root := filepath.Join(t.TempDir(), "state")
	s, err := NewCapabilityStore(root)
	if err != nil {
		t.Fatal(err)
	}
	e := storeEvidence()
	id, err := s.Create(ctx, e)
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := s.Load(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	wire, _ := capabilityproof.EncodeEvidence(loaded)
	if err := s.Invalidate(ctx, id, "explicit", sha256.Sum256(wire)); err != nil {
		t.Fatal(err)
	}
	if err := s.Invalidate(ctx, id, "profile-removed", sha256.Sum256(wire)); !errors.Is(err, ErrConflict) {
		t.Fatal("stale invalidation", err)
	}
	loaded, err = s.Load(ctx, id)
	if err != nil || loaded.InvalidatedReason != "explicit" {
		t.Fatal(loaded, err)
	}
	if state, _ := capabilityproof.Evaluate(loaded, e.Binding, nil, "read", e.ObservedAt); state != capabilityproof.Unknown {
		t.Fatal(state)
	}
	// Changing data without its content address must fail closed.
	loaded.Binding.Version = "2"
	wire, _ = capabilityproof.EncodeEvidence(loaded)
	if err := os.WriteFile(filepath.Join(root, "capability-evidence", "v1", id+".json"), wire, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Load(ctx, id); !errors.Is(err, ErrUnsafe) {
		t.Fatal("tamper", err)
	}
}
func TestCapabilityStoreConcurrentCreate(t *testing.T) {
	s, _ := NewCapabilityStore(filepath.Join(t.TempDir(), "state"))
	e := storeEvidence()
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); _, err := s.Create(context.Background(), e); results <- err }()
	}
	wg.Wait()
	close(results)
	success := 0
	for err := range results {
		if err == nil {
			success++
		} else if !errors.Is(err, ErrConflict) {
			t.Fatalf("unexpected concurrent publication failure: %v", err)
		}
	}
	if success != 1 {
		t.Fatalf("successful publications: %d", success)
	}
}
func TestCapabilityRecoveryAndInventory(t *testing.T) {
	for _, stage := range []FaultStage{FaultF2, FaultF3, FaultF4, FaultF5, FaultF6, FaultF7, FaultF8} {
		t.Run(string(stage), func(t *testing.T) {
			ctx := context.Background()
			root := filepath.Join(t.TempDir(), "state")
			s, _ := NewCapabilityStore(root)
			e := storeEvidence()
			s.hooks.fault = func(at FaultStage) error {
				if at == stage {
					return ErrSimulatedInterruption
				}
				return nil
			}
			_, err := s.Create(ctx, e)
			if !errors.Is(err, ErrSimulatedInterruption) {
				t.Fatalf("interruption boundary not reached: %v", err)
			}
			var publication *PublicationError
			if !errors.As(err, &publication) || publication.Stage != stage || publication.Committed != (stage >= FaultF6) {
				t.Fatalf("incorrect interruption outcome: %v", err)
			}
			if (stage == FaultF6 || stage == FaultF8) && !errors.Is(err, ErrRecoveryRequired) {
				t.Fatalf("committed interruption lost recovery requirement: %v", err)
			}
			report, err := InspectRecovery(ctx, RecoveryRoots{State: root})
			if err != nil {
				t.Fatal(err)
			}
			if len(report.Plans) != 1 || report.Plans[0].Directory != "capability-evidence/v1" {
				t.Fatalf("unrecognized recovery: %+v", report)
			}
			inventory, err := InspectStateInventory(ctx, root)
			if err != nil {
				t.Fatal(err)
			}
			found := false
			for _, entry := range inventory.Entries {
				if entry.Kind == InventoryRecovery {
					found = true
				}
			}
			if !found {
				t.Fatal("inventory missed interrupted evidence")
			}
		})
	}
	root := filepath.Join(t.TempDir(), "state")
	s, _ := NewCapabilityStore(root)
	if _, err := s.Create(context.Background(), storeEvidence()); err != nil {
		t.Fatal(err)
	}
	inventory, err := InspectStateInventory(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, entry := range inventory.Entries {
		if entry.Kind == InventoryCapabilityEvidence {
			found = true
			if !entry.Kind.Supported() {
				t.Fatal("unsupported evidence")
			}
		}
	}
	if !found {
		t.Fatal("missing evidence inventory")
	}
}
func TestCapabilityStoreRejectsUnsafeBeforeEffects(t *testing.T) {
	s, _ := NewCapabilityStore(filepath.Join(t.TempDir(), "state"))
	e := storeEvidence()
	e.Binding.Version = "/private/config"
	if _, err := s.Create(context.Background(), e); !errors.Is(err, capabilityproof.ErrInvalidEvidence) {
		t.Fatal(err)
	}
	e = storeEvidence()
	e.InvalidatedReason = "explicit"
	if _, err := s.Create(context.Background(), e); !errors.Is(err, capabilityproof.ErrInvalidEvidence) {
		t.Fatal(err)
	}
}

func TestCapabilityStoreRetainsExpiredEvidenceWithoutPromotion(t *testing.T) {
	e := storeEvidence()
	e.ObservedAt = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	s, err := NewCapabilityStore(filepath.Join(t.TempDir(), "state"))
	if err != nil {
		t.Fatal(err)
	}
	id, err := s.Create(context.Background(), e)
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := s.Load(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	if state, _ := capabilityproof.Evaluate(loaded, e.Binding, nil, "read", e.ObservedAt.Add(capabilityproof.MaxAge+time.Second)); state != capabilityproof.Unknown {
		t.Fatal("expired evidence promoted")
	}
}
