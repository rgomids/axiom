package runtimeadapter

import (
	"github.com/rgomids/axiom/internal/windowsfs"
	"os"
)

func effortObjectIdentity(file *os.File) (string, error) { return windowsfs.Identity(file) }

func effortCanonicalTarget(path string) (string, error) {
	canonical, err := windowsfs.Canonical(path)
	if err != nil {
		return "", ErrEffortConfigurationUnproven
	}
	return canonical, nil
}
