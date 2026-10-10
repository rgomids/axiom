package cli

import (
	"context"
	"encoding/json"
	"flag"
	"io"

	"github.com/rgomids/axiom/internal/completion"
	"github.com/rgomids/axiom/internal/provenance"
	"github.com/rgomids/axiom/internal/workflowcompiler"
	"github.com/rgomids/axiom/internal/workflowdefinition"
)

const workflowStagePlanAction action = "workflow_stage_plan"

const maxStagePlanOutputBytes = 4 << 20

// WorkflowStagePlanInput is the closed input of `workflow stage plan`: an exact
// ExecutionRef (selectors, Execution ID and expected revision), the pinned
// stage ID and the bounded approved Plan document. It carries no authority.
type WorkflowStagePlanInput struct {
	Project, Repository, WorkItem, Execution string
	ExpectedRevision                         uint64
	Stage, PlanFile                          string
}

// WorkflowStagePlanService is optional so existing presentation services and
// mocks need not implement stage planning. It stays out of Service.
type WorkflowStagePlanService interface {
	WorkflowStagePlan(context.Context, WorkflowStagePlanInput) WorkflowStagePlanResponse
}

// WorkflowStagePlanResponse is either a stage planning completion (Planned) or
// an earlier context/admission Result rendered through the ordinary path.
type WorkflowStagePlanResponse struct {
	Result     Result
	Planned    bool
	Execution  *workflowcompiler.ExecutionRef
	Workflow   *workflowdefinition.Ref
	StageID    string
	Conditions []string
	Plan       *workflowcompiler.StagePlan
}

type stagePlanEvent struct {
	completionEvent
	Category         string                         `json:"category"`
	ConfirmedEffects []string                       `json:"confirmedEffects"`
	Execution        *workflowcompiler.ExecutionRef `json:"executionRef,omitempty"`
	Workflow         *workflowdefinition.Ref        `json:"workflowRef,omitempty"`
	StageID          string                         `json:"stageId,omitempty"`
	Conditions       []string                       `json:"conditions,omitempty"`
	Plan             *workflowcompiler.StagePlan    `json:"plan,omitempty"`
}

// workflowStagePlanFlagSet is the single argument registry of the command.
func workflowStagePlanFlagSet(input *WorkflowStagePlanInput) *flag.FlagSet {
	set := flag.NewFlagSet(string(workflowStagePlanAction), flag.ContinueOnError)
	set.SetOutput(io.Discard)
	set.StringVar(&input.Project, "project", "", "Configured Project identity; `<uuid-or-slug>`.")
	set.StringVar(&input.Repository, "repository", "", "Project-scoped Repository selector; `<key>`.")
	set.StringVar(&input.WorkItem, "work-item", "", "Exact existing Work Item; `github:<owner>/<repository>#<number>`.")
	set.StringVar(&input.Execution, "execution", "", "Exact Execution identity; `<execution-id>`.")
	set.Uint64Var(&input.ExpectedRevision, "expected-revision", 0, "Exact current Execution revision; `<positive-integer>`. A changed Execution needs a fresh plan.")
	set.StringVar(&input.Stage, "stage", "", "Stage ID in the Execution's pinned workflow revision; `<stage-id>`.")
	set.StringVar(&input.PlanFile, "plan", "", "Bounded approved stage Plan document; `<json-file>`. Supplies work scope, controls and authority ceiling.")
	return set
}

func workflowStagePlanFlags(args []string, requireInputs bool) (WorkflowStagePlanInput, bool) {
	var input WorkflowStagePlanInput
	set := workflowStagePlanFlagSet(&input)
	if invalidFlagSyntax(set, args, nil) {
		return WorkflowStagePlanInput{}, false
	}
	if err := set.Parse(args); err != nil || set.NArg() != 0 {
		return WorkflowStagePlanInput{}, false
	}
	if input.WorkItem != "" {
		if _, _, _, ok := parseWorkItemSelector(input.WorkItem); !ok {
			return WorkflowStagePlanInput{}, false
		}
	}
	if requireInputs && missingStagePlanInputs(input) {
		return WorkflowStagePlanInput{}, false
	}
	return input, true
}

func missingStagePlanInputs(input WorkflowStagePlanInput) bool {
	for _, rule := range stagePlanRequirements(input) {
		if rule.required && rule.missing {
			return true
		}
	}
	return false
}

func stagePlanRequirements(v WorkflowStagePlanInput) []inputRequirement {
	return []inputRequirement{
		{name: "project", when: "no effective Project context is available", missing: v.Project == ""},
		{name: "repository", required: true, missing: v.Repository == ""},
		{name: "work-item", required: true, missing: v.WorkItem == ""},
		{name: "execution", required: true, missing: v.Execution == ""},
		{name: "expected-revision", required: true, missing: v.ExpectedRevision == 0},
		{name: "stage", required: true, missing: v.Stage == ""},
		{name: "plan", required: true, missing: v.PlanFile == ""},
	}
}

// runWorkflowStagePlan handles `workflow stage plan`. It reports handled=false
// for any other command so the ordinary dispatch continues untouched.
func runWorkflowStagePlan(ctx context.Context, args []string, service Service, source provenance.Value, mode outputMode, stdout io.Writer) (bool, int) {
	if !commandMatches(args, workflowStagePlanAction) {
		return false, 0
	}
	input, ok := workflowStagePlanFlags(args[3:], true)
	if !ok {
		return true, emitStagePlanParserFailure(stdout, mode, source)
	}
	planner, ok := service.(WorkflowStagePlanService)
	if !ok {
		return true, emit(stdout, mode, event{Operation: workflowStagePlanAction, Status: Failed, Category: "application_unavailable"})
	}
	response := planner.WorkflowStagePlan(ctx, input)
	if !response.Planned || response.Result.Completion == nil {
		return true, emitResponse(stdout, mode, workflowStagePlanAction, response.Result)
	}
	return true, emitStagePlan(stdout, mode, response)
}

func emitStagePlanParserFailure(writer io.Writer, mode outputMode, source provenance.Value) int {
	statement, err := provenance.NewText("Stage plan input is invalid", provenance.AxiomAuthored)
	if err != nil {
		return ExitFailure
	}
	next, err := provenance.NewText(helpNext(writer, "Provide exact --repository, --work-item, --execution, --expected-revision, --stage and --plan, with no other input"), provenance.AxiomAuthored)
	if err != nil {
		return ExitFailure
	}
	result, err := completion.NewValidationFailure(statement, next, source)
	if err != nil {
		return ExitFailure
	}
	return emitCompletion(writer, mode, result)
}

func emitStagePlan(writer io.Writer, mode outputMode, response WorkflowStagePlanResponse) int {
	result := *response.Result.Completion
	if writer == nil || !result.Valid() {
		return ExitFailure
	}
	base := completionEvent{Status: result.Status(), Result: result.Result().String(), References: result.References(), Next: result.Next().String(), Details: result.Details(), Provenance: provenanceEvent{Product: result.Provenance().Product(), Version: result.Provenance().Version(), Revision: result.Provenance().Revision(), SourceState: result.Provenance().SourceState()}}
	// Planning is read-only: it never confirms an effect.
	wire, err := json.Marshal(stagePlanEvent{completionEvent: base, Category: response.Result.Category, ConfirmedEffects: []string{}, Execution: response.Execution, Workflow: response.Workflow, StageID: response.StageID, Conditions: response.Conditions, Plan: response.Plan})
	if err != nil {
		return ExitFailure
	}
	return presentEvent(writer, mode, result.Status(), append(wire, '\n'), maxStagePlanOutputBytes)
}
