package cli

import (
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/rgomids/axiom/internal/provenance"
)

func Help(writer io.Writer) int { return renderHelp(writer, &publicCommands, nil) }

// HandleHelp runs before application composition. Flag values and tokens after
// -- are data, never requests for help. Output selection cannot override help.
func HandleHelp(args []string, source provenance.Value, writer io.Writer) (bool, int) {
	command, path, help, valid := helpTarget(args)
	if !help {
		return false, 0
	}
	if !valid {
		if writer == nil {
			return true, ExitFailure
		}
		mode, _ := parseOutputMode(args)
		operation, issue := action("unknown"), "invalid_command"
		if command.operation != "" && command.operation != "help" {
			operation, issue = command.operation, "invalid_input"
		}
		return true, emitParserFailure(guidanceWriter{writer, helpInvocation(path)}, mode, operation, issue, source)
	}
	return true, renderHelp(writer, command, path)
}

func helpTarget(args []string) (*commandDefinition, []string, bool, bool) {
	return helpTargetFrom(&publicCommands, args)
}

func helpTargetFrom(root *commandDefinition, args []string) (*commandDefinition, []string, bool, bool) {
	current := root
	var path []string
	var provided []string
	help, valid := false, true
	for i := 0; i < len(args); i++ {
		token := args[i]
		if token == "--" {
			if help {
				valid = false
			}
			break
		}
		if token == "--help" || token == "-h" {
			help = true
			continue
		}
		if token == "--json" || token == "--human" {
			continue
		}
		if current == root && token == "--session" {
			i++
			continue
		}
		if current == root && strings.HasPrefix(token, "--session=") {
			continue
		}
		if len(current.children) != 0 {
			if !valid {
				continue
			}
			child := commandChild(current, token)
			if child == nil {
				valid = false
				continue
			}
			current = child
			path = append(path, token)
			continue
		}
		if current.operation == "help" && len(path) == 1 {
			help = true
			valid = false
			continue
		}
		provided = append(provided, token)
		name, _, inline := strings.Cut(strings.TrimPrefix(token, "--"), "=")
		if strings.HasPrefix(token, "--") {
			if f := commandFlagSet(current.operation).Lookup(name); f != nil && !inline && !booleanFlag(f) {
				if i+1 < len(args) {
					provided = append(provided, args[i+1])
				}
				i++
			}
		}
	}
	if current.operation == "help" && valid {
		return root, nil, true, true
	}
	if help && valid && current.operation != "" {
		valid = validHelpInputs(current.operation, provided)
	}
	return current, path, help, valid
}

// Validate only supplied syntax with execution's own registrations. Required
// input absence and domain validation never gate help or trigger resolution.
func validHelpInputs(operation action, args []string) bool {
	if operation == skillInspectAction {
		if len(args) == 0 {
			return true
		}
		if len(args) != 1 {
			return false
		}
		_, ok := inspectSkill(args[0])
		return ok
	}
	if operation == windowsPermissionsRestoreAction {
		return windowsPermissionsSyntax(args, false)
	}
	switch {
	case knownWorkItem(operation):
		values, ok := workItemFlags(operation, args)
		if !ok {
			return false
		}
		return selectorRequestIssueWithPresence(operation, values, false) == ""
	case knownWorkflow(operation):
		values, ok := workflowFlags(operation, args)
		if !ok {
			return false
		}
		return selectorRequestIssueWithPresence(operation, values, false) == ""
	case operation == configureAction:
		values, ok := flags(operation, args)
		if !ok {
			return false
		}
		issue := configureRequestIssue(values)
		return issue == "" || issue == "missing_required_input"
	case operation == validateAction:
		slug, input, ok := projectValidationFlags(args)
		return ok && !projectValidationSelectorsConflict(slug, input.Project)
	case strings.HasPrefix(string(operation), "context_"):
		_, ok := projectContextFlags(append([]string{strings.TrimPrefix(string(operation), "context_")}, args...), false)
		return ok
	case operation == runtimeProfilePreviewAction:
		_, ok := runtimePreviewFlagsWithPresence(args, false)
		return ok
	case integrationOperation(operation):
		_, ok := integrationFlagsWithPresence(operation, args, false)
		return ok
	case operation == workflowListAction:
		_, ok := executionListFlagsWithPresence(args, false)
		return ok
	}
	set := commandFlagSet(operation)
	return !invalidFlagSyntax(set, args, repeatableFlags(set)) && set.Parse(args) == nil && set.NArg() == 0
}

