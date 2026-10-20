//go:build unix

package runtimeadapter

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/rgomids/axiom/internal/runtimeprofile"
)

type effortInspectorFixture struct {
	operationalEffortFixture
	base string
}

func newEffortInspectorFixture(t *testing.T, id string) *effortInspectorFixture {
	t.Helper()
	return newExplicitEffortInspectorFixture(t, id, false)
}

func newExplicitEffortInspectorFixture(t *testing.T, id string, explicit bool) *effortInspectorFixture {
	t.Helper()
	f := &effortInspectorFixture{operationalEffortFixture: newOperationalEffortFixture(t, id, explicit)}
	f.base = filepath.Dir(f.request.Roots.User)
	for _, s := range f.rules().Sources {
		path := filepath.Join(s.Root, s.Relative)
		switch {
		case s.Directory:
			writeFixtureFile(t, filepath.Join(path, "10-base.json"), "{}")
		case s.Format == "toml":
			writeFixtureFile(t, path, "# no effort selector\n")
		default:
			writeFixtureFile(t, path, "{}")
		}
	}
	return f
}

func (f *effortInspectorFixture) rules() effortRules {
	return fixedEffortRules(f.request.Invocation.RuntimeID, f.request.Roots)
}

func (f *effortInspectorFixture) profile() runtimeprofile.ModelProfile {
	return f.request.Configuration.ModelProfiles[0]
}

func (f *effortInspectorFixture) source(t *testing.T, scope string) string {
	t.Helper()
	for _, s := range f.rules().Sources {
		if s.Scope == scope {
			return filepath.Join(s.Root, s.Relative)
		}
	}
	t.Fatal("no source for scope", scope)
	return ""
}

func (f *effortInspectorFixture) observe(rules effortRules) ([]effortSourceFact, error) {
	p := f.profile()
	return observeEffortFacts(context.Background(), effortIntent{p.RuntimeID, f.request.InvocationMode, f.request.ExecutableDigest, p.Model, p.Effort}, rules, f.request.Invocation)
}

func (f *effortInspectorFixture) digest() (string, error) {
	return InspectEffortSourcesV4(context.Background(), f.request)
}

func writeFixtureFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

var canonicalDigest = regexp.MustCompile(`^[0-9a-f]{64}$`)

func TestInspectEffortSourcesDigest(t *testing.T) {
	for _, id := range []string{"codex", "claude"} {
		t.Run(id, func(t *testing.T) {
			variants := map[string]func(*testing.T) *effortInspectorFixture{
				"baseline": func(t *testing.T) *effortInspectorFixture { return newEffortInspectorFixture(t, id) },
				"absent-user-source": func(t *testing.T) *effortInspectorFixture {
					f := newEffortInspectorFixture(t, id)
					if err := os.Remove(f.source(t, "user")); err != nil {
						t.Fatal(err)
					}
					return f
				},
				"different-non-selector-bytes": func(t *testing.T) *effortInspectorFixture {
					f := newEffortInspectorFixture(t, id)
					content := "# other\nunrelated = true\n"
					if id == "claude" {
						content = `{"unrelated":true}`
					}
					writeFixtureFile(t, f.source(t, "project"), content)
					return f
				},
				"explicit-profile-effort": func(t *testing.T) *effortInspectorFixture { return newExplicitEffortInspectorFixture(t, id, true) },
			}
			seen := map[string]bool{}
			for name, build := range variants {
				f := build(t)
				digest, err := f.digest()
				if err != nil || !canonicalDigest.MatchString(digest) {
					t.Fatalf("%s: %q %v", name, digest, err)
				}
				projection, _, err := inspectEffortV4(context.Background(), f.request)
				wire, _ := json.Marshal(projection)
				if err != nil || digest != effortHash(wire) {
					t.Fatalf("%s: digest is not the hash of the canonical projection: %v", name, err)
				}
				if again, err := f.digest(); err != nil || again != digest {
					t.Fatalf("%s: not deterministic: %v", name, err)
				}
				seen[digest] = true
			}
			if len(seen) != len(variants) {
				t.Fatal("variant digests collided")
			}
			f := newEffortInspectorFixture(t, id)
			before, _ := f.digest()
			writeFixtureFile(t, f.source(t, "managed"), map[bool]string{true: "# drift\n", false: `{"drift":1}`}[id == "codex"])
			after, err := f.digest()
			if err != nil || after == before {
				t.Fatalf("source content drift not bound into the digest: %v", err)
			}
			if id == "claude" {
				writeFixtureFile(t, filepath.Join(f.source(t, "managed-drop-in"), "20-extra.json"), "{}")
				extended, err := f.digest()
				if err != nil || extended == after {
					t.Fatal("drop-in membership not bound:", err)
				}
			}
		})
	}
}

