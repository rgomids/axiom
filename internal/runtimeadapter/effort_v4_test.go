package runtimeadapter

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/runtimeprofile"
)

type operationalEffortFixture struct {
	request EffortSourceRequestV4
	stage   StageInvocationRequest
	child   string
}

func newOperationalEffortFixture(t *testing.T, runtimeID string, explicit bool) operationalEffortFixture {
	t.Helper()
	base, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	roots := EffortSourceRoots{User: filepath.Join(base, "home"), Project: filepath.Join(base, "repository"), Managed: filepath.Join(base, "managed")}
	child, inbox := filepath.Join(base, "child"), filepath.Join(base, "inbox")
	for _, dir := range []string{roots.User, roots.Project, roots.Managed, child, inbox, filepath.Join(base, "bin")} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	executable := filepath.Join(base, "bin", runtimeID)
	if err := os.WriteFile(executable, []byte("synthetic executable identity; never run"), 0o700); err != nil {
		t.Fatal(err)
	}
	identity, err := ExecutableIdentity(executable)
	if err != nil {
		t.Fatal(err)
	}
	effort := runtimeprofile.Effort{Mode: runtimeprofile.EffortRuntimeDefault, Value: "default"}
	if explicit {
		effort = runtimeprofile.Effort{Mode: runtimeprofile.EffortExplicit, Value: "future-effort"}
	}
	profile := runtimeprofile.ModelProfile{ID: runtimeID + "-profile", RuntimeID: runtimeID, Model: "synthetic-future-model", Effort: effort, Capabilities: []string{"read"}}
	configuration := runtimeprofile.ConfigurationV4{FormatVersion: 4, Revision: 1, Runtimes: []runtimeprofile.RuntimeV4{{ID: runtimeID, Adapter: runtimeID, Enabled: true, AllowlistedProfileIDs: []string{profile.ID}}}, ModelProfiles: []runtimeprofile.ModelProfile{profile}}
	stage := StageInvocationRequest{Profile: CommandProfile{RuntimeID: runtimeID, ModelProfileID: profile.ID, Model: profile.Model, Executable: executable}, Workspace: child, Inbox: inbox, AxiomPath: filepath.Join(base, "bin", "axiom"), RepositoryWrite: true, ParentEnvironment: []string{"HOME=" + roots.User, "PATH=/synthetic/bin"}}
	inv, err := BuildStageInvocation(stage)
	if err != nil {
		t.Fatal(err)
	}
	inv, err = withStageEffort(inv, effort)
	if err != nil {
		t.Fatal(err)
	}
	return operationalEffortFixture{request: EffortSourceRequestV4{Consented: true, Configuration: configuration, ProfileID: profile.ID, ExecutableDigest: identity, InvocationMode: "noninteractive", Roots: roots, Invocation: inv}, stage: stage, child: child}
}

func TestOperationalEffortHasNoVersionOrDeclarationPrerequisite(t *testing.T) {
	for _, runtimeID := range []string{"codex", "claude"} {
		for _, explicit := range []bool{false, true} {
			t.Run(runtimeID+"/"+map[bool]string{false: "runtime-default", true: "explicit"}[explicit], func(t *testing.T) {
				f := newOperationalEffortFixture(t, runtimeID, explicit)
				digest, err := InspectEffortSourcesV4(context.Background(), f.request)
				if err != nil || len(digest) != 64 {
					t.Fatalf("direct observation: %s %v", digest, err)
				}
				if err := VerifyChildEquivalenceV4(context.Background(), f.request, f.child); err != nil {
					t.Fatal("equivalent absent sources", err)
				}
				projection, _, err := inspectEffortV4(context.Background(), f.request)
				wire, _ := json.Marshal(projection)
				if err != nil || !strings.Contains(string(wire), "axiom-effort-source/v2/"+runtimeID) {
					t.Fatal(err)
				}
				for _, value := range []string{"proof", "precedence", "runtimeVersion", f.request.Roots.User, f.request.Roots.Project, f.request.Roots.Managed} {
					if strings.Contains(string(wire), value) {
						t.Fatal("certification or host path in projection", value)
					}
				}
			})
		}
	}
}

