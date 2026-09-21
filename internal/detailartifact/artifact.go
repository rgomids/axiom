// Package detailartifact defines the bounded machine-local detail artifact contract.
package detailartifact

import (
	"crypto/sha256"
	"errors"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/rgomids/axiom/internal/provenance"
)

const (
	MaxContentBytes        = 1 << 20
	MaxMetadataBytes       = 64 << 10
	MaxCapturedOutputBytes = 256 << 10
	MaxLiveArtifacts       = 10_000
	MaxAggregateBytes      = 1 << 30
	MaxReferences          = 64
	MaxReferenceBytes      = 512
)

type RetentionClass string

const (
	Active          RetentionClass = "active"
	Evidence        RetentionClass = "evidence"
	Diagnostic      RetentionClass = "diagnostic"
	PreservedReview RetentionClass = "preserved_review"
)

func (r RetentionClass) Valid() bool {
	return r == Active || r == Evidence || r == Diagnostic || r == PreservedReview
}

type Reference struct {
	Kind  string `json:"kind"`
	Value string `json:"value"`
}

type Draft struct {
	ExecutionID         string
	CorrelationID       string
	References          []Reference
	Category            string
	Outcome             string
	Retention           RetentionClass
	LiveReferences      []string
	SupersededBy        string
	CapturedOutputBytes int
	Markdown            []byte
	Provenance          provenance.Value
}

type Artifact struct {
	ID                  string
	ExecutionID         string
	CorrelationID       string
	CreatedAt           time.Time
	References          []Reference
	Category            string
	Outcome             string
	Retention           RetentionClass
	ContentBytes        int
	CapturedOutputBytes int
	Digest              [32]byte
	LiveReferences      []string
	SupersededBy        string
	CleanupState        string
	Markdown            []byte
	Provenance          provenance.Value
}

func (a Artifact) Reference() string { return "artifact:" + a.ID }

func (a Artifact) Valid() bool {
	if !ValidID(a.ID) || a.CreatedAt.IsZero() || !a.Provenance.Valid() || !a.Retention.Valid() {
		return false
	}
	if !validCorrelation(a.ExecutionID, a.CorrelationID) || !validToken(a.Category, 64) || !validToken(a.Outcome, 64) {
		return false
	}
	if a.CleanupState != "retained" || a.ContentBytes != len(a.Markdown) || a.ContentBytes > MaxContentBytes {
		return false
	}
	if a.CapturedOutputBytes < 0 || a.CapturedOutputBytes > MaxCapturedOutputBytes || a.CapturedOutputBytes > a.ContentBytes {
		return false
	}
	if sha256.Sum256(a.Markdown) != a.Digest || !SafeMarkdown(a.Markdown) {
		return false
	}
	return validReferences(a.References) && validStrings(a.LiveReferences, MaxReferences) && optionalID(a.SupersededBy)
}

func New(id string, at time.Time, draft Draft) (Artifact, error) {
	artifact := Artifact{
		ID: id, ExecutionID: draft.ExecutionID, CorrelationID: draft.CorrelationID,
		CreatedAt: at.UTC(), References: cloneReferences(draft.References), Category: draft.Category,
		Outcome: draft.Outcome, Retention: draft.Retention, ContentBytes: len(draft.Markdown),
		CapturedOutputBytes: draft.CapturedOutputBytes, Digest: sha256.Sum256(draft.Markdown),
		LiveReferences: append([]string(nil), draft.LiveReferences...), SupersededBy: draft.SupersededBy,
		CleanupState: "retained", Markdown: append([]byte(nil), draft.Markdown...), Provenance: draft.Provenance,
	}
	if !artifact.Valid() {
		return Artifact{}, errors.New("invalid detail artifact")
	}
	return artifact, nil
}

var (
	uuidV4              = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	sensitiveAssignment = regexp.MustCompile(`(?i)(token|access[_-]?token|refresh[_-]?token|id[_-]?token|auth[_-]?token|oauth[_-]?token|password|passwd|pwd|api[_-]?key|client[_-]?secret|authorization|credential|secret)[ \t]*[:=]`)
)

func ValidID(value string) bool { return uuidV4.MatchString(value) }

func SafeMarkdown(content []byte) bool {
	if len(content) == 0 || len(content) > MaxContentBytes || !utf8.Valid(content) {
		return false
	}
	text := string(content)
	if sensitiveText(text) {
		return false
	}
	for _, current := range text {
		if current == '\n' || current == '\r' || current == '\t' {
			continue
		}
		if unicode.IsControl(current) {
			return false
		}
	}
	return true
}

func validCorrelation(execution, correlation string) bool {
	if execution == "" && correlation == "" {
		return false
	}
	return (execution == "" || ValidID(execution)) && (correlation == "" || ValidID(correlation))
}

func optionalID(value string) bool { return value == "" || ValidID(value) }

func validToken(value string, limit int) bool {
	if value == "" || len(value) > limit || !utf8.ValidString(value) || strings.TrimSpace(value) != value {
		return false
	}
	for _, current := range value {
		if unicode.IsControl(current) || unicode.IsSpace(current) {
			return false
		}
	}
	return true
}

func validReferences(values []Reference) bool {
	if len(values) > MaxReferences {
		return false
	}
	for _, value := range values {
		if !validToken(value.Kind, 64) || !validMetadata(value.Value, MaxReferenceBytes) {
			return false
		}
	}
	return true
}

func validStrings(values []string, limit int) bool {
	if len(values) > limit {
		return false
	}
	for _, value := range values {
		if !validMetadata(value, MaxReferenceBytes) {
			return false
		}
	}
	return true
}

func validMetadata(value string, limit int) bool {
	if value == "" || len(value) > limit || !utf8.ValidString(value) || strings.TrimSpace(value) != value {
		return false
	}
	if sensitiveText(value) {
		return false
	}
	for _, current := range value {
		if unicode.IsControl(current) {
			return false
		}
	}
	return true
}

func sensitiveText(value string) bool {
	privateKeyBoundary := "-----BEGIN PRIVATE " + "KEY-----"
	return sensitiveAssignment.MatchString(value) || strings.Contains(value, privateKeyBoundary) || strings.Contains(value, "<|assistant|") || strings.Contains(value, "<|user|")
}

func cloneReferences(values []Reference) []Reference {
	result := make([]Reference, len(values))
	copy(result, values)
	return result
}
