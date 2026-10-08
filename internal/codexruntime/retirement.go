package codexruntime

import (
	"errors"
	"os"
	"slices"
)

// retirementProofName is the private stage that carries a retired entry's
// verified SKILL.md bytes while it is removed. It is written before the first
// deletion, so an interruption between removing SKILL.md and removing the
// directory never destroys the only ownership proof; it is removed last.
func retirementProofName(name string) string {
	return UpgradeStagePrefix + "retire." + name
}

// retirementProofIn reports whether root holds Axiom's retirement proof for
// name: a private regular file whose bytes are a known revision of name.
func (i integration) retirementProofIn(root *os.Root, name string) bool {
	content, ok := privateRegularFileIn(root, retirementProofName(name))
	return ok && int64(len(content)) <= maxUpgradeSkillBytes && i.knownDigest(name, digestOf(content))
}

// retiredOwnedIn permits only receipt-attested, known bytes, or an empty
// directory left after an interrupted removal that a recognized receipt or
// Axiom's retirement proof attests by name. Links, extra files, and edits are
// always conflicts. The root receipt remains historical until every
// retirement and canonical publication completes.
func (i integration) retiredOwnedIn(root *os.Root, rootPath, name string) bool {
	_, err := root.Lstat(name)
	if os.IsNotExist(err) {
		return true
	}
	wire, ok := privateRegularFileIn(root, receiptName)
	present := true
	if _, err := root.Lstat(receiptName); os.IsNotExist(err) {
		present = false
	}
	receipt, attested := i.receiptState(rootPath, present, ok, wire)
	if receipt == ReceiptUnrecognized || receipt == ReceiptUnsafe {
		return false
	}
	child, err := privateChild(root, name)
	if err != nil {
		return false
	}
	defer child.Close()
	content, ok := singleSkillContentIn(child)
	if ok {
		return i.knownDigest(name, digestOf(content)) && childStillAt(root, child, name) == nil
	}
	directory, err := child.Open(".")
	if err != nil {
		return false
	}
	defer directory.Close()
	entries, err := directory.ReadDir(-1)
	if err != nil || len(entries) != 0 || childStillAt(root, child, name) != nil {
		return false
	}
	return receipt == ReceiptLegacy && attested[name] || i.retirementProofIn(root, name)
}

// retiredOwned applies retiredOwnedIn through a freshly anchored root, for
// read-only inspection outside an install or upgrade session.
func (s Service) retiredOwned(name string) bool {
	root, identity, err := anchoredRoot(s.root, false)
	if err != nil {
		return false
	}
	defer root.Close()
	return identity.verify(root) == nil && s.integration.retiredOwnedIn(root, s.root, name)
}

// retirementPendingIn reports whether name or its retirement proof remains.
func retirementPendingIn(root *os.Root, name string) bool {
	_, entry := root.Lstat(name)
	_, proof := root.Lstat(retirementProofName(name))
	return !os.IsNotExist(entry) || !os.IsNotExist(proof)
}

// recordRetirementIn durably stores content, the verified SKILL.md bytes of
// name, as its retirement proof. An existing proof is reused only when it is
// exactly those bytes; anything else at that name is a conflict.
func (i integration) recordRetirementIn(root *os.Root, name string, content []byte, verify func() error) error {
	proof := retirementProofName(name)
	if existing, ok := privateRegularFileIn(root, proof); ok {
		if string(existing) != string(content) {
			return ErrUpgradeConflict
		}
		return verify()
	} else if _, err := root.Lstat(proof); !os.IsNotExist(err) {
		return ErrUpgradeConflict
	}
	if err := verify(); err != nil {
		return err
	}
	file, err := root.OpenFile(proof, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return ErrUpgradeConflict
	}
	written, writeErr := file.Write(content)
	syncErr := file.Sync()
	closeErr := file.Close()
	if writeErr != nil || syncErr != nil || closeErr != nil || written != len(content) {
		_ = root.Remove(proof)
		return errors.New("skill retirement proof write failed")
	}
	syncRootObject(root)
	if !matchesPrivateFileIn(root, proof, content) {
		return ErrUpgradeConflict
	}
	return verify()
}

// clearRetirementProofIn removes name's proof once name itself is gone. A
// file at the proof name that is not a known revision is never removed.
func (i integration) clearRetirementProofIn(root *os.Root, name string, verify func() error) error {
	proof := retirementProofName(name)
	if _, err := root.Lstat(proof); os.IsNotExist(err) {
		return nil
	}
	if _, err := root.Lstat(name); !os.IsNotExist(err) || !i.retirementProofIn(root, name) {
		return ErrUpgradeConflict
	}
	if err := verify(); err != nil {
		return err
	}
	if err := root.Remove(proof); err != nil {
		return err
	}
	syncRootObject(root)
	return nil
}

func (i integration) removeRetiredIn(root *os.Root, rootPath, name, expected string, verify func() error) error {
	if !slices.Contains(retiredSkillNames, name) || verify() != nil || !i.retiredOwnedIn(root, rootPath, name) {
		return ErrUpgradeConflict
	}
	if _, err := root.Lstat(name); os.IsNotExist(err) {
		return i.clearRetirementProofIn(root, name, verify)
	}
	child, err := privateChild(root, name)
	if err != nil {
		return ErrUpgradeConflict
	}
	defer child.Close()
	content, present := singleSkillContentIn(child)
	if present {
		if expected != "" && digestOf(content) != expected {
			return ErrUpgradeConflict
		}
		if verify() != nil || childStillAt(root, child, name) != nil || !i.retiredOwnedIn(root, rootPath, name) {
			return ErrUpgradeConflict
		}
		if err := i.recordRetirementIn(root, name, content, verify); err != nil {
			return err
		}
		if current, ok := singleSkillContentIn(child); !ok || string(current) != string(content) || childStillAt(root, child, name) != nil {
			return ErrUpgradeConflict
		}
		if err := child.Remove("SKILL.md"); err != nil {
			return err
		}
		syncRootObject(child)
	}
	if verify() != nil || childStillAt(root, child, name) != nil {
		return ErrTargetReplaced
	}
	if err := root.Remove(name); err != nil {
		return err
	}
	syncRootObject(root)
	if err := i.clearRetirementProofIn(root, name, verify); err != nil {
		return err
	}
	if verify() != nil {
		return errors.New("skill retirement publication uncertain")
	}
	return nil
}

// RemoveRetiredSkill removes an obsolete entry using anchored ownership proof.
func (u *UpgradeSession) RemoveRetiredSkill(name, expected string) error {
	return u.integration.removeRetiredIn(u.root, u.rootPath, name, expected, u.StillAtPath)
}
