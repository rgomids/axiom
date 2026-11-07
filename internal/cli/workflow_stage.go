package cli

import (
	"context"
	"encoding/json"
	"flag"
	"io"
	"strings"
	"time"

	"github.com/rgomids/axiom/internal/provenance"
)

// Canonical Execution operations (ADR-0037, Spec 015 S1): one run action
// starts or continues the linked Work Item's Execution and progresses eligible
// work; stage status/cancel/retry/reconcile inspect and control that same
// Execution. Delivery packet is read-only canonical readiness; delivery advance
// performs the current exactly authorized phase over nested Execution facts.
const (
	workflowRunAction                 action = "workflow_run"
	workflowDeliveryAdvanceAction     action = "workflow_delivery_advance"
	workflowDeliveryPacketAction      action = "workflow_delivery_packet"
	workflowStageDiagnoseAction       action = "workflow_stage_diagnose"
	workflowStageStatusAction         action = "workflow_stage_status"
	workflowStageCancelAction         action = "workflow_stage_cancel"
	workflowStageRejectAction         action = "workflow_stage_reject"
	workflowStageRetryAction          action = "workflow_stage_retry"
	workflowStageReconcileAction      action = "workflow_stage_reconcile"
	workflowCoordinationListAction    action = "workflow_coordination_list"
	workflowCoordinationShowAction    action = "workflow_coordination_show"
	workflowCoordinationPublishAction action = "workflow_coordination_publish"
)

// retiredStageCommands were the Stage Plan / Graph dispatch ceremony. They
// return bounded replacement guidance; there is no compatibility executor.
var retiredStageCommands = map[string]bool{"plan": true, "run": true, "integrate": true}

// MaxCoordinationWait bounds `coordination list --wait`; callers' own tool
// timeouts must stay longer.
const MaxCoordinationWait = 30 * time.Second

// WorkflowStageInput is the closed input of one Execution, delivery or
// coordination command. It carries no lineage for publish: the running child
// context is resolved by the trusted launcher, never from flags.
type WorkflowStageInput struct {
	AuthorizePush, AuthorizePR               bool
	DeliveryRequest                          string
	InspectDelivery                          int
	AuthorizeHuman                           bool
	Operation                                string
	Project, Repository, WorkItem, Execution string
	ExpectedRevision                         uint64
	Stage                                    string
	PreviewDigest                            string
	AuthorizeLocal, AuthorizeExternal        bool
	Child, Attempt                           string
	Kind, Record                             string
	Fields                                   []string
	After                                    uint64
	Wait                                     time.Duration
	// Canonical Execution inputs. Selections are explicit per-agent run
	// overrides `[<stage-id>/]<agent-id>.<field>=<value>`; NewSelection
	// authorizes a retry to freeze them instead of the previous invocation.
	Workflow, Agent, ApproveGate, Abandon     string
	Inputs                                    []string
	ReviewStage, ResultDigest, ReviewEvidence string
	// ReviewNote is the human's own review text; the engine retains it as
	// the result-bound Evidence instead of a pre-retained --evidence ID.
	ReviewNote           string
	ReworkSource         string
	AuthorizeCommands    bool
	Selections           []string
	NewSelection         bool
	WorkspaceDisposition string
}

// WorkflowStageService is optional so presentation services and mocks need
// not implement Execution dispatch.
type WorkflowStageService interface {
	WorkflowStage(context.Context, WorkflowStageInput) WorkflowStageResponse
}

// WorkflowStageResponse is a canonical completion plus an operation payload
// rendered identically to JSON and human output.
type WorkflowStageResponse struct {
	Result  Result
	Payload any
}

var workflowStageOperations = map[action]string{
	workflowStageDiagnoseAction:   "diagnose",
	workflowRunAction:             "execution-run",
	workflowDeliveryAdvanceAction: "delivery-advance",
	workflowDeliveryPacketAction:  "delivery-packet", workflowStageStatusAction: "status", workflowStageCancelAction: "cancel",
	workflowStageRejectAction: "reject", workflowStageRetryAction: "retry", workflowStageReconcileAction: "reconcile",
	workflowCoordinationListAction: "coordination-list", workflowCoordinationShowAction: "coordination-show", workflowCoordinationPublishAction: "coordination-publish",
}

