package local

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

const leaseParent = "00000000-0000-4000-8000-000000000001"

func TestRunLeaseAcquireProbeRestart(t *testing.T) {
	ctx := context.Background()
	root := privateTestRoot(t)
	s, err := NewRunLeaseStore(root)
	if err != nil {
		t.Fatal(err)
	}
	held, err := s.Probe(ctx, "project-1", leaseParent)
	if err != nil || held {
		t.Fatal(held, err)
	}
	first, err := s.Acquire(ctx, "project-1", leaseParent, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer first.Close()
	if first.Epoch != 1 {
		t.Fatal(first.Epoch)
	}
	done := make(chan error, 1)
	go func() {
		held, err := s.Probe(ctx, "project-1", leaseParent)
		if err == nil && !held {
			err = errors.New("lease appeared free")
		}
		done <- err
	}()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("probe blocked")
	}
	if _, err := s.Acquire(ctx, "project-1", leaseParent, 0); !errors.Is(err, ErrConflict) {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if l, err := s.Acquire(ctx, "project-1", leaseParent, 0); !errors.Is(err, ErrConflict) {
				if l != nil {
					l.Close()
				}
				t.Errorf("concurrent acquire: %v", err)
			}
		}()
	}
	wg.Wait()
	if err := first.Close(); err != nil {
		t.Fatal(err)
	}
	restarted, err := NewRunLeaseStore(root)
	if err != nil {
		t.Fatal(err)
	}
	second, err := restarted.Acquire(ctx, "project-1", leaseParent, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer second.Close()
	if second.Epoch != 2 {
		t.Fatal(second.Epoch)
	}
	if err := first.Validate(); !errors.Is(err, ErrConflict) {
		t.Fatal("closed lease validated", err)
	}
}

func TestRunLeaseReplacementAndEpochFencing(t *testing.T) {
	for _, mode := range []string{"rename directory", "symlink directory", "replace epoch", "symlink epoch", "lower epoch"} {
		t.Run(mode, func(t *testing.T) {
			ctx := context.Background()
			root := privateTestRoot(t)
			s, err := NewRunLeaseStore(root)
			if err != nil {
				t.Fatal(err)
			}
			l, err := s.Acquire(ctx, "project-1", leaseParent, 7)
			if err != nil {
				t.Fatal(err)
			}
			defer l.Close()
			dir := filepath.Join(root, "run-leases", "v1", "project-1", graphName(leaseParent))
			file := filepath.Join(dir, "epoch.json")
			switch mode {
			case "rename directory", "symlink directory":
				if err := os.Rename(dir, dir+"-old"); err != nil {
					t.Fatal(err)
				}
				if mode == "rename directory" {
					if err := os.Mkdir(dir, 0o700); err != nil {
						t.Fatal(err)
					}
				} else {
					if err := os.Symlink(dir+"-old", dir); err != nil {
						t.Skip(err)
					}
				}
			case "replace epoch", "symlink epoch":
				if err := os.Rename(file, file+"-old"); err != nil {
					t.Fatal(err)
				}
				if mode == "replace epoch" {
					if err := os.WriteFile(file, []byte("{}"), 0o600); err != nil {
						t.Fatal(err)
					}
				} else {
					if err := os.Symlink("epoch.json-old", file); err != nil {
						t.Skip(err)
					}
				}
			case "lower epoch":
				wire := []byte(`{"formatVersion":1,"projectId":"project-1","parentId":"` + leaseParent + `","epoch":7}`)
				if err := os.WriteFile(file, wire, 0o600); err != nil {
					t.Fatal(err)
				}
			}
			if err := l.Validate(); err == nil {
				t.Fatal("replaced/lowered lease validated")
			}
		})
	}
}

