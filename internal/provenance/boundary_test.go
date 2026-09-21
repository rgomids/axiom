package provenance

import (
	"go/parser"
	"go/token"
	"path/filepath"
	"testing"
)

func TestProductionBoundaryUsesBuildInputsOnly(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), filepath.Join("provenance.go"), nil, parser.ImportsOnly)
	if err != nil {
		t.Fatal(err)
	}
	allowed := map[string]bool{
		`"errors"`:       true,
		`"regexp"`:       true,
		`"strings"`:      true,
		`"unicode"`:      true,
		`"unicode/utf8"`: true,
	}
	for _, current := range file.Imports {
		if !allowed[current.Path.Value] {
			t.Errorf("provenance imports forbidden dependency %s", current.Path.Value)
		}
	}
}
