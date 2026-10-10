package graphapplication

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/rgomids/axiom/internal/executiongraph"
	"github.com/rgomids/axiom/internal/gitworkspace"
	"github.com/rgomids/axiom/internal/runtimeadapter"
	"github.com/rgomids/axiom/internal/runtimeapplication"
)

// recordingAuth is a mocked preflight: its reports are EvidenceFake.
type recordingAuth struct {
	targets []runtimeadapter.AuthTarget
	status  runtimeadapter.AuthStatus
	reason  string
}

func (r *recordingAuth) Check(_ context.Context, target runtimeadapter.AuthTarget) runtimeadapter.AuthReport {
	r.targets = append(r.targets, target)
	return runtimeadapter.AuthReport{RuntimeID: target.RuntimeID, Status: r.status, Reason: r.reason, Method: "unknown", Version: "unknown", EvidenceKind: runtimeadapter.EvidenceFake, Usability: runtimeadapter.UsabilityUnproven, Revalidation: runtimeadapter.RevalidateBeforeDispatch}
}

// scriptedStatus drives the real AuthPreflight with fixed vendor output.
type scriptedStatus struct{ calls int }

func (*scriptedStatus) EvidenceKind() string { return runtimeadapter.EvidenceFake }
func (s *scriptedStatus) RunStatus(_ context.Context, command runtimeadapter.StatusCommand) runtimeadapter.StatusResult {
	s.calls++
	if command.Argv[1] == "--version" {
		return runtimeadapter.StatusResult{Started: true, Output: []byte("codex-cli 0.159.1")}
	}
	return runtimeadapter.StatusResult{Started: true, Output: []byte("Logged in using ChatGPT")}
}

func subscriptionGuard(t *testing.T, runtimeID, credential string, environment []string, auth SubscriptionAuthenticator, effort ...string) (policyInvocations, executiongraph.Graph, *countedCredentials, string) {
	t.Helper()
	graph := buildGraph(t, canonicalTempDir(t))
	if len(effort) != 0 {
		for i := range graph.Children {
			graph.Children[i].Envelope.Controls.ReasoningEffort = effort[0]
			graph.Children[i].Envelope.Capability.Capabilities = append(graph.Children[i].Envelope.Capability.Capabilities, "reasoning-effort-"+effort[0])
		}
	}
	executable, _ := stubRuntime(t, t.TempDir(), runtimeID)
	policy, previews, _ := policyFixture(t, &graph, runtimeID, executable, credential)
	credentials := &countedCredentials{}
	profiles := []runtimeadapter.CommandProfile{}
	for _, child := range graph.Children {
		profiles = append(profiles, runtimeadapter.CommandProfile{RuntimeID: runtimeID, ModelProfileID: child.Envelope.Resolution.ModelProfileID, Executable: executable, Model: "local-test-profile", Environment: environment, CredentialReference: credential, OutputMax: 4096})
	}
	adapter, err := runtimeadapter.NewInvocationResolver(profiles, credentials)
	if err != nil {
		t.Fatal(err)
	}
	guard, err := newPolicyInvocations(LocalConfiguration{Graph: graph, RuntimePolicy: policy, RuntimePreviews: previews, RuntimeProfiles: profiles, SubscriptionAuth: auth}, adapter)
	if err != nil {
		t.Fatal(err)
	}
	return guard, graph, credentials, previews[graph.Children[0].ExecutionID].Choice.ExecutableDigest
}

