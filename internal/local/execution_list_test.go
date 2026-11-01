package local

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestCanonicalExecutionListStrictReadOnly(t *testing.T) {
	ctx := context.Background()
	s, root := newTestExecutionStore(t)
	if xs, err := s.List(ctx, "proj"); err != nil || xs == nil || len(xs) != 0 {
		t.Fatalf("empty: %+v %v", xs, err)
	}
	x := storeTestExecution(t)
	if err := s.Create(ctx, x); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(root, "executions", "v2", "proj", x.ExecutionID+".json")
	before, _ := os.ReadFile(p)
	xs, err := s.List(ctx, "proj")
	if err != nil || len(xs) != 1 || xs[0].ExecutionID != x.ExecutionID {
		t.Fatalf("list: %+v %v", xs, err)
	}
	after, _ := os.ReadFile(p)
	if string(before) != string(after) {
		t.Fatal("listing changed record")
	}
	for _, name := range []string{"unexpected", "00000000-0000-4000-8000-000000000099.json", ".axiom-recovery-x"} {
		path := filepath.Join(filepath.Dir(p), name)
		if err := os.WriteFile(path, []byte("{}"), 0600); err != nil {
			t.Fatal(err)
		}
		if got, err := s.List(ctx, "proj"); err == nil || got != nil {
			t.Fatalf("accepted %s", name)
		}
		if err := os.Remove(path); err != nil {
			t.Fatal(err)
		}
	}
}

func TestCanonicalExecutionListNeverReadsLegacyOrLinks(t *testing.T) {
	s, root := newTestExecutionStore(t)
	ctx := context.Background()
	legacy := filepath.Join(root, "executions", "v1", "proj")
	if err := os.MkdirAll(legacy, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(legacy, "legacy.json"), []byte("unreadable old bytes"), 0000); err != nil {
		t.Fatal(err)
	}
	if got, err := s.List(ctx, "proj"); err != nil || len(got) != 0 {
		t.Fatalf("read old state %+v %v", got, err)
	}
	x := storeTestExecution(t)
	if err := s.Create(ctx, x); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(root, "executions", "v2", "proj", "00000000-0000-4000-8000-000000000099.json")
	if err := os.Symlink(filepath.Join(legacy, "legacy.json"), file); err != nil {
		t.Fatal(err)
	}
	if got, err := s.List(ctx, "proj"); err == nil || got != nil {
		t.Fatalf("followed link %+v", got)
	}
}
