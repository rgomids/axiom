package executiongraph

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

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

// IntegrationReceipt is the separate, single-use local authority for one
// integration attempt. It binds the exact graph storage revision, the stage
// integrate preview, the reviewed validator policy and the Execution ledger
// generation. The integration attempt's packetDigest references it, so the
// stage outcome is derivable from persisted state without repeating effects.
type IntegrationReceipt struct {
	FormatVersion            int              `json:"formatVersion"`
	ProjectID                string           `json:"projectId"`
	ParentID                 string           `json:"parentId"`
	ExecutionID              string           `json:"executionId,omitempty"`
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
	// Format 1 used the graph parent as the ledger Execution identity. Preserve
	// its exact encoding and digest; format 2 binds the distinct Execution.
	if (a.FormatVersion != 1 && a.FormatVersion != 2) || a.FormatVersion == 1 && a.ExecutionID != "" || a.FormatVersion == 2 && !validOpaqueID(a.ExecutionID) || !validToken(a.ProjectID) || !validOpaqueID(a.ParentID) || a.Revision == 0 || a.ExecutionRevision != a.Revision || !successorDigest(a.GraphDigest) || !successorDigest(a.PreviewDigest) || !successorDigest(a.IntegrationPreviewDigest) || !successorDigest(a.ValidatorPolicyDigest) || !validOpaqueID(a.IntegrationChildID) || a.AttemptNumber == 0 || a.OptionalWaivers == nil || len(a.OptionalWaivers) > maxListItems || !validDispatchEffects(a.Effects) || !successorDigest(a.LedgerDigest) || !validToken(a.StageID) || a.StageOrdinal < 0 || a.ActorKind != "human-operator" || !boundedIdentity(a.Actor) || !utc(a.RecordedAt) {
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
