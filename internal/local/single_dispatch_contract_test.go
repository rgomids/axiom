package local

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/rgomids/axiom/internal/executiongraph"
	"github.com/rgomids/axiom/internal/workflow"
)

func TestSingleDispatchCannotInventAnExecution5AttemptGeneration(t *testing.T) {
	store, err := NewWorkflowStore(privateTestRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	state := createLocalOperationalExecution(t, store)
	ledger, err := encodeExecution(state)
	if err != nil {
		t.Fatal(err)
	}
	a := executiongraph.DispatchAuthority{FormatVersion: 1, ProjectID: state.ProjectID, ParentID: state.ExecutionID, Revision: state.Revision, Scope: "single-first-attempt", PreviewDigest: strings.Repeat("a", 64), ProposalDigest: strings.Repeat("b", 64), LedgerDigest: hexDigest(ledger), SingleStageID: string(state.Stage), SingleStageOrdinal: state.StageOrdinal, SingleAttemptID: "00000000-0000-4000-8000-000000000050", Children: []executiongraph.DispatchChildAuthority{{ChildID: state.ExecutionID, Effects: []executiongraph.Effect{}}}, ActorKind: "human-operator", Actor: "synthetic-operator", RecordedAt: state.UpdatedAt}
	if err := store.CommitSingleDispatch(context.Background(), state, a, a.PreviewDigest); !errors.Is(err, ErrConflict) {
		t.Fatalf("unchanged ledger bypassed generation fencing: %v", err)
	}
	a.RecordedAt = state.UpdatedAt.Add(time.Second)
	if err := store.CommitSingleDispatch(context.Background(), state, a, a.PreviewDigest); !errors.Is(err, ErrUnsafe) {
		t.Fatalf("authority newer than the ledger accepted: %v", err)
	}
	changedTime := state
	changedTime.UpdatedAt = a.RecordedAt
	if workflow.ValidState(changedTime) {
		t.Fatal("timestamp-only attempt update changed workflow truth")
	}
	changedRevision := state
	changedRevision.Revision++
	if workflow.ValidState(changedRevision) {
		t.Fatal("attempt counter reinterpreted as workflow revision")
	}
	var changed map[string]any
	if err := json.Unmarshal(ledger, &changed); err != nil {
		t.Fatal(err)
	}
	changed["stageAttempts"] = []any{}
	withAttempts, err := json.Marshal(changed)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := decodeExecution(withAttempts); err == nil {
		t.Fatal("attempt fields silently extended the closed Execution5 codec")
	}
	stored, err := store.Load(context.Background(), state.ProjectID, state.RepositoryKey, state.WorkItem)
	if err != nil || stored.StorageRevision != state.StorageRevision {
		t.Fatal("refusal changed workflow truth", err)
	}
}
