package local

import (
	"github.com/rgomids/axiom/internal/project"
	"github.com/rgomids/axiom/internal/projectapp"
	"github.com/rgomids/axiom/internal/runtimeapplication"
)

func policyRuntimeReadiness(p project.Project, workflow projectapp.PolicyWorkflowReport, facts RuntimeProfileSnapshot) []projectapp.PolicyRuntimeReadiness {
	consumer, code := runtimeapplication.ConsumerFromRecordV3(facts.State, facts.V3)
	if code != "" {
		return []projectapp.PolicyRuntimeReadiness{{Code: code}}
	}
	if !project.RuntimePolicyDeclared(p.State()) {
		return []projectapp.PolicyRuntimeReadiness{{Code: "policy_unconfigured"}}
	}
	rows := []projectapp.PolicyRuntimeReadiness{}
	for _, impact := range workflow.Impact {
		matches := 0
		code = "no_allowed_match"
		for _, profile := range impact.After.Profiles {
			request := runtimeapplication.Request{Role: impact.Role, Complexity: impact.Complexity, Capabilities: impact.Capabilities, ModelProfileID: profile.Profile, RuntimeID: profile.Runtime}
			_, blocker := runtimeapplication.EvaluatePolicyV3(p, consumer, request)
			if blocker == nil {
				matches++
			}
		}
		if matches == 1 {
			code = "inventory_unavailable"
		} else if matches > 1 {
			code = "ambiguous_match"
		}
		rows = append(rows, projectapp.PolicyRuntimeReadiness{Stage: impact.Stage, Agent: impact.Agent, Code: code})
	}
	if len(rows) == 0 {
		rows = append(rows, projectapp.PolicyRuntimeReadiness{Code: runtimeapplication.EvaluateAvailabilityV3(p, consumer)})
	}
	return rows
}
