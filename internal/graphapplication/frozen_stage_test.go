package graphapplication

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/executiongraph"
	"github.com/rgomids/axiom/internal/runtimeadapter"
	"github.com/rgomids/axiom/internal/runtimeapplication"
	"github.com/rgomids/axiom/internal/runtimeprofile"
)

type frozenChecker struct {
	current *runtimeprofile.ResolvedProfile
	err     error
	calls   int
}

func (f *frozenChecker) Check(context.Context, runtimeapplication.Preview) (runtimeapplication.Binding, error) {
	f.calls++
	return runtimeapplication.Binding{Choice: runtimeprofile.Choice{ResolvedProfile: f.current}}, f.err
}

type frozenSources struct {
	digest     string
	err        error
	secondErr  error
	calls      int
	invocation executiongraph.Invocation
}

func (f *frozenSources) CheckEffortSources(_ context.Context, _ runtimeprofile.ResolvedProfile, inv executiongraph.Invocation) (string, error) {
	f.calls++
	f.invocation = inv
	if f.calls == 2 && f.secondErr != nil {
		return "", f.secondErr
	}
	return f.digest, f.err
}
func frozenFixture(id, mode string) (runtimeadapter.StageInvocationRequest, runtimeprofile.ResolvedProfile, runtimeapplication.Preview) {
	hash := strings.Repeat("a", 64)
	version := "0.159.1"
	if id == "claude" {
		version = "2.1.296"
	}
	effort := runtimeprofile.Effort{Mode: mode, Value: "default"}
	if mode == "explicit" {
		effort.Value = "high"
	}
	p := runtimeprofile.ResolvedProfile{ProfileID: "profile", RuntimeID: id, Model: "model", Effort: effort, ProfileDigest: hash, ConfigurationRevision: 1, ConfigurationDigest: hash, ExecutableDigest: hash, RuntimeVersion: version, InvocationMode: "noninteractive", ObservationRevision: 1, ObservationDigest: hash, EffortSourceDigest: hash, CapabilityProofReference: "proof"}
	r := runtimeadapter.StageInvocationRequest{Profile: runtimeadapter.CommandProfile{RuntimeID: id, ModelProfileID: p.ProfileID, Executable: "/fake/" + id, Model: p.Model}, Version: version, Workspace: "/workspace", Inbox: "/inbox", AxiomPath: "/fake/axiom"}
	return r, p, runtimeapplication.Preview{Choice: &runtimeprofile.Choice{ResolvedProfile: &p}}
}
func TestFrozenStageEffortAndPerAttemptChecks(t *testing.T) {
	for _, id := range []string{"codex", "claude"} {
		for _, mode := range []string{"explicit", "runtime-default"} {
			t.Run(id+mode, func(t *testing.T) {
				r, p, preview := frozenFixture(id, mode)
				checker := &frozenChecker{current: &p}
				sources := &frozenSources{digest: p.EffortSourceDigest}
				auth := &stageAuthFake{status: runtimeadapter.AuthSubscriptionObserved, version: p.RuntimeVersion}
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
					if !reflect.DeepEqual(auth.target.Environment, executiongraph.EffectiveInvocationEnvironment(inv)) {
						t.Fatal("auth environment differs")
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
	r, p, preview := frozenFixture("codex", "explicit")
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
			auth := &stageAuthFake{status: runtimeadapter.AuthSubscriptionObserved, version: p.RuntimeVersion}
			inv, _, err := PrepareFrozenStageInvocation(context.Background(), r, p, preview, &frozenChecker{current: &current}, sources, auth)
			if !errors.Is(err, runtimeapplication.ErrStale) || len(inv.Argv) != 0 || sources.calls != 0 || auth.target.Executable != "" {
				t.Fatal("drift reached dispatch", err)
			}
		})
	}
}
func TestFrozenStageNoProofNoFallbackAndSourceFailures(t *testing.T) {
	for _, name := range []string{"removed", "tombstoned", "edited", "disabled", "missing-proof", "fallback-default", "selector-after-preview", "unproven-sources", "source-digest-drift"} {
		t.Run(name, func(t *testing.T) {
			r, p, preview := frozenFixture("codex", "explicit")
			checker := &frozenChecker{current: &p}
			sources := &frozenSources{digest: p.EffortSourceDigest}
			auth := &stageAuthFake{status: runtimeadapter.AuthSubscriptionObserved, version: p.RuntimeVersion}
			want := runtimeapplication.ErrStale
			switch name {
			case "removed", "tombstoned", "edited", "disabled":
				checker.err = runtimeapplication.ErrStale
			case "missing-proof":
				checker.current = nil
			case "fallback-default":
				current := p
				current.Effort = runtimeprofile.Effort{Mode: "runtime-default", Value: "default"}
				checker.current = &current
			case "selector-after-preview":
				sources.err = runtimeadapter.ErrEffortOverrideConflict
				want = sources.err
			case "unproven-sources":
				sources.err = runtimeadapter.ErrEffortConfigurationUnproven
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

type frozenLocalSource struct {
	source *policySource
	err    error
}

func (s *frozenLocalSource) Load(ctx context.Context, id string) (runtimeapplication.Snapshot, error) {
	if s.err != nil {
		return runtimeapplication.Snapshot{}, s.err
	}
	return s.source.Load(ctx, id)
}

// Exercises the delivered resolver's Check with fake local inputs. No Runtime
// executable is invoked and no observation probe runs.
func TestFrozenStageCurrentLocalProfileRefusals(t *testing.T) {
	for _, name := range []string{"removed", "tombstoned", "edited", "disabled", "proof-removed", "wrong-model-proof"} {
		t.Run(name, func(t *testing.T) {
			r, _, _ := frozenFixture("codex", "runtime-default")
			directory := t.TempDir()
			exe := filepath.Join(directory, "codex")
			// Identity inspection requires a runnable file; no fixture executes it.
			if err := os.WriteFile(exe, []byte("inert fixture; never execute"), 0700); err != nil {
				t.Fatal(err)
			}
			graph := buildGraph(t, canonicalTempDir(t))
			_, previews, source := currentPolicyFixture(t, &graph, "codex", exe, "")
			localSource := &frozenLocalSource{source: source}
			policy := runtimeapplication.New(localSource)
			child := graph.Children[0]
			preview := previews[child.ExecutionID]
			p := *preview.Choice.ResolvedProfile
			// Match the stage builder's reviewed version through a fake observation.
			observation, err := source.snapshot.Observer.Observe(context.Background(), "codex")
			if err != nil {
				t.Fatal(err)
			}
			observation.Version = "0.159.1"
			observation.ProfileProofs[0].RuntimeVersion = observation.Version
			inventory, err := runtimeadapter.NewInventory([]runtimeprofile.Observation{observation})
			if err != nil {
				t.Fatal(err)
			}
			source.snapshot.Observer = inventory
			preview, err = policy.Preview(context.Background(), preview.ProjectID, preview.Request)
			if err != nil {
				t.Fatal(err)
			}
			p = *preview.Choice.ResolvedProfile
			r.Profile = runtimeadapter.CommandProfile{RuntimeID: p.RuntimeID, ModelProfileID: p.ProfileID, Model: p.Model, Executable: exe}
			r.Version = p.RuntimeVersion
			auth := &stageAuthFake{status: runtimeadapter.AuthSubscriptionObserved, version: p.RuntimeVersion}
			switch name {
			case "removed":
				source.snapshot.Configuration.ModelProfiles = source.snapshot.Configuration.ModelProfiles[1:]
			case "tombstoned":
				localSource.err = runtimeapplication.LocalStateError{State: "retired"}
			case "edited":
				source.snapshot.Configuration.ModelProfiles[0].Model = "changed-model"
			case "disabled":
				source.snapshot.Configuration.Runtimes[0].Enabled = false
			case "proof-removed", "wrong-model-proof":
				if name == "proof-removed" {
					observation.ProfileProofs = nil
				} else {
					observation.ProfileProofs[0].Model = "other-model"
				}
				inventory, err = runtimeadapter.NewInventory([]runtimeprofile.Observation{observation})
				if err != nil {
					t.Fatal(err)
				}
				source.snapshot.Observer = inventory
			}
			inv, _, err := PrepareFrozenStageInvocation(context.Background(), r, p, preview, policy, &frozenSources{digest: p.EffortSourceDigest}, auth)
			if err == nil || len(inv.Argv) != 0 || auth.target.Executable != "" {
				t.Fatal("current local drift dispatched", err)
			}
			if *preview.Choice.ResolvedProfile != p {
				t.Fatal("frozen tuple replaced")
			}
		})
	}
}

func TestFrozenStageSourceChangeDuringAuthenticationAndRedaction(t *testing.T) {
	for _, failure := range []error{runtimeadapter.ErrEffortOverrideConflict, errors.New("private raw configuration text")} {
		r, p, preview := frozenFixture("codex", "runtime-default")
		sources := &frozenSources{digest: p.EffortSourceDigest, secondErr: failure}
		auth := &stageAuthFake{status: runtimeadapter.AuthSubscriptionObserved, version: p.RuntimeVersion}
		inv, _, err := PrepareFrozenStageInvocation(context.Background(), r, p, preview, &frozenChecker{current: &p}, sources, auth)
		if err == nil || len(inv.Argv) != 0 || auth.target.Executable == "" || sources.calls != 2 || strings.Contains(err.Error(), "private raw") {
			t.Fatal("late source change or redaction failed", err)
		}
		want := runtimeadapter.ErrEffortConfigurationUnproven
		if errors.Is(failure, runtimeadapter.ErrEffortOverrideConflict) {
			want = failure
		}
		if !errors.Is(err, want) {
			t.Fatal(err)
		}
	}
}
