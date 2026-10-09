package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/rgomids/axiom/internal/compatibility"
	"github.com/rgomids/axiom/internal/install"
	"golang.org/x/sys/windows"
)

func installReleaseCommand(args []string, out, stderr io.Writer) (bool, int) {
	if len(args) > 0 && args[0] == "windows-permissions" {
		if err := restoreWindowsPermissions(args, out); err != nil {
			fmt.Fprintf(stderr, "permission_restore_error: %v\n", err)
			return true, 1
		}
		return true, 0
	}
	if len(args) == 0 || args[0] != "install-release" {
		return false, 0
	}
	fail := func(err error) (bool, int) { fmt.Fprintf(stderr, "install_error: %v\n", err); return true, 1 }
	values := map[string]string{}
	for i := 1; i < len(args); i += 2 {
		key := args[i]
		if i+1 >= len(args) || values[key] != "" || (key != "--archive" && key != "--checksums" && key != "--bin-dir" && key != "--receipt-dir") {
			return fail(fmt.Errorf("invalid or duplicate installation argument"))
		}
		value := args[i+1]
		if !filepath.IsAbs(value) || strings.ContainsAny(value, "\r\n=") {
			return fail(fmt.Errorf("absolute installation paths required"))
		}
		values[key] = filepath.Clean(value)
	}
	if len(values) != 4 {
		return fail(fmt.Errorf("archive, checksums, bin-dir and receipt-dir are required"))
	}
	candidate, err := install.LoadCandidate(values["--archive"], values["--checksums"])
	if err != nil {
		return fail(err)
	}
	projects, err := projectsRoot()
	if err != nil {
		return fail(err)
	}
	state, err := stateRoot()
	if err != nil {
		return fail(err)
	}
	skills := codexSkillsRoot()
	target := install.Target{BinaryDir: values["--bin-dir"], ReceiptDir: values["--receipt-dir"], SkillsRoot: skills, State: compatibility.Roots{Projects: projects, State: state, Skills: skills}, Self: selfBuild(), Archive: archiveRoot(values["--receipt-dir"])}
	// Host/candidate eligibility must be established before onboarding effects.
	productType := windows.RtlGetVersion().ProductType
	if (productType == 1 || productType == 3) && candidate.Values["platform"] == "windows" && candidate.Values["goos"] == "windows" && candidate.Values["architecture"] == "amd64" {
		if err := prepareWindowsOnboarding(target, os.Stdin, out); err != nil {
			return windowsInstallReleaseResult(install.Result{}, err, target, out, stderr)
		}
	}
	result, err := install.InstallRelease(context.Background(), target, candidate)
	return windowsInstallReleaseResult(result, err, target, out, stderr)
}
