package local

import (
	"fmt"
	"net/url"
	"path"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/rgomids/axiom/internal/project"
	"github.com/rgomids/axiom/internal/projectapp"
)

func validateState(s RecordState) []Issue {
	for _, issue := range project.ValidateIdentity(s.ProjectID, s.ObservedSlug) {
		field := "installation.projectId"
		if issue.Field == "project.slug" {
			field = "installation.observedSlug"
		}
		return problem(field, issue.Code)
	}
	if !localLocation(s.SourceLocation) {
		return problem("installation.sourceLocation", "invalid_location")
	}
	if !logicalReference(s.LocalRevision) {
		return problem("installation.localRevision", "invalid_reference")
	}
	if _, ok := s.PortableRevision.Digest(); !ok {
		return problem("installation.portableRevision", "invalid_revision")
	}
	names := map[string]bool{}
	for i, a := range s.ArtifactDigests {
		if !artifactName(a.Name) || names[a.Name] {
			return problem(fmt.Sprintf("installation.artifactDigests[%d]", i), "invalid_artifact")
		}
		names[a.Name] = true
	}
	if !names["axiom.yaml"] {
		return problem("installation.artifactDigests", "missing_manifest_digest")
	}
	if issues := validateRepositories(s.Repositories); len(issues) > 0 {
		return issues
	}
	if issues := validateCredentials(s.Credentials); len(issues) > 0 {
		return issues
	}
	if s.Runtime != (projectapp.RuntimeBinding{}) && (!logicalReference(s.Runtime.RuntimeID) || !optionalLocation(s.Runtime.ExplicitPath) || !validObservation(s.Runtime.Observation)) {
		return problem("installation.runtime", "invalid_metadata")
	}
	if s.Attempt != (projectapp.AttemptMetadata{}) && (!logicalReference(s.Attempt.Correlation) || !validTime(s.Attempt.At)) {
		return problem("installation.attempt", "invalid_metadata")
	}
	return nil
}
func validateRepositories(bindings []projectapp.RepositoryBinding) []Issue {
	keys := map[string]bool{}
	for i, r := range bindings {
		if !logicalReference(r.RepositoryKey) || keys[r.RepositoryKey] || !optionalLocation(r.ExplicitPath) || !optionalReference(r.CanonicalIdentity) || !validObservation(r.Observation) {
			return problem(fmt.Sprintf("installation.repositories[%d]", i), "invalid_metadata")
		}
		keys[r.RepositoryKey] = true
	}
	return nil
}
func validateCredentials(bindings []projectapp.CredentialBinding) []Issue {
	keys := map[string]bool{}
	for i, c := range bindings {
		if !logicalReference(c.ReferenceKey) || keys[c.ReferenceKey] || !optionalReference(c.SourceKind) || !optionalReference(c.ItemReference) || (c.SourceKind == "" && c.ItemReference != "") {
			return problem(fmt.Sprintf("installation.credentials[%d]", i), "invalid_reference")
		}
		keys[c.ReferenceKey] = true
	}
	return nil
}
func validObservation(o projectapp.Observation) bool {
	return availabilityName(o.Availability) != "" && basisName(o.Basis) != "" && validTime(o.ObservedAt)
}
func validTime(t time.Time) bool { return t.Year() >= 0 && t.Year() <= 9999 }
func cleanText(s string) bool {
	return utf8.ValidString(s) && !strings.ContainsRune(s, '\ufffd') && strings.IndexFunc(s, unicode.IsControl) < 0
}
func optionalLocation(s string) bool { return s == "" || localLocation(s) }

// Local paths are serialized metadata, not portable references or filesystem
// authority. Preserve spelling; resolution and physical validation belong to
// future adapters. Metacharacters are literal data and are never evaluated.
func localLocation(s string) bool {
	return s != "" && cleanText(s)
}
func artifactName(s string) bool {
	if s == "axiom.yaml" {
		return true
	}
	if s == "" || path.Clean(s) != s || strings.HasPrefix(s, "/") || strings.HasPrefix(s, "~") || strings.ContainsAny(s, "\\:$`?#") || !cleanText(s) {
		return false
	}
	for _, part := range strings.Split(s, "/") {
		if part == ".." || part == "." {
			return false
		}
	}
	return true
}
func optionalReference(s string) bool { return s == "" || logicalReference(s) }

// Reference syntax rejects payloads structurally, independent of their values.
// Namespace/item identifiers and unknown source kinds remain metadata; no store
// is selected or resolved. This is not heuristic secret scanning.
func logicalReference(s string) bool {
	budget := 4 * len(s)
	for {
		if s == "" || len(s) > budget || !referenceSyntax(s) {
			return false
		}
		budget -= len(s)
		if !strings.Contains(s, "%") {
			return true
		}
		decoded, err := url.PathUnescape(s)
		if err != nil {
			return false
		}
		s = decoded
	}
}
func referenceSyntax(s string) bool {
	if !cleanText(s) || strings.IndexFunc(s, unicode.IsSpace) >= 0 || strings.ContainsAny(s, "=\\$`?#{}[]\"@") || strings.Contains(s, "://") {
		return false
	}
	for _, suffix := range strings.Split(s, ":") {
		if strings.HasPrefix(suffix, "/") || strings.HasPrefix(suffix, "~") {
			return false
		}
		for _, part := range strings.Split(suffix, "/") {
			if part == ".." || part == "." {
				return false
			}
		}
	}
	prefix, _, found := strings.Cut(s, ":")
	if found && sensitiveName(prefix) {
		return false
	}
	return true
}
func sensitiveName(s string) bool {
	s = strings.ToLower(strings.NewReplacer("-", "", "_", "").Replace(s))
	switch s {
	case "token", "accesstoken", "refreshtoken", "idtoken", "authtoken", "oauthtoken", "password", "passwd", "pwd", "apikey", "key", "secret", "clientsecret", "signature", "sig", "credential", "authorization", "auth", "xamzsignature", "xamzcredential", "xamzsecuritytoken", "xgoogsignature", "xgoogcredential":
		return true
	}
	return false
}
