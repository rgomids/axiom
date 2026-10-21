//go:build unix

package operationaldocs

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"syscall"
)

const (
	StateVerified = "verified"
	StateAbsent   = "absent"
	StateInvalid  = "invalid"
	StateUnknown  = "unknown"

	MaxBinary   = 256 << 20
	MaxBindings = 64
)

type Refusal struct{ State, Reason string }

func (r Refusal) Error() string { return r.State + ":" + r.Reason }

type Resolution struct {
	State, Reason string
	Binary        string
	BinarySHA256  string
	Binding       Binding
	BindingSHA256 string
	Root          string
	Bundle        Bundle
}

func refuse(state, reason string) error { return Refusal{state, reason} }

func classify(err error) error {
	var refusal Refusal
	switch {
	case errors.As(err, &refusal):
		return err
	case errors.Is(err, fs.ErrNotExist):
		return refuse(StateAbsent, "binding_absent")
	case errors.Is(err, fs.ErrPermission):
		return refuse(StateUnknown, "permission_denied")
	}
	return refuse(StateUnknown, "io_unavailable")
}

func codeFrom(err error) (string, string) {
	var refusal Refusal
	if errors.As(classify(err), &refusal) {
		return refusal.State, refusal.Reason
	}
	return StateUnknown, "io_unavailable"
}

func descriptorPlatform(d Descriptor) bool {
	return (d.Platform == "macos" && runtime.GOOS == "darwin" || d.Platform == "linux" && runtime.GOOS == "linux") && d.Architecture == runtime.GOARCH
}

func checkOwner(info fs.FileInfo, private bool) error {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || int(stat.Uid) != os.Geteuid() {
		return refuse(StateInvalid, "unsafe_permissions")
	}
	mask := fs.FileMode(0o022)
	if private {
		mask = 0o077
	}
	if info.Mode().Perm()&mask != 0 || info.Mode()&(fs.ModeSetuid|fs.ModeSetgid|fs.ModeSticky) != 0 {
		return refuse(StateInvalid, "unsafe_permissions")
	}
	return nil
}

func privateDir(path string, private bool) (fs.FileInfo, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if info.Mode()&fs.ModeSymlink != 0 || !info.IsDir() {
		return nil, refuse(StateInvalid, "unsafe_path")
	}
	return info, checkOwner(info, private)
}

func devIno(info fs.FileInfo) (uint64, uint64) {
	stat := info.Sys().(*syscall.Stat_t)
	return uint64(stat.Dev), uint64(stat.Ino)
}

func openRegular(path string, limit int64) (*os.File, fs.FileInfo, error) {
	before, err := os.Lstat(path)
	if err != nil {
		return nil, nil, err
	}
	if !before.Mode().IsRegular() {
		return nil, nil, refuse(StateInvalid, "unsafe_path")
	}
	file, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return nil, nil, err
	}
	info, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, nil, err
	}
	dev, ino := devIno(info)
	bdev, bino := devIno(before)
	if !info.Mode().IsRegular() || dev != bdev || ino != bino {
		file.Close()
		return nil, nil, refuse(StateUnknown, "concurrent_change")
	}
	if info.Size() > limit {
		file.Close()
		return nil, nil, refuse(StateUnknown, "quota_exceeded")
	}
	return file, info, nil
}

