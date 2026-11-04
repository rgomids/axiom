package local

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/rgomids/axiom/internal/executiongraph"
	"github.com/rgomids/axiom/internal/runtimeprofile"
	"github.com/rgomids/axiom/internal/workflow"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Codec-only historical fixtures; no operational legacy stores are recreated.
func localDispatch(t *testing.T) (executiongraph.SuccessorGraph, executiongraph.DispatchAuthority) {
	g := localSuccessor(t, "running")
	for i := range g.Children {
		g.Children[i].Attempts = nil
	}
	a := executiongraph.DispatchAuthority{FormatVersion: 1, ProjectID: g.Children[0].Envelope.Scope.ProjectID, PreviewDigest: strings.Repeat("a", 64), Scope: "graph-first-attempts", ParentID: g.Parent.ExecutionID, Revision: g.Parent.GraphRevision, ProposalDigest: g.Parent.ProposalDigest, ActorKind: "human-operator", Actor: "operator", RecordedAt: g.Parent.CreatedAt}
	for _, c := range g.Children {
		a.Children = append(a.Children, executiongraph.DispatchChildAuthority{ChildID: c.ExecutionID, Effects: c.Envelope.AllowedEffects})
	}
	d, e := executiongraph.DispatchAuthorityDigest(a)
	if e != nil {
		t.Fatal(e)
	}
	g.DispatchAuthorityRef = d
	for i := range g.Children {
		g.Children[i].Envelope.AuthorityReference = d
	}
	return g, a
}

func localSuccessor(t *testing.T, name string) executiongraph.SuccessorGraph {
	t.Helper()
	wire, err := os.ReadFile("../executiongraph/testdata/successor/valid/" + name + ".json")
	if err != nil {
		t.Fatal(err)
	}
	g, err := executiongraph.DecodeSuccessorGraph(wire)
	if err != nil {
		t.Fatal(err)
	}
	g.Children[0].Envelope.Controls.MaximumAttempts = 3
	return g
}

func localSingleStage(t *testing.T, state workflow.State) (executiongraph.SingleStageRun, executiongraph.SingleStageAuthority) {
	t.Helper()
	profile := &runtimeprofile.ResolvedProfileV4{ProfileID: "codex-high", RuntimeID: "codex", Model: "gpt-synthetic", Effort: runtimeprofile.Effort{Mode: runtimeprofile.EffortRuntimeDefault, Value: "default"}, ProfileDigest: strings.Repeat("1", 64), ConfigurationRevision: 1, ConfigurationDigest: strings.Repeat("2", 64), ExecutableDigest: strings.Repeat("3", 64), InvocationMode: "noninteractive", ObservationRevision: 1, ObservationDigest: strings.Repeat("4", 64), EffortSourceDigest: strings.Repeat("5", 64)}
	item := state.WorkItem
	scope := executiongraph.SingleStageScope{ProjectID: state.ProjectID, RepositoryKey: state.RepositoryKey, WorkItem: item.Provider + ":" + item.Resource + "#" + item.ExternalID, ExecutionID: state.ExecutionID, StageID: string(state.Stage), StageOrdinal: state.StageOrdinal}
	digest, err := executiongraph.SingleStageScopeDigest(scope)
	if err != nil {
		t.Fatal(err)
	}
	created := state.UpdatedAt.Add(time.Second).UTC()
	binding := executiongraph.SingleStageBinding{ExecutionGeneration: hexDigestRevision(state.StorageRevision), WorkflowRevision: state.Revision, PreviewDigest: strings.Repeat("7", 64), ProposalDigest: strings.Repeat("8", 64), InputDigest: strings.Repeat("9", 64), AxiomDigest: strings.Repeat("a", 64), Workspace: "/synthetic/worktrees/one", Inbox: "/synthetic/inboxes/one", Effects: []executiongraph.Effect{{Kind: "repository-write", Target: "src"}}, Controls: executiongraph.ExecutionControls{Timeout: time.Hour, MaximumAttempts: 3}, Resolution: executiongraph.Resolution{FrozenProfile: profile, RuntimeID: "codex", ModelProfileID: "codex-high", ConfigurationRevision: 1, ObservationRevision: 1}}
	a := executiongraph.SingleStageAuthority{FormatVersion: 2, Scope: executiongraph.SingleStageAuthorityScope, ProjectID: scope.ProjectID, ExecutionID: scope.ExecutionID, StageID: scope.StageID, StageOrdinal: scope.StageOrdinal, ScopeDigest: digest, ExecutionGeneration: binding.ExecutionGeneration, WorkflowRevision: binding.WorkflowRevision, PreviewDigest: binding.PreviewDigest, DispatchDigest: binding.PreviewDigest, ProposalDigest: binding.ProposalDigest, AttemptNumber: 1, Effects: binding.Effects, Disposition: executiongraph.DispositionNotRequired, ActorKind: "human-operator", Actor: "synthetic-operator", RecordedAt: created}
	ref, err := executiongraph.SingleStageAuthorityDigest(a)
	if err != nil {
		t.Fatal(err)
	}
	return executiongraph.SingleStageRun{Kind: executiongraph.SingleStageRunKind, FormatVersion: executiongraph.OperationalSingleStageRunVersion, ScopeDigest: digest, Scope: scope, Binding: binding, CreatedAt: created, Authorities: []string{ref}, Attempts: []executiongraph.SuccessorAttempt{}}, a
}

