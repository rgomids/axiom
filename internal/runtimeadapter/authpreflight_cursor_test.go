package runtimeadapter

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

const cursorSubscription = `{
  "status": "authenticated",
  "isAuthenticated": true,
  "hasAccessToken": true,
  "hasRefreshToken": true,
  "message": "Logged in as person@example.com",
  "userInfo": {"email":"person@example.com","userId":"user-secret","teamName":"Private Team"}
}`

func TestClassifyCursorStatusStructuredOutcomes(t *testing.T) {
	for name, tc := range map[string]struct {
		status         StatusResult
		want           AuthStatus
		reason, method string
	}{
		"subscription login pretty json": {
			ok(cursorSubscription), AuthSubscriptionObserved, "", "cursor-login",
		},
		"subscription login compact whitespace": {
			ok(`{"status":"authenticated","isAuthenticated": true ,"hasAccessToken":true,"hasRefreshToken": true}`),
			AuthSubscriptionObserved, "", "cursor-login",
		},
		"login absent": {
			ok(`{"status":"unauthenticated","isAuthenticated":false,"hasAccessToken":false,"hasRefreshToken":false,"message":"Not logged in"}`),
			AuthUnavailable, "not_logged_in", "none",
		},
		"login absent exit 1": {
			exit(1, `{"status":"unauthenticated","isAuthenticated":false,"hasAccessToken":false,"hasRefreshToken":false}`),
			AuthUnavailable, "not_logged_in", "none",
		},
		"api key authMethod": {
			ok(`{"status":"authenticated","isAuthenticated":true,"hasAccessToken":true,"hasRefreshToken":true,"authMethod":"api_key"}`),
			AuthIncompatible, "api_key_login", "api_key",
		},
		"api key authType": {
			ok(`{"status":"authenticated","isAuthenticated":true,"hasAccessToken":true,"hasRefreshToken":true,"authType":"api-key"}`),
			AuthIncompatible, "api_key_login", "api_key",
		},
		"provider mode": {
			ok(`{"status":"authenticated","isAuthenticated":true,"hasAccessToken":true,"hasRefreshToken":true,"authMethod":"bedrock-provider"}`),
			AuthIncompatible, "provider_override", "third_party",
		},
		"isAuthenticated alone insufficient": {
			ok(`{"isAuthenticated":true}`),
			AuthUnproven, "status_unrecognized", "unknown",
		},
		"authenticated without refresh token": {
			ok(`{"status":"authenticated","isAuthenticated":true,"hasAccessToken":true,"hasRefreshToken":false}`),
			AuthUnproven, "status_unrecognized", "unknown",
		},
		"partial login": {
			ok(`{"status":"partially-authenticated","isAuthenticated":false,"hasAccessToken":true,"hasRefreshToken":false}`),
			AuthUnproven, "status_ambiguous", "unknown",
		},
		"unsupported shape": {
			ok(`{"ok":true,"loggedIn":true}`),
			AuthUnproven, "status_unrecognized", "unknown",
		},
		"malformed output": {
			ok(`{"status":"authenticated","isAuthenticated":`),
			AuthUnproven, "status_unrecognized", "unknown",
		},
		"empty output": {
			ok(""), AuthUnproven, "status_empty", "unknown",
		},
		"substring true is not evidence": {
			ok("isAuthenticated:true authenticated=true loggedIn=true"),
			AuthUnproven, "status_unrecognized", "unknown",
		},
		"authenticated but failed exit": {
			exit(2, `{"status":"authenticated","isAuthenticated":true,"hasAccessToken":true,"hasRefreshToken":true}`),
			AuthUnproven, "status_ambiguous", "unknown",
		},
		"garbage exit unsupported": {
			exit(2, "error: unknown command"),
			AuthUnsupported, "status_surface_unsupported", "unknown",
		},
		"unrecognized auth method": {
			ok(`{"status":"authenticated","isAuthenticated":true,"hasAccessToken":true,"hasRefreshToken":true,"authMethod":"future"}`),
			AuthUnproven, "auth_method_unrecognized", "unknown",
		},
	} {
		t.Run(name, func(t *testing.T) {
			method, outcome, reason := classifyCursorStatus(tc.status)
			if method != tc.method || outcome != tc.want || reason != tc.reason {
				t.Fatalf("method=%q outcome=%q reason=%q", method, outcome, reason)
			}
		})
	}
}

func TestCursorAuthPreflightIgnoresIndependentClaudeSettings(t *testing.T) {
	fixture := newAuthFixture(t, "cursor-agent")
	write(t, filepath.Join(fixture.home, ".claude", "settings.json"), `{"apiKeyHelper":"/usr/local/bin/print-secret","forceLoginMethod":"console"}`)
	write(t, filepath.Join(fixture.cwd, ".claude", "settings.local.json"), `{"awsCredentialExport":"/bin/creds"}`)
	write(t, filepath.Join(fixture.home, ".cursor", "cli-config.json"), `{"version":1,"permissions":{"allow":[],"deny":[]}}`)
	runner := &fakeStatus{version: ok("2026.10.22-27dcf5f"), status: ok(cursorSubscription)}
	report := check(t, "cursor-agent", fixture, runner)
	if report.Status != AuthSubscriptionObserved || report.Method != "cursor-login" || len(report.Overrides) != 0 || !report.DispatchAllowed() {
		t.Fatalf("claude settings affected cursor: %+v", report)
	}
	if len(runner.calls) != 2 || strings.Join(runner.calls[1], " ") != fixture.executable+" status --format json" {
		t.Fatalf("cursor status command=%v", runner.calls)
	}
	wire, _ := json.Marshal(report)
	for _, secret := range []string{"print-secret", "person@example.com", "user-secret", "Private Team"} {
		if strings.Contains(string(wire), secret) {
			t.Fatalf("secret leaked: %s", wire)
		}
	}
}

