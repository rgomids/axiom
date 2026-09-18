package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestRunDelegatesEachLifecycleOperation(t *testing.T) {
	service := &recordingService{}
	cases := []struct {
		name string
		args []string
		call string
	}{
		{"init", []string{"project", "init", "--slug", "alpha", "--name", "Alpha"}, "init:alpha:Alpha"},
		{"validate", []string{"project", "validate", "--slug", "alpha"}, "validate:alpha"},
		{"reopen", []string{"project", "reopen", "--slug", "alpha"}, "reopen:alpha"},
		{"update", []string{"project", "update", "--slug", "alpha", "--name", "Renamed"}, "update:alpha:Renamed"},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			var output bytes.Buffer
			if code := Run(context.Background(), test.args, service, &output); code != ExitSuccess {
				t.Fatalf("exit code = %d", code)
			}
			if service.call != test.call {
				t.Fatalf("call = %q, want %q", service.call, test.call)
			}
			assertEvent(t, output.String(), string(test.args[1]), Succeeded, "applied")
		})
	}
}

func TestRunFailsPromptlyWithoutRequiredInput(t *testing.T) {
	var output bytes.Buffer
	code := Run(context.Background(), []string{"project", "init", "--slug", "alpha"}, &recordingService{}, &output)
	if code != ExitFailure {
		t.Fatalf("exit code = %d", code)
	}
	assertEvent(t, output.String(), "init", Failed, "missing_required_input")
}

func TestRunDoesNotExposeRejectedInput(t *testing.T) {
	const sentinel = "do-not-render-this-value"
	var output bytes.Buffer
	code := Run(context.Background(), []string{"project", "init", "--slug", "alpha", "--name", "Alpha", "--unexpected", sentinel}, &recordingService{}, &output)
	if code != ExitFailure {
		t.Fatalf("exit code = %d", code)
	}
	if strings.Contains(output.String(), sentinel) {
		t.Fatalf("output exposed rejected input: %q", output.String())
	}
	assertEvent(t, output.String(), "init", Failed, "invalid_input")
}

func TestRunUsesCancelledExitCode(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	var output bytes.Buffer
	if code := Run(ctx, []string{"project", "validate", "--slug", "alpha"}, &recordingService{}, &output); code != ExitCancelled {
		t.Fatalf("exit code = %d", code)
	}
	assertEvent(t, output.String(), "validate", Cancelled, "cancelled")
}

func assertEvent(t *testing.T, output, operation string, status Status, category string) {
	t.Helper()
	var value event
	if err := json.Unmarshal([]byte(output), &value); err != nil {
		t.Fatalf("output is not a JSON event: %v; output=%q", err, output)
	}
	if value.Operation != action(operation) || value.Status != status || value.Category != category {
		t.Fatalf("event = %#v", value)
	}
}

type recordingService struct{ call string }

func (s *recordingService) Init(_ context.Context, input InitInput) Result {
	s.call = "init:" + input.Slug + ":" + input.Name
	return Result{Status: Succeeded, Category: "applied"}
}
func (s *recordingService) Validate(_ context.Context, input ProjectInput) Result {
	s.call = "validate:" + input.Slug
	return Result{Status: Succeeded, Category: "applied"}
}
func (s *recordingService) Reopen(_ context.Context, input ProjectInput) Result {
	s.call = "reopen:" + input.Slug
	return Result{Status: Succeeded, Category: "applied"}
}
func (s *recordingService) Update(_ context.Context, input UpdateInput) Result {
	s.call = "update:" + input.Slug + ":" + input.Name
	return Result{Status: Succeeded, Category: "applied"}
}
