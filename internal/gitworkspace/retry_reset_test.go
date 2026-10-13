package gitworkspace

import (
	"context"
	"errors"
	"testing"

	"github.com/rgomids/axiom/internal/executiongraph"
)

// The fallback deliberately accepts no authority claim. Even successful
// validation cannot reach an effect, so revocation between effects and partial
// reset are impossible. This does not claim to test an implemented guard.
func TestExplicitResetFailClosedBeforeEffects(t *testing.T) {
	for _, name := range []string{"validation-success", "validation-failure", "missing-callbacks"} {
		t.Run(name, func(t *testing.T) {
			checks, effects := 0, 0
			validate := func(bool) error {
				checks++
				if name == "validation-failure" {
					return errors.New("ownership changed")
				}
				return nil
			}
			run := func(context.Context, string, ...string) error { effects++; return nil }
			if name == "missing-callbacks" {
				validate, run = nil, nil
			}
			err := applyWorkspaceReset(context.Background(), "/synthetic/owned-worktree", "0123456789012345678901234567890123456789", validate, run)
			if !errors.Is(err, ErrRetryResetFencingUnavailable) || checks != 0 || effects != 0 {
				t.Fatalf("err=%v checks=%d effects=%d", err, checks, effects)
			}
		})
	}
}

func TestResetWorkspaceFailClosedWithoutManager(t *testing.T) {
	// A nil receiver proves refusal precedes manager inspection or Git access.
	var manager *Manager
	for i := 0; i < 2; i++ {
		if err := manager.ResetWorkspace(context.Background(), executiongraph.ChildExecution{}); !errors.Is(err, ErrRetryResetFencingUnavailable) {
			t.Fatal(err)
		}
	}
}
