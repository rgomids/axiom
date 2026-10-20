package main

import (
	"encoding/hex"
	"strconv"

	"github.com/rgomids/axiom/internal/cli"
	"github.com/rgomids/axiom/internal/completion"
	"github.com/rgomids/axiom/internal/projectapp"
	"github.com/rgomids/axiom/internal/stageexecution"
)

// diagnosticBase uses only bounded, trusted mapper constants. Empty fields
// explicitly mean unobserved; absence is never evidence of no prior effects.
func diagnosticBase(operation, category, read string) *cli.Diagnostic {
	verification := cli.DiagnosticVerification{Operation: read, ExpectedFact: "Reobserve the exact target and blocker; a successful read is not workflow success"}
	return &cli.Diagnostic{
		SchemaVersion: 1, Operation: operation, Category: category,
		Target: map[string]string{}, ObservedRevision: "",
		Observed:    []string{"Operation returned the mapped category"},
		NotObserved: []string{"Missing identities and revisions were not established", "Prior effects are not established by this diagnostic"},
		Hypotheses:  []string{}, MissingPrerequisites: []string{"Exact canonical target and current authority"},
		Capability: "unknown", Authority: "not established by diagnostic", Confinement: "unknown",
		ConfirmedEffects: []string{}, UncertainEffects: []string{"Prior effects not established; inspect before replay"},
		Options: []cli.DiagnosticOption{
			{ID: "inspect", Availability: "conditional", Prerequisites: []string{"Resolve exact canonical selectors"}, Effects: []string{"read-only"}, RequiredAuthority: []string{}, Operation: read, Arguments: map[string]string{}, MissingArguments: []string{"canonical selectors"}, Purpose: "Investigate current state without repeating an effect", Verification: verification},
			{ID: "handoff", Availability: "conditional", Prerequisites: []string{"Authorized working Evidence destination"}, Effects: []string{"Sanitized working Evidence only"}, RequiredAuthority: []string{"Authorized working Evidence destination"}, Arguments: map[string]string{}, MissingArguments: []string{}, Purpose: "Request the missing fact or bounded human decision; no grant is created", Verification: verification},
		}, Verification: verification,
	}
}

func stageDiagnostic(category string, status completion.Status) *cli.Diagnostic {
	d := diagnosticBase("workflow stage", category, "workflow stage status")
	switch category {
	case "runtime_unresolvable":
		d.Hypotheses = []string{"Project Runtime policy, Profile intent or current executable/source observation may be unresolved; inspect the actual blocker"}
		d.Options = []cli.DiagnosticOption{d.Options[0], {ID: "inspect-profiles", Availability: "available", Prerequisites: []string{}, Effects: []string{"read-only"}, RequiredAuthority: []string{}, Operation: "runtime profile list", Arguments: map[string]string{}, MissingArguments: []string{}, Purpose: "Inspect configured Profile intent; scoped operational preview needs its own inspection consent", Verification: cli.DiagnosticVerification{Operation: "runtime profile list", ExpectedFact: "Observe configured Profiles without inferring authentication or compatibility"}}, d.Options[1]}
	case "stop_orphan_then_reconcile", "reconcile_lease_held":
		d.MissingPrerequisites = []string{"Recorded process identity, observed liveness and lease ownership", "Separate stop authority where a supported stop is available"}
		d.Options[1].Purpose = "Wait or hand off process inspection and a separately authorized stop; uncertain liveness never authorizes signaling"
	case "confirmed_effect_reconciliation_required":
		d.ConfirmedEffects = []string{"Requested effect confirmed by application; consult unchanged stage payload for its exact extent"}
		d.UncertainEffects = []string{"Secondary publication or projection incomplete; inspect before reconciliation"}
	case "recovery_required":
		d.MissingPrerequisites = []string{"Reobserve persisted authority, receipts and effects before recovery; never blindly redispatch"}
	case "stale_preview", "stale_execution_revision", "authority_required", "authority_denied", "supervisor_authority_required":
		d.MissingPrerequisites = []string{"Fresh exact preview and operation-specific authority"}
	case "retry_not_eligible", "retry_attempts_exhausted", "workspace_disposition_required":
		d.MissingPrerequisites = []string{"Own retry authority, available attempt capacity and workspace disposition", "Reconciliation fact for an unknown attempt; fact append does not reclassify status"}
	}
	if status == completion.Interrupted {
		d.UncertainEffects = []string{"Interruption does not establish acknowledgement or absence of effects; inspect before replay"}
	}
	return d
}

