package cli

import (
	"encoding/json"
	"flag"
	"io"
	"strings"

	"github.com/rgomids/axiom/internal/completion"
	"github.com/rgomids/axiom/internal/provenance"
)

type skillArgument struct {
	Name          string   `json:"name"`
	Required      bool     `json:"required"`
	RequiredWhen  string   `json:"requiredWhen,omitempty"`
	Description   string   `json:"description"`
	AcceptedForms []string `json:"acceptedForms"`
	Repeatable    bool     `json:"repeatable"`
}

type skillCommand struct {
	Command   string          `json:"command"`
	Arguments []skillArgument `json:"arguments"`
}

// skillOperationMode is one distinct authority path of a domain operation.
// Effect is read-only, preview-only, local-mutation or external-mutation.
// AuthorityInputs are the explicit authority inputs that path accepts;
// RejectedInputs are inputs Lingo refuses on that path before any effect.
// Semantic is "allowed" for read-only paths and "unambiguous-only" for every
// other path: an ambiguous intent never resolves to it.
type skillOperationMode struct {
	Name            string   `json:"name"`
	Commands        []string `json:"commands"`
	Selector        string   `json:"selector,omitempty"`
	Effect          string   `json:"effect"`
	Authority       string   `json:"authority"`
	AuthorityInputs []string `json:"authorityInputs"`
	RejectedInputs  []string `json:"rejectedInputs"`
	Semantic        string   `json:"semanticResolution"`
	Example         string   `json:"example"`
}

type skillOperation struct {
	Name     string               `json:"name"`
	Commands []string             `json:"commands"`
	Modes    []skillOperationMode `json:"modes"`
}

type skillInspection struct {
	Name       string           `json:"name"`
	Operations []skillOperation `json:"operations"`
	Commands   []skillCommand   `json:"commands"`
}

type skillModeSpec struct {
	name, selector, effect, authority, example string
	actions                                    []action
	authorityInputs, rejectedInputs            []string
}

// actions fixes the operation's public command order; every action belongs to
// exactly one mode.
type skillOperationSpec struct {
	name    string
	actions []action
	modes   []skillModeSpec
}

const (
	effectReadOnly         = "read-only"
	effectPreviewOnly      = "preview-only"
	effectLocalMutation    = "local-mutation"
	effectExternalMutation = "external-mutation"
)

