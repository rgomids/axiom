package codexruntime

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// UpgradeStagePrefix names private staging files that an authorized binary
// upgrade leaves inside an Axiom skill directory only when interrupted.
const UpgradeStagePrefix = ".axiom-upgrade-skill."

const maxUpgradeSkillBytes = 1 << 20

// ErrUpgradeConflict reports Axiom skill state that an upgrade must not touch.
var ErrUpgradeConflict = errors.New("codex skill set is not upgradeable")

// UpgradeSkill is a read-only observation of one Axiom-owned skill directory.
type UpgradeSkill struct {
	Name      string
	Directory bool
	SHA256    string
	Owned     bool
	Leftovers []string
}

// SkillSetReceiptName is the Codex skill-set receipt inside the skill root.
const SkillSetReceiptName = receiptName

// UpgradeInventory is the skill state an upgrade plans against. Configured is
// false only when no Axiom skill directory and no skill-set receipt exist.
// ReceiptSHA256 is the present receipt's digest ("unsafe" for anything that
// is not a bounded private regular file); ReceiptOwned reports that it is this
// binary's receipt or a receipt an earlier Axiom revision wrote. Leftovers are
// interrupted receipt stages directly inside the skill root.
type UpgradeInventory struct {
	Configured    bool
	Receipt       bool
	ReceiptSHA256 string
	ReceiptOwned  bool
	Leftovers     []string
	Skills        []UpgradeSkill
}

// CurrentReceipt is the Codex skill-set receipt of the skill set embedded in
// this binary, derived from this binary's own runtime constants.
func CurrentReceipt() ([]byte, error) { return receiptBytes() }

// EmbeddedSkillDigests reports the SKILL.md digest of every skill this binary
// publishes.
func EmbeddedSkillDigests() (map[string]string, error) {
	revision, err := currentRevision()
	if err != nil {
		return nil, err
	}
	return revision.skills, nil
}

// InspectUpgrade reads the Axiom skill directories without creating the root,
// taking the install lock, or modifying anything. Owned reports content equal
// to this binary's skill or a known legacy digest; callers may hold stronger
// ownership proofs. Unknown entries and unsafe files are a conflict.
func (s Service) InspectUpgrade(ctx context.Context) (UpgradeInventory, error) {
	if err := ctx.Err(); err != nil {
		return UpgradeInventory{}, err
	}
	inventory := UpgradeInventory{Skills: make([]UpgradeSkill, 0, len(skillNames))}
	if _, err := os.Lstat(s.root); os.IsNotExist(err) {
		for _, name := range ownershipSkillNames() {
			inventory.Skills = append(inventory.Skills, UpgradeSkill{Name: name})
		}
		return inventory, nil
	} else if err != nil || !skillRootDirectory(s.root) {
		return UpgradeInventory{}, ErrUpgradeConflict
	}
	if _, err := os.Lstat(filepath.Join(s.root, ".axiom-skill-set-receipt-stage")); !os.IsNotExist(err) {
		return UpgradeInventory{}, ErrUpgradeConflict
	}
	if _, err := os.Lstat(filepath.Join(s.root, receiptName)); err == nil {
		inventory.Receipt, inventory.Configured = true, true
		inventory.ReceiptSHA256 = "unsafe"
		if privateRegularFile(filepath.Join(s.root, receiptName)) {
			if content, ok := readBoundedSkill(filepath.Join(s.root, receiptName)); ok {
				current, err := s.integration.receipt(s.root)
				inventory.ReceiptSHA256 = digestOf(content)
				inventory.ReceiptOwned = err == nil && string(current) == string(content) || s.integration.matchesLegacyReceipt(s.root)
			}
		}
	} else if !os.IsNotExist(err) {
		return UpgradeInventory{}, ErrUpgradeConflict
	}
	entries, err := os.ReadDir(s.root)
	if err != nil {
		return UpgradeInventory{}, ErrUpgradeConflict
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), UpgradeStagePrefix) {
			path := filepath.Join(s.root, entry.Name())
			if !privateRegularFile(path) {
				return UpgradeInventory{}, ErrUpgradeConflict
			}
			inventory.Leftovers = append(inventory.Leftovers, path)
		}
	}
	for _, name := range ownershipSkillNames() {
		skill, err := s.inspectUpgradeSkill(name)
		if err != nil {
			return UpgradeInventory{}, err
		}
		inventory.Configured = inventory.Configured || skill.Directory
		inventory.Skills = append(inventory.Skills, skill)
	}
	return inventory, nil
}

