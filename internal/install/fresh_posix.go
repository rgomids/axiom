//go:build !windows

package install

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/rgomids/axiom/internal/local"
)

// InstallerResult preserves the POSIX installer's output and exit protocol.
// An owned upgrade delegates presentation to the command's existing upgrade
// formatter; the lifecycle never makes its own compatibility decision.
type InstallerResult struct {
	Stdout         string
	Stderr         string
	ExitCode       int
	UpgradePhase   string
	UpgradePreview Preview
	UpgradeResult  Result
	UpgradeError   error
}

func installRefusal(message string) InstallerResult {
	return InstallerResult{Stderr: "install_error: " + message + "\n", ExitCode: 1}
}

// InstallReleasePOSIX installs a verified candidate. Fresh interruption markers
// always require operator recovery; only an exact owned upgrade can resume.
// failStage is the explicit test seam used by AXIOM_INSTALL_TEST_FAIL_STAGE.
// upgradePreflight defers command configuration errors until an upgrade is needed.
func InstallReleasePOSIX(ctx context.Context, target Target, candidate Candidate, failStage string, upgradePreflight ...func() error) InstallerResult {
	if len(candidate.Binary) == 0 || !digestPattern.MatchString(candidate.ArchiveSHA256) {
		return installRefusal("invalid candidate")
	}
	row := candidate.Values["platform"] + ":" + candidate.Values["goos"] + ":" + candidate.Values["architecture"]
	if hostRow() != row {
		return installRefusal("unsupported host for release row " + candidate.Values["platform"] + "/" + candidate.Values["architecture"] + "; supported OS family and architecture required")
	}
	for _, root := range []string{target.BinaryDir, target.ReceiptDir} {
		if !filepath.IsAbs(root) || root == "/" {
			return installRefusal("unsafe destination ownership, permissions, ACL, or type")
		}
		if posixHasSymlink(root) {
			return installRefusal("symlink destination refused")
		}
		if err := local.CheckInstallDirectory(root, root == target.ReceiptDir); err != nil {
			if posixHasSymlink(root) {
				return installRefusal("symlink destination refused")
			}
			return installRefusal("unsafe destination ownership, permissions, ACL, or type")
		}
	}
	destination := target.BinaryDir + "/" + binaryName
	receipt := filepath.Join(target.ReceiptDir, receiptName)
	marker := filepath.Join(target.ReceiptDir, markerName)
	if posixExists(destination) && !posixExists(receipt) && !posixExists(marker) {
		return installRefusal("foreign binary preserved")
	}
	receipts, err := local.CreateOwnedDirectory(target.ReceiptDir)
	if err != nil {
		return installRefusal("unsafe destination ownership, permissions, ACL, or type")
	}
	defer receipts.Close()
	if err := receipts.Mkdir(lockName); err != nil {
		return installRefusal("concurrent installation refused")
	}
	lockInfo, err := receipts.Lstat(lockName)
	if err != nil {
		return installRefusal("concurrent installation refused")
	}
	locked := true
	defer func() {
		if locked {
			_ = posixRemoveLock(receipts, lockInfo)
		}
	}()
	var binaries local.AnchoredDirectory
	if posixExists(target.BinaryDir) {
		if target.BinaryDir == target.ReceiptDir {
			binaries, err = local.OpenOwnedDirectory(target.BinaryDir)
		} else {
			binaries, err = local.OpenPublicationDirectory(target.BinaryDir)
		}
	} else {
		binaries, err = local.CreateOwnedDirectory(target.BinaryDir)
	}
	if err != nil {
		return installRefusal("unsafe destination ownership, permissions, ACL, or type")
	}
	defer binaries.Close()
	upgrade := func() InstallerResult {
		if err := posixRemoveLock(receipts, lockInfo); err != nil {
			return installRefusal("concurrent installation refused")
		}
		locked = false
		for _, preflight := range upgradePreflight {
			if err := preflight(); err != nil {
				return InstallerResult{ExitCode: 1, UpgradePhase: "preview", UpgradeError: err}
			}
		}
		service := NewService()
		upgradeTarget := target
		upgradeTarget.BinaryDir = filepath.Clean(target.BinaryDir)
		upgradeTarget.ReceiptDir = filepath.Clean(target.ReceiptDir)
		preview, err := service.Preview(ctx, upgradeTarget, candidate)
		if err != nil {
			return InstallerResult{ExitCode: 1, UpgradePhase: "preview", UpgradePreview: preview, UpgradeError: err}
		}
		authority, err := Authorize(preview, preview.Digest)
		if err != nil {
			return InstallerResult{ExitCode: 1, UpgradePhase: "apply", UpgradePreview: preview, UpgradeError: err}
		}
		result, err := service.Apply(ctx, preview, authority)
		code := 1
		if err == nil && result.Status == "success" {
			code = 0
		}
		return InstallerResult{ExitCode: code, UpgradePhase: "apply", UpgradePreview: preview, UpgradeResult: result, UpgradeError: err}
	}
	if _, err := receipts.Lstat(markerName); !os.IsNotExist(err) {
		wire, readErr := receipts.ReadFile(markerName, 4096)
		if readErr == nil && strings.Contains("\n"+string(wire), "\noperation=upgrade\n") && strings.Contains("\n"+string(wire), "\narchiveSha256="+candidate.ArchiveSHA256+"\n") {
			return upgrade()
		}
		return installRefusal("recovery_required")
	}
	if info, err := binaries.Lstat(binaryName); !os.IsNotExist(err) {
		if err != nil || !info.Mode().IsRegular() {
			return installRefusal("unsafe binary destination")
		}
		receiptInfo, err := receipts.Lstat(receiptName)
		if err != nil || !receiptInfo.Mode().IsRegular() {
			return installRefusal("foreign binary preserved")
		}
		if !posixOwner(info) || !posixOwner(receiptInfo) {
			return installRefusal("unowned installation preserved")
		}
		if info.Mode().Perm() != 0o700 || receiptInfo.Mode().Perm() != 0o600 || runtime.GOOS == "linux" && (info.Mode()|receiptInfo.Mode())&(os.ModeSetuid|os.ModeSetgid|os.ModeSticky) != 0 {
			return installRefusal("unsafe installation permissions")
		}
		current, binaryErr := binaries.InspectInstallFile(binaryName, maxBinaryBytes)
		receiptWire, err := receipts.InspectInstallFile(receiptName, maxReceiptBytes)
		if err != nil || binaryErr != nil {
			return installRefusal("unsafe installation ACL")
		}
		values, err := posixReceiptSchema(receiptWire)
		if err != nil {
			return installRefusal("invalid receipt schema preserved")
		}
		if !posixSingleLink(info) || !posixSingleLink(receiptInfo) {
			return installRefusal("hard-linked installation preserved")
		}
		if digest(current) != values["sha256"] {
			return installRefusal("modified binary preserved")
		}
		if values["sha256"] == digest(candidate.Binary) {
			if !bytes.Equal(receiptWire, expectedReceipt(destination, candidate, values["installedAt"])) {
				return installRefusal("divergent receipt preserved")
			}
			if binaries.StillAtPath() != nil || receipts.StillAtPath() != nil {
				return installRefusal("binary publication uncertain")
			}
			return InstallerResult{Stdout: "install_status=unchanged\n"}
		}
		return upgrade()
	}
	if _, err := receipts.Lstat(receiptName); !os.IsNotExist(err) {
		return installRefusal("foreign receipt preserved")
	}
	if ctx.Err() != nil {
		return installRefusal("installation cancelled")
	}
	for _, directory := range []local.AnchoredDirectory{binaries, receipts} {
		available, err := directory.AvailableBytes()
		if err != nil || available < uint64(len(candidate.Binary))+maxReceiptBytes {
			return installRefusal("insufficient space")
		}
	}
	prepare := []byte("formatVersion=1\nstage=prepare\narchiveSha256=" + candidate.ArchiveSHA256 + "\n")
	if err := receipts.CreateExclusive(markerName, prepare, 0o600); err != nil {
		return installRefusal("recovery_required")
	}
	markerInfo, err := receipts.Lstat(markerName)
	if err != nil {
		return installRefusal("recovery_required")
	}
	binaryCommitted := false
	defer func() {
		if !binaryCommitted {
			_ = posixRemoveFile(receipts, markerName, markerInfo, prepare, maxReceiptBytes)
			_ = receipts.Sync()
		}
	}()
	if err := receipts.Sync(); err != nil {
		return installRefusal("recovery_required")
	}
	if failStage == "before_binary" {
		return InstallerResult{Stderr: "install_error: injected pre-commit interruption\n", ExitCode: 75}
	}
	committedBinary, err := posixPublishFresh(binaries, binaryName, binaryStage, candidate.Binary, 0o700, maxBinaryBytes)
	binaryCommitted = committedBinary
	if err != nil {
		if os.IsExist(err) {
			return installRefusal("binary publication conflict")
		}
		return installRefusal("binary publication uncertain")
	}
	committed := []byte("formatVersion=1\nstage=binary_committed\narchiveSha256=" + candidate.ArchiveSHA256 + "\n")
	stage, err := receipts.Stage(markerName+".", committed, 0o600)
	if err != nil {
		return installRefusal("binary publication uncertain")
	}
	stageInfo, err := receipts.Lstat(stage)
	if err != nil {
		return installRefusal("binary publication uncertain")
	}
	defer posixRemoveFile(receipts, stage, stageInfo, committed, maxReceiptBytes)
	if err := errors.Join(receipts.Rename(stage, markerName), receipts.Sync()); err != nil {
		return installRefusal("binary publication uncertain")
	}
	if failStage == "after_binary" {
		return InstallerResult{Stderr: "install_status=partial\n", ExitCode: 75}
	}
	receiptWire := expectedReceipt(destination, candidate, time.Now().UTC().Format("2006-01-02T15:04:05Z"))
	if ctx.Err() != nil {
		return InstallerResult{Stderr: "install_status=partial\n", ExitCode: 1}
	}
	_, receiptErr := posixPublishFresh(receipts, receiptName, receiptStage, receiptWire, 0o600, maxReceiptBytes)
	if receiptErr != nil || binaries.StillAtPath() != nil || receipts.StillAtPath() != nil {
		return InstallerResult{Stderr: "install_status=partial\n", ExitCode: 1}
	}
	if err := errors.Join(posixRemoveFile(receipts, markerName, stageInfo, committed, maxReceiptBytes), receipts.Sync()); err != nil {
		return InstallerResult{Stderr: "install_status=partial\n", ExitCode: 1}
	}
	return InstallerResult{Stdout: "install_status=installed\n"}
}

