package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/completion"
	"github.com/rgomids/axiom/internal/projectapp"
	"github.com/rgomids/axiom/internal/provenance"
)

const validDigest = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

type integrationRecordingService struct {
	recordingService
	operation string
	input     IntegrationInput
	response  Result
	calls     int
}

func (s *integrationRecordingService) record(operation string, input IntegrationInput) Result {
	s.operation, s.input = operation, input
	s.calls++
	return s.response
}

func (s *integrationRecordingService) IntegrationList(_ context.Context, i IntegrationInput) Result {
	return s.record("list", i)
}
func (s *integrationRecordingService) IntegrationShow(_ context.Context, i IntegrationInput) Result {
	return s.record("show", i)
}
func (s *integrationRecordingService) IntegrationValidate(_ context.Context, i IntegrationInput) Result {
	return s.record("validate", i)
}
func (s *integrationRecordingService) IntegrationDisable(_ context.Context, i IntegrationInput) Result {
	return s.record("disable", i)
}
func (s *integrationRecordingService) IntegrationEnable(_ context.Context, i IntegrationInput) Result {
	return s.record("enable", i)
}
func (s *integrationRecordingService) IntegrationRemove(_ context.Context, i IntegrationInput) Result {
	return s.record("remove", i)
}

func TestIntegrationGroupParsesExactInputsAndDelegates(t *testing.T) {
	for name, test := range map[string]struct {
		args      []string
		operation string
		input     IntegrationInput
	}{
		"list":              {[]string{"integration", "list", "--project", "sample"}, "list", IntegrationInput{Project: "sample"}},
		"show":              {[]string{"integration", "show", "--project", "sample", "--integration", "work-items"}, "show", IntegrationInput{Project: "sample", Integration: "work-items"}},
		"validate all":      {[]string{"integration", "validate", "--project", "sample"}, "validate", IntegrationInput{Project: "sample"}},
		"validate filtered": {[]string{"integration", "validate", "--project=sample", "--integration=chat"}, "validate", IntegrationInput{Project: "sample", Integration: "chat"}},
		"disable preview":   {[]string{"integration", "disable", "--project", "sample", "--integration", "chat"}, "disable", IntegrationInput{Project: "sample", Integration: "chat"}},
		"disable apply":     {[]string{"integration", "disable", "--project", "sample", "--integration", "chat", "--preview-digest", validDigest, "--authorize-local"}, "disable", IntegrationInput{Project: "sample", Integration: "chat", PreviewDigest: validDigest, AuthorizeLocal: true}},
		"enable apply":      {[]string{"integration", "enable", "--project", "sample", "--integration", "chat", "--preview-digest", validDigest, "--authorize-local"}, "enable", IntegrationInput{Project: "sample", Integration: "chat", PreviewDigest: validDigest, AuthorizeLocal: true}},
		"remove apply":      {[]string{"integration", "remove", "--project", "sample", "--integration", "chat", "--project-id", "123e4567-e89b-42d3-a456-426614174000", "--preview-digest", validDigest, "--authorize-local"}, "remove", IntegrationInput{Project: "sample", Integration: "chat", ProjectID: "123e4567-e89b-42d3-a456-426614174000", PreviewDigest: validDigest, AuthorizeLocal: true}},
	} {
		t.Run(name, func(t *testing.T) {
			service := &integrationRecordingService{response: Result{Status: Succeeded}}
			var output bytes.Buffer
			if code := Run(context.Background(), test.args, service, completionProvenance(t), &output); code != ExitSuccess {
				t.Fatalf("code=%d output=%s", code, &output)
			}
			if service.calls != 1 || service.call != "" || service.operation != test.operation || !reflect.DeepEqual(service.input, test.input) {
				t.Fatalf("calls=%d operation=%s input=%+v", service.calls, service.operation, service.input)
			}
		})
	}
}

func TestIntegrationGroupRejectsInvalidInputBeforeAnyService(t *testing.T) {
	const secret = "private-input-marker"
	base := func(verb string, extra ...string) []string {
		return append([]string{"integration", verb, "--project", "sample"}, extra...)
	}
	for name, args := range map[string][]string{
		"missing project":           {"integration", "list"},
		"empty project":             {"integration", "list", "--project", ""},
		"unknown flag":              base("list", "--"+secret, "x"),
		"duplicate project":         base("list", "--project", "other"),
		"single hyphen":             base("list", "-integration", "chat"),
		"positional":                base("list", secret),
		"list with integration":     base("list", "--integration", "chat"),
		"show without integration":  base("show"),
		"empty integration":         base("show", "--integration", ""),
		"duplicate integration":     base("show", "--integration", "a", "--integration", "b"),
		"disable without key":       base("disable"),
		"enable without key":        base("enable", "--authorize-local"),
		"validate preview digest":   base("validate", "--preview-digest", validDigest),
		"show authorize":            base("show", "--integration", "chat", "--authorize-local"),
		"disable project-id":        base("disable", "--integration", "chat", "--project-id", "x"),
		"bad digest":                base("disable", "--integration", "chat", "--preview-digest", "ABC"),
		"upper digest":              base("remove", "--integration", "chat", "--preview-digest", strings.ToUpper(validDigest)),
		"duplicate authorize":       base("remove", "--integration", "chat", "--authorize-local", "--authorize-local"),
		"separator":                 base("show", "--integration", "chat", "--"),
		"oversize project":          {"integration", "list", "--project", strings.Repeat("a", 257)},
		"remove duplicate id":       base("remove", "--integration", "chat", "--project-id", "a", "--project-id", "b"),
		"validate duplicate filter": base("validate", "--integration=a", "--integration=b"),
	} {
		t.Run(name, func(t *testing.T) {
			for _, mode := range []string{"--json", "--human"} {
				service := &integrationRecordingService{}
				var output bytes.Buffer
				code := RunInteractive(context.Background(), append([]string{mode}, args...), service, completionProvenance(t), strings.NewReader(""), &output, &bytes.Buffer{})
				if code != ExitFailure || service.calls != 0 || strings.Contains(output.String(), secret) || !strings.Contains(output.String(), "Integration input is invalid") {
					t.Fatalf("%s: code=%d calls=%d output=%s", mode, code, service.calls, &output)
				}
			}
		})
	}
}

