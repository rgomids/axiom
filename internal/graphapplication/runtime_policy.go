package graphapplication

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"
	"sync"

	"github.com/rgomids/axiom/internal/executiongraph"
	"github.com/rgomids/axiom/internal/runtimeadapter"
	"github.com/rgomids/axiom/internal/runtimeapplication"
	"github.com/rgomids/axiom/internal/runtimeprofile"
)

// policyInvocations checks the exact reviewed selection and its concrete
// binding before the adapter may resolve credentials. It never substitutes
// another runtime, model, credential reference or executable.
type policyInvocations struct {
	policy         *runtimeapplication.Service
	previews       map[string]runtimeapplication.Preview
	profiles       map[string]runtimeadapter.CommandProfile
	invocations    executiongraph.InvocationResolver
	auth           SubscriptionAuthenticator
	authentication *authenticationLedger
}

// SubscriptionAuthenticator is the machine-local CLI subscription
// authentication preflight (runtimeadapter.AuthPreflight).
type SubscriptionAuthenticator interface {
	Check(context.Context, runtimeadapter.AuthTarget) runtimeadapter.AuthReport
}

// authenticationLedger keeps the latest sanitized report per child.
type authenticationLedger struct {
	mu     sync.Mutex
	latest map[string]runtimeadapter.AuthReport
}

func (l *authenticationLedger) record(childID string, report runtimeadapter.AuthReport) {
	l.mu.Lock()
	defer l.mu.Unlock()
	report.Overrides = slices.Clone(report.Overrides)
	l.latest[childID] = report
}

func (l *authenticationLedger) reports() []runtimeadapter.AuthReport {
	l.mu.Lock()
	defer l.mu.Unlock()
	ids := slices.Sorted(maps.Keys(l.latest))
	result := make([]runtimeadapter.AuthReport, 0, len(ids))
	for _, id := range ids {
		report := l.latest[id]
		report.Overrides = slices.Clone(report.Overrides)
		result = append(result, report)
	}
	return result
}

func newPolicyInvocations(cfg LocalConfiguration, invocations executiongraph.InvocationResolver) (policyInvocations, error) {
	guard := policyInvocations{policy: cfg.RuntimePolicy, previews: make(map[string]runtimeapplication.Preview), profiles: make(map[string]runtimeadapter.CommandProfile), invocations: invocations, auth: cfg.SubscriptionAuth}
	if guard.auth != nil {
		guard.authentication = &authenticationLedger{latest: map[string]runtimeadapter.AuthReport{}}
	}
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
	if g.auth == nil {
		return g.invocations.ResolveInvocation(ctx, child)
	}
	return g.subscriptionInvocation(ctx, child, binding)
}

// subscriptionInvocation enforces the subscription scenario at the dispatch
// boundary. A credential reference is refused before anything is resolved;
// otherwise the preflight inspects the exact invocation the process runner
// will start, including the inherited environment, and binds it to the
// reviewed executable identity. Each attempt re-runs it, so no earlier
// observation authorizes a later dispatch.
func (g policyInvocations) subscriptionInvocation(ctx context.Context, child executiongraph.ChildExecution, binding runtimeapplication.Binding) (executiongraph.Invocation, error) {
	report := runtimeadapter.AuthReport{RuntimeID: binding.Choice.RuntimeID, Status: runtimeadapter.AuthIncompatible, Reason: "credential_reference_configured", Method: "unknown", Version: "unknown", EvidenceKind: runtimeadapter.EvidenceLocalObservation, Usability: runtimeadapter.UsabilityUnproven, Revalidation: runtimeadapter.RevalidateBeforeDispatch}
	if binding.CredentialReference != "" {
		g.authentication.record(child.ExecutionID, report)
		return executiongraph.Invocation{}, fmt.Errorf("%w: %s", executiongraph.ErrAuthenticationBlocked, report.Reason)
	}
	invocation, err := g.invocations.ResolveInvocation(ctx, child)
	if err != nil {
		return executiongraph.Invocation{}, err
	}
	// The effective argv is re-checked, not only the configured profile.
	if len(invocation.Argv) < 4 || invocation.RuntimeID != binding.Choice.RuntimeID || binding.Choice.ExecutableDigest == "" || !selectionSafeArguments(invocation.RuntimeID, invocation.Argv[4:]) {
		return executiongraph.Invocation{}, ErrInvalidComposition
	}
	report = g.auth.Check(ctx, runtimeadapter.AuthTarget{RuntimeID: invocation.RuntimeID, Executable: invocation.Argv[0], ExpectedDigest: binding.Choice.ExecutableDigest, WorkingDirectory: invocation.CWD, Environment: executiongraph.EffectiveEnvironment(invocation.Env)})
	g.authentication.record(child.ExecutionID, report)
	if report.Reason == "executable_identity_changed" {
		return executiongraph.Invocation{}, fmt.Errorf("%w: %w", executiongraph.ErrAuthenticationBlocked, runtimeapplication.ErrStale)
	}
	if !report.DispatchAllowed() {
		return executiongraph.Invocation{}, fmt.Errorf("%w: %s", executiongraph.ErrAuthenticationBlocked, report.Reason)
	}
	return invocation, nil
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
