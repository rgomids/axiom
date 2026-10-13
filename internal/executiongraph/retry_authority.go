package executiongraph

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/rgomids/axiom/internal/strictjson"
)

const MaxRetryAuthorityBytes = 64 << 10

// Workspace dispositions after effects (PD-8). A retry never resets a
// workspace implicitly: "preserve" keeps it, "reset" discards it, and
// "not_required" is valid only when no prior attempt could have written.
const (
	DispositionNotRequired = "not_required"
	DispositionPreserve    = "preserve"
	DispositionReset       = "reset"
)

// RetryAuthority is a separate, single-use authority for one new attempt of
// one child. It binds the exact graph storage revision observed at preview,
// so any later write (including the retry attempt itself) consumes it. First
// attempt authority can never be reused for a retry, nor this for integration.
type RetryAuthority struct {
	FormatVersion  int       `json:"formatVersion"`
	ProjectID      string    `json:"projectId"`
	ParentID       string    `json:"parentId"`
	Revision       uint64    `json:"revision"`
	GraphDigest    string    `json:"graphDigest"`
	PreviewDigest  string    `json:"previewDigest"`
	DispatchDigest string    `json:"dispatchDigest"`
	ChildID        string    `json:"childId"`
	PriorAttemptID string    `json:"priorAttemptId"`
	AttemptNumber  uint32    `json:"attemptNumber"`
	Effects        []Effect  `json:"effects"`
	Disposition    string    `json:"disposition"`
	ActorKind      string    `json:"actorKind"`
	Actor          string    `json:"actor"`
	RecordedAt     time.Time `json:"recordedAt"`
}

func ValidateRetryAuthority(a RetryAuthority) error {
	if a.FormatVersion != 1 || !validToken(a.ProjectID) || !validOpaqueID(a.ParentID) || a.Revision == 0 || !successorDigest(a.GraphDigest) || !successorDigest(a.PreviewDigest) || !successorDigest(a.DispatchDigest) || !validOpaqueID(a.ChildID) || !validOpaqueID(a.PriorAttemptID) || a.AttemptNumber < 2 || !validDispatchEffects(a.Effects) || a.ActorKind != "human-operator" || !boundedIdentity(a.Actor) || !utc(a.RecordedAt) {
		return ErrInvalidGraph
	}
	switch a.Disposition {
	case DispositionNotRequired, DispositionPreserve, DispositionReset:
		return nil
	}
	return ErrInvalidGraph
}

func EncodeRetryAuthority(a RetryAuthority) ([]byte, error) {
	if ValidateRetryAuthority(a) != nil {
		return nil, ErrInvalidGraph
	}
	b, err := json.Marshal(a)
	if err != nil || len(b)+1 > MaxRetryAuthorityBytes {
		return nil, ErrInvalidGraph
	}
	return append(b, '\n'), nil
}
func DecodeRetryAuthority(b []byte) (RetryAuthority, error) {
	var a RetryAuthority
	if strictjson.Decode(b, &a, MaxRetryAuthorityBytes) != nil || ValidateRetryAuthority(a) != nil {
		return RetryAuthority{}, ErrInvalidGraph
	}
	return a, nil
}
func RetryAuthorityDigest(a RetryAuthority) (string, error) {
	b, err := EncodeRetryAuthority(a)
	if err != nil {
		return "", err
	}
	d := sha256.Sum256(b)
	return hex.EncodeToString(d[:]), nil
}

// RetryDispositionRequired reports whether a prior attempt could have changed
// the workspace: a repository-write child, or any ambiguous effect.
func RetryDispositionRequired(c SuccessorChild) bool {
	if len(c.Attempts) == 0 {
		return false
	}
	for _, a := range c.Attempts {
		if a.AmbiguousEffect {
			return true
		}
	}
	for _, e := range c.Envelope.AllowedEffects {
		if e.Kind == "repository-write" {
			return true
		}
	}
	return false
}

// CheckRetryDispatch is called immediately before starting a retry attempt.
// It returns the attempt number the authority covers.
func CheckRetryDispatch(g SuccessorGraph, a RetryAuthority) (uint32, error) {
	if ValidateRetryAuthority(a) != nil || ValidateSuccessorGraph(g) != nil || g.StorageRevision == ([32]byte{}) || fmt.Sprintf("%x", g.StorageRevision) != a.GraphDigest || a.ParentID != g.Parent.ExecutionID || a.Revision != g.Parent.GraphRevision || a.ProjectID != g.Children[0].Envelope.Scope.ProjectID || g.CancelRequest != nil {
		return 0, ErrInvalidGraph
	}
	for _, c := range g.Children {
		if c.ExecutionID != a.ChildID {
			continue
		}
		if c.Envelope.IntegrationOwner || len(c.Attempts) == 0 || c.Attempts[len(c.Attempts)-1].AttemptID != a.PriorAttemptID || uint32(len(c.Attempts))+1 != a.AttemptNumber || !sameEffects(a.Effects, c.Envelope.AllowedEffects) {
			return 0, ErrInvalidGraph
		}
		if RetryDispositionRequired(c) == (a.Disposition == DispositionNotRequired) {
			return 0, ErrInvalidGraph
		}
		eligible, err := SuccessorRetryEligible(g, c.ExecutionID, true)
		if err != nil || !eligible {
			return 0, ErrInvalidGraph
		}
		return a.AttemptNumber, nil
	}
	return 0, ErrInvalidGraph
}
