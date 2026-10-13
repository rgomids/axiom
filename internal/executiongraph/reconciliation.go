package executiongraph

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sort"
)

// AttemptHealth is an observation, never a replacement for persisted status.
type AttemptHealth struct {
	Stale           bool `json:"stale"`
	Orphan          bool `json:"orphan"`
	LivenessUnknown bool `json:"livenessUnknown"`
}

// ObserveAttemptHealth requires an exact PID/start/group match. PID reuse is
// not evidence that the recorded process remains alive. Observation errors
// remain unknown and cannot authorize a retry.
func ObserveAttemptHealth(a SuccessorAttempt, leaseHeld bool, observed *ProcessCorrelation, observationErr error) AttemptHealth {
	if a.Status != AttemptRunning || leaseHeld {
		return AttemptHealth{}
	}
	h := AttemptHealth{Stale: true}
	if observationErr != nil || a.Process == nil {
		h.LivenessUnknown = true
		return h
	}
	h.Orphan = observed != nil && *observed == *a.Process
	return h
}

// ReconciliationObservations contains bounded, digest-only projections gathered
// at preview and rechecked under the run lease at apply. Unknown observations
// fail closed; quarantined records are diagnostics, never result qualification.
type ReconciliationObservations struct {
	ParentID               string `json:"parentId"`
	ChildID                string `json:"childId"`
	GraphRevision          uint64 `json:"graphRevision"`
	AttemptID              string `json:"attemptId"`
	LeaseEpoch             uint64 `json:"leaseEpoch"`
	LeaseHeld              bool   `json:"leaseHeld"`
	ProcessAlive           bool   `json:"processAlive"`
	ProcessUnknown         bool   `json:"processUnknown"`
	WorktreeDigest         string `json:"worktreeDigest"`
	CanonicalRecordsDigest string `json:"canonicalRecordsDigest"`
	QuarantinedInboxDigest string `json:"quarantinedInboxDigest"`
}

func (o ReconciliationObservations) Digest() (string, error) {
	if !validOpaqueID(o.ParentID) || !validOpaqueID(o.ChildID) || !validOpaqueID(o.AttemptID) || o.GraphRevision == 0 || o.LeaseEpoch == 0 || !successorDigest(o.WorktreeDigest) || !successorDigest(o.CanonicalRecordsDigest) || !successorDigest(o.QuarantinedInboxDigest) {
		return "", ErrInvalidGraph
	}
	b, err := json.Marshal(o)
	if err != nil {
		return "", err
	}
	d := sha256.Sum256(b)
	return hex.EncodeToString(d[:]), nil
}

// AppendReconciliationFact returns independent state, preserving every prior
// fact and all attempt fields. Authority must be supplied by the authorized
// reconciliation operation; a fact never grants effect authority.
func AppendReconciliationFact(g SuccessorGraph, f ReconciliationFact, o ReconciliationObservations) (SuccessorGraph, error) {
	if ValidateSuccessorGraph(g) != nil {
		return g, ErrInvalidGraph
	}
	digest, err := o.Digest()
	if err != nil || digest != f.ObservationsDigest || f.ParentID != o.ParentID || f.ChildID != o.ChildID || f.GraphRevision != o.GraphRevision || f.AttemptID != o.AttemptID || f.ParentID != g.Parent.ExecutionID || f.GraphRevision != g.Parent.GraphRevision || o.LeaseHeld {
		return g, ErrInvalidGraph
	}
	if f.Classification == "retry_safe" && (o.ProcessAlive || o.ProcessUnknown) {
		return g, ErrInvalidGraph
	}
	next := g
	next.Children = append([]SuccessorChild(nil), g.Children...)
	for i, c := range next.Children {
		if c.ExecutionID != f.ChildID {
			continue
		}
		next.Children[i].Attempts = append([]SuccessorAttempt(nil), c.Attempts...)
		for j, a := range c.Attempts {
			if a.AttemptID != f.AttemptID {
				continue
			}
			if o.LeaseEpoch != a.LeaseEpoch {
				return g, ErrInvalidGraph
			}
			next.Children[i].Attempts[j].ReconciliationFacts = append(append([]ReconciliationFact(nil), a.ReconciliationFacts...), f)
			if ValidateSuccessorUpdate(g, next) != nil {
				return g, ErrInvalidGraph
			}
			return next, nil
		}
	}
	return g, ErrInvalidGraph
}

