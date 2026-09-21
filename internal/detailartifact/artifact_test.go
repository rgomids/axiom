package detailartifact

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/rgomids/axiom/internal/provenance"
)

func source(t *testing.T) provenance.Value {
	t.Helper()
	value, err := provenance.FromBuild(provenance.Build{Version: provenance.Development, Revision: "123456789abc", SourceState: provenance.Clean}, nil)
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func validDraft(t *testing.T) Draft {
	return Draft{CorrelationID: "123e4567-e89b-42d3-a456-426614174000", Category: "diagnostic", Outcome: "failure", Retention: Diagnostic, Markdown: []byte("# Diagnosis\n\nSynthetic safe detail.\n"), Provenance: source(t)}
}

func TestArtifactClosedCodecRoundTrip(t *testing.T) {
	artifact, err := New("123e4567-e89b-42d3-a456-426614174001", time.Unix(1, 2), validDraft(t))
	if err != nil {
		t.Fatal(err)
	}
	metadata, err := EncodeMetadata(artifact)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := Decode(metadata, artifact.Markdown)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Reference() != artifact.Reference() || decoded.Digest != artifact.Digest || decoded.ContentBytes != len(artifact.Markdown) {
		t.Fatalf("round trip mismatch: %#v", decoded)
	}
	for _, invalid := range [][]byte{
		bytes.Replace(metadata, []byte(`"formatVersion":1`), []byte(`"formatVersion":2`), 1),
		bytes.Replace(metadata, []byte(`"artifactId"`), []byte(`"unknown"`), 1),
		bytes.Replace(metadata, []byte(`"category"`), []byte(`"category":"x","category"`), 1),
	} {
		if _, err := Decode(invalid, artifact.Markdown); err == nil {
			t.Fatal("invalid metadata accepted")
		}
	}
}

func TestArtifactBoundsAndSanitization(t *testing.T) {
	for _, content := range [][]byte{
		{}, bytes.Repeat([]byte("a"), MaxContentBytes+1), []byte("to" + "ken=SYNTHETIC_VALUE_SENTINEL"),
		[]byte("-----BEGIN PRIVATE " + "KEY-----"), []byte("<|assistant| raw chat"), {0xff},
	} {
		draft := validDraft(t)
		draft.Markdown = content
		if _, err := New("123e4567-e89b-42d3-a456-426614174001", time.Now(), draft); err == nil {
			t.Fatalf("unsafe content accepted: %q", content[:min(len(content), 40)])
		}
	}
	draft := validDraft(t)
	draft.CapturedOutputBytes = MaxCapturedOutputBytes + 1
	if _, err := New("123e4567-e89b-42d3-a456-426614174001", time.Now(), draft); err == nil {
		t.Fatal("captured output bound ignored")
	}
}

func TestArtifactRejectsSensitiveReferenceMetadataOnCreateAndDecode(t *testing.T) {
	for _, sentinel := range []string{
		"pass" + "word=SYNTHETIC_VALUE_SENTINEL",
		"api_" + "key:SYNTHETIC_VALUE_SENTINEL",
		"authori" + "zation=Bearer SYNTHETIC_VALUE_SENTINEL",
	} {
		t.Run(sentinel[:3], func(t *testing.T) {
			for _, apply := range []func(*Draft){
				func(draft *Draft) { draft.References = []Reference{{Kind: "source", Value: sentinel}} },
				func(draft *Draft) { draft.LiveReferences = []string{sentinel} },
			} {
				draft := validDraft(t)
				apply(&draft)
				if _, err := New("123e4567-e89b-42d3-a456-426614174001", time.Now(), draft); err == nil {
					t.Fatalf("sensitive metadata accepted: %q", sentinel)
				}
			}

			for _, apply := range []func(*Draft){
				func(draft *Draft) { draft.References = []Reference{{Kind: "source", Value: "safe-reference"}} },
				func(draft *Draft) { draft.LiveReferences = []string{"safe-reference"} },
			} {
				draft := validDraft(t)
				apply(&draft)
				artifact, err := New("123e4567-e89b-42d3-a456-426614174001", time.Now(), draft)
				if err != nil {
					t.Fatal(err)
				}
				metadata, err := EncodeMetadata(artifact)
				if err != nil {
					t.Fatal(err)
				}
				metadata = bytes.Replace(metadata, []byte("safe-reference"), []byte(sentinel), 1)
				if _, err := Decode(metadata, artifact.Markdown); err == nil {
					t.Fatalf("adulterated sensitive metadata accepted: %q", sentinel)
				}
			}
		})
	}
}

func TestArtifactRequiresStableCorrelationAndDigest(t *testing.T) {
	draft := validDraft(t)
	draft.CorrelationID = ""
	if _, err := New("123e4567-e89b-42d3-a456-426614174001", time.Now(), draft); err == nil {
		t.Fatal("uncorrelated artifact accepted")
	}
	artifact, err := New("123e4567-e89b-42d3-a456-426614174001", time.Now(), validDraft(t))
	if err != nil {
		t.Fatal(err)
	}
	metadata, _ := EncodeMetadata(artifact)
	if _, err := Decode(metadata, []byte("# Changed\n")); err == nil {
		t.Fatal("digest mismatch accepted")
	}
}

func TestArtifactMetadataLimitIsEnforcedWithoutDroppingReferences(t *testing.T) {
	draft := validDraft(t)
	value := strings.Repeat("r", MaxReferenceBytes)
	for index := 0; index < MaxReferences; index++ {
		draft.References = append(draft.References, Reference{Kind: "source", Value: value})
		draft.LiveReferences = append(draft.LiveReferences, value)
	}
	artifact, err := New("123e4567-e89b-42d3-a456-426614174001", time.Now(), draft)
	if err != nil {
		t.Fatal(err)
	}
	if metadata, err := EncodeMetadata(artifact); err == nil || metadata != nil {
		t.Fatal("oversized metadata was truncated or accepted")
	}
}
