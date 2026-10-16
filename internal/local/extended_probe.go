//go:build unix

package local

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
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

// ExtendedRootEffects is the host-side measurement backend of the extended
// temp-root probe. It touches only the reviewed probe filename directly below
// the exact observed root, never follows a link and removes only an empty
// regular file of the current user. Run executes the reviewed Runtime canary;
// its outcome is observed afterwards on the host, never read from the child.
type ExtendedRootEffects struct {
	ExtendedRootObserver
	Run func(context.Context, capabilityproof.ExtendedRoot, string) error
}

func extendedProbeName(name string) bool {
	id, ok := strings.CutPrefix(name, ".axiom-probe-")
	return ok && id != "" && filepath.Base(name) == name && !strings.ContainsAny(name, "\x00\r\n")
}

func (e ExtendedRootEffects) open(ctx context.Context, root capabilityproof.ExtendedRoot, name string) (*os.Root, error) {
	if !extendedProbeName(name) {
		return nil, capabilityproof.ErrExtendedRoot
	}
	snapshot, err := e.Inspect(ctx, root.Path)
	if err != nil || snapshot != root.Snapshot {
		return nil, capabilityproof.ErrStalePreview
	}
	r, err := os.OpenRoot(root.Path)
	if err != nil {
		return nil, err
	}
	info, err := r.Stat(".")
	stat, ok := info.Sys().(*syscall.Stat_t)
	if err != nil || !ok || uint64(stat.Dev) != root.Snapshot.Device || uint64(stat.Ino) != root.Snapshot.Inode {
		r.Close()
		return nil, capabilityproof.ErrStalePreview
	}
	return r, nil
}

func ownedProbeFile(info os.FileInfo) bool {
	stat, ok := info.Sys().(*syscall.Stat_t)
	return ok && info.Mode().IsRegular() && info.Size() == 0 && stat.Nlink == 1 && int(stat.Uid) == os.Geteuid()
}

func (e ExtendedRootEffects) Control(ctx context.Context, root capabilityproof.ExtendedRoot, name string) (capabilityproof.Control, error) {
	c := capabilityproof.Control{UID: uint64(os.Geteuid()), GID: uint64(os.Getegid())}
	r, err := e.open(ctx, root, name)
	if err != nil {
		return c, err
	}
	defer r.Close()
	if _, err := r.Lstat(name); !errors.Is(err, os.ErrNotExist) {
		return c, capabilityproof.ErrProbeResidue
	}
	file, err := r.OpenFile(name, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return c, err
	}
	created, err := file.Stat()
	if closeErr := file.Close(); err != nil || closeErr != nil {
		return c, errors.Join(capabilityproof.ErrProbeResidue, err, closeErr)
	}
	c.Created = true
	current, err := r.Lstat(name)
	if err != nil || !os.SameFile(created, current) || !ownedProbeFile(current) {
		return c, capabilityproof.ErrProbeResidue
	}
	if err := r.Remove(name); err != nil {
		return c, errors.Join(capabilityproof.ErrProbeResidue, err)
	}
	c.Removed = true
	if _, err := r.Lstat(name); !errors.Is(err, os.ErrNotExist) {
		return c, capabilityproof.ErrProbeResidue
	}
	c.RemovalVerified = true
	snapshot, err := e.Inspect(ctx, root.Path)
	if err != nil || snapshot != root.Snapshot {
		return c, capabilityproof.ErrStalePreview
	}
	c.Snapshot = snapshot
	return c, nil
}

func (e ExtendedRootEffects) Canary(ctx context.Context, root capabilityproof.ExtendedRoot, name string) (capabilityproof.CanaryAttempt, capabilityproof.ProcessIdentity, error) {
	attempt := capabilityproof.CanaryAttempt{Target: root.Kind}
	if e.Run == nil {
		return attempt, capabilityproof.ProcessIdentity{}, capabilityproof.ErrBlocked
	}
	r, err := e.open(ctx, root, name)
	if err != nil {
		return attempt, capabilityproof.ProcessIdentity{}, err
	}
	defer r.Close()
	if _, err := r.Lstat(name); !errors.Is(err, os.ErrNotExist) {
		return attempt, capabilityproof.ProcessIdentity{}, capabilityproof.ErrProbeResidue
	}
	runErr := e.Run(ctx, root, name)
	info, err := r.Lstat(name)
	if errors.Is(err, os.ErrNotExist) {
		return attempt, capabilityproof.ProcessIdentity{}, runErr
	}
	if err != nil || !ownedProbeFile(info) {
		return attempt, capabilityproof.ProcessIdentity{}, errors.Join(capabilityproof.ErrProbeResidue, runErr, err)
	}
	stat := info.Sys().(*syscall.Stat_t)
	attempt.Created = true
	return attempt, capabilityproof.ProcessIdentity{UID: uint64(stat.Uid), GID: uint64(stat.Gid), Process: "runtime-child"}, runErr
}

func (e ExtendedRootEffects) RemoveVerify(ctx context.Context, root capabilityproof.ExtendedRoot, name string) error {
	r, err := e.open(ctx, root, name)
	if err != nil {
		return err
	}
	defer r.Close()
	info, err := r.Lstat(name)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil || !ownedProbeFile(info) {
		return capabilityproof.ErrProbeResidue
	}
	if err := r.Remove(name); err != nil {
		return errors.Join(capabilityproof.ErrProbeResidue, err)
	}
	if _, err := r.Lstat(name); !errors.Is(err, os.ErrNotExist) {
		return capabilityproof.ErrProbeResidue
	}
	return nil
}

// CreateExtendedCanary is the deterministic child side of the extended probe.
// The working directory must belong to the digest-verified envelope of id;
// path must be that probe's filename directly below the child's own TMPDIR or
// /tmp. It makes one exclusive empty file and nothing else.
func CreateExtendedCanary(cwd, id, digest, path string) error {
	if !filepath.IsAbs(cwd) || filepath.Clean(cwd) != cwd || !slices.Contains([]string{"workspace", "inbox"}, filepath.Base(cwd)) || filepath.Base(filepath.Dir(cwd)) != id || filepath.Base(filepath.Dir(filepath.Dir(cwd))) != "capability-probes" {
		return capabilityproof.ErrCanaryContext
	}
	directory, err := OpenOwnedDirectory(filepath.Dir(cwd))
	if err != nil {
		return err
	}
	wire, err := directory.ReadFile("envelope.json", 16384)
	directory.Close()
	var m EnvelopeManifest
	if err != nil || strictEnvelopeJSON(wire, &m) != nil || digestEnvelope(wire) != digest || m.ProbeID != id {
		return capabilityproof.ErrCanaryContext
	}
	if !filepath.IsAbs(path) || filepath.Clean(path) != path || filepath.Base(path) != ".axiom-probe-"+id {
		return capabilityproof.ErrExtendedRoot
	}
	parent := filepath.Dir(path)
	allowed := false
	for _, candidate := range []string{os.TempDir(), "/tmp"} {
		if physical, err := filepath.EvalSymlinks(candidate); err == nil && physical == parent {
			allowed = true
		}
	}
	if !allowed {
		return capabilityproof.ErrExtendedRoot
	}
	r, err := os.OpenRoot(parent)
	if err != nil {
		return err
	}
	defer r.Close()
	file, err := r.OpenFile(filepath.Base(path), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	return file.Close()
}
