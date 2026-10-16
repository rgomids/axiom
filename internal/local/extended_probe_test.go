package local

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/rgomids/axiom/internal/capabilityproof"
)

func TestExtendedRootObserverReadOnlyAndUnavailableEffects(t *testing.T) {
	base, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(base, "fake-temp")
	if err := os.Mkdir(root, 0700); err != nil {
		t.Fatal(err)
	}
	observer := ExtendedRootObserver{TempDir: root}
	s := capabilityproof.ExtendedService{Policy: capabilityproof.ExtendedPolicy{TempDir: root, Home: filepath.Join(base, "fake-home")}, Inspector: observer}
	ctx := context.Background()
	p, err := s.Preview(ctx, "fake-probe", []string{"tmpdir"})
	if err != nil || p.Digest == "" || !p.Roots[0].Snapshot.SymlinkFree || p.Roots[0].Snapshot.ResolvedTarget != "tmpdir" {
		t.Fatalf("preview=%+v error=%v", p, err)
	}
	if fresh, err := s.Preview(ctx, "fake-probe", []string{"tmpdir"}); err != nil || !reflect.DeepEqual(fresh, p) {
		t.Fatal("non-deterministic read-only observation", err)
	}
	a := capabilityproof.ExtendedAuthority{Actor: "synthetic-operator", Preview: p, Local: true, External: true}
	if _, err := s.Apply(ctx, p, a); !errors.Is(err, capabilityproof.ErrBlocked) {
		t.Fatal("unavailable effects accepted", err)
	}
	entries, err := os.ReadDir(root)
	if err != nil || len(entries) != 0 {
		t.Fatal("preview or unavailable-effect refusal wrote files", err)
	}
	if err := os.Rename(root, root+"-prior"); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(root, 0700); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Apply(ctx, p, a); !errors.Is(err, capabilityproof.ErrStalePreview) {
		t.Fatal("root identity drift accepted", err)
	}
}

func TestExtendedRootObserverRefusals(t *testing.T) {
	base, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	root, link := filepath.Join(base, "fake-temp"), filepath.Join(base, "alias")
	if err := os.Mkdir(root, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(root, link); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{link, "/", "relative", filepath.Join(base, "missing")} {
		_, err := (ExtendedRootObserver{TempDir: path}).Inspect(context.Background(), path)
		if err == nil {
			t.Fatal("unsafe root accepted", path)
		}
	}
	if _, err := (ExtendedRootObserver{TempDir: root}).Inspect(context.Background(), base); !errors.Is(err, capabilityproof.ErrExtendedRoot) {
		t.Fatal("foreign root accepted", err)
	}
	if err := os.Mkdir(filepath.Join(root, ".git"), 0700); err != nil {
		t.Fatal(err)
	}
	if _, err := (ExtendedRootObserver{TempDir: root}).Inspect(context.Background(), root); !errors.Is(err, capabilityproof.ErrExtendedRoot) {
		t.Fatal("repository root accepted", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := (ExtendedRootObserver{TempDir: root}).Inspect(ctx, root); !errors.Is(err, context.Canceled) {
		t.Fatal("cancelled inspection accepted", err)
	}
}
