//go:build linux

package codexruntime

import (
	"context"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/sys/unix"
)

func TestInstallRejectsRootWithDefaultACL(t *testing.T) {
	root := filepath.Join(t.TempDir(), "skills")
	if err := os.Mkdir(root, 0o700); err != nil {
		t.Fatal(err)
	}
	acl := make([]byte, 4+5*8)
	binary.LittleEndian.PutUint32(acl[:4], 2)
	entries := []struct {
		tag  uint16
		perm uint16
		id   uint32
	}{{1, 7, ^uint32(0)}, {2, 4, 65534}, {4, 0, ^uint32(0)}, {16, 4, ^uint32(0)}, {32, 0, ^uint32(0)}}
	for index, entry := range entries {
		position := 4 + index*8
		binary.LittleEndian.PutUint16(acl[position:], entry.tag)
		binary.LittleEndian.PutUint16(acl[position+2:], entry.perm)
		binary.LittleEndian.PutUint32(acl[position+4:], entry.id)
	}
	if err := unix.Setxattr(root, "system.posix_acl_default", acl, 0); err != nil {
		t.Fatalf("set synthetic default ACL: %v", err)
	}
	service, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	if got := service.Install(context.Background()); got.Status != Failed || got.Category != "codex_skill_root_unavailable" {
		t.Fatalf("ACL root accepted: %#v", got)
	}
}