func TestIntegrationGroupUnknownVerbAndMissingService(t *testing.T) {
	for _, args := range [][]string{{"integration"}, {"integration", "update", "--project", "sample"}, {"integration", "configure"}} {
		var output bytes.Buffer
		service := &integrationRecordingService{}
		if code := Run(context.Background(), args, service, completionProvenance(t), &output); code != ExitFailure || service.calls != 0 || !strings.Contains(output.String(), "invalid_command") {
			t.Fatalf("%v: code=%d output=%s", args, code, &output)
		}
	}
	var output bytes.Buffer
	if code := Run(context.Background(), []string{"integration", "list", "--project", "sample"}, &recordingService{}, completionProvenance(t), &output); code != ExitFailure || !strings.Contains(output.String(), "application_unavailable") {
		t.Fatalf("unsupported service: code=%d output=%s", code, &output)
	}
}

func TestIntegrationReportRendersJSONAndHumanWithoutLeakingMachineData(t *testing.T) {
	report := &projectapp.IntegrationReport{
		ProjectID: "123e4567-e89b-42d3-a456-426614174000",
		Integrations: []projectapp.IntegrationView{
			{Key: "chat", ProviderRef: "chat", Provider: "slack", Capabilities: []string{"chat"}, Transport: "mcp", CredentialRef: "chat-token", Local: projectapp.IntegrationDisabled},
		},
		StaleDisabled: []string{"gone"},
		Validation:    &projectapp.IntegrationValidation{Status: projectapp.IntegrationInvalid, Findings: []projectapp.IntegrationFinding{{Code: "provider_unsupported", Severity: "error", Integration: "chat", Capability: "chat"}}},
	}
	service := &integrationRecordingService{}
	fixed := canonicalIntegrationResult(t, completion.Facts{ValidationFailed: true})
	fixed.Integrations, fixed.Category = report, "integration_invalid"
	service.response = fixed

	var output bytes.Buffer
	if code := Run(context.Background(), []string{"integration", "validate", "--project", "sample"}, service, completionProvenance(t), &output); code != ExitFailure {
		t.Fatalf("code=%d output=%s", code, &output)
	}
	var event struct {
		Status       completion.Status             `json:"status"`
		Category     string                        `json:"category"`
		Integrations *projectapp.IntegrationReport `json:"integrations"`
	}
	if err := json.Unmarshal(output.Bytes(), &event); err != nil || event.Status != completion.ValidationFailure || event.Category != "integration_invalid" || !reflect.DeepEqual(event.Integrations, report) {
		t.Fatalf("event=%+v err=%v output=%s", event, err, &output)
	}
	output.Reset()
	if code := RunInteractive(context.Background(), []string{"--human", "integration", "validate", "--project", "sample"}, service, completionProvenance(t), nil, &output, &bytes.Buffer{}); code != ExitFailure {
		t.Fatalf("human code=%d output=%s", code, &output)
	}
	for _, expected := range []string{"category: integration_invalid", "integration: chat local=disabled provider=slack providerRef=chat capabilities=[chat] transport=mcp credentialRef=chat-token", "stale-disabled: gone", "validation: invalid", "finding: error provider_unsupported integration=chat capability=chat"} {
		if !strings.Contains(output.String(), expected) {
			t.Fatalf("%q absent from %s", expected, &output)
		}
	}
	// A refusal without a report still carries its stable category.
	refusal := canonicalIntegrationResult(t, completion.Facts{ValidationFailed: true})
	refusal.Category = "integration_not_found"
	service.response = refusal
	output.Reset()
	if code := Run(context.Background(), []string{"integration", "show", "--project", "sample", "--integration", "chat"}, service, completionProvenance(t), &output); code != ExitFailure || !strings.Contains(output.String(), `"category":"integration_not_found"`) || strings.Contains(output.String(), `"integrations"`) {
		t.Fatalf("code=%d output=%s", code, &output)
	}
}

func canonicalIntegrationResult(t *testing.T, facts completion.Facts) Result {
	t.Helper()
	statement, err := provenance.NewText("Integration declarations are invalid", provenance.AxiomAuthored)
	if err != nil {
		t.Fatal(err)
	}
	next, err := provenance.NewText("Correct the reported declarations", provenance.AxiomAuthored)
	if err != nil {
		t.Fatal(err)
	}
	result, err := completion.New(facts, statement, []string{"project:123e4567-e89b-42d3-a456-426614174000"}, next, "", completionProvenance(t))
	if err != nil {
		t.Fatal(err)
	}
	return Result{Status: Failed, Completion: &result}
}
