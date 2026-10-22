package codexruntime

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/projectapp"
	"github.com/rgomids/axiom/internal/workflowdefinition"
)

// Offline evaluation only. This is neither a product mapper nor a model/host
// response: tests cannot establish that an AI follows the installed guidance.
type presentationEnvelope struct {
	Status     string                     `json:"status"`
	Result     string                     `json:"result"`
	Category   string                     `json:"category"`
	References []string                   `json:"references"`
	Next       string                     `json:"next"`
	Provenance map[string]string          `json:"provenance"`
	Report     *projectapp.WorkflowReport `json:"workflowAuthoring"`
}
type presentationInventory struct {
	Reference workflowdefinition.Ref
	Selection string
	Stages    []string
	Nodes     []string
	Edges     []string
	Gates     []string
	// Full typed declarations retained, including I/O, criteria and ownership.
	Definition workflowdefinition.Definition
}

func presentationFixture(t *testing.T, name string) ([]byte, presentationEnvelope) {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", "rendered-workflow", name+".json"))
	if err != nil {
		t.Fatal(err)
	}
	var e presentationEnvelope
	if err = json.Unmarshal(b, &e); err != nil {
		t.Fatal(err)
	}
	return b, e
}
func inspectPresentation(e presentationEnvelope) (presentationInventory, error) {
	var v presentationInventory
	if e.Status != "success" || e.Category != "inspected" || e.Result == "" || len(e.Provenance) == 0 || e.Report == nil || e.Report.Definition == nil || e.Report.Reference == nil {
		return v, fmt.Errorf("incomplete/unsuccessful inspection")
	}
	r := e.Report
	wire, err := json.Marshal(r.Definition)
	if err != nil {
		return v, err
	}
	doc, issues := workflowdefinition.Decode(wire)
	if len(issues) > 0 {
		return v, fmt.Errorf("canonical diagnostics: %v", issues)
	}
	if !r.Reference.Valid() || doc.Ref(r.Reference.Source) != *r.Reference {
		return v, fmt.Errorf("identity mismatch")
	}
	v.Reference = *r.Reference
	v.Selection = "not reported"
	if r.Selection != nil {
		if !r.Selection.Valid() {
			return v, fmt.Errorf("invalid selection")
		}
		v.Selection = "divergent"
		if *r.Selection == v.Reference {
			v.Selection = "selected"
		}
	}
	v.Definition = doc.Definition
	for _, s := range doc.Definition.Stages {
		v.Stages = append(v.Stages, s.ID)
		for _, a := range s.Agents {
			v.Nodes = append(v.Nodes, s.ID+"/"+a.ID)
			for _, dep := range a.DependsOn {
				v.Edges = append(v.Edges, s.ID+"/"+dep+" -> "+s.ID+"/"+a.ID)
			}
		}
		for _, g := range s.HumanGates {
			v.Gates = append(v.Gates, s.ID+"/"+g.Kind+"/"+g.Timing)
		}
	}
	return v, nil
}
func samePresentation(a, b presentationInventory) bool { return reflect.DeepEqual(a, b) }

// Hand-checked oracle, separate from both the projection and specimens.
var presentationStageOrder = []string{"intake", "specification", "clarification", "plan", "tasks", "implementation", "review", "evidence", "reconciliation", "completion"}
var presentationGates = []string{"plan/planning_authority/before", "implementation/implementation_authority/before", "review/review_started/before", "completion/human_acceptance/after"}
var presentationJoin = []string{"implementation/implementer -> implementation/integrator", "implementation/reviewer -> implementation/integrator"}

