package local

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/rgomids/axiom/internal/capabilityproof"
	"github.com/rgomids/axiom/internal/executiongraph"
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
	effects := &ExtendedRootEffects{ExtendedRootObserver: observer, Run: func(ctx context.Context, r capabilityproof.ExtendedRoot, name string) error { return run(ctx, r, name) }}
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
		effects := s.Effects.(*ExtendedRootEffects)
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
	effects := s.Effects.(*ExtendedRootEffects)
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

func TestExtendedRootEffectsPreservePreexistingEmptyControlFile(t *testing.T) {
	s, root, _ := extendedEffectsFixture(t)
	p, err := s.Preview(context.Background(), "fake-probe", []string{"tmpdir"})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, p.Filename)
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	effects := s.Effects.(*ExtendedRootEffects)
	if _, err := effects.Control(context.Background(), p.Roots[0], p.Filename); !errors.Is(err, capabilityproof.ErrProbeResidue) {
		t.Fatal("preexisting file was accepted as a control", err)
	}
	if err := effects.RemoveVerify(context.Background(), p.Roots[0], p.Filename); !errors.Is(err, capabilityproof.ErrProbeResidue) {
		t.Fatal("unobserved control file was eligible for cleanup", err)
	}
	info, err := os.Lstat(path)
	if err != nil || !ownedProbeFile(info) {
		t.Fatal("preexisting empty file was changed", err)
	}
}

