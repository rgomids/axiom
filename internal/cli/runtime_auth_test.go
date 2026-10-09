package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/completion"
	"github.com/rgomids/axiom/internal/provenance"
	"github.com/rgomids/axiom/internal/runtimeadapter"
)

type runtimeAuthRecordingService struct {
	recordingService
	runtimes []string
	source   provenance.Value
}

func (s *runtimeAuthRecordingService) RuntimeAuth(_ context.Context, runtimeID string) Result {
	s.runtimes = append(s.runtimes, runtimeID)
	statement, err := provenance.NewText("subscription login is unavailable", provenance.AxiomAuthored)
	if err != nil {
		return Result{Status: Failed}
	}
	value, err := completion.New(completion.Facts{ValidationFailed: true}, statement, nil, provenance.Text{}, "", s.source)
	if err != nil {
		return Result{Status: Failed}
	}
	report := runtimeadapter.AuthReport{RuntimeID: runtimeID, Status: runtimeadapter.AuthUnavailable, Reason: "not_logged_in", Method: "none", Version: "2.1.295", EvidenceKind: runtimeadapter.EvidenceFake, Usability: runtimeadapter.UsabilityUnproven, Revalidation: runtimeadapter.RevalidateBeforeDispatch}
	return Result{Completion: &value, RuntimeAuth: &report}
}

func TestRuntimeAuthDelegatesAndPresentsSanitizedReport(t *testing.T) {
	for _, runtimeID := range []string{"codex", "claude"} {
		service := &runtimeAuthRecordingService{source: completionProvenance(t)}
		var output bytes.Buffer
		code := Run(context.Background(), []string{"runtime", runtimeID, "auth"}, service, completionProvenance(t), &output)
		var event struct {
			Status      completion.Status         `json:"status"`
			RuntimeAuth runtimeadapter.AuthReport `json:"runtimeAuth"`
		}
		if err := json.Unmarshal(output.Bytes(), &event); err != nil || code != ExitFailure || event.Status != completion.ValidationFailure || event.RuntimeAuth.Reason != "not_logged_in" || strings.Join(service.runtimes, ",") != runtimeID {
			t.Fatalf("code=%d event=%+v err=%v output=%s", code, event, err, &output)
		}
		output.Reset()
		if RunInteractive(context.Background(), []string{"--human", "runtime", runtimeID, "auth"}, service, completionProvenance(t), strings.NewReader(""), &output, &bytes.Buffer{}); !strings.Contains(output.String(), "#### runtimeAuth\n\n- **runtimeId:** `"+runtimeID+"`") {
			t.Fatalf("human output=%s", &output)
		}
	}
	var output bytes.Buffer
	if code := Run(context.Background(), []string{"runtime", "codex", "auth"}, &recordingService{}, completionProvenance(t), &output); code != ExitFailure || !strings.Contains(output.String(), "application_unavailable") {
		t.Fatalf("unsupported service: code=%d output=%s", code, &output)
	}
	for _, args := range [][]string{{"runtime", "codex", "auth", "--login"}, {"runtime", "claude", "auth", "private-input"}} {
		service := &runtimeAuthRecordingService{}
		output.Reset()
		if code := Run(context.Background(), args, service, completionProvenance(t), &output); code != ExitFailure || len(service.runtimes) != 0 || strings.Contains(output.String(), "private-input") {
			t.Fatalf("args=%v code=%d output=%s", args, code, &output)
		}
	}
	output.Reset()
	if code := Run(context.Background(), []string{"runtime", "gemini", "auth"}, &runtimeAuthRecordingService{}, completionProvenance(t), &output); code != ExitFailure {
		t.Fatalf("unknown runtime accepted: %s", &output)
	}
}