func workflowStageOperation(operation action) bool {
	_, ok := workflowStageOperations[operation]
	return ok
}

// executionOperation reports the canonical Execution commands, which take no
// Repository/Work Item selectors after start.
func executionOperation(operation action) bool {
	switch operation {
	case workflowStageDiagnoseAction, workflowStageRejectAction, workflowDeliveryAdvanceAction, workflowStageStatusAction, workflowStageCancelAction, workflowStageRetryAction, workflowStageReconcileAction:
		return true
	}
	return false
}

func workflowStageCommands() []commandDefinition {
	return []commandDefinition{
		leaf("diagnose", "Inspect one effective invocation read-only; no authentication or inference", workflowStageDiagnoseAction),
		leaf("status", "Inspect the Execution, attempts, effects and next safe action", workflowStageStatusAction),
		leaf("cancel", "Record cancel intent; the runner stops only its own process group", workflowStageCancelAction),
		leaf("reject", "Record an immutable human rejection of exact stage results", workflowStageRejectAction),
		leaf("retry", "Start one new attempt (frozen invocation unless --new-selection)", workflowStageRetryAction),
		leaf("reconcile", "Observe a lost runner's attempts; never signals or replays", workflowStageReconcileAction),
	}
}

func workflowCoordinationCommands() []commandDefinition {
	return []commandDefinition{
		leaf("list", "List canonical Execution messages (read-only)", workflowCoordinationListAction),
		leaf("show", "Inspect one canonical Execution message", workflowCoordinationShowAction),
		leaf("publish", "Retired publication spelling; always refuses (private engine ingress only)", workflowCoordinationPublishAction),
	}
}

