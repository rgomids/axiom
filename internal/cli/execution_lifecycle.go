package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"

	"github.com/rgomids/axiom/internal/completion"
	"github.com/rgomids/axiom/internal/provenance"
)

// ExecutionListService is optional so existing presentation services and
// mocks need not implement Execution discovery. It stays out of Service.
type ExecutionListService interface {
	WorkflowList(context.Context, ExecutionListInput) ExecutionListResponse
}

const workflowListAction action = "workflow_list"

const maxExecutionListOutputBytes = 2 * 1024 * 1024

// ExecutionListInput is the closed input of `workflow list`.
type ExecutionListInput struct{ Project, Repository string }

// ExecutionSummaryView is the presentation shape of one discovered Execution.
type ExecutionSummaryView struct {
	ExecutionID    string               `json:"executionId"`
	RepositoryKey  string               `json:"repositoryKey"`
	WorkItem       ExecutionWorkItemRef `json:"workItem"`
	Status         string               `json:"status"`
	CurrentGate    string               `json:"currentGate"`
	LifecycleStage string               `json:"lifecycleStage,omitempty"`
	Revision       uint64               `json:"revision"`
	CreatedAt      string               `json:"createdAt"`
	UpdatedAt      string               `json:"updatedAt"`
}

// ExecutionWorkItemRef is the Work Item identity of a discovered Execution.
type ExecutionWorkItemRef struct {
	Provider   string `json:"provider"`
	Resource   string `json:"resource"`
	ExternalID string `json:"externalId"`
}

// ExecutionListResponse carries either the listing (Listed) or a denial or
// failure Result rendered through the ordinary response path.
type ExecutionListResponse struct {
	Result     Result
	Executions []ExecutionSummaryView
	Listed     bool
}

type executionListCompletionEvent struct {
	completionEvent
	Executions []ExecutionSummaryView `json:"executions"`
}

// runWorkflowList handles `workflow list`. It reports handled=false for any
// other command so the ordinary dispatch continues untouched.
func runWorkflowList(ctx context.Context, args []string, service Service, source provenance.Value, mode outputMode, stdout io.Writer) (bool, int) {
	if len(args) < 2 || args[0] != "workflow" || args[1] != "list" {
		return false, 0
	}
	input, ok := executionListFlags(args[2:])
	if !ok {
		return true, emitExecutionListParserFailure(stdout, mode, source)
	}
	lister, ok := service.(ExecutionListService)
	if !ok {
		return true, emit(stdout, mode, event{Operation: workflowListAction, Status: Failed, Category: "application_unavailable"})
	}
	response := lister.WorkflowList(ctx, input)
	if !response.Listed || response.Result.Completion == nil {
		return true, emitResponse(stdout, mode, workflowListAction, response.Result)
	}
	return true, emitExecutionList(stdout, mode, *response.Result.Completion, response.Executions)
}

// executionListFlags is strict: --project is required, --repository optional,
// unknown, duplicate, single-hyphen and positional input is rejected.
func executionListFlags(args []string) (ExecutionListInput, bool) {
	var input ExecutionListInput
	set := flag.NewFlagSet(string(workflowListAction), flag.ContinueOnError)
	set.SetOutput(io.Discard)
	set.StringVar(&input.Project, "project", "", "")
	set.StringVar(&input.Repository, "repository", "", "")
	if invalidFlagSyntax(set, args, nil) {
		return ExecutionListInput{}, false
	}
	if err := set.Parse(args); err != nil || set.NArg() != 0 {
		return ExecutionListInput{}, false
	}
	if input.Project == "" || len(input.Project) > 256 || len(input.Repository) > 256 || flagSupplied(args, "--repository") && input.Repository == "" {
		return ExecutionListInput{}, false
	}
	return input, true
}

func emitExecutionListParserFailure(writer io.Writer, mode outputMode, source provenance.Value) int {
	statement, err := provenance.NewText("Execution list input is invalid", provenance.AxiomAuthored)
	if err != nil {
		return ExitFailure
	}
	next, err := provenance.NewText("Provide --project and optionally --repository, with no other input", provenance.AxiomAuthored)
	if err != nil {
		return ExitFailure
	}
	result, err := completion.NewValidationFailure(statement, next, source)
	if err != nil {
		return ExitFailure
	}
	return emitCompletion(writer, mode, result)
}

func emitExecutionList(writer io.Writer, mode outputMode, result completion.Result, executions []ExecutionSummaryView) int {
	if writer == nil || !result.Valid() || executions == nil {
		return ExitFailure
	}
	var content []byte
	if mode == humanOutput {
		content = renderCompletionHuman(result)
		var extra bytes.Buffer
		if len(executions) == 0 {
			extra.WriteString("executions: none\n")
		}
		for _, item := range executions {
			fmt.Fprintf(&extra, "execution: %s repository=%s work-item=%s:%s#%s status=%s gate=%s revision=%d\n", item.ExecutionID, item.RepositoryKey, item.WorkItem.Provider, item.WorkItem.Resource, item.WorkItem.ExternalID, item.Status, item.CurrentGate, item.Revision)
		}
		content = append(content, extra.Bytes()...)
	} else {
		base := completionEvent{Status: result.Status(), Result: result.Result().String(), References: result.References(), Next: result.Next().String(), Details: result.Details(), Provenance: provenanceEvent{Product: result.Provenance().Product(), Version: result.Provenance().Version(), Revision: result.Provenance().Revision(), SourceState: result.Provenance().SourceState()}}
		encoded, err := json.Marshal(executionListCompletionEvent{completionEvent: base, Executions: executions})
		if err != nil {
			return ExitFailure
		}
		content = append(encoded, '\n')
	}
	if len(content) > maxExecutionListOutputBytes {
		return ExitFailure
	}
	written, err := writer.Write(content)
	if err != nil || written != len(content) {
		return ExitFailure
	}
	return completionExitCode(result.Status())
}
