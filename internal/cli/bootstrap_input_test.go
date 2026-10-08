package cli

import (
	"bytes"
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/completion"
	"github.com/rgomids/axiom/internal/projectapp"
)

func TestCreateConfigureInputSplitsBootstrapSyntaxOnly(t *testing.T) {
	service := newEditRecordingService(t)
	args := []string{"--json", "project", "configure", "--slug", "sample", "--name", "Sample",
		"--repository", "core=/work/core", "--repository", "/work/Web App", "--work-item-provider", "none",
		"--repository-remote", "core=https://github.com/acme/core.git", "--runtime", "claude", "--model-profile", "careful",
		"--runtime-preference", "implementation/high=careful", "--technology", "cloud.aws=aws", "--remove-technology", "package-manager.npm",
		"--documentation", "arch=repository:core/docs/architecture", "--documentation", "notes=local-file:/home/user/notes.md",
		"--business-context", "Context", "--context-source", "arch", "--glossary", "work-item=Work Item: A bounded unit"}
	var output bytes.Buffer
	RunInteractive(context.Background(), args, service, completionProvenance(t), nil, &output, nil)
	if len(service.inputs) != 1 {
		t.Fatalf("configure not dispatched: %s", output.String())
	}
	want := ConfigureInput{Slug: "sample", Name: "Sample", Repositories: []RepositoryInput{{Key: "core", Path: "/work/core"}, {Path: "/work/Web App"}},
		RepositoryRemotes: map[string]string{"core": "https://github.com/acme/core.git"}, Runtimes: []string{"claude"}, ModelProfiles: []string{"careful"},
		RuntimePreferences: []RuntimePreferenceInput{{Role: "implementation", Complexity: "high", ModelProfile: "careful"}},
		Technology:         []KeyValueInput{{Key: "cloud.aws", Value: "aws"}}, RemoveTechnology: []string{"package-manager.npm"},
		Documentation:   []DocumentationInput{{Key: "arch", Kind: "repository", Repository: "core", Path: "docs/architecture"}, {Key: "notes", Kind: "local-file", Path: "/home/user/notes.md"}},
		BusinessContext: "Context", ContextSources: []string{"arch"}, Glossary: []GlossaryInput{{Key: "work-item", Term: "Work Item", Definition: "A bounded unit"}}}
	if !reflect.DeepEqual(service.inputs[0], want) {
		t.Fatalf("got %#v\nwant %#v", service.inputs[0], want)
	}
}

func TestBootstrapSyntaxErrorsAndEditModeRejection(t *testing.T) {
	for name, extra := range map[string][]string{
		"remote without key":       {"--repository-remote", "https://x"},
		"duplicate remote":         {"--repository-remote", "core=none", "--repository-remote", "core=none"},
		"preference shape":         {"--runtime-preference", "implementation=careful"},
		"documentation kind":       {"--documentation", "x=notion:page"},
		"repository doc no path":   {"--documentation", "x=repository:core"},
		"glossary without term":    {"--glossary", "k=definition only"},
		"technology without value": {"--technology", "language.go"},
	} {
		t.Run(name, func(t *testing.T) {
			service := newEditRecordingService(t)
			args := append([]string{"--json", "project", "configure", "--slug", "s", "--name", "S", "--repository", "core=/w", "--work-item-provider", "none"}, extra...)
			var output bytes.Buffer
			if code := RunInteractive(context.Background(), args, service, completionProvenance(t), nil, &output, nil); code == ExitSuccess || len(service.inputs) != 0 {
				t.Fatalf("malformed bootstrap input dispatched: %s", output.String())
			}
		})
	}
	for _, flag := range [][]string{{"--runtime", "claude"}, {"--technology", "a=b"}, {"--documentation", "a=local-file:/x"}, {"--repository-remote", "a=none"}, {"--glossary", "k=T:D"}} {
		service := newEditRecordingService(t)
		var output bytes.Buffer
		args := append([]string{"--json", "project", "configure", "--project", "sample"}, flag...)
		if code := RunInteractive(context.Background(), args, service, completionProvenance(t), nil, &output, nil); code == ExitSuccess || len(service.inputs) != 0 {
			t.Fatalf("EDIT accepted CREATE-only bootstrap flag %v", flag)
		}
	}
}

type readinessService struct {
	recordingService
	result Result
}

func (s readinessService) Validate(context.Context, ProjectInput) Result { return s.result }

func TestReadinessRenderingIsBoundedAndStable(t *testing.T) {
	result := canonicalResult(t, completion.Success, nil, "", completionProvenance(t))
	report := projectapp.ReadinessReport{Structure: "valid", Effective: "partial", Repositories: []projectapp.RepositoryReadiness{{Key: "core", Binding: "bound", Availability: "available", Remote: "declared"}},
		Operations: []projectapp.OperationReadiness{{Operation: projectapp.OperationWorkItem, Status: "ready", Blockers: []projectapp.Finding{}}, {Operation: projectapp.OperationExecution, Status: "blocked", Blockers: []projectapp.Finding{{Code: "runtime_policy_unavailable", Detail: "policy_unconfigured"}}}},
		Warnings:   []projectapp.Finding{{Code: "documentation_binding_missing", Subject: "notes"}}}
	service := readinessService{result: Result{Completion: &result, Readiness: &report}}
	for _, mode := range []string{"--json", "--human"} {
		var output bytes.Buffer
		if code := RunInteractive(context.Background(), []string{mode, "project", "validate", "--slug", "sample"}, &service, completionProvenance(t), nil, &output, nil); code != ExitSuccess {
			t.Fatalf("%s exit %d: %s", mode, code, output.String())
		}
		for _, want := range map[string][]string{
			"--json":  {`"readiness":{`, `"effective":"partial"`, `"code":"runtime_policy_unavailable"`},
			"--human": {"readiness: partial structure=valid", "operation: execution blocked", "blocker: runtime_policy_unavailable", "warning: documentation_binding_missing notes"},
		}[mode] {
			if !strings.Contains(output.String(), want) {
				t.Fatalf("%s output lacks %q:\n%s", mode, want, output.String())
			}
		}
	}
}
