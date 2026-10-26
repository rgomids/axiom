package graphapplication

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/rgomids/axiom/internal/executiongraph"
	"github.com/rgomids/axiom/internal/project"
	"github.com/rgomids/axiom/internal/runtimeadapter"
	"github.com/rgomids/axiom/internal/runtimeapplication"
	"github.com/rgomids/axiom/internal/runtimeprofile"
)

type frozenChecker struct {
	current *runtimeprofile.ResolvedProfileV4
	err     error
	calls   int
}

func (f *frozenChecker) Check(context.Context, runtimeapplication.Preview) (runtimeapplication.Binding, error) {
	f.calls++
	return runtimeapplication.Binding{Choice: runtimeprofile.Choice{FrozenProfile: f.current}}, f.err
}

type frozenSources struct {
	digest     string
	err        error
	secondErr  error
	calls      int
	invocation executiongraph.Invocation
}

func (f *frozenSources) CheckEffortSources(_ context.Context, _ runtimeprofile.ResolvedProfileV4, inv executiongraph.Invocation) (string, error) {
	f.calls++
	f.invocation = inv
	if f.calls == 2 && f.secondErr != nil {
		return "", f.secondErr
	}
	return f.digest, f.err
}

type stageAuthFake struct {
	scope   string
	target  runtimeadapter.AuthTarget
	status  runtimeadapter.AuthStatus
	reason  string
	version string
}

func (f *stageAuthFake) Check(_ context.Context, target runtimeadapter.AuthTarget) runtimeadapter.AuthReport {
	f.target = target
	scope := f.scope
	if scope == "" {
		scope = runtimeadapter.AuthScopeInvocation
	}
	return runtimeadapter.AuthReport{Scope: scope, RuntimeID: target.RuntimeID, Status: f.status, Reason: f.reason, Version: f.version, ExecutableDigest: target.ExpectedDigest, EvidenceKind: runtimeadapter.EvidenceFake}
}

func frozenFixture(t *testing.T, id, mode string) (runtimeadapter.StageInvocationRequest, runtimeprofile.ResolvedProfileV4, runtimeapplication.Preview) {
	t.Helper()
	executable := filepath.Join(t.TempDir(), id)
	if err := os.WriteFile(executable, []byte("inert fixture; never execute"), 0o700); err != nil {
		t.Fatal(err)
	}
	identity, err := runtimeadapter.ExecutableIdentity(executable)
	if err != nil {
		t.Fatal(err)
	}
	hash := strings.Repeat("a", 64)
	effort := runtimeprofile.Effort{Mode: mode, Value: "default"}
	if mode == "explicit" {
		effort.Value = "high"
	}
	p := runtimeprofile.ResolvedProfileV4{ProfileID: "profile", RuntimeID: id, Model: "model", Effort: effort, ProfileDigest: hash, ConfigurationRevision: 1, ConfigurationDigest: hash, ExecutableDigest: identity, InvocationMode: "noninteractive", ObservationRevision: 1, ObservationDigest: hash, EffortSourceDigest: hash}
	r := runtimeadapter.StageInvocationRequest{Profile: runtimeadapter.CommandProfile{RuntimeID: id, ModelProfileID: p.ProfileID, Executable: executable, Model: p.Model}, Workspace: "/workspace", Inbox: "/inbox", AxiomPath: "/fake/axiom"}
	return r, p, runtimeapplication.Preview{Choice: &runtimeprofile.Choice{FrozenProfile: &p}}
}

func TestFrozenStageEffortAndPerAttemptChecks(t *testing.T) {
	for _, id := range []string{"codex", "claude"} {
		for _, mode := range []string{"explicit", "runtime-default"} {
			t.Run(id+mode, func(t *testing.T) {
				r, p, preview := frozenFixture(t, id, mode)
				checker := &frozenChecker{current: &p}
				sources := &frozenSources{digest: p.EffortSourceDigest}
				auth := &stageAuthFake{status: runtimeadapter.AuthSubscriptionObserved, version: "unknown"}
				for attempt := 0; attempt < 2; attempt++ {
					inv, _, err := PrepareFrozenStageInvocation(context.Background(), r, p, preview, checker, sources, auth)
					if err != nil {
						t.Fatal(err)
					}
					argv := strings.Join(inv.Argv, " ")
					env := strings.Join(inv.Env, " ")
					hasEffort := strings.Contains(argv, "model_reasoning_effort=high") || strings.Contains(env, "CLAUDE_CODE_EFFORT_LEVEL=high")
					if hasEffort != (mode == "explicit") {
						t.Fatalf("effort encoding: %v %v", inv.Argv, inv.Env)
					}
					if !reflect.DeepEqual(auth.target.Environment, executiongraph.EffectiveInvocationEnvironment(inv)) || auth.target.ExpectedDigest != p.ExecutableDigest {
						t.Fatal("auth did not inspect the exact invocation")
					}
				}
				if checker.calls != 2 || sources.calls != 4 {
					t.Fatal("checks not repeated before each attempt")
				}
			})
		}
	}
}