// ReconciliationDisposition surfaces conflicts and retains duplicates. No
// last-writer-wins rule can turn recovery_required into retry_safe.
func ReconciliationDisposition(a SuccessorAttempt) (retrySafe, contradictory bool) {
	recovery := false
	for _, f := range a.ReconciliationFacts {
		retrySafe = retrySafe || f.Classification == "retry_safe"
		recovery = recovery || f.Classification == "recovery_required"
	}
	return retrySafe && !recovery, retrySafe && recovery
}

// SuccessorRetryEligible is shared by scheduler and integration attempt reads.
// Explicit retry authority is still required; reconciliation never makes an
// attempt successful or unlocks dependent children.
func SuccessorRetryEligible(g SuccessorGraph, childID string, explicitRetry bool) (bool, error) {
	if ValidateSuccessorGraph(g) != nil {
		return false, ErrInvalidGraph
	}
	for _, c := range g.Children {
		if c.ExecutionID != childID {
			continue
		}
		if g.CancelRequest != nil || !explicitRetry || len(c.Attempts) == 0 || uint32(len(c.Attempts)) >= c.Envelope.Controls.MaximumAttempts {
			return false, nil
		}
		a := c.Attempts[len(c.Attempts)-1]
		safe, conflict := ReconciliationDisposition(a)
		if conflict {
			return false, nil
		}
		for _, f := range a.ReconciliationFacts {
			if f.Classification == "recovery_required" {
				return false, nil
			}
		}
		switch a.Status {
		case AttemptUnknown:
			return safe, nil
		case AttemptRunning:
			return false, nil // running attempts are never redispatched
		case AttemptFailed, AttemptBlocked, AttemptCancelled:
			return !a.AmbiguousEffect || safe, nil
		default:
			return false, nil
		}
	}
	return false, errors.New("child not found")
}

// ValidateLeaseEpoch fences writes to the currently held epoch. Previous
// attempts retain their immutable epochs; a lower epoch cannot dispatch/save.
func ValidateLeaseEpoch(g SuccessorGraph, epoch uint64) error {
	if epoch == 0 || ValidateSuccessorGraph(g) != nil {
		return ErrInvalidGraph
	}
	for _, c := range g.Children {
		for _, a := range c.Attempts {
			if a.LeaseEpoch > epoch {
				return ErrInvalidGraph
			}
		}
	}
	return nil
}

// SuccessorIntegrationAttemptAvailable consumes the same validated facts as
// the scheduler. Facts never qualify a child result for integration.
func SuccessorIntegrationAttemptAvailable(g SuccessorGraph) (bool, error) {
	if ValidateSuccessorGraph(g) != nil {
		return false, ErrInvalidGraph
	}
	if g.CancelRequest != nil {
		return false, nil
	}
	for _, c := range g.Children {
		if !c.Envelope.IntegrationOwner {
			continue
		}
		if len(c.Attempts) == 0 {
			return true, nil
		}
		return SuccessorRetryEligible(g, c.ExecutionID, true)
	}
	return false, ErrInvalidGraph
}

// ReconciliationSources is the bounded observation boundary. Callers supply
// deterministic worktree and inbox snapshots; canonical records are gathered
// independently of quarantined inbox content. No observer invokes inference.
type ReconciliationSources struct {
	LeaseProbe       func() (bool, error)
	ProcessProbe     func(ProcessCorrelation) (*ProcessCorrelation, error)
	Worktree         func() (string, error)
	CanonicalRecords func() (string, error)
	QuarantinedInbox func() (string, error)
}