func readBounded(path string, limit int64, private bool) ([]byte, error) {
	file, info, err := openRegular(path, limit)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	if err := checkOwner(info, private); err != nil {
		return nil, err
	}
	wire, err := io.ReadAll(io.LimitReader(file, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(wire)) != info.Size() {
		return nil, refuse(StateUnknown, "concurrent_change")
	}
	return wire, nil
}

func hashBinary(path string) (string, fs.FileInfo, error) {
	file, info, err := openRegular(path, MaxBinary)
	if err != nil {
		return "", nil, err
	}
	defer file.Close()
	sum := sha256.New()
	n, err := io.Copy(sum, file)
	if err != nil {
		return "", nil, err
	}
	if n != info.Size() {
		return "", nil, refuse(StateUnknown, "concurrent_change")
	}
	return hex.EncodeToString(sum.Sum(nil)), info, nil
}

// Resolve verifies the installation owning executable against the compiled
// descriptor. It reads only; it never repairs, downloads or consults a checkout.
func Resolve(executable string, d Descriptor) Resolution {
	result := Resolution{State: StateUnknown, Reason: "source_unqualified"}
	if !d.Valid() || !descriptorPlatform(d) {
		return result
	}
	resolved, err := resolveInstallation(executable, d)
	if err != nil {
		result.State, result.Reason = codeFrom(err)
		return result
	}
	return resolved
}

func resolveInstallation(executable string, d Descriptor) (Resolution, error) {
	var zero Resolution
	physical, err := filepath.EvalSymlinks(executable)
	if err != nil {
		return zero, classify(err)
	}
	if !cleanPath(physical) {
		return zero, refuse(StateInvalid, "unsafe_path")
	}
	parent := filepath.Dir(physical)
	parentInfo, err := os.Lstat(parent)
	if err != nil {
		return zero, classify(err)
	}
	namespace := filepath.Join(parent, ".axiom-docs", "axiom")
	for _, dir := range []string{filepath.Join(parent, ".axiom-docs"), namespace} {
		if _, err := privateDir(dir, true); err != nil {
			return zero, classify(err)
		}
	}
	digest, binaryInfo, err := hashBinary(physical)
	if err != nil {
		return zero, classify(err)
	}
	dir := filepath.Join(namespace, "bindings", digest)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return zero, classify(err)
	}
	if len(entries) > MaxBindings {
		return zero, refuse(StateUnknown, "quota_exceeded")
	}
	var found []Binding
	var names []string
	for _, entry := range entries {
		wire, err := readBounded(filepath.Join(dir, entry.Name()), MaxBinding, false)
		if err != nil {
			return zero, classify(err)
		}
		binding, err := ParseBinding(wire)
		if err != nil {
			return zero, refuse(StateInvalid, err.Error())
		}
		if entry.Name() != Hash(wire)+".json" || binding.BinarySHA256 != digest {
			return zero, refuse(StateInvalid, "identity_mismatch")
		}
		found = append(found, binding)
		names = append(names, strings.TrimSuffix(entry.Name(), ".json"))
	}
	var matches []int
	pdev, pino := devIno(parentInfo)
	bdev, bino := devIno(binaryInfo)
	for i, b := range found {
		if b.BinaryPath == physical && b.ParentDevice == pdev && b.ParentInode == pino && b.BinaryDevice == bdev && b.BinaryInode == bino {
			matches = append(matches, i)
		}
	}
	if len(matches) == 0 {
		return zero, refuse(StateAbsent, "binding_absent")
	}
	if len(matches) > 1 {
		return zero, refuse(StateInvalid, "binding_conflict")
	}
	binding, bindingHash := found[matches[0]], names[matches[0]]
	if binding.Generation != d.Generation || binding.SourceCommit != d.SourceCommit {
		return zero, refuse(StateInvalid, "identity_mismatch")
	}
	root := filepath.Join(namespace, "generations", binding.Generation)
	bundle, err := loadGeneration(root, binding)
	if err != nil {
		return zero, classify(err)
	}
	return Resolution{State: StateVerified, Reason: "ok", Binary: physical, BinarySHA256: digest, Binding: binding, BindingSHA256: bindingHash, Root: root, Bundle: bundle}, nil
}

func loadGeneration(root string, binding Binding) (Bundle, error) {
	var zero Bundle
	files, err := ReadGeneration(root)
	if err != nil {
		return zero, err
	}
	bundle, err := Validate(files, binding.SourceCommit)
	if err != nil {
		reason := err.Error()
		state := StateInvalid
		if reason == "quota_exceeded" {
			state = StateUnknown
		}
		return zero, refuse(state, reason)
	}
	if bundle.Generation != binding.Generation || bundle.ProvenanceSHA256 != binding.ProvenanceSHA256 || bundle.ManMapSHA256 != binding.ManMapSHA256 {
		return zero, refuse(StateInvalid, "identity_mismatch")
	}
	return bundle, nil
}

// ReadGeneration reads one published generation tree under the resolver's path,
// ownership and bound rules, without validating its documentary content.
func ReadGeneration(root string) (map[string][]byte, error) {
	if _, err := privateDir(filepath.Dir(root), true); err != nil {
		return nil, err
	}
	if _, err := privateDir(root, false); err != nil {
		return nil, err
	}
	files := map[string][]byte{}
	total := 0
	var walk func(dir string) error
	walk = func(dir string) error {
		entries, err := os.ReadDir(filepath.Join(root, dir))
		if err != nil {
			return err
		}
		sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
		for _, entry := range entries {
			rel := filepath.ToSlash(filepath.Join(dir, entry.Name()))
			if entry.IsDir() {
				if _, err := privateDir(filepath.Join(root, rel), false); err != nil {
					return err
				}
				if err := walk(rel); err != nil {
					return err
				}
				continue
			}
			if len(files) >= MaxOutputs+3 {
				return refuse(StateUnknown, "quota_exceeded")
			}
			wire, err := readBounded(filepath.Join(root, rel), MaxFile, false)
			if err != nil {
				return err
			}
			if total += len(wire); total > MaxContent+MaxControl*3 {
				return refuse(StateUnknown, "quota_exceeded")
			}
			files[rel] = wire
		}
		return nil
	}
	if err := walk("docs"); err != nil {
		return nil, err
	}
	return files, nil
}

func (r Resolution) Locator(name string) string {
	if r.State != StateVerified {
		return ""
	}
	return filepath.Join(r.Root, filepath.FromSlash(name))
}
