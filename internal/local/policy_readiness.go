package local

import (
	"context"
	"github.com/rgomids/axiom/internal/project"
	"github.com/rgomids/axiom/internal/projectapp"
	"github.com/rgomids/axiom/internal/runtimeapplication"
)

func policyRuntimeReadiness(p project.Project, workflow projectapp.PolicyWorkflowReport, facts RuntimeProfileSnapshot) []projectapp.PolicyRuntimeReadiness {
	if !project.RuntimePolicyDeclared(p.State()) {
		return []projectapp.PolicyRuntimeReadiness{{Code: "policy_unconfigured"}}
	}
	consumer, code := runtimeapplication.ConsumerFromRecordV3(facts.State, facts.V3)
	if code != "" {
		return []projectapp.PolicyRuntimeReadiness{{Code: code}}
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

func ObservePolicyRuntime(ctx context.Context, subject projectapp.ReadinessSubject, stateRoot string) string {
	if ctx.Err() != nil {
		return "inventory_unavailable"
	}
	root, err := existingPrivateRoot(subject.Local.SourceLocation)
	if err != nil {
		return "inventory_unavailable"
	}
	defer root.Close()
	catalog, err := readWorkflowCatalog(root)
	if err != nil {
		return "inventory_unavailable"
	}
	store, err := NewRuntimeProfileStore(stateRoot)
	if err != nil {
		return "inventory_unavailable"
	}
	facts, err := store.Inspect(ctx)
	if err != nil {
		return "inventory_unavailable"
	}
	workflow := projectapp.CheckPolicyWorkflow(subject.Project, subject.Project, catalog, "")
	if !workflow.Compatible {
		return "no_allowed_match"
	}
	return projectapp.PolicyRuntimeCode(policyRuntimeReadiness(subject.Project, workflow, facts))
}
