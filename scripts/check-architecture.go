//go:build ignore

// This standalone repository verification tool inspects one product layer's
// sources, including its tests, against that layer's architecture policy. It is
// maintainer tooling, not product code: it is not imported by any package and
// is not exposed through Lingo.
//
// Run from the repository root:
//
//	go run ./scripts/check-architecture.go domain
//	go run ./scripts/check-architecture.go application
package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

const usage = "usage: go run ./scripts/check-architecture.go <domain|application>"

// policy is one layer's independent architecture rules. Profiles never share
// or inherit rules: each declares its own allowlist, exceptions and messages.
type policy struct {
	// label names the layer in every diagnostic.
	label string
	// dir is the layer's source directory, relative to the repository root.
	dir string
	// allowed restricts qualified standard-library and internal symbols to the
	// pure operations consumed, keyed by import path. Any expansion requires an
	// explicit review of its I/O behavior.
	allowed map[string]string
	// testSelfImport is the layer's own import path, accepted only in
	// _test.go files (external test packages).
	testSelfImport string
	// testOnlyImports are allowlisted imports rejected in production files.
	testOnlyImports []string
	// testOnlyMessage reports a testOnlyImports entry in a production file.
	testOnlyMessage string
	// forbiddenSelectors are selector names rejected anywhere, such as test
	// I/O helpers.
	forbiddenSelectors []string
}

// ---------------------------------------------------------------------------
// Domain policy: internal/project stays pure.
// ---------------------------------------------------------------------------

var domainPolicy = policy{
	label: "domain",
	dir:   "internal/project",
	allowed: map[string]string{
		"github.com/rgomids/axiom/internal/portableconfig": "SafeProse",
		"fmt":          "Sprint Sprintf",
		"reflect":      "DeepEqual",
		"regexp":       "MustCompile",
		"sort":         "Slice SliceStable Strings",
		"strings":      "TrimSpace HasPrefix ContainsAny Split ContainsRune IndexFunc Contains Index IndexAny LastIndex ToLower EqualFold IndexByte Count TrimSuffix Repeat",
		"net/url":      "Parse URL",
		"net/netip":    "ParseAddr",
		"unicode":      "IsSpace IsControl",
		"unicode/utf8": "ValidString RuneError",
		"testing":      "T",
	},
	testSelfImport:     "github.com/rgomids/axiom/internal/project",
	testOnlyImports:    []string{"testing"},
	testOnlyMessage:    "testing imported by production domain",
	forbiddenSelectors: []string{"TempDir", "Setenv", "Chdir", "Output", "Attr"},
}

// ---------------------------------------------------------------------------
// Application policy: internal/projectapp depends only inward and stays free
// of ambient I/O.
// ---------------------------------------------------------------------------

var applicationPolicy = policy{
	label: "application",
	dir:   "internal/projectapp",
	allowed: map[string]string{
		"context":         "Context Background WithCancel Canceled DeadlineExceeded",
		"crypto/sha256":   "Sum256",
		"encoding/binary": "BigEndian",
		"encoding/hex":    "EncodeToString",
		"encoding/json":   "Marshal",
		"errors":          "Is New",
		"sort":            "Slice SliceStable Strings",
		"strings":         "ContainsAny HasPrefix Split Contains Join TrimSpace IndexFunc Repeat Trim",
		"sync":            "Mutex",
		"sync/atomic":     "Bool",
		"path/filepath":   "Clean IsAbs",
		"time":            "Time Unix",
		"reflect":         "DeepEqual",
		"testing":         "T",
		"unicode/utf8":    "ValidString",
		"unicode":         "IsControl",
		"regexp":          "MustCompile",
		"github.com/rgomids/axiom/internal/project":  "ContextValue DocumentationSource GlossaryEntry LocalFileSource MaxDocumentationSources MaxGlossaryDefinitionLen MaxGlossaryTermBytes MaxTechnologyFacts MaxTechnologyValueBytes NormalizeLocator Present RepositoryRelativePath RepositorySource RuntimePolicyDeclared RuntimePreference TechnologyFact ValidContextKey PortableContextValue MaxBusinessContextBytes Project State Issue New Configured BusinessContext Declaration Repository Provider Runtime Integration ModelProfile CredentialReference Intent Set ValidSlug Absent NotConfigured Unconfigured",
		"github.com/rgomids/axiom/internal/manifest": "Codec Decode",
		"github.com/rgomids/axiom/internal/testfs":   "Path",
	},
	testSelfImport:     "github.com/rgomids/axiom/internal/projectapp",
	testOnlyImports:    []string{"testing", "reflect", "sync", "github.com/rgomids/axiom/internal/manifest", "github.com/rgomids/axiom/internal/testfs"},
	testOnlyMessage:    "test-only helper imported by production application",
	forbiddenSelectors: []string{"TempDir", "Setenv", "Chdir", "Output", "Attr"},
}

var profiles = map[string]policy{
	"domain":      domainPolicy,
	"application": applicationPolicy,
}

// ---------------------------------------------------------------------------
// Shared engine: walks one layer directory and applies exactly one policy.
// ---------------------------------------------------------------------------

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(2)
	}
	p, ok := profiles[os.Args[1]]
	if !ok {
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(2)
	}
	entries, err := os.ReadDir(p.dir)
	if err != nil {
		fail("cannot inspect " + p.label + " directory")
	}
	count := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			fail("unexpected " + p.label + " file type")
		}
		check(p, filepath.Join(p.dir, entry.Name()))
		count++
	}
	if count == 0 {
		fail("no " + p.label + " sources checked")
	}
	fmt.Printf("PASS: %d %s source/test files; pure import/symbol allowlist; no test I/O helpers or compiler directives\n", count, p.label)
}

func check(p policy, path string) {
	f, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ParseComments)
	if err != nil {
		fail("cannot parse " + p.label + " source")
	}
	test := strings.HasSuffix(path, "_test.go")
	imports := map[string]string{}
	for _, imp := range f.Imports {
		name, _ := strconv.Unquote(imp.Path.Value)
		if imp.Name != nil {
			fail(p.label + " import aliases require review")
		}
		if name == p.testSelfImport && test {
			continue
		}
		if _, ok := p.allowed[name]; !ok {
			fail("non-allowlisted " + p.label + " import: " + name)
		}
		if slices.Contains(p.testOnlyImports, name) && !test {
			fail(p.testOnlyMessage)
		}
		imports[filepath.Base(name)] = name
	}
	for _, group := range f.Comments {
		for _, comment := range group.List {
			if strings.Contains(comment.Text, "go:") {
				fail(p.label + " compiler directive requires review")
			}
		}
	}
	ast.Inspect(f, func(node ast.Node) bool {
		selector, ok := node.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		if slices.Contains(p.forbiddenSelectors, selector.Sel.Name) {
			fail(p.label + " test I/O helper forbidden")
		}
		identifier, ok := selector.X.(*ast.Ident)
		if !ok || identifier.Obj != nil {
			return true
		}
		name, imported := imports[identifier.Name]
		if imported && !strings.Contains(" "+p.allowed[name]+" ", " "+selector.Sel.Name+" ") {
			fail("non-allowlisted " + p.label + " symbol: " + name + "." + selector.Sel.Name)
		}
		return true
	})
}

func fail(message string) { fmt.Fprintln(os.Stderr, "FAIL: "+message); os.Exit(1) }