func posixPublishFresh(directory local.AnchoredDirectory, name, prefix string, wire []byte, mode os.FileMode, limit int) (bool, error) {
	stage, err := directory.Stage(prefix, wire, mode)
	if err != nil {
		return false, err
	}
	stageInfo, err := directory.Lstat(stage)
	if err != nil {
		return false, err
	}
	defer posixRemoveFile(directory, stage, stageInfo, wire, limit)
	staged, err := directory.ReadFile(stage, limit)
	if err != nil || !bytes.Equal(staged, wire) {
		return false, local.ErrUnsafe
	}
	if err := directory.PublishNew(stage, name); err != nil {
		return false, err
	}
	if err := directory.Sync(); err != nil {
		return true, err
	}
	confirmed, err := directory.ReadFile(name, limit)
	if err != nil || !bytes.Equal(confirmed, wire) {
		return true, local.ErrUnsafe
	}
	return true, directory.StillAtPath()
}
func posixOwner(info os.FileInfo) bool {
	stat, ok := info.Sys().(*syscall.Stat_t)
	return ok && stat.Uid == uint32(os.Getuid())
}
func posixSingleLink(info os.FileInfo) bool {
	stat, ok := info.Sys().(*syscall.Stat_t)
	return ok && stat.Nlink == 1
}
func posixExists(path string) bool { _, err := os.Lstat(path); return !os.IsNotExist(err) }
func posixHasSymlink(path string) bool {
	current := "/"
	for _, part := range strings.Split(path, "/") {
		if part == "" {
			continue
		}
		current += part
		info, err := os.Lstat(current)
		if err == nil && info.Mode()&os.ModeSymlink != 0 {
			return true
		}
		current += "/"
	}
	return false
}