func (s Service) inspectUpgradeSkill(name string) (UpgradeSkill, error) {
	skill := UpgradeSkill{Name: name}
	directory := filepath.Join(s.root, name)
	if _, err := os.Lstat(directory); os.IsNotExist(err) {
		return skill, nil
	} else if err != nil || !privateDirectory(directory) {
		return UpgradeSkill{}, ErrUpgradeConflict
	}
	skill.Directory = true
	entries, err := os.ReadDir(directory)
	if err != nil {
		return UpgradeSkill{}, ErrUpgradeConflict
	}
	for _, entry := range entries {
		path := filepath.Join(directory, entry.Name())
		switch {
		case entry.Name() == "SKILL.md" && privateRegularFile(path):
			content, ok := readBoundedSkill(path)
			if !ok {
				return UpgradeSkill{}, ErrUpgradeConflict
			}
			digest := sha256.Sum256(content)
			skill.SHA256 = hex.EncodeToString(digest[:])
			embedded, embeddedErr := fs.ReadFile(skillFiles, "skills/"+name+"/SKILL.md")
			skill.Owned = embeddedErr == nil && string(embedded) == string(content) || s.integration.knownDigest(name, skill.SHA256)
		case strings.HasPrefix(entry.Name(), UpgradeStagePrefix) && privateRegularFile(path):
			skill.Leftovers = append(skill.Leftovers, path)
		default:
			return UpgradeSkill{}, ErrUpgradeConflict
		}
	}
	if len(entries) == 0 && slices.Contains(retiredSkillNames, name) {
		root, identity, err := anchoredRoot(s.root, false)
		if err == nil {
			skill.Owned = identity.verify(root) == nil && s.integration.retiredOwnedIn(root, s.root, name)
			root.Close()
		}
	}
	return skill, nil
}

// UpgradeSession holds the skill-set lock and an anchored handle to the
// skill root, opened once by LockForUpgrade: every PublishSkill call for this
// session resolves against that same validated directory object (and a
// per-skill child directory opened from it), so a later replacement of the
// root or a skill directory at its pathname cannot redirect a publication,
// and the session's anchor lets every publication detect and refuse such a
// replacement instead of completing in the displaced object (ADR-0005
// property 3, controlled ancestor/leaf replacement).
type UpgradeSession struct {
	lock        *os.File
	root        *os.Root
	anchor      anchor
	integration integration
	rootPath    string
}

// LockForUpgrade takes the same exclusive skill-set lock as Install and opens
// the skill root once, anchored: a real directory, owned by the current
// user, without group/other write, without extended ACL, with every
// container ancestor safe from replacement by another principal. The
// returned session is what every PublishSkill call for this upgrade
// operates through. The lock is taken only while the root is still the object
// visible at its authorized pathname.
func (s Service) LockForUpgrade() (*UpgradeSession, error) {
	root, identity, err := anchoredRoot(s.root, false)
	if err != nil {
		return nil, ErrUpgradeConflict
	}
	info, err := root.Stat(".")
	if err != nil || forbiddenPermissions(info, 0o022) || !ownedByUser(info) {
		root.Close()
		return nil, ErrUpgradeConflict
	}
	directory, err := root.Open(".")
	if err != nil {
		root.Close()
		return nil, ErrUpgradeConflict
	}
	aclErr := checkPrivateACL(directory)
	directory.Close()
	if aclErr != nil {
		root.Close()
		return nil, ErrUpgradeConflict
	}
	if identity.verify(root) != nil {
		root.Close()
		return nil, ErrTargetReplaced
	}
	lock, category := acquireInstallLock(root)
	if category != "" {
		root.Close()
		return nil, errors.New(category)
	}
	if identity.verify(root) != nil {
		_ = lock.Close()
		root.Close()
		return nil, ErrTargetReplaced
	}
	return &UpgradeSession{lock: lock, root: root, anchor: identity, integration: s.integration, rootPath: s.root}, nil
}

// StillAtPath verifies the session's authorized root and ancestors.
func (u *UpgradeSession) StillAtPath() error { return u.anchor.verify(u.root) }

// AvailableBytes reads capacity on the session filesystem.
func (u *UpgradeSession) AvailableBytes() (uint64, error) {
	directory, err := u.root.Open(".")
	if err != nil {
		return 0, err
	}
	defer directory.Close()
	return availableBytes(directory)
}

