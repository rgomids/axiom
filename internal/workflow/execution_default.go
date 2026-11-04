package workflow

// Minimal default Workflow (Spec 015 normal operation 3): one work stage, one
// agent, one declared completion output, no human gate and no SDD phases. Its
// agent carries no Runtime/model/effort, so invocation comes from explicit
// run overrides or local defaults; no Profile is required. Its one input is
// the Execution's own linked Work Item (Provider, ID and Repository key), so
// the agent knows which Work Item it serves without any Provider fetch. Full
// SDD remains an explicitly selected template.
const (
	DefaultWorkflowID = "minimal"
	// r2 added the Work Item input; r1 snapshots stay as recorded.
	DefaultWorkflowRevision = "r2"
	DefaultStageID          = "work"
	DefaultAgentID          = "agent"
	DefaultOutput           = "result"
	DefaultWorkItemInput    = "work-item"
	DefaultStageTimeout     = 3600
	defaultInstructions     = "Complete the linked Work Item in the bound repository and report the result."
)

func DefaultWorkflowStages() []StageSpec {
	return []StageSpec{{
		ID:             DefaultStageID,
		WorkItemInputs: []string{DefaultWorkItemInput},
		Agents:         []AgentSpec{{ID: DefaultAgentID, Instructions: defaultInstructions, Inputs: []string{DefaultWorkItemInput}, Effects: []string{"workspace_write"}}},
		Outputs:        []string{DefaultOutput},
		TimeoutSeconds: DefaultStageTimeout,
	}}
}
