package local

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rgomids/axiom/internal/executiongraph"
	"github.com/rgomids/axiom/internal/workflow"
)

// createLocalV3Execution persists the delivered format-3 fixture and returns
// it as loaded, with its exact ledger storage revision.
func createLocalV3Execution(t *testing.T, s WorkflowStore) workflow.State {
	t.Helper()
	wire, err := os.ReadFile("../workflowcompiler/testdata/issue306/execution-v3.json")
	if err != nil {
		t.Fatal(err)
	}
	state, err := decodeExecution(wire)
	if err != nil {
		t.Fatal(err)
	}
	state.StorageRevision = [32]byte{}
	if err := s.Create(context.Background(), state); err != nil {
		t.Fatal(err)
	}
	loaded, err := s.Load(context.Background(), state.ProjectID, state.RepositoryKey, state.WorkItem)
	if err != nil {
		t.Fatal(err)
	}
	return loaded
}

func localStageOutcome(state workflow.State, integrationAttempt string) executiongraph.StageOutcome {
	rollup := executiongraph.StageRollup{Status: "success", ParentID: state.ExecutionID, GraphRevision: state.Revision, IntegrationChildID: "00000000-0000-4000-8000-000000000093", IntegrationAttemptID: integrationAttempt, Children: []executiongraph.StageChildOutcome{
		{ChildID: "00000000-0000-4000-8000-000000000092", AttemptID: "00000000-0000-4000-8000-000000000094", Status: executiongraph.AttemptSucceeded, Reason: executiongraph.ResultRecorded, Required: true},
		{ChildID: "00000000-0000-4000-8000-000000000093", AttemptID: integrationAttempt, Status: executiongraph.AttemptSucceeded, Reason: executiongraph.IntegrationApplied, Required: true},
	}}
	return executiongraph.StageOutcome{FormatVersion: 1, ProjectID: state.ProjectID, ExecutionID: state.ExecutionID, ExecutionRevision: state.Revision, LedgerDigest: hexDigestRevision(state.StorageRevision), StageID: string(state.Stage), StageOrdinal: state.StageOrdinal, GraphDigest: strings.Repeat("2", 64), IntegrationReceiptRef: strings.Repeat("3", 64), ValidatorPolicyDigest: strings.Repeat("4", 64), Rollup: rollup, IntegrationResultReference: strings.Repeat("5", 64), ValidationOutputDigest: strings.Repeat("6", 64), RecordedAt: time.Unix(30, 0).UTC()}
}

func TestStageOutcomeProjectionBindsLedgerGeneration(t *testing.T) {
	ctx := context.Background()
	root := privateTestRoot(t)
	s, err := NewWorkflowStore(root)
	if err != nil {
		t.Fatal(err)
	}
	state := createLocalV3Execution(t, s)
	before, err := os.ReadFile(executionPath(t, root, state))
	if err != nil {
		t.Fatal(err)
	}
	o := localStageOutcome(state, "00000000-0000-4000-8000-000000000095")
	ref, err := s.RecordStageOutcome(ctx, state, o)
	if err != nil {
		t.Fatal(err)
	}
	if again, err := s.RecordStageOutcome(ctx, state, o); err != nil || again != ref {
		t.Fatalf("repeat projection = %s %v", again, err)
	}
	other := o
	other.RecordedAt = time.Unix(31, 0).UTC()
	if _, err := s.RecordStageOutcome(ctx, state, other); !errors.Is(err, ErrConflict) {
		t.Fatalf("second outcome for one attempt = %v", err)
	}
	outcomes, err := s.LoadStageOutcomes(ctx, state.ProjectID, state.ExecutionID)
	if err != nil || len(outcomes) != 1 || outcomes[ref].Rollup.IntegrationAttemptID != o.Rollup.IntegrationAttemptID {
		t.Fatalf("outcomes = %+v %v", outcomes, err)
	}
	// Execution3 bytes are never rewritten by the projection.
	after, err := os.ReadFile(executionPath(t, root, state))
	if err != nil || string(after) != string(before) {
		t.Fatal("projection changed the Execution ledger bytes")
	}
	if loaded, err := s.Load(ctx, state.ProjectID, state.RepositoryKey, state.WorkItem); err != nil || loaded.StorageRevision != state.StorageRevision {
		t.Fatalf("ledger reload = %v", err)
	}
	inventory, err := InspectStateInventory(ctx, root)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, entry := range inventory.Entries {
		if strings.HasSuffix(entry.Relative, ref+".json") {
			found = entry.Kind == InventoryStageOutcome
		}
		if !entry.Kind.Supported() {
			t.Errorf("%s classified %s", entry.Relative, entry.Kind)
		}
	}
	if !found {
		t.Fatal("stage outcome not inventoried")
	}
}

