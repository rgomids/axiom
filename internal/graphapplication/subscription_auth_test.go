package graphapplication

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
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

func subscriptionGuard(t *testing.T, runtimeID, credential string, environment []string, auth SubscriptionAuthenticator) (policyInvocations, executiongraph.Graph, *countedCredentials, string) {
	t.Helper()
	graph := buildGraph(t, canonicalTempDir(t))
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
			if target.RuntimeID != runtimeID || target.Executable != invocation.Argv[0] || target.WorkingDirectory != invocation.CWD || target.ExpectedDigest != digest || !slices.Equal(target.Environment, executiongraph.EffectiveEnvironment(invocation.Env)) {
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
