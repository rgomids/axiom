// Package local contains the in-memory installation record codec. Filesystem
// access, persistence, discovery and observation execution are later tasks.
package local

import (
	"slices"

	"github.com/rgomids/axiom/internal/projectapp"
)

// RecordState uses existing consumer-owned metadata contracts. SourceLocation is
// data, never a Destination capability. LocalRevision is an opaque store-supplied
// revision label; it is not the exact-byte CAS revision returned by the reader.
// The codec neither allocates nor increments revisions.
type RecordState struct {
	ProjectID, ObservedSlug, SourceLocation, LocalRevision string
	PortableRevision                                       projectapp.PortableRevision
	ArtifactDigests                                        []projectapp.ArtifactDigest
	Repositories                                           []projectapp.RepositoryBinding
	Credentials                                            []projectapp.CredentialBinding
	Runtime                                                projectapp.RuntimeBinding
	Attempt                                                projectapp.AttemptMetadata
}

// Record is sealed after complete validation. Its zero value is unusable.
// It contains metadata only; portable declarations and document bodies have no slot.
type Record struct {
	state RecordState
	valid bool
}

func clone(s RecordState) RecordState {
	s.ArtifactDigests = slices.Clone(s.ArtifactDigests)
	s.Repositories = slices.Clone(s.Repositories)
	s.Credentials = slices.Clone(s.Credentials)
	return s
}
func (r Record) State() RecordState { return clone(r.state) }

func NewRecord(s RecordState) (Record, []Issue) {
	s = clone(s)
	if issues := validateState(s); len(issues) > 0 {
		return Record{}, issues
	}
	return Record{state: s, valid: true}, nil
}

// DecodeObservedRecord consumes an explicit read observation, not a filesystem
// error. A future reader must classify not-found separately from all read errors.
// Even empty existing bytes are malformed. Errors never yield a usable record or
// a MissingLocalRevision, so a caller cannot mistake corruption for new install.
func DecodeObservedRecord(input []byte, exists bool) (Record, projectapp.LocalRevision, []Issue) {
	if !exists {
		if len(input) > 0 {
			return Record{}, projectapp.LocalRevision{}, problem("installation", "invalid_observation")
		}
		return Record{}, projectapp.MissingLocalRevision(), nil
	}
	r, issues := DecodeRecord(input)
	if len(issues) > 0 {
		return Record{}, projectapp.LocalRevision{}, issues
	}
	return r, projectapp.ObserveLocalRevision(input), nil
}

// Issue contains only fixed codes/schema paths and numeric collection indices.
// Raw parser errors and rejected keys or values are never returned.
type Issue struct{ Field, Code string }

func (i Issue) Category() string {
	if i.Code == "unsupported_local_format" {
		return i.Code
	}
	return "invalid_local_state"
}
func (i Issue) Severity() string         { return "error" }
func (i Issue) Remedy() string           { return "preserve_record_and_inspect_local_state" }
func problem(field, code string) []Issue { return []Issue{{field, code}} }
