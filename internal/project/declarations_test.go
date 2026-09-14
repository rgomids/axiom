package project_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/project"
)

func checkForms[T any](t *testing.T, set func(*project.State, project.Declaration[T]), configured T, unconfiguredAllowed bool) {
	t.Helper()
	forms := []project.Declaration[T]{{}, project.Unconfigured[T](), project.Configured(configured)}
	for i, form := range forms {
		s := minimal()
		set(&s, form)
		p, issues := project.New(s)
		if i == 1 && !unconfiguredAllowed {
			if len(issues) == 0 {
				t.Fatal("forbidden unconfigured accepted")
			}
			continue
		}
		if len(issues) != 0 {
			t.Fatal(issues)
		}
		repeated, issues := p.Propose(project.Intent{})
		if len(issues) != 0 || !p.Equivalent(repeated) {
			t.Fatal("omission changed declaration")
		}
		for j, other := range forms {
			if j == 1 && !unconfiguredAllowed {
				continue
			}
			s := minimal()
			set(&s, other)
			if p.Equivalent(valid(t, s)) != (i == j) {
				t.Fatal("different declaration forms collapsed")
			}
		}
	}
}

func TestAllTopLevelDeclarationForms(t *testing.T) {
	t.Run("repositories", func(t *testing.T) {
		checkForms(t, func(s *project.State, d project.Declaration[[]project.Repository]) { s.Repositories = d }, []project.Repository{}, false)
	})
	t.Run("runtime", func(t *testing.T) {
		checkForms(t, func(s *project.State, d project.Declaration[project.Runtime]) { s.Runtime = d }, project.Runtime{ID: "unconfigured"}, true)
	})
	t.Run("providers", func(t *testing.T) {
		checkForms(t, func(s *project.State, d project.Declaration[[]project.Provider]) { s.Providers = d }, []project.Provider{}, true)
	})
	t.Run("integrations", func(t *testing.T) {
		checkForms(t, func(s *project.State, d project.Declaration[[]project.Integration]) { s.Integrations = d }, []project.Integration{}, true)
	})
	t.Run("profiles", func(t *testing.T) {
		checkForms(t, func(s *project.State, d project.Declaration[[]project.ModelProfile]) { s.ModelProfiles = d }, []project.ModelProfile{}, true)
	})
	t.Run("context", func(t *testing.T) {
		checkForms(t, func(s *project.State, d project.Declaration[project.BusinessContext]) { s.BusinessContext = d }, project.BusinessContext{}, true)
	})
	t.Run("credentials", func(t *testing.T) {
		checkForms(t, func(s *project.State, d project.Declaration[[]project.CredentialReference]) {
			s.CredentialReferences = d
		}, []project.CredentialReference{}, true)
	})
	t.Run("policies", func(t *testing.T) {
		checkForms(t, func(s *project.State, d project.Declaration[[]string]) { s.Policies = d }, []string{}, true)
	})
}

func TestNestedOptionalStrings(t *testing.T) {
	fields := []struct {
		name         string
		value        string
		blankAllowed bool
		set          func(*project.State, project.Declaration[string])
	}{
		{"remote", "https://example.com/Repo", false, func(s *project.State, d project.Declaration[string]) {
			s.Repositories = project.Configured([]project.Repository{{Key: "repo", Remote: d}})
		}},
		{"providerRef", "provider", false, func(s *project.State, d project.Declaration[string]) {
			s.Providers = project.Configured([]project.Provider{{Key: "provider", ID: "opaque"}})
			s.Integrations = project.Configured([]project.Integration{{Key: "integration", ProviderRef: d}})
		}},
		{"credentialRef", "reference", false, func(s *project.State, d project.Declaration[string]) {
			s.CredentialReferences = project.Configured([]project.CredentialReference{{Key: "reference"}})
			s.Integrations = project.Configured([]project.Integration{{Key: "integration", CredentialRef: d}})
		}},
		{"transportReference", "logical", false, func(s *project.State, d project.Declaration[string]) {
			s.Providers = project.Configured([]project.Provider{{Key: "p", ID: "opaque"}})
			s.Integrations = project.Configured([]project.Integration{{Key: "i", ProviderRef: project.Configured("p"), Transport: project.Configured(project.Transport{ID: "t", Reference: d})}})
		}},
		{"sourceHint", "unconfigured", false, func(s *project.State, d project.Declaration[string]) {
			s.CredentialReferences = project.Configured([]project.CredentialReference{{Key: "ref", SourceHint: d}})
		}},
		{"contextText", "unconfigured", true, func(s *project.State, d project.Declaration[string]) {
			s.BusinessContext = project.Configured(project.BusinessContext{Text: d})
		}},
	}
	for _, field := range fields {
		t.Run(field.name, func(t *testing.T) {
			a, b := minimal(), minimal()
			field.set(&a, project.Declaration[string]{})
			field.set(&b, project.Configured(field.value))
			if valid(t, a).Equivalent(valid(t, b)) {
				t.Fatal("nested omission lost")
			}
			field.set(&b, project.Unconfigured[string]())
			invalid(t, b, "invalid_declaration")
			field.set(&b, project.Configured(""))
			_, issues := project.New(b)
			if (len(issues) == 0) != field.blankAllowed {
				t.Fatalf("empty string policy: %v", issues)
			}
		})
	}
}

