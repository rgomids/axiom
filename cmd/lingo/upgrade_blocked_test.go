package main

import (
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/install"
)

func TestUpgradeBlockedDiagnosticsStayTruthfulDuringResume(t *testing.T) {
	manual := func(next string) bool {
		return strings.Contains(next, "compatibility inspect") || strings.Contains(next, "compatibility backup") || strings.Contains(next, "compatibility export")
	}
	categories := []string{"state_inspection_failed", "state_transition_unavailable", "state_unsupported", "state_unsafe", "state_recovery_required"}
	for _, category := range categories {
		message, next := upgradeBlocked(install.Preview{Resume: true}, category)
		if strings.Contains(next, "nothing was changed") || strings.Contains(message, "before any effect") || !strings.Contains(next, "interrupted upgrade") {
			t.Fatalf("%s during resume: message=%q next=%q", category, message, next)
		}
		if manual(next) {
			t.Fatalf("%s routes to compatibility commands: %q", category, next)
		}
		message, next = upgradeBlocked(install.Preview{}, category)
		if message != "Upgrade blocked before any effect: "+category || next != upgradeNext(category) || manual(next) {
			t.Fatalf("%s fresh: message=%q next=%q", category, message, next)
		}
	}
}
