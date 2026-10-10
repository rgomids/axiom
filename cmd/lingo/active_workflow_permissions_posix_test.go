//go:build !windows

package main

import (
	"bytes"
	"os"
	"sync"
	"testing"
)

// Denial is proved by an actual read, not inferred from the permission bits.
func workflow303DenyRead(t *testing.T, path string) {
	t.Helper()
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	var once sync.Once
	restore := func() {
		once.Do(func() {
			if err := os.Chmod(path, info.Mode().Perm()); err != nil {
				t.Error(err)
				return
			}
			after, err := os.ReadFile(path)
			if err != nil || !bytes.Equal(before, after) {
				t.Errorf("unreadable fixture bytes changed: %v", err)
			}
		})
	}
	t.Cleanup(restore)
	if err := os.Chmod(path, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := os.ReadFile(path); !os.IsPermission(err) {
		t.Fatalf("fixture must deny actual read: %v", err)
	}
}

func workflow303AccessState(*testing.T, string) string { return "" }
