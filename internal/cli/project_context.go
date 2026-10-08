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
	if len(args) == 0 {
		return ctx, args, true
	}
	session := ""
	if args[0] == "--session" {
		if len(args) < 3 {
			return ctx, args, false
		}
		session, args = args[1], args[2:]
	} else if strings.HasPrefix(args[0], "--session=") {
		session, args = strings.TrimPrefix(args[0], "--session="), args[1:]
	} else {
		return ctx, args, true
	}
	if session == "" || !projectapp.ValidProjectSession(session) {
		return ctx, args, false
	}
	return projectapp.WithProjectSession(ctx, session), args, true
}

func runProjectContext(ctx context.Context, args []string, service ProjectContextService) Result {
	if len(args) == 0 {
		return Result{Status: Failed, Category: "invalid_input"}
	}
	input := ProjectContextInput{Action: args[0]}
	switch input.Action {
	case "show", "default-set", "default-clear", "session-set", "session-clear", "session-end":
	default:
		return Result{Status: Failed, Category: "invalid_input"}
	}
	set := flag.NewFlagSet("project context", flag.ContinueOnError)
	set.SetOutput(io.Discard)
	set.StringVar(&input.Selector, "selector", "", "Project UUID or slug")
	set.BoolVar(&input.AuthorizeLocal, "authorize-local", false, "Authorize preference mutation")
	if invalidFlagSyntax(set, args[1:], nil) {
		return Result{Status: Failed, Category: "invalid_input"}
	}
	if err := set.Parse(args[1:]); err != nil || set.NArg() != 0 {
		return Result{Status: Failed, Category: "invalid_input"}
	}
	isSet := input.Action == "default-set" || input.Action == "session-set"
	if flagSupplied(args[1:], "--selector") && input.Selector == "" {
		return Result{Status: Failed, Category: "invalid_input"}
	}
	if isSet && input.Selector == "" || !isSet && input.Action != "show" && input.Selector != "" || input.Action == "show" && input.AuthorizeLocal {
		return Result{Status: Failed, Category: "invalid_input"}
	}
	return service.ProjectContext(ctx, input)
}

// Supply the application's resolved identity to the existing typed parser and
// interactive flows. Explicit flags are preserved, including invalid empties.
func effectiveProjectArgs(ctx context.Context, args []string, service ProjectContextService) ([]string, *Result) {
	if len(args) < 2 {
		return args, nil
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
		return args, nil
	}
	id, result := service.EffectiveProject(ctx, "")
	if id == "" {
		return args, &result
	}
	return append(append([]string(nil), args...), flagName, id), nil
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
