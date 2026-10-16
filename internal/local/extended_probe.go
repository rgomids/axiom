//go:build unix

package local

import (
	"context"
	"os"
	"path/filepath"
	"syscall"

	"github.com/rgomids/axiom/internal/capabilityproof"
	"golang.org/x/sys/unix"
)

// ExtendedRootObserver reads only the exact host-selected physical temp roots.
// It provides no canary execution, effect authority or confinement attestation.
type ExtendedRootObserver struct{ TempDir, SystemTemp string }

func (o ExtendedRootObserver) Inspect(ctx context.Context, path string) (capabilityproof.ObjectSnapshot, error) {
	if err := ctx.Err(); err != nil {
		return capabilityproof.ObjectSnapshot{}, err
	}
	kind := ""
	if path == o.TempDir {
		kind = "tmpdir"
	}
	if path == o.SystemTemp {
		kind = "system-temp"
	}
	if kind == "" || !filepath.IsAbs(path) || filepath.Clean(path) != path || path == string(filepath.Separator) {
		return capabilityproof.ObjectSnapshot{}, capabilityproof.ErrExtendedRoot
	}
	for parent := path; ; parent = filepath.Dir(parent) {
		info, err := os.Lstat(parent)
		if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return capabilityproof.ObjectSnapshot{}, capabilityproof.ErrExtendedRoot
		}
		if _, err := os.Lstat(filepath.Join(parent, ".git")); !os.IsNotExist(err) {
			return capabilityproof.ObjectSnapshot{}, capabilityproof.ErrExtendedRoot
		}
		if filepath.Dir(parent) == parent {
			break
		}
	}
	before, err := os.Lstat(path)
	if err != nil {
		return capabilityproof.ObjectSnapshot{}, err
	}
	root, err := os.OpenRoot(path)
	if err != nil {
		return capabilityproof.ObjectSnapshot{}, err
	}
	defer root.Close()
	file, err := root.Open(".")
	if err != nil {
		return capabilityproof.ObjectSnapshot{}, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !os.SameFile(before, info) {
		return capabilityproof.ObjectSnapshot{}, capabilityproof.ErrExtendedRoot
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return capabilityproof.ObjectSnapshot{}, capabilityproof.ErrExtendedRoot
	}
	var filesystem unix.Statfs_t
	if err := unix.Fstatfs(int(file.Fd()), &filesystem); err != nil {
		return capabilityproof.ObjectSnapshot{}, err
	}
	aclReadable := checkPrivateACL(file) == nil
	aclDigest := ""
	if aclReadable {
		aclDigest = digestEnvelope([]byte("no-extended-acl"))
	}
	after, err := os.Lstat(path)
	if err != nil || !os.SameFile(info, after) {
		return capabilityproof.ObjectSnapshot{}, capabilityproof.ErrExtendedRoot
	}
	return capabilityproof.ObjectSnapshot{Device: uint64(stat.Dev), Inode: uint64(stat.Ino), Owner: uint64(stat.Uid), Mode: uint64(info.Mode()), ACLDigest: aclDigest, ACLReadable: aclReadable, ReadOnly: probeFilesystemReadOnly(filesystem), SymlinkFree: true, ResolvedTarget: kind}, nil
}
