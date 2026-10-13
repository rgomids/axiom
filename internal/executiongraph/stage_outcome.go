package executiongraph

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"time"

	"github.com/rgomids/axiom/internal/strictjson"
)

const MaxStageOutcomeBytes = 256 << 10

// StageChildOutcome is copied from persisted attempts; it is never computed by
// classifying records again.
type StageChildOutcome struct {
	ChildID   string               `json:"childId"`
	AttemptID string               `json:"attemptId,omitempty"`
	Status    AttemptStatus        `json:"status"`
	Reason    ClassificationReason `json:"reason,omitempty"`
	Required  bool                 `json:"required"`
	Waived    bool                 `json:"waived,omitempty"`
}

// StageRollup is the deterministic parent roll-up of one successor graph.
// "success" needs the integration attempt integration_applied and every
// required child result_recorded; it never accepts the Work Item.
type StageRollup struct {
	Status               string              `json:"status"`
	ParentID             string              `json:"parentId"`
	GraphRevision        uint64              `json:"graphRevision"`
	IntegrationChildID   string              `json:"integrationChildId"`
	IntegrationAttemptID string              `json:"integrationAttemptId,omitempty"`
	Children             []StageChildOutcome `json:"children"`
}

// RollupSuccessor derives the roll-up from persisted fields only. Waivers name
// optional children the authorized integration preview excluded.
func RollupSuccessor(g SuccessorGraph, waivers []OptionalWaiver) (StageRollup, error) {
	if ValidateSuccessorGraph(g) != nil {
		return StageRollup{}, ErrInvalidGraph
	}
	waived := map[string]bool{}
	for _, w := range waivers {
		waived[w.ChildID] = true
	}
	r := StageRollup{Status: "partial", ParentID: g.Parent.ExecutionID, GraphRevision: g.Parent.GraphRevision, IntegrationChildID: g.Parent.IntegrationChild}
	complete := true
	var integration *SuccessorAttempt
	for _, c := range g.Children {
		o := StageChildOutcome{ChildID: c.ExecutionID, Status: AttemptNotStarted, Required: !c.Envelope.Optional}
		if len(c.Attempts) > 0 {
			a := c.Attempts[len(c.Attempts)-1]
			o.AttemptID, o.Status, o.Reason = a.AttemptID, a.Status, a.ClassificationReason
			if c.Envelope.IntegrationOwner {
				copy := a
				integration = &copy
			}
		}
		if c.Envelope.IntegrationOwner {
			r.Children = append(r.Children, o)
			continue
		}
		if waived[c.ExecutionID] {
			if !c.Envelope.Optional {
				return StageRollup{}, ErrInvalidGraph
			}
			o.Waived = true
		} else if o.Status != AttemptSucceeded || o.Reason != ResultRecorded {
			complete = false
		}
		r.Children = append(r.Children, o)
	}
	sort.Slice(r.Children, func(i, j int) bool { return r.Children[i].ChildID < r.Children[j].ChildID })
	if integration != nil {
		r.IntegrationAttemptID = integration.AttemptID
		switch {
		case integration.Status == AttemptSucceeded && integration.ClassificationReason == IntegrationApplied && complete:
			r.Status = "success"
		case integration.Status == AttemptFailed:
			r.Status = "failure"
		case integration.Status == AttemptUnknown || integration.Status == AttemptRunning:
			r.Status = "recovery_required"
		}
	}
	return r, nil
}

// IntegrationReceipt is the separate, single-use local authority for one
// integration attempt. It binds the exact graph storage revision, the stage
// integrate preview, the reviewed validator policy and the Execution ledger
// generation. The integration attempt's packetDigest references it, so the
// stage outcome is derivable from persisted state without repeating effects.
type IntegrationReceipt struct {
	FormatVersion            int              `json:"formatVersion"`
	ProjectID                string           `json:"projectId"`
	ParentID                 string           `json:"parentId"`
	ExecutionID              string           `json:"executionId"`
	Revision                 uint64           `json:"revision"`
	GraphDigest              string           `json:"graphDigest"`
	PreviewDigest            string           `json:"previewDigest"`
	IntegrationPreviewDigest string           `json:"integrationPreviewDigest"`
	ValidatorPolicyDigest    string           `json:"validatorPolicyDigest"`
	IntegrationChildID       string           `json:"integrationChildId"`
	AttemptNumber            uint32           `json:"attemptNumber"`
	OptionalWaivers          []OptionalWaiver `json:"optionalWaivers"`
	Effects                  []Effect         `json:"effects"`
	ExecutionRevision        uint64           `json:"executionRevision"`
	LedgerDigest             string           `json:"ledgerDigest"`
	StageID                  string           `json:"stageId"`
	StageOrdinal             int              `json:"stageOrdinal"`
	ActorKind                string           `json:"actorKind"`
	Actor                    string           `json:"actor"`
	RecordedAt               time.Time        `json:"recordedAt"`
}

