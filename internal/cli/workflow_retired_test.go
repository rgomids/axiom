package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
)

// The legacy lifecycle verbs and the delivery mutation verbs over retired
// ledgers were retired by Spec 015 S1.7. They are unknown commands: no flags
// are parsed, no service is called and no state is read.
func TestRetiredWorkflowVerbsRefuseBeforeDispatch(t *testing.T) {
	for _, verb := range []string{"start", "advance", "fact", "resume", "reconcile", "delivery decision", "delivery rework", "delivery prepare", "delivery preview", "delivery apply", "delivery inspect", "delivery reconcile"} {
		command := append([]string{"workflow"}, strings.Fields(verb)...)
		for _, extra := range [][]string{
			nil,
			{"--project", "sample", "--repository", "main", "--number", "7", "--expected-revision", "1"},
			{"--project", "sample", "--execution", "018f4a44-7c31-7dd4-9d00-111111111111", "--authorize-local"},
		} {
			args := append(append([]string{"--json"}, command...), extra...)
			var output bytes.Buffer
			service := &recordingService{}
			code := Run(context.Background(), args, service, completionProvenance(t), &output)
			var event struct {
				Category string `json:"category"`
			}
			if code == ExitSuccess || service.call != "" || json.Unmarshal(output.Bytes(), &event) != nil || event.Category != "invalid_command" {
				t.Fatalf("%v: code=%d call=%q output=%s", args, code, service.call, output.String())
			}
		}
	}
	// Canonical inspection cannot name or switch a Runtime.
	for _, command := range []string{"status", "evidence"} {
		assertStrictParserFailure(t, []string{"workflow", command, "--project", "sample", "--execution", "018f4a44-7c31-7dd4-9d00-111111111111", "--runtime", "claude"})
	}
}