// Each spec mirrors one existing application path and the authority Lingo
// already enforces on it; the metadata describes, never grants, authority.
var (
	projectCreateMode = skillModeSpec{name: "create", effect: effectLocalMutation, actions: []action{configureAction},
		authority:       "preview first; publish only with the exact --preview-digest plus --authorize-local",
		authorityInputs: []string{"--preview-digest", "--authorize-local"}, rejectedInputs: []string{},
		example: "axiom project configure --slug <slug> --name <name> --repository <key>=<absolute-path> --work-item-provider <provider>"}
	projectEditMode = skillModeSpec{name: "edit", selector: "--project", effect: effectLocalMutation, actions: []action{configureAction},
		authority:       "preview first; publish only with the returned --project-id, the exact --preview-digest plus --authorize-local; Repository attach/update/detach and provider changes use this mode",
		authorityInputs: []string{"--project-id", "--preview-digest", "--authorize-local"}, rejectedInputs: []string{},
		example: "axiom project configure --project <uuid-or-slug> --name <name>"}
	projectListMode = skillModeSpec{name: "default", effect: effectReadOnly, actions: []action{listAction}, authority: "none",
		authorityInputs: []string{}, rejectedInputs: []string{}, example: "axiom project list"}
	projectShowMode = skillModeSpec{name: "default", selector: "--selector", effect: effectReadOnly, actions: []action{showAction}, authority: "none",
		authorityInputs: []string{}, rejectedInputs: []string{}, example: "axiom project show --selector <slug-or-id>"}
	projectValidateMode = skillModeSpec{name: "default", selector: "--project", effect: effectReadOnly, actions: []action{validateAction}, authority: "none",
		authorityInputs: []string{}, rejectedInputs: []string{}, example: "axiom project validate --project <uuid-or-slug>"}
	projectArchiveMode = skillModeSpec{name: "default", selector: "--project", effect: effectLocalMutation, actions: []action{projectArchiveAction},
		authority:       "machine-local only; preview first; archive only with the exact --preview-digest plus --authorize-local",
		authorityInputs: []string{"--preview-digest", "--authorize-local"}, rejectedInputs: []string{},
		example: "axiom project archive --project <uuid-or-slug>"}
	projectReactivateMode = skillModeSpec{name: "default", selector: "--project", effect: effectLocalMutation, actions: []action{projectReactivateAction},
		authority:       "machine-local only; preview first; reactivate only with the exact --preview-digest plus --authorize-local",
		authorityInputs: []string{"--preview-digest", "--authorize-local"}, rejectedInputs: []string{},
		example: "axiom project reactivate --project <uuid-or-slug>"}
	integrationListMode = skillModeSpec{name: "list", selector: "--project", effect: effectReadOnly, actions: []action{integrationListAction}, authority: "none",
		authorityInputs: []string{}, rejectedInputs: []string{}, example: "axiom integration list --project <uuid-or-slug>"}
	integrationShowMode = skillModeSpec{name: "show", selector: "--project", effect: effectReadOnly, actions: []action{integrationShowAction}, authority: "none",
		authorityInputs: []string{}, rejectedInputs: []string{}, example: "axiom integration show --project <uuid-or-slug> --integration <key>"}
	integrationValidateMode = skillModeSpec{name: "validate", selector: "--project", effect: effectReadOnly, actions: []action{integrationValidateAction}, authority: "none",
		authorityInputs: []string{}, rejectedInputs: []string{}, example: "axiom integration validate --project <uuid-or-slug>"}
	integrationDisableMode = skillModeSpec{name: "disable", selector: "--project", effect: effectLocalMutation, actions: []action{integrationDisableAction},
		authority:       "machine-local eligibility only; preview first; disable only with the exact --preview-digest plus --authorize-local; never revokes credentials or touches Providers",
		authorityInputs: []string{"--preview-digest", "--authorize-local"}, rejectedInputs: []string{},
		example: "axiom integration disable --project <uuid-or-slug> --integration <key>"}
	integrationEnableMode = skillModeSpec{name: "enable", selector: "--project", effect: effectLocalMutation, actions: []action{integrationEnableAction},
		authority:       "machine-local eligibility only; preview first; enable only with the exact --preview-digest plus --authorize-local",
		authorityInputs: []string{"--preview-digest", "--authorize-local"}, rejectedInputs: []string{},
		example: "axiom integration enable --project <uuid-or-slug> --integration <key>"}
	integrationRemoveMode = skillModeSpec{name: "remove", selector: "--project", effect: effectLocalMutation, actions: []action{integrationRemoveAction},
		authority:       "portable declaration only; preview first; remove only with the returned --project-id, the exact --preview-digest plus --authorize-local; no credential, MCP, Runtime or Provider cleanup",
		authorityInputs: []string{"--project-id", "--preview-digest", "--authorize-local"}, rejectedInputs: []string{},
		example: "axiom integration remove --project <uuid-or-slug> --integration <key>"}
	workItemListMode = skillModeSpec{name: "default", selector: "--project", effect: effectReadOnly, actions: []action{workItemListAction}, authority: "none",
		authorityInputs: []string{}, rejectedInputs: []string{}, example: "axiom work-item list --project <uuid-or-slug>"}
	workItemShowMode = skillModeSpec{name: "default", selector: "--project", effect: effectReadOnly, actions: []action{workItemShowAction}, authority: "none",
		authorityInputs: []string{}, rejectedInputs: []string{}, example: "axiom work-item show --project <uuid-or-slug> --repository <key> --work-item github:<owner>/<repository>#<number>"}
	workItemUpdateMode = skillModeSpec{name: "default", selector: "--project", effect: effectExternalMutation, actions: []action{workItemUpdateAction},
		authority:       "preview first; update the title or Axiom-authored sections only with the exact --preview-digest plus --authorize-external",
		authorityInputs: []string{"--preview-digest", "--authorize-external"}, rejectedInputs: []string{},
		example: "axiom work-item update --project <uuid-or-slug> --repository <key> --work-item github:<owner>/<repository>#<number> --title <text>"}
	workItemCommentMode = skillModeSpec{name: "default", selector: "--project", effect: effectExternalMutation, actions: []action{workItemCommentAction},
		authority:       "preview first; comment only with the exact --preview-digest plus --authorize-external",
		authorityInputs: []string{"--preview-digest", "--authorize-external"}, rejectedInputs: []string{},
		example: "axiom work-item comment --project <uuid-or-slug> --repository <key> --work-item github:<owner>/<repository>#<number> --message <text>"}
	workItemCloseMode = skillModeSpec{name: "default", selector: "--project", effect: effectExternalMutation, actions: []action{workItemCloseAction},
		authority:       "preview first; close only with the exact --preview-digest plus --authorize-external; never deletes the Provider Work Item",
		authorityInputs: []string{"--preview-digest", "--authorize-external"}, rejectedInputs: []string{},
		example: "axiom work-item close --project <uuid-or-slug> --repository <key> --work-item github:<owner>/<repository>#<number>"}
	workItemReopenMode = skillModeSpec{name: "default", selector: "--project", effect: effectExternalMutation, actions: []action{workItemReopenAction},
		authority:       "preview first; reopen only with the exact --preview-digest plus --authorize-external",
		authorityInputs: []string{"--preview-digest", "--authorize-external"}, rejectedInputs: []string{},
		example: "axiom work-item reopen --project <uuid-or-slug> --repository <key> --work-item github:<owner>/<repository>#<number>"}
	executionListMode = skillModeSpec{name: "list", selector: "--project", effect: effectReadOnly, actions: []action{workflowListAction}, authority: "none",
		authorityInputs: []string{}, rejectedInputs: []string{}, example: "axiom workflow list --project <uuid-or-slug>"}
	workItemNewMode = skillModeSpec{name: "new", effect: effectExternalMutation, actions: []action{workItemCreateAction},
		authority:       "preview first; create only with the exact --preview-digest plus --authorize-external",
		authorityInputs: []string{"--preview-digest", "--authorize-external"}, rejectedInputs: []string{},
		example: "axiom work-item create --project <uuid-or-slug> --repository <key> --provider-repository <owner>/<repository>"}
	workItemExistingMode = skillModeSpec{name: "existing", effect: effectLocalMutation, actions: []action{workItemSelectAction},
		authority:       "preview first; link only with the exact --preview-digest plus --authorize-local",
		authorityInputs: []string{"--preview-digest", "--authorize-local"}, rejectedInputs: []string{},
		example: "axiom work-item select --project <uuid-or-slug> --repository <key> --work-item github:<owner>/<repository>#<number>"}
	workflowTransitionMode = skillModeSpec{name: "transition", effect: effectLocalMutation, actions: []action{workflowStartAction, workflowAdvanceAction, workflowResumeAction},
		authority:       "workflow start previews the Runtime/Profile resolution first and starts only with the exact --runtime-preview; Lingo enforces the exact Execution revision and gate rules",
		authorityInputs: []string{}, rejectedInputs: []string{},
		example: "axiom workflow start --project <uuid-or-slug> --repository <key> --work-item github:<owner>/<repository>#<number> --role <role> --complexity <complexity> --capabilities <capabilities> --runtime <codex|claude>"}
	workflowFactMode = skillModeSpec{name: "fact", effect: effectLocalMutation, actions: []action{workflowFactAction},
		authority:       "records a workflow fact only with --authorize-local and the exact Execution revision",
		authorityInputs: []string{"--authorize-local"}, rejectedInputs: []string{},
		example: "axiom workflow fact --project <uuid-or-slug> --repository <key> --work-item github:<owner>/<repository>#<number> --execution <id> --expected-revision <revision> --fact <fact> --active --reference <reference> --authorize-local"}
	workflowReconcileMode = skillModeSpec{name: "reconcile", effect: effectExternalMutation, actions: []action{workflowReconcileAction},
		authority:       "preview first; publish a Provider projection only with the exact --preview-digest plus --authorize-external",
		authorityInputs: []string{"--preview-digest", "--authorize-external"}, rejectedInputs: []string{},
		example: "axiom workflow reconcile --project <uuid-or-slug> --repository <key> --work-item github:<owner>/<repository>#<number> --execution <id> --expected-revision <revision>"}
	workItemStatusMode = skillModeSpec{name: "default", effect: effectReadOnly, actions: []action{workflowStatusAction, workflowEvidenceAction}, authority: "none",
		authorityInputs: []string{}, rejectedInputs: []string{},
		example: "axiom workflow status --project <uuid-or-slug> --repository <key> --work-item github:<owner>/<repository>#<number> --execution <id>"}
)

