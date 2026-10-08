package project_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/project"
)

func contextState() project.State {
	s := policyState()
	s.SchemaVersion = 3
	s.Repositories = project.Configured([]project.Repository{{Key: "core"}, {Key: "web"}})
	s.TechnologyContext = project.Configured([]project.TechnologyFact{{Key: "language.go", Value: "go"}, {Key: "infrastructure.terraform", Value: "terraform"}})
	s.DocumentationSources = project.Configured([]project.DocumentationSource{
		{Key: "product-notes", Kind: project.LocalFileSource},
		{Key: "architecture", Kind: project.RepositorySource, RepositoryRef: project.Configured("core"), Path: project.Configured("docs/architecture")},
	})
	s.BusinessContext = project.Configured(project.BusinessContext{
		Text:       project.Configured("Bounded context."),
		SourceRefs: project.Configured([]string{"product-notes", "architecture"}),
		Glossary: project.Configured([]project.GlossaryEntry{
			{Key: "work-item", Term: "Work Item", Definition: "A bounded unit.\nTracked by Axiom."},
			{Key: "execution", Term: "Execution", Definition: "One run."},
		}),
	})
	return s
}

func TestSchemaV3ContextIsValidAndCanonical(t *testing.T) {
	p := valid(t, contextState())
	s := p.State()
	if s.TechnologyContext.Form() != project.Present || fmt.Sprint(s.TechnologyContext) == "" {
		t.Fatal("technology context lost")
	}
	facts, _ := s.TechnologyContext.Value()
	sources, _ := s.DocumentationSources.Value()
	context, _ := s.BusinessContext.Value()
	refs, _ := context.SourceRefs.Value()
	glossary, _ := context.Glossary.Value()
	if facts[0].Key != "infrastructure.terraform" || sources[0].Key != "architecture" || refs[0] != "architecture" || glossary[0].Key != "execution" {
		t.Fatalf("canonical set ordering not applied: %+v %+v %+v %+v", facts, sources, refs, glossary)
	}
	reversed := contextState()
	rf, _ := reversed.TechnologyContext.Value()
	rf[0], rf[1] = rf[1], rf[0]
	if !p.Equivalent(valid(t, reversed)) {
		t.Fatal("set order changed equivalence")
	}
}

func TestSchemaV3RetainsV2RuntimeSemantics(t *testing.T) {
	v2, v3 := policyState(), contextState()
	if reflect.DeepEqual(issues(v2), nil) != reflect.DeepEqual(issues(v3), nil) {
		t.Fatal("v3 runtime validity diverged")
	}
	for name, edit := range map[string]func(*project.State){
		"singular runtime": func(s *project.State) { s.Runtime = project.Configured(project.Runtime{ID: "codex"}) },
		"dangling profile": func(s *project.State) {
			s.RuntimePreferences = project.Configured([]project.RuntimePreference{{Role: "r", Complexity: "c", ModelProfileRef: "missing"}})
		},
		"runtime limit":      func(s *project.State) { s.Runtimes = project.Configured(make([]project.Runtime, 9)) },
		"undeclared runtime": func(s *project.State) { s.Runtimes = project.Configured([]project.Runtime{{ID: "other"}}) },
	} {
		for _, version := range []int{2, 3} {
			s := contextState()
			if version == 2 {
				s = policyState()
			}
			edit(&s)
			if _, got := project.New(s); len(got) == 0 {
				t.Fatalf("%s v%d accepted", name, version)
			}
		}
	}
}

func issues(s project.State) []project.Issue { _, got := project.New(s); return got }

func TestSchemaV3FieldsAreVersionGated(t *testing.T) {
	for _, version := range []int{1, 2} {
		for name, edit := range map[string]func(*project.State){
			"technology":    func(s *project.State) { s.TechnologyContext = project.Configured([]project.TechnologyFact{}) },
			"documentation": func(s *project.State) { s.DocumentationSources = project.Unconfigured[[]project.DocumentationSource]() },
			"sourceRefs": func(s *project.State) {
				s.BusinessContext = project.Configured(project.BusinessContext{SourceRefs: project.Configured([]string{})})
			},
			"glossary": func(s *project.State) {
				s.BusinessContext = project.Configured(project.BusinessContext{Glossary: project.Configured([]project.GlossaryEntry{})})
			},
		} {
			s := minimal()
			if version == 2 {
				s = policyState()
			}
			edit(&s)
			if !hasCode(issues(s), "unsupported_field") {
				t.Fatalf("v%d accepted %s", version, name)
			}
		}
	}
}

func TestSchemaV3PresenceForms(t *testing.T) {
	for _, edit := range []func(*project.State){
		func(s *project.State) { s.TechnologyContext = project.Declaration[[]project.TechnologyFact]{} },
		func(s *project.State) { s.TechnologyContext = project.Unconfigured[[]project.TechnologyFact]() },
		func(s *project.State) { s.TechnologyContext = project.Configured([]project.TechnologyFact{}) },
		func(s *project.State) { s.DocumentationSources = project.Unconfigured[[]project.DocumentationSource]() },
		func(s *project.State) {
			s.BusinessContext = project.Configured(project.BusinessContext{Glossary: project.Configured([]project.GlossaryEntry{})})
		},
	} {
		s := contextState()
		edit(&s)
		if s.DocumentationSources.Form() != project.Present {
			s.BusinessContext = project.Declaration[project.BusinessContext]{}
		}
		got := valid(t, s).State()
		if got.TechnologyContext.Form() != s.TechnologyContext.Form() || got.DocumentationSources.Form() != s.DocumentationSources.Form() {
			t.Fatal("presence form not preserved")
		}
	}
}

