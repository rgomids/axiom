//go:build darwin && !cgo

package codexruntime

import (
	"os"

	"github.com/rgomids/axiom/internal/darwinacl"
)

func checkPrivateACL(file *os.File) error {
	return darwinacl.CheckPrivate(file)
}
