package projectapp

import (
	"github.com/rgomids/axiom/internal/project"
	"github.com/rgomids/axiom/internal/runtimeprofile"
	"github.com/rgomids/axiom/internal/workflowdefinition"
)

// WorkflowProfileBindingV3 binds one schema-2 agent to the exact Profile
// intent it references, with the ProfileDigest and DigestV3 it was read from.
type WorkflowProfileBindingV3 struct {
	StageID string                         `json:"stageId"`
	AgentID string                         `json:"agentId"`
	Profile runtimeprofile.ProfileIntentV3 `json:"profile"`
}

// ValidateWorkflowProfilesV3 checks a schema-2 Workflow's Profile references
// against the consumed format-3 configuration and portable Project policy,
// with the categories migrateWorkflow uses (Specification 007 AXM-22
// BLD-001B.3c-2b.2). Effort is Profile-owned (ADR-0024): any agent effort is
// refused. Schema-1 records are never reinterpreted as Profile configuration.
// Workflow authoring applies it to every Profile-binding migration.
func ValidateWorkflowProfilesV3(d workflowdefinition.Definition, consumer runtimeprofile.ConsumerV3, p project.State) ([]WorkflowProfileBindingV3, string) {
	if d.SchemaVersion != 2 {
		return nil, "workflow_migration_required"
	}
	if consumer.Digest() == "" {
		return nil, "profile_effort_unconfigured"
	}
	owners := map[string]string{}
	for _, profile := range consumer.Configuration().ModelProfiles {
		owners[profile.ID] = profile.RuntimeID
	}
	profiles, _ := p.ModelProfiles.Value()
	runtimes, _ := p.Runtimes.Value()
	bindings := []WorkflowProfileBindingV3{}
	for _, stage := range d.Stages {
		for _, agent := range stage.Agents {
			if agent.Effort != (workflowdefinition.Effort{}) {
				return nil, "forbidden_effort"
			}
			id := agent.ProfileRef
			if !runtimeprofile.SafeIdentifier(id) {
				return nil, "invalid_profile_mapping"
			}
			owner, ok := owners[id]
			if !ok {
				return nil, "profile_effort_unconfigured"
			}
			intent, err := consumer.ProfileIntent(owner, id)
			if err != nil {
				// Disabled, absent or non-allowlisted: never another Profile.
				return nil, "project_policy_update_required"
			}
			runtimeAllowed := false
			for _, r := range runtimes {
				runtimeAllowed = runtimeAllowed || r.ID == intent.RuntimeID
			}
			projectAllowed := false
			for _, profile := range profiles {
				runtimeID, rok := profile.RuntimeRef.Value()
				model, mok := profile.Model.Value()
				projectAllowed = projectAllowed || profile.Key == id && rok && mok && profile.State.Form() != project.NotConfigured && runtimeID == intent.RuntimeID && model == intent.Model
			}
			if !runtimeAllowed || !projectAllowed {
				return nil, "project_policy_update_required"
			}
			if len(agent.RuntimeConstraints) > 0 {
				match := false
				for _, runtimeID := range agent.RuntimeConstraints {
					match = match || runtimeID == intent.RuntimeID
				}
				if !match {
					return nil, "profile_runtime_mismatch"
				}
			}
			bindings = append(bindings, WorkflowProfileBindingV3{StageID: stage.ID, AgentID: agent.ID, Profile: intent})
		}
	}
	return bindings, ""
}
