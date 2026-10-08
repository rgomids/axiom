package projectapp_test

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/project"
	"github.com/rgomids/axiom/internal/projectapp"
)

func readyProject(t *testing.T, edit func(*project.State)) project.Project {
	t.Helper()
	s := project.State{SchemaVersion: 3, ID: bootstrapID, Slug: "sample", Name: "Sample",
		Repositories:      project.Configured([]project.Repository{{Key: "core"}}),
		Providers:         project.Configured([]project.Provider{{Key: "work-items", ID: "github"}}),
		Integrations:      project.Configured([]project.Integration{{Key: "work-items", ProviderRef: project.Configured("work-items"), Capabilities: project.Configured([]string{"work-item"})}}),
		Runtimes:          project.Configured([]project.Runtime{{ID: "claude"}}),
		ModelProfiles:     project.Configured([]project.ModelProfile{{Key: "careful", RuntimeRef: project.Configured("claude"), Model: project.Configured("m")}}),
		TechnologyContext: project.Configured([]project.TechnologyFact{{Key: "language.go", Value: "go"}}),
		DocumentationSources: project.Configured([]project.DocumentationSource{
			{Key: "architecture", Kind: project.RepositorySource, RepositoryRef: project.Configured("core"), Path: project.Configured("docs")},
			{Key: "notes", Kind: project.LocalFileSource},
		}),
		BusinessContext: project.Configured(project.BusinessContext{Text: project.Configured("ctx"), Glossary: project.Configured([]project.GlossaryEntry{{Key: "k", Term: "T", Definition: "D"}})}),
	}
	if edit != nil {
		edit(&s)
	}
	p, issues := project.New(s)
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	return p
}

func readyInput(t *testing.T, edit func(*project.State)) projectapp.ReadinessInput {
	return projectapp.ReadinessInput{
		Project:       readyProject(t, edit),
		Local:         projectapp.LocalRecordState{Repositories: []projectapp.RepositoryBinding{{RepositoryKey: "core", ExplicitPath: "/w/core"}}},
		Repositories:  map[string]bool{"core": true},
		Documentation: map[string]projectapp.DocumentationStatus{"architecture": projectapp.DocumentationAvailable, "notes": projectapp.DocumentationAvailable},
		Catalog:       projectapp.SupportedProviders,
	}
}

func codes(findings []projectapp.Finding) string {
	out := []string{}
	for _, f := range findings {
		out = append(out, f.Code)
	}
	return strings.Join(out, ",")
}

