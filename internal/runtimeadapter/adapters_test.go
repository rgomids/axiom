package runtimeadapter

import (
	"context"
	"errors"
	"github.com/rgomids/axiom/internal/testfs"
	"os"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/rgomids/axiom/internal/executiongraph"
	"github.com/rgomids/axiom/internal/runtimeprofile"
)

type credentialFake struct{ references []string }

func (f *credentialFake) ResolveEnvironment(_ context.Context, reference string) ([]string, error) {
	f.references = append(f.references, reference)
	return []string{"RUNTIME_TOKEN=SECRET_SENTINEL"}, nil
}

func TestInventoryExposesOnlyObservedCodexAndClaude(t *testing.T) {
	observedAt := time.Unix(1, 0).UTC()
	inventory, err := NewInventory([]runtimeprofile.Observation{{RuntimeID: "codex", Adapter: "codex", Installed: true, Available: true, Revision: 1, ObservedAt: observedAt}, {RuntimeID: "claude", Adapter: "claude", Installed: true, Available: false, Revision: 1, ObservedAt: observedAt}})
	if err != nil {
		t.Fatal(err)
	}
	observation, err := inventory.Observe(context.Background(), "claude")
	if err != nil || observation.Available {
		t.Fatalf("observation=%+v err=%v", observation, err)
	}
	if _, err := inventory.Observe(context.Background(), "other"); err == nil {
		t.Fatal("unknown adapter observed")
	}
}

func TestConcreteResolversProduceExplicitArgvAndResolveCredentialByReference(t *testing.T) {
	credentials := &credentialFake{}
	resolver, err := NewInvocationResolver([]CommandProfile{
		{RuntimeID: "codex", ModelProfileID: "codex-high", Executable: testfs.Path("/opt/bin/codex"), Model: "local-codex-model", Arguments: []string{"--json"}, Environment: []string{"PATH=/opt/bin"}, CredentialReference: "keychain:codex", OutputMax: 1024},
		{RuntimeID: "claude", ModelProfileID: "claude-high", Executable: testfs.Path("/opt/bin/claude"), Model: "local-claude-model", Arguments: []string{"--output-format", "json"}, Environment: []string{"PATH=/opt/bin"}, OutputMax: 1024},
	}, credentials)
	if err != nil {
		t.Fatal(err)
	}
	codex := adapterChild("codex", "codex-high")
	invocation, err := resolver.ResolveInvocation(context.Background(), codex)
	if err != nil || !reflect.DeepEqual(invocation.Argv, []string{testfs.Path("/opt/bin/codex"), "exec", "--model", "local-codex-model", "--json"}) || invocation.CWD != codex.Envelope.Workspace {
		t.Fatalf("invocation=%+v err=%v", invocation, err)
	}
	if !reflect.DeepEqual(credentials.references, []string{"keychain:codex"}) {
		t.Fatalf("references=%v", credentials.references)
	}
	claude := adapterChild("claude", "claude-high")
	invocation, err = resolver.ResolveInvocation(context.Background(), claude)
	if err != nil || !reflect.DeepEqual(invocation.Argv, []string{testfs.Path("/opt/bin/claude"), "--print", "--model", "local-claude-model", "--output-format", "json"}) {
		t.Fatalf("invocation=%+v err=%v", invocation, err)
	}
	codex.Envelope.Controls.ReasoningEffort = "high"
	codex.Envelope.Capability.Capabilities = []string{"reasoning-effort-high"}
	invocation, err = resolver.ResolveInvocation(context.Background(), codex)
	if err != nil || !reflect.DeepEqual(invocation.Argv, []string{testfs.Path("/opt/bin/codex"), "exec", "--model", "local-codex-model", "--json", "--config", "model_reasoning_effort=high"}) {
		t.Fatalf("Codex effort invocation=%+v err=%v", invocation, err)
	}
	claude.Envelope.Controls.ReasoningEffort = "high"
	claude.Envelope.Capability.Capabilities = []string{"reasoning-effort-high"}
	invocation, err = resolver.ResolveInvocation(context.Background(), claude)
	if err != nil || !reflect.DeepEqual(invocation.Env, []string{"CLAUDE_CODE_EFFORT_LEVEL=high", "PATH=/opt/bin"}) {
		t.Fatalf("Claude effort environment=%+v err=%v", invocation, err)
	}
}

