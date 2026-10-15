package local

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"

	"github.com/rgomids/axiom/internal/runtimeprofile"
	"github.com/rgomids/axiom/internal/runtimeprofile/runtimeprofiletest"
)

// authoringConfig is the synthetic canonical v3 record at revision.
func authoringConfig(revision uint64) runtimeprofile.ConfigurationV3 {
	return runtimeprofiletest.WithRevision(runtimeprofiletest.MultiRuntime(), revision)
}
func authoringWire(t *testing.T, revision uint64) []byte {
	t.Helper()
	return runtimeprofiletest.Encode(t, authoringConfig(revision))
}

// legacyConfig is a valid recognized pre-stable record of format 1 or 2.
func legacyConfig(format int, revision uint64) runtimeprofile.Configuration {
	cfg := runtimeConfiguration(revision)
	if format == runtimeprofile.FormatVersionV2 {
		cfg.FormatVersion = runtimeprofile.FormatVersionV2
		for i := range cfg.ModelProfiles {
			cfg.ModelProfiles[i].Effort = runtimeprofile.Effort{Mode: "explicit", Value: "high"}
		}
	}
	return cfg
}

// writeLegacyRecord places pre-stable bytes exactly as an earlier release left
// them. No current writer produces format 1/2 (Specification 007 PM-11).
func writeLegacyRecord(t *testing.T, root string, wire []byte) {
	t.Helper()
	dir := filepath.Join(root, "runtime-profiles", "v1")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	for _, d := range []string{root, filepath.Join(root, "runtime-profiles"), dir} {
		if err := os.Chmod(d, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, runtimeProfileObject), wire, 0o600); err != nil {
		t.Fatal(err)
	}
}

