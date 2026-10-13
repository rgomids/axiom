package runtimeadapter

import (
	"context"
	"github.com/rgomids/axiom/internal/executiongraph"
	"slices"
	"testing"
)

func TestStageArgvGolden(t *testing.T) {
	skipPOSIXStagePaths(t)
	for _, tc := range []struct {
		runtime, version string
		write            bool
		want             []string
		cwd              string
	}{
		{"codex", "0.159.1", false, []string{"/bin/codex", "exec", "--model", "model", "--ignore-user-config", "--ephemeral", "-", "--sandbox", "workspace-write", "-C", "/inbox"}, "/inbox"},
		{"codex", "0.159.1", true, []string{"/bin/codex", "exec", "--model", "model", "--ignore-user-config", "--ephemeral", "-", "--sandbox", "workspace-write", "-C", "/work", "--add-dir", "/inbox"}, "/work"},
		{"claude", "2.1.296", false, []string{"/bin/claude", "--print", "--model", "model", "--input-format", "text", "--permission-prompts", "none", "--permission-mode", "dontAsk", "--allowedTools", "Read,Glob,Grep,Write,Bash(/bin/axiom:*)", "--add-dir", "/inbox"}, "/inbox"},
		{"claude", "2.1.296", true, []string{"/bin/claude", "--print", "--model", "model", "--input-format", "text", "--permission-prompts", "none", "--permission-mode", "dontAsk", "--allowedTools", "Read,Glob,Grep,Write,Edit,Bash(/bin/axiom:*)", "--add-dir", "/inbox"}, "/work"},
	} {
		t.Run(tc.runtime+tc.cwd, func(t *testing.T) {
			inv, err := BuildStageInvocation(StageInvocationRequest{Profile: CommandProfile{RuntimeID: tc.runtime, Executable: "/bin/" + tc.runtime, Model: "model"}, Version: tc.version, Workspace: "/work", Inbox: "/inbox", AxiomPath: "/bin/axiom", RepositoryWrite: tc.write})
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
			version := "0.159.1"
			if runtime == "claude" {
				version = "2.1.296"
			}
			_, err := BuildStageInvocation(StageInvocationRequest{Profile: CommandProfile{RuntimeID: runtime, Executable: "/bin/" + runtime, Model: "model", Arguments: []string{argument}}, Version: version, Workspace: "/work", Inbox: "/inbox", AxiomPath: "/bin/axiom"})
			if err == nil {
				t.Fatal(argument)
			}
		}
	}
	for _, version := range []string{"", "unknown", "0.159.2"} {
		_, err := BuildStageInvocation(StageInvocationRequest{Profile: CommandProfile{RuntimeID: "codex", Executable: "/bin/codex", Model: "model"}, Version: version, Workspace: "/work", Inbox: "/inbox", AxiomPath: "/bin/axiom"})
		if err == nil {
			t.Fatal(version)
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
		want := []string{"HOME=/home/example", "PATH=/bin", "LANG=C", "LC_ALL=C", "AXIOM_ATTEMPT_ID=a"}
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
	skipPOSIXStagePaths(t)
	request := StageInvocationRequest{Profile: CommandProfile{RuntimeID: "claude", Executable: "/bin/claude", Model: "model"}, Version: "2.1.296", Workspace: "/work", Inbox: "/inbox", AxiomPath: "/bin/axiom"}
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
