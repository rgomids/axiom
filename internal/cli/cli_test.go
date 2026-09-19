package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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
		{"runtime install", []string{"runtime", "codex", "install"}, "runtime-install"},
		{"runtime status", []string{"runtime", "codex", "status"}, "runtime-status"},
		{"resolve", []string{"project", "resolve", "--selector", "alpha"}, "resolve:alpha"},
		{"configure", []string{"project", "configure", "--slug", "alpha", "--name", "Alpha", "--repository", "main=/tmp/alpha"}, "configure:alpha:Alpha:main:/tmp/alpha"},
		{"work item select", []string{"work-item", "select", "--project", "alpha", "--repository", "main", "--number", "7"}, "work-item-select:alpha:main:7"},
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
			operation := test.args[1]
			if test.args[0] == "runtime" {
				operation = "runtime_codex_" + test.args[2]
			}
			if test.args[0] == "work-item" {
				operation = "work_item_" + test.args[1]
			}
			assertEvent(t, output.String(), operation, Succeeded, "applied")
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

func TestRunInteractiveGuidesProjectConfiguration(t *testing.T) {
	var output, prompts bytes.Buffer
	input := strings.NewReader("alpha\nAlpha\nmain\n/tmp/alpha\n")
	code := RunInteractive(context.Background(), []string{"project", "configure"}, &recordingService{}, input, &output, &prompts)
	if code != ExitSuccess {
		t.Fatalf("exit code = %d, output=%s", code, output.String())
	}
	assertEvent(t, output.String(), "configure", Succeeded, "applied")
	if !strings.Contains(prompts.String(), "Repository path") {
		t.Fatalf("prompts = %q", prompts.String())
	}
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

func TestRunDoesNotExposeRejectedCommand(t *testing.T) {
	const sentinel = "do-not-render-this-command"
	var output bytes.Buffer
	code := Run(context.Background(), []string{"project", sentinel}, &recordingService{}, &output)
	if code != ExitFailure {
		t.Fatalf("exit code = %d", code)
	}
	if strings.Contains(output.String(), sentinel) {
		t.Fatalf("output exposed rejected command: %q", output.String())
	}
	assertEvent(t, output.String(), "unknown", Failed, "invalid_command")
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
func (s *recordingService) Install(_ context.Context, input InstallInput) Result {
	s.call = "install:" + input.Source
	return Result{Status: Succeeded, Category: "installed"}
}
func (s *recordingService) RuntimeCodexInstall(context.Context) Result {
	s.call = "runtime-install"
	return Result{Status: Succeeded, Category: "applied"}
}
func (s *recordingService) RuntimeCodexStatus(context.Context) Result {
	s.call = "runtime-status"
	return Result{Status: Succeeded, Category: "applied"}
}
func (s *recordingService) Resolve(_ context.Context, input ResolveInput) Result {
	s.call = "resolve:" + input.Selector
	return Result{Status: Succeeded, Category: "applied"}
}
func (s *recordingService) Configure(_ context.Context, input ConfigureInput) Result {
	repository := input.Repositories[0]
	s.call = "configure:" + input.Slug + ":" + input.Name + ":" + repository.Key + ":" + repository.Path
	return Result{Status: Succeeded, Category: "applied"}
}
func (s *recordingService) WorkItemCreate(context.Context, WorkItemInput) Result {
	s.call = "work-item-create"
	return Result{Status: Succeeded, Category: "applied"}
}
func (s *recordingService) WorkItemSelect(_ context.Context, input WorkItemInput) Result {
	s.call = "work-item-select:" + input.Project + ":" + input.Repository + ":" + fmt.Sprint(input.Number)
	return Result{Status: Succeeded, Category: "applied"}
}
func (s *recordingService) WorkItemShow(context.Context, WorkItemInput) Result {
	return Result{Status: Succeeded, Category: "applied"}
}
func (s *recordingService) WorkItemComment(context.Context, WorkItemInput) Result {
	return Result{Status: Succeeded, Category: "applied"}
}
func (s *recordingService) WorkItemComplete(context.Context, WorkItemInput) Result {
	return Result{Status: Succeeded, Category: "applied"}
}
