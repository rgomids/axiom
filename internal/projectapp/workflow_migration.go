package projectapp

import (
	"context"

	"github.com/rgomids/axiom/internal/project"
	"github.com/rgomids/axiom/internal/runtimeprofile"
	"github.com/rgomids/axiom/internal/workflowdefinition"
)

// WorkflowProfilePort reads canonical v3 local intent without acquiring vendor
// proof. Publication adapters must recheck the configuration generation while
// committing.
type WorkflowProfilePort interface {
	ObserveWorkflowProfiles(context.Context, string) (runtimeprofile.ConsumerV4, string, string)
}
type WorkflowAgentProfileMapping struct {
	StageID   string `json:"stageId"`
	AgentID   string `json:"agentId"`
	ProfileID string `json:"profileId"`
}
type WorkflowEffortComparison struct {
	StageID               string                    `json:"stageId"`
	AgentID               string                    `json:"agentId"`
	LegacyRequestedEffort workflowdefinition.Effort `json:"legacyRequestedEffort"`
	ProfileID             string                    `json:"profileId"`
	ProfileEffort         runtimeprofile.Effort     `json:"profileEffort"`
	ChangedBehavior       bool                      `json:"changedBehavior"`
}

// migrateWorkflow maps schema-1 agents to explicit Profiles of the canonical v3
// configuration; Profile references, allowlists, Project policy and Runtime
// constraints are then checked by ValidateWorkflowProfilesV3. Effort stays
// Profile-owned (ADR-0024).
func migrateWorkflow(source workflowdefinition.Document, revision int, mappings []WorkflowAgentProfileMapping, consumer runtimeprofile.ConsumerV4, p project.State) (workflowdefinition.Document, []WorkflowEffortComparison, string) {
	if source.Definition.SchemaVersion != 1 || revision <= source.Definition.Revision {
		return workflowdefinition.Document{}, nil, "workflow_migration_required"
	}
	if consumer.Digest() == "" {
		return workflowdefinition.Document{}, nil, "profile_effort_unconfigured"
	}
	cfg := consumer.Configuration()
	byAgent := map[string]string{}
	for _, mapping := range mappings {
		key := mapping.StageID + "\x00" + mapping.AgentID
		if !workflowdefinition.ValidKey(mapping.StageID) || !workflowdefinition.ValidKey(mapping.AgentID) || !runtimeprofile.SafeIdentifier(mapping.ProfileID) || byAgent[key] != "" {
			return workflowdefinition.Document{}, nil, "invalid_profile_mapping"
		}
		byAgent[key] = mapping.ProfileID
	}
	// Decode a fresh copy: never mutate the source document's shared stage slices.
	copied, issues := workflowdefinition.Decode(source.Canonical)
	if len(issues) != 0 {
		return workflowdefinition.Document{}, nil, "invalid_definition"
	}
	d := copied.Definition
	comparisons := []WorkflowEffortComparison{}
	for i := range d.Stages {
		stage := &d.Stages[i]
		for j := range stage.Agents {
			agent := &stage.Agents[j]
			key := stage.ID + "\x00" + agent.ID
			id := byAgent[key]
			if id == "" {
				return workflowdefinition.Document{}, nil, "invalid_profile_mapping"
			}
			delete(byAgent, key)
			var selected *runtimeprofile.ModelProfile
			for index := range cfg.ModelProfiles {
				if cfg.ModelProfiles[index].ID == id {
					selected = &cfg.ModelProfiles[index]
				}
			}
			if selected == nil || !runtimeprofile.ValidEffort(selected.Effort) {
				return workflowdefinition.Document{}, nil, "profile_effort_unconfigured"
			}
			comparisons = append(comparisons, WorkflowEffortComparison{StageID: stage.ID, AgentID: agent.ID, LegacyRequestedEffort: agent.Effort, ProfileID: id, ProfileEffort: selected.Effort, ChangedBehavior: agent.Effort.Mode != selected.Effort.Mode || agent.Effort.Value != selected.Effort.Value})
			agent.ProfileRef = id
			agent.Effort = workflowdefinition.Effort{}
		}
	}
	if len(byAgent) != 0 {
		return workflowdefinition.Document{}, nil, "invalid_profile_mapping"
	}
	d.SchemaVersion = 2
	d.Revision = revision
	if _, category := ValidateWorkflowProfilesV3(d, consumer, p); category != "" {
		return workflowdefinition.Document{}, nil, category
	}
	doc, issues := workflowdefinition.Encode(d)
	if len(issues) > 0 {
		return workflowdefinition.Document{}, nil, "invalid_definition"
	}
	return doc, comparisons, ""
}
