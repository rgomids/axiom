package local

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestProtocolStateScanIsBoundedAndFailClosed(t *testing.T) {
	for _, test := range []struct {
		name    string
		entries []string
		limit   int
		pending bool
		wantErr bool
	}{
		{name: "empty", limit: 3},
		{name: "few safe", entries: []string{"a", "b"}, limit: 3},
		{name: "marker", entries: []string{"a", ".axiom-recovery-test"}, limit: 3, pending: true},
		{name: "at limit", entries: []string{"a", "b", "c"}, limit: 3},
		{name: "over limit", entries: []string{"a", "b", "c", "d"}, limit: 3, wantErr: true},
		{name: "many unknown", entries: []string{"a", "b", "c", "d", "e", "f"}, limit: 3, wantErr: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			path := privateTestRoot(t)
			for _, name := range test.entries {
				if err := os.WriteFile(filepath.Join(path, name), []byte("x"), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			root, err := os.OpenRoot(path)
			if err != nil {
				t.Fatal(err)
			}
			defer root.Close()
			pending, err := protocolStatePresentBounded(root, test.limit)
			if pending != test.pending || (err != nil) != test.wantErr {
				t.Fatalf("pending=%t err=%v", pending, err)
			}
		})
	}
}

func TestDirectoryEnumerationStopsAtExplicitLimit(t *testing.T) {
	path := privateTestRoot(t)
	for index := 0; index < 129; index++ {
		if err := os.WriteFile(filepath.Join(path, fmt.Sprintf("entry-%03d", index)), nil, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	root, err := os.OpenRoot(path)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	if _, err := readDirectoryNamesBounded(root, 128); !errors.Is(err, ErrUnsafe) {
		t.Fatalf("over-limit directory = %v", err)
	}
}