// The installer shell validates field shape and timestamp only. Owned upgrade
// applies its stronger receipt validation through Service.Preview afterwards.
func posixReceiptSchema(wire []byte) (map[string]string, error) {
	if !strings.HasSuffix(string(wire), "\n") {
		return nil, local.ErrUnsafe
	}
	lines := strings.Split(strings.TrimSuffix(string(wire), "\n"), "\n")
	if len(lines) != len(receiptFields) {
		return nil, local.ErrUnsafe
	}
	allowed := map[string]bool{}
	for _, field := range receiptFields {
		allowed[field] = true
	}
	values := map[string]string{}
	for _, line := range lines {
		key, value, ok := strings.Cut(line, "=")
		_, duplicate := values[key]
		if !ok || !allowed[key] || duplicate {
			return nil, local.ErrUnsafe
		}
		// awk -F= reads the second field, matching the shell schema.
		value, _, _ = strings.Cut(value, "=")
		values[key] = value
	}
	if !installedPattern.MatchString(values["installedAt"]) {
		return nil, local.ErrUnsafe
	}
	return values, nil
}

// Cleanup is authorized only for the same object with the exact staged bytes.
func posixRemoveFile(directory local.AnchoredDirectory, name string, identity os.FileInfo, wire []byte, limit int) error {
	if directory.StillAtPath() != nil {
		return local.ErrReplaced
	}
	current, err := directory.Lstat(name)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil || !os.SameFile(identity, current) {
		return local.ErrReplaced
	}
	actual, err := directory.ReadFile(name, limit)
	if err != nil || !bytes.Equal(actual, wire) {
		return local.ErrReplaced
	}
	return directory.Remove(name)
}
func posixRemoveLock(directory local.AnchoredDirectory, identity os.FileInfo) error {
	if directory.StillAtPath() != nil {
		return local.ErrReplaced
	}
	current, err := directory.Lstat(lockName)
	if err != nil || !os.SameFile(identity, current) || !current.IsDir() {
		return local.ErrReplaced
	}
	return directory.Remove(lockName)
}
