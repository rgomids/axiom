package workflow

import "testing"

func TestMissingLocalStateNeverSynthesizesExecution(t *testing.T) {
	provider := ProjectionObservation{Provider: "github", Resource: "owner/repo", IssueExternalID: "94", IssueURL: "https://github.com/owner/repo/issues/94", IssueState: "OPEN", IssueLabels: []string{"axiom:stage:reviewed"}}
	result := InspectReconciliation(ReconciliationInput{Provider: &provider, RepositoryArtifacts: lifecycleReferences()})
	if result.Kind != ReconciliationRequired || !result.NeedsHuman || result.Recovery != nil {
		t.Fatalf("provider-only result = %#v", result)
	}
}

func TestValidatedLocalGenerationProducesExactPlanAndFreshAuthority(t *testing.T) {
	state := lifecycleTestState()
	digest := GenerationDigest(state)
	result := InspectReconciliation(ReconciliationInput{Generations: []LocalGeneration{{Name: "prior-1", State: state, Digest: digest, Validated: true}}})
	if result.Kind != ReconciliationRecoveryPlan || result.Recovery == nil {
		t.Fatalf("recovery result = %#v", result)
	}
	if result.Authorizes(RecoveryAuthority{GenerationName: "prior-1", Digest: testDigest}) {
		t.Fatal("stale recovery authority accepted")
	}
	if !result.Authorizes(RecoveryAuthority{GenerationName: "prior-1", Digest: digest}) {
		t.Fatal("exact recovery authority rejected")
	}
}

func TestCurrentLocalTruthRequiresAlignedProviderProjection(t *testing.T) {
	state := lifecycleTestState()
	provider := ProjectionObservation{Provider: "github", Resource: "owner/repo", IssueExternalID: "94", IssueURL: "https://github.com/owner/repo/issues/94", IssueState: "OPEN", IssueLabels: []string{"axiom:stage:intake"}}
	result := InspectReconciliation(ReconciliationInput{Current: &state, Provider: &provider})
	if result.Kind != ReconciliationCurrentTruth || result.Lifecycle == nil || result.Lifecycle.Stage != LifecycleIntake {
		t.Fatalf("aligned result = %#v", result)
	}
	provider.IssueLabels = []string{"axiom:stage:accepted"}
	result = InspectReconciliation(ReconciliationInput{Current: &state, Provider: &provider})
	if result.Kind != ReconciliationRequired {
		t.Fatalf("drift result = %#v", result)
	}
}

func TestReconciliationContradictionsStayReadOnlyAndRequireHumanDecision(t *testing.T) {
	state := lifecycleTestState()
	before := digest(state)
	provider := ProjectionObservation{Provider: "github", Resource: "owner/repo", IssueExternalID: "94", IssueURL: "https://github.com/owner/repo/issues/94", IssueState: "OPEN", IssueLabels: []string{"axiom:stage:invented"}}
	for name, input := range map[string]ReconciliationInput{
		"repository only":   {RepositoryArtifacts: lifecycleReferences()},
		"changed artifacts": {Current: &state, ArtifactsChanged: true},
		"unknown label":     {Current: &state, Provider: &provider},
		"multiple generations": {
			Generations: []LocalGeneration{
				{Name: "prior", State: state, Digest: GenerationDigest(state), Validated: true},
				{Name: "staged", State: state, Digest: GenerationDigest(state), Validated: true},
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			result := InspectReconciliation(input)
			if result.Kind != ReconciliationRequired || !result.NeedsHuman || result.Recovery != nil {
				t.Fatalf("result = %#v", result)
			}
		})
	}
	if after := digest(state); after != before {
		t.Fatalf("inspection mutated state: before=%s after=%s", before, after)
	}
}