// Inspect reads upgrade evidence from the session capability, never rootPath.
func (u *UpgradeSession) Inspect(ctx context.Context) (UpgradeInventory, error) {
	if err := ctx.Err(); err != nil {
		return UpgradeInventory{}, err
	}
	if err := u.StillAtPath(); err != nil {
		return UpgradeInventory{}, err
	}
	inventory := UpgradeInventory{Skills: make([]UpgradeSkill, 0, len(skillNames))}
	if _, err := u.root.Lstat(".axiom-skill-set-receipt-stage"); !os.IsNotExist(err) {
		return inventory, ErrUpgradeConflict
	}
	if _, err := u.root.Lstat(receiptName); err == nil {
		inventory.Receipt, inventory.Configured = true, true
		inventory.ReceiptSHA256 = currentReceiptDigestIn(u.root)
		if inventory.ReceiptSHA256 != "unsafe" {
			current, err := u.integration.receipt(u.rootPath)
			inventory.ReceiptOwned = err == nil && digestOf(current) == inventory.ReceiptSHA256 || u.integration.matchesLegacyReceiptIn(u.root, u.rootPath)
		}
	} else if !os.IsNotExist(err) {
		return inventory, ErrUpgradeConflict
	}
	directory, err := u.root.Open(".")
	if err != nil {
		return inventory, ErrUpgradeConflict
	}
	entries, err := directory.ReadDir(-1)
	directory.Close()
	if err != nil {
		return inventory, ErrUpgradeConflict
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), UpgradeStagePrefix) {
			if _, ok := privateRegularFileIn(u.root, entry.Name()); !ok {
				return inventory, ErrUpgradeConflict
			}
			inventory.Leftovers = append(inventory.Leftovers, filepath.Join(u.rootPath, entry.Name()))
		}
	}
	for _, name := range ownershipSkillNames() {
		skill := UpgradeSkill{Name: name}
		if _, err := u.root.Lstat(name); os.IsNotExist(err) {
			inventory.Skills = append(inventory.Skills, skill)
			continue
		}
		child, err := privateChild(u.root, name)
		if err != nil {
			return inventory, ErrUpgradeConflict
		}
		skill, err = u.inspectSkillIn(child, name)
		identityErr := childStillAt(u.root, child, name)
		child.Close()
		if err != nil || identityErr != nil {
			return inventory, ErrUpgradeConflict
		}
		inventory.Configured = true
		inventory.Skills = append(inventory.Skills, skill)
	}
	return inventory, u.StillAtPath()
}

func (u *UpgradeSession) inspectSkillIn(child *os.Root, name string) (UpgradeSkill, error) {
	skill := UpgradeSkill{Name: name, Directory: true}
	directory, err := child.Open(".")
	if err != nil {
		return skill, ErrUpgradeConflict
	}
	entries, err := directory.ReadDir(-1)
	directory.Close()
	if err != nil {
		return skill, ErrUpgradeConflict
	}
	for _, entry := range entries {
		info, err := child.Lstat(entry.Name())
		if err != nil || info.Size() > maxUpgradeSkillBytes {
			return skill, ErrUpgradeConflict
		}
		content, ok := privateRegularFileIn(child, entry.Name())
		if !ok || len(content) > maxUpgradeSkillBytes {
			return skill, ErrUpgradeConflict
		}
		switch {
		case entry.Name() == "SKILL.md":
			skill.SHA256 = digestOf(content)
			embedded, err := fs.ReadFile(skillFiles, "skills/"+name+"/SKILL.md")
			skill.Owned = err == nil && string(embedded) == string(content) || u.integration.knownDigest(name, skill.SHA256)
		case strings.HasPrefix(entry.Name(), UpgradeStagePrefix):
			skill.Leftovers = append(skill.Leftovers, filepath.Join(u.rootPath, name, entry.Name()))
		default:
			return skill, ErrUpgradeConflict
		}
	}
	if len(entries) == 0 && slices.Contains(retiredSkillNames, name) {
		skill.Owned = u.integration.retiredOwnedIn(u.root, u.rootPath, name)
	}
	return skill, nil
}

// Close releases the install lock and the anchored skill-root handle.
func (u *UpgradeSession) Close() {
	_ = u.lock.Close()
	_ = u.root.Close()
}

