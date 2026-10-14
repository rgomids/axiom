package local

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/rgomids/axiom/internal/runtimeprofile"
)

const codexDeclarationInput = `{"declarationId":"codex-effort-sources","declarationVersion":1,"runtimeVersion":"synthetic-codex-1","executableDigest":"1111111111111111111111111111111111111111111111111111111111111111","invocationMode":"noninteractive","precedenceEstablished":true,"jsonKeys":[],"environmentKeys":[],"sources":[{"scope":"user","anchor":"user","path":".codex/config.toml","format":"toml","directory":false},{"scope":"project","anchor":"project","path":".codex/config.toml","format":"toml","directory":false},{"scope":"managed","anchor":"managed","path":"managed_config.toml","format":"toml","directory":false}]}`

// v3Wire is a canonical format-3 record authored directly (Specification 007
// PM-1): the Runtime/Profile content of authoringConfig, with the Codex
// declaration when declared is set.
func v3Wire(t *testing.T, revision uint64, declared bool) []byte {
	t.Helper()
	cfg := authoringConfig(revision)
	next := runtimeprofile.ConfigurationV3{FormatVersion: runtimeprofile.FormatVersionV3, Revision: revision, ModelProfiles: cfg.ModelProfiles, Preferences: cfg.Preferences}
	for _, r := range cfg.Runtimes {
		runtime := runtimeprofile.RuntimeV3{ID: r.ID, Adapter: r.Adapter, Enabled: r.Enabled, AllowlistedProfileIDs: r.AllowlistedProfileIDs}
		if declared && r.ID == "codex" {
			d, err := runtimeprofile.DecodeEffortDeclaration([]byte(codexDeclarationInput), "codex")
			if err != nil {
				t.Fatal(err)
			}
			runtime.EffortDeclarations = []runtimeprofile.EffortDeclaration{d}
		}
		next.Runtimes = append(next.Runtimes, runtime)
	}
	wire, err := runtimeprofile.EncodeV3(next)
	if err != nil {
		t.Fatal(err)
	}
	return wire
}

// activeV3Store initializes format 3 directly over fresh state; no v2 record
// or migration is involved.
func activeV3Store(t *testing.T) (RuntimeProfileStore, RuntimeProfileSnapshot, string) {
	t.Helper()
	ctx := context.Background()
	store := authoringStore(t)
	fresh, err := store.Inspect(ctx)
	if err != nil || fresh.State != "fresh" {
		t.Fatalf("%+v %v", fresh, err)
	}
	active, err := store.Publish(ctx, fresh.Token, v3Wire(t, 1, false))
	if err != nil || active.State != "active-v3" || active.Revision != 1 || active.V3 == nil {
		t.Fatalf("%+v %v", active, err)
	}
	return store, active, filepath.Join(store.root, "runtime-profiles", "v1", runtimeProfileObject)
}

// PM-1: fresh → v3 and v3 → v3 at revision+1 with exact canonical bytes; no
// format 1/2 writer exists (PM-11).
func TestRuntimeProfileV3Publication(t *testing.T) {
	ctx := context.Background()
	store, active, path := activeV3Store(t)
	if current, _ := os.ReadFile(path); !bytes.Equal(current, v3Wire(t, 1, false)) {
		t.Fatal("initialized bytes differ from the canonical candidate")
	}
	if _, err := store.Publish(ctx, active.Token, legacyWire(t, 2, 2)); !errors.Is(err, runtimeprofile.ErrUnsupportedConfigurationFormat) {
		t.Fatalf("legacy publication replaced v3: %v", err)
	}
	declared := v3Wire(t, 2, true)
	published, err := store.Publish(ctx, active.Token, declared)
	if err != nil || published.State != "active-v3" || published.Revision != 2 || published.V3 == nil {
		t.Fatalf("%+v %v", published, err)
	}
	if current, _ := os.ReadFile(path); !bytes.Equal(current, declared) {
		t.Fatal("published bytes differ from the canonical candidate")
	}
	versioned, err := store.Inspect(ctx)
	if err != nil || versioned.Token != published.Token || versioned.V3.Revision != 2 {
		t.Fatalf("%+v %v", versioned, err)
	}

	// Declaration authoring on v3.
	d, _ := runtimeprofile.DecodeEffortDeclaration([]byte(strings.Replace(strings.Replace(codexDeclarationInput, "codex-effort-sources", "codex-next", 1), "synthetic-codex-1", "synthetic-codex-2", 1)), "codex")
	edited, _, err := runtimeprofile.WithEffortDeclaration(*versioned.V3, "codex", d)
	if err != nil {
		t.Fatal(err)
	}
	editWire, _ := runtimeprofile.EncodeV3(edited)
	next, err := store.Publish(ctx, versioned.Token, editWire)
	if err != nil || next.State != "active-v3" || next.Revision != 3 {
		t.Fatalf("%+v %v", next, err)
	}
	if current, _ := os.ReadFile(path); !bytes.Equal(current, editWire) {
		t.Fatal("edited bytes differ")
	}
}