func TestExtendedRootEffectsPreserveReplacementAfterCanary(t *testing.T) {
	s, root, run := extendedEffectsFixture(t)
	p, err := s.Preview(context.Background(), "fake-probe", []string{"tmpdir"})
	if err != nil {
		t.Fatal(err)
	}
	*run = func(_ context.Context, r capabilityproof.ExtendedRoot, name string) error {
		return os.WriteFile(filepath.Join(r.Path, name), nil, 0o600)
	}
	effects := s.Effects.(*ExtendedRootEffects)
	if _, _, err := effects.Canary(context.Background(), p.Roots[0], p.Filename); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, p.Filename)
	replacement := path + ".replacement"
	if err := os.Rename(path, replacement); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := effects.RemoveVerify(context.Background(), p.Roots[0], p.Filename); !errors.Is(err, capabilityproof.ErrProbeResidue) {
		t.Fatal("replacement file was treated as the canary object", err)
	}
	if _, err := os.Lstat(path); err != nil {
		t.Fatal("replacement file was removed", err)
	}
	if _, err := os.Lstat(replacement); err != nil {
		t.Fatal("original canary object was unexpectedly removed", err)
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

func TestExtendedRootPermitBindsExactRootAndCleansWithEnvelope(t *testing.T) {
	e := envelopeFixture(t)
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("TMPDIR", root)
	service, err := ExtendedProbeService([]string{"tmpdir"})
	if err != nil {
		t.Fatal(err)
	}
	preview, err := service.Preview(context.Background(), e.manifest.ProbeID, []string{"tmpdir"})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, preview.Filename)
	cwd := filepath.Join(e.Path(), "workspace")
	if err := CreateExtendedCanary(cwd, e.manifest.ProbeID, e.digest, path); err == nil {
		t.Fatal("unchanged prepared envelope authorized a canary without a permit")
	}
	if entries, err := os.ReadDir(root); err != nil || len(entries) != 0 {
		t.Fatal("no-permit refusal changed the selected root", err)
	}
	if _, err := os.Lstat(filepath.Join(e.Path(), extendedRootPermitName)); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("no-permit refusal changed the prepared envelope", err)
	}
	session := openProbeSession(t, e)
	if err := BindExtendedRootPermit(context.Background(), session, preview); err != nil {
		t.Fatal(err)
	}
	if err := CreateExtendedCanary(cwd, e.manifest.ProbeID, e.digest, path); err != nil {
		t.Fatal(err)
	}
	info, err := os.Lstat(path)
	if err != nil || !ownedProbeFile(info) {
		t.Fatal("authorized root was not written", err)
	}
	alternate := filepath.Join("/tmp", preview.Filename)
	if alternate != path {
		if err := CreateExtendedCanary(cwd, e.manifest.ProbeID, e.digest, alternate); !errors.Is(err, capabilityproof.ErrExtendedRoot) {
			t.Fatal("unreviewed root was accepted", err)
		}
		if _, err := os.Lstat(alternate); !errors.Is(err, os.ErrNotExist) {
			t.Fatal("unreviewed root was changed", err)
		}
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("foreign"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := session.Cleanup(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(e.Path()); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("cleanup did not remove the permit envelope", err)
	}
	if wire, err := os.ReadFile(path); err != nil || string(wire) != "foreign" {
		t.Fatal("envelope cleanup removed a foreign root file", err)
	}
}

func TestExtendedRootPermitRejectsSystemTempAlternativeAndTampering(t *testing.T) {
	t.Run("system-temp refuses distinct TMPDIR", func(t *testing.T) {
		e := envelopeFixture(t)
		temp, err := filepath.EvalSymlinks(t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		t.Setenv("TMPDIR", temp)
		service, err := ExtendedProbeService([]string{"system-temp"})
		if err != nil {
			t.Fatal(err)
		}
		preview, err := service.Preview(context.Background(), e.manifest.ProbeID, []string{"system-temp"})
		if err != nil {
			t.Fatal(err)
		}
		session := openProbeSession(t, e)
		if err := BindExtendedRootPermit(context.Background(), session, preview); err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(temp, preview.Filename)
		if path == filepath.Join("/tmp", preview.Filename) {
			t.Skip("TMPDIR resolves to system-temp")
		}
		if err := CreateExtendedCanary(filepath.Join(e.Path(), "workspace"), e.manifest.ProbeID, e.digest, path); !errors.Is(err, capabilityproof.ErrExtendedRoot) {
			t.Fatal("unselected TMPDIR was accepted for system-temp", err)
		}
		if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
			t.Fatal("unselected TMPDIR was changed", err)
		}
	})
	t.Run("tampered snapshot is refused", func(t *testing.T) {
		e := envelopeFixture(t)
		root, err := filepath.EvalSymlinks(t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		t.Setenv("TMPDIR", root)
		service, err := ExtendedProbeService([]string{"tmpdir"})
		if err != nil {
			t.Fatal(err)
		}
		preview, err := service.Preview(context.Background(), e.manifest.ProbeID, []string{"tmpdir"})
		if err != nil {
			t.Fatal(err)
		}
		session := openProbeSession(t, e)
		if err := BindExtendedRootPermit(context.Background(), session, preview); err != nil {
			t.Fatal(err)
		}
		permitPath := filepath.Join(e.Path(), extendedRootPermitName)
		wire, err := os.ReadFile(permitPath)
		var permit extendedRootPermit
		if err != nil || json.Unmarshal(wire, &permit) != nil {
			t.Fatal("permit unavailable", err)
		}
		permit.Preview.Roots[0].Snapshot.Inode++
		wire, err = json.Marshal(permit)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(permitPath, wire, 0o600); err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(root, preview.Filename)
		if err := CreateExtendedCanary(filepath.Join(e.Path(), "workspace"), e.manifest.ProbeID, e.digest, path); err == nil {
			t.Fatal("tampered root snapshot was accepted")
		}
		if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
			t.Fatal("tampered permit changed the root", err)
		}
	})
}

func TestExtendedRootPermitCannotBindClosedOrQuarantinedSession(t *testing.T) {
	for _, state := range []string{"closed", "quarantined"} {
		t.Run(state, func(t *testing.T) {
			e := envelopeFixture(t)
			root, err := filepath.EvalSymlinks(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			t.Setenv("TMPDIR", root)
			service, err := ExtendedProbeService([]string{"tmpdir"})
			if err != nil {
				t.Fatal(err)
			}
			preview, err := service.Preview(context.Background(), e.manifest.ProbeID, []string{"tmpdir"})
			if err != nil {
				t.Fatal(err)
			}
			session := openProbeSession(t, e)
			if state == "closed" {
				err = session.Cleanup(context.Background())
			} else {
				err = session.Quarantine(context.Background())
			}
			if err != nil {
				t.Fatal(err)
			}
			if err := BindExtendedRootPermit(context.Background(), session, preview); err == nil {
				t.Fatal("closed session acquired an extended permit")
			}
			if _, err := os.Lstat(filepath.Join(e.Path(), extendedRootPermitName)); !errors.Is(err, os.ErrNotExist) {
				t.Fatal("closed session wrote a permit", err)
			}
		})
	}
}

func TestExtendedRootPermitRejectsWrongVariantWorkingDirectory(t *testing.T) {
	e := envelopeVariantFixture(t, "read-only")
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("TMPDIR", root)
	service, err := ExtendedProbeService([]string{"tmpdir"})
	if err != nil {
		t.Fatal(err)
	}
	preview, err := service.Preview(context.Background(), e.manifest.ProbeID, []string{"tmpdir"})
	if err != nil {
		t.Fatal(err)
	}
	session := openProbeSession(t, e)
	if err := BindExtendedRootPermit(context.Background(), session, preview); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, preview.Filename)
	if err := CreateExtendedCanary(filepath.Join(e.Path(), "workspace"), e.manifest.ProbeID, e.digest, path); !errors.Is(err, capabilityproof.ErrCanaryContext) {
		t.Fatal("workspace cwd accepted for read-only envelope", err)
	}
	if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("wrong-variant cwd changed the selected root", err)
	}
}

