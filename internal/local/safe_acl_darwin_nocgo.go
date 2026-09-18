//go:build darwin && !cgo

package local

import "os"

// No verified ACL inspection is available without cgo on macOS.
func checkPrivateACL(*os.File) error { return ErrUnsafe }
