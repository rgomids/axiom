package executiongraph

import (
	"fmt"
	"testing"
	"time"
)

func mustProposal(t *testing.T) Proposal {
	t.Helper()
	return historicalProposalFixture(t)
}

func mustGraph(t *testing.T) Graph {
	t.Helper()
	p := mustProposal(t)
	workspaces := map[string]string{}
	resolutions := map[string]Resolution{}
	for _, n := range p.Nodes {
		workspaces[n.Key] = "/tmp/axiom-s8/" + n.Key
		resolutions[n.Key] = Resolution{RuntimeID: "codex", ModelProfileID: "profile-1", ConfigurationRevision: 7, ObservationRevision: 7}
	}
	return fixtureGraph(t, p, 3, workspaces, resolutions)
}

// fixtureGraph constructs inert integration/codec test data. It has no writer,
// admission, scheduler or process path.
func fixtureGraph(t *testing.T, p Proposal, revision uint64, workspaces map[string]string, resolutions map[string]Resolution) Graph {
	t.Helper()
	ids := map[string]string{}
	for i, n := range p.Nodes {
		ids[n.Key] = fmt.Sprintf("00000000-0000-4000-8000-%012d", i+2)
	}
	effects := []Effect{}
	for _, n := range p.Nodes {
		effects = append(effects, n.Effects...)
	}
	authority, e := effectsDigest(effects)
	if e != nil {
		t.Fatal(e)
	}
	g := Graph{FormatVersion: p.FormatVersion, Parent: ParentExecution{ExecutionID: "00000000-0000-4000-8000-000000000001", GraphRevision: revision, ProposalDigest: p.Digest, AuthorityDigest: authority, AuthorityEffects: sortedEffects(effects), CreatedAt: time.Unix(10, 0).UTC()}}
	for _, n := range p.Nodes {
		deps := []string{}
		for _, key := range n.Dependencies {
			deps = append(deps, ids[key])
		}
		g.Children = append(g.Children, ChildExecution{ExecutionID: ids[n.Key], ParentID: g.Parent.ExecutionID, GraphRevision: revision, NodeKey: n.Key, Envelope: ChildEnvelope{Capability: n.Capability, Inputs: n.Inputs, Outputs: n.Outputs, Scope: n.Scope, Workspace: workspaces[n.Key], Dependencies: deps, AllowedEffects: sortedEffects(n.Effects), AuthorityReference: "authority:" + n.Key, Controls: n.Controls, ValidationOwner: n.ValidationOwner, IntegrationOwner: n.IntegrationOwner, Optional: n.Optional, Resolution: resolutions[n.Key]}})
		if n.IntegrationOwner {
			g.Parent.IntegrationChild = ids[n.Key]
		}
	}
	if !ValidGraph(g) {
		t.Fatal("invalid inert graph fixture")
	}
	return g
}

func TestHistoricalGraphCodecRoundTrip(t *testing.T) {
	g := mustGraph(t)
	wire, e := EncodeGraph(g)
	if e != nil {
		t.Fatal(e)
	}
	decoded, e := DecodeGraph(wire)
	if e != nil || decoded.Parent.ExecutionID != g.Parent.ExecutionID {
		t.Fatalf("roundtrip: %v", e)
	}
}
