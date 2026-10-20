package runtimeadapter

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEffortSourcesConsentAndNoOverride(t *testing.T) {
	for _, id := range []string{"codex", "claude"} {
		t.Run(id, func(t *testing.T) {
			f := newEffortInspectorFixture(t, id)
			before, e := f.digest()
			if e != nil || len(before) != 64 {
				t.Fatal(before, e)
			}
			f.request.Consented = false
			if _, e = f.digest(); !errors.Is(e, ErrEffortConfigurationUnsafe) {
				t.Fatal("unconsented", e)
			}
			f.request.Consented = true
			path := f.source(t, "user")
			content := `model_reasoning_effort = "private-canary"`
			if id == "claude" {
				content = `{"effortLevel":"private-canary"}`
			}
			if e = os.WriteFile(path, []byte(content), 0o600); e != nil {
				t.Fatal(e)
			}
			if _, e = f.digest(); !errors.Is(e, ErrEffortOverrideConflict) || strings.Contains(e.Error(), "private-canary") {
				t.Fatal("override/redaction", e)
			}
			after, _ := os.ReadFile(path)
			if string(after) != content {
				t.Fatal("inspection mutated configuration")
			}
		})
	}
}

func TestEffortSourcesMalformedAndSymlinkRefusal(t *testing.T) {
	f := newEffortInspectorFixture(t, "claude")
	path := f.source(t, "user")
	for _, content := range []string{`{"effortLevel":1,"effortLevel":2}`, `{`, `null`} {
		if e := os.WriteFile(path, []byte(content), 0o600); e != nil {
			t.Fatal(e)
		}
		if _, e := f.digest(); !errors.Is(e, ErrEffortConfigurationUnsafe) {
			t.Fatalf("malformed accepted %q: %v", content, e)
		}
	}
	if e := os.Rename(path, path+".saved"); e != nil {
		t.Fatal(e)
	}
	if e := os.Symlink(path+".saved", path); e != nil {
		t.Fatalf("native symlink setup failed on a supported host: %v", e)
	}
	if _, e := f.digest(); !errors.Is(e, ErrEffortConfigurationUnsafe) {
		t.Fatal("followed symlink", e)
	}
}

func TestEffortSourcesBindExactBytesAndObjectAncestors(t *testing.T) {
	for _, kind := range []string{"bytes", "file", "ancestor", "absent-parent"} {
		t.Run(kind, func(t *testing.T) {
			f := newEffortInspectorFixture(t, "claude")
			path := f.source(t, "user")
			root := filepath.Dir(path)
			if kind == "absent-parent" {
				os.Remove(path)
			}
			before, err := f.digest()
			if err != nil {
				t.Fatal(err)
			}
			switch kind {
			case "bytes":
				os.WriteFile(path, []byte(`{"unrelated":"changed"}`), 0o600)
			case "file":
				os.Rename(path, path+".saved")
				os.WriteFile(path, []byte(`{}`), 0o600)
			case "ancestor", "absent-parent":
				os.Rename(root, root+".saved")
				os.Mkdir(root, 0o700)
				if kind == "ancestor" {
					os.WriteFile(path, []byte(`{}`), 0o600)
				}
			}
			after, err := f.digest()
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
	f := newEffortInspectorFixture(t, "claude")
	dropIn := f.source(t, "managed-drop-in")
	before, err := f.digest()
	if err != nil {
		t.Fatal(err)
	}
	inserted := filepath.Join(dropIn, "new.json")
	os.WriteFile(inserted, []byte(`{}`), 0o600)
	after, err := f.digest()
	if err != nil || before == after {
		t.Fatalf("inventory drift: %v", err)
	}
	os.WriteFile(inserted, []byte(`{"effortLevel":"private-canary"}`), 0o600)
	if _, err := f.digest(); !errors.Is(err, ErrEffortOverrideConflict) {
		t.Fatalf("unobserved drop-in: %v", err)
	}
	rules := f.rules()
	rules.Sources[len(rules.Sources)-1].Directory = false
	if _, err := f.observe(rules); !errors.Is(err, ErrEffortConfigurationUnsafe) {
		t.Fatal("file-only drop-in scope accepted", err)
	}
}

func TestEffortSourcesDropInUnknownAndSymlinkRefusal(t *testing.T) {
	for _, kind := range []string{"unknown", "directory", "symlink", "oversized-inventory"} {
		t.Run(kind, func(t *testing.T) {
			f := newEffortInspectorFixture(t, "claude")
			dropIn := f.source(t, "managed-drop-in")
			path := filepath.Join(dropIn, "added.json")
			switch kind {
			case "unknown":
				os.WriteFile(filepath.Join(dropIn, "unknown.txt"), []byte("unknown"), 0o600)
			case "directory":
				os.Mkdir(path, 0o700)
			case "symlink":
				if err := os.Symlink(filepath.Join(dropIn, "10-base.json"), path); err != nil {
					t.Fatalf("native symlink setup failed on a supported host: %v", err)
				}
			case "oversized-inventory":
				for n := 0; n < 65; n++ {
					os.WriteFile(filepath.Join(dropIn, fmt.Sprintf("%d.json", n)), []byte(`{}`), 0o600)
				}
			}
			if _, err := f.digest(); !errors.Is(err, ErrEffortConfigurationUnsafe) {
				t.Fatalf("unsupported inventory accepted: %v", err)
			}
		})
	}
}

func TestEffortSourcesTOMLNarrowGrammar(t *testing.T) {
	invalid := []string{`a = [`, `a = arbitrary-bare-text`, `a = { x = 1 }`, `a = [1,2]`, `a = 1` + "\n" + `a = 2`, `a.b = 1` + "\n" + `[a]` + "\n" + `b = 2`, `[a]` + "\n" + `[a]`, `a = 1` + "\n" + `[a]`, `[[array]]`, `a = 'closed' trailing`, `a = "\x61"`, `a = 2026-10-11`, `"\x61" = true`, `[profiles.test]`}
	for _, wire := range invalid {
		if _, err := tomlEffortSelectorsBounded([]byte(wire), false); !errors.Is(err, ErrEffortConfigurationUnsafe) {
			t.Fatalf("malformed/unsupported TOML accepted %q: %v", wire, err)
		}
	}
	for _, wire := range []string{`a = "valid"`, `a = 'literal'`, `a = true`, `a = -12`, `a = 1.25e+2`, `[a]` + "\n" + `b = 2`, `a = "# literal" # comment`} {
		if _, err := tomlEffortSelectorsBounded([]byte(wire), false); err != nil {
			t.Fatalf("scalar refused %q: %v", wire, err)
		}
	}
	selectors, err := tomlEffortSelectorsBounded([]byte(`"model_reasoning_effort" = "high"`), false)
	if err != nil || len(selectors) != 1 {
		t.Fatal("quoted effort selector missed")
	}
	if selectors, err := tomlEffortSelectorsBounded([]byte("[profiles.test]\nmodel_reasoning_effort = \"high\""), true); err != nil || len(selectors) != 1 {
		t.Fatal("nested profile selector missed", err)
	}
}

func TestEffortSourcesOversizedRefusal(t *testing.T) {
	f := newEffortInspectorFixture(t, "claude")
	os.WriteFile(f.source(t, "user"), make([]byte, maxConfigFileBytes+1), 0o600)
	if _, err := f.digest(); !errors.Is(err, ErrEffortConfigurationUnsafe) {
		t.Fatal("oversized source accepted")
	}
}
