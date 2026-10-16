package local

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
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

func extendedEffectsFixture(t *testing.T) (capabilityproof.ExtendedService, string, *func(context.Context, capabilityproof.ExtendedRoot, string) error) {
	t.Helper()
	base, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(base, "fake-temp")
	if err := os.Mkdir(root, 0o700); err != nil {
		t.Fatal(err)
	}
	run := func(context.Context, capabilityproof.ExtendedRoot, string) error { return nil }
	observer := ExtendedRootObserver{TempDir: root}
	effects := ExtendedRootEffects{ExtendedRootObserver: observer, Run: func(ctx context.Context, r capabilityproof.ExtendedRoot, name string) error { return run(ctx, r, name) }}
	return capabilityproof.ExtendedService{Policy: capabilityproof.ExtendedPolicy{TempDir: root, Home: filepath.Join(base, "fake-home")}, Inspector: observer, Effects: effects}, root, &run
}

func applyExtended(t *testing.T, s capabilityproof.ExtendedService) ([]capabilityproof.ExtendedResult, error) {
	t.Helper()
	p, err := s.Preview(context.Background(), "fake-probe", []string{"tmpdir"})
	if err != nil {
		t.Fatal(err)
	}
	return s.Apply(context.Background(), p, capabilityproof.ExtendedAuthority{Actor: "synthetic-operator", Local: true, External: true, Preview: p})
}

func TestExtendedRootEffectsObserveOnlyOnTheHost(t *testing.T) {
	t.Run("created by the child is advisory", func(t *testing.T) {
		s, root, run := extendedEffectsFixture(t)
		calls := 0
		*run = func(_ context.Context, r capabilityproof.ExtendedRoot, name string) error {
			calls++
			if r.Path != root || name != ".axiom-probe-fake-probe" {
				t.Errorf("canary target %s %s", r.Path, name)
			}
			return os.WriteFile(filepath.Join(r.Path, name), nil, 0o600)
		}
		results, err := applyExtended(t, s)
		if err != nil || len(results) != 1 || results[0].State != "advisory" || !results[0].Measurement.Canary.Created || results[0].Measurement.Identity.UID != uint64(os.Geteuid()) || calls != 1 {
			t.Fatalf("results=%+v err=%v", results, err)
		}
		if entries, _ := os.ReadDir(root); len(entries) != 0 {
			t.Fatal("canary residue survived", entries)
		}
	})
	t.Run("absent file stays unknown", func(t *testing.T) {
		s, root, _ := extendedEffectsFixture(t)
		results, err := applyExtended(t, s)
		if err != nil || len(results) != 1 || results[0].State != "unknown" || results[0].Measurement.Canary.Created || results[0].Measurement.Canary.Errno != "" {
			t.Fatalf("results=%+v err=%v", results, err)
		}
		if !results[0].Measurement.Pre.RemovalVerified || !results[0].Measurement.Post.RemovalVerified {
			t.Fatal("controls not verified", results[0].Measurement)
		}
		if entries, _ := os.ReadDir(root); len(entries) != 0 {
			t.Fatal("control residue survived", entries)
		}
	})
	for name, plant := range map[string]func(string) error{
		"directory": func(path string) error { return os.Mkdir(path, 0o700) },
		"symlink":   func(path string) error { return os.Symlink("/etc/passwd", path) },
		"content":   func(path string) error { return os.WriteFile(path, []byte("data"), 0o600) },
		"hard link": func(path string) error {
			other := filepath.Join(filepath.Dir(filepath.Dir(path)), "outside")
			if err := os.WriteFile(other, nil, 0o600); err != nil {
				return err
			}
			return os.Link(other, path)
		},
	} {
		t.Run("foreign "+name+" is residue and preserved", func(t *testing.T) {
			s, root, run := extendedEffectsFixture(t)
			*run = func(_ context.Context, r capabilityproof.ExtendedRoot, n string) error {
				return plant(filepath.Join(r.Path, n))
			}
			if _, err := applyExtended(t, s); !errors.Is(err, capabilityproof.ErrProbeResidue) {
				t.Fatal(err)
			}
			if _, err := os.Lstat(filepath.Join(root, ".axiom-probe-fake-probe")); err != nil {
				t.Fatal("non-owned object was removed", err)
			}
		})
	}
	t.Run("pre-existing name is never touched", func(t *testing.T) {
		s, root, run := extendedEffectsFixture(t)
		path := filepath.Join(root, ".axiom-probe-fake-probe")
		if err := os.WriteFile(path, []byte("operator"), 0o600); err != nil {
			t.Fatal(err)
		}
		*run = func(context.Context, capabilityproof.ExtendedRoot, string) error {
			t.Error("canary ran beside an existing object")
			return nil
		}
		if _, err := applyExtended(t, s); !errors.Is(err, capabilityproof.ErrProbeResidue) {
			t.Fatal(err)
		}
		if wire, err := os.ReadFile(path); err != nil || string(wire) != "operator" {
			t.Fatal("existing object changed", err)
		}
	})
	t.Run("run failure still cleans owned file", func(t *testing.T) {
		s, root, run := extendedEffectsFixture(t)
		*run = func(_ context.Context, r capabilityproof.ExtendedRoot, n string) error {
			if err := os.WriteFile(filepath.Join(r.Path, n), nil, 0o600); err != nil {
				return err
			}
			return errors.New("synthetic child failure")
		}
		results, err := applyExtended(t, s)
		if err == nil || len(results) != 1 || results[0].State != "unknown" {
			t.Fatalf("results=%+v err=%v", results, err)
		}
		if entries, _ := os.ReadDir(root); len(entries) != 0 {
			t.Fatal("owned residue survived a failed run", entries)
		}
	})
	t.Run("missing runner is blocked without residue", func(t *testing.T) {
		s, root, _ := extendedEffectsFixture(t)
		effects := s.Effects.(ExtendedRootEffects)
		effects.Run = nil
		s.Effects = effects
		if _, err := applyExtended(t, s); !errors.Is(err, capabilityproof.ErrBlocked) {
			t.Fatal(err)
		}
		if entries, _ := os.ReadDir(root); len(entries) != 0 {
			t.Fatal("blocked probe left residue", entries)
		}
	})
}

