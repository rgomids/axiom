package graphapplication

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/rgomids/axiom/internal/executiongraph"
	"github.com/rgomids/axiom/internal/runtimeadapter"
	"github.com/rgomids/axiom/internal/runtimeprofile"
	"github.com/rgomids/axiom/internal/workflowcompiler"
)

func TestCompiledPacketProjectsGraphChildAndSingleAgent(t *testing.T) {
	skipPOSIXStagePaths(t)
	for _, node := range []string{"child-1", ""} {
		input := workflowcompiler.ExecutionInput{AgentID: "worker", NodeKey: node, Role: "engineer", Responsibilities: "Implement approved plan", Instructions: "Read input refs", Controls: executiongraph.ExecutionControls{Timeout: time.Minute, ReasoningEffort: "private-effort"}, ResolvedProfile: &runtimeprofile.ResolvedProfile{Model: "private-model", Effort: runtimeprofile.Effort{Value: "private-effort"}}, Inputs: []workflowcompiler.InputBinding{{InputID: "plan", Reference: "artifact:plan", Digest: strings.Repeat("a", 64)}}}
		binding := runtimeadapter.PacketContext{AttemptID: "attempt-1", Inbox: "/state/inbox", AxiomPath: "/bin/axiom", AxiomDigest: strings.Repeat("b", 64)}
		invocation, receipt, err := AttachCompiledTaskPacket(executiongraph.Invocation{}, input, binding)
		if err != nil {
			t.Fatal(err)
		}
		if receipt.AttemptID != binding.AttemptID || len(receipt.PacketDigest) != 64 {
			t.Fatal("missing attempt receipt")
		}
		if bytes.Contains(invocation.Stdin, []byte("private-")) {
			t.Fatal("runtime selection leaked")
		}
		var packet runtimeadapter.TaskPacket
		if err := json.Unmarshal(invocation.Stdin, &packet); err != nil {
			t.Fatal(err)
		}
		if packet.AgentID != input.AgentID || packet.Inputs[0].Digest != input.Inputs[0].Digest {
			t.Fatal("compiled input not consumed")
		}
	}
}
