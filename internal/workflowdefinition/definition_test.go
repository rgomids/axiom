package workflowdefinition

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cyberphone/json-canonicalization/go/src/webpki.org/jsoncanonicalizer"
)

func TestApprovedFixturesAndFrozenDigests(t *testing.T) {
	root := "../../docs/specifications/007-configurable-workflows"
	approvedSchema, err := os.ReadFile(filepath.Join(root, "workflow.schema.json"))
	if err != nil || !bytes.Equal(approvedSchema, schemaWire) {
		t.Fatalf("embedded schema differs from accepted contract: %v", err)
	}
	b, e := os.ReadFile(filepath.Join(root, "example-digests.json"))
	if e != nil {
		t.Fatal(e)
	}
	var expected map[string]string
	if e = json.Unmarshal(b, &expected); e != nil {
		t.Fatal(e)
	}
	for name, digest := range expected {
		t.Run(name, func(t *testing.T) {
			b, e := os.ReadFile(filepath.Join(root, "examples", name))
			if e != nil {
				t.Fatal(e)
			}
			doc, issues := Decode(b)
			if len(issues) > 0 || doc.Digest != digest {
				t.Fatalf("decode: %+v digest=%s", issues, doc.Digest)
			}
			var pretty bytes.Buffer
			if e = json.Indent(&pretty, b, "", "    "); e != nil {
				t.Fatal(e)
			}
			again, issues := Decode(pretty.Bytes())
			if len(issues) > 0 || again.Digest != digest {
				t.Fatal("format changed identity")
			}
		})
	}
	builtin := Builtin()
	b, e = os.ReadFile(filepath.Join(root, "examples/default-sdd-r1.json"))
	if e != nil {
		t.Fatal(e)
	}
	doc, _ := Decode(b)
	if builtin.Digest != doc.Digest {
		t.Fatal("seed drift")
	}
	for _, s := range builtin.Definition.Stages {
		if len(s.Agents) != 1 {
			t.Fatal("default requires extra agents")
		}
	}
}

func TestRejectInvalidContracts(t *testing.T) {
	cases := map[string]func(*Definition){
		"cycle":               func(d *Definition) { d.Stages[0].Agents[0].DependsOn = []string{d.Stages[0].Agents[0].ID} },
		"dangling dependency": func(d *Definition) { d.Stages[0].Agents[0].DependsOn = []string{"missing"} },
		"forward input": func(d *Definition) {
			d.Stages[0].Inputs[0].Kind = "stage-output"
			d.Stages[0].Inputs[0].Source = "completion/result"
		},
		"unknown input":   func(d *Definition) { d.Stages[0].Agents[0].Inputs = []string{"missing"} },
		"unknown output":  func(d *Definition) { d.Stages[0].Agents[0].Outputs = []string{"missing"} },
		"criterion":       func(d *Definition) { d.Stages[0].CompletionCriteria[0].ValidatorRef = "missing" },
		"stage duplicate": func(d *Definition) { d.Stages[1].ID = d.Stages[0].ID },
		"phase":           func(d *Definition) { d.Stages[0].Phase = "review" },
		"gate omitted":    func(d *Definition) { d.Stages[3].HumanGates = []Gate{} },
		"gate timing":     func(d *Definition) { d.Stages[3].HumanGates[0].Timing = "after" },
		"checkpoint":      func(d *Definition) { d.Stages[0].Checkpoint = false },
		"concurrency":     func(d *Definition) { d.Stages[0].Concurrency = 2 },
		"timeout":         func(d *Definition) { d.Stages[0].Agents[0].TimeoutSeconds = 86401 },
		"attempts":        func(d *Definition) { d.Stages[0].Agents[0].MaximumAttempts = 11 },
		"effort":          func(d *Definition) { d.Stages[0].Agents[0].Effort.Value = "high" },
		"owner":           func(d *Definition) { d.Stages[0].Agents[0].IntegrationOwner = false },
		"text bytes":      func(d *Definition) { d.Stages[0].Instructions = strings.Repeat("é", 8193) },
		"executable":      func(d *Definition) { d.Stages[0].Inputs[0].Source = "/usr/bin/tool" },
		"credential":      func(d *Definition) { d.Stages[0].Instructions = "password=private-value" },
		"machine path":    func(d *Definition) { d.Stages[0].Instructions = "Use C:\\tools\\agent.exe" },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			d := Builtin().Definition
			mutate(&d)
			if _, issues := Encode(d); len(issues) == 0 {
				t.Fatal("accepted invalid contract")
			}
		})
	}
	for _, wire := range []string{`{"schemaVersion":1,"schemaVersion":1}`, `{"x":"\udead"}`, `{"x":null}`, `{"x":1.0}`, `[]`, string(append([]byte{0xff}, defaultWire...)), strings.Repeat("[", 35) + "0" + strings.Repeat("]", 35)} {
		if _, issues := Decode([]byte(wire)); len(issues) == 0 {
			t.Fatal("accepted invalid JSON")
		}
	}
	var raw map[string]any
	json.Unmarshal(defaultWire, &raw)
	raw["unexpected"] = true
	b, _ := json.Marshal(raw)
	if _, issues := Decode(b); len(issues) == 0 {
		t.Fatal("accepted unknown field")
	}
}

// Upstream RFC 8785 conformance corpus includes nested sorting, non-BMP UTF-16
// ordering, controls, Unicode preservation, literals and ECMAScript numbers.
func TestJCSConformanceCorpus(t *testing.T) {
	paths, e := filepath.Glob("testdata/jcs/input/*.json")
	if e != nil || len(paths) != 6 {
		t.Fatal("missing conformance corpus", e)
	}
	for _, path := range paths {
		t.Run(filepath.Base(path), func(t *testing.T) {
			input, e := os.ReadFile(path)
			if e != nil {
				t.Fatal(e)
			}
			expected, e := os.ReadFile(filepath.Join("testdata/jcs/output", filepath.Base(path)))
			if e != nil {
				t.Fatal(e)
			}
			actual, e := jsoncanonicalizer.Transform(input)
			if e != nil || !bytes.Equal(actual, expected) {
				t.Fatalf("JCS mismatch: %s %v", actual, e)
			}
		})
	}
	for _, bad := range []string{`"\udead"`, `"\ud800"`, `{"a":1,"a":2}`, `1e999`} {
		if _, e := jsoncanonicalizer.Transform([]byte(bad)); e == nil {
			t.Fatalf("JCS accepted %s", bad)
		}
	}
	d := Builtin().Definition
	d.Name = "ação 😀 <>&"
	doc, issues := Encode(d)
	if len(issues) > 0 || !bytes.Contains(doc.Canonical, []byte(d.Name)) {
		t.Fatalf("Unicode escaped/altered: %v", issues)
	}
}