var (
	projectConfigure = skillOperationSpec{name: "configure", actions: []action{configureAction}, modes: []skillModeSpec{projectCreateMode, projectEditMode}}
	projectList      = skillOperationSpec{name: "list", actions: []action{listAction}, modes: []skillModeSpec{projectListMode}}
	projectShow      = skillOperationSpec{name: "show", actions: []action{showAction}, modes: []skillModeSpec{projectShowMode}}
	workItemCreate   = skillOperationSpec{name: "create", actions: []action{workItemCreateAction, workItemSelectAction}, modes: []skillModeSpec{workItemNewMode, workItemExistingMode}}
	workItemRun      = skillOperationSpec{name: "run",
		actions: []action{workflowStartAction, workflowAdvanceAction, workflowFactAction, workflowResumeAction, workflowReconcileAction},
		modes:   []skillModeSpec{workflowTransitionMode, workflowFactMode, workflowReconcileMode}}
	workItemStatus = skillOperationSpec{name: "status", actions: []action{workflowStatusAction, workflowEvidenceAction, workflowListAction}, modes: []skillModeSpec{workItemStatusMode, executionListMode}}

	// Issue #230 lifecycle operations.
	projectValidate    = skillOperationSpec{name: "validate", actions: []action{validateAction}, modes: []skillModeSpec{projectValidateMode}}
	projectArchive     = skillOperationSpec{name: "archive", actions: []action{projectArchiveAction}, modes: []skillModeSpec{projectArchiveMode}}
	projectReactivate  = skillOperationSpec{name: "reactivate", actions: []action{projectReactivateAction}, modes: []skillModeSpec{projectReactivateMode}}
	projectIntegration = skillOperationSpec{name: "integration",
		actions: []action{integrationListAction, integrationShowAction, integrationValidateAction, integrationDisableAction, integrationEnableAction, integrationRemoveAction},
		modes:   []skillModeSpec{integrationListMode, integrationShowMode, integrationValidateMode, integrationDisableMode, integrationEnableMode, integrationRemoveMode}}
	workItemList    = skillOperationSpec{name: "list", actions: []action{workItemListAction}, modes: []skillModeSpec{workItemListMode}}
	workItemShow    = skillOperationSpec{name: "show", actions: []action{workItemShowAction}, modes: []skillModeSpec{workItemShowMode}}
	workItemUpdate  = skillOperationSpec{name: "update", actions: []action{workItemUpdateAction}, modes: []skillModeSpec{workItemUpdateMode}}
	workItemComment = skillOperationSpec{name: "comment", actions: []action{workItemCommentAction}, modes: []skillModeSpec{workItemCommentMode}}
	workItemClose   = skillOperationSpec{name: "close", actions: []action{workItemCloseAction}, modes: []skillModeSpec{workItemCloseMode}}
	workItemReopen  = skillOperationSpec{name: "reopen", actions: []action{workItemReopenAction}, modes: []skillModeSpec{workItemReopenMode}}
)