func TestReadinessMatrix(t *testing.T) {
	for name, tc := range map[string]struct {
		edit      func(*project.State)
		input     func(*projectapp.ReadinessInput)
		effective string
		workItem  string
		execution string
		warnings  string
	}{
		"fully ready": {effective: "ready"},
		"runtime policy missing blocks only execution": {
			edit: func(s *project.State) {
				s.Runtimes, s.ModelProfiles = project.Declaration[[]project.Runtime]{}, project.Declaration[[]project.ModelProfile]{}
			},
			input:     func(i *projectapp.ReadinessInput) { i.RuntimeCode = "policy_unconfigured" },
			effective: "partial", execution: "runtime_policy_unavailable",
		},
		"runtime ambiguous/stale resolution blocks execution": {
			input:     func(i *projectapp.ReadinessInput) { i.RuntimeCode = "no_allowed_match" },
			effective: "partial", execution: "runtime_resolution_blocked",
		},
		"capability mapping missing blocks both": {
			edit: func(s *project.State) {
				s.Providers, s.Integrations = project.Unconfigured[[]project.Provider](), project.Unconfigured[[]project.Integration]()
			},
			effective: "blocked", workItem: "capability_mapping_missing", execution: "capability_mapping_missing",
		},
		"provider unsupported": {
			edit: func(s *project.State) {
				s.Providers = project.Configured([]project.Provider{{Key: "work-items", ID: "linear"}})
			},
			effective: "blocked", workItem: "provider_unsupported", execution: "provider_unsupported",
		},
		"two unconventional integrations are ambiguous": {
			edit: func(s *project.State) {
				s.Providers = project.Configured([]project.Provider{{Key: "gh", ID: "github"}, {Key: "tracker", ID: "jira"}})
				s.Integrations = project.Configured([]project.Integration{
					{Key: "issues", ProviderRef: project.Configured("gh"), Capabilities: project.Configured([]string{"work-item"})},
					{Key: "tracker", ProviderRef: project.Configured("tracker"), Capabilities: project.Configured([]string{"work-item"})},
				})
			},
			effective: "blocked", workItem: "capability_mapping_ambiguous", execution: "capability_mapping_ambiguous",
		},
		"conventional work-items integration keeps pre-231 mapping": {
			edit: func(s *project.State) {
				s.Providers = project.Configured([]project.Provider{{Key: "work-items", ID: "github"}, {Key: "tracker", ID: "jira"}})
				s.Integrations = project.Configured([]project.Integration{
					{Key: "work-items", ProviderRef: project.Configured("work-items"), Capabilities: project.Configured([]string{"work-item"})},
					{Key: "tracker", ProviderRef: project.Configured("tracker"), Capabilities: project.Configured([]string{"work-item"})},
				})
			},
			effective: "ready",
		},
		"non-conventional single mapping is ready": {
			edit: func(s *project.State) {
				s.Providers = project.Configured([]project.Provider{{Key: "gh", ID: "github"}})
				s.Integrations = project.Configured([]project.Integration{{Key: "issues", ProviderRef: project.Configured("gh"), Capabilities: project.Configured([]string{"work-item"})}})
			},
			effective: "ready",
		},
		"credential binding absent warns without blocking": {
			edit: func(s *project.State) {
				s.CredentialReferences = project.Configured([]project.CredentialReference{{Key: "github-token"}})
				s.Integrations = project.Configured([]project.Integration{{Key: "work-items", ProviderRef: project.Configured("work-items"), Capabilities: project.Configured([]string{"work-item"}), CredentialRef: project.Configured("github-token")}})
			},
			effective: "ready", warnings: "credential_binding_missing",
		},
		"repository binding missing": {
			input:     func(i *projectapp.ReadinessInput) { i.Local.Repositories = nil },
			effective: "blocked", workItem: "repository_binding_missing", execution: "repository_binding_missing",
		},
		"repository unavailable": {
			input:     func(i *projectapp.ReadinessInput) { i.Repositories["core"] = false },
			effective: "blocked", workItem: "repository_unavailable", execution: "repository_unavailable",
		},
		"documentation warnings never block": {
			input: func(i *projectapp.ReadinessInput) {
				i.Documentation = map[string]projectapp.DocumentationStatus{"architecture": projectapp.DocumentationMissing, "notes": projectapp.DocumentationUnbound}
			},
			effective: "ready", warnings: "documentation_binding_missing,documentation_unavailable",
		},
		"stale and unsafe documentation warn": {
			input: func(i *projectapp.ReadinessInput) {
				i.Documentation = map[string]projectapp.DocumentationStatus{"architecture": projectapp.DocumentationUnsafe, "notes": projectapp.DocumentationStale}
			},
			effective: "ready", warnings: "documentation_stale,documentation_unsafe",
		},
		"absent context warns": {
			edit: func(s *project.State) {
				s.TechnologyContext, s.BusinessContext = project.Declaration[[]project.TechnologyFact]{}, project.Declaration[project.BusinessContext]{}
			},
			effective: "ready", warnings: "business_context_absent,technology_context_absent",
		},
	} {
		t.Run(name, func(t *testing.T) {
			input := readyInput(t, tc.edit)
			if tc.input != nil {
				tc.input(&input)
			}
			report := projectapp.EvaluateReadiness(input)
			if report.Structure != "valid" || report.Effective != tc.effective {
				t.Fatalf("effective %s/%s, want %s", report.Structure, report.Effective, tc.effective)
			}
			if got := codes(report.Operation(projectapp.OperationWorkItem).Blockers); got != tc.workItem {
				t.Fatalf("work-item blockers %q, want %q", got, tc.workItem)
			}
			want := strings.Trim(strings.Join([]string{tc.workItem, tc.execution}, ","), ",")
			if tc.workItem == tc.execution {
				want = tc.workItem
			}
			if got := codes(report.Operation(projectapp.OperationExecution).Blockers); got != want {
				t.Fatalf("execution blockers %q, want %q", got, want)
			}
			if got := codes(report.Warnings); got != tc.warnings {
				t.Fatalf("warnings %q, want %q", got, tc.warnings)
			}
			again := projectapp.EvaluateReadiness(input)
			if !reflect.DeepEqual(report, again) {
				t.Fatal("readiness not deterministic")
			}
			if report.Authority.GrantsAuthority {
				t.Fatal("readiness granted authority")
			}
		})
	}
}

