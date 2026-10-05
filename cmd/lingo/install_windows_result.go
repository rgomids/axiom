package main

import (
	"errors"
	"fmt"
	"io"
	"path/filepath"

	"github.com/rgomids/axiom/internal/install"
)

// windowsInstallReleaseResult presents the Windows installer's status and cause.
// Kept platform independent so its diagnostics can be checked on every host.
func windowsInstallReleaseResult(result install.Result, err error, target install.Target, out, stderr io.Writer) (bool, int) {
	fail := func(err error) (bool, int) { fmt.Fprintf(stderr, "install_error: %v\n", err); return true, 1 }
	status := result.Status
	if err == nil && status == "success" && result.Preservation != "" {
		fmt.Fprintf(out, "install_preserved=%s\n", result.Preservation)
		status = "upgraded"
	}
	if status != "" {
		fmt.Fprintf(out, "install_status=%s\n", status)
	}
	var installErr *install.Error
	if status == "partial" && errors.As(err, &installErr) &&
		(installErr.Category == "skill_receipt_"+install.SkillReceiptRefreshRequired ||
			installErr.Category == "skill_receipt_"+install.SkillReceiptConflict) {
		receipt := install.SkillReceiptRefreshRequired
		if installErr.Category == "skill_receipt_"+install.SkillReceiptConflict {
			receipt = install.SkillReceiptConflict
		}
		message, next := upgradeSkillReceiptPartial(install.Result{SkillReceipt: receipt})
		fmt.Fprintf(stderr, "install_error: owned upgrade partially applied: %s\ninstall_next: %s\n", message, next)
		return true, 1
	}
	if err != nil {
		return fail(err)
	}
	fmt.Fprintf(out, "installed_binary=%s\npath_notice: add %s to your PATH, then run axiom first-run\n", filepath.Join(target.BinaryDir, "axiom.exe"), target.BinaryDir)
	return true, 0
}