// These are thin Runtime-to-Lingo delegations, not a second argument registry.
// Argument metadata still comes from each executable command's real FlagSet.
// Only canonical domain skills expose these operation modes.
func skillOperationSpecs(name string) []skillOperationSpec {
	switch name {
	case "axiom-project":
		return []skillOperationSpec{projectConfigure, projectList, projectShow, projectValidate, projectArchive, projectReactivate, projectIntegration}
	case "axiom-work-item":
		return []skillOperationSpec{workItemCreate, workItemRun, workItemStatus, workItemList, workItemShow, workItemUpdate, workItemComment, workItemClose, workItemReopen}

	}
	return nil
}

func semanticResolution(effect string) string {
	if effect == effectReadOnly {
		return "allowed"
	}
	return "unambiguous-only"
}

func skillOperations(name string) []action {
	specs := skillOperationSpecs(name)
	seen := map[action]bool{}
	operations := make([]action, 0)
	for _, spec := range specs {
		for _, operation := range spec.actions {
			if seen[operation] {
				continue
			}
			seen[operation] = true
			operations = append(operations, operation)
		}
	}
	return operations
}

func skillFlagSet(operation action, values *requestInput) *flag.FlagSet {
	switch operation {
	case projectArchiveAction, projectReactivateAction:
		return projectLifecycleFlagSet(operation, &ProjectLifecycleInput{})
	case listAction:
		return projectListFlagSet(&ProjectListInput{})
	case validateAction:
		return projectValidateFlagSet(new(string), &ProjectLifecycleInput{})
	case workflowListAction:
		return executionListFlagSet(&ExecutionListInput{})
	}
	if integrationOperation(operation) {
		return integrationFlagSet(operation, &IntegrationInput{})
	}
	if knownWorkItem(operation) {
		return workItemFlagSet(operation, values)
	}
	if knownWorkflow(operation) {
		return workflowFlagSet(operation, values)
	}
	return projectFlagSet(operation, values)
}

