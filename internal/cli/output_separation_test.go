package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/codexruntime"
	"github.com/rgomids/axiom/internal/completion"
	"github.com/rgomids/axiom/internal/projectapp"
	"github.com/rgomids/axiom/internal/workflowdefinition"
	"github.com/rgomids/axiom/internal/workitem"
)

type readinessFixture struct {
	name, structure, effective, workItem, execution string
	blockers                                        []projectapp.Finding
	absent                                          bool
	candidate                                       string
}

func readinessFixtures() []readinessFixture {
	return []readinessFixture{
		{"policy", "valid", "partial", "ready", "blocked", []projectapp.Finding{{Code: "runtime_policy_unavailable", Subject: "execution"}}, false, "Validation completed. Structure valid; effective partial. Work Item ready; Execution blocked: runtime_policy_unavailable (execution). Inspect Runtime policy; no authority granted."},
		{"ready", "valid", "ready", "ready", "ready", nil, false, "Validation completed. Structure valid; effective ready. Work Item ready; Execution ready. No authority granted."},
		{"blocked", "valid", "blocked", "blocked", "blocked", []projectapp.Finding{{Code: "repository_unavailable", Subject: "main"}}, false, "Validation completed. Structure valid; effective blocked. Work Item blocked; Execution blocked: repository_unavailable (main). Inspect the repository; no authority granted."},
		{"invalid", "invalid", "blocked", "blocked", "blocked", []projectapp.Finding{{Code: "project_state_invalid", Subject: "project"}}, false, "Validation failed. Structure invalid; effective blocked. Work Item blocked; Execution blocked: project_state_invalid (project). Inspect Project state; no authority granted."},
		{"absent", "", "", "", "", nil, true, "Validation completed. Readiness unknown. Inspect canonical readiness before dependent operations; no authority granted."},
		{"resolution", "valid", "partial", "ready", "blocked", []projectapp.Finding{{Code: "runtime_resolution_blocked", Subject: "execution"}}, false, "Validation completed. Structure valid; effective partial. Work Item ready; Execution blocked: runtime_resolution_blocked (execution), cause unknown. Inspect resolution; no authority granted."},
		{"multiple", "valid", "partial", "ready", "blocked", []projectapp.Finding{{Code: "runtime_policy_unavailable", Subject: "execution"}, {Code: "repository_unavailable", Subject: "secondary"}}, false, "Validation completed. Structure valid; effective partial. Work Item ready; Execution blocked: runtime_policy_unavailable (execution), repository_unavailable (secondary). Inspect both blockers; no authority granted."},
	}
}

type validationProbe struct {
	lifecycleRecorder
	response Result
	calls    int
}

func (s *validationProbe) Validate(context.Context, ProjectInput) Result {
	s.calls++
	return s.response
}

func (s *validationProbe) ProjectValidateInstalled(context.Context, ProjectLifecycleInput) Result {
	s.calls++
	return s.response
}

func readinessObligations(event readinessCompletionEvent, candidate string) error {
	wants := []string{"no authority granted"}
	expected := completion.Success
	if event.Readiness != nil && event.Readiness.Structure == "invalid" {
		expected = completion.ValidationFailure
		wants = append(wants, "Validation failed")
	} else {
		wants = append(wants, "Validation completed")
	}
	if event.Status != expected {
		return fmt.Errorf("completion differs from the typed fixture")
	}
	if event.Readiness == nil {
		wants = append(wants, "Readiness unknown")
	} else {
		r := event.Readiness
		wants = append(wants, "Structure "+r.Structure, "effective "+r.Effective)
		for _, op := range r.Operations {
			name := map[projectapp.ReadinessOperation]string{projectapp.OperationWorkItem: "Work Item", projectapp.OperationExecution: "Execution"}[op.Operation]
			wants = append(wants, name+" "+op.Status)
			for _, finding := range op.Blockers {
				wants = append(wants, finding.Code+" ("+finding.Subject+")")
			}
		}
		for _, finding := range append(append([]projectapp.Finding{}, r.Blockers...), r.Warnings...) {
			wants = append(wants, finding.Code+" ("+finding.Subject+")")
		}
	}
	for _, want := range wants {
		if !strings.Contains(strings.ToLower(candidate), strings.ToLower(want)) {
			return fmt.Errorf("missing typed claim %q", want)
		}
	}
	for _, forbidden := range []string{"all operations ready", "Project invalid", "authority granted automatically", "Execution ready"} {
		if strings.Contains(candidate, forbidden) && (forbidden != "Execution ready" || event.Readiness == nil || event.Readiness.Operation(projectapp.OperationExecution).Status != "ready") {
			return fmt.Errorf("unsupported claim %q", forbidden)
		}
	}
	if strings.Contains(candidate, "Runtime policy") {
		policy := false
		if event.Readiness != nil {
			for _, f := range event.Readiness.Operation(projectapp.OperationExecution).Blockers {
				policy = policy || f.Code == "runtime_policy_unavailable"
			}
		}
		if !policy {
			return fmt.Errorf("policy cause is unsupported")
		}
	}
	return nil
}