func TestOperationalEffortConflictsAtEveryFixedSource(t *testing.T) {
	for _, runtimeID := range []string{"codex", "claude"} {
		fixture := newOperationalEffortFixture(t, runtimeID, true)
		for index := range fixedEffortRules(runtimeID, fixture.request.Roots).Sources {
			t.Run(runtimeID+"/"+string(rune('a'+index)), func(t *testing.T) {
				f := newOperationalEffortFixture(t, runtimeID, true)
				source := fixedEffortRules(runtimeID, f.request.Roots).Sources[index]
				path := filepath.Join(source.Root, source.Relative)
				content := `model_reasoning_effort = "future-effort"`
				if runtimeID == "claude" {
					content = `{"effortLevel":"future-effort"}`
				}
				if source.Directory {
					path = filepath.Join(path, "policy.json")
				}
				write(t, path, content)
				if _, err := InspectEffortSourcesV4(context.Background(), f.request); !errors.Is(err, ErrEffortOverrideConflict) {
					t.Fatal("equal-valued competing source accepted", source.Scope, err)
				}
			})
		}
	}
	for name, content := range map[string]string{
		"environment settings":   `{"env":{"CLAUDE_CODE_EFFORT_LEVEL":"future-effort"}}`,
		"per-model setting":      `{"modelSettings":{"synthetic-future-model":{"effortLevel":"future-effort"}}}`,
		"cap":                    `{"maxEffortLevel":"future-effort"}`,
		"per-model future shape": `{"modelSettings":{"synthetic-future-model":{"futureEffortShape":"future-effort"}}}`,
	} {
		t.Run("claude/"+name, func(t *testing.T) {
			f := newOperationalEffortFixture(t, "claude", true)
			write(t, filepath.Join(f.request.Roots.User, ".claude", "settings.json"), content)
			if _, err := InspectEffortSourcesV4(context.Background(), f.request); !errors.Is(err, ErrEffortOverrideConflict) {
				t.Fatal("reachable selector accepted", err)
			}
		})
	}
	for _, content := range []string{"[profiles.fixture]\nmodel_reasoning_effort = \"high\"", "[models.new_thread]\nmodel_reasoning_effort = \"high\"", "profiles.fixture.model_reasoning_effort = \"high\""} {
		f := newOperationalEffortFixture(t, "codex", true)
		write(t, filepath.Join(f.request.Roots.User, ".codex", "config.toml"), content)
		if _, err := InspectEffortSourcesV4(context.Background(), f.request); !errors.Is(err, ErrEffortOverrideConflict) {
			t.Fatal("nested Codex selector accepted", err)
		}
	}
}

func TestOperationalEffortRefusesUnsafeSourcesAndIdentityDrift(t *testing.T) {
	for _, runtimeID := range []string{"codex", "claude"} {
		for _, scenario := range []string{"no-consent", "disabled", "unlisted", "relocation", "duplicate-home", "symlink", "malformed", "oversize", "executable-drift", "unsafe-root", "equal-roots", "inherited-effort", "argv-effort"} {
			t.Run(runtimeID+"/"+scenario, func(t *testing.T) {
				f := newOperationalEffortFixture(t, runtimeID, true)
				r := &f.request
				source := fixedEffortRules(runtimeID, r.Roots).Sources[0]
				path := filepath.Join(source.Root, source.Relative)
				switch scenario {
				case "no-consent":
					r.Consented = false
				case "disabled":
					r.Configuration.Runtimes[0].Enabled = false
				case "unlisted":
					r.ProfileID = "other-profile"
				case "relocation":
					r.Invocation.Env = append(r.Invocation.Env, "CODEX_HOME="+r.Roots.User)
				case "duplicate-home":
					r.Invocation.Env = append(r.Invocation.Env, "HOME="+r.Roots.User)
				case "symlink":
					write(t, path+"-target", "")
					if err := os.Symlink(path+"-target", path); err != nil {
						t.Fatal(err)
					}
				case "malformed":
					write(t, path, "{ malformed")
				case "oversize":
					write(t, path, strings.Repeat(" ", maxConfigFileBytes+1))
				case "executable-drift":
					if err := os.WriteFile(r.Invocation.Argv[0], []byte("different executable bytes"), 0o700); err != nil {
						t.Fatal(err)
					}
				case "unsafe-root":
					if err := os.Chmod(r.Roots.User, 0o777); err != nil {
						t.Fatal(err)
					}
				case "equal-roots":
					r.Roots.Project = r.Roots.User
				case "inherited-effort":
					r.Invocation.Env = append(r.Invocation.Env, "CLAUDE_CODE_EFFORT_LEVEL=future-effort")
					if runtimeID == "codex" {
						r.Invocation.Argv = append(r.Invocation.Argv, "-c", "model_reasoning_effort=future-effort")
					}
				case "argv-effort":
					r.Invocation.Argv = append(r.Invocation.Argv, "--effort", "future-effort")
				}
				if _, err := InspectEffortSourcesV4(context.Background(), *r); err == nil {
					t.Fatal("unsafe observation accepted")
				}
			})
		}
	}
}

