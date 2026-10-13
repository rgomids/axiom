package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"io"
	"strings"
	"testing"
)

func TestHelpDocumentedFlagsParseWithExecutionRegistrations(t *testing.T) {
	walkHelpCommands(&publicCommands, nil, func(command *commandDefinition, path []string) {
		if command.operation == "" {
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
				case "repository", "repository-remote", "technology", "runtime-preference", "documentation", "glossary", "waive", "field":
					value = "sample=value"
				case "wait":
					value = "1s"
				case "enabled":
					value = "true"
				case "agent-profile":
					value = "intake:owner:sample"
				case "profile-effort":
					value = "sample:runtime-default:default"
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

func TestHelpGlobalSessionUsesExecutionValidation(t *testing.T) {
	for _, alias := range []string{"--help", "-h"} {
		for _, mode := range []string{"--json", "--human"} {
			for _, test := range []struct {
				name  string
				args  []string
				valid bool
			}{
				{"separate", []string{"--session", "Session_1", "project", "list", alias}, true},
				{"inline", []string{"--session=Session_1", "project", "list", alias}, true},
				{"empty", []string{"--session=", "project", "list", alias}, false},
				{"invalid", []string{"--session", "bad/value", "project", "list", alias}, false},
				{"too-long", []string{"--session=" + strings.Repeat("a", 129), "project", "list", alias}, false},
				{"missing", []string{alias, "--session"}, false},
				{"duplicate", []string{"--session=one", "--session", "two", "project", "list", alias}, false},
				{"help-value", []string{"--session", alias, "project", "list", alias}, true},
			} {
				t.Run(test.name+alias+mode, func(t *testing.T) {
					var output bytes.Buffer
					args := append([]string{mode}, test.args...)
					code := RunInteractive(context.Background(), args, nil, completionProvenance(t), nil, &output, io.Discard)
					if test.valid {
						if code != ExitSuccess || !strings.Contains(output.String(), "Usage:") || json.Valid(output.Bytes()) {
							t.Fatalf("valid help: code=%d output=%s", code, &output)
						}
						var repeated bytes.Buffer
						HandleHelp(args, completionProvenance(t), &repeated)
						if !bytes.Equal(output.Bytes(), repeated.Bytes()) {
							t.Fatal("nondeterministic help")
						}
					} else if code != ExitFailure || !strings.Contains(output.String(), "validation_failure") || !strings.Contains(output.String(), HelpCommand(test.args)) || mode == "--json" && !json.Valid(output.Bytes()) {
						t.Fatalf("invalid session: code=%d output=%s", code, &output)
					}
				})
			}
			// A help token consumed as the value is never itself a help request.
			if handled, _ := HandleHelp([]string{mode, "--session", alias, "project", "list"}, completionProvenance(t), io.Discard); handled {
				t.Fatal("session value treated as help")
			}
		}
	}
}
