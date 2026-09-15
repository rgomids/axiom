package manifest

import (
	"fmt"

	"github.com/rgomids/axiom/internal/project"
	"go.yaml.in/yaml/v3"
)

type field struct {
	name     string
	required bool
	shape    *shape
}
type shape struct {
	tag          string
	fields       []field
	item         *shape
	unconfigured bool
	literal      string
	security     string
}

func object(fields ...field) *shape             { return &shape{tag: "!!map", fields: fields} }
func sequence(item *shape) *shape               { return &shape{tag: "!!seq", item: item} }
func union(s *shape) *shape                     { c := *s; c.unconfigured = true; return &c }
func required(name string, s *shape) field      { return field{name, true, s} }
func optionalField(name string, s *shape) field { return field{name, false, s} }

// Version 1 is closed at every object; no extension or local-state bag exists.
func versionOne() *shape {
	plain := &shape{tag: "!!str"}
	identifier := &shape{tag: "!!str", security: "identifier"}
	logical := &shape{tag: "!!str", security: "logical"}
	remote := &shape{tag: "!!str", security: "remote"}
	transport := object(required("id", identifier), optionalField("reference", logical))
	return object(
		required("schemaVersion", &shape{tag: "!!int", literal: "1"}),
		required("project", object(required("id", plain), required("slug", plain), required("name", plain))),
		optionalField("repositories", sequence(object(required("key", identifier), optionalField("remote", remote)))),
		optionalField("runtime", union(object(required("id", identifier)))),
		optionalField("providers", union(sequence(object(required("key", identifier), required("id", identifier))))),
		optionalField("integrations", union(sequence(object(required("key", identifier), optionalField("providerRef", logical), optionalField("capabilities", sequence(identifier)), optionalField("transport", transport), optionalField("credentialRef", logical))))),
		optionalField("modelProfiles", union(sequence(object(required("key", identifier), optionalField("state", &shape{tag: "!!str", literal: "unconfigured"}), optionalField("runtimeRef", logical), optionalField("model", identifier))))),
		optionalField("businessContext", union(object(optionalField("text", plain), optionalField("documents", sequence(plain))))),
		optionalField("credentialReferences", union(sequence(object(required("key", logical), optionalField("sourceHint", logical))))),
		optionalField("policies", union(sequence(plain))),
	)
}

func validateShape(n *yaml.Node, s *shape, path string) []project.Issue {
	if s.unconfigured && n.Kind == yaml.ScalarNode && n.Tag == "!!str" && n.Value == "unconfigured" {
		return nil
	}
	if n.Tag != s.tag {
		return problem(path, "invalid_type")
	}
	if s.literal != "" && n.Value != s.literal {
		return problem(path, "unsupported_value")
	}
	if s.security != "" && !safeValue(n.Value, s.security) {
		return problem(path, "forbidden_value")
	}
	if s.tag == "!!map" {
		return validateObject(n, s, path)
	}
	if s.tag == "!!seq" {
		return validateSequence(n, s, path)
	}
	return nil
}
func validateObject(n *yaml.Node, s *shape, path string) []project.Issue {
	known := make(map[string]*shape, len(s.fields))
	for _, f := range s.fields {
		known[f.name] = f.shape
	}
	seen := make(map[string]*yaml.Node, len(n.Content)/2)
	for i := 0; i < len(n.Content); i += 2 {
		name := n.Content[i].Value
		if known[name] == nil {
			return problem(path, "unknown_key")
		}
		seen[name] = n.Content[i+1]
	}
	// Schema order is fixed; unknown user keys are never part of a diagnostic path.
	for _, f := range s.fields {
		childPath := f.name
		if path != "manifest" {
			childPath = path + "." + f.name
		}
		child := seen[f.name]
		if child == nil && f.required {
			return problem(childPath, "required")
		}
		if child == nil {
			continue
		}
		if issues := validateShape(child, f.shape, childPath); len(issues) != 0 {
			return issues
		}
	}
	return nil
}
func validateSequence(n *yaml.Node, s *shape, path string) []project.Issue {
	for i, child := range n.Content {
		if issues := validateShape(child, s.item, fmt.Sprintf("%s[%d]", path, i)); len(issues) != 0 {
			return issues
		}
	}
	return nil
}
func problem(path, code string) []project.Issue { return []project.Issue{{Field: path, Code: code}} }
