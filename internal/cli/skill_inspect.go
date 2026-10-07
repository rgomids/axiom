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

type skillOperation struct {
	Name      string   `json:"name"`
	Commands  []string `json:"commands"`
	Authority string   `json:"authority"`
	Example   string   `json:"example"`
}

type skillInspection struct {
	Name       string           `json:"name"`
	Operations []skillOperation `json:"operations"`
	Commands   []skillCommand   `json:"commands"`
}

type skillOperationSpec struct {
	name      string
	actions   []action
	authority string
	example   string
}

// These are thin Runtime-to-Lingo delegations, not a second argument registry.
// Argument metadata still comes from each executable command's real FlagSet.
func skillOperationSpecs(name string) []skillOperationSpec {
	switch name {
	case "axiom-project":
		return []skillOperationSpec{
			{name: "configure", actions: []action{configureAction}, authority: "local mutation; exact preview digest plus --authorize-local", example: "axiom-project configure"},
			{name: "list", actions: []action{listAction}, authority: "read-only", example: "axiom-project list"},
			{name: "show", actions: []action{showAction}, authority: "read-only", example: "axiom-project show <project-selector>"},
		}
	case "axiom-work-item":
		return []skillOperationSpec{
			{name: "create", actions: []action{workItemCreateAction, workItemSelectAction}, authority: "external create requires exact preview digest plus --authorize-external; selecting an existing item requires --authorize-local", example: "axiom-work-item create"},
			{name: "run", actions: []action{workflowStartAction, workflowAdvanceAction, workflowFactAction, workflowResumeAction, workflowReconcileAction}, authority: "workflow transitions preserve existing revision gates; local workflow facts require --authorize-local", example: "axiom-work-item run"},
			{name: "status", actions: []action{workflowStatusAction, workflowEvidenceAction}, authority: "read-only", example: "axiom-work-item status"},
		}
	case "axiom-project-configure":
		return []skillOperationSpec{{name: "configure", actions: []action{configureAction}, authority: "local mutation; exact preview digest plus --authorize-local", example: "axiom-project-configure"}}
	case "axiom-project-list":
		return []skillOperationSpec{{name: "list", actions: []action{listAction}, authority: "read-only", example: "axiom-project-list"}}
	case "axiom-project-show":
		return []skillOperationSpec{{name: "show", actions: []action{showAction}, authority: "read-only", example: "axiom-project-show <project-selector>"}}
	case "axiom-work-item-create":
		return []skillOperationSpec{{name: "create", actions: []action{workItemCreateAction, workItemSelectAction}, authority: "external create requires exact preview digest plus --authorize-external; selecting an existing item requires --authorize-local", example: "axiom-work-item-create"}}
	case "axiom-work-item-run":
		return []skillOperationSpec{{name: "run", actions: []action{workflowStartAction, workflowAdvanceAction, workflowFactAction, workflowResumeAction, workflowStatusAction, workflowEvidenceAction, workflowReconcileAction}, authority: "workflow transitions preserve existing revision gates; local workflow facts require --authorize-local", example: "axiom-work-item-run"}}
	case "axiom-work-item-status":
		return []skillOperationSpec{{name: "status", actions: []action{workflowStatusAction, workflowEvidenceAction}, authority: "read-only", example: "axiom-work-item-status"}}
	}
	return nil
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
		item := skillOperation{Name: spec.name, Authority: spec.authority, Example: spec.example, Commands: make([]string, 0, len(spec.actions))}
		for _, operation := range spec.actions {
			item.Commands = append(item.Commands, skillCommandName(operation))
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
			fmt.Fprintf(&text, "operation %s\n  authority: %s\n  example: %s\n  commands: %s\n", operation.Name, operation.Authority, operation.Example, strings.Join(operation.Commands, "; "))
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
