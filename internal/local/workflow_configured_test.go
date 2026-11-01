package local

import (
	"github.com/rgomids/axiom/internal/workflow"
	"github.com/rgomids/axiom/internal/workflowdefinition"
	"strings"
)

func validConfiguredExecution() workflow.State {
	state := validExecution()
	doc := workflowdefinition.Builtin()
	state.FormatVersion = 2
	state.WorkflowVersion = workflow.ConfiguredWorkflowVersion
	state.Binding = &workflow.WorkflowBinding{ProjectID: state.ProjectID, Definition: doc.Ref("builtin"), SchemaVersion: 1, SnapshotRef: "execution:" + state.ExecutionID + "#/binding/snapshot", Snapshot: doc.Canonical, ObservationDigest: strings.Repeat("a", 64), RuntimePreview: strings.Repeat("b", 64), BoundAt: state.CreatedAt, Contexts: map[string]workflow.ContextObservation{}}
	state.Binding.ProjectRevision, state.Binding.LocalRevision = strings.Repeat("c", 64), strings.Repeat("d", 64)
	return state
}
