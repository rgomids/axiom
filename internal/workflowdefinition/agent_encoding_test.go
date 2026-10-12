package workflowdefinition

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestAgentMarshalSchema2OmitsEffortAndLegacyKeepsRequest(t *testing.T) {
	next, _ := BuiltinRevision(2)
	wire, err := json.Marshal(next.Definition)
	if err != nil || strings.Contains(string(wire), "\"effort\"") {
		t.Fatal("schema2 leaks Agent effort", err)
	}
	parsed, issues := Decode(wire)
	if len(issues) > 0 || parsed.Digest != next.Digest {
		t.Fatal("standard JSON schema2 roundtrip", issues)
	}
	legacy := Builtin()
	before := legacy.Digest
	wire, err = json.Marshal(legacy.Definition)
	if err != nil || !strings.Contains(string(wire), "\"effort\":{\"mode\":\"runtime-default\",\"value\":\"default\"}") {
		t.Fatal("historical request omitted", err)
	}
	parsed, issues = Decode(wire)
	if len(issues) > 0 || parsed.Digest != before {
		t.Fatal("legacy digest drift", issues)
	}
	// A direct standard encoder cannot make an authored override disappear.
	next.Definition.Stages[0].Agents[0].Effort = Effort{Mode: "explicit", Value: "high"}
	wire, err = json.Marshal(next.Definition)
	if err != nil {
		t.Fatal(err)
	}
	if _, issues = Decode(wire); len(issues) == 0 {
		t.Fatal("accepted supplied effort in schema2")
	}
}
