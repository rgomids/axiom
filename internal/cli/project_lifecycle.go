package cli

import (
	"context"
	"flag"
	"io"

	"github.com/rgomids/axiom/internal/completion"
	"github.com/rgomids/axiom/internal/projectapp"
	"github.com/rgomids/axiom/internal/provenance"
)

// Issue #230 (I230-T03) presentation of Project archive/reactivate and
// Project discovery. Syntax only: exact selection, admission, authority and
// every status rule belong to the application. The service is optional so
// existing presentation services and mocks need not implement it.

// ProjectLifecycleInput carries one exact Project selector and, for a
// mutation, the reviewed preview digest and explicit local authority.
type ProjectLifecycleInput struct {
	Project        string
	PreviewDigest  string
	AuthorizeLocal bool
}

// ProjectListInput selects the Project listing.
type ProjectListInput struct{ IncludeArchived bool }

// ProjectStateView is the machine-local operational status of one installed
// Project: active or archived with the complete local state, or invalid or
// recovery_required without any claim about the state.
type ProjectStateView struct {
	Status      string                      `json:"status"`
	Operational *projectapp.OperationalView `json:"operational,omitempty"`
}

type ProjectLifecycleService interface {
	ProjectArchive(context.Context, ProjectLifecycleInput) Result
	ProjectReactivate(context.Context, ProjectLifecycleInput) Result
	ProjectListFiltered(context.Context, ProjectListInput) Result
	ProjectValidateInstalled(context.Context, ProjectLifecycleInput) Result
}

const (
	projectArchiveAction    action = "project_archive"
	projectReactivateAction action = "project_reactivate"
)

// runProjectLifecycle handles only the #230 forms of project archive,
// reactivate, list and validate. Every other invocation, including the POC
// forms of list and validate, is left to the established parser.
func runProjectLifecycle(ctx context.Context, mode outputMode, args []string, service Service, source provenance.Value, stdout io.Writer) (bool, int) {
	command, _, rest, valid := resolveCommand(args)
	if !valid || !operationInGroup(command.operation, "project") {
		return false, 0
	}
	switch command.operation {
	case projectArchiveAction, projectReactivateAction:
		operation, noun := projectArchiveAction, "Project archive"
		if command.operation == projectReactivateAction {
			operation, noun = projectReactivateAction, "Project reactivation"
		}
		input, issue := projectLifecycleFlags(operation, rest)
		if issue != "" {
			return true, emitLifecycleParserFailure(stdout, mode, noun, issue, source)
		}
		lifecycle, ok := service.(ProjectLifecycleService)
		if !ok {
			return true, emit(stdout, mode, event{Operation: operation, Status: Failed, Category: "application_unavailable"})
		}
		var response Result
		if operation == projectArchiveAction {
			response = lifecycle.ProjectArchive(ctx, input)
		} else {
			response = lifecycle.ProjectReactivate(ctx, input)
		}
		if response.Completion != nil && response.Operational == nil && response.Category != "" {
			return true, emitOperationalCompletion(stdout, mode, *response.Completion, response)
		}
		return true, emitResponse(stdout, mode, operation, response)
	case listAction:
		if len(rest) == 0 {
			return false, 0
		}
		var input ProjectListInput
		set := projectListFlagSet(&input)
		if invalidFlagSyntax(set, rest, nil) || set.Parse(rest) != nil || set.NArg() != 0 {
			return true, emitLifecycleParserFailure(stdout, mode, "Project listing", "invalid_input", source)
		}
		lifecycle, ok := service.(ProjectLifecycleService)
		if !ok {
			return true, emit(stdout, mode, event{Operation: listAction, Status: Failed, Category: "application_unavailable"})
		}
		return true, emitResponse(stdout, mode, listAction, lifecycle.ProjectListFiltered(ctx, input))
	case validateAction:
		slug, input, ok := projectValidationFlags(rest)
		if !ok || input.Project == "" {
			// Not a selector-based validation: the established parser decides.
			return false, 0
		}
		if projectValidationSelectorsConflict(slug, input.Project) {
			return true, emitLifecycleParserFailure(stdout, mode, "Project validation", "invalid_input", source)
		}
		lifecycle, ok := service.(ProjectLifecycleService)
		if !ok {
			return true, emit(stdout, mode, event{Operation: validateAction, Status: Failed, Category: "application_unavailable"})
		}
		return true, emitResponse(stdout, mode, validateAction, lifecycle.ProjectValidateInstalled(ctx, input))
	}
	return false, 0
}