func TestOutputSeparationReadinessChannelsAndTypedClaims(t *testing.T) {
	for _, fixture := range readinessFixtures() {
		t.Run(fixture.name, func(t *testing.T) {
			status := completion.Success
			if fixture.structure == "invalid" {
				status = completion.ValidationFailure
			}
			result := canonicalResult(t, status, nil, "", completionProvenance(t))
			response := Result{Completion: &result}
			if !fixture.absent {
				response.Readiness = &projectapp.ReadinessReport{Structure: fixture.structure, Effective: fixture.effective,
					Operations: []projectapp.OperationReadiness{{Operation: projectapp.OperationWorkItem, Status: fixture.workItem, Blockers: []projectapp.Finding{}}, {Operation: projectapp.OperationExecution, Status: fixture.execution, Blockers: fixture.blockers}},
					Warnings:   []projectapp.Finding{{Code: "documentation_stale", Subject: "guide"}}}
			}
			candidate := fixture.candidate
			if !fixture.absent {
				candidate += " Warning: documentation_stale (guide); refresh documentation before relying on it."
			}
			var human string
			for _, mode := range []string{"--json", "", "--human"} {
				probe := &validationProbe{response: response}
				args := []string{"project", "validate", "--project", "sample"}
				if mode != "" {
					args = append([]string{mode}, args...)
				}
				var output bytes.Buffer
				if code := RunInteractive(context.Background(), args, probe, completionProvenance(t), nil, &output, io.Discard); code != completionExitCode(status) || probe.calls != 1 {
					t.Fatalf("code=%d calls=%d: %s", code, probe.calls, &output)
				}
				if mode == "--json" {
					var event readinessCompletionEvent
					if err := json.Unmarshal(output.Bytes(), &event); err != nil {
						t.Fatal(err)
					}
					if err := readinessObligations(event, candidate); err != nil {
						t.Fatal(err)
					}
					if err := readinessObligations(event, candidate+" All operations ready; Project invalid; authority granted automatically."); err == nil {
						t.Fatal("accepted contradictory prose despite correct source facts")
					}
					if fixture.name == "resolution" && readinessObligations(event, candidate+" Runtime policy is missing.") == nil {
						t.Fatal("invented policy cause accepted")
					}
					if fixture.name == "multiple" && readinessObligations(event, strings.ReplaceAll(candidate, "repository_unavailable (secondary)", "")) == nil {
						t.Fatal("omitted secondary blocker accepted")
					}
					rendered, err := renderTerminal(output.Bytes())
					if err != nil {
						t.Fatal(err)
					}
					human = string(rendered)
				} else if output.String() != human {
					t.Fatal("default/human diverge from canonical observation")
				}
			}
			if strings.Contains(human, "###") || strings.Contains(human, "**") || strings.Contains(human, "```") {
				t.Fatal("renderer generated Markdown")
			}
		})
	}
}

