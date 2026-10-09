package cli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/rgomids/axiom/internal/completion"
	"github.com/rgomids/axiom/internal/projectapp"
)

type ProjectContextService interface {
	EffectiveProject(context.Context, string) (string, Result)
	ProjectContext(context.Context, ProjectContextInput) Result
}

type ProjectContextInput struct {
	Action, Selector string
	AuthorizeLocal   bool
}

// A session identifier is supplied by the caller, never inferred from CWD,
// process ID, a Runtime name, or conversation content.
func projectSessionArgs(ctx context.Context, args []string) (context.Context, []string, bool) {
	session, rest, ok := parseProjectSessionArgs(args)
	if !ok || session == "" {
		return ctx, rest, ok
	}
	return projectapp.WithProjectSession(ctx, session), rest, true
}

// parseProjectSessionArgs validates caller syntax without resolving local state.
func parseProjectSessionArgs(args []string) (string, []string, bool) {
	if len(args) == 0 {
		return "", args, true
	}
	session := ""
	if args[0] == "--session" {
		if len(args) < 3 {
			return "", args, false
		}
		session, args = args[1], args[2:]
	} else if strings.HasPrefix(args[0], "--session=") {
		session, args = strings.TrimPrefix(args[0], "--session="), args[1:]
	} else {
		return "", args, true
	}
	return session, args, session != "" && projectapp.ValidProjectSession(session)
}

func runProjectContext(ctx context.Context, args []string, service ProjectContextService) Result {
	input, ok := projectContextFlags(args, true)
	if !ok {
		return Result{Status: Failed, Category: "invalid_input"}
	}
	return service.ProjectContext(ctx, input)
}

func projectContextFlags(args []string, requireInputs bool) (ProjectContextInput, bool) {
	if len(args) == 0 {
		return ProjectContextInput{}, false
	}
	input := ProjectContextInput{Action: args[0]}
	if command, _, _, ok := resolveCommand([]string{"project", "context", input.Action}); !ok || command.operation == "" {
		return ProjectContextInput{}, false
	}
	set := projectContextFlagSet(&input)
	if invalidFlagSyntax(set, args[1:], nil) {
		return ProjectContextInput{}, false
	}
	if err := set.Parse(args[1:]); err != nil || set.NArg() != 0 {
		return ProjectContextInput{}, false
	}
	isSet := input.Action == "default-set" || input.Action == "session-set"
	if flagSupplied(args[1:], "--selector") && input.Selector == "" {
		return ProjectContextInput{}, false
	}
	if requireInputs && isSet && input.Selector == "" || !isSet && input.Action != "show" && input.Selector != "" || input.Action == "show" && input.AuthorizeLocal {
		return ProjectContextInput{}, false
	}
	return input, true
}

// Supply the application's resolved identity to the existing typed parser and
// interactive flows. Explicit flags are preserved, including invalid empties.
func effectiveProjectArgs(ctx context.Context, args []string, service ProjectContextService) (context.Context, []string, *Result) {
	if len(args) < 2 {
		return ctx, args, nil
	}
	flagName := ""
	valid := true
	switch {
	case args[0] == "project" && (args[1] == "show" || args[1] == "resolve"):
		flagName = "--selector"
		_, valid = flags(action(args[1]), args[2:])
	case args[0] == "work-item" && knownWorkItem(action("work_item_"+args[1])):
		flagName = "--project"
		_, valid = workItemFlags(action("work_item_"+args[1]), args[2:])
	case args[0] == "workflow" && knownWorkflow(action("workflow_"+args[1])):
		flagName = "--project"
		_, valid = workflowFlags(action("workflow_"+args[1]), args[2:])
	case len(args) >= 3 && args[0] == "runtime" && args[1] == "profile" && args[2] == "preview":
		flagName = "--project"
	}
	if flagName == "" || !valid || flagSupplied(args[2:], flagName) {
		return ctx, args, nil
	}
	id, result := service.EffectiveProject(ctx, "")
	if id == "" {
		return ctx, args, &result
	}
	if args[0] == "workflow" && args[1] == "start" {
		ctx = context.WithValue(ctx, workflowProjectKey{}, id)
	}
	return ctx, append(append([]string(nil), args...), flagName, id), nil
}

func emitContextCompletion(writer io.Writer, mode outputMode, result completion.Result, view projectapp.EffectiveContext) int {
	base := completionEvent{Status: result.Status(), Result: result.Result().String(), References: result.References(), Next: result.Next().String(), Details: result.Details(), Provenance: provenanceEvent{Product: result.Provenance().Product(), Version: result.Provenance().Version(), Revision: result.Provenance().Revision(), SourceState: result.Provenance().SourceState()}}
	var content []byte
	if mode == humanOutput {
		content = append(renderCompletionHuman(result), []byte(fmt.Sprintf("context: source=%s effective=%s default=%s session=%s issue=%s\n", view.Source, view.Effective, view.Default, view.Session, view.Issue))...)
	} else {
		wire, err := json.Marshal(struct {
			completionEvent
			Context projectapp.EffectiveContext `json:"context"`
		}{base, view})
		if err != nil {
			return ExitFailure
		}
		content = append(wire, '\n')
	}
	if len(content) > MaxCompletionOutputBytes {
		return ExitFailure
	}
	n, err := writer.Write(content)
	if err != nil || n != len(content) {
		return ExitFailure
	}
	return completionExitCode(result.Status())
}

// Retain whether the CLI supplied a canonical UUID only to satisfy parser
// requirements. The application must resolve the original winning context again.
type workflowProjectKey struct{}

func WorkflowProjectSelector(ctx context.Context, selector string) string {
	if injected, ok := ctx.Value(workflowProjectKey{}).(string); ok && injected == selector {
		return ""
	}
	return selector
}

func projectContextFlagSet(input *ProjectContextInput) *flag.FlagSet {
	set := flag.NewFlagSet("project context", flag.ContinueOnError)
	set.SetOutput(io.Discard)
	if input.Action == "show" || input.Action == "default-set" || input.Action == "session-set" {
		set.StringVar(&input.Selector, "selector", "", "Project identity; `<uuid-or-slug>`. Required for set; optional for show; rejected by clear/end.")
	}
	set.BoolVar(&input.AuthorizeLocal, "authorize-local", false, "Explicit authority for preference mutation; boolean. Must remain false for show.")
	return set
}
