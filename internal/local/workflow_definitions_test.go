package local

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/manifest"
	"github.com/rgomids/axiom/internal/project"
	"github.com/rgomids/axiom/internal/workflowdefinition"
)

func workflowStoreFixture(t *testing.T) (PortableStore, []byte, workflowdefinition.Document, []byte) {
	t.Helper()
	store, e := NewPortableStore(filepath.Join(privateTestRoot(t), "projects"))
	if e != nil {
		t.Fatal(e)
	}
	wire := []byte("schemaVersion: 1\nproject: {id: 12345678-1234-4abc-8def-123456789abc, slug: sample, name: Sample}\n")
	if e = store.Create(context.Background(), "sample", wire); e != nil {
		t.Fatal(e)
	}
	d := workflowdefinition.Builtin().Definition
	d.WorkflowID = "custom"
	doc, issues := workflowdefinition.Encode(d)
	if len(issues) > 0 {
		t.Fatal(issues)
	}
	idx := WorkflowIndex{SchemaVersion: 1, Revisions: []WorkflowEntry{{WorkflowID: d.WorkflowID, Revision: 1, Digest: doc.Digest, State: "published"}}}
	index, e := EncodeWorkflowIndex(idx)
	if e != nil {
		t.Fatal(e)
	}
	return store, wire, doc, index
}
func TestWorkflowImmutablePublicationAndInventory(t *testing.T) {
	ctx := context.Background()
	store, manifest, doc, index := workflowStoreFixture(t)
	if e := store.PublishWorkflow(ctx, "sample", manifest, nil, &doc, index); e != nil {
		t.Fatal(e)
	}
	catalog, e := store.WorkflowCatalog(ctx, "sample")
	if e != nil || len(catalog.Index.Revisions) != 1 || len(catalog.Documents) != 1 {
		t.Fatalf("catalog %+v %v", catalog, e)
	}
	if _, e = store.Read(ctx, "sample"); e != nil {
		t.Fatal(e)
	}
	altered := doc.Definition
	altered.Name = "Other"
	other, _ := workflowdefinition.Encode(altered)
	idx := WorkflowIndex{SchemaVersion: 1, Revisions: []WorkflowEntry{{WorkflowID: other.Definition.WorkflowID, Revision: 1, Digest: other.Digest, State: "published"}}}
	next, _ := EncodeWorkflowIndex(idx)
	if e = store.PublishWorkflow(ctx, "sample", manifest, index, &other, next); !errors.Is(e, ErrConflict) {
		t.Fatalf("reassigned immutable revision: %v", e)
	}
	inventory, e := InspectPortableInventory(ctx, store.root)
	if e != nil {
		t.Fatal(e)
	}
	kinds := map[InventoryKind]bool{}
	for _, entry := range inventory.Entries {
		if !entry.Kind.Supported() {
			t.Fatalf("unsupported %+v", entry)
		}
		kinds[entry.Kind] = true
	}
	if !kinds[InventoryWorkflowIndex] || !kinds[InventoryWorkflowDefinition] {
		t.Fatal("missing inventory")
	}
}
func TestWorkflowInterruptedContentAndIndexRecovery(t *testing.T) {
	for _, stage := range []FaultStage{FaultF3, FaultF4, FaultF5, FaultF6, FaultF7, FaultF8} {
		for _, object := range []string{"content", "index"} {
			t.Run(string(stage)+"-"+object, func(t *testing.T) {
				ctx := context.Background()
				store, wire, doc, index := workflowStoreFixture(t)
				calls := 0
				store.workflowFault = func(at FaultStage) error {
					if at == FaultF0 {
						calls++
					}
					if at == stage && (object == "content" && calls == 1 || object == "index" && calls == 2) {
						return ErrSimulatedInterruption
					}
					return nil
				}
				if e := store.PublishWorkflow(ctx, "sample", wire, nil, &doc, index); e == nil {
					t.Fatal("fault ignored")
				}
				store.workflowFault = nil
				if _, e := store.Read(ctx, "sample"); e == nil {
					t.Fatal("interrupted publication accepted")
				}
				roots := RecoveryRoots{Projects: store.root}
				report, e := InspectRecovery(ctx, roots)
				if e != nil || len(report.Plans) != 1 {
					t.Fatalf("recovery %+v %v", report, e)
				}
				plan := report.Plans[0]
				if plan.Action == PreservedReview {
					t.Fatalf("known protocol not classified: %+v", plan)
				}
				authority, e := AuthorizeRecovery(plan, plan.Digest)
				if e != nil {
					t.Fatal(e)
				}
				if _, e = ApplyRecovery(ctx, roots, plan, authority); e != nil {
					t.Fatal(e)
				}
				catalog, e := store.WorkflowCatalog(ctx, "sample")
				if e != nil {
					t.Fatal(e)
				}
				if len(catalog.Unindexed) > 0 {
					if len(catalog.Unindexed) != 1 || catalog.Unindexed[0] != doc.Ref("project") {
						t.Fatal("orphan identity lost")
					}
				}
				if len(catalog.Index.Revisions) == 0 {
					if e = store.PublishWorkflow(ctx, "sample", wire, catalog.Wire, &doc, index); e != nil {
						t.Fatal(e)
					}
				}
				if _, e = store.Read(ctx, "sample"); e != nil {
					t.Fatalf("recovered read %v", e)
				}
			})
		}
	}
}
func TestWorkflowCompanionsRejectTamperingAndLinks(t *testing.T) {
	store, wire, doc, index := workflowStoreFixture(t)
	ctx := context.Background()
	if e := store.PublishWorkflow(ctx, "sample", wire, nil, &doc, index); e != nil {
		t.Fatal(e)
	}
	name := filepath.Join(store.root, "sample", "workflows", "custom", "1.json")
	if e := os.WriteFile(name, []byte(`{"schemaVersion":99}`), 0o600); e != nil {
		t.Fatal(e)
	}
	if _, e := store.Read(ctx, "sample"); e == nil {
		t.Fatal("tampered content accepted")
	}
	if e := os.Remove(name); e != nil {
		t.Fatal(e)
	}
	if e := os.Symlink(filepath.Join(t.TempDir(), "foreign"), name); e != nil {
		t.Skip(e)
	}
	if _, e := store.Read(ctx, "sample"); e == nil {
		t.Fatal("linked content accepted")
	}
}

