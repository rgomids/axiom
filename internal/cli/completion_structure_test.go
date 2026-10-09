package cli

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"testing"
)

// The canonical renderers present a classified result; they never classify one
// or hardcode provenance.
func TestCompletionRenderersDoNotClassifyOrHardcodeVersion(t *testing.T) {
	renderers := map[string][]string{"completion.go": {"renderCompletionJSON"}, "presentation.go": {"presentEvent", "renderMarkdownView", "writeChildren", "writeItem", "inlineValue", "inlineObject"}}
	for name, functions := range renderers {
		file, err := parser.ParseFile(token.NewFileSet(), filepath.Join(name), nil, parser.ParseComments)
		if err != nil {
			t.Fatal(err)
		}
		found := 0
		for _, declaration := range file.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if !ok || !contains(functions, function.Name.Name) {
				continue
			}
			found++
			ast.Inspect(function.Body, func(node ast.Node) bool {
				selector, ok := node.(*ast.SelectorExpr)
				if ok {
					identifier, isIdentifier := selector.X.(*ast.Ident)
					// Converting a decoded code to completion.Status only looks up its label.
					if isIdentifier && identifier.Name == "completion" && selector.Sel.Name != "Status" {
						t.Errorf("%s classifies with completion.%s", function.Name.Name, selector.Sel.Name)
					}
				}
				literal, ok := node.(*ast.BasicLit)
				if ok && (literal.Value == `"development"` || literal.Value == `"Axiom"`) {
					t.Errorf("%s hardcodes provenance literal %s", function.Name.Name, literal.Value)
				}
				return true
			})
		}
		if found != len(functions) {
			t.Fatalf("%s: inspected %d of %v", name, found, functions)
		}
	}
}

func contains(values []string, value string) bool {
	for _, current := range values {
		if current == value {
			return true
		}
	}
	return false
}
