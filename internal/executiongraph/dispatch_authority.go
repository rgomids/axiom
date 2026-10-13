package executiongraph

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/rgomids/axiom/internal/strictjson"
)

const MaxDispatchAuthorityBytes = 64 << 10

// DispatchAuthority is an immutable receipt for first attempts, never retry,
// cancellation, integration or reconciliation authority. Revisions are logical
// identities; no persisted field is interpreted as a host filesystem path.
type DispatchAuthority struct {
	FormatVersion      int                      `json:"formatVersion"`
	ProjectID          string                   `json:"projectId"`
	PreviewDigest      string                   `json:"previewDigest"`
	Scope              string                   `json:"scope"`
	ParentID           string                   `json:"parentId"`
	Revision           uint64                   `json:"revision"`
	ProposalDigest     string                   `json:"proposalDigest"`
	Children           []DispatchChildAuthority `json:"children"`
	SingleStageID      string                   `json:"singleStageId,omitempty"`
	SingleStageOrdinal int                      `json:"singleStageOrdinal,omitempty"`
	SingleAttemptID    string                   `json:"singleAttemptId,omitempty"`
	LedgerDigest       string                   `json:"ledgerDigest,omitempty"`
	ActorKind          string                   `json:"actorKind"`
	Actor              string                   `json:"actor"`
	RecordedAt         time.Time                `json:"recordedAt"`
}

type DispatchChildAuthority struct {
	ChildID string   `json:"childId"`
	Effects []Effect `json:"effects"`
}

func ValidateDispatchAuthority(a DispatchAuthority) error {
	if a.FormatVersion != 1 || !validToken(a.ProjectID) || !successorDigest(a.PreviewDigest) || !validOpaqueID(a.ParentID) || a.Revision == 0 || !successorDigest(a.ProposalDigest) || a.ActorKind != "human-operator" || !boundedIdentity(a.Actor) || !utc(a.RecordedAt) || len(a.Children) == 0 || len(a.Children) > maxListItems {
		return ErrInvalidGraph
	}
	if a.Scope == "graph-first-attempts" {
		if a.SingleAttemptID != "" || a.LedgerDigest != "" || a.SingleStageID != "" || a.SingleStageOrdinal != 0 {
			return ErrInvalidGraph
		}
	} else if a.Scope == "single-first-attempt" {
		if !validToken(a.SingleStageID) || a.SingleStageOrdinal < 0 || !validOpaqueID(a.SingleAttemptID) || !successorDigest(a.LedgerDigest) || len(a.Children) != 1 || a.Children[0].ChildID != a.ParentID {
			return ErrInvalidGraph
		}
	} else {
		return ErrInvalidGraph
	}
	seen := map[string]bool{}
	for _, c := range a.Children {
		if !validOpaqueID(c.ChildID) || seen[c.ChildID] || !validDispatchEffects(c.Effects) {
			return ErrInvalidGraph
		}
		seen[c.ChildID] = true
	}
	return nil
}

func EncodeDispatchAuthority(a DispatchAuthority) ([]byte, error) {
	if ValidateDispatchAuthority(a) != nil {
		return nil, ErrInvalidGraph
	}
	b, err := json.Marshal(a)
	if err != nil || len(b)+1 > MaxDispatchAuthorityBytes {
		return nil, ErrInvalidGraph
	}
	return append(b, '\n'), nil
}
func DecodeDispatchAuthority(b []byte) (DispatchAuthority, error) {
	var a DispatchAuthority
	if strictjson.Decode(b, &a, MaxDispatchAuthorityBytes) != nil || ValidateDispatchAuthority(a) != nil {
		return DispatchAuthority{}, ErrInvalidGraph
	}
	return a, nil
}
func DispatchAuthorityDigest(a DispatchAuthority) (string, error) {
	b, err := EncodeDispatchAuthority(a)
	if err != nil {
		return "", err
	}
	d := sha256.Sum256(b)
	return hex.EncodeToString(d[:]), nil
}

// ValidateGraphDispatchAuthority resolves the frozen reference and its exact
// revision/effect scope. This does not authorize an integration attempt.
func ValidateGraphDispatchAuthority(g SuccessorGraph, a DispatchAuthority, preview string) error {
	d, err := DispatchAuthorityDigest(a)
	if err != nil || ValidateSuccessorGraph(g) != nil || a.Scope != "graph-first-attempts" || a.PreviewDigest != preview || d != g.DispatchAuthorityRef || a.ParentID != g.Parent.ExecutionID || a.Revision != g.Parent.GraphRevision || a.ProposalDigest != g.Parent.ProposalDigest || a.ProjectID != g.Children[0].Envelope.Scope.ProjectID || len(a.Children) != len(g.Children) || a.RecordedAt.After(g.Parent.CreatedAt) {
		return ErrInvalidGraph
	}
	for _, c := range g.Children {
		found := false
		for _, bound := range a.Children {
			if bound.ChildID == c.ExecutionID && sameEffects(bound.Effects, c.Envelope.AllowedEffects) {
				found = true
			}
		}
		if !found || c.Envelope.AuthorityReference != d {
			return ErrInvalidGraph
		}
	}
	return nil
}

// CheckFirstDispatch is called immediately before starting a child. The first
// attempt receipt cannot be reused for a retry or an integration operation.
func CheckFirstDispatch(g SuccessorGraph, a DispatchAuthority, preview, childID string, number uint32) error {
	if ValidateGraphDispatchAuthority(g, a, preview) != nil || number != 1 || g.CancelRequest != nil {
		return ErrInvalidGraph
	}
	for _, c := range g.Children {
		if c.ExecutionID == childID && !c.Envelope.IntegrationOwner && len(c.Attempts) == 0 {
			return nil
		}
	}
	return ErrInvalidGraph
}

func validDispatchEffects(effects []Effect) bool {
	if len(effects) > maxListItems {
		return false
	}
	seen := map[string]bool{}
	for _, e := range effects {
		if (e.Kind != "read" && e.Kind != "repository-write" && e.Kind != "process" && e.Kind != "artifact-publish" && e.Kind != "integration") || !validTarget(e.Target) || seen[e.Kind+"\x00"+e.Target] {
			return false
		}
		seen[e.Kind+"\x00"+e.Target] = true
	}
	return true
}