func TestOutputSeparationInstalledSyntheticHosts(t *testing.T) {
	for _, host := range []string{"codex", "claude"} {
		t.Run(host, func(t *testing.T) {
			root := filepath.Join(t.TempDir(), host, "skills")
			constructor := codexruntime.New
			if host == "claude" {
				constructor = codexruntime.NewClaude
			}
			service, err := constructor(root)
			if err != nil {
				t.Fatal(err)
			}
			if result := service.Install(context.Background()); result.Status != codexruntime.Applied {
				t.Fatal(result)
			}
			for _, skill := range canonicalDomainSkills {
				wire, err := os.ReadFile(filepath.Join(root, skill, "SKILL.md"))
				if err != nil {
					t.Fatal(err)
				}
				for _, required := range []string{"Use leading `axiom --json` internally", "stop dependent actions", "No formatter agent", "copy them exactly", "confirmed and uncertain effects"} {
					if !bytes.Contains(wire, []byte(required)) {
						t.Fatalf("%s missing %s", skill, required)
					}
				}
				for _, scenario := range presentationScenarios(t) {
					structured := renderScenario(t, scenario, jsonOutput)
					human := renderScenario(t, scenario, humanOutput)
					assertHumanPreservesEvent(t, host+"/"+skill+"/"+scenario.name, structured, human)
					var event semanticObservation
					if err := json.Unmarshal([]byte(structured), &event); err != nil {
						t.Fatal(err)
					}
					candidate, err := os.ReadFile(filepath.Join("testdata", "presentation", scenario.name+".conversation.txt"))
					if err != nil {
						t.Fatal(err)
					}
					if err := checkCandidate(event, string(candidate)); err != nil {
						t.Fatal(err)
					}
					t.Logf("synthetic host=%s skill=%s fixture=%s status=%s extra formatting/runtime calls=0", host, skill, scenario.name, event.Status)
				}
			}
		})
	}
}

func TestOutputSeparationWriterFailureNeverReplays(t *testing.T) {
	result := canonicalResult(t, completion.Partial, []string{"github:acme/app#42"}, "Inspect confirmed effect", completionProvenance(t))
	for _, writer := range []io.Writer{shortWriter{}, failingWriter{}} {
		for _, mode := range []string{"--json", "--human"} {
			probe := &validationProbe{response: Result{Completion: &result}}
			if code := RunInteractive(context.Background(), []string{mode, "project", "validate", "--project", "sample"}, probe, completionProvenance(t), nil, writer, io.Discard); code != ExitFailure || probe.calls != 1 || result.Status() != completion.Partial {
				t.Fatalf("code=%d calls=%d status=%s", code, probe.calls, result.Status())
			}
		}
	}
}

type semanticObservation struct {
	completionEvent
	Category    string                         `json:"category"`
	WorkItem    *WorkItemView                  `json:"workItem"`
	Change      *workitem.ChangePreview        `json:"change"`
	Operational *projectapp.OperationalPreview `json:"operational"`
	Workflow    *WorkflowView                  `json:"workflow"`
	Diagnostic  *Diagnostic                    `json:"diagnostic"`
}

type sourceClaim struct{ source, value string }

func semanticClaims(event semanticObservation) []sourceClaim {
	claims := []sourceClaim{{"status", string(event.Status)}}
	for _, ref := range event.References {
		claims = append(claims, sourceClaim{"references", ref})
	}
	if event.Details != "" {
		claims = append(claims, sourceClaim{"details", event.Details})
	}
	if event.Category != "" {
		claims = append(claims, sourceClaim{"category", event.Category})
	}
	if event.Provenance.SourceState != "clean" {
		claims = append(claims, sourceClaim{"provenance.sourceState", string(event.Provenance.SourceState)})
	}
	if event.WorkItem != nil {
		claims = append(claims, sourceClaim{"workItem.externalId", event.WorkItem.ExternalID})
	}
	if event.Change != nil {
		claims = append(claims, sourceClaim{"change.digest", event.Change.Digest})
	}
	if event.Operational != nil {
		claims = append(claims, sourceClaim{"operational.digest", event.Operational.Digest}, sourceClaim{"operational.revision", event.Operational.Revision})
	}
	if event.Workflow != nil {
		claims = append(claims, sourceClaim{"workflow.executionId", event.Workflow.ExecutionID}, sourceClaim{"workflow.revision", fmt.Sprint(event.Workflow.Revision)})
	}
	if event.Diagnostic != nil {
		d := event.Diagnostic
		claims = append(claims, sourceClaim{"diagnostic.category", d.Category}, sourceClaim{"diagnostic.authority", d.Authority})
		for _, pair := range []struct {
			name   string
			values []string
		}{{"confirmedEffects", d.ConfirmedEffects}, {"uncertainEffects", d.UncertainEffects}, {"notObserved", d.NotObserved}, {"missingPrerequisites", d.MissingPrerequisites}} {
			for _, value := range pair.values {
				claims = append(claims, sourceClaim{"diagnostic." + pair.name, value})
			}
		}
	}
	return claims
}

