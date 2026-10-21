package install

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// Covers the repaired integrity seams, not exhaustive gzip/tar framing parity.
func TestShellArchiveIntegrityBeforeExecution(t *testing.T) {
	for _, version := range []int{1, 2} {
		for _, mutation := range []string{"valid", "trailing", "truncated", "crc", "duplicate directory", "duplicate file", "path", "manifest", "metadata", "file limit"} {
			t.Run(string(rune('0'+version))+"/"+mutation, func(t *testing.T) {
				work := t.TempDir()
				marker := filepath.Join(work, "candidate-executed")
				sentinel := []byte("#!/bin/sh\nprintf probe > '" + marker + "'\nexit 91\n")
				var archive, sums string
				if version == 1 {
					b := newBundle("1.1.0", sentinel)
					if runtime.GOOS == "linux" {
						metadata := string(b.contents()["release-metadata.txt"])
						metadata = strings.ReplaceAll(metadata, "platform=macos-27", "platform=linux")
						metadata = strings.ReplaceAll(metadata, "goos=darwin", "goos=linux")
						metadata = strings.ReplaceAll(metadata, "architecture=arm64", "architecture="+runtime.GOARCH)
						b.files["release-metadata.txt"] = []byte(metadata)
					}
					switch mutation {
					case "path":
						b.files["../escape"] = []byte("x")
					case "manifest":
						b.files["skills-manifest.txt"] = []byte("invalid\n")
					case "metadata":
						b.files["release-metadata.txt"] = []byte("invalid\n")
					case "file limit":
						b.files["LICENSE"] = bytes.Repeat([]byte("x"), maxBundleFile+1)
					}
					archive, sums = b.write(t, work)
				} else {
					b := newV2ArchiveFor(t, string(sentinel), "")
					switch mutation {
					case "path":
						b.files["../escape"] = []byte("x")
						b.seal()
					case "manifest":
						b.files["skills-manifest.txt"] = []byte("invalid\n")
						b.seal()
					case "metadata":
						b.files["release-metadata.txt"] = []byte("invalid\n")
						b.seal()
					case "file limit":
						b.files["LICENSE"] = bytes.Repeat([]byte("x"), maxBundleFile+1)
						b.seal()
					}
					archive, sums = b.write(t)
				}
				raw := tarBytes(t, archive)
				wire := gzipMembers(t, raw)
				switch mutation {
				case "trailing":
					wire = append(wire, []byte("garbage")...)
				case "truncated":
					wire = wire[:len(wire)-3]
				case "crc":
					wire[len(wire)-8] ^= 1
				case "duplicate directory":
					wire = gzipMembers(t, append(append(append([]byte{}, raw[:512]...), raw[:512]...), raw[512:]...))
				case "duplicate file":
					// The first regular file follows the root directory in both fixtures.
					n := 512
					for raw[n+156] == '5' {
						n += 512
					}
					end := n + 1024
					wire = gzipMembers(t, append(append(append([]byte{}, raw[:end]...), raw[n:end]...), raw[end:]...))
				}
				replaceArchive(t, archive, sums, wire)
				_, goErr := LoadCandidate(archive, sums)
				args := []string{"--verify-only", "--archive", archive, "--checksums", sums, "--bin-dir", filepath.Join(work, "bin"), "--receipt-dir", filepath.Join(work, "receipts")}
				script, _ := filepath.Abs("../../scripts/install-release.sh")
				cmd := exec.Command("bash", append([]string{script}, args...)...)
				out, shellErr := cmd.CombinedOutput()
				t.Logf("go_accept=%t shell_accept=%t shell=%s", goErr == nil, shellErr == nil, bytes.TrimSpace(out))
				if _, err := os.Stat(marker); !os.IsNotExist(err) {
					t.Fatal("verify-only executed candidate")
				}
				for _, dest := range []string{"bin", "receipts"} {
					if _, err := os.Stat(filepath.Join(work, dest)); !os.IsNotExist(err) {
						t.Fatal("persistent destination created")
					}
				}
				if (goErr == nil) != (shellErr == nil) {
					t.Fatalf("shell/Go integrity mismatch: %v / %v", goErr, shellErr)
				}
				cmd = exec.Command("bash", append([]string{script}, args[1:]...)...)
				cmd.CombinedOutput()
				if goErr == nil {
					if _, err := os.Stat(marker); err != nil {
						t.Fatal("valid control did not reach sentinel candidate")
					}
				} else {
					if _, err := os.Stat(marker); !os.IsNotExist(err) {
						t.Fatal("invalid archive executed candidate")
					}
					for _, dest := range []string{"bin", "receipts"} {
						if _, err := os.Stat(filepath.Join(work, dest)); !os.IsNotExist(err) {
							t.Fatal("invalid archive published destination")
						}
					}
				}
			})
		}
	}
}