func TestWorkflowPublicationSupportsDefinitionByteWindow(t *testing.T) {
	store, wire, doc, _ := workflowStoreFixture(t)
	d := doc.Definition
	for i := range d.Stages {
		d.Stages[i].Instructions = strings.Repeat("a", 16000)
		d.Stages[i].Purpose = strings.Repeat("b", 16000)
		d.Stages[i].Agents[0].Responsibilities = strings.Repeat("c", 16000)
	}
	doc, issues := workflowdefinition.Encode(d)
	if len(issues) > 0 || len(doc.Canonical) <= MaxRecordBytes {
		t.Fatalf("large definition fixture: %d %v", len(doc.Canonical), issues)
	}
	index, e := EncodeWorkflowIndex(WorkflowIndex{SchemaVersion: 1, Revisions: []WorkflowEntry{{WorkflowID: d.WorkflowID, Revision: 1, Digest: doc.Digest, State: "published"}}})
	if e != nil {
		t.Fatal(e)
	}
	if e = store.PublishWorkflow(context.Background(), "sample", wire, nil, &doc, index); e != nil {
		t.Fatal(e)
	}
	if _, e = store.Read(context.Background(), "sample"); e != nil {
		t.Fatal(e)
	}
}

func TestWorkflowPublicationRechecksLocalAndSelectionIndexAuthority(t *testing.T) {
	ctx := context.Background()
	f := newEditPublicationFixture(t)
	observed, category := f.installation.Inspect(ctx, editTestProjectID)
	if category != "" {
		t.Fatal(category)
	}
	wire, err := f.portable.Read(ctx, "sample")
	if err != nil {
		t.Fatal(err)
	}
	d := workflowdefinition.Builtin().Definition
	d.WorkflowID = "custom"
	doc, issues := workflowdefinition.Encode(d)
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	idx, err := EncodeWorkflowIndex(WorkflowIndex{SchemaVersion: 1, Revisions: []WorkflowEntry{{WorkflowID: d.WorkflowID, Revision: 1, Digest: doc.Digest, State: "published"}}})
	if err != nil {
		t.Fatal(err)
	}
	changed := append(append([]byte(nil), observed.Wire...), '\n')
	if err := os.WriteFile(f.recordPath(), changed, 0600); err != nil {
		t.Fatal(err)
	}
	if err := f.installation.PublishWorkflow(ctx, f.portable, editTestProjectID, "sample", observed.Wire, wire, nil, &doc, idx); !errors.Is(err, ErrConflict) {
		t.Fatalf("stale local authority: %v", err)
	}
	if _, err := os.Stat(filepath.Join(f.source, "workflows")); !os.IsNotExist(err) {
		t.Fatalf("stale authority produced effects: %v", err)
	}
	if err := f.installation.PublishWorkflow(ctx, f.portable, editTestProjectID, "sample", changed, wire, nil, &doc, idx); err != nil {
		t.Fatal(err)
	}
	p, problems := manifest.Decode(wire)
	if len(problems) != 0 {
		t.Fatal(problems)
	}
	ref := workflowdefinition.Builtin().Ref("builtin")
	next, problems := p.SelectWorkflow(project.WorkflowSelection{WorkflowID: ref.WorkflowID, Revision: ref.Revision, Digest: ref.Digest, Source: ref.Source})
	if len(problems) != 0 {
		t.Fatal(problems)
	}
	nextWire, problems := manifest.Encode(next)
	if len(problems) != 0 {
		t.Fatal(problems)
	}
	if err := f.portable.WithWorkflowIndex(nil).Update(ctx, "sample", wire, nextWire); !errors.Is(err, ErrConflict) {
		t.Fatalf("stale catalog selection: %v", err)
	}
	current, err := f.portable.Read(ctx, "sample")
	if err != nil || string(current) != string(wire) {
		t.Fatalf("selection changed: %v", err)
	}
	if err := f.portable.WithWorkflowIndex(idx).Update(ctx, "sample", wire, nextWire); err != nil {
		t.Fatal(err)
	}
}

