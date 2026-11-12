package local

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"strings"

	"github.com/rgomids/axiom/internal/detailartifact"
	"github.com/rgomids/axiom/internal/provenance"
	"github.com/rgomids/axiom/internal/strictjson"
	"github.com/rgomids/axiom/internal/workflow"
)

// ValidateExecutionOutput preserves builtin-sdd-v1 schema, category, outcome,
// digest and correlation checks without constructing a legacy Workflow state.
// New attempts additionally bind repository and attempt, preventing replay of
// a successful artifact from a different attempt or repository.
func (v WorkflowReferenceValidator) ValidateExecutionOutput(ctx context.Context, subject workflow.OutputSubject, wire []byte) (workflow.ArtifactRef, error) {
	// Parallel attempts of any Execution read, validate and retain outputs
	// concurrently; a nonblocking lock turned that contention into launch or
	// output refusals (AXM-12), so every Execution path waits, bounded.
	v.artifacts.waitForLocks = true
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

// ObserveExecutionArtifact re-reads the exact attempt-owned content. Retained
// typed output is checked by the same schema/lineage validator as collection;
// ordinary output is hashed in its private, rooted ingress directory.
func (v WorkflowReferenceValidator) ObserveExecutionArtifact(ctx context.Context, x workflow.Execution, ref workflow.TaskRef, a workflow.Attempt, artifact workflow.ArtifactRef) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	recorded, actualRef, ok := x.FindAttempt(a.AttemptID)
	if !ok || actualRef != ref || recorded.Authority.RepositoryKey != artifact.RepositoryKey || a.Authority.RepositoryKey != artifact.RepositoryKey || !workflow.ValidToken(artifact.Name) {
		return ErrUnsafe
	}
	found := false
	for _, r := range recorded.Artifacts {
		found = found || r == artifact
	}
	if !found {
		return ErrUnsafe
	}
	for _, stage := range x.Workflow.Stages {
		if stage.ID == ref.StageID {
			for _, check := range stage.OutputChecks {
				if check.Name == artifact.Name {
					wire, _ := json.Marshal(map[string]string{"kind": "artifact", "id": artifact.RetainedID, "digest": strings.TrimPrefix(artifact.Digest, "sha256:")})
					observed, err := v.ValidateExecutionOutput(ctx, workflow.OutputSubject{Isolated: a.Workspace != nil, ExecutionID: x.ExecutionID, AttemptID: a.AttemptID, StageID: ref.StageID, RepositoryKey: artifact.RepositoryKey, DefinitionDigest: stage.DefinitionDigest, Check: check}, wire)
					if err != nil || observed != artifact {
						return ErrUnsafe
					}
					return nil
				}
			}
		}
	}
	if artifact.RetainedID != "" {
		return ErrUnsafe
	}
	root, err := existingPrivateRoot(v.artifacts.root)
	if err != nil {
		return err
	}
	for _, name := range []string{"executions", "v2-attempt-outputs", x.ProjectID, x.ExecutionID, a.AttemptID} {
		child, err := existingPrivateChild(root, name)
		root.Close()
		if err != nil {
			return err
		}
		root = child
	}
	defer root.Close()
	info, err := root.Lstat(artifact.Name)
	if err != nil || !info.Mode().IsRegular() || info.Size() > maxWorkflowEvidenceBytes {
		return ErrUnsafe
	}
	file, err := root.OpenFile(artifact.Name, os.O_RDONLY|noFollow, 0)
	if err != nil {
		return err
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil || !os.SameFile(info, opened) || !singleLink(file, opened) {
		return ErrUnsafe
	}
	data, err := io.ReadAll(io.LimitReader(file, maxWorkflowEvidenceBytes+1))
	if err != nil || len(data) > maxWorkflowEvidenceBytes || int64(len(data)) != artifact.Size || workflow.DigestText(data) != artifact.Digest {
		return ErrUnsafe
	}
	return ctx.Err()
}

// RetainExecutionOutput retains a child's delivered content for a checked
// output, correlated to the exact attempt, stage, output and repository, then
// validates it as a typed reference. Stage children cannot mint this
// correlation themselves; only the engine that collected the bytes does.
func (v WorkflowReferenceValidator) RetainExecutionOutput(ctx context.Context, subject workflow.OutputSubject, content []byte, source provenance.Value) (workflow.ArtifactRef, error) {
	v.artifacts.waitForLocks = true
	retention := detailartifact.Active
	if subject.Check.Evidence {
		retention = detailartifact.Evidence
	}
	artifact, err := v.artifacts.Create(ctx, detailartifact.Draft{
		ExecutionID: subject.ExecutionID,
		References: []detailartifact.Reference{
			{Kind: "workflow-stage", Value: subject.StageID},
			{Kind: "workflow-definition", Value: strings.TrimPrefix(subject.DefinitionDigest, "sha256:")},
			{Kind: "workflow-output", Value: subject.Check.Name},
			{Kind: "workflow-attempt", Value: subject.AttemptID},
			{Kind: "workflow-repository", Value: subject.RepositoryKey},
		},
		Category: subject.Check.Kind, Outcome: "success", Retention: retention, Markdown: content, Provenance: source,
	})
	if err != nil {
		return workflow.ArtifactRef{}, err
	}
	wire, err := json.Marshal(map[string]string{"kind": "artifact", "id": artifact.ID, "digest": hex.EncodeToString(artifact.Digest[:])})
	if err != nil {
		return workflow.ArtifactRef{}, err
	}
	return v.ValidateExecutionOutput(ctx, subject, wire)
}

// ReadExecutionArtifact returns the exact bytes of one recorded stage output,
// so a successor stage receives its predecessor's content and not only a
// reference it cannot open. Digest and Execution binding are re-checked.
func (v WorkflowReferenceValidator) ReadExecutionArtifact(ctx context.Context, x workflow.Execution, attemptID string, artifact workflow.ArtifactRef) ([]byte, error) {
	v.artifacts.waitForLocks = true
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if artifact.RetainedID != "" {
		retained, err := v.artifacts.Read(ctx, artifact.RetainedID)
		if err != nil || retained.ExecutionID != x.ExecutionID || "sha256:"+hex.EncodeToString(retained.Digest[:]) != artifact.Digest {
			return nil, ErrUnsafe
		}
		return retained.Markdown, nil
	}
	if !workflow.ValidToken(attemptID) || !workflow.ValidToken(artifact.Name) {
		return nil, ErrUnsafe
	}
	root, err := existingPrivateRoot(v.artifacts.root)
	if err != nil {
		return nil, err
	}
	for _, name := range []string{"executions", "v2-attempt-outputs", x.ProjectID, x.ExecutionID, attemptID} {
		child, err := existingPrivateChild(root, name)
		root.Close()
		if err != nil {
			return nil, err
		}
		root = child
	}
	defer root.Close()
	file, err := root.OpenFile(artifact.Name, os.O_RDONLY|noFollow, 0)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maxWorkflowEvidenceBytes+1))
	if err != nil || len(data) > maxWorkflowEvidenceBytes || workflow.DigestText(data) != artifact.Digest {
		return nil, ErrUnsafe
	}
	return data, nil
}
