// Package project implements Specification 002's pure portable domain.
// It neither allocates identities nor reads or writes external state.
package project

import "reflect"

// Form records declared intent independently of a Go value's zero value.
type Form uint8

const (
	Absent Form = iota
	NotConfigured
	Present
)

// Declaration retains absence, explicit unconfigured, and configured values.
// Field-specific validation determines which forms are permitted.
// A configured nil slice denotes an explicitly empty collection, never absence.
type Declaration[T any] struct {
	form  Form
	value T
}

func Configured[T any](value T) Declaration[T] { return Declaration[T]{Present, value} }
func Unconfigured[T any]() Declaration[T]      { return Declaration[T]{form: NotConfigured} }
func (d Declaration[T]) Form() Form            { return d.form }
func (d Declaration[T]) Value() (T, bool)      { return d.value, d.form == Present }

type Repository struct {
	Key    string
	Remote Declaration[string]
}
type Runtime struct{ ID string }
type Provider struct{ Key, ID string }
type Transport struct {
	ID        string
	Reference Declaration[string]
}
type Integration struct {
	Key           string
	ProviderRef   Declaration[string]
	Capabilities  Declaration[[]string]
	Transport     Declaration[Transport]
	CredentialRef Declaration[string]
}
type ModelProfile struct {
	Key               string
	State             Declaration[string]
	RuntimeRef, Model Declaration[string]
}
type BusinessContext struct {
	Text      Declaration[string]
	Documents Declaration[[]string]
}
type CredentialReference struct {
	Key        string
	SourceHint Declaration[string]
}

// State is complete portable domain input, not a wire DTO or a validated Project.
// It cannot carry local bindings, observations, credentials, or Git backing.
type State struct {
	SchemaVersion        int
	ID, Slug, Name       string
	Repositories         Declaration[[]Repository]
	Runtime              Declaration[Runtime]
	Providers            Declaration[[]Provider]
	Integrations         Declaration[[]Integration]
	ModelProfiles        Declaration[[]ModelProfile]
	BusinessContext      Declaration[BusinessContext]
	CredentialReferences Declaration[[]CredentialReference]
	Policies             Declaration[[]string]
}

// Project owns an immutable validated snapshot. Its zero value is invalid.
type Project struct {
	state State
	valid bool
}

// New validates and normalizes complete input, retaining all presence distinctions.
// Failed construction never returns a partially valid Project.
func New(input State) (Project, []Issue) {
	s := cloneState(input)
	issues := validate(s)
	if len(issues) != 0 {
		return Project{}, issues
	}
	normalize(&s)
	return Project{state: s, valid: true}, nil
}

// State returns a detached snapshot; editing it cannot mutate this Project.
func (p Project) State() State { return cloneState(p.state) }

// Equivalent compares normalized validated intent, not declaration order or I/O.
// Document and capability list order and all nested presence remain significant.
func (p Project) Equivalent(other Project) bool {
	return p.valid && other.valid && reflect.DeepEqual(p.state, other.state)
}

// Change distinguishes omitted intent from an explicit field replacement.
type Change[T any] struct {
	supplied bool
	value    T
}

func Set[T any](value T) Change[T] { return Change[T]{true, value} }

// Intent replaces only explicitly supplied top-level mutable fields.
// Set(Declaration[T]{}) removes a declaration; zero Change retains it.
// Supplied objects/collections replace that entire field (no implicit deep merge).
// ID and schema version are deliberately not mutable input.
type Intent struct {
	Slug, Name           Change[string]
	Repositories         Change[Declaration[[]Repository]]
	Runtime              Change[Declaration[Runtime]]
	Providers            Change[Declaration[[]Provider]]
	Integrations         Change[Declaration[[]Integration]]
	ModelProfiles        Change[Declaration[[]ModelProfile]]
	BusinessContext      Change[Declaration[BusinessContext]]
	CredentialReferences Change[Declaration[[]CredentialReference]]
	Policies             Change[Declaration[[]string]]
}

// Propose materializes and validates the entire result, including retained refs.
// It never changes the receiver, including on failed validation.
func (p Project) Propose(intent Intent) (Project, []Issue) {
	if !p.valid {
		return Project{}, []Issue{{Field: "project", Code: "invalid_current_state"}}
	}
	s := p.State()
	apply(&s.Slug, intent.Slug)
	apply(&s.Name, intent.Name)
	apply(&s.Repositories, intent.Repositories)
	apply(&s.Runtime, intent.Runtime)
	apply(&s.Providers, intent.Providers)
	apply(&s.Integrations, intent.Integrations)
	apply(&s.ModelProfiles, intent.ModelProfiles)
	apply(&s.BusinessContext, intent.BusinessContext)
	apply(&s.CredentialReferences, intent.CredentialReferences)
	apply(&s.Policies, intent.Policies)
	return New(s)
}

func apply[T any](target *T, change Change[T]) {
	if change.supplied {
		*target = change.value
	}
}