func GatherReconciliationObservations(g SuccessorGraph, childID, attemptID string, s ReconciliationSources) (ReconciliationObservations, error) {
	if ValidateSuccessorGraph(g) != nil || s.LeaseProbe == nil || s.ProcessProbe == nil || s.Worktree == nil || s.CanonicalRecords == nil || s.QuarantinedInbox == nil {
		return ReconciliationObservations{}, ErrInvalidGraph
	}
	var attempt *SuccessorAttempt
	for _, c := range g.Children {
		if c.ExecutionID == childID {
			for _, a := range c.Attempts {
				if a.AttemptID == attemptID {
					copy := a
					attempt = &copy
				}
			}
		}
	}
	if attempt == nil {
		return ReconciliationObservations{}, ErrInvalidGraph
	}
	o := ReconciliationObservations{ParentID: g.Parent.ExecutionID, ChildID: childID, GraphRevision: g.Parent.GraphRevision, AttemptID: attemptID, LeaseEpoch: attempt.LeaseEpoch}
	var err error
	o.LeaseHeld, err = s.LeaseProbe()
	if err != nil {
		return o, err
	}
	if attempt.Process == nil {
		o.ProcessUnknown = true
	} else {
		observed, err := s.ProcessProbe(*attempt.Process)
		o.ProcessUnknown = err != nil
		o.ProcessAlive = observed != nil && *observed == *attempt.Process
	}
	o.WorktreeDigest, err = s.Worktree()
	if err != nil {
		return o, err
	}
	o.CanonicalRecordsDigest, err = s.CanonicalRecords()
	if err != nil {
		return o, err
	}
	o.QuarantinedInboxDigest, err = s.QuarantinedInbox()
	if err != nil {
		return o, err
	}
	_, err = o.Digest()
	return o, err
}

// ReadySuccessorChildren reads only validated persisted attempts/facts. It
// supplies the successor scheduler read path without redispatching running
// attempts, even when a stale attempt has diagnostic reconciliation facts.
func ReadySuccessorChildren(g SuccessorGraph, retries map[string]bool) ([]string, map[string]string, error) {
	if ValidateSuccessorGraph(g) != nil {
		return nil, nil, ErrInvalidGraph
	}
	ready := []string{}
	blocked := map[string]string{}
	if g.CancelRequest != nil {
		for _, c := range g.Children {
			blocked[c.ExecutionID] = "cancel_requested"
		}
		return ready, blocked, nil
	}
	succeeded := map[string]bool{}
	for _, c := range g.Children {
		if len(c.Attempts) > 0 && c.Attempts[len(c.Attempts)-1].Status == AttemptSucceeded {
			succeeded[c.ExecutionID] = true
		}
	}
	for _, c := range g.Children {
		if c.Envelope.IntegrationOwner {
			blocked[c.ExecutionID] = "integration_pending"
			continue
		}
		if succeeded[c.ExecutionID] {
			continue
		}
		if len(c.Attempts) > 0 && c.Attempts[len(c.Attempts)-1].Status == AttemptRunning {
			blocked[c.ExecutionID] = "attempt_running"
			continue
		}
		dependenciesReady := true
		for _, id := range c.Envelope.Dependencies {
			if !succeeded[id] {
				dependenciesReady = false
			}
		}
		if !dependenciesReady {
			blocked[c.ExecutionID] = "dependencies_unsatisfied"
			continue
		}
		if len(c.Attempts) > 0 {
			eligible, err := SuccessorRetryEligible(g, c.ExecutionID, retries[c.ExecutionID])
			if err != nil {
				return nil, nil, err
			}
			if !eligible {
				blocked[c.ExecutionID] = "retry_not_eligible"
				continue
			}
		}
		ready = append(ready, c.ExecutionID)
	}
	sort.Strings(ready)
	return ready, blocked, nil
}
