package executiongraph

import (
	"errors"
	"sort"
	"testing"
	"time"
)

// historicalProposalFixture constructs inert data for codecs/integration tests;
// no Plan, approval, capability admission or executable producer is involved.
func historicalProposalFixture(t *testing.T) Proposal {
	t.Helper()
	controls := ExecutionControls{Timeout: 10 * time.Minute, MaximumAttempts: 2}
	p := Proposal{FormatVersion: ProposalFormatVersion, PlanRevision: "plan-7", PlanDigest: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Nodes: []ProposedNode{
		{Key: "api-contract", Capability: CapabilityRequest{Role: "contract", Complexity: "medium", Capabilities: []string{"go"}}, Inputs: []string{"plan"}, Outputs: []string{"contract"}, Scope: Scope{ProjectID: "project-1", RepositoryKey: "main", Paths: []string{"internal/api"}}, Effects: []Effect{{Kind: "repository-write", Target: "internal/api"}}, Controls: controls},
		{Key: "docs", Capability: CapabilityRequest{Role: "documentation", Complexity: "low", Capabilities: []string{"documentation"}}, Inputs: []string{"plan"}, Outputs: []string{"docs"}, Scope: Scope{ProjectID: "project-1", RepositoryKey: "main", Paths: []string{"docs/runtime.md"}}, Effects: []Effect{{Kind: "repository-write", Target: "docs/runtime.md"}}, Controls: controls},
		{Key: "integrate", Capability: CapabilityRequest{Role: "integration", Complexity: "high", Capabilities: []string{"go", "validation"}}, Dependencies: []string{"api-contract", "docs"}, Inputs: []string{"contract", "docs"}, Outputs: []string{"delivery"}, Scope: Scope{ProjectID: "project-1", RepositoryKey: "main", Paths: []string{"internal/api", "docs/runtime.md"}}, Effects: []Effect{{Kind: "integration", Target: "internal/api"}, {Kind: "integration", Target: "docs/runtime.md"}}, ValidationOwner: true, IntegrationOwner: true, Controls: controls},
	}}
	normalizeProposal(&p)
	var err error
	p.Digest, err = proposalDigest(p)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestHistoricalProposalRejectsUnsafeTopology(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*Proposal)
		want   error
	}{
		{"cycle", func(p *Proposal) { p.Nodes[0].Dependencies = []string{"integrate"} }, ErrCycle},
		{"unknown dependency", func(p *Proposal) { p.Nodes[0].Dependencies = []string{"missing"} }, ErrInvalidGraph},
		{"orphan", func(p *Proposal) { p.Nodes[2].Dependencies = []string{"api-contract"} }, ErrInvalidGraph},
		{"injected key", func(p *Proposal) { p.Nodes[0].Key = "run; rm -rf" }, ErrInvalidGraph},
		{"overlap", func(p *Proposal) {
			p.Nodes[0].Effects = []Effect{{Kind: "repository-write", Target: "shared/file"}}
			p.Nodes[1].Effects = []Effect{{Kind: "repository-write", Target: "shared"}}
		}, ErrUnsafeOverlap},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := historicalProposalFixture(t)
			tc.mutate(&p)
			if err := validateProposalTopology(p, maxNodes); !errors.Is(err, tc.want) {
				t.Fatalf("got %v want %v", err, tc.want)
			}
		})
	}
}

// Normalization is fixture preparation only, never production admission.
func normalizeProposal(proposal *Proposal) {
	sort.Slice(proposal.Nodes, func(i, j int) bool { return proposal.Nodes[i].Key < proposal.Nodes[j].Key })
	for index := range proposal.Nodes {
		node := &proposal.Nodes[index]
		node.Capability.Capabilities = sortedUnique(node.Capability.Capabilities)
		node.Dependencies = sortedUnique(node.Dependencies)
		node.Inputs = sortedUnique(node.Inputs)
		node.Outputs = sortedUnique(node.Outputs)
		node.Scope.Paths = sortedUnique(node.Scope.Paths)
		sort.Slice(node.Effects, func(i, j int) bool {
			return node.Effects[i].Kind+"\x00"+node.Effects[i].Target < node.Effects[j].Kind+"\x00"+node.Effects[j].Target
		})
	}
}

func sortedUnique(values []string) []string {
	result := append([]string(nil), values...)
	sort.Strings(result)
	for index := 1; index < len(result); index++ {
		if result[index] == result[index-1] {
			return nil
		}
	}
	return result
}
