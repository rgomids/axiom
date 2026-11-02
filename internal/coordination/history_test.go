package coordination

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"testing"
)

// Compatibility recognition may survive; an operational writer/encoder may not.
func TestHistoryExportsAreReadOnly(t *testing.T) {
	f, err := parser.ParseFile(token.NewFileSet(), "coordination.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range f.Decls {
		if fn, ok := d.(*ast.FuncDecl); ok && fn.Name.IsExported() {
			switch fn.Name.Name {
			case "DecodeStream", "ValidRecord", "ValidStream":
			default:
				t.Errorf("operational API survived: %s", fn.Name.Name)
			}
		}
	}
}

func TestFrozenHistoryRejectsTampering(t *testing.T) {
	paths, err := filepath.Glob("../compatibility/testdata/stable-v1/snapshots/*/state/coordination/v1/*/*.json")
	if err != nil || len(paths) == 0 {
		t.Fatalf("fixtures: %v", err)
	}
	for _, p := range paths {
		wire, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		stream, err := DecodeStream(wire)
		if err != nil {
			t.Fatalf("%s: %v", p, err)
		}
		altered := bytes.Replace(wire, []byte(stream.Records[0].Digest), bytes.Repeat([]byte("0"), 64), 1)
		if _, err := DecodeStream(altered); err == nil {
			t.Fatalf("tampered history accepted: %s", p)
		}
	}
}
