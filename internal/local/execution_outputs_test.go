package local

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/detailartifact"
	"github.com/rgomids/axiom/internal/provenance"
	"github.com/rgomids/axiom/internal/workflow"
)

func TestExecutionOutputValidatorsRequireExactRetainedArtifact(t *testing.T) {
	for _, mode := range []string{"schema", "evidence", "wrong-execution", "wrong-attempt", "wrong-repository", "wrong-stage", "wrong-definition", "wrong-output", "wrong-category", "failed", "wrong-digest", "missing-correlation", "duplicate-correlation", "not-evidence", "plain-file", "unknown-field", "duplicate-key", "oversize", "cancelled"} {
		t.Run(mode, func(t *testing.T) {
			ctx := context.Background()
			store, err := NewArtifactStore(filepath.Join(t.TempDir(), "state"))
			if err != nil {
				t.Fatal(err)
			}
			subject := workflow.OutputSubject{ExecutionID: "123e4567-e89b-42d3-a456-426614174000", AttemptID: "attempt-1", StageID: "work", DefinitionDigest: "sha256:" + strings.Repeat("a", 64), RepositoryKey: "main", Check: workflow.OutputCheck{Name: "result", Kind: "evidence", Evidence: mode == "evidence" || mode == "not-evidence"}}
			refs := []detailartifact.Reference{{Kind: "workflow-stage", Value: subject.StageID}, {Kind: "workflow-definition", Value: strings.Repeat("a", 64)}, {Kind: "workflow-output", Value: "result"}, {Kind: "workflow-attempt", Value: subject.AttemptID}, {Kind: "workflow-repository", Value: subject.RepositoryKey}}
			source, _ := provenance.FromBuild(provenance.Build{Version: provenance.Development, Revision: "abc123", SourceState: provenance.Clean}, nil)
			draft := detailartifact.Draft{ExecutionID: subject.ExecutionID, Category: "evidence", Outcome: "success", Retention: detailartifact.Evidence, Markdown: []byte("# Scoped output\n"), Provenance: source, References: refs}
			switch mode {
			case "wrong-execution":
				subject.ExecutionID = "123e4567-e89b-42d3-a456-426614174001"
			case "wrong-attempt":
				subject.AttemptID = "attempt-2"
			case "wrong-repository":
				subject.RepositoryKey = "other"
			case "wrong-stage":
				subject.StageID = "other"
			case "wrong-definition":
				subject.DefinitionDigest = "sha256:" + strings.Repeat("b", 64)
			case "wrong-output":
				subject.Check.Name = "other"
			case "wrong-category":
				subject.Check.Kind = "decision"
			case "failed":
				draft.Outcome = "failure"
			case "missing-correlation":
				draft.References = refs[:4]
			case "duplicate-correlation":
				draft.References = append(refs, refs[0])
			case "not-evidence":
				draft.Retention = detailartifact.Active
			}
			artifact, err := store.Create(ctx, draft)
			if err != nil {
				t.Fatal(err)
			}
			digest := hex.EncodeToString(artifact.Digest[:])
			if mode == "wrong-digest" {
				digest = strings.Repeat("c", 64)
			}
			wire, _ := json.Marshal(map[string]string{"kind": "artifact", "id": artifact.ID, "digest": digest})
			switch mode {
			case "plain-file":
				wire = []byte("a file exists")
			case "unknown-field":
				wire = []byte(`{"kind":"artifact","id":"x","digest":"x","approved":true}`)
			case "duplicate-key":
				wire = []byte(`{"kind":"artifact","kind":"artifact","id":"x","digest":"x"}`)
			case "oversize":
				wire = make([]byte, 4097)
			case "cancelled":
				cancelled, cancel := context.WithCancel(ctx)
				cancel()
				ctx = cancelled
			}
			got, err := NewWorkflowReferenceValidator(store).ValidateExecutionOutput(ctx, subject, wire)
			if mode == "schema" || mode == "evidence" {
				if err != nil || got.RetainedID != artifact.ID || got.Digest != "sha256:"+digest {
					t.Fatalf("%+v: %v", got, err)
				}
			} else if err == nil {
				t.Fatal("invalid output accepted")
			}
		})
	}
}

