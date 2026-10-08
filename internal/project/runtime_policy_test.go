package project_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/project"
)

func policyState() project.State {
	s := minimal()
	s.SchemaVersion = 2
	s.Runtimes = project.Configured([]project.Runtime{{ID: "codex"}, {ID: "claude"}})
	s.ModelProfiles = project.Configured([]project.ModelProfile{
		{Key: "worker", RuntimeRef: project.Configured("codex"), Model: project.Configured("Opaque/Model")},
		{Key: "careful", RuntimeRef: project.Configured("claude"), Model: project.Configured("Another Model")},
		{Key: "later", State: project.Unconfigured[string]()},
	})
	s.RuntimePreferences = project.Configured([]project.RuntimePreference{
		{Role: "review", Complexity: "high", ModelProfileRef: "careful"},
		{Role: "implementation", Complexity: "low", ModelProfileRef: "worker"},
		{Role: "implementation", Complexity: "high", ModelProfileRef: "careful"},
	})
	return s
}

func TestRuntimePolicyVersionGates(t *testing.T) {
	for _, d := range []project.Declaration[[]project.Runtime]{project.Unconfigured[[]project.Runtime](), project.Configured([]project.Runtime{})} {
		s := minimal()
		s.Runtimes = d
		invalid(t, s, "unsupported_field")
	}
	for _, d := range []project.Declaration[[]project.RuntimePreference]{project.Unconfigured[[]project.RuntimePreference](), project.Configured([]project.RuntimePreference{})} {
		s := minimal()
		s.RuntimePreferences = d
		invalid(t, s, "unsupported_field")
	}
	for _, d := range []project.Declaration[project.Runtime]{project.Unconfigured[project.Runtime](), project.Configured(project.Runtime{ID: "codex"})} {
		s := policyState()
		s.Runtime = d
		invalid(t, s, "unsupported_field")
	}
}

func TestRuntimePolicyValidation(t *testing.T) {
	for name, edit := range map[string]func(*project.State){
		"blank runtime": func(s *project.State) { s.Runtimes = project.Configured([]project.Runtime{{ID: " "}}) },
		"duplicate runtime": func(s *project.State) {
			s.Runtimes = project.Configured([]project.Runtime{{ID: "codex"}, {ID: "codex"}})
		},
		"missing runtime": func(s *project.State) { s.Runtimes = project.Unconfigured[[]project.Runtime]() },
		"duplicate preference": func(s *project.State) {
			s.RuntimePreferences = project.Configured([]project.RuntimePreference{{Role: "review", Complexity: "high", ModelProfileRef: "careful"}, {Role: "review", Complexity: "high", ModelProfileRef: "worker"}})
		},
		"missing profile": func(s *project.State) {
			s.RuntimePreferences = project.Configured([]project.RuntimePreference{{Role: "review", Complexity: "high", ModelProfileRef: "missing"}})
		},
		"unconfigured profile": func(s *project.State) {
			s.RuntimePreferences = project.Configured([]project.RuntimePreference{{Role: "review", Complexity: "high", ModelProfileRef: "later"}})
		},
		"blank ref": func(s *project.State) {
			s.RuntimePreferences = project.Configured([]project.RuntimePreference{{Role: "review", Complexity: "high"}})
		},
		"uppercase role": func(s *project.State) {
			s.RuntimePreferences = project.Configured([]project.RuntimePreference{{Role: "Review", Complexity: "high", ModelProfileRef: "careful"}})
		},
		"wildcard complexity": func(s *project.State) {
			s.RuntimePreferences = project.Configured([]project.RuntimePreference{{Role: "review", Complexity: "*", ModelProfileRef: "careful"}})
		},
		"oversized token": func(s *project.State) {
			s.RuntimePreferences = project.Configured([]project.RuntimePreference{{Role: strings.Repeat("a", 257), Complexity: "high", ModelProfileRef: "careful"}})
		},
	} {
		t.Run(name, func(t *testing.T) {
			s := policyState()
			edit(&s)
			if _, issues := project.New(s); len(issues) == 0 {
				t.Fatal("invalid policy accepted")
			}
		})
	}
}

