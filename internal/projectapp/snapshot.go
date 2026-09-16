// Package projectapp owns the application contracts for Specification 002.
// It does not implement codecs, filesystem protocols or complete use cases.
package projectapp

import (
	"crypto/sha256"
	"encoding/binary"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/rgomids/axiom/internal/project"
)

// Revisions are exact observations, never wildcard preconditions. Zero is invalid;
// explicit absence differs from an existing empty artifact/record. Local revision
// encoding stays behind the local adapter. Neither revision proves authenticity.
type PortableRevision struct {
	digest         [32]byte
	present, valid bool
}
type LocalRevision struct {
	digest         [32]byte
	present, valid bool
}

func MissingPortableRevision() PortableRevision { return PortableRevision{valid: true} }
func MissingLocalRevision() LocalRevision       { return LocalRevision{valid: true} }
func ObserveLocalRevision(record []byte) LocalRevision {
	return LocalRevision{sha256.Sum256(record), true, true}
}

// RecordedPortableRevision restores a digest from a validated local record. It
// is historical evidence, never a fresh observation or approval. Readers must
// compare it with a freshly read complete snapshot before reusing local facts.
func RecordedPortableRevision(digest [32]byte) PortableRevision {
	return PortableRevision{digest: digest, present: true, valid: true}
}
func (r PortableRevision) Digest() ([32]byte, bool) { return r.digest, r.valid && r.present }
func (r LocalRevision) Digest() ([32]byte, bool)    { return r.digest, r.valid && r.present }

// ArtifactDigest is metadata only; no document body can enter a local record.
type ArtifactDigest struct {
	Name   string
	Digest [32]byte
}

func (s ArtifactSnapshot) Digests() []ArtifactDigest {
	if !s.valid() {
		return nil
	}
	result := []ArtifactDigest{{Name: "axiom.yaml", Digest: sha256.Sum256(s.manifest)}}
	for _, doc := range s.documents {
		result = append(result, ArtifactDigest{Name: doc.Name, Digest: sha256.Sum256(doc.Content)})
	}
	return result
}

type ExpectedRevisions struct {
	Portable PortableRevision
	Local    LocalRevision
}

// Destination is an opaque in-process capability identity, not a path, root
// override or approval. Trusted adapters bind it to an inspected directory object
// and retain that binding through commit; constructing it performs no I/O.
// Copies retain identity. Re-resolving a destination requires a new preview.
type Destination struct{ identity *destinationIdentity }
type destinationIdentity struct{ marker byte }

func NewDestination() Destination { return Destination{&destinationIdentity{1}} }
func (d Destination) valid() bool { return d.identity != nil }

// Document is full untrusted content, never a patch or an instruction.
// Physical allowlisting, containment and text safety are later adapter/use-case
// obligations. Names are relative artifact identifiers, not write destinations.
type Document struct {
	Name    string
	Content []byte
}

type ArtifactSnapshot struct {
	definition project.Project
	manifest   []byte
	documents  []Document
	revision   PortableRevision
}

// ReadSnapshot couples manifest bytes to their decoded complete domain state.
// Only the injected codec interprets the wire format. Every referenced document
// must be supplied; no filesystem lookup, source fetch or secret read occurs.
func ReadSnapshot(codec ManifestCodec, manifest []byte, documents []Document) (ArtifactSnapshot, []Issue) {
	if codec == nil || len(manifest) == 0 {
		return ArtifactSnapshot{}, problem(InvalidSnapshot)
	}
	bytes := append([]byte(nil), manifest...)
	p, issues := codec.Decode(append([]byte(nil), bytes...))
	if len(issues) != 0 {
		return ArtifactSnapshot{}, OrderedIssues(issues)
	}
	if !p.Equivalent(p) {
		return ArtifactSnapshot{}, problem(InvalidSnapshot)
	}
	docs := cloneDocuments(documents)
	sort.Slice(docs, func(i, j int) bool { return docs[i].Name < docs[j].Name })
	names := map[string]bool{}
	for _, doc := range docs {
		if !ValidDocumentName(doc.Name) || names[doc.Name] {
			return ArtifactSnapshot{}, problem(InvalidSnapshot)
		}
		names[doc.Name] = true
	}
	if !completeDocuments(p, names) {
		return ArtifactSnapshot{}, problem(InvalidSnapshot)
	}
	s := ArtifactSnapshot{definition: p, manifest: bytes, documents: docs}
	s.revision = PortableRevision{digest: artifactDigest(bytes, docs), present: true, valid: true}
	return s, nil
}

// ValidDocumentName requires valid UTF-8 and lexical names for ReadSnapshot.
// Local digest metadata reuses it without replacing or normalizing names.
// The manifest name is reserved; this check grants no filesystem authority or
// text-safety proof.
func ValidDocumentName(name string) bool {
	if !utf8.ValidString(name) || name == "" || name == "axiom.yaml" || strings.ContainsAny(name, "\\:$`\x00\r\n") || strings.HasPrefix(name, "~") {
		return false
	}
	for _, part := range strings.Split(name, "/") {
		if part == "" || part == "." || part == ".." {
			return false
		}
	}
	return true
}
func completeDocuments(p project.Project, names map[string]bool) bool {
	s := p.State()
	context, _ := s.BusinessContext.Value()
	docs, _ := context.Documents.Value()
	policies, _ := s.Policies.Value()
	for _, name := range append(docs, policies...) {
		if !names[name] {
			return false
		}
	}
	return true
}
func cloneDocuments(input []Document) []Document {
	output := make([]Document, len(input))
	for i, doc := range input {
		output[i] = Document{doc.Name, append([]byte(nil), doc.Content...)}
	}
	return output
}
func artifactDigest(manifest []byte, docs []Document) [32]byte {
	// Length framing separates names/content unambiguously; sorted names make
	// identical observations independent of enumeration order.
	data := frame(nil, []byte("axiom.yaml"))
	data = frame(data, manifest)
	for _, doc := range docs {
		data = frame(data, []byte(doc.Name))
		data = frame(data, doc.Content)
	}
	return sha256.Sum256(data)
}
func frame(dst, value []byte) []byte {
	dst = binary.BigEndian.AppendUint64(dst, uint64(len(value)))
	return append(dst, value...)
}
func (s ArtifactSnapshot) Manifest() []byte           { return append([]byte(nil), s.manifest...) }
func (s ArtifactSnapshot) Documents() []Document      { return cloneDocuments(s.documents) }
func (s ArtifactSnapshot) Project() project.Project   { return s.definition }
func (s ArtifactSnapshot) Revision() PortableRevision { return s.revision }
func (s ArtifactSnapshot) valid() bool                { return s.revision.valid && s.revision.present }
func (s ArtifactSnapshot) names() []string {
	if !s.valid() {
		return nil
	}
	names := []string{"axiom.yaml"}
	for _, doc := range s.documents {
		names = append(names, doc.Name)
	}
	return names
}
