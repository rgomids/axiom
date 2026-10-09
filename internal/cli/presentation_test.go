package cli

import (
	"bytes"
	"encoding/json"
	"flag"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/completion"
	"github.com/rgomids/axiom/internal/manifest"
	"github.com/rgomids/axiom/internal/project"
	"github.com/rgomids/axiom/internal/projectapp"
	"github.com/rgomids/axiom/internal/provenance"
	"github.com/rgomids/axiom/internal/workitem"
)

var updatePresentation = flag.Bool("update-presentation", false, "rewrite canonical presentation golden files")

type presentationScenario struct {
	name      string
	operation action
	response  Result
	wantCode  int
}

func presentationResult(t *testing.T, facts completion.Facts, statement string, references []string, next, details string) *completion.Result {
	t.Helper()
	text, err := provenance.NewText(statement, provenance.AxiomAuthored)
	if err != nil {
		t.Fatal(err)
	}
	var nextText provenance.Text
	if next != "" {
		if nextText, err = provenance.NewText(next, provenance.AxiomAuthored); err != nil {
			t.Fatal(err)
		}
	}
	result, err := completion.New(facts, text, references, nextText, details, completionProvenance(t))
	if err != nil {
		t.Fatal(err)
	}
	return &result
}

// presentationScenarios covers every outcome class #232 requires, each through
// the same emitResponse routing the CLI and both Runtime skills use.
func presentationScenarios(t *testing.T) []presentationScenario {
	const projectID = "123e4567-e89b-42d3-a456-426614174000"
	workItem := &WorkItemView{ProjectID: projectID, RepositoryKey: "app", Provider: "github", Resource: "acme/app", ExternalID: "42", URL: "https://github.com/acme/app/issues/42", State: "closed"}
	workflowView := &WorkflowView{ExecutionID: "exec-7", WorkflowVersion: "1", Status: "interrupted", CurrentGate: "implementation", Revision: 4, RepositoryKey: "app", WorkItem: *workItem, RuntimeID: "codex", Transitions: []WorkflowStepView{{Revision: 4, From: "plan", To: "implementation", Outcome: "advanced", CommittedAt: "2026-10-09T12:00:00Z"}}}
	change := &workitem.ChangePreview{Operation: "close", Target: workitem.DraftTarget{ProjectID: projectID, RepositoryKey: "app", Provider: "github", Resource: "acme/app"}, Effects: []string{}, Digest: strings.Repeat("d", 64)}
	operational := &projectapp.OperationalPreview{ProjectID: projectID, Operation: "archive", Current: projectapp.OperationalView{ProjectStatus: "active", DisabledIntegrations: []string{}}, Result: projectapp.OperationalView{ProjectStatus: "archived", DisabledIntegrations: []string{}}, Revision: "r1", Effects: []projectapp.EditEffect{{Scope: "local", Code: "project_archived"}}, Boundary: projectapp.OperationalBoundary{Portable: "unchanged", Provider: "unchanged", Credentials: "unchanged"}, Digest: strings.Repeat("e", 64)}
	hostile := *workItem
	hostile.URL = "https://example.test/`x`\x1b[31m\u202Eevil"
	return []presentationScenario{
		{"success-project-show", showAction, Result{Completion: presentationResult(t, completion.Facts{Completed: true}, "Project inspected", []string{"project:" + projectID}, "", ""), Project: &ProjectView{ID: projectID, Slug: "acme", Source: "explicit", Repositories: []RepositoryView{{Key: "app", Path: "/work/app", Availability: "available"}}}}, ExitSuccess},
		{"no-op-work-item-close", workItemCloseAction, Result{Completion: presentationResult(t, completion.Facts{Completed: true}, "Work Item is already closed", []string{"github:acme/app#42"}, "", ""), Category: "work_item_already_closed", WorkItemChange: change, WorkItem: workItem}, ExitSuccess},
		{"partial-work-item-create", workItemCreateAction, Result{Completion: presentationResult(t, completion.Facts{RequestedEffectConfirmed: true, SecondaryFailure: true}, "Work Item created; local link not persisted", []string{"github:acme/app#42"}, "Run work-item select with the created Work Item to link it", "artifact:"+projectID), WorkItem: workItem}, ExitFailure},
		{"validation-failure", validateAction, Result{Completion: presentationResult(t, completion.Facts{ValidationFailed: true}, "Project slug is required", nil, "Provide a Project slug and retry validation", ""), Category: "missing_required_input"}, ExitFailure},
		{"denied-authority-project-archive", projectArchiveAction, Result{Completion: presentationResult(t, completion.Facts{AuthorityDenied: true}, "Preview digest does not match the current Project", nil, "Preview again and authorize the new digest", ""), Category: "authority_denied", Operational: operational}, ExitFailure},
		{"interrupted-execution", workflowStatusAction, Result{Completion: presentationResult(t, completion.Facts{WasInterrupted: true}, "Execution was interrupted", []string{"execution:exec-7"}, "Resume the Execution at revision 4", ""), Workflow: workflowView}, ExitCancelled},
		{"recovery-required-execution", workflowReconcileAction, Result{Completion: presentationResult(t, completion.Facts{RequestedEffectConfirmed: true, SecondaryFailure: true}, "Provider labels published; local projection not recorded", []string{"execution:exec-7", "github:acme/app#42"}, "Run workflow reconcile again with the exact Execution revision", "artifact:"+projectID), Workflow: workflowView}, ExitFailure},
		{"provider-failure-unconfirmed", workItemCommentAction, Result{Completion: presentationResult(t, completion.Facts{Failed: true}, "Provider outcome is unconfirmed", nil, "Inspect the Work Item before repeating the comment", ""), Category: "publication_uncertain", WorkItem: workItem}, ExitFailure},
		{"provider-failure-retryable", workItemUpdateAction, Result{Completion: presentationResult(t, completion.Facts{RetrySafeFailure: true}, "Provider is temporarily unavailable before any effect", nil, "Retry the same preview later", ""), Category: "temporarily_unavailable", WorkItem: workItem}, ExitFailure},
		{"hostile-provider-values", workItemShowAction, Result{Completion: presentationResult(t, completion.Facts{Completed: true}, "Work Item inspected", nil, "", ""), WorkItem: &hostile}, ExitSuccess},
	}
}