func legacyWire(t *testing.T, format int, revision uint64) []byte {
	t.Helper()
	wire, err := runtimeprofile.Encode(legacyConfig(format, revision))
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
	if err != nil || active.State != "active-v3" || active.V3 == nil {
		t.Fatalf("%+v %v", active, err)
	}
	if _, err := s.Publish(ctx, fresh.Token, authoringWire(t, 1)); !errors.Is(err, ErrConflict) {
		t.Fatalf("replay: %v", err)
	}
	// No format 1/2 configuration is ever published over v3.
	if _, err := s.Publish(ctx, active.Token, legacyWire(t, 2, 2)); !errors.Is(err, runtimeprofile.ErrUnsupportedConfigurationFormat) {
		t.Fatalf("v2 downgrade: %v", err)
	}
	// Retirement of the last Runtime binds DigestV3 (PM-9); the historical
	// Digest of the same record is refused.
	historical, _ := runtimeprofile.Encode(legacyConfig(2, 1))
	if decoded, err := runtimeprofile.Decode(historical); err == nil {
		wrongDigest, _ := runtimeprofile.Digest(decoded)
		wrong, _ := runtimeprofile.EncodeTombstone(runtimeprofile.Tombstone{Kind: runtimeprofile.TombstoneKind, FormatVersion: 1, Revision: 2, PriorConfigurationDigest: wrongDigest})
		if _, err := s.Publish(ctx, active.Token, wrong); !errors.Is(err, ErrConflict) {
			t.Fatalf("non-DigestV3 retirement: %v", err)
		}
	}
	digest := runtimeprofiletest.Digest(t, *active.V3)
	tomb, _ := runtimeprofile.EncodeTombstone(runtimeprofile.Tombstone{Kind: runtimeprofile.TombstoneKind, FormatVersion: 1, Revision: 2, PriorConfigurationDigest: digest})
	retired, err := s.Publish(ctx, active.Token, tomb)
	if err != nil || retired.State != "retired" || retired.V3 != nil {
		t.Fatalf("%+v %v", retired, err)
	}
	recreated, err := s.Publish(ctx, retired.Token, authoringWire(t, 3))
	if err != nil || recreated.Revision != 3 || recreated.State != "active-v3" {
		t.Fatalf("%+v %v", recreated, err)
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
				wire, _ := runtimeprofile.EncodeV3(cfg)
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

// TestRuntimeAuthoringLegacyRetirement covers the store side of PM-11 A1: a
// recognized pre-stable record is never overwritten by v3 or v1/v2, only
// retired by the existing tombstone bound to its historical Digest, after
// which canonical v3 is created explicitly.
func TestRuntimeAuthoringLegacyRetirement(t *testing.T) {
	for _, format := range []int{runtimeprofile.FormatVersion, runtimeprofile.FormatVersionV2} {
		t.Run(fmt.Sprintf("v%d", format), func(t *testing.T) {
			ctx := context.Background()
			s := authoringStore(t)
			source := legacyWire(t, format, 1)
			writeLegacyRecord(t, s.root, source)
			prior, err := s.Inspect(ctx)
			if err != nil || prior.State != fmt.Sprintf("active-v%d", format) || prior.V3 != nil || prior.Legacy.FormatVersion != format {
				t.Fatalf("%+v %v", prior, err)
			}
			// No direct v1/v2 -> v3 overwrite, no migration and no partial
			// legacy rewrite.
			if _, err := s.Publish(ctx, prior.Token, authoringWire(t, 2)); !errors.Is(err, runtimeprofile.ErrUnsupportedConfigurationFormat) {
				t.Fatalf("v3 over legacy: %v", err)
			}
			if _, err := s.Publish(ctx, prior.Token, legacyWire(t, 2, 2)); !errors.Is(err, runtimeprofile.ErrUnsupportedConfigurationFormat) {
				t.Fatalf("legacy rewrite: %v", err)
			}
			historical, _ := runtimeprofile.Digest(prior.Legacy)
			for _, wrong := range []runtimeprofile.Tombstone{
				{Kind: runtimeprofile.TombstoneKind, FormatVersion: 1, Revision: 3, PriorConfigurationDigest: historical},
				{Kind: runtimeprofile.TombstoneKind, FormatVersion: 1, Revision: 2, PriorConfigurationDigest: runtimeprofiletest.Digest(t, authoringConfig(1))},
			} {
				wire, _ := runtimeprofile.EncodeTombstone(wrong)
				if _, err := s.Publish(ctx, prior.Token, wire); !errors.Is(err, ErrConflict) {
					t.Fatalf("mismatched tombstone %+v: %v", wrong, err)
				}
			}
			if current, _ := os.ReadFile(filepath.Join(s.root, "runtime-profiles", "v1", runtimeProfileObject)); !bytes.Equal(current, source) {
				t.Fatal("refusal changed the pre-stable source bytes")
			}
			tomb, _ := runtimeprofile.EncodeTombstone(runtimeprofile.Tombstone{Kind: runtimeprofile.TombstoneKind, FormatVersion: 1, Revision: 2, PriorConfigurationDigest: historical})
			retired, err := s.Publish(ctx, prior.Token, tomb)
			if err != nil || retired.State != "retired" || retired.Revision != 2 {
				t.Fatalf("%+v %v", retired, err)
			}
			if decoded, err := runtimeprofile.DecodeTombstone(tomb); err != nil || decoded.FormatVersion != 1 {
				t.Fatal("tombstone format changed")
			}
			created, err := s.Publish(ctx, retired.Token, authoringWire(t, 3))
			if err != nil || created.State != "active-v3" || created.Revision != 3 {
				t.Fatalf("%+v %v", created, err)
			}
		})
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
				digest := runtimeprofiletest.Digest(t, *prior.V3)
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
		{`{"formatVersion":1,"revision":1}`, InventoryMalformed},
		{`{"formatVersion":3,"revision":1}`, InventoryMalformed},
		{`{"formatVersion":4,"revision":1}`, InventoryNewer},
		{`{"formatVersion":0,"revision":1}`, InventoryOlder},
		{string(legacyWire(t, 1, 1)), InventoryRuntimeProfilePreStable},
		{string(legacyWire(t, 2, 1)), InventoryRuntimeProfilePreStable},
		{string(authoringWire(t, 1)), InventoryRuntimeProfile},
	} {
		if err := os.WriteFile(path, []byte(test.wire), 0600); err != nil {
			t.Fatal(err)
		}
		if kind := kindOf(t, store.root, "runtime-profiles/v1/configuration.json"); kind != test.kind {
			t.Fatalf("classification %s, expected %s", kind, test.kind)
		}
	}
}
