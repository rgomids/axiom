package runtimeadapter

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// fakeStatus scripts vendor status/version output. Every report it produces
// is labelled EvidenceFake: no real CLI runs.
type fakeStatus struct {
	version, status StatusResult
	calls           [][]string
	envs            [][]string
	during          func()
}

func (f *fakeStatus) EvidenceKind() string { return EvidenceFake }

func (f *fakeStatus) RunStatus(_ context.Context, command StatusCommand) StatusResult {
	f.calls = append(f.calls, command.Argv)
	f.envs = append(f.envs, command.Env)
	if len(command.Argv) > 1 && command.Argv[1] == "--version" {
		return f.version
	}
	if f.during != nil {
		f.during()
	}
	return f.status
}

func ok(output string) StatusResult { return StatusResult{Started: true, Output: []byte(output)} }
func exit(code int, output string) StatusResult {
	return StatusResult{Started: true, ExitCode: code, Output: []byte(output)}
}

const claudeSubscription = `{"loggedIn":true,"authMethod":"claude.ai","apiProvider":"firstParty","email":"person@example.com","orgId":"org-secret-id","orgName":"Private Org","subscriptionType":"max"}`

type authFixture struct {
	home, cwd, executable string
	env                   []string
}

func newAuthFixture(t *testing.T, runtimeID string) authFixture {
	t.Helper()
	root := t.TempDir()
	fixture := authFixture{home: filepath.Join(root, "home"), cwd: filepath.Join(root, "workspace"), executable: filepath.Join(root, "bin", runtimeID)}
	for _, directory := range []string{fixture.home, fixture.cwd, filepath.Dir(fixture.executable)} {
		if err := os.MkdirAll(directory, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(fixture.executable, []byte("#!/bin/sh\nexit 0\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	fixture.env = []string{"HOME=" + fixture.home, "PATH=/usr/bin:/bin"}
	return fixture
}

func (f authFixture) target(runtimeID string) AuthTarget {
	return AuthTarget{RuntimeID: runtimeID, Executable: f.executable, WorkingDirectory: f.cwd, Environment: append([]string(nil), f.env...)}
}

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func check(t *testing.T, runtimeID string, fixture authFixture, runner *fakeStatus, managed ...string) AuthReport {
	t.Helper()
	configuration := ManagedConfiguration{}
	for _, path := range managed {
		if strings.HasSuffix(path, ".toml") {
			configuration.CodexConfig = append(configuration.CodexConfig, path)
		} else {
			configuration.ClaudeSettings = append(configuration.ClaudeSettings, path)
		}
	}
	preflight, err := NewAuthPreflight(runner, configuration)
	if err != nil {
		t.Fatal(err)
	}
	return preflight.Check(context.Background(), fixture.target(runtimeID))
}

func TestAuthPreflightVendorStatusOutcomes(t *testing.T) {
	for name, tc := range map[string]struct {
		runtimeID      string
		version        StatusResult
		status         StatusResult
		want           AuthStatus
		reason, method string
	}{
		"codex chatgpt":             {"codex", ok("codex-cli 0.159.1\n"), ok("WARNING: unrelated\nLogged in using ChatGPT\n"), AuthSubscriptionObserved, "", "chatgpt"},
		"claude claude.ai":          {"claude", ok("2.1.295 (Claude Code)\n"), ok(claudeSubscription), AuthSubscriptionObserved, "", "claude.ai"},
		"codex not logged in":       {"codex", ok("codex-cli 0.159.1"), exit(1, "Not logged in\n"), AuthUnavailable, "not_logged_in", "none"},
		"claude not logged in":      {"claude", ok("2.1.295"), exit(1, `{"loggedIn":false,"authMethod":"none","apiProvider":"firstParty"}`), AuthUnavailable, "not_logged_in", "none"},
		"codex api key login":       {"codex", ok("codex-cli 0.159.1"), ok("Logged in using an API key - sk-proj-***abcd\n"), AuthIncompatible, "api_key_login", "api_key"},
		"claude api key login":      {"claude", ok("2.1.295"), ok(`{"loggedIn":true,"authMethod":"api_key","apiProvider":"firstParty"}`), AuthIncompatible, "api_key_login", "api_key"},
		"claude bedrock provider":   {"claude", ok("2.1.295"), ok(`{"loggedIn":true,"authMethod":"third_party","apiProvider":"bedrock"}`), AuthIncompatible, "provider_override", "third_party"},
		"codex ambiguous lines":     {"codex", ok("codex-cli 0.159.1"), ok("Logged in using ChatGPT\nNot logged in\n"), AuthUnproven, "status_ambiguous", "unknown"},
		"codex login but failed":    {"codex", ok("codex-cli 0.159.1"), exit(2, "Logged in using ChatGPT\n"), AuthUnproven, "status_ambiguous", "unknown"},
		"claude login but failed":   {"claude", ok("2.1.295"), exit(1, claudeSubscription), AuthUnproven, "status_ambiguous", "unknown"},
		"codex unrecognized":        {"codex", ok("codex-cli 0.159.1"), ok("Logged in somehow\n"), AuthUnproven, "status_unrecognized", "unknown"},
		"claude unknown method":     {"claude", ok("2.1.295"), ok(`{"loggedIn":true,"authMethod":"future","apiProvider":"firstParty"}`), AuthUnproven, "auth_method_unrecognized", "unknown"},
		"claude cached flag only":   {"claude", ok("2.1.295"), ok(`{"loggedIn":true}`), AuthUnproven, "status_unrecognized", "unknown"},
		"codex unsupported surface": {"codex", ok("codex-cli 0.1.0"), exit(2, "error: unrecognized subcommand 'login'"), AuthUnsupported, "status_surface_unsupported", "unknown"},
		"claude unsupported":        {"claude", ok("1.0.0"), exit(1, "error: unknown command 'auth'"), AuthUnsupported, "status_surface_unsupported", "unknown"},
		"status timeout":            {"codex", ok("codex-cli 0.159.1"), StatusResult{Started: true, TimedOut: true, ExitCode: -1}, AuthUnproven, "status_timeout", "unknown"},
		"version failure":           {"claude", exit(1, ""), ok(claudeSubscription), AuthUnproven, "version_unavailable", "unknown"},
		"version unrecognized":      {"claude", ok("Claude Code"), ok(claudeSubscription), AuthUnproven, "version_unrecognized", "unknown"},
		"prerelease version":        {"codex", ok("codex-cli 1.2.3-rc.4"), ok("Logged in using ChatGPT"), AuthSubscriptionObserved, "", "chatgpt"},
		"process not started":       {"codex", StatusResult{}, ok(""), AuthUnavailable, "executable_unavailable", "unknown"},
	} {
		t.Run(name, func(t *testing.T) {
			fixture := newAuthFixture(t, tc.runtimeID)
			runner := &fakeStatus{version: tc.version, status: tc.status}
			report := check(t, tc.runtimeID, fixture, runner)
			if report.Status != tc.want || report.Reason != tc.reason || report.Method != tc.method {
				t.Fatalf("report=%+v", report)
			}
			if report.EvidenceKind != EvidenceFake || report.Usability != UsabilityUnproven || report.Revalidation != RevalidateBeforeDispatch || report.DispatchAllowed() != (tc.want == AuthSubscriptionObserved) {
				t.Fatalf("evidence labels=%+v", report)
			}
		})
	}
}

func TestAuthPreflightInspectsTheExactEffectiveInvocation(t *testing.T) {
	fixture := newAuthFixture(t, "claude")
	runner := &fakeStatus{version: ok("2.1.295"), status: ok(claudeSubscription)}
	report := check(t, "claude", fixture, runner)
	if !report.DispatchAllowed() || report.Version != "2.1.295" || report.ExecutableDigest == "" || len(runner.calls) != 2 {
		t.Fatalf("report=%+v calls=%v", report, runner.calls)
	}
	if strings.Join(runner.calls[1], " ") != fixture.executable+" auth status --json" || strings.Join(runner.envs[1], ",") != strings.Join(fixture.env, ",") {
		t.Fatalf("status command=%v env=%v", runner.calls[1], runner.envs[1])
	}
	codex := newAuthFixture(t, "codex")
	runner = &fakeStatus{version: ok("codex-cli 0.159.1"), status: ok("Logged in using ChatGPT")}
	check(t, "codex", codex, runner)
	if strings.Join(runner.calls[1], " ") != codex.executable+" login status" {
		t.Fatalf("codex status command=%v", runner.calls[1])
	}
}

func TestAuthPreflightRejectsEnvironmentOverridesBeforeRunningStatus(t *testing.T) {
	for name, tc := range map[string]struct{ runtimeID, entry, override string }{
		"openai key":          {"codex", "OPENAI_API_KEY=sk-live-secret", "env:OPENAI_API_KEY"},
		"openai base url":     {"codex", "OPENAI_BASE_URL=https://gateway.invalid", "env:OPENAI_BASE_URL"},
		"codex exec key":      {"codex", "CODEX_API_KEY=sk-live-secret", "env:CODEX_API_KEY"},
		"azure openai":        {"codex", "AZURE_OPENAI_API_KEY=secret", "env:AZURE_OPENAI_API_KEY"},
		"anthropic key":       {"claude", "ANTHROPIC_API_KEY=sk-ant-secret", "env:ANTHROPIC_API_KEY"},
		"anthropic token":     {"claude", "ANTHROPIC_AUTH_TOKEN=secret", "env:ANTHROPIC_AUTH_TOKEN"},
		"anthropic gateway":   {"claude", "ANTHROPIC_BASE_URL=https://gateway.invalid", "env:ANTHROPIC_BASE_URL"},
		"bedrock":             {"claude", "CLAUDE_CODE_USE_BEDROCK=1", "env:CLAUDE_CODE_USE_BEDROCK"},
		"vertex":              {"claude", "CLAUDE_CODE_USE_VERTEX=1", "env:CLAUDE_CODE_USE_VERTEX"},
		"oauth token env":     {"claude", "CLAUDE_CODE_OAUTH_TOKEN=secret", "env:CLAUDE_CODE_OAUTH_TOKEN"},
		"bedrock bearer":      {"claude", "AWS_BEARER_TOKEN_BEDROCK=secret", "env:AWS_BEARER_TOKEN_BEDROCK"},
		"lowercase inherited": {"claude", "anthropic_api_key=secret", "env:ANTHROPIC_API_KEY"},
		"api key descriptor":  {"claude", "CLAUDE_CODE_API_KEY_FILE_DESCRIPTOR=3", "env:CLAUDE_CODE_API_KEY_FILE_DESCRIPTOR"},
		"oauth descriptor":    {"claude", "CLAUDE_CODE_OAUTH_TOKEN_FILE_DESCRIPTOR=3", "env:CLAUDE_CODE_OAUTH_TOKEN_FILE_DESCRIPTOR"},
		"codex refresh url":   {"codex", "CODEX_REFRESH_TOKEN_URL_OVERRIDE=https://x.invalid", "env:CODEX_REFRESH_TOKEN_URL_OVERRIDE"},
	} {
		t.Run(name, func(t *testing.T) {
			fixture := newAuthFixture(t, tc.runtimeID)
			fixture.env = append(fixture.env, tc.entry)
			runner := &fakeStatus{version: ok("1.2.3"), status: ok("Logged in using ChatGPT")}
			report := check(t, tc.runtimeID, fixture, runner)
			if report.Status != AuthIncompatible || report.Reason != "environment_override" || strings.Join(report.Overrides, ",") != tc.override || len(runner.calls) != 0 {
				t.Fatalf("report=%+v calls=%v", report, runner.calls)
			}
		})
	}
	// Variables of the other vendor, or unrelated tokens, do not change this path.
	fixture := newAuthFixture(t, "codex")
	fixture.env = append(fixture.env, "ANTHROPIC_API_KEY=other-vendor", "GITHUB_TOKEN=unrelated", "CODEX_HOME=", "CLAUDE_CODE_ENTRYPOINT=cli")
	if report := check(t, "codex", fixture, &fakeStatus{version: ok("codex-cli 0.159.1"), status: ok("Logged in using ChatGPT")}); !report.DispatchAllowed() {
		t.Fatalf("unrelated variables blocked: %+v", report)
	}
	fixture.env = append(fixture.env, "HOME=/duplicate")
	if report := check(t, "codex", fixture, &fakeStatus{}); report.Status != AuthUnproven || report.Reason != "environment_ambiguous" {
		t.Fatalf("duplicate key report=%+v", report)
	}
}

func TestAuthPreflightRejectsConfigurationOverrides(t *testing.T) {
	codexStatus := func() *fakeStatus {
		return &fakeStatus{version: ok("codex-cli 0.159.1"), status: ok("Logged in using ChatGPT")}
	}
	claudeStatus := func() *fakeStatus { return &fakeStatus{version: ok("2.1.295"), status: ok(claudeSubscription)} }
	for name, tc := range map[string]struct {
		runtimeID, file, content string
		managed                  bool
		want                     AuthStatus
		reason, override         string
	}{
		"codex custom provider":     {"codex", "home/.codex/config.toml", "model = \"o4\"\nmodel_provider = \"azure\" # gateway\n", false, AuthIncompatible, "configuration_override", "codex_config:user:model_provider"},
		"codex provider table":      {"codex", "home/.codex/config.toml", "[model_providers.proxy]\nbase_url = \"https://x.invalid\"\nenv_key = \"PROXY_KEY\"\n", false, AuthIncompatible, "configuration_override", "codex_config:user:base_url,codex_config:user:model_providers"},
		"codex inline profile":      {"codex", "home/.codex/config.toml", "profiles = { fast = { model_provider = \"azure\" } }\n", false, AuthIncompatible, "configuration_override", "codex_config:user:profiles"},
		"codex base url":            {"codex", "home/.codex/config.toml", "openai_base_url = \"https://gateway.invalid\"\n", false, AuthIncompatible, "configuration_override", "codex_config:user:openai_base_url"},
		"codex home managed":        {"codex", "home/.codex/managed_config.toml", "forced_login_method = \"api\"\n", false, AuthIncompatible, "configuration_override", "codex_config:managed:forced_login_method"},
		"codex system managed":      {"codex", "etc/codex/managed_config.toml", "model_provider = \"oss\"\n", true, AuthIncompatible, "configuration_override", "codex_config:managed:model_provider"},
		"claude managed drop-in":    {"claude", "managed/managed-settings.d/10-key.json", `{"apiKeyHelper":"x"}`, false, AuthIncompatible, "configuration_override", "claude_settings:managed:apiKeyHelper"},
		"codex profile provider":    {"codex", "home/.codex/config.toml", "[profiles.fast]\nmodel_provider = 'ollama'\n", false, AuthIncompatible, "configuration_override", "codex_config:user:model_provider"},
		"codex api key preference":  {"codex", "home/.codex/config.toml", "preferred_auth_method = \"apikey\"\n", false, AuthIncompatible, "configuration_override", "codex_config:user:preferred_auth_method"},
		"codex forced api login":    {"codex", "home/.codex/config.toml", "forced_login_method = \"api\"\n", false, AuthIncompatible, "configuration_override", "codex_config:user:forced_login_method"},
		"codex project config":      {"codex", "workspace/.codex/config.toml", "model_provider = \"oss\"\n", false, AuthIncompatible, "configuration_override", "codex_config:project:model_provider"},
		"codex compatible config":   {"codex", "home/.codex/config.toml", "# model_provider = \"azure\"\nmodel_provider = \"openai\"\npreferred_auth_method = \"chatgpt\"\nforced_login_method = \"chatgpt\"\n", false, AuthSubscriptionObserved, "", ""},
		"claude api key helper":     {"claude", "home/.claude/settings.json", `{"apiKeyHelper":"/usr/local/bin/print-secret"}`, false, AuthIncompatible, "configuration_override", "claude_settings:user:apiKeyHelper"},
		"claude settings env key":   {"claude", "workspace/.claude/settings.json", `{"env":{"ANTHROPIC_API_KEY":"sk-ant-secret"}}`, false, AuthIncompatible, "configuration_override", "claude_settings:project:env:ANTHROPIC_API_KEY"},
		"claude local bedrock":      {"claude", "workspace/.claude/settings.local.json", `{"env":{"CLAUDE_CODE_USE_BEDROCK":"1"}}`, false, AuthIncompatible, "configuration_override", "claude_settings:project_local:env:CLAUDE_CODE_USE_BEDROCK"},
		"claude console login":      {"claude", "home/.claude/settings.json", `{"forceLoginMethod":"console"}`, false, AuthIncompatible, "configuration_override", "claude_settings:user:forceLoginMethod"},
		"claude aws export":         {"claude", "home/.claude/settings.json", `{"awsCredentialExport":"/bin/creds"}`, false, AuthIncompatible, "configuration_override", "claude_settings:user:awsCredentialExport"},
		"claude managed helper":     {"claude", "managed/managed-settings.json", `{"apiKeyHelper":"x"}`, true, AuthIncompatible, "configuration_override", "claude_settings:managed:apiKeyHelper"},
		"claude invalid settings":   {"claude", "home/.claude/settings.json", `{"apiKeyHelper":`, false, AuthUnproven, "configuration_unreadable", ""},
		"claude compatible setting": {"claude", "home/.claude/settings.json", `{"forceLoginMethod":"claudeai","env":{"DISABLE_TELEMETRY":"1"},"permissions":{}}`, false, AuthSubscriptionObserved, "", ""},
	} {
		t.Run(name, func(t *testing.T) {
			fixture := newAuthFixture(t, tc.runtimeID)
			root := filepath.Dir(fixture.home)
			write(t, filepath.Join(root, tc.file), tc.content)
			managed := []string{filepath.Join(root, "managed", "managed-settings.json")}
			if tc.managed {
				managed = []string{filepath.Join(root, tc.file)}
			}
			runner := codexStatus()
			if tc.runtimeID == "claude" {
				runner = claudeStatus()
			}
			report := check(t, tc.runtimeID, fixture, runner, managed...)
			if report.Status != tc.want || report.Reason != tc.reason || strings.Join(report.Overrides, ",") != tc.override {
				t.Fatalf("report=%+v", report)
			}
			if tc.want != AuthSubscriptionObserved && len(runner.calls) != 0 {
				t.Fatalf("status ran despite blocked configuration: %v", runner.calls)
			}
		})
	}
	t.Run("relocated roots", func(t *testing.T) {
		fixture := newAuthFixture(t, "claude")
		relocated := filepath.Join(filepath.Dir(fixture.home), "relocated")
		write(t, filepath.Join(relocated, "settings.json"), `{"apiKeyHelper":"x"}`)
		fixture.env = append(fixture.env, "CLAUDE_CONFIG_DIR="+relocated)
		if report := check(t, "claude", fixture, claudeStatus()); report.Reason != "configuration_override" {
			t.Fatalf("CLAUDE_CONFIG_DIR ignored: %+v", report)
		}
		codex := newAuthFixture(t, "codex")
		write(t, filepath.Join(relocated, "config.toml"), "model_provider = \"azure\"\n")
		codex.env = append(codex.env, "CODEX_HOME="+relocated)
		if report := check(t, "codex", codex, codexStatus()); report.Reason != "configuration_override" {
			t.Fatalf("CODEX_HOME ignored: %+v", report)
		}
	})
	t.Run("unresolved or unreadable configuration fails closed", func(t *testing.T) {
		fixture := newAuthFixture(t, "codex")
		fixture.env = []string{"PATH=/usr/bin"}
		if report := check(t, "codex", fixture, codexStatus()); report.Status != AuthUnproven || report.Reason != "configuration_root_unresolved" {
			t.Fatalf("missing HOME report=%+v", report)
		}
		fixture = newAuthFixture(t, "codex")
		if err := os.MkdirAll(filepath.Join(fixture.home, ".codex", "config.toml"), 0o700); err != nil {
			t.Fatal(err)
		}
		if report := check(t, "codex", fixture, codexStatus()); report.Status != AuthUnproven || report.Reason != "configuration_unreadable" {
			t.Fatalf("directory config report=%+v", report)
		}
	})
}

func TestAuthPreflightBindsExecutableIdentity(t *testing.T) {
	fixture := newAuthFixture(t, "codex")
	runner := &fakeStatus{version: ok("codex-cli 0.159.1"), status: ok("Logged in using ChatGPT")}
	reviewed := check(t, "codex", fixture, runner)
	if !reviewed.DispatchAllowed() {
		t.Fatalf("baseline=%+v", reviewed)
	}
	preflight, _ := NewAuthPreflight(runner, ManagedConfiguration{})
	target := fixture.target("codex")
	target.ExpectedDigest = strings.Repeat("0", 64)
	if report := preflight.Check(context.Background(), target); report.Status != AuthUnproven || report.Reason != "executable_identity_changed" {
		t.Fatalf("stale binding report=%+v", report)
	}
	target.ExpectedDigest = reviewed.ExecutableDigest
	runner.during = func() { write(t, fixture.executable, "#!/bin/sh\necho replaced\n") }
	if report := preflight.Check(context.Background(), target); report.Status != AuthUnproven || report.Reason != "executable_identity_changed" {
		t.Fatalf("replacement during status report=%+v", report)
	}
	runner.during = nil
	target.Executable = filepath.Join(filepath.Dir(fixture.executable), "missing", "codex")
	if report := preflight.Check(context.Background(), target); report.Status != AuthUnavailable || report.Reason != "executable_unavailable" {
		t.Fatalf("missing executable report=%+v", report)
	}
	other := newAuthFixture(t, "claude")
	target = other.target("codex")
	if report := preflight.Check(context.Background(), target); report.Status != AuthIncompatible || report.Reason != "executable_mismatch" {
		t.Fatalf("mismatched executable report=%+v", report)
	}
	if report := preflight.Check(context.Background(), AuthTarget{RuntimeID: "gemini"}); report.Status != AuthUnsupported || report.RuntimeID != "" {
		t.Fatalf("unsupported runtime report=%+v", report)
	}
	if _, err := NewAuthPreflight(nil, ManagedConfiguration{}); err == nil {
		t.Fatal("nil runner accepted")
	}
	if _, err := NewAuthPreflight(runner, ManagedConfiguration{CodexConfig: []string{"relative/managed.toml"}}); err == nil {
		t.Fatal("relative managed settings accepted")
	}
}

// The report is the persisted/public Evidence: no credential, account
// identifier, raw vendor output, configuration value or path may enter it.
func TestAuthReportExcludesSecretsAndAccountIdentifiers(t *testing.T) {
	secrets := []string{"sk-live-secret", "sk-ant-secret", "person@example.com", "org-secret-id", "Private Org", "print-secret", "gateway.invalid", "max"}
	reports := []AuthReport{}
	claude := newAuthFixture(t, "claude")
	reports = append(reports, check(t, "claude", claude, &fakeStatus{version: ok("2.1.295 person@example.com"), status: ok(claudeSubscription)}))
	reports = append(reports, check(t, "claude", claude, &fakeStatus{version: ok("2.1.295-maxplan.secret"), status: ok(claudeSubscription)}))
	if reports[1].Version != "2.1.295" {
		t.Fatalf("version suffix retained: %q", reports[1].Version)
	}
	codex := newAuthFixture(t, "codex")
	reports = append(reports, check(t, "codex", codex, &fakeStatus{version: ok("codex-cli 0.159.1"), status: ok("Logged in using an API key - sk-live-secret\n")}))
	codex.env = append(codex.env, "OPENAI_API_KEY=sk-live-secret", "OPENAI_BASE_URL=https://gateway.invalid")
	reports = append(reports, check(t, "codex", codex, &fakeStatus{}))
	settings := newAuthFixture(t, "claude")
	write(t, filepath.Join(settings.home, ".claude", "settings.json"), `{"apiKeyHelper":"/opt/print-secret","env":{"ANTHROPIC_API_KEY":"sk-ant-secret","anthropic_bad name=x":"sk-ant-secret"}}`)
	reports = append(reports, check(t, "claude", settings, &fakeStatus{}))
	for _, report := range reports {
		wire, err := json.Marshal(report)
		if err != nil {
			t.Fatal(err)
		}
		for _, secret := range append(secrets, claude.home, codex.home, settings.home, claude.executable) {
			if strings.Contains(string(wire), secret) {
				t.Fatalf("evidence leaked %q: %s", secret, wire)
			}
		}
	}
	if !strings.Contains(strings.Join(reports[4].Overrides, ","), "claude_settings:user:env:unnamed") {
		t.Fatalf("unsafe key name not replaced: %+v", reports[4])
	}
}

// OSStatusRunner gives the child exactly the effective environment: nothing
// inherited from the Axiom process, with bounded output and recorded exit.
func TestOSStatusRunnerUsesOnlyTheGivenEnvironment(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX shell fixture")
	}
	t.Setenv("AXM_PARENT_ONLY", "inherited")
	fixture := newAuthFixture(t, "codex")
	script := "#!/bin/sh\nif [ \"$1\" = --version ]; then echo 'codex-cli 9.8.7'; exit 0; fi\n" +
		"if [ -n \"$AXM_PARENT_ONLY\" ]; then echo 'Not logged in'; exit 1; fi\necho 'Logged in using ChatGPT' >&2\n"
	write(t, fixture.executable, script)
	if err := os.Chmod(fixture.executable, 0o700); err != nil {
		t.Fatal(err)
	}
	preflight, err := NewAuthPreflight(OSStatusRunner{}, ManagedConfiguration{})
	if err != nil {
		t.Fatal(err)
	}
	report := preflight.Check(context.Background(), fixture.target("codex"))
	if !report.DispatchAllowed() || report.Version != "9.8.7" || report.EvidenceKind != EvidenceLocalObservation {
		t.Fatalf("report=%+v", report)
	}
	result := OSStatusRunner{}.RunStatus(context.Background(), StatusCommand{Argv: []string{"relative"}})
	if result.Started {
		t.Fatal("relative executable started")
	}
}

// Review of #284: quoted and escaped TOML keys are decoded before
// classification; keys the scan cannot decode are unproven, never safe.
func TestCodexConfigDecodesEscapedKeys(t *testing.T) {
	status := func() *fakeStatus {
		return &fakeStatus{version: ok("codex-cli 0.159.1"), status: ok("Logged in using ChatGPT")}
	}
	for name, tc := range map[string]struct {
		content          string
		want             AuthStatus
		reason, override string
	}{
		"escaped provider key":          {"\"model_\\u0070rovider\" = \"proxy\"\n", AuthIncompatible, "configuration_override", "codex_config:user:model_provider"},
		"escaped provider table":        {"[\"model_\\u0070roviders\".proxy]\n\"base_\\u0075rl\" = \"https://gateway.invalid/v1\"\nenv_key = \"PROXY_KEY\"\n", AuthIncompatible, "configuration_override", "codex_config:user:base_url,codex_config:user:model_providers"},
		"escaped array table":           {"[[ \"model_\\U00000070roviders\" ]]\n", AuthIncompatible, "configuration_override", "codex_config:user:model_providers"},
		"literal quoted key":            {"'model_provider' = 'oss'\n", AuthIncompatible, "configuration_override", "codex_config:user:model_provider"},
		"escaped openai value":          {"model_provider = \"opena\\u0069\"\n", AuthSubscriptionObserved, "", ""},
		"escaped other value":           {"model_provider = \"ollam\\u0061\"\n", AuthIncompatible, "configuration_override", "codex_config:user:model_provider"},
		"escaped inline table":          {"profiles = { fast = { \"model_\\u0070rovider\" = \"azure\" } }\n", AuthUnproven, "configuration_unreadable", ""},
		"undecodable key":               {"\"model_\\eprovider\" = \"proxy\"\n", AuthUnproven, "configuration_unreadable", ""},
		"unterminated key":              {"\"model_provider = \"proxy\"\n", AuthUnproven, "configuration_unreadable", ""},
		"equals inside key":             {"\"a=b\".model_provider = \"oss\"\n", AuthIncompatible, "configuration_override", "codex_config:user:model_provider"},
		"multiline string body":         {"instructions = \"\"\"\nnot a key: model_provider = \"azure\"\nfree text = here\n\"\"\"\nmodel = \"o4\"\n", AuthSubscriptionObserved, "", ""},
		"delimiters in comment":         {"model = \"\"\"gpt-5\"\"\" # \"\"\"\nmodel_provider = \"proxy\"\n[model_providers.proxy]\nname = \"Controlled fixture\"\nbase_url = \"https://gateway.invalid/v1\"\nenv_key = \"PROXY_KEY\"\nwire_api = \"responses\"\n", AuthIncompatible, "configuration_override", "codex_config:user:base_url,codex_config:user:model_provider,codex_config:user:model_providers"},
		"literal delimiters in comment": {"model = '''gpt-5''' # '''\nmodel_provider = \"proxy\"\n", AuthIncompatible, "configuration_override", "codex_config:user:model_provider"},
		"escaped delimiter inside":      {"note = \"\"\"a \\\"\"\" still open\nmodel_provider = \"azure\"\n\"\"\"\nmodel = \"o4\"\n", AuthSubscriptionObserved, "", ""},
		"hash inside string body":       {"note = \"\"\"\nfirst # \"\"\" closes here\n\"\"\" # done\nmodel_provider = \"oss\"\n", AuthUnproven, "configuration_unreadable", ""},
		"comment after closing line":    {"note = \"\"\"\nbody\n\"\"\" # \"\"\"\nmodel_provider = \"oss\"\n", AuthIncompatible, "configuration_override", "codex_config:user:model_provider"},
		"text after closing delimiter":  {"note = \"\"\"a\"\"\" trailing\nmodel_provider = \"oss\"\n", AuthUnproven, "configuration_unreadable", ""},
		"unclosed at end of file":       {"model_provider = \"openai\"\nnote = \"\"\"\nnever closed\n", AuthUnproven, "configuration_unreadable", ""},
		"apostrophe in comment":         {"model = \"gpt-5\" # user's preferred model\n# it's \"quoted\" here\n[profiles.fast] # Bob's profile\nmodel = 'o4' # \"x\n", AuthSubscriptionObserved, "", ""},
		"hash inside quoted value":      {"model_provider = \"proxy#1\" # it's custom\n", AuthIncompatible, "configuration_override", "codex_config:user:model_provider"},
		"provider after quoted comment": {"model = \"gpt-5\" # user's\nmodel_provider = \"azure\" # don't\n", AuthIncompatible, "configuration_override", "codex_config:user:model_provider"},
		"compatible decoded":            {"\"model_provider\" = \"openai\" # default\n[profiles.fast]\nmodel = \"o4\"\n", AuthSubscriptionObserved, "", ""},
	} {
		t.Run(name, func(t *testing.T) {
			fixture := newAuthFixture(t, "codex")
			write(t, filepath.Join(fixture.home, ".codex", "config.toml"), tc.content)
			report := check(t, "codex", fixture, status())
			if report.Status != tc.want || report.Reason != tc.reason || strings.Join(report.Overrides, ",") != tc.override {
				t.Fatalf("report=%+v", report)
			}
		})
	}
}

// Review of #284: project settings are inspected in every directory the
// vendor may load them from — ancestors of the working directory and the
// main checkout of a Git worktree — not only in the working directory.
func TestAuthPreflightInspectsAncestorAndWorktreeProjectSettings(t *testing.T) {
	claudeStatus := func() *fakeStatus { return &fakeStatus{version: ok("2.1.295"), status: ok(claudeSubscription)} }
	t.Run("repository root from subdirectory", func(t *testing.T) {
		fixture := newAuthFixture(t, "claude")
		write(t, filepath.Join(fixture.cwd, ".git", "HEAD"), "ref: refs/heads/main\n")
		write(t, filepath.Join(fixture.cwd, ".claude", "settings.local.json"), `{"env":{"ANTHROPIC_API_KEY":"synthetic-fixture"}}`)
		fixture.cwd = filepath.Join(fixture.cwd, "subdirectory")
		if err := os.MkdirAll(fixture.cwd, 0o700); err != nil {
			t.Fatal(err)
		}
		report := check(t, "claude", fixture, claudeStatus())
		if report.Status != AuthIncompatible || strings.Join(report.Overrides, ",") != "claude_settings:project_local:env:ANTHROPIC_API_KEY" {
			t.Fatalf("report=%+v", report)
		}
	})
	t.Run("main checkout of a worktree", func(t *testing.T) {
		fixture := newAuthFixture(t, "claude")
		root := filepath.Dir(fixture.home)
		main := filepath.Join(root, "main-checkout")
		write(t, filepath.Join(main, ".git", "worktrees", "feature", "commondir"), "../..\n")
		write(t, filepath.Join(main, ".claude", "settings.local.json"), `{"apiKeyHelper":"/opt/helper"}`)
		write(t, filepath.Join(fixture.cwd, ".git"), "gitdir: "+filepath.Join(main, ".git", "worktrees", "feature")+"\n")
		report := check(t, "claude", fixture, claudeStatus())
		if report.Status != AuthIncompatible || strings.Join(report.Overrides, ",") != "claude_settings:project_local:apiKeyHelper" {
			t.Fatalf("report=%+v", report)
		}
	})
	t.Run("codex parent project config", func(t *testing.T) {
		fixture := newAuthFixture(t, "codex")
		write(t, filepath.Join(fixture.cwd, ".codex", "config.toml"), "model_provider = \"oss\"\n")
		fixture.cwd = filepath.Join(fixture.cwd, "nested", "deeper")
		if err := os.MkdirAll(fixture.cwd, 0o700); err != nil {
			t.Fatal(err)
		}
		report := check(t, "codex", fixture, &fakeStatus{version: ok("codex-cli 0.159.1"), status: ok("Logged in using ChatGPT")})
		if report.Status != AuthIncompatible || strings.Join(report.Overrides, ",") != "codex_config:project:model_provider" {
			t.Fatalf("report=%+v", report)
		}
	})
	t.Run("unparseable worktree link", func(t *testing.T) {
		fixture := newAuthFixture(t, "claude")
		write(t, filepath.Join(fixture.cwd, ".git"), "not a gitdir link\n")
		if report := check(t, "claude", fixture, claudeStatus()); report.Status != AuthUnproven || report.Reason != "configuration_root_unresolved" {
			t.Fatalf("report=%+v", report)
		}
	})
}