// RemoveSkillLeftover removes a stray interrupted-stage artifact directly
// inside name's skill directory, through this session's anchored skill-root
// handle and a child directory opened from it once, never by re-deriving
// either by pathname.
func (u *UpgradeSession) RemoveSkillLeftover(name, leftoverName string) error {
	if !slices.Contains(skillNames, name) || filepath.Base(leftoverName) != leftoverName || !strings.HasPrefix(leftoverName, UpgradeStagePrefix) {
		return ErrUpgradeConflict
	}
	if err := u.StillAtPath(); err != nil {
		return err
	}
	child, err := privateChild(u.root, name)
	if err != nil {
		return err
	}
	defer child.Close()
	if _, ok := privateRegularFileIn(child, leftoverName); !ok {
		return ErrUpgradeConflict
	}
	if err := errors.Join(u.StillAtPath(), childStillAt(u.root, child, name)); err != nil {
		return err
	}
	if err := child.Remove(leftoverName); err != nil {
		return err
	}
	return errors.Join(u.StillAtPath(), childStillAt(u.root, child, name))
}

// PublishSkill stages content privately inside name's skill directory
// (creating it if absent and expected is ""), rechecks that SKILL.md still
// has the expected digest, renames, and confirms the published digest,
// entirely through this session's anchored skill-root handle and a child
// directory opened from it once, never by re-deriving either by pathname.
// Before the directory create, the stage, and the rename, the skill root (with
// every ancestor) and the skill directory must still be the objects visible
// at their authorized pathnames, otherwise ErrTargetReplaced is returned with
// nothing committed; the same proof must hold after the rename for the
// publication to be confirmed.
func (u *UpgradeSession) PublishSkill(name string, content []byte, expected string) error {
	known := false
	for _, candidate := range ownershipSkillNames() {
		known = known || candidate == name
	}
	if !known || len(content) == 0 || len(content) > maxUpgradeSkillBytes {
		return ErrUpgradeConflict
	}
	nextDigest := sha256.Sum256(content)
	next := hex.EncodeToString(nextDigest[:])
	if err := u.anchor.verify(u.root); err != nil {
		return err
	}
	if _, err := u.root.Lstat(name); os.IsNotExist(err) && expected == "" {
		if err := u.root.Mkdir(name, 0o700); err != nil {
			return err
		}
		syncRootObject(u.root)
	}
	child, err := privateChild(u.root, name)
	if err != nil {
		return ErrUpgradeConflict
	}
	defer child.Close()
	still := func() error {
		if err := u.anchor.verify(u.root); err != nil {
			return err
		}
		return childStillAt(u.root, child, name)
	}
	if err := still(); err != nil {
		return err
	}
	var entropy [8]byte
	if _, err := rand.Read(entropy[:]); err != nil {
		return err
	}
	stage := UpgradeStagePrefix + hex.EncodeToString(entropy[:])
	file, err := child.OpenFile(stage, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			_ = child.Remove(stage)
		}
	}()
	written, writeErr := file.Write(content)
	chmodErr := file.Chmod(0o600)
	syncErr := file.Sync()
	closeErr := file.Close()
	if err := errors.Join(writeErr, chmodErr, syncErr, closeErr); err != nil || written != len(content) {
		return errors.New("skill stage write failed")
	}
	if staged, ok := readBoundedSkillIn(child, stage); !ok || digestOf(staged) != next {
		return errors.New("skill stage verification failed")
	}
	if current := currentSkillDigestIn(child); current != expected {
		return ErrUpgradeConflict
	}
	if err := still(); err != nil {
		return err
	}
	if err := child.Rename(stage, "SKILL.md"); err != nil {
		return err
	}
	committed = true
	syncRootObject(child)
	if currentSkillDigestIn(child) != next || still() != nil {
		return errors.New("skill publication uncertain")
	}
	return nil
}

// RemoveRootLeftover removes a stray interrupted receipt stage directly inside
// the skill root, through this session's anchored skill-root handle.
func (u *UpgradeSession) RemoveRootLeftover(leftoverName string) error {
	if filepath.Base(leftoverName) != leftoverName || !strings.HasPrefix(leftoverName, UpgradeStagePrefix) {
		return ErrUpgradeConflict
	}
	if err := u.StillAtPath(); err != nil {
		return err
	}
	if _, ok := privateRegularFileIn(u.root, leftoverName); !ok {
		return ErrUpgradeConflict
	}
	if err := u.root.Remove(leftoverName); err != nil {
		return err
	}
	syncRootObject(u.root)
	return u.StillAtPath()
}

