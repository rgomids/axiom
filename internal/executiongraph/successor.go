package executiongraph

import (
	"crypto/sha256"
	"encoding/json"
	"reflect"
	"strings"
	"time"

	"github.com/rgomids/axiom/internal/strictjson"
)

const SuccessorGraphVersion = 3
const OperationalGraphVersion = 4
const ClassificationPolicyV1 = "stage-attempt-v1"
const MaxStageConcurrency = 64
const MaxReconciliationFacts = 64

// SuccessorGraph is a separate wire family. Historical Graph is never extended.
type SuccessorGraph struct {
	FormatVersion        int               `json:"formatVersion"`
	Parent               ParentExecution   `json:"parent"`
	Children             []SuccessorChild  `json:"children"`
	ClassificationPolicy string            `json:"classificationPolicy"`
	DispatchAuthorityRef string            `json:"dispatchAuthorityRef"`
	StageConcurrency     uint32            `json:"stageConcurrency"`
	CancelRequest        *CancelRequest    `json:"cancelRequest,omitempty"`
	StorageRevision      [sha256.Size]byte `json:"-"`
}

type SuccessorChild struct {
	ExecutionID   string             `json:"executionId"`
	ParentID      string             `json:"parentId"`
	GraphRevision uint64             `json:"graphRevision"`
	NodeKey       string             `json:"nodeKey"`
	Envelope      ChildEnvelope      `json:"envelope"`
	Attempts      []SuccessorAttempt `json:"attempts"`
}

type SuccessorAttempt struct {
	AttemptID            string               `json:"attemptId"`
	Number               uint32               `json:"number"`
	Status               AttemptStatus        `json:"status"`
	StartedAt            *time.Time           `json:"startedAt,omitempty"`
	FinishedAt           *time.Time           `json:"finishedAt,omitempty"`
	ResultReference      string               `json:"resultReference,omitempty"`
	OutputDigest         string               `json:"outputDigest,omitempty"`
	AmbiguousEffect      bool                 `json:"ambiguousEffect"`
	CancellationSeen     bool                 `json:"cancellationSeen,omitempty"`
	ExitCode             int                  `json:"exitCode"`
	ClassificationReason ClassificationReason `json:"classificationReason,omitempty"`
	QualifyingRecordRef  string               `json:"qualifyingRecordRef,omitempty"`
	LeaseEpoch           uint64               `json:"leaseEpoch"`
	Process              *ProcessCorrelation  `json:"process,omitempty"`
	PacketDigest         string               `json:"packetDigest"`
	ReconciliationFacts  []ReconciliationFact `json:"reconciliationFacts"`
	Diagnostic           *AttemptDiagnostic   `json:"diagnostic,omitempty"`
}

type ProcessCorrelation struct {
	PID           int64  `json:"pid"`
	StartIdentity string `json:"startIdentity"`
	GroupIdentity string `json:"groupIdentity"`
}

type CancelRequest struct {
	RequestID     string    `json:"requestId"`
	ParentID      string    `json:"parentId"`
	GraphRevision uint64    `json:"graphRevision"`
	ActorKind     string    `json:"actorKind"`
	Actor         string    `json:"actor"`
	RequestedAt   time.Time `json:"requestedAt"`
	AuthorityRef  string    `json:"authorityRef"`
}

type ReconciliationFact struct {
	FactID             string    `json:"factId"`
	ParentID           string    `json:"parentId"`
	ChildID            string    `json:"childId"`
	GraphRevision      uint64    `json:"graphRevision"`
	AttemptID          string    `json:"attemptId"`
	ObservationsDigest string    `json:"observationsDigest"`
	Classification     string    `json:"classification"`
	AuthorityRef       string    `json:"authorityRef"`
	ActorKind          string    `json:"actorKind"`
	Actor              string    `json:"actor"`
	RecordedAt         time.Time `json:"recordedAt"`
}

func utc(t time.Time) bool          { _, offset := t.Zone(); return !t.IsZero() && offset == 0 }
func successorDigest(s string) bool { return validDigest(s) && s == strings.ToLower(s) }
func boundedIdentity(s string) bool { return validTarget(s) && len(s) <= 512 }

