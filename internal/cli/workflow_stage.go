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

// Issue #276 public stage and coordination operations. Every mutation is a
// preview first; apply needs the exact --preview-digest plus the operation's
// own authority flags. Resume is a `stage run` re-invocation over an existing
// committed authority record, never a separate command.
const (
	workflowStageRunAction            action = "workflow_stage_run"
	workflowStageStatusAction         action = "workflow_stage_status"
	workflowStageCancelAction         action = "workflow_stage_cancel"
	workflowStageRetryAction          action = "workflow_stage_retry"
	workflowStageReconcileAction      action = "workflow_stage_reconcile"
	workflowStageIntegrateAction      action = "workflow_stage_integrate"
	workflowCoordinationListAction    action = "workflow_coordination_list"
	workflowCoordinationShowAction    action = "workflow_coordination_show"
	workflowCoordinationPublishAction action = "workflow_coordination_publish"
)

// MaxCoordinationWait bounds `coordination list --wait`; callers' own tool
// timeouts must stay longer.
const MaxCoordinationWait = 30 * time.Second

// WorkflowStageInput is the closed input of one stage or coordination command.
// It carries no lineage for publish: the running child context is resolved by
// the trusted launcher, never from flags.
type WorkflowStageInput struct {
	Operation                                string
	Project, Repository, WorkItem, Execution string
	ExpectedRevision                         uint64
	Stage, PlanFile, ValidatorPolicy         string
	WorkspaceRoot                            string
	PreviewDigest                            string
	AuthorizeLocal, AuthorizeExternal        bool
	Child, Attempt, Disposition              string
	Waivers                                  map[string]string
	Kind, Record                             string
	Fields                                   []string
	After                                    uint64
	Wait                                     time.Duration
}

// WorkflowStageService is optional so presentation services and mocks need
// not implement stage dispatch.
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
	workflowStageRunAction: "run", workflowStageStatusAction: "status", workflowStageCancelAction: "cancel",
	workflowStageRetryAction: "retry", workflowStageReconcileAction: "reconcile", workflowStageIntegrateAction: "integrate",
	workflowCoordinationListAction: "coordination-list", workflowCoordinationShowAction: "coordination-show", workflowCoordinationPublishAction: "coordination-publish",
}

func workflowStageOperation(operation action) bool {
	_, ok := workflowStageOperations[operation]
	return ok
}

func workflowStageCommands() []commandDefinition {
	return []commandDefinition{
		leaf("plan", "Preview deterministic stage proposal; never dispatches", workflowStagePlanAction),
		leaf("run", "Preview or start authorized first attempts; re-run resumes", workflowStageRunAction),
		leaf("status", "Inspect persisted stage children and attempts", workflowStageStatusAction),
		leaf("cancel", "Preview or request cancellation (never a stop claim)", workflowStageCancelAction),
		leaf("retry", "Preview or start one new authorized attempt", workflowStageRetryAction),
		leaf("reconcile", "Preview or append a reconciliation fact", workflowStageReconcileAction),
		leaf("integrate", "Preview or apply digest-bound local integration", workflowStageIntegrateAction),
	}
}

func workflowCoordinationCommands() []commandDefinition {
	return []commandDefinition{
		leaf("list", "List canonical coordination records (read-only)", workflowCoordinationListAction),
		leaf("show", "Inspect one canonical coordination record", workflowCoordinationShowAction),
		leaf("publish", "Publish into the running child's own inbox (child only)", workflowCoordinationPublishAction),
	}
}

