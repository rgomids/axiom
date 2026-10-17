package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/completion"
	"github.com/rgomids/axiom/internal/provenance"
)

// stageRecorder is a WorkflowStageService that records the exact input it
// received and returns a canonical completion with an operation payload.
type stageRecorder struct {
	recordingService
	inputs []WorkflowStageInput
	status completion.Facts
	source provenance.Value
}

func (s *stageRecorder) WorkflowStage(_ context.Context, in WorkflowStageInput) WorkflowStageResponse {
	s.inputs = append(s.inputs, in)
	statement, _ := provenance.NewText("Stage operation result", provenance.AxiomAuthored)
	next, _ := provenance.NewText("Review the preview", provenance.AxiomAuthored)
	result, err := completion.New(s.status, statement, []string{"stage-preview:" + strings.Repeat("a", 64)}, next, "", s.source)
	if err != nil {
		panic(err)
	}
	return WorkflowStageResponse{Result: Result{Completion: &result, Category: "stage_preview_ready"}, Payload: map[string]any{"operation": in.Operation, "previewDigest": strings.Repeat("a", 64)}}
}

const stageSelectors = "--project alpha --repository main --work-item github:owner/repo#7 --execution e-1"

func TestWorkflowStageCommandsRouteExactInputs(t *testing.T) {
	cases := map[string]struct {
		args  string
		check func(WorkflowStageInput) bool
	}{
		"run preview": {"workflow stage run " + stageSelectors + " --expected-revision 3 --stage implementation --plan p.json --validator-policy v --workspace-root /w", func(in WorkflowStageInput) bool {
			return in.Operation == "run" && in.ExpectedRevision == 3 && !in.AuthorizeLocal && !in.AuthorizeExternal && in.PreviewDigest == ""
		}},
		"run apply": {"workflow stage run " + stageSelectors + " --expected-revision 3 --stage implementation --plan p.json --validator-policy v --workspace-root /w --preview-digest d --authorize-local --authorize-external", func(in WorkflowStageInput) bool {
			return in.PreviewDigest == "d" && in.AuthorizeLocal && in.AuthorizeExternal
		}},
		"delivery packet": {"workflow delivery packet " + stageSelectors + " --packet digest", func(in WorkflowStageInput) bool { return in.Operation == "delivery-packet" && in.Record == "digest" }},
		"status":          {"workflow stage status " + stageSelectors, func(in WorkflowStageInput) bool { return in.Operation == "status" && in.Execution == "e-1" }},
		"cancel": {"workflow stage cancel " + stageSelectors + " --preview-digest d --authorize-local", func(in WorkflowStageInput) bool {
			return in.Operation == "cancel" && in.AuthorizeLocal && !in.AuthorizeExternal
		}},
		"retry": {"workflow stage retry " + stageSelectors + " --expected-revision 3 --stage implementation --plan p.json --validator-policy v --child c-1 --disposition preserve", func(in WorkflowStageInput) bool {
			return in.Operation == "retry" && in.Child == "c-1" && in.Disposition == "preserve"
		}},
		"reconcile": {"workflow stage reconcile " + stageSelectors + " --child c-1 --attempt a-1", func(in WorkflowStageInput) bool { return in.Operation == "reconcile" && in.Attempt == "a-1" }},
		"integrate": {"workflow stage integrate " + stageSelectors + " --expected-revision 3 --stage implementation --validator-policy v --waive c-2=waiver:1 --waive c-3=waiver:2", func(in WorkflowStageInput) bool {
			return in.Operation == "integrate" && len(in.Waivers) == 2 && in.Waivers["c-3"] == "waiver:2"
		}},
		"coord list": {"workflow coordination list " + stageSelectors + " --child c-1 --kind result --wait 2s", func(in WorkflowStageInput) bool {
			return in.Operation == "coordination-list" && in.Kind == "result" && in.Wait.Seconds() == 2
		}},
		"coord show": {"workflow coordination show " + stageSelectors + " --child c-1 --record r-1", func(in WorkflowStageInput) bool { return in.Operation == "coordination-show" && in.Record == "r-1" }},
		"coord publish": {"workflow coordination publish --kind result --field outcome=succeeded --field validation_reference=v:1", func(in WorkflowStageInput) bool {
			return in.Operation == "coordination-publish" && len(in.Fields) == 2 && in.Project == "" && in.Execution == ""
		}},
	}
	for name, tt := range cases {
		t.Run(name, func(t *testing.T) {
			service := &stageRecorder{status: completion.Facts{Completed: true}, source: completionProvenance(t)}
			var out bytes.Buffer
			code := Run(context.Background(), strings.Fields(tt.args), service, completionProvenance(t), &out)
			if code != ExitSuccess || len(service.inputs) != 1 || !tt.check(service.inputs[0]) {
				t.Fatalf("code=%d inputs=%+v out=%s", code, service.inputs, &out)
			}
		})
	}
}