func TestSchemaV3RejectsInvalidContext(t *testing.T) {
	for name, tc := range map[string]struct {
		edit func(*project.State)
		code string
	}{
		"duplicate fact": {func(s *project.State) {
			s.TechnologyContext = project.Configured([]project.TechnologyFact{{"a", "x"}, {"a", "y"}})
		}, "duplicate_key"},
		"uppercase key": {func(s *project.State) {
			s.TechnologyContext = project.Configured([]project.TechnologyFact{{"Language", "x"}})
		}, "invalid_key"},
		"padded value": {func(s *project.State) {
			s.TechnologyContext = project.Configured([]project.TechnologyFact{{"a", " x"}})
		}, "invalid_value"},
		"multiline value": {func(s *project.State) {
			s.TechnologyContext = project.Configured([]project.TechnologyFact{{"a", "x\ny"}})
		}, "invalid_value"},
		"long value": {func(s *project.State) {
			s.TechnologyContext = project.Configured([]project.TechnologyFact{{"a", strings.Repeat("x", 257)}})
		}, "invalid_value"},
		"fact limit": {func(s *project.State) { s.TechnologyContext = project.Configured(facts(65)) }, "collection_limit"},
		"unknown kind": {func(s *project.State) {
			s.DocumentationSources = sources(project.DocumentationSource{Key: "d", Kind: "notion"})
		}, "unsupported_source_kind"},
		"dangling repository": {func(s *project.State) { s.DocumentationSources = sources(repoSource("missing", "docs")) }, "dangling_reference"},
		"missing path": {func(s *project.State) {
			s.DocumentationSources = sources(project.DocumentationSource{Key: "d", Kind: project.RepositorySource, RepositoryRef: project.Configured("core")})
		}, "invalid_source"},
		"local with path": {func(s *project.State) {
			s.DocumentationSources = sources(project.DocumentationSource{Key: "d", Kind: project.LocalFileSource, Path: project.Configured("x")})
		}, "invalid_source"},
		"duplicate source": {func(s *project.State) {
			s.DocumentationSources = sources(repoSource("core", "a"), repoSource("core", "b"))
		}, "duplicate_key"},
		"dangling sourceRef": {func(s *project.State) {
			s.BusinessContext = project.Configured(project.BusinessContext{SourceRefs: project.Configured([]string{"nope"})})
		}, "dangling_reference"},
		"duplicate sourceRef": {func(s *project.State) {
			s.BusinessContext = project.Configured(project.BusinessContext{SourceRefs: project.Configured([]string{"architecture", "architecture"})})
		}, "duplicate_key"},
		"unconfigured glossary": {func(s *project.State) {
			s.BusinessContext = project.Configured(project.BusinessContext{Glossary: project.Unconfigured[[]project.GlossaryEntry]()})
		}, "invalid_declaration"},
		"duplicate glossary": {func(s *project.State) {
			s.BusinessContext = project.Configured(project.BusinessContext{Glossary: project.Configured([]project.GlossaryEntry{{"k", "T", "D"}, {"k", "U", "E"}})})
		}, "duplicate_key"},
		"long definition": {func(s *project.State) {
			s.BusinessContext = project.Configured(project.BusinessContext{Glossary: project.Configured([]project.GlossaryEntry{{"k", "T", strings.Repeat("d", 2049)}})})
		}, "invalid_value"},
		"control in term": {func(s *project.State) {
			s.BusinessContext = project.Configured(project.BusinessContext{Glossary: project.Configured([]project.GlossaryEntry{{"k", "T\x00", "D"}})})
		}, "invalid_value"},
	} {
		s := contextState()
		tc.edit(&s)
		if !hasCode(issues(s), tc.code) {
			t.Fatalf("%s: want %s, got %v", name, tc.code, issues(s))
		}
	}
	for _, path := range []string{"/abs/docs", "../escape", "docs/../x", "docs//x", "./docs", "docs/", "~/docs", "C:/docs", `docs\x`, "docs/$HOME", "docs/a\x00", strings.Repeat("a", 1025)} {
		s := contextState()
		s.DocumentationSources = sources(repoSource("core", path))
		s.BusinessContext = project.Declaration[project.BusinessContext]{}
		if !hasCode(issues(s), "invalid_document_reference") {
			t.Fatalf("path %q accepted", path)
		}
	}
}

func TestSchemaV3ProposeReplacesContext(t *testing.T) {
	p := valid(t, contextState())
	next, got := p.Propose(project.Intent{TechnologyContext: project.Set(project.Configured([]project.TechnologyFact{{Key: "language.rust", Value: "rust"}}))})
	if len(got) != 0 {
		t.Fatal(got)
	}
	facts, _ := next.State().TechnologyContext.Value()
	if len(facts) != 1 || facts[0].Key != "language.rust" || next.State().SchemaVersion != 3 {
		t.Fatalf("unexpected proposal %+v", facts)
	}
	if facts, _ := p.State().TechnologyContext.Value(); len(facts) != 2 {
		t.Fatal("receiver mutated")
	}
}

func facts(n int) []project.TechnologyFact {
	out := make([]project.TechnologyFact, n)
	for i := range out {
		out[i] = project.TechnologyFact{Key: fmt.Sprintf("k%d", i), Value: "v"}
	}
	return out
}
func sources(items ...project.DocumentationSource) project.Declaration[[]project.DocumentationSource] {
	return project.Configured(items)
}
func repoSource(ref, path string) project.DocumentationSource {
	return project.DocumentationSource{Key: "d" + fmt.Sprint(len(path)), Kind: project.RepositorySource, RepositoryRef: project.Configured(ref), Path: project.Configured(path)}
}
func hasCode(issues []project.Issue, code string) bool {
	for _, issue := range issues {
		if issue.Code == code {
			return true
		}
	}
	return false
}
