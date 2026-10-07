package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/completion"
	"github.com/rgomids/axiom/internal/runtimeapplication"
	"github.com/rgomids/axiom/internal/runtimeprofile"
)

type policyRecordingService struct {
	recordingService
	input  RuntimeProfilePreviewInput
	calls  int
	result Result
}

func (s *policyRecordingService) RuntimeProfilePreview(_ context.Context, input RuntimeProfilePreviewInput) Result {
	s.input = input
	s.calls++
	return s.result
}

func TestRuntimePreviewStrictReadOnlySurface(t *testing.T) {
	base := []string{"runtime", "profile", "preview", "--project", "sample", "--role", "implementation", "--complexity", "high", "--capabilities", "code,review", "--observations", filepath.Join(t.TempDir(), "inventory.json")}
	service := &policyRecordingService{result: Result{Status: Succeeded, Category: "preview"}}
	var output bytes.Buffer
	if code := Run(context.Background(), base, service, completionProvenance(t), &output); code != 0 || service.calls != 1 || service.input.Runtime != "" || len(service.input.Capabilities) != 2 {
		t.Fatalf("code=%d service=%+v", code, service)
	}
	for _, extra := range [][]string{{"--authorize-local"}, {"--preview-digest", "private-input"}, {"--runtime", ""}, {"--runtime", "other"}, {"--role", "private-input"}, {"-runtime", "claude"}, {"--observations", "relative.json"}, {"--"}, {"private-input"}} {
		service.calls = 0
		output.Reset()
		code := Run(context.Background(), append(append([]string{}, base...), extra...), service, completionProvenance(t), &output)
		if code != 1 || service.calls != 0 || strings.Contains(output.String(), "private-input") {
			t.Fatalf("args=%v code=%d calls=%d output=%s", extra, code, service.calls, &output)
		}
	}
	for _, name := range []string{"role", "complexity", "capabilities", "observations", "project"} {
		args := append([]string{}, base...)
		for index, value := range args {
			if value == "--"+name {
				args = append(args[:index], args[index+2:]...)
				break
			}
		}
		service.calls = 0
		output.Reset()
		if code := Run(context.Background(), args, service, completionProvenance(t), &output); code != 1 || service.calls != 0 {
			t.Fatalf("missing %s: code=%d calls=%d", name, code, service.calls)
		}
	}
}

func TestRuntimeResolutionCompletionHumanAndJSON(t *testing.T) {
	result := canonicalResult(t, completion.Success, nil, "", completionProvenance(t))
	preview := runtimeapplication.Preview{ProjectID: "project", Request: runtimeapplication.Request{Role: "implementation", Complexity: "high", Capabilities: []string{"code"}}, Choice: &runtimeprofile.Choice{RuntimeID: "claude", ModelProfileID: "worker", Model: "approved-model", RuntimeVersion: "1.0", ConfigurationRevision: 1, ObservationRevision: 2}}
	response := Result{Completion: &result, RuntimeResolution: &preview, PreviewDigest: preview.Digest()}
	for _, mode := range []outputMode{humanOutput, jsonOutput} {
		var output bytes.Buffer
		if code := emitResponse(&output, mode, runtimeProfilePreviewAction, response); code != 0 || output.Len() > MaxCompletionOutputBytes {
			t.Fatalf("code=%d output=%s", code, &output)
		}
		if !strings.Contains(output.String(), "runtimeResolution") || !strings.Contains(output.String(), preview.Digest()) {
			t.Fatalf("output=%s", &output)
		}
		if mode == jsonOutput {
			var event runtimeResolutionCompletionEvent
			if err := json.Unmarshal(output.Bytes(), &event); err != nil || event.RuntimeResolution.Choice.RuntimeID != "claude" {
				t.Fatalf("event=%+v err=%v", event, err)
			}
		}
	}
}

func TestRuntimeResolutionCompletionRejectsOversizedOutput(t *testing.T) {
	result := canonicalResult(t, completion.Success, nil, "", completionProvenance(t))
	preview := runtimeapplication.Preview{Choice: &runtimeprofile.Choice{Model: strings.Repeat("x", MaxCompletionOutputBytes)}}
	response := Result{Completion: &result, RuntimeResolution: &preview, PreviewDigest: preview.Digest()}
	for _, mode := range []outputMode{humanOutput, jsonOutput} {
		var output bytes.Buffer
		if code := emitResponse(&output, mode, runtimeProfilePreviewAction, response); code != ExitFailure || output.Len() != 0 {
			t.Fatalf("code=%d bytes=%d", code, output.Len())
		}
	}
}

func TestWorkflowStartForwardsExplicitPolicyInputs(t *testing.T) {
	path := filepath.Join(t.TempDir(), "observations.json")
	digest := strings.Repeat("a", 64)
	values, ok := workflowFlags(workflowStartAction, []string{"--project", "sample", "--repository", "main", "--number", "7", "--role", "implementation", "--complexity", "high", "--capabilities", "code,review", "--observations", path, "--runtime-preview", digest})
	if !ok || values.runtime != "" || values.role != "implementation" || values.complexity != "high" || values.capabilities != "code,review" || values.observations != path || values.runtimePreview != digest {
		t.Fatalf("ok=%v values=%+v", ok, values)
	}
	for _, name := range []string{"role", "complexity", "capabilities", "observations", "runtime-preview"} {
		assertStrictParserFailure(t, []string{"workflow", "start", "--" + name, "first", "--" + name, "second"})
	}
}
