package cli

import (
	"context"
	"flag"
	"fmt"
	"io"

	"github.com/rgomids/axiom/internal/completion"
	"github.com/rgomids/axiom/internal/workitem"
)

// Issue #230 (I230-T06): the work-item list, update, close and reopen verbs.
// Presentation parses and renders only; list, preview, no-op and authority
// semantics belong to the Work Item application.

// WorkItemLifecycleService is optional so existing presentation services and
// mocks need not implement the #230 Work Item lifecycle.
type WorkItemLifecycleService interface {
	WorkItemList(context.Context, WorkItemInput) Result
	WorkItemUpdate(context.Context, WorkItemInput) Result
	WorkItemClose(context.Context, WorkItemInput) Result
	WorkItemReopen(context.Context, WorkItemInput) Result
}

func workItemLifecycleAction(operation action) bool {
	return operation == workItemListAction || operation == workItemUpdateAction || operation == workItemCloseAction || operation == workItemReopenAction
}

// workItemUpdateFlags declares only the title and the Axiom-authored section
// flags create uses. Type, classification and story fields stay create-time.
func workItemUpdateFlags(set *flag.FlagSet, values *requestInput) {
	set.StringVar(&values.title, "title", "", "Replacement Issue title; `<text>`. Omission preserves the current title.")
	set.Var(&values.elaboratedSections, "elaborated-section", "Axiom-authored replacement section; `<name>=<content>`. Names: problem, desired_outcome, context, scope, constraints, non_goals, acceptance_expectations. Conflicts with a verbatim value for that section.")
	set.StringVar(&values.problem, "problem", "", "Verbatim replacement problem section; `<text>`. Omission preserves it.")
	set.StringVar(&values.desiredOutcome, "desired-outcome", "", "Verbatim replacement desired outcome; `<text>`. Omission preserves it.")
	set.StringVar(&values.context, "context", "", "Verbatim replacement context; `<text>`. Omission preserves it.")
	set.StringVar(&values.scope, "scope", "", "Verbatim replacement scope; `<text>`. Omission preserves it.")
	set.StringVar(&values.constraints, "constraints", "", "Verbatim replacement constraints; `<text>`. Omission preserves it.")
	set.StringVar(&values.nonGoals, "non-goals", "", "Verbatim replacement exclusions; `<text>`. Omission preserves it.")
	set.StringVar(&values.acceptance, "acceptance", "", "Verbatim replacement acceptance expectations; `<text>`. Omission preserves it.")
}

func dispatchWorkItemLifecycle(ctx context.Context, operation action, value WorkItemInput, service Service) Result {
	lifecycle, ok := service.(WorkItemLifecycleService)
	if !ok {
		return Result{Status: Failed, Category: "application_unavailable"}
	}
	switch operation {
	case workItemListAction:
		return lifecycle.WorkItemList(ctx, WorkItemInput{Project: value.Project, Repository: value.Repository})
	case workItemUpdateAction:
		return lifecycle.WorkItemUpdate(ctx, value)
	case workItemCloseAction:
		return lifecycle.WorkItemClose(ctx, value)
	default:
		return lifecycle.WorkItemReopen(ctx, value)
	}
}

type workItemLifecycleCompletionEvent struct {
	completionEvent
	Category string                  `json:"category"`
	Change   *workitem.ChangePreview `json:"change,omitempty"`
	WorkItem *WorkItemView           `json:"workItem,omitempty"`
	// WorkItems is present, possibly empty, exactly for work-item list.
	WorkItems *[]WorkItemView `json:"workItems,omitempty"`
}

func emitWorkItemLifecycleCompletion(writer io.Writer, mode outputMode, result completion.Result, response Result) int {
	if writer == nil || !result.Valid() {
		return ExitFailure
	}
	base := completionEvent{Status: result.Status(), Result: result.Result().String(), References: result.References(), Next: result.Next().String(), Details: result.Details(), Provenance: provenanceEvent{Product: result.Provenance().Product(), Version: result.Provenance().Version(), Revision: result.Provenance().Revision(), SourceState: result.Provenance().SourceState()}}
	value := workItemLifecycleCompletionEvent{completionEvent: base, Category: response.Category, Change: response.WorkItemChange, WorkItem: response.WorkItem}
	if response.WorkItems != nil {
		value.WorkItems = &response.WorkItems
	}
	var content []byte
	if mode == humanOutput {
		content = renderCompletionHuman(result)
		content = fmt.Appendf(content, "category: %s\n", response.Category)
		for _, item := range response.WorkItems {
			content = fmt.Appendf(content, "work-item %s [%s] repository-key=%s provider=%s resource=%s external-id=%s\n", item.URL, item.State, item.RepositoryKey, item.Provider, item.Resource, item.ExternalID)
		}
		if response.WorkItemChange != nil {
			extra, err := marshalWorkItemValue(response.WorkItemChange, true)
			if err != nil {
				return ExitFailure
			}
			content = append(content, "preview: "...)
			content = append(content, extra...)
		}
	} else {
		wire, err := marshalWorkItemValue(value, false)
		if err != nil {
			return ExitFailure
		}
		content = wire
	}
	if len(content) > maxWorkItemPreviewOutputBytes {
		return ExitFailure
	}
	if written, err := writer.Write(content); err != nil || written != len(content) {
		return ExitFailure
	}
	return completionExitCode(result.Status())
}
