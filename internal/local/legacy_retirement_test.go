package local

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"
)

// Historical recognition must not grow a second executor or effect writer.
func TestLegacyLifecycleHasNoProductionMutators(t *testing.T) {
	retired := map[string][]string{
		"local":          {"RecordDeliveryPacket", "SaveDeliveryPublication", "RecordDeliveryDecision", "ApplyDeliveryRework", "ExecutionDigest"},
		"workflow":       {"ComposeDeliveryPacket", "PrepareDeliveryDecision", "ApplyDeliveryDecision", "PreviewDeliveryRework", "ApplyDeliveryRework"},
		"executiongraph": {"BuildEvidence", "PrepareRunEnvelope", "BuildStageOutcome", "CheckSingleStageDispatch", "CheckSingleStageRetry", "AppendSingleStageReconciliation", "AppendReconciliationFact"},
		"runtimeadapter": {"NewInvocationResolver", "BuildFrozenStageInvocation"},
		"gitworkspace":   {"NewManager", "NewSingleManager", "NewCombinedValidator"},
	}
	for pkg, names := range retired {
		paths, err := filepath.Glob(filepath.Join("..", pkg, "*.go"))
		if err != nil || len(paths) == 0 {
			t.Fatalf("%s: %v", pkg, err)
		}
		for _, path := range paths {
			if strings.HasSuffix(path, "_test.go") {
				continue
			}
			f, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
			if err != nil {
				t.Fatal(err)
			}
			for _, decl := range f.Decls {
				fn, ok := decl.(*ast.FuncDecl)
				if !ok {
					continue
				}
				for _, name := range names {
					if fn.Name.Name == name {
						t.Errorf("retired production API restored: %s:%s", path, name)
					}
				}
				if fn.Recv != nil && pkg == "workflow" {
					switch fn.Name.Name {
					case "Start", "Transition", "RecordLifecycleFact":
						t.Errorf("legacy lifecycle restored: %s:%s", path, fn.Name.Name)
					}
				}
			}
		}
	}
}