func TestObserveExecutionArtifactRechecksBytesAndLineage(t *testing.T) {
	ctx := context.Background()
	root := privateTestRoot(t)
	x := storeTestExecution(t)
	a := workflow.Attempt{AttemptID: "attempt-1", Authority: workflow.AttemptAuthority{RepositoryKey: "app"}}
	ref := workflow.TaskRef{StageID: "work", AgentID: "builder"}
	content := []byte("observed local result")
	path := filepath.Join(root, "executions", "v2-attempt-outputs", x.ProjectID, x.ExecutionID, a.AttemptID, "result")
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}
	artifact := workflow.ArtifactRef{Name: "result", RepositoryKey: "app", Digest: workflow.DigestText(content), Size: int64(len(content))}
	if err := x.ClaimRunner(1, storeT0); err != nil {
		t.Fatal(err)
	}
	if err := x.PrepareAttempt(workflow.PrepareInput{Ref: ref, AttemptID: a.AttemptID, Generation: 1, Invocation: storeInvocation(), Authority: workflow.AttemptAuthority{RepositoryKey: "app", Actor: "user:local", Effects: []string{}}, Now: storeT0}); err != nil {
		t.Fatal(err)
	}
	if err := x.MarkStarted(a.AttemptID, 1, workflow.ProcessIdentity{PID: 100, StartIdentity: "s", GroupIdentity: "100"}, storeT0); err != nil {
		t.Fatal(err)
	}
	if err := x.CompleteAttempt(a.AttemptID, 1, workflow.AttemptResult{Artifacts: []workflow.ArtifactRef{artifact}, Now: storeT0}); err != nil {
		t.Fatal(err)
	}
	validatorStore, _ := NewArtifactStore(root)
	v := NewWorkflowReferenceValidator(validatorStore)
	if err := v.ObserveExecutionArtifact(ctx, x, ref, a, artifact); err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"digest", "size", "repo", "name", "retained"} {
		t.Run(mode, func(t *testing.T) {
			bad := artifact
			switch mode {
			case "digest":
				bad.Digest = workflow.DigestText([]byte("forged"))
			case "size":
				bad.Size++
			case "repo":
				bad.RepositoryKey = "other"
			case "name":
				bad.Name = "../result"
			case "retained":
				bad.RetainedID = "missing"
			}
			if v.ObserveExecutionArtifact(ctx, x, ref, a, bad) == nil {
				t.Fatal("fabricated artifact accepted")
			}
		})
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if v.ObserveExecutionArtifact(ctx, x, ref, a, artifact) == nil {
		t.Fatal("missing artifact ready")
	}
	outside := filepath.Join(t.TempDir(), "outside")
	if err := os.WriteFile(outside, content, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, path); err != nil {
		t.Fatal(err)
	}
	if v.ObserveExecutionArtifact(ctx, x, ref, a, artifact) == nil {
		t.Fatal("symlink artifact ready")
	}
}