func TestReadinessReportCarriesNoSecretsPathsOrContent(t *testing.T) {
	input := readyInput(t, func(s *project.State) {
		s.CredentialReferences = project.Configured([]project.CredentialReference{{Key: "github-token"}})
		s.Integrations = project.Configured([]project.Integration{{Key: "work-items", ProviderRef: project.Configured("work-items"), Capabilities: project.Configured([]string{"work-item"}), CredentialRef: project.Configured("github-token")}})
	})
	input.Local.Credentials = []projectapp.CredentialBinding{{ReferenceKey: "github-token", SourceKind: "environment", ItemReference: "SYNTHETIC_SECRET_ITEM"}}
	input.Local.Documentation = []projectapp.DocumentationBinding{{SourceKey: "notes", ExplicitPath: "/home/user/private/notes.md"}}
	report := projectapp.EvaluateReadiness(input)
	wire, _ := json.Marshal(report)
	for _, leak := range []string{"SYNTHETIC_SECRET_ITEM", "environment", "/home/user", "/w/core"} {
		if strings.Contains(string(wire), leak) {
			t.Fatalf("readiness leaked %q: %s", leak, wire)
		}
	}
	if report.Capabilities[0].Credential != "bound" || report.Effective != "ready" {
		t.Fatalf("bound credential not recognized: %+v", report.Capabilities[0])
	}
}

type fakeReadinessPorts struct {
	subject projectapp.ReadinessSubject
	code    string
	runtime string
	calls   *[]string
}

func (f fakeReadinessPorts) LoadReadiness(_ context.Context, selector string) (projectapp.ReadinessSubject, string) {
	*f.calls = append(*f.calls, "load:"+selector)
	return f.subject, f.code
}
func (f fakeReadinessPorts) RepositoryAvailable(_ context.Context, b projectapp.RepositoryBinding) bool {
	*f.calls = append(*f.calls, "repo:"+b.RepositoryKey)
	return true
}
func (f fakeReadinessPorts) ResolveDocumentation(_ context.Context, r projectapp.DocumentationRequest) projectapp.DocumentationStatus {
	*f.calls = append(*f.calls, "doc:"+r.Source.Key+":"+r.RepositoryPath)
	return projectapp.DocumentationAvailable
}
func (f fakeReadinessPorts) Availability(_ context.Context, id string) string {
	*f.calls = append(*f.calls, "runtime:"+id)
	return f.runtime
}

func TestProjectReadinessGathersThroughPortsOnly(t *testing.T) {
	calls := []string{}
	ports := fakeReadinessPorts{subject: projectapp.ReadinessSubject{Project: readyProject(t, nil), Local: readyInput(t, nil).Local}, calls: &calls}
	service := projectapp.ProjectReadiness{Projects: ports, Repositories: ports, Documentation: ports, Runtime: ports, Catalog: projectapp.SupportedProviders}
	report := service.Evaluate(context.Background(), "sample")
	if report.Effective != "ready" {
		t.Fatalf("unexpected %+v", report)
	}
	want := []string{"load:sample", "repo:core", "doc:architecture:/w/core", "doc:notes:", "runtime:" + bootstrapID}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("calls %v", calls)
	}
	for _, code := range []string{"project_not_installed", "recovery_required", "installation_stale", "project_source_unavailable", "project_state_invalid"} {
		calls = nil
		blocked := projectapp.ProjectReadiness{Projects: fakeReadinessPorts{code: code, calls: &calls}, Repositories: ports, Documentation: ports, Runtime: ports}.Evaluate(context.Background(), "sample")
		if blocked.Effective != "blocked" || codes(blocked.Operation(projectapp.OperationWorkItem).Blockers) != code || codes(blocked.Operation(projectapp.OperationExecution).Blockers) != code {
			t.Fatalf("%s not blocking both operations: %+v", code, blocked)
		}
		if len(calls) != 1 {
			t.Fatalf("%s continued observing after structural failure: %v", code, calls)
		}
	}
	if (projectapp.ProjectReadiness{}).Evaluate(context.Background(), "x").Effective != "blocked" {
		t.Fatal("unconfigured evaluator not fail-closed")
	}
}
