package manifest

import "github.com/rgomids/axiom/internal/project"

func toDeclaration[A, B any](o optional[A], convert func(A) B) project.Declaration[B] {
	if !o.Present {
		return project.Declaration[B]{}
	}
	if o.Unconfigured {
		return project.Unconfigured[B]()
	}
	return project.Configured(convert(o.Value))
}
func fromDeclaration[A, B any](d project.Declaration[A], convert func(A) B) optional[B] {
	if d.Form() == project.Absent {
		return optional[B]{}
	}
	if d.Form() == project.NotConfigured {
		return optional[B]{Present: true, Unconfigured: true}
	}
	value, _ := d.Value()
	return optional[B]{Present: true, Value: convert(value)}
}
func same[T any](v T) T { return v }
func list[A, B any](convert func(A) B) func([]A) []B {
	return func(input []A) []B {
		output := make([]B, len(input))
		for i, v := range input {
			output[i] = convert(v)
		}
		return output
	}
}
func toText(s text) project.Declaration[string] {
	if !s.Present {
		return project.Declaration[string]{}
	}
	return project.Configured(s.Value)
}
func fromText(d project.Declaration[string]) text {
	v, present := d.Value()
	return text{Present: present, Value: v}
}

func toDomain(d manifestDTO) project.State {
	return project.State{SchemaVersion: d.SchemaVersion, ID: d.Project.ID, Slug: d.Project.Slug, Name: d.Project.Name,
		Repositories:         toDeclaration(d.Repositories, list(toRepository)),
		Runtime:              toDeclaration(d.Runtime, toRuntime),
		Providers:            toDeclaration(d.Providers, list(toProvider)),
		Integrations:         toDeclaration(d.Integrations, list(toIntegration)),
		ModelProfiles:        toDeclaration(d.ModelProfiles, list(toProfile)),
		BusinessContext:      toDeclaration(d.BusinessContext, toContext),
		CredentialReferences: toDeclaration(d.CredentialReferences, list(toCredential)),
		Policies:             toDeclaration(d.Policies, same[[]string]),
	}
}
func toRepository(d repositoryDTO) project.Repository {
	return project.Repository{
		Key:    d.Key,
		Remote: toText(d.Remote),
	}
}
func toRuntime(d runtimeDTO) project.Runtime {
	return project.Runtime{
		ID: d.ID,
	}
}
func toProvider(d providerDTO) project.Provider {
	return project.Provider{
		Key: d.Key,
		ID:  d.ID,
	}
}
func toIntegration(d integrationDTO) project.Integration {
	return project.Integration{
		Key:           d.Key,
		ProviderRef:   toText(d.ProviderRef),
		Capabilities:  toDeclaration(d.Capabilities, same[[]string]),
		Transport:     toDeclaration(d.Transport, toTransport),
		CredentialRef: toText(d.CredentialRef),
	}
}
func toTransport(d transportDTO) project.Transport {
	return project.Transport{
		ID:        d.ID,
		Reference: toText(d.Reference),
	}
}
func toProfile(d profileDTO) project.ModelProfile {
	return project.ModelProfile{
		Key:        d.Key,
		State:      toDeclaration(d.State, same[string]),
		RuntimeRef: toText(d.RuntimeRef),
		Model:      toText(d.Model),
	}
}
func toContext(d contextDTO) project.BusinessContext {
	return project.BusinessContext{
		Text:      toText(d.Text),
		Documents: toDeclaration(d.Documents, same[[]string]),
	}
}
func toCredential(d credentialDTO) project.CredentialReference {
	return project.CredentialReference{
		Key:        d.Key,
		SourceHint: toText(d.SourceHint),
	}
}

func fromDomain(d project.State) manifestDTO {
	return manifestDTO{SchemaVersion: d.SchemaVersion, Project: identityDTO{d.ID, d.Slug, d.Name},
		Repositories:         fromDeclaration(d.Repositories, list(fromRepository)),
		Runtime:              fromDeclaration(d.Runtime, fromRuntime),
		Providers:            fromDeclaration(d.Providers, list(fromProvider)),
		Integrations:         fromDeclaration(d.Integrations, list(fromIntegration)),
		ModelProfiles:        fromDeclaration(d.ModelProfiles, list(fromProfile)),
		BusinessContext:      fromDeclaration(d.BusinessContext, fromContext),
		CredentialReferences: fromDeclaration(d.CredentialReferences, list(fromCredential)),
		Policies:             fromDeclaration(d.Policies, same[[]string]),
	}
}
func fromRepository(d project.Repository) repositoryDTO {
	return repositoryDTO{
		Key:    d.Key,
		Remote: fromText(d.Remote),
	}
}
func fromRuntime(d project.Runtime) runtimeDTO {
	return runtimeDTO{
		ID: d.ID,
	}
}
func fromProvider(d project.Provider) providerDTO {
	return providerDTO{
		Key: d.Key,
		ID:  d.ID,
	}
}
func fromIntegration(d project.Integration) integrationDTO {
	return integrationDTO{
		Key:           d.Key,
		ProviderRef:   fromText(d.ProviderRef),
		Capabilities:  fromDeclaration(d.Capabilities, same[[]string]),
		Transport:     fromDeclaration(d.Transport, fromTransport),
		CredentialRef: fromText(d.CredentialRef),
	}
}
func fromTransport(d project.Transport) transportDTO {
	return transportDTO{
		ID:        d.ID,
		Reference: fromText(d.Reference),
	}
}
func fromProfile(d project.ModelProfile) profileDTO {
	return profileDTO{
		Key:        d.Key,
		State:      fromDeclaration(d.State, same[string]),
		RuntimeRef: fromText(d.RuntimeRef),
		Model:      fromText(d.Model),
	}
}
func fromContext(d project.BusinessContext) contextDTO {
	return contextDTO{
		Text:      fromText(d.Text),
		Documents: fromDeclaration(d.Documents, same[[]string]),
	}
}
func fromCredential(d project.CredentialReference) credentialDTO {
	return credentialDTO{
		Key:        d.Key,
		SourceHint: fromText(d.SourceHint),
	}
}
