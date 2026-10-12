package runtimeadapter

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/executiongraph"
	"github.com/rgomids/axiom/internal/runtimeprofile"
)

func effortInspectionFixture(t *testing.T, id string) (EffortInspection, runtimeprofile.ResolvedProfile, executiongraph.Invocation) {
	t.Helper()
	root := t.TempDir()
	root, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	exe := filepath.Join(root, "synthetic-executable")
	if e := os.WriteFile(exe, []byte("synthetic identity; never execute\n"), 0700); e != nil {
		t.Fatal(e)
	}
	identity, e := ExecutableIdentity(exe)
	if e != nil {
		t.Fatal(e)
	}
	hash := strings.Repeat("a", 64)
	p := runtimeprofile.ResolvedProfile{ProfileID: "test-profile", RuntimeID: id, Model: "operator-model", Effort: runtimeprofile.Effort{Mode: "runtime-default", Value: "default"}, ProfileDigest: hash, ConfigurationRevision: 1, ConfigurationDigest: hash, ExecutableDigest: identity, RuntimeVersion: "synthetic-1", InvocationMode: "noninteractive", ObservationRevision: 1, ObservationDigest: hash, EffortSourceDigest: hash, CapabilityProofReference: "synthetic-proof"}
	d := EffortDeclaration{RuntimeID: id, RuntimeVersion: p.RuntimeVersion, InvocationMode: p.InvocationMode, ExecutableDigest: identity, ProofReference: "synthetic-source-proof", PrecedenceProven: true}
	scopes := []string{"user", "project", "managed"}
	if id == "claude" {
		scopes = append(scopes, "project-local", "managed-drop-in")
		d.JSONKeys = []string{"fixtureEffortKey"}
		d.EnvironmentKeys = []string{"CLAUDE_CODE_EFFORT_LEVEL"}
	}
	for _, scope := range scopes {
		dir := filepath.Join(root, scope)
		if e := os.Mkdir(dir, 0700); e != nil {
			t.Fatal(e)
		}
		format, name, content := "toml", "config.toml", "# no effort selector\n"
		if id == "claude" {
			format, name, content = "json", "settings.json", "{}"
		}
		if e := os.WriteFile(filepath.Join(dir, name), []byte(content), 0600); e != nil {
			t.Fatal(e)
		}
		source := EffortSource{Scope: scope, Root: dir, Relative: name, Format: format}
		if scope == "managed-drop-in" {
			source.Relative = "."
			source.Directory = true
		}
		d.Sources = append(d.Sources, source)
	}
	inv := executiongraph.Invocation{RuntimeID: id, Argv: []string{exe, "exec", "--model", p.Model}, CWD: root, Env: []string{"PATH=fixture"}}
	if id == "claude" {
		inv.Argv[1] = "--print"
	}
	return EffortInspection{Consented: true, Declaration: d}, p, inv
}
func TestEffortSourcesConsentExactVersionAndNoOverride(t *testing.T) {
	for _, id := range []string{"codex", "claude"} {
		t.Run(id, func(t *testing.T) {
			i, p, inv := effortInspectionFixture(t, id)
			ctx := context.Background()
			before, e := i.CheckEffortSources(ctx, p, inv)
			if e != nil || len(before) != 64 {
				t.Fatal(before, e)
			}
			i.Consented = false
			if _, e = i.CheckEffortSources(ctx, p, inv); !errors.Is(e, ErrEffortConfigurationUnproven) {
				t.Fatal("unconsented", e)
			}
			i.Consented = true
			i.Declaration.RuntimeVersion = "other"
			if _, e = i.CheckEffortSources(ctx, p, inv); e == nil {
				t.Fatal("wrong version accepted")
			}
			i.Declaration.RuntimeVersion = p.RuntimeVersion
			i.Declaration.PrecedenceProven = false
			if _, e = i.CheckEffortSources(ctx, p, inv); e == nil {
				t.Fatal("precedence invented")
			}
			i.Declaration.PrecedenceProven = true
			source := i.Declaration.Sources[0]
			path := filepath.Join(source.Root, source.Relative)
			content := `model_reasoning_effort = "private-canary"`
			if id == "claude" {
				content = `{"fixtureEffortKey":"private-canary"}`
			}
			if e = os.WriteFile(path, []byte(content), 0600); e != nil {
				t.Fatal(e)
			}
			if _, e = i.CheckEffortSources(ctx, p, inv); !errors.Is(e, ErrEffortOverrideConflict) || strings.Contains(e.Error(), "private-canary") {
				t.Fatal("override/redaction", e)
			}
			after, _ := os.ReadFile(path)
			if string(after) != content {
				t.Fatal("inspection mutated configuration")
			}
		})
	}
}
func TestEffortSourcesOnlyCanonicalExplicitOverride(t *testing.T) {
	for _, id := range []string{"codex", "claude"} {
		t.Run(id, func(t *testing.T) {
			i, p, inv := effortInspectionFixture(t, id)
			p.Effort = runtimeprofile.Effort{Mode: "explicit", Value: "operator-token"}
			if id == "codex" {
				inv.Argv = append(inv.Argv, "--config", "model_reasoning_effort=operator-token")
			} else {
				inv.Env = append(inv.Env, "CLAUDE_CODE_EFFORT_LEVEL=operator-token")
			}
			if _, e := i.CheckEffortSources(context.Background(), p, inv); e != nil {
				t.Fatal(e)
			}
			inv.Env = append(inv.Env, "CLAUDE_CODE_EFFORT_LEVEL=other")
			if id == "codex" {
				i.Declaration.EnvironmentKeys = []string{"CLAUDE_CODE_EFFORT_LEVEL"}
			}
			if _, e := i.CheckEffortSources(context.Background(), p, inv); !errors.Is(e, ErrEffortOverrideConflict) {
				t.Fatal("inherited conflict", e)
			}
		})
	}
}
func TestEffortSourcesMalformedAndSymlinkRefusal(t *testing.T) {
	i, p, inv := effortInspectionFixture(t, "claude")
	source := i.Declaration.Sources[0]
	path := filepath.Join(source.Root, source.Relative)
	for _, content := range []string{`{"fixtureEffortKey":1,"fixtureEffortKey":2}`, `{`, `null`} {
		if e := os.WriteFile(path, []byte(content), 0600); e != nil {
			t.Fatal(e)
		}
		if _, e := i.CheckEffortSources(context.Background(), p, inv); !errors.Is(e, ErrEffortConfigurationUnproven) {
			t.Fatalf("malformed accepted %q: %v", content, e)
		}
	}
	if e := os.Rename(path, path+".saved"); e != nil {
		t.Fatal(e)
	}
	if e := os.Symlink(path+".saved", path); e != nil {
		t.Skip("native symlink unavailable")
	}
	if _, e := i.CheckEffortSources(context.Background(), p, inv); e == nil {
		t.Fatal("followed symlink")
	}
}

