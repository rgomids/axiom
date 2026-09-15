// Package manifest implements the bounded, strict portable axiom.yaml v1 codec.
// All operations are in memory. It performs no I/O or credential resolution.
package manifest

import (
	"bytes"
	"io"
	"unicode/utf8"

	"github.com/rgomids/axiom/internal/project"
	"github.com/rgomids/axiom/internal/projectapp"
	"go.yaml.in/yaml/v3"
)

const (
	MaxBytes = 256 * 1024
	MaxDepth = 32
	MaxNodes = 16384
)

// Decode checks UTF-8 and bytes before parser allocation. Node/depth limits count
// keys and values, root at depth 1, excluding the YAML document wrapper. The
// dependency also caps flow/indent nesting at 10,000 while building the AST.
// Byte-bounded node parsing does not resolve aliases; no alias expansion occurs.
func Decode(input []byte) (project.Project, []project.Issue) {
	n, issues := parse(input)
	if len(issues) != 0 {
		return project.Project{}, issues
	}
	if issues = validateShape(n, versionOne(), "manifest"); len(issues) != 0 {
		return project.Project{}, issues
	}
	var dto manifestDTO
	if err := n.Decode(&dto); err != nil {
		return project.Project{}, problem("manifest", "invalid_structure")
	}
	return project.New(toDomain(dto))
}

func parse(input []byte) (root *yaml.Node, issues []project.Issue) {
	if len(input) > MaxBytes {
		return nil, problem("manifest", "byte_limit")
	}
	if !utf8.Valid(input) {
		return nil, problem("manifest", "invalid_utf8")
	}
	// Defense in depth: never expose a dependency panic value or parser excerpt.
	defer func() {
		if recover() != nil {
			root = nil
			issues = problem("manifest", "invalid_yaml")
		}
	}()
	decoder := yaml.NewDecoder(bytes.NewReader(input))
	var doc yaml.Node
	if err := decoder.Decode(&doc); err != nil || len(doc.Content) != 1 {
		return nil, problem("manifest", "invalid_yaml")
	}
	root = doc.Content[0]
	if issues = validateTree(root); len(issues) != 0 {
		return nil, issues
	}
	var extra yaml.Node
	if err := decoder.Decode(&extra); err != io.EOF {
		return nil, problem("manifest", "multiple_documents")
	}
	return root, nil
}

type visit struct {
	node  *yaml.Node
	depth int
}

func validateTree(root *yaml.Node) []project.Issue {
	stack := []visit{{root, 1}}
	count := 0
	for len(stack) > 0 {
		entry := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		count++
		if count > MaxNodes {
			return problem("manifest", "node_limit")
		}
		if entry.depth > MaxDepth {
			return problem("manifest", "depth_limit")
		}
		n := entry.node
		if issues := validateNode(n); len(issues) != 0 {
			return issues
		}
		for i := len(n.Content) - 1; i >= 0; i-- {
			stack = append(stack, visit{n.Content[i], entry.depth + 1})
		}
	}
	return nil
}
func validateNode(n *yaml.Node) []project.Issue {
	if n.Anchor != "" || n.Kind == yaml.AliasNode {
		return problem("manifest", "anchor_or_alias")
	}
	switch n.Tag {
	case "!!map", "!!seq", "!!str", "!!int", "!!bool", "!!float", "!!null", "!!timestamp":
	default:
		return problem("manifest", "forbidden_tag")
	}
	if n.Kind != yaml.MappingNode {
		return nil
	}
	seen := map[string]bool{}
	for i := 0; i < len(n.Content); i += 2 {
		key := n.Content[i]
		if key.Kind != yaml.ScalarNode || key.Tag != "!!str" {
			return problem("manifest", "non_string_key")
		}
		if key.Value == "<<" {
			return problem("manifest", "merge_key")
		}
		if seen[key.Value] {
			return problem("manifest", "duplicate_key")
		}
		seen[key.Value] = true
	}
	return nil
}

// Encode emits only complete portable DTOs. Structural security and resource
// limits also apply to domain-created Projects; rejected output is never returned.
// The checked round trip prevents parser coercion, invalid UTF-8 or lossy output.
func Encode(p project.Project) ([]byte, []project.Issue) {
	if !p.Equivalent(p) {
		return nil, problem("project", "invalid_project")
	}
	var output bytes.Buffer
	encoder := yaml.NewEncoder(&output)
	encoder.SetIndent(2)
	if err := encoder.Encode(fromDomain(p.State())); err != nil {
		return nil, problem("manifest", "invalid_encoding")
	}
	if err := encoder.Close(); err != nil {
		return nil, problem("manifest", "invalid_encoding")
	}
	result := output.Bytes()
	q, issues := Decode(result)
	if len(issues) != 0 {
		return nil, issues
	}
	if !p.Equivalent(q) {
		return nil, problem("manifest", "lossy_encoding")
	}
	return result, nil
}

// Codec satisfies the existing T02 consumer port without extending its vocabulary.
// Detailed pure functions above retain safe schema paths. The port reports a
// validation error for the artifact with its existing category/remedy contract.
type Codec struct{}

var _ projectapp.ManifestCodec = Codec{}

func (Codec) Decode(input []byte) (project.Project, []projectapp.Issue) {
	p, issues := Decode(input)
	return p, applicationIssues(issues)
}
func (Codec) Encode(p project.Project) ([]byte, []projectapp.Issue) {
	b, issues := Encode(p)
	return b, applicationIssues(issues)
}
func applicationIssues(issues []project.Issue) []projectapp.Issue {
	if len(issues) == 0 {
		return nil
	}
	return []projectapp.Issue{{Phase: projectapp.ValidationPhase, Field: projectapp.ArtifactsField, Code: projectapp.InvalidSnapshot}}
}
