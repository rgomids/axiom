package local

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rgomids/axiom/internal/capabilityproof"
)

func selectionStore(t *testing.T) (CapabilityStore, string, string) {
	t.Helper()
	root := filepath.Join(t.TempDir(), "state")
	s, err := NewCapabilityStore(root)
	if err != nil {
		t.Fatal(err)
	}
	return s, root, filepath.Join(root, "capability-evidence", "v1")
}

func selectAt(s CapabilityStore, e capabilityproof.Evidence) (string, string, error) {
	selector := capabilityproof.Selector{Evidence: s, Now: func() time.Time { return e.ObservedAt.Add(time.Hour) }}
	return selector.Select(context.Background(), e.Binding, e.Model)
}

// snapshotTree records names, modes, sizes, times and content for every entry.
func snapshotTree(t *testing.T, root string) map[string]string {
	t.Helper()
	snapshot := map[string]string{}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := os.Lstat(path)
		if err != nil {
			return err
		}
		value := fmt.Sprintf("%v %d %d", info.Mode(), info.Size(), info.ModTime().UnixNano())
		if info.Mode().IsRegular() {
			wire, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			value += fmt.Sprintf(" %x", sha256.Sum256(wire))
		}
		snapshot[path] = value
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return snapshot
}

func TestCapabilityEvidenceScanSelectsUniqueReadOnly(t *testing.T) {
	s, root, _ := selectionStore(t)
	readOnly := storeEvidence()
	write := storeEvidence()
	write.Binding.Variant = capabilityproof.RepositoryWriteVariant
	profile := storeEvidence()
	profile.Model = &capabilityproof.ModelBinding{ProfileID: "profile", ProfileDigest: strings.Repeat("b", 64), Model: "model", EffortMode: "explicit", EffortToken: "high", ConfigurationRevision: 1, EffortSourceDigest: strings.Repeat("c", 64)}
	refs := map[string]capabilityproof.Evidence{}
	for _, e := range []capabilityproof.Evidence{readOnly, write, profile} {
		id, err := s.Create(context.Background(), e)
		if err != nil {
			t.Fatal(err)
		}
		refs[id] = e
	}
	before := snapshotTree(t, root)
	for want, e := range refs {
		ref, code, err := selectAt(s, e)
		if err != nil || code != "" || ref != want {
			t.Fatalf("%s: %q %q %v", e.Binding.Variant, ref, code, err)
		}
	}
	// Selection creates, rewrites, invalidates and migrates nothing.
	after := snapshotTree(t, root)
	if len(after) != len(before) {
		t.Fatalf("entries changed: %d -> %d", len(before), len(after))
	}
	for path, value := range before {
		if after[path] != value {
			t.Fatalf("selection changed %s", path)
		}
	}
}

func TestCapabilityEvidenceScanAbsentStoreCreatesNothing(t *testing.T) {
	s, root, _ := selectionStore(t)
	ref, code, err := selectAt(s, storeEvidence())
	if ref != "" || code != "capability_unknown" || !errors.Is(err, capabilityproof.ErrEvidenceMissing) {
		t.Fatalf("%q %q %v", ref, code, err)
	}
	if _, err := os.Lstat(root); !os.IsNotExist(err) {
		t.Fatal("selection created the store", err)
	}
}

func TestCapabilityEvidenceScanAmbiguousAndInvalidated(t *testing.T) {
	s, root, _ := selectionStore(t)
	first := storeEvidence()
	second := storeEvidence()
	second.ObservedAt = first.ObservedAt.Add(time.Minute)
	firstID, err := s.Create(context.Background(), first)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Create(context.Background(), second); err != nil {
		t.Fatal(err)
	}
	before := snapshotTree(t, root)
	if ref, code, err := selectAt(s, second); ref != "" || code != "capability_unknown" || !errors.Is(err, capabilityproof.ErrEvidenceAmbiguous) {
		t.Fatalf("%q %q %v", ref, code, err)
	}
	after := snapshotTree(t, root)
	for path, value := range before {
		if after[path] != value {
			t.Fatalf("ambiguity mutated %s", path)
		}
	}
	// Only an explicit, authorized invalidation narrows the candidates.
	loaded, err := s.Load(context.Background(), firstID)
	if err != nil {
		t.Fatal(err)
	}
	wire, _ := capabilityproof.EncodeEvidence(loaded)
	if err := s.Invalidate(context.Background(), firstID, "explicit", sha256.Sum256(wire)); err != nil {
		t.Fatal(err)
	}
	secondID, _ := capabilityproof.Digest(second)
	if ref, code, err := selectAt(s, second); err != nil || code != "" || ref != secondID {
		t.Fatalf("%q %q %v", ref, code, err)
	}
	if loaded, err = s.Load(context.Background(), firstID); err != nil || loaded.InvalidatedReason != "explicit" {
		t.Fatal("invalidated record changed", err)
	}
}

