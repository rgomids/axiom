package runtimeadapter

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/rgomids/axiom/internal/executiongraph"
)

func identityRequest(runtime string, write, process bool) StageInvocationRequest {
	return StageInvocationRequest{Profile: CommandProfile{RuntimeID: runtime, Executable: nativeFixturePath("/bin/" + runtime), Model: "model"}, Workspace: nativeFixturePath("/work"), Inbox: nativeFixturePath("/inbox"), AxiomPath: nativeFixturePath("/bin/axiom"), RepositoryWrite: write, Process: process}
}

func TestStageArgvGolden(t *testing.T) {
	for _, tc := range []struct {
		runtime string
		write   bool
		want    []string
		cwd     string
	}{
		{"codex", false, []string{"/bin/codex", "exec", "--model", "model", "--ignore-user-config", "--ephemeral", "-", "--sandbox", "workspace-write", "-C", "/inbox"}, "/inbox"},
		{"codex", true, []string{"/bin/codex", "exec", "--model", "model", "--ignore-user-config", "--ephemeral", "-", "--sandbox", "workspace-write", "-C", "/work", "--add-dir", "/inbox"}, "/work"},
		{"claude", false, []string{"/bin/claude", "--print", "--model", "model", "--input-format", "text", "--permission-prompts", "none", "--permission-mode", "dontAsk", "--allowedTools", "Read,Glob,Grep,Write,Bash(/bin/axiom:*)", "--add-dir", "/inbox"}, "/inbox"},
		{"claude", true, []string{"/bin/claude", "--print", "--model", "model", "--input-format", "text", "--permission-prompts", "none", "--permission-mode", "dontAsk", "--allowedTools", "Read,Glob,Grep,Write,Edit,Bash(/bin/axiom:*)", "--add-dir", "/inbox"}, "/work"},
	} {
		t.Run(tc.runtime+tc.cwd, func(t *testing.T) {
			inv, err := BuildStageInvocation(StageInvocationRequest{Profile: CommandProfile{RuntimeID: tc.runtime, Executable: "/bin/" + tc.runtime, Model: "model"}, Workspace: "/work", Inbox: "/inbox", AxiomPath: "/bin/axiom", RepositoryWrite: tc.write})
			if err != nil || !slices.Equal(inv.Argv, tc.want) || inv.CWD != tc.cwd {
				t.Fatalf("%+v %v", inv, err)
			}
			if len(executiongraph.EffectiveInvocationEnvironment(inv)) != 0 {
				t.Fatal("empty environment inherited")
			}
		})
	}
}

func TestStageRejectsConfiguredArguments(t *testing.T) {
	for _, argument := range []string{"--dangerously-bypass-approvals-and-sandbox", "--sandbox=danger-full-access", "--dangerously-skip-permissions", "--allow-dangerously-skip-permissions", "--permission-mode=bypassPermissions", "--bare", "--settings", "--profile", "--config", "--unknown"} {
		for _, runtime := range []string{"codex", "claude"} {
			_, err := BuildStageInvocation(StageInvocationRequest{Profile: CommandProfile{RuntimeID: runtime, Executable: "/bin/" + runtime, Model: "model", Arguments: []string{argument}}, Workspace: "/work", Inbox: "/inbox", AxiomPath: "/bin/axiom"})
			if err == nil {
				t.Fatal(argument)
			}
		}
	}
}

func TestStageInvocationHasNoVersionGate(t *testing.T) {
	for _, runtimeID := range []string{"codex", "claude"} {
		if _, err := BuildStageInvocation(identityRequest(runtimeID, false, false)); err != nil {
			t.Fatal(runtimeID, err)
		}
	}
}

func TestStageInvocationRefusesInheritedConfigurationAndEffort(t *testing.T) {
	for _, runtimeID := range []string{"codex", "claude"} {
		for _, variable := range []string{"CODEX_HOME", "CLAUDE_CONFIG_DIR", "codex_home", "claude_config_dir"} {
			request := identityRequest(runtimeID, true, false)
			request.ParentEnvironment = []string{"HOME=/home/example", variable + "=/vendor"}
			if _, err := BuildStageInvocation(request); !errors.Is(err, ErrEffortConfigurationUnsafe) {
				t.Fatal(runtimeID, variable, err)
			}
		}
		request := identityRequest(runtimeID, true, false)
		request.ParentEnvironment = []string{"CLAUDE_CODE_EFFORT_LEVEL=high"}
		_, err := BuildStageInvocation(request)
		if runtimeID == "claude" && !errors.Is(err, ErrEffortOverrideConflict) || runtimeID == "codex" && err != nil {
			t.Fatal(runtimeID, err)
		}
	}
}

func TestStageEnvironmentClosed(t *testing.T) {
	parent := []string{"HOME=/home/example", "PATH=/bin", "CODEX_HOME=/vendor", "CLAUDE_CONFIG_DIR=/other", "LANG=C", "LC_ALL=C", "AXIOM_ATTEMPT_ID=a", "GH_TOKEN=synthetic", "OPENAI_API_KEY=synthetic", "AWS_ACCESS_KEY_ID=synthetic", "AXIOM_UNKNOWN=synthetic"}
	for _, runtime := range []string{"codex", "claude"} {
		env, err := StageEnvironment(runtime, parent)
		if err != nil {
			t.Fatal(err)
		}
		want := []string{"HOME=/home/example", "PATH=/bin", "LANG=C", "LC_ALL=C"}
		if runtime == "codex" {
			want = append(want, "CODEX_HOME=/vendor")
		} else {
			want = append(want, "CLAUDE_CONFIG_DIR=/other")
		}
		slices.Sort(want)
		if !slices.Equal(env, want) {
			t.Fatal(env)
		}
	}
	if _, err := StageEnvironment("codex", []string{"HOME=a", "HOME=b"}); err == nil {
		t.Fatal("duplicate")
	}
}

