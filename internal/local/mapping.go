package local

import (
	"encoding/hex"
	"time"

	"github.com/rgomids/axiom/internal/projectapp"
)

func toDTO(s RecordState) recordDTO {
	revision, _ := s.PortableRevision.Digest()
	d := recordDTO{FormatVersion: 1, ProjectID: s.ProjectID, ObservedSlug: s.ObservedSlug, SourceLocation: s.SourceLocation, LocalRevision: s.LocalRevision, PortableRevision: hex.EncodeToString(revision[:]), ArtifactDigests: []digestDTO{}, Repositories: []repositoryDTO{}, Credentials: []credentialDTO{}}
	for _, a := range s.ArtifactDigests {
		d.ArtifactDigests = append(d.ArtifactDigests, digestDTO{a.Name, hex.EncodeToString(a.Digest[:])})
	}
	for _, r := range s.Repositories {
		d.Repositories = append(d.Repositories, repositoryDTO{r.RepositoryKey, r.ExplicitPath, r.CanonicalIdentity, observationToDTO(r.Observation)})
	}
	for _, c := range s.Credentials {
		d.Credentials = append(d.Credentials, credentialDTO{c.ReferenceKey, c.SourceKind, c.ItemReference})
	}
	if s.Runtime != (projectapp.RuntimeBinding{}) {
		d.Runtime = &runtimeDTO{s.Runtime.RuntimeID, s.Runtime.ExplicitPath, observationToDTO(s.Runtime.Observation)}
	}
	if s.Attempt != (projectapp.AttemptMetadata{}) {
		d.Attempt = &attemptDTO{s.Attempt.Correlation, s.Attempt.At.Format(time.RFC3339Nano)}
	}
	return d
}
func observationToDTO(o projectapp.Observation) observationDTO {
	return observationDTO{availabilityName(o.Availability), basisName(o.Basis), o.ObservedAt.Format(time.RFC3339Nano)}
}
func availabilityName(a projectapp.Availability) string {
	switch a {
	case projectapp.Available:
		return "available"
	case projectapp.Unavailable:
		return "unavailable"
	case projectapp.Unverified:
		return "unverified"
	}
	return ""
}
func basisName(b projectapp.ObservationBasis) string {
	switch b {
	case projectapp.NotChecked:
		return "not_checked"
	case projectapp.PresentMetadata:
		return "present_metadata"
	case projectapp.MissingMetadata:
		return "missing_metadata"
	case projectapp.UnsupportedCheck:
		return "unsupported_check"
	case projectapp.UncertainPermissions:
		return "uncertain_permissions"
	case projectapp.MismatchedMetadata:
		return "mismatched_metadata"
	}
	return ""
}
func fromObservation(d observationDTO) (projectapp.Observation, bool) {
	at, err := time.Parse(time.RFC3339Nano, d.ObservedAt)
	if err != nil {
		return projectapp.Observation{}, false
	}
	for a := projectapp.Unverified; a <= projectapp.Unavailable; a++ {
		if availabilityName(a) != d.Availability {
			continue
		}
		for b := projectapp.NotChecked; b <= projectapp.MismatchedMetadata; b++ {
			if basisName(b) == d.Basis {
				return projectapp.Observation{Availability: a, Basis: b, ObservedAt: at}, true
			}
		}
	}
	return projectapp.Observation{}, false
}
func parseDigest(value string) ([32]byte, bool) {
	var result [32]byte
	if len(value) != 64 {
		return result, false
	}
	bytes, err := hex.DecodeString(value)
	if err != nil {
		return result, false
	}
	copy(result[:], bytes)
	return result, true
}
func fromDTO(d recordDTO) (RecordState, []Issue) {
	s := RecordState{ProjectID: d.ProjectID, ObservedSlug: d.ObservedSlug, SourceLocation: d.SourceLocation, LocalRevision: d.LocalRevision, ArtifactDigests: []projectapp.ArtifactDigest{}, Repositories: []projectapp.RepositoryBinding{}, Credentials: []projectapp.CredentialBinding{}}
	revision, ok := parseDigest(d.PortableRevision)
	if !ok {
		return RecordState{}, problem("installation.portableRevision", "invalid_digest")
	}
	s.PortableRevision = projectapp.RecordedPortableRevision(revision)
	for _, a := range d.ArtifactDigests {
		digest, ok := parseDigest(a.Digest)
		if !ok {
			return RecordState{}, problem("installation.artifactDigests", "invalid_digest")
		}
		s.ArtifactDigests = append(s.ArtifactDigests, projectapp.ArtifactDigest{Name: a.Name, Digest: digest})
	}
	for _, r := range d.Repositories {
		o, ok := fromObservation(r.Observation)
		if !ok {
			return RecordState{}, problem("installation.repositories", "invalid_observation")
		}
		s.Repositories = append(s.Repositories, projectapp.RepositoryBinding{RepositoryKey: r.RepositoryKey, ExplicitPath: r.ExplicitPath, CanonicalIdentity: r.CanonicalIdentity, Observation: o})
	}
	for _, c := range d.Credentials {
		s.Credentials = append(s.Credentials, projectapp.CredentialBinding{ReferenceKey: c.ReferenceKey, SourceKind: c.SourceKind, ItemReference: c.ItemReference})
	}
	if d.Runtime != nil {
		o, ok := fromObservation(d.Runtime.Observation)
		if !ok {
			return RecordState{}, problem("installation.runtime", "invalid_observation")
		}
		s.Runtime = projectapp.RuntimeBinding{RuntimeID: d.Runtime.RuntimeID, ExplicitPath: d.Runtime.ExplicitPath, Observation: o}
	}
	if d.Attempt != nil {
		at, err := time.Parse(time.RFC3339Nano, d.Attempt.At)
		if err != nil {
			return RecordState{}, problem("installation.attempt", "invalid_time")
		}
		s.Attempt = projectapp.AttemptMetadata{Correlation: d.Attempt.Correlation, At: at}
	}
	return s, nil
}