func TestSubscriptionDispatchRevalidatesTheEffectiveInvocation(t *testing.T) {
	for _, runtimeID := range []string{"codex", "claude"} {
		t.Run(runtimeID, func(t *testing.T) {
			auth := &recordingAuth{status: runtimeadapter.AuthSubscriptionObserved}
			guard, graph, _, digest := subscriptionGuard(t, runtimeID, "", []string{"HOME=/synthetic/home", "PATH=/usr/bin"}, auth)
			invocation, err := guard.ResolveInvocation(context.Background(), graph.Children[0])
			if err != nil {
				t.Fatal(err)
			}
			if _, err := guard.ResolveInvocation(context.Background(), graph.Children[0]); err != nil || len(auth.targets) != 2 {
				t.Fatalf("each dispatch must revalidate: calls=%d err=%v", len(auth.targets), err)
			}
			target := auth.targets[0]
			if target.RuntimeID != runtimeID || target.Executable != invocation.Argv[0] || target.WorkingDirectory != invocation.CWD || target.ExpectedDigest != digest || !slices.Equal(target.Environment, executiongraph.EffectiveInvocationEnvironment(invocation)) {
				t.Fatalf("target=%+v invocation=%+v", target, invocation)
			}
			evidence := guard.authentication.reports()
			if len(evidence) != 1 || evidence[0].Status != runtimeadapter.AuthSubscriptionObserved || evidence[0].EvidenceKind != runtimeadapter.EvidenceFake {
				t.Fatalf("evidence=%+v", evidence)
			}
		})
	}
}

func TestSubscriptionDispatchFailsClosed(t *testing.T) {
	for _, status := range []runtimeadapter.AuthStatus{runtimeadapter.AuthUnproven, runtimeadapter.AuthIncompatible, runtimeadapter.AuthUnavailable, runtimeadapter.AuthUnsupported} {
		auth := &recordingAuth{status: status, reason: "synthetic_reason"}
		guard, graph, _, _ := subscriptionGuard(t, "claude", "", []string{"HOME=/synthetic/home"}, auth)
		_, err := guard.ResolveInvocation(context.Background(), graph.Children[0])
		if !errors.Is(err, executiongraph.ErrAuthenticationBlocked) || !strings.Contains(err.Error(), "synthetic_reason") {
			t.Fatalf("%s err=%v", status, err)
		}
	}
	auth := &recordingAuth{status: runtimeadapter.AuthUnproven, reason: "executable_identity_changed"}
	guard, graph, _, _ := subscriptionGuard(t, "codex", "", []string{"HOME=/synthetic/home"}, auth)
	if _, err := guard.ResolveInvocation(context.Background(), graph.Children[0]); !errors.Is(err, executiongraph.ErrAuthenticationBlocked) || !errors.Is(err, runtimeapplication.ErrStale) {
		t.Fatalf("stale identity err=%v", err)
	}
}

// A reviewed credential reference is the API path: the subscription scenario
// refuses it before any credential is resolved or any status command runs.
func TestSubscriptionDispatchRefusesCredentialReferenceBeforeResolution(t *testing.T) {
	auth := &recordingAuth{status: runtimeadapter.AuthSubscriptionObserved}
	guard, graph, credentials, _ := subscriptionGuard(t, "codex", "synthetic:credential", nil, auth)
	_, err := guard.ResolveInvocation(context.Background(), graph.Children[0])
	if !errors.Is(err, executiongraph.ErrAuthenticationBlocked) || credentials.calls != 0 || len(auth.targets) != 0 {
		t.Fatalf("err=%v credentials=%d preflight=%d", err, credentials.calls, len(auth.targets))
	}
	evidence := guard.authentication.reports()
	if len(evidence) != 1 || evidence[0].Reason != "credential_reference_configured" || strings.Contains(strings.Join(evidence[0].Overrides, ","), "synthetic") {
		t.Fatalf("evidence=%+v", evidence)
	}
}

