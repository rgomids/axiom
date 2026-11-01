package main

import (
	"encoding/json"
	"github.com/rgomids/axiom/internal/workflowdefinition"
	"os/exec"
	"strconv"
	"testing"
)

func selectBuiltinWorkflowExecutable(t *testing.T, binary string, environment []string, project string) {
	t.Helper()
	ref := workflowdefinition.Builtin().Ref("builtin")
	args := []string{"--json", "project", "workflow", "select", "--project", project, "--workflow", ref.WorkflowID, "--revision", strconv.Itoa(ref.Revision), "--digest", ref.Digest, "--source", ref.Source}
	run := func(category string, args []string) authoringEvent {
		t.Helper()
		command := exec.Command(binary, args...)
		command.Env = environment
		wire, err := command.CombinedOutput()
		var event authoringEvent
		if err != nil || json.Unmarshal(wire, &event) != nil || event.Category != category {
			t.Fatalf("workflow selection: %v %s", err, wire)
		}
		return event
	}
	preview := run("previewed", args)
	run("applied", append(args, "--expected-revision", preview.Workflow.ProjectRevision, "--preview-digest", preview.Workflow.PreviewDigest, "--authorize-local"))
}