func TestNestedCollectionsAndTransportPresence(t *testing.T) {
	for name, set := range map[string]func(*project.State, project.Declaration[[]string]){
		"capabilities": func(s *project.State, d project.Declaration[[]string]) {
			s.Integrations = project.Configured([]project.Integration{{Key: "requested", Capabilities: d}})
		},
		"documents": func(s *project.State, d project.Declaration[[]string]) {
			s.BusinessContext = project.Configured(project.BusinessContext{Documents: d})
		},
	} {
		t.Run(name, func(t *testing.T) { checkForms(t, set, []string{}, false) })
	}
	s := minimal()
	s.Providers = project.Configured([]project.Provider{{Key: "p", ID: "provider"}})
	s.Integrations = project.Configured([]project.Integration{{Key: "i", ProviderRef: project.Configured("p")}})
	absent := valid(t, s)
	s.Integrations = project.Configured([]project.Integration{{Key: "i", ProviderRef: project.Configured("p"), Transport: project.Configured(project.Transport{ID: "unconfigured"})}})
	if absent.Equivalent(valid(t, s)) {
		t.Fatal("transport presence lost")
	}
}

func TestConfiguredShapesAndReferenceIntegrity(t *testing.T) {
	cases := map[string]func(*project.State){
		"repository key":  func(s *project.State) { s.Repositories = project.Configured([]project.Repository{{}}) },
		"runtime id":      func(s *project.State) { s.Runtime = project.Configured(project.Runtime{}) },
		"provider id":     func(s *project.State) { s.Providers = project.Configured([]project.Provider{{Key: "p"}}) },
		"provider key":    func(s *project.State) { s.Providers = project.Configured([]project.Provider{{ID: "p"}}) },
		"integration key": func(s *project.State) { s.Integrations = project.Configured([]project.Integration{{}}) },
		"capability": func(s *project.State) {
			s.Integrations = project.Configured([]project.Integration{{Key: "i", Capabilities: project.Configured([]string{" "})}})
		},
		"transport id": func(s *project.State) {
			s.Integrations = project.Configured([]project.Integration{{Key: "i", Transport: project.Configured(project.Transport{})}})
		},
		"transport requires provider": func(s *project.State) {
			s.Integrations = project.Configured([]project.Integration{{Key: "i", Transport: project.Configured(project.Transport{ID: "t"})}})
		},
		"transport form": func(s *project.State) {
			s.Integrations = project.Configured([]project.Integration{{Key: "i", Transport: project.Unconfigured[project.Transport]()}})
		},
		"provider missing": func(s *project.State) {
			s.Integrations = project.Configured([]project.Integration{{Key: "i", ProviderRef: project.Configured("missing")}})
		},
		"credential missing": func(s *project.State) {
			s.Integrations = project.Configured([]project.Integration{{Key: "i", CredentialRef: project.Configured("missing")}})
		},
		"profile key": func(s *project.State) {
			s.ModelProfiles = project.Configured([]project.ModelProfile{{State: project.Unconfigured[string]()}})
		},
		"profile fields": func(s *project.State) { s.ModelProfiles = project.Configured([]project.ModelProfile{{Key: "p"}}) },
		"profile runtime": func(s *project.State) {
			s.ModelProfiles = project.Configured([]project.ModelProfile{{Key: "p", RuntimeRef: project.Configured("missing"), Model: project.Configured("opaque")}})
		},
		"profile blank model": func(s *project.State) {
			s.Runtime = project.Configured(project.Runtime{ID: "r"})
			s.ModelProfiles = project.Configured([]project.ModelProfile{{Key: "p", RuntimeRef: project.Configured("r"), Model: project.Configured(" ")}})
		},
		"profile forbidden state value": func(s *project.State) {
			s.ModelProfiles = project.Configured([]project.ModelProfile{{Key: "p", State: project.Configured("unconfigured")}})
		},
		"profile unconfigured plus model": func(s *project.State) {
			s.ModelProfiles = project.Configured([]project.ModelProfile{{Key: "p", State: project.Unconfigured[string](), Model: project.Configured("opaque")}})
		},
		"credential key": func(s *project.State) { s.CredentialReferences = project.Configured([]project.CredentialReference{{}}) },
	}
	for name, edit := range cases {
		t.Run(name, func(t *testing.T) {
			s := minimal()
			edit(&s)
			if _, issues := project.New(s); len(issues) == 0 {
				t.Fatal("invalid configured shape accepted")
			}
		})
	}
	// Selection, request, and binding are independent; no vendor/default discovery.
	s := minimal()
	s.Providers = project.Configured([]project.Provider{{Key: "p", ID: "unconfigured"}})
	s.Integrations = project.Configured([]project.Integration{{Key: "request-only"}})
	valid(t, s)
}