func skillCommandName(operation action) string {
	_, path := commandByAction(operation)
	return "axiom " + strings.Join(path, " ")
}

func inspectSkill(name string) (skillInspection, bool) {
	operations := skillOperations(name)
	if len(operations) == 0 {
		return skillInspection{}, false
	}
	result := skillInspection{Name: name, Operations: []skillOperation{}, Commands: make([]skillCommand, 0, len(operations))}
	for _, spec := range skillOperationSpecs(name) {
		item := skillOperation{Name: spec.name, Commands: []string{}, Modes: make([]skillOperationMode, 0, len(spec.modes))}
		for _, operation := range spec.actions {
			item.Commands = append(item.Commands, skillCommandName(operation))
		}
		for _, mode := range spec.modes {
			view := skillOperationMode{Name: mode.name, Commands: make([]string, 0, len(mode.actions)), Selector: mode.selector, Effect: mode.effect, Authority: mode.authority, AuthorityInputs: mode.authorityInputs, RejectedInputs: mode.rejectedInputs, Semantic: semanticResolution(mode.effect), Example: mode.example}
			for _, operation := range mode.actions {
				view.Commands = append(view.Commands, skillCommandName(operation))
			}
			item.Modes = append(item.Modes, view)
		}
		result.Operations = append(result.Operations, item)
	}
	for _, operation := range operations {
		var values requestInput
		set := skillFlagSet(operation, &values)
		repeatable := repeatableFlags(set)
		rules := skillRequirements(operation, values)
		command := skillCommand{Command: skillCommandName(operation), Arguments: []skillArgument{}}
		set.VisitAll(func(f *flag.Flag) {
			valueName, description := flag.UnquoteUsage(f)
			arg := skillArgument{Name: "--" + f.Name, Description: description, Repeatable: repeatable[f.Name]}
			if b, ok := f.Value.(interface{ IsBoolFlag() bool }); ok && b.IsBoolFlag() {
				arg.AcceptedForms = []string{arg.Name, arg.Name + "=<boolean>"}
			} else {
				arg.AcceptedForms = []string{arg.Name + " " + valueName, arg.Name + "=" + valueName}
			}
			for _, rule := range rules {
				if rule.name == f.Name {
					arg.Required, arg.RequiredWhen = rule.required, rule.when
				}
			}
			command.Arguments = append(command.Arguments, arg)
		})
		result.Commands = append(result.Commands, command)
	}
	return result, true
}

// InspectSkill handles the closed inspection surface before service composition.
// It depends only on embedded declarations, provenance and its output writer.
func InspectSkill(args []string, source provenance.Value, output io.Writer) (bool, int) {
	output = withHelpGuidance(output, args)
	mode, args := parseOutputMode(args)
	if len(args) == 0 || args[0] != "skill" {
		return false, 0
	}
	if len(args) != 3 || !commandMatches(args, skillInspectAction) {
		return true, emitParserFailure(output, mode, skillInspectAction, "invalid_input", source)
	}
	skill, ok := inspectSkill(args[2])
	if !ok {
		return true, emitParserFailure(output, mode, skillInspectAction, "invalid_input", source)
	}
	message, _ := provenance.NewText("Skill arguments inspected", provenance.AxiomAuthored)
	result, err := completion.New(completion.Facts{Completed: true}, message, nil, provenance.Text{}, "", source)
	if err != nil || output == nil {
		return true, ExitFailure
	}
	base := completionEvent{Status: result.Status(), Result: result.Result().String(), Provenance: provenanceEvent{Product: source.Product(), Version: source.Version(), Revision: source.Revision(), SourceState: source.SourceState()}}
	wire, err := json.Marshal(struct {
		completionEvent
		Skill skillInspection `json:"skill"`
	}{base, skill})
	if err != nil {
		return true, ExitFailure
	}
	// A multi-command skill can exceed the ordinary completion-only bound.
	return true, presentEvent(output, mode, result.Status(), append(wire, '\n'), 64*1024)
}

const skillInspectAction action = "skill_inspect"