func TestOperationalEffortBindsSourceBytesAndChildView(t *testing.T) {
	for _, runtimeID := range []string{"codex", "claude"} {
		f := newOperationalEffortFixture(t, runtimeID, false)
		before, err := InspectEffortSourcesV4(context.Background(), f.request)
		if err != nil {
			t.Fatal(err)
		}
		source := fixedEffortRules(runtimeID, f.request.Roots).Sources[1]
		content, other := "unrelated_setting = true\n", "unrelated_setting = false\n"
		if runtimeID == "claude" {
			content, other = `{"unrelatedSetting":true,"ultracode":true}`, `{"unrelatedSetting":false}`
		} else {
			content = "[profiles.fixture]\nunrelated_setting = true\n"
		}
		write(t, filepath.Join(source.Root, source.Relative), content)
		after, err := InspectEffortSourcesV4(context.Background(), f.request)
		if err != nil || before == after {
			t.Fatal("source presence/bytes not bound", err)
		}
		if err := VerifyChildEquivalenceV4(context.Background(), f.request, f.child); err == nil {
			t.Fatal("absent child source accepted")
		}
		write(t, filepath.Join(f.child, source.Relative), other)
		if err := VerifyChildEquivalenceV4(context.Background(), f.request, f.child); err == nil {
			t.Fatal("different child bytes accepted")
		}
		write(t, filepath.Join(f.child, source.Relative), content)
		if err := VerifyChildEquivalenceV4(context.Background(), f.request, f.child); err != nil {
			t.Fatal("equal child bytes refused", err)
		}
		projection, _, err := inspectEffortV4(context.Background(), f.request)
		wire, _ := json.Marshal(projection)
		if err != nil || bytes.Contains(wire, []byte("unrelated_setting")) || bytes.Contains(wire, []byte("unrelatedSetting")) {
			t.Fatal("raw config key/value retained", err)
		}
	}
}

func TestOperationalFrozenInvocationBindsIdentityAndProfileIntent(t *testing.T) {
	for _, runtimeID := range []string{"codex", "claude"} {
		f := newOperationalEffortFixture(t, runtimeID, true)
		profile := f.request.Configuration.ModelProfiles[0]
		pd, _ := runtimeprofile.ProfileDigest(profile)
		cd, _ := runtimeprofile.DigestV4(f.request.Configuration)
		digest := strings.Repeat("a", 64)
		frozen := runtimeprofile.ResolvedProfileV4{ProfileID: profile.ID, RuntimeID: runtimeID, Model: profile.Model, Effort: profile.Effort, ProfileDigest: pd, ConfigurationRevision: 1, ConfigurationDigest: cd, ExecutableDigest: f.request.ExecutableDigest, InvocationMode: "noninteractive", ObservationRevision: 1, ObservationDigest: digest, EffortSourceDigest: digest}
		inv, err := BuildFrozenStageInvocation(f.stage, frozen)
		if err != nil || !reflect.DeepEqual(inv, f.request.Invocation) {
			t.Fatal("frozen intent encoding", err)
		}
		frozen.ExecutableDigest = strings.Repeat("b", 64)
		if _, err := BuildFrozenStageInvocation(f.stage, frozen); err == nil {
			t.Fatal("executable drift accepted")
		}
		frozen.ExecutableDigest = f.request.ExecutableDigest
		frozen.ProfileID = "other-profile"
		if _, err := BuildFrozenStageInvocation(f.stage, frozen); err == nil {
			t.Fatal("Profile substitution accepted")
		}
	}
}

