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

// Specification 007 AXM-22 BLD-001B.3a: the PE-05a argv scan must not treat
// the spelling of a stage path as an effort override. Only the clean absolute
// operand of the adapter's own path options is exempt; every override stays
// refused and no path enters the EffortSourceDigest.

type effortStagePaths struct{ workspace, inbox, axiom string }

// stageEffortFixture binds the inspector fixture to a canonical stage
// invocation: an executable named after the Runtime and the adapter's
// supported version, so BuildStageInvocation and withStageEffort produce argv.
func stageEffortFixture(t *testing.T, id string, effort runtimeprofile.Effort) *effortInspectorFixture {
	t.Helper()
	f := newEffortInspectorFixture(t, id)
	f.exe = filepath.Join(f.base, "bin", id)
	writeFixtureFile(t, f.exe, "synthetic "+id+"; never execute\n")
	if err := os.Chmod(f.exe, 0o700); err != nil {
		t.Fatal(err)
	}
	identity, err := ExecutableIdentity(f.exe)
	if err != nil {
		t.Fatal(err)
	}
	d := f.declaration()
	d.RuntimeVersion, d.ExecutableDigest = bindingVersions[id], identity
	f.req.Observed.RuntimeVersion, f.req.Observed.ExecutableDigest = d.RuntimeVersion, identity
	f.profile().Effort = effort
	return f
}

func (f *effortInspectorFixture) stage(t *testing.T, paths effortStagePaths, write, process bool) {
	t.Helper()
	profile := f.profile()
	inv, err := BuildStageInvocation(StageInvocationRequest{Profile: CommandProfile{RuntimeID: profile.RuntimeID, ModelProfileID: profile.ID, Executable: f.exe, Model: profile.Model}, Version: f.req.Observed.RuntimeVersion, Workspace: paths.workspace, Inbox: paths.inbox, AxiomPath: paths.axiom, Process: process, RepositoryWrite: write, ParentEnvironment: []string{"PATH=fixture"}})
	if err != nil {
		t.Fatalf("stage invocation %+v: %v", paths, err)
	}
	if f.req.Invocation, err = withStageEffort(inv, profile.Effort); err != nil {
		t.Fatal(err)
	}
}

