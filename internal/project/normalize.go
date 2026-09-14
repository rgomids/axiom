package project

import "sort"

func cloneList[T any](d Declaration[[]T]) Declaration[[]T] {
	if d.form != Present {
		return d
	}
	items := make([]T, len(d.value))
	copy(items, d.value)
	return Configured(items)
}

func cloneState(s State) State {
	s.Repositories = cloneList(s.Repositories)
	s.Providers = cloneList(s.Providers)
	s.Integrations = cloneList(s.Integrations)
	for i := range s.Integrations.value {
		s.Integrations.value[i].Capabilities = cloneList(s.Integrations.value[i].Capabilities)
	}
	s.ModelProfiles = cloneList(s.ModelProfiles)
	s.CredentialReferences = cloneList(s.CredentialReferences)
	s.Policies = cloneList(s.Policies)
	s.BusinessContext.value.Documents = cloneList(s.BusinessContext.value.Documents)
	return s
}

func normalize(s *State) {
	for i := range s.Repositories.value {
		normalizeRepository(&s.Repositories.value[i])
	}
	sort.Slice(s.Repositories.value, func(i, j int) bool { return s.Repositories.value[i].Key < s.Repositories.value[j].Key })
	sort.Slice(s.Providers.value, func(i, j int) bool { return s.Providers.value[i].Key < s.Providers.value[j].Key })
	sort.Slice(s.Integrations.value, func(i, j int) bool { return s.Integrations.value[i].Key < s.Integrations.value[j].Key })
	sort.Slice(s.ModelProfiles.value, func(i, j int) bool { return s.ModelProfiles.value[i].Key < s.ModelProfiles.value[j].Key })
	sort.Slice(s.CredentialReferences.value, func(i, j int) bool { return s.CredentialReferences.value[i].Key < s.CredentialReferences.value[j].Key })
}

func normalizeRepository(r *Repository) {
	if r.Remote.form != Present {
		return
	}
	r.Remote.value, _ = NormalizeLocator(r.Remote.value)
}
