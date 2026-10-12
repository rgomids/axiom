package workflowdefinition

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"
)

func TestSchema2KeepsHistoricalBuiltinAndForbidsEffort(t *testing.T) {
	original, _ := os.ReadFile("default-sdd-r1.json")
	legacy, issues := Decode(original)
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	before, issues := Encode(legacy.Definition)
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	next, ok := BuiltinRevision(2)
	if !ok || next.Definition.SchemaVersion != 2 || next.Definition.Revision != 2 || !IsBuiltin(next.Ref("builtin")) {
		t.Fatal("missing v2 builtin")
	}
	after, issues := Encode(legacy.Definition)
	if len(issues) != 0 || legacy.Digest != after.Digest || !bytes.Equal(before.Canonical, after.Canonical) {
		t.Fatal("historical bytes changed")
	}
	encoded, issues := Encode(next.Definition)
	if len(issues) != 0 || encoded.Digest != next.Digest {
		t.Fatal("v2 encode drift", issues)
	}
	var raw map[string]any
	_ = json.Unmarshal(next.Canonical, &raw)
	agent := raw["stages"].([]any)[0].(map[string]any)["agents"].([]any)[0].(map[string]any)
	for _, effort := range []any{nil, map[string]any{}, map[string]any{"mode": "runtime-default", "value": "default"}} {
		agent["effort"] = effort
		wire, _ := json.Marshal(raw)
		if _, issues := Decode(wire); len(issues) == 0 {
			t.Fatalf("accepted forbidden effort %v", effort)
		}
	}
	delete(agent, "effort")
	agent["capabilities"] = []string{"reasoning-effort-high"}
	wire, _ := json.Marshal(raw)
	if _, issues := Decode(wire); len(issues) == 0 {
		t.Fatal("accepted second owner")
	}
	next.Definition.Stages[0].Agents[0].Effort = Effort{Mode: "explicit", Value: "high"}
	if _, issues := Encode(next.Definition); len(issues) == 0 {
		t.Fatal("encoder dropped supplied effort")
	}
}

func TestEncodeSchema2InvalidStructureReturnsDiagnostics(t *testing.T) {
	if _, issues := Encode(Definition{SchemaVersion: 2}); len(issues) == 0 {
		t.Fatal("invalid empty definition")
	}
}
