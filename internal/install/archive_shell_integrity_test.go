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

func TestShellArchiveIntegrityBeforeExecution(t *testing.T) {
	helperEnv := trustedHelperEnv(t)
	for _, version := range []int{1, 2} {
		mutations := []string{"valid", "complete tars", "tar trailing", "zero tar tail", "zero trailing", "trailing", "truncated", "crc", "compress tail", "split invalid", "duplicate directory", "duplicate file", "path", "manifest", "metadata", "file limit", "split", "extra empty", "prefix empty"}
		for _, mutation := range mutations {
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
				case "compress tail":
					wire = append(wire, 0x1f, 0x9d, 0x90)
				case "split invalid":
					wire = append(gzipMembers(t, raw[:17]), gzipMembers(t, raw[18:])...)
				case "split":
					wire = gzipMembers(t, raw, 17)
				case "extra empty":
					wire = append(wire, gzipMembers(t, nil)...)
				case "prefix empty":
					wire = append(gzipMembers(t, nil), wire...)
				case "complete tars":
					wire = append(wire, gzipMembers(t, raw)...)
				case "tar trailing":
					wire = gzipMembers(t, append(raw, []byte("garbage")...))
				case "zero tar tail":
					wire = gzipMembers(t, append(raw, make([]byte, 512)...))
				case "zero trailing":
					wire = append(wire, make([]byte, 16)...)
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
				// Identical compressed bytes go to the host gzip comparator.
				// GNU accepts zero suffixes; comparator agreement is not authority.
				reference := exec.Command("gzip", "-t", archive)
				referenceOut, referenceErr := reference.CombinedOutput()
				t.Logf("host_gzip_accept=%t detail=%s", referenceErr == nil, bytes.TrimSpace(referenceOut))
				if container := os.Getenv("AXIOM_TEST_GNU_GZIP_CONTAINER"); container != "" {
					reference = exec.Command("docker", "run", "--rm", "--network", "none", "--read-only", "-v", filepath.Dir(archive)+":/fixtures:ro", container, "gzip", "-t", "/fixtures/"+filepath.Base(archive))
					referenceOut, referenceErr = reference.CombinedOutput()
					t.Logf("gnu_gzip_accept=%t detail=%s", referenceErr == nil, bytes.TrimSpace(referenceOut))
					if bytes.Contains(referenceOut, []byte("docker:")) {
						t.Fatalf("GNU comparator unavailable: %s", referenceOut)
					}
				}
				_, goErr := LoadCandidate(archive, sums)
				wantAccept := mutation == "valid" || mutation == "zero tar tail" || version == 1 && (mutation == "split" || mutation == "extra empty" || mutation == "prefix empty")
				if (goErr == nil) != wantAccept {
					t.Fatalf("unexpected Go acceptance: %v", goErr)
				}
				args := []string{"--verify-only", "--archive", archive, "--checksums", sums, "--bin-dir", filepath.Join(work, "bin"), "--receipt-dir", filepath.Join(work, "receipts")}
				script, _ := filepath.Abs("../../scripts/install-release.sh")
				cmd := exec.Command("bash", append([]string{script}, args...)...)
				cmd.Env = append(os.Environ(), helperEnv...)
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
				cmd.Env = append(os.Environ(), helperEnv...)
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

func TestShellHelperNeedsNoPython(t *testing.T) {
	helperEnv := trustedHelperEnv(t)
	for _, version := range []int{1, 2} {
		for _, prerequisite := range []string{"missing", "incompatible"} {
			t.Run(string(rune('0'+version))+"/"+prerequisite, func(t *testing.T) {
				work := t.TempDir()
				marker := filepath.Join(work, "candidate-executed")
				sentinel := "#!/bin/sh\nprintf probe > '" + marker + "'\nexit 91\n"
				var archive, sums string
				if version == 1 {
					archive, sums = newBundle("1.1.0", []byte(sentinel)).write(t, work)
				} else {
					archive, sums = newV2ArchiveFor(t, sentinel, "").write(t)
				}
				tools := filepath.Join(work, "tools")
				if err := os.Mkdir(tools, 0700); err != nil {
					t.Fatal(err)
				}
				for _, name := range strings.Fields("bash basename dirname wc awk tr sha256sum shasum uname stat ls id mktemp cp chmod gzip head dd od tail tar sed sort uniq grep find rm cmp cut") {
					path, err := exec.LookPath(name)
					if err != nil {
						continue
					}
					if err := os.Symlink(path, filepath.Join(tools, name)); err != nil {
						t.Fatal(err)
					}
				}
				if prerequisite == "incompatible" {
					if err := os.WriteFile(filepath.Join(tools, "python3"), []byte("#!/bin/sh\nexit 2\n"), 0700); err != nil {
						t.Fatal(err)
					}
				}
				script, _ := filepath.Abs("../../scripts/install-release.sh")
				for _, verify := range []bool{true} {
					if version == 1 && !verify {
						continue
					}
					args := []string{script, "--archive", archive, "--checksums", sums, "--bin-dir", filepath.Join(work, "bin"), "--receipt-dir", filepath.Join(work, "receipts")}
					if verify {
						args = append(args, "--verify-only")
					}
					cmd := exec.Command("bash", args...)
					cmd.Env = append(append(os.Environ(), helperEnv...), "PATH="+tools)
					out, err := cmd.CombinedOutput()
					if verify && err != nil {
						t.Fatalf("helper must not require Python: %v: %s", err, out)
					}
					for _, path := range []string{marker, filepath.Join(work, "bin"), filepath.Join(work, "receipts")} {
						if _, err := os.Stat(path); !os.IsNotExist(err) {
							t.Fatalf("unexpected effect: %s", path)
						}
					}
				}
			})
		}
	}
}

func trustedHelperEnv(t *testing.T) []string {
	t.Helper()
	name := "axiom-verifier-1.1.0-aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa-" + runtime.GOOS + "-" + runtime.GOARCH
	helper := filepath.Join(t.TempDir(), name)
	cmd := exec.Command("go", "build", "-trimpath", "-ldflags", "-X main.identity="+name, "-o", helper, "../../cmd/axiom-archive-verify")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build trusted source helper: %v: %s", err, out)
	}
	wire, err := os.ReadFile(helper)
	if err != nil {
		t.Fatal(err)
	}
	return []string{"AXIOM_VERIFIER_PATH=" + helper, "AXIOM_VERIFIER_SHA256=" + digest(wire)}
}

func TestShellHelperAuthenticationBeforeExecution(t *testing.T) {
	env := trustedHelperEnv(t)
	work := t.TempDir()
	marker := filepath.Join(work, "executed")
	archive, sums := newBundle("1.1.0", []byte("#!/bin/sh\ntouch '"+marker+"'\n")).write(t, work)
	script, _ := filepath.Abs("../../scripts/install-release.sh")
	for _, tc := range []string{"missing", "wrong-pin", "wrong-platform", "symlink"} {
		t.Run(tc, func(t *testing.T) {
			localEnv := append([]string{}, env...)
			switch tc {
			case "missing":
				localEnv = []string{"AXIOM_VERIFIER_PATH=", "AXIOM_VERIFIER_SHA256="}
			case "wrong-pin":
				localEnv[1] = "AXIOM_VERIFIER_SHA256=" + strings.Repeat("0", 64)
			case "wrong-platform":
				fake := filepath.Join(work, "axiom-verifier-1.1.0-aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa-"+runtime.GOOS+"-"+runtime.GOARCH)
				wire := []byte("#!/bin/sh\ntouch '" + marker + "'\n")
				if err := os.WriteFile(fake, wire, 0700); err != nil {
					t.Fatal(err)
				}
				localEnv = []string{"AXIOM_VERIFIER_PATH=" + fake, "AXIOM_VERIFIER_SHA256=" + digest(wire)}
			case "symlink":
				helper := strings.TrimPrefix(env[0], "AXIOM_VERIFIER_PATH=")
				link := filepath.Join(t.TempDir(), filepath.Base(helper))
				if err := os.Symlink(helper, link); err != nil {
					t.Fatal(err)
				}
				localEnv[0] = "AXIOM_VERIFIER_PATH=" + link
			}
			cmd := exec.Command("bash", script, "--archive", archive, "--checksums", sums, "--bin-dir", filepath.Join(work, "bin"), "--receipt-dir", filepath.Join(work, "receipts"))
			cmd.Env = append(os.Environ(), localEnv...)
			if out, err := cmd.CombinedOutput(); err == nil {
				t.Fatalf("untrusted helper accepted: %s", out)
			}
			for _, p := range []string{marker, filepath.Join(work, "bin"), filepath.Join(work, "receipts")} {
				if _, err := os.Lstat(p); !os.IsNotExist(err) {
					t.Fatalf("unexpected effect: %s", p)
				}
			}
		})
	}
}
