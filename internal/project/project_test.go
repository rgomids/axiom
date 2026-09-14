package project_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/project"
)

const identity = "12345678-1234-4abc-8def-123456789abc"

func minimal() project.State {
	return project.State{SchemaVersion: 1, ID: identity, Slug: "sample-project", Name: "Sample Project"}
}

func configured() project.State {
	s := minimal()
	s.Repositories = project.Configured([]project.Repository{
		{Key: "api", Remote: project.Configured("https://EXAMPLE.com/Org/API.git")},
		{Key: "web", Remote: project.Configured("git@OTHER.example:Team/Web.git")},
		{Key: "local"},
	})
	s.Runtime = project.Configured(project.Runtime{ID: "opaque-runtime"})
	s.Providers = project.Configured([]project.Provider{{Key: "work", ID: "opaque-provider"}})
	s.CredentialReferences = project.Configured([]project.CredentialReference{{Key: "work-ref", SourceHint: project.Configured("environment")}})
	s.Integrations = project.Configured([]project.Integration{{
		Key: "issues", ProviderRef: project.Configured("work"),
		Capabilities:  project.Configured([]string{"read-work", "write-work"}),
		Transport:     project.Configured(project.Transport{ID: "opaque-transport", Reference: project.Configured("logical-binding")}),
		CredentialRef: project.Configured("work-ref"),
	}})
	s.ModelProfiles = project.Configured([]project.ModelProfile{
		{Key: "draft", RuntimeRef: project.Configured("opaque-runtime"), Model: project.Configured("Opaque/Model")},
		{Key: "later", State: project.Unconfigured[string]()},
	})
	s.BusinessContext = project.Configured(project.BusinessContext{Text: project.Configured("Untrusted text is data: ${SYNTHETIC_NAME}"), Documents: project.Configured([]string{"context/purpose.md"})})
	s.Policies = project.Configured([]string{"policies/review.md"})
	return s
}

func valid(t *testing.T, s project.State) project.Project {
	t.Helper()
	p, issues := project.New(s)
	if len(issues) != 0 {
		t.Fatalf("expected valid state: %v", issues)
	}
	return p
}

func invalid(t *testing.T, s project.State, code string) {
	t.Helper()
	_, issues := project.New(s)
	for _, issue := range issues {
		if issue.Code == code {
			return
		}
	}
	t.Fatalf("missing %s: %v", code, issues)
}

func TestMinimalAndConfiguredProject(t *testing.T) {
	valid(t, minimal())
	valid(t, configured())
	s := minimal()
	s.ID = "87654321-4321-4abc-bdef-123456789abc"
	valid(t, s) // Friendly names are not a uniqueness constraint.
	for _, edit := range []func(*project.State){
		func(s *project.State) { s.SchemaVersion = 0 },
		func(s *project.State) { s.SchemaVersion = 2 },
		func(s *project.State) { s.ID = "" },
		func(s *project.State) { s.Slug = "" },
		func(s *project.State) { s.Name = " \t\n" },
	} {
		s := minimal()
		edit(&s)
		if _, issues := project.New(s); len(issues) == 0 {
			t.Fatal("missing required invariant")
		}
	}
}

func TestUUIDVersionVariantAndCanonicalIdentity(t *testing.T) {
	for _, variant := range []string{"8", "9", "a", "b"} {
		s := minimal()
		s.ID = "12345678-1234-4abc-" + variant + "def-123456789abc"
		valid(t, s)
	}
	for _, id := range []string{"", "1234567812344abc8def123456789abc", "{12345678-1234-4abc-8def-123456789abc}", "12345678-1234-1abc-8def-123456789abc", "12345678-1234-4abc-7def-123456789abc", "12345678-1234-4abc-cdef-123456789abc", "00000000-0000-0000-0000-000000000000", "12345678-1234-4abc-8def-123456789abz"} {
		s := minimal()
		s.ID = id
		invalid(t, s, "invalid_id")
	}
	p := valid(t, minimal())
	updated, issues := p.Propose(project.Intent{Name: project.Set("Renamed"), Slug: project.Set("new-slug")})
	if len(issues) != 0 || updated.State().ID != identity {
		t.Fatalf("identity changed: %v", issues)
	}
	copy := p.State()
	copy.ID = "87654321-4321-4abc-bdef-123456789abc"
	if p.State().ID != identity {
		t.Fatal("snapshot mutated identity")
	}
}

