package local

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

func TestLocalProductionDependencyBoundary(t *testing.T) {
	allowed := map[string]bool{"bytes": true, "encoding/json": true, "encoding/hex": true, "fmt": true, "io": true, "net/url": true, "path": true, "reflect": true, "slices": true, "strings": true, "time": true, "unicode": true, "unicode/utf8": true, "github.com/rgomids/axiom/internal/project": true, "github.com/rgomids/axiom/internal/projectapp": true}
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
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
				t.Fatalf("unexpected dependency in %s", entry.Name())
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
			if s, ok := call.Fun.(*ast.SelectorExpr); ok {
				if p, ok := s.X.(*ast.Ident); ok && ((p.Name == "fmt" && s.Sel.Name != "Sprintf") || (p.Name == "time" && s.Sel.Name != "Parse")) {
					t.Error("unexpected effect")
				}
			}
			return true
		})
		for _, g := range f.Comments {
			for _, c := range g.List {
				if strings.HasPrefix(c.Text, "//go:") {
					t.Error("compiler directive")
				}
			}
		}
	}
}

// Exact field/type assertions force review if any body/value/extension container
// becomes representable, even if a behavioral fixture does not exercise it.
func TestRecordDTOClosedMetadataInventory(t *testing.T) {
	inventories := map[reflect.Type]string{
		reflect.TypeFor[recordDTO]():      "formatVersion projectId observedSlug sourceLocation portableRevision artifactDigests repositories credentials runtime attempt",
		reflect.TypeFor[digestDTO]():      "name digest",
		reflect.TypeFor[repositoryDTO]():  "repositoryKey explicitPath canonicalIdentity observation",
		reflect.TypeFor[credentialDTO]():  "referenceKey sourceKind itemReference",
		reflect.TypeFor[runtimeDTO]():     "runtimeId explicitPath observation",
		reflect.TypeFor[observationDTO](): "availability basis observedAt",
		reflect.TypeFor[attemptDTO]():     "correlation at",
	}
	for typ, names := range inventories {
		fields := strings.Fields(names)
		if typ.NumField() != len(fields) {
			t.Fatalf("DTO changed: %s", typ.Name())
		}
		for i, name := range fields {
			f := typ.Field(i)
			if strings.Split(f.Tag.Get("json"), ",")[0] != name {
				t.Fatalf("unexpected field in %s", typ.Name())
			}
			base := f.Type
			for base.Kind() == reflect.Pointer || base.Kind() == reflect.Slice {
				base = base.Elem()
			}
			if base.Kind() != reflect.String && base.Kind() != reflect.Int && inventories[base] == "" {
				t.Fatalf("unapproved payload type in %s", typ.Name())
			}
		}
	}
}