func TestDocumentReferencesAreLexicalAndTextIsUninterpreted(t *testing.T) {
	for _, reference := range []string{"", " ", "/etc/synthetic", "../outside", "context/../outside", "C:/synthetic", "context\\outside", "${SYNTHETIC}/file", "`synthetic`", "~/file"} {
		s := minimal()
		s.Policies = project.Configured([]string{reference})
		invalid(t, s, "invalid_document_reference")
	}
	s := minimal()
	s.BusinessContext = project.Configured(project.BusinessContext{Text: project.Configured("$(synthetic) ${SYNTHETIC} ../data"), Documents: project.Configured([]string{"context/not-on-disk.md"})})
	valid(t, s) // No filesystem lookup, expansion or evaluation.
}

func TestProjectOwnsSnapshotsAndIntentInputs(t *testing.T) {
	s := configured()
	p := valid(t, s)
	repos, _ := s.Repositories.Value()
	repos[0].Key = "mutated"
	integrations, _ := s.Integrations.Value()
	capabilities, _ := integrations[0].Capabilities.Value()
	capabilities[0] = "mutated"
	context, _ := s.BusinessContext.Value()
	docs, _ := context.Documents.Value()
	docs[0] = "mutated"
	policies, _ := s.Policies.Value()
	policies[0] = "mutated"
	if !p.Equivalent(valid(t, configured())) {
		t.Fatal("input aliases mutate Project")
	}
	snapshot := p.State()
	repos, _ = snapshot.Repositories.Value()
	repos[0].Key = "mutated"
	if !p.Equivalent(valid(t, configured())) {
		t.Fatal("snapshot alias mutates Project")
	}
	newRepos := []project.Repository{{Key: "replacement"}}
	proposed, issues := p.Propose(project.Intent{Repositories: project.Set(project.Configured(newRepos))})
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	newRepos[0].Key = "changed-after-proposal"
	got, _ := proposed.State().Repositories.Value()
	if got[0].Key != "replacement" {
		t.Fatal("intent alias mutates proposal")
	}
}