// ValidateSuccessorGraph validates domain state without serializing. References
// are digests; resolving authority and canonical records belongs to consumers.
func ValidateSuccessorGraph(g SuccessorGraph) error {
	if g.FormatVersion != SuccessorGraphVersion && g.FormatVersion != OperationalGraphVersion || g.ClassificationPolicy != ClassificationPolicyV1 || !successorDigest(g.DispatchAuthorityRef) || g.StageConcurrency == 0 || g.StageConcurrency > MaxStageConcurrency || !utc(g.Parent.CreatedAt) {
		return ErrInvalidGraph
	}
	// Reuse immutable Graph2/3 envelope/topology checks, without converting state.
	base := Graph{FormatVersion: g.projectionVersion(), Parent: g.Parent}
	for _, c := range g.Children {
		base.Children = append(base.Children, ChildExecution{ExecutionID: c.ExecutionID, ParentID: c.ParentID, GraphRevision: c.GraphRevision, NodeKey: c.NodeKey, Envelope: c.Envelope})
	}
	if !ValidGraph(base) {
		return ErrInvalidGraph
	}
	if r := g.CancelRequest; r != nil {
		if !validOpaqueID(r.RequestID) || r.ParentID != g.Parent.ExecutionID || r.GraphRevision != g.Parent.GraphRevision || r.ActorKind != "human-operator" || !boundedIdentity(r.Actor) || !utc(r.RequestedAt) || r.RequestedAt.Before(g.Parent.CreatedAt) || !successorDigest(r.AuthorityRef) {
			return ErrInvalidGraph
		}
	}
	ids := map[string]bool{g.Parent.ExecutionID: true}
	for _, c := range g.Children {
		ids[c.ExecutionID] = true
	}
	facts := map[string]bool{}
	for _, c := range g.Children {
		if err := validateAttemptSequence(c.Attempts, attemptScope{diagnostics: g.FormatVersion == OperationalGraphVersion, created: g.Parent.CreatedAt, cancel: g.CancelRequest, parentID: g.Parent.ExecutionID, childID: c.ExecutionID, revision: g.Parent.GraphRevision, maximum: c.Envelope.Controls.MaximumAttempts, integration: c.Envelope.IntegrationOwner, authority: map[string]bool{g.DispatchAuthorityRef: true}}, ids, facts); err != nil {
			return err
		}
	}
	for id := range facts {
		if ids[id] {
			return ErrInvalidGraph
		}
	}
	if g.CancelRequest != nil && (ids[g.CancelRequest.RequestID] || facts[g.CancelRequest.RequestID]) {
		return ErrInvalidGraph
	}
	return nil
}

type attemptScope struct {
	diagnostics       bool
	created           time.Time
	cancel            *CancelRequest
	parentID, childID string
	revision          uint64
	maximum           uint32
	integration       bool
	authority         map[string]bool
}

func validateAttemptSequence(attempts []SuccessorAttempt, scope attemptScope, ids, facts map[string]bool) error {
	if len(attempts) > int(scope.maximum) {
		return ErrInvalidGraph
	}
	var epoch uint64
	var previousFinish *time.Time
	for i, a := range attempts {
		if !validOpaqueID(a.AttemptID) || ids[a.AttemptID] || a.Number != uint32(i+1) || a.LeaseEpoch == 0 || a.LeaseEpoch < epoch || !successorDigest(a.PacketDigest) || a.StartedAt == nil || !utc(*a.StartedAt) || a.StartedAt.Before(scope.created) || len(a.ReconciliationFacts) > MaxReconciliationFacts {
			return ErrInvalidGraph
		}
		if previousFinish != nil && a.StartedAt.Before(*previousFinish) || a.AmbiguousEffect && a.Status != AttemptUnknown || a.Diagnostic != nil && (!scope.diagnostics || validateDiagnostic(a) != nil) {
			return ErrInvalidGraph
		}
		if a.CancellationSeen && scope.cancel == nil {
			return ErrInvalidGraph
		}
		if a.ClassificationReason == CancelConfirmed && (scope.cancel == nil || a.FinishedAt == nil || a.FinishedAt.Before(scope.cancel.RequestedAt)) {
			return ErrInvalidGraph
		}
		previousFinish = a.FinishedAt
		epoch = a.LeaseEpoch
		ids[a.AttemptID] = true
		if a.OutputDigest != "" && !successorDigest(a.OutputDigest) || a.ResultReference != "" && !successorDigest(a.ResultReference) {
			return ErrInvalidGraph
		}
		if a.Process != nil && (a.Process.PID <= 0 || !boundedIdentity(a.Process.StartIdentity) || !boundedIdentity(a.Process.GroupIdentity)) {
			return ErrInvalidGraph
		}
		if a.Status == AttemptRunning {
			if a.FinishedAt != nil || a.ClassificationReason != "" || a.QualifyingRecordRef != "" || a.Process == nil || i != len(attempts)-1 {
				return ErrInvalidGraph
			}
		} else {
			if a.FinishedAt == nil || !utc(*a.FinishedAt) || a.FinishedAt.Before(*a.StartedAt) || !validClassification(a, scope.integration) {
				return ErrInvalidGraph
			}
		}
		last := *a.StartedAt
		if a.FinishedAt != nil {
			last = *a.FinishedAt
		}
		for _, f := range a.ReconciliationFacts {
			if !validOpaqueID(f.FactID) || facts[f.FactID] || ids[f.FactID] || f.ParentID != scope.parentID || f.ChildID != scope.childID || f.GraphRevision != scope.revision || f.AttemptID != a.AttemptID || !successorDigest(f.ObservationsDigest) || !successorDigest(f.AuthorityRef) || scope.authority[f.AuthorityRef] || f.ActorKind != "human-operator" || !boundedIdentity(f.Actor) || !utc(f.RecordedAt) || f.RecordedAt.Before(last) || (f.Classification != "retry_safe" && f.Classification != "recovery_required") {
				return ErrInvalidGraph
			}
			facts[f.FactID] = true
			last = f.RecordedAt
		}
	}
	return nil
}

