package local

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
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

func TestPortableProcessHelper(t *testing.T) {
	if os.Getenv("AXIOM_POC_HELPER") != "hold" {
		return
	}
	store, err := NewPortableStore(os.Getenv("AXIOM_POC_ROOT"))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.withLock("sample", false, true, func(*os.Root) error {
		fmt.Fprintln(os.Stdout, "locked")
		select {}
	}); err != nil {
		t.Fatal(err)
	}
}