func TestWorkflowStageParserRefusals(t *testing.T) {
	for _, args := range []string{
		"workflow stage run " + stageSelectors + " --stage implementation --plan p.json --validator-policy v",            // missing revision
		"workflow stage retry " + stageSelectors + " --expected-revision 3 --stage s --plan p.json --validator-policy v", // missing child
		"workflow stage reconcile " + stageSelectors + " --child c-1",                                                    // missing attempt
		"workflow stage integrate " + stageSelectors + " --expected-revision 3 --stage s --validator-policy v --waive bad",
		"workflow stage integrate " + stageSelectors + " --expected-revision 3 --stage s --validator-policy v --waive c=1 --waive c=2",
		"workflow coordination list " + stageSelectors + " --wait 31s",
		"workflow coordination show " + stageSelectors + " --child c-1",
		"workflow coordination publish --kind result --project alpha", // lineage never comes from flags
		"workflow coordination publish --field outcome=succeeded",
		"workflow stage cancel " + stageSelectors + " --authorize-external",
		"workflow stage status " + stageSelectors + " --preview-digest d",
		"workflow stage status " + stageSelectors + " --execution e-2",
	} {
		t.Run(args, func(t *testing.T) {
			service := &stageRecorder{status: completion.Facts{Completed: true}, source: completionProvenance(t)}
			var out bytes.Buffer
			code := Run(context.Background(), strings.Fields(args), service, completionProvenance(t), &out)
			if code == ExitSuccess || len(service.inputs) != 0 || !strings.Contains(out.String(), "validation_failure") {
				t.Fatalf("accepted: code=%d out=%s", code, &out)
			}
		})
	}
}

func TestWorkflowCancelRemainsInvalidCommand(t *testing.T) {
	service := &stageRecorder{status: completion.Facts{Completed: true}, source: completionProvenance(t)}
	var out bytes.Buffer
	code := Run(context.Background(), []string{"workflow", "cancel", "--project", "alpha"}, service, completionProvenance(t), &out)
	if code == ExitSuccess || len(service.inputs) != 0 || !strings.Contains(out.String(), "invalid_command") {
		t.Fatalf("workflow cancel = %d %s", code, &out)
	}
}

func TestWorkflowStageJSONAndHumanParity(t *testing.T) {
	args := strings.Fields("workflow stage status " + stageSelectors)
	service := &stageRecorder{status: completion.Facts{Completed: true}, source: completionProvenance(t)}
	var jsonOut, humanOut bytes.Buffer
	if code := Run(context.Background(), args, service, completionProvenance(t), &jsonOut); code != ExitSuccess {
		t.Fatal(jsonOut.String())
	}
	if code := RunInteractive(context.Background(), args, service, completionProvenance(t), nil, &humanOut, io.Discard); code != ExitSuccess {
		t.Fatal(humanOut.String())
	}
	var event map[string]any
	if err := json.Unmarshal(jsonOut.Bytes(), &event); err != nil {
		t.Fatal(err)
	}
	stage, ok := event["stage"].(map[string]any)
	if !ok || event["category"] != "stage_preview_ready" || stage["operation"] != "status" {
		t.Fatalf("json = %s", &jsonOut)
	}
	for _, want := range []string{"Stage operation result", "stage_preview_ready", strings.Repeat("a", 64), "status"} {
		if !strings.Contains(humanOut.String(), want) {
			t.Fatalf("human output lacks %q: %s", want, &humanOut)
		}
	}
}