// projectLifecycleFlags parses --project (required) and the optional exact
// authority pair. --preview-digest without --authorize-local, or the reverse,
// is a valid request that the application previews or denies.
func projectLifecycleFlags(operation action, args []string) (ProjectLifecycleInput, string) {
	var input ProjectLifecycleInput
	set := projectLifecycleFlagSet(operation, &input)
	if invalidFlagSyntax(set, args, nil) || set.Parse(args) != nil || set.NArg() != 0 {
		return ProjectLifecycleInput{}, "invalid_input"
	}
	if input.Project == "" {
		return ProjectLifecycleInput{}, "missing_required_input"
	}
	return input, ""
}

// emitLifecycleParserFailure reports a parser failure without echoing input.
func emitLifecycleParserFailure(writer io.Writer, mode outputMode, noun, issue string, source provenance.Value) int {
	message, next := noun+" input is invalid", "Remove unknown, duplicate, or conflicting flags and retry"
	if issue == "missing_required_input" {
		message, next = noun+" input is incomplete", "Provide an exact Project selector with --project <uuid-or-slug>"
	}
	if noun == "Project validation" {
		next = "Provide exactly one of --slug or --project"
	}
	statement, err := provenance.NewText(message, provenance.AxiomAuthored)
	if err != nil {
		return ExitFailure
	}
	nextAction, err := provenance.NewText(helpNext(writer, next), provenance.AxiomAuthored)
	if err != nil {
		return ExitFailure
	}
	result, err := completion.NewValidationFailure(statement, nextAction, source)
	if err != nil {
		return ExitFailure
	}
	return emitCompletion(writer, mode, result)
}

// The flag sets below are the single argument registry of the #230 Project
// lifecycle commands: the parsers and `skill inspect` read the same sets.

func projectLifecycleFlagSet(operation action, input *ProjectLifecycleInput) *flag.FlagSet {
	set := flag.NewFlagSet(string(operation), flag.ContinueOnError)
	set.SetOutput(io.Discard)
	set.StringVar(&input.Project, "project", "", "Configured Project identity; `<uuid-or-slug>`.")
	set.StringVar(&input.PreviewDigest, "preview-digest", "", "Exact reviewed preview digest; `<digest>`. Required with --authorize-local.")
	set.BoolVar(&input.AuthorizeLocal, "authorize-local", false, "Explicit authority for the exact machine-local effect; boolean. Never inferred by discovery.")
	return set
}

func projectListFlagSet(input *ProjectListInput) *flag.FlagSet {
	set := flag.NewFlagSet(string(listAction), flag.ContinueOnError)
	set.SetOutput(io.Discard)
	set.BoolVar(&input.IncludeArchived, "include-archived", false, "Also list Projects archived on this machine; boolean.")
	return set
}

func projectValidateFlagSet(slug *string, input *ProjectLifecycleInput) *flag.FlagSet {
	set := flag.NewFlagSet(string(validateAction), flag.ContinueOnError)
	set.SetOutput(io.Discard)
	set.StringVar(slug, "slug", "", "Portable Project slug (historical form); `<slug>`. Conflicts with --project.")
	set.StringVar(&input.Project, "project", "", "Installed Project identity, validated from its recorded source; `<uuid-or-slug>`.")
	return set
}

func projectValidationFlags(args []string) (string, ProjectLifecycleInput, bool) {
	var slug string
	var input ProjectLifecycleInput
	set := projectValidateFlagSet(&slug, &input)
	ok := !invalidFlagSyntax(set, args, nil) && set.Parse(args) == nil && set.NArg() == 0
	return slug, input, ok
}

func projectValidationSelectorsConflict(slug, project string) bool {
	return slug != "" && project != ""
}
