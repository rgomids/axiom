package local

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"

	"github.com/rgomids/axiom/internal/runtimeprofile"
)

func authoringConfig(revision uint64) runtimeprofile.Configuration {
	cfg := runtimeConfiguration(revision)
	cfg.FormatVersion = runtimeprofile.FormatVersionV2
	for i := range cfg.ModelProfiles {
		cfg.ModelProfiles[i].Effort = runtimeprofile.Effort{Mode: "explicit", Value: "high"}
	}
	return cfg
}
func authoringWire(t *testing.T, revision uint64) []byte {
	t.Helper()
	wire, err := runtimeprofile.Encode(authoringConfig(revision))
	if err != nil {
		t.Fatal(err)
	}
	return wire
}
func authoringStore(t *testing.T) RuntimeProfileStore {
	t.Helper()
	store, err := NewRuntimeProfileStore(filepath.Join(t.TempDir(), "state"))
	if err != nil {
		t.Fatal(err)
	}
	return store
}
func TestRuntimeAuthoringLifecycle(t *testing.T) {
	ctx := context.Background()
	s := authoringStore(t)
	fresh, err := s.Inspect(ctx)
	if err != nil || fresh.State != "fresh" {
		t.Fatalf("%+v %v", fresh, err)
	}
	if _, err := os.Stat(s.root); !os.IsNotExist(err) {
		t.Fatal("inspection mutated filesystem")
	}
	active, err := s.Publish(ctx, fresh.Token, authoringWire(t, 1))
	if err != nil || active.State != "active-v2" {
		t.Fatalf("%+v %v", active, err)
	}
	if _, err := s.Publish(ctx, fresh.Token, authoringWire(t, 1)); !errors.Is(err, ErrConflict) {
		t.Fatalf("replay: %v", err)
	}
	digest, _ := runtimeprofile.Digest(active.Configuration)
	tomb, _ := runtimeprofile.EncodeTombstone(runtimeprofile.Tombstone{Kind: runtimeprofile.TombstoneKind, FormatVersion: 1, Revision: 2, PriorConfigurationDigest: digest})
	retired, err := s.Publish(ctx, active.Token, tomb)
	if err != nil || retired.State != "retired" {
		t.Fatalf("%+v %v", retired, err)
	}
	if _, err := s.Load(ctx); err == nil {
		t.Fatal("retired resolves")
	}
	recreated, err := s.Publish(ctx, retired.Token, authoringWire(t, 3))
	if err != nil || recreated.Revision != 3 {
		t.Fatalf("%+v %v", recreated, err)
	}
	if err := s.Save(ctx, runtimeConfiguration(4)); err == nil {
		t.Fatal("legacy writer downgraded v2")
	}
}
func TestRuntimeAuthoringIdentityConflict(t *testing.T) {
	for _, kind := range []string{"file", "ancestor", "same-revision", "fresh-parent"} {
		t.Run(kind, func(t *testing.T) {
			ctx := context.Background()
			s := authoringStore(t)
			prior, _ := s.Inspect(ctx)
			if kind != "fresh-parent" {
				var err error
				prior, err = s.Publish(ctx, prior.Token, authoringWire(t, 1))
				if err != nil {
					t.Fatal(err)
				}
			}
			target := filepath.Join(s.root, "runtime-profiles", "v1", "configuration.json")
			switch kind {
			case "file":
				wire, _ := os.ReadFile(target)
				if err := os.Rename(target, target+".old"); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(target, wire, 0600); err != nil {
					t.Fatal(err)
				}
				os.Remove(target + ".old")
			case "ancestor":
				if err := os.Rename(s.root, s.root+".old"); err != nil {
					t.Fatal(err)
				}
				os.Mkdir(s.root, 0700)
			case "same-revision":
				cfg := authoringConfig(1)
				cfg.ModelProfiles[0].Model = "changed-model"
				wire, _ := runtimeprofile.Encode(cfg)
				os.WriteFile(target, wire, 0600)
			case "fresh-parent":
				parent := filepath.Dir(s.root)
				if err := os.Rename(parent, parent+".old"); err != nil {
					t.Fatal(err)
				}
				os.Mkdir(parent, 0700)
			}
			if _, err := s.Publish(ctx, prior.Token, authoringWire(t, prior.Revision+1)); err == nil {
				t.Fatal("stale identity accepted")
			}
		})
	}
}
func TestRuntimeAuthoringConcurrentWinner(t *testing.T) {
	ctx := context.Background()
	s := authoringStore(t)
	prior, _ := s.Inspect(ctx)
	wire := authoringWire(t, 1)
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for range 2 {
		wg.Add(1)
		go func() { defer wg.Done(); _, err := s.Publish(ctx, prior.Token, wire); results <- err }()
	}
	wg.Wait()
	close(results)
	successes := 0
	for err := range results {
		if err == nil {
			successes++
		} else if !errors.Is(err, ErrConflict) {
			t.Fatal(err)
		}
	}
	if successes != 1 {
		t.Fatalf("successes %d", successes)
	}
}
func TestRuntimeAuthoringRecoveryFaults(t *testing.T) {
	for _, stage := range []FaultStage{FaultF1, FaultF2, FaultF3, FaultF4, FaultF5, FaultF6, FaultF7, FaultF8} {
		t.Run(string(stage), func(t *testing.T) {
			ctx := context.Background()
			s := authoringStore(t)
			prior, _ := s.Inspect(ctx)
			prior, err := s.Publish(ctx, prior.Token, authoringWire(t, 1))
			if err != nil {
				t.Fatal(err)
			}
			s.hooks.fault = func(observed FaultStage) error {
				if stage == observed {
					return ErrSimulatedInterruption
				}
				return nil
			}
			_, err = s.Publish(ctx, prior.Token, authoringWire(t, 2))
			if err == nil {
				t.Fatal("fault not reached")
			}
			s.hooks = publicationHooks{}
			observed, inspectErr := s.Inspect(ctx)
			if inspectErr == nil {
				if observed.Revision != 1 {
					t.Fatal("unexpected clean generation")
				}
				return
			}
			if !errors.Is(inspectErr, ErrRecoveryRequired) || observed.State != "pending" {
				t.Fatalf("%+v %v", observed, inspectErr)
			}
			plan, found, err := s.InspectRuntimeRecovery(ctx)
			if err != nil || !found {
				t.Fatalf("%+v %v", plan, err)
			}
			report, err := InspectRecovery(ctx, RecoveryRoots{State: s.root})
			if err != nil || len(report.Plans) != 1 {
				t.Fatalf("discovery %+v %v", report, err)
			}
			if plan.Action == PreservedReview {
				return
			}
			authority, err := AuthorizeRecovery(plan, plan.Digest)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := s.ApplyRuntimeRecovery(ctx, plan, authority); err != nil {
				t.Fatal(err)
			}
			recovered, err := s.Inspect(ctx)
			if err != nil || recovered.Revision < 1 || recovered.Revision > 2 {
				t.Fatalf("%+v %v", recovered, err)
			}
		})
	}
}