func TestRuntimePolicyBounds(t *testing.T) {
	for name, limit := range map[string]int{"runtimes": 8, "profiles": 32, "preferences": 32} {
		t.Run(name, func(t *testing.T) {
			for _, size := range []int{limit, limit + 1} {
				s := policyState()
				switch name {
				case "runtimes":
					runtimes, _ := s.Runtimes.Value()
					for i := len(runtimes); i < size; i++ {
						runtimes = append(runtimes, project.Runtime{ID: fmt.Sprintf("runtime-%d", i)})
					}
					s.Runtimes = project.Configured(runtimes)
				case "profiles":
					profiles, _ := s.ModelProfiles.Value()
					for i := len(profiles); i < size; i++ {
						profiles = append(profiles, project.ModelProfile{Key: fmt.Sprintf("profile-%d", i), State: project.Unconfigured[string]()})
					}
					s.ModelProfiles = project.Configured(profiles)
				case "preferences":
					preferences := []project.RuntimePreference{}
					for i := 0; i < size; i++ {
						preferences = append(preferences, project.RuntimePreference{Role: fmt.Sprintf("role-%d", i), Complexity: "high", ModelProfileRef: "careful"})
					}
					s.RuntimePreferences = project.Configured(preferences)
				}
				if size == limit {
					valid(t, s)
				} else {
					invalid(t, s, "collection_limit")
				}
			}
		})
	}
}

func TestRuntimePolicyNormalizationAndOwnership(t *testing.T) {
	s := policyState()
	p := valid(t, s)
	runtimes, _ := p.State().Runtimes.Value()
	preferences, _ := p.State().RuntimePreferences.Value()
	if runtimes[0].ID != "claude" || preferences[0].Role != "implementation" || preferences[0].Complexity != "high" {
		t.Fatal("policy order not canonical")
	}
	reversed := p.State()
	rs, _ := reversed.Runtimes.Value()
	rs[0], rs[1] = rs[1], rs[0]
	ps, _ := reversed.RuntimePreferences.Value()
	ps[0], ps[2] = ps[2], ps[0]
	if !p.Equivalent(valid(t, reversed)) {
		t.Fatal("declaration order changed intent")
	}
	originals, _ := s.Runtimes.Value()
	originals[0].ID = "changed"
	originalPrefs, _ := s.RuntimePreferences.Value()
	originalPrefs[0].ModelProfileRef = "changed"
	runtimes[0].ID = "changed"
	preferences[0].Role = "changed"
	if !p.Equivalent(valid(t, policyState())) {
		t.Fatal("policy snapshot has mutable aliases")
	}
	before := p.State()
	if _, issues := p.Propose(project.Intent{Runtimes: project.Set(project.Configured([]project.Runtime{}))}); len(issues) == 0 {
		t.Fatal("retained dangling runtime reference accepted")
	}
	if _, issues := p.Propose(project.Intent{ModelProfiles: project.Set(project.Configured([]project.ModelProfile{}))}); len(issues) == 0 {
		t.Fatal("retained dangling profile reference accepted")
	}
	if !reflect.DeepEqual(before, p.State()) {
		t.Fatal("failed intent mutated project")
	}
	refs := []project.RuntimePreference{{Role: "review", Complexity: "high", ModelProfileRef: "worker"}}
	proposed, issues := p.Propose(project.Intent{RuntimePreferences: project.Set(project.Configured(refs))})
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	refs[0].ModelProfileRef = "changed"
	got, _ := proposed.State().RuntimePreferences.Value()
	if got[0].ModelProfileRef != "worker" {
		t.Fatal("intent alias mutated project")
	}
}

func TestRuntimePolicyDeclarationForms(t *testing.T) {
	forms := []project.Form{project.Absent, project.NotConfigured, project.Present}
	for _, a := range forms {
		for _, b := range forms {
			s := minimal()
			s.SchemaVersion = 2
			set := func(form project.Form) {
				switch form {
				case project.Absent:
					s.Runtimes = project.Declaration[[]project.Runtime]{}
					s.RuntimePreferences = project.Declaration[[]project.RuntimePreference]{}
					s.ModelProfiles = project.Declaration[[]project.ModelProfile]{}
				case project.NotConfigured:
					s.Runtimes = project.Unconfigured[[]project.Runtime]()
					s.RuntimePreferences = project.Unconfigured[[]project.RuntimePreference]()
					s.ModelProfiles = project.Unconfigured[[]project.ModelProfile]()
				case project.Present:
					s.Runtimes = project.Configured[[]project.Runtime](nil)
					s.RuntimePreferences = project.Configured[[]project.RuntimePreference](nil)
					s.ModelProfiles = project.Configured[[]project.ModelProfile](nil)
				}
			}
			set(a)
			p := valid(t, s)
			set(b)
			q := valid(t, s)
			if p.Equivalent(q) != (a == b) {
				t.Fatal("policy declaration state collapsed")
			}
			unchanged, issues := p.Propose(project.Intent{})
			if len(issues) != 0 || !unchanged.Equivalent(p) {
				t.Fatal("omission changed policy")
			}
		}
	}
}
