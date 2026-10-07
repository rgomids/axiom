package cli

import (
	"encoding/json"
	"flag"
	"fmt"
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
		example: "axiom --json project configure --slug <slug> --name <name> --repository <key>=<absolute-path> --work-item-provider <provider>"}
	projectEditMode = skillModeSpec{name: "edit", selector: "--project", effect: effectPreviewOnly, actions: []action{configureAction},
		authority:       "preview only; publication is not available and the replay inputs fail with unsupported_edit_authority",
		authorityInputs: []string{}, rejectedInputs: []string{"--project-id", "--preview-digest", "--authorize-local"},
		example: "axiom --json project configure --project <uuid-or-slug> --name <name>"}
	projectListMode = skillModeSpec{name: "default", effect: effectReadOnly, actions: []action{listAction}, authority: "none",
		authorityInputs: []string{}, rejectedInputs: []string{}, example: "axiom --json project list"}
	projectShowMode = skillModeSpec{name: "default", selector: "--selector", effect: effectReadOnly, actions: []action{showAction}, authority: "none",
		authorityInputs: []string{}, rejectedInputs: []string{}, example: "axiom --json project show --selector <slug-or-id>"}
	workItemNewMode = skillModeSpec{name: "new", effect: effectExternalMutation, actions: []action{workItemCreateAction},
		authority:       "preview first; create only with the exact --preview-digest plus --authorize-external",
		authorityInputs: []string{"--preview-digest", "--authorize-external"}, rejectedInputs: []string{},
		example: "axiom --json work-item create --project <uuid-or-slug> --repository <key> --provider-repository <owner>/<repository>"}
	workItemExistingMode = skillModeSpec{name: "existing", effect: effectLocalMutation, actions: []action{workItemSelectAction},
		authority:       "preview first; link only with the exact --preview-digest plus --authorize-local",
		authorityInputs: []string{"--preview-digest", "--authorize-local"}, rejectedInputs: []string{},
		example: "axiom --json work-item select --project <uuid-or-slug> --repository <key> --work-item github:<owner>/<repository>#<number>"}
	workflowTransitionMode = skillModeSpec{name: "transition", effect: effectLocalMutation, actions: []action{workflowStartAction, workflowAdvanceAction, workflowResumeAction},
		authority:       "workflow start previews the Runtime/Profile resolution first and starts only with the exact --runtime-preview; Lingo enforces the exact Execution revision and gate rules",
		authorityInputs: []string{}, rejectedInputs: []string{},
		example: "axiom --json workflow start --project <uuid-or-slug> --repository <key> --work-item github:<owner>/<repository>#<number> --role <role> --complexity <complexity> --capabilities <capabilities> --runtime <codex|claude>"}
	workflowFactMode = skillModeSpec{name: "fact", effect: effectLocalMutation, actions: []action{workflowFactAction},
		authority:       "records a workflow fact only with --authorize-local and the exact Execution revision",
		authorityInputs: []string{"--authorize-local"}, rejectedInputs: []string{},
		example: "axiom --json workflow fact --project <uuid-or-slug> --repository <key> --work-item github:<owner>/<repository>#<number> --execution <id> --expected-revision <revision> --fact <fact> --active --reference <reference> --authorize-local"}
	workflowReconcileMode = skillModeSpec{name: "reconcile", effect: effectExternalMutation, actions: []action{workflowReconcileAction},
		authority:       "preview first; publish a Provider projection only with the exact --preview-digest plus --authorize-external",
		authorityInputs: []string{"--preview-digest", "--authorize-external"}, rejectedInputs: []string{},
		example: "axiom --json workflow reconcile --project <uuid-or-slug> --repository <key> --work-item github:<owner>/<repository>#<number> --execution <id> --expected-revision <revision>"}
	workItemStatusMode = skillModeSpec{name: "default", effect: effectReadOnly, actions: []action{workflowStatusAction, workflowEvidenceAction}, authority: "none",
		authorityInputs: []string{}, rejectedInputs: []string{},
		example: "axiom --json workflow status --project <uuid-or-slug> --repository <key> --work-item github:<owner>/<repository>#<number> --execution <id>"}
)

var (
	projectConfigure = skillOperationSpec{name: "configure", actions: []action{configureAction}, modes: []skillModeSpec{projectCreateMode, projectEditMode}}
	projectList      = skillOperationSpec{name: "list", actions: []action{listAction}, modes: []skillModeSpec{projectListMode}}
	projectShow      = skillOperationSpec{name: "show", actions: []action{showAction}, modes: []skillModeSpec{projectShowMode}}
	workItemCreate   = skillOperationSpec{name: "create", actions: []action{workItemCreateAction, workItemSelectAction}, modes: []skillModeSpec{workItemNewMode, workItemExistingMode}}
	workItemRun      = skillOperationSpec{name: "run",
		actions: []action{workflowStartAction, workflowAdvanceAction, workflowFactAction, workflowResumeAction, workflowReconcileAction},
		modes:   []skillModeSpec{workflowTransitionMode, workflowFactMode, workflowReconcileMode}}
	workItemStatus = skillOperationSpec{name: "status", actions: []action{workflowStatusAction, workflowEvidenceAction}, modes: []skillModeSpec{workItemStatusMode}}
)

