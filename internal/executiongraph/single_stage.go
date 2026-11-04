package executiongraph

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"

	"time"

	"github.com/rgomids/axiom/internal/runtimeprofile"
	"github.com/rgomids/axiom/internal/strictjson"
)

const (
	SingleStageRunKind               = "single_stage_run"
	SingleStageRunVersion            = 1
	OperationalSingleStageRunVersion = 2
	SingleStageAuthorityVersion      = 2
	SingleStageAuthorityScope        = "single-stage-attempt"
	MaxSingleStageRunBytes           = 1 << 20
	MaxSingleStageAttempts           = 10
	MaxSingleStageAttemptTime        = 24 * time.Hour
)

var ErrUnsupportedSingleStageFormat = errors.New("unsupported_single_stage_format")

// SingleStageScope is the exact identity of one single-agent stage of one
// Execution. It never names a Graph parent or child.
type SingleStageScope struct {
	ProjectID     string `json:"projectId"`
	RepositoryKey string `json:"repositoryKey"`
	WorkItem      string `json:"workItem"`
	ExecutionID   string `json:"executionId"`
	StageID       string `json:"stageId"`
	StageOrdinal  int    `json:"stageOrdinal"`
}

// SingleStageBinding is immutable for the life of the ledger: a changed source
// needs a new reviewed operation, never a silent re-resolution.
type SingleStageBinding struct {
	ExecutionGeneration string            `json:"executionGeneration"`
	WorkflowRevision    uint64            `json:"workflowRevision"`
	PreviewDigest       string            `json:"previewDigest"`
	ProposalDigest      string            `json:"proposalDigest"`
	InputDigest         string            `json:"inputDigest"`
	AxiomDigest         string            `json:"axiomDigest"`
	Workspace           string            `json:"workspace"`
	Inbox               string            `json:"inbox"`
	Effects             []Effect          `json:"effects"`
	Controls            ExecutionControls `json:"controls"`
	Resolution          Resolution        `json:"resolution"`
}

// SingleStageRun is the operational attempt ledger of ADR-0027. Authorities[i]
// authorizes attempt i+1; a trailing reference without its attempt is pending.
type SingleStageRun struct {
	Kind            string             `json:"kind"`
	FormatVersion   int                `json:"formatVersion"`
	ScopeDigest     string             `json:"scopeDigest"`
	Scope           SingleStageScope   `json:"scope"`
	Binding         SingleStageBinding `json:"binding"`
	CreatedAt       time.Time          `json:"createdAt"`
	RunEpoch        uint64             `json:"runEpoch"`
	Authorities     []string           `json:"authorityRefs"`
	Attempts        []SuccessorAttempt `json:"attempts"`
	CancelRequest   *CancelRequest     `json:"cancelRequest,omitempty"`
	StorageRevision [sha256.Size]byte  `json:"-"`
}

// SingleStageAuthority is the version-2 single-stage dispatch receipt. It is a
// separate codec from the version-1 DispatchAuthority and never decodes as one.
// DispatchDigest is the dispatch preview re-observed for this attempt; for the
// first attempt it is the bound preview itself.
type SingleStageAuthority struct {
	FormatVersion       int       `json:"formatVersion"`
	Scope               string    `json:"scope"`
	ProjectID           string    `json:"projectId"`
	ExecutionID         string    `json:"executionId"`
	StageID             string    `json:"stageId"`
	StageOrdinal        int       `json:"stageOrdinal"`
	ScopeDigest         string    `json:"scopeDigest"`
	ExecutionGeneration string    `json:"executionGeneration"`
	WorkflowRevision    uint64    `json:"workflowRevision"`
	LedgerGeneration    string    `json:"ledgerGeneration,omitempty"`
	PreviewDigest       string    `json:"previewDigest"`
	DispatchDigest      string    `json:"dispatchDigest"`
	ProposalDigest      string    `json:"proposalDigest"`
	AttemptNumber       uint32    `json:"attemptNumber"`
	PriorAttemptID      string    `json:"priorAttemptId,omitempty"`
	Effects             []Effect  `json:"effects"`
	Disposition         string    `json:"disposition"`
	ActorKind           string    `json:"actorKind"`
	Actor               string    `json:"actor"`
	RecordedAt          time.Time `json:"recordedAt"`
}

func SingleStageScopeDigest(s SingleStageScope) (string, error) {
	if !validSingleScope(s) {
		return "", ErrInvalidGraph
	}
	b, err := json.Marshal(s)
	if err != nil {
		return "", err
	}
	d := sha256.Sum256(b)
	return hex.EncodeToString(d[:]), nil
}

