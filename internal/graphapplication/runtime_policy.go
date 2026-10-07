package graphapplication

import (
	"context"
	"maps"
	"slices"
	"strings"

	"github.com/rgomids/axiom/internal/executiongraph"
	"github.com/rgomids/axiom/internal/runtimeadapter"
	"github.com/rgomids/axiom/internal/runtimeapplication"
	"github.com/rgomids/axiom/internal/runtimeprofile"
)

// policyInvocations checks the exact reviewed selection before the adapter may
// resolve credentials. It never substitutes another runtime or model.
type policyInvocations struct {
	policy      *runtimeapplication.Service
	previews    map[string]runtimeapplication.Preview
	profiles    map[string]runtimeadapter.CommandProfile
	invocations executiongraph.InvocationResolver
}

func newPolicyInvocations(cfg LocalConfiguration, invocations executiongraph.InvocationResolver) (policyInvocations, error) {
	guard := policyInvocations{policy: cfg.RuntimePolicy, previews: make(map[string]runtimeapplication.Preview), profiles: make(map[string]runtimeadapter.CommandProfile), invocations: invocations}
	for _, profile := range cfg.RuntimeProfiles {
		if !selectionSafeArguments(profile.RuntimeID, profile.Arguments) {
			return policyInvocations{}, ErrInvalidComposition
		}
		guard.profiles[profile.RuntimeID+"\x00"+profile.ModelProfileID] = profile
	}
	for _, child := range cfg.Graph.Children {
		preview, ok := cfg.RuntimePreviews[child.ExecutionID]
		if !ok || !matchesChild(preview, child) || !guard.matchesProfile(*preview.Choice) {
			return policyInvocations{}, ErrInvalidComposition
		}
		preview.Request.Capabilities = slices.Clone(preview.Request.Capabilities)
		preview.ObservationRevisions = maps.Clone(preview.ObservationRevisions)
		choice := *preview.Choice
		choice.Capabilities = slices.Clone(choice.Capabilities)
		preview.Choice = &choice
		guard.previews[child.ExecutionID] = preview
	}
	return guard, nil
}

func (g policyInvocations) ResolveInvocation(ctx context.Context, child executiongraph.ChildExecution) (executiongraph.Invocation, error) {
	preview, ok := g.previews[child.ExecutionID]
	if !ok || !matchesChild(preview, child) {
		return executiongraph.Invocation{}, ErrInvalidComposition
	}
	choice, err := g.policy.Check(ctx, preview)
	if err != nil {
		return executiongraph.Invocation{}, err
	}
	checked := preview
	checked.Choice = &choice
	if !matchesChild(checked, child) || !g.matchesProfile(choice) {
		return executiongraph.Invocation{}, ErrInvalidComposition
	}
	return g.invocations.ResolveInvocation(ctx, child)
}

func (g policyInvocations) matchesProfile(choice runtimeprofile.Choice) bool {
	profile, ok := g.profiles[choice.RuntimeID+"\x00"+choice.ModelProfileID]
	return ok && profile.Model == choice.Model
}

func matchesChild(preview runtimeapplication.Preview, child executiongraph.ChildExecution) bool {
	choice := preview.Choice
	capability := child.Envelope.Capability
	resolution := child.Envelope.Resolution
	return choice != nil && preview.Blocker == nil && preview.ProjectID == child.Envelope.Scope.ProjectID &&
		preview.Request.Role == capability.Role && preview.Request.Complexity == capability.Complexity &&
		sameCapabilities(preview.Request.Capabilities, capability.Capabilities) &&
		(preview.Request.RuntimeID == "" || preview.Request.RuntimeID == resolution.RuntimeID) &&
		choice.RuntimeID == resolution.RuntimeID && choice.ModelProfileID == resolution.ModelProfileID &&
		choice.ConfigurationRevision == resolution.ConfigurationRevision && choice.ObservationRevision == resolution.ObservationRevision
}

func sameCapabilities(a, b []string) bool {
	a, b = slices.Clone(a), slices.Clone(b)
	slices.Sort(a)
	slices.Sort(b)
	return slices.Equal(a, b)
}

// Keep ordinary adapter arguments while rejecting alternate selection paths.
func selectionSafeArguments(runtimeID string, arguments []string) bool {
	options := []string{"--model", "--config", "--settings", "--setting-sources", "--resume", "--continue"}
	shortOptions := "mc"
	if runtimeID == "codex" {
		options = append(options, "--profile", "--oss", "--local-provider")
		shortOptions += "p"
	}
	if runtimeID == "claude" {
		options = append(options, "--fallback-model", "--agent", "--agents", "--from-pr", "--teleport", "--fork-session")
		shortOptions += "r"
	}
	for _, argument := range arguments {
		if argument == "--" || argument == "resume" || argument == "fork" {
			return false
		}
		for _, option := range options {
			if argument == option || strings.HasPrefix(argument, option+"=") {
				return false
			}
		}
		if strings.HasPrefix(argument, "-") && !strings.HasPrefix(argument, "--") && strings.ContainsAny(strings.TrimPrefix(argument, "-"), shortOptions) {
			return false
		}
	}
	return runtimeID == "codex" || runtimeID == "claude"
}
