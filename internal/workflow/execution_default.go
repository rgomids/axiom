package workflow

// Minimal default Workflow (Spec 015 normal operation 3): one work stage, one
// agent, one declared completion output, no human gate and no SDD phases. Its
// agent carries no Runtime/model/effort, so invocation comes from explicit
// run overrides or local defaults; no Profile is required. Full SDD remains
// an explicitly selected template.
const (
	DefaultWorkflowID       = "minimal"
	DefaultWorkflowRevision = "r1"
	DefaultStageID          = "work"
	DefaultAgentID          = "agent"
	DefaultOutput           = "result"
	DefaultStageTimeout     = 3600
	defaultInstructions     = "Complete the linked Work Item in the bound repository and report the result."
)

func DefaultWorkflowStages() []StageSpec {
	return []StageSpec{{
		ID:             DefaultStageID,
		Agents:         []AgentSpec{{ID: DefaultAgentID, Instructions: defaultInstructions, Effects: []string{"workspace_write"}}},
		Outputs:        []string{DefaultOutput},
		TimeoutSeconds: DefaultStageTimeout,
	}}
}