func TestRenderedWorkflowCanonicalFixturesAndOracle(t *testing.T) {
	manifestWire, err := os.ReadFile("testdata/rendered-workflow/manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	var manifest map[string]struct {
		Classification string                 `json:"classification"`
		SHA256         string                 `json:"sha256"`
		Reference      workflowdefinition.Ref `json:"reference"`
	}
	if err = json.Unmarshal(manifestWire, &manifest); err != nil {
		t.Fatal(err)
	}
	if len(manifest) != 4 {
		t.Fatal("fixture inventory drift")
	}
	for _, name := range []string{"default-sdd-r1", "default-sdd-r2", "custom-r1", "custom-r2"} {
		t.Run(name, func(t *testing.T) {
			b, e := presentationFixture(t, name)
			before := append([]byte(nil), b...)
			m := manifest[name+".json"]
			if m.Classification != "SYNTHETIC" || m.SHA256 != digestOf(b) {
				t.Fatal("synthetic fixture hash drift")
			}
			v, err := inspectPresentation(e)
			if err != nil {
				t.Fatal(err)
			}
			if v.Reference != m.Reference || v.Selection != "selected" || !reflect.DeepEqual(v.Stages, presentationStageOrder) || !reflect.DeepEqual(v.Gates, presentationGates) {
				t.Fatal("identity/order/gate oracle mismatch")
			}
			source, err := os.ReadFile("../../docs/specifications/007-configurable-workflows/examples/" + name + ".json")
			if err != nil {
				t.Fatal(err)
			}
			doc, issues := workflowdefinition.Decode(source)
			if len(issues) > 0 || doc.Digest != m.Reference.Digest {
				t.Fatal("published definition digest changed")
			}
			wantNodes := []string{}
			for _, s := range presentationStageOrder {
				if strings.HasPrefix(name, "custom") && s == "implementation" {
					wantNodes = append(wantNodes, "implementation/implementer", "implementation/reviewer", "implementation/integrator")
				} else {
					wantNodes = append(wantNodes, s+"/owner")
				}
			}
			if !reflect.DeepEqual(v.Nodes, wantNodes) {
				t.Fatal("scoped node oracle mismatch")
			}
			if strings.HasPrefix(name, "custom") {
				if !reflect.DeepEqual(v.Edges, presentationJoin) {
					t.Fatal("independent roots/join changed")
				}
			} else if len(v.Edges) != 0 {
				t.Fatal("invented edge")
			}
			if e.Report.ProjectRevision != "synthetic-project-revision-17" {
				t.Fatal("project revision confused")
			}
			for _, s := range v.Definition.Stages {
				for _, a := range s.Agents {
					if (v.Definition.SchemaVersion == 1) != (a.Effort != (workflowdefinition.Effort{})) {
						t.Fatal("historical effort changed")
					}
					wantRuntime := []string{"codex", "claude"}
					if a.ID == "implementer" {
						wantRuntime = []string{"codex"}
					}
					if a.ID == "reviewer" {
						wantRuntime = []string{"claude"}
					}
					if a.ProfileRef != "policy-default" || !reflect.DeepEqual(a.RuntimeConstraints, wantRuntime) {
						t.Fatal("declared binding lost")
					}
				}
			}
			again, _ := inspectPresentation(e)
			if !samePresentation(v, again) || !bytes.Equal(b, before) {
				t.Fatal("not deterministic/read-only")
			}
			compact, _ := json.Marshal(e)
			var variant presentationEnvelope
			json.Unmarshal(compact, &variant)
			got, err := inspectPresentation(variant)
			if err != nil || !samePresentation(v, got) {
				t.Fatal("key/whitespace variant changed facts")
			}
			// Negative controls mutate candidate inventories, not the expected oracle.
			for _, kind := range []string{"missing-node", "extra-edge", "wrong-order", "wrong-identity", "wrong-gate", "wrong-constraint", "missing-edge", "reversed-edge"} {
				var changed presentationInventory
				copyWire, _ := json.Marshal(v)
				json.Unmarshal(copyWire, &changed)
				switch kind {
				case "missing-node":
					changed.Nodes = changed.Nodes[1:]
				case "extra-edge":
					changed.Edges = append(changed.Edges, "implementation/implementer -> implementation/reviewer")
				case "wrong-order":
					changed.Stages[0], changed.Stages[1] = changed.Stages[1], changed.Stages[0]
				case "wrong-identity":
					changed.Reference.Source = "other"
				case "wrong-gate":
					changed.Gates[0] = "plan/planning_authority/after"
				case "wrong-constraint":
					changed.Definition.Stages[0].Agents[0].RuntimeConstraints = []string{"resolved"}
				case "missing-edge":
					if len(changed.Edges) == 0 {
						return
					}
					changed.Edges = changed.Edges[1:]
				case "reversed-edge":
					if len(changed.Edges) == 0 {
						return
					}
					changed.Edges[0] = "implementation/integrator -> implementation/implementer"
				}
				if samePresentation(v, changed) {
					t.Fatalf("negative control undetected: %s", kind)
				}
			}
		})
	}
}

