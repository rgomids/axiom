package local

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestPortableLockSerializesProcessesAndSurvivesCrash(t *testing.T) {
	root := privateTestRoot(t)
	store, err := NewPortableStore(root)
	if err != nil {
		t.Fatal(err)
	}
	command := exec.Command(os.Args[0], "-test.run=^TestPortableProcessHelper$")
	command.Env = append(os.Environ(), "AXIOM_POC_HELPER=hold", "AXIOM_POC_ROOT="+root)
	output, err := command.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = command.Process.Kill()
		_ = command.Wait()
	}()
	line, err := bufio.NewReader(output).ReadString('\n')
	if err != nil || strings.TrimSpace(line) != "locked" {
		t.Fatalf("helper did not acquire lock: %q, %v", line, err)
	}
	if err := store.Create(context.Background(), "sample", []byte("first")); !errors.Is(err, ErrConflict) {
		t.Fatalf("concurrent create = %v", err)
	}
	if err := command.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	_ = command.Wait()
	if err := store.Create(context.Background(), "sample", []byte("first")); err != nil {
		t.Fatalf("create after process death = %v", err)
	}
	if body, err := store.Read(context.Background(), "sample"); err != nil || string(body) != "first" {
		t.Fatalf("published bytes = %q, %v", body, err)
	}
}

func TestPortableReadersAndWritersConflictWithOtherProcess(t *testing.T) {
	root := privateTestRoot(t)
	store, err := NewPortableStore(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Create(context.Background(), "sample", []byte("old")); err != nil {
		t.Fatal(err)
	}
	command := exec.Command(os.Args[0], "-test.run=^TestPortableProcessHelper$")
	command.Env = append(os.Environ(), "AXIOM_POC_HELPER=hold", "AXIOM_POC_ROOT="+root)
	output, err := command.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = command.Process.Kill()
		_ = command.Wait()
	}()
	line, err := bufio.NewReader(output).ReadString('\n')
	if err != nil || strings.TrimSpace(line) != "locked" {
		t.Fatalf("helper did not acquire lock: %q, %v", line, err)
	}
	if _, err := store.Read(context.Background(), "sample"); !errors.Is(err, ErrConflict) {
		t.Fatalf("concurrent reader = %v", err)
	}
	if err := store.Update(context.Background(), "sample", []byte("old"), []byte("new")); !errors.Is(err, ErrConflict) {
		t.Fatalf("concurrent writer = %v", err)
	}
	if err := command.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	_ = command.Wait()
	if body, err := store.Read(context.Background(), "sample"); err != nil || string(body) != "old" {
		t.Fatalf("bytes after conflict = %q, %v", body, err)
	}
}

func TestPortableCreateCrashBoundaryRequiresRecovery(t *testing.T) {
	for _, point := range []string{"stage", "published"} {
		t.Run(point, func(t *testing.T) {
			root := privateTestRoot(t)
			store, err := NewPortableStore(root)
			if err != nil {
				t.Fatal(err)
			}
			command := exec.Command(os.Args[0], "-test.run=^TestPortableProcessHelper$")
			command.Env = append(os.Environ(), "AXIOM_POC_HELPER="+point, "AXIOM_POC_ROOT="+root)
			output, err := command.StdoutPipe()
			if err != nil {
				t.Fatal(err)
			}
			if err := command.Start(); err != nil {
				t.Fatal(err)
			}
			defer func() {
				_ = command.Process.Kill()
				_ = command.Wait()
			}()
			line, err := bufio.NewReader(output).ReadString('\n')
			if err != nil || strings.TrimSpace(line) != point {
				t.Fatalf("crash barrier not reached: %q, %v", line, err)
			}
			if err := command.Process.Kill(); err != nil {
				t.Fatal(err)
			}
			_ = command.Wait()
			if _, err := store.Read(context.Background(), "sample"); !errors.Is(err, ErrRecoveryRequired) {
				t.Fatalf("crash at %s accepted: %v", point, err)
			}
			_, err = os.Stat(filepath.Join(root, "sample", manifestName))
			if point == "stage" && !os.IsNotExist(err) {
				t.Fatalf("precommit target visible: %v", err)
			}
			if point == "published" && err != nil {
				t.Fatalf("postcommit target missing: %v", err)
			}
		})
	}
}

func TestPortableUpdateCrashBoundaryRequiresRecovery(t *testing.T) {
	for _, point := range []string{"update-stage", "update-published"} {
		t.Run(point, func(t *testing.T) {
			root := privateTestRoot(t)
			store, err := NewPortableStore(root)
			if err != nil {
				t.Fatal(err)
			}
			if err := store.Create(context.Background(), "sample", []byte("old")); err != nil {
				t.Fatal(err)
			}
			command := exec.Command(os.Args[0], "-test.run=^TestPortableProcessHelper$")
			command.Env = append(os.Environ(), "AXIOM_POC_HELPER="+point, "AXIOM_POC_ROOT="+root)
			output, err := command.StdoutPipe()
			if err != nil {
				t.Fatal(err)
			}
			if err := command.Start(); err != nil {
				t.Fatal(err)
			}
			defer func() {
				_ = command.Process.Kill()
				_ = command.Wait()
			}()
			line, err := bufio.NewReader(output).ReadString('\n')
			if err != nil || strings.TrimSpace(line) != point {
				t.Fatalf("crash barrier not reached: %q, %v", line, err)
			}
			if err := command.Process.Kill(); err != nil {
				t.Fatal(err)
			}
			_ = command.Wait()
			if _, err := store.Read(context.Background(), "sample"); !errors.Is(err, ErrRecoveryRequired) {
				t.Fatalf("crash at %s accepted: %v", point, err)
			}
			body, err := os.ReadFile(filepath.Join(root, "sample", manifestName))
			want := "old"
			if point == "update-published" {
				want = "new"
			}
			if err != nil || string(body) != want {
				t.Fatalf("crash at %s left bytes %q, %v", point, body, err)
			}
		})
	}
}

func TestPortableProcessHelper(t *testing.T) {
	point := os.Getenv("AXIOM_POC_HELPER")
	if point == "" {
		return
	}
	store, err := NewPortableStore(os.Getenv("AXIOM_POC_ROOT"))
	if err != nil {
		t.Fatal(err)
	}
	if point == "hold" {
		if err := store.withLock("sample", false, true, func(*os.Root) error {
			fmt.Fprintln(os.Stdout, "locked")
			time.Sleep(time.Hour)
			return nil
		}); err != nil {
			t.Fatal(err)
		}
		return
	}
	if strings.HasPrefix(point, "update-") {
		store.beforeUpdatePublication = func() {
			if point == "update-stage" {
				fmt.Fprintln(os.Stdout, point)
				time.Sleep(time.Hour)
			}
		}
		store.afterUpdatePublication = func() {
			if point == "update-published" {
				fmt.Fprintln(os.Stdout, point)
				time.Sleep(time.Hour)
			}
		}
		if err := store.Update(context.Background(), "sample", []byte("old"), []byte("new")); err != nil {
			t.Fatal(err)
		}
		return
	}
	store.beforeCreatePublication = func() {
		if point == "stage" {
			fmt.Fprintln(os.Stdout, point)
			time.Sleep(time.Hour)
		}
	}
	store.afterCreatePublication = func() {
		if point == "published" {
			fmt.Fprintln(os.Stdout, point)
			time.Sleep(time.Hour)
		}
	}
	if err := store.Create(context.Background(), "sample", []byte("complete")); err != nil {
		t.Fatal(err)
	}
}
