package cli

import (
	"context"
	"encoding/json"
	"flag"
	"io"

	"github.com/rgomids/axiom/internal/completion"
	"github.com/rgomids/axiom/internal/projectapp"
	"github.com/rgomids/axiom/internal/provenance"
)

// Issue #230 Integration lifecycle presentation (I230-T04). Parsing and
// rendering only: admission, inventory, validation, local transitions and
// portable removal are application rules composed by the service.

// IntegrationService is optional, like RuntimeProfileService, so existing
// presentation services and mocks need not implement it; an application
// without it reports application_unavailable.
type IntegrationService interface {
	IntegrationList(context.Context, IntegrationInput) Result
	IntegrationShow(context.Context, IntegrationInput) Result
	IntegrationValidate(context.Context, IntegrationInput) Result
	IntegrationDisable(context.Context, IntegrationInput) Result
	IntegrationEnable(context.Context, IntegrationInput) Result
	IntegrationRemove(context.Context, IntegrationInput) Result
}

// IntegrationInput carries explicit selectors and the reviewed-effect replay
// inputs. ProjectID is accepted by remove only (the EDIT replay identity).
type IntegrationInput struct {
	Project, Integration, ProjectID, PreviewDigest string
	AuthorizeLocal                                 bool
}

const (
	integrationListAction     action = "integration_list"
	integrationShowAction     action = "integration_show"
	integrationValidateAction action = "integration_validate"
	integrationDisableAction  action = "integration_disable"
	integrationEnableAction   action = "integration_enable"
	integrationRemoveAction   action = "integration_remove"
)

// integrationAction recognizes the integration group. An unknown verb inside
// the group is reported by the group, never as another command.
func integrationAction(args []string) (action, []string, bool) {
	if len(args) == 0 || args[0] != "integration" {
		return "", nil, false
	}
	command, _, rest, ok := resolveCommand(args)
	if ok && operationInGroup(command.operation, "integration") {
		return command.operation, rest, true
	}
	return "unknown", nil, true
}

func integrationFlags(operation action, args []string) (IntegrationInput, bool) {
	return integrationFlagsWithPresence(operation, args, true)
}

func integrationFlagsWithPresence(operation action, args []string, requireInputs bool) (IntegrationInput, bool) {
	var input IntegrationInput
	set := integrationFlagSet(operation, &input)
	if invalidFlagSyntax(set, args, nil) {
		return IntegrationInput{}, false
	}
	if err := set.Parse(args); err != nil || set.NArg() != 0 {
		return IntegrationInput{}, false
	}
	// Selectors are bounded and never empty when supplied; the application
	// validates the Project selector and Integration key grammar exactly.
	if requireInputs && input.Project == "" || len(input.Project) > 256 || len(input.Integration) > 256 || len(input.ProjectID) > 256 {
		return IntegrationInput{}, false
	}
	if flagSupplied(args, "--integration") && input.Integration == "" || flagSupplied(args, "--preview-digest") && !validPreviewDigest(input.PreviewDigest) {
		return IntegrationInput{}, false
	}
	switch operation {
	case integrationShowAction, integrationDisableAction, integrationEnableAction, integrationRemoveAction:
		if requireInputs && input.Integration == "" {
			return IntegrationInput{}, false
		}
	}
	return input, true
}

