package project

import (
	"fmt"
	"regexp"
)

var preferenceTokenPattern = regexp.MustCompile(`^[a-z0-9._-]{1,256}$`)

func (v *validation) runtimePolicy(s State) {
	if s.SchemaVersion == 1 {
		if s.Runtimes.form != Absent {
			v.add("runtimes", "unsupported_field")
		}
		if s.RuntimePreferences.form != Absent {
			v.add("runtimePreferences", "unsupported_field")
		}
		return
	}
	if !MultiRuntimeSchema(s.SchemaVersion) {
		return
	}
	if s.Runtime.form != Absent {
		v.add("runtime", "unsupported_field")
	}
	for _, collection := range []struct {
		field       string
		size, limit int
	}{
		{"runtimes", len(s.Runtimes.value), 8},
		{"modelProfiles", len(s.ModelProfiles.value), 32},
		{"runtimePreferences", len(s.RuntimePreferences.value), 32},
	} {
		if collection.size > collection.limit {
			v.add(collection.field, "collection_limit")
		}
	}
	ids := map[string]bool{}
	for i, runtime := range s.Runtimes.value {
		v.key(fmt.Sprintf("runtimes[%d].id", i), runtime.ID, ids)
	}
	configuredProfiles := map[string]bool{}
	for _, profile := range s.ModelProfiles.value {
		if profile.State.form == Absent && profile.RuntimeRef.form == Present && profile.Model.form == Present {
			configuredProfiles[profile.Key] = true
		}
	}
	pairs := map[[2]string]bool{}
	for i, preference := range s.RuntimePreferences.value {
		field := fmt.Sprintf("runtimePreferences[%d]", i)
		if !preferenceTokenPattern.MatchString(preference.Role) {
			v.add(field+".role", "invalid_token")
		}
		if !preferenceTokenPattern.MatchString(preference.Complexity) {
			v.add(field+".complexity", "invalid_token")
		}
		v.required(field+".modelProfileRef", preference.ModelProfileRef)
		if !configuredProfiles[preference.ModelProfileRef] {
			v.add(field+".modelProfileRef", "dangling_reference")
		}
		pair := [2]string{preference.Role, preference.Complexity}
		if pairs[pair] {
			v.add(field, "duplicate_key")
		}
		pairs[pair] = true
	}
}

func declaredRuntime(s State, id string) bool {
	if s.SchemaVersion == 1 {
		return s.Runtime.form == Present && s.Runtime.value.ID == id
	}
	if MultiRuntimeSchema(s.SchemaVersion) && s.Runtimes.form == Present {
		for _, runtime := range s.Runtimes.value {
			if runtime.ID == id {
				return true
			}
		}
	}
	return false
}

// MultiRuntimeSchema reports versions carrying the #140 Runtime policy with
// exactly v2 semantics. Schema v3 adds context fields and changes none of it.
func MultiRuntimeSchema(version int) bool { return version == 2 || version == 3 || version == 4 }

// AllowedRuntimes returns the portable Runtime allowlist for any supported
// schema: v1's singular Runtime or the v2/v3 Runtime collection.
func AllowedRuntimes(s State) []string {
	allowed := []string{}
	if s.SchemaVersion == 1 {
		if runtime, ok := s.Runtime.Value(); ok {
			allowed = append(allowed, runtime.ID)
		}
	} else if MultiRuntimeSchema(s.SchemaVersion) {
		runtimes, _ := s.Runtimes.Value()
		for _, runtime := range runtimes {
			allowed = append(allowed, runtime.ID)
		}
	}
	return allowed
}

// RuntimePolicyDeclared is the portable half of #140 execution policy: an
// allowed Runtime, at least one Model Profile declaration and no unresolved
// preference intent. It observes nothing machine-local and selects nothing.
func RuntimePolicyDeclared(s State) bool {
	profiles, ok := s.ModelProfiles.Value()
	return s.RuntimePreferences.Form() != NotConfigured && len(AllowedRuntimes(s)) != 0 && ok && len(profiles) != 0
}