func TestExtendedRootPermitRequiresLiveOwnerAndUnexpiredDeadline(t *testing.T) {
	for _, invalid := range []string{"expired", "future", "owner-drift", "missing-owner"} {
		t.Run(invalid, func(t *testing.T) {
			e := envelopeFixture(t)
			root, err := filepath.EvalSymlinks(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			t.Setenv("TMPDIR", root)
			service, err := ExtendedProbeService([]string{"tmpdir"})
			if err != nil {
				t.Fatal(err)
			}
			preview, err := service.Preview(context.Background(), e.manifest.ProbeID, []string{"tmpdir"})
			if err != nil {
				t.Fatal(err)
			}
			session := openProbeSession(t, e)
			if err := BindExtendedRootPermit(context.Background(), session, preview); err != nil {
				t.Fatal(err)
			}
			permitPath := filepath.Join(e.Path(), extendedRootPermitName)
			wire, err := os.ReadFile(permitPath)
			var permit extendedRootPermit
			if err != nil || json.Unmarshal(wire, &permit) != nil {
				t.Fatal("permit unavailable", err)
			}
			switch invalid {
			case "expired":
				permit.IssuedAt = time.Unix(1, 0).UTC()
				permit.ExpiresAt = permit.IssuedAt.Add(permit.Preview.Timeout)
			case "future":
				permit.IssuedAt = time.Now().UTC().Add(time.Hour)
				permit.ExpiresAt = permit.IssuedAt.Add(permit.Preview.Timeout)
			case "owner-drift":
				permit.Owner.StartIdentity += "-changed"
			case "missing-owner":
				permit.Owner = executiongraph.ProcessCorrelation{}
			}
			wire, err = json.Marshal(permit)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(permitPath, wire, 0o600); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(root, preview.Filename)
			if err := CreateExtendedCanary(filepath.Join(e.Path(), "workspace"), e.manifest.ProbeID, e.digest, path); err == nil {
				t.Fatal("invalid owner or deadline authorized an extended canary")
			}
			if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
				t.Fatal("invalid permit changed the root", err)
			}
		})
	}
}
