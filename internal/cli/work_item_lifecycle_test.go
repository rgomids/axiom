package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/completion"
	"github.com/rgomids/axiom/internal/workitem"
)

type lifecycleRecordingService struct {
	recordingService
	inputs  []WorkItemInput
	calls   []string
	respond Result
}

func (s *lifecycleRecordingService) record(call string, input WorkItemInput) Result {
	s.calls = append(s.calls, call)
	s.inputs = append(s.inputs, input)
	return s.respond
}
func (s *lifecycleRecordingService) WorkItemList(_ context.Context, input WorkItemInput) Result {
	return s.record("list", input)
}
func (s *lifecycleRecordingService) WorkItemUpdate(_ context.Context, input WorkItemInput) Result {
	return s.record("update", input)
}
func (s *lifecycleRecordingService) WorkItemClose(_ context.Context, input WorkItemInput) Result {
	return s.record("close", input)
}
func (s *lifecycleRecordingService) WorkItemReopen(_ context.Context, input WorkItemInput) Result {
	return s.record("reopen", input)
}

func TestWorkItemLifecycleVerbsParseOnlyTheirDeclaredInputs(t *testing.T) {
	service := &lifecycleRecordingService{respond: Result{Status: Succeeded, Category: "applied"}}
	exact := []string{"--project", "alpha", "--repository", "main", "--work-item", "github:owner/repo#7"}
	for _, args := range [][]string{
		{"work-item", "list", "--project", "alpha"},
		{"work-item", "list", "--project", "alpha", "--repository", "main"},
		append(append([]string{"work-item", "update"}, exact...), "--title", "New title", "--scope", "Scope", "--elaborated-section", "context=Elaborated", "--preview-digest", strings.Repeat("a", 64), "--authorize-external"),
		append([]string{"work-item", "close"}, exact...),
		{"work-item", "reopen", "--project", "alpha", "--repository", "main", "--provider-repository", "owner/repo", "--number", "7", "--authorize-external"},
	} {
		var output bytes.Buffer
		if code := Run(context.Background(), args, service, completionProvenance(t), &output); code != ExitSuccess {
			t.Fatalf("%v: exit=%d output=%s", args, code, output.String())
		}
	}
	if strings.Join(service.calls, ",") != "list,list,update,close,reopen" {
		t.Fatalf("calls = %v", service.calls)
	}
	if list := service.inputs[1]; list.Project != "alpha" || list.Repository != "main" || list.ExternalID != "" || list.ProviderRepository != "" {
		t.Fatalf("list input = %+v", list)
	}
	update := service.inputs[2]
	if update.Title != "New title" || update.Scope != "Scope" || update.ElaboratedSections["context"] != "Elaborated" || update.Provider != "github" || update.ProviderRepository != "owner/repo" || update.ExternalID != "7" || !update.AuthorizeExternal || update.PreviewDigest == "" {
		t.Fatalf("update input = %+v", update)
	}
	if reopen := service.inputs[4]; reopen.ExternalID != "7" || reopen.ProviderRepository != "owner/repo" || !reopen.AuthorizeExternal {
		t.Fatalf("reopen input = %+v", reopen)
	}
}

func TestWorkItemLifecycleRejectsUndeclaredInputsAndUnknownVerbs(t *testing.T) {
	for _, args := range [][]string{
		{"work-item", "list"},
		{"work-item", "list", "--project", "alpha", "--number", "7"},
		{"work-item", "list", "--project", "alpha", "--authorize-external"},
		{"work-item", "update", "--project", "alpha", "--repository", "main", "--number", "7", "--type", "bug"},
		{"work-item", "update", "--project", "alpha", "--repository", "main", "--number", "7", "--classification", "area:cli"},
		{"work-item", "update", "--project", "alpha", "--repository", "main", "--number", "7", "--label", "x"},
		{"work-item", "close", "--project", "alpha", "--repository", "main", "--number", "7", "--title", "x"},
		{"work-item", "close", "--project", "alpha", "--repository", "main"},
		{"work-item", "reopen", "--project", "alpha", "--repository", "main", "--work-item", "gitlab:owner/repo#7"},
	} {
		var output bytes.Buffer
		service := &lifecycleRecordingService{}
		if code := Run(context.Background(), args, service, completionProvenance(t), &output); code != ExitFailure || len(service.calls) != 0 {
			t.Fatalf("%v: exit=%d calls=%v output=%s", args, code, service.calls, output.String())
		}
	}
	for _, verb := range []string{"delete", "remove", "edit"} {
		var output bytes.Buffer
		service := &lifecycleRecordingService{}
		code := Run(context.Background(), []string{"work-item", verb, "--project", "alpha", "--repository", "main", "--number", "7"}, service, completionProvenance(t), &output)
		if code != ExitFailure || !strings.Contains(output.String(), `"category":"invalid_command"`) || len(service.calls) != 0 {
			t.Fatalf("%s: exit=%d output=%s", verb, code, output.String())
		}
	}
}

func TestWorkItemLifecycleRequiresTheOptionalService(t *testing.T) {
	var output bytes.Buffer
	if code := Run(context.Background(), []string{"work-item", "list", "--project", "alpha"}, &recordingService{}, completionProvenance(t), &output); code != ExitFailure || !strings.Contains(output.String(), "application_unavailable") {
		t.Fatalf("exit=%d output=%s", code, output.String())
	}
}

func TestWorkItemLifecycleEmitsChangeAndListPayloads(t *testing.T) {
	source := completionProvenance(t)
	result := canonicalResult(t, completion.Success, nil, "Review result", source)
	change := &workitem.ChangePreview{Operation: workitem.ChangeClose, Effects: []string{"set_provider_work_item_state"}, Digest: strings.Repeat("d", 64)}
	for name, test := range map[string]struct {
		response Result
		check    func(map[string]json.RawMessage) bool
	}{
		"empty list": {Result{Completion: &result, Category: "work_items_listed", WorkItems: []WorkItemView{}}, func(event map[string]json.RawMessage) bool {
			return string(event["workItems"]) == "[]" && string(event["category"]) == `"work_items_listed"`
		}},
		"change": {Result{Completion: &result, Category: "work_item_close_ready", WorkItemChange: change}, func(event map[string]json.RawMessage) bool {
			return strings.Contains(string(event["change"]), strings.Repeat("d", 64)) && event["workItems"] == nil
		}},
	} {
		service := &lifecycleRecordingService{respond: test.response}
		var output bytes.Buffer
		if code := Run(context.Background(), []string{"work-item", "list", "--project", "alpha"}, service, source, &output); code != ExitSuccess {
			t.Fatalf("%s: exit=%d", name, code)
		}
		var event map[string]json.RawMessage
		if err := json.Unmarshal(output.Bytes(), &event); err != nil || !test.check(event) {
			t.Fatalf("%s: %s %v", name, output.String(), err)
		}
		var human bytes.Buffer
		if code := RunInteractive(context.Background(), []string{"--human", "work-item", "list", "--project", "alpha"}, service, source, nil, &human, &bytes.Buffer{}); code != ExitSuccess || !strings.Contains(human.String(), "category: "+test.response.Category) {
			t.Fatalf("%s human: exit=%d %s", name, code, human.String())
		}
	}
}
