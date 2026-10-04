package install

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/rgomids/axiom/internal/local"
)

// InstallRelease installs a fully verified Windows candidate. Existing installs
// use the same upgrade, compatibility, ownership and receipt protocol as POSIX.
// The caller runs the verified candidate from a private staging directory, so
// Windows never has to overwrite the executable running the installer.
// Preserve the owned upgrade's complete result, including its archive path.
func InstallRelease(ctx context.Context, target Target, candidate Candidate) (Result, error) {
	if hostRow() != "windows:windows:amd64" || candidate.Values["platform"] != "windows" || candidate.Values["goos"] != "windows" || candidate.Values["architecture"] != "amd64" {
		return Result{}, &Error{Category: "unsupported_host"}
	}
	overlap, err := local.RootsOverlap(target.BinaryDir, target.ReceiptDir)
	if err != nil || overlap {
		return Result{}, &Error{Category: "invalid_target"}
	}
	if len(candidate.Binary) == 0 || !digestPattern.MatchString(candidate.ArchiveSHA256) {
		return Result{}, &Error{Category: "invalid_candidate"}
	}
	marker := []byte("formatVersion=1\nstage=prepare\narchiveSha256=" + candidate.ArchiveSHA256 + "\n")
	fresh := false
	if root, err := local.OpenOwnedDirectory(target.ReceiptDir); err == nil {
		wire, readErr := root.ReadFile(markerName, maxReceiptBytes)
		root.Close()
		fresh = readErr == nil && bytes.Equal(wire, marker)
	}
	// Existing objects are inspected before any directory is created.
	if _, err := os.Lstat(filepath.Join(target.ReceiptDir, receiptName)); err == nil && !fresh {
		service := NewService()
		preview, err := service.Preview(ctx, target, candidate)
		if err != nil {
			return Result{}, err
		}
		if len(preview.Effects) == 0 && !preview.Resume {
			return Result{Status: "unchanged"}, nil
		}
		authority, err := Authorize(preview, preview.Digest)
		if err != nil {
			return Result{}, err
		}
		result, err := service.Apply(ctx, preview, authority)
		if err == nil && result.Status == "partial" {
			// Confirmed effects with a Codex skill-set receipt that is not
			// current are partial truth, never a successful exit.
			return result, &Error{Category: "skill_receipt_" + result.SkillReceipt}
		}
		return result, err
	} else if err != nil && !os.IsNotExist(err) {
		return Result{}, err
	}
	if _, err := os.Lstat(filepath.Join(target.BinaryDir, binaryName)); err == nil {
		if !fresh {
			return Result{}, &Error{Category: "foreign_binary_preserved"}
		}
	} else if !os.IsNotExist(err) {
		return Result{}, err
	}
	receipts, err := local.CreateOwnedDirectory(target.ReceiptDir)
	if err != nil {
		return Result{}, err
	}
	defer receipts.Close()
	if err := receipts.Mkdir(lockName); err != nil {
		return Result{}, &Error{Category: "installation_busy_or_interrupted"}
	}
	defer receipts.Remove(lockName)
	binaries, err := local.CreateOwnedDirectory(target.BinaryDir)
	if err != nil {
		return Result{}, err
	}
	defer binaries.Close()
	for _, directory := range []local.AnchoredDirectory{binaries, receipts} {
		available, err := directory.AvailableBytes()
		if err != nil || available < uint64(len(candidate.Binary))+maxReceiptBytes {
			return Result{}, &Error{Category: "insufficient_space"}
		}
	}
	if _, err := receipts.Lstat(receiptName); !os.IsNotExist(err) && !fresh {
		return Result{}, &Error{Category: "installation_changed"}
	}
	if current, err := readFreshOptional(receipts, markerName, maxReceiptBytes); err == nil {
		if !bytes.Equal(current, marker) {
			return Result{}, &Error{Category: "recovery_required"}
		}
	} else if os.IsNotExist(err) {
		if _, err := binaries.Lstat(binaryName); !os.IsNotExist(err) {
			return Result{}, &Error{Category: "foreign_binary_preserved"}
		}
		if _, err := receipts.Lstat(receiptName); !os.IsNotExist(err) {
			return Result{}, &Error{Category: "installation_changed"}
		}
		if err := publishFresh(receipts, markerName, marker, 0o600); err != nil {
			return Result{}, err
		}
	} else {
		return Result{}, err
	}
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	if current, err := readFreshOptional(binaries, binaryName, maxBinaryBytes); err == nil {
		if !bytes.Equal(current, candidate.Binary) {
			return Result{}, &Error{Category: "recovery_required"}
		}
	} else if os.IsNotExist(err) {
		if err := publishFresh(binaries, binaryName, candidate.Binary, 0o700); err != nil {
			return Result{}, err
		}
	} else {
		return Result{}, err
	}
	confirmed, err := binaries.ReadFile(binaryName, maxBinaryBytes)
	if err != nil || !bytes.Equal(confirmed, candidate.Binary) || binaries.StillAtPath() != nil {
		return Result{Status: "partial"}, &Error{Category: "publication_uncertain"}
	}
	installedAt := time.Now().UTC().Format("2006-01-02T15:04:05Z")
	currentReceipt, readErr := readFreshOptional(receipts, receiptName, maxReceiptBytes)
	if readErr == nil {
		values, err := parseReceipt(currentReceipt)
		if err != nil {
			return Result{Status: "partial"}, &Error{Category: "recovery_required"}
		}
		installedAt = values["installedAt"]
	} else if !os.IsNotExist(readErr) {
		return Result{Status: "partial"}, readErr
	}
	receipt := expectedReceipt(filepath.Join(target.BinaryDir, binaryName), candidate, installedAt)
	if readErr == nil {
		if !bytes.Equal(currentReceipt, receipt) {
			return Result{Status: "partial"}, &Error{Category: "recovery_required"}
		}
	} else if err := publishFresh(receipts, receiptName, receipt, 0o600); err != nil {
		return Result{Status: "partial"}, err
	}
	confirmed, err = receipts.ReadFile(receiptName, maxReceiptBytes)
	if err != nil || !bytes.Equal(confirmed, receipt) || receipts.StillAtPath() != nil {
		return Result{Status: "partial"}, &Error{Category: "publication_uncertain"}
	}
	if err := errors.Join(receipts.Remove(markerName), receipts.Sync()); err != nil {
		return Result{Status: "partial"}, err
	}
	return Result{Status: "installed"}, nil
}

func publishFresh(directory local.AnchoredDirectory, name string, wire []byte, mode os.FileMode) error {
	stage, err := directory.Stage(".axiom-fresh-", wire, mode)
	if err != nil {
		return err
	}
	defer directory.Remove(stage)
	if err := directory.PublishNew(stage, name); err != nil {
		return err
	}
	return directory.Sync()
}

func readFreshOptional(directory local.AnchoredDirectory, name string, limit int) ([]byte, error) {
	if _, err := directory.Lstat(name); err != nil {
		return nil, err
	}
	return directory.ReadFile(name, limit)
}
