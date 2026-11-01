package local

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rgomids/axiom/internal/executiongraph"
	"github.com/rgomids/axiom/internal/runtimeprofile"
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
	return seedHistoricalExecutionFixture(t, s.root, state)
}

func createLocalOperationalExecution(t *testing.T, s WorkflowStore) workflow.State {
	t.Helper()
	wire, err := os.ReadFile("../workflowcompiler/testdata/issue306/execution-v3.json")
	if err != nil {
		t.Fatal(err)
	}
	state, err := decodeExecution(wire)
	if err != nil {
		t.Fatal(err)
	}
	profileWire, err := os.ReadFile("../runtimeprofile/testdata/axm36/resolved-v4.json")
	if err != nil {
		t.Fatal(err)
	}
	var profile runtimeprofile.ResolvedProfileV4
	if err := json.Unmarshal(profileWire, &profile); err != nil {
		t.Fatal(err)
	}
	state.FormatVersion = workflow.OperationalConfiguredFormatVersion
	state.Binding.ResolvedProfile = nil
	state.Binding.FrozenProfile = &profile
	state.StorageRevision = [32]byte{}
	return seedHistoricalExecutionFixture(t, s.root, state)
}

func localStageOutcome(state workflow.State, integrationAttempt string) executiongraph.StageOutcome {
	rollup := executiongraph.StageRollup{Status: "success", ParentID: state.ExecutionID, GraphRevision: state.Revision, IntegrationChildID: "00000000-0000-4000-8000-000000000093", IntegrationAttemptID: integrationAttempt, Children: []executiongraph.StageChildOutcome{
		{ChildID: "00000000-0000-4000-8000-000000000092", AttemptID: "00000000-0000-4000-8000-000000000094", Status: executiongraph.AttemptSucceeded, Reason: executiongraph.ResultRecorded, Required: true},
		{ChildID: "00000000-0000-4000-8000-000000000093", AttemptID: integrationAttempt, Status: executiongraph.AttemptSucceeded, Reason: executiongraph.IntegrationApplied, Required: true},
	}}
	return executiongraph.StageOutcome{FormatVersion: 1, ProjectID: state.ProjectID, ExecutionID: state.ExecutionID, ExecutionRevision: state.Revision, LedgerDigest: hexDigestRevision(state.StorageRevision), StageID: string(state.Stage), StageOrdinal: state.StageOrdinal, GraphDigest: strings.Repeat("2", 64), IntegrationReceiptRef: strings.Repeat("3", 64), ValidatorPolicyDigest: strings.Repeat("4", 64), Rollup: rollup, IntegrationResultReference: strings.Repeat("5", 64), ValidationOutputDigest: strings.Repeat("6", 64), RecordedAt: time.Unix(30, 0).UTC()}
}

func localIntegrationReceipt(g executiongraph.SuccessorGraph) executiongraph.IntegrationReceipt {
	return executiongraph.IntegrationReceipt{FormatVersion: 2, ProjectID: g.Children[0].Envelope.Scope.ProjectID, ParentID: g.Parent.ExecutionID, Revision: g.Parent.GraphRevision, GraphDigest: hexDigestRevision(g.StorageRevision), PreviewDigest: strings.Repeat("a", 64), IntegrationPreviewDigest: strings.Repeat("b", 64), ValidatorPolicyDigest: strings.Repeat("c", 64), IntegrationChildID: g.Parent.IntegrationChild, ExecutionID: "00000000-0000-4000-8000-0000000005e1", AttemptNumber: 1, OptionalWaivers: []executiongraph.OptionalWaiver{}, Effects: []executiongraph.Effect{}, ExecutionRevision: g.Parent.GraphRevision, LedgerDigest: strings.Repeat("d", 64), StageID: "implementation", ActorKind: "human-operator", Actor: "operator", RecordedAt: time.Unix(40, 0).UTC()}
}

func executionPath(t *testing.T, root string, state workflow.State) string {
	t.Helper()
	return filepath.Join(root, "executions", "v1", state.ProjectID, executionName(state.RepositoryKey, state.WorkItem))
}
