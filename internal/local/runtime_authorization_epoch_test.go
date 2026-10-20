package local

import (
	"bytes"
	"context"
	"errors"
	"math"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"testing"

	"github.com/rgomids/axiom/internal/runtimeprofile"
)

// Specification 007 AXM-22 PM-6: the persistent authorization epoch. Every
// test is synthetic local state; no Runtime, Provider or network is involved.

func writeEpochRecord(t *testing.T, root string, epoch uint64) {
	t.Helper()
	path := filepath.Join(root, "runtime-profiles", "v1", authorizationEpochObject)
	if err := os.WriteFile(path, encodeAuthorizationEpoch(epoch), 0o600); err != nil {
		t.Fatal(err)
	}
}

func epochOf(t *testing.T, s RuntimeProfileStore) uint64 {
	t.Helper()
	wire, err := os.ReadFile(filepath.Join(s.root, "runtime-profiles", "v1", authorizationEpochObject))
	if errors.Is(err, os.ErrNotExist) {
		return 0
	}
	if err != nil {
		t.Fatal(err)
	}
	epoch, err := decodeAuthorizationEpoch(wire)
	if err != nil {
		t.Fatal(err)
	}
	return epoch
}

func profileDirectory(t *testing.T, s RuntimeProfileStore) []string {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(s.root, "runtime-profiles", "v1"))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		t.Fatal(err)
	}
	names := []string{}
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names
}

func configurationBytes(t *testing.T, s RuntimeProfileStore) []byte {
	t.Helper()
	wire, err := os.ReadFile(filepath.Join(s.root, "runtime-profiles", "v1", runtimeProfileObject))
	if err != nil {
		t.Fatal(err)
	}
	return wire
}

func faultAt(stage FaultStage, err error) func(FaultStage) error {
	return func(observed FaultStage) error {
		if observed == stage {
			return err
		}
		return nil
	}
}

func recoverRuntime(t *testing.T, s RuntimeProfileStore) RecoveryPlan {
	t.Helper()
	ctx := context.Background()
	plan, found, err := s.InspectRuntimeRecovery(ctx)
	if err != nil || !found || plan.Action == PreservedReview {
		t.Fatalf("recovery plan %+v found %v: %v", plan, found, err)
	}
	authority, err := AuthorizeRecovery(plan, plan.Digest)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.ApplyRuntimeRecovery(ctx, plan, authority); err != nil {
		t.Fatal(err)
	}
	return plan
}

// Fresh preview, exact apply and reuse of an applied preview; preview stays
// zero-write and restart observes the same persisted epoch.
func TestAuthorizationEpochPreviewApplyRestart(t *testing.T) {
	ctx := context.Background()
	s := authoringStore(t)
	fresh, err := s.Inspect(ctx)
	if err != nil || fresh.State != "fresh" || fresh.epoch != 0 {
		t.Fatalf("%+v %v", fresh, err)
	}
	if _, err := os.Lstat(s.root); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("fresh preview created state")
	}
	active, err := s.Publish(ctx, fresh.Token, operationalWire(t, 1))
	if err != nil || active.epoch != 1 || epochOf(t, s) != 1 {
		t.Fatalf("%+v %v", active, err)
	}
	if _, err := s.Publish(ctx, fresh.Token, operationalWire(t, 2)); !errors.Is(err, ErrConflict) {
		t.Fatalf("applied preview reused: %v", err)
	}
	before := profileDirectory(t, s)
	beforeBytes := configurationBytes(t, s)
	for range 3 {
		again, err := s.Inspect(ctx)
		if err != nil || again.Token != active.Token {
			t.Fatalf("preview not stable: %v", err)
		}
	}
	if !slices.Equal(before, profileDirectory(t, s)) || !bytes.Equal(beforeBytes, configurationBytes(t, s)) || epochOf(t, s) != 1 {
		t.Fatal("preview wrote state")
	}
	restarted, err := NewRuntimeProfileStore(s.root)
	if err != nil {
		t.Fatal(err)
	}
	observed, err := restarted.Inspect(ctx)
	if err != nil || observed.Token != active.Token || observed.epoch != 1 {
		t.Fatalf("restart %+v %v", observed, err)
	}
	if _, err := restarted.Publish(ctx, observed.Token, operationalWire(t, 2)); err != nil || epochOf(t, s) != 2 {
		t.Fatalf("restarted apply: %v", err)
	}
}