func renderScenario(t *testing.T, scenario presentationScenario, mode outputMode) string {
	t.Helper()
	var output bytes.Buffer
	if code := emitResponse(&output, mode, scenario.operation, scenario.response); code != scenario.wantCode {
		t.Fatalf("%s %s exit = %d, want %d\n%s", scenario.name, mode, code, scenario.wantCode, output.String())
	}
	return output.String()
}

func TestPresentationGoldenScenarios(t *testing.T) {
	for _, scenario := range presentationScenarios(t) {
		t.Run(scenario.name, func(t *testing.T) {
			for _, golden := range []struct {
				mode      outputMode
				extension string
			}{{jsonOutput, ".json"}, {humanOutput, ".md"}} {
				got := renderScenario(t, scenario, golden.mode)
				path := filepath.Join("testdata", "presentation", scenario.name+golden.extension)
				if *updatePresentation {
					if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
						t.Fatal(err)
					}
				}
				want, err := os.ReadFile(path)
				if err != nil {
					t.Fatalf("%v (run go test ./internal/cli -run TestPresentationGoldenScenarios -update-presentation)", err)
				}
				if got != string(want) {
					t.Fatalf("%s output drifted from %s:\n%s", golden.mode, path, got)
				}
			}
		})
	}
}

// Every canonical value and key of the JSON event, including confirmed
// references, details, provenance and operation payload, appears in the
// human view in its literal rendering.
func TestPresentationPreservesEveryCanonicalValue(t *testing.T) {
	for _, scenario := range presentationScenarios(t) {
		assertHumanPreservesEvent(t, scenario.name, renderScenario(t, scenario, jsonOutput), renderScenario(t, scenario, humanOutput))
	}
}