var effortEfforts = map[string]runtimeprofile.Effort{
	"explicit":        {Mode: runtimeprofile.EffortExplicit, Value: "high"},
	"runtime-default": {Mode: runtimeprofile.EffortRuntimeDefault, Value: "default"},
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
		for effortName, effort := range effortEfforts {
			for _, write := range []bool{false, true} {
				for _, process := range []bool{false, true} {
					f := stageEffortFixture(t, id, effort)
					f.stage(t, neutral, write, process)
					want, err := compareWithPE05a(t, f)
					if err != nil {
						t.Fatalf("%s/%s write=%v process=%v neutral baseline refused: %v", id, effortName, write, process, err)
					}
					// The real temporary root carries this test's name ("Effort").
					cases["temp-root"] = effortStagePaths{filepath.Join(f.base, "work"), filepath.Join(f.base, "inbox"), filepath.Join(f.base, "bin", "axiom")}
					for name, paths := range cases {
						f.stage(t, paths, write, process)
						got, err := compareWithPE05a(t, f)
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
	explicit := effortEfforts["explicit"]
	insert := func(f *effortInspectorFixture, tokens ...string) {
		argv := f.req.Invocation.Argv
		if f.req.Observed.RuntimeID == "codex" { // keep the canonical trailing --config pair last
			argv = append(append(append([]string{}, argv[:len(argv)-2]...), tokens...), argv[len(argv)-2:]...)
		} else {
			argv = append(append([]string{}, argv...), tokens...)
		}
		f.req.Invocation.Argv = argv
	}
	cases := []struct {
		id, name string
		want     error
		apply    func(*testing.T, *effortInspectorFixture)
	}{
		{"codex", "conflicting-config-option", ErrEffortOverrideConflict, func(_ *testing.T, f *effortInspectorFixture) {
			insert(f, "--config", "model_reasoning_effort=low")
		}},
		{"codex", "short-config-option", ErrEffortOverrideConflict, func(_ *testing.T, f *effortInspectorFixture) { insert(f, "-c", "model_reasoning_effort=low") }},
		{"codex", "profile-option", ErrEffortOverrideConflict, func(_ *testing.T, f *effortInspectorFixture) { insert(f, "--profile", "fast") }},
		{"codex", "non-path-add-dir-operand", ErrEffortOverrideConflict, func(_ *testing.T, f *effortInspectorFixture) {
			insert(f, "--add-dir", "model_reasoning_effort=low")
		}},
		{"codex", "relative-cd-operand", ErrEffortOverrideConflict, func(_ *testing.T, f *effortInspectorFixture) { insert(f, "-C", "effort/work") }},
		{"codex", "unclean-add-dir-operand", ErrEffortOverrideConflict, func(_ *testing.T, f *effortInspectorFixture) {
			insert(f, "--add-dir", "/srv/x/../effort")
		}},
		{"codex", "path-option-swallows-config", ErrEffortOverrideConflict, func(_ *testing.T, f *effortInspectorFixture) { insert(f, "-C", "--config") }},
		{"claude", "effort-option", ErrEffortOverrideConflict, func(_ *testing.T, f *effortInspectorFixture) { insert(f, "--effort", "low") }},
		{"claude", "allowed-tools-effort-element", ErrEffortOverrideConflict, func(_ *testing.T, f *effortInspectorFixture) {
			insert(f, "--allowedTools", "Read,EffortTool")
		}},
		{"claude", "allowed-tools-swallows-option", ErrEffortOverrideConflict, func(_ *testing.T, f *effortInspectorFixture) { insert(f, "--allowedTools", "-p") }},
		{"claude", "inherited-environment-conflict", ErrEffortOverrideConflict, func(_ *testing.T, f *effortInspectorFixture) {
			f.req.Invocation = executiongraph.WithExactEnvironment(f.req.Invocation, append(f.req.Invocation.Env, "CLAUDE_CODE_EFFORT_LEVEL=low"))
		}},
		{"codex", "declared-environment-conflict", ErrEffortOverrideConflict, func(_ *testing.T, f *effortInspectorFixture) {
			f.declaration().EnvironmentKeys = []string{"FIXTURE_EFFORT_LEVEL"}
			f.req.Invocation = executiongraph.WithExactEnvironment(f.req.Invocation, append(f.req.Invocation.Env, "FIXTURE_EFFORT_LEVEL=low"))
		}},
		{"codex", "configured-selector", ErrEffortOverrideConflict, func(t *testing.T, f *effortInspectorFixture) {
			writeFixtureFile(t, f.source(t, "project"), `model_reasoning_effort = "low"`)
		}},
		{"claude", "configured-selector", ErrEffortOverrideConflict, func(t *testing.T, f *effortInspectorFixture) {
			writeFixtureFile(t, f.source(t, "project"), `{"fixtureEffortKey":"low"}`)
		}},
		{"codex", "symlinked-source", ErrEffortConfigurationUnproven, func(t *testing.T, f *effortInspectorFixture) {
			path := f.source(t, "user")
			if err := os.Rename(path, path+".saved"); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(path+".saved", path); err != nil {
				t.Fatalf("native symlink setup failed on a supported host: %v", err)
			}
		}},
		{"claude", "symlinked-source", ErrEffortConfigurationUnproven, func(t *testing.T, f *effortInspectorFixture) {
			path := f.source(t, "user")
			if err := os.Rename(path, path+".saved"); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(path+".saved", path); err != nil {
				t.Fatalf("native symlink setup failed on a supported host: %v", err)
			}
		}},
		{"codex", "unsupported-toml", ErrEffortConfigurationUnproven, func(t *testing.T, f *effortInspectorFixture) {
			writeFixtureFile(t, f.source(t, "managed"), "[[array]]\n")
		}},
		{"claude", "ambiguous-json", ErrEffortConfigurationUnproven, func(t *testing.T, f *effortInspectorFixture) {
			writeFixtureFile(t, f.source(t, "managed"), `{"fixtureEffortKey":1,"fixtureEffortKey":2}`)
		}},
	}
	for _, c := range cases {
		t.Run(c.id+"/"+c.name, func(t *testing.T) {
			f := stageEffortFixture(t, c.id, explicit)
			f.stage(t, paths, true, false)
			if _, err := compareWithPE05a(t, f); err != nil {
				t.Fatalf("effort-named stage baseline refused: %v", err)
			}
			c.apply(t, f)
			if _, err := compareWithPE05a(t, f); !errors.Is(err, c.want) {
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
