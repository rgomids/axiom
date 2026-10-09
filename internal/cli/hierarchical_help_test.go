package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func walkHelpCommands(node *commandDefinition, path []string, visit func(*commandDefinition, []string)) {
	visit(node, path)
	for i := range node.children {
		child := &node.children[i]
		walkHelpCommands(child, append(append([]string(nil), path...), child.name), visit)
	}
}

func TestHierarchicalHelpEveryCommandWithoutService(t *testing.T) {
	source := completionProvenance(t)
	walkHelpCommands(&publicCommands, nil, func(node *commandDefinition, path []string) {
		t.Run(strings.Join(path, "/"), func(t *testing.T) {
			var baseline string
			for _, helpFlag := range []string{"--help", "-h"} {
				for _, outputMode := range [][]string{nil, {"--json"}, {"--human"}, {"--json", "--human"}} {
					args := append(append(append([]string(nil), outputMode...), path...), helpFlag)
					var output, prompts bytes.Buffer
					code := RunInteractive(context.Background(), args, nil, source, nil, &output, &prompts)
					if code != ExitSuccess || prompts.Len() != 0 {
						t.Fatalf("%v: code=%d prompts=%q output=%q", args, code, &prompts, &output)
					}
					if baseline == "" {
						baseline = output.String()
					}
					if output.String() != baseline {
						t.Fatalf("%v changes deterministic human help", args)
					}
				}
			}
			if !strings.Contains(baseline, "Usage:\n") || json.Valid([]byte(baseline)) {
				t.Fatalf("not human help: %s", baseline)
			}
			if node.operation == "help" {
				return
			}
			if !strings.Contains(baseline, helpInvocation(path)) {
				t.Fatalf("missing own help invocation: %s", baseline)
			}
			if len(node.children) > 0 {
				for _, child := range node.children {
					if !strings.Contains(baseline, fmt.Sprintf("  %-16s %s\n", child.name, child.summary)) {
						t.Errorf("missing direct child %s", child.name)
					}
				}
				return
			}
			commandFlagSet(node.operation).VisitAll(func(f *flag.Flag) {
				requirement := "optional"
				for _, rule := range commandRequirements(node.operation) {
					if rule.name != f.Name {
						continue
					}
					if rule.required {
						requirement = "required"
					} else if rule.when != "" {
						requirement = "required when " + rule.when
					}
				}
				if repeatableFlags(commandFlagSet(node.operation))[f.Name] {
					requirement += "; repeatable"
				}
				value, description := flag.UnquoteUsage(f)
				prefix := "  --" + f.Name
				if !booleanFlag(f) {
					prefix += " " + value
				}
				if !strings.Contains(baseline, prefix+" ("+requirement+"):") {
					t.Errorf("missing parser flag/requirement %s", prefix)
				}
				if description != "" && !strings.Contains(baseline, description) {
					t.Errorf("missing parser description for %s", f.Name)
				}
			})
		})
	})
}

// Read executable parser action declarations independently of the presentation
// tree, so deleting a command from the tree cannot also delete its test oracle.
func TestHierarchicalHelpCoversDeclaredParserActions(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	found := 0
	for _, name := range files {
		if strings.HasSuffix(name, "_test.go") || name == "commands.go" {
			continue
		}
		file, err := parser.ParseFile(token.NewFileSet(), name, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(file, func(n ast.Node) bool {
			spec, ok := n.(*ast.ValueSpec)
			if !ok {
				return true
			}
			typeName, ok := spec.Type.(*ast.Ident)
			if !ok || typeName.Name != "action" {
				return true
			}
			for _, value := range spec.Values {
				literal, ok := value.(*ast.BasicLit)
				if !ok || literal.Kind != token.STRING {
					t.Errorf("unhandled action declaration in %s", name)
					continue
				}
				operation, err := strconv.Unquote(literal.Value)
				if err != nil {
					t.Fatal(err)
				}
				found++
				if node, _ := commandByAction(action(operation)); node == nil {
					t.Errorf("parser action %q from %s has no help command", operation, name)
				}
			}
			return true
		})
	}
	if found == 0 {
		t.Fatal("no independent parser action declarations discovered")
	}
}

func TestHierarchicalHelpDoesNotConsumeFlagValuesOrTerminatedArguments(t *testing.T) {
	source := completionProvenance(t)
	for _, args := range [][]string{
		{"project", "show", "--selector", "--help"},
		{"project", "show", "--selector", "-h"},
		{"project", "show", "--selector=--help"},
		{"project", "configure", "--name", "--help"},
		{"--session", "--help", "project", "list"},
		{"--session=-h", "project", "list"},
		{"project", "show", "--", "--help"},
		{"--", "-h"},
	} {
		var output bytes.Buffer
		if handled, _ := HandleHelp(args, source, &output); handled || output.Len() != 0 {
			t.Errorf("data interpreted as help for %v: %s", args, &output)
		}
	}
	for _, args := range [][]string{{"project", "show", "--selector", "--help", "-h"}} {
		if handled, code := HandleHelp(args, source, io.Discard); !handled || code != ExitSuccess {
			t.Errorf("real help token missed: %v", args)
		}
	}
}

func TestHierarchicalHelpUnknownPathFailsWithNearestGuidance(t *testing.T) {
	source := completionProvenance(t)
	for _, test := range []struct {
		path     []string
		guidance string
	}{
		{[]string{"unknown"}, "axiom --help"},
		{[]string{"project", "unknown"}, "axiom project --help"},
		{[]string{"runtime", "codex", "unknown"}, "axiom runtime codex --help"},
	} {
		for _, mode := range []string{"--human", "--json"} {
			args := append(append([]string{mode}, test.path...), "--help")
			var output bytes.Buffer
			handled, code := HandleHelp(args, source, &output)
			if !handled || code != ExitFailure {
				t.Fatalf("%v: handled=%v code=%d", args, handled, code)
			}
			if !strings.Contains(output.String(), test.guidance) {
				t.Errorf("%v missing guidance: %s", args, &output)
			}
			if mode == "--json" && !json.Valid(output.Bytes()) {
				t.Errorf("invalid JSON: %s", &output)
			}
			if mode == "--human" && json.Valid(output.Bytes()) {
				t.Errorf("unexpected JSON: %s", &output)
			}
			if strings.Contains(output.String(), "Usage:\n") {
				t.Errorf("unknown command received successful help: %s", &output)
			}
		}
	}
}