func TestStageOutcomeProjectionRefusals(t *testing.T) {
	ctx := context.Background()
	root := privateTestRoot(t)
	s, err := NewWorkflowStore(root)
	if err != nil {
		t.Fatal(err)
	}
	state := createLocalV3Execution(t, s)
	t.Run("stale ledger generation", func(t *testing.T) {
		stale := state
		stale.StorageRevision[0] ^= 1
		o := localStageOutcome(stale, "00000000-0000-4000-8000-000000000096")
		if _, err := s.RecordStageOutcome(ctx, stale, o); !errors.Is(err, ErrConflict) {
			t.Fatalf("stale ledger = %v", err)
		}
	})
	t.Run("outcome for another Execution generation", func(t *testing.T) {
		o := localStageOutcome(state, "00000000-0000-4000-8000-000000000097")
		o.ExecutionRevision++
		o.Rollup.GraphRevision++
		if _, err := s.RecordStageOutcome(ctx, state, o); !errors.Is(err, ErrUnsafe) {
			t.Fatalf("foreign generation = %v", err)
		}
	})
	t.Run("unknown integration never projected", func(t *testing.T) {
		o := localStageOutcome(state, "00000000-0000-4000-8000-000000000098")
		o.Rollup.Status = "recovery_required"
		if _, err := s.RecordStageOutcome(ctx, state, o); err == nil {
			t.Fatal("unknown integration projected")
		}
	})
	t.Run("tampered record", func(t *testing.T) {
		o := localStageOutcome(state, "00000000-0000-4000-8000-000000000099")
		wire, err := executiongraph.EncodeStageOutcome(o)
		if err != nil {
			t.Fatal(err)
		}
		if kind := classifyStageOutcome(state.ProjectID, strings.Repeat("0", 64)+".json", wire); kind != InventoryMalformed {
			t.Fatalf("renamed outcome inventoried as %s", kind)
		}
	})
}

func TestIntegrationReceiptCommitLoadAndInventory(t *testing.T) {
	ctx := context.Background()
	root, graphs, _, g := committedLocalDispatch(t)
	a := localIntegrationReceipt(g)
	ref, err := graphs.CommitIntegrationReceipt(ctx, g, a)
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := graphs.LoadIntegrationReceipt(ctx, a.ProjectID, a.ParentID, ref)
	if err != nil || executiongraph.CheckIntegrationDispatch(g, loaded) != nil {
		t.Fatalf("receipt readback = %v", err)
	}
	if _, err := graphs.LoadRetry(ctx, a.ProjectID, a.ParentID, ref); !errors.Is(err, ErrUnsafe) {
		t.Fatalf("integration receipt accepted as retry authority: %v", err)
	}
	inventory, err := InspectStateInventory(ctx, root)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, entry := range inventory.Entries {
		if strings.HasSuffix(entry.Relative, ref+".json") {
			found = entry.Kind == InventoryIntegrationAuthority
		}
	}
	if !found {
		t.Fatal("integration receipt not inventoried")
	}
	wrong := a
	wrong.AttemptNumber = 2
	if _, err := graphs.CommitIntegrationReceipt(ctx, g, wrong); err == nil {
		t.Fatal("receipt for a later integration attempt committed")
	}
}

func localIntegrationReceipt(g executiongraph.SuccessorGraph) executiongraph.IntegrationReceipt {
	return executiongraph.IntegrationReceipt{FormatVersion: 2, ProjectID: g.Children[0].Envelope.Scope.ProjectID, ParentID: g.Parent.ExecutionID, Revision: g.Parent.GraphRevision, GraphDigest: hexDigestRevision(g.StorageRevision), PreviewDigest: strings.Repeat("a", 64), IntegrationPreviewDigest: strings.Repeat("b", 64), ValidatorPolicyDigest: strings.Repeat("c", 64), IntegrationChildID: g.Parent.IntegrationChild, ExecutionID: "00000000-0000-4000-8000-0000000005e1", AttemptNumber: 1, OptionalWaivers: []executiongraph.OptionalWaiver{}, Effects: []executiongraph.Effect{}, ExecutionRevision: g.Parent.GraphRevision, LedgerDigest: strings.Repeat("d", 64), StageID: "implementation", ActorKind: "human-operator", Actor: "operator", RecordedAt: time.Unix(40, 0).UTC()}
}

func executionPath(t *testing.T, root string, state workflow.State) string {
	t.Helper()
	return filepath.Join(root, "executions", "v1", state.ProjectID, executionName(state.RepositoryKey, state.WorkItem))
}