func runIntegration(ctx context.Context, mode outputMode, operation action, args []string, service Service, source provenance.Value, stdout io.Writer) int {
	if operation == "unknown" {
		return emit(stdout, mode, event{Operation: "unknown", Status: Failed, Category: "invalid_command"})
	}
	input, ok := integrationFlags(operation, args)
	if !ok {
		return emitIntegrationParserFailure(stdout, mode, source)
	}
	integrations, ok := service.(IntegrationService)
	if !ok {
		return emit(stdout, mode, event{Operation: operation, Status: Failed, Category: "application_unavailable"})
	}
	if err := ctx.Err(); err != nil {
		return emit(stdout, mode, event{Operation: operation, Status: Cancelled, Category: "cancelled"})
	}
	var response Result
	switch operation {
	case integrationListAction:
		response = integrations.IntegrationList(ctx, input)
	case integrationShowAction:
		response = integrations.IntegrationShow(ctx, input)
	case integrationValidateAction:
		response = integrations.IntegrationValidate(ctx, input)
	case integrationDisableAction:
		response = integrations.IntegrationDisable(ctx, input)
	case integrationEnableAction:
		response = integrations.IntegrationEnable(ctx, input)
	default:
		response = integrations.IntegrationRemove(ctx, input)
	}
	// Mutations and gate denials carry their own canonical presentation
	// (operational preview, EDIT preview, admission); everything else, failures
	// included, is presented here so the stable category is never dropped.
	if response.Completion != nil && response.Operational == nil && response.Edit == nil && response.Admission == nil && response.Preflight == nil {
		return emitIntegrationCompletion(stdout, mode, *response.Completion, response)
	}
	return emitResponse(stdout, mode, operation, response)
}

// emitIntegrationParserFailure never echoes rejected input.
func emitIntegrationParserFailure(writer io.Writer, mode outputMode, source provenance.Value) int {
	statement, err := provenance.NewText("Integration input is invalid", provenance.AxiomAuthored)
	if err != nil {
		return ExitFailure
	}
	nextAction, err := provenance.NewText(helpNext(writer, "Provide --project and, where required, one exact --integration key; remove unknown, duplicate or conflicting flags"), provenance.AxiomAuthored)
	if err != nil {
		return ExitFailure
	}
	result, err := completion.NewValidationFailure(statement, nextAction, source)
	if err != nil {
		return ExitFailure
	}
	return emitCompletion(writer, mode, result)
}

type integrationCompletionEvent struct {
	completionEvent
	Category     string                        `json:"category,omitempty"`
	Integrations *projectapp.IntegrationReport `json:"integrations,omitempty"`
}

func emitIntegrationCompletion(writer io.Writer, mode outputMode, result completion.Result, response Result) int {
	if writer == nil || !result.Valid() {
		return ExitFailure
	}
	base := completionEvent{Status: result.Status(), Result: result.Result().String(), References: result.References(), Next: result.Next().String(), Details: result.Details(), Provenance: provenanceEvent{Product: result.Provenance().Product(), Version: result.Provenance().Version(), Revision: result.Provenance().Revision(), SourceState: result.Provenance().SourceState()}}
	wire, err := json.Marshal(integrationCompletionEvent{completionEvent: base, Category: response.Category, Integrations: response.Integrations})
	if err != nil {
		return ExitFailure
	}
	return presentEvent(writer, mode, result.Status(), append(wire, '\n'), MaxCompletionOutputBytes)
}

// integrationFlagSet is the single argument registry of the integration
// group: the parser and `skill inspect` read the same set.
func integrationFlagSet(operation action, input *IntegrationInput) *flag.FlagSet {
	set := flag.NewFlagSet(string(operation), flag.ContinueOnError)
	set.SetOutput(io.Discard)
	set.StringVar(&input.Project, "project", "", "Configured Project identity; `<uuid-or-slug>`.")
	if operation != integrationListAction {
		usage := "Declared Integration key; `<key>`."
		if operation == integrationValidateAction {
			usage = "Optional declared Integration key to validate; `<key>`."
		}
		set.StringVar(&input.Integration, "integration", "", usage)
	}
	if operation == integrationDisableAction || operation == integrationEnableAction || operation == integrationRemoveAction {
		set.StringVar(&input.PreviewDigest, "preview-digest", "", "Exact reviewed preview digest; `<digest>`. Required with --authorize-local.")
		set.BoolVar(&input.AuthorizeLocal, "authorize-local", false, "Explicit authority for the exact reviewed effect; boolean. Never inferred by discovery.")
	}
	if operation == integrationRemoveAction {
		set.StringVar(&input.ProjectID, "project-id", "", "Project UUID returned by the reviewed remove preview; `<uuid>`. Required for publication.")
	}
	return set
}

func integrationOperation(operation action) bool {
	switch operation {
	case integrationListAction, integrationShowAction, integrationValidateAction, integrationDisableAction, integrationEnableAction, integrationRemoveAction:
		return true
	}
	return false
}