func TestRenderedWorkflowSelectionAndRefusals(t *testing.T) {
	for _, field := range []string{"absent", "source", "revision", "digest", "workflowId"} {
		t.Run(field, func(t *testing.T) {
			_, e := presentationFixture(t, "custom-r2")
			r := *e.Report.Reference
			r2 := r
			switch field {
			case "absent":
				e.Report.Selection = nil
			case "source":
				r2.Source = "builtin"
			case "revision":
				r2.Revision++
			case "digest":
				r2.Digest = strings.Repeat("a", 64)
			case "workflowId":
				r2.WorkflowID = "different"
			}
			if field != "absent" {
				e.Report.Selection = &r2
			}
			v, err := inspectPresentation(e)
			want := "divergent"
			if field == "absent" {
				want = "not reported"
			}
			if err != nil || v.Selection != want || v.Reference != r {
				t.Fatal("selection inferred/mutated", err)
			}
		})
	}
	for _, kind := range []string{"failure", "partial", "unknown", "missing-status", "missing-report", "missing-definition", "missing-reference", "stale-digest", "mismatched-revision", "unknown-schema", "cycle", "dangling", "self", "duplicate"} {
		t.Run(kind, func(t *testing.T) {
			_, e := presentationFixture(t, "custom-r2")
			switch kind {
			case "failure", "partial", "unknown":
				e.Status = kind
			case "missing-status":
				e.Status = ""
			case "missing-report":
				e.Report = nil
			case "missing-definition":
				e.Report.Definition = nil
			case "missing-reference":
				e.Report.Reference = nil
			case "stale-digest":
				e.Report.Reference.Digest = strings.Repeat("a", 64)
			case "mismatched-revision":
				e.Report.Definition.Revision++
			case "unknown-schema":
				e.Report.Definition.SchemaVersion = 99
			case "cycle":
				e.Report.Definition.Stages[5].Agents[0].DependsOn = []string{"integrator"}
			case "dangling":
				e.Report.Definition.Stages[5].Agents[2].DependsOn = []string{"absent"}
			case "self":
				e.Report.Definition.Stages[5].Agents[2].DependsOn = []string{"integrator"}
			case "duplicate":
				e.Report.Definition.Stages[5].Agents[2].DependsOn = []string{"implementer", "implementer", "reviewer"}
			}
			before, _ := json.Marshal(e)
			if _, err := inspectPresentation(e); err == nil {
				t.Fatal("refusal became successful graph")
			}
			after, _ := json.Marshal(e)
			if !bytes.Equal(before, after) {
				t.Fatal("refusal repaired input")
			}
		})
	}
	for _, wire := range []string{"", `{"status":"success"`, "null", `{"status":"unknown"}`} {
		var e presentationEnvelope
		err := json.Unmarshal([]byte(wire), &e)
		if err == nil {
			_, err = inspectPresentation(e)
		}
		if err == nil {
			t.Fatal("malformed/absent input accepted")
		}
	}
}

