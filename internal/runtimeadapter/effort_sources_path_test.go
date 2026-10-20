//go:build unix

package runtimeadapter

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/rgomids/axiom/internal/executiongraph"
	"github.com/rgomids/axiom/internal/runtimeprofile"
)

// Specification 007 AXM-22 BLD-001B.3a: the argv scan must not treat
// the spelling of a stage path as an effort override. Only the clean absolute
// operand of the adapter's own path options is exempt; every override stays
// refused and no path enters the EffortSourceDigest.

type effortStagePaths struct{ workspace, inbox, axiom string }

func (f *effortInspectorFixture) stagePaths(t *testing.T, paths effortStagePaths, write, process bool) {
	t.Helper()
	request := f.stage
	request.Workspace, request.Inbox, request.AxiomPath, request.RepositoryWrite, request.Process = paths.workspace, paths.inbox, paths.axiom, write, process
	inv, err := BuildStageInvocation(request)
	if err != nil {
		t.Fatalf("stage invocation %+v: %v", paths, err)
	}
	if f.request.Invocation, err = withStageEffort(inv, f.profile().Effort); err != nil {
		t.Fatal(err)
	}
}

func (f *effortInspectorFixture) insertArgv(tokens ...string) {
	argv := f.request.Invocation.Argv
	if f.request.Invocation.RuntimeID == "codex" && f.profile().Effort.Mode == runtimeprofile.EffortExplicit {
		argv = append(append(append([]string{}, argv[:len(argv)-2]...), tokens...), argv[len(argv)-2:]...)
	} else {
		argv = append(append([]string{}, argv...), tokens...)
	}
	f.request.Invocation.Argv = argv
}

func TestEffortSourcesPathSpellingIsNotOverride(t *testing.T) {
	neutral := effortStagePaths{"/srv/stage/work", "/srv/stage/inbox", "/srv/stage/bin/axiom"}
	cases := map[string]effortStagePaths{
		"component-contains-effort": {"/srv/effort-tracking/work", "/srv/effort-tracking/inbox", neutral.axiom},
		"workspace-named-effort":    {"/srv/effort", "/srv/inbox", neutral.axiom},
		"nested-mixed-case":         {"/home/u/projects/low-effort/apps/Effort/repo", "/home/u/.axiom/inbox/EFFORT", neutral.axiom},
		"selector-spelled-dir":      {"/srv/model_reasoning_effort=low/work", "/srv/CLAUDE_CODE_EFFORT_LEVEL=low/inbox", neutral.axiom},
		"axiom-path-effort":         {neutral.workspace, neutral.inbox, "/opt/effort/bin/axiom"},
	}
	for _, id := range []string{"codex", "claude"} {
		for effortName, explicit := range map[string]bool{"explicit": true, "runtime-default": false} {
			for _, write := range []bool{false, true} {
				for _, process := range []bool{false, true} {
					f := newExplicitEffortInspectorFixture(t, id, explicit)
					f.stagePaths(t, neutral, write, process)
					want, err := f.digest()
					if err != nil {
						t.Fatalf("%s/%s write=%v process=%v neutral baseline refused: %v", id, effortName, write, process, err)
					}
					// The real temporary root carries this test's name ("Effort").
					cases["temp-root"] = effortStagePaths{filepath.Join(f.base, "work"), filepath.Join(f.base, "inbox"), filepath.Join(f.base, "bin", "axiom")}
					for name, paths := range cases {
						f.stagePaths(t, paths, write, process)
						got, err := f.digest()
						// Stage paths are not digest facts: the digest is unchanged.
						if err != nil || got != want {
							t.Fatalf("%s/%s write=%v process=%v %s: digest %q err %v, want %q", id, effortName, write, process, name, got, err, want)
						}
					}
				}
			}
		}
	}
}

