package runtimeadapter

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/rgomids/axiom/internal/runtimeprofile"
)

// Specification 007 AXM-22 D6: a lexically valid configured source path does
// not prove filesystem safety. Resolution below its anchor still uses the
// existing trusted boundary, which refuses a symlinked component escaping it.
// Supported hosts (macOS, Linux) provide native symlinks; setup failure fails.
func TestEffortSourcesRefuseSymlinkEscapeOfValidConfiguredPath(t *testing.T) {
	const relative = ".codex/config.toml"
	if !runtimeprofile.ValidEffortSourcePath(relative) {
		t.Fatal("fixture path must be lexically valid")
	}
	i, p, inv := effortInspectionFixture(t, "codex")
	anchor := i.Declaration.Sources[0].Root
	outside := filepath.Join(filepath.Dir(anchor), "outside")
	if err := os.Mkdir(outside, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outside, "config.toml"), []byte("# no effort selector\n"), 0600); err != nil {
		t.Fatal(err)
	}
	i.Declaration.Sources[0].Relative = relative
	if err := os.Mkdir(filepath.Join(anchor, ".codex"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(anchor, relative), []byte("# no effort selector\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := i.CheckEffortSources(context.Background(), p, inv); err != nil {
		t.Fatal("contained path refused:", err)
	}
	if err := os.RemoveAll(filepath.Join(anchor, ".codex")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(anchor, ".codex")); err != nil {
		t.Fatal("symlink setup failed on a supported host:", err)
	}
	if info, err := os.Lstat(filepath.Join(anchor, ".codex")); err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatal("symlink setup not effective:", err)
	}
	if _, err := i.CheckEffortSources(context.Background(), p, inv); !errors.Is(err, ErrEffortConfigurationUnproven) {
		t.Fatal("symlink escape followed:", err)
	}
}
