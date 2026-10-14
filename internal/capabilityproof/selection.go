package capabilityproof

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// Selection outcomes wrap ErrBlocked so existing callers keep failing closed.
var (
	ErrEvidenceMissing   = fmt.Errorf("%w: no matching capability evidence", ErrBlocked)
	ErrEvidenceAmbiguous = fmt.Errorf("%w: ambiguous capability evidence", ErrBlocked)
)

// StoredEvidence is one persisted record and the content address it was read under.
type StoredEvidence struct {
	Ref      string
	Evidence Evidence
}

// EvidenceScanner enumerates every persisted record read-only. Any unreadable,
// unsafe or unverifiable entry must abort the scan with an error, never be
// skipped, so a partial view cannot produce a false unique selection.
type EvidenceScanner interface {
	Scan(context.Context, func(StoredEvidence) error) error
}

// Selector finds the single fresh, valid Evidence whose complete Binding and
// ModelBinding equal an independently observed current identity. It never
// writes, invalidates, probes or derives identity from the records it reads.
type Selector struct {
	Evidence EvidenceScanner
	Now      func() time.Time
}

// Select returns the content-addressed reference of the unique admissible
// record. Zero or several matches fail closed; no timestamp, name or digest
// order is ever used to pick one. The blocker follows Check's vocabulary.
func (s Selector) Select(ctx context.Context, b Binding, m *ModelBinding) (string, string, error) {
	if err := ctx.Err(); err != nil {
		return "", "capability_unknown", err
	}
	if s.Evidence == nil || !ValidBinding(b) {
		return "", "capability_unknown", ErrBlocked
	}
	now := time.Now()
	if s.Now != nil {
		now = s.Now()
	}
	var match string
	matches := 0
	seen := map[string]bool{}
	err := s.Evidence.Scan(ctx, func(c StoredEvidence) error {
		if seen[c.Ref] {
			return ErrBlocked
		}
		seen[c.Ref] = true
		original := c.Evidence
		original.InvalidatedReason = ""
		if ref, err := Digest(original); err != nil || ref != c.Ref {
			return ErrInvalidEvidence
		}
		if admissible(c.Evidence, b, m, now) {
			match = c.Ref
			matches++
		}
		return nil
	})
	switch {
	// Cancellation keeps its identity, but only as the bare sentinel so a
	// wrapping scanner error never exposes storage details.
	case errors.Is(err, context.Canceled):
		return "", "capability_unknown", context.Canceled
	case errors.Is(err, context.DeadlineExceeded):
		return "", "capability_unknown", context.DeadlineExceeded
	case err != nil:
		return "", "capability_unknown", ErrBlocked
	case matches == 0:
		return "", "capability_unknown", ErrEvidenceMissing
	case matches > 1:
		return "", "capability_unknown", ErrEvidenceAmbiguous
	}
	return match, "", nil
}

func admissible(e Evidence, b Binding, m *ModelBinding, now time.Time) bool {
	if !ValidEvidence(e) || e.InvalidatedReason != "" || e.Binding != b {
		return false
	}
	if (e.Model == nil) != (m == nil) || e.Model != nil && *e.Model != *m {
		return false
	}
	return !now.Before(e.ObservedAt) && now.Sub(e.ObservedAt) <= MaxAge
}
