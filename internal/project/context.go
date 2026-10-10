package project

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/rgomids/axiom/internal/portableconfig"
)

// Schema v3 context bounds (Issue #231 contract §1).
const (
	MaxBusinessContextBytes  = 4096
	MaxTechnologyFacts       = 64
	MaxDocumentationSources  = 64
	MaxContextSourceRefs     = 64
	MaxGlossaryEntries       = 128
	MaxTechnologyValueBytes  = 256
	MaxDocumentationPath     = 1024
	MaxGlossaryTermBytes     = 256
	MaxGlossaryDefinitionLen = 2048
)

var contextKeyPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,127}$`)

// ValidContextKey is the v3 logical key grammar shared by technology facts,
// documentation sources and glossary entries.
func ValidContextKey(key string) bool { return contextKeyPattern.MatchString(key) }

func (v *validation) contextRegistry(s State) {
	if s.SchemaVersion != 3 && s.SchemaVersion != 4 {
		for field, form := range map[string]Form{
			"technologyContext":          s.TechnologyContext.form,
			"documentationSources":       s.DocumentationSources.form,
			"businessContext.sourceRefs": s.BusinessContext.value.SourceRefs.form,
			"businessContext.glossary":   s.BusinessContext.value.Glossary.form,
		} {
			if form != Absent {
				v.add(field, "unsupported_field")
			}
		}
		return
	}
	if len(s.TechnologyContext.value) > MaxTechnologyFacts {
		v.add("technologyContext", "collection_limit")
	}
	keys := map[string]bool{}
	for i, fact := range s.TechnologyContext.value {
		field := fmt.Sprintf("technologyContext[%d]", i)
		v.contextKey(field+".key", fact.Key, keys)
		if !ContextValue(fact.Value, MaxTechnologyValueBytes, false) {
			v.add(field+".value", "invalid_value")
		}
	}
	repositories := map[string]bool{}
	for _, repository := range s.Repositories.value {
		repositories[repository.Key] = true
	}
	if len(s.DocumentationSources.value) > MaxDocumentationSources {
		v.add("documentationSources", "collection_limit")
	}
	sources := map[string]bool{}
	for i, source := range s.DocumentationSources.value {
		v.documentationSource(fmt.Sprintf("documentationSources[%d]", i), source, sources, repositories)
	}
	v.contextAdditions(s.BusinessContext, sources)
}

func (v *validation) contextKey(field, key string, seen map[string]bool) {
	if !ValidContextKey(key) {
		v.add(field, "invalid_key")
	}
	if seen[key] {
		v.add(field, "duplicate_key")
	}
	seen[key] = true
}

func (v *validation) documentationSource(field string, source DocumentationSource, keys, repositories map[string]bool) {
	v.contextKey(field+".key", source.Key, keys)
	switch source.Kind {
	case RepositorySource:
		if source.RepositoryRef.form != Present || source.Path.form != Present {
			v.add(field, "invalid_source")
			return
		}
		if !repositories[source.RepositoryRef.value] {
			v.add(field+".repositoryRef", "dangling_reference")
		}
		if !RepositoryRelativePath(source.Path.value) {
			v.add(field+".path", "invalid_document_reference")
		}
	case LocalFileSource:
		if source.RepositoryRef.form != Absent || source.Path.form != Absent {
			v.add(field, "invalid_source")
		}
	default:
		v.add(field+".kind", "unsupported_source_kind")
	}
}

func (v *validation) contextAdditions(d Declaration[BusinessContext], sources map[string]bool) {
	if d.form != Present {
		return
	}
	if d.value.Text.form == Present && !PortableContextValue(d.value.Text.value, MaxBusinessContextBytes, true) {
		v.add("businessContext.text", "invalid_value")
	}
	refs, glossary := d.value.SourceRefs, d.value.Glossary
	if refs.form == NotConfigured {
		v.add("businessContext.sourceRefs", "invalid_declaration")
	}
	if len(refs.value) > MaxContextSourceRefs {
		v.add("businessContext.sourceRefs", "collection_limit")
	}
	seen := map[string]bool{}
	for i, ref := range refs.value {
		field := fmt.Sprintf("businessContext.sourceRefs[%d]", i)
		if seen[ref] {
			v.add(field, "duplicate_key")
		}
		seen[ref] = true
		if !sources[ref] {
			v.add(field, "dangling_reference")
		}
	}
	if glossary.form == NotConfigured {
		v.add("businessContext.glossary", "invalid_declaration")
	}
	if len(glossary.value) > MaxGlossaryEntries {
		v.add("businessContext.glossary", "collection_limit")
	}
	keys := map[string]bool{}
	for i, entry := range glossary.value {
		field := fmt.Sprintf("businessContext.glossary[%d]", i)
		v.contextKey(field+".key", entry.Key, keys)
		if !PortableContextValue(entry.Term, MaxGlossaryTermBytes, false) {
			v.add(field+".term", "invalid_value")
		}
		if !PortableContextValue(entry.Definition, MaxGlossaryDefinitionLen, true) {
			v.add(field+".definition", "invalid_value")
		}
	}
}

// ContextValue bounds operator-confirmed context text: nonblank, unpadded,
// valid UTF-8, no control character except line feeds when multiline.
func ContextValue(value string, limit int, multiline bool) bool {
	if value == "" || len(value) > limit || !utf8.ValidString(value) || strings.TrimSpace(value) != value {
		return false
	}
	return strings.IndexFunc(value, func(r rune) bool {
		return unicode.IsControl(r) && !(multiline && r == '\n') || r == utf8.RuneError
	}) < 0
}

// RepositoryRelativePath accepts only canonical slash-separated relative paths
// with no empty, dot or traversal component. It is lexical only; resolution
// against a local binding must still reject links and escapes.
func RepositoryRelativePath(value string) bool {
	if len(value) > MaxDocumentationPath || !relativeDocument(value) {
		return false
	}
	for _, part := range strings.Split(value, "/") {
		if part == "" || part == "." || part == ".." || strings.TrimSpace(part) != part {
			return false
		}
	}
	return true
}

// PortableContextValue combines the context text contract with shared structural
// safety. Schema v3 uses this policy; legacy context validation stays unchanged.
func PortableContextValue(value string, limit int, multiline bool) bool {
	return ContextValue(value, limit, multiline) && portableconfig.SafeProse(value)
}
