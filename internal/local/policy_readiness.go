package local

import "github.com/rgomids/axiom/internal/projectapp"

// Policy edits disclose the same non-predictive basis as project validate.
// Explicit Runtime/model/effort restrictions are resolved on the attempt.
func policyRuntimeReadiness(workflow projectapp.PolicyWorkflowReport) []projectapp.PolicyRuntimeReadiness {
	rows := []projectapp.PolicyRuntimeReadiness{}
	for _, impact := range workflow.Impact {
		rows = append(rows, projectapp.PolicyRuntimeReadiness{Stage: impact.Stage, Agent: impact.Agent, Code: projectapp.RuntimeSelectedAtDispatch})
	}
	if len(rows) == 0 {
		rows = append(rows, projectapp.PolicyRuntimeReadiness{Code: projectapp.RuntimeSelectedAtDispatch})
	}
	return rows
}
