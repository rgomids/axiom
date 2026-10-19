package projectapp

import (
	"encoding/json"
	"sort"
	"strconv"

	"github.com/rgomids/axiom/internal/project"
	"github.com/rgomids/axiom/internal/workflowdefinition"
)

type PolicyProfileReference struct {
	Profile string `json:"profile"`
	Runtime string `json:"runtime,omitempty"`
}

type PolicyAgentCompatibility struct {
	Compatible bool                     `json:"compatible"`
	Reason     string                   `json:"reason"`
	Profiles   []PolicyProfileReference `json:"profiles"`
}

type PolicyWorkflowImpact struct {
	Stage              string                   `json:"stage"`
	Agent              string                   `json:"agent"`
	Role               string                   `json:"role"`
	Complexity         string                   `json:"complexity"`
	Capabilities       []string                 `json:"capabilities"`
	RuntimeConstraints []string                 `json:"runtimeConstraints"`
	Before             PolicyAgentCompatibility `json:"before"`
	After              PolicyAgentCompatibility `json:"after"`
	Change             string                   `json:"change"`
}

type PolicyWorkflowReport struct {
	Workflow      ActiveWorkflow         `json:"workflow"`
	SchemaVersion int                    `json:"schemaVersion,omitempty"`
	Compatible    bool                   `json:"compatible"`
	Reason        string                 `json:"reason,omitempty"`
	Impact        []PolicyWorkflowImpact `json:"impact"`
	Inventory     string                 `json:"inventory"`
	Scopes        []string               `json:"scopes"`
}

func CheckPolicyWorkflow(before, after project.Project, catalog workflowdefinition.Catalog, category string) PolicyWorkflowReport {
	beforeState, afterState := before.State(), after.State()
	if category != "" {
		category = "workflow_unresolvable"
	}
	out := PolicyWorkflowReport{Workflow: ResolveActiveWorkflow(beforeState, catalog, category), Impact: []PolicyWorkflowImpact{}, Inventory: "unknown_uninspected", Scopes: []string{
		"future_admission_requires_current_policy_workflow_readiness_authority",
		"existing_execution_stage_plan_bindings_immutable_resume_dispatch_recheck_permission_freshness",
		"work_items_no_effects",
		"other_projects_local_configuration_unchanged",
	}}
	out.Workflow.Name = ""
	if !before.Equivalent(before) || !after.Equivalent(after) || beforeState.WorkflowSelection != afterState.WorkflowSelection {
		out.Reason = "workflow_selection_changed"
		return out
	}
	switch out.Workflow.Status {
	case "none":
		out.Compatible = true
		return out
	case "selected":
	default:
		out.Reason = "workflow_unresolvable"
		return out
	}
	selection, _ := beforeState.WorkflowSelection.Value()
	doc, ok := workflowdefinition.BuiltinRevision(selection.Revision)
	if selection.Source == "project" {
		doc, ok = catalog.Documents[selection.WorkflowID+"/"+strconv.Itoa(selection.Revision)+".json"]
	}
	if !ok {
		out.Reason = "workflow_unresolvable"
		return out
	}
	out.SchemaVersion = doc.Definition.SchemaVersion
	out.Compatible = true
	impactBytes := 0
	for _, stage := range doc.Definition.Stages {
		for _, agent := range stage.Agents {
			previous := portableAgentCompatibility(beforeState, doc.Definition.SchemaVersion, agent)
			next := portableAgentCompatibility(afterState, doc.Definition.SchemaVersion, agent)
			change := "compatible"
			if !next.Compatible {
				out.Compatible = false
				out.Reason = "workflow_policy_incompatible"
				change = "introduced_incompatibility"
				if !previous.Compatible {
					change = "unresolved_incompatibility"
				}
			} else if !previous.Compatible {
				change = "resolved_incompatibility"
			}
			row := PolicyWorkflowImpact{Stage: stage.ID, Agent: agent.ID, Role: agent.Role, Complexity: agent.Complexity, Capabilities: append([]string{}, agent.Capabilities...), RuntimeConstraints: append([]string{}, agent.RuntimeConstraints...), Before: previous, After: next, Change: change}
			rowWire, err := json.Marshal(row)
			metadata := out
			metadata.Impact = []PolicyWorkflowImpact{}
			header, headerErr := json.Marshal(metadata)
			if len(out.Impact) > 0 {
				impactBytes++
			}
			impactBytes += len(rowWire)
			if err != nil || headerErr != nil || len(header)+impactBytes > MaxPolicyDraftBytes {
				out.Compatible = false
				out.Reason = "workflow_impact_limit"
				out.Impact = []PolicyWorkflowImpact{}
				return out
			}
			out.Impact = append(out.Impact, row)
		}
	}
	sort.Slice(out.Impact, func(i, j int) bool {
		a, b := out.Impact[i], out.Impact[j]
		if a.Stage != b.Stage {
			return a.Stage < b.Stage
		}
		return a.Agent < b.Agent
	})

	return out
}

