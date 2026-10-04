package main

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/install"
)

func TestWindowsInstallReleasePartialCause(t *testing.T) {
	for _, tc := range []struct {
		name       string
		err        error
		diagnostic string
		recovery   string
		receipt    bool
	}{
		{"refresh_required", &install.Error{Category: "skill_receipt_refresh_required"}, "did not run as the candidate release", "axiom first-run", true},
		{"conflict", fmt.Errorf("wrapped: %w", &install.Error{Category: "skill_receipt_conflict"}), "was not written by Axiom and was preserved", "axiom runtime codex status", true},
		{"marker_cleanup_failed", &install.Error{Category: "marker_cleanup_failed"}, "upgrade: marker_cleanup_failed", "", false},
		{"marker_unavailable", &install.Error{Category: "marker_unavailable"}, "upgrade: marker_unavailable", "", false},
		{"target_changed", &install.Error{Category: "target_changed"}, "upgrade: target_changed", "", false},
		{"cancelled", context.Canceled, "context canceled", "", false},
		{"unknown_receipt_category", &install.Error{Category: "skill_receipt_unknown"}, "upgrade: skill_receipt_unknown", "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out, stderr bytes.Buffer
			handled, code := windowsInstallReleaseResult("partial", tc.err, install.Target{}, &out, &stderr)
			if !handled || code != 1 || out.String() != "install_status=partial\n" {
				t.Fatalf("handled=%v code=%d stdout=%q", handled, code, out.String())
			}
			diagnostic := stderr.String()
			if !strings.Contains(diagnostic, tc.diagnostic) || (tc.recovery != "" && !strings.Contains(diagnostic, tc.recovery)) {
				t.Fatalf("cause/recovery lost: %q", diagnostic)
			}
			if !tc.receipt && (strings.Contains(diagnostic, "Codex skill-set receipt") || strings.Contains(diagnostic, "axiom first-run")) {
				t.Fatalf("unrelated partial misdiagnosed: %q", diagnostic)
			}
		})
	}
}
