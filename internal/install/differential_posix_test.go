//go:build !windows

package install

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

// This opt-in comparison uses a caller-supplied, immutable baseline script.
// Host fixtures establish contract equivalence, not native release acceptance.
func TestPOSIXInstallerBaselineDifferential(t *testing.T) {
	baseline := os.Getenv("AXIOM_POSIX_BASELINE_SCRIPT")
	if baseline == "" {
		t.Skip("set AXIOM_POSIX_BASELINE_SCRIPT to immutable baseline shell")
	}
	for _, scenario := range []string{"fresh", "unchanged", "before_binary", "after_binary", "foreign_binary", "foreign_receipt", "locked", "modified", "divergent", "sticky_receipt_root"} {
		t.Run(scenario, func(t *testing.T) {
			target, candidate := freshPOSIXFixture(t)
			base := filepath.Dir(target.BinaryDir)
			release := newBundle("1.0.0", []byte("binary\n"))
			row := testRow
			if runtime.GOOS == "linux" {
				row = "linux:linux:" + runtime.GOARCH
				metadata := string(release.contents()["release-metadata.txt"])
				metadata = strings.ReplaceAll(metadata, "platform=macos-27", "platform=linux")
				metadata = strings.ReplaceAll(metadata, "goos=darwin", "goos=linux")
				metadata = strings.ReplaceAll(metadata, "architecture=arm64", "architecture="+runtime.GOARCH)
				release.files["release-metadata.txt"] = []byte(metadata)
			}
			hostRow = func() string { return row }
			archive, checksums := release.write(t, base)
			var err error
			candidate, err = LoadCandidate(archive, checksums)
			if err != nil {
				t.Fatal(err)
			}
			setup := func() {
				switch scenario {
				case "unchanged", "modified", "divergent":
					got := InstallReleasePOSIX(context.Background(), target, candidate, "")
					if got.ExitCode != 0 {
						t.Fatalf("setup=%+v", got)
					}
					if scenario == "modified" {
						writeFile(t, filepath.Join(target.BinaryDir, binaryName), []byte("modified\n"), 0o700)
					}
					if scenario == "divergent" {
						path := filepath.Join(target.ReceiptDir, receiptName)
						writeFile(t, path, []byte(strings.Replace(read(t, path), "revision=123456789abc", "revision=abcdef123456", 1)), 0o600)
					}
				case "sticky_receipt_root":
					privateDirectory(t, base, "receipts")
					if err := os.Chmod(target.ReceiptDir, os.ModeSticky|0o700); err != nil {
						t.Fatal(err)
					}
				case "foreign_binary":
					privateDirectory(t, base, "bin")
					writeFile(t, filepath.Join(target.BinaryDir, binaryName), []byte("foreign\n"), 0o700)
				case "foreign_receipt":
					privateDirectory(t, base, "receipts")
					writeFile(t, filepath.Join(target.ReceiptDir, receiptName), []byte("foreign\n"), 0o600)
				case "locked":
					privateDirectory(t, base, "receipts")
					privateDirectory(t, target.ReceiptDir, lockName)
				}
			}
			tools := privateDirectory(t, base, "tools")
			if runtime.GOOS == "darwin" {
				writeFile(t, filepath.Join(tools, "sw_vers"), []byte("#!/bin/sh\nprintf '27.0\\n'\n"), 0o700)
			}
			failStage := ""
			if scenario == "before_binary" || scenario == "after_binary" {
				failStage = scenario
			}
			setup()
			command := exec.Command("bash", baseline, "--archive", archive, "--checksums", checksums, "--bin-dir", target.BinaryDir, "--receipt-dir", target.ReceiptDir)
			command.Env = append(os.Environ(), "PATH="+tools+":"+os.Getenv("PATH"), "AXIOM_INSTALL_TEST_FAIL_STAGE="+failStage)
			var stdout, stderr bytes.Buffer
			command.Stdout = &stdout
			command.Stderr = &stderr
			shellErr := command.Run()
			shellCode := 0
			if shellErr != nil {
				if exit, ok := shellErr.(*exec.ExitError); ok {
					shellCode = exit.ExitCode()
				} else {
					t.Fatal(shellErr)
				}
			}
			before := posixSnapshot(t, target)
			for _, root := range []string{target.BinaryDir, target.ReceiptDir} {
				if err := os.RemoveAll(root); err != nil {
					t.Fatal(err)
				}
			}
			setup()
			actual := InstallReleasePOSIX(context.Background(), target, candidate, failStage)
			after := posixSnapshot(t, target)
			if actual.ExitCode != shellCode || actual.Stdout != stdout.String() || actual.Stderr != stderr.String() || before != after {
				t.Fatalf("shell=(%d,%q,%q) API=(%d,%q,%q)\nshell state=%s\nAPI state=%s", shellCode, stdout.String(), stderr.String(), actual.ExitCode, actual.Stdout, actual.Stderr, before, after)
			}
		})
	}
}

func posixSnapshot(t *testing.T, target Target) string {
	t.Helper()
	timestamp := regexp.MustCompile(`(?m)^installedAt=[^\n]*`)
	var snapshot strings.Builder
	for _, root := range []string{target.BinaryDir, target.ReceiptDir} {
		if !posixExists(root) {
			fmt.Fprintf(&snapshot, "absent %s\n", root)
			continue
		}
		err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			info, err := entry.Info()
			if err != nil {
				return err
			}
			fmt.Fprintf(&snapshot, "%s %s\n", path, info.Mode())
			if info.Mode().IsRegular() {
				wire, err := os.ReadFile(path)
				if err != nil {
					return err
				}
				snapshot.Write(timestamp.ReplaceAll(wire, []byte("installedAt=<timestamp>")))
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	return snapshot.String()
}
