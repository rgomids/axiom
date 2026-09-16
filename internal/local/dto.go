package local

// These closed DTOs contain only Plan section 3 metadata. Required collections
// use arrays (including []); there is no local unconfigured declaration union.
// Runtime/attempt absence maps to the existing T02 zero metadata values.
type recordDTO struct {
	FormatVersion    int             `json:"formatVersion"`
	ProjectID        string          `json:"projectId"`
	ObservedSlug     string          `json:"observedSlug"`
	SourceLocation   string          `json:"sourceLocation"`
	PortableRevision string          `json:"portableRevision"`
	ArtifactDigests  []digestDTO     `json:"artifactDigests"`
	Repositories     []repositoryDTO `json:"repositories"`
	Credentials      []credentialDTO `json:"credentials"`
	Runtime          *runtimeDTO     `json:"runtime,omitempty"`
	Attempt          *attemptDTO     `json:"attempt,omitempty"`
}
type digestDTO struct {
	Name   string `json:"name"`
	Digest string `json:"digest"`
}
type observationDTO struct {
	Availability string `json:"availability"`
	Basis        string `json:"basis"`
	ObservedAt   string `json:"observedAt"`
}
type repositoryDTO struct {
	RepositoryKey     string         `json:"repositoryKey"`
	ExplicitPath      string         `json:"explicitPath"`
	CanonicalIdentity string         `json:"canonicalIdentity"`
	Observation       observationDTO `json:"observation"`
}
type credentialDTO struct {
	ReferenceKey  string `json:"referenceKey"`
	SourceKind    string `json:"sourceKind"`
	ItemReference string `json:"itemReference"`
}
type runtimeDTO struct {
	RuntimeID    string         `json:"runtimeId"`
	ExplicitPath string         `json:"explicitPath"`
	Observation  observationDTO `json:"observation"`
}
type attemptDTO struct {
	Correlation string `json:"correlation"`
	At          string `json:"at"`
}
