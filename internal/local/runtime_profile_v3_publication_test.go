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

func migratedWire(t *testing.T, source runtimeprofile.Configuration) []byte {
	t.Helper()
	d, err := runtimeprofile.DecodeEffortDeclaration([]byte(codexDeclarationInput), "codex")
	if err != nil {
		t.Fatal(err)
	}
	next, err := runtimeprofile.MigrateV2ToV3(source, "codex", d)
	if err != nil {
		t.Fatal(err)
	}
	wire, err := runtimeprofile.EncodeV3(next)
	if err != nil {
		t.Fatal(err)
	}
	return wire
}

func activeV2Store(t *testing.T) (RuntimeProfileStore, RuntimeProfileSnapshot, string) {
	t.Helper()
	ctx := context.Background()
	store := authoringStore(t)
	fresh, err := store.InspectVersioned(ctx)
	if err != nil || fresh.State != "fresh" {
		t.Fatalf("%+v %v", fresh, err)
	}
	active, err := store.Publish(ctx, fresh.Token, authoringWire(t, 1))
	if err != nil || active.State != "active-v2" {
		t.Fatalf("%+v %v", active, err)
	}
	return store, active, filepath.Join(store.root, "runtime-profiles", "v1", runtimeProfileObject)
}

// BI-3.2 item 10: v2 → v3 and v3 → v3 at revision+1, exact bytes, legacy
// v2-only readers still refusing the published record (D8).
func TestRuntimeProfileV3Publication(t *testing.T) {
	ctx := context.Background()
	store, active, path := activeV2Store(t)
	legacy, err := store.Inspect(ctx)
	if err != nil || legacy.Token != active.Token || legacy.State != "active-v2" {
		t.Fatalf("versioned inspection changed v2 identity: %+v %v", legacy, err)
	}
	v3 := migratedWire(t, active.Configuration)
	published, err := store.Publish(ctx, active.Token, v3)
	if err != nil || published.State != "active-v3" || published.Revision != 2 || published.V3 == nil {
		t.Fatalf("%+v %v", published, err)
	}
	if current, _ := os.ReadFile(path); !bytes.Equal(current, v3) {
		t.Fatal("published bytes differ from the canonical candidate")
	}
	versioned, err := store.InspectVersioned(ctx)
	if err != nil || versioned.Token != published.Token || versioned.V3.Revision != 2 {
		t.Fatalf("%+v %v", versioned, err)
	}
	if _, err := store.Inspect(ctx); !errors.Is(err, runtimeprofile.ErrUnsupportedConfigurationFormat) {
		t.Fatalf("v2-only reader accepted v3: %v", err)
	}
	if _, err := store.Load(ctx); !errors.Is(err, runtimeprofile.ErrUnsupportedConfigurationFormat) {
		t.Fatalf("v2-only load accepted v3: %v", err)
	}
	if err := store.Save(ctx, authoringConfig(3)); err == nil {
		t.Fatal("legacy save replaced v3")
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

// BI-3.2 item 10: no downgrade, no v3 over v1/fresh/retired, no skipped or
// repeated revision, no non-canonical bytes; refusals never mutate.
func TestRuntimeProfileV3PublicationRefusals(t *testing.T) {
	ctx := context.Background()
	t.Run("revision-and-canonical", func(t *testing.T) {
		store, active, path := activeV2Store(t)
		v3 := migratedWire(t, active.Configuration)
		v, _ := runtimeprofile.DecodeVersioned(v3)
		skipped := *v.V3
		skipped.Revision = 3
		skippedWire, _ := runtimeprofile.EncodeV3(skipped)
		indented := append([]byte(" "), v3...)
		for name, wire := range map[string][]byte{"skipped": skippedWire, "non-canonical": indented} {
			if _, err := store.Publish(ctx, active.Token, wire); err == nil {
				t.Fatalf("%s accepted", name)
			}
		}
		if current, _ := os.ReadFile(path); !bytes.Equal(current, authoringWire(t, 1)) {
			t.Fatal("refusal mutated v2")
		}
	})
	t.Run("downgrade", func(t *testing.T) {
		store, active, path := activeV2Store(t)
		v3 := migratedWire(t, active.Configuration)
		published, err := store.Publish(ctx, active.Token, v3)
		if err != nil {
			t.Fatal(err)
		}
		digest, _ := runtimeprofile.Digest(authoringConfig(1))
		tomb, _ := runtimeprofile.EncodeTombstone(runtimeprofile.Tombstone{Kind: runtimeprofile.TombstoneKind, FormatVersion: 1, Revision: 3, PriorConfigurationDigest: digest})
		for name, wire := range map[string][]byte{"v2": authoringWire(t, 3), "tombstone": tomb} {
			if _, err := store.Publish(ctx, published.Token, wire); !errors.Is(err, runtimeprofile.ErrUnsupportedConfigurationFormat) {
				t.Fatalf("%s over v3: %v", name, err)
			}
		}
		if current, _ := os.ReadFile(path); !bytes.Equal(current, v3) {
			t.Fatal("downgrade refusal mutated v3")
		}
	})
	t.Run("v1", func(t *testing.T) {
		store := authoringStore(t)
		if err := store.Create(ctx, runtimeConfiguration(1)); err != nil {
			t.Fatal(err)
		}
		v1, err := store.InspectVersioned(ctx)
		if err != nil || v1.State != "active-v1" {
			t.Fatalf("%+v %v", v1, err)
		}
		if _, err := store.Publish(ctx, v1.Token, migratedWire(t, authoringConfig(1))); !errors.Is(err, runtimeprofile.ErrUnsupportedConfigurationFormat) {
			t.Fatalf("v1 → v3: %v", err)
		}
	})
	t.Run("retired", func(t *testing.T) {
		store, active, _ := activeV2Store(t)
		digest, _ := runtimeprofile.Digest(active.Configuration)
		tomb, _ := runtimeprofile.EncodeTombstone(runtimeprofile.Tombstone{Kind: runtimeprofile.TombstoneKind, FormatVersion: 1, Revision: 2, PriorConfigurationDigest: digest})
		retired, err := store.Publish(ctx, active.Token, tomb)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := store.Publish(ctx, retired.Token, migratedWire(t, authoringConfig(2))); !errors.Is(err, runtimeprofile.ErrUnsupportedConfigurationFormat) {
			t.Fatalf("retired → v3: %v", err)
		}
	})
}

// BI-3.2 item 11: an interrupted v3 publication is never success; scoped
// recovery stays authoritative and the pre-recovery token is never reusable.
func TestRuntimeProfileV3PublicationRecovery(t *testing.T) {
	ctx := context.Background()
	for _, stage := range []FaultStage{FaultF3, FaultF6} {
		t.Run(string(stage), func(t *testing.T) {
			store, active, _ := activeV2Store(t)
			v3 := migratedWire(t, active.Configuration)
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
			pending, err := store.InspectVersioned(ctx)
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
			recovered, err := store.InspectVersioned(ctx)
			if err != nil || recovered.State != "active-v2" && recovered.State != "active-v3" {
				t.Fatalf("%+v %v", recovered, err)
			}
			current, _ := os.ReadFile(filepath.Join(store.root, "runtime-profiles", "v1", runtimeProfileObject))
			if stage == FaultF6 && recovered.State != "active-v3" {
				t.Fatalf("committed publication recovered to %s", recovered.State)
			}
			if recovered.State == "active-v2" {
				// Rollback restores the exact pre-attempt snapshot (identities and
				// bytes), so a fresh preview is identical to the earlier one.
				if recovered.Token != active.Token || !bytes.Equal(current, authoringWire(t, 1)) {
					t.Fatal("rollback did not restore the exact source snapshot")
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
