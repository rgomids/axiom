package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/completion"
	"github.com/rgomids/axiom/internal/projectapp"
	"github.com/rgomids/axiom/internal/workflowdefinition"
)

type workflowPresentationService struct {
	Service
	response Result
	calls    int
}

func (s *workflowPresentationService) ProjectWorkflow(context.Context, ProjectWorkflowInput) Result {
	s.calls++
	return s.response
}

func TestWorkflowAuthoringPresentsExactCanonicalEvent(t *testing.T) {
	for _, scenario := range []struct {
		name  string
		facts completion.Facts
	}{
		{"previewed", completion.Facts{Completed: true}},
		{"stale_authority", completion.Facts{AuthorityDenied: true}},
		{"partial", completion.Facts{RequestedEffectConfirmed: true, SecondaryFailure: true}},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			definition := workflowdefinition.Builtin().Definition
			result := presentationResult(t, scenario.facts, "Project workflow "+scenario.name, []string{"workflow:custom/1"}, "Review the exact preview", "Preserved canonical details")
			service := &workflowPresentationService{response: Result{Completion: result, Category: scenario.name, WorkflowAuthoring: &projectapp.WorkflowReport{
				ProjectID: "123e4567-e89b-42d3-a456-426614174000", Definition: &definition, PreviewDigest: strings.Repeat("a", 64), Effects: []string{"publish_immutable_definition", "append_revision_index"},
			}}}
			var structured, human bytes.Buffer
			args := []string{"project", "workflow", "create", "--project", "alpha", "--from-default", "--workflow", "custom"}
			for _, view := range []struct {
				mode   outputMode
				output *bytes.Buffer
			}{{jsonOutput, &structured}, {humanOutput, &human}} {
				handled, code := runProjectWorkflow(context.Background(), view.mode, args, service, completionProvenance(t), view.output)
				if !handled || code != completionExitCode(result.Status()) {
					t.Fatalf("handled=%v exit=%d", handled, code)
				}
			}
			if !json.Valid(structured.Bytes()) {
				t.Fatal("machine output is not canonical JSON")
			}
			expected, err := renderMarkdown(structured.Bytes())
			if err != nil || !bytes.Equal(expected, human.Bytes()) {
				t.Fatalf("human workflow output diverges from canonical renderer: %v", err)
			}
			if !strings.Contains(human.String(), "workflowAuthoring") || !strings.Contains(human.String(), strings.Repeat("a", 64)) {
				t.Fatal("workflow payload lost")
			}
			if service.calls != 2 {
				t.Fatal("presentation repeated an application operation")
			}
		})
	}
}