func TestEffortSourcesOverridesStillRefusedUnderEffortPaths(t *testing.T) {
	paths := effortStagePaths{"/srv/effort/work", "/srv/effort/inbox", "/opt/effort/bin/axiom"}
	symlinkUser := func(t *testing.T, f *effortInspectorFixture) {
		path := f.source(t, "user")
		if err := os.Rename(path, path+".saved"); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(path+".saved", path); err != nil {
			t.Fatalf("native symlink setup failed on a supported host: %v", err)
		}
	}
	cases := []struct {
		id, name string
		want     error
		apply    func(*testing.T, *effortInspectorFixture)
	}{
		{"codex", "conflicting-config-option", ErrEffortOverrideConflict, func(_ *testing.T, f *effortInspectorFixture) {
			f.insertArgv("--config", "model_reasoning_effort=low")
		}},
		{"codex", "short-config-option", ErrEffortOverrideConflict, func(_ *testing.T, f *effortInspectorFixture) { f.insertArgv("-c", "model_reasoning_effort=low") }},
		{"codex", "profile-option", ErrEffortOverrideConflict, func(_ *testing.T, f *effortInspectorFixture) { f.insertArgv("--profile", "fast") }},
		{"codex", "non-path-add-dir-operand", ErrEffortOverrideConflict, func(_ *testing.T, f *effortInspectorFixture) {
			f.insertArgv("--add-dir", "model_reasoning_effort=low")
		}},
		{"codex", "relative-cd-operand", ErrEffortOverrideConflict, func(_ *testing.T, f *effortInspectorFixture) { f.insertArgv("-C", "effort/work") }},
		{"codex", "unclean-add-dir-operand", ErrEffortOverrideConflict, func(_ *testing.T, f *effortInspectorFixture) {
			f.insertArgv("--add-dir", "/srv/x/../effort")
		}},
		{"codex", "path-option-swallows-config", ErrEffortOverrideConflict, func(_ *testing.T, f *effortInspectorFixture) { f.insertArgv("-C", "--config") }},
		{"claude", "effort-option", ErrEffortOverrideConflict, func(_ *testing.T, f *effortInspectorFixture) { f.insertArgv("--effort", "low") }},
		{"claude", "allowed-tools-effort-element", ErrEffortOverrideConflict, func(_ *testing.T, f *effortInspectorFixture) {
			f.insertArgv("--allowedTools", "Read,EffortTool")
		}},
		{"claude", "allowed-tools-swallows-option", ErrEffortOverrideConflict, func(_ *testing.T, f *effortInspectorFixture) { f.insertArgv("--allowedTools", "-p") }},
		{"claude", "inherited-environment-conflict", ErrEffortOverrideConflict, func(_ *testing.T, f *effortInspectorFixture) {
			f.request.Invocation = executiongraph.WithExactEnvironment(f.request.Invocation, append(f.request.Invocation.Env, "CLAUDE_CODE_EFFORT_LEVEL=low"))
		}},
		{"codex", "configured-selector", ErrEffortOverrideConflict, func(t *testing.T, f *effortInspectorFixture) {
			writeFixtureFile(t, f.source(t, "project"), `model_reasoning_effort = "low"`)
		}},
		{"claude", "configured-selector", ErrEffortOverrideConflict, func(t *testing.T, f *effortInspectorFixture) {
			writeFixtureFile(t, f.source(t, "project"), `{"effortLevel":"low"}`)
		}},
		{"codex", "symlinked-source", ErrEffortConfigurationUnsafe, symlinkUser},
		{"claude", "symlinked-source", ErrEffortConfigurationUnsafe, symlinkUser},
		{"codex", "unsupported-toml", ErrEffortConfigurationUnsafe, func(t *testing.T, f *effortInspectorFixture) {
			writeFixtureFile(t, f.source(t, "managed"), "[[array]]\n")
		}},
		{"claude", "ambiguous-json", ErrEffortConfigurationUnsafe, func(t *testing.T, f *effortInspectorFixture) {
			writeFixtureFile(t, f.source(t, "managed"), `{"effortLevel":1,"effortLevel":2}`)
		}},
	}
	for _, c := range cases {
		t.Run(c.id+"/"+c.name, func(t *testing.T) {
			f := newExplicitEffortInspectorFixture(t, c.id, true)
			f.stagePaths(t, paths, true, false)
			if _, err := f.digest(); err != nil {
				t.Fatalf("effort-named stage baseline refused: %v", err)
			}
			c.apply(t, f)
			if _, err := f.digest(); !errors.Is(err, c.want) {
				t.Fatalf("got %v, want %v", err, c.want)
			}
		})
	}
	// Both canonical option spellings name the same refusal on both Runtimes.
	for _, id := range []string{"codex", "claude"} {
		if effortPathOperand(id, "--config", "/srv/effort") || effortPathOperand(id, "-c", "/srv/effort") {
			t.Fatalf("%s: config option operand exempted", id)
		}
	}
}

func TestEffortSourcesPathOperandRefusesRelativeOrNonPath(t *testing.T) {
	for _, c := range []struct {
		id, option, value string
		want              bool
	}{
		{"codex", "-C", "/srv/effort", true},
		{"codex", "--add-dir", "/srv/effort", true},
		{"claude", "--add-dir", "/srv/effort", true},
		{"claude", "-C", "/srv/effort", false},
		{"codex", "-C", "effort", false},
		{"codex", "-C", "/srv/../effort", false},
		{"codex", "--add-dir", "/srv/effort\n", false},
		{"claude", "--allowedTools", "Read,Glob,Grep,Write,Bash(/opt/effort/axiom:*)", true},
		{"claude", "--allowedTools", "Read,Bash(effort:*)", false},
		{"claude", "--allowedTools", "Read,Bash(/opt/effort/axiom)", false},
		{"codex", "--allowedTools", "Bash(/opt/effort/axiom:*)", false},
		{"claude", "--allowedTools", "--effort", false},
	} {
		if got := effortPathOperand(c.id, c.option, c.value); got != c.want {
			t.Fatalf("%s %s %q: got %v", c.id, c.option, c.value, got)
		}
	}
}

// Specification 007 AXM-22 D6: a lexically valid configured source path does
// not prove filesystem safety. Resolution below its anchor still uses the
// existing trusted boundary, which refuses a symlinked component escaping it.
// Supported hosts (macOS, Linux) provide native symlinks; setup failure fails.
func TestEffortSourcesRefuseSymlinkEscapeOfValidConfiguredPath(t *testing.T) {
	const relative = ".codex/config.toml"
	if !runtimeprofile.ValidEffortSourcePath(relative) {
		t.Fatal("fixture path must be lexically valid")
	}
	f := newEffortInspectorFixture(t, "codex")
	anchor := f.request.Roots.User
	outside := filepath.Join(f.base, "outside")
	writeFixtureFile(t, filepath.Join(outside, "config.toml"), "# no effort selector\n")
	if _, err := f.digest(); err != nil {
		t.Fatal("contained path refused:", err)
	}
	if err := os.RemoveAll(filepath.Join(anchor, ".codex")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(anchor, ".codex")); err != nil {
		t.Fatal("symlink setup failed on a supported host:", err)
	}
	if info, err := os.Lstat(filepath.Join(anchor, ".codex")); err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatal("symlink setup not effective:", err)
	}
	if _, err := f.digest(); !errors.Is(err, ErrEffortConfigurationUnsafe) {
		t.Fatal("symlink escape followed:", err)
	}
}
