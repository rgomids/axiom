package projectapp

import (
	"encoding/json"
	"sort"

	"github.com/rgomids/axiom/internal/portableconfig"
	"github.com/rgomids/axiom/internal/project"
	"github.com/rgomids/axiom/internal/strictjson"
)

const MaxPolicyDraftBytes = 256 * 1024

type PolicyDraft struct {
	runtimes    *policyGroup[policyRuntime, string]
	profiles    *policyGroup[policyProfile, string]
	preferences *policyGroup[policyPreference, preferenceKey]
	valid       bool
}

type policyGroup[T any, K comparable] struct {
	Mode    string `json:"mode"`
	Values  *[]T   `json:"values,omitempty"`
	Add     *[]T   `json:"add,omitempty"`
	Replace *[]T   `json:"replace,omitempty"`
	Remove  *[]K   `json:"remove,omitempty"`
}
type policyRuntime struct {
	ID string `json:"id"`
}
type policyProfile struct {
	Key        string  `json:"key"`
	State      *string `json:"state,omitempty"`
	RuntimeRef *string `json:"runtimeRef,omitempty"`
	Model      *string `json:"model,omitempty"`
}
type preferenceKey struct {
	Role       string `json:"role"`
	Complexity string `json:"complexity"`
}
type policyPreference struct {
	Role            string `json:"role"`
	Complexity      string `json:"complexity"`
	ModelProfileRef string `json:"modelProfileRef"`
}

func DecodePolicyDraft(wire []byte) (PolicyDraft, EditFailure) {
	var input struct {
		Runtimes    *policyGroup[policyRuntime, string]           `json:"runtimes,omitempty"`
		Profiles    *policyGroup[policyProfile, string]           `json:"modelProfiles,omitempty"`
		Preferences *policyGroup[policyPreference, preferenceKey] `json:"runtimePreferences,omitempty"`
	}
	if strictjson.Decode(wire, &input, MaxPolicyDraftBytes) != nil {
		return PolicyDraft{}, EditInvalidIntent
	}
	var tree any
	if json.Unmarshal(wire, &tree) != nil {
		return PolicyDraft{}, EditInvalidIntent
	}
	nodes := 0
	if !policyTreeBounded(tree, 1, &nodes) || !validPolicyGroup(input.Runtimes, 16) || !validPolicyGroup(input.Profiles, 64) || !validPolicyGroup(input.Preferences, 64) {
		return PolicyDraft{}, EditInvalidIntent
	}
	return PolicyDraft{input.Runtimes, input.Profiles, input.Preferences, true}, EditOK
}

func policyTreeBounded(value any, depth int, nodes *int) bool {
	*nodes++
	if depth > 32 || *nodes > 16384 {
		return false
	}
	switch v := value.(type) {
	case map[string]any:
		for _, child := range v {
			*nodes++
			if depth+1 > 32 || *nodes > 16384 || !policyTreeBounded(child, depth+1, nodes) {
				return false
			}
		}
	case []any:
		for _, child := range v {
			if !policyTreeBounded(child, depth+1, nodes) {
				return false
			}
		}
	}
	return true
}

func policyList[T any](list *[]T) []T {
	if list == nil {
		return nil
	}
	return *list
}

func validPolicyGroup[T any, K comparable](g *policyGroup[T, K], limit int) bool {
	if g == nil {
		return true
	}
	members := g.Add != nil || g.Replace != nil || g.Remove != nil
	switch g.Mode {
	case "absent", "unconfigured":
		return g.Values == nil && !members
	case "replace":
		return g.Values != nil && !members && len(*g.Values) <= limit/2
	case "members":
		return g.Values == nil && len(policyList(g.Add))+len(policyList(g.Replace))+len(policyList(g.Remove)) <= limit
	default:
		return false
	}
}

