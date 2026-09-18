// Package cli is Lingo's presentation boundary for Project lifecycle actions.
package cli

import (
	"context"
	"encoding/json"
	"errors"
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
	initAction     action = "init"
	validateAction action = "validate"
	reopenAction   action = "reopen"
	updateAction   action = "update"
)

type requestInput struct {
	slug string
	name string
}

func request(args []string, service Service) (action, requestInput, *string) {
	if service == nil {
		return "unknown", requestInput{}, category("application_unavailable")
	}
	if len(args) < 2 || args[0] != "project" {
		return "unknown", requestInput{}, category("invalid_command")
	}
	operation := action(args[1])
	if !known(operation) {
		return operation, requestInput{}, category("invalid_command")
	}
	values, ok := flags(operation, args[2:])
	if !ok {
		return operation, requestInput{}, category("invalid_input")
	}
	if operation == initAction && (values.slug == "" || values.name == "") {
		return operation, values, category("missing_required_input")
	}
	if operation != initAction && values.slug == "" {
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
	if err := set.Parse(args); err != nil || set.NArg() != 0 {
		return requestInput{}, false
	}
	return values, true
}

func known(operation action) bool {
	return operation == initAction || operation == validateAction || operation == reopenAction || operation == updateAction
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
func unavailable() Result { return Result{Status: Failed, Category: "application_unavailable"} }

var ErrUnavailable = errors.New("application unavailable")