func TestEffortSourcesBindExactBytesAndObjectAncestors(t *testing.T) {
	for _, kind := range []string{"bytes", "file", "ancestor", "absent-parent"} {
		t.Run(kind, func(t *testing.T) {
			i, p, inv := effortInspectionFixture(t, "claude")
			source := i.Declaration.Sources[0]
			path := filepath.Join(source.Root, source.Relative)
			if kind == "absent-parent" {
				os.Remove(path)
			}
			before, err := i.CheckEffortSources(context.Background(), p, inv)
			if err != nil {
				t.Fatal(err)
			}
			switch kind {
			case "bytes":
				os.WriteFile(path, []byte(`{"unrelated":"changed"}`), 0600)
			case "file":
				os.Rename(path, path+".saved")
				os.WriteFile(path, []byte(`{}`), 0600)
			case "ancestor", "absent-parent":
				os.Rename(source.Root, source.Root+".saved")
				os.Mkdir(source.Root, 0700)
				if kind == "ancestor" {
					os.WriteFile(path, []byte(`{}`), 0600)
				}
			}
			after, err := i.CheckEffortSources(context.Background(), p, inv)
			if err != nil {
				t.Fatal(err)
			}
			if before == after {
				t.Fatal("source drift did not change exact digest")
			}
		})
	}
}
func TestEffortSourcesManagedDropInInventory(t *testing.T) {
	i, p, inv := effortInspectionFixture(t, "claude")
	source := i.Declaration.Sources[len(i.Declaration.Sources)-1]
	before, err := i.CheckEffortSources(context.Background(), p, inv)
	if err != nil {
		t.Fatal(err)
	}
	inserted := filepath.Join(source.Root, "new.json")
	os.WriteFile(inserted, []byte(`{}`), 0600)
	after, err := i.CheckEffortSources(context.Background(), p, inv)
	if err != nil || before == after {
		t.Fatalf("inventory drift: %v", err)
	}
	os.WriteFile(inserted, []byte(`{"fixtureEffortKey":"private-canary"}`), 0600)
	if _, err := i.CheckEffortSources(context.Background(), p, inv); !errors.Is(err, ErrEffortOverrideConflict) {
		t.Fatalf("unobserved drop-in: %v", err)
	}
	i.Declaration.Sources[len(i.Declaration.Sources)-1].Directory = false
	if _, err := i.CheckEffortSources(context.Background(), p, inv); !errors.Is(err, ErrEffortConfigurationUnproven) {
		t.Fatal("file-only drop-in scope accepted")
	}
}
func TestEffortSourcesDropInUnknownAndSymlinkRefusal(t *testing.T) {
	for _, kind := range []string{"unknown", "directory", "symlink", "oversized-inventory"} {
		t.Run(kind, func(t *testing.T) {
			i, p, inv := effortInspectionFixture(t, "claude")
			source := i.Declaration.Sources[len(i.Declaration.Sources)-1]
			path := filepath.Join(source.Root, "added.json")
			switch kind {
			case "unknown":
				os.WriteFile(filepath.Join(source.Root, "unknown.txt"), []byte("unknown"), 0600)
			case "directory":
				os.Mkdir(path, 0700)
			case "symlink":
				if err := os.Symlink(filepath.Join(source.Root, "settings.json"), path); err != nil {
					t.Skip("native symlink unavailable")
				}
			case "oversized-inventory":
				for n := 0; n < 65; n++ {
					os.WriteFile(filepath.Join(source.Root, fmt.Sprintf("%d.json", n)), []byte(`{}`), 0600)
				}
			}
			if _, err := i.CheckEffortSources(context.Background(), p, inv); !errors.Is(err, ErrEffortConfigurationUnproven) {
				t.Fatalf("unsupported inventory accepted: %v", err)
			}
		})
	}
}
func TestEffortSourcesTOMLNarrowGrammar(t *testing.T) {
	invalid := []string{`a = [`, `a = arbitrary-bare-text`, `a = { x = 1 }`, `a = [1,2]`, `a = 1` + "\n" + `a = 2`, `a.b = 1` + "\n" + `[a]` + "\n" + `b = 2`, `[a]` + "\n" + `[a]`, `a = 1` + "\n" + `[a]`, `[[array]]`, `a = 'closed' trailing`, `a = "\x61"`, `a = 2026-10-11`, `"\x61" = true`, `[profiles.test]`}
	for _, wire := range invalid {
		if _, err := tomlEffortSelectors([]byte(wire)); !errors.Is(err, ErrEffortConfigurationUnproven) {
			t.Fatalf("malformed/unsupported TOML accepted %q: %v", wire, err)
		}
	}
	for _, wire := range []string{`a = "valid"`, `a = 'literal'`, `a = true`, `a = -12`, `a = 1.25e+2`, `[a]` + "\n" + `b = 2`, `a = "# literal" # comment`} {
		if _, err := tomlEffortSelectors([]byte(wire)); err != nil {
			t.Fatalf("scalar refused %q: %v", wire, err)
		}
	}
	selectors, err := tomlEffortSelectors([]byte(`"model_reasoning_effort" = "high"`))
	if err != nil || len(selectors) != 1 {
		t.Fatal("quoted effort selector missed")
	}
}

