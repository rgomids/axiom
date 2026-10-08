package cli

import (
	"flag"
	"slices"
)

// inputRequirement is both discovery metadata and executable presence validation.
// Value/domain validation and conflict checks stay at their existing boundaries.
type inputRequirement struct {
	name     string
	required bool
	when     string
	applies  bool
	missing  bool
}

func skillRequirements(operation action, v requestInput) []inputRequirement {
	if operation == showAction || operation == resolveAction {
		return []inputRequirement{{name: "selector", required: true, missing: v.selector == ""}}
	}
	if operation == configureAction {
		return []inputRequirement{
			{"slug", false, "creating (no --project)", !v.projectSupplied, v.slug == ""},
			{"name", false, "creating (no --project)", !v.projectSupplied, v.name == ""},
			{"repository", false, "creating (no --project)", !v.projectSupplied, len(v.repositories) == 0},
		}
	}
	switch {
	case operation == projectArchiveAction || operation == projectReactivateAction || operation == workflowListAction || operation == integrationListAction || operation == integrationValidateAction:
		return []inputRequirement{{name: "project", required: true, missing: v.project == ""}}
	case integrationOperation(operation):
		return []inputRequirement{{name: "project", required: true, missing: v.project == ""}, {name: "integration", required: true, missing: true}}
	}
	if !knownWorkItem(operation) && !knownWorkflow(operation) {
		return nil
	}
	if operation == workItemListAction {
		return []inputRequirement{{name: "project", required: true, missing: v.project == ""}}
	}
	rules := []inputRequirement{
		{name: "project", required: true, missing: v.project == ""},
		{name: "repository", required: true, missing: v.repository == ""},
	}
	if operation == workItemCreateAction {
		return append(rules, inputRequirement{name: "provider-repository", required: true, missing: v.providerRepository == ""})
	}
	rules = append(rules, inputRequirement{"number", false, "--work-item is absent (legacy alternative)", v.workItem == "", v.number <= 0})
	if operation == workItemSelectAction {
		rules = append(rules, inputRequirement{"provider-repository", false, "--work-item is absent", v.workItem == "", v.providerRepository == ""})
	}
	if operation == workItemCommentAction {
		rules = append(rules, inputRequirement{name: "message", required: true, missing: v.message == ""})
	}
	if operation == workflowStartAction {
		rules = append(rules,
			inputRequirement{name: "role", required: true, missing: v.role == ""},
			inputRequirement{name: "complexity", required: true, missing: v.complexity == ""},
			inputRequirement{name: "capabilities", required: true, missing: v.capabilities == ""},
			inputRequirement{name: "runtime-preview", when: "creating Execution after review", missing: v.runtimePreview == ""},
		)
	}
	if knownWorkflow(operation) && operation != workflowStartAction {
		rules = append(rules, inputRequirement{"execution", false, "using --work-item", v.workItem != "", v.execution == ""})
	}
	if operation == workflowAdvanceAction || operation == workflowFactAction || operation == workflowResumeAction || operation == workflowReconcileAction {
		rules = append(rules, inputRequirement{name: "expected-revision", required: true, missing: v.expectedRevision == 0})
	}
	if operation == workflowAdvanceAction {
		rules = append(rules, inputRequirement{name: "gate", required: true, missing: v.gate == ""}, inputRequirement{name: "outcome", required: true, missing: v.outcome == ""})
	}
	if operation == workflowFactAction {
		rules = append(rules, inputRequirement{name: "fact", required: true, missing: v.fact == ""}, inputRequirement{name: "reference", required: true, missing: v.reference == ""})
	}
	return rules
}

func missingRequiredInputs(operation action, values requestInput, names ...string) bool {
	for _, rule := range skillRequirements(operation, values) {
		if len(names) != 0 && !slices.Contains(names, rule.name) {
			continue
		}
		if (rule.required || rule.applies) && rule.missing {
			return true
		}
	}
	return false
}

func repeatableFlags(set *flag.FlagSet) map[string]bool {
	result := make(map[string]bool)
	set.VisitAll(func(f *flag.Flag) {
		switch f.Value.(type) {
		case *repositoryFlags, *elaboratedSectionFlags:
			result[f.Name] = true
		}
	})
	return result
}
