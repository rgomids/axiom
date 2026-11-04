//go:build linux || darwin

package stageexecution

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/runtimeadapter"
	"github.com/rgomids/axiom/internal/workflow"
)

func TestI369TypedLaunchFailure(t *testing.T) {
	for _, c := range []struct {
		err    error
		reason string
	}{
		{errors.Join(runtimeadapter.ErrEffortConfigurationUnsafe, runtimeadapter.ErrEffortConfigurationInvalid), "configuration_invalid"},
		{errors.Join(runtimeadapter.ErrEffortConfigurationInvalid, runtimeadapter.ErrEffortConfigurationUnsafe), "configuration_invalid"},
		{runtimeadapter.ErrEffortOverrideConflict, "selector_conflict"},
		{runtimeadapter.ErrEffortConfigurationUnsafe, "configuration_unsafe"},
		{runtimeadapter.ErrExecutableUnavailable, "executable_unavailable"},
		{&os.PathError{Op: "open", Path: "/private/canary", Err: os.ErrPermission}, "permission_denied"},
		{errors.New("private-canary"), "builder_failed"},
	} {
		t.Run(c.reason, func(t *testing.T) {
			_, err := refuseLaunch(context.Background(), c.err, "selector_inspection", "")
			if !errors.Is(err, ErrRuntimeLaunch) || !errors.Is(err, c.err) || strings.Contains(err.Error(), "canary") {
				t.Fatal("unsafe cause handling", err)
			}
			f := newEngineFixture(t, twoStages(false), succeed(""))
			f.engine.Launcher = stubLauncher{buildErr: err}
			x, e := f.run(t)
			if e != nil {
				t.Fatal(e)
			}
			a := x.Stages[0].Tasks[0].Attempts[0]
			if a.LaunchFailure == nil || a.LaunchFailure.Reason != c.reason || a.LaunchFailure.Phase != "selector_inspection" || f.runner.spawns.Load() != 0 {
				t.Fatal("failure lost", a.LaunchFailure)
			}
			wire, e := workflow.EncodeExecution(x)
			if e != nil || strings.Contains(string(wire), "canary") {
				t.Fatal("unsafe persistence", e)
			}
			after, _, e := f.engine.Reconcile(context.Background(), x.ProjectID, x.ExecutionID)
			if e != nil {
				t.Fatal(e)
			}
			back := after.Stages[0].Tasks[0].Attempts[0]
			if *back.LaunchFailure != *a.LaunchFailure {
				t.Fatal("reconcile changed failure")
			}
		})
	}
}
func TestI369LaunchCancellation(t *testing.T) {
	l, r := runtimeLauncherFixture(t, "codex", "runtime-default")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := l.Build(ctx, r); !errors.Is(err, context.Canceled) {
		t.Fatal("cancellation collapsed", err)
	}
	f := newEngineFixture(t, twoStages(false), succeed(""))
	f.engine.Launcher = stubLauncher{buildErr: context.Canceled}
	x, err := f.run(t)
	if err != nil || x.Stages[0].Tasks[0].Attempts[0].Status != workflow.AttemptCancelled || f.runner.spawns.Load() != 0 {
		t.Fatal("cancellation not retained", err)
	}
}
func TestI369SourceClass(t *testing.T) {
	f := launchFailure(&runtimeadapter.SourceError{Source: "user", Cause: runtimeadapter.ErrEffortConfigurationInvalid}, "selector_inspection")
	wire, _ := json.Marshal(f)
	if f.Source != "user" || !f.Valid() || strings.Contains(string(wire), "Cause") {
		t.Fatal("source projection", f)
	}
}