// workflowStageFlagSet is the single argument registry of each command.
func workflowStageFlagSet(operation action, in *WorkflowStageInput) *flag.FlagSet {
	set := flag.NewFlagSet(string(operation), flag.ContinueOnError)
	set.SetOutput(io.Discard)
	in.Operation = workflowStageOperations[operation]
	selections := func() {
		set.Func("select", "Explicit per-agent run override; repeat `[<stage-id>/]<agent-id>.<runtime|model|effort|instructions|profile>=<value>`.", func(v string) error {
			in.Selections = append(in.Selections, v)
			return nil
		})
	}
	if operation == workflowRunAction || executionOperation(operation) {
		set.StringVar(&in.Project, "project", "", "Configured Project identity; `<uuid-or-slug>`.")
		set.StringVar(&in.Execution, "execution", "", "Exact Execution identity; `<execution-id>`.")
		switch operation {
		case workflowDeliveryAdvanceAction:
			set.StringVar(&in.DeliveryRequest, "request", "", "Exact JSON DeliveryRequest; omit only for --inspect.")
			set.StringVar(&in.PreviewDigest, "grant", "", "Digest of the exact reviewed request.")
			set.IntVar(&in.InspectDelivery, "inspect", -1, "Inspect this unknown delivery index only; no effect or disposition.")
			set.BoolVar(&in.AuthorizePush, "authorize-push", false, "Authorize only the exact push in --grant.")
			set.BoolVar(&in.AuthorizePR, "authorize-pr", false, "Authorize only the exact PR creation in --grant.")
			set.BoolVar(&in.AuthorizeHuman, "authorize-human", false, "Confirm operator authority; Runtime children are refused.")
			set.BoolVar(&in.AuthorizeLocal, "authorize-local", false, "Persist delivery intent/result or recovery observations in Execution.")
		case workflowStageRejectAction:
			set.StringVar(&in.ReviewStage, "stage", "", "Exact stage result to reject.")
			set.StringVar(&in.ResultDigest, "result-digest", "", "Exact result digest from status; the disposition grant.")
			set.StringVar(&in.ReviewEvidence, "evidence", "", "Retained result-bound human Evidence ID; or use --note.")
			set.StringVar(&in.ReviewNote, "note", "", "Human rejection note; retained by Axiom as the result-bound Evidence instead of --evidence.")
			set.BoolVar(&in.AuthorizeHuman, "authorize-human", false, "Confirm human disposition; Runtime children are refused.")
			set.BoolVar(&in.AuthorizeLocal, "authorize-local", false, "Persist the immutable rejection only; no external effect.")
		case workflowRunAction:
			set.StringVar(&in.ReworkSource, "rework-of", "", "Rejected source Execution; create a new linked run with fresh authority and selection.")
			set.StringVar(&in.Repository, "repository", "", "Project-scoped Repository selector of the linked Work Item; `<key>`.")
			set.StringVar(&in.WorkItem, "work-item", "", "Exact existing linked Work Item; `github:<owner>/<repository>#<number>`.")
			set.StringVar(&in.Workflow, "workflow", "", "Run the built-in minimal Workflow instead of the Project selection; `minimal`.")
			selections()
			set.Func("input", "Bind a logical artifact input source to a retained artifact ID; repeat once per source `<source>=<artifact-id>`.", func(v string) error { in.Inputs = append(in.Inputs, v); return nil })
			set.StringVar(&in.ReviewStage, "review-stage", "", "Stage whose exact successful results the human approves.")
			set.StringVar(&in.ResultDigest, "result-digest", "", "Exact stage result digest displayed by status.")
			set.StringVar(&in.ReviewEvidence, "evidence", "", "Retained Evidence artifact ID for this result review; or use --note.")
			set.StringVar(&in.ReviewNote, "note", "", "Human review note; retained by Axiom as the result-bound Evidence instead of --evidence.")
			set.BoolVar(&in.AuthorizeCommands, "authorize-commands", false, "Authorize the exact configured validation command argv within its declared effect bounds.")
			set.StringVar(&in.ApproveGate, "approve-gate", "", "Record the human gate decision of a stage awaiting it; `<stage-id>`.")
			set.BoolVar(&in.AuthorizeHuman, "authorize-human", false, "Record only this explicit human gate decision.")
			set.BoolVar(&in.AuthorizeExternal, "authorize-external", false, "Allow spawning the selected Runtime processes (vendor inference) within the Workflow effect bounds.")
		case workflowStageDiagnoseAction:
			set.StringVar(&in.Stage, "stage", "", "Stage ID in the frozen Workflow.")
			set.StringVar(&in.Agent, "agent", "", "Agent ID of that stage.")
			set.BoolVar(&in.NewSelection, "new-selection", false, "Preview a prospective selection only; grants no retry authority.")
			selections()
		case workflowStageRetryAction:
			set.BoolVar(&in.AuthorizeCommands, "authorize-commands", false, "Authorize the frozen validation command argv and effect bounds on this retry.")
			set.StringVar(&in.WorkspaceDisposition, "disposition", "", "Explicitly preserve a prior writable workspace; preserve. Reset is unsupported.")
			set.StringVar(&in.Stage, "stage", "", "Stage ID in the Execution's frozen Workflow; `<stage-id>`.")
			set.StringVar(&in.Agent, "agent", "", "Agent ID of that stage; `<agent-id>`.")
			selections()
			set.BoolVar(&in.NewSelection, "new-selection", false, "Authorize freezing a newly resolved invocation instead of the previous one.")
			set.BoolVar(&in.AuthorizeExternal, "authorize-external", false, "Allow spawning the selected Runtime process (vendor inference) within the Workflow effect bounds.")
		case workflowStageCancelAction:
			set.BoolVar(&in.AuthorizeLocal, "authorize-local", false, "Record only the cancel intent; the runner reports the observed stop.")
		case workflowStageReconcileAction:
			set.BoolVar(&in.AuthorizeLocal, "authorize-local", false, "Record lost-runner attempts as unknown; nothing is signalled or replayed.")
			set.StringVar(&in.Abandon, "abandon", "", "Record the human abandonment of one unknown attempt; nothing is replayed; `<attempt-id>`.")
			set.BoolVar(&in.AuthorizeHuman, "authorize-human", false, "Record only this explicit human disposition.")
		}
		return set
	}
	if operation != workflowCoordinationPublishAction {
		set.StringVar(&in.Project, "project", "", "Configured Project identity; `<uuid-or-slug>`.")
		set.StringVar(&in.Repository, "repository", "", "Project-scoped Repository selector; `<key>`.")
		set.StringVar(&in.WorkItem, "work-item", "", "Exact existing Work Item; `github:<owner>/<repository>#<number>`.")
		set.StringVar(&in.Execution, "execution", "", "Exact Execution identity; `<execution-id>`.")
	}
	switch operation {
	case workflowCoordinationListAction:
		set.StringVar(&in.Child, "child", "", "Sender agent filter; `<agent-id>` (no child Execution).")
		set.StringVar(&in.Attempt, "attempt", "", "Attempt filter; `<attempt-id>`.")
		set.StringVar(&in.Kind, "kind", "", "Message kind filter; question, answer, blocker or result.")
		set.Uint64Var(&in.After, "after", 0, "Retired stream cursor; nonzero values are refused.")
		set.DurationVar(&in.Wait, "wait", 0, "Retired stream wait; nonzero values are refused.")
	case workflowCoordinationShowAction:
		set.StringVar(&in.Child, "child", "", "Sender agent filter; `<agent-id>` (no child Execution).")
		set.StringVar(&in.Record, "record", "", "Execution message identity; `<message-id>`.")
	case workflowCoordinationPublishAction:
		set.StringVar(&in.Kind, "kind", "", "Record kind; `<kind>`.")
		set.Func("field", "Record field; repeat `<name>=<value>`.", func(v string) error {
			if !strings.Contains(v, "=") {
				return flag.ErrHelp
			}
			in.Fields = append(in.Fields, v)
			return nil
		})
	}
	return set
}

