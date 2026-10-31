package local

import (
	"context"
	"encoding/hex"
	"strings"

	"github.com/rgomids/axiom/internal/strictjson"
	"github.com/rgomids/axiom/internal/workflow"
)

// ValidateExecutionOutput preserves builtin-sdd-v1 schema, category, outcome,
// digest and correlation checks without constructing a legacy Workflow state.
// New attempts additionally bind repository and attempt, preventing replay of
// a successful artifact from a different attempt or repository.
func (v WorkflowReferenceValidator) ValidateExecutionOutput(ctx context.Context, subject workflow.OutputSubject, wire []byte) (workflow.ArtifactRef, error) {
	fail := func() (workflow.ArtifactRef, error) { return workflow.ArtifactRef{}, ErrUnsafe }
	var ref struct {
		Kind   string `json:"kind"`
		ID     string `json:"id"`
		Digest string `json:"digest"`
	}
	if strictjson.Decode(wire, &ref, 4096) != nil || ref.Kind != "artifact" || !workflow.ValidExecutionID(subject.ExecutionID) || !workflow.ValidToken(subject.AttemptID) || !workflow.ValidToken(subject.StageID) || !workflow.ValidToken(subject.RepositoryKey) || !workflow.ValidToken(subject.Check.Name) || !workflow.ValidToken(subject.Check.Kind) {
		return fail()
	}
	digest := strings.TrimPrefix(subject.DefinitionDigest, "sha256:")
	if len(subject.DefinitionDigest) != 71 || !strings.HasPrefix(subject.DefinitionDigest, "sha256:") {
		return fail()
	}
	decoded, err := hex.DecodeString(digest)
	if err != nil || len(decoded) != 32 || hex.EncodeToString(decoded) != digest {
		return fail()
	}
	artifact, err := v.artifacts.Read(ctx, ref.ID)
	if err != nil || artifact.ExecutionID != subject.ExecutionID || hex.EncodeToString(artifact.Digest[:]) != ref.Digest || artifact.Category != subject.Check.Kind || artifact.Outcome != "success" || subject.Check.Evidence && artifact.Retention != "evidence" {
		return fail()
	}
	for kind, expected := range map[string]string{
		"workflow-stage": subject.StageID, "workflow-definition": digest, "workflow-output": subject.Check.Name,
		"workflow-attempt": subject.AttemptID, "workflow-repository": subject.RepositoryKey,
	} {
		count := 0
		for _, actual := range artifact.References {
			if actual.Kind == kind {
				if actual.Value != expected {
					return fail()
				}
				count++
			}
		}
		if count != 1 {
			return fail()
		}
	}
	return workflow.ArtifactRef{Name: subject.Check.Name, RetainedID: artifact.ID, RepositoryKey: subject.RepositoryKey, Digest: "sha256:" + ref.Digest, Size: int64(artifact.ContentBytes)}, nil
}
