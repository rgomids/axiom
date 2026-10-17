package githubissues

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/workflow"
	"github.com/rgomids/axiom/internal/workitem"
)

func TestDeliveryPRAdapterUsesExactPayloadAndBoundedObservation(t *testing.T) {
	dir := t.TempDir()
	binary := filepath.Join(dir, "gh")
	args, body := filepath.Join(dir, "args"), filepath.Join(dir, "body")
	t.Setenv("AXIOM_TEST_ARGS", args)
	t.Setenv("AXIOM_TEST_STDIN", body)
	script := `#!/bin/sh
printf '%s\n' "$@" > "$AXIOM_TEST_ARGS"
case "$*" in
 *POST*) cat > "$AXIOM_TEST_STDIN"; printf '{}';;
 *) printf '[]';;
esac
`
	if err := os.WriteFile(binary, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	a, err := New(binary)
	if err != nil {
		t.Fatal(err)
	}
	metadata, err := DeliveryMetadata("fixture/repository", strings.Repeat("a", 64), workitem.MetadataPolicy{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	p := workflow.DeliveryPreview{PacketDigest: strings.Repeat("b", 64), Target: workflow.DeliveryTarget{Repository: "fixture/repository", Title: "$(do-not-run)", Body: "`do-not-run`", BaseRef: "refs/heads/main", HeadRef: "refs/heads/delivery", Metadata: metadata}}
	if err = a.CreateDeliveryPR(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	var payload map[string]any
	wire, _ := os.ReadFile(body)
	if json.Unmarshal(wire, &payload) != nil || payload["body"] != p.PRBody() || payload["draft"] != true || payload["base"] != "main" || payload["head"] != "delivery" {
		t.Fatal(string(wire))
	}
	arguments, _ := os.ReadFile(args)
	if strings.Contains(string(arguments), "do-not-run") || !strings.Contains(string(arguments), "repos/fixture/repository/pulls") {
		t.Fatal(string(arguments))
	}
	if candidates, conclusive, err := a.ObserveDeliveryPR(context.Background(), p); err != nil || !conclusive || len(candidates) != 0 {
		t.Fatal(candidates, conclusive, err)
	}
	if _, err = DeliveryMetadata("fixture/repository", strings.Repeat("a", 64), workitem.MetadataPolicy{PullRequest: map[string]workitem.MetadataRule{"reviewers": {Requirement: workitem.MetadataRequired}}}, nil); err == nil {
		t.Fatal("unresolved required metadata accepted")
	}
	p.Target.Repository = "../outside"
	if err = a.CreateDeliveryPR(context.Background(), p); err == nil {
		t.Fatal("invalid resource accepted")
	}
}
func TestDeliveryPRDecodeRejectsForeignIdentity(t *testing.T) {
	raw := []byte(`{"number":7,"html_url":"https://github.com/fixture/repository/pull/7","title":"delivery","body":"result","head":{"ref":"delivery","sha":"abc","repo":{"full_name":"other/repository"}},"base":{"ref":"main","sha":"def","repo":{"full_name":"fixture/repository"}}}`)
	if _, err := decodeDeliveryPR("fixture/repository", raw, nil); err == nil {
		t.Fatal("foreign head accepted")
	}
}