func ValidateIntegrationReceipt(a IntegrationReceipt) error {
	if a.FormatVersion != 1 || !validToken(a.ProjectID) || !validOpaqueID(a.ParentID) || !validOpaqueID(a.ExecutionID) || a.Revision == 0 || a.ExecutionRevision != a.Revision || !successorDigest(a.GraphDigest) || !successorDigest(a.PreviewDigest) || !successorDigest(a.IntegrationPreviewDigest) || !successorDigest(a.ValidatorPolicyDigest) || !validOpaqueID(a.IntegrationChildID) || a.AttemptNumber == 0 || a.OptionalWaivers == nil || len(a.OptionalWaivers) > maxListItems || !validDispatchEffects(a.Effects) || !successorDigest(a.LedgerDigest) || !validToken(a.StageID) || a.StageOrdinal < 0 || a.ActorKind != "human-operator" || !boundedIdentity(a.Actor) || !utc(a.RecordedAt) {
		return ErrInvalidGraph
	}
	for i, w := range a.OptionalWaivers {
		if !validOpaqueID(w.ChildID) || !validTarget(w.Reference) || i > 0 && a.OptionalWaivers[i-1].ChildID >= w.ChildID {
			return ErrInvalidGraph
		}
	}
	return nil
}

func EncodeIntegrationReceipt(a IntegrationReceipt) ([]byte, error) {
	if ValidateIntegrationReceipt(a) != nil {
		return nil, ErrInvalidGraph
	}
	b, err := json.Marshal(a)
	if err != nil || len(b)+1 > MaxRetryAuthorityBytes {
		return nil, ErrInvalidGraph
	}
	return append(b, '\n'), nil
}

func DecodeIntegrationReceipt(b []byte) (IntegrationReceipt, error) {
	var a IntegrationReceipt
	if strictjson.Decode(b, &a, MaxRetryAuthorityBytes) != nil || ValidateIntegrationReceipt(a) != nil {
		return IntegrationReceipt{}, ErrInvalidGraph
	}
	return a, nil
}

func IntegrationReceiptDigest(a IntegrationReceipt) (string, error) {
	b, err := EncodeIntegrationReceipt(a)
	if err != nil {
		return "", err
	}
	d := sha256.Sum256(b)
	return hex.EncodeToString(d[:]), nil
}

// CheckIntegrationDispatch is called under the run lease immediately before
// the running integration attempt is saved. A receipt authorizes exactly the
// next integration attempt of the graph revision it observed.
func CheckIntegrationDispatch(g SuccessorGraph, a IntegrationReceipt) error {
	if ValidateIntegrationReceipt(a) != nil || ValidateSuccessorGraph(g) != nil || g.StorageRevision == ([32]byte{}) || hex.EncodeToString(g.StorageRevision[:]) != a.GraphDigest || a.ParentID != g.Parent.ExecutionID || a.Revision != g.Parent.GraphRevision || a.ProjectID != g.Children[0].Envelope.Scope.ProjectID || a.IntegrationChildID != g.Parent.IntegrationChild {
		return ErrInvalidGraph
	}
	available, err := SuccessorIntegrationAttemptAvailable(g)
	if err != nil || !available {
		return ErrIntegrationBlocked
	}
	for _, c := range g.Children {
		if c.ExecutionID == a.IntegrationChildID {
			if !c.Envelope.IntegrationOwner || uint32(len(c.Attempts))+1 != a.AttemptNumber {
				return ErrInvalidGraph
			}
			return nil
		}
	}
	return ErrInvalidGraph
}

// StageOutcome projects one terminal integration attempt into the Execution
// ledger without changing the Execution3 codec: it binds the exact ledger
// generation, graph storage revision and integration receipt. It records an
// outcome; it is not acceptance of the Work Item and grants no authority.
type StageOutcome struct {
	FormatVersion              int         `json:"formatVersion"`
	ProjectID                  string      `json:"projectId"`
	ExecutionID                string      `json:"executionId"`
	ExecutionRevision          uint64      `json:"executionRevision"`
	LedgerDigest               string      `json:"ledgerDigest"`
	StageID                    string      `json:"stageId"`
	StageOrdinal               int         `json:"stageOrdinal"`
	GraphDigest                string      `json:"graphDigest"`
	IntegrationReceiptRef      string      `json:"integrationReceiptRef"`
	ValidatorPolicyDigest      string      `json:"validatorPolicyDigest"`
	Rollup                     StageRollup `json:"rollup"`
	IntegrationResultReference string      `json:"integrationResultReference,omitempty"`
	ValidationOutputDigest     string      `json:"validationOutputDigest,omitempty"`
	RecordedAt                 time.Time   `json:"recordedAt"`
}

