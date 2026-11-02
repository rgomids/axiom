package workflowdefinition

import (
	"bytes"
	"testing"
)

func TestSchema2InvocationRoundTripAndInvalidSelectors(t *testing.T) {
	for _, inv := range []AgentInvocation{{Runtime: "codex", Model: "vendor/model-1", Effort: "runtime-default"}, {Model: "chosen"}, {Effort: "high"}} {
		builtin, _ := BuiltinRevision(2)
		d := builtin.Definition
		d.Stages[0].Agents[0].ProfileRef = ""
		d.Stages[0].Agents[0].Invocation = &inv
		doc, issues := Encode(d)
		if len(issues) != 0 {
			t.Fatal(issues)
		}
		decoded, issues := Decode(doc.Canonical)
		if len(issues) != 0 || decoded.Definition.Stages[0].Agents[0].Invocation == nil || *decoded.Definition.Stages[0].Agents[0].Invocation != inv || !bytes.Equal(decoded.Canonical, doc.Canonical) {
			t.Fatalf("roundtrip: %v", issues)
		}
	}
	for _, inv := range []AgentInvocation{{}, {Runtime: "unconfigured"}, {Model: "bad model"}, {Effort: "high;bad"}, {Model: "token:synthetic"}} {
		builtin, _ := BuiltinRevision(2)
		d := builtin.Definition
		d.Stages[0].Agents[0].Invocation = &inv
		if _, issues := Encode(d); len(issues) == 0 {
			t.Fatalf("invalid invocation accepted: %+v", inv)
		}
	}
	// Version 1 retains its historical effort object and cannot gain schema-2 semantics.
	old := Builtin()
	again, issues := Encode(old.Definition)
	if len(issues) != 0 || !bytes.Equal(old.Canonical, again.Canonical) {
		t.Fatalf("historical digest changed: %v", issues)
	}
	old.Definition.Stages[0].Agents[0].Invocation = &AgentInvocation{Effort: "high"}
	if _, issues := Encode(old.Definition); len(issues) == 0 {
		t.Fatal("schema 1 reinterpreted invocation")
	}
	builtin, _ := BuiltinRevision(2)
	d := builtin.Definition
	d.Stages[0].Agents[0].Effort = Effort{Mode: "explicit", Value: "high"}
	if _, issues := Encode(d); len(issues) == 0 {
		t.Fatal("schema 2 accepted legacy effort")
	}
}
