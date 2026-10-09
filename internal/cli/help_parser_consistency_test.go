package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"io"
	"runtime"
	"strings"
	"testing"
)

func TestHelpDocumentedFlagsParseWithExecutionRegistrations(t *testing.T) {
	walkHelpCommands(&publicCommands, nil, func(command *commandDefinition, path []string) {
		if command.operation == "" || command.operation == windowsPermissionsRestoreAction {
			return
		}
		commandFlagSet(command.operation).VisitAll(func(f *flag.Flag) {
			t.Run(strings.Join(path, "/")+"/"+f.Name, func(t *testing.T) {
				value := "sample"
				if getter, ok := f.Value.(flag.Getter); ok {
					switch getter.Get().(type) {
					case bool:
						value = "true"
					case int, uint64:
						value = "1"
					}
				}
				switch f.Name {
				case "repository", "repository-remote", "technology", "runtime-preference", "documentation", "glossary":
					value = "sample=value"
				case "elaborated-section":
					value = "problem=Sample problem"
				}
				for _, args := range [][]string{{"--" + f.Name + "=" + value}, {"--" + f.Name, value}} {
					if booleanFlag(f) {
						args = []string{"--" + f.Name + "=" + value}
					}
					set := commandFlagSet(command.operation)
					if invalidFlagSyntax(set, args, repeatableFlags(set)) || set.Parse(args) != nil || set.NArg() != 0 {
						t.Fatalf("execution flag registration rejected %v", args)
					}
				}
			})
		})
	})
}

func TestHelpValidationExampleUsesOneSelectorForm(t *testing.T) {
	example := commandExample(validateAction, []string{"project", "validate"})
	if strings.Contains(example, "--slug") || !strings.Contains(example, "--project") {
		t.Fatalf("conflicting example: %s", example)
	}
	// Substitute the documented placeholder and ask the real typed parser to
	// admit this installed validation form; no application service is invoked.
	args := []string{"project", "validate", "--project", "sample"}
	var output bytes.Buffer
	handled, code := runProjectLifecycle(context.Background(), humanOutput, args, &recordingService{}, completionProvenance(t), &output)
	if !handled || code != ExitFailure || !strings.Contains(output.String(), "application_unavailable") {
		t.Fatalf("example failed parser admission: handled=%t code=%d output=%s", handled, code, &output)
	}
}

func TestHelpWindowsRecoveryKeepsFixedOrderSyntax(t *testing.T) {
	group := windowsPermissionsCommand()
	leaf := &group.children[0]
	path := []string{group.name, leaf.name}
	var output bytes.Buffer
	if renderHelp(&output, leaf, path) != ExitSuccess {
		t.Fatal("Windows help failed")
	}
	for _, token := range []string{"--backup", "--approve", "required", "exact order"} {
		if !strings.Contains(output.String(), token) {
			t.Errorf("missing %s", token)
		}
	}
	if strings.Contains(output.String(), "--backup=") || strings.Contains(output.String(), "[--session") || strings.Contains(output.String(), "[--human") {
		t.Fatal("advertised unsupported Windows recovery forms")
	}
	valid := []string{group.name, leaf.name, "--backup", "<absolute-json-path>", "--approve", "<digest>"}
	if backup, approval, ok := ParseWindowsPermissionsRestore(valid); !ok || backup != valid[3] || approval != valid[5] {
		t.Fatal("fixed-order syntax changed")
	}
	for _, args := range [][]string{
		{group.name, leaf.name, "--approve", "digest", "--backup", "path"},
		{group.name, leaf.name, "--backup=path", "--approve=digest"},
		{group.name, "unknown", "--backup", "path", "--approve", "digest"},
	} {
		if _, _, ok := ParseWindowsPermissionsRestore(args); ok {
			t.Fatalf("broadened syntax: %v", args)
		}
	}
	node, _ := commandByAction(windowsPermissionsRestoreAction)
	if (node != nil) != (runtime.GOOS == "windows") {
		t.Fatal("Windows command visibility does not match executable support")
	}
}

func TestHelpOmittedUnsupportedSessionOptionsAndWriterFailures(t *testing.T) {
	for _, path := range [][]string{{"version"}, {"skill", "inspect"}} {
		var output bytes.Buffer
		if handled, code := HandleHelp(append(path, "--help"), completionProvenance(t), &output); !handled || code != ExitSuccess || strings.Contains(output.String(), "--session") {
			t.Fatalf("unsupported session documented for %v: %s", path, &output)
		}
	}
	for _, args := range [][]string{{"--help"}, {"unknown", "--help"}} {
		if handled, code := HandleHelp(args, completionProvenance(t), nil); !handled || code != ExitFailure {
			t.Fatal("nil output did not fail")
		}
	}
	if Help(failingHelpWriter{}) != ExitFailure {
		t.Fatal("write error ignored")
	}
}