func TestFrozenStageEveryTupleElementDrift(t *testing.T) {
	r, p, preview := frozenFixture(t, "codex", "explicit")
	typ := reflect.TypeOf(p)
	for n := 0; n < typ.NumField(); n++ {
		t.Run(typ.Field(n).Name, func(t *testing.T) {
			current := p
			field := reflect.ValueOf(&current).Elem().Field(n)
			switch field.Kind() {
			case reflect.String:
				field.SetString(field.String() + "changed")
			case reflect.Uint64:
				field.SetUint(field.Uint() + 1)
			case reflect.Struct:
				current.Effort.Value = "low"
			}
			sources := &frozenSources{digest: p.EffortSourceDigest}
			auth := &stageAuthFake{status: runtimeadapter.AuthSubscriptionObserved}
			inv, _, err := PrepareFrozenStageInvocation(context.Background(), r, p, preview, &frozenChecker{current: &current}, sources, auth)
			if !errors.Is(err, runtimeapplication.ErrStale) || len(inv.Argv) != 0 || sources.calls != 0 || auth.target.Executable != "" {
				t.Fatal("drift reached dispatch", err)
			}
		})
	}
}

func TestFrozenStageNoFallbackAndSourceFailures(t *testing.T) {
	for _, name := range []string{"removed", "tombstoned", "edited", "disabled", "missing-tuple", "fallback-default", "selector-after-preview", "unsafe-sources", "source-digest-drift"} {
		t.Run(name, func(t *testing.T) {
			r, p, preview := frozenFixture(t, "codex", "explicit")
			checker := &frozenChecker{current: &p}
			sources := &frozenSources{digest: p.EffortSourceDigest}
			auth := &stageAuthFake{status: runtimeadapter.AuthSubscriptionObserved}
			want := runtimeapplication.ErrStale
			switch name {
			case "removed", "tombstoned", "edited", "disabled":
				checker.err = runtimeapplication.ErrStale
			case "missing-tuple":
				checker.current = nil
			case "fallback-default":
				current := p
				current.Effort = runtimeprofile.Effort{Mode: "runtime-default", Value: "default"}
				checker.current = &current
			case "selector-after-preview":
				sources.err = runtimeadapter.ErrEffortOverrideConflict
				want = sources.err
			case "unsafe-sources":
				sources.err = runtimeadapter.ErrEffortConfigurationUnsafe
				want = sources.err
			case "source-digest-drift":
				sources.digest = strings.Repeat("b", 64)
			}
			inv, _, err := PrepareFrozenStageInvocation(context.Background(), r, p, preview, checker, sources, auth)
			if !errors.Is(err, want) || len(inv.Argv) != 0 || auth.target.Executable != "" || p.Effort.Value != "high" {
				t.Fatal("refusal failed", err)
			}
		})
	}
}

func TestFrozenStageAuthenticationIsObservedNotVersioned(t *testing.T) {
	for _, id := range []string{"codex", "claude"} {
		for _, version := range []string{"unknown", "999.0.0-future", "0.0.1"} {
			r, p, preview := frozenFixture(t, id, "runtime-default")
			auth := &stageAuthFake{status: runtimeadapter.AuthSubscriptionObserved, version: version}
			if _, report, err := PrepareFrozenStageInvocation(context.Background(), r, p, preview, &frozenChecker{current: &p}, &frozenSources{digest: p.EffortSourceDigest}, auth); err != nil || report.Version != version {
				t.Fatalf("%s %s refused: %v", id, version, err)
			}
		}
		for _, status := range []runtimeadapter.AuthStatus{runtimeadapter.AuthUnavailable, runtimeadapter.AuthUnproven, runtimeadapter.AuthIncompatible, runtimeadapter.AuthUnsupported} {
			r, p, preview := frozenFixture(t, id, "runtime-default")
			auth := &stageAuthFake{status: status, reason: "not_logged_in"}
			inv, report, err := PrepareFrozenStageInvocation(context.Background(), r, p, preview, &frozenChecker{current: &p}, &frozenSources{digest: p.EffortSourceDigest}, auth)
			if !errors.Is(err, executiongraph.ErrAuthenticationBlocked) || len(inv.Argv) != 0 || report.Status != status || !strings.Contains(err.Error(), "not_logged_in") {
				t.Fatalf("%s %s dispatched: %v", id, status, err)
			}
		}
		r, p, preview := frozenFixture(t, id, "runtime-default")
		auth := &stageAuthFake{status: runtimeadapter.AuthUnproven, reason: "executable_identity_changed"}
		if _, _, err := PrepareFrozenStageInvocation(context.Background(), r, p, preview, &frozenChecker{current: &p}, &frozenSources{digest: p.EffortSourceDigest}, auth); !errors.Is(err, runtimeapplication.ErrStale) {
			t.Fatal("identity drift during status was not stale", err)
		}
		credentialed := r
		credentialed.Profile.CredentialReference = "env:API_KEY"
		auth = &stageAuthFake{status: runtimeadapter.AuthSubscriptionObserved}
		if _, _, err := PrepareFrozenStageInvocation(context.Background(), credentialed, p, preview, &frozenChecker{current: &p}, &frozenSources{digest: p.EffortSourceDigest}, auth); !errors.Is(err, executiongraph.ErrAuthenticationBlocked) || auth.target.Executable != "" {
			t.Fatal("credential reference reached authentication", err)
		}
		if _, _, err := PrepareFrozenStageInvocation(context.Background(), r, p, preview, &frozenChecker{current: &p}, &frozenSources{digest: p.EffortSourceDigest}, nil); !errors.Is(err, executiongraph.ErrAuthenticationBlocked) {
			t.Fatal("missing authenticator dispatched", err)
		}
	}
}

