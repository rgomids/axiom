// Package completion defines Axiom's canonical terminal operation result.
package completion

import (
	"errors"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/rgomids/axiom/internal/provenance"
)

const (
	maxReferenceCount = 16
	maxReferenceBytes = 256
	maxDetailsBytes   = 256
)

type Status string

const (
	Success           Status = "success"
	Failure           Status = "failure"
	ValidationFailure Status = "validation_failure"
	DeniedAuthority   Status = "denied_authority"
	Partial           Status = "partial"
	Interrupted       Status = "interrupted"
	RetryableFailure  Status = "retryable_failure"
)

var statuses = [...]Status{Success, Failure, ValidationFailure, DeniedAuthority, Partial, Interrupted, RetryableFailure}

func Statuses() []Status {
	result := make([]Status, len(statuses))
	copy(result, statuses[:])
	return result
}

func (s Status) Valid() bool {
	for _, candidate := range statuses {
		if s == candidate {
			return true
		}
	}
	return false
}

type Facts struct {
	Completed                bool
	RequestedEffectConfirmed bool
	SecondaryFailure         bool
	Failed                   bool
	ValidationFailed         bool
	AuthorityDenied          bool
	WasInterrupted           bool
	RetrySafeFailure         bool
}

func Classify(facts Facts) (Status, error) {
	if facts.RequestedEffectConfirmed && facts.SecondaryFailure && terminalCauseCount(facts) == 1 {
		return Partial, nil
	}
	if facts.SecondaryFailure || terminalCauseCount(facts) != 1 {
		return "", errors.New("ambiguous completion facts")
	}
	if facts.RequestedEffectConfirmed || facts.Completed {
		return Success, nil
	}
	if facts.ValidationFailed {
		return ValidationFailure, nil
	}
	if facts.AuthorityDenied {
		return DeniedAuthority, nil
	}
	if facts.WasInterrupted {
		return Interrupted, nil
	}
	if facts.RetrySafeFailure {
		return RetryableFailure, nil
	}
	return Failure, nil
}

func terminalCauseCount(facts Facts) int {
	count := 0
	for _, current := range []bool{facts.Completed, facts.RequestedEffectConfirmed, facts.Failed, facts.ValidationFailed, facts.AuthorityDenied, facts.WasInterrupted, facts.RetrySafeFailure} {
		if current {
			count++
		}
	}
	return count
}

type Result struct {
	status     Status
	result     provenance.Text
	references []string
	next       provenance.Text
	details    string
	provenance provenance.Value
}

func New(facts Facts, result provenance.Text, references []string, next provenance.Text, details string, source provenance.Value) (Result, error) {
	status, err := Classify(facts)
	if err != nil {
		return Result{}, err
	}
	if !source.Valid() {
		return Result{}, errors.New("invalid completion identity")
	}
	if _, err = provenance.RequireAxiomAuthored(result); err != nil {
		return Result{}, err
	}
	if !next.Empty() {
		if _, err := provenance.RequireAxiomAuthored(next); err != nil {
			return Result{}, err
		}
	}
	if err := validateReferences(references); err != nil {
		return Result{}, err
	}
	if details != "" && !validMetadata(details, maxDetailsBytes) {
		return Result{}, errors.New("invalid detail reference")
	}
	if (status == Partial || status == RetryableFailure) && next.Empty() {
		return Result{}, errors.New("incomplete result requires next action")
	}
	if status == Partial && len(references) == 0 {
		return Result{}, errors.New("partial result requires confirmed reference")
	}
	stableReferences := make([]string, len(references))
	copy(stableReferences, references)
	sort.Strings(stableReferences)
	return Result{status: status, result: result, references: stableReferences, next: next, details: details, provenance: source}, nil
}

func (r Result) Status() Status               { return r.status }
func (r Result) Result() provenance.Text      { return r.result }
func (r Result) Next() provenance.Text        { return r.next }
func (r Result) Details() string              { return r.details }
func (r Result) Provenance() provenance.Value { return r.provenance }
func (r Result) Valid() bool                  { return r.status.Valid() && r.provenance.Valid() && !r.result.Empty() }
func (r Result) References() []string {
	result := make([]string, len(r.references))
	copy(result, r.references)
	return result
}

func validateReferences(values []string) error {
	if len(values) > maxReferenceCount {
		return errors.New("too many completion references")
	}
	for _, value := range values {
		if !validMetadata(value, maxReferenceBytes) {
			return errors.New("invalid completion reference")
		}
	}
	return nil
}

func validMetadata(value string, limit int) bool {
	if value == "" || len(value) > limit || !utf8.ValidString(value) || strings.TrimSpace(value) != value {
		return false
	}
	for _, current := range value {
		if unicode.IsControl(current) {
			return false
		}
	}
	return true
}
