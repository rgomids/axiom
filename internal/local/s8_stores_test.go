package local

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/rgomids/axiom/internal/executiongraph"
	"github.com/rgomids/axiom/internal/runtimeprofile"
)

func TestRuntimeProfileStorePreservesRevisionAndCompletePriorGeneration(t *testing.T) {
	ctx := context.Background()
	root := filepath.Join(t.TempDir(), "state")
	store, err := NewRuntimeProfileStore(root)
	if err != nil {
		t.Fatal(err)
	}
	fresh, _ := store.Inspect(ctx)
	first, err := store.Publish(ctx, fresh.Token, authoringWire(t, 1))
	if err != nil {
		t.Fatal(err)
	}
	store.hooks.fault = func(stage FaultStage) error {
		if stage == FaultF2 {
			return ErrSimulatedInterruption
		}
		return nil
	}
	if _, err := store.Publish(ctx, first.Token, authoringWire(t, 2)); err == nil {
		t.Fatal("faulted update succeeded")
	}
	store.hooks = publicationHooks{}
	if _, err := store.Inspect(ctx); !errors.Is(err, ErrRecoveryRequired) {
		t.Fatalf("interrupted reader err=%v", err)
	}
	wire, err := os.ReadFile(filepath.Join(root, "runtime-profiles", "v1", "configuration.json"))
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := runtimeprofile.DecodeVersioned(wire)
	if err != nil || loaded.V4 == nil || loaded.V4.Revision != 1 {
		t.Fatalf("canonical=%+v err=%v", loaded, err)
	}
	root = filepath.Join(t.TempDir(), "state")
	store, err = NewRuntimeProfileStore(root)
	if err != nil {
		t.Fatalf("fresh store err=%v", err)
	}
	fresh, _ = store.Inspect(ctx)
	first, err = store.Publish(ctx, fresh.Token, authoringWire(t, 1))
	if err != nil {
		t.Fatalf("fresh create err=%v", err)
	}
	if _, err := store.Publish(ctx, first.Token, authoringWire(t, 3)); !errors.Is(err, ErrConflict) {
		t.Fatalf("stale revision err=%v", err)
	}
}

func runtimeConfiguration(revision uint64) runtimeprofile.Configuration {
	return runtimeprofile.Configuration{FormatVersion: 1, Revision: revision,
		Runtimes:      []runtimeprofile.Runtime{{ID: "codex", Adapter: "codex", Enabled: true, AllowlistedProfileIDs: []string{"codex-default"}}, {ID: "claude", Adapter: "claude", Enabled: true, AllowlistedProfileIDs: []string{"claude-default"}}},
		ModelProfiles: []runtimeprofile.ModelProfile{{ID: "codex-default", RuntimeID: "codex", Model: "local-codex", Capabilities: []string{"go"}}, {ID: "claude-default", RuntimeID: "claude", Model: "local-claude", Capabilities: []string{"go"}}},
	}
}