// --input repeats once per logical source; the application refuses a repeated
// source key when it freezes the bindings.
var workflowStageRepeatable = map[string]bool{"field": true, "select": true, "input": true}

func workflowStageFlags(operation action, args []string, require bool) (WorkflowStageInput, bool) {
	var in WorkflowStageInput
	set := workflowStageFlagSet(operation, &in)
	if invalidFlagSyntax(set, args, workflowStageRepeatable) || set.Parse(args) != nil || set.NArg() != 0 {
		return WorkflowStageInput{}, false
	}
	if in.WorkItem != "" {
		if _, _, _, ok := parseWorkItemSelector(in.WorkItem); !ok {
			return WorkflowStageInput{}, false
		}
	}
	if in.Wait < 0 || in.Wait > MaxCoordinationWait {
		return WorkflowStageInput{}, false
	}
	if require {
		for _, rule := range workflowStageRequirements(operation, in) {
			if rule.required && rule.missing {
				return WorkflowStageInput{}, false
			}
		}
	}
	return in, true
}

func workflowStageRequirements(operation action, v WorkflowStageInput) []inputRequirement {
	if operation == workflowCoordinationPublishAction {
		return []inputRequirement{{name: "kind", required: true, missing: v.Kind == ""}}
	}
	project := inputRequirement{name: "project", when: "no effective Project context is available", missing: v.Project == ""}
	if operation == workflowRunAction {
		return []inputRequirement{project,
			{name: "repository", required: v.Execution == "", when: "--execution is absent", missing: v.Repository == ""},
			{name: "work-item", required: v.Execution == "", when: "--execution is absent", missing: v.WorkItem == ""},
		}
	}
	if executionOperation(operation) {
		rules := []inputRequirement{project, {name: "execution", required: true, missing: v.Execution == ""}}
		if operation == workflowStageRejectAction {
			rules = append(rules, inputRequirement{name: "stage", required: true, missing: v.ReviewStage == ""}, inputRequirement{name: "result-digest", required: true, missing: v.ResultDigest == ""})
		}
		if operation == workflowStageRetryAction || operation == workflowStageDiagnoseAction {
			rules = append(rules, inputRequirement{name: "stage", required: true, missing: v.Stage == ""}, inputRequirement{name: "agent", required: true, missing: v.Agent == ""})
		}
		return rules
	}
	rules := []inputRequirement{
		project,
		{name: "repository", required: true, missing: v.Repository == ""},
		{name: "work-item", required: true, missing: v.WorkItem == ""},
		{name: "execution", required: true, missing: v.Execution == ""},
	}
	if operation == workflowCoordinationShowAction {
		rules = append(rules, inputRequirement{name: "child", required: true, missing: v.Child == ""}, inputRequirement{name: "record", required: true, missing: v.Record == ""})
	}
	return rules
}