func TestRunLeasePublicationFaults(t *testing.T) {
	for _, stage := range []FaultStage{FaultF0, FaultF1, FaultF2, FaultF3, FaultF4, FaultF5, FaultF6, FaultF7, FaultF8} {
		t.Run(string(stage), func(t *testing.T) {
			ctx := context.Background()
			root := privateTestRoot(t)
			s, err := NewRunLeaseStore(root)
			if err != nil {
				t.Fatal(err)
			}
			s.hooks.fault = func(at FaultStage) error {
				if at == stage {
					return errors.New("injected")
				}
				return nil
			}
			l, err := s.Acquire(ctx, "project-1", leaseParent, 0)
			if l != nil {
				l.Close()
			}
			if err == nil {
				t.Fatal("fault not reported")
			}
			s.hooks = publicationHooks{}
			// A reported fault may be precommit or committed with residual recovery
			// state. It must never leave an active lease or accept a torn epoch.
			l, err = s.Acquire(ctx, "project-1", leaseParent, 0)
			if err == nil {
				if l.Epoch == 0 {
					t.Fatal("zero epoch")
				}
				l.Close()
			} else if !errors.Is(err, ErrRecoveryRequired) {
				t.Fatal(err)
			}
		})
	}
}

// Exercise the OS primitive independently of private storage ancestry checks.
// This does not change or bypass those checks in any store.
func TestRunLeaseOSLockContention(t *testing.T) {
	root, err := os.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	if err := os.Chmod(root.Name(), 0o700); err != nil {
		t.Fatal(err)
	}
	first, err := lockDirectory(root, true)
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			start := time.Now()
			other, err := lockDirectory(root, true)
			if other != nil {
				other.Close()
			}
			if !errors.Is(err, ErrConflict) {
				t.Errorf("second holder: %v", err)
			}
			if time.Since(start) > time.Second {
				t.Error("lock blocked")
			}
		}()
	}
	wg.Wait()
	first.Close()
	second, err := lockDirectory(root, true)
	if err != nil {
		t.Fatal(err)
	}
	second.Close()
}

func TestRunLeaseOSLockCrossProcess(t *testing.T) {
	if os.Getenv("AXIOM_TEST_LEASE_ROOT") != "" {
		t.Fatal("unexpected helper environment")
	}
	path := t.TempDir()
	if err := os.Chmod(path, 0o700); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(path)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	lock, err := lockDirectory(root, true)
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Close()
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, executable, "-test.run=^TestRunLeaseLockProcessHelper$")
	cmd.Env = append(os.Environ(), "AXIOM_TEST_LEASE_ROOT="+path)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("second process: %v %s", err, out)
	}
}

func TestRunLeaseLockProcessHelper(t *testing.T) {
	path := os.Getenv("AXIOM_TEST_LEASE_ROOT")
	if path == "" {
		t.Skip("subprocess helper")
	}
	root, err := os.OpenRoot(path)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	lock, err := lockDirectory(root, true)
	if lock != nil {
		lock.Close()
	}
	if !errors.Is(err, ErrConflict) {
		t.Fatal("second process acquired lease", err)
	}
}

func TestRunLeaseRecoveryRecognizesInterruptedEpoch(t *testing.T) {
	ctx := context.Background()
	root := privateTestRoot(t)
	s, err := NewRunLeaseStore(root)
	if err != nil {
		t.Fatal(err)
	}
	s.hooks.fault = func(stage FaultStage) error {
		if stage == FaultF6 {
			return ErrSimulatedInterruption
		}
		return nil
	}
	if lease, err := s.Acquire(ctx, "project-1", leaseParent, 0); err == nil {
		lease.Close()
		t.Fatal("interruption ignored")
	}
	report, err := InspectRecovery(ctx, RecoveryRoots{State: root})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, p := range report.Plans {
		if p.Directory == "run-leases/v1/project-1/"+graphName(leaseParent) {
			found = true
			if p.Action != FinalizeCommitted {
				t.Fatal(p.Action)
			}
		}
	}
	if !found {
		t.Fatalf("epoch protocol absent: %+v", report)
	}
}