func portableAgentCompatibility(state project.State, schema int, agent workflowdefinition.Agent) PolicyAgentCompatibility {
	out := PolicyAgentCompatibility{Profiles: []PolicyProfileReference{}, Reason: "profile_missing"}
	target := agent.ProfileRef
	if schema == 1 && target == "policy-default" {
		target = ""
		prefs, _ := state.RuntimePreferences.Value()
		for _, p := range prefs {
			if p.Role == agent.Role && p.Complexity == agent.Complexity {
				target = p.ModelProfileRef
				break
			}
		}
	}
	profiles, _ := state.ModelProfiles.Value()
	rejected := []PolicyProfileReference{}
	for _, p := range profiles {
		if target != "" && p.Key != target {
			continue
		}
		runtime, _, reason := portableProfileAssociation(state, p.Key)
		reference := PolicyProfileReference{Profile: p.Key, Runtime: runtime}
		if reason != "" {
			if target != "" {
				out.Reason = reason
			}
			rejected = append(rejected, reference)
			continue
		}
		if !policyRuntimePermitted(agent, runtime) {
			if target != "" {
				out.Reason = "profile_runtime_mismatch"
			}
			rejected = append(rejected, reference)
			continue
		}
		out.Profiles = append(out.Profiles, reference)
	}
	if len(out.Profiles) > 0 {
		out.Compatible = true
		out.Reason = "portable_compatible_local_readiness_required"
	} else if target == "" {
		out.Reason = "portable_candidates_empty"
	}
	if !out.Compatible {
		out.Profiles = rejected
		if len(out.Profiles) == 0 && target != "" {
			out.Profiles = append(out.Profiles, PolicyProfileReference{Profile: target})
		}
	}
	sort.Slice(out.Profiles, func(i, j int) bool { return out.Profiles[i].Profile < out.Profiles[j].Profile })
	return out
}

func portableProfileAssociation(state project.State, key string) (string, string, string) {
	profiles, _ := state.ModelProfiles.Value()
	for _, p := range profiles {
		if p.Key != key {
			continue
		}
		runtime, runtimeConfigured := p.RuntimeRef.Value()
		model, modelConfigured := p.Model.Value()
		if p.State.Form() != project.Absent || !runtimeConfigured || !modelConfigured {
			return runtime, model, "profile_unconfigured"
		}
		for _, allowed := range project.AllowedRuntimes(state) {
			if allowed == runtime {
				return runtime, model, ""
			}
		}
		return runtime, model, "runtime_disallowed"
	}
	return "", "", "profile_missing"
}

func policyRuntimePermitted(agent workflowdefinition.Agent, runtime string) bool {
	if len(agent.RuntimeConstraints) == 0 {
		return true
	}
	for _, constraint := range agent.RuntimeConstraints {
		if constraint == runtime {
			return true
		}
	}
	return false
}
