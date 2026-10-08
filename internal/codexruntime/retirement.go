package codexruntime

import (
	"errors"
	"os"
	"slices"
)

// retiredOwnedIn permits only receipt-attested, known bytes, or an empty
// receipt-attested directory left after an interrupted removal. Links, extra
// files, and edits are always conflicts. The root receipt remains historical
// until every retirement and canonical publication completes.
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
	return receipt == ReceiptLegacy && attested[name] && err == nil && len(entries) == 0 && childStillAt(root, child, name) == nil
}

func (i integration) removeRetiredIn(root *os.Root, rootPath, name, expected string, verify func() error) error {
	if !slices.Contains(retiredSkillNames, name) || verify() != nil || !i.retiredOwnedIn(root, rootPath, name) {
		return ErrUpgradeConflict
	}
	if _, err := root.Lstat(name); os.IsNotExist(err) {
		return nil
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
	if verify() != nil {
		return errors.New("skill retirement publication uncertain")
	}
	return nil
}

// RemoveRetiredSkill removes an obsolete entry using anchored ownership proof.
func (u *UpgradeSession) RemoveRetiredSkill(name, expected string) error {
	return u.integration.removeRetiredIn(u.root, u.rootPath, name, expected, u.StillAtPath)
}
