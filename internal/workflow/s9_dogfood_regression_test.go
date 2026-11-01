package workflow

import (
	"context"
	"testing"
)

// Regressions for the two Major findings of the S9 dogfood against
// v0.1.2-rc.1: the Execution Runtime was always persisted as codex, and the
// first projection of an Issue without any axiom:stage:* marker required
// recovery.

func TestS9StartPersistsTheExplicitlySelectedRuntime(t *testing.T) {
	for _, runtimeID := range []string{"claude", "codex"} {
		t.Run(runtimeID, func(t *testing.T) {
			service, store, _ := newS4Service(t)
			target := s4Target()
			target.RuntimeID = runtimeID
			started := service.Start(context.Background(), target)
			if started.Category != "execution_started" || started.State.RuntimeID != runtimeID || store.state.RuntimeID != runtimeID {
				t.Fatalf("start = %#v persisted=%q", started, store.state.RuntimeID)
			}
		})
	}
}

func TestS9StartRefusesUnsupportedRuntimeWithoutCreatingExecution(t *testing.T) {
	for name, runtimeID := range map[string]string{"empty": "", "unknown": "gemini", "case": "Claude", "space": " claude"} {
		t.Run(name, func(t *testing.T) {
			service, store, _ := newS4Service(t)
			target := s4Target()
			target.RuntimeID = runtimeID
			refused := service.Start(context.Background(), target)
			if refused.Status != ValidationFailed || refused.Category != "invalid_execution_input" || store.created {
				t.Fatalf("start = %#v created=%v", refused, store.created)
			}
		})
	}
}

func TestS9ExistingExecutionRuntimeCannotBeReassigned(t *testing.T) {
	for _, pair := range [][2]string{{"claude", "codex"}, {"codex", "claude"}} {
		t.Run(pair[0]+"_to_"+pair[1], func(t *testing.T) {
			service, store, _ := newS4Service(t)
			original := s4Target()
			original.RuntimeID = pair[0]
			started := service.Start(context.Background(), original)
			other := s4Target()
			other.RuntimeID = pair[1]
			restart := service.Start(context.Background(), other)
			advance := service.Transition(context.Background(), other, TransitionInput{ExpectedRevision: 1, Stage: Intake, Outcome: OutcomePassed})
			for name, got := range map[string]Result{"start": restart, "advance": advance} {
				if got.Status != ValidationFailed || got.Category != "execution_scope_conflict" {
					t.Fatalf("%s = %#v", name, got)
				}
			}
			if store.state.RuntimeID != pair[0] || store.state.Revision != started.State.Revision {
				t.Fatalf("persisted Execution changed: %#v", store.state)
			}
			same := service.Start(context.Background(), original)
			if same.Category != "execution_already_started" || same.State.RuntimeID != pair[0] {
				t.Fatalf("idempotent start = %#v", same)
			}
		})
	}
}

// newIssueObservation reproduces an Issue created by `work-item create`: no
// axiom:stage:* marker on the Issue and no stage label in the repository.
