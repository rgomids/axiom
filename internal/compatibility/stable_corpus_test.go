package compatibility

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/local"
)

// stableCorpus is persisted state written by released v1 writers. It is
// append-only: a newer release must keep resolving it to a direct upgrade,
// so state an earlier release wrote can never block installation.
const stableCorpus = "testdata/stable-v1"

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
		if !ok {
			t.Fatalf("malformed MANIFEST line %q", scanner.Text())
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
	for _, tree := range []string{"projects", "state"} {
		err := filepath.WalkDir(filepath.Join(stableCorpus, tree), func(path string, entry fs.DirEntry, err error) error {
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
}

func TestStableCorpusResolvesToDirectUpgrade(t *testing.T) {
	roots := copyCorpus(t, stableCorpus, "projects", "state")
	report := inspect(t, roots)
	if report.Classification != ValidV1 {
		t.Fatalf("released v1 state classified %s/%s: findings=%+v", report.Classification, report.Reason, report.Findings)
	}
	decision := Resolve(Owned, report)
	if decision.Strategy != StrategyDirect || decision.Outcome != OutcomeCompatible {
		t.Fatalf("released v1 state does not upgrade directly: %+v", decision)
	}
	observed := map[local.InventoryKind]bool{}
	for _, object := range report.objects {
		observed[object.Kind] = true
	}
	for _, kind := range local.V1Kinds() {
		if !observed[kind] {
			t.Errorf("v1 kind %s is missing from the stable corpus: run the freeze generator (see PROVENANCE) before releasing", kind)
		}
	}
}
