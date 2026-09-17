package project

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// Issue contains only fixed codes and schema paths with numeric indices.
// Rejected scalar values and user-provided keys never enter diagnostics.
type Issue struct{ Field, Code string }

var (
	uuidPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	slugPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)
)

type validation struct{ issues []Issue }

func (v *validation) add(field, code string) { v.issues = append(v.issues, Issue{field, code}) }
func (v *validation) required(field, value string) {
	if strings.TrimSpace(value) == "" {
		v.add(field, "required")
	}
}
func (v *validation) optionalString(field string, d Declaration[string], blankAllowed bool) {
	if d.form == NotConfigured {
		v.add(field, "invalid_declaration")
	}
	if d.form == Present && !blankAllowed {
		v.required(field, d.value)
	}
}
func (v *validation) key(field, key string, seen map[string]bool) {
	v.required(field, key)
	if seen[key] {
		v.add(field, "duplicate_key")
	}
	seen[key] = true
}

func validate(s State) []Issue {
	v := &validation{}
	if s.SchemaVersion != 1 {
		v.add("schemaVersion", "unsupported_schema")
	}
	v.issues = append(v.issues, ValidateIdentity(s.ID, s.Slug)...)
	v.required("project.name", s.Name)
	v.repositories(s.Repositories)
	if s.Runtime.form == Present {
		v.required("runtime.id", s.Runtime.value.ID)
	}
	v.providers(s.Providers)
	v.credentials(s.CredentialReferences)
	v.integrations(s)
	v.profiles(s)
	v.context(s.BusinessContext)
	v.documents("policies", s.Policies.value)
	sort.SliceStable(v.issues, func(i, j int) bool {
		if v.issues[i].Field != v.issues[j].Field {
			return v.issues[i].Field < v.issues[j].Field
		}
		return v.issues[i].Code < v.issues[j].Code
	})
	return v.issues
}

func (v *validation) repositories(d Declaration[[]Repository]) {
	if d.form == NotConfigured {
		v.add("repositories", "invalid_declaration")
	}
	keys, locators := map[string]bool{}, map[string]bool{}
	for i, r := range d.value {
		field := fmt.Sprintf("repositories[%d]", i)
		v.key(field+".key", r.Key, keys)
		v.optionalString(field+".remote", r.Remote, false)
		if r.Remote.form != Present {
			continue
		}
		locator, issues := NormalizeLocator(r.Remote.value)
		if len(issues) != 0 {
			v.add(field+".remote", "invalid_locator")
			continue
		}
		if locators[locator] {
			v.add(field+".remote", "duplicate_locator")
		}
		locators[locator] = true
	}
}

func (v *validation) providers(d Declaration[[]Provider]) {
	keys := map[string]bool{}
	for i, p := range d.value {
		field := fmt.Sprintf("providers[%d]", i)
		v.key(field+".key", p.Key, keys)
		v.required(field+".id", p.ID)
	}
}

func (v *validation) credentials(d Declaration[[]CredentialReference]) {
	keys := map[string]bool{}
	for i, c := range d.value {
		field := fmt.Sprintf("credentialReferences[%d]", i)
		v.key(field+".key", c.Key, keys)
		v.optionalString(field+".sourceHint", c.SourceHint, false)
	}
}

func (v *validation) reference(field string, ref Declaration[string], exists bool) {
	v.optionalString(field, ref, false)
	if ref.form == Present && !exists {
		v.add(field, "dangling_reference")
	}
}

func (v *validation) integrations(s State) {
	providers, credentials := map[string]bool{}, map[string]bool{}
	for _, p := range s.Providers.value {
		providers[p.Key] = true
	}
	for _, c := range s.CredentialReferences.value {
		credentials[c.Key] = true
	}
	keys := map[string]bool{}
	for i, integration := range s.Integrations.value {
		v.integration(fmt.Sprintf("integrations[%d]", i), integration, keys, providers, credentials)
	}
}

func (v *validation) integration(field string, in Integration, keys, providers, credentials map[string]bool) {
	v.key(field+".key", in.Key, keys)
	v.reference(field+".providerRef", in.ProviderRef, providers[in.ProviderRef.value])
	v.reference(field+".credentialRef", in.CredentialRef, credentials[in.CredentialRef.value])
	if in.Capabilities.form == NotConfigured {
		v.add(field+".capabilities", "invalid_declaration")
	}
	for i, capability := range in.Capabilities.value {
		v.required(fmt.Sprintf("%s.capabilities[%d]", field, i), capability)
	}
	if in.Transport.form == NotConfigured {
		v.add(field+".transport", "invalid_declaration")
	}
	if in.Transport.form != Present {
		return
	}
	v.required(field+".transport.id", in.Transport.value.ID)
	v.optionalString(field+".transport.reference", in.Transport.value.Reference, false)
	if in.ProviderRef.form != Present {
		v.add(field+".providerRef", "required")
	}
}

func (v *validation) profiles(s State) {
	keys := map[string]bool{}
	for i, profile := range s.ModelProfiles.value {
		v.profile(fmt.Sprintf("modelProfiles[%d]", i), profile, keys, s.Runtime)
	}
}

func (v *validation) profile(field string, p ModelProfile, keys map[string]bool, runtime Declaration[Runtime]) {
	v.key(field+".key", p.Key, keys)
	if p.State.form == Present {
		v.add(field+".state", "invalid_declaration")
	}
	if p.State.form == NotConfigured {
		if p.RuntimeRef.form != Absent || p.Model.form != Absent {
			v.add(field, "invalid_profile")
		}
		return
	}
	if p.RuntimeRef.form != Present || p.Model.form != Present {
		v.add(field, "invalid_profile")
	}
	v.reference(field+".runtimeRef", p.RuntimeRef, runtime.form == Present && runtime.value.ID == p.RuntimeRef.value)
	v.optionalString(field+".model", p.Model, false)
}

func (v *validation) context(d Declaration[BusinessContext]) {
	if d.form != Present {
		return
	}
	v.optionalString("businessContext.text", d.value.Text, true)
	if d.value.Documents.form == NotConfigured {
		v.add("businessContext.documents", "invalid_declaration")
	}
	v.documents("businessContext.documents", d.value.Documents.value)
}

// Lexical document checks enforce only portable syntax. Existence, containment
// against actual files, permissions and document inspection belong to later tasks.
func (v *validation) documents(field string, documents []string) {
	for i, document := range documents {
		if !relativeDocument(document) {
			v.add(fmt.Sprintf("%s[%d]", field, i), "invalid_document_reference")
		}
	}
}

func relativeDocument(value string) bool {
	if strings.TrimSpace(value) == "" || strings.HasPrefix(value, "/") || strings.HasPrefix(value, "~") {
		return false
	}
	if strings.ContainsAny(value, "\\:$`\x00\r\n") {
		return false
	}
	for _, part := range strings.Split(value, "/") {
		if part == ".." {
			return false
		}
	}
	return true
}

// ValidateIdentity applies the same domain rules to portable identity and local
// observed identity metadata. It performs no lookup or filesystem operation.
func ValidateIdentity(id, slug string) []Issue {
	v := &validation{}
	if !uuidPattern.MatchString(id) {
		v.add("project.id", "invalid_id")
	}
	if !slugPattern.MatchString(slug) {
		v.add("project.slug", "invalid_slug")
	}
	return v.issues
}