// A profile without explicit environment makes the child inherit Axiom's
// environment; an inherited API key there blocks dispatch.
func TestSubscriptionDispatchRejectsInheritedEnvironment(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("CODEX_API_KEY", "sk-inherited-secret")
	status := &scriptedStatus{}
	preflight, err := runtimeadapter.NewAuthPreflight(status, runtimeadapter.ManagedConfiguration{})
	if err != nil {
		t.Fatal(err)
	}
	guard, graph, _, _ := subscriptionGuard(t, "codex", "", nil, preflight)
	_, err = guard.ResolveInvocation(context.Background(), graph.Children[0])
	evidence := guard.authentication.reports()
	if !errors.Is(err, executiongraph.ErrAuthenticationBlocked) || status.calls != 0 || len(evidence) != 1 || evidence[0].Reason != "environment_override" || strings.Join(evidence[0].Overrides, ",") != "env:CODEX_API_KEY" {
		t.Fatalf("err=%v status=%d evidence=%+v", err, status.calls, evidence)
	}
	if strings.Contains(err.Error(), "sk-inherited-secret") {
		t.Fatal("error leaked the credential value")
	}
	t.Setenv("CODEX_API_KEY", "")
	if _, err := guard.ResolveInvocation(context.Background(), graph.Children[0]); !errors.Is(err, executiongraph.ErrAuthenticationBlocked) {
		t.Fatalf("an empty override variable is still an override: %v", err)
	}
}

// Without SubscriptionAuth the existing reviewed-binding behavior is unchanged.
func TestNoSubscriptionScenarioKeepsExistingBinding(t *testing.T) {
	guard, graph, credentials, _ := subscriptionGuard(t, "codex", "synthetic:credential", nil, nil)
	if _, err := guard.ResolveInvocation(context.Background(), graph.Children[0]); err != nil || credentials.calls != 1 || guard.authentication != nil {
		t.Fatalf("err=%v credentials=%d", err, credentials.calls)
	}
}