func assertHumanPreservesEvent(t *testing.T, name, wire, human string) {
	t.Helper()
	root, err := decodeOrdered([]byte(wire))
	if err != nil {
		t.Fatal(err)
	}
	status := root.field("status").text
	if !strings.Contains(human, "("+codeSpan(status)+")") {
		t.Errorf("%s: canonical status %q absent", name, status)
	}
	var walk func(path string, node *jsonNode)
	walk = func(path string, node *jsonNode) {
		for index, child := range node.children {
			childPath := path + "[" + strconv.Itoa(index) + "]"
			if node.kind == jsonObject {
				childPath = path + "." + node.keys[index]
				if path != "" && !strings.Contains(human, markdownKey(node.keys[index])) {
					t.Errorf("%s: key %s absent", name, childPath)
				}
			}
			walk(childPath, child)
		}
		if node.kind == jsonObject || node.kind == jsonArray {
			return
		}
		want := inlineValue(node)
		switch {
		case path == ".result" || path == ".next":
			want = plainText(node.text)
		case node.kind == jsonString && strings.ContainsAny(node.text, "\n\r"):
			want = escapeControls(strings.Split(node.text, "\n")[0], true)
		}
		if !strings.Contains(human, want) {
			t.Errorf("%s: value %s = %q absent from human view as %q", name, path, node.text, want)
		}
	}
	walk("", root)
}

func TestPresentationIsDeterministicAndModeIndependentInOutcome(t *testing.T) {
	for _, scenario := range presentationScenarios(t) {
		first := renderScenario(t, scenario, humanOutput)
		for range 20 {
			if renderScenario(t, scenario, humanOutput) != first {
				t.Fatalf("%s human view is not deterministic", scenario.name)
			}
		}
	}
}

func TestPresentationStatusLabelsAreExhaustiveAndDistinct(t *testing.T) {
	seen := map[string]completion.Status{}
	for _, status := range completion.Statuses() {
		label := statusLabels[status]
		if label == "" {
			t.Fatalf("status %s has no label", status)
		}
		if previous, duplicate := seen[label]; duplicate {
			t.Fatalf("statuses %s and %s share label %q", previous, status, label)
		}
		seen[label] = status
		if status != completion.Success && strings.Contains(strings.ToLower(label), "succeed") {
			t.Fatalf("non-success status %s is labelled as success: %q", status, label)
		}
	}
	if len(statusLabels) != len(completion.Statuses()) {
		t.Fatalf("labels %d for %d statuses", len(statusLabels), len(completion.Statuses()))
	}
	unknown := []byte(`{"status":"mystery","result":"x","provenance":{"product":"Axiom"}}`)
	rendered, err := renderMarkdown(unknown)
	if err != nil || !strings.HasPrefix(string(rendered), "### Unrecognized status (`mystery`)") {
		t.Fatalf("unknown status rendered as %q, %v", rendered, err)
	}
	if _, err := renderMarkdown([]byte(`{"result":"x","provenance":{}}`)); err == nil {
		t.Fatal("event without canonical status rendered")
	}
}

func TestPresentationNeutralizesProviderAuthoredMarkup(t *testing.T) {
	scenarios := presentationScenarios(t)
	human := renderScenario(t, scenarios[len(scenarios)-1], humanOutput)
	for _, raw := range []string{"\x1b", "\u202E"} {
		if strings.Contains(human, raw) {
			t.Fatalf("human view carries raw control %q", raw)
		}
	}
	if !strings.Contains(human, "``https://example.test/`x`\\u001B[31m\\u202Eevil``") {
		t.Fatalf("hostile value not rendered literally:\n%s", human)
	}
	block, err := renderMarkdown([]byte(`{"status":"success","result":"x","provenance":{},"body":"line\n# heading\n` + "```" + `"}`))
	if err != nil || !strings.Contains(string(block), "  ````\n  line\n  # heading\n  ```\n  ````\n") {
		t.Fatalf("multi-line value escaped its fence:\n%s", block)
	}
}

