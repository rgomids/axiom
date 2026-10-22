package workflowdefinition

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

// Synthetic acceptance inputs for Spec 011. Existing decoder remains the only
// definition validator/JCS implementation; no presentation production code here.
func presentationBoundedDefinition(t *testing.T) Definition {
	t.Helper()
	d := Builtin().Definition
	for i := range d.Stages {
		d.Stages[i].Agents[0].Effort = Effort{}
	}
	d.SchemaVersion = 2
	return d
}
func presentationEncode(t *testing.T, d Definition) Document {
	t.Helper()
	doc, issues := Encode(d)
	if len(issues) > 0 {
		t.Fatal(issues)
	}
	return doc
}
func presentationAgents(d *Definition, n int, dense bool) {
	s := &d.Stages[5]
	seed := s.Agents[0]
	s.Agents = nil
	s.Mode = "dag"
	s.Concurrency = n
	for i := 0; i < n; i++ {
		a := seed
		a.ID = fmt.Sprintf("agent-%02d", i)
		a.DependsOn = []string{}
		a.IntegrationOwner = i == n-1
		a.ValidationOwner = i == n-1
		if dense || i == n-1 {
			for j := 0; j < i; j++ {
				a.DependsOn = append(a.DependsOn, fmt.Sprintf("agent-%02d", j))
			}
		}
		s.Agents = append(s.Agents, a)
	}
}
func TestPresentationSequentialAndDAGCases(t *testing.T) {
	for _, kind := range []string{"sequential-extra", "independent-join", "dag-concurrency-one", "single-agent-dag", "dense-496"} {
		t.Run(kind, func(t *testing.T) {
			d := presentationBoundedDefinition(t)
			n := 3
			if kind == "single-agent-dag" {
				n = 1
			}
			if kind == "dense-496" {
				n = 32
			}
			presentationAgents(&d, n, kind == "dense-496")
			s := &d.Stages[5]
			if kind == "dag-concurrency-one" {
				s.Concurrency = 1
			}
			if kind == "sequential-extra" {
				s.Mode = "sequential"
				s.Concurrency = 1
				s.Agents[1].DependsOn = []string{"agent-00"}
			}
			doc := presentationEncode(t, d)
			edges := 0
			for _, a := range doc.Definition.Stages[5].Agents {
				edges += len(a.DependsOn)
			}
			want := 2
			if kind == "sequential-extra" {
				want = 3
			}
			if n == 1 {
				want = 0
			}
			if n == 32 {
				want = 496
			}
			if edges != want {
				t.Fatalf("declared edges=%d want=%d", edges, want)
			}
			if kind == "independent-join" || kind == "dag-concurrency-one" {
				if len(s.Agents[0].DependsOn) != 0 || len(s.Agents[1].DependsOn) != 0 {
					t.Fatal("invented chain")
				}
			}
			if n == 32 {
				s.Agents[31].DependsOn = s.Agents[31].DependsOn[1:]
				changed := presentationEncode(t, d)
				if doc.Digest == changed.Digest {
					t.Fatal("missing transitive edge not detected")
				}
			}
		})
	}
}
func TestPresentationDomainCeilings(t *testing.T) {
	d := presentationBoundedDefinition(t)
	seed := d.Stages[0]
	extra := make([]Stage, 22)
	for i := range extra {
		extra[i] = seed
		extra[i].ID = fmt.Sprintf("extra-%02d", i)
		extra[i].Checkpoint = false
	}
	d.Stages = append(extra, d.Stages...)
	for i := range d.Stages {
		d.Stages[i].Checkpoint = i == len(d.Stages)-1 || d.Stages[i].Phase != d.Stages[i+1].Phase
	}
	doc := presentationEncode(t, d)
	if len(doc.Definition.Stages) != 32 {
		t.Fatal("stage ceiling lost")
	}
	over := d
	over.Stages = append([]Stage{extra[0]}, d.Stages...)
	if _, issues := Encode(over); len(issues) == 0 {
		t.Fatal("33 stages accepted")
	}
	// Canonical snapshot exactly 1 MiB, with each prose field within byte ceiling.
	for i := range d.Stages {
		for _, field := range []*string{&d.Stages[i].Instructions, &d.Stages[i].Purpose} {
			wire, _ := json.Marshal(d)
			remaining := MaxBytes - len(wire)
			if remaining <= 0 {
				break
			}
			add := 16384 - len(*field)
			if add > remaining {
				add = remaining
			}
			*field += strings.Repeat("x", add)
		}
	}
	wire, _ := json.Marshal(d)
	if len(wire) != MaxBytes {
		t.Fatalf("boundary fixture size %d", len(wire))
	}
	doc, issues := Decode(wire)
	if len(issues) != 0 || len(doc.Canonical) != MaxBytes {
		t.Fatalf("1 MiB boundary: %d %v", len(doc.Canonical), issues)
	}
	if _, issues = Decode(append(wire, ' ')); len(issues) == 0 {
		t.Fatal("over-bound definition accepted")
	}
	// Whitespace affects input byte bounds but not JCS identity.
	small, _ := json.Marshal(presentationBoundedDefinition(t))
	padded := append(small, []byte(strings.Repeat(" ", MaxBytes-len(small)))...)
	a, issues := Decode(padded)
	if len(issues) > 0 {
		t.Fatal(issues)
	}
	b, _ := Decode(small)
	if a.Digest != b.Digest {
		t.Fatal("whitespace changed digest")
	}
}
func TestPresentationDeclaredBoundsAndUntrustedProse(t *testing.T) {
	for _, test := range []struct {
		name   string
		change func(*Definition)
		valid  bool
	}{
		{"unicode-16384-bytes", func(d *Definition) { d.Stages[0].Purpose = strings.Repeat("é", 8192) }, true},
		{"unicode-over-bytes", func(d *Definition) { d.Stages[0].Purpose = strings.Repeat("é", 8193) }, false},
		{"attempt-one", func(d *Definition) { d.Stages[0].Agents[0].MaximumAttempts = 1 }, true},
		{"attempt-ten", func(d *Definition) { d.Stages[0].Agents[0].MaximumAttempts = 10 }, true},
		{"attempt-zero", func(d *Definition) { d.Stages[0].Agents[0].MaximumAttempts = 0 }, false},
		{"attempt-eleven", func(d *Definition) { d.Stages[0].Agents[0].MaximumAttempts = 11 }, false},
		{"timeout-one", func(d *Definition) { d.Stages[0].Agents[0].TimeoutSeconds = 1 }, true},
		{"timeout-max", func(d *Definition) { d.Stages[0].Agents[0].TimeoutSeconds = 86400 }, true},
		{"timeout-zero", func(d *Definition) { d.Stages[0].Agents[0].TimeoutSeconds = 0 }, false},
		{"timeout-over", func(d *Definition) { d.Stages[0].Agents[0].TimeoutSeconds = 86401 }, false},
		{"32-agents", func(d *Definition) { presentationAgents(d, 32, true) }, true},
		{"33-agents", func(d *Definition) { presentationAgents(d, 33, true) }, false},
		{"unicode-control", func(d *Definition) { d.Name = "label\x1b[31m" }, false},
		{"schema2-effort", func(d *Definition) { d.Stages[0].Agents[0].Effort = Effort{"runtime-default", "default"} }, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			d := presentationBoundedDefinition(t)
			test.change(&d)
			_, issues := Encode(d)
			if (len(issues) == 0) != test.valid {
				t.Fatalf("valid=%v diagnostics=%v", test.valid, issues)
			}
		})
	}
	for _, n := range []int{64, 65} {
		t.Run(fmt.Sprintf("lists-%d", n), func(t *testing.T) {
			d := presentationBoundedDefinition(t)
			s := &d.Stages[0]
			for i := 1; i < n; i++ {
				id := fmt.Sprintf("item-%02d", i)
				s.Inputs = append(s.Inputs, Input{id, "work-item", "target", false})
				s.Outputs = append(s.Outputs, Output{id, "artifact", false})
				s.Validators = append(s.Validators, Validator{id, "artifact-schema", "builtin-sdd-v1"})
				s.CompletionCriteria = append(s.CompletionCriteria, Criterion{id, "Declared criterion", "result", "validate-result"})
			}
			_, issues := Encode(d)
			if (len(issues) == 0) != (n == 64) {
				t.Fatal(issues)
			}
		})
	}
}
