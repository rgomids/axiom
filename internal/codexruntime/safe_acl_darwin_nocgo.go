//go:build darwin && !cgo

package codexruntime

import (
	"errors"
	"os"
)

func checkPrivateACL(*os.File) error {
	return errors.New("ACL inspection unavailable")
}
