package detailartifact

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"time"

	"github.com/rgomids/axiom/internal/provenance"
)

type metadataDTO struct {
	FormatVersion       int            `json:"formatVersion"`
	ArtifactID          string         `json:"artifactId"`
	ExecutionID         string         `json:"executionId,omitempty"`
	CorrelationID       string         `json:"correlationId,omitempty"`
	CreatedAt           string         `json:"createdAt"`
	Provenance          provenanceDTO  `json:"provenance"`
	References          []Reference    `json:"references"`
	Category            string         `json:"category"`
	Outcome             string         `json:"outcome"`
	RetentionClass      RetentionClass `json:"retentionClass"`
	ContentBytes        int            `json:"contentBytes"`
	CapturedOutputBytes int            `json:"capturedOutputBytes"`
	Digest              string         `json:"sha256"`
	LiveReferences      []string       `json:"liveReferences"`
	SupersededBy        string         `json:"supersededBy,omitempty"`
	CleanupState        string         `json:"cleanupState"`
}

type provenanceDTO struct {
	Product     string                 `json:"product"`
	Version     string                 `json:"version"`
	Revision    string                 `json:"revision"`
	SourceState provenance.SourceState `json:"sourceState"`
}

func EncodeMetadata(artifact Artifact) ([]byte, error) {
	if !artifact.Valid() {
		return nil, errors.New("invalid detail artifact")
	}
	dto := metadataDTO{
		FormatVersion: 1, ArtifactID: artifact.ID, ExecutionID: artifact.ExecutionID,
		CorrelationID: artifact.CorrelationID, CreatedAt: artifact.CreatedAt.Format(time.RFC3339Nano),
		Provenance: provenanceDTO{artifact.Provenance.Product(), artifact.Provenance.Version(), artifact.Provenance.Revision(), artifact.Provenance.SourceState()},
		References: cloneReferences(artifact.References), Category: artifact.Category, Outcome: artifact.Outcome,
		RetentionClass: artifact.Retention, ContentBytes: artifact.ContentBytes,
		CapturedOutputBytes: artifact.CapturedOutputBytes, Digest: hex.EncodeToString(artifact.Digest[:]),
		LiveReferences: append([]string(nil), artifact.LiveReferences...), SupersededBy: artifact.SupersededBy,
		CleanupState: artifact.CleanupState,
	}
	wire, err := json.Marshal(dto)
	if err != nil {
		return nil, err
	}
	wire = append(wire, '\n')
	if len(wire) > MaxMetadataBytes {
		return nil, errors.New("detail metadata exceeds limit")
	}
	return wire, nil
}

func Decode(metadata, markdown []byte) (Artifact, error) {
	if len(metadata) == 0 || len(metadata) > MaxMetadataBytes || len(markdown) == 0 || len(markdown) > MaxContentBytes {
		return Artifact{}, errors.New("detail artifact exceeds limit")
	}
	if err := rejectDuplicateJSONFields(metadata); err != nil {
		return Artifact{}, err
	}
	decoder := json.NewDecoder(bytes.NewReader(metadata))
	decoder.DisallowUnknownFields()
	var dto metadataDTO
	if err := decoder.Decode(&dto); err != nil {
		return Artifact{}, err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) || dto.FormatVersion != 1 {
		return Artifact{}, errors.New("unsupported detail metadata")
	}
	at, err := time.Parse(time.RFC3339Nano, dto.CreatedAt)
	if err != nil {
		return Artifact{}, errors.New("invalid detail creation time")
	}
	value, err := provenance.FromBuild(provenance.Build{Release: dto.Provenance.Version != provenance.Development, Version: dto.Provenance.Version, Revision: dto.Provenance.Revision, SourceState: dto.Provenance.SourceState}, nil)
	if err != nil || dto.Provenance.Product != provenance.Product {
		return Artifact{}, errors.New("invalid detail provenance")
	}
	digest, err := hex.DecodeString(dto.Digest)
	if err != nil || len(digest) != 32 {
		return Artifact{}, errors.New("invalid detail digest")
	}
	artifact := Artifact{
		ID: dto.ArtifactID, ExecutionID: dto.ExecutionID, CorrelationID: dto.CorrelationID,
		CreatedAt: at, References: cloneReferences(dto.References), Category: dto.Category,
		Outcome: dto.Outcome, Retention: dto.RetentionClass, ContentBytes: dto.ContentBytes,
		CapturedOutputBytes: dto.CapturedOutputBytes, LiveReferences: append([]string(nil), dto.LiveReferences...),
		SupersededBy: dto.SupersededBy, CleanupState: dto.CleanupState,
		Markdown: append([]byte(nil), markdown...), Provenance: value,
	}
	copy(artifact.Digest[:], digest)
	if !artifact.Valid() {
		return Artifact{}, errors.New("invalid detail artifact")
	}
	return artifact, nil
}

func rejectDuplicateJSONFields(input []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(input))
	if err := inspectJSONValue(decoder); err != nil {
		return err
	}
	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		return errors.New("trailing detail metadata")
	}
	return nil
}

func inspectJSONValue(decoder *json.Decoder) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delim, ok := token.(json.Delim)
	if !ok {
		return nil
	}
	switch delim {
	case '{':
		seen := map[string]bool{}
		for decoder.More() {
			keyToken, err := decoder.Token()
			if err != nil {
				return err
			}
			key, ok := keyToken.(string)
			if !ok || seen[key] {
				return errors.New("duplicate detail metadata field")
			}
			seen[key] = true
			if err := inspectJSONValue(decoder); err != nil {
				return err
			}
		}
		end, err := decoder.Token()
		if err != nil || end != json.Delim('}') {
			return errors.New("invalid detail metadata object")
		}
	case '[':
		for decoder.More() {
			if err := inspectJSONValue(decoder); err != nil {
				return err
			}
		}
		end, err := decoder.Token()
		if err != nil || end != json.Delim(']') {
			return errors.New("invalid detail metadata array")
		}
	default:
		return errors.New("invalid detail metadata delimiter")
	}
	return nil
}