func localGraphFixture(t *testing.T) executiongraph.Graph {
	t.Helper()
	proposal := executiongraph.Proposal{FormatVersion: executiongraph.ProposalFormatVersion, PlanRevision: "plan-1", PlanDigest: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Nodes: []executiongraph.ProposedNode{
		{Key: "implementation", Capability: executiongraph.CapabilityRequest{Role: "implementation", Complexity: "low", Capabilities: []string{"go"}}, Inputs: []string{"plan"}, Outputs: []string{"result"}, Scope: executiongraph.Scope{ProjectID: "project-1", RepositoryKey: "main", Paths: []string{"internal/api"}}, Effects: []executiongraph.Effect{{Kind: "repository-write", Target: "internal/api"}}, Controls: executiongraph.ExecutionControls{Timeout: time.Minute, MaximumAttempts: 2}},
		{Key: "integration", Capability: executiongraph.CapabilityRequest{Role: "integration", Complexity: "high", Capabilities: []string{"validation"}}, Dependencies: []string{"implementation"}, Inputs: []string{"result"}, Outputs: []string{"delivery"}, Scope: executiongraph.Scope{ProjectID: "project-1", RepositoryKey: "main", Paths: []string{"internal/api"}}, Effects: []executiongraph.Effect{{Kind: "integration", Target: "internal/api"}}, ValidationOwner: true, IntegrationOwner: true, Controls: executiongraph.ExecutionControls{Timeout: time.Minute, MaximumAttempts: 2}},
	}}
	// This is inert fixture identity, with no operational Plan producer.
	wire, err := json.Marshal(proposal)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(wire)
	proposal.Digest = hex.EncodeToString(sum[:])
	workspaces := map[string]string{}
	resolutions := map[string]executiongraph.Resolution{}
	for _, node := range proposal.Nodes {
		workspaces[node.Key] = filepath.Join(t.TempDir(), node.Key)
		resolutions[node.Key] = executiongraph.Resolution{RuntimeID: "codex", ModelProfileID: "profile-1", ConfigurationRevision: 1, ObservationRevision: 1}
	}
	graph := fixtureGraph(t, proposal, 1, workspaces, resolutions)
	return graph
}

// fixtureGraph constructs inert integration/codec test data. It has no writer,
// admission, scheduler or process path.
func fixtureGraph(t *testing.T, p executiongraph.Proposal, revision uint64, workspaces map[string]string, resolutions map[string]executiongraph.Resolution) executiongraph.Graph {
	t.Helper()
	ids := map[string]string{}
	for i, n := range p.Nodes {
		ids[n.Key] = fmt.Sprintf("00000000-0000-4000-8000-%012d", i+2)
	}
	effects := []executiongraph.Effect{}
	for _, n := range p.Nodes {
		effects = append(effects, n.Effects...)
	}
	authority, e := effectsDigest(effects)
	if e != nil {
		t.Fatal(e)
	}
	g := executiongraph.Graph{FormatVersion: p.FormatVersion, Parent: executiongraph.ParentExecution{ExecutionID: "00000000-0000-4000-8000-000000000001", GraphRevision: revision, ProposalDigest: p.Digest, AuthorityDigest: authority, AuthorityEffects: sortedEffects(effects), CreatedAt: time.Unix(10, 0).UTC()}}
	for _, n := range p.Nodes {
		deps := []string{}
		for _, key := range n.Dependencies {
			deps = append(deps, ids[key])
		}
		g.Children = append(g.Children, executiongraph.ChildExecution{ExecutionID: ids[n.Key], ParentID: g.Parent.ExecutionID, GraphRevision: revision, NodeKey: n.Key, Envelope: executiongraph.ChildEnvelope{Capability: n.Capability, Inputs: n.Inputs, Outputs: n.Outputs, Scope: n.Scope, Workspace: workspaces[n.Key], Dependencies: deps, AllowedEffects: sortedEffects(n.Effects), AuthorityReference: "authority:" + n.Key, Controls: n.Controls, ValidationOwner: n.ValidationOwner, IntegrationOwner: n.IntegrationOwner, Optional: n.Optional, Resolution: resolutions[n.Key]}})
		if n.IntegrationOwner {
			g.Parent.IntegrationChild = ids[n.Key]
		}
	}
	if !executiongraph.ValidGraph(g) {
		t.Fatal("invalid inert graph fixture")
	}
	return g
}

func sortedEffects(effects []executiongraph.Effect) []executiongraph.Effect {
	result := append([]executiongraph.Effect(nil), effects...)
	sort.Slice(result, func(i, j int) bool {
		return result[i].Kind+"\x00"+result[i].Target < result[j].Kind+"\x00"+result[j].Target
	})
	return result
}
func effectsDigest(effects []executiongraph.Effect) (string, error) {
	wire, e := json.Marshal(sortedEffects(effects))
	if e != nil {
		return "", e
	}
	digest := sha256.Sum256(wire)
	return hex.EncodeToString(digest[:]), nil
}