func localRetryAuthority(g executiongraph.SuccessorGraph) executiongraph.RetryAuthority {
	c := g.Children[0]
	disposition := executiongraph.DispositionPreserve
	return executiongraph.RetryAuthority{FormatVersion: 1, ProjectID: c.Envelope.Scope.ProjectID, ParentID: g.Parent.ExecutionID, Revision: g.Parent.GraphRevision, GraphDigest: fmt.Sprintf("%x", g.StorageRevision), PreviewDigest: strings.Repeat("c", 64), DispatchDigest: strings.Repeat("d", 64), ChildID: c.ExecutionID, PriorAttemptID: c.Attempts[len(c.Attempts)-1].AttemptID, AttemptNumber: uint32(len(c.Attempts)) + 1, Effects: c.Envelope.AllowedEffects, Disposition: disposition, ActorKind: "human-operator", Actor: "operator", RecordedAt: time.Unix(20, 0).UTC()}
}

func localDispatchInputs(g executiongraph.SuccessorGraph, executionID, stageID string) executiongraph.DispatchInputs {
	preview, _ := json.Marshal(map[string]any{"stagePlan": map[string]any{"executionRef": map[string]any{"executionId": executionID, "revision": g.Parent.GraphRevision}, "stageId": stageID}, "digest": "synthetic"})
	return executiongraph.DispatchInputs{FormatVersion: 1, ProjectID: g.Children[0].Envelope.Scope.ProjectID, ParentID: g.Parent.ExecutionID, PreviewDigest: g.DispatchAuthorityRef, Preview: preview}
}

func mustHistoricalWire(wire []byte, err error) []byte {
	if err != nil {
		panic(err)
	}
	return wire
}
func historicalFixtureWire(t *testing.T, root, family, project, name string, wire []byte) {
	t.Helper()
	dir := filepath.Join(root, family, "v1", project)
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), wire, 0600); err != nil {
		t.Fatal(err)
	}
}
func historicalFixtureDigestWire(t *testing.T, root, family, project string, wire []byte) {
	t.Helper()
	historicalFixtureWire(t, root, family, project, hexDigest(wire)+".json", wire)
}

// These raw codec fixtures exercise historical preservation without reinstating
// a legacy operational writer, CAS protocol or resume API.
func seedHistoricalExecutionFixture(t *testing.T, root string, state workflow.State) workflow.State {
	t.Helper()
	wire := mustHistoricalWire(encodeExecution(state))
	historicalFixtureWire(t, root, "executions", state.ProjectID, executionName(state.RepositoryKey, state.WorkItem), wire)
	state.StorageRevision = sha256.Sum256(wire)
	return state
}
func readHistoricalExecutionFixture(store WorkflowStore, state workflow.State) (workflow.State, error) {
	wire, err := os.ReadFile(filepath.Join(store.root, "executions", "v1", state.ProjectID, executionName(state.RepositoryKey, state.WorkItem)))
	if err != nil {
		return workflow.State{}, err
	}
	decoded, err := decodeExecution(wire)
	decoded.StorageRevision = sha256.Sum256(wire)
	if err != nil {
		return decoded, err
	}
	root, err := os.OpenRoot(filepath.Dir(filepath.Join(store.root, "executions", "v1", state.ProjectID, executionName(state.RepositoryKey, state.WorkItem))))
	if err != nil {
		return decoded, err
	}
	defer root.Close()
	return attachDeliveryDecision(root, decoded)
}