func TestWorkflowStageExitCodesFollowCanonicalStatus(t *testing.T) {
	for _, tt := range []struct {
		facts completion.Facts
		code  int
	}{{completion.Facts{Completed: true}, ExitSuccess}, {completion.Facts{AuthorityDenied: true}, ExitFailure}, {completion.Facts{ValidationFailed: true}, ExitFailure}} {
		service := &stageRecorder{status: tt.facts, source: completionProvenance(t)}
		var out bytes.Buffer
		if code := Run(context.Background(), strings.Fields("workflow stage status "+stageSelectors), service, completionProvenance(t), &out); code != tt.code {
			t.Fatalf("facts %+v exit %d, want %d: %s", tt.facts, code, tt.code, &out)
		}
	}
}

func TestWorkflowStageHelpAndUnavailableApplication(t *testing.T) {
	for _, path := range []string{"workflow stage run", "workflow stage status", "workflow stage cancel", "workflow stage retry", "workflow stage reconcile", "workflow stage integrate", "workflow coordination list", "workflow coordination show", "workflow coordination publish"} {
		var help bytes.Buffer
		if handled, code := HandleHelp(append(strings.Fields(path), "--help"), completionProvenance(t), &help); !handled || code != ExitSuccess || !strings.Contains(help.String(), "axiom "+path) {
			t.Fatalf("%s help: handled=%v code=%d %s", path, handled, code, &help)
		}
	}
	var out bytes.Buffer
	code := Run(context.Background(), strings.Fields("workflow stage status "+stageSelectors), &recordingService{}, completionProvenance(t), &out)
	if code == ExitSuccess || !strings.Contains(out.String(), "application_unavailable") {
		t.Fatalf("missing service = %d %s", code, &out)
	}
}

type packetRecorder struct{ stageRecorder }

func (s *packetRecorder) WorkflowStage(ctx context.Context, in WorkflowStageInput) WorkflowStageResponse {
	response := s.stageRecorder.WorkflowStage(ctx, in)
	response.Payload = map[string]any{"packetDigest": strings.Repeat("c", 64), "packet": map[string]any{"ready": false, "blockers": []string{"review:review_unverified"}, "next": "resolve packet blockers"}}
	return response
}

func TestDeliveryPacketJSONAndHumanExposeSameReadiness(t *testing.T) {
	args := strings.Fields("workflow delivery packet " + stageSelectors)
	service := &packetRecorder{stageRecorder: stageRecorder{status: completion.Facts{Completed: true}, source: completionProvenance(t)}}
	var jsonOut, humanOut bytes.Buffer
	if code := Run(context.Background(), args, service, completionProvenance(t), &jsonOut); code != ExitSuccess {
		t.Fatal(jsonOut.String())
	}
	if code := RunInteractive(context.Background(), args, service, completionProvenance(t), nil, &humanOut, io.Discard); code != ExitSuccess {
		t.Fatal(humanOut.String())
	}
	for _, value := range []string{strings.Repeat("c", 64), "review:review_unverified", "resolve packet blockers", "false"} {
		if !strings.Contains(jsonOut.String(), value) || !strings.Contains(humanOut.String(), value) {
			t.Fatalf("readiness differs for %s: %s %s", value, &jsonOut, &humanOut)
		}
	}
}