func TestFrozenStageExecutableIdentity(t *testing.T) {
	r, p, preview := frozenFixture(t, "claude", "explicit")
	if err := os.WriteFile(r.Profile.Executable, []byte("replaced; never execute"), 0o700); err != nil {
		t.Fatal(err)
	}
	auth := &stageAuthFake{status: runtimeadapter.AuthSubscriptionObserved}
	if _, _, err := PrepareFrozenStageInvocation(context.Background(), r, p, preview, &frozenChecker{current: &p}, &frozenSources{digest: p.EffortSourceDigest}, auth); !errors.Is(err, runtimeapplication.ErrStale) || auth.target.Executable != "" {
		t.Fatal("replaced executable dispatched", err)
	}
	if err := os.Remove(r.Profile.Executable); err != nil {
		t.Fatal(err)
	}
	if _, _, err := PrepareFrozenStageInvocation(context.Background(), r, p, preview, &frozenChecker{current: &p}, &frozenSources{digest: p.EffortSourceDigest}, auth); !errors.Is(err, runtimeadapter.ErrExecutableUnavailable) || auth.target.Executable != "" {
		t.Fatal("missing executable dispatched", err)
	}
}

func TestFrozenStageSourceChangeDuringAuthenticationAndRedaction(t *testing.T) {
	for _, failure := range []error{runtimeadapter.ErrEffortOverrideConflict, errors.New("private raw configuration text")} {
		r, p, preview := frozenFixture(t, "codex", "runtime-default")
		sources := &frozenSources{digest: p.EffortSourceDigest, secondErr: failure}
		auth := &stageAuthFake{status: runtimeadapter.AuthSubscriptionObserved}
		inv, _, err := PrepareFrozenStageInvocation(context.Background(), r, p, preview, &frozenChecker{current: &p}, sources, auth)
		if err == nil || len(inv.Argv) != 0 || auth.target.Executable == "" || sources.calls != 2 || strings.Contains(err.Error(), "private raw") {
			t.Fatal("late source change or redaction failed", err)
		}
		want := runtimeadapter.ErrEffortConfigurationUnsafe
		if errors.Is(failure, runtimeadapter.ErrEffortOverrideConflict) {
			want = failure
		}
		if !errors.Is(err, want) {
			t.Fatal(err)
		}
	}
}

type frozenLocalSource struct {
	project  project.Project
	config   runtimeprofile.ConfigurationV4
	observed runtimeprofile.OperationalObservation
	err      error
}

func (s *frozenLocalSource) Load(context.Context, string) (runtimeapplication.Snapshot, error) {
	if s.err != nil {
		return runtimeapplication.Snapshot{}, s.err
	}
	consumer, _ := runtimeprofile.NewConsumerV4(s.config)
	return runtimeapplication.Snapshot{Project: s.project, Consumer: consumer, Observer: frozenObserver{s.observed}}, nil
}

type frozenObserver struct {
	observed runtimeprofile.OperationalObservation
}

func (o frozenObserver) ObserveExecutable(context.Context, string) (runtimeprofile.OperationalObservation, error) {
	v := o.observed
	v.EffortSourceDigest = ""
	return v, nil
}
func (o frozenObserver) ObserveOperational(context.Context, string, string) (runtimeprofile.OperationalObservation, error) {
	return o.observed, nil
}

