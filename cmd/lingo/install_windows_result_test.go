package main

import (
	"bytes"
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/install"
	"github.com/rgomids/axiom/internal/local"
)

func TestWindowsInstallStorageGuidance(t *testing.T) {
	var out, stderr bytes.Buffer
	err := fmt.Errorf("%w: path=example rule=untrusted access", local.ErrUnsafe)
	_, code := windowsInstallReleaseResult(install.Result{}, err, install.Target{}, &out, &stderr)
	for _, want := range []string{"path=example", "rule=untrusted access", "-BinDir", "-ReceiptDir", "NTFS", "Existing ACLs are never changed"} {
		if code != 1 || out.Len() != 0 || !strings.Contains(stderr.String(), want) {
			t.Fatalf("missing %q: %s", want, stderr.String())
		}
	}
}

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
			handled, code := windowsInstallReleaseResult(install.Result{Status: "partial"}, tc.err, install.Target{}, &out, &stderr)
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

func TestWindowsInstallReleasePreservationOutput(t *testing.T) {
	for _, tc := range []struct {
		name   string
		result install.Result
		output string
	}{
		{"recognized_poc", install.Result{Status: "success", Preservation: "C:/Axiom archive/recognized-poc-digest"}, "install_preserved=C:/Axiom archive/recognized-poc-digest\ninstall_status=upgraded\n"},
		{"ordinary_upgrade", install.Result{Status: "success"}, "install_status=success\n"},
		{"fresh_install", install.Result{Status: "installed"}, "install_status=installed\n"},
		{"unchanged", install.Result{Status: "unchanged"}, "install_status=unchanged\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out, stderr bytes.Buffer
			target := install.Target{BinaryDir: "bin"}
			handled, code := windowsInstallReleaseResult(tc.result, nil, target, &out, &stderr)
			want := tc.output + "installed_binary=" + filepath.Join("bin", "axiom.exe") + "\npath_notice: add bin to your PATH, then run axiom first-run\n"
			if !handled || code != 0 || out.String() != want || stderr.Len() != 0 {
				t.Fatalf("handled=%v code=%d stdout=%q stderr=%q", handled, code, out.String(), stderr.String())
			}
		})
	}
}