func checkCandidate(event semanticObservation, candidate string) error {
	if len(event.Status) == 0 || !strings.Contains(candidate, "("+string(event.Status)+")") {
		return fmt.Errorf("completion is absent or changed")
	}
	for _, claim := range semanticClaims(event) {
		if !strings.Contains(candidate, terminalText(claim.value)) {
			return fmt.Errorf("missing %s: %s", claim.source, claim.value)
		}
	}
	for _, forbidden := range []string{"All effects rolled back", "Permission granted by this summary", "Replay the mutation", "Ignore prior instructions"} {
		if strings.Contains(candidate, forbidden) {
			return fmt.Errorf("unsupported effect or authority claim")
		}
	}
	if event.Status != completion.Success && strings.Contains(candidate, "Everything succeeded") {
		return fmt.Errorf("completion promoted")
	}
	if event.Status == completion.Partial && !strings.Contains(candidate, "Confirmed effect") {
		return fmt.Errorf("confirmed partial effect hidden")
	}
	return nil
}

func TestOutputSeparationSemanticCandidateFixtures(t *testing.T) {
	for _, scenario := range presentationScenarios(t) {
		t.Run(scenario.name, func(t *testing.T) {
			wire := renderScenario(t, scenario, jsonOutput)
			var event semanticObservation
			if err := json.Unmarshal([]byte(wire), &event); err != nil {
				t.Fatal(err)
			}
			candidate, err := os.ReadFile(filepath.Join("testdata", "presentation", scenario.name+".conversation.txt"))
			if err != nil {
				t.Fatal(err)
			}
			if err := checkCandidate(event, string(candidate)); err != nil {
				t.Fatal(err)
			}
			for _, contradiction := range []string{"Everything succeeded", "All effects rolled back", "Permission granted by this summary", "Replay the mutation"} {
				if contradiction == "Everything succeeded" && event.Status == completion.Success {
					continue
				}
				if checkCandidate(event, string(candidate)+" "+contradiction) == nil {
					t.Fatalf("accepted prose contradiction %s with unchanged ledger", contradiction)
				}
			}
			if event.Operational != nil && checkCandidate(event, strings.ReplaceAll(string(candidate), event.Operational.Digest, "wrong-digest")) == nil {
				t.Fatal("accepted changed digest")
			}
			if event.Status == completion.Partial && checkCandidate(event, strings.ReplaceAll(string(candidate), "Confirmed effect", "No effect")) == nil {
				t.Fatal("accepted hidden partial effect")
			}
		})
	}
}

func checkSyntheticToolObservation(wire []byte, projectID string) error {
	var event semanticObservation
	if err := json.Unmarshal(wire, &event); err != nil {
		return err
	}
	if !slices.Contains(completion.Statuses(), event.Status) || event.Provenance.Product == "" {
		return fmt.Errorf("unsupported or incomplete completion")
	}
	if event.WorkItem == nil || event.WorkItem.ProjectID != projectID {
		return fmt.Errorf("target is unresolved or different")
	}
	if event.Status == completion.Success && event.Diagnostic != nil && len(event.Diagnostic.UncertainEffects) != 0 {
		return fmt.Errorf("contradictory observation requires inspection")
	}
	return nil
}

func TestOutputSeparationSyntheticHostStopsOnUnusableObservation(t *testing.T) {
	scenario := presentationScenarios(t)[len(presentationScenarios(t))-1]
	valid := []byte(renderScenario(t, scenario, jsonOutput))
	projectID := scenario.response.WorkItem.ProjectID
	if err := checkSyntheticToolObservation(valid, projectID); err != nil {
		t.Fatal(err)
	}
	cases := map[string][]byte{
		"absent":        nil,
		"truncated":     valid[:len(valid)/2],
		"malformed":     []byte("not JSON"),
		"trailing":      append(append([]byte{}, valid...), valid...),
		"status absent": bytes.Replace(valid, []byte(`"status":"success",`), nil, 1),
		"unsupported":   bytes.Replace(valid, []byte(`"success"`), []byte(`"new_status"`), 1),
		"wrong target":  bytes.ReplaceAll(valid, []byte(projectID), []byte("another-project")),
		"contradictory": bytes.Replace(valid, []byte(`"workItem":`), []byte(`"diagnostic":{"uncertainEffects":["external acknowledgement unknown"]},"workItem":`), 1),
	}
	for _, host := range []string{"codex", "claude"} {
		for _, skill := range canonicalDomainSkills {
			for name, wire := range cases {
				t.Run(host+"/"+skill+"/"+name, func(t *testing.T) {
					if checkSyntheticToolObservation(wire, projectID) == nil {
						t.Fatal("dependent action would receive unusable observation")
					}
				})
			}
		}
	}
}

