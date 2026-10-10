//go:build !windows

package main

import (
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

// A FIFO or symlink is refused before opening, so a stalled writer cannot block
// planning and a link cannot redirect the reviewed Plan document.
func TestReadPlanDocumentRefusesNonRegularFiles(t *testing.T) {
	dir := t.TempDir()
	fifo := filepath.Join(dir, "plan.fifo")
	if err := syscall.Mkfifo(fifo, 0o600); err != nil {
		t.Skipf("mkfifo unavailable: %v", err)
	}
	regular := filepath.Join(dir, "plan.json")
	if err := os.WriteFile(regular, []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link.json")
	if err := os.Symlink(regular, link); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{fifo, link, dir} {
		done := make(chan error, 1)
		go func() { _, _, err := readPlanDocument(path); done <- err }()
		select {
		case err := <-done:
			if !errors.Is(err, errPlanFile) {
				t.Fatalf("%s accepted: %v", filepath.Base(path), err)
			}
		case <-time.After(5 * time.Second):
			t.Fatalf("%s blocked planning", filepath.Base(path))
		}
	}
}