// Configuration or epoch change between preview and apply is stale.
func TestAuthorizationEpochStaleBetweenPreviewAndApply(t *testing.T) {
	ctx := context.Background()
	s, active, _ := activeOperationalStore(t)
	changed, err := s.Publish(ctx, active.Token, operationalWire(t, 2))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Publish(ctx, active.Token, operationalWire(t, 3)); !errors.Is(err, ErrConflict) {
		t.Fatalf("configuration change: %v", err)
	}
	// Only the epoch changes: a different identity, same bytes.
	writeEpochRecord(t, s.root, changed.epoch+1)
	if bumped, err := s.Inspect(ctx); err != nil || bumped.Token == changed.Token || bumped.Revision != changed.Revision {
		t.Fatalf("epoch not bound into preview identity: %v", err)
	}
	if _, err := s.Publish(ctx, changed.Token, operationalWire(t, 3)); !errors.Is(err, ErrConflict) {
		t.Fatalf("epoch change: %v", err)
	}
}

// Central PM-6 regression: PREVIEW P → attempted authorized publication → F3
// rollback restores the original configuration → applying P again is stale.
func TestAuthorizationEpochF3RollbackCannotRevivePreview(t *testing.T) {
	ctx := context.Background()
	for _, mode := range []string{"cleanup", "recovery"} {
		t.Run(mode, func(t *testing.T) {
			s, p, path := activeOperationalStore(t)
			original, _ := os.ReadFile(path)
			identity, _ := FileIdentity(path)
			fault := errors.New("synthetic pre-commit failure")
			if mode == "recovery" {
				fault = ErrSimulatedInterruption
			}
			s.hooks.fault = faultAt(FaultF3, fault)
			if _, err := s.Publish(ctx, p.Token, operationalWire(t, 2)); err == nil {
				t.Fatal("fault not reached")
			}
			s.hooks = publicationHooks{}
			if mode == "recovery" {
				if plan := recoverRuntime(t, s); plan.Action != RestorePrior {
					t.Fatalf("action %s", plan.Action)
				}
			}
			current, _ := os.ReadFile(path)
			after, _ := FileIdentity(path)
			if !bytes.Equal(current, original) || after != identity {
				t.Fatal("rollback did not restore the identical object")
			}
			observed, err := s.Inspect(ctx)
			if err != nil || observed.Revision != p.Revision || observed.Token == p.Token {
				t.Fatalf("observed %+v %v", observed, err)
			}
			if _, err := s.Publish(ctx, p.Token, operationalWire(t, 2)); !errors.Is(err, ErrConflict) {
				t.Fatalf("pre-F3 preview revived: %v", err)
			}
			if !bytes.Equal(configurationBytes(t, s), original) {
				t.Fatal("stale preview published")
			}
			if _, err := s.Publish(ctx, observed.Token, operationalWire(t, 2)); err != nil {
				t.Fatalf("fresh exact preview: %v", err)
			}
		})
	}
}

// F6 recovery finalizes the committed generation and advances the epoch; no
// pre-recovery preview or recovery plan becomes valid afterward.
func TestAuthorizationEpochF6RecoveryCannotRevivePreview(t *testing.T) {
	ctx := context.Background()
	s, p, _ := activeOperationalStore(t)
	s.hooks.fault = faultAt(FaultF6, ErrSimulatedInterruption)
	next := operationalWire(t, 2)
	if _, err := s.Publish(ctx, p.Token, next); err == nil {
		t.Fatal("fault not reached")
	}
	s.hooks = publicationHooks{}
	epoch := epochOf(t, s)
	plan := recoverRuntime(t, s)
	if plan.Action != FinalizeCommitted || epochOf(t, s) != epoch+1 {
		t.Fatalf("action %s epoch %d", plan.Action, epochOf(t, s))
	}
	if _, err := s.Publish(ctx, p.Token, operationalWire(t, 2)); !errors.Is(err, ErrConflict) {
		t.Fatalf("pre-F6 preview revived: %v", err)
	}
	authority, _ := AuthorizeRecovery(plan, plan.Digest)
	if _, err := s.ApplyRuntimeRecovery(ctx, plan, authority); err == nil {
		t.Fatal("applied recovery plan reused")
	}
	observed, err := s.Inspect(ctx)
	if err != nil || observed.Revision != 2 || !bytes.Equal(configurationBytes(t, s), next) {
		t.Fatalf("%+v %v", observed, err)
	}
}