// runWorkflowStage handles every Execution, delivery and coordination
// command. Authority flags are checked here only for shape; the application
// re-verifies them and fails closed.
func runWorkflowStage(ctx context.Context, mode outputMode, args []string, service Service, source provenance.Value, out io.Writer) (bool, int) {
	if len(args) >= 3 && args[0] == "workflow" && args[1] == "stage" && retiredStageCommands[args[2]] {
		return true, emitRetiredStageCommand(out, mode, args[2])
	}
	command, _, rest, ok := resolveCommand(args)
	if !ok || !workflowStageOperation(command.operation) {
		return false, 0
	}
	in, ok := workflowStageFlags(command.operation, rest, true)
	if !ok || duplicateFlag(filterRepeatable(rest, "field", "select", "input")) {
		return true, emitParserFailure(out, mode, command.operation, "invalid_input", source)
	}
	stages, ok := service.(WorkflowStageService)
	if !ok {
		return true, emit(out, mode, event{Operation: command.operation, Status: Failed, Category: "application_unavailable"})
	}
	response := stages.WorkflowStage(ctx, in)
	if response.Result.Completion == nil {
		return true, emitResponse(out, mode, command.operation, response.Result)
	}
	c := *response.Result.Completion
	value := struct {
		completionEvent
		Category string `json:"category"`
		Stage    any    `json:"stage,omitempty"`
	}{completionEvent{Status: c.Status(), Result: c.Result().String(), References: c.References(), Next: c.Next().String(), Details: c.Details(), Provenance: provenanceEvent{Product: c.Provenance().Product(), Version: c.Provenance().Version(), Revision: c.Provenance().Revision(), SourceState: c.Provenance().SourceState()}}, response.Result.Category, response.Payload}
	wire, err := json.Marshal(value)
	if err != nil {
		return true, ExitFailure
	}
	return true, presentDiagnosticEvent(out, mode, c.Status(), append(wire, '\n'), 1<<20, response.Result.Diagnostic)
}

// emitRetiredStageCommand reports the bounded replacement of a retired Stage
// Plan / Graph command. Nothing is read, written or spawned.
func emitRetiredStageCommand(out io.Writer, mode outputMode, name string) int {
	return emit(out, mode, event{Operation: action("workflow_stage_" + name), Status: Failed, Category: "command_retired",
		NextAction: "workflow stage " + name + " was retired with the Stage Plan and Validator Policy ceremony; use `axiom workflow run --repository <key> --work-item github:<owner>/<repository>#<number> --authorize-external`, then `axiom workflow stage status --execution <id>`"})
}

func filterRepeatable(args []string, names ...string) []string {
	out := []string{}
	for i := 0; i < len(args); i++ {
		skip := false
		for _, name := range names {
			if args[i] == "--"+name {
				i++
				skip = true
			} else if strings.HasPrefix(args[i], "--"+name+"=") {
				skip = true
			}
		}
		if !skip {
			out = append(out, args[i])
		}
	}
	return out
}