func mergePolicyGroup[T any, W any, K comparable](current project.Declaration[[]T], g *policyGroup[W, K], key func(T) K, convert func(W) (T, bool), less func(T, T) bool) (project.Declaration[[]T], bool) {
	if g == nil {
		return current, true
	}
	switch g.Mode {
	case "absent":
		return project.Declaration[[]T]{}, true
	case "unconfigured":
		return project.Unconfigured[[]T](), true
	}
	values, _ := current.Value()
	if g.Mode == "replace" {
		values = nil
	}
	byKey := make(map[K]T, len(values))
	for _, v := range values {
		byKey[key(v)] = v
	}
	seen := map[K]bool{}
	apply := func(items []W, replace bool) bool {
		for _, wire := range items {
			v, valid := convert(wire)
			if !valid {
				return false
			}
			k := key(v)
			_, exists := byKey[k]
			if seen[k] || exists != replace {
				return false
			}
			seen[k] = true
			byKey[k] = v
		}
		return true
	}
	if g.Mode == "replace" {
		if !apply(policyList(g.Values), false) {
			return current, false
		}
	} else {
		if !apply(policyList(g.Add), false) || !apply(policyList(g.Replace), true) {
			return current, false
		}
		for _, k := range policyList(g.Remove) {
			if _, exists := byKey[k]; !exists || seen[k] {
				return current, false
			}
			seen[k] = true
			delete(byKey, k)
		}
	}
	result := make([]T, 0, len(byKey))
	for _, v := range byKey {
		result = append(result, v)
	}
	sort.Slice(result, func(i, j int) bool { return less(result[i], result[j]) })
	return project.Configured(result), true
}

func (d PolicyDraft) Propose(current project.Project) (project.Project, EditFailure) {
	s := current.State()
	if !d.valid || (s.SchemaVersion != 2 && s.SchemaVersion != 3) {
		return project.Project{}, EditInvalidIntent
	}
	runtimes, ok := mergePolicyGroup(s.Runtimes, d.runtimes, func(r project.Runtime) string { return r.ID }, func(r policyRuntime) (project.Runtime, bool) {
		return project.Runtime{ID: r.ID}, portableconfig.SafeValue(r.ID, "identifier")
	}, func(a, b project.Runtime) bool { return a.ID < b.ID })
	if !ok {
		return project.Project{}, EditInvalidIntent
	}
	profiles, ok := mergePolicyGroup(s.ModelProfiles, d.profiles, func(p project.ModelProfile) string { return p.Key }, func(p policyProfile) (project.ModelProfile, bool) {
		result := project.ModelProfile{Key: p.Key}
		if !portableconfig.SafeValue(p.Key, "identifier") {
			return result, false
		}
		if p.State != nil {
			if *p.State != "unconfigured" || p.RuntimeRef != nil || p.Model != nil {
				return result, false
			}
			result.State = project.Unconfigured[string]()
			return result, true
		}
		if p.RuntimeRef == nil || p.Model == nil || !portableconfig.SafeValue(*p.RuntimeRef, "logical") || !portableconfig.SafeValue(*p.Model, "identifier") {
			return result, false
		}
		result.RuntimeRef, result.Model = project.Configured(*p.RuntimeRef), project.Configured(*p.Model)
		return result, true
	}, func(a, b project.ModelProfile) bool { return a.Key < b.Key })
	if !ok {
		return project.Project{}, EditInvalidIntent
	}
	preferences, ok := mergePolicyGroup(s.RuntimePreferences, d.preferences, func(p project.RuntimePreference) preferenceKey { return preferenceKey{p.Role, p.Complexity} }, func(p policyPreference) (project.RuntimePreference, bool) {
		return project.RuntimePreference{Role: p.Role, Complexity: p.Complexity, ModelProfileRef: p.ModelProfileRef}, portableconfig.SafeValue(p.Role, "token") && portableconfig.SafeValue(p.Complexity, "token") && portableconfig.SafeValue(p.ModelProfileRef, "logical")
	}, func(a, b project.RuntimePreference) bool {
		if a.Role != b.Role {
			return a.Role < b.Role
		}
		return a.Complexity < b.Complexity
	})
	if !ok {
		return project.Project{}, EditInvalidIntent
	}
	candidate, issues := current.Propose(project.Intent{Runtimes: project.Set(runtimes), ModelProfiles: project.Set(profiles), RuntimePreferences: project.Set(preferences)})
	if len(issues) != 0 {
		return project.Project{}, EditCandidateInvalid
	}
	return candidate, EditOK
}