func booleanFlag(f *flag.Flag) bool {
	b, ok := f.Value.(interface{ IsBoolFlag() bool })
	return ok && b.IsBoolFlag()
}

func helpInvocation(path []string) string {
	if len(path) == 0 {
		return "axiom --help"
	}
	return "axiom " + strings.Join(path, " ") + " --help"
}

type guidanceWriter struct {
	io.Writer
	help string
}

func withHelpGuidance(writer io.Writer, args []string) io.Writer {
	if writer == nil {
		return nil
	}
	return guidanceWriter{writer, HelpCommand(args)}
}

// HelpCommand returns the closest registered help invocation without echoing input.
func HelpCommand(args []string) string {
	_, path, _, _ := helpTarget(args)
	return helpInvocation(path)
}

func helpNext(writer io.Writer, next string) string {
	if guided, ok := writer.(guidanceWriter); ok {
		return next + "; see " + guided.help
	}
	return next
}

func renderHelp(writer io.Writer, command *commandDefinition, path []string) int {
	if writer == nil {
		return ExitFailure
	}
	var text strings.Builder
	fmt.Fprintf(&text, "%s\n\nUsage:\n  axiom", command.summary)
	if command.operation != windowsPermissionsRestoreAction && command.name != "windows-permissions" {
		text.WriteString(" [--human|--json]")
	}
	if commandSessionSupported(command, path) {
		text.WriteString(" [--session <id>]")
	}
	if len(path) > 0 {
		fmt.Fprintf(&text, " %s", strings.Join(path, " "))
	}
	if len(command.children) > 0 {
		text.WriteString(" <command>")
	} else if command.operation == skillInspectAction {
		text.WriteString(" <skill-name>")
	} else {
		hasFlags := false
		commandFlagSet(command.operation).VisitAll(func(*flag.Flag) { hasFlags = true })
		if hasFlags {
			text.WriteString(" [options]")
		}
	}
	text.WriteString("\n")
	fmt.Fprintf(&text, "  %s\n", helpInvocation(path))
	if len(path) == 0 {
		text.WriteString("  axiom [--human|--json] <command>\n  axiom help\n")
	}
	if len(command.children) > 0 {
		text.WriteString("\nCommands:\n")
		for _, child := range command.children {
			fmt.Fprintf(&text, "  %-16s %s\n", child.name, child.summary)
		}
		text.WriteString("\nAppend --help or -h to any command above to discover its children or inputs.\n")
	} else {
		set := commandFlagSet(command.operation)
		rules := commandRequirements(command.operation)
		repeats := repeatableFlags(set)
		count := 0
		set.VisitAll(func(f *flag.Flag) {
			if count == 0 {
				text.WriteString("\nOptions (noninteractive requirements; guided input remains available):\n")
			}
			count++
			requirement := "optional"
			for _, rule := range rules {
				if rule.name == f.Name {
					if rule.required {
						requirement = "required"
					} else if rule.when != "" {
						requirement = "required when " + rule.when
					}
				}
			}
			if repeats[f.Name] {
				requirement += "; repeatable"
			}
			value, description := flag.UnquoteUsage(f)
			if description == "" {
				description = "Explicit " + f.Name + " input."
			}
			if booleanFlag(f) {
				fmt.Fprintf(&text, "  --%s (%s): %s\n    forms: --%s | --%s=<boolean>\n", f.Name, requirement, description, f.Name, f.Name)
			} else if command.operation == windowsPermissionsRestoreAction {
				fmt.Fprintf(&text, "  --%s %s (%s): %s\n    form: --%s %s\n", f.Name, value, requirement, description, f.Name, value)
			} else {
				fmt.Fprintf(&text, "  --%s %s (%s): %s\n    forms: --%s %s | --%s=%s\n", f.Name, value, requirement, description, f.Name, value, f.Name, value)
			}
		})
		if count == 0 && command.operation != skillInspectAction {
			if commandSessionSupported(command, path) {
				text.WriteString("\nAccepts no flags or arguments beyond global output/session options and help.\n")
			} else {
				text.WriteString("\nAccepts no flags or arguments beyond global output options and help.\n")
			}
		}
		if command.operation == skillInspectAction {
			text.WriteString("\nRequired argument: <skill-name> = axiom-project | axiom-work-item.\n")
		}
		fmt.Fprintf(&text, "\nExample:\n  %s\n", commandExample(command.operation, path))
	}
	text.WriteString("\nHelp: --help | -h (human text; takes precedence over --json/--human).\n")
	if command.operation != windowsPermissionsRestoreAction && command.name != "windows-permissions" {
		text.WriteString("Global output: leading --human | --json.\n")
	}
	if commandSessionSupported(command, path) {
		text.WriteString("Global session: --session <id> before application commands.\n")
	}
	if len(path) == 0 {
		text.WriteString("\nStable Runtime skills: $axiom-project and $axiom-work-item.\nMutation authority stays explicit through --authorize-external or --authorize-local.\n")
	}
	if _, err := io.WriteString(writer, text.String()); err != nil {
		return ExitFailure
	}
	return ExitSuccess
}