func TestCursorAuthPreflightRejectsCursorAuthSources(t *testing.T) {
	cursorStatus := func() *fakeStatus {
		return &fakeStatus{version: ok("2026.10.22"), status: ok(cursorSubscription)}
	}
	t.Run("api key environment", func(t *testing.T) {
		fixture := newAuthFixture(t, "cursor-agent")
		fixture.env = append(fixture.env, "CURSOR_API_KEY=synthetic-cursor-key")
		report := check(t, "cursor-agent", fixture, cursorStatus())
		if report.Status != AuthIncompatible || report.Reason != "configuration_override" || strings.Join(report.Overrides, ",") != "cursor_env:CURSOR_API_KEY" {
			t.Fatalf("report=%+v", report)
		}
	})
	t.Run("endpoint environment", func(t *testing.T) {
		fixture := newAuthFixture(t, "cursor-agent")
		fixture.env = append(fixture.env, "CURSOR_API_ENDPOINT=https://gateway.invalid")
		report := check(t, "cursor-agent", fixture, cursorStatus())
		if report.Status != AuthIncompatible || !strings.Contains(strings.Join(report.Overrides, ","), "cursor_env:CURSOR_API_ENDPOINT") {
			t.Fatalf("report=%+v", report)
		}
	})
	t.Run("cli-config api key", func(t *testing.T) {
		fixture := newAuthFixture(t, "cursor-agent")
		write(t, filepath.Join(fixture.home, ".cursor", "cli-config.json"), `{"apiKey":"synthetic-cursor-key"}`)
		report := check(t, "cursor-agent", fixture, cursorStatus())
		if report.Status != AuthIncompatible || strings.Join(report.Overrides, ",") != "cursor_config:user:apiKey" {
			t.Fatalf("report=%+v", report)
		}
	})
	t.Run("cli-config endpoint override", func(t *testing.T) {
		fixture := newAuthFixture(t, "cursor-agent")
		write(t, filepath.Join(fixture.home, ".cursor", "cli-config.json"), `{"endpoint":"https://gateway.invalid"}`)
		report := check(t, "cursor-agent", fixture, cursorStatus())
		if report.Status != AuthIncompatible || strings.Join(report.Overrides, ",") != "cursor_config:user:endpoint" {
			t.Fatalf("report=%+v", report)
		}
	})
	t.Run("relocated CURSOR_CONFIG_DIR", func(t *testing.T) {
		fixture := newAuthFixture(t, "cursor-agent")
		relocated := filepath.Join(filepath.Dir(fixture.home), "relocated-cursor")
		write(t, filepath.Join(relocated, "cli-config.json"), `{"authToken":"synthetic-token"}`)
		fixture.env = append(fixture.env, "CURSOR_CONFIG_DIR="+relocated)
		report := check(t, "cursor-agent", fixture, cursorStatus())
		if report.Reason != "configuration_override" || strings.Join(report.Overrides, ",") != "cursor_config:user:authToken" {
			t.Fatalf("report=%+v", report)
		}
	})
	t.Run("claude-only override must not block cursor", func(t *testing.T) {
		fixture := newAuthFixture(t, "cursor-agent")
		write(t, filepath.Join(fixture.home, ".claude", "settings.json"), `{"apiKeyHelper":"x"}`)
		report := check(t, "cursor-agent", fixture, cursorStatus())
		if !report.DispatchAllowed() {
			t.Fatalf("independent claude override blocked cursor: %+v", report)
		}
	})
	t.Run("invalid cursor config fails closed", func(t *testing.T) {
		fixture := newAuthFixture(t, "cursor-agent")
		write(t, filepath.Join(fixture.home, ".cursor", "cli-config.json"), `{"apiKey":`)
		report := check(t, "cursor-agent", fixture, cursorStatus())
		if report.Status != AuthUnproven || report.Reason != "configuration_unreadable" {
			t.Fatalf("report=%+v", report)
		}
	})
}

func TestCursorAuthPreflightVendorStatusMatrix(t *testing.T) {
	for name, tc := range map[string]struct {
		status         StatusResult
		want           AuthStatus
		reason, method string
	}{
		"cursor login":               {ok(cursorSubscription), AuthSubscriptionObserved, "", "cursor-login"},
		"cursor not logged in":       {ok(`{"status":"unauthenticated","isAuthenticated":false,"hasAccessToken":false,"hasRefreshToken":false}`), AuthUnavailable, "not_logged_in", "none"},
		"cursor api key method":      {ok(`{"status":"authenticated","isAuthenticated":true,"hasAccessToken":true,"hasRefreshToken":true,"authMethod":"api_key"}`), AuthIncompatible, "api_key_login", "api_key"},
		"cursor flag only":           {ok(`{"isAuthenticated":true,"authenticated":true}`), AuthUnproven, "status_unrecognized", "unknown"},
		"cursor unsupported surface": {exit(2, "error: unrecognized"), AuthUnsupported, "status_surface_unsupported", "unknown"},
	} {
		t.Run(name, func(t *testing.T) {
			fixture := newAuthFixture(t, "cursor-agent")
			runner := &fakeStatus{version: ok("2026.10.22"), status: tc.status}
			report := check(t, "cursor-agent", fixture, runner)
			if report.Status != tc.want || report.Reason != tc.reason || report.Method != tc.method {
				t.Fatalf("report=%+v", report)
			}
		})
	}
}