func TestSlugGrammar(t *testing.T) {
	for _, slug := range []string{"a", "0", "a1", "a-b-2", "123"} {
		s := minimal()
		s.Slug = slug
		valid(t, s)
	}
	for _, slug := range []string{"", "A", "a_1", "a--b", "-a", "a-", "a/b", "a\\b", "..", ".", "á", " a", "a ", "a\n", "a\x00"} {
		s := minimal()
		s.Slug = slug
		invalid(t, s, "invalid_slug")
	}
}

func TestDuplicateDeclarations(t *testing.T) {
	for name, edit := range map[string]func(*project.State){
		"repository": func(s *project.State) {
			s.Repositories = project.Configured([]project.Repository{{Key: "same"}, {Key: "same"}})
		},
		"provider": func(s *project.State) {
			s.Providers = project.Configured([]project.Provider{{Key: "same", ID: "a"}, {Key: "same", ID: "b"}})
		},
		"integration": func(s *project.State) {
			s.Integrations = project.Configured([]project.Integration{{Key: "same"}, {Key: "same"}})
		},
		"profile": func(s *project.State) {
			s.ModelProfiles = project.Configured([]project.ModelProfile{{Key: "same", State: project.Unconfigured[string]()}, {Key: "same", State: project.Unconfigured[string]()}})
		},
		"credential": func(s *project.State) {
			s.CredentialReferences = project.Configured([]project.CredentialReference{{Key: "same"}, {Key: "same"}})
		},
	} {
		t.Run(name, func(t *testing.T) { s := minimal(); edit(&s); invalid(t, s, "duplicate_key") })
	}
	s := minimal()
	s.Repositories = project.Configured([]project.Repository{{Key: "a", Remote: project.Configured("https://EXAMPLE.com/Team/Repo.git")}, {Key: "b", Remote: project.Configured("https://example.com/Team/Repo.git")}})
	invalid(t, s, "duplicate_locator")
}

func TestPartialIntentRetainsCompleteStateAndValidatesRetainedReferences(t *testing.T) {
	p := valid(t, configured())
	before := p.State()
	proposed, issues := p.Propose(project.Intent{Name: project.Set("New name")})
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	expected := p.State()
	expected.Name = "New name"
	if !reflect.DeepEqual(proposed.State(), expected) {
		t.Fatal("name intent lost untouched values")
	}
	if !reflect.DeepEqual(before, p.State()) {
		t.Fatal("current state mutated")
	}
	for name, intent := range map[string]project.Intent{
		"provider removal":    {Providers: project.Set(project.Configured([]project.Provider{}))},
		"credential removal":  {CredentialReferences: project.Set(project.Declaration[[]project.CredentialReference]{})},
		"runtime replacement": {Runtime: project.Set(project.Configured(project.Runtime{ID: "other"}))},
	} {
		t.Run(name, func(t *testing.T) {
			_, issues := p.Propose(intent)
			if len(issues) == 0 {
				t.Fatal("retained dangling reference accepted")
			}
			if !reflect.DeepEqual(before, p.State()) {
				t.Fatal("failed intent changed current state")
			}
		})
	}
	_, issues = p.Propose(project.Intent{
		Providers:    project.Set(project.Declaration[[]project.Provider]{}),
		Integrations: project.Set(project.Configured([]project.Integration{})),
	})
	if len(issues) != 0 {
		t.Fatalf("explicitly resolved removal rejected: %v", issues)
	}
}

