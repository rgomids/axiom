package projectapp

import (
	"encoding/json"
	"sort"

	"github.com/rgomids/axiom/internal/project"
)

type policyCollection[T any] struct {
	Form   string `json:"form"`
	Values []T    `json:"values,omitempty"`
}

type PolicyView struct {
	Runtimes           policyCollection[policyRuntime]    `json:"runtimes"`
	ModelProfiles      policyCollection[policyProfile]    `json:"modelProfiles"`
	RuntimePreferences policyCollection[policyPreference] `json:"runtimePreferences"`
}

type policyMember struct{ Member, Role, Complexity string }

type PolicyChange struct {
	Collection string          `json:"collection"`
	Member     string          `json:"member,omitempty"`
	Role       string          `json:"role,omitempty"`
	Complexity string          `json:"complexity,omitempty"`
	Before     json.RawMessage `json:"before,omitempty"`
	After      json.RawMessage `json:"after,omitempty"`
}

func policyForm(form project.Form) string {
	switch form {
	case project.Absent:
		return "absent"
	case project.NotConfigured:
		return "unconfigured"
	default:
		return "configured"
	}
}

func ProjectPolicy(p project.Project) PolicyView {
	s := p.State()
	out := PolicyView{Runtimes: policyCollection[policyRuntime]{Form: policyForm(s.Runtimes.Form())}, ModelProfiles: policyCollection[policyProfile]{Form: policyForm(s.ModelProfiles.Form())}, RuntimePreferences: policyCollection[policyPreference]{Form: policyForm(s.RuntimePreferences.Form())}}
	rs, _ := s.Runtimes.Value()
	for _, r := range rs {
		out.Runtimes.Values = append(out.Runtimes.Values, policyRuntime{r.ID})
	}
	ps, _ := s.ModelProfiles.Value()
	for _, p := range ps {
		v := policyProfile{Key: p.Key}
		if p.State.Form() == project.NotConfigured {
			state := "unconfigured"
			v.State = &state
		} else {
			runtime, _ := p.RuntimeRef.Value()
			model, _ := p.Model.Value()
			v.RuntimeRef = &runtime
			v.Model = &model
		}
		out.ModelProfiles.Values = append(out.ModelProfiles.Values, v)
	}
	prefs, _ := s.RuntimePreferences.Value()
	for _, p := range prefs {
		out.RuntimePreferences.Values = append(out.RuntimePreferences.Values, policyPreference{p.Role, p.Complexity, p.ModelProfileRef})
	}
	return out
}

func PolicyDiff(before, after project.Project) []PolicyChange {
	a, b := ProjectPolicy(before), ProjectPolicy(after)
	changes := []PolicyChange{}
	changes = append(changes, policyCollectionDiff("runtimes", a.Runtimes, b.Runtimes, func(r policyRuntime) policyMember { return policyMember{Member: r.ID} })...)
	changes = append(changes, policyCollectionDiff("modelProfiles", a.ModelProfiles, b.ModelProfiles, func(p policyProfile) policyMember { return policyMember{Member: p.Key} })...)
	changes = append(changes, policyCollectionDiff("runtimePreferences", a.RuntimePreferences, b.RuntimePreferences, func(p policyPreference) policyMember { return policyMember{Role: p.Role, Complexity: p.Complexity} })...)
	return changes
}

func policyCollectionDiff[T any](name string, a, b policyCollection[T], identity func(T) policyMember) []PolicyChange {
	changes := []PolicyChange{}
	if a.Form != b.Form {
		before, _ := json.Marshal(a.Form)
		after, _ := json.Marshal(b.Form)
		changes = append(changes, PolicyChange{Collection: name, Before: before, After: after})
	}
	old, next := map[policyMember]string{}, map[policyMember]string{}
	for _, v := range a.Values {
		wire, _ := json.Marshal(v)
		old[identity(v)] = string(wire)
	}
	for _, v := range b.Values {
		wire, _ := json.Marshal(v)
		next[identity(v)] = string(wire)
	}
	keys := map[policyMember]bool{}
	for k := range old {
		keys[k] = true
	}
	for k := range next {
		keys[k] = true
	}
	members := []PolicyChange{}
	for k := range keys {
		if old[k] == next[k] {
			continue
		}
		members = append(members, PolicyChange{Collection: name, Member: k.Member, Role: k.Role, Complexity: k.Complexity, Before: json.RawMessage(old[k]), After: json.RawMessage(next[k])})
	}
	sort.Slice(members, func(i, j int) bool {
		a, b := members[i], members[j]
		if a.Member != b.Member {
			return a.Member < b.Member
		}
		if a.Role != b.Role {
			return a.Role < b.Role
		}
		return a.Complexity < b.Complexity
	})
	return append(changes, members...)
}