func TestOutputSeparationWorkflowSummaryKeepsObservedOrderAndGates(t *testing.T) {
	doc := workflowdefinition.Builtin()
	rows := []struct{ id, source, gate string }{
		{"intake", "target", ""},
		{"specification", "intake/result", ""},
		{"clarification", "specification/result", ""},
		{"plan", "clarification/result", "planning_authority:before"},
		{"tasks", "plan/result", ""},
		{"implementation", "tasks/result", "implementation_authority:before"},
		{"review", "implementation/result", "review_started:before"},
		{"evidence", "review/result", ""},
		{"reconciliation", "evidence/result", ""},
		{"completion", "reconciliation/result", "human_acceptance:after"},
	}
	if len(rows) != len(doc.Definition.Stages) {
		t.Fatal("summary omitted stages")
	}
	for index, row := range rows {
		stage := doc.Definition.Stages[index]
		if stage.ID != row.id || len(stage.Inputs) != 1 || stage.Inputs[0].Source != row.source {
			t.Fatalf("summary contradicts observed stage %d", index)
		}
		gate := ""
		if len(stage.HumanGates) != 0 {
			gate = stage.HumanGates[0].Kind + ":" + stage.HumanGates[0].Timing
		}
		if gate != row.gate {
			t.Fatalf("summary changed gate %s", stage.ID)
		}
	}
}

// AXM-39: offline candidate oracle. Context flags below describe the requested
// view, not new wire fields. Specimens are authored separately from extraction.
// A passing specimen proves this bounded oracle, never an AI/installed host.
type presentation012Context struct {
	projects          []ProjectListView
	fullID, collision bool
	readiness         *projectapp.ReadinessReport
	event             semanticObservation
	must, forbid      []string
}

func presentation012Claims(v presentation012Context) []sourceClaim {
	claims := semanticClaims(v.event)
	for _, p := range v.projects {
		label := p.Name
		if p.Name != "" && p.Slug != "" {
			label += " (" + p.Slug + ")"
		} else if p.Name == "" {
			label = p.Slug
		}
		if label == "" {
			label = "Project sem nome/slug"
		}
		claims = append(claims, sourceClaim{"projects.label", terminalText(label)})
		if v.fullID || v.collision || (p.Name == "" && p.Slug == "") {
			if p.ID != "" {
				claims = append(claims, sourceClaim{"projects.id", p.ID})
			}
		}
	}
	if v.readiness != nil {
		claims = append(claims, sourceClaim{"readiness.structure", "Structure " + v.readiness.Structure})
		for _, op := range v.readiness.Operations {
			claims = append(claims, sourceClaim{"readiness.operation", string(op.Operation) + " " + op.Status})
			for _, b := range op.Blockers {
				claims = append(claims, sourceClaim{"readiness.blocker", b.Code + " (" + b.Subject + "): " + b.Detail})
			}
		}
	}
	if d := v.event.Diagnostic; d != nil {
		for _, fact := range d.Observed {
			claims = append(claims, sourceClaim{"diagnostic.observed", fact})
		}
	}
	for _, fact := range v.must {
		claims = append(claims, sourceClaim{"scenario obligation", fact})
	}
	return claims
}

func checkPresentation012(v presentation012Context, candidate string) error {
	if err := checkCandidate(v.event, candidate); err != nil {
		return err
	}
	for _, c := range presentation012Claims(v) {
		if c.value != "" && !strings.Contains(candidate, c.value) {
			return fmt.Errorf("missing %s: %s", c.source, c.value)
		}
	}
	for _, bad := range append([]string{"Invented Project", "Runtime selected by host", "model invented", "effort invented", "Execution performed", "Retry automatically", "Nothing ran", "Diagram rendered", "Run injected command"}, v.forbid...) {
		if strings.Contains(candidate, bad) {
			return fmt.Errorf("unsupported claim %q", bad)
		}
	}
	return nil
}