func TestWorkflowIndexRejectsUnknownMissingAndCaseAliases(t *testing.T) {
	_, _, _, index := workflowStoreFixture(t)
	for _, invalid := range []string{
		strings.Replace(string(index), "schemaVersion", "SchemaVersion", 1),
		strings.Replace(string(index), "workflowId", "WorkflowID", 1),
		strings.Replace(string(index), `"state":"published"`, `"state":"published","unknown":true`, 1),
		strings.Replace(string(index), `"schemaVersion":1,`, "", 1),
		strings.Replace(string(index), `"revisions":`, `"revisions":null,"other":`, 1),
	} {
		if _, err := DecodeWorkflowIndex([]byte(invalid)); err == nil {
			t.Fatal("non-closed index accepted")
		}
	}
}

func TestWorkflowPublicationAcceptsTrustedMacOSSourceAlias(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("macOS trusted host alias")
	}
	ctx := context.Background()
	f := newEditPublicationFixture(t)
	if !strings.HasPrefix(f.source, "/private/var/") {
		t.Skip("host does not expose /var alias")
	}
	observed, category := f.installation.Inspect(ctx, editTestProjectID)
	if category != "" {
		t.Fatal(category)
	}
	state := observed.Record.State()
	state.SourceLocation = strings.TrimPrefix(f.source, "/private")
	record, problems := NewRecord(state)
	if len(problems) != 0 {
		t.Fatal(problems)
	}
	localWire, problems := EncodeRecord(record)
	if len(problems) != 0 {
		t.Fatal(problems)
	}
	if err := os.WriteFile(f.recordPath(), localWire, 0600); err != nil {
		t.Fatal(err)
	}
	wire, err := f.portable.Read(ctx, "sample")
	if err != nil {
		t.Fatal(err)
	}
	d := workflowdefinition.Builtin().Definition
	d.WorkflowID = "custom"
	doc, issues := workflowdefinition.Encode(d)
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	idx, err := EncodeWorkflowIndex(WorkflowIndex{SchemaVersion: 1, Revisions: []WorkflowEntry{{WorkflowID: d.WorkflowID, Revision: 1, Digest: doc.Digest, State: "published"}}})
	if err != nil {
		t.Fatal(err)
	}
	if err := f.installation.PublishWorkflow(ctx, f.portable, editTestProjectID, "sample", localWire, wire, nil, &doc, idx); err != nil {
		t.Fatalf("trusted macOS alias refused: %v", err)
	}
}
