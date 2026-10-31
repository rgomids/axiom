package local

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Breaking cutover support (ADR-0037 §7, Plan "Breaking cutover and
// rollback"). Everything here is read-only over the state root except the
// backup, which writes only into a separate empty private directory. Nothing
// converts, migrates, resumes or deletes legacy state; nothing signals a
// process. Real installed state is inspected only under separate authority.

// legacyFamilies are the pre-cutover operational roots. Their bytes are
// preserved as inert history; they are never decoded by the new engine.
var legacyFamilies = []string{
	"executions/v1", "graphs/v1", "single-stage-runs/v1", "validator-policies/v1",
	"run-leases/v1", "coordination", "attempt-inboxes", "delivery",
}

// cutoverKnownRoots are the top-level names whose meaning is established.
// Anything else makes the inventory incomplete.
var cutoverKnownRoots = map[string]bool{"executions": true, "graphs": true, "single-stage-runs": true, "validator-policies": true, "run-leases": true, "coordination": true, "attempt-inboxes": true, "delivery": true, "projects": true, "work-items": true, "workflows": true, "runtime-profiles": true, "artifacts": true}

type CutoverFile struct {
	Path   string `json:"path"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
}

type CutoverInventory struct {
	Files    []CutoverFile `json:"files"`
	Blockers []string      `json:"blockers"`
}

// Blocked reports whether the inventory alone forbids switching.
func (i CutoverInventory) Blocked() bool { return len(i.Blockers) != 0 }

// Digest identifies the exact inventory an operator disposition refers to.
func (i CutoverInventory) Digest() string {
	h := sha256.New()
	for _, f := range i.Files {
		fmt.Fprintf(h, "%s\x00%d\x00%s\n", f.Path, f.Size, f.SHA256)
	}
	return hex.EncodeToString(h.Sum(nil))
}

const maxCutoverFiles = 100_000

// InspectCutover inventories legacy operational state. It BLOCKS on a held
// runner lease (possibly active attempts), interrupted publications,
// symlinks, unknown top-level entries or any read failure: an incomplete
// inventory never authorizes cutover.
func InspectCutover(ctx context.Context, stateRoot string) (CutoverInventory, error) {
	var inv CutoverInventory
	root, err := trustedCanonical(stateRoot)
	if err != nil {
		return inv, err
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return inv, err
	}
	for _, e := range entries {
		if !cutoverKnownRoots[e.Name()] {
			inv.Blockers = append(inv.Blockers, "unknown_state_entry: "+e.Name())
		}
	}
	for _, family := range legacyFamilies {
		base := filepath.Join(root, filepath.FromSlash(family))
		err := filepath.WalkDir(base, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				if errors.Is(err, fs.ErrNotExist) && path == base {
					return fs.SkipDir
				}
				return err
			}
			if ctx.Err() != nil {
				return ctx.Err()
			}
			rel, _ := filepath.Rel(root, path)
			rel = filepath.ToSlash(rel)
			if d.Type()&fs.ModeSymlink != 0 {
				inv.Blockers = append(inv.Blockers, "symlink: "+rel)
				return nil
			}
			name := d.Name()
			if strings.HasPrefix(name, ".axiom-recovery-") || strings.HasPrefix(name, ".axiom-stage-") {
				inv.Blockers = append(inv.Blockers, "interrupted_publication: "+rel)
			}
			if d.IsDir() {
				if family == "run-leases/v1" && path != base && filepath.Dir(filepath.Dir(path)) == base {
					if held, err := probeLeaseDir(ctx, path); err != nil || held {
						inv.Blockers = append(inv.Blockers, "runner_lease_held_or_unreadable: "+rel)
					}
				}
				return nil
			}
			if !d.Type().IsRegular() {
				inv.Blockers = append(inv.Blockers, "irregular_file: "+rel)
				return nil
			}
			if len(inv.Files) >= maxCutoverFiles {
				return errors.New("cutover inventory bound exceeded")
			}
			sum, size, err := hashFile(path)
			if err != nil {
				return err
			}
			inv.Files = append(inv.Files, CutoverFile{Path: rel, Size: size, SHA256: sum})
			return nil
		})
		if err != nil && !errors.Is(err, fs.SkipDir) {
			inv.Blockers = append(inv.Blockers, "inventory_incomplete: "+family)
		}
	}
	sort.Slice(inv.Files, func(a, b int) bool { return inv.Files[a].Path < inv.Files[b].Path })
	sort.Strings(inv.Blockers)
	return inv, nil
}

// probeLeaseDir takes the lease directory lock non-blockingly without
// changing its epoch. A held lock means a runner may own live attempts.
func probeLeaseDir(ctx context.Context, dir string) (bool, error) {
	if ctx.Err() != nil {
		return false, ctx.Err()
	}
	r, err := os.OpenRoot(dir)
	if err != nil {
		return false, err
	}
	defer r.Close()
	lock, err := lockDirectory(r, true)
	if errors.Is(err, ErrConflict) {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	return false, lock.Close()
}

func hashFile(path string) (string, int64, error) {
	wire, err := os.ReadFile(path)
	if err != nil {
		return "", 0, err
	}
	sum := sha256.Sum256(wire)
	return hex.EncodeToString(sum[:]), int64(len(wire)), nil
}

// BackupLegacyState copies the raw bytes of every inventoried file into an
// empty private directory outside the state root and verifies byte identity.
// It never decodes, converts or removes the source.
func BackupLegacyState(ctx context.Context, stateRoot, backupDir string, inv CutoverInventory) error {
	root, err := trustedCanonical(stateRoot)
	if err != nil {
		return err
	}
	dest, err := trustedCanonical(backupDir)
	if err != nil {
		return err
	}
	if overlap, err := RootsOverlap(root, dest); err != nil || overlap {
		return fmt.Errorf("%w: backup must be outside the state root", ErrUnsafe)
	}
	info, err := os.Lstat(dest)
	if err != nil || !info.IsDir() || info.Mode().Perm()&0o077 != 0 {
		return fmt.Errorf("%w: backup directory must be a private directory", ErrUnsafe)
	}
	if entries, err := os.ReadDir(dest); err != nil || len(entries) != 0 {
		return fmt.Errorf("%w: backup directory must be empty", ErrUnsafe)
	}
	for _, f := range inv.Files {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		src := filepath.Join(root, filepath.FromSlash(f.Path))
		wire, err := os.ReadFile(src)
		if err != nil {
			return err
		}
		sum := sha256.Sum256(wire)
		if hex.EncodeToString(sum[:]) != f.SHA256 {
			return fmt.Errorf("%w: %s changed since inventory", ErrConflict, f.Path)
		}
		target := filepath.Join(dest, filepath.FromSlash(f.Path))
		if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
			return err
		}
		if err := os.WriteFile(target, wire, 0o600); err != nil {
			return err
		}
	}
	return VerifyLegacyBackup(backupDir, inv)
}

// VerifyLegacyBackup re-reads every backed-up file and compares its bytes.
func VerifyLegacyBackup(backupDir string, inv CutoverInventory) error {
	for _, f := range inv.Files {
		sum, size, err := hashFile(filepath.Join(backupDir, filepath.FromSlash(f.Path)))
		if err != nil || sum != f.SHA256 || size != f.Size {
			return fmt.Errorf("%w: backup of %s is missing or differs", ErrConflict, f.Path)
		}
	}
	return nil
}

// CutoverDisposition is the operator's explicit decision for one exact
// inventory: whether a verified backup exists (or why it is infeasible) and
// that every legacy attempt is stopped or knowingly abandoned without replay.
type CutoverDisposition struct {
	InventoryDigest  string
	BackupVerified   bool
	BackupInfeasible string
	AttemptsDisposed bool
	Actor            string
}

// CutoverDecision returns nil only when new-format-only starts may be
// enabled. Every other outcome is BLOCK with the reasons to resolve.
func CutoverDecision(inv CutoverInventory, d CutoverDisposition) error {
	reasons := append([]string(nil), inv.Blockers...)
	if !strings.HasPrefix(d.Actor, "human:") {
		reasons = append(reasons, "disposition_requires_human_actor")
	}
	if d.InventoryDigest != inv.Digest() {
		reasons = append(reasons, "disposition_does_not_match_inventory")
	}
	if !d.BackupVerified && d.BackupInfeasible == "" {
		reasons = append(reasons, "backup_not_verified")
	}
	if len(inv.Files) != 0 && !d.AttemptsDisposed {
		reasons = append(reasons, "legacy_attempt_disposition_missing")
	}
	if len(reasons) != 0 {
		return fmt.Errorf("%w: cutover BLOCKED: %s", ErrRecoveryRequired, strings.Join(reasons, "; "))
	}
	return nil
}