func TestInspectEffortSourcesRefusals(t *testing.T) {
	for _, id := range []string{"codex", "claude"} {
		t.Run(id, func(t *testing.T) {
			cases := map[string]struct {
				want  error
				apply func(*testing.T, *effortInspectorFixture)
			}{
				"selector-in-source": {ErrEffortOverrideConflict, func(t *testing.T, f *effortInspectorFixture) {
					writeFixtureFile(t, f.source(t, "user"), map[bool]string{true: `model_reasoning_effort = "private-canary"`, false: `{"effortLevel":"private-canary"}`}[id == "codex"])
				}},
				"unsupported-source-grammar": {ErrEffortConfigurationUnsafe, func(t *testing.T, f *effortInspectorFixture) {
					writeFixtureFile(t, f.source(t, "project"), map[bool]string{true: "[[array]]\n", false: `[]`}[id == "codex"])
				}},
				"argv-effort-override": {ErrEffortOverrideConflict, func(t *testing.T, f *effortInspectorFixture) {
					f.request.Invocation.Argv = append(f.request.Invocation.Argv, "--effort", "high")
				}},
			}
			if id == "claude" {
				cases["inherited-environment-override"] = struct {
					want  error
					apply func(*testing.T, *effortInspectorFixture)
				}{ErrEffortOverrideConflict, func(t *testing.T, f *effortInspectorFixture) {
					f.request.Invocation.Env = append(f.request.Invocation.Env, "CLAUDE_CODE_EFFORT_LEVEL=high")
				}}
			}
			for name, c := range cases {
				f := newEffortInspectorFixture(t, id)
				c.apply(t, f)
				if digest, err := f.digest(); !errors.Is(err, c.want) || digest != "" || strings.Contains(err.Error(), "private-canary") {
					t.Fatalf("%s: %q %v", name, digest, err)
				}
			}
		})
	}
}

func TestObserveEffortFactsCanonicalExplicitOverride(t *testing.T) {
	for _, id := range []string{"codex", "claude"} {
		t.Run(id, func(t *testing.T) {
			f := newExplicitEffortInspectorFixture(t, id, true)
			if _, err := f.observe(f.rules()); err != nil {
				t.Fatal(err)
			}
			if id == "claude" {
				f.request.Invocation.Env = append(f.request.Invocation.Env, "claude_code_effort_level=future-effort")
				if _, err := f.observe(f.rules()); !errors.Is(err, ErrEffortOverrideConflict) {
					t.Fatal("normalized inherited effort accepted", err)
				}
				return
			}
			f.request.Invocation.Argv = append(f.request.Invocation.Argv, "--config", "model_reasoning_effort=other")
			if _, err := f.observe(f.rules()); err == nil {
				t.Fatal("non-canonical override accepted")
			}
		})
	}
}

func TestObserveEffortFactsEnvironmentKeys(t *testing.T) {
	f := newEffortInspectorFixture(t, "codex")
	rules := f.rules()
	rules.EnvironmentKeys = []string{"FIXTURE_EFFORT_LEVEL"}
	if _, err := f.observe(rules); err != nil {
		t.Fatal(err)
	}
	f.request.Invocation.Env = append(f.request.Invocation.Env, "FIXTURE_EFFORT_LEVEL=low")
	if _, err := f.observe(rules); !errors.Is(err, ErrEffortOverrideConflict) {
		t.Fatal("declared environment selector accepted", err)
	}
	f = newEffortInspectorFixture(t, "claude")
	f.request.Invocation.Env = append(f.request.Invocation.Env, "claude_code_effort_level=private-canary")
	if _, err := f.observe(f.rules()); !errors.Is(err, ErrEffortOverrideConflict) || strings.Contains(err.Error(), "private-canary") {
		t.Fatalf("normalized selector accepted: %v", err)
	}
	f = newEffortInspectorFixture(t, "claude")
	rules = f.rules()
	rules.EnvironmentKeys = append(rules.EnvironmentKeys, "claude_code_effort_level")
	if _, err := f.observe(rules); !errors.Is(err, ErrEffortConfigurationUnsafe) {
		t.Fatalf("duplicate normalized key accepted: %v", err)
	}
}