type failingHelpWriter struct{}

func (failingHelpWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }

func TestHelpWindowsRecoveryRoutingWithoutWindowsEffects(t *testing.T) {
	root := group("axiom", "Test root", windowsPermissionsCommand())
	for _, path := range [][]string{{"windows-permissions"}, {"windows-permissions", "restore"}} {
		for _, alias := range []string{"--help", "-h"} {
			command, found, help, valid := helpTargetFrom(&root, append(append([]string{"--json"}, path...), alias))
			if !help || !valid || strings.Join(found, " ") != strings.Join(path, " ") {
				t.Fatalf("Windows help route failed: %v", path)
			}
			var output bytes.Buffer
			if renderHelp(&output, command, found) != ExitSuccess {
				t.Fatal("Windows help rendering failed")
			}
		}
	}
	for _, test := range []struct {
		args []string
		want string
	}{
		{[]string{"windows-permissions", "unknown"}, "axiom windows-permissions --help"},
		{[]string{"windows-permissions", "restore", "--unknown"}, "axiom windows-permissions restore --help"},
	} {
		_, path, _, _ := helpTargetFrom(&root, test.args)
		if got := helpInvocation(path); got != test.want {
			t.Fatalf("Windows invalid-input guidance = %s, want %s", got, test.want)
		}
		if runtime.GOOS == "windows" && HelpCommand(test.args) != test.want {
			t.Fatal("Windows adapter guidance does not match registered path")
		}
	}
	for _, token := range []string{"--help", "-h"} {
		_, _, help, _ := helpTargetFrom(&root, []string{"windows-permissions", "restore", "--backup", token, "--approve", token})
		if help {
			t.Fatal("Windows backup/approval value treated as help")
		}
	}
}

func TestHelpRejectsProvidedInvalidLeafSyntaxWithoutServices(t *testing.T) {
	for _, leafArgs := range [][]string{
		{"project", "show", "--unknown", "--help"},
		{"project", "validate", "--slug", "old", "--project", "installed", "--help"},
		{"workflow", "advance", "--automatic", "--gate", "intake", "--help"},
		{"work-item", "show", "--work-item", "github:owner/repo#1", "--number", "1", "--help"},
		{"project", "context", "show", "--authorize-local", "--help"},
		{"runtime", "profile", "preview", "--runtime", "unsupported", "--help"},
		{"integration", "show", "--integration=", "--help"},
		{"project", "show", "--selector", "first", "--selector", "second", "-h"},
		{"project", "show", "extra", "--help"},
		{"project", "show", "-selector", "first", "--help"},
		{"project", "show", "--help", "--", "extra"},
		{"work-item", "show", "--number=invalid", "--help"},
		{"work-item", "create", "--work-item", "github:owner/repo#1", "--help"},
		{"work-item", "show", "--work-item", "invalid", "--help"},
		{"workflow", "start", "--work-item", "github:owner/repo#1", "--execution", "existing", "--help"},
		{"workflow", "show", "--work-item", "invalid", "--help"},
		{"skill", "inspect", "unknown-skill", "--help"},
	} {
		for _, mode := range []string{"--human", "--json"} {
			var output bytes.Buffer
			args := append([]string{mode}, leafArgs...)
			if handled, code := HandleHelp(args, completionProvenance(t), &output); !handled || code != ExitFailure {
				t.Fatalf("invalid supplied syntax succeeded: %v: %s", args, &output)
			}
			if !strings.Contains(output.String(), HelpCommand(leafArgs)) || strings.Contains(output.String(), "Usage:") {
				t.Fatalf("invalid leaf guidance: %s", &output)
			}
			if mode == "--json" && !json.Valid(output.Bytes()) {
				t.Fatalf("unstructured help failure: %s", &output)
			}
		}
	}
}

func TestHelpDirectMutationExamplesStateRequiredAuthority(t *testing.T) {
	for _, operation := range []action{"context_default-set", "context_default-clear", "context_session-set", "context_session-clear", "context_session-end", workflowFactAction} {
		_, path := commandByAction(operation)
		example := commandExample(operation, path)
		if !strings.Contains(example, "--authorize-local") {
			t.Fatalf("direct mutation example lacks explicit local authority: %s", example)
		}
		if strings.HasPrefix(string(operation), "context_session-") && !strings.Contains(example, "--session") {
			t.Fatalf("session example lacks caller session: %s", example)
		}
	}
}