func TestCompleteIntentAndOrderedData(t *testing.T) {
	p := valid(t, configured())
	intent := project.Intent{
		Name: project.Set("Renamed"), Slug: project.Set("renamed"),
		Repositories:         project.Set(project.Configured([]project.Repository{})),
		Runtime:              project.Set(project.Unconfigured[project.Runtime]()),
		Providers:            project.Set(project.Unconfigured[[]project.Provider]()),
		Integrations:         project.Set(project.Unconfigured[[]project.Integration]()),
		ModelProfiles:        project.Set(project.Unconfigured[[]project.ModelProfile]()),
		BusinessContext:      project.Set(project.Configured(project.BusinessContext{})),
		CredentialReferences: project.Set(project.Unconfigured[[]project.CredentialReference]()),
		Policies:             project.Set(project.Configured([]string{})),
	}
	result, issues := p.Propose(intent)
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	want := minimal()
	want.Name, want.Slug = "Renamed", "renamed"
	want.Repositories = project.Configured([]project.Repository{})
	want.Runtime = project.Unconfigured[project.Runtime]()
	want.Providers = project.Unconfigured[[]project.Provider]()
	want.Integrations = project.Unconfigured[[]project.Integration]()
	want.ModelProfiles = project.Unconfigured[[]project.ModelProfile]()
	want.BusinessContext = project.Configured(project.BusinessContext{})
	want.CredentialReferences = project.Unconfigured[[]project.CredentialReference]()
	want.Policies = project.Configured([]string{})
	if !result.Equivalent(valid(t, want)) {
		t.Fatal("complete proposed state differs from intended result")
	}
	for _, setter := range []func(*project.State, []string){
		func(s *project.State, values []string) { s.Policies = project.Configured(values) },
		func(s *project.State, values []string) {
			s.BusinessContext = project.Configured(project.BusinessContext{Documents: project.Configured(values)})
		},
	} {
		a, b := minimal(), minimal()
		setter(&a, []string{"a.md", "b.md"})
		setter(&b, []string{"b.md", "a.md"})
		if valid(t, a).Equivalent(valid(t, b)) {
			t.Fatal("document order lost")
		}
	}
	var zero project.Project
	if zero.Equivalent(zero) {
		t.Fatal("invalid zero Projects equivalent")
	}
	if _, issues := zero.Propose(project.Intent{}); len(issues) == 0 {
		t.Fatal("invalid current accepted")
	}
}

func TestDeterministicSafeIssues(t *testing.T) {
	s := minimal()
	s.Name = ""
	s.Slug = "SYNTHETIC_REJECTED_VALUE"
	s.Integrations = project.Configured([]project.Integration{{Key: "i", ProviderRef: project.Configured("SYNTHETIC_REJECTED_VALUE")}})
	_, expected := project.New(s)
	for i := 0; i < 20; i++ {
		_, got := project.New(s)
		if !reflect.DeepEqual(got, expected) {
			t.Fatal("unstable issues")
		}
	}
	for _, issue := range expected {
		if strings.Contains(issue.Field+issue.Code, "SYNTHETIC_REJECTED_VALUE") {
			t.Fatal("raw rejected input leaked")
		}
	}
}

func TestAllKeyedCollectionsNormalizeWithoutChangingOpaqueData(t *testing.T) {
	a := configured()
	a.Providers = project.Configured([]project.Provider{{Key: "work", ID: "Opaque/Provider"}, {Key: "other", ID: "Opaque/Provider"}})
	a.CredentialReferences = project.Configured([]project.CredentialReference{{Key: "work-ref"}, {Key: "other-ref"}})
	integrations, _ := a.Integrations.Value()
	a.Integrations = project.Configured(append(integrations, project.Integration{Key: "other"}))
	p := valid(t, a)
	b := p.State()
	providers, _ := b.Providers.Value()
	providers[0], providers[1] = providers[1], providers[0]
	credentials, _ := b.CredentialReferences.Value()
	credentials[0], credentials[1] = credentials[1], credentials[0]
	integrations, _ = b.Integrations.Value()
	integrations[0], integrations[1] = integrations[1], integrations[0]
	profiles, _ := b.ModelProfiles.Value()
	profiles[0], profiles[1] = profiles[1], profiles[0]
	q := valid(t, b)
	if !p.Equivalent(q) || !q.Equivalent(p) || !q.Equivalent(valid(t, q.State())) {
		t.Fatal("keyed order affected equivalence")
	}
	providers[0].ID = "opaque/provider"
	if p.Equivalent(valid(t, b)) {
		t.Fatal("opaque provider case lost")
	}
	nilState, emptyState := minimal(), minimal()
	nilState.Providers = project.Configured[[]project.Provider](nil)
	emptyState.Providers = project.Configured([]project.Provider{})
	if !valid(t, nilState).Equivalent(valid(t, emptyState)) {
		t.Fatal("explicit empty collections differ by Go allocation")
	}
}
