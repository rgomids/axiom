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

// policyInvocations checks the exact reviewed selection and its concrete
// binding before the adapter may resolve credentials. It never substitutes
// another runtime, model, credential reference or executable.
type policyInvocations struct {
	policy      *runtimeapplication.Service
	previews    map[string]runtimeapplication.Preview
	profiles    map[string]runtimeadapter.CommandProfile
	invocations executiongraph.InvocationResolver
}

func newPolicyInvocations(cfg LocalConfiguration, invocations executiongraph.InvocationResolver) (policyInvocations, error) {
	guard := policyInvocations{policy: cfg.RuntimePolicy, previews: make(map[string]runtimeapplication.Preview), profiles: make(map[string]runtimeadapter.CommandProfile), invocations: invocations}
	for _, profile := range cfg.RuntimeProfiles {
		if !selectionSafeArguments(profile.RuntimeID, profile.Arguments) || !selectionSafeEnvironment(profile.Environment) {
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
	binding, err := g.policy.Check(ctx, preview)
	if err != nil {
		return executiongraph.Invocation{}, err
	}
	checked := preview
	checked.Choice = &binding.Choice
	if !matchesChild(checked, child) {
		return executiongraph.Invocation{}, ErrInvalidComposition
	}
	if err := g.matchesBinding(binding); err != nil {
		return executiongraph.Invocation{}, err
	}
	return g.invocations.ResolveInvocation(ctx, child)
}

func (g policyInvocations) matchesProfile(choice runtimeprofile.Choice) bool {
	profile, ok := g.profiles[choice.RuntimeID+"\x00"+choice.ModelProfileID]
	return ok && profile.RuntimeID == choice.Adapter && profile.Model == choice.Model && choice.ExecutableDigest != ""
}

// matchesBinding requires the command profile to be the reviewed concrete
// binding: same Runtime, profile, model and credential reference, and an
// executable whose current identity is the one observed for the preview.
func (g policyInvocations) matchesBinding(binding runtimeapplication.Binding) error {
	choice := binding.Choice
	if !g.matchesProfile(choice) {
		return ErrInvalidComposition
	}
	profile := g.profiles[choice.RuntimeID+"\x00"+choice.ModelProfileID]
	if profile.CredentialReference != binding.CredentialReference {
		return ErrInvalidComposition
	}
	identity, err := runtimeadapter.ExecutableIdentity(profile.Executable)
	if err != nil || identity != choice.ExecutableDigest {
		return runtimeapplication.ErrStale
	}
	return nil
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

// Reject environment that selects another credential, provider, model or
// Runtime configuration root; credentials arrive only through the reviewed
// reference. Ambient logins under an explicitly configured HOME remain the
// Runtime's own and are outside this binding.
func selectionSafeEnvironment(environment []string) bool {
	for _, entry := range environment {
		key, _, _ := strings.Cut(entry, "=")
		key = strings.ToUpper(key)
		for _, fragment := range []string{"API_KEY", "APIKEY", "TOKEN", "SECRET", "PASSWORD", "CREDENTIAL", "AUTH"} {
			if strings.Contains(key, fragment) {
				return false
			}
		}
		for _, prefix := range []string{"ANTHROPIC_", "OPENAI_", "CODEX_", "CLAUDE_", "AWS_", "AZURE_", "GOOGLE_", "GCLOUD_", "VERTEX", "BEDROCK"} {
			if strings.HasPrefix(key, prefix) {
				return false
			}
		}
	}
	return true
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