// End to end through LocalService: an unavailable login blocks every child at
// the dispatch boundary with no attempt, persisted change or process start,
// and the sanitized report is available as Evidence.
func TestProductionDispatchBlockedByAuthenticationHasNoAttempts(t *testing.T) {
	root := canonicalTempDir(t)
	repository, revision := initRepository(t, filepath.Join(root, "repository"))
	graph := buildGraph(t, filepath.Join(root, "workspaces"))
	executable, marker := stubRuntime(t, filepath.Join(root, "bin"), "claude")
	policy, previews, _ := policyFixture(t, &graph, "claude", executable, "")
	profiles := []runtimeadapter.CommandProfile{}
	for _, child := range graph.Children {
		profiles = append(profiles, runtimeadapter.CommandProfile{RuntimeID: "claude", ModelProfileID: child.Envelope.Resolution.ModelProfileID, Executable: executable, Model: "local-test-profile", Environment: []string{"HOME=" + root}, OutputMax: 4096})
	}
	gitPath, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	store := &memoryGraphStore{wire: mustEncodeGraph(t, graph)}
	before := string(store.wire)
	auth := &recordingAuth{status: runtimeadapter.AuthUnavailable, reason: "not_logged_in"}
	service, err := NewLocalService(context.Background(), LocalConfiguration{
		Repository: repository, WorkspaceRoot: filepath.Join(root, "workspaces"), BaseRevision: revision,
		Graph: graph, GraphStore: store, RuntimeProfiles: profiles, RuntimePolicy: policy, RuntimePreviews: previews, SubscriptionAuth: auth,
		Validators:        []gitworkspace.ValidationCommand{{Reference: "diff-check", Argv: []string{gitPath, "diff", "--check"}, OutputMax: 4096}},
		AllocateAttemptID: func() (string, error) { return "attempt-1", nil },
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.PrepareWorkspaces(context.Background()); err != nil {
		t.Fatal(err)
	}
	result, err := service.DispatchReady(context.Background(), nil, false)
	if err != nil || len(result.Records) != 0 || string(store.wire) != before {
		t.Fatalf("records=%v persistedChanged=%t err=%v", result.Records, string(store.wire) != before, err)
	}
	blocked := 0
	for _, category := range result.Blocked {
		if category == "authentication_blocked" {
			blocked++
		}
	}
	evidence := service.AuthenticationEvidence()
	if blocked == 0 || len(evidence) != blocked || evidence[0].Reason != "not_logged_in" {
		t.Fatalf("blocked=%v evidence=%+v", result.Blocked, evidence)
	}
	if _, err := os.Lstat(marker); !os.IsNotExist(err) {
		t.Fatal("blocked authentication started the Runtime process")
	}
}

func TestSubscriptionDispatchWithValidatedEffort(t *testing.T) {
	for _, runtimeID := range []string{"codex", "claude"} {
		t.Run(runtimeID, func(t *testing.T) {
			t.Setenv("HOME", t.TempDir())
			t.Setenv("USERPROFILE", t.TempDir())
			t.Setenv("AXIOM_INHERITED_MARKER", "synthetic")
			auth := &recordingAuth{status: runtimeadapter.AuthSubscriptionObserved}
			guard, graph, _, _ := subscriptionGuard(t, runtimeID, "", nil, auth, "high")
			invocation, err := guard.ResolveInvocation(context.Background(), graph.Children[0])
			if err != nil {
				t.Fatal(err)
			}
			if len(auth.targets) != 1 || !slices.Equal(auth.targets[0].Environment, executiongraph.EffectiveInvocationEnvironment(invocation)) {
				t.Fatal("preflight and runner environments differ")
			}
			for _, key := range []string{"HOME", "USERPROFILE", "PATH", "AXIOM_INHERITED_MARKER"} {
				if !slices.Contains(auth.targets[0].Environment, key+"="+os.Getenv(key)) {
					t.Fatalf("missing inherited key %s", key)
				}
			}
			if runtimeID == "codex" && !slices.Equal(invocation.Argv[len(invocation.Argv)-2:], []string{"--config", "model_reasoning_effort=high"}) {
				t.Fatal("missing exact effort arguments")
			}
			if runtimeID == "claude" && !slices.Contains(invocation.Env, "CLAUDE_CODE_EFFORT_LEVEL=high") {
				t.Fatal("missing effort environment")
			}
			child := graph.Children[0]
			child.Envelope.Resolution.ModelProfileID = "unreviewed"
			if _, err := guard.ResolveInvocation(context.Background(), child); !errors.Is(err, ErrInvalidComposition) {
				t.Fatal("invalid binding accepted")
			}
		})
	}
}

func TestSubscriptionCodexEffortArgumentsFailClosed(t *testing.T) {
	child := executiongraph.ChildExecution{}
	child.Envelope.Resolution.RuntimeID = "codex"
	child.Envelope.Controls.ReasoningEffort = "high"
	child.Envelope.Capability.Capabilities = []string{"reasoning-effort-high"}
	for _, arguments := range [][]string{
		{"--config", "model_reasoning_effort=low"},
		{"--config", "model_provider=other"},
		{"--config=model_reasoning_effort=high"},
		{"--config", "model_reasoning_effort=high", "--config", "model_reasoning_effort=high"},
		{"--config", "model_provider=other", "--config", "model_reasoning_effort=high"},
		{"--model", "other", "--config", "model_reasoning_effort=high"},
	} {
		if subscriptionSafeArguments(child, arguments) {
			t.Fatal("unsafe arguments accepted")
		}
		if selectionSafeArguments("codex", arguments) {
			t.Fatal("configured override accepted")
		}
	}
	child.Envelope.Capability.Capabilities = nil
	if subscriptionSafeArguments(child, []string{"--config", "model_reasoning_effort=high"}) {
		t.Fatal("unproven effort accepted")
	}
}

type claudeEffortStatus struct{ environments [][]string }

func (*claudeEffortStatus) EvidenceKind() string { return runtimeadapter.EvidenceFake }
func (s *claudeEffortStatus) RunStatus(_ context.Context, command runtimeadapter.StatusCommand) runtimeadapter.StatusResult {
	s.environments = append(s.environments, slices.Clone(command.Env))
	output := `{"loggedIn":true,"authMethod":"claude.ai","apiProvider":"firstParty"}`
	if command.Argv[1] == "--version" {
		output = "2.1.295 (Claude Code)"
	}
	return runtimeadapter.StatusResult{Started: true, Output: []byte(output)}
}

func TestClaudeSubscriptionEffortUsesInheritedEnvironmentInRealPreflight(t *testing.T) {
	// Isolate the controlled login observation from host provider/configuration
	// overrides, while retaining the ordinary inherited environment.
	for _, entry := range os.Environ() {
		key, _, _ := strings.Cut(entry, "=")
		if strings.HasPrefix(key, "CLAUDE_") || strings.HasPrefix(key, "ANTHROPIC_") || strings.HasPrefix(key, "CLOUD_ML_") || strings.HasPrefix(key, "VERTEX_") || key == "AWS_BEARER_TOKEN_BEDROCK" {
			t.Setenv(key, "")
			if err := os.Unsetenv(key); err != nil {
				t.Fatal(err)
			}
		}
	}
	t.Setenv("HOME", canonicalTempDir(t))
	t.Setenv("USERPROFILE", os.Getenv("HOME"))
	status := &claudeEffortStatus{}
	preflight, err := runtimeadapter.NewAuthPreflight(status, runtimeadapter.ManagedConfiguration{})
	if err != nil {
		t.Fatal(err)
	}
	guard, graph, _, _ := subscriptionGuard(t, "claude", "", nil, preflight, "high")
	child := graph.Children[0]
	if err := os.MkdirAll(child.Envelope.Workspace, 0o700); err != nil {
		t.Fatal(err)
	}
	invocation, err := guard.ResolveInvocation(context.Background(), child)
	if err != nil {
		t.Fatal(err)
	}
	if len(status.environments) != 2 {
		t.Fatal("missing version/login observations")
	}
	for _, environment := range status.environments {
		if !slices.Equal(environment, executiongraph.EffectiveInvocationEnvironment(invocation)) {
			t.Fatal("preflight status and process runner environments differ")
		}
	}
	// Inheritance must not hide an API credential from the authentication guard.
	t.Setenv("ANTHROPIC_API_KEY", "synthetic-private-value")
	_, err = guard.ResolveInvocation(context.Background(), child)
	reports := guard.authentication.reports()
	if !errors.Is(err, executiongraph.ErrAuthenticationBlocked) || len(status.environments) != 2 || len(reports) != 1 || reports[0].Reason != "environment_override" {
		t.Fatal("inherited API credential was not blocked before status execution")
	}
	if strings.Contains(err.Error(), "synthetic-private-value") || strings.Contains(strings.Join(reports[0].Overrides, ","), "synthetic-private-value") {
		t.Fatal("credential value leaked")
	}
}

type effortWorkspace struct{}

func (effortWorkspace) ValidateWorkspace(context.Context, executiongraph.ChildExecution) error {
	return nil
}

type effortRunner struct {
	mu           sync.Mutex
	environments [][]string
}

func (r *effortRunner) Run(_ context.Context, invocation executiongraph.Invocation) executiongraph.ProcessResult {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.environments = append(r.environments, executiongraph.EffectiveInvocationEnvironment(invocation))
	return executiongraph.ProcessResult{ExitCode: 0}
}
func TestClaudeEffortDispatchPreservesLargeInheritedEnvironment(t *testing.T) {
	for i := 0; i < 70; i++ {
		t.Setenv(fmt.Sprintf("AXM_INHERITED_%d", i), "synthetic")
	}
	t.Setenv("lowercase_inherited", "synthetic")
	auth := &recordingAuth{status: runtimeadapter.AuthSubscriptionObserved}
	guard, graph, _, _ := subscriptionGuard(t, "claude", "", nil, auth, "high")
	runner := &effortRunner{}
	scheduler := executiongraph.NewScheduler(guard, runner, effortWorkspace{}, nil, nil, nil)
	result, err := scheduler.DispatchReady(context.Background(), executiongraph.DispatchRequest{Graph: graph})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Records) == 0 || len(auth.targets) != len(runner.environments) {
		t.Fatal("effort invocation failed scheduler admission")
	}
	for i, environment := range runner.environments {
		if !slices.Equal(environment, auth.targets[i].Environment) {
			t.Fatal("preflight and runner received different environments")
		}
		if !slices.Contains(environment, "lowercase_inherited=synthetic") || !slices.Contains(environment, "CLAUDE_CODE_EFFORT_LEVEL=high") {
			t.Fatal("inherited environment or effort missing at dispatch")
		}
	}
}