func TestRuntimeAuthoringPostCommitReplacement(t *testing.T) {
	ctx := context.Background()
	s := authoringStore(t)
	prior, _ := s.Inspect(ctx)
	s.hooks.afterCommit = func() {
		target := filepath.Join(s.root, "runtime-profiles", "v1", runtimeProfileObject)
		wire, _ := os.ReadFile(target)
		os.Rename(target, target+".old")
		os.WriteFile(target, wire, 0600)
		os.Remove(target + ".old")
	}
	_, err := s.Publish(ctx, prior.Token, authoringWire(t, 1))
	var publication *PublicationError
	if !errors.As(err, &publication) || !publication.EffectCommitted() {
		t.Fatalf("unconfirmed replacement reported as %v", err)
	}
}
func TestRuntimeAuthoringMigration(t *testing.T) {
	ctx := context.Background()
	s := authoringStore(t)
	if err := s.Create(ctx, runtimeConfiguration(1)); err != nil {
		t.Fatal(err)
	}
	prior, err := s.Inspect(ctx)
	if err != nil || prior.State != "active-v1" {
		t.Fatal(err)
	}
	next, err := s.Publish(ctx, prior.Token, authoringWire(t, 2))
	if err != nil || next.State != "active-v2" || next.Revision != 2 {
		t.Fatalf("%+v %v", next, err)
	}
}
func TestRuntimeAuthoringUnsafeRefusal(t *testing.T) {
	for _, kind := range []string{"empty-version", "symlink", "unknown"} {
		t.Run(kind, func(t *testing.T) {
			ctx := context.Background()
			s := authoringStore(t)
			prior, _ := s.Inspect(ctx)
			if kind == "empty-version" {
				os.MkdirAll(filepath.Join(s.root, "runtime-profiles", "v1"), 0700)
			} else {
				var err error
				prior, err = s.Publish(ctx, prior.Token, authoringWire(t, 1))
				if err != nil {
					t.Fatal(err)
				}
				target := filepath.Join(s.root, "runtime-profiles", "v1", runtimeProfileObject)
				if kind == "symlink" {
					os.Rename(target, target+".old")
					os.Symlink(target+".old", target)
				} else {
					os.WriteFile(filepath.Join(filepath.Dir(target), "unknown"), []byte("unknown"), 0600)
				}
			}
			if _, err := s.Publish(ctx, prior.Token, authoringWire(t, prior.Revision+1)); err == nil {
				t.Fatal("unsafe accepted")
			}
		})
	}
}

