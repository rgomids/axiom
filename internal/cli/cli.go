// Package cli is Lingo's presentation boundary for Project lifecycle actions.
package cli

import (
	"context"
	"encoding/json"
	"flag"
	"io"
)

const (
	ExitSuccess   = 0
	ExitFailure   = 1
	ExitCancelled = 2
)

// Service is deliberately operation-shaped. Presentation can request a lifecycle
// action but cannot issue a generic command or a raw filesystem mutation.
type Service interface {
	Init(context.Context, InitInput) Result
	Validate(context.Context, ProjectInput) Result
	Reopen(context.Context, ProjectInput) Result
	Update(context.Context, UpdateInput) Result
	Install(context.Context, InstallInput) Result
	RuntimeCodexInstall(context.Context) Result
	RuntimeCodexStatus(context.Context) Result
	Resolve(context.Context, ResolveInput) Result
}

type InitInput struct {
	Slug string
	Name string
}

type ProjectInput struct{ Slug string }

type UpdateInput struct {
	Slug string
	Name string
}
type InstallInput struct{ Source string }
type ResolveInput struct{ Selector string }

type Status string

const (
	Succeeded Status = "success"
	Failed    Status = "error"
	Cancelled Status = "cancelled"
)

// Result only permits fixed, presentation-safe categories. Application adapters
// must not return raw paths, input values, parser output, or operating-system
// error strings for this surface.
type Result struct {
	Status   Status
	Category string
}

// Run parses one CLI action, delegates it, and emits one safe structured event.
// It never turns a parser error into user-visible text because parser text may
// contain rejected input.
func Run(ctx context.Context, args []string, service Service, stdout io.Writer) int {
	operation, input, result := request(args, service)
	if result != nil {
		return emit(stdout, event{Operation: operation, Status: Failed, Category: *result})
	}
	response := dispatch(ctx, operation, input, service)
	return emit(stdout, event{Operation: operation, Status: response.Status, Category: response.Category})
}

type action string

const (
	initAction         action = "init"
	validateAction     action = "validate"
	reopenAction       action = "reopen"
	updateAction       action = "update"
	installAction      action = "install"
	resolveAction      action = "resolve"
	codexInstallAction action = "runtime_codex_install"
	codexStatusAction  action = "runtime_codex_status"
)

type requestInput struct {
	slug     string
	name     string
	source   string
	selector string
}

func request(args []string, service Service) (action, requestInput, *string) {
	if service == nil {
		return "unknown", requestInput{}, category("application_unavailable")
	}
	if len(args) == 3 && args[0] == "runtime" && args[1] == "codex" {
		operation := action("runtime_codex_" + args[2])
		if operation == codexInstallAction || operation == codexStatusAction {
			return operation, requestInput{}, nil
		}
		return "unknown", requestInput{}, category("invalid_command")
	}
	if len(args) < 2 || args[0] != "project" {
		return "unknown", requestInput{}, category("invalid_command")
	}
	operation := action(args[1])
	if !known(operation) {
		return "unknown", requestInput{}, category("invalid_command")
	}
	values, ok := flags(operation, args[2:])
	if !ok {
		return operation, requestInput{}, category("invalid_input")
	}
	if operation == initAction && (values.slug == "" || values.name == "") {
		return operation, values, category("missing_required_input")
	}
	if operation == installAction && values.source == "" {
		return operation, values, category("missing_required_input")
	}
	if operation == resolveAction && values.selector == "" {
		return operation, values, category("missing_required_input")
	}
	if operation != initAction && operation != installAction && operation != resolveAction && values.slug == "" {
		return operation, values, category("missing_required_input")
	}
	if operation == updateAction && values.name == "" {
		return operation, values, category("missing_required_input")
	}
	return operation, values, nil
}

func flags(operation action, args []string) (requestInput, bool) {
	set := flag.NewFlagSet(string(operation), flag.ContinueOnError)
	set.SetOutput(io.Discard)
	var values requestInput
	set.StringVar(&values.slug, "slug", "", "")
	if operation == initAction || operation == updateAction {
		set.StringVar(&values.name, "name", "", "")
	}
	if operation == installAction {
		set.StringVar(&values.source, "source", "", "")
	}
	if operation == resolveAction {
		set.StringVar(&values.selector, "selector", "", "")
	}
	if err := set.Parse(args); err != nil || set.NArg() != 0 {
		return requestInput{}, false
	}
	return values, true
}

func known(operation action) bool {
	return operation == initAction || operation == validateAction || operation == reopenAction || operation == updateAction || operation == installAction || operation == resolveAction
}

func dispatch(ctx context.Context, operation action, input requestInput, service Service) Result {
	if err := ctx.Err(); err != nil {
		return Result{Status: Cancelled, Category: "cancelled"}
	}
	switch operation {
	case initAction:
		return service.Init(ctx, InitInput{Slug: input.slug, Name: input.name})
	case validateAction:
		return service.Validate(ctx, ProjectInput{Slug: input.slug})
	case reopenAction:
		return service.Reopen(ctx, ProjectInput{Slug: input.slug})
	case updateAction:
		return service.Update(ctx, UpdateInput{Slug: input.slug, Name: input.name})
	case installAction:
		return service.Install(ctx, InstallInput{Source: input.source})
	case resolveAction:
		return service.Resolve(ctx, ResolveInput{Selector: input.selector})
	case codexInstallAction:
		return service.RuntimeCodexInstall(ctx)
	case codexStatusAction:
		return service.RuntimeCodexStatus(ctx)
	}
	return Result{Status: Failed, Category: "invalid_command"}
}

type event struct {
	Operation action `json:"operation"`
	Status    Status `json:"status"`
	Category  string `json:"category"`
}

func emit(writer io.Writer, value event) int {
	if writer == nil {
		return ExitFailure
	}
	_ = json.NewEncoder(writer).Encode(value)
	if value.Status == Succeeded {
		return ExitSuccess
	}
	if value.Status == Cancelled {
		return ExitCancelled
	}
	return ExitFailure
}

func category(value string) *string { return &value }

// UnavailableService makes the executable fail closed until its composition root
// receives the authorized application use cases in subsequent POC delivery work.
type UnavailableService struct{}

func (UnavailableService) Init(context.Context, InitInput) Result {
	return unavailable()
}
func (UnavailableService) Validate(context.Context, ProjectInput) Result {
	return unavailable()
}
func (UnavailableService) Reopen(context.Context, ProjectInput) Result {
	return unavailable()
}
func (UnavailableService) Update(context.Context, UpdateInput) Result {
	return unavailable()
}
func (UnavailableService) Install(context.Context, InstallInput) Result { return unavailable() }
func (UnavailableService) RuntimeCodexInstall(context.Context) Result   { return unavailable() }
func (UnavailableService) RuntimeCodexStatus(context.Context) Result    { return unavailable() }
func (UnavailableService) Resolve(context.Context, ResolveInput) Result { return unavailable() }
func unavailable() Result                                               { return Result{Status: Failed, Category: "application_unavailable"} }