// The recovery preview binds the epoch it advances.
func TestAuthorizationEpochBindsRecoveryPlan(t *testing.T) {
	ctx := context.Background()
	s, p, _ := activeOperationalStore(t)
	s.hooks.fault = faultAt(FaultF3, ErrSimulatedInterruption)
	s.Publish(ctx, p.Token, operationalWire(t, 2))
	s.hooks = publicationHooks{}
	plan, _, err := s.InspectRuntimeRecovery(ctx)
	if err != nil {
		t.Fatal(err)
	}
	writeEpochRecord(t, s.root, epochOf(t, s)+1)
	authority, _ := AuthorizeRecovery(plan, plan.Digest)
	if _, err := s.ApplyRuntimeRecovery(ctx, plan, authority); !errors.Is(err, ErrConflict) {
		t.Fatalf("recovery plan across an epoch change: %v", err)
	}
}

// Retirement and authorized reconfiguration never reset the epoch, and a
// fresh-looking state after retirement keeps its history.
func TestAuthorizationEpochSurvivesRetirementAndReconfiguration(t *testing.T) {
	ctx := context.Background()
	s, p, _ := activeOperationalStore(t)
	tomb, err := runtimeprofile.EncodeTombstone(runtimeprofile.Tombstone{Kind: runtimeprofile.TombstoneKind, FormatVersion: 1, Revision: 2, PriorConfigurationDigest: authoringDigest(t, *p.V4)})
	if err != nil {
		t.Fatal(err)
	}
	retired, err := s.Publish(ctx, p.Token, tomb)
	if err != nil || retired.State != "retired" || epochOf(t, s) != 2 {
		t.Fatalf("%+v %v", retired, err)
	}
	recreated, err := s.Publish(ctx, retired.Token, operationalWire(t, 3))
	if err != nil || recreated.State != "active-v4" || epochOf(t, s) != 3 {
		t.Fatalf("%+v %v", recreated, err)
	}
	for _, stale := range []string{p.Token, retired.Token} {
		if _, err := s.Publish(ctx, stale, operationalWire(t, 4)); !errors.Is(err, ErrConflict) {
			t.Fatalf("pre-reconfiguration preview: %v", err)
		}
	}
	// A first publication interrupted before commit keeps the advanced epoch:
	// the state is fresh again but never at its initial identity.
	f := authoringStore(t)
	initial, _ := f.Inspect(ctx)
	f.hooks.fault = faultAt(FaultF2, errors.New("synthetic pre-commit failure"))
	if _, err := f.Publish(ctx, initial.Token, operationalWire(t, 1)); err == nil {
		t.Fatal("fault not reached")
	}
	f.hooks = publicationHooks{}
	again, err := f.Inspect(ctx)
	if err != nil || again.State != "fresh" || again.epoch != 1 || again.Token == initial.Token {
		t.Fatalf("%+v %v", again, err)
	}
	if _, err := f.Publish(ctx, initial.Token, operationalWire(t, 1)); !errors.Is(err, ErrConflict) {
		t.Fatalf("initial fresh preview revived: %v", err)
	}
	if _, err := f.Publish(ctx, again.Token, operationalWire(t, 1)); err != nil || epochOf(t, f) != 2 {
		t.Fatalf("fresh exact preview: %v", err)
	}
}