// workflowStageFlagSet is the single argument registry of each command.
func workflowStageFlagSet(operation action, in *WorkflowStageInput) *flag.FlagSet {
	set := flag.NewFlagSet(string(operation), flag.ContinueOnError)
	set.SetOutput(io.Discard)
	in.Operation = workflowStageOperations[operation]
	if operation != workflowCoordinationPublishAction {
		set.StringVar(&in.Project, "project", "", "Configured Project identity; `<uuid-or-slug>`.")
		set.StringVar(&in.Repository, "repository", "", "Project-scoped Repository selector; `<key>`.")
		set.StringVar(&in.WorkItem, "work-item", "", "Exact existing Work Item; `github:<owner>/<repository>#<number>`.")
		set.StringVar(&in.Execution, "execution", "", "Exact Execution identity; `<execution-id>`.")
	}
	planned := func() {
		set.Uint64Var(&in.ExpectedRevision, "expected-revision", 0, "Exact current Execution revision; `<positive-integer>`.")
		set.StringVar(&in.Stage, "stage", "", "Stage ID in the Execution's pinned workflow revision; `<stage-id>`.")
	}
	preview := func() {
		set.StringVar(&in.PreviewDigest, "preview-digest", "", "Exact digest of the reviewed preview; absent means preview only; `<digest>`.")
		set.BoolVar(&in.AuthorizeLocal, "authorize-local", false, "Apply only the reviewed machine-local effects.")
	}
	switch operation {
	case workflowStageRunAction, workflowStageRetryAction:
		planned()
		set.StringVar(&in.PlanFile, "plan", "", "Bounded approved stage Plan document recompiled under the bound revision; `<json-file>`.")
		set.StringVar(&in.ValidatorPolicy, "validator-policy", "", "Reviewed validator policy referenced by digest; `<digest>`.")
		preview()
		set.BoolVar(&in.AuthorizeExternal, "authorize-external", false, "Allow the previewed vendor inference (Runtime child processes).")
		if operation == workflowStageRunAction {
			set.StringVar(&in.WorkspaceRoot, "workspace-root", "", "Operator-chosen private directory for isolated worktrees and attempt inboxes, outside the repository and state root; `<absolute-path>`.")
		}
		if operation == workflowStageRetryAction {
			set.StringVar(&in.Child, "child", "", "Child execution receiving one new attempt; `<child-id>`.")
			set.StringVar(&in.Disposition, "disposition", "", "Explicit workspace disposition after effects; `<preserve|reset|not_required>`.")
		}
	case workflowStageCancelAction:
		preview()
	case workflowStageReconcileAction:
		set.StringVar(&in.Child, "child", "", "Child execution of the attempt; `<child-id>`.")
		set.StringVar(&in.Attempt, "attempt", "", "Stale or unknown attempt to reconcile; `<attempt-id>`.")
		preview()
	case workflowStageIntegrateAction:
		planned()
		set.StringVar(&in.ValidatorPolicy, "validator-policy", "", "Reviewed validator policy run by integration; `<digest>`.")
		in.Waivers = map[string]string{}
		set.Func("waive", "Waive one optional child by reference; repeat `<child-id>=<reference>`.", func(v string) error {
			child, reference, ok := strings.Cut(v, "=")
			if !ok || child == "" || reference == "" {
				return flag.ErrHelp
			}
			if _, exists := in.Waivers[child]; exists {
				return flag.ErrHelp
			}
			in.Waivers[child] = reference
			return nil
		})
		preview()
	case workflowCoordinationListAction:
		set.StringVar(&in.Child, "child", "", "Child stream; `<child-id>`.")
		set.StringVar(&in.Attempt, "attempt", "", "Attempt filter; `<attempt-id>`.")
		set.StringVar(&in.Kind, "kind", "", "Record kind filter; `<kind>`.")
		set.Uint64Var(&in.After, "after", 0, "Only revisions after this one; `<revision>`.")
		set.DurationVar(&in.Wait, "wait", 0, "Bounded wait for a newer record; `<duration>` (max 30s).")
	case workflowCoordinationShowAction:
		set.StringVar(&in.Child, "child", "", "Child stream; `<child-id>`.")
		set.StringVar(&in.Record, "record", "", "Canonical record identity; `<record-id>`.")
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

func workflowStageFlags(operation action, args []string, require bool) (WorkflowStageInput, bool) {
	var in WorkflowStageInput
	set := workflowStageFlagSet(operation, &in)
	if invalidFlagSyntax(set, args, map[string]bool{"waive": true, "field": true}) || set.Parse(args) != nil || set.NArg() != 0 {
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
	rules := []inputRequirement{
		{name: "project", when: "no effective Project context is available", missing: v.Project == ""},
		{name: "repository", required: true, missing: v.Repository == ""},
		{name: "work-item", required: true, missing: v.WorkItem == ""},
		{name: "execution", required: true, missing: v.Execution == ""},
	}
	switch operation {
	case workflowStageRunAction, workflowStageRetryAction:
		rules = append(rules, inputRequirement{name: "expected-revision", required: true, missing: v.ExpectedRevision == 0}, inputRequirement{name: "stage", required: true, missing: v.Stage == ""}, inputRequirement{name: "plan", required: true, missing: v.PlanFile == ""}, inputRequirement{name: "validator-policy", required: true, missing: v.ValidatorPolicy == ""})
		if operation == workflowStageRunAction {
			rules = append(rules, inputRequirement{name: "workspace-root", required: true, missing: v.WorkspaceRoot == ""})
		}
		if operation == workflowStageRetryAction {
			rules = append(rules, inputRequirement{name: "child", required: true, missing: v.Child == ""})
		}
	case workflowStageReconcileAction:
		rules = append(rules, inputRequirement{name: "child", required: true, missing: v.Child == ""}, inputRequirement{name: "attempt", required: true, missing: v.Attempt == ""})
	case workflowStageIntegrateAction:
		rules = append(rules, inputRequirement{name: "expected-revision", required: true, missing: v.ExpectedRevision == 0}, inputRequirement{name: "stage", required: true, missing: v.Stage == ""}, inputRequirement{name: "validator-policy", required: true, missing: v.ValidatorPolicy == ""})
	case workflowCoordinationShowAction:
		rules = append(rules, inputRequirement{name: "child", required: true, missing: v.Child == ""}, inputRequirement{name: "record", required: true, missing: v.Record == ""})
	}
	return rules
}

// runWorkflowStage handles every #276 stage/coordination command. Authority
// flags are checked here only for shape; the application re-verifies them
// against the fresh preview and fails closed.
func runWorkflowStage(ctx context.Context, mode outputMode, args []string, service Service, source provenance.Value, out io.Writer) (bool, int) {
	command, _, rest, ok := resolveCommand(args)
	if !ok || !workflowStageOperation(command.operation) {
		return false, 0
	}
	in, ok := workflowStageFlags(command.operation, rest, true)
	if !ok || duplicateFlag(filterRepeatable(rest, "waive", "field")) {
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
	return true, presentEvent(out, mode, c.Status(), append(wire, '\n'), 1<<20)
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
	read := func(name string, act action, example string) skillOperationSpec {
		return skillOperationSpec{name: name, actions: []action{act}, modes: []skillModeSpec{{name: "default", effect: effectReadOnly, authority: "none; persisted state only, never a stop or success claim", actions: []action{act}, authorityInputs: []string{}, rejectedInputs: []string{}, example: example}}}
	}
	local := func(name string, act action, example string) skillOperationSpec {
		return skillOperationSpec{name: name, actions: []action{act}, modes: []skillModeSpec{{name: "default", effect: effectLocalMutation, authority: "preview first; apply only with the exact --preview-digest and --authorize-local", actions: []action{act}, authorityInputs: []string{"--preview-digest", "--authorize-local"}, rejectedInputs: []string{}, example: example}}}
	}
	inference := func(name string, act action, example string) skillOperationSpec {
		return skillOperationSpec{name: name, actions: []action{act}, modes: []skillModeSpec{{name: "default", effect: effectExternalMutation, authority: "preview first; dispatch only with the exact --preview-digest, --authorize-local and --authorize-external (vendor inference)", actions: []action{act}, authorityInputs: []string{"--preview-digest", "--authorize-local", "--authorize-external"}, rejectedInputs: []string{}, example: example}}}
	}
	planned := " --expected-revision <revision> --stage <stage-id>"
	return []skillOperationSpec{
		inference("stage.run", workflowStageRunAction, "axiom workflow stage run "+selectors+planned+" --plan <json-file> --validator-policy <digest> --workspace-root <absolute-path>"),
		read("stage.status", workflowStageStatusAction, "axiom workflow stage status "+selectors),
		local("stage.cancel", workflowStageCancelAction, "axiom workflow stage cancel "+selectors),
		inference("stage.retry", workflowStageRetryAction, "axiom workflow stage retry "+selectors+planned+" --plan <json-file> --validator-policy <digest> --child <child-id>"),
		local("stage.reconcile", workflowStageReconcileAction, "axiom workflow stage reconcile "+selectors+" --child <child-id> --attempt <attempt-id>"),
		local("stage.integrate", workflowStageIntegrateAction, "axiom workflow stage integrate "+selectors+planned+" --validator-policy <digest>"),
		read("coordination.list", workflowCoordinationListAction, "axiom workflow coordination list "+selectors),
		read("coordination.show", workflowCoordinationShowAction, "axiom workflow coordination show "+selectors+" --child <child-id> --record <record-id>"),
	}
}
