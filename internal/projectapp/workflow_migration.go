package projectapp

import (
	"context"

	"github.com/rgomids/axiom/internal/project"
	"github.com/rgomids/axiom/internal/runtimeprofile"
	"github.com/rgomids/axiom/internal/workflowdefinition"
)

// WorkflowProfilePort reads canonical local intent without acquiring vendor proof.
// Publication adapters must recheck the configuration generation while committing.
type WorkflowProfilePort interface {
	ObserveWorkflowProfiles(context.Context, string) (runtimeprofile.Configuration, string, string)
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

func migrateWorkflow(source workflowdefinition.Document, revision int, mappings []WorkflowAgentProfileMapping, cfg runtimeprofile.Configuration, p project.State) (workflowdefinition.Document, []WorkflowEffortComparison, string) {
	if source.Definition.SchemaVersion != 1 || revision <= source.Definition.Revision || cfg.FormatVersion != 2 {
		return workflowdefinition.Document{}, nil, "workflow_migration_required"
	}
	if _, err := runtimeprofile.Encode(cfg); err != nil {
		return workflowdefinition.Document{}, nil, "profile_effort_unconfigured"
	}
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
	profiles, _ := p.ModelProfiles.Value()
	runtimes, _ := p.Runtimes.Value()
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
			allowed := false
			for _, r := range cfg.Runtimes {
				if r.ID == selected.RuntimeID && r.Enabled {
					for _, profileID := range r.AllowlistedProfileIDs {
						allowed = allowed || profileID == id
					}
				}
			}
			runtimeAllowed := false
			for _, r := range runtimes {
				runtimeAllowed = runtimeAllowed || r.ID == selected.RuntimeID
			}
			projectAllowed := false
			for _, profile := range profiles {
				runtimeID, rok := profile.RuntimeRef.Value()
				model, mok := profile.Model.Value()
				projectAllowed = projectAllowed || profile.Key == id && rok && mok && profile.State.Form() != project.NotConfigured && runtimeID == selected.RuntimeID && model == selected.Model
			}
			if !allowed || !runtimeAllowed || !projectAllowed {
				return workflowdefinition.Document{}, nil, "project_policy_update_required"
			}
			if len(agent.RuntimeConstraints) > 0 {
				match := false
				for _, runtimeID := range agent.RuntimeConstraints {
					match = match || runtimeID == selected.RuntimeID
				}
				if !match {
					return workflowdefinition.Document{}, nil, "profile_runtime_mismatch"
				}
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
	doc, issues := workflowdefinition.Encode(d)
	if len(issues) > 0 {
		return workflowdefinition.Document{}, nil, "invalid_definition"
	}
	return doc, comparisons, ""
}
