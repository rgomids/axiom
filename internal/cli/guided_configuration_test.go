package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"reflect"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/completion"
	"github.com/rgomids/axiom/internal/projectapp"
)

type guidedConfigurationService struct {
	lifecycleRecorder
	response   Result
	configured []ConfigureInput
	validated  []ProjectLifecycleInput
}

func (s *guidedConfigurationService) Configure(_ context.Context, in ConfigureInput) Result {
	s.configured = append(s.configured, in)
	return s.response
}

func (s *guidedConfigurationService) ProjectValidateInstalled(_ context.Context, in ProjectLifecycleInput) Result {
	s.validated = append(s.validated, in)
	return s.response
}

func TestGuidedConfigurationJSONReadinessPreservesIndependentFacts(t *testing.T) {
	for _, scenario := range []struct {
		name, effective, execution, blocker string
		absent                              bool
	}{
		{"C01-ready", "ready", "ready", "", false},
		{"C04-policy-unavailable", "partial", "blocked", "runtime_policy_unavailable", false},
		{"C12-synthetic-ready", "ready", "ready", "", false},
		{"C13-resolution-blocked", "partial", "blocked", "runtime_resolution_blocked", false},
		{"readiness-not-observed", "", "", "", true},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			result := canonicalResult(t, completion.Success, []string{"project:123e4567-e89b-42d3-a456-426614174000"}, "Inspect before any separate start", completionProvenance(t))
			response := Result{Completion: &result}
			if !scenario.absent {
				execution := projectapp.OperationReadiness{Operation: projectapp.OperationExecution, Status: scenario.execution}
				if scenario.blocker != "" {
					execution.Blockers = []projectapp.Finding{{Code: scenario.blocker, Subject: "execution", Detail: "synthetic observation; no new proof"}}
				}
				response.Readiness = &projectapp.ReadinessReport{Structure: "valid", Effective: scenario.effective, Operations: []projectapp.OperationReadiness{{Operation: projectapp.OperationWorkItem, Status: "ready"}, execution}, Warnings: []projectapp.Finding{{Code: "documentation_binding_missing", Subject: "notes"}}}
				response.Admission = &projectapp.AdmissionDecision{Operation: projectapp.AdmitProjectEdit, Class: projectapp.AdmissionAdministrative, Allowed: true, ProjectID: "123e4567-e89b-42d3-a456-426614174000", GrantsAuthority: false}
			}
			service := &guidedConfigurationService{response: response}
			var output bytes.Buffer
			if code := RunInteractive(context.Background(), []string{"--json", "project", "validate", "--project", "alpha"}, service, completionProvenance(t), nil, &output, io.Discard); code != ExitSuccess {
				t.Fatalf("exit=%d output=%s", code, &output)
			}
			var event readinessCompletionEvent
			if err := json.Unmarshal(output.Bytes(), &event); err != nil {
				t.Fatal(err)
			}
			if event.Status != completion.Success || !reflect.DeepEqual(event.Readiness, response.Readiness) || !reflect.DeepEqual(event.Admission, response.Admission) {
				t.Fatalf("completion/readiness/authority changed: %+v", event)
			}
			if len(service.validated) != 1 || service.validated[0].Project != "alpha" || len(service.configured) != 0 || service.call != "" {
				t.Fatalf("unexpected dispatch: %+v", service)
			}
		})
	}
}