func commandExample(operation action, path []string) string {
	result := "axiom " + strings.Join(path, " ")
	if strings.HasPrefix(string(operation), "context_session-") {
		result = "axiom --session '<session-id>' " + strings.Join(path, " ")
	}
	if operation == windowsPermissionsRestoreAction {
		return result + " --backup '<absolute-json-path>' --approve '<backup-digest>'"
	}
	if operation == skillInspectAction {
		return result + " axiom-project"
	}
	// Examples use syntactic placeholders; they confer no mutation authority.
	seen := map[string]bool{}
	set := commandFlagSet(operation)
	for _, rule := range commandRequirements(operation) {
		if operation == validateAction && rule.name == "slug" {
			continue
		}
		if !rule.required && rule.when == "" || seen[rule.name] {
			continue
		}
		f := set.Lookup(rule.name)
		if f == nil || rule.name == "authorize-local" || rule.name == "preview-digest" || rule.name == "project-id" || rule.name == "runtime-preview" {
			continue
		}
		if rule.name == "number" {
			result += " --work-item 'github:owner/repository#1'"
			seen[rule.name] = true
			continue
		}
		if rule.name == "provider-repository" && operation != workItemCreateAction {
			continue
		}
		if booleanFlag(f) {
			result += " --" + rule.name
		} else {
			value, _ := flag.UnquoteUsage(f)
			if value == "string" {
				value = "<" + rule.name + ">"
			}
			if value == "uint" || value == "uint64" || value == "int" {
				value = "1"
			}
			result += " --" + rule.name + " '" + value + "'"
		}
		seen[rule.name] = true
	}
	if strings.HasPrefix(string(operation), "context_") && operation != "context_show" || operation == workflowFactAction {
		result += " --authorize-local"
	}
	if operation == recoveryApplyAction {
		result += " --preview-digest '<reviewed-digest>' --authorize-local"
	}
	return result
}

func commandSessionSupported(command *commandDefinition, path []string) bool {
	return command.operation != "version" && command.operation != "help" && command.operation != windowsPermissionsRestoreAction && command.name != "windows-permissions" && (len(path) == 0 || path[0] != "skill")
}