func TestObserveEffortFactsRefusesUnrecognizedRules(t *testing.T) {
	for _, id := range []string{"codex", "claude"} {
		t.Run(id, func(t *testing.T) {
			edit := map[string]func(*effortRules){
				"traversal-path":     func(r *effortRules) { r.Sources[0].Relative = "../outside/config.toml" },
				"absolute-path":      func(r *effortRules) { r.Sources[0].Relative = "/etc/config.toml" },
				"unsupported-format": func(r *effortRules) { r.Sources[0].Format = "yaml" },
				"unsupported-file":   func(r *effortRules) { r.Sources[0].Relative = "credentials.json" },
				"unsupported-scope":  func(r *effortRules) { r.Sources[0].Scope = "home" },
				"duplicate-source":   func(r *effortRules) { r.Sources = append(r.Sources, r.Sources[0]) },
				"missing-scope":      func(r *effortRules) { r.Sources = r.Sources[1:] },
				"unsupported-key":    func(r *effortRules) { r.EnvironmentKeys = append(r.EnvironmentKeys, "BAD KEY") },
				"too-many-sources": func(r *effortRules) {
					for len(r.Sources) <= 32 {
						r.Sources = append(r.Sources, r.Sources[0])
					}
				},
			}
			if id == "claude" {
				edit["missing-required-environment-key"] = func(r *effortRules) { r.EnvironmentKeys = []string{"OTHER_EFFORT_KEY"} }
				edit["no-json-keys"] = func(r *effortRules) { r.JSONKeys = nil }
			}
			for name, apply := range edit {
				f := newEffortInspectorFixture(t, id)
				rules := f.rules()
				apply(&rules)
				if _, err := f.observe(rules); !errors.Is(err, ErrEffortConfigurationUnsafe) {
					t.Fatalf("%s accepted: %v", name, err)
				}
			}
		})
	}
	f := newEffortInspectorFixture(t, "claude")
	source := f.rules().Sources[0]
	source.Root += string(filepath.Separator) + ".." + string(filepath.Separator) + filepath.Base(source.Root)
	if recognizedEffortSource("claude", source) {
		t.Fatal("unclean root accepted")
	}
}

