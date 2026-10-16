package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/rgomids/axiom/internal/cli"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rgomids/axiom/internal/executiongraph"
	"github.com/rgomids/axiom/internal/local"
	"github.com/rgomids/axiom/internal/stageexecution"
	"github.com/rgomids/axiom/internal/stagepublication"
	"github.com/rgomids/axiom/internal/workflowcompiler"
)

func TestStageDisposerChecksDispositionWithoutEffects(t *testing.T) {
	d := stageDisposer{repository: "/synthetic/nonexistent-repository"}
	for _, disposition := range []string{executiongraph.DispositionPreserve, executiongraph.DispositionReset} {
		err := d.Dispose(context.Background(), executiongraph.SuccessorGraph{}, executiongraph.SuccessorChild{}, disposition)
		if err == nil || errors.Is(err, stageexecution.ErrDispositionRequired) {
			t.Fatalf("%s: %v", disposition, err)
		}
	}
	for _, disposition := range []string{"", executiongraph.DispositionNotRequired, "discard"} {
		if err := d.Dispose(context.Background(), executiongraph.SuccessorGraph{}, executiongraph.SuccessorChild{}, disposition); !errors.Is(err, stageexecution.ErrDispositionRequired) {
			t.Fatalf("%q: %v", disposition, err)
		}
	}
	guard := func(context.Context) error { return nil }
	if err := d.ResetChild(context.Background(), executiongraph.SuccessorGraph{}, "child", guard); err == nil {
		t.Fatal("reset without a managed worktree succeeded")
	}
}

func syntheticDispatch(t *testing.T) stageexecution.DispatchPreview {
	t.Helper()
	wire, err := os.ReadFile("../../internal/workflowcompiler/testdata/issue306/stage-plan-v2.json")
	if err != nil {
		t.Fatal(err)
	}
	var plan workflowcompiler.StagePlan
	if err = json.Unmarshal(wire, &plan); err != nil {
		t.Fatal(err)
	}
	r := executiongraph.PublicationRequest{Proposal: *plan.Compilation.Proposal, ExpectedDigest: plan.Compilation.Proposal.Digest, GraphRevision: plan.Execution.Revision, ParentAuthority: plan.Compilation.AuthorityCeiling, ChildAuthorities: map[string][]executiongraph.Effect{}, AuthorityReferences: map[string]string{}, Workspaces: map[string]string{}, Resolutions: map[string]executiongraph.Resolution{}}
	for _, a := range plan.Compilation.Agents {
		key, f := a.Input.NodeKey, a.Input.ResolvedProfile
		r.ChildAuthorities[key] = a.Effects
		r.AuthorityReferences[key] = "pending-dispatch-authority"
		r.Workspaces[key] = filepath.Join("/synthetic/stage/worktrees", a.AgentID)
		r.Resolutions[key] = executiongraph.Resolution{RuntimeID: f.RuntimeID, ModelProfileID: f.ProfileID, ConfigurationRevision: f.ConfigurationRevision, ObservationRevision: f.ObservationRevision, ResolvedProfile: f}
	}
	p := stageexecution.DispatchPreview{Plan: plan, Publication: &r, PolicyDigest: strings.Repeat("a", 64), ValidatorPolicyDigest: strings.Repeat("b", 64), AxiomDigest: strings.Repeat("c", 64)}
	wire, err = json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(wire)
	p.Digest = hex.EncodeToString(sum[:])
	return p
}

// This exercises the production publication adapter with local stores only.
// No runner, authenticator, detacher, Provider or network port exists here.
func TestRetainedDispatchPublicationAndSupervisorInput(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	graphs, err := local.NewGraphStore(root)
	if err != nil {
		t.Fatal(err)
	}
	leases, err := local.NewRunLeaseStore(root)
	if err != nil {
		t.Fatal(err)
	}
	executions, err := local.NewWorkflowStore(root)
	if err != nil {
		t.Fatal(err)
	}
	next := 0
	p := &retainedPublisher{LocalPublisher: stagepublication.LocalPublisher{Graphs: graphs, Leases: leases, Executions: executions, AllocateID: func() (string, error) { next++; return fmt.Sprintf("00000000-0000-4000-8000-%012d", next), nil }}}
	preview := syntheticDispatch(t)
	publication, err := p.Build(ctx, preview, "synthetic-operator", time.Unix(20, 0).UTC())
	if err != nil {
		t.Fatal(err)
	}
	if publication.Graph.Parent.ExecutionID == preview.Plan.Execution.ExecutionID {
		t.Fatal("graph parent reused Execution ID")
	}
	if err = p.Commit(ctx, publication); err != nil {
		t.Fatal(err)
	}
	read, err := p.ReadBack(ctx, publication)
	if err != nil {
		t.Fatal(err)
	}
	r, err := committedRequest(ctx, root, graphs, *read.Graph)
	if err != nil {
		t.Fatal(err)
	}
	retained, err := retainedPreview(ctx, graphs, r)
	if err != nil {
		t.Fatal(err)
	}
	if retained.Digest != preview.Digest || retained.Plan.Execution != preview.Plan.Execution {
		t.Fatal("retained input changed")
	}
	parent, err := graphs.FindStageGraph(ctx, r.ProjectID, preview.Plan.Execution.ExecutionID, preview.Plan.StageID)
	if err != nil || parent != r.ParentID {
		t.Fatal(parent, err)
	}
	for _, child := range read.Graph.Children {
		if len(child.Attempts) != 0 {
			t.Fatal("publication started an attempt")
		}
	}
	r.PreviewDigest = strings.Repeat("f", 64)
	if _, err = retainedPreview(ctx, graphs, r); err == nil {
		t.Fatal("foreign preview accepted")
	}
}

func TestRetainedDispatchRefusesForgedInnerDigest(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	graphs, err := local.NewGraphStore(root)
	if err != nil {
		t.Fatal(err)
	}
	p := syntheticDispatch(t)
	p.Plan.StageID = "foreign-stage"
	wire, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	r := stageexecution.SupervisorRequest{ProjectID: p.Plan.ProjectID, ParentID: "00000000-0000-4000-8000-000000000001", PreviewDigest: p.Digest}
	if _, err = graphs.CommitDispatchInputs(ctx, executiongraph.DispatchInputs{FormatVersion: 1, ProjectID: r.ProjectID, ParentID: r.ParentID, PreviewDigest: r.PreviewDigest, Preview: wire}); err != nil {
		t.Fatal(err)
	}
	if _, err = retainedPreview(ctx, graphs, r); err == nil {
		t.Fatal("forged inner digest accepted")
	}
	if _, err = graphs.FindStageGraph(ctx, r.ProjectID, p.Plan.Execution.ExecutionID, p.Plan.StageID); err == nil {
		t.Fatal("orphan inputs became a graph")
	}
}

func TestPublishRefusesAbsentRunningContext(t *testing.T) {
	root := t.TempDir()
	s := lifecycleService{stateRoot: root, provenance: unknownProvenance()}
	for _, key := range []string{"AXIOM_CORRELATION_ID", "AXIOM_EXECUTION_ID", "AXIOM_GRAPH_ID", "AXIOM_ATTEMPT_ID"} {
		t.Setenv(key, "")
	}
	result := s.stagePublish(context.Background(), cli.WorkflowStageInput{Kind: "progress", Fields: []string{"summary=synthetic"}})
	if result.Result.Category != "child_context_required" {
		t.Fatal(result.Result.Category)
	}
}
