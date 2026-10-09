package local

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstallationFormatClassification(t *testing.T) {
	for wire, want := range map[string]InventoryKind{
		`{"formatVersion":1}`: InventoryMalformed,
		`{"formatVersion":2}`: InventoryMalformed,
		`{"formatVersion":3}`: InventoryNewer,
		`{"formatVersion":0}`: InventoryOlder,
	} {
		if got := versionedFailureWithin([]byte(wire), InstallationFormatVersion); got != want {
			t.Fatalf("%s: %s, want %s", wire, got, want)
		}
	}
	// Other record kinds keep their format 1 window.
	if versionedFailure([]byte(`{"formatVersion":2}`)) != InventoryNewer {
		t.Fatal("non-installation window widened")
	}
}

func TestPortableManifestVersionWindowIncludesSchemaFour(t *testing.T) {
	for version, want := range map[string]InventoryKind{"3": InventoryMalformed, "2": InventoryMalformed, "4": InventoryMalformed, "5": InventoryNewer} {
		root := t.TempDir()
		if err := os.Chmod(root, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.Mkdir(filepath.Join(root, "sample"), 0o700); err != nil {
			t.Fatal(err)
		}
		// Valid version line, invalid body: a supported version that fails
		// decoding is malformed, never "upgrade Lingo".
		if err := os.WriteFile(filepath.Join(root, "sample", "axiom.yaml"), []byte("schemaVersion: "+version+"\nunknown: true\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		inventory, err := InspectPortableInventory(context.Background(), root)
		if err != nil {
			t.Fatal(err)
		}
		found := false
		for _, entry := range inventory.Entries {
			if strings.HasSuffix(entry.Relative, "axiom.yaml") {
				found = true
				if entry.Kind != want {
					t.Fatalf("v%s: %s, want %s", version, entry.Kind, want)
				}
			}
		}
		if !found {
			t.Fatalf("v%s manifest not inventoried: %+v", version, inventory.Entries)
		}
	}
}