func snapshotTree(t *testing.T, root string) map[string]string {
	t.Helper()
	out := map[string]string{}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := os.Lstat(path)
		if err != nil {
			return err
		}
		entry := info.Mode().String() + info.ModTime().String()
		if info.Mode().IsRegular() {
			b, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			entry += string(b)
		}
		out[path] = entry
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func TestInspectEffortSourcesIsReadOnly(t *testing.T) {
	for _, id := range []string{"codex", "claude"} {
		f := newEffortInspectorFixture(t, id)
		if err := os.Remove(f.source(t, "user")); err != nil {
			t.Fatal(err)
		}
		before := snapshotTree(t, f.base)
		if _, err := f.digest(); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(before, snapshotTree(t, f.base)) {
			t.Fatal("inspection created or mutated a source")
		}
	}
}

func TestInspectEffortSourcesFailsClosed(t *testing.T) {
	other := strings.Repeat("b", 64)
	type mutation func(*testing.T, *effortInspectorFixture)
	cases := map[string]mutation{
		"unconsented": func(_ *testing.T, f *effortInspectorFixture) { f.request.Consented = false },
		"empty-configuration": func(_ *testing.T, f *effortInspectorFixture) {
			f.request.Configuration = runtimeprofile.ConfigurationV4{}
		},
		"disabled-runtime":             func(_ *testing.T, f *effortInspectorFixture) { f.request.Configuration.Runtimes[0].Enabled = false },
		"unknown-profile":              func(_ *testing.T, f *effortInspectorFixture) { f.request.ProfileID = "other-profile" },
		"wrong-executable-digest":      func(_ *testing.T, f *effortInspectorFixture) { f.request.ExecutableDigest = other },
		"wrong-invocation-runtime":     func(_ *testing.T, f *effortInspectorFixture) { f.request.Invocation.RuntimeID = "other" },
		"wrong-invocation-mode":        func(_ *testing.T, f *effortInspectorFixture) { f.request.InvocationMode = "interactive" },
		"empty-argv":                   func(_ *testing.T, f *effortInspectorFixture) { f.request.Invocation.Argv = nil },
		"argv-mode-drift":              func(_ *testing.T, f *effortInspectorFixture) { f.request.Invocation.Argv[1] = "--interactive" },
		"argv-model-drift":             func(_ *testing.T, f *effortInspectorFixture) { f.request.Invocation.Argv[3] = "other-model" },
		"missing-user-root":            func(_ *testing.T, f *effortInspectorFixture) { f.request.Roots.User = "" },
		"relative-root":                func(_ *testing.T, f *effortInspectorFixture) { f.request.Roots.Project = "project" },
		"unclean-root":                 func(_ *testing.T, f *effortInspectorFixture) { f.request.Roots.Project += "/../repository" },
		"root-is-filesystem-root":      func(_ *testing.T, f *effortInspectorFixture) { f.request.Roots.Project = "/" },
		"user-root-differs-from-home":  func(_ *testing.T, f *effortInspectorFixture) { f.request.Roots.User = f.request.Roots.Managed },
		"project-equals-managed-roots": func(_ *testing.T, f *effortInspectorFixture) { f.request.Roots.Project = f.request.Roots.Managed },
		// PM-13: only a missing managed root may be an absence fact
		// (TestTrustedEffortAnchorAbsentManaged); user and project must exist.
		"absent-user-root": func(t *testing.T, f *effortInspectorFixture) {
			if err := os.RemoveAll(f.request.Roots.User); err != nil {
				t.Fatal(err)
			}
		},
		"absent-project-root": func(t *testing.T, f *effortInspectorFixture) {
			if err := os.RemoveAll(f.request.Roots.Project); err != nil {
				t.Fatal(err)
			}
		},
		"root-is-file": func(t *testing.T, f *effortInspectorFixture) {
			f.request.Roots.Managed = f.request.Invocation.Argv[0]
		},
		"symlinked-root": func(t *testing.T, f *effortInspectorFixture) {
			link := filepath.Join(f.base, "project-link")
			if err := os.Symlink(f.request.Roots.Project, link); err != nil {
				t.Fatal("symlink setup failed on a supported host:", err)
			}
			f.request.Roots.Project = link
		},
		"symlink-escape": func(t *testing.T, f *effortInspectorFixture) {
			outside := filepath.Join(f.base, "outside")
			writeFixtureFile(t, filepath.Join(outside, "settings.json"), "{}")
			writeFixtureFile(t, filepath.Join(outside, "config.toml"), "# no effort selector\n")
			dir := filepath.Dir(f.source(t, "user"))
			if err := os.RemoveAll(dir); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(outside, dir); err != nil {
				t.Fatal("symlink setup failed on a supported host:", err)
			}
		},
		"group-writable-root": func(t *testing.T, f *effortInspectorFixture) {
			chmodFixture(t, f.request.Roots.User, 0o770)
		},
		"sticky-world-writable-root": func(t *testing.T, f *effortInspectorFixture) {
			chmodFixture(t, f.request.Roots.User, 0o1777)
		},
		"unsafe-ancestor": func(t *testing.T, f *effortInspectorFixture) {
			chmodFixture(t, f.base, 0o777)
		},
		"special-file-source": func(t *testing.T, f *effortInspectorFixture) {
			path := f.source(t, "project")
			if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}
			if err := syscall.Mkfifo(path, 0o600); err != nil {
				t.Fatal("fifo setup failed on a supported host:", err)
			}
		},
		"directory-where-file-required": func(t *testing.T, f *effortInspectorFixture) {
			path := f.source(t, "project")
			if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}
			if err := os.Mkdir(path, 0o700); err != nil {
				t.Fatal(err)
			}
		},
		"malformed-source": func(t *testing.T, f *effortInspectorFixture) {
			writeFixtureFile(t, f.source(t, "managed"), "{ malformed")
			writeFixtureFile(t, f.source(t, "user"), "{ malformed")
		},
	}
	claudeOnly := map[string]mutation{
		"drop-in-non-json": func(t *testing.T, f *effortInspectorFixture) {
			writeFixtureFile(t, filepath.Join(f.source(t, "managed-drop-in"), "notes.txt"), "")
		},
		"drop-in-malformed-json": func(t *testing.T, f *effortInspectorFixture) {
			writeFixtureFile(t, filepath.Join(f.source(t, "managed-drop-in"), "20-bad.json"), "{")
		},
	}
	for _, id := range []string{"codex", "claude"} {
		t.Run(id, func(t *testing.T) {
			run := func(name string, m mutation) {
				f := newEffortInspectorFixture(t, id)
				if _, err := f.digest(); err != nil {
					t.Fatalf("%s: baseline refused: %v", name, err)
				}
				m(t, f)
				if digest, err := f.digest(); !errors.Is(err, ErrEffortConfigurationUnsafe) || digest != "" {
					t.Fatalf("%s: accepted or wrong refusal: %q %v", name, digest, err)
				}
			}
			for name, m := range cases {
				run(name, m)
			}
			if id == "claude" {
				for name, m := range claudeOnly {
					run(name, m)
				}
			}
		})
	}
	f := newExplicitEffortInspectorFixture(t, "claude", true)
	f.request.Invocation.Env = slicesWithout(f.request.Invocation.Env, "CLAUDE_CODE_EFFORT_LEVEL=")
	if _, err := f.digest(); err == nil {
		t.Fatal("explicit effort without its canonical environment value accepted")
	}
}

