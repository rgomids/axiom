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
