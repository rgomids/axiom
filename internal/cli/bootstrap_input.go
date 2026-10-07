package cli

import (
	"path/filepath"
	"strings"
)

// createConfigureInput splits CREATE bootstrap flag syntax into typed intent.
// It validates only shape (separators, emptiness, duplicates of a selector);
// every semantic rule belongs to projectapp.PrepareSetup.
func createConfigureInput(values requestInput) (ConfigureInput, bool) {
	repositories, ok := parseBootstrapRepositories(values.repositories)
	if !ok {
		return ConfigureInput{}, false
	}
	provider := values.workItemProvider
	if provider == "none" {
		provider = ""
	}
	input := ConfigureInput{
		ProjectID: values.projectID, Slug: values.slug, Name: values.name, Repositories: repositories,
		WorkItemProvider: provider, PreviewDigest: values.previewDigest, AuthorizeLocal: values.authorizeLocal,
		Runtimes: append([]string(nil), values.runtimes...), ModelProfiles: append([]string(nil), values.modelProfiles...),
		RemoveTechnology: append([]string(nil), values.removeTechnology...), BusinessContext: values.businessContext,
		ContextSources: append([]string(nil), values.contextSources...),
	}
	for _, value := range values.repositoryRemotes {
		key, remote, ok := strings.Cut(value, "=")
		if !ok || key == "" || remote == "" {
			return ConfigureInput{}, false
		}
		if input.RepositoryRemotes == nil {
			input.RepositoryRemotes = map[string]string{}
		}
		if _, duplicate := input.RepositoryRemotes[key]; duplicate {
			return ConfigureInput{}, false
		}
		input.RepositoryRemotes[key] = remote
	}
	for _, value := range values.runtimePreferences {
		selector, profile, ok := strings.Cut(value, "=")
		role, complexity, split := strings.Cut(selector, "/")
		if !ok || !split || role == "" || complexity == "" || profile == "" {
			return ConfigureInput{}, false
		}
		input.RuntimePreferences = append(input.RuntimePreferences, RuntimePreferenceInput{Role: role, Complexity: complexity, ModelProfile: profile})
	}
	for _, value := range values.technology {
		key, fact, ok := strings.Cut(value, "=")
		if !ok || key == "" || fact == "" {
			return ConfigureInput{}, false
		}
		input.Technology = append(input.Technology, KeyValueInput{Key: key, Value: fact})
	}
	for _, value := range values.documentation {
		documentation, ok := parseDocumentation(value)
		if !ok {
			return ConfigureInput{}, false
		}
		input.Documentation = append(input.Documentation, documentation)
	}
	for _, value := range values.glossary {
		key, entry, ok := strings.Cut(value, "=")
		term, definition, split := strings.Cut(entry, ":")
		if !ok || !split || key == "" || strings.TrimSpace(term) == "" || strings.TrimSpace(definition) == "" {
			return ConfigureInput{}, false
		}
		input.Glossary = append(input.Glossary, GlossaryInput{Key: key, Term: strings.TrimSpace(term), Definition: strings.TrimSpace(definition)})
	}
	return input, true
}

// parseBootstrapRepositories accepts `<key>=<absolute-path>` or a bare
// absolute path whose key is proposed by discovery. Keys never contain path
// separators, so `/a=b` is a bare path, never key "/a".
func parseBootstrapRepositories(values []string) ([]RepositoryInput, bool) {
	result := make([]RepositoryInput, 0, len(values))
	for _, value := range values {
		if filepath.IsAbs(value) || strings.HasPrefix(value, "/") {
			result = append(result, RepositoryInput{Path: value})
			continue
		}
		key, path, ok := strings.Cut(value, "=")
		if !ok || key == "" || path == "" {
			return nil, false
		}
		result = append(result, RepositoryInput{Key: key, Path: path})
	}
	return result, true
}

func parseDocumentation(value string) (DocumentationInput, bool) {
	key, target, ok := strings.Cut(value, "=")
	kind, location, typed := strings.Cut(target, ":")
	if !ok || !typed || key == "" || location == "" {
		return DocumentationInput{}, false
	}
	switch kind {
	case "repository":
		repository, path, split := strings.Cut(location, "/")
		if !split || repository == "" || path == "" {
			return DocumentationInput{}, false
		}
		return DocumentationInput{Key: key, Kind: kind, Repository: repository, Path: path}, true
	case "local-file":
		return DocumentationInput{Key: key, Kind: kind, Path: location}, true
	}
	return DocumentationInput{}, false
}