func slicesWithout(values []string, prefix string) []string {
	kept := []string{}
	for _, v := range values {
		if !strings.HasPrefix(v, prefix) {
			kept = append(kept, v)
		}
	}
	return kept
}

func chmodFixture(t *testing.T, path string, mode os.FileMode) {
	t.Helper()
	if mode&0o1000 != 0 {
		mode = mode&0o777 | os.ModeSticky
	}
	if err := os.Chmod(path, mode); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(path, 0o700) })
	info, err := os.Lstat(path)
	if err != nil || info.Mode().Perm()&0o022 == 0 {
		t.Fatal("permission setup not effective:", err)
	}
}

func TestInspectEffortSourcesContextBoundary(t *testing.T) {
	f := newEffortInspectorFixture(t, "claude")
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := InspectEffortSourcesV4(cancelled, f.request); !errors.Is(err, ErrEffortConfigurationUnsafe) {
		t.Fatal("cancelled context accepted:", err)
	}
	expired, stop := context.WithDeadline(context.Background(), time.Unix(0, 0))
	defer stop()
	if _, err := InspectEffortSourcesV4(expired, f.request); !errors.Is(err, ErrEffortConfigurationUnsafe) {
		t.Fatal("expired deadline accepted:", err)
	}
}

// A root replaced at its pathname is a different identity; the inspector
// compares the root binding before and after reading sources.
func TestTrustedEffortRootIdentity(t *testing.T) {
	f := newEffortInspectorFixture(t, "codex")
	before, err := trustedEffortRoot(f.request.Roots.User)
	if err != nil {
		t.Fatal(err)
	}
	if again, err := trustedEffortRoot(f.request.Roots.User); err != nil || again != before {
		t.Fatal("root binding not stable:", err)
	}
	if err := os.Rename(f.request.Roots.User, f.request.Roots.User+"-old"); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(f.request.Roots.User, 0o700); err != nil {
		t.Fatal(err)
	}
	if after, err := trustedEffortRoot(f.request.Roots.User); err != nil || after == before {
		t.Fatal("replaced root kept its identity:", err)
	}
}