func TestCapabilityEvidenceScanUnsafeEntriesFailClosed(t *testing.T) {
	valid := storeEvidence()
	validID, _ := capabilityproof.Digest(valid)
	other := storeEvidence()
	other.Binding.Version = "2"
	otherWire, _ := capabilityproof.EncodeEvidence(other)
	unsupported := storeEvidence()
	unsupported.SchemaVersion = 2
	unsupportedWire, _ := json.Marshal(unsupported)
	unsupportedID := fmt.Sprintf("%x", sha256.Sum256(unsupportedWire))
	cases := map[string]struct {
		prepare func(t *testing.T, dir string)
		want    error
	}{
		"corrupted-content": {func(t *testing.T, dir string) {
			writeFixture(t, filepath.Join(dir, strings.Repeat("1", 64)+".json"), []byte("{not json"), 0o600)
		}, ErrUnsafe},
		"invalid-content-address": {func(t *testing.T, dir string) {
			writeFixture(t, filepath.Join(dir, strings.Repeat("2", 64)+".json"), otherWire, 0o600)
		}, ErrUnsafe},
		"unsupported-schema": {func(t *testing.T, dir string) {
			writeFixture(t, filepath.Join(dir, unsupportedID+".json"), unsupportedWire, 0o600)
		}, ErrUnsafe},
		"unexpected-name": {func(t *testing.T, dir string) {
			writeFixture(t, filepath.Join(dir, "notes.txt"), []byte("x"), 0o600)
		}, ErrUnsafe},
		"symlink-entry": {func(t *testing.T, dir string) {
			target := filepath.Join(t.TempDir(), "evidence.json")
			writeFixture(t, target, otherWire, 0o600)
			if err := os.Symlink(target, filepath.Join(dir, strings.Repeat("3", 64)+".json")); err != nil {
				t.Fatal(err)
			}
		}, ErrUnsafe},
		"directory-entry": {func(t *testing.T, dir string) {
			if err := os.Mkdir(filepath.Join(dir, strings.Repeat("4", 64)+".json"), 0o700); err != nil {
				t.Fatal(err)
			}
		}, ErrUnsafe},
		"unreadable-permissions": {func(t *testing.T, dir string) {
			id, _ := capabilityproof.Digest(other)
			writeFixture(t, filepath.Join(dir, id+".json"), otherWire, 0o644)
		}, ErrUnsafe},
		"pending-publication": {func(t *testing.T, dir string) {
			writeFixture(t, filepath.Join(dir, ".axiom-stage-pending"), []byte("x"), 0o600)
		}, ErrRecoveryRequired},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			s, root, dir := selectionStore(t)
			if _, err := s.Create(context.Background(), valid); err != nil {
				t.Fatal(err)
			}
			tc.prepare(t, dir)
			before := snapshotTree(t, root)
			if err := s.Scan(context.Background(), func(capabilityproof.StoredEvidence) error { return nil }); !errors.Is(err, tc.want) {
				t.Fatalf("scan: %v", err)
			}
			// The valid record is never selected from a partial, unsafe view.
			ref, code, err := selectAt(s, valid)
			if ref != "" || code != "capability_unknown" || err != capabilityproof.ErrBlocked {
				t.Fatalf("%q %q %v", ref, code, err)
			}
			after := snapshotTree(t, root)
			for path, value := range before {
				if after[path] != value {
					t.Fatalf("scan mutated %s", path)
				}
			}
			if _, err := os.Lstat(filepath.Join(dir, validID+".json")); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func writeFixture(t *testing.T, path string, wire []byte, mode os.FileMode) {
	t.Helper()
	if err := os.WriteFile(path, wire, mode); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, mode); err != nil {
		t.Fatal(err)
	}
}