func TestFrozenStageCurrentLocalProfileRefusals(t *testing.T) {
	const projectID = "12345678-1234-4abc-8def-123456789abc"
	for _, name := range []string{"removed", "tombstoned", "edited", "disabled", "effort-edited", "sources-changed"} {
		t.Run(name, func(t *testing.T) {
			r, _, _ := frozenFixture(t, "codex", "runtime-default")
			identity, err := runtimeadapter.ExecutableIdentity(r.Profile.Executable)
			if err != nil {
				t.Fatal(err)
			}
			state := project.State{SchemaVersion: 2, ID: projectID, Slug: "sample", Name: "Sample", Runtimes: project.Configured([]project.Runtime{{ID: "codex"}}), ModelProfiles: project.Configured([]project.ModelProfile{{Key: "profile", RuntimeRef: project.Configured("codex"), Model: project.Configured("model")}})}
			p, issues := project.New(state)
			if len(issues) != 0 {
				t.Fatal(issues)
			}
			source := &frozenLocalSource{project: p, config: runtimeprofile.ConfigurationV4{FormatVersion: runtimeprofile.FormatVersionV4, Revision: 1, Runtimes: []runtimeprofile.RuntimeV4{{ID: "codex", Adapter: "codex", Enabled: true, AllowlistedProfileIDs: []string{"profile"}}}, ModelProfiles: []runtimeprofile.ModelProfile{{ID: "profile", RuntimeID: "codex", Model: "model", Effort: runtimeprofile.Effort{Mode: "runtime-default", Value: "default"}, Capabilities: []string{"read"}, Complexities: []string{"low"}}}},
				observed: runtimeprofile.OperationalObservation{RuntimeID: "codex", Adapter: "codex", Installed: true, Available: true, ExecutableDigest: identity, InvocationMode: "noninteractive", Revision: 1, EffortSourceDigest: strings.Repeat("e", 64), ObservedAt: time.Unix(1, 0)}}
			policy := runtimeapplication.New(source)
			preview, err := policy.Preview(context.Background(), projectID, runtimeapplication.Request{Role: "implementation", Complexity: "low", Capabilities: []string{"read"}})
			if err != nil || preview.Choice == nil || preview.Choice.FrozenProfile == nil {
				t.Fatal(preview, err)
			}
			frozen := *preview.Choice.FrozenProfile
			auth := &stageAuthFake{status: runtimeadapter.AuthSubscriptionObserved}
			if _, _, err := PrepareFrozenStageInvocation(context.Background(), r, frozen, preview, policy, &frozenSources{digest: frozen.EffortSourceDigest}, auth); err != nil {
				t.Fatal("unchanged local state refused", err)
			}
			auth = &stageAuthFake{status: runtimeadapter.AuthSubscriptionObserved}
			switch name {
			case "removed":
				source.config.ModelProfiles = nil
				source.config.Runtimes[0].AllowlistedProfileIDs = nil
			case "tombstoned":
				source.err = runtimeapplication.LocalStateError{State: "retired"}
			case "edited":
				source.config.ModelProfiles[0].Model = "changed-model"
			case "disabled":
				source.config.Runtimes[0].Enabled = false
			case "effort-edited":
				source.config.ModelProfiles[0].Effort = runtimeprofile.Effort{Mode: "explicit", Value: "high"}
			case "sources-changed":
				source.observed.EffortSourceDigest = strings.Repeat("f", 64)
			}
			inv, _, err := PrepareFrozenStageInvocation(context.Background(), r, frozen, preview, policy, &frozenSources{digest: frozen.EffortSourceDigest}, auth)
			if err == nil || len(inv.Argv) != 0 || auth.target.Executable != "" {
				t.Fatal("current local drift dispatched", err)
			}
			if *preview.Choice.FrozenProfile != frozen {
				t.Fatal("frozen tuple replaced")
			}
		})
	}
}

func TestFrozenStageRejectsHostLoginReport(t *testing.T) {
	r, p, preview := frozenFixture(t, "claude", "runtime-default")
	auth := &stageAuthFake{scope: runtimeadapter.AuthScopeHost, status: runtimeadapter.AuthSubscriptionObserved}
	inv, report, err := PrepareFrozenStageInvocation(context.Background(), r, p, preview, &frozenChecker{current: &p}, &frozenSources{digest: p.EffortSourceDigest}, auth)
	if !errors.Is(err, executiongraph.ErrAuthenticationBlocked) || len(inv.Argv) != 0 || report.Scope != runtimeadapter.AuthScopeHost {
		t.Fatalf("host login admitted: report=%+v err=%v", report, err)
	}
}
