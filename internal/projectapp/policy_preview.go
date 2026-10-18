package projectapp

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"github.com/rgomids/axiom/internal/project"
	"github.com/rgomids/axiom/internal/workflowdefinition"
)

type PolicyEditDependencies struct {
	Installation string
	Source       string
	Workflow     string
	Operational  string
	Runtime      string
	Readiness    string
}

type PolicyEditFacts struct {
	Dependencies  PolicyEditDependencies
	Catalog       workflowdefinition.Catalog
	Operational   OperationalObservation
	Runtime       []PolicyRuntimeReadiness
	Repositories  map[string]bool
	Documentation map[string]DocumentationStatus
}

type PolicyEditObserver interface {
	ObservePolicyEdit(context.Context, EditSelection, project.Project) (PolicyEditFacts, EditFailure)
}

type PolicyEditPorts struct {
	Edit  EditPorts
	Facts PolicyEditObserver
}

type PolicyRuntimeReadiness struct {
	Stage   string `json:"stage,omitempty"`
	Agent   string `json:"agent,omitempty"`
	Code    string `json:"code"`
	Blocker string `json:"blocker,omitempty"`
	Next    string `json:"next"`
}

type PolicyEditPreview struct {
	Operation        string                   `json:"operation"`
	Scope            string                   `json:"scope"`
	ProjectID        string                   `json:"projectId"`
	SchemaVersion    int                      `json:"schemaVersion"`
	PortableRevision string                   `json:"portableRevision"`
	LocalRevision    string                   `json:"localRevision"`
	Policy           PolicyView               `json:"policy"`
	Diff             []PolicyChange           `json:"diff"`
	Workflow         PolicyWorkflowReport     `json:"workflow"`
	Admission        AdmissionDecision        `json:"admission"`
	Runtime          []PolicyRuntimeReadiness `json:"runtime"`
	Readiness        []OperationReadiness     `json:"readiness"`
	ReadinessBasis   string                   `json:"readinessBasis"`
	Effects          []EditEffect             `json:"effects"`
	Digest           string                   `json:"digest"`
}

type PolicyEditProposal struct {
	edit         EditProposal
	preview      PolicyEditPreview
	dependencies PolicyEditDependencies
}

func (p PolicyEditProposal) Preview() PolicyEditPreview {
	wire, _ := json.Marshal(p.preview)
	var out PolicyEditPreview
	_ = json.Unmarshal(wire, &out)
	return out
}

type selectedPolicyEdit struct{ selection EditSelection }

func (s selectedPolicyEdit) SelectForEdit(context.Context, string) (EditSelection, EditFailure) {
	return s.selection, EditOK
}

