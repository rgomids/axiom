package runtimeadapter

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

type operationalStatus struct {
	run func(context.Context, StatusCommand) StatusResult
}

func (operationalStatus) EvidenceKind() string { return EvidenceFake }
func (s operationalStatus) RunStatus(ctx context.Context, command StatusCommand) StatusResult {
	return s.run(ctx, command)
}

func TestOperationalAuthVersionIsOnlyDiagnosticMetadata(t *testing.T) {
	for _, runtimeID := range []string{"codex", "claude"} {
		for name, version := range map[string]StatusResult{
			"arbitrary":    ok("v9999.99.99"),
			"unrecognized": ok("vendor development build"),
			"empty":        ok(""),
			"nonzero":      exit(2, "unsupported version option"),
		} {
			t.Run(runtimeID+"/"+name, func(t *testing.T) {
				fixture := newAuthFixture(t, runtimeID)
				status := "Logged in using ChatGPT"
				if runtimeID == "claude" {
					status = claudeSubscription
				}
				runner := &fakeStatus{version: version, status: ok(status)}
				preflight, err := NewAuthPreflight(runner, ManagedConfiguration{})
				if err != nil {
					t.Fatal(err)
				}
				report := preflight.Check(context.Background(), fixture.target(runtimeID))
				if report.Status != AuthSubscriptionObserved || !report.DispatchAllowed() || report.Usability != UsabilityUnproven || len(runner.calls) != 2 {
					t.Fatalf("auth depended on version: %+v calls=%d", report, len(runner.calls))
				}
				if name != "arbitrary" && report.Version != "unknown" {
					t.Fatal("failed/unrecognized version became identity", report.Version)
				}
				want := []string{fixture.executable, "login", "status"}
				if runtimeID == "claude" {
					want = []string{fixture.executable, "auth", "status", "--json"}
				}
				if strings.Join(runner.calls[1], "\x00") != strings.Join(want, "\x00") {
					t.Fatal("status invocation changed", runner.calls[1])
				}
			})
		}
	}
}

func TestOperationalAuthPreservesConcreteFailureBoundaries(t *testing.T) {
	for _, runtimeID := range []string{"codex", "claude"} {
		for _, scenario := range []string{"version-timeout", "version-cancelled", "version-drift", "version-disappeared", "status-cancelled", "status-drift", "status-unrecognized", "not-logged-in", "api-override", "unsafe-configuration"} {
			t.Run(runtimeID+"/"+scenario, func(t *testing.T) {
				fixture := newAuthFixture(t, runtimeID)
				target := fixture.target(runtimeID)
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				calls := 0
				status := "Logged in using ChatGPT"
				if runtimeID == "claude" {
					status = claudeSubscription
				}
				runner := operationalStatus{run: func(_ context.Context, command StatusCommand) StatusResult {
					calls++
					if command.Argv[1] == "--version" {
						switch scenario {
						case "version-timeout":
							return StatusResult{Started: true, TimedOut: true, ExitCode: -1}
						case "version-cancelled":
							cancel()
						case "version-drift":
							if err := os.WriteFile(fixture.executable, []byte("changed synthetic executable"), 0o700); err != nil {
								t.Fatal(err)
							}
						case "version-disappeared":
							if err := os.Remove(fixture.executable); err != nil {
								t.Fatal(err)
							}
						}
						return ok("unrecognized version")
					}
					switch scenario {
					case "status-cancelled":
						cancel()
					case "status-drift":
						if err := os.WriteFile(fixture.executable, []byte("changed synthetic executable"), 0o700); err != nil {
							t.Fatal(err)
						}
					case "status-unrecognized":
						return ok("future ambiguous login status")
					case "not-logged-in":
						if runtimeID == "codex" {
							return exit(1, "Not logged in")
						}
						return exit(1, `{"loggedIn":false,"authMethod":"none","apiProvider":"firstParty"}`)
					}
					return ok(status)
				}}
				wantCalls := 2
				switch scenario {
				case "api-override":
					variable := "OPENAI_API_KEY"
					if runtimeID == "claude" {
						variable = "ANTHROPIC_API_KEY"
					}
					target.Environment = append(target.Environment, variable+"=synthetic-override")
					wantCalls = 0
				case "unsafe-configuration":
					path := fixture.home + "/.codex/config.toml"
					content := `forced_login_method = "api"`
					if runtimeID == "claude" {
						path, content = fixture.home+"/.claude/settings.json", `{"apiKeyHelper":"synthetic-helper"}`
					}
					write(t, path, content)
					wantCalls = 0
				case "version-timeout", "version-cancelled", "version-drift", "version-disappeared":
					wantCalls = 1
				}
				preflight, err := NewAuthPreflight(runner, ManagedConfiguration{})
				if err != nil {
					t.Fatal(err)
				}
				report := preflight.Check(ctx, target)
				if report.DispatchAllowed() || report.Reason == "" || calls != wantCalls {
					t.Fatalf("boundary failure: %+v calls=%d want=%d", report, calls, wantCalls)
				}
			})
		}
	}
}

func TestOperationalAuthDoesNotExposeStatusOrVersionPayloads(t *testing.T) {
	for _, runtimeID := range []string{"codex", "claude"} {
		fixture := newAuthFixture(t, runtimeID)
		payload := "private-transcript /private/host/path account@example.invalid synthetic-sensitive-value"
		runner := &fakeStatus{version: exit(2, payload), status: ok(payload)}
		preflight, _ := NewAuthPreflight(runner, ManagedConfiguration{})
		report := preflight.Check(context.Background(), fixture.target(runtimeID))
		wire, err := json.Marshal(report)
		if err != nil || report.DispatchAllowed() {
			t.Fatal(report, err)
		}
		for _, value := range strings.Fields(payload) {
			if strings.Contains(string(wire), value) {
				t.Fatal("raw subprocess payload entered report", value)
			}
		}
	}
}
