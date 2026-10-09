package cli

import (
	"slices"
	"testing"
)

func TestCommandRequirementsMatchMaintenanceParserPresence(t *testing.T) {
	tests := []struct {
		operation action
		args      []string
	}{
		{compatibilityBackupAction, []string{"--target", "/backup"}},
		{compatibilityExportAction, []string{"--target", "/export"}},
		{artifactRetireAction, []string{"--artifact", "artifact-id"}},
		{upgradeAction, []string{"--archive", "/archive", "--checksums", "/checksums", "--bin-dir", "/bin", "--receipt-dir", "/receipts"}},
		{recoveryApplyAction, []string{"--preview-digest", "reviewed-digest", "--authorize-local"}},
	}
	for _, test := range tests {
		t.Run(string(test.operation), func(t *testing.T) {
			if _, issue := maintenanceFlags(test.operation, test.args); issue != "" {
				t.Fatalf("complete presence tuple rejected: %s", issue)
			}
			set := maintenanceFlagSet(test.operation, &MaintenanceInput{})
			for _, requirement := range commandRequirements(test.operation) {
				if !requirement.required {
					continue
				}
				index := slices.Index(test.args, "--"+requirement.name)
				if index < 0 {
					t.Fatalf("required flag %s absent from valid tuple", requirement.name)
				}
				end := index + 2
				if boolean, ok := set.Lookup(requirement.name).Value.(interface{ IsBoolFlag() bool }); ok && boolean.IsBoolFlag() {
					end = index + 1
				}
				without := append(slices.Clone(test.args[:index]), test.args[end:]...)
				if _, issue := maintenanceFlags(test.operation, without); issue != "missing_required_input" {
					t.Errorf("missing %s: got %q, want missing_required_input", requirement.name, issue)
				}
			}
		})
	}
}

func TestCommandRequirementsPreserveSkillRequirements(t *testing.T) {
	for _, operation := range []action{showAction, resolveAction, workItemCreateAction, workItemSelectAction, workItemCommentAction, workflowStartAction, workflowAdvanceAction, workflowFactAction, integrationShowAction, projectArchiveAction} {
		if !slices.Equal(commandRequirements(operation), skillRequirements(operation, requestInput{})) {
			t.Errorf("%s changed shared skill requirements", operation)
		}
	}
	// Configure adds help-only replay conditions but retains the shared prefix.
	shared := skillRequirements(configureAction, requestInput{})
	if !slices.Equal(commandRequirements(configureAction)[:len(shared)], shared) {
		t.Fatal("configure changed shared skill requirements")
	}
}
