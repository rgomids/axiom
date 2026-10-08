package manifest

import (
	"go.yaml.in/yaml/v3"
)

// Version dispatch selects v2 or v3 only for their exact integer literals. All other
// inputs pass through v1's original closed shape, preserving its diagnostics
// and rejection of missing, unsupported or coerced versions before DTO decode.
func versionShape(n *yaml.Node) *shape {
	if n.Tag == "!!map" {
		for i := 0; i < len(n.Content); i += 2 {
			if n.Content[i].Value == "schemaVersion" && n.Content[i+1].Tag == "!!int" && n.Content[i+1].Value == "2" {
				return versionTwo()
			}
			if n.Content[i].Value == "schemaVersion" && n.Content[i+1].Tag == "!!int" && n.Content[i+1].Value == "3" {
				return versionThree()
			}
		}
	}
	return versionOne()
}

func boundedSequence(item *shape, limit int) *shape {
	s := sequence(item)
	s.maxItems = limit
	return s
}

func versionTwo() *shape {
	s := versionOne()
	for i, f := range s.fields {
		switch f.name {
		case "schemaVersion":
			s.fields[i].shape.literal = "2"
		case "runtime":
			s.fields[i] = optionalField("runtimes", union(boundedSequence(object(required("id", &shape{tag: "!!str", security: "identifier"})), 8)))
		case "modelProfiles":
			s.fields[i].shape.maxItems = 32
		}
	}
	token := &shape{tag: "!!str", security: "token"}
	logical := &shape{tag: "!!str", security: "logical"}
	s.fields = append(s.fields, optionalField("runtimePreferences", union(boundedSequence(object(
		required("role", token), required("complexity", token), required("modelProfileRef", logical),
	), 32))))
	return s
}

// Version 3 is v2 plus the closed Issue #231 context registry. Domain
// validation owns reference integrity, kinds and text bounds.
func versionThree() *shape {
	s := versionTwo()
	token := &shape{tag: "!!str", security: "token"}
	identifier := &shape{tag: "!!str", security: "identifier"}
	logical := &shape{tag: "!!str", security: "logical"}
	prose := &shape{tag: "!!str", security: "prose"}
	for i, f := range s.fields {
		switch f.name {
		case "schemaVersion":
			s.fields[i].shape.literal = "3"
		case "businessContext":
			context := *f.shape
			context.fields = append([]field(nil), context.fields...)
			for j, field := range context.fields {
				if field.name == "text" {
					context.fields[j].shape = prose
				}
			}
			context.fields = append(context.fields,
				optionalField("sourceRefs", boundedSequence(token, 64)),
				optionalField("glossary", boundedSequence(object(required("key", token), required("term", prose), required("definition", prose)), 128)),
			)
			s.fields[i].shape = &context
		}
	}
	s.fields = append(s.fields,
		optionalField("technologyContext", union(boundedSequence(object(required("key", token), required("value", identifier)), 64))),
		optionalField("documentationSources", union(boundedSequence(object(
			required("key", token), required("kind", token), optionalField("repositoryRef", logical), optionalField("path", identifier),
		), 64))),
	)
	return s
}