// These are thin Runtime-to-Lingo delegations, not a second argument registry.
// Argument metadata still comes from each executable command's real FlagSet.
// The canonical domain skills and the compatibility skills share mode specs,
// so one operation cannot describe two authority contracts.
func skillOperationSpecs(name string) []skillOperationSpec {
	switch name {
	case "axiom-project":
		return []skillOperationSpec{projectConfigure, projectList, projectShow}
	case "axiom-work-item":
		return []skillOperationSpec{workItemCreate, workItemRun, workItemStatus}
	case "axiom-project-configure":
		return []skillOperationSpec{projectConfigure}
	case "axiom-project-list":
		return []skillOperationSpec{projectList}
	case "axiom-project-show":
		return []skillOperationSpec{projectShow}
	case "axiom-work-item-create":
		return []skillOperationSpec{workItemCreate}
	case "axiom-work-item-run":
		// The compatibility run skill also reads status and Evidence.
		return []skillOperationSpec{{name: "run",
			actions: []action{workflowStartAction, workflowAdvanceAction, workflowFactAction, workflowResumeAction, workflowStatusAction, workflowEvidenceAction, workflowReconcileAction},
			modes:   []skillModeSpec{workflowTransitionMode, workflowFactMode, workflowReconcileMode, workItemStatusMode}}}
	case "axiom-work-item-status":
		return []skillOperationSpec{workItemStatus}
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
	if knownWorkItem(operation) {
		return workItemFlagSet(operation, values)
	}
	if knownWorkflow(operation) {
		return workflowFlagSet(operation, values)
	}
	return projectFlagSet(operation, values)
}

func skillCommandName(operation action) string {
	if knownWorkItem(operation) {
		return "axiom work-item " + strings.TrimPrefix(string(operation), "work_item_")
	}
	if knownWorkflow(operation) {
		return "axiom workflow " + strings.TrimPrefix(string(operation), "workflow_")
	}
	return "axiom project " + string(operation)
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
	mode, args := parseOutputMode(args)
	if len(args) == 0 || args[0] != "skill" {
		return false, 0
	}
	if len(args) != 3 || args[1] != "inspect" {
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
	var content []byte
	if mode == jsonOutput {
		base := completionEvent{Status: result.Status(), Result: result.Result().String(), Provenance: provenanceEvent{Product: source.Product(), Version: source.Version(), Revision: source.Revision(), SourceState: source.SourceState()}}
		content, err = json.Marshal(struct {
			completionEvent
			Skill skillInspection `json:"skill"`
		}{base, skill})
		content = append(content, '\n')
	} else {
		var text strings.Builder
		text.Write(renderCompletionHuman(result))
		fmt.Fprintf(&text, "skill: %s\nRequired inputs apply to noninteractive requests; guided invocation remains available.\n", skill.Name)
		for _, operation := range skill.Operations {
			fmt.Fprintf(&text, "operation %s\n", operation.Name)
			for _, mode := range operation.Modes {
				fmt.Fprintf(&text, "  mode %s: effect=%s semantic=%s\n    authority: %s\n    example: %s\n    commands: %s\n", mode.Name, mode.Effect, mode.Semantic, mode.Authority, mode.Example, strings.Join(mode.Commands, "; "))
				if len(mode.RejectedInputs) != 0 {
					fmt.Fprintf(&text, "    rejected: %s\n", strings.Join(mode.RejectedInputs, " "))
				}
			}
		}
		for _, command := range skill.Commands {
			fmt.Fprintln(&text, command.Command)
			if len(command.Arguments) == 0 {
				fmt.Fprintln(&text, "  No arguments.")
			}
			for _, arg := range command.Arguments {
				requirement := "optional"
				if arg.Required {
					requirement = "required"
				} else if arg.RequiredWhen != "" {
					requirement = "required when " + arg.RequiredWhen
				}
				fmt.Fprintf(&text, "  %s (%s; repeatable=%t): %s\n    forms: %s\n", arg.Name, requirement, arg.Repeatable, arg.Description, strings.Join(arg.AcceptedForms, "; "))
			}
		}
		content = []byte(text.String())
	}
	// A multi-command skill can exceed the ordinary completion-only bound.
	if err != nil || len(content) > 64*1024 {
		return true, ExitFailure
	}
	n, err := output.Write(content)
	if err != nil || n != len(content) {
		return true, ExitFailure
	}
	return true, ExitSuccess
}

const skillInspectAction action = "skill_inspect"