// workflowStageSkillSpecs are the thin axiom-work-item routes (I276-T35).
// Routing grants no authority; capability operations are not routed here.
func workflowStageSkillSpecs() []skillOperationSpec {
	selectors := "--project <uuid-or-slug> --repository <key> --work-item github:<owner>/<repository>#<number> --execution <id>"
	execution := "--project <uuid-or-slug> --execution <id>"
	read := func(name string, act action, example string) skillOperationSpec {
		return skillOperationSpec{name: name, actions: []action{act}, modes: []skillModeSpec{{name: "default", effect: effectReadOnly, authority: "none; persisted state only, never a stop or success claim", actions: []action{act}, authorityInputs: []string{}, rejectedInputs: []string{}, example: example}}}
	}
	return []skillOperationSpec{
		{name: "execution.run", actions: []action{workflowRunAction}, modes: []skillModeSpec{{name: "default", effect: effectExternalMutation, authority: "one run action; spawning the selected Runtimes needs --authorize-external; a human gate records only with --approve-gate and --authorize-human", actions: []action{workflowRunAction}, authorityInputs: []string{"--authorize-external", "--authorize-human"}, rejectedInputs: []string{}, example: "axiom workflow run --project <uuid-or-slug> --repository <key> --work-item github:<owner>/<repository>#<number> --select agent.runtime=<codex|claude> --select agent.model=<model> --select agent.effort=<runtime-default|token> --authorize-external"}}},
		{name: "delivery.advance", actions: []action{workflowDeliveryAdvanceAction}, modes: []skillModeSpec{{name: "default", effect: effectExternalMutation, authority: "preview without grant; exact --grant with --authorize-human --authorize-local and --authorize-push OR --authorize-pr; --inspect only reads the granted remote target and records observations", actions: []action{workflowDeliveryAdvanceAction}, authorityInputs: []string{"--grant", "--authorize-human", "--authorize-local", "--authorize-push", "--authorize-pr"}, rejectedInputs: []string{}, example: "axiom workflow delivery advance " + execution + " --request '<json>'"}}},
		{name: "stage.reject", actions: []action{workflowStageRejectAction}, modes: []skillModeSpec{{name: "default", effect: effectLocalMutation, authority: "exact --stage --result-digest and --evidence or --note with --authorize-human --authorize-local; Runtime children refused", actions: []action{workflowStageRejectAction}, authorityInputs: []string{"--authorize-human", "--authorize-local"}, rejectedInputs: []string{}, example: "axiom workflow stage reject " + execution + " --stage <stage-id> --result-digest <digest> --note <text> --authorize-human --authorize-local"}}},
		read("delivery.packet", workflowDeliveryPacketAction, "axiom workflow delivery packet "+selectors),
		read("stage.diagnose", workflowStageDiagnoseAction, "axiom workflow stage diagnose "+execution+" --stage <stage-id> --agent <agent-id>"),
		read("stage.status", workflowStageStatusAction, "axiom workflow stage status "+execution),
		{name: "stage.cancel", actions: []action{workflowStageCancelAction}, modes: []skillModeSpec{{name: "default", effect: effectLocalMutation, authority: "--authorize-local records cancel intent only; the runner reports requested versus observed stop", actions: []action{workflowStageCancelAction}, authorityInputs: []string{"--authorize-local"}, rejectedInputs: []string{}, example: "axiom workflow stage cancel " + execution}}},
		{name: "stage.retry", actions: []action{workflowStageRetryAction}, modes: []skillModeSpec{{name: "default", effect: effectExternalMutation, authority: "one new attempt with --authorize-external; prior writable workspace needs --disposition preserve; frozen invocation unless --new-selection", actions: []action{workflowStageRetryAction}, authorityInputs: []string{"--authorize-external", "--new-selection", "--disposition"}, rejectedInputs: []string{}, example: "axiom workflow stage retry " + execution + " --stage <stage-id> --agent <agent-id> --authorize-external"}}},
		{name: "stage.reconcile", actions: []action{workflowStageReconcileAction}, modes: []skillModeSpec{{name: "default", effect: effectLocalMutation, authority: "--authorize-local records lost-runner attempts unknown; abandoning one needs --abandon and --authorize-human; nothing is signalled or replayed", actions: []action{workflowStageReconcileAction}, authorityInputs: []string{"--authorize-local", "--authorize-human"}, rejectedInputs: []string{}, example: "axiom workflow stage reconcile " + execution}}},
		read("coordination.list", workflowCoordinationListAction, "axiom workflow coordination list "+selectors),
		read("coordination.show", workflowCoordinationShowAction, "axiom workflow coordination show "+selectors+" --child <child-id> --record <record-id>"),
	}
}
