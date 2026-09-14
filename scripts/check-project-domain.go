//go:build ignore

// This standalone verification tool inspects domain sources outside domain tests.
// Run from the repository root: go run ./scripts/check-project-domain.go
package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Restrict qualified standard-library symbols to the pure operations consumed.
// Any expansion requires an explicit review of its I/O behavior.
var allowed = map[string]string{
	"fmt":       "Sprintf",
	"reflect":   "DeepEqual",
	"regexp":    "MustCompile",
	"sort":      "Slice SliceStable",
	"strings":   "TrimSpace HasPrefix ContainsAny Split ContainsRune IndexFunc Contains Index IndexAny LastIndex ToLower EqualFold IndexByte Count TrimSuffix Repeat",
	"net/url":   "Parse URL",
	"net/netip": "ParseAddr",
	"unicode":   "IsSpace IsControl",
	"testing":   "T",
}

func main() {
	entries, err := os.ReadDir("internal/project")
	if err != nil {
		fail("cannot inspect domain directory")
	}
	count := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			fail("unexpected domain file type")
		}
		check(filepath.Join("internal/project", entry.Name()))
		count++
	}
	if count == 0 {
		fail("no domain sources checked")
	}
	fmt.Printf("PASS: %d domain source/test files; pure import/symbol allowlist; no test I/O helpers or compiler directives\n", count)
}

func check(path string) {
	f, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ParseComments)
	if err != nil {
		fail("cannot parse domain source")
	}
	imports := map[string]string{}
	for _, imp := range f.Imports {
		name, _ := strconv.Unquote(imp.Path.Value)
		if imp.Name != nil {
			fail("domain import aliases require review")
		}
		if name == "github.com/rgomids/axiom/internal/project" && strings.HasSuffix(path, "_test.go") {
			continue
		}
		if _, ok := allowed[name]; !ok {
			fail("non-allowlisted domain import: " + name)
		}
		if name == "testing" && !strings.HasSuffix(path, "_test.go") {
			fail("testing imported by production domain")
		}
		imports[filepath.Base(name)] = name
	}
	for _, group := range f.Comments {
		for _, comment := range group.List {
			if strings.Contains(comment.Text, "go:") {
				fail("domain compiler directive requires review")
			}
		}
	}
	ast.Inspect(f, func(node ast.Node) bool {
		selector, ok := node.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		for _, forbidden := range []string{"TempDir", "Setenv", "Chdir", "Output", "Attr"} {
			if selector.Sel.Name == forbidden {
				fail("domain test I/O helper forbidden")
			}
		}
		identifier, ok := selector.X.(*ast.Ident)
		if !ok {
			return true
		}
		name, imported := imports[identifier.Name]
		if imported && !strings.Contains(" "+allowed[name]+" ", " "+selector.Sel.Name+" ") {
			fail("non-allowlisted domain symbol: " + name + "." + selector.Sel.Name)
		}
		return true
	})
}

func fail(message string) { fmt.Fprintln(os.Stderr, "FAIL: "+message); os.Exit(1) }
