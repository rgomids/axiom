package projectapp

import (
	"context"
	"time"

	"github.com/rgomids/axiom/internal/project"
)

// ManifestCodec implementations must enforce the approved strict wire/security
// contract and return safe issues, never raw parser excerpts. Encode consumes a
// validated domain value; no patch encoding method exists.
type ManifestCodec interface {
	Decode([]byte) (project.Project, []Issue)
	Encode(project.Project) ([]byte, []Issue)
}
type IdentityAllocator interface{ NewID() (string, []Issue) }
type Clock interface{ Now() time.Time }

// Readers have no write capability. Missing targets are explicit revisions;
// malformed/unknown state is an issue, never silently interpreted as absent.
type PortableReader interface {
	ReadPortable(context.Context, Destination) (ArtifactSnapshot, PortableRevision, []Issue)
}
type LocalReader interface {
	ReadLocal(context.Context, project.Project) (LocalSnapshot, LocalRevision, []Issue)
}

// Writers accept only sealed complete requests, not raw paths, bytes or patches.
// An adapter MUST validate the permit immediately before commit, compare expected
// revisions under its concurrency protection, preserve prior state on pre-commit
// failure, and return actual commit status even on cancellation/error. Destination
// identity, confinement, namespace protection and revocation must remain protected
// through its commit point. T05/T06/T07 must prove these protocol obligations.
// Codecs/use cases must validate the full portable/local payload before submission;
// adapters must reject unsafe/unsupported content before any write. T02 snapshot
// construction alone does not prove wire security or physical artifact containment.
// ctx cancellation alone is never evidence that a commit did not happen.
type PortableWriter interface {
	CommitPortable(context.Context, AuthorizedPortable) MutationResult
}
type LocalWriter interface {
	CommitLocal(context.Context, AuthorizedLocal) MutationResult
}

// Presence observations cannot establish capability, authentication or readiness.
type Availability uint8

const (
	Unverified Availability = iota
	Available
	Unavailable
)

type ObservationBasis uint8

const (
	NotChecked ObservationBasis = iota
	PresentMetadata
	MissingMetadata
	UnsupportedCheck
	UncertainPermissions
	MismatchedMetadata
)

type Observation struct {
	Availability Availability
	Basis        ObservationBasis
	ObservedAt   time.Time
}
type CheckoutRequest struct{ RepositoryKey, ExplicitPath string }
type CheckoutFacts struct {
	Observation       Observation
	CanonicalIdentity string
	Remotes           []string
}
type CheckoutObserver interface {
	ObserveCheckout(context.Context, CheckoutRequest) (CheckoutFacts, []Issue)
}
type RuntimeRequest struct{ RuntimeID, ExplicitPath string }
type RuntimeObserver interface {
	ObserveRuntime(context.Context, RuntimeRequest) (Observation, []Issue)
}

// Credential metadata names references only; no port reads their values. Strings
// are untrusted metadata requiring T04/T14 validation, never diagnostic payloads.
type CredentialBinding struct{ ReferenceKey, SourceKind, ItemReference string }
type RepositoryBinding struct {
	RepositoryKey, ExplicitPath, CanonicalIdentity string
	Observation                                    Observation
}
type RuntimeBinding struct {
	RuntimeID, ExplicitPath string
	Observation             Observation
}
type AttemptMetadata struct {
	Correlation string
	At          time.Time
}

// LocalState is a complete in-memory record proposal. Wire format/version and
// safe metadata validation belong to T04. No document bodies or secret value bag.
type LocalState struct {
	Source           Destination
	PortableRevision PortableRevision
	ArtifactDigests  []ArtifactDigest
	Repositories     []RepositoryBinding
	Credentials      []CredentialBinding
	Runtime          RuntimeBinding
	Attempt          AttemptMetadata
}
type LocalSnapshot struct {
	id, slug string
	state    LocalState
	valid    bool
}

func NewLocalSnapshot(p project.Project, state LocalState) (LocalSnapshot, []Issue) {
	if !p.Equivalent(p) || !state.Source.valid() || !state.PortableRevision.valid || !state.PortableRevision.present {
		return LocalSnapshot{}, problem(InvalidSnapshot)
	}
	state = cloneLocal(state)
	return LocalSnapshot{p.State().ID, p.State().Slug, state, true}, nil
}
func cloneLocal(s LocalState) LocalState {
	s.ArtifactDigests = append([]ArtifactDigest(nil), s.ArtifactDigests...)
	s.Repositories = append([]RepositoryBinding(nil), s.Repositories...)
	s.Credentials = append([]CredentialBinding(nil), s.Credentials...)
	return s
}
func (s LocalSnapshot) State() LocalState { return cloneLocal(s.state) }
func (s LocalSnapshot) ProjectID() string { return s.id }
func (s LocalSnapshot) Slug() string      { return s.slug }

type InspectionStatus uint8

const (
	InspectionUnavailable InspectionStatus = iota
	InspectionCompleted
	InspectionFinding
	InspectionFailed
)

type Inspection struct {
	Status InspectionStatus
	Issues []Issue
}

// Inspection receives untrusted text only. It cannot rewrite, certify secret
// absence, resolve credentials or confer authority. Implementations must be local.
type TextInspector interface {
	InspectText(context.Context, []byte) Inspection
}

// InspectionWarnings keeps optional inspection failure independent from validity.
func InspectionWarnings(result Inspection) []Issue {
	code := ScannerUnavailable
	switch result.Status {
	case InspectionCompleted:
		return nil
	case InspectionFinding:
		code = ScannerFinding
	case InspectionFailed:
		code = ScannerFailed
	}
	return []Issue{{Phase: InspectionPhase, Field: TextField, Code: code}}
}