// Concurrent writers with one reviewed preview: exactly one accepted attempt
// advances the epoch exactly once; no update is lost.
func TestAuthorizationEpochConcurrentWriters(t *testing.T) {
	ctx := context.Background()
	s, p, _ := activeOperationalStore(t)
	var wg sync.WaitGroup
	var mu sync.Mutex
	successes := 0
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := s.Publish(ctx, p.Token, operationalWire(t, 2)); err == nil {
				mu.Lock()
				successes++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if successes != 1 || epochOf(t, s) != p.epoch+1 {
		t.Fatalf("successes %d epoch %d", successes, epochOf(t, s))
	}
}

// Unauthorized, stale or invalid requests never advance the epoch, so no
// caller without the exact preview can invalidate or escalate authority; a
// failed attempt is never retried implicitly.
func TestAuthorizationEpochNoEscalationOrRetry(t *testing.T) {
	ctx := context.Background()
	s, p, _ := activeOperationalStore(t)
	for _, token := range []string{"", "synthetic-forged-token"} {
		if _, err := s.Publish(ctx, token, operationalWire(t, 2)); !errors.Is(err, ErrConflict) || epochOf(t, s) != 1 {
			t.Fatalf("unauthorized publication: %v epoch %d", err, epochOf(t, s))
		}
	}
	if _, err := s.Publish(ctx, p.Token, operationalWire(t, 9)); err == nil || epochOf(t, s) != 1 {
		t.Fatalf("invalid candidate advanced the epoch: %v", err)
	}
	calls := 0
	s.hooks.fault = func(stage FaultStage) error {
		if stage == FaultF2 {
			calls++
			return errors.New("synthetic failure")
		}
		return nil
	}
	if _, err := s.Publish(ctx, p.Token, operationalWire(t, 2)); err == nil || calls != 1 || epochOf(t, s) != 2 {
		t.Fatalf("implicit retry: calls %d epoch %d %v", calls, epochOf(t, s), err)
	}
}

// Overflow, corrupt, missing, symlinked and permissive epoch state fails
// closed and is never repaired or reset.
func TestAuthorizationEpochUnsafeStateFailsClosed(t *testing.T) {
	ctx := context.Background()
	t.Run("overflow", func(t *testing.T) {
		s, _, path := activeOperationalStore(t)
		writeEpochRecord(t, s.root, math.MaxUint64)
		before, _ := os.ReadFile(path)
		p, err := s.Inspect(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := s.Publish(ctx, p.Token, operationalWire(t, 2)); !errors.Is(err, ErrConflict) || epochOf(t, s) != math.MaxUint64 {
			t.Fatalf("overflow: %v", err)
		}
		if after, _ := os.ReadFile(path); !bytes.Equal(before, after) {
			t.Fatal("overflow published configuration")
		}
	})
	epochPath := func(s RuntimeProfileStore) string {
		return filepath.Join(s.root, "runtime-profiles", "v1", authorizationEpochObject)
	}
	for name, corrupt := range map[string][]byte{
		"garbage":       []byte("not json"),
		"zero":          []byte(`{"schema":"axiom-runtime-authorization-epoch/v1","epoch":0}` + "\n"),
		"noncanonical":  []byte(`{ "schema":"axiom-runtime-authorization-epoch/v1","epoch":3}` + "\n"),
		"unknown":       []byte(`{"schema":"axiom-runtime-authorization-epoch/v1","epoch":3,"x":1}` + "\n"),
		"wrong-schema":  []byte(`{"schema":"axiom-runtime-authorization-epoch/v2","epoch":3}` + "\n"),
		"negative-text": []byte(`{"schema":"axiom-runtime-authorization-epoch/v1","epoch":-3}` + "\n"),
	} {
		t.Run(name, func(t *testing.T) {
			s, p, _ := activeOperationalStore(t)
			if err := os.WriteFile(epochPath(s), corrupt, 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := s.Inspect(ctx); !errors.Is(err, ErrUnsafe) {
				t.Fatalf("corrupt epoch inspected: %v", err)
			}
			if _, err := s.Publish(ctx, p.Token, operationalWire(t, 2)); err == nil {
				t.Fatal("publication over corrupt epoch")
			}
			if kind := kindOf(t, s.root, "runtime-profiles/v1/"+authorizationEpochObject); kind != InventoryMalformed && kind != InventoryUnsafe {
				t.Fatalf("inventory %s", kind)
			}
			if current, _ := os.ReadFile(epochPath(s)); !bytes.Equal(current, corrupt) {
				t.Fatal("corrupt epoch repaired")
			}
		})
	}
	t.Run("missing", func(t *testing.T) {
		s, p, _ := activeOperationalStore(t)
		os.Remove(epochPath(s))
		if _, err := s.Inspect(ctx); !errors.Is(err, ErrUnsafe) {
			t.Fatalf("missing epoch: %v", err)
		}
		if _, err := s.Publish(ctx, p.Token, operationalWire(t, 2)); err == nil {
			t.Fatal("publication with missing epoch")
		}
		if _, err := os.Lstat(epochPath(s)); !errors.Is(err, os.ErrNotExist) {
			t.Fatal("missing epoch reset")
		}
	})
	t.Run("symlink", func(t *testing.T) {
		s, p, _ := activeOperationalStore(t)
		elsewhere := filepath.Join(t.TempDir(), "epoch")
		os.WriteFile(elsewhere, encodeAuthorizationEpoch(1), 0o600)
		os.Remove(epochPath(s))
		if err := os.Symlink(elsewhere, epochPath(s)); err != nil {
			t.Fatal(err)
		}
		if _, err := s.Inspect(ctx); !errors.Is(err, ErrUnsafe) {
			t.Fatalf("symlinked epoch: %v", err)
		}
		if _, err := s.Publish(ctx, p.Token, operationalWire(t, 2)); err == nil {
			t.Fatal("publication through symlinked epoch")
		}
	})
	t.Run("permissions", func(t *testing.T) {
		s, p, _ := activeOperationalStore(t)
		if err := os.Chmod(epochPath(s), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := s.Inspect(ctx); !errors.Is(err, ErrUnsafe) {
			t.Fatalf("permissive epoch: %v", err)
		}
		if _, err := s.Publish(ctx, p.Token, operationalWire(t, 2)); err == nil {
			t.Fatal("publication over permissive epoch")
		}
	})
}

// An interrupted epoch advance publishes nothing, leaves recovery state every
// reader refuses, and scoped recovery advances the committed epoch.
func TestAuthorizationEpochInterruptedPersistence(t *testing.T) {
	ctx := context.Background()
	s, p, path := activeOperationalStore(t)
	original, _ := os.ReadFile(path)
	s.hooks.fault = faultAt(faultEpoch, ErrSimulatedInterruption)
	if _, err := s.Publish(ctx, p.Token, operationalWire(t, 2)); !errors.Is(err, ErrRecoveryRequired) {
		t.Fatalf("interrupted epoch: %v", err)
	}
	s.hooks = publicationHooks{}
	if epochOf(t, s) != 1 || !bytes.Equal(configurationBytes(t, s), original) {
		t.Fatal("interrupted epoch changed committed state")
	}
	if pending, err := s.Inspect(ctx); !errors.Is(err, ErrRecoveryRequired) || pending.State != "pending" {
		t.Fatalf("%+v %v", pending, err)
	}
	if _, err := s.Publish(ctx, p.Token, operationalWire(t, 2)); err == nil {
		t.Fatal("publication over interrupted epoch")
	}
	plan := recoverRuntime(t, s)
	if plan.Action != RestorePrior || plan.Marker != "" || len(plan.Objects) != 1 || plan.Objects[0].Role != "epoch_stage" {
		t.Fatalf("plan %+v", plan)
	}
	if epochOf(t, s) != 2 || !slices.Equal(profileDirectory(t, s), []string{authorizationEpochObject, runtimeProfileObject}) {
		t.Fatalf("epoch %d entries %v", epochOf(t, s), profileDirectory(t, s))
	}
	if _, err := s.Publish(ctx, p.Token, operationalWire(t, 2)); !errors.Is(err, ErrConflict) {
		t.Fatalf("pre-interruption preview revived: %v", err)
	}
	observed, err := s.Inspect(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Publish(ctx, observed.Token, operationalWire(t, 2)); err != nil {
		t.Fatalf("fresh exact preview: %v", err)
	}
	if kind := kindOf(t, s.root, "runtime-profiles/v1/"+authorizationEpochObject); kind != InventoryRuntimeProfile {
		t.Fatalf("inventory %s", kind)
	}
}
