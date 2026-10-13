package validatorpolicy

import (
	"bytes"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func examplePolicy(t *testing.T) Policy {
	t.Helper()
	path, err := filepath.Abs("synthetic-validator")
	if err != nil {
		t.Fatal(err)
	}
	d := Definition{FormatVersion: 1, ProjectID: "project-1", Commands: []Command{{Reference: "test", Argv: []string{path, "test", "./..."}, Env: []string{"LC_ALL=C"}, OutputMax: 4096}}}
	preview, err := PreviewDigest(d)
	if err != nil {
		t.Fatal(err)
	}
	return Policy{Definition: d, Review: Review{PreviewDigest: preview, ActorKind: "human-operator", Actor: "operator", ReviewedAt: time.Unix(10, 0).UTC(), AuthorityRef: strings.Repeat("a", 64)}}
}
func TestPolicyStrictRoundTripAndDigest(t *testing.T) {
	p := examplePolicy(t)
	wire, err := Encode(p)
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := Decode(wire)
	if err != nil {
		t.Fatal(err)
	}
	again, err := Encode(loaded)
	if err != nil || !bytes.Equal(wire, again) {
		t.Fatal(err)
	}
	id, err := Digest(p)
	if err != nil {
		t.Fatal(err)
	}
	commands, err := Commands(loaded, id)
	if err != nil || commands[0].Reference != "test" {
		t.Fatal(commands, err)
	}
	commands[0].Argv[1] = "mutated"
	commands[0].Env[0] = "changed"
	if loaded.Commands[0].Argv[1] != "test" || loaded.Commands[0].Env[0] != "LC_ALL=C" {
		t.Fatal("aliased commands")
	}
	if _, err := Commands(loaded, strings.Repeat("b", 64)); !errors.Is(err, ErrDigest) {
		t.Fatal(err)
	}
	for _, edit := range []func(map[string]any){
		func(m map[string]any) { m["extra"] = true }, func(m map[string]any) { m["formatVersion"] = 2 }, func(m map[string]any) { delete(m, "review") }, func(m map[string]any) { m["ProjectId"] = m["projectId"]; delete(m, "projectId") }, func(m map[string]any) { m["review"].(map[string]any)["actorKind"] = "agent-record" }, func(m map[string]any) { m["commands"].([]any)[0].(map[string]any)["argv"] = []string{"/unreviewed"} }, func(m map[string]any) { m["commands"].([]any)[0].(map[string]any)["outputMax"] = nil },
	} {
		var m map[string]any
		if err := json.Unmarshal(wire, &m); err != nil {
			t.Fatal(err)
		}
		edit(m)
		bad, _ := json.Marshal(m)
		if _, err := Decode(bad); err == nil {
			t.Fatal("invalid policy accepted", string(bad))
		}
	}
	duplicate := bytes.Replace(wire, []byte(`"formatVersion":1`), []byte(`"formatVersion":1,"formatVersion":1`), 1)
	if _, err := Decode(duplicate); err == nil {
		t.Fatal("duplicate")
	}
}
func TestPolicyRejectsShellArgvAndInvalidCommands(t *testing.T) {
	for _, shell := range []string{"sh", "bash", "dash", "zsh", "ksh", "csh", "tcsh", "fish", "ash", "busybox", "env", "cmd", "powershell", "pwsh", "bash.exe"} {
		p := examplePolicy(t)
		path, _ := filepath.Abs(shell)
		p.Commands[0].Argv = []string{path, "-c", "echo injected"}
		if _, err := PreviewDigest(p.Definition); err == nil {
			t.Fatal(shell)
		}
	}
	for _, edit := range []func(*Definition){
		func(d *Definition) { d.Commands[0].Argv[0] = "relative" }, func(d *Definition) { d.Commands[0].Argv = nil }, func(d *Definition) { d.Commands[0].Argv = append(d.Commands[0].Argv, "bad\x00value") }, func(d *Definition) { d.Commands[0].Env = []string{"LC_ALL=C", "LC_ALL=D"} }, func(d *Definition) { d.Commands[0].OutputMax = 0 }, func(d *Definition) { d.Commands = append(d.Commands, d.Commands[0]) }, func(d *Definition) { d.ProjectID = "../escape" },
	} {
		p := examplePolicy(t)
		edit(&p.Definition)
		if _, err := PreviewDigest(p.Definition); err == nil {
			t.Fatal("invalid definition accepted")
		}
	}
}