// PM-1: no downgrade, no v3 over a v1/v2 record (no conversion by overwrite),
// no skipped or repeated revision, no non-canonical bytes; refusals never
// mutate. A retired record may be re-authored directly in format 3.
func TestRuntimeProfileV3PublicationRefusals(t *testing.T) {
	ctx := context.Background()
	t.Run("revision-and-canonical", func(t *testing.T) {
		store := authoringStore(t)
		fresh, _ := store.Inspect(ctx)
		indented := append([]byte(" "), v3Wire(t, 1, false)...)
		for name, wire := range map[string][]byte{"skipped": v3Wire(t, 2, false), "non-canonical": indented} {
			if _, err := store.Publish(ctx, fresh.Token, wire); err == nil {
				t.Fatalf("%s accepted", name)
			}
		}
		if after, _ := store.Inspect(ctx); after.State != "fresh" {
			t.Fatal("refusal mutated fresh state")
		}
	})
	t.Run("downgrade", func(t *testing.T) {
		store, active, path := activeV3Store(t)
		// A tombstone not bound to DigestV3 of the exact prior record is
		// refused; a v2 record is never written.
		digest, _ := runtimeprofile.Digest(legacyConfig(2, 1))
		tomb, _ := runtimeprofile.EncodeTombstone(runtimeprofile.Tombstone{Kind: runtimeprofile.TombstoneKind, FormatVersion: 1, Revision: 2, PriorConfigurationDigest: digest})
		if _, err := store.Publish(ctx, active.Token, legacyWire(t, 2, 2)); !errors.Is(err, runtimeprofile.ErrUnsupportedConfigurationFormat) {
			t.Fatalf("v2 over v3: %v", err)
		}
		if _, err := store.Publish(ctx, active.Token, tomb); !errors.Is(err, ErrConflict) {
			t.Fatalf("historical-digest tombstone over v3: %v", err)
		}
		if current, _ := os.ReadFile(path); !bytes.Equal(current, v3Wire(t, 1, false)) {
			t.Fatal("downgrade refusal mutated v3")
		}
	})
	t.Run("v2", func(t *testing.T) {
		store := authoringStore(t)
		writeLegacyRecord(t, store.root, legacyWire(t, 2, 1))
		v2, err := store.Inspect(ctx)
		if err != nil || v2.State != "active-v2" {
			t.Fatalf("%+v %v", v2, err)
		}
		if _, err := store.Publish(ctx, v2.Token, v3Wire(t, 2, true)); !errors.Is(err, runtimeprofile.ErrUnsupportedConfigurationFormat) {
			t.Fatalf("v2 → v3: %v", err)
		}
		if current, _ := os.ReadFile(filepath.Join(store.root, "runtime-profiles", "v1", runtimeProfileObject)); !bytes.Equal(current, legacyWire(t, 2, 1)) {
			t.Fatal("refusal mutated v2")
		}
	})
	t.Run("v1", func(t *testing.T) {
		store := authoringStore(t)
		writeLegacyRecord(t, store.root, legacyWire(t, 1, 1))
		v1, err := store.Inspect(ctx)
		if err != nil || v1.State != "active-v1" {
			t.Fatalf("%+v %v", v1, err)
		}
		if _, err := store.Publish(ctx, v1.Token, v3Wire(t, 2, true)); !errors.Is(err, runtimeprofile.ErrUnsupportedConfigurationFormat) {
			t.Fatalf("v1 → v3: %v", err)
		}
	})
	t.Run("retired", func(t *testing.T) {
		store := authoringStore(t)
		fresh, _ := store.Inspect(ctx)
		active, err := store.Publish(ctx, fresh.Token, authoringWire(t, 1))
		if err != nil {
			t.Fatal(err)
		}
		digest, _ := runtimeprofile.DigestV3(authoringConfig(1))
		tomb, _ := runtimeprofile.EncodeTombstone(runtimeprofile.Tombstone{Kind: runtimeprofile.TombstoneKind, FormatVersion: 1, Revision: 2, PriorConfigurationDigest: digest})
		retired, err := store.Publish(ctx, active.Token, tomb)
		if err != nil {
			t.Fatal(err)
		}
		if next, err := store.Publish(ctx, retired.Token, v3Wire(t, 3, false)); err != nil || next.State != "active-v3" || next.Revision != 3 {
			t.Fatalf("retired → v3: %+v %v", next, err)
		}
	})
}