func bindStageDiagnostic(d *cli.Diagnostic, t stageTarget) {
	d.Target = map[string]string{"project": t.state.ProjectID, "execution": t.state.ExecutionID, "repository": t.state.RepositoryKey, "stage": string(t.state.Stage)}
	if w := t.state.WorkItem; w.Provider != "" && w.Resource != "" && w.ExternalID != "" {
		d.Target["work-item"] = w.Provider + ":" + w.Resource + "#" + w.ExternalID
	}
	o := &d.Options[0]
	o.Arguments = map[string]string{"--project": t.state.ProjectID, "--execution": t.state.ExecutionID, "--repository": t.state.RepositoryKey}
	if w := d.Target["work-item"]; w != "" && t.state.RepositoryKey != "" {
		o.Arguments["--work-item"] = w
		o.MissingArguments = []string{}
		o.Availability = "available"
		o.Prerequisites = []string{"Reobserve current state; diagnostic grants no effect authority"}
	}
	d.ObservedRevision = strconv.FormatUint(t.state.Revision, 10)
	d.Observed = append(d.Observed, "Execution identity and revision observed through canonical resolution")
	if t.single != nil && len(t.single.Attempts) > 0 {
		d.Target["latestAttempt"] = t.single.Attempts[len(t.single.Attempts)-1].AttemptID
	}
	if t.request.ParentID != "" {
		d.Target["parent"] = t.request.ParentID
	}
}

// Partial completion requires an existing confirmed reference. Consume only
// typed application results, never synthesize a reference from error prose.
func stageConfirmedReferences(status completion.Status, payload any) []string {
	if status != completion.Partial {
		return nil
	}
	var ref, kind string
	switch p := payload.(type) {
	case stageexecution.RunResult:
		kind = "dispatch-authority:"
		if p.Publication.Graph != nil {
			ref = p.Publication.Graph.DispatchAuthorityRef
		}
		if p.Publication.Single != nil && len(p.Publication.Single.Run.Authorities) > 0 {
			ref = p.Publication.Single.Run.Authorities[len(p.Publication.Single.Run.Authorities)-1]
		}
	case stageexecution.RetryResult:
		kind, ref = "retry-authority:", p.AuthorityRef
	case stageexecution.SingleRetryResult:
		kind, ref = "retry-authority:", p.AuthorityRef
	case stageexecution.IntegrateResult:
		kind, ref = "integration-receipt:", p.ReceiptRef
	}
	if wire, err := hex.DecodeString(ref); err == nil && len(wire) == 32 {
		return []string{kind + ref}
	}
	return nil
}