func ValidateStageOutcome(o StageOutcome) error {
	if o.FormatVersion != 1 || !validToken(o.ProjectID) || !validOpaqueID(o.ExecutionID) || o.ExecutionRevision == 0 || !successorDigest(o.LedgerDigest) || !validToken(o.StageID) || o.StageOrdinal < 0 || !successorDigest(o.GraphDigest) || !successorDigest(o.IntegrationReceiptRef) || !successorDigest(o.ValidatorPolicyDigest) || !utc(o.RecordedAt) {
		return ErrInvalidGraph
	}
	r := o.Rollup
	// The graph parent is an allocated identity distinct from the Execution.
	if !validOpaqueID(r.ParentID) || r.GraphRevision != o.ExecutionRevision || !validOpaqueID(r.IntegrationChildID) || !validOpaqueID(r.IntegrationAttemptID) || len(r.Children) == 0 || len(r.Children) > maxListItems {
		return ErrInvalidGraph
	}
	switch r.Status {
	case "success":
		if !successorDigest(o.IntegrationResultReference) || !successorDigest(o.ValidationOutputDigest) {
			return ErrInvalidGraph
		}
	case "failure":
		if o.IntegrationResultReference != "" && !successorDigest(o.IntegrationResultReference) || o.ValidationOutputDigest != "" && !successorDigest(o.ValidationOutputDigest) {
			return ErrInvalidGraph
		}
	default:
		// Unknown, running or incomplete integration outcomes are never projected.
		return ErrInvalidGraph
	}
	seen := map[string]bool{}
	integration := false
	for i, c := range r.Children {
		if !validOpaqueID(c.ChildID) || seen[c.ChildID] || i > 0 && r.Children[i-1].ChildID >= c.ChildID || c.AttemptID != "" && !validOpaqueID(c.AttemptID) {
			return ErrInvalidGraph
		}
		seen[c.ChildID] = true
		if c.ChildID == r.IntegrationChildID {
			integration = c.AttemptID == r.IntegrationAttemptID && (r.Status == "success" && c.Status == AttemptSucceeded && c.Reason == IntegrationApplied || r.Status == "failure" && c.Status == AttemptFailed && c.Reason == IntegrationFailed)
			continue
		}
		if r.Status == "success" && (c.Required && c.Waived || !c.Waived && (c.Status != AttemptSucceeded || c.Reason != ResultRecorded)) {
			return ErrInvalidGraph
		}
	}
	if !integration {
		return ErrInvalidGraph
	}
	return nil
}

// BuildStageOutcome derives the projection only from the persisted graph, its
// terminal integration attempt and that attempt's own integration receipt.
func BuildStageOutcome(g SuccessorGraph, receiptRef string, receipt IntegrationReceipt, recordedAt time.Time) (StageOutcome, error) {
	if ValidateIntegrationReceipt(receipt) != nil || receipt.ParentID != g.Parent.ExecutionID || receipt.Revision != g.Parent.GraphRevision {
		return StageOutcome{}, ErrInvalidGraph
	}
	r, err := RollupSuccessor(g, receipt.OptionalWaivers)
	if err != nil {
		return StageOutcome{}, err
	}
	var attempt *SuccessorAttempt
	for _, c := range g.Children {
		if c.ExecutionID == receipt.IntegrationChildID && len(c.Attempts) > 0 {
			a := c.Attempts[len(c.Attempts)-1]
			attempt = &a
		}
	}
	if attempt == nil || attempt.PacketDigest != receiptRef || attempt.Number != receipt.AttemptNumber {
		return StageOutcome{}, ErrInvalidGraph
	}
	o := StageOutcome{FormatVersion: 1, ProjectID: receipt.ProjectID, ExecutionID: receipt.ExecutionID, ExecutionRevision: receipt.ExecutionRevision, LedgerDigest: receipt.LedgerDigest, StageID: receipt.StageID, StageOrdinal: receipt.StageOrdinal, GraphDigest: hex.EncodeToString(g.StorageRevision[:]), IntegrationReceiptRef: receiptRef, ValidatorPolicyDigest: receipt.ValidatorPolicyDigest, Rollup: r, IntegrationResultReference: attempt.ResultReference, ValidationOutputDigest: attempt.OutputDigest, RecordedAt: recordedAt.UTC()}
	return o, ValidateStageOutcome(o)
}

func EncodeStageOutcome(o StageOutcome) ([]byte, error) {
	if ValidateStageOutcome(o) != nil {
		return nil, ErrInvalidGraph
	}
	b, err := json.Marshal(o)
	if err != nil || len(b)+1 > MaxStageOutcomeBytes {
		return nil, ErrInvalidGraph
	}
	return append(b, '\n'), nil
}

func DecodeStageOutcome(b []byte) (StageOutcome, error) {
	var o StageOutcome
	if strictjson.Decode(b, &o, MaxStageOutcomeBytes) != nil || ValidateStageOutcome(o) != nil {
		return StageOutcome{}, ErrInvalidGraph
	}
	return o, nil
}

func StageOutcomeDigest(o StageOutcome) (string, error) {
	b, err := EncodeStageOutcome(o)
	if err != nil {
		return "", err
	}
	d := sha256.Sum256(b)
	return hex.EncodeToString(d[:]), nil
}

// IntegrationResultDigest is the successor attempt ResultReference for an
// applied integration tree; the tree itself is retained by the stage outcome.
func IntegrationResultDigest(resultTree string) string {
	d := sha256.Sum256([]byte(IntegrationResultReference(resultTree)))
	return hex.EncodeToString(d[:])
}
