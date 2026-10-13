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
func TestBuilderRefusesMissingSingleExecutor(t *testing.T) {
	p := LocalPublisher{AllocateID: func() (string, error) { t.Fatal("allocated before checking single seam"); return "", nil }}
	if _, e := p.Build(context.Background(), stageexecution.DispatchPreview{}, "operator", time.Unix(20, 0).UTC()); e != stageexecution.ErrPreview {
		t.Fatal(e)
	}
}