// PublishReceipt replaces the skill-set receipt whose digest is expected (""
// for absent) with content, entirely through this session's anchored
// skill-root handle. A present receipt is replaced only while it is still this
// binary's receipt or one an earlier Axiom revision wrote, so a receipt Axiom
// cannot recognize is never overwritten. The stage is private and named with
// UpgradeStagePrefix so an interruption leaves a recoverable leftover.
func (u *UpgradeSession) PublishReceipt(content []byte, expected string) error {
	if len(content) == 0 || len(content) > maxUpgradeSkillBytes {
		return ErrUpgradeConflict
	}
	next := digestOf(content)
	owned := func() bool {
		current, err := u.integration.receipt(u.rootPath)
		return err == nil && matchesPrivateFileIn(u.root, receiptName, current) || u.integration.matchesLegacyReceiptIn(u.root, u.rootPath)
	}
	if err := u.StillAtPath(); err != nil {
		return err
	}
	if currentReceiptDigestIn(u.root) != expected || expected != "" && !owned() {
		return ErrUpgradeConflict
	}
	var entropy [8]byte
	if _, err := rand.Read(entropy[:]); err != nil {
		return err
	}
	stage := UpgradeStagePrefix + "receipt." + hex.EncodeToString(entropy[:])
	file, err := u.root.OpenFile(stage, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			_ = u.root.Remove(stage)
		}
	}()
	written, writeErr := file.Write(content)
	chmodErr := file.Chmod(0o600)
	syncErr := file.Sync()
	closeErr := file.Close()
	if err := errors.Join(writeErr, chmodErr, syncErr, closeErr); err != nil || written != len(content) {
		return errors.New("receipt stage write failed")
	}
	if staged, ok := privateRegularFileIn(u.root, stage); !ok || digestOf(staged) != next {
		return errors.New("receipt stage verification failed")
	}
	if currentReceiptDigestIn(u.root) != expected || expected != "" && !owned() {
		return ErrUpgradeConflict
	}
	if err := u.StillAtPath(); err != nil {
		return err
	}
	if err := u.root.Rename(stage, receiptName); err != nil {
		return err
	}
	committed = true
	syncRootObject(u.root)
	if currentReceiptDigestIn(u.root) != next || u.StillAtPath() != nil {
		return errors.New("receipt publication uncertain")
	}
	return nil
}

// currentReceiptDigestIn is "" for an absent receipt and "unsafe" for
// anything that is not a bounded private regular file.
func currentReceiptDigestIn(root *os.Root) string {
	info, err := root.Lstat(receiptName)
	if os.IsNotExist(err) {
		return ""
	}
	if err != nil || !info.Mode().IsRegular() || info.Size() > maxUpgradeSkillBytes {
		return "unsafe"
	}
	content, ok := privateRegularFileIn(root, receiptName)
	if !ok || len(content) > maxUpgradeSkillBytes {
		return "unsafe"
	}
	return digestOf(content)
}

// currentSkillDigestIn is "" for an absent SKILL.md and "unsafe" for anything
// that is not a private regular file, evaluated through child, the already
// anchored skill directory.
func currentSkillDigestIn(child *os.Root) string {
	info, err := child.Lstat("SKILL.md")
	if os.IsNotExist(err) {
		return ""
	}
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Size() > maxUpgradeSkillBytes {
		return "unsafe"
	}
	content, ok := privateRegularFileIn(child, "SKILL.md")
	if !ok || int64(len(content)) > maxUpgradeSkillBytes {
		return "unsafe"
	}
	return digestOf(content)
}

func readBoundedSkillIn(child *os.Root, name string) ([]byte, bool) {
	info, err := child.Lstat(name)
	if err != nil || !info.Mode().IsRegular() || info.Size() > maxUpgradeSkillBytes {
		return nil, false
	}
	content, err := child.ReadFile(name)
	return content, err == nil && len(content) <= maxUpgradeSkillBytes
}

// syncRootObject fsyncs the anchored directory itself, to surface a reported
// I/O error and make a preceding create/rename/remove durable against the
// directory entry.
func syncRootObject(root *os.Root) {
	if directory, err := root.Open("."); err == nil {
		_ = directory.Sync()
		_ = directory.Close()
	}
}

func readBoundedSkill(path string) ([]byte, bool) {
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Size() > maxUpgradeSkillBytes {
		return nil, false
	}
	content, err := os.ReadFile(path)
	return content, err == nil && len(content) <= maxUpgradeSkillBytes
}

func digestOf(content []byte) string {
	digest := sha256.Sum256(content)
	return hex.EncodeToString(digest[:])
}

func syncPath(path string) {
	if directory, err := os.Open(path); err == nil {
		_ = directory.Sync()
		_ = directory.Close()
	}
}
