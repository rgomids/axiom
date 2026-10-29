package cli

import (
	"flag"
	"strings"
)

// selectorInputDiagnosis names the first input a Work Item or workflow selector
// command rejected, as a stable `<code>` or `<code>:--<flag>` handle carried in
// the existing completion details. It explains a refusal the parser already
// decided; it never relaxes, reorders or replaces that decision. An input name
// is echoed only when it is a well-formed flag name.
func selectorInputDiagnosis(operation action, args []string, issue string) string {
	var values requestInput
	var set *flag.FlagSet
	var repeatable map[string]bool
	switch {
	case knownWorkItem(operation):
		set = workItemFlagSet(operation, &values)
		repeatable = repeatableFlags(set)
	case knownWorkflow(operation):
		set = workflowFlagSet(operation, &values)
	default:
		return ""
	}
	if code, name := flagSyntaxIssue(set, args, repeatable); code != "" {
		return diagnosisHandle(code, name)
	}
	if set.Parse(args) != nil || set.NArg() != 0 {
		return "invalid_value"
	}
	if values.workItem != "" && (values.providerRepository != "" || values.number != 0 || operation == workItemCreateAction) {
		return diagnosisHandle("conflicting_input", "work-item")
	}
	if values.workItem != "" {
		if _, _, _, ok := parseWorkItemSelector(values.workItem); !ok {
			return diagnosisHandle("invalid_value", "work-item")
		}
	}
	if issue == "missing_required_input" {
		for _, rule := range skillRequirements(operation, values) {
			if (rule.required || rule.applies) && rule.missing {
				return diagnosisHandle("missing_required_input", rule.name)
			}
		}
	}
	if issue == "" {
		return ""
	}
	return issue
}

// flagSyntaxIssue walks arguments exactly like invalidFlagSyntax and reports
// the first rejected one.
func flagSyntaxIssue(set *flag.FlagSet, args []string, repeatable map[string]bool) (string, string) {
	seen := make(map[string]bool)
	for index := 0; index < len(args); index++ {
		value := args[index]
		if value == "--" || !strings.HasPrefix(value, "--") {
			return "unsupported_syntax", ""
		}
		name := strings.TrimPrefix(value, "--")
		hasValue := false
		if separator := strings.IndexByte(name, '='); separator >= 0 {
			name = name[:separator]
			hasValue = true
		}
		current := set.Lookup(name)
		if current == nil {
			return "unknown_input", name
		}
		if !repeatable[name] && seen[name] {
			return "duplicate_input", name
		}
		seen[name] = true
		boolean, isBoolean := current.Value.(interface{ IsBoolFlag() bool })
		if hasValue || isBoolean && boolean.IsBoolFlag() {
			continue
		}
		if index+1 >= len(args) {
			return "missing_value", name
		}
		index++
	}
	return "", ""
}

func diagnosisHandle(code, name string) string {
	if !diagnosableFlagName(name) {
		return code
	}
	return code + ":--" + name
}

func diagnosableFlagName(name string) bool {
	if name == "" || len(name) > 48 || name[0] == '-' {
		return false
	}
	for _, character := range name {
		if character >= 'a' && character <= 'z' || character >= '0' && character <= '9' || character == '-' {
			continue
		}
		return false
	}
	return true
}
