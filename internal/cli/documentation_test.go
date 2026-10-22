package cli

import (
	"bytes"
	"flag"
	"strings"
	"testing"
)

func TestDocumentationRenderingDeterministicAndComplete(t *testing.T) {
	var first, second bytes.Buffer
	if err := RenderDocumentationCLI(&first, "test", strings.Repeat("a", 40)); err != nil {
		t.Fatal(err)
	}
	if err := RenderDocumentationCLI(&second, "test", strings.Repeat("a", 40)); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first.Bytes(), second.Bytes()) {
		t.Fatal("CLI reference changed between renders")
	}
	var leaves, flags int
	documentationWalk(&publicCommands, nil, func(command *commandDefinition, path []string) {
		if command.operation == "" {
			return
		}
		leaves++
		name := "axiom " + strings.Join(path, " ")
		if !strings.Contains(first.String(), "## `"+name+"`\n") {
			t.Errorf("reference missing %s", name)
		}
		var help bytes.Buffer
		if renderHelp(&help, command, path) != ExitSuccess {
			t.Fatalf("help render failed: %s", name)
		}
		commandFlagSet(command.operation).VisitAll(func(f *flag.Flag) {
			flags++
			if !strings.Contains(help.String(), "--"+f.Name) {
				t.Errorf("help missing flag %s on %s", f.Name, name)
			}
			if !strings.Contains(first.String(), "Default values:\n  --"+f.Name+": "+f.DefValue) {
				t.Errorf("reference missing default for %s", f.Name)
			}
		})
	})
	if leaves != 106 || flags != 549 {
		t.Fatalf("public tree drifted from the accepted coverage map: leaves=%d flags=%d", leaves, flags)
	}
	if leaves == 0 || flags == 0 {
		t.Fatalf("coverage is empty: leaves=%d flags=%d", leaves, flags)
	}
	pages, err := DocumentationPages("test", strings.Repeat("a", 40))
	if err != nil {
		t.Fatal(err)
	}
	mapped := map[string]bool{}
	for _, page := range pages {
		for _, mapping := range page.Mappings {
			mapped[mapping.Command] = true
		}
	}
	documentationWalk(&publicCommands, nil, func(command *commandDefinition, path []string) {
		if command.operation == "" {
			return
		}
		name := "axiom " + strings.Join(path, " ")
		if !mapped[name] {
			t.Errorf("man map missing %s", name)
		}
	})
}

func TestDocumentationManPagesDeterministicAndMapped(t *testing.T) {
	first, err := DocumentationPages("1.2.3", strings.Repeat("a", 40))
	if err != nil {
		t.Fatal(err)
	}
	second, err := DocumentationPages("1.2.3", strings.Repeat("a", 40))
	if err != nil {
		t.Fatal(err)
	}
	if len(first) != 15 || len(second) != len(first) {
		t.Fatalf("pages=%d", len(first))
	}
	seen := map[string]bool{}
	for i := range first {
		if first[i].Page != second[i].Page || !bytes.Equal(first[i].Text, second[i].Text) {
			t.Fatalf("nondeterministic page %s", first[i].Page)
		}
		if seen[first[i].Page] {
			t.Fatalf("duplicate page %s", first[i].Page)
		}
		seen[first[i].Page] = true
		if !strings.Contains(string(first[i].Text), "Version: 1.2.3") || !strings.Contains(string(first[i].Text), strings.Repeat("a", 40)) || !strings.Contains(string(first[i].Text), "Transform: cli-render-v1") {
			t.Errorf("missing identity in %s", first[i].Page)
		}
	}
	var mapped int
	for _, page := range first {
		mapped += len(page.Mappings)
	}
	if mapped == 0 {
		t.Fatal("no man mappings")
	}
}

func TestRoffEscapingPreventsRequestsAndControls(t *testing.T) {
	got := roffText(".request\n'comment\n\\X'hostile'\x1b[31m\x7f")
	if strings.Contains(got, "\x1b") || strings.Contains(got, "\x7f") {
		t.Fatalf("control byte survived: %q", got)
	}
	if !strings.Contains(got, "\\&.request") || !strings.Contains(got, "\\&'comment") || !strings.Contains(got, "\\eX") {
		t.Fatalf("roff was not escaped: %q", got)
	}
}
