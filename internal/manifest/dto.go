package manifest

import "go.yaml.in/yaml/v3"

// Wire-only DTOs retain presence without importing parser types into the domain.
// Decode is called only after the entire node tree passes the closed schema.
type optional[T any] struct {
	Present      bool
	Unconfigured bool
	Value        T
}

func (o *optional[T]) UnmarshalYAML(n *yaml.Node) error {
	o.Present = true
	if n.Kind == yaml.ScalarNode && n.Tag == "!!str" && n.Value == "unconfigured" {
		o.Unconfigured = true
		return nil
	}
	return n.Decode(&o.Value)
}
func (o optional[T]) IsZero() bool { return !o.Present }
func (o optional[T]) MarshalYAML() (any, error) {
	if o.Unconfigured {
		return "unconfigured", nil
	}
	return o.Value, nil
}

// Optional strings never interpret the word unconfigured as a declaration state.
type text struct {
	Present bool
	Value   string
}

func (s *text) UnmarshalYAML(n *yaml.Node) error { s.Present = true; s.Value = n.Value; return nil }
func (s text) IsZero() bool                      { return !s.Present }
func (s text) MarshalYAML() (any, error)         { return s.Value, nil }

type manifestDTO struct {
	SchemaVersion        int                        `yaml:"schemaVersion"`
	Project              identityDTO                `yaml:"project"`
	Repositories         optional[[]repositoryDTO]  `yaml:"repositories,omitempty"`
	Runtime              optional[runtimeDTO]       `yaml:"runtime,omitempty"`
	Providers            optional[[]providerDTO]    `yaml:"providers,omitempty"`
	Integrations         optional[[]integrationDTO] `yaml:"integrations,omitempty"`
	ModelProfiles        optional[[]profileDTO]     `yaml:"modelProfiles,omitempty"`
	BusinessContext      optional[contextDTO]       `yaml:"businessContext,omitempty"`
	CredentialReferences optional[[]credentialDTO]  `yaml:"credentialReferences,omitempty"`
	Policies             optional[[]string]         `yaml:"policies,omitempty"`
}
type identityDTO struct {
	ID   string `yaml:"id"`
	Slug string `yaml:"slug"`
	Name string `yaml:"name"`
}
type repositoryDTO struct {
	Key    string `yaml:"key"`
	Remote text   `yaml:"remote,omitempty"`
}
type runtimeDTO struct {
	ID string `yaml:"id"`
}
type providerDTO struct {
	Key string `yaml:"key"`
	ID  string `yaml:"id"`
}
type integrationDTO struct {
	Key           string                 `yaml:"key"`
	ProviderRef   text                   `yaml:"providerRef,omitempty"`
	Capabilities  optional[[]string]     `yaml:"capabilities,omitempty"`
	Transport     optional[transportDTO] `yaml:"transport,omitempty"`
	CredentialRef text                   `yaml:"credentialRef,omitempty"`
}
type transportDTO struct {
	ID        string `yaml:"id"`
	Reference text   `yaml:"reference,omitempty"`
}
type profileDTO struct {
	Key        string           `yaml:"key"`
	State      optional[string] `yaml:"state,omitempty"`
	RuntimeRef text             `yaml:"runtimeRef,omitempty"`
	Model      text             `yaml:"model,omitempty"`
}
type contextDTO struct {
	Text      text               `yaml:"text,omitempty"`
	Documents optional[[]string] `yaml:"documents,omitempty"`
}
type credentialDTO struct {
	Key        string `yaml:"key"`
	SourceHint text   `yaml:"sourceHint,omitempty"`
}