func editFailureDiagnostic(f projectapp.EditFailure) *cli.Diagnostic {
	category := "edit_invalid_intent"
	switch f {
	case projectapp.EditProjectNotFound:
		category = "edit_project_not_found"
	case projectapp.EditProjectAmbiguous:
		category = "edit_project_ambiguous"
	case projectapp.EditStateUnsafe:
		category = "edit_state_unsafe"
	case projectapp.EditRecoveryRequired:
		category = "edit_recovery_required"
	case projectapp.EditUnknownRepository:
		category = "edit_unknown_repository"
	case projectapp.EditRepositoryUnavailable:
		category = "edit_repository_unavailable"
	case projectapp.EditCandidateInvalid:
		category = "edit_candidate_invalid"
	case projectapp.EditCancelled:
		category = "edit_cancelled"
	case projectapp.EditUnavailable:
		category = "edit_unavailable"
	case projectapp.EditUnknownIntegration:
		category = "edit_unknown_integration"
	case projectapp.EditIncompleteAuthority:
		category = "edit_incomplete_authority"
	}
	d := diagnosticBase("project configure edit", category, "project list")
	d.ConfirmedEffects = []string{}
	d.UncertainEffects = []string{}
	d.Observed = append(d.Observed, "Edit rejected before publication")
	d.Options[0].Availability = "available"
	d.Options[0].Prerequisites = []string{}
	d.Options[0].MissingArguments = []string{}
	if f == projectapp.EditProjectAmbiguous {
		d.MissingPrerequisites = []string{"Exact Project UUID; ask only for the missing identity"}
	}
	return d
}

func editOutcomeDiagnostic(out projectapp.EditResult, p projectapp.EditPreview) *cli.Diagnostic {
	d := diagnosticBase("project configure edit", "edit_"+string(out.Outcome), "project show")
	d.Target = map[string]string{"project": p.ProjectID}
	d.ObservedRevision = p.PortableRevision
	d.Options[0].Arguments = map[string]string{"--selector": p.ProjectID}
	d.Options[0].MissingArguments = []string{}
	d.Options[0].Availability = "available"
	d.Options[0].Prerequisites = []string{"Inspect current state before another preview; never replay stale authority"}
	switch out.Outcome {
	case projectapp.EditDenied, projectapp.EditConflicted:
		d.Observed = append(d.Observed, "Publication refused; current proposal remains in the edit payload")
		d.UncertainEffects = []string{}
		d.MissingPrerequisites = []string{"Fresh preview and exact Project ID, digest and local authority"}
	case projectapp.EditPartial:
		d.ConfirmedEffects = []string{"Portable publication confirmed"}
		d.UncertainEffects = []string{"Local publication or recovery bookkeeping incomplete"}
		if out.Publication.LocalCommitted {
			d.ConfirmedEffects = append(d.ConfirmedEffects, "Local publication confirmed")
			d.UncertainEffects = []string{"Recovery bookkeeping incomplete"}
		}
	case projectapp.EditRecovery:
		if out.Publication.LocalCommitted {
			d.ConfirmedEffects = []string{"Project publication confirmed"}
		}
		d.UncertainEffects = []string{"Recovery bookkeeping or publication must be inspected"}
	}
	return d
}

func workflowDiagnostic(out projectapp.WorkflowAuthoringResult) *cli.Diagnostic {
	d := diagnosticBase("project workflow", out.Category, "project workflow list")
	r := out.Report
	d.ObservedRevision = r.ProjectRevision
	if r.ProjectID != "" {
		d.Target["project"] = r.ProjectID
		d.Options[0].Arguments = map[string]string{"--project": r.ProjectID}
		d.Options[0].MissingArguments = []string{}
		d.Options[0].Availability = "available"
	}
	if ref := r.Reference; ref != nil && ref.Valid() {
		d.Target["workflow"] = ref.WorkflowID
		d.Target["revision"] = strconv.Itoa(ref.Revision)
		d.Target["digest"] = ref.Digest
	}
	switch out.Category {
	case "partial":
		d.ConfirmedEffects = []string{"Workflow revision publication confirmed by the canonical reference"}
		d.UncertainEffects = []string{"Index/projection publication incomplete; inspect recovery before any replay"}
	case "recovery_required", "cancelled":
		d.UncertainEffects = []string{"Publication outcome or recovery state requires inspection; no automatic replay"}
	default:
		d.Observed = append(d.Observed, "Authoring operation refused; existing report retained")
		d.UncertainEffects = []string{}
	}
	if out.Category == "builtin_read_only" {
		d.MissingPrerequisites = []string{"Human-chosen editable Project copy key and exact create preview/authority", "Selection is a separate operation with its own preview/authority"}
	}
	return d
}
