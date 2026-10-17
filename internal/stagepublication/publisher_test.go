package stagepublication

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/rgomids/axiom/internal/executiongraph"
	"github.com/rgomids/axiom/internal/stageexecution"
	"github.com/rgomids/axiom/internal/workflow"
	"github.com/rgomids/axiom/internal/workflowcompiler"
)

func TestBuildSuccessorAndReceiptWithoutPublication(t *testing.T) {
	wire, e := os.ReadFile("../workflowcompiler/testdata/issue306/stage-plan-v2.json")
	if e != nil {
		t.Fatal(e)
	}
	var plan workflowcompiler.StagePlan
	if e = json.Unmarshal(wire, &plan); e != nil {
		t.Fatal(e)
	}
	req := executiongraph.PublicationRequest{Proposal: *plan.Compilation.Proposal, ExpectedDigest: plan.Compilation.Proposal.Digest, GraphRevision: plan.Execution.Revision, ParentAuthority: plan.Compilation.AuthorityCeiling, ChildAuthorities: map[string][]executiongraph.Effect{}, AuthorityReferences: map[string]string{}, Workspaces: map[string]string{}, Resolutions: map[string]executiongraph.Resolution{}}
	for _, agent := range plan.Compilation.Agents {
		key := agent.Input.NodeKey
		f := agent.Input.ResolvedProfile
		req.ChildAuthorities[key] = agent.Effects
		req.AuthorityReferences[key] = "pending-dispatch-authority"
		req.Workspaces[key] = "/synthetic/workspace/" + agent.AgentID
		req.Resolutions[key] = executiongraph.Resolution{RuntimeID: f.RuntimeID, ModelProfileID: f.ProfileID, ConfigurationRevision: f.ConfigurationRevision, ObservationRevision: f.ObservationRevision, ResolvedProfile: f}
	}
	preview := stageexecution.DispatchPreview{Plan: plan, Publication: &req, Digest: strings.Repeat("a", 64)}
	// No stores or runtime ports are configured: Build must remain pure.
	next := 0
	p := LocalPublisher{AllocateID: func() (string, error) { next++; return fmt.Sprintf("00000000-0000-4000-8000-%012d", next), nil }}
	publication, e := p.Build(context.Background(), preview, "operator", time.Unix(20, 0).UTC())
	if e != nil {
		t.Fatal(e)
	}
	if publication.Graph == nil || publication.Single != nil || publication.Graph.FormatVersion != 3 || publication.Graph.StageConcurrency != uint32(plan.Compilation.Concurrency) || publication.Authority.Scope != "graph-first-attempts" || executiongraph.ValidateGraphDispatchAuthority(*publication.Graph, publication.Authority, preview.Digest) != nil {
		t.Fatal("invalid candidate", publication)
	}
	for _, c := range publication.Graph.Children {
		if len(c.Attempts) != 0 || c.Envelope.AuthorityReference != publication.Graph.DispatchAuthorityRef {
			t.Fatal("candidate started an attempt")
		}
	}
}
func TestBuildSingleBindsExecution3AndPreviewWithoutGraph(t *testing.T) {
	wire, e := os.ReadFile("../workflowcompiler/testdata/issue306/stage-plan-v2.json")
	if e != nil {
		t.Fatal(e)
	}
	var plan workflowcompiler.StagePlan
	if e = json.Unmarshal(wire, &plan); e != nil {
		t.Fatal(e)
	}
	agent := plan.Compilation.Agents[0]
	plan.Execution.ExecutionID = "00000000-0000-4000-8000-000000000001"
	plan.ExecutionKind, plan.Compilation.ExecutionKind, plan.Compilation.Proposal, plan.Compilation.Agents = "single", "single", nil, plan.Compilation.Agents[:1]
	state := workflow.State{ExecutionID: plan.Execution.ExecutionID, Revision: plan.Execution.Revision, ProjectID: plan.ProjectID, RepositoryKey: agent.Input.RepositoryKey, Stage: workflow.Stage(plan.StageID), StageOrdinal: 2, WorkItem: workflow.WorkItem{Provider: "github", Resource: "owner/repo", ExternalID: "7"}, StorageRevision: [32]byte{1}}
	child := stageexecution.ChildRights{AgentID: agent.AgentID, Workspace: "/synthetic/worktrees/a", Inbox: "/synthetic/inboxes/a", Effects: agent.Effects, Controls: agent.Controls}
	preview := stageexecution.DispatchPreview{Plan: plan, AxiomDigest: strings.Repeat("e", 64), Children: []stageexecution.ChildRights{child}, Digest: strings.Repeat("a", 64)}
	at := time.Unix(20, 0).UTC()
	p, e := BuildSingle(preview, state, "operator", at)
	if e != nil {
		t.Fatal(e)
	}
	if p.Run.Scope.ExecutionID != state.ExecutionID || p.Run.Scope.WorkItem != "github:owner/repo#7" || p.Run.Binding.ExecutionGeneration != fmt.Sprintf("%x", state.StorageRevision) || p.Run.Binding.PreviewDigest != preview.Digest || len(p.Run.Attempts) != 0 || p.Authority.AttemptNumber != 1 || p.Authority.Actor != "operator" {
		t.Fatalf("single publication = %+v", p)
	}
	if n, e := executiongraph.CheckSingleStageDispatch(p.Run, p.Authority); e != nil || n != 1 {
		t.Fatal(n, e)
	}
	stale := state
	stale.Revision++
	if _, e := BuildSingle(preview, stale, "operator", at); e != stageexecution.ErrStalePreview {
		t.Fatalf("stale Execution3 = %v", e)
	}
	unread := state
	unread.StorageRevision = [32]byte{}
	if _, e := BuildSingle(preview, unread, "operator", at); e != stageexecution.ErrPreview {
		t.Fatalf("unread Execution3 generation = %v", e)
	}
	graph := preview
	graph.Publication = &executiongraph.PublicationRequest{}
	if _, e := BuildSingle(graph, state, "operator", at); e != stageexecution.ErrPreview {
		t.Fatal("graph preview built as single")
	}
}