func TestStageAuthDoesNotInheritDesktopRouting(t *testing.T) {
	fixture := newAuthFixture(t, "claude")
	fixture.env = append(fixture.env, "ANTHROPIC_BASE_URL=https://gateway.invalid", "CLAUDE_CODE_MESSAGING_TOKEN=synthetic-session-token", "CLAUDE_CODE_OAUTH_TOKEN=synthetic-oauth-token")
	runner := &fakeStatus{version: ok("2.1.310"), status: ok(claudeSubscription)}
	preflight, err := NewAuthPreflight(runner, ManagedConfiguration{})
	if err != nil {
		t.Fatal(err)
	}
	parent := fixture.target("claude")
	if report := preflight.Check(context.Background(), parent); report.Status != AuthIncompatible || len(runner.calls) != 0 {
		t.Fatalf("parent override not blocked: %+v", report)
	}
	child := fixture.target("claude")
	child.Environment, err = StageEnvironment("claude", parent.Environment)
	if err != nil {
		t.Fatal(err)
	}
	if report := preflight.Check(context.Background(), child); !report.DispatchAllowed() || len(runner.calls) != 2 {
		t.Fatalf("isolated child login not observed: %+v", report)
	}
	if report := preflight.Check(context.Background(), parent); report.Status != AuthIncompatible || len(runner.calls) != 2 {
		t.Fatalf("parent environment changed: %+v", report)
	}
}

func TestStageAuthMissingLoginRoot(t *testing.T) {
	for _, runtimeID := range []string{"codex", "claude"} {
		fixture := newAuthFixture(t, runtimeID)
		status := "Logged in using ChatGPT"
		if runtimeID == "claude" {
			status = claudeSubscription
		}
		runner := &fakeStatus{version: ok("0.159.1"), status: ok(status)}
		preflight, err := NewAuthPreflight(runner, ManagedConfiguration{})
		if err != nil {
			t.Fatal(err)
		}
		env, err := StageEnvironment(runtimeID, []string{"GH_TOKEN=synthetic"})
		if err != nil {
			t.Fatal(err)
		}
		report := preflight.Check(context.Background(), AuthTarget{RuntimeID: runtimeID, Executable: fixture.executable, WorkingDirectory: fixture.cwd, Environment: env})
		if report.DispatchAllowed() || report.Reason != "configuration_root_unresolved" || len(runner.calls) != 0 {
			t.Fatalf("%+v", report)
		}
	}
}

func TestStagePathAndProcessBoundary(t *testing.T) {
	request := StageInvocationRequest{Profile: CommandProfile{RuntimeID: "claude", Executable: "/bin/claude", Model: "model"}, Workspace: "/work", Inbox: "/inbox", AxiomPath: "/bin/axiom"}
	for _, inbox := range []string{"/work/inbox", "/", "/inbox/../inbox", "/inbox\n", "/inbox\x00"} {
		invalid := request
		invalid.Inbox = inbox
		if _, err := BuildStageInvocation(invalid); err == nil {
			t.Fatal(inbox)
		}
	}
	request.Process = true
	invocation, err := BuildStageInvocation(request)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(invocation.Argv, "Read,Glob,Grep,Write,Bash") {
		t.Fatal(invocation.Argv)
	}
	if _, err := StageEnvironment("codex", []string{"LC_BAD-NAME=C"}); err == nil {
		t.Fatal("invalid key")
	}
}

func TestStageInvocationCorrelationIsAttemptOwned(t *testing.T) {
	correlation := AttemptCorrelation{ProjectID: "project-1", ParentID: "parent-1", ChildID: "child-1", AttemptID: "attempt-1"}
	request := identityRequest("codex", false, false)
	request.ParentEnvironment = []string{"PATH=/bin", "AXIOM_ATTEMPT_ID=stale"}
	request.Correlation = correlation
	inv, err := BuildStageInvocation(request)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"AXIOM_ATTEMPT_ID=attempt-1", "AXIOM_CORRELATION_ID=project-1", "AXIOM_EXECUTION_ID=parent-1", "AXIOM_GRAPH_ID=child-1", "PATH=/bin"}
	if got := executiongraph.EffectiveInvocationEnvironment(inv); !slices.Equal(got, want) {
		t.Fatalf("environment = %v", got)
	}
	for _, bad := range []AttemptCorrelation{{ProjectID: "p", ParentID: "a=b", ChildID: "c", AttemptID: "d"}, {ProjectID: "p", ParentID: "", ChildID: "c", AttemptID: "d"}} {
		request.Correlation = bad
		if _, err := BuildStageInvocation(request); err == nil {
			t.Fatalf("invalid correlation accepted: %+v", bad)
		}
	}
}