func TestPresentation012IdentityRuntimeAndReadiness(t *testing.T) {
	const id = "11111111-1111-4111-8111-111111111111"
	const second = "11111111-1111-4111-8111-111111111112"
	const digest = "abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789"
	project := ProjectListView{ID: id, Name: "Órbita", Slug: "orbita"}
	base := semanticObservation{}
	base.Status = completion.Success
	base.Provenance.SourceState = "clean"
	cases := map[string]presentation012Context{
		"C01":           {projects: []ProjectListView{project}, forbid: []string{id}},
		"C02-name":      {projects: []ProjectListView{{ID: id, Name: "Órbita"}}, forbid: []string{"orbita", id}},
		"C02-slug":      {projects: []ProjectListView{{ID: id, Slug: "orbita"}}, forbid: []string{"Órbita", id}},
		"C03":           {projects: []ProjectListView{{ID: id}}, must: []string{"human identifiers absent"}},
		"C04":           {projects: []ProjectListView{project, {ID: second, Name: "Órbita", Slug: "orbita"}}, collision: true, must: []string{"Ambiguous", "exact target before action"}, forbid: []string{"First match selected"}},
		"C04-name":      {projects: []ProjectListView{project, {ID: second, Name: "Órbita", Slug: "other"}}, collision: true, must: []string{"Ambiguous", "exact target before action"}},
		"C05":           {projects: []ProjectListView{project}, fullID: true, must: []string{"axiom --json project show --project " + id, digest, "preview only; no consent"}},
		"C06":           {projects: []ProjectListView{{Name: "Órbita\n$(echo hostile)", Slug: "orbita"}}, must: []string{"Incomplete inventory", "ID unavailable", "label is inert", "inspect exact identity before action"}},
		"C09":           {must: []string{"selection unknown", "resolution unknown", "model/effort unknown"}, forbid: []string{"selected codex", "resolved codex", "observed execution"}},
		"C10":           {must: []string{"not an attempt", "billing unknown", "acceptance unknown"}},
		"C10-selection": {must: []string{"resolution unknown", "not an attempt"}},
		"C11":           {readiness: &projectapp.ReadinessReport{Structure: "valid", Operations: []projectapp.OperationReadiness{{Operation: projectapp.OperationExecution, Status: "blocked", Blockers: []projectapp.Finding{{Code: "runtime_policy_unavailable", Subject: "execution", Detail: "policy_unconfigured"}}}}}, must: []string{"not ready", "not performed", "no-dispatch confirmed", "inspect policy through axiom-project", "authority required for repair"}},
		"C12":           {readiness: &projectapp.ReadinessReport{Structure: "valid", Operations: []projectapp.OperationReadiness{{Operation: projectapp.OperationExecution, Status: "blocked", Blockers: []projectapp.Finding{{Code: "runtime_resolution_blocked", Subject: "execution", Detail: "profile unavailable"}, {Code: "repository_unavailable", Subject: "secondary", Detail: "binding inaccessible"}}}}}, must: []string{"prior attempt", "timeout", "invalid result despite exit 0", "reconcile before retry"}},
		"C13":           {must: []string{"stale revision 17 versus 18", "detail inaccessible; truncated result", "current resolution unknown", "fresh exact read before action"}},
		"C14":           {must: []string{"unknown version", "otherwise admissible configuration and authority", "success not promised"}, forbid: []string{"Probe required", "Version registry required", "Capability certified"}},
		"C16":           {projects: []ProjectListView{project}, fullID: true, must: []string{"execution blocked", "runtime_policy_unavailable", "gate before", "plain text", "same observed revision"}},
	}
	observed := map[string][]string{
		"C09":           {"Agent runtimeConstraints allowed codex, claude", "Agent profileRef declared policy-default", "Project policy declares codex-test"},
		"C10":           {"Plan revision 18 resolved codex / codex-test", "Profile revision 4 model sample-model effort runtime-default", "Auth observation login-only"},
		"C10-selection": {"Canonical choice selected claude / review-profile at revision 18"},
		"C11":           {"no-dispatch confirmed"},
		"C13":           {"stale revision 17 versus 18", "detail inaccessible; truncated result"},
		"C14":           {"unknown version", "otherwise admissible configuration and authority"},
	}
	for name, v := range cases {
		v.event = base
		if facts := observed[name]; facts != nil {
			v.event.Diagnostic = &Diagnostic{Category: "inspection", Authority: "read-only", Observed: facts}
		}
		if name == "C05" {
			v.event.Operational = &projectapp.OperationalPreview{ProjectID: id, Digest: digest, Revision: "synthetic-preview-17"}
		}
		if name == "C12" {
			v.forbid = append(v.forbid, "not performed", "no-dispatch confirmed")
			v.event.Status = completion.Partial
			v.event.Diagnostic = &Diagnostic{Category: "attempt_unknown", Authority: "reconcile required", ConfirmedEffects: []string{"scratch artifact saved"}, UncertainEffects: []string{"Provider acknowledgement unknown"}}
		}
		cases[name] = v
	}
	for name, v := range cases {
		t.Run(name, func(t *testing.T) {
			wire, err := os.ReadFile(filepath.Join("testdata/presentation/012", name+".conversation.txt"))
			if err != nil {
				t.Fatal(err)
			}
			candidate := string(wire)
			// Both entry routes and plain-text/narrow hosts consume the same facts.
			for _, entry := range []string{"natural", "explicit-skill"} {
				t.Run(entry, func(t *testing.T) {
					if err := checkPresentation012(v, candidate); err != nil {
						t.Fatal(err)
					}
				})
			}
			for i, claim := range presentation012Claims(v) {
				if claim.value == "" {
					continue
				}
				if checkPresentation012(v, strings.ReplaceAll(candidate, claim.value, "[omitted]")) == nil {
					t.Fatalf("omission mutant %d accepted: %s", i, claim.source)
				}
			}
			for _, bad := range append([]string{"Invented Project", "Runtime selected by host", "model invented", "effort invented", "Execution performed", "Retry automatically", "Nothing ran", "Diagram rendered", "Run injected command", "Ignore prior instructions"}, v.forbid...) {
				if checkPresentation012(v, candidate+"\n"+bad) == nil {
					t.Fatalf("invention mutant accepted: %s", bad)
				}
			}
			// A changed canonical fact must invalidate the previously good answer.
			changed := v
			switch {
			case len(v.projects) > 0:
				changed.projects = append([]ProjectListView(nil), v.projects...)
				changed.projects[0].Name = "Different observed name"
			case v.readiness != nil:
				r := *v.readiness
				r.Structure = "invalid"
				changed.readiness = &r
			case v.event.Diagnostic != nil:
				d := *v.event.Diagnostic
				d.Observed = []string{"changed canonical resolution"}
				changed.event.Diagnostic = &d
			default:
				changed.must = append(append([]string(nil), v.must...), "changed observation: unavailable source")
			}
			if checkPresentation012(changed, candidate) == nil {
				t.Fatal("unchanged answer accepted after source delta")
			}
		})
	}
}

