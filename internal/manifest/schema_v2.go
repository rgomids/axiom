package manifest

import (
	"go.yaml.in/yaml/v3"
)

// Version dispatch selects v2 only for its exact integer literal. All other
// inputs pass through v1's original closed shape, preserving its diagnostics
// and rejection of missing, unsupported or coerced versions before DTO decode.
func versionShape(n *yaml.Node) *shape {
	if n.Tag == "!!map" {
		for i := 0; i < len(n.Content); i += 2 {
			if n.Content[i].Value == "schemaVersion" && n.Content[i+1].Tag == "!!int" && n.Content[i+1].Value == "2" {
				return versionTwo()
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
