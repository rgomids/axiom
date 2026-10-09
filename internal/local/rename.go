package local

import "os"

// RenameNoReplace publishes oldName as newName inside root without ever
// replacing an existing newName: an occupied destination fails the rename.
// Callers own authority and identity checks for root.
func RenameNoReplace(root *os.Root, oldName, newName string) error {
	if !safeEntryName(oldName) || !safeEntryName(newName) {
		return ErrUnsafe
	}
	return renameNoReplace(root, oldName, newName)
}