func TestOperationalReadOnlyEffortInspectsActualInbox(t *testing.T) {
	for _, runtimeID := range []string{"codex", "claude"} {
		f := newOperationalEffortFixture(t, runtimeID, true)
		f.stage.RepositoryWrite = false
		inv, err := BuildStageInvocation(f.stage)
		if err != nil || inv.CWD != f.stage.Inbox {
			t.Fatal("read-only child escaped inbox", err)
		}
		inv, _ = withStageEffort(inv, f.request.Configuration.ModelProfiles[0].Effort)
		roots, err := (StageEffortRoots{Repository: f.request.Roots.Project}).ResolveOperational(context.Background(), inv, false)
		if err != nil || roots.Project != inv.CWD || roots.User != f.request.Roots.User {
			t.Fatal("read-only roots do not reflect child", roots, err)
		}
		f.request.Roots.Project = roots.Project
		f.request.Invocation = inv
		if _, err := InspectEffortSourcesV4(context.Background(), f.request); err != nil {
			t.Fatal("read-only observation needs no qualification", err)
		}
		source := fixedEffortRules(runtimeID, f.request.Roots).Sources[1]
		content := `model_reasoning_effort = "future-effort"`
		if runtimeID == "claude" {
			content = `{"effortLevel":"future-effort"}`
		}
		write(t, filepath.Join(source.Root, source.Relative), content)
		if _, err := InspectEffortSourcesV4(context.Background(), f.request); !errors.Is(err, ErrEffortOverrideConflict) {
			t.Fatal("inbox selector was not observed", err)
		}
		roots, err = (StageEffortRoots{Repository: f.child}).ResolveOperational(context.Background(), inv, true)
		if err != nil || roots.Project != f.child {
			t.Fatal("repository-write roots changed", roots, err)
		}
	}
}

func TestOperationalBuilderRefusesInheritedEffortBeforeEnvironmentFiltering(t *testing.T) {
	for _, runtimeID := range []string{"codex", "claude"} {
		f := newOperationalEffortFixture(t, runtimeID, true)
		for _, variable := range []string{"CODEX_HOME", "CLAUDE_CONFIG_DIR", "codex_home", "claude_config_dir"} {
			request := f.stage
			request.ParentEnvironment = append(append([]string(nil), request.ParentEnvironment...), variable+"="+f.request.Roots.User)
			if _, err := BuildStageInvocation(request); !errors.Is(err, ErrEffortConfigurationUnsafe) {
				t.Fatal("relocation was silently removed or accepted", variable, err)
			}
		}
		if runtimeID != "claude" {
			continue
		}
		for _, variable := range []string{"CLAUDE_CODE_EFFORT_LEVEL", "claude_code_effort_level"} {
			request := f.stage
			request.ParentEnvironment = append(append([]string(nil), request.ParentEnvironment...), variable+"=future-effort")
			if _, err := BuildStageInvocation(request); !errors.Is(err, ErrEffortOverrideConflict) {
				t.Fatal("equal-valued inherited effort was silently removed", variable, err)
			}
		}
	}
}

func TestOperationalEffortSeparatesInvalidSyntaxFromUnsafeStorage(t *testing.T) {
	for _, id := range []string{"codex", "claude"} {
		t.Run(id, func(t *testing.T) {
			f := newOperationalEffortFixture(t, id, false)
			source := fixedEffortRules(id, f.request.Roots).Sources[0]
			path := filepath.Join(source.Root, source.Relative)
			if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte("{ malformed"), 0600); err != nil {
				t.Fatal(err)
			}
			if _, err := InspectEffortSourcesV4(context.Background(), f.request); !errors.Is(err, runtimeprofile.ErrConfigurationInvalid) {
				t.Fatal("invalid syntax lost", err)
			}
			if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(path+"-missing", path); err != nil {
				t.Fatal(err)
			}
			if _, err := InspectEffortSourcesV4(context.Background(), f.request); !errors.Is(err, ErrEffortConfigurationUnsafe) || errors.Is(err, runtimeprofile.ErrConfigurationInvalid) {
				t.Fatal("unsafe storage became syntax failure", err)
			}
		})
	}
}