func TestEquivalencePreservesPresenceAndIgnoresKeyedOrder(t *testing.T) {
	p := valid(t, configured())
	s := p.State()
	r, _ := s.Repositories.Value()
	r[0], r[2] = r[2], r[0]
	s.Repositories = project.Configured(r)
	if !p.Equivalent(valid(t, s)) {
		t.Fatal("keyed collection order changed intent")
	}
	for _, form := range []project.Declaration[[]project.Provider]{{}, project.Unconfigured[[]project.Provider](), project.Configured([]project.Provider{})} {
		s := minimal()
		s.Providers = form
		p := valid(t, s)
		if !p.Equivalent(valid(t, s)) {
			t.Fatal("same form not equivalent")
		}
		for _, other := range []project.Declaration[[]project.Provider]{{}, project.Unconfigured[[]project.Provider](), project.Configured([]project.Provider{})} {
			s.Providers = other
			if p.Equivalent(valid(t, s)) != (form.Form() == other.Form()) {
				t.Fatal("declaration forms collapsed")
			}
		}
	}
	contexts := []project.Declaration[project.BusinessContext]{{}, project.Unconfigured[project.BusinessContext](), project.Configured(project.BusinessContext{}), project.Configured(project.BusinessContext{Text: project.Configured("")}), project.Configured(project.BusinessContext{Documents: project.Configured([]string{})})}
	for i, a := range contexts {
		for j, b := range contexts {
			x, y := minimal(), minimal()
			x.BusinessContext, y.BusinessContext = a, b
			if valid(t, x).Equivalent(valid(t, y)) != (i == j) {
				t.Fatal("nested presence collapsed")
			}
		}
	}
}

func TestLocatorNormalizationIsConservative(t *testing.T) {
	for _, tc := range []struct{ input, want string }{
		{"HTTPS://EXAMPLE.COM/Team/Repo.git", "https://example.com/Team/Repo.git"},
		{"git@EXAMPLE.COM:Team/Repo.git", "git@example.com:Team/Repo.git"},
		{"git@H:Repo", "git@h:Repo"},
		{"ssh://git@EXAMPLE.COM:22/Team/Repo.git", "ssh://git@example.com:22/Team/Repo.git"},
		{"https://EXAMPLE.COM:443/Team/%52epo.git/", "https://example.com:443/Team/%52epo.git/"},
		{"SSH://GitUser@EXAMPLE.COM:022/Team/Repo", "ssh://GitUser@example.com:022/Team/Repo"},
		{"https://EXAMPLE.COM/Repo?ref=Main%2fTree#ReadMe", "https://example.com/Repo?ref=Main%2fTree#ReadMe"},
		{"https://[2001:DB8::1]/Repo", "https://[2001:DB8::1]/Repo"},
		{"https://127.0.0.1/Repo", "https://127.0.0.1/Repo"},
	} {
		got, issues := project.NormalizeLocator(tc.input)
		if len(issues) != 0 || got != tc.want {
			t.Fatalf("normalization: %q %v", got, issues)
		}
		again, issues := project.NormalizeLocator(got)
		if len(issues) != 0 || again != got {
			t.Fatal("normalization is not idempotent")
		}
	}
	locators := []string{"https://example.com/Team/Repo", "https://example.com/Team/Repo.git", "https://example.com/Team/Repo/", "https://example.com/team/Repo", "https://example.com:443/Team/Repo", "https://example.com/Team/%52epo", "git@example.com:Team/Repo", "ssh://git@example.com/Team/Repo"}
	s := minimal()
	var repos []project.Repository
	for i, locator := range locators {
		repos = append(repos, project.Repository{Key: strings.Repeat("a", i+1), Remote: project.Configured(locator)})
	}
	s.Repositories = project.Configured(repos)
	valid(t, s)
	for _, locator := range []string{"", "./local", "/tmp/local", "https:///repo", "https://example.com/%GG", "https://user:synthetic@example.com/repo", "https://user@example.com/repo", "git@:repo", "git@example.com:", "https://example.com/a b", "git@bad_host:Repo", "git@one@two:Repo", "C:/local", "git@/host:Repo", "https://-host.example/Repo"} {
		if _, issues := project.NormalizeLocator(locator); len(issues) == 0 {
			t.Fatal("invalid locator accepted")
		}
	}
}