func TestEffortSourcesNormalizedEnvironmentRefusal(t *testing.T) {
	i, p, inv := effortInspectionFixture(t, "claude")
	inv.Env = append(inv.Env, "claude_code_effort_level=private-canary")
	if _, err := i.CheckEffortSources(context.Background(), p, inv); !errors.Is(err, ErrEffortOverrideConflict) || strings.Contains(err.Error(), "private-canary") {
		t.Fatalf("normalized selector accepted: %v", err)
	}
	i, p, inv = effortInspectionFixture(t, "claude")
	i.Declaration.EnvironmentKeys = append(i.Declaration.EnvironmentKeys, "claude_code_effort_level")
	if _, err := i.CheckEffortSources(context.Background(), p, inv); !errors.Is(err, ErrEffortConfigurationUnproven) {
		t.Fatalf("duplicate declaration accepted: %v", err)
	}
}
func TestEffortSourcesOversizedAndUncleanPathRefusal(t *testing.T) {
	i, p, inv := effortInspectionFixture(t, "claude")
	source := i.Declaration.Sources[0]
	os.WriteFile(filepath.Join(source.Root, source.Relative), make([]byte, maxConfigFileBytes+1), 0600)
	if _, err := i.CheckEffortSources(context.Background(), p, inv); !errors.Is(err, ErrEffortConfigurationUnproven) {
		t.Fatal("oversized source accepted")
	}
	i.Declaration.Sources[0].Root = source.Root + string(filepath.Separator) + ".." + string(filepath.Separator) + filepath.Base(source.Root)
	if recognizedEffortSource("claude", i.Declaration.Sources[0]) {
		t.Fatal("unclean root accepted")
	}
}