func validClassification(a SuccessorAttempt, integration bool) bool {
	if a.AmbiguousEffect && a.Status != AttemptUnknown {
		return false
	}
	if a.QualifyingRecordRef != "" && !successorDigest(a.QualifyingRecordRef) {
		return false
	}
	if integration {
		if a.QualifyingRecordRef != "" {
			return false
		}
		return a.Status == AttemptSucceeded && a.ClassificationReason == IntegrationApplied && a.ExitCode == 0 || a.Status == AttemptFailed && a.ClassificationReason == IntegrationFailed || a.Status == AttemptUnknown && a.ClassificationReason == IntegrationUnknown
	}
	record := a.ClassificationReason == ResultRecorded || a.ClassificationReason == BlockerPublished || a.ClassificationReason == ResultNotSuccessful
	if record != (a.QualifyingRecordRef != "") {
		return false
	}
	switch a.ClassificationReason {
	case ResultRecorded:
		return a.Status == AttemptSucceeded && a.ExitCode == 0
	case BlockerPublished:
		return a.Status == AttemptBlocked
	case ResultNotSuccessful, ResultMissing:
		return a.Status == AttemptFailed && a.ExitCode == 0
	case ProcessFailed:
		return a.Status == AttemptFailed && a.ExitCode != 0
	case TimeoutStopped:
		return a.Status == AttemptFailed
	case CancelConfirmed:
		return a.Status == AttemptCancelled && a.CancellationSeen
	case ProcessOutcomeUnknown:
		return a.Status == AttemptUnknown
	default:
		return false
	}
}

// ValidateSuccessorRecords resolves qualifying references against accepted
// canonical projections. It validates persisted reasons, never reclassifies.
func ValidateSuccessorRecords(g SuccessorGraph, records []AttemptRecord) error {
	if err := ValidateSuccessorGraph(g); err != nil {
		return err
	}
	for _, c := range g.Children {
		for _, a := range c.Attempts {
			if a.QualifyingRecordRef == "" {
				continue
			}
			found := false
			for _, r := range records {
				if !r.Accepted || r.Revision == 0 || r.Digest != a.QualifyingRecordRef || r.ParentID != g.Parent.ExecutionID || r.ChildID != c.ExecutionID || r.GraphRevision != g.Parent.GraphRevision || r.AttemptID != a.AttemptID {
					continue
				}
				found = a.ClassificationReason == BlockerPublished && r.Kind == "blocker" || a.ClassificationReason == ResultRecorded && r.Kind == "result" && r.Outcome == "succeeded" || a.ClassificationReason == ResultNotSuccessful && r.Kind == "result" && r.Outcome == "failed"
				if found {
					break
				}
			}
			if !found {
				return ErrInvalidGraph
			}
		}
	}
	return nil
}