// Full fallback specimen uses labelled JSON-pointer rows solely as an offline
// comparison notation. Not a product schema, parser, wire format or renderer.
// Stored specimen produced independently of this Go traversal; all leaf values,
// ordered arrays and empty collections must survive, including unsafe labels.
func presentationLeaves(value any, path string, out map[string]string) {
	switch v := value.(type) {
	case map[string]any:
		if len(v) == 0 {
			out[path] = "{}"
		}
		for k, x := range v {
			presentationLeaves(x, path+"/"+k, out)
		}
	case []any:
		if len(v) == 0 {
			out[path] = "[]"
		}
		for i, x := range v {
			presentationLeaves(x, path+"/"+strconv.Itoa(i), out)
		}
	default:
		b, _ := json.Marshal(v)
		out[path] = string(b)
	}
}
func readPresentationSpecimen(wire []byte) (map[string]string, error) {
	out := map[string]string{}
	for _, line := range strings.Split(string(wire), "\n") {
		if !strings.HasPrefix(line, "- /") {
			continue
		}
		parts := strings.SplitN(strings.TrimPrefix(line, "- "), ": ", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("bad row")
		}
		if _, ok := out[parts[0]]; ok {
			return nil, fmt.Errorf("duplicate row")
		}
		var value any
		if err := json.Unmarshal([]byte(parts[1]), &value); err != nil {
			return nil, err
		}
		b, _ := json.Marshal(value)
		out[parts[0]] = string(b)
	}
	return out, nil
}
func TestRenderedWorkflowFallbackSpecimens(t *testing.T) {
	for _, name := range []string{"custom-r1", "custom-r2", "default-sdd-r1", "default-sdd-r2"} {
		t.Run(name, func(t *testing.T) {
			b, _ := presentationFixture(t, name)
			var raw any
			json.Unmarshal(b, &raw)
			want := map[string]string{}
			presentationLeaves(raw, "", want)
			wire, err := os.ReadFile("testdata/rendered-workflow/" + name + ".fallback.md")
			if err != nil {
				t.Fatal(err)
			}
			got, err := readPresentationSpecimen(wire)
			if err != nil || !reflect.DeepEqual(got, want) {
				t.Fatal("fallback lost canonical fields", err)
			}
			for _, condition := range []string{"unknown capability", "no renderer", "literal fence", "link only", "empty graphic", "clipped graphic", "renderer error"} {
				if !strings.Contains(string(wire), condition) {
					t.Fatal("missing fallback limitation", condition)
				}
			}
			keys := make([]string, 0, len(got))
			for key := range got {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			delete(got, keys[0])
			if reflect.DeepEqual(got, want) {
				t.Fatal("missing fallback fact undetected")
			}
		})
	}
}

func TestRenderedWorkflowGuidanceContract(t *testing.T) {
	b, err := fs.ReadFile(skillFiles, "skills/axiom-workflow/SKILL.md")
	if err != nil {
		t.Fatal(err)
	}
	// Guidance/packaging check only, never host fidelity evidence.
	for _, required := range []string{"actually rendered", "Workflow:", "Stage:", "Agent:", "not reported", "all four reference fields", "Never transitive-reduce", "concurrency 1", "runtimeConstraints", "historical", "496", "cross-segment", "Unicode controls/bidi", "remote resource", "manual witness", "literal fence", "same source", "satisfaction not", "source paths"} {
		if !strings.Contains(string(b), required) {
			t.Errorf("missing guidance: %s", required)
		}
	}
	if strings.Contains(string(b), "A diagram is optional only") {
		t.Fatal("old optional-only visual contract remains")
	}
}

func TestRenderedWorkflowCandidateDiagramAgainstManualOracle(t *testing.T) {
	// Parse only this deliberately restricted offline candidate grammar. No general
	// Mermaid interpreter/renderer is adopted. Real visible result remains T03.
	wire, err := os.ReadFile("testdata/rendered-workflow/custom-dag.candidate.mmd")
	if err != nil {
		t.Fatal(err)
	}
	node := regexp.MustCompile(`^    ([sa][0-9]+)\["([a-z-]+)"\]$`)
	edge := regexp.MustCompile(`^    ([sa][0-9]+) --> ([sa][0-9]+)$`)
	labels := map[string]string{}
	var ordered []string
	var edges []string
	for _, line := range strings.Split(string(wire), "\n") {
		if m := node.FindStringSubmatch(line); m != nil {
			if _, ok := labels[m[1]]; ok {
				t.Fatal("duplicate syntax ID")
			}
			labels[m[1]] = m[2]
			if m[1][0] == 's' {
				ordered = append(ordered, m[2])
			}
		}
		if m := edge.FindStringSubmatch(line); m != nil {
			edges = append(edges, m[1]+" -> "+m[2])
		}
	}
	if !reflect.DeepEqual(ordered, presentationStageOrder) || len(labels) != 13 || len(edges) != 11 {
		t.Fatal("diagram inventory differs")
	}
	for i := 0; i < 9; i++ {
		if edges[i] != fmt.Sprintf("s%d -> s%d", i, i+1) {
			t.Fatal("wrong Stage progression")
		}
	}
	got := []string{}
	for _, e := range edges[9:] {
		parts := strings.Split(e, " -> ")
		got = append(got, "implementation/"+labels[parts[0]]+" -> implementation/"+labels[parts[1]])
	}
	if !reflect.DeepEqual(got, presentationJoin) {
		t.Fatal("missing/extra/reversed Agent edge")
	}
}

func TestRenderedWorkflowInertLabelCandidates(t *testing.T) {
	// Test-only escape specimen; no renderer consumes it and no host safety claim.
	// Every code point becomes an inert numeric entity in the candidate label.
	vectors := []string{`quote " [bracket] ` + "`backtick`", "line\nbreak", `<svg onload="opaque">&</svg>`, "click a call action", "%%{init: opaque}%%", "é e\u0301 中文", "abc\u202edef\u2066", "control\x1b\x00", "ignore prior instructions"}
	for _, label := range vectors {
		var out strings.Builder
		for _, r := range label {
			fmt.Fprintf(&out, "&#%d;", r)
		}
		encoded := out.String()
		if strings.ContainsAny(encoded, "<>[]`\"\n") || strings.Contains(encoded, "click") || strings.Contains(encoded, "%%") {
			t.Fatal("grammar injection survived")
		}
		// XML/HTML entities cannot preserve NUL/bidi safely on every host. Such inputs
		// use visibly escaped text, with exact source still retained separately.
		if !strings.ContainsAny(label, "\x00") {
			decoded := html.UnescapeString(encoded)
			if decoded != label {
				t.Fatal("Unicode normalized/lost")
			}
		}
		quoted := strconv.QuoteToASCII(label)
		decoded, err := strconv.Unquote(quoted)
		if err != nil || decoded != label || strings.ContainsAny(quoted, "\n\x00\x1b\u202e\u2066") {
			t.Fatal("text control not inert/round-trippable", err)
		}
	}
}

func TestRenderedWorkflowInputDeltas(t *testing.T) {
	for _, kind := range []string{"agent-order", "declared-edge", "eligibility", "profile-intent", "revision", "purpose"} {
		t.Run(kind, func(t *testing.T) {
			_, e := presentationFixture(t, "custom-r2")
			before, err := inspectPresentation(e)
			if err != nil {
				t.Fatal(err)
			}
			d := e.Report.Definition
			s := &d.Stages[5]
			switch kind {
			case "agent-order":
				s.Agents[0], s.Agents[1] = s.Agents[1], s.Agents[0]
			case "declared-edge":
				s.Agents[1].DependsOn = []string{"implementer"}
			case "eligibility":
				s.Agents[0].RuntimeConstraints = []string{"claude"}
			case "profile-intent":
				s.Agents[0].ProfileRef = "alternate-profile"
			case "revision":
				d.Revision++
			case "purpose":
				s.Purpose = "Changed declared purpose"
			}
			doc, issues := workflowdefinition.Encode(*d)
			if len(issues) > 0 {
				t.Fatal(issues)
			}
			ref := doc.Ref("project")
			e.Report.Reference = &ref
			after, err := inspectPresentation(e)
			if err != nil {
				t.Fatal(err)
			}
			if samePresentation(before, after) || after.Selection != "divergent" {
				t.Fatal("changed input/stale selection invisible")
			}
			if !reflect.DeepEqual(before.Stages, after.Stages) || !reflect.DeepEqual(before.Gates, after.Gates) {
				t.Fatal("unrelated Stage order/gates changed")
			}
			if kind != "declared-edge" && !reflect.DeepEqual(before.Edges, after.Edges) {
				t.Fatal("unrelated edges changed")
			}
			if kind == "declared-edge" && len(after.Edges) != 3 {
				t.Fatal("new declared edge missing")
			}
			if kind == "agent-order" && reflect.DeepEqual(before.Nodes, after.Nodes) {
				t.Fatal("array ordering lost")
			}
		})
	}
}
