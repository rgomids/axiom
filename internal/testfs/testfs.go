// Package testfs supplies filesystem assertions for tests.
package testfs

import "os"

// PrivateMode reports whether path itself has exactly the given permissions.
func PrivateMode(path string, mode os.FileMode) bool {
	info, err := os.Lstat(path)
	return err == nil && info.Mode().Perm() == mode
}
