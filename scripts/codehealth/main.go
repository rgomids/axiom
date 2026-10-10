// Command codehealth measures Go function complexity using the standard AST.
// It is a repository tool, not a Lingo product command.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

type function struct {
	ID         string `json:"id"`
	File       string `json:"file"`
	Line       int    `json:"line"`
	Complexity int    `json:"complexity"`
}

// Nested literal decisions belong to the enclosing declaration. Each if, for,
// range, non-default case and short-circuit boolean operator adds one path.
func complexity(body *ast.BlockStmt) int {
	n := 1
	ast.Inspect(body, func(node ast.Node) bool {
		switch x := node.(type) {
		case *ast.IfStmt, *ast.ForStmt, *ast.RangeStmt:
			n++
		case *ast.CaseClause:
			if x.List != nil {
				n++
			}
		case *ast.CommClause:
			if x.Comm != nil {
				n++
			}
		case *ast.BinaryExpr:
			if x.Op == token.LAND || x.Op == token.LOR {
				n++
			}
		}
		return true
	})
	return n
}

func functions(path string, source []byte) ([]function, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, source, 0)
	if err != nil {
		return nil, err
	}
	result := []function{}
	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		name := fn.Name.Name
		if fn.Recv != nil {
			var recv bytes.Buffer
			if err := printer.Fprint(&recv, fset, fn.Recv.List[0].Type); err != nil {
				return nil, err
			}
			name = recv.String() + "." + name
		}
		result = append(result, function{path + ":" + name, path, fset.Position(fn.Pos()).Line, complexity(fn.Body)})
	}
	return result, nil
}

func collect() ([]function, error) {
	files, err := exec.Command("git", "ls-files", "-z", "--", "*.go").Output()
	if err != nil {
		return nil, err
	}
	result := []function{}
	for _, path := range strings.Split(string(files), "\x00") {
		if path == "" || strings.HasSuffix(path, "_test.go") || (!strings.HasPrefix(path, "internal/") && !strings.HasPrefix(path, "cmd/")) {
			continue
		}
		info, err := os.Lstat(filepath.FromSlash(path))
		if err != nil {
			return nil, err
		}
		if !info.Mode().IsRegular() {
			return nil, fmt.Errorf("non-regular Go source: %s", path)
		}
		source, err := os.ReadFile(filepath.FromSlash(path))
		if err != nil {
			return nil, err
		}
		rows, err := functions(path, source)
		if err != nil {
			return nil, err
		}
		result = append(result, rows...)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result, nil
}

func main() {
	rows, err := collect()
	if err == nil {
		err = json.NewEncoder(os.Stdout).Encode(rows)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
