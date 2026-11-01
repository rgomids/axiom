package local

import (
	"bytes"
	"github.com/rgomids/axiom/internal/provenance"
	"github.com/rgomids/axiom/internal/workflow"
	"time"
)

func validExecution() workflow.State {
	source, _ := provenance.FromBuild(provenance.Build{Version: provenance.Development, Revision: "abc123", SourceState: provenance.Clean}, nil)
	now := time.Date(2026, 9, 23, 12, 0, 0, 0, time.UTC)
	return workflow.State{ExecutionID: "018f4a44-7c31-7dd4-9d00-111111111111", FormatVersion: workflow.FormatVersion, WorkflowVersion: workflow.WorkflowVersion, ProjectID: "123e4567-e89b-42d3-a456-426614174000", RepositoryKey: "main", WorkItem: workflow.WorkItem{Provider: "github", Resource: "owner/repo", ExternalID: "7", URL: "https://github.com/owner/repo/issues/7", State: "OPEN"}, RuntimeID: "codex", Stage: workflow.Intake, Revision: 1, Status: workflow.ExecutionActive, CreatedAt: now, UpdatedAt: now, Provenance: workflow.Identity{Product: source.Product(), Version: source.Version(), Revision: source.Revision(), SourceState: string(source.SourceState())}}
}

func advancedExecution(state workflow.State) workflow.State {
	now := state.UpdatedAt.Add(time.Second)
	state.Revision, state.Stage, state.UpdatedAt = 2, workflow.Specification, now
	state.Transitions = []workflow.Transition{{Revision: 2, From: workflow.Intake, To: workflow.Specification, Outcome: workflow.OutcomePassed, RequestDigest: string(bytes.Repeat([]byte{'a'}, 64)), CommittedAt: now, Provenance: state.Provenance}}
	return state
}

func interruptedExecution(state workflow.State) workflow.State {
	now := state.UpdatedAt.Add(time.Second)
	state.Revision, state.Status, state.UpdatedAt = 2, workflow.ExecutionInterrupted, now
	state.Transitions = []workflow.Transition{{Revision: 2, From: workflow.Intake, To: workflow.Intake, Outcome: workflow.OutcomeFailed, RequestDigest: string(bytes.Repeat([]byte{'b'}, 64)), CommittedAt: now, Provenance: state.Provenance}}
	return state
}