func TestObserveExecutionArtifactRechecksRetainedEvidence(t *testing.T) {
	ctx := context.Background()
	root := privateTestRoot(t)
	store, err := NewArtifactStore(root)
	if err != nil {
		t.Fatal(err)
	}
	definition := workflow.DigestText([]byte("definition"))
	source, _ := provenance.FromBuild(provenance.Build{Version: provenance.Development, Revision: "abc123", SourceState: provenance.Clean}, nil)
	retained, err := store.Create(ctx, detailartifact.Draft{ExecutionID: storeTestExecutionID, Category: "evidence", Outcome: "success", Retention: detailartifact.Evidence, Markdown: []byte("Observed evidence"), Provenance: source, References: []detailartifact.Reference{{Kind: "workflow-stage", Value: "work"}, {Kind: "workflow-attempt", Value: "attempt-1"}, {Kind: "workflow-repository", Value: "app"}, {Kind: "workflow-output", Value: "result"}, {Kind: "workflow-definition", Value: strings.TrimPrefix(definition, "sha256:")}}})
	if err != nil {
		t.Fatal(err)
	}
	artifact := workflow.ArtifactRef{Name: "result", RepositoryKey: "app", RetainedID: retained.ID, Digest: "sha256:" + hex.EncodeToString(retained.Digest[:]), Size: int64(retained.ContentBytes)}
	x, err := workflow.NewExecution(workflow.NewExecutionInput{ExecutionID: storeTestExecutionID, ProjectID: "proj", WorkItem: workflow.WorkItemLink{Provider: "linear", ID: "AXM-42", RepositoryKey: "app"}, Repositories: []workflow.RepositoryBinding{{Key: "app", Identity: "github.com/example/app"}}, WorkflowID: "minimal", Revision: "r1", Now: storeT0, Stages: []workflow.StageSpec{{ID: "work", DefinitionDigest: definition, Agents: []workflow.AgentSpec{{ID: "builder"}}, Outputs: []string{"result"}, OutputChecks: []workflow.OutputCheck{{Name: "result", Kind: "evidence", Evidence: true}}, TimeoutSeconds: 60}}})
	if err != nil {
		t.Fatal(err)
	}
	ref := workflow.TaskRef{StageID: "work", AgentID: "builder"}
	if err := x.ClaimRunner(1, storeT0); err != nil {
		t.Fatal(err)
	}
	if err := x.PrepareAttempt(workflow.PrepareInput{Ref: ref, AttemptID: "attempt-1", Generation: 1, Invocation: storeInvocation(), Authority: workflow.AttemptAuthority{RepositoryKey: "app", Actor: "user:local", Effects: []string{}}, Now: storeT0}); err != nil {
		t.Fatal(err)
	}
	if err := x.MarkStarted("attempt-1", 1, workflow.ProcessIdentity{PID: 100, StartIdentity: "s", GroupIdentity: "100"}, storeT0); err != nil {
		t.Fatal(err)
	}
	if err := x.CompleteAttempt("attempt-1", 1, workflow.AttemptResult{Artifacts: []workflow.ArtifactRef{artifact}, Now: storeT0}); err != nil {
		t.Fatal(err)
	}
	a, _, _ := x.FindAttempt("attempt-1")
	v := NewWorkflowReferenceValidator(store)
	if err := v.ObserveExecutionArtifact(ctx, x, ref, *a, artifact); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "artifacts", "v1", "objects", retained.ID[:2], retained.ID, "details.md")
	if err := os.WriteFile(path, []byte("forged success"), 0600); err != nil {
		t.Fatal(err)
	}
	if v.ObserveExecutionArtifact(ctx, x, ref, *a, artifact) == nil {
		t.Fatal("corrupt retained Evidence ready")
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if v.ObserveExecutionArtifact(ctx, x, ref, *a, artifact) == nil {
		t.Fatal("missing retained Evidence ready")
	}
}

// A child delivers plain content; the engine retains it with the exact
// correlation, so the same typed check accepts it, bound to that attempt only.
func TestRetainExecutionOutputCorrelatesDeliveredContent(t *testing.T) {
	ctx := context.Background()
	store, err := NewArtifactStore(filepath.Join(t.TempDir(), "state"))
	if err != nil {
		t.Fatal(err)
	}
	v := NewWorkflowReferenceValidator(store)
	source, _ := provenance.FromBuild(provenance.Build{Version: provenance.Development, Revision: "abc123", SourceState: provenance.Clean}, nil)
	for _, evidence := range []bool{false, true} {
		subject := workflow.OutputSubject{ExecutionID: "123e4567-e89b-42d3-a456-426614174000", AttemptID: "attempt-1", StageID: "work", DefinitionDigest: "sha256:" + strings.Repeat("a", 64), RepositoryKey: "main", Check: workflow.OutputCheck{Name: "result", Kind: "artifact", Evidence: evidence}}
		ref, err := v.RetainExecutionOutput(ctx, subject, []byte("# Intake\n\nBounded intent.\n"), source)
		if err != nil || ref.Name != "result" || ref.RepositoryKey != "main" || ref.RetainedID == "" {
			t.Fatal(ref, err)
		}
		wire, _ := json.Marshal(map[string]string{"kind": "artifact", "id": ref.RetainedID, "digest": strings.TrimPrefix(ref.Digest, "sha256:")})
		other := subject
		other.AttemptID = "attempt-2"
		if _, err := v.ValidateExecutionOutput(ctx, other, wire); err == nil {
			t.Fatal("retained output replayed into another attempt")
		}
	}
	if _, err := v.RetainExecutionOutput(ctx, workflow.OutputSubject{ExecutionID: "123e4567-e89b-42d3-a456-426614174000", AttemptID: "attempt-1", StageID: "work", DefinitionDigest: "sha256:" + strings.Repeat("a", 64), RepositoryKey: "main", Check: workflow.OutputCheck{Name: "result", Kind: "artifact"}}, []byte("api_key: leaked\n"), source); err == nil {
		t.Fatal("unsafe content retained")
	}
}