// PM-1 keeps the existing F3/F6 protocol for v3 → v3: an interrupted
// publication is never success and scoped recovery stays authoritative.
// Rollback restoring an identical reusable preview token is the recorded
// pending defect closed by the authorization epoch (BLD-001B.3c-3).
func TestRuntimeProfileV3PublicationRecovery(t *testing.T) {
	ctx := context.Background()
	for _, stage := range []FaultStage{FaultF3, FaultF6} {
		t.Run(string(stage), func(t *testing.T) {
			store, active, _ := activeV3Store(t)
			v3 := v3Wire(t, 2, true)
			store.hooks.fault = func(at FaultStage) error {
				if at == stage {
					return ErrSimulatedInterruption
				}
				return nil
			}
			if _, err := store.Publish(ctx, active.Token, v3); err == nil {
				t.Fatal("interrupted publication reported success")
			}
			store.hooks = publicationHooks{}
			pending, err := store.Inspect(ctx)
			if !errors.Is(err, ErrRecoveryRequired) || pending.State != "pending" {
				t.Fatalf("%+v %v", pending, err)
			}
			if _, err := store.Publish(ctx, active.Token, v3); err == nil {
				t.Fatal("publication over pending state")
			}
			plan, found, err := store.InspectRuntimeRecovery(ctx)
			if err != nil || !found {
				t.Fatalf("%+v %v", plan, err)
			}
			authority, err := AuthorizeRecovery(plan, plan.Digest)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := store.ApplyRuntimeRecovery(ctx, plan, authority); err != nil {
				t.Fatal(err)
			}
			recovered, err := store.Inspect(ctx)
			if err != nil || recovered.State != "active-v3" {
				t.Fatalf("%+v %v", recovered, err)
			}
			current, _ := os.ReadFile(filepath.Join(store.root, "runtime-profiles", "v1", runtimeProfileObject))
			if recovered.Revision == 1 {
				if stage == FaultF6 {
					t.Fatal("committed publication rolled back")
				}
				if !bytes.Equal(current, v3Wire(t, 1, false)) {
					t.Fatal("rollback did not restore the exact source bytes")
				}
				return
			}
			if !bytes.Equal(current, v3) {
				t.Fatal("recovered v3 bytes differ")
			}
			if _, err := store.Publish(ctx, active.Token, v3); err == nil {
				t.Fatal("pre-recovery preview token reused")
			}
		})
	}
}

// BI-3.1: one bounded read of a regular file reached through no symlink.
func TestReadEffortDeclarationInput(t *testing.T) {
	dir := t.TempDir()
	regular := filepath.Join(dir, "declaration.json")
	if err := os.WriteFile(regular, []byte(codexDeclarationInput), 0o644); err != nil {
		t.Fatal(err)
	}
	if wire, err := ReadEffortDeclarationInput(regular); err != nil || string(wire) != codexDeclarationInput {
		t.Fatalf("regular file refused: %v", err)
	}
	link := filepath.Join(dir, "link.json")
	if err := os.Symlink(regular, link); err != nil {
		t.Fatal(err)
	}
	linkedDir := filepath.Join(dir, "linked")
	if err := os.Symlink(dir, linkedDir); err != nil {
		t.Fatal(err)
	}
	oversized := filepath.Join(dir, "oversized.json")
	if err := os.WriteFile(oversized, bytes.Repeat([]byte(" "), runtimeprofile.MaxConfigurationBytes+1), 0o644); err != nil {
		t.Fatal(err)
	}
	fifo := filepath.Join(dir, "fifo.json")
	if err := syscall.Mkfifo(fifo, 0o600); err != nil {
		t.Fatal(err)
	}
	for name, path := range map[string]string{"symlink": link, "symlinked-ancestor": filepath.Join(linkedDir, "declaration.json"), "missing": filepath.Join(dir, "absent.json"), "directory": dir, "oversized": oversized, "fifo": fifo, "empty": "", "nul": regular + "\x00"} {
		if _, err := ReadEffortDeclarationInput(path); !errors.Is(err, ErrUnsafe) {
			t.Errorf("%s accepted: %v", name, err)
		}
	}
}