func validSingleScope(s SingleStageScope) bool {
	return validToken(s.ProjectID) && validToken(s.RepositoryKey) && validTarget(s.WorkItem) && validOpaqueID(s.ExecutionID) && validToken(s.StageID) && s.StageOrdinal >= 0
}

func validSingleBinding(version int, b SingleStageBinding) bool {
	if !successorDigest(b.ExecutionGeneration) || b.WorkflowRevision == 0 || !successorDigest(b.PreviewDigest) || !successorDigest(b.ProposalDigest) || !successorDigest(b.InputDigest) || !successorDigest(b.AxiomDigest) || !validTarget(b.Workspace) || !validTarget(b.Inbox) || b.Workspace == b.Inbox || !validDispatchEffects(b.Effects) {
		return false
	}
	if b.Controls.MaximumAttempts < 1 || b.Controls.MaximumAttempts > MaxSingleStageAttempts || b.Controls.Timeout <= 0 || b.Controls.Timeout > MaxSingleStageAttemptTime {
		return false
	}
	if version == OperationalSingleStageRunVersion {
		p := b.Resolution.FrozenProfile
		return p != nil && b.Resolution.ResolvedProfile == nil && validResolution(b.Resolution) && validEffortControls(p.Effort, b.Controls, CapabilityRequest{Capabilities: effortCapability(p.Effort)})
	}
	p := b.Resolution.ResolvedProfile
	return p != nil && b.Resolution.FrozenProfile == nil && runtimeprofile.ValidateResolvedProfile(*p) == nil && b.Resolution.RuntimeID == p.RuntimeID && b.Resolution.ModelProfileID == p.ProfileID
}

func effortCapability(e runtimeprofile.Effort) []string {
	if e.Mode == runtimeprofile.EffortExplicit {
		return []string{"reasoning-effort-" + e.Value}
	}
	return nil
}

func ValidateSingleStageRun(r SingleStageRun) error {
	digest, err := SingleStageScopeDigest(r.Scope)
	if r.Kind != SingleStageRunKind || r.FormatVersion != SingleStageRunVersion && r.FormatVersion != OperationalSingleStageRunVersion || err != nil || digest != r.ScopeDigest || !validSingleBinding(r.FormatVersion, r.Binding) || !utc(r.CreatedAt) {
		return ErrInvalidGraph
	}
	if len(r.Authorities) == 0 || len(r.Authorities) > MaxSingleStageAttempts || len(r.Attempts) > len(r.Authorities) || len(r.Authorities) > len(r.Attempts)+1 {
		return ErrInvalidGraph
	}
	refs := map[string]bool{}
	for _, ref := range r.Authorities {
		if !successorDigest(ref) || refs[ref] {
			return ErrInvalidGraph
		}
		refs[ref] = true
	}
	if c := r.CancelRequest; c != nil {
		if !validOpaqueID(c.RequestID) || c.ParentID != r.Scope.ExecutionID || c.GraphRevision != r.Binding.WorkflowRevision || c.ActorKind != "human-operator" || !boundedIdentity(c.Actor) || !utc(c.RequestedAt) || c.RequestedAt.Before(r.CreatedAt) || !successorDigest(c.AuthorityRef) || refs[c.AuthorityRef] {
			return ErrInvalidGraph
		}
	}
	ids := map[string]bool{r.Scope.ExecutionID: true}
	facts := map[string]bool{}
	scope := attemptScope{diagnostics: r.FormatVersion == OperationalSingleStageRunVersion, created: r.CreatedAt, cancel: r.CancelRequest, parentID: r.Scope.ExecutionID, childID: r.Scope.ExecutionID, revision: r.Binding.WorkflowRevision, maximum: r.Binding.Controls.MaximumAttempts, authority: refs}
	if err := validateAttemptSequence(r.Attempts, scope, ids, facts); err != nil {
		return err
	}
	for _, a := range r.Attempts {
		if a.LeaseEpoch > r.RunEpoch {
			return ErrInvalidGraph
		}
	}
	if r.CancelRequest != nil && (ids[r.CancelRequest.RequestID] || facts[r.CancelRequest.RequestID]) {
		return ErrInvalidGraph
	}
	return nil
}