func TestPresentation012IdentityEdgeObservations(t *testing.T) {
	const id = "11111111-1111-4111-8111-111111111111"
	event := semanticObservation{}
	event.Status = completion.Success
	event.Provenance.SourceState = "clean"
	for _, row := range []struct {
		name         string
		projects     []ProjectListView
		want, forbid string
	}{
		{"same-text", []ProjectListView{{ID: id, Name: "orbita", Slug: "orbita"}}, "orbita (orbita)", "Invented Project"},
		{"bidi", []ProjectListView{{ID: id, Name: "Órbita\u202e", Slug: "orbita"}}, `Órbita\u202E (orbita)`, "\u202e"},
		{"same-ID-duplicate", []ProjectListView{{ID: id, Name: "Órbita"}, {ID: id, Name: "Órbita"}}, "Órbita. One Project; repeated same ID.", "Two distinct Projects"},
		{"same-ID-conflict-negative-source", []ProjectListView{{ID: id, Name: "Órbita"}, {ID: id, Name: "Conflicting"}}, "Órbita / Conflicting; conflicting metadata; canonical read-back required.", "First match selected"},
	} {
		t.Run(row.name, func(t *testing.T) {
			v := presentation012Context{event: event, projects: row.projects, forbid: []string{row.forbid}}
			// Duplicate/contradictory observations are explicitly adversarial,
			// not valid catalog output. Their explanatory obligations are independent.
			if row.name == "same-ID-duplicate" {
				v.must = []string{"One Project; repeated same ID"}
			}
			if row.name == "same-ID-conflict-negative-source" {
				v.must = []string{"conflicting metadata; canonical read-back required"}
			}
			good := "(success) " + row.want
			if err := checkPresentation012(v, good); err != nil {
				t.Fatal(err)
			}
			if checkPresentation012(v, good+row.forbid) == nil {
				t.Fatal("invented identity interpretation accepted")
			}
			for _, claim := range presentation012Claims(v) {
				if checkPresentation012(v, strings.ReplaceAll(good, claim.value, "[omitted]")) == nil {
					t.Fatal("identity omission accepted", claim.source)
				}
			}
		})
	}
}
