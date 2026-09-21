package cli

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"testing"
)

func TestCompletionRenderersDoNotClassifyOrHardcodeVersion(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), filepath.Join("completion.go"), nil, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || (function.Name.Name != "renderCompletionHuman" && function.Name.Name != "renderCompletionJSON") {
			continue
		}
		ast.Inspect(function.Body, func(node ast.Node) bool {
			selector, ok := node.(*ast.SelectorExpr)
			if ok {
				identifier, isIdentifier := selector.X.(*ast.Ident)
				if isIdentifier && identifier.Name == "completion" {
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
}
