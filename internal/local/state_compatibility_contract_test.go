package local

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/manifest"
	"github.com/rgomids/axiom/internal/project"
	"github.com/rgomids/axiom/internal/projectapp"
	"github.com/rgomids/axiom/internal/workflow"
	"github.com/rgomids/axiom/internal/workflowdefinition"
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
	for _, kind := range append(V1Kinds(), AdditiveKinds()...) {
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
	if label := os.Getenv(freezeCorpusEnv); label != "" {
		corpus := filepath.Join("..", "compatibility", "testdata", "stable-v1")
		// Every t.TempDir of this test shares one parent; normalize all of them.
		machine := filepath.Dir(t.TempDir())
		frozen, err := freezeSnapshot(corpus, label, map[string]string{"projects": portable, "state": state}, machine)
		if err != nil {
			t.Fatal(err)
		}
		if frozen {
			t.Logf("froze snapshot %s; commit it with MANIFEST", label)
		} else {
			t.Logf("writer output is already preserved byte-for-byte by an existing snapshot; nothing frozen")
		}
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
	manifestWire := []byte("schemaVersion: 1\nproject:\n  id: 123e4567-e89b-42d3-a456-426614174000\n  slug: sample\n  name: Sample\n")
	if err := os.WriteFile(filepath.Join(source, manifestName), manifestWire, 0o600); err != nil {
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
	operational, err := NewOperationalStore(state)
	if err != nil {
		t.Fatal(err)
	}
	archivedState := projectapp.OperationalState{ProjectStatus: projectapp.ProjectArchived, DisabledIntegrations: []string{"work-items"}}
	if err := operational.CommitOperational(context.Background(), "123e4567-e89b-42d3-a456-426614174000", projectapp.OperationalRevisionAbsent, archivedState); err != nil {
		t.Fatal(err)
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
	contexts, err := NewInstallationStore(state)
	if err != nil {
		t.Fatal(err)
	}
	if err := contexts.WriteProjectContext(context.Background(), "", validExecution().ProjectID); err != nil {
		t.Fatal(err)
	}
	// Exercise schema 1/2/3 writers independently; the selected schema 4
	// candidate comes from the same explicit upgrade operation as the product.
	ps, e := NewPortableStore(portable)
	if e != nil {
		t.Fatal(e)
	}
	for version := 1; version <= 3; version++ {
		p, issues := project.New(project.State{SchemaVersion: version, ID: "87654321-4321-4abc-bdef-123456789abc", Slug: fmt.Sprintf("schema-%d", version), Name: "Compatibility"})
		if len(issues) > 0 {
			t.Fatal(issues)
		}
		b, issues := manifest.Encode(p)
		if len(issues) > 0 {
			t.Fatal(issues)
		}
		if e = ps.Create(context.Background(), p.State().Slug, b); e != nil {
			t.Fatal(e)
		}
	}
	doc := workflowdefinition.Builtin()
	d := doc.Definition
	d.WorkflowID = "custom"
	doc, definitionIssues := workflowdefinition.Encode(d)
	if len(definitionIssues) > 0 {
		t.Fatal(definitionIssues)
	}
	index := WorkflowIndex{SchemaVersion: 1, Revisions: []WorkflowEntry{{WorkflowID: d.WorkflowID, Revision: 1, Digest: doc.Digest, State: "published"}}}
	indexWire, e := EncodeWorkflowIndex(index)
	if e != nil {
		t.Fatal(e)
	}
	if e = ps.PublishWorkflow(context.Background(), "sample", manifestWire, nil, &doc, indexWire); e != nil {
		t.Fatal(e)
	}
	p, issues := manifest.Decode(manifestWire)
	if len(issues) > 0 {
		t.Fatal(issues)
	}
	next, issues := p.SelectWorkflow(project.WorkflowSelection{WorkflowID: d.WorkflowID, Revision: 1, Digest: doc.Digest, Source: "project"})
	if len(issues) > 0 {
		t.Fatal(issues)
	}
	nextWire, issues := manifest.Encode(next)
	if len(issues) > 0 {
		t.Fatal(issues)
	}
	if e = ps.Update(context.Background(), "sample", manifestWire, nextWire); e != nil {
		t.Fatal(e)
	}
	observation, category := installations.Inspect(context.Background(), p.State().ID)
	if category != "" {
		t.Fatal(category)
	}
	snap, problems := projectapp.ReadSnapshot(manifest.Codec{}, nextWire, nil)
	if len(problems) > 0 {
		t.Fatal(problems)
	}
	localState := observation.Record.State()
	localState.PortableRevision = snap.Revision()
	localState.ArtifactDigests = snap.Digests()
	localRecord, recordIssues := NewRecord(localState)
	if len(recordIssues) > 0 {
		t.Fatal(recordIssues)
	}
	localWire, recordIssues := EncodeRecord(localRecord)
	if len(recordIssues) > 0 {
		t.Fatal(recordIssues)
	}
	if e = installations.ReplaceRecord(context.Background(), p.State().ID, observation.Wire, localWire); e != nil {
		t.Fatal(e)
	}

	return portable, state
}

// corpusMachineRoot replaces the temporary machine path writers record (for
// example an installation sourceLocation) so the public corpus carries no
// machine-specific path.
const corpusMachineRoot = "/axiom-corpus"

// corpusSnapshotLabel names the release whose writers produced a snapshot.
var corpusSnapshotLabel = regexp.MustCompile(`^v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.]+)?$`)

// freezeSnapshot preserves writer output as a complete, release-labelled
// snapshot below <corpus>/snapshots/<label>. Snapshots are append-only: a
// frozen file or label is never replaced, and a representation that differs
// from every frozen snapshot, even at an identical logical path, is always
// added as a new snapshot instead of being skipped. Output identical to an
// existing snapshot is not duplicated; it reports false.
func freezeSnapshot(corpus, label string, trees map[string]string, machine string) (bool, error) {
	if !corpusSnapshotLabel.MatchString(label) {
		return false, fmt.Errorf("%s=%q: want the release whose writers produced the state, for example v0.4.0", freezeCorpusEnv, label)
	}
	output := map[string]string{}
	wires := map[string][]byte{}
	names := make([]string, 0, len(trees))
	for tree := range trees {
		names = append(names, tree)
	}
	sort.Strings(names)
	for _, tree := range names {
		from := trees[tree]
		err := filepath.WalkDir(from, func(path string, entry fs.DirEntry, err error) error {
			if err != nil || entry.IsDir() {
				return err
			}
			relative, _ := filepath.Rel(from, path)
			wire, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			// JSON escapes Windows backslashes; normalize that representation too.
			escapedMachine, _ := json.Marshal(machine)
			wire = []byte(strings.ReplaceAll(string(wire), string(escapedMachine[1:len(escapedMachine)-1]), corpusMachineRoot))
			wire = []byte(strings.ReplaceAll(string(wire), machine, corpusMachineRoot))
			logical := tree + "/" + filepath.ToSlash(relative)
			digest := sha256.Sum256(wire)
			output[logical] = hex.EncodeToString(digest[:])
			wires[logical] = wire
			return nil
		})
		if err != nil {
			return false, err
		}
	}
	manifestPath := filepath.Join(corpus, "MANIFEST")
	existing, err := os.ReadFile(manifestPath)
	if err != nil && !os.IsNotExist(err) {
		return false, err
	}
	lines := []string{}
	snapshots := map[string]map[string]string{}
	for _, line := range strings.Split(strings.TrimSpace(string(existing)), "\n") {
		if line == "" {
			continue
		}
		digest, path, ok := strings.Cut(line, "  ")
		parts := strings.SplitN(path, "/", 3)
		if !ok || len(parts) != 3 || parts[0] != "snapshots" {
			return false, fmt.Errorf("malformed MANIFEST line %q", line)
		}
		if snapshots[parts[1]] == nil {
			snapshots[parts[1]] = map[string]string{}
		}
		snapshots[parts[1]][parts[2]] = digest
		lines = append(lines, line)
	}
	for _, frozen := range snapshots {
		if maps.Equal(frozen, output) {
			return false, nil
		}
	}
	destination := filepath.Join(corpus, "snapshots", label)
	if _, err := os.Lstat(destination); err == nil || snapshots[label] != nil {
		return false, fmt.Errorf("snapshot %s is already frozen with a different representation; freeze this output under a new release label", label)
	}
	logicals := make([]string, 0, len(output))
	for logical := range output {
		logicals = append(logicals, logical)
	}
	sort.Strings(logicals)
	for _, logical := range logicals {
		path := filepath.Join(destination, filepath.FromSlash(logical))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return false, err
		}
		file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if err != nil {
			return false, err
		}
		_, err = file.Write(wires[logical])
		if closeErr := file.Close(); err == nil {
			err = closeErr
		}
		if err != nil {
			return false, err
		}
		lines = append(lines, fmt.Sprintf("%s  snapshots/%s/%s", output[logical], label, logical))
	}
	sort.Slice(lines, func(i, j int) bool { return lines[i][66:] < lines[j][66:] })
	return true, os.WriteFile(manifestPath, []byte(strings.Join(lines, "\n")+"\n"), 0o644)
}

// A release that writes different bytes at a logical path an earlier release
// already used must add a snapshot, never skip, replace or hide the earlier
// representation.
func TestFreezeSnapshotPreservesEveryRepresentation(t *testing.T) {
	corpus := t.TempDir()
	output := filepath.Join(t.TempDir(), "state")
	record := filepath.Join(output, "graphs", "v1", "p", "record.json")
	write := func(content string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(record), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(record, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	freeze := func(label string) (bool, error) {
		return freezeSnapshot(corpus, label, map[string]string{"state": output}, "/unused-machine-root")
	}
	read := func(label string) string {
		t.Helper()
		wire, err := os.ReadFile(filepath.Join(corpus, "snapshots", label, "state", "graphs", "v1", "p", "record.json"))
		if err != nil {
			t.Fatal(err)
		}
		return string(wire)
	}

	write("representation-x")
	if frozen, err := freeze("v1.0.0"); err != nil || !frozen {
		t.Fatalf("first freeze = %v, %v", frozen, err)
	}
	if frozen, err := freeze("v1.0.1"); err != nil || frozen {
		t.Fatalf("identical output must not add a snapshot: %v, %v", frozen, err)
	}
	if _, err := os.Lstat(filepath.Join(corpus, "snapshots", "v1.0.1")); !os.IsNotExist(err) {
		t.Fatalf("identical output created a snapshot: %v", err)
	}

	write("representation-y")
	if _, err := freeze("v1.0.0"); err == nil {
		t.Fatal("a different representation replaced a frozen release snapshot")
	}
	if frozen, err := freeze("v1.1.0"); err != nil || !frozen {
		t.Fatalf("different representation at the same logical path was not preserved: %v, %v", frozen, err)
	}
	if read("v1.0.0") != "representation-x" || read("v1.1.0") != "representation-y" {
		t.Fatalf("snapshots = %q, %q", read("v1.0.0"), read("v1.1.0"))
	}
	manifest, err := os.ReadFile(filepath.Join(corpus, "MANIFEST"))
	if err != nil {
		t.Fatal(err)
	}
	for _, label := range []string{"v1.0.0", "v1.1.0"} {
		if !strings.Contains(string(manifest), "  snapshots/"+label+"/state/graphs/v1/p/record.json\n") {
			t.Errorf("MANIFEST does not list %s representation:\n%s", label, manifest)
		}
	}

	write("representation-x")
	if frozen, err := freeze("v1.2.0"); err != nil || frozen {
		t.Fatalf("output identical to an older snapshot must not be duplicated: %v, %v", frozen, err)
	}
	if _, err := freeze("1.2.0"); err == nil {
		t.Fatal("a snapshot label that does not name a release was accepted")
	}
}

func TestFreezeSnapshotNormalizesWindowsJSONPaths(t *testing.T) {
	source, corpus := t.TempDir(), filepath.Join(t.TempDir(), "corpus")
	machine := `C:\synthetic-machine`
	wire, err := json.Marshal(map[string]string{"location": machine + `\owned`})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "fixture.json"), append(wire, '\n'), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := freezeSnapshot(corpus, "v0.8.1-issue233", map[string]string{"state": source}, machine); err != nil {
		t.Fatal(err)
	}
	normalized, err := os.ReadFile(filepath.Join(corpus, "snapshots", "v0.8.1-issue233", "state", "fixture.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(normalized), corpusMachineRoot) || strings.Contains(string(normalized), "synthetic-machine") {
		t.Fatalf("machine path retained: %s", normalized)
	}
}
