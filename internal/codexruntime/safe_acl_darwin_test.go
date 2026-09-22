//go:build darwin

package codexruntime

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestPrivateACLInspectionAcceptsNoACL(t *testing.T) {
	root := filepath.Join(t.TempDir(), "skills")
	if err := os.Mkdir(root, 0o700); err != nil {
		t.Fatal(err)
	}
	directory, err := os.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	defer directory.Close()
	if err := checkPrivateACL(directory); err != nil {
		t.Fatalf("safe root rejected: %v", err)
	}
}

func TestInstallRejectsRootWithExtendedACL(t *testing.T) {
	root := filepath.Join(t.TempDir(), "skills")
	if err := os.Mkdir(root, 0o700); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("/bin/chmod", "+a", "everyone allow read,search", root)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("set synthetic ACL: %v: %s", err, output)
	}
	service, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	if got := service.Install(context.Background()); got.Status != Failed || got.Category != "codex_skill_root_unavailable" {
		t.Fatalf("ACL root accepted: %#v", got)
	}
}
