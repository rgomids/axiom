//go:build !windows

package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/rgomids/axiom/internal/compatibility"
	"github.com/rgomids/axiom/internal/install"
)

// install-release is the private command reached by the verified POSIX facade.
func installReleaseCommand(args []string, out, stderr io.Writer) (bool, int) {
	if len(args) == 0 || args[0] != "install-release" {
		return false, 0
	}
	fail := func(message string) (bool, int) { fmt.Fprintln(stderr, "install_error: "+message); return true, 1 }
	values := map[string]string{}
	for i := 1; i < len(args); i += 2 {
		if i+1 >= len(args) {
			return fail("invalid argument")
		}
		switch args[i] {
		case "--archive", "--checksums", "--bin-dir", "--receipt-dir":
		default:
			return fail("invalid argument")
		}
		values[args[i]] = args[i+1]
	}
	for _, key := range []string{"--archive", "--checksums", "--bin-dir", "--receipt-dir"} {
		if !strings.HasPrefix(values[key], "/") {
			return fail("absolute archive and destinations required")
		}
		if strings.ContainsAny(values[key], "\r\n=") {
			return fail("unsafe path")
		}
	}
	candidate, err := install.LoadCandidate(values["--archive"], values["--checksums"])
	if err != nil {
		return fail(err.Error())
	}
	// Match upgrade's state resolution, while fresh install and exact no-op never
	// inspect these roots or infer authority from the current working directory.
	projects, projectsErr := projectsRoot()
	state, stateErr := stateRoot()
	target := install.Target{BinaryDir: values["--bin-dir"], ReceiptDir: values["--receipt-dir"], SkillsRoot: codexSkillsRoot(), State: compatibility.Roots{Projects: projects, State: state}, Self: selfBuild(), Archive: archiveRoot(values["--receipt-dir"])}
	result := install.InstallReleasePOSIX(context.Background(), target, candidate, os.Getenv("AXIOM_INSTALL_TEST_FAIL_STAGE"), func() error {
		if projectsErr != nil {
			return projectsErr
		}
		return stateErr
	})
	fmt.Fprint(out, result.Stdout)
	fmt.Fprint(stderr, result.Stderr)
	if result.UpgradePhase == "" {
		return true, result.ExitCode
	}
	if projectsErr != nil || stateErr != nil {
		fmt.Fprintln(stderr, "install_error: owned upgrade refused: unavailable\ninstall_next: unavailable")
		return true, 1
	}
	category := upgradeCategory(result.UpgradeError)
	if result.UpgradePhase == "preview" {
		message, next := upgradeBlocked(result.UpgradePreview, category)
		fmt.Fprintf(stderr, "install_error: owned upgrade refused: %s\ninstall_next: %s\n", installDiagnostic(message), installDiagnostic(next))
		return true, 1
	}
	switch {
	case result.UpgradeError == nil && result.UpgradeResult.Status == "success":
		if result.UpgradeResult.Preservation != "" {
			fmt.Fprintf(out, "install_preserved=%s\n", result.UpgradeResult.Preservation)
		}
		fmt.Fprintln(out, "install_status=upgraded")
		return true, 0
	case result.UpgradeError == nil:
		fmt.Fprintln(out, "install_status=partial")
		message, next := upgradeSkillReceiptPartial(result.UpgradeResult)
		fmt.Fprintf(stderr, "install_error: owned upgrade partially applied: %s\ninstall_next: %s\n", installDiagnostic(message), installDiagnostic(next))
	case len(result.UpgradeResult.Ledger) > 0:
		fmt.Fprintln(out, "install_status=partial")
		fmt.Fprintf(stderr, "install_error: owned upgrade partially applied: Upgrade partially applied: %s\ninstall_next: %s\n", category, installDiagnostic(upgradeResumeNext(category)))
	case category == "authority_denied":
		fmt.Fprintln(stderr, "install_error: owned upgrade failed: Upgrade state changed after review")
	default:
		fmt.Fprintf(stderr, "install_error: owned upgrade failed: Upgrade failed before any confirmed effect: %s\n", category)
	}
	return true, 1
}

// Preserve the Shell facade's bounded ASCII diagnostics.
func installDiagnostic(value string) string {
	return strings.Map(func(r rune) rune {
		if r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || strings.ContainsRune(" :;,._`-", r) {
			return r
		}
		return -1
	}, value)
}
