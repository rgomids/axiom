//go:build darwin

package local

import (
	"os"

	"golang.org/x/sys/unix"
)

func renameNoReplace(root *os.Root, oldName, newName string) error {
	dir, err := root.Open(".")
	if err != nil {
		return err
	}
	defer dir.Close()
	fd := int(dir.Fd())
	return unix.RenameatxNp(fd, oldName, fd, newName, unix.RENAME_EXCL)
}
