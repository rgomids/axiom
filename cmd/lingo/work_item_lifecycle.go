package main

import (
	"context"
	"strings"

	"github.com/rgomids/axiom/internal/cli"
	"github.com/rgomids/axiom/internal/completion"
	"github.com/rgomids/axiom/internal/projectapp"
	"github.com/rgomids/axiom/internal/workitem"
)

// Issue #230 (I230-T06) composition of the Work Item lifecycle verbs. Each
// method passes the single pre-effect admission gate first; every rule lives
// in internal/workitem and every GitHub detail in internal/githubissues.

var _ cli.WorkItemLifecycleService = lifecycleService{}

func (s lifecycleService) WorkItemList(ctx context.Context, input cli.WorkItemInput) cli.Result {
	if blocked := s.gate(ctx, input.Project, projectapp.AdmitWorkItemList); blocked != nil {
		return *blocked
	}
	result := s.workItems.List(ctx, workitem.ListTarget{ProjectSelector: input.Project, RepositoryKey: input.Repository})
	response := workItemResult(result, s.provenance)
	if result.Status == completion.Success {
		response.WorkItems = make([]cli.WorkItemView, 0, len(result.Links))
		for _, link := range result.Links {
			response.WorkItems = append(response.WorkItems, cli.WorkItemView{ProjectID: link.ProjectID, RepositoryKey: link.RepositoryKey, Provider: link.Provider, Resource: link.Resource, ExternalID: link.ExternalID, URL: link.URL, State: link.State})
		}
	}
	return response
}

func (s lifecycleService) WorkItemUpdate(ctx context.Context, input cli.WorkItemInput) cli.Result {
	if blocked := s.gate(ctx, input.Project, projectapp.AdmitWorkItemUpdate); blocked != nil {
		return *blocked
	}
	if failed := unsupportedWorkItemProvider(input, s); failed != nil {
		return *failed
	}
	section := func(supplied, name string) workitem.SectionInput {
		return workitem.SectionInput{Supplied: supplied, Elaborated: input.ElaboratedSections[name]}
	}
	update := workitem.UpdateInput{
		Target: lifecycleTarget(input), Selector: workItemExternalID(input), Title: input.Title,
		Problem: section(input.Problem, "problem"), DesiredOutcome: section(input.DesiredOutcome, "desired_outcome"),
		Context: section(input.Context, "context"), Scope: section(input.Scope, "scope"),
		Constraints: section(input.Constraints, "constraints"), NonGoals: section(input.NonGoals, "non_goals"),
		Acceptance: section(input.Acceptance, "acceptance_expectations"),
	}
	return workItemResult(s.workItems.Update(ctx, update, input.PreviewDigest, input.AuthorizeExternal), s.provenance)
}

func (s lifecycleService) WorkItemClose(ctx context.Context, input cli.WorkItemInput) cli.Result {
	if blocked := s.gate(ctx, input.Project, projectapp.AdmitWorkItemClose); blocked != nil {
		return *blocked
	}
	if failed := unsupportedWorkItemProvider(input, s); failed != nil {
		return *failed
	}
	return workItemResult(s.workItems.Close(ctx, lifecycleTarget(input), workItemExternalID(input), input.PreviewDigest, input.AuthorizeExternal), s.provenance)
}

func (s lifecycleService) WorkItemReopen(ctx context.Context, input cli.WorkItemInput) cli.Result {
	if blocked := s.gate(ctx, input.Project, projectapp.AdmitWorkItemReopen); blocked != nil {
		return *blocked
	}
	if failed := unsupportedWorkItemProvider(input, s); failed != nil {
		return *failed
	}
	return workItemResult(s.workItems.Reopen(ctx, lifecycleTarget(input), workItemExternalID(input), input.PreviewDigest, input.AuthorizeExternal), s.provenance)
}

func lifecycleTarget(input cli.WorkItemInput) workitem.Target {
	return workitem.Target{ProjectSelector: input.Project, RepositoryKey: input.Repository, ProviderResource: input.ProviderRepository}
}

// unsupportedWorkItemProvider rejects an exact selector naming a Provider this
// build does not implement, before any Provider call.
func unsupportedWorkItemProvider(input cli.WorkItemInput, s lifecycleService) *cli.Result {
	if input.Provider == "" || input.Provider == "github" {
		return nil
	}
	result := workItemResult(workitem.Result{Status: completion.ValidationFailure, Category: "invalid_work_item_input"}, s.provenance)
	return &result
}

var workItemLifecycleTexts = map[string][2]string{
	"work_items_listed":               {"Axiom-linked Work Items listed", "Inspect one with work-item show; unlinked Provider Issues are never listed"},
	"work_item_update_ready":          {"Work Item change ready for review", "Repeat the same command with this preview digest and explicit external authority"},
	"work_item_comment_ready":         {"Work Item change ready for review", "Repeat the same command with this preview digest and explicit external authority"},
	"work_item_close_ready":           {"Work Item change ready for review", "Repeat the same command with this preview digest and explicit external authority"},
	"work_item_reopen_ready":          {"Work Item change ready for review", "Repeat the same command with this preview digest and explicit external authority"},
	"work_item_unchanged":             {"Work Item already matches the requested fields", "No Provider change is needed"},
	"work_item_already_closed":        {"Work Item is already closed", "No Provider change is needed"},
	"work_item_already_open":          {"Work Item is already open", "No Provider change is needed"},
	"work_item_updated":               {"Work Item updated", "Inspect the Issue; type and classification were not changed"},
	"work_item_commented":             {"Work Item comment added", "Inspect the Issue; a comment is not workflow progress or acceptance"},
	"work_item_closed":                {"Work Item closed", "Closing is not workflow progress or acceptance; reopen if it was closed in error"},
	"work_item_reopened":              {"Work Item reopened", "Inspect the Issue before continuing work"},
	"work_item_update_empty":          {"No Work Item field to update", "Supply --title or at least one section flag"},
	"invalid_work_item_title":         {"Work Item change input is invalid", "Provide bounded single-line text without control characters or surrounding whitespace"},
	"invalid_work_item_comment":       {"Work Item comment is invalid", "Provide bounded text without control characters or surrounding whitespace"},
	"work_item_document_unrecognized": {"Current Issue body does not have the Axiom section structure", "Update only the title, or edit the Issue in the Provider; nothing was changed"},
	"provider_mutation_failed":        {"Provider change did not complete", "Preview again to observe current Provider state; if the Provider already shows the change, refresh the link with work-item select"},
	"provider_comment_ambiguous":      {"Provider comment outcome is uncertain", "Inspect the Issue comments before retrying; a retry may duplicate the comment"},
	"work_item_change_cancelled":      {"Work Item operation cancelled", "Preview again when ready; no Provider change was issued"},
	"work_item_list_exceeds_limit":    {"Too many linked Work Items to list at once", "Filter with --repository"},
}

func workItemLifecycleText(result workitem.Result) (string, string, bool) {
	if result.Change != nil && strings.HasPrefix(result.Category, "provider_confirmed_local_") || result.Category == "local_link_committed_recovery_required" && result.Change != nil {
		return "Provider change confirmed but the local Work Item link was not updated", "Do not repeat the Provider change; run recovery inspect if it reports state, then refresh the link with work-item select", true
	}
	text, ok := workItemLifecycleTexts[result.Category]
	return text[0], text[1], ok
}