func adapterChild(runtimeID, profileID string) executiongraph.ChildExecution {
	return executiongraph.ChildExecution{Envelope: executiongraph.ChildEnvelope{Workspace: "/tmp/isolated", Resolution: executiongraph.Resolution{RuntimeID: runtimeID, ModelProfileID: profileID, ConfigurationRevision: 1, ObservationRevision: 1}}}
}

func TestEffortMappingRejectsMissingCapabilityAndConflictingEnvironment(t *testing.T) {
	for _, runtimeID := range []string{"codex", "claude"} {
		t.Run(runtimeID, func(t *testing.T) {
			resolver, err := NewInvocationResolver([]CommandProfile{{RuntimeID: runtimeID, ModelProfileID: "profile", Executable: testfs.Path("/opt/bin/" + runtimeID), Model: "local-model", OutputMax: 1024}}, nil)
			if err != nil {
				t.Fatal(err)
			}
			child := adapterChild(runtimeID, "profile")
			child.Envelope.Controls.ReasoningEffort = "unsupported"
			if _, err := resolver.ResolveInvocation(context.Background(), child); !errors.Is(err, ErrInvalidAdapterConfiguration) {
				t.Fatalf("unproven effort err=%v", err)
			}
		})
	}
	resolver, err := NewInvocationResolver([]CommandProfile{{RuntimeID: "claude", ModelProfileID: "profile", Executable: testfs.Path("/opt/bin/claude"), Model: "local-model", Environment: []string{"CLAUDE_CODE_EFFORT_LEVEL=low"}, OutputMax: 1024}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	child := adapterChild("claude", "profile")
	child.Envelope.Controls.ReasoningEffort = "high"
	child.Envelope.Capability.Capabilities = []string{"reasoning-effort-high"}
	if _, err := resolver.ResolveInvocation(context.Background(), child); !errors.Is(err, ErrInvalidAdapterConfiguration) {
		t.Fatalf("conflicting effort environment err=%v", err)
	}
}

func TestClaudeEffortPreservesInheritedEnvironment(t *testing.T) {
	t.Setenv("HOME", testfs.Path("/tmp/claude-home"))
	t.Setenv("PATH", testfs.Path("/tmp/claude-bin"))
	t.Setenv("AXIOM_INHERITED_SNAPSHOT", "before")
	t.Setenv("CLAUDE_CODE_EFFORT_LEVEL", "")
	if err := os.Unsetenv("CLAUDE_CODE_EFFORT_LEVEL"); err != nil {
		t.Fatal(err)
	}
	resolver, err := NewInvocationResolver([]CommandProfile{{RuntimeID: "claude", ModelProfileID: "profile", Executable: testfs.Path("/opt/bin/claude"), Model: "local-model", OutputMax: 1024}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	child := adapterChild("claude", "profile")
	invocation, err := resolver.ResolveInvocation(context.Background(), child)
	if err != nil || len(invocation.Env) != 0 {
		t.Fatal("Claude without effort must retain implicit environment inheritance")
	}
	child.Envelope.Controls.ReasoningEffort = "high"
	child.Envelope.Capability.Capabilities = []string{"reasoning-effort-high"}
	want := append(os.Environ(), "CLAUDE_CODE_EFFORT_LEVEL=high")
	sort.Strings(want)
	invocation, err = resolver.ResolveInvocation(context.Background(), child)
	if err != nil || !reflect.DeepEqual(invocation.Env, []string{"CLAUDE_CODE_EFFORT_LEVEL=high"}) {
		t.Fatal("Claude effort must remain the only explicit environment override")
	}
	t.Setenv("AXIOM_INHERITED_SNAPSHOT", "after")
	effective := executiongraph.EffectiveInvocationEnvironment(invocation)
	sort.Strings(effective)
	if !reflect.DeepEqual(effective, want) {
		t.Fatal("Claude effort must preserve the complete inherited environment")
	}
}

type effortCredentialFake []string

func (f effortCredentialFake) ResolveEnvironment(context.Context, string) ([]string, error) {
	return append([]string(nil), f...), nil
}

func TestClaudeEffortPreservesExplicitEnvironmentIsolation(t *testing.T) {
	t.Setenv("AXIOM_PARENT_ONLY", "parent-placeholder")
	t.Setenv("CLAUDE_CODE_EFFORT_LEVEL", "low")
	for _, tc := range []struct {
		name        string
		environment []string
		reference   string
		credentials effortCredentialFake
		want        []string
	}{
		{name: "explicit", environment: []string{"PATH=/explicit/bin"}, want: []string{"CLAUDE_CODE_EFFORT_LEVEL=high", "PATH=/explicit/bin"}},
		{name: "credential", reference: "keychain:claude", credentials: effortCredentialFake{"RUNTIME_TOKEN=PLACEHOLDER"}, want: []string{"CLAUDE_CODE_EFFORT_LEVEL=high", "RUNTIME_TOKEN=PLACEHOLDER"}},
		{name: "empty credential", reference: "keychain:claude", want: []string{"CLAUDE_CODE_EFFORT_LEVEL=high"}},
		{name: "explicit and credential", environment: []string{"PATH=/explicit/bin"}, reference: "keychain:claude", credentials: effortCredentialFake{"RUNTIME_TOKEN=PLACEHOLDER"}, want: []string{"CLAUDE_CODE_EFFORT_LEVEL=high", "PATH=/explicit/bin", "RUNTIME_TOKEN=PLACEHOLDER"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resolver, err := NewInvocationResolver([]CommandProfile{{RuntimeID: "claude", ModelProfileID: "profile", Executable: testfs.Path("/opt/bin/claude"), Model: "local-model", Environment: tc.environment, CredentialReference: tc.reference, OutputMax: 1024}}, tc.credentials)
			if err != nil {
				t.Fatal(err)
			}
			child := adapterChild("claude", "profile")
			child.Envelope.Controls.ReasoningEffort = "high"
			child.Envelope.Capability.Capabilities = []string{"reasoning-effort-high"}
			invocation, err := resolver.ResolveInvocation(context.Background(), child)
			if err != nil || !reflect.DeepEqual(invocation.Env, tc.want) || !reflect.DeepEqual(executiongraph.EffectiveInvocationEnvironment(invocation), tc.want) {
				t.Fatal("Claude effort must preserve explicit environment and credential isolation")
			}
		})
	}
}

func TestClaudeEffortRejectsDuplicateEnvironmentFromEverySource(t *testing.T) {
	for _, source := range []string{"inherited", "explicit", "credential"} {
		for _, value := range []string{"high", "low"} {
			t.Run(source+"/"+value, func(t *testing.T) {
				profile := CommandProfile{RuntimeID: "claude", ModelProfileID: "profile", Executable: testfs.Path("/opt/bin/claude"), Model: "local-model", OutputMax: 1024}
				var credentials effortCredentialFake
				switch source {
				case "inherited":
					t.Setenv("CLAUDE_CODE_EFFORT_LEVEL", value)
				case "explicit":
					profile.Environment = []string{"CLAUDE_CODE_EFFORT_LEVEL=" + value}
				case "credential":
					profile.CredentialReference = "keychain:claude"
					credentials = effortCredentialFake{"CLAUDE_CODE_EFFORT_LEVEL=" + value}
				}
				resolver, err := NewInvocationResolver([]CommandProfile{profile}, credentials)
				if err != nil {
					t.Fatal(err)
				}
				child := adapterChild("claude", "profile")
				child.Envelope.Controls.ReasoningEffort = "high"
				child.Envelope.Capability.Capabilities = []string{"reasoning-effort-high"}
				if _, err := resolver.ResolveInvocation(context.Background(), child); !errors.Is(err, ErrInvalidAdapterConfiguration) {
					t.Fatal("Claude effort must reject matching and conflicting duplicate environment values")
				}
			})
		}
	}
}
