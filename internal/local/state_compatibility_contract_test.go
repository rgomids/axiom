package local

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/workflow"
	"github.com/rgomids/axiom/internal/workitem"
)

// These guards keep one invariant: a newer release never refuses persisted
// state that an earlier release legitimately wrote. Each layer closes a
// different gap:
//
//   - every top-level directory a store opens is classified by the inventory;
//   - every v1 kind is produced by a real writer and read back as supported;
//   - the frozen stable corpus (internal/compatibility/testdata/stable-v1),
//     written by earlier writers, keeps resolving to a direct upgrade.

const freezeCorpusEnv = "AXIOM_FREEZE_STATE_CORPUS"

// stateRootOpeners are the helpers stores use to open a child of the state
// root. A top-level directory opened any other way is not detected here.
var stateRootOpeners = map[string]bool{"openChild": true, "privateChild": true, "existingPrivateChild": true}

func TestEveryStateRootWriterDirectoryIsInventoried(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	opened := map[string]string{}
	set := token.NewFileSet()
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		parsed, err := parser.ParseFile(set, file, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(parsed, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok || len(call.Args) < 2 {
				return true
			}
			name := ""
			switch fun := call.Fun.(type) {
			case *ast.Ident:
				name = fun.Name
			case *ast.SelectorExpr:
				name = fun.Sel.Name
			}
			var parent ast.Expr
			switch {
			case stateRootOpeners[name]:
				parent = call.Args[0]
			case name == "Join":
				parent = call.Args[0]
			default:
				return true
			}
			if !isStateRootExpr(parent) {
				return true
			}
			literal, ok := call.Args[1].(*ast.BasicLit)
			if !ok || literal.Kind != token.STRING {
				return true
			}
			value, _ := strconv.Unquote(literal.Value)
			opened[value] = set.Position(call.Pos()).String()
			return true
		})
	}
	if len(opened) == 0 {
		t.Fatal("no state-root directory openers found; the guard no longer matches the store idiom")
	}
	for directory, position := range opened {
		if !slices.Contains(StateRootDirectories, directory) {
			t.Errorf("%s opens state-root directory %q that InspectStateInventory does not classify: add it to StateRootDirectories with a v1 walker, or upgrades will refuse state this release writes", position, directory)
		}
	}
}

// isStateRootExpr matches the state root as stores name it: the opened root
// handle `root` or the configured path `s.root`.
func isStateRootExpr(expr ast.Expr) bool {
	switch value := expr.(type) {
	case *ast.Ident:
		return value.Name == "root"
	case *ast.SelectorExpr:
		receiver, ok := value.X.(*ast.Ident)
		return ok && receiver.Name == "s" && value.Sel.Name == "root"
	}
	return false
}

func TestEveryV1WriterIsRecognizedByInventory(t *testing.T) {
	portable, state := writeEveryV1Kind(t)
	observed := map[InventoryKind]bool{}
	for _, inspected := range []struct {
		path    string
		inspect func(context.Context, string) (Inventory, error)
	}{{portable, InspectPortableInventory}, {state, InspectStateInventory}} {
		inventory, err := inspected.inspect(context.Background(), inspected.path)
		if err != nil {
			t.Fatal(err)
		}
		for _, entry := range inventory.Entries {
			if !entry.Kind.Supported() || entry.Kind == InventoryPOCWorkflow {
				t.Errorf("v1 writer output %s classified as %s", entry.Relative, entry.Kind)
			}
			observed[entry.Kind] = true
		}
	}
	for _, kind := range V1Kinds() {
		if !observed[kind] {
			t.Errorf("v1 kind %s is not produced by any writer in writeEveryV1Kind: exercise its store here", kind)
		}
	}
	names, err := os.ReadDir(state)
	if err != nil {
		t.Fatal(err)
	}
	written := []string{}
	for _, name := range names {
		written = append(written, name.Name())
	}
	// workflows/ holds only frozen historical POC records; no v1 writer uses it.
	expected := slices.DeleteFunc(slices.Clone(StateRootDirectories), func(name string) bool { return name == "workflows" })
	if !slices.Equal(written, expected) {
		t.Errorf("writers produced top-level directories %v, want every v1 directory %v", written, expected)
	}
	if os.Getenv(freezeCorpusEnv) != "" {
		corpus := filepath.Join("..", "compatibility", "testdata", "stable-v1")
		// Every t.TempDir of this test shares one parent; normalize all of them.
		machine := filepath.Dir(t.TempDir())
		freezeCorpus(t, portable, filepath.Join(corpus, "projects"), corpus, "projects", machine)
		freezeCorpus(t, state, filepath.Join(corpus, "state"), corpus, "state", machine)
	}
}

