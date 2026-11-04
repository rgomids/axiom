//go:build unix

package runtimeadapter

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
)

// The frozen-invocation check reuses the Profile-v4 fixture's real bounded
// sources, but takes the frozen tuple directly: no Profile, no auth probe.
func TestCheckFrozenSelectorsRefusesInheritedOverridesWithoutLeaking(t *testing.T) {
	for _, id := range []string{"codex", "claude"} {
		t.Run(id, func(t *testing.T) {
			f := newEffortInspectorFixture(t, id)
			model := f.profile().Model
			roots := f.request.Roots
			inv := f.request.Invocation
			if err := CheckFrozenSelectors(context.Background(), id, model, RuntimeDefaultEffort, roots, inv); err != nil {
				t.Fatalf("clean runtime-default refused: %v", err)
			}
			if err := CheckFrozenSelectors(context.Background(), id, "other-model", RuntimeDefaultEffort, roots, inv); !errors.Is(err, ErrEffortConfigurationUnsafe) {
				t.Fatalf("argv model differs from frozen model: %v", err)
			}
			// runtime-default must not silently accept an explicit selector the
			// frozen snapshot did not choose.
			if err := CheckFrozenSelectors(context.Background(), id, model, "high", roots, inv); err == nil {
				t.Fatal("frozen explicit effort accepted an argv without that selector")
			}
			path := f.source(t, "user")
			content := `model_reasoning_effort = "private-canary"`
			if id == "claude" {
				content = `{"effortLevel":"private-canary"}`
			}
			if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
				t.Fatal(err)
			}
			err := CheckFrozenSelectors(context.Background(), id, model, RuntimeDefaultEffort, roots, inv)
			if !errors.Is(err, ErrEffortOverrideConflict) || strings.Contains(err.Error(), "private-canary") {
				t.Fatalf("inherited configuration selector: %v", err)
			}
			injected := inv
			injected.Argv = append(append([]string(nil), inv.Argv...), "--profile", "fast")
			if err := CheckFrozenSelectors(context.Background(), id, model, RuntimeDefaultEffort, roots, injected); !errors.Is(err, ErrEffortOverrideConflict) {
				t.Fatalf("inherited argv selector: %v", err)
			}
		})
	}
}

func TestCheckFrozenSelectorsRefusesUnknownRuntimeAndMissingExecutable(t *testing.T) {
	f := newEffortInspectorFixture(t, "codex")
	inv := f.request.Invocation
	if err := CheckFrozenSelectors(context.Background(), "stub", "m", RuntimeDefaultEffort, f.request.Roots, inv); !errors.Is(err, ErrEffortConfigurationUnsafe) {
		t.Fatalf("unknown runtime: %v", err)
	}
	missing := inv
	missing.Argv = append([]string{"/nonexistent/codex"}, inv.Argv[1:]...)
	if err := CheckFrozenSelectors(context.Background(), "codex", f.profile().Model, RuntimeDefaultEffort, f.request.Roots, missing); !errors.Is(err, ErrExecutableUnavailable) {
		t.Fatalf("missing executable: %v", err)
	}
}
