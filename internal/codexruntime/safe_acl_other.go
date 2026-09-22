//go:build !darwin && !linux

package codexruntime

import (
	"errors"
	"os"
)

func checkPrivateACL(*os.File) error {
	return errors.New("ACL inspection unavailable")
}
