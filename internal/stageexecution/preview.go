package stageexecution

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"time"

	"github.com/rgomids/axiom/internal/executiongraph"
	"github.com/rgomids/axiom/internal/runtimeprofile"
	"github.com/rgomids/axiom/internal/workflowcompiler"
	"github.com/rgomids/axiom/internal/workflowdefinition"
)

var ErrStalePreview = errors.New("stale_preview")
var ErrPreview = errors.New("invalid_dispatch_preview")

// StagePlanCompiler is the existing compiler's read-only PlanStageV2 boundary.
type StagePlanCompiler interface {
	PlanStageV2(context.Context, workflowcompiler.StagePlanRequest) (workflowcompiler.StagePlan, error)
}

// PreviewObservation is loaded from trusted local stores, never from agent output.
// Request must use the Execution's immutable WorkflowBinding snapshot and revision.
type PreviewObservation struct {
	Contract              Contract
	Request               workflowcompiler.StagePlanRequest
	Reviewed              workflowcompiler.StagePlan
	PolicyDigest          string
	ValidatorPolicyDigest string
	AxiomDigest           string
	Workspaces            map[string]string
	Inboxes               map[string]string
}
type PreviewSource interface {
	Observe(context.Context, workflowcompiler.ExecutionRef, string) (PreviewObservation, error)
}
type PreviewService struct {
	Source   PreviewSource
	Compiler StagePlanCompiler
}

type ChildRights struct {
	AgentID      string                           `json:"agentId"`
	Workspace    string                           `json:"workspace"`
	Inbox        string                           `json:"inbox"`
	Reads        string                           `json:"reads"`
	Writes       []string                         `json:"writes"`
	Commands     string                           `json:"commands"`
	Network      string                           `json:"network"`
	Effects      []executiongraph.Effect          `json:"effects"`
	Controls     executiongraph.ExecutionControls `json:"controls"`
	Confinement  string                           `json:"confinement"`
	Prohibitions map[string]string                `json:"prohibitions"`
	NotTested    []string                         `json:"not_tested"`
}
type DispatchPreview struct {
	Plan                  workflowcompiler.StagePlan         `json:"stagePlan"`
	Publication           *executiongraph.PublicationRequest `json:"publication,omitempty"`
	PolicyDigest          string                             `json:"policyDigest"`
	ValidatorPolicyDigest string                             `json:"validatorPolicyDigest"`
	AxiomDigest           string                             `json:"axiomDigest"`
	Children              []ChildRights                      `json:"children"`
	Digest                string                             `json:"digest"`
}

func canonicalDigest(v any) (string, error) {
	b, e := json.Marshal(v)
	if e != nil {
		return "", e
	}
	d := sha256.Sum256(b)
	return hex.EncodeToString(d[:]), nil
}
func previewDigest(p DispatchPreview) (string, error) { p.Digest = ""; return canonicalDigest(p) }