func TestRuntimeAuthoringProcessHelper(t *testing.T) {
	root := os.Getenv("AXIOM_RUNTIME_STORE_TEST_ROOT")
	if root == "" {
		return
	}
	store, err := NewRuntimeProfileStore(root)
	if err != nil {
		os.Exit(4)
	}
	_, err = store.Publish(context.Background(), os.Getenv("AXIOM_RUNTIME_STORE_TEST_TOKEN"), authoringWire(t, 1))
	if err == nil {
		os.Exit(0)
	}
	if errors.Is(err, ErrConflict) {
		os.Exit(3)
	}
	os.Exit(4)
}
func TestRuntimeAuthoringConcurrentProcesses(t *testing.T) {
	s := authoringStore(t)
	prior, err := s.Inspect(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	commands := make([]*exec.Cmd, 2)
	for i := range commands {
		commands[i] = exec.Command(os.Args[0], "-test.run=^TestRuntimeAuthoringProcessHelper$")
		commands[i].Env = append(os.Environ(), "AXIOM_RUNTIME_STORE_TEST_ROOT="+s.root, "AXIOM_RUNTIME_STORE_TEST_TOKEN="+prior.Token)
		if err := commands[i].Start(); err != nil {
			t.Fatal(err)
		}
	}
	wins := 0
	for _, cmd := range commands {
		err := cmd.Wait()
		if err == nil {
			wins++
			continue
		}
		var exit *exec.ExitError
		if !errors.As(err, &exit) || exit.ExitCode() != 3 {
			t.Fatalf("child: %v", err)
		}
	}
	if wins != 1 {
		t.Fatalf("winners %d", wins)
	}
	next, err := s.Inspect(context.Background())
	if err != nil || next.Revision != 1 {
		t.Fatalf("readback revision %d error %v", next.Revision, err)
	}
}

func TestRuntimeAuthoringRetireRecreateRecovery(t *testing.T) {
	for _, operation := range []string{"retire", "recreate"} {
		for _, stage := range []FaultStage{FaultF3, FaultF6} {
			t.Run(operation+string(stage), func(t *testing.T) {
				ctx := context.Background()
				s := authoringStore(t)
				prior, _ := s.Inspect(ctx)
				prior, err := s.Publish(ctx, prior.Token, authoringWire(t, 1))
				if err != nil {
					t.Fatal(err)
				}
				digest, _ := runtimeprofile.Digest(prior.Configuration)
				tomb, _ := runtimeprofile.EncodeTombstone(runtimeprofile.Tombstone{Kind: runtimeprofile.TombstoneKind, FormatVersion: 1, Revision: 2, PriorConfigurationDigest: digest})
				next := tomb
				if operation == "recreate" {
					prior, err = s.Publish(ctx, prior.Token, tomb)
					if err != nil {
						t.Fatal(err)
					}
					next = authoringWire(t, 3)
				}
				s.hooks.fault = func(at FaultStage) error {
					if at == stage {
						return ErrSimulatedInterruption
					}
					return nil
				}
				if _, err := s.Publish(ctx, prior.Token, next); err == nil {
					t.Fatal("missing fault")
				}
				s.hooks = publicationHooks{}
				plan, found, err := s.InspectRuntimeRecovery(ctx)
				if err != nil || !found || plan.Action == PreservedReview {
					t.Fatalf("recovery action %s found %v error %v", plan.Action, found, err)
				}
				authority, _ := AuthorizeRecovery(plan, plan.Digest)
				if _, err := s.ApplyRuntimeRecovery(ctx, plan, authority); err != nil {
					t.Fatal(err)
				}
				recovered, err := s.Inspect(ctx)
				if err != nil {
					t.Fatal(err)
				}
				want := prior.Revision
				if stage == FaultF6 {
					want++
				}
				if recovered.Revision != want {
					t.Fatalf("revision %d want %d", recovered.Revision, want)
				}
			})
		}
	}
}
func TestRuntimeRecoverySameBytesReplacementConflict(t *testing.T) {
	ctx := context.Background()
	s := authoringStore(t)
	prior, _ := s.Inspect(ctx)
	prior, err := s.Publish(ctx, prior.Token, authoringWire(t, 1))
	if err != nil {
		t.Fatal(err)
	}
	s.hooks.fault = func(at FaultStage) error {
		if at == FaultF3 {
			return ErrSimulatedInterruption
		}
		return nil
	}
	s.Publish(ctx, prior.Token, authoringWire(t, 2))
	s.hooks = publicationHooks{}
	plan, found, err := s.InspectRuntimeRecovery(ctx)
	if err != nil || !found {
		t.Fatal(err)
	}
	authority, _ := AuthorizeRecovery(plan, plan.Digest)
	target := filepath.Join(s.root, plan.Directory, plan.Marker)
	wire, _ := os.ReadFile(target)
	os.Rename(target, target+".old")
	os.WriteFile(target, wire, 0600)
	os.Remove(target + ".old")
	if _, err := s.ApplyRuntimeRecovery(ctx, plan, authority); !errors.Is(err, ErrConflict) {
		t.Fatalf("recovery identity replay: %v", err)
	}
	if _, err := os.Stat(target); err != nil {
		t.Fatal("marker changed on refusal")
	}
}

func TestRuntimeProfileInventoryInvalidSupportedV2IsMalformed(t *testing.T) {
	ctx := context.Background()
	store := authoringStore(t)
	fresh, err := store.Inspect(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Publish(ctx, fresh.Token, authoringWire(t, 1)); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(store.root, "runtime-profiles", "v1", "configuration.json")
	for _, test := range []struct {
		wire string
		kind InventoryKind
	}{
		{`{"formatVersion":2,"revision":1,"runtimes":[]}`, InventoryMalformed},
		{`{"formatVersion":3,"revision":1}`, InventoryNewer},
	} {
		if err := os.WriteFile(path, []byte(test.wire), 0600); err != nil {
			t.Fatal(err)
		}
		if kind := kindOf(t, store.root, "runtime-profiles/v1/configuration.json"); kind != test.kind {
			t.Fatalf("classification %s, expected %s", kind, test.kind)
		}
	}
}