func PreviewPolicyEdit(ctx context.Context, ports PolicyEditPorts, intent EditIntent) (PolicyEditProposal, EditFailure) {
	if intent.Policy == nil || ValidateEditIntent(intent) != EditOK {
		return PolicyEditProposal{}, EditInvalidIntent
	}
	if ports.Edit.Source == nil || ports.Facts == nil {
		return PolicyEditProposal{}, EditUnavailable
	}
	if ctx.Err() != nil {
		return PolicyEditProposal{}, EditCancelled
	}
	selection, failure := ports.Edit.Source.SelectForEdit(ctx, intent.Selector)
	if failure != EditOK {
		return PolicyEditProposal{}, failure
	}
	editPorts := ports.Edit
	editPorts.Source = selectedPolicyEdit{selection}
	proposal, failure := PreviewEdit(ctx, editPorts, intent)
	if failure != EditOK {
		return PolicyEditProposal{}, failure
	}
	facts, failure := ports.Facts.ObservePolicyEdit(ctx, selection, proposal.project)
	if failure != EditOK {
		return PolicyEditProposal{}, failure
	}
	if !validPolicyDependencies(facts.Dependencies) || !ValidOperationalObservation(facts.Operational) {
		return PolicyEditProposal{}, EditStateUnsafe
	}
	state := proposal.project.State()
	workflow := CheckPolicyWorkflow(selection.Portable.Project(), proposal.project, facts.Catalog, "")
	if workflow.Workflow.Status == "unresolvable" || workflow.Reason == "workflow_impact_limit" {
		return PolicyEditProposal{}, EditStateUnsafe
	}
	if !validPolicyRuntimeReadiness(facts.Runtime, workflow) {
		return PolicyEditProposal{}, EditStateUnsafe
	}
	admission := AdmissionDecision{Operation: AdmitProjectEdit, Class: AdmissionAdministrative, Allowed: true, ProjectID: state.ID}
	input := ReadinessInput{Project: proposal.project, Local: proposal.local, Repositories: map[string]bool{}, RuntimeCode: PolicyRuntimeCode(facts.Runtime), Catalog: SupportedProviders}
	input.Repositories = facts.Repositories
	input.Documentation = facts.Documentation
	for i := range facts.Runtime {
		facts.Runtime[i].Next = PolicyRuntimeGuidance(facts.Runtime[i].Code)
		facts.Runtime[i].Blocker = BlockerRuntimeResolution
		if facts.Runtime[i].Code == "policy_unconfigured" {
			facts.Runtime[i].Blocker = BlockerRuntimePolicyUnavailable
		}
	}
	readiness := EvaluateReadiness(input)
	preview := PolicyEditPreview{
		Operation: string(AdmitProjectEdit), Scope: "runtime_policy", ProjectID: state.ID, SchemaVersion: state.SchemaVersion,
		PortableRevision: selection.PortableRevision, LocalRevision: selection.LocalRevision,
		Policy: ProjectPolicy(proposal.project), Diff: PolicyDiff(selection.Portable.Project(), proposal.project),
		Workflow: workflow, Admission: admission, Runtime: facts.Runtime, Readiness: readiness.Operations, ReadinessBasis: "retained_installation_metadata_no_new_proof",
		Effects: append([]EditEffect{}, proposal.preview.Effects...),
	}
	sealed, err := json.Marshal(struct {
		Preview      PolicyEditPreview
		EditDigest   string
		Dependencies PolicyEditDependencies
	}{preview, proposal.preview.Digest, facts.Dependencies})
	if err != nil || len(sealed) > MaxPolicyDraftBytes {
		return PolicyEditProposal{}, EditStateUnsafe
	}
	digest := sha256.Sum256(sealed)
	preview.Digest = hex.EncodeToString(digest[:])
	return PolicyEditProposal{edit: proposal, preview: preview, dependencies: facts.Dependencies}, EditOK
}

func validPolicyRuntimeReadiness(rows []PolicyRuntimeReadiness, workflow PolicyWorkflowReport) bool {
	if len(rows) == 1 && rows[0].Stage == "" && rows[0].Agent == "" {
		return policyRuntimeCode(rows[0].Code)
	}
	if len(rows) != len(workflow.Impact) || len(rows) == 0 {
		return false
	}
	for i, row := range rows {
		if !policyRuntimeCode(row.Code) || row.Stage != workflow.Impact[i].Stage || row.Agent != workflow.Impact[i].Agent {
			return false
		}
	}
	return true
}

func policyRuntimeCode(code string) bool {
	switch code {
	case "configuration_missing", "configuration_retired", "unsupported_configuration_format", "policy_unconfigured", "no_allowed_match", "runtime_unavailable", "inventory_unavailable", "ambiguous_match":
		return true
	}
	return false
}

func validPolicyDependencies(d PolicyEditDependencies) bool {
	for _, token := range []string{d.Installation, d.Source, d.Workflow, d.Operational, d.Runtime, d.Readiness} {
		wire, err := hex.DecodeString(token)
		if err != nil || len(wire) != sha256.Size || hex.EncodeToString(wire) != token {
			return false
		}
	}
	return true
}

func PolicyRuntimeCode(rows []PolicyRuntimeReadiness) string {
	for _, row := range rows {
		if row.Code != "inventory_unavailable" {
			return row.Code
		}
	}
	return "inventory_unavailable"
}

func PolicyRuntimeGuidance(code string) string {
	switch code {
	case "policy_unconfigured":
		return "Use axiom-project to declare portable Runtime and Model Profile policy, then run project validate"
	case "configuration_missing", "configuration_retired", "no_allowed_match", "runtime_unavailable", "unsupported_configuration_format":
		return "Use axiom-runtime to inspect local Runtime and Model Profile configuration; preserve portable declarations and request a fresh preview"
	case "ambiguous_match":
		return "Use axiom-project to review exact Profile references and preferences; do not choose a fallback"
	default:
		return "Runtime executable, authentication and capability proofs are uninspected; inspect them through axiom-runtime before Execution"
	}
}
