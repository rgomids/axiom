package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"reflect"
	"strconv"
	"testing"

	"github.com/rgomids/axiom/internal/completion"
	"github.com/rgomids/axiom/internal/projectapp"
	"github.com/rgomids/axiom/internal/provenance"
)

// Synthetic wrappers must match the real producer's read-only show shape.
// No installed CLI, Project store or host is touched by these tests.
func TestWorkflowShowSyntheticPresentationEnvelopeMatchesProducer(t *testing.T) {
	for _, name := range []string{"default-sdd-r1", "default-sdd-r2", "custom-r1", "custom-r2"} {
		t.Run(name, func(t *testing.T) {
			wire, err := os.ReadFile("../codexruntime/testdata/rendered-workflow/" + name + ".json")
			if err != nil {
				t.Fatal(err)
			}
			var fixture struct {
				Report     projectapp.WorkflowReport `json:"workflowAuthoring"`
				References []string                  `json:"references"`
				Provenance struct {
					Product     string                 `json:"product"`
					Version     string                 `json:"version"`
					Revision    string                 `json:"revision"`
					SourceState provenance.SourceState `json:"sourceState"`
				} `json:"provenance"`
			}
			if err = json.Unmarshal(wire, &fixture); err != nil {
				t.Fatal(err)
			}
			before, _ := json.Marshal(fixture.Report)
			result := presentationResult(t, completion.Facts{Completed: true}, "Project workflow inspected", fixture.References, "Inspect exact definition; no mutation", "")
			source, err := provenance.FromBuild(provenance.Build{Version: fixture.Provenance.Version, Revision: fixture.Provenance.Revision, SourceState: fixture.Provenance.SourceState}, nil)
			if err != nil || source.Product() != fixture.Provenance.Product || source.Version() != fixture.Provenance.Version {
				t.Fatal("fixture provenance is not producer-valid", err)
			}
			withSource, err := completion.New(completion.Facts{Completed: true}, result.Result(), fixture.References, result.Next(), "", source)
			if err != nil {
				t.Fatal(err)
			}

			result = &withSource
			service := &workflowPresentationService{response: Result{Completion: result, Category: "inspected", WorkflowAuthoring: &fixture.Report}}
			ref := fixture.Report.Reference
			args := []string{"project", "workflow", "show", "--project", fixture.Report.ProjectID, "--workflow", ref.WorkflowID, "--revision", strconv.Itoa(ref.Revision), "--digest", ref.Digest, "--source", ref.Source}
			var structured, human bytes.Buffer
			for _, view := range []struct {
				mode outputMode
				out  *bytes.Buffer
			}{{jsonOutput, &structured}, {humanOutput, &human}} {
				handled, code := runProjectWorkflow(context.Background(), view.mode, args, service, completionProvenance(t), view.out)
				if !handled || code != 0 {
					t.Fatalf("show handled=%v exit=%d", handled, code)
				}
			}
			var got struct {
				Status   string                    `json:"status"`
				Category string                    `json:"category"`
				Report   projectapp.WorkflowReport `json:"workflowAuthoring"`
			}
			if err = json.Unmarshal(structured.Bytes(), &got); err != nil {
				t.Fatal(err)
			}
			if got.Status != "success" || got.Category != "inspected" || !reflect.DeepEqual(got.Report, fixture.Report) {
				t.Fatal("fixture differs from producer")
			}
			var wantEnvelope, actualEnvelope any
			if err := json.Unmarshal(wire, &wantEnvelope); err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(structured.Bytes(), &actualEnvelope); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(wantEnvelope, actualEnvelope) {
				t.Fatal("complete synthetic envelope differs from producer")
			}

			native, err := renderTerminal(structured.Bytes())
			if err != nil || !bytes.Equal(native, human.Bytes()) {
				t.Fatal("native terminal diverged", err)
			}
			after, _ := json.Marshal(fixture.Report)
			if !bytes.Equal(before, after) || service.calls != 2 {
				t.Fatal("inspection mutated input or repeated operation")
			}
		})
	}
}