// writeEveryV1Kind drives each v1 store's public write path into fresh roots.
func writeEveryV1Kind(t *testing.T) (string, string) {
	t.Helper()
	base := privateTestRoot(t)
	portable := filepath.Join(base, "projects")
	source := filepath.Join(portable, "sample")
	if err := os.MkdirAll(source, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(portable, 0o700); err != nil {
		t.Fatal(err)
	}
	manifest := []byte("schemaVersion: 1\nproject:\n  id: 123e4567-e89b-42d3-a456-426614174000\n  slug: sample\n  name: Sample\n")
	if err := os.WriteFile(filepath.Join(source, manifestName), manifest, 0o600); err != nil {
		t.Fatal(err)
	}

	fixture := newCleanupFixture(t)
	state := fixture.state
	installations, err := NewInstallationStore(state)
	if err != nil {
		t.Fatal(err)
	}
	if got := installations.Install(context.Background(), source); got.Status != InstallationApplied {
		t.Fatalf("install = %+v", got)
	}

	workItems, err := NewWorkItemStore(state)
	if err != nil {
		t.Fatal(err)
	}
	link := testWorkItemLink()
	if err := workItems.Save(context.Background(), link); err != nil {
		t.Fatal(err)
	}
	target := workitem.DraftTarget{ProjectID: link.ProjectID, RepositoryKey: "main", Provider: "github", Resource: "owner/repo"}
	attempt := workitem.CreateAttempt{Target: target, Correlation: strings.Repeat("a", 64), PreviewDigest: strings.Repeat("b", 64), State: workitem.CreateAttemptConfirmed, ExternalID: "1"}
	if err := workItems.SaveCreateAttempt(context.Background(), attempt); err != nil {
		t.Fatal(err)
	}

	fixture.execution(t, workflow.ExecutionCompleted, validExecution().ExecutionID)
	fixture.create(t, 31*day, nil)
	fixture.cleanupAt(t, cleanupClock)
	fixture.create(t, day, nil)
	evidence := fixture.create(t, 800*day, evidenceDraft)
	fixture.retire(t, evidence.ID, cleanupClock)

	graphs, err := NewGraphStore(state)
	if err != nil {
		t.Fatal(err)
	}
	graph := localGraphFixture(t)
	if err := graphs.Create(context.Background(), graph); err != nil {
		t.Fatal(err)
	}
	streams, err := NewCoordinationStore(state, "project-1")
	if err != nil {
		t.Fatal(err)
	}
	if err := streams.Publish(context.Background(), coordinationRecord(graph)); err != nil {
		t.Fatal(err)
	}
	profiles, err := NewRuntimeProfileStore(state)
	if err != nil {
		t.Fatal(err)
	}
	if err := profiles.Create(context.Background(), runtimeConfiguration(1)); err != nil {
		t.Fatal(err)
	}
	return portable, state
}

// corpusMachineRoot replaces the temporary machine path writers record (for
// example an installation sourceLocation) so the public corpus carries no
// machine-specific path.
const corpusMachineRoot = "/axiom-corpus"

// freezeCorpus adds writer output to the stable corpus without ever replacing
// a frozen file: released state must stay readable exactly as it was written.
func freezeCorpus(t *testing.T, from, to, corpus, tree, machine string) {
	t.Helper()
	manifestPath := filepath.Join(corpus, "MANIFEST")
	existing, err := os.ReadFile(manifestPath)
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(existing)), "\n")
	if len(existing) == 0 {
		lines = nil
	}
	err = filepath.WalkDir(from, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		relative, _ := filepath.Rel(from, path)
		wire, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		wire = []byte(strings.ReplaceAll(string(wire), machine, corpusMachineRoot))
		destination := filepath.Join(to, relative)
		// A frozen file is never replaced, even when a newer writer differs.
		if _, err := os.Lstat(destination); err == nil {
			return nil
		}
		if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(destination, wire, 0o644); err != nil {
			return err
		}
		digest := sha256.Sum256(wire)
		lines = append(lines, fmt.Sprintf("%s  %s", hex.EncodeToString(digest[:]), filepath.ToSlash(filepath.Join(tree, relative))))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Slice(lines, func(i, j int) bool { return lines[i][66:] < lines[j][66:] })
	if err := os.WriteFile(manifestPath, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}
