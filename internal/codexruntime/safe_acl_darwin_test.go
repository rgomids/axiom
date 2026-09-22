//go:build darwin && cgo

package codexruntime

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

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
