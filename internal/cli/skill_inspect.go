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

type skillInspection struct {
	Name     string         `json:"name"`
	Commands []skillCommand `json:"commands"`
}

// These are the existing thin-skill delegations, not a second argument registry.
// Inventory and inspection pointers are checked against the skill bundle.
func skillOperations(name string) []action {
	switch name {
	case "axiom-project-configure":
		return []action{configureAction}
	case "axiom-project-list":
		return []action{listAction}
	case "axiom-project-show":
		return []action{showAction, resolveAction}
	case "axiom-work-item-create":
		return []action{workItemCreateAction, workItemSelectAction}
	case "axiom-work-item-run":
		return []action{workflowStartAction, workflowAdvanceAction, workflowFactAction, workflowResumeAction, workflowStatusAction, workflowEvidenceAction, workflowReconcileAction}
	case "axiom-work-item-status":
		return []action{workflowStatusAction, workflowEvidenceAction}
	}
	return nil
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
	result := skillInspection{Name: name, Commands: make([]skillCommand, 0, len(operations))}
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
