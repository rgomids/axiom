package manifest_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strconv"
	"strings"
	"testing"
)

// Test-only source inspection performs reads; production codec has no filesystem,
// network, environment, process or output API. Imports are explicitly allowlisted.
func TestManifestProductionDependencyBoundary(t *testing.T) {
	allowed := map[string]bool{
		"bytes": true, "io": true, "unicode/utf8": true, "fmt": true, "net/url": true, "strings": true, "unicode": true,
		"github.com/rgomids/axiom/internal/project":    true,
		"github.com/rgomids/axiom/internal/projectapp": true, "go.yaml.in/yaml/v3": true,
	}
	files, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range files {
		if !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		f, err := parser.ParseFile(token.NewFileSet(), entry.Name(), nil, parser.ParseComments)
		if err != nil {
			t.Fatal(err)
		}
		for _, imp := range f.Imports {
			name, err := strconv.Unquote(imp.Path.Value)
			if err != nil || !allowed[name] || imp.Name != nil {
				t.Fatalf("unapproved import in %s", entry.Name())
			}
		}
		ast.Inspect(f, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			if id, ok := call.Fun.(*ast.Ident); ok && (id.Name == "print" || id.Name == "println" || id.Name == "panic") {
				t.Error("effectful builtin")
			}
			if selector, ok := call.Fun.(*ast.SelectorExpr); ok {
				if pkg, ok := selector.X.(*ast.Ident); ok && pkg.Name == "fmt" && selector.Sel.Name != "Sprintf" {
					t.Error("unexpected fmt effect")
				}
			}
			return true
		})
		for _, group := range f.Comments {
			for _, comment := range group.List {
				if strings.HasPrefix(comment.Text, "//go:") {
					t.Error("compiler directive")
				}
			}
		}
	}
}
