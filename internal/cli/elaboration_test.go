package cli

import (
	"bytes"
	"context"
	"io"
	"reflect"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/completion"
)

func TestElaboratedSectionsRejectAmbiguousOrMalformedInputBeforeDispatch(t *testing.T) {
	for _, args := range [][]string{
		{"--elaborated-section", "unknown=value"},
		{"--elaborated-section", "context="},
		{"--elaborated-section", "context"},
		{"--elaborated-section", "context=first", "--elaborated-section", "context=second"},
		{"-elaborated-section", "context=value"},
	} {
		service := &guidedWorkItemService{}
		var output bytes.Buffer
		if code := Run(context.Background(), append([]string{"work-item", "create"}, args...), service, completionProvenance(t), &output); code != ExitFailure || len(service.inputs) != 0 {
			t.Fatalf("args=%q code=%d inputs=%#v", args, code, service.inputs)
		}
	}
}

func TestInterviewReusesSuppliedAndElaboratedSectionsBeforeExactAuthority(t *testing.T) {
	for _, decision := range []string{"yes", "no", ""} {
		t.Run("decision="+decision, func(t *testing.T) {
			service := &guidedWorkItemService{completion: canonicalResult(t, completion.Success, nil, "Review result", completionProvenance(t))}
			args := []string{"--json", "work-item", "create", "--project", "alpha", "--repository", "main", "--provider-repository", "owner/repo", "--type", "task", "--intent", "Export fails", "--desired-outcome", "Export succeeds", "--elaborated-section", "context=No additional context supplied", "--elaborated-section", "scope=Restore export", "--elaborated-section", "constraints=Preserve existing behavior", "--elaborated-section", "non_goals=No unrelated changes"}
			var output, prompts bytes.Buffer
			input := strings.NewReader("A saved export contains all selected rows\n" + decision + "\n")
			code := RunInteractive(context.Background(), args, service, completionProvenance(t), input, &output, &prompts)
			if code != ExitSuccess || len(service.inputs) != 2 {
				t.Fatalf("code=%d inputs=%#v output=%s", code, service.inputs, output.String())
			}
			first, second := service.inputs[0], service.inputs[1]
			if first.Intent != "Export fails" || first.DesiredOutcome != "Export succeeds" || first.Acceptance != "A saved export contains all selected rows" || first.AuthorizeExternal {
				t.Fatalf("preview input=%#v", first)
			}
			if !reflect.DeepEqual(first.ElaboratedSections, second.ElaboratedSections) {
				t.Fatal("elaboration lost across authority boundary")
			}
			if decision == "yes" {
				if second.PreviewDigest != "reviewed-digest" || !second.AuthorizeExternal {
					t.Fatalf("authorization=%#v", second)
				}
			} else if !second.Cancelled || second.AuthorizeExternal {
				t.Fatalf("cancellation=%#v", second)
			}
			if strings.Contains(prompts.String(), "Where or when") || strings.Contains(prompts.String(), "What should work differently") || !strings.Contains(prompts.String(), "How could we check") {
				t.Fatalf("missing-only prompts=%s", prompts.String())
			}
			if strings.Index(prompts.String(), "reviewed-digest") > strings.Index(prompts.String(), "Publish this create-attempt fence") {
				t.Fatal("authority asked before preview")
			}
		})
	}
}

func TestCompleteElaboratedDraftNeedsNoInterviewAndRemainsReadOnly(t *testing.T) {
	service := &guidedWorkItemService{completion: canonicalResult(t, completion.Success, nil, "Review result", completionProvenance(t))}
	args := []string{"--json", "work-item", "create", "--project", "alpha", "--repository", "main", "--provider-repository", "owner/repo", "--type", "task", "--intent", "Export fails"}
	for _, field := range []string{"desired_outcome", "context", "scope", "constraints", "non_goals", "acceptance_expectations"} {
		args = append(args, "--elaborated-section", field+"=Draft content with = sign")
	}
	var output bytes.Buffer
	if code := RunInteractive(context.Background(), args, service, completionProvenance(t), nil, &output, io.Discard); code != ExitSuccess || len(service.inputs) != 1 || service.inputs[0].AuthorizeExternal || len(service.inputs[0].ElaboratedSections) != 6 {
		t.Fatalf("inputs=%#v output=%s", service.inputs, output.String())
	}
}
