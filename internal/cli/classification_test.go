package cli

import (
	"bytes"
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/completion"
)

func TestCreateFlagsTransportTypeStoryValueAndRepeatedClassifications(t *testing.T) {
	service := &guidedWorkItemService{completion: canonicalResult(t, completion.Success, []string{"operation:work_item_create"}, "Review result", completionProvenance(t))}
	var output bytes.Buffer
	code := Run(context.Background(), []string{"work-item", "create", "--project", "alpha", "--repository", "main", "--provider-repository", "owner/repo", "--type", "story", "--beneficiary", "Maintainers", "--value", "Prioritize delivery by user value", "--classification", "type:story", "--classification", "area:cli"}, service, completionProvenance(t), &output)
	if code != ExitSuccess || len(service.inputs) != 1 {
		t.Fatalf("code=%d inputs=%+v output=%s", code, service.inputs, output.String())
	}
	input := service.inputs[0]
	if input.Type != "story" || input.Beneficiary != "Maintainers" || input.Value != "Prioritize delivery by user value" || !reflect.DeepEqual(input.Classification, []string{"type:story", "area:cli"}) || input.AuthorizeExternal {
		t.Fatalf("input=%+v", input)
	}
}

func TestGuidedStoryCollectsValueBeforeExactPreviewAuthorization(t *testing.T) {
	service := &guidedWorkItemService{completion: canonicalResult(t, completion.Success, []string{"operation:work_item_create"}, "Review result", completionProvenance(t))}
	input := strings.NewReader("alpha\nmain\nowner/repo\nUnclear delivery intent\nstory\nVisible outcomes\nContext\nImplement classification\nPreserve authority\nNo new labels\nTests pass\nMaintainers\nPrioritize meaningful delivery outcomes\nyes\n")
	var output, prompts bytes.Buffer
	code := RunInteractive(context.Background(), []string{"--json", "work-item", "create"}, service, completionProvenance(t), input, &output, &prompts)
	if code != ExitSuccess || len(service.inputs) != 2 {
		t.Fatalf("code=%d inputs=%+v output=%s", code, service.inputs, output.String())
	}
	for _, input := range service.inputs {
		if input.Type != "story" || input.Beneficiary != "Maintainers" || input.Value != "Prioritize meaningful delivery outcomes" {
			t.Fatalf("input=%+v", input)
		}
	}
	if service.inputs[0].AuthorizeExternal || !service.inputs[1].AuthorizeExternal || !strings.Contains(prompts.String(), "Concrete user/product benefit") {
		t.Fatalf("inputs=%+v prompts=%s", service.inputs, prompts.String())
	}
}