func TestGuidedConfigurationPolicyJSONPreservesPreviewAndReadBack(t *testing.T) {
	const id = "123e4567-e89b-42d3-a456-426614174000"
	for _, scenario := range []struct {
		name                  string
		status                completion.Status
		outcome               projectapp.EditOutcome
		publication           projectapp.EditPublicationStatus
		readBack              string
		committed, authorized bool
	}{
		{"preview", completion.Success, projectapp.EditPreviewed, "", "", false, false},
		{"C07-denied", completion.DeniedAuthority, projectapp.EditDenied, "", "", false, true},
		{"C08-stale", completion.DeniedAuthority, projectapp.EditConflicted, projectapp.EditPublicationConflict, "", false, true},
		{"C09-unverified", completion.Partial, projectapp.EditPublished, projectapp.EditPublicationApplied, "unverified", true, true},
		{"C09-drift", completion.Partial, projectapp.EditPublished, projectapp.EditPublicationApplied, "drift", true, true},
		{"C12-synthetic-confirmed", completion.Success, projectapp.EditPublished, projectapp.EditPublicationApplied, "confirmed", true, true},
		{"C13-confirmed-execution-blocked", completion.Success, projectapp.EditPublished, projectapp.EditPublicationApplied, "confirmed", true, true},
		{"noop", completion.Success, projectapp.EditUnchanged, projectapp.EditPublicationUnchanged, "confirmed", false, true},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			digest := strings.Repeat("a1", 32)
			portableRevision, localRevision := strings.Repeat("b2", 32), strings.Repeat("c3", 32)
			preview := projectapp.PolicyEditPreview{Operation: string(projectapp.AdmitProjectEdit), Scope: "runtime_policy", ProjectID: id, SchemaVersion: 3, PortableRevision: portableRevision, LocalRevision: localRevision, Digest: digest,
				Diff:           []projectapp.PolicyChange{{Collection: "runtimes", Before: json.RawMessage(`"absent"`), After: json.RawMessage(`"configured"`)}},
				Workflow:       projectapp.PolicyWorkflowReport{Workflow: projectapp.ActiveWorkflow{Status: "selected", WorkflowID: "custom", Source: "project", Revision: 2, Digest: strings.Repeat("d4", 32)}, Compatible: true, Inventory: "unknown_uninspected", Scopes: []string{"work_items_no_effects", "other_projects_local_configuration_unchanged"}},
				Admission:      projectapp.AdmissionDecision{Operation: projectapp.AdmitProjectEdit, Allowed: true, ProjectID: id, GrantsAuthority: false},
				Runtime:        []projectapp.PolicyRuntimeReadiness{{Code: "no_allowed_match", Blocker: "runtime_resolution_blocked", Next: "Inspect local candidate"}},
				Readiness:      []projectapp.OperationReadiness{{Operation: projectapp.OperationExecution, Status: "blocked", Blockers: []projectapp.Finding{{Code: "runtime_resolution_blocked", Subject: "execution"}}}},
				ReadinessBasis: "read_only_metadata_no_new_runtime_proof", Effects: []projectapp.EditEffect{{Scope: projectapp.PortableScope, Code: "update_portable_project"}}}
			if err := json.Unmarshal([]byte(`{"runtimes":{"form":"configured","values":[{"id":"claude"}]},"modelProfiles":{"form":"configured","values":[{"key":"worker","runtimeRef":"claude","model":"synthetic-model"}]},"runtimePreferences":{"form":"configured"}}`), &preview.Policy); err != nil {
				t.Fatal(err)
			}
			if scenario.name == "noop" {
				preview.Diff, preview.Effects = nil, nil
			}
			if scenario.name == "C12-synthetic-confirmed" {
				preview.Readiness = []projectapp.OperationReadiness{{Operation: projectapp.OperationExecution, Status: "ready"}}
				preview.Runtime = nil
			}
			report := projectapp.PolicyEditReport{Outcome: scenario.outcome, Preview: preview, Publication: scenario.publication, LocalCommitted: scenario.committed}
			if scenario.readBack != "" {
				observed := preview
				observed.PortableRevision = strings.Repeat("e5", 32)
				report.ReadBack = &projectapp.PolicyEditReadBack{Status: scenario.readBack, ExpectedPortableRevision: observed.PortableRevision, ExpectedLocalRevision: localRevision, ReadinessChanged: scenario.readBack == "drift"}
				if scenario.readBack == "drift" {
					observed.PortableRevision = strings.Repeat("f6", 32)
				}
				if scenario.readBack != "unverified" {
					report.ReadBack.Observed = &observed
				}
			}
			result := canonicalResult(t, scenario.status, []string{"project:" + id, "workflow:project/custom/2"}, "Read canonical state; no Execution authority", completionProvenance(t))
			service := &guidedConfigurationService{response: Result{Completion: &result, PolicyEdit: &report}}
			draft := `{"runtimes":{"mode":"members","add":[{"id":"claude"}]}}`
			if _, failure := projectapp.DecodePolicyDraft([]byte(draft)); failure != projectapp.EditOK {
				t.Fatal("synthetic draft violates owner contract", failure)
			}
			args := []string{"--json", "project", "configure", "--project", "alpha", "--policy-draft", draft}
			if scenario.authorized {
				previewResult := canonicalResult(t, completion.Success, result.References(), "Review exact preview", completionProvenance(t))
				service.response = Result{Completion: &previewResult, PolicyEdit: &projectapp.PolicyEditReport{Outcome: projectapp.EditPreviewed, Preview: preview}}
				var previewOutput bytes.Buffer
				if code := RunInteractive(context.Background(), args, service, completionProvenance(t), nil, &previewOutput, io.Discard); code != ExitSuccess {
					t.Fatalf("preview exit=%d: %s", code, &previewOutput)
				}
				var reviewed struct {
					Policy projectapp.PolicyEditReport `json:"policyEdit"`
				}
				if err := json.Unmarshal(previewOutput.Bytes(), &reviewed); err != nil || !reflect.DeepEqual(reviewed.Policy.Preview, preview) {
					t.Fatal("preview facts changed", err)
				}
				in := service.configured[0]
				if in.AuthorizeLocal || in.PreviewDigest != "" || in.ProjectID != "" {
					t.Fatal("preview granted authority", in)
				}
				args = append(args, "--project-id", reviewed.Policy.Preview.ProjectID, "--preview-digest", reviewed.Policy.Preview.Digest, "--authorize-local")
				service.response = Result{Completion: &result, PolicyEdit: &report}
			}
			var output bytes.Buffer
			if code := RunInteractive(context.Background(), args, service, completionProvenance(t), nil, &output, io.Discard); code != completionExitCode(scenario.status) {
				t.Fatalf("exit=%d output=%s", code, &output)
			}
			var event struct {
				completionEvent
				Policy projectapp.PolicyEditReport `json:"policyEdit"`
			}
			if err := json.Unmarshal(output.Bytes(), &event); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(event.Policy, report) || event.Status != scenario.status || !reflect.DeepEqual(event.References, result.References()) || event.Next != result.Next().String() {
				t.Fatalf("material facts changed: %s", &output)
			}
			expectedCalls := 1
			if scenario.authorized {
				expectedCalls++
			}
			if len(service.configured) != expectedCalls || len(service.validated) != 0 || service.call != "" {
				t.Fatalf("presentation dispatched another operation: %+v", service)
			}
			in := service.configured[len(service.configured)-1]
			if in.Project != "alpha" || in.PolicyDraft == nil || *in.PolicyDraft != draft || in.AuthorizeLocal != scenario.authorized || (scenario.authorized && (in.ProjectID != id || in.PreviewDigest != event.Policy.Preview.Digest)) {
				t.Fatalf("preview authority changed: %+v", in)
			}
		})
	}
}

func TestGuidedConfigurationOutputFailureDoesNotRepeatPolicyCall(t *testing.T) {
	result := canonicalResult(t, completion.Partial, []string{"project:123e4567-e89b-42d3-a456-426614174000"}, "Inspect uncertain publication", completionProvenance(t))
	for _, writer := range []io.Writer{shortWriter{}, failingWriter{}} {
		service := &guidedConfigurationService{response: Result{Completion: &result, PolicyEdit: &projectapp.PolicyEditReport{Outcome: projectapp.EditPublished, LocalCommitted: true, ReadBack: &projectapp.PolicyEditReadBack{Status: "unverified"}}}}
		args := []string{"--json", "project", "configure", "--project", "alpha", "--policy-draft", "{}", "--project-id", "123e4567-e89b-42d3-a456-426614174000", "--preview-digest", strings.Repeat("a", 64), "--authorize-local"}
		if code := RunInteractive(context.Background(), args, service, completionProvenance(t), nil, writer, io.Discard); code != ExitFailure || len(service.configured) != 1 || len(service.validated) != 0 || service.call != "" {
			t.Fatalf("exit=%d service=%+v", code, service)
		}
	}
}