func TestPresentationIsBoundedWithoutTruncation(t *testing.T) {
	result := presentationResult(t, completion.Facts{Completed: true}, "Projects listed", nil, "", "")
	projects := make([]ProjectListView, 0)
	for index := 0; ; index++ {
		next := append(projects, ProjectListView{ID: "123e4567-e89b-42d3-a456-426614174000", Slug: "p" + strconv.Itoa(index), Name: strings.Repeat("n", 40), Status: "active"})
		wire, _ := json.Marshal(projectListCompletionEvent{Projects: next})
		if len(wire) > MaxCompletionOutputBytes {
			break
		}
		projects = next
	}
	var structured, human bytes.Buffer
	if code := emitProjectListCompletion(&structured, jsonOutput, *result, projects); code != ExitSuccess {
		t.Fatalf("JSON exit = %d", code)
	}
	if code := emitProjectListCompletion(&human, humanOutput, *result, projects); code != ExitSuccess {
		t.Fatalf("human exit = %d", code)
	}
	if human.Len() > humanOutputLimit(structured.Len()) {
		t.Fatalf("human %d bytes exceeds bound for JSON %d bytes", human.Len(), structured.Len())
	}
	if strings.Count(human.String(), " · slug `p") != len(projects) {
		t.Fatal("human view dropped list entries instead of failing")
	}
	var refused bytes.Buffer
	if code := presentEvent(&refused, humanOutput, completion.Success, structured.Bytes(), structured.Len()-1); code != ExitFailure || refused.Len() != 0 {
		t.Fatalf("over-limit event wrote %d bytes, exit %d", refused.Len(), code)
	}
}

// The renderer is a pure function of the canonical event: it imports no
// Runtime, network or process package and therefore cannot call a model.
func TestPresentationMakesNoModelOrRuntimeCall(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "presentation.go", nil, parser.ImportsOnly)
	if err != nil {
		t.Fatal(err)
	}
	allowed := map[string]bool{`"bytes"`: true, `"encoding/json"`: true, `"errors"`: true, `"fmt"`: true, `"io"`: true, `"regexp"`: true, `"strings"`: true, `"unicode"`: true, `"github.com/rgomids/axiom/internal/completion"`: true}
	for _, imported := range file.Imports {
		if !allowed[imported.Path.Value] {
			t.Fatalf("presentation imports %s", imported.Path.Value)
		}
	}
}

// Review of #291: a validated setup preview whose glossary definitions are
// runs of backticks must keep its outcome, effects and digest in both modes.
// The shortest absent fence keeps such values near their JSON size.
func TestPresentationBacktickHeavyPreviewKeepsEveryField(t *testing.T) {
	glossary := make([]project.GlossaryEntry, 0, 6)
	for index := range 6 {
		glossary = append(glossary, project.GlossaryEntry{Key: "term-" + strconv.Itoa(index), Term: "Term " + strconv.Itoa(index), Definition: "x" + strings.Repeat("`", 2046) + "x"})
	}
	proposal, issues := projectapp.PrepareSetup(manifest.Codec{}, projectapp.SetupInput{
		ProjectID: "123e4567-e89b-42d3-a456-426614174000", Slug: "sample", Name: "Sample",
		Repositories: []projectapp.SetupRepository{{Key: "web", Path: "/work/web", Revision: "web-revision"}}, WorkItemProvider: "github", Glossary: glossary,
	}, projectapp.SetupObservation{PortableDestination: "/portable/sample", LocalDestination: "/state/projects/id", PortableRevision: "absent", LocalRevision: "absent"})
	if len(issues) != 0 || !proposal.Valid() {
		t.Fatalf("setup preview invalid: %v", issues)
	}
	for _, statement := range []string{"Project setup preview ready", "Project setup published"} {
		result := presentationResult(t, completion.Facts{Completed: true}, statement, []string{"project:123e4567-e89b-42d3-a456-426614174000"}, "Review preview", "")
		var structured, human bytes.Buffer
		jsonCode := emitSetupCompletion(&structured, jsonOutput, *result, proposal.Preview())
		humanCode := emitSetupCompletion(&human, humanOutput, *result, proposal.Preview())
		if jsonCode != ExitSuccess || humanCode != jsonCode {
			t.Fatalf("exit JSON=%d human=%d", jsonCode, humanCode)
		}
		if human.Len() > structured.Len()+structured.Len()/4 {
			t.Fatalf("human %d bytes for JSON %d bytes", human.Len(), structured.Len())
		}
		for _, want := range []string{"`publish_portable_project`", "`publish_local_bindings`", "- **digest:** `" + proposal.Preview().Digest + "`"} {
			if !strings.Contains(human.String(), want) {
				t.Fatalf("human view lacks %q", want)
			}
		}
		assertHumanPreservesEvent(t, statement, structured.String(), human.String())
	}
}

