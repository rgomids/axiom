package cli

import (
	"reflect"
	"testing"
)

func TestProjectWorkflowSelectionAndWorkItemMetadataPreserved303(t *testing.T) {
	specs := workflowAuthoringSkillSpecs(true)
	if len(specs) != 1 || specs[0].name != "workflow.select" || !reflect.DeepEqual(specs[0].actions, []action{action("project_workflow_select")}) {
		t.Fatalf("selection changed: %+v", specs)
	}
	mode := specs[0].modes[0]
	if mode.effect != effectLocalMutation || mode.selector != "--project" || mode.authority != "preview first; exact --expected-revision and --preview-digest plus --authorize-local" || !reflect.DeepEqual(mode.authorityInputs, []string{"--expected-revision", "--preview-digest", "--authorize-local"}) {
		t.Fatalf("selection authority changed: %+v", mode)
	}
	for _, tc := range []struct {
		spec    skillOperationSpec
		name    string
		actions []action
	}{{workItemRun, "run", []action{workflowStartAction, workflowAdvanceAction, workflowFactAction, workflowResumeAction, workflowReconcileAction}}, {workItemStatus, "status", []action{workflowStatusAction, workflowEvidenceAction, workflowListAction}}, {workItemPlan, "plan", []action{workflowStagePlanAction}}} {
		if tc.spec.name != tc.name || !reflect.DeepEqual(tc.spec.actions, tc.actions) {
			t.Fatalf("metadata changed: %+v", tc.spec)
		}
	}
}
