package compatibility

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/local"
	"github.com/rgomids/axiom/internal/projectapp"
)

// stableCorpus is persisted state written by released v1 writers, one
// complete snapshot per release representation below snapshots/<release>. It
// is append-only: a newer release must keep resolving every snapshot to a
// direct upgrade, so state an earlier release wrote can never block
// installation, even when a later release writes different bytes at the same
// logical path.
const stableCorpus = "testdata/stable-v1"

func stableSnapshots(t *testing.T) []string {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(stableCorpus, "snapshots"))
	if err != nil {
		t.Fatal(err)
	}
	labels := []string{}
	for _, entry := range entries {
		if !entry.IsDir() {
			t.Fatalf("unexpected corpus entry snapshots/%s", entry.Name())
		}
		labels = append(labels, entry.Name())
	}
	if len(labels) == 0 {
		t.Fatal("stable corpus has no snapshot")
	}
	return labels
}

func TestStableCorpusIsAppendOnly(t *testing.T) {
	file, err := os.Open(filepath.Join(stableCorpus, "MANIFEST"))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	listed := map[string]bool{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		digest, relative, ok := strings.Cut(scanner.Text(), "  ")
		if !ok || !strings.HasPrefix(relative, "snapshots/") || listed[relative] {
			t.Fatalf("malformed or duplicate MANIFEST line %q", scanner.Text())
		}
		wire, err := os.ReadFile(filepath.Join(stableCorpus, filepath.FromSlash(relative)))
		if err != nil {
			t.Fatalf("frozen corpus file %s was removed: released state must stay covered", relative)
		}
		sum := sha256.Sum256(wire)
		if hex.EncodeToString(sum[:]) != digest {
			t.Fatalf("frozen corpus file %s changed: released state must stay readable exactly as written", relative)
		}
		listed[relative] = true
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	err = filepath.WalkDir(filepath.Join(stableCorpus, "snapshots"), func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		relative, _ := filepath.Rel(stableCorpus, path)
		if !listed[filepath.ToSlash(relative)] {
			t.Errorf("corpus file %s is not in MANIFEST; add files only through the freeze generator", relative)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestStableCorpusResolvesToDirectUpgrade(t *testing.T) {
	observed := map[local.InventoryKind]bool{}
	for _, label := range stableSnapshots(t) {
		roots := copyCorpus(t, filepath.Join(stableCorpus, "snapshots", label), "projects", "state")
		report := inspect(t, roots)
		if report.Classification != ValidV1 {
			t.Fatalf("released v1 snapshot %s classified %s/%s: findings=%+v", label, report.Classification, report.Reason, report.Findings)
		}
		decision := Resolve(Owned, report)
		if decision.Strategy != StrategyDirect || decision.Outcome != OutcomeCompatible {
			t.Fatalf("released v1 snapshot %s does not upgrade directly: %+v", label, decision)
		}
		for _, object := range report.objects {
			observed[object.Kind] = true
		}
	}
	for _, kind := range append(local.V1Kinds(), local.AdditiveKinds()...) {
		if !observed[kind] {
			t.Errorf("v1 kind %s is missing from the stable corpus: run the freeze generator (see PROVENANCE) before releasing", kind)
		}
	}
}

// This is the exact historical v0.10.0 writer output, reconstructed from the
// published tag. Classification alone must not mask a semantic reader change.
func TestV0100OperationalStateRemainsReadable(t *testing.T) {
	roots := copyCorpus(t, filepath.Join(stableCorpus, "snapshots", "v0.10.0"), "projects", "state")
	store, err := local.NewOperationalStore(roots.State)
	if err != nil {
		t.Fatal(err)
	}
	observed, err := store.InspectOperational(context.Background(), "123e4567-e89b-42d3-a456-426614174000")
	if err != nil || observed.State.ProjectStatus != projectapp.ProjectArchived || len(observed.State.DisabledIntegrations) != 1 || observed.State.DisabledIntegrations[0] != "work-items" {
		t.Fatalf("historical operational state = %+v, %v", observed, err)
	}
}

// Preparing release artifacts must run the frozen writer/reader contract before
// building. Clearing the generator variable prevents release preparation from
// silently blessing newly generated state instead of validating frozen bytes.
func TestReleasePreparationEnforcesStableCorpus(t *testing.T) {
	wire, err := os.ReadFile("../../.github/workflows/release-artifacts.yml")
	if err != nil {
		t.Fatal(err)
	}
	text := string(wire)
	guard := "run: AXIOM_FREEZE_STATE_CORPUS= go test ./internal/local ./internal/compatibility -count=1"
	position := strings.Index(text, guard)
	build := strings.Index(text, "name: Build supported release archives")
	if position < 0 || build < 0 || position >= build {
		t.Fatal("release preparation must validate the frozen compatibility corpus before building")
	}
}