// A readable view beyond its bound becomes the literal view: the same summary
// and exit code, with every payload field as its exact canonical JSON.
func TestPresentationLiteralViewKeepsEveryPayloadField(t *testing.T) {
	items := make([]string, 4000)
	for index := range items {
		items[index] = `"a"`
	}
	nested := "[[[[[[[[[" + strings.Join(items, ",") + "]]]]]]]]]"
	wire := []byte(`{"status":"partial","result":"Effect confirmed","references":["github:acme/app#42"],"next":"Reconcile","provenance":{"product":"Axiom","version":"development","revision":"abc","sourceState":"clean"},"category":"applied","deep":` + nested + `,"digest":"d1"}` + "\n")
	readable, err := renderMarkdown(wire)
	if err != nil || len(readable) <= humanOutputLimit(len(wire)) {
		t.Fatalf("fixture does not exceed the readable bound: %d bytes, %v", len(readable), err)
	}
	var human bytes.Buffer
	if code := presentEvent(&human, humanOutput, completion.Partial, wire, len(wire)); code != completionExitCode(completion.Partial) {
		t.Fatalf("literal view exit %d", code)
	}
	for _, want := range []string{"### Partially completed — recovery required (`partial`)", "- **Next:** Reconcile", "`github:acme/app#42`", "**Provenance:** product `Axiom`"} {
		if !strings.Contains(human.String(), want) {
			t.Fatalf("literal view lacks %q", want)
		}
	}
	for key, raw := range map[string]string{"category": `"applied"`, "deep": nested, "digest": `"d1"`} {
		if !strings.Contains(human.String(), "#### "+key+" (canonical JSON)\n\n```json\n"+raw+"\n```\n") {
			t.Fatalf("literal view lacks canonical %s", key)
		}
	}
}

// The literal view fits humanOutputLimit for the smallest JSON bound: the
// summary is bounded by internal/completion and the payload at most triples.
func TestPresentationWorstCaseLiteralViewFits(t *testing.T) {
	references := make([]string, 16)
	for index := range references {
		references[index] = "r" + strconv.Itoa(index) + strings.Repeat("`", 252) + "r"
	}
	text := strings.Repeat("`", 512)
	result := presentationResult(t, completion.Facts{RequestedEffectConfirmed: true, SecondaryFailure: true}, text, references, text, "d"+strings.Repeat("`", 254)+"d")
	summary, err := renderCompletionJSON(*result)
	if err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(strings.Repeat("\u0085", (MaxCompletionOutputBytes-len(summary)-16)/2))
	if err != nil {
		t.Fatal(err)
	}
	wire := append(append(bytes.TrimSuffix(summary, []byte("}\n")), `,"x":`...), payload...)
	wire = append(wire, "}\n"...)
	if len(wire) > MaxCompletionOutputBytes {
		t.Fatalf("fixture %d bytes exceeds the JSON bound", len(wire))
	}
	literal, err := renderMarkdownView(wire, false)
	if err != nil || len(literal) > humanOutputLimit(MaxCompletionOutputBytes) {
		t.Fatalf("worst-case literal view %d bytes exceeds %d: %v", len(literal), humanOutputLimit(MaxCompletionOutputBytes), err)
	}
	for _, reference := range references {
		if !strings.Contains(string(literal), codeSpan(reference)) {
			t.Fatal("worst-case literal view dropped a confirmed reference")
		}
	}
}
