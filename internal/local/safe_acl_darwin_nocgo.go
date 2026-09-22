//go:build darwin && !cgo

package local

import (
	"os"

	"github.com/rgomids/axiom/internal/darwinacl"
)

func checkPrivateACL(file *os.File) error {
	if err := darwinacl.CheckPrivate(file); err == nil {
		return nil
	}
	return ErrUnsafe
}