func (s PreviewService) Preview(ctx context.Context, ref workflowcompiler.ExecutionRef, stage string) (DispatchPreview, error) {
	if s.Source == nil || s.Compiler == nil {
		return DispatchPreview{}, ErrPreview
	}
	o, e := s.Source.Observe(ctx, ref, stage)
	if e != nil {
		return DispatchPreview{}, e
	}
	if _, e = Admit(StageRun, o.Contract); e != nil {
		return DispatchPreview{}, e
	}
	if o.Contract.Execution.ExecutionID != ref.ExecutionID || o.Contract.Execution.Revision != ref.Revision || string(o.Contract.Execution.Stage) != stage || o.Contract.Execution.Binding.Definition != o.Request.Compile.Workflow {
		return DispatchPreview{}, ErrStalePreview
	}
	if o.Request.Execution != ref || o.Reviewed.Execution != ref || o.Reviewed.StageID != stage || o.Request.Compile.StageID != stage || o.Request.Compile.Workflow != o.Contract.Definition.Ref(o.Request.Compile.Workflow.Source) || o.Request.Compile.Definition.Digest != o.Contract.Definition.Digest {
		return DispatchPreview{}, ErrStalePreview
	}
	if !workflowdefinition.ValidDigest(o.PolicyDigest) || !workflowdefinition.ValidDigest(o.ValidatorPolicyDigest) || !workflowdefinition.ValidDigest(o.AxiomDigest) {
		return DispatchPreview{}, ErrPreview
	}
	plan, e := s.Compiler.PlanStageV2(ctx, o.Request)
	if e != nil {
		return DispatchPreview{}, e
	}
	if plan.FormatVersion != workflowcompiler.OperationalStagePlanVersion || plan.Execution != ref || plan.StageID != stage || plan.Digest != o.Reviewed.Digest || plan.PlanDocumentDigest != o.Reviewed.PlanDocumentDigest || !reflect.DeepEqual(plan, o.Reviewed) {
		return DispatchPreview{}, ErrStalePreview
	}
	if len(plan.Blockers) != 0 || len(plan.Compilation.Agents) == 0 || len(plan.Compilation.Agents) > 32 {
		return DispatchPreview{}, ErrPreview
	}
	p := DispatchPreview{Plan: plan, PolicyDigest: o.PolicyDigest, ValidatorPolicyDigest: o.ValidatorPolicyDigest, AxiomDigest: o.AxiomDigest}
	if plan.ExecutionKind != "single" {
		if plan.Compilation.Proposal == nil || len(plan.Compilation.Proposal.Nodes) > 32 {
			return DispatchPreview{}, ErrPreview
		}
		p.Publication = &executiongraph.PublicationRequest{Proposal: *plan.Compilation.Proposal, ExpectedDigest: plan.Compilation.Proposal.Digest, GraphRevision: ref.Revision, ParentAuthority: plan.Compilation.AuthorityCeiling, ChildAuthorities: map[string][]executiongraph.Effect{}, AuthorityReferences: map[string]string{}, Workspaces: map[string]string{}, Resolutions: map[string]executiongraph.Resolution{}}
	} else if len(plan.Compilation.Agents) != 1 {
		return DispatchPreview{}, ErrPreview
	}
	for _, a := range plan.Compilation.Agents {
		frozen := a.Input.FrozenProfile
		if frozen == nil || runtimeprofile.ValidateResolvedProfileV4(*frozen) != nil || a.Choice.FrozenProfile == nil || *a.Choice.FrozenProfile != *frozen || !executiongraph.ValidFrozenProfileControls(frozen, a.Controls, a.Capability) {
			return DispatchPreview{}, ErrPreview
		}
		if a.Controls.MaximumAttempts < 1 || a.Controls.MaximumAttempts > 10 || a.Controls.Timeout <= 0 || a.Controls.Timeout > 24*time.Hour {
			return DispatchPreview{}, ErrPreview
		}
		workspace, inbox := o.Workspaces[a.Input.NodeKey], o.Inboxes[a.Input.NodeKey]
		if workspace == "" || inbox == "" || workspace == inbox {
			return DispatchPreview{}, ErrPreview
		}
		writes := []string{inbox}
		for _, effect := range a.Effects {
			if effect.Kind == "repository-write" {
				writes = append(writes, workspace)
			}
		}
		// Axiom passes the adapter's mandatory permission and sandbox options
		// but does not verify vendor confinement; every prohibition is policy.
		prohibitions := map[string]string{}
		for _, k := range []string{"push", "PR", "merge", "Provider calls", "credential access", "nested agents", "writes outside observed writable roots"} {
			prohibitions[k] = "policy-only"
		}
		untested := []string{"external-network", "home", "runtime-sandbox", "system-temp", "tmpdir", "vendor-home"}
		p.Children = append(p.Children, ChildRights{AgentID: a.AgentID, Workspace: workspace, Inbox: inbox, Reads: "worktree; reads outside are not OS-confined", Writes: writes, Commands: "within worktree under the Runtime's own permission mode; not verified by Axiom", Network: "external-network: not_tested and not confined by Axiom", Effects: a.Effects, Controls: a.Controls, Confinement: "unknown", Prohibitions: prohibitions, NotTested: untested})
		if p.Publication != nil {
			key := a.Input.NodeKey
			p.Publication.ChildAuthorities[key] = a.Effects
			p.Publication.Workspaces[key] = workspace
			p.Publication.AuthorityReferences[key] = "pending-dispatch-authority"
			p.Publication.Resolutions[key] = executiongraph.Resolution{RuntimeID: frozen.RuntimeID, ModelProfileID: frozen.ProfileID, ConfigurationRevision: frozen.ConfigurationRevision, ObservationRevision: frozen.ObservationRevision, FrozenProfile: frozen}
		}
	}
	sort.Slice(p.Children, func(i, j int) bool { return p.Children[i].AgentID < p.Children[j].AgentID })
	if p.Publication != nil {
		n := 0
		_, e = executiongraph.NewGraphService(nil, func() (string, error) { n++; return fmt.Sprintf("00000000-0000-4000-8000-%012d", n), nil }, func() time.Time { return time.Unix(1, 0).UTC() }).Build(*p.Publication)
		if e != nil {
			return DispatchPreview{}, e
		}
	}
	p.Digest, e = previewDigest(p)
	return p, e
}
func uniqueStrings(v []string) []string {
	out := v[:0]
	for _, s := range v {
		if len(out) == 0 || out[len(out)-1] != s {
			out = append(out, s)
		}
	}
	return out
}
