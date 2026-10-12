package graphapplication

import (
	"context"
	"errors"
	"testing"

	"github.com/rgomids/axiom/internal/executiongraph"
	"github.com/rgomids/axiom/internal/runtimeadapter"
	"github.com/rgomids/axiom/internal/runtimeapplication"
	"github.com/rgomids/axiom/internal/runtimeprofile"
)

// Deterministic consumption of an injected safe source proof; no vendor inspection.
type injectedEffortSources struct{}

func (injectedEffortSources) CheckEffortSources(_ context.Context, p runtimeprofile.ResolvedProfile, _ executiongraph.Invocation) (string, error) {
	return p.EffortSourceDigest, nil
}
func currentPolicyFixture(t *testing.T, graph *executiongraph.Graph, runtimeID, executable, credential string) (*runtimeapplication.Service, map[string]runtimeapplication.Preview, *policySource) {
	t.Helper()
	_, _, source := policyFixture(t, graph, runtimeID, executable, credential)
	source.snapshot.Configuration.FormatVersion = 2
	for i := range source.snapshot.Configuration.Runtimes {
		source.snapshot.Configuration.Runtimes[i].CredentialReference = ""
	}
	for i := range source.snapshot.Configuration.ModelProfiles {
		source.snapshot.Configuration.ModelProfiles[i].Effort = runtimeprofile.Effort{Mode: "runtime-default", Value: "default"}
	}
	observation, err := source.snapshot.Observer.Observe(context.Background(), runtimeID)
	if err != nil {
		t.Fatal(err)
	}
	observation.ProfileProofs = []runtimeprofile.ProfileProof{{RuntimeID: runtimeID, Model: "local-test-profile", Effort: runtimeprofile.Effort{Mode: "runtime-default", Value: "default"}, ExecutableDigest: observation.ExecutableDigest, RuntimeVersion: observation.Version, InvocationMode: "noninteractive", ObservationRevision: observation.Revision, Reference: "injected-test-proof", EffortSourceDigest: digest("effort-sources"), Authentication: runtimeprofile.CapabilityProven, ModelAvailability: runtimeprofile.CapabilityProven, EffortSupport: runtimeprofile.CapabilityProven, NoOverride: runtimeprofile.CapabilityProven, CapabilityStatus: observation.CapabilityStatus}}
	inventory, err := runtimeadapter.NewInventory([]runtimeprofile.Observation{observation})
	if err != nil {
		t.Fatal(err)
	}
	source.snapshot.Observer = inventory
	policy := runtimeapplication.New(source)
	previews := map[string]runtimeapplication.Preview{}
	graph.FormatVersion = 2
	for i := range graph.Children {
		child := &graph.Children[i]
		capability := child.Envelope.Capability
		preview, err := policy.Preview(context.Background(), child.Envelope.Scope.ProjectID, runtimeapplication.Request{Role: capability.Role, Complexity: capability.Complexity, Capabilities: capability.Capabilities, RuntimeID: runtimeID})
		if err != nil || preview.Choice == nil || preview.Choice.ResolvedProfile == nil {
			t.Fatalf("current preview: %+v err=%v", preview, err)
		}
		p := *preview.Choice.ResolvedProfile
		child.Envelope.Resolution.ResolvedProfile = &p
		previews[child.ExecutionID] = preview
	}
	if !executiongraph.ValidGraph(*graph) {
		t.Fatal("invalid injected current graph")
	}
	return &policy, previews, source
}
func buildCurrentGraph(t *testing.T, root string) executiongraph.Graph {
	t.Helper()
	graph := buildGraph(t, root)
	graph.FormatVersion = 2
	for i := range graph.Children {
		child := &graph.Children[i]
		r := child.Envelope.Resolution
		d := digest("synthetic-current-binding")
		child.Envelope.Resolution.ResolvedProfile = &runtimeprofile.ResolvedProfile{ProfileID: r.ModelProfileID, RuntimeID: r.RuntimeID, Model: "local-test-profile", Effort: runtimeprofile.Effort{Mode: "runtime-default", Value: "default"}, ProfileDigest: d, ConfigurationRevision: r.ConfigurationRevision, ConfigurationDigest: d, ExecutableDigest: d, RuntimeVersion: "synthetic-1", InvocationMode: "noninteractive", ObservationRevision: r.ObservationRevision, ObservationDigest: d, EffortSourceDigest: d, CapabilityProofReference: "injected-proof"}
	}
	if !executiongraph.ValidGraph(graph) {
		t.Fatal("invalid test graph2")
	}
	return graph
}
func TestCurrentCompositionRefusesLegacyGraphBeforeEffects(t *testing.T) {
	legacy := buildGraph(t, canonicalTempDir(t))
	wire := mustEncodeGraph(t, legacy)
	_, err := NewLocalService(context.Background(), LocalConfiguration{Graph: legacy})
	if !errors.Is(err, ErrLegacyExecutionContract) {
		t.Fatal(err)
	}
	service := LocalService{graph: legacy}
	_, err = service.DispatchReady(context.Background(), nil, false)
	if !errors.Is(err, ErrLegacyExecutionContract) || string(wire) != string(mustEncodeGraph(t, service.Graph())) {
		t.Fatal("legacy dispatch altered graph", err)
	}
	if _, err := executiongraph.DecodeGraph(wire); err != nil {
		t.Fatal("legacy inspection blocked", err)
	}
}