func EncodeSingleStageRun(r SingleStageRun) ([]byte, error) {
	if ValidateSingleStageRun(r) != nil {
		return nil, ErrInvalidGraph
	}
	b, err := json.Marshal(r)
	if err != nil || len(b)+1 > MaxSingleStageRunBytes {
		return nil, ErrInvalidGraph
	}
	return append(b, '\n'), nil
}

// DecodeSingleStageRun admits only kind single_stage_run format 1. A different
// kind is not this codec's record; an unknown format fails closed.
func DecodeSingleStageRun(wire []byte) (SingleStageRun, error) {
	var header struct {
		Kind          string `json:"kind"`
		FormatVersion int    `json:"formatVersion"`
	}
	if len(wire) > MaxSingleStageRunBytes || json.Unmarshal(wire, &header) != nil || header.Kind != SingleStageRunKind {
		return SingleStageRun{}, ErrInvalidGraph
	}
	if header.FormatVersion != SingleStageRunVersion && header.FormatVersion != OperationalSingleStageRunVersion {
		return SingleStageRun{}, ErrUnsupportedSingleStageFormat
	}
	var r SingleStageRun
	if strictjson.Decode(wire, &r, MaxSingleStageRunBytes) != nil || ValidateSingleStageRun(r) != nil {
		return SingleStageRun{}, ErrInvalidGraph
	}
	r.StorageRevision = sha256.Sum256(wire)
	return r, nil
}

func ValidateSingleStageAuthority(a SingleStageAuthority) error {
	if a.FormatVersion != SingleStageAuthorityVersion || a.Scope != SingleStageAuthorityScope || !validToken(a.ProjectID) || !validOpaqueID(a.ExecutionID) || !validToken(a.StageID) || a.StageOrdinal < 0 || !successorDigest(a.ScopeDigest) || !successorDigest(a.ExecutionGeneration) || a.WorkflowRevision == 0 || !successorDigest(a.PreviewDigest) || !successorDigest(a.DispatchDigest) || !successorDigest(a.ProposalDigest) || !validDispatchEffects(a.Effects) || a.ActorKind != "human-operator" || !boundedIdentity(a.Actor) || !utc(a.RecordedAt) {
		return ErrInvalidGraph
	}
	if a.AttemptNumber < 1 || a.AttemptNumber > MaxSingleStageAttempts {
		return ErrInvalidGraph
	}
	if a.AttemptNumber == 1 {
		if a.LedgerGeneration != "" || a.PriorAttemptID != "" || a.Disposition != DispositionNotRequired || a.DispatchDigest != a.PreviewDigest {
			return ErrInvalidGraph
		}
		return nil
	}
	if !successorDigest(a.LedgerGeneration) || !validOpaqueID(a.PriorAttemptID) {
		return ErrInvalidGraph
	}
	switch a.Disposition {
	case DispositionNotRequired, DispositionPreserve, DispositionReset:
		return nil
	}
	return ErrInvalidGraph
}

func EncodeSingleStageAuthority(a SingleStageAuthority) ([]byte, error) {
	if ValidateSingleStageAuthority(a) != nil {
		return nil, ErrInvalidGraph
	}
	b, err := json.Marshal(a)
	if err != nil || len(b)+1 > MaxDispatchAuthorityBytes {
		return nil, ErrInvalidGraph
	}
	return append(b, '\n'), nil
}

// DecodeSingleStageAuthority accepts only version 2 with the single-stage
// scope. A version-1 receipt or an unknown version is refused, never mapped.
func DecodeSingleStageAuthority(wire []byte) (SingleStageAuthority, error) {
	var header struct {
		FormatVersion int    `json:"formatVersion"`
		Scope         string `json:"scope"`
	}
	if len(wire) > MaxDispatchAuthorityBytes || json.Unmarshal(wire, &header) != nil || header.Scope != SingleStageAuthorityScope {
		return SingleStageAuthority{}, ErrInvalidGraph
	}
	if header.FormatVersion != SingleStageAuthorityVersion {
		return SingleStageAuthority{}, ErrUnsupportedSingleStageFormat
	}
	var a SingleStageAuthority
	if strictjson.Decode(wire, &a, MaxDispatchAuthorityBytes) != nil || ValidateSingleStageAuthority(a) != nil {
		return SingleStageAuthority{}, ErrInvalidGraph
	}
	return a, nil
}

func SingleStageAuthorityDigest(a SingleStageAuthority) (string, error) {
	b, err := EncodeSingleStageAuthority(a)
	if err != nil {
		return "", err
	}
	d := sha256.Sum256(b)
	return hex.EncodeToString(d[:]), nil
}