// ValidateSuccessorUpdate protects frozen publication fields, terminal attempts
// and fact prefixes. Contradictory new facts remain visible rather than replaced.
func ValidateSuccessorUpdate(old, next SuccessorGraph) error {
	if ValidateSuccessorGraph(old) != nil || ValidateSuccessorGraph(next) != nil {
		return ErrInvalidGraph
	}
	oldHeader, nextHeader := old, next
	oldHeader.Children, nextHeader.Children = nil, nil
	oldHeader.CancelRequest, nextHeader.CancelRequest = nil, nil
	oldHeader.StorageRevision, nextHeader.StorageRevision = [32]byte{}, [32]byte{}
	if !reflect.DeepEqual(oldHeader, nextHeader) || len(old.Children) != len(next.Children) || old.CancelRequest != nil && !reflect.DeepEqual(old.CancelRequest, next.CancelRequest) {
		return ErrInvalidGraph
	}
	for i, c := range old.Children {
		n := next.Children[i]
		ca, na := c.Attempts, n.Attempts
		c.Attempts, n.Attempts = nil, nil
		if !reflect.DeepEqual(c, n) || len(na) < len(ca) {
			return ErrInvalidGraph
		}
		for j, a := range ca {
			b := na[j]
			af, bf := a.ReconciliationFacts, b.ReconciliationFacts
			a.ReconciliationFacts, b.ReconciliationFacts = nil, nil
			if len(bf) < len(af) {
				return ErrInvalidGraph
			}
			for k := range af {
				if !reflect.DeepEqual(af[k], bf[k]) {
					return ErrInvalidGraph
				}
			}
			if a.Status != AttemptRunning || b.Status == AttemptRunning {
				if !reflect.DeepEqual(a, b) {
					return ErrInvalidGraph
				}
			} else {
				if a.AttemptID != b.AttemptID || a.Number != b.Number || a.LeaseEpoch != b.LeaseEpoch || a.PacketDigest != b.PacketDigest || !reflect.DeepEqual(a.StartedAt, b.StartedAt) || !reflect.DeepEqual(a.Process, b.Process) {
					return ErrInvalidGraph
				}
			}
		}
	}
	return nil
}

func EncodeSuccessorGraph(g SuccessorGraph) ([]byte, error) {
	if ValidateSuccessorGraph(g) != nil {
		return nil, ErrInvalidGraph
	}
	wire, err := json.Marshal(g)
	if err != nil || len(wire)+1 > MaxGraphBytes {
		return nil, ErrInvalidGraph
	}
	return append(wire, '\n'), nil
}
func DecodeSuccessorGraph(wire []byte) (SuccessorGraph, error) {
	var g SuccessorGraph
	if strictjson.Decode(wire, &g, MaxGraphBytes) != nil || ValidateSuccessorGraph(g) != nil {
		return SuccessorGraph{}, ErrInvalidGraph
	}
	g.StorageRevision = sha256.Sum256(wire)
	return g, nil
}

// GraphDocument dispatches inspection by version without historical conversion.
// Exactly one pointer is populated. Mutation callers require Successor.
type GraphDocument struct {
	Historical *Graph
	Successor  *SuccessorGraph
}

func DecodeGraphDocument(wire []byte) (GraphDocument, error) {
	var raw map[string]json.RawMessage
	if strictjson.Decode(wire, &raw, MaxGraphBytes) != nil {
		return GraphDocument{}, ErrInvalidGraph
	}
	var version int
	if json.Unmarshal(raw["formatVersion"], &version) != nil {
		return GraphDocument{}, ErrInvalidGraph
	}
	switch version {
	case 1, 2:
		g, err := DecodeGraph(wire)
		if err != nil {
			return GraphDocument{}, err
		}
		return GraphDocument{Historical: &g}, nil
	case SuccessorGraphVersion, OperationalGraphVersion:
		g, err := DecodeSuccessorGraph(wire)
		if err != nil {
			return GraphDocument{}, err
		}
		return GraphDocument{Successor: &g}, nil
	default:
		return GraphDocument{}, ErrInvalidGraph
	}
}
func EncodeGraphDocument(d GraphDocument) ([]byte, error) {
	if d.Historical != nil && d.Successor == nil {
		return EncodeGraph(*d.Historical)
	}
	if d.Successor != nil && d.Historical == nil {
		return EncodeSuccessorGraph(*d.Successor)
	}
	return nil, ErrInvalidGraph
}

// projectionVersion is the in-memory Graph generation whose envelope profile
// rules match this ledger: Graph 2 for successor 3, projection-only 3 for 4.
func (g SuccessorGraph) projectionVersion() int {
	if g.FormatVersion == OperationalGraphVersion {
		return OperationalProposalVersion
	}
	return 2
}
