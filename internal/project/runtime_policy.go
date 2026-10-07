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
	if s.SchemaVersion != 2 {
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
	if s.SchemaVersion == 2 && s.Runtimes.form == Present {
		for _, runtime := range s.Runtimes.value {
			if runtime.ID == id {
				return true
			}
		}
	}
	return false
}