func TestExtendedRootEffectsRefuseUnreviewedTargets(t *testing.T) {
	s, root, _ := extendedEffectsFixture(t)
	p, err := s.Preview(context.Background(), "fake-probe", []string{"tmpdir"})
	if err != nil {
		t.Fatal(err)
	}
	effects := s.Effects.(ExtendedRootEffects)
	for _, name := range []string{"plain", "../.axiom-probe-x", ".axiom-probe-", "dir/.axiom-probe-x"} {
		if _, err := effects.Control(context.Background(), p.Roots[0], name); err == nil {
			t.Fatalf("name %q accepted", name)
		}
		if err := effects.RemoveVerify(context.Background(), p.Roots[0], name); err == nil {
			t.Fatalf("removal of %q accepted", name)
		}
	}
	if err := os.Rename(root, root+"-prior"); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(root, 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := effects.Control(context.Background(), p.Roots[0], p.Filename); !errors.Is(err, capabilityproof.ErrStalePreview) {
		t.Fatal("replaced root accepted", err)
	}
	if entries, _ := os.ReadDir(root); len(entries) != 0 {
		t.Fatal("replaced root changed", entries)
	}
}

func TestCreateExtendedCanaryRefusesOutsideItsEnvelope(t *testing.T) {
	base, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	digest := strings.Repeat("a", 64)
	for _, cwd := range []string{"relative", base, filepath.Join(base, "capability-probes", "other", "workspace"), filepath.Join(base, "capability-probes", "probe", "elsewhere")} {
		if err := CreateExtendedCanary(cwd, "probe", digest, "/tmp/.axiom-probe-probe"); !errors.Is(err, capabilityproof.ErrCanaryContext) {
			t.Fatalf("%s: %v", cwd, err)
		}
	}
	if _, err := os.Lstat("/tmp/.axiom-probe-probe"); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("refused canary created a file")
	}
}
