package completion

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"testing"
)

func TestProductionBoundaryHasNoIOProviderOrPresentationDependency(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), filepath.Join("completion.go"), nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	allowed := map[string]bool{
		`"errors"`:       true,
		`"sort"`:         true,
		`"strings"`:      true,
		`"unicode"`:      true,
		`"unicode/utf8"`: true,
		`"github.com/rgomids/axiom/internal/provenance"`: true,
	}
	for _, current := range file.Imports {
		if !allowed[current.Path.Value] {
			t.Errorf("completion imports forbidden dependency %s", current.Path.Value)
		}
	}
	if len(file.Decls) == 0 || !containsType(file, "Result") {
		t.Fatal("canonical Result type missing")
	}
}

func containsType(file *ast.File, name string) bool {
	for _, declaration := range file.Decls {
		general, ok := declaration.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, specification := range general.Specs {
			typed, ok := specification.(*ast.TypeSpec)
			if ok && typed.Name.Name == name {
				return true
			}
		}
	}
	return false
}
