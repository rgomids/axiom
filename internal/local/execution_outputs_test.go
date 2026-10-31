package local

import (
	"context"
	"encoding/hex"
	"encoding/json"
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
