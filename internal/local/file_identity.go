//go:build unix

package local

import (
	"errors"
	"os"
	"path/filepath"
)

// FileIdentity observes one explicit regular file without following a link at
// the selected path and without reading its content. It binds a portable
// local-file documentation source to the same filesystem object (Issue #231);
// a replaced or relocated file yields a different identity.
func FileIdentity(path string) (string, error) {
	if !filepath.IsAbs(path) {
		return "", errors.New("file path is not absolute")
	}
	clean, stat, err := lstatIdentity(path, isRegular)
	if err != nil {
		return "", errors.New("file is unavailable")
	}
	return identityValue("file:", clean, stat, false), nil
}

// MatchesFileIdentity is the file counterpart of MatchesDirectoryIdentity.
func MatchesFileIdentity(path, recorded string) bool {
	clean, stat, err := lstatIdentity(path, isRegular)
	return err == nil && matchesIdentity("file:", clean, stat, recorded)
}

func isRegular(info os.FileInfo) bool { return info.Mode().IsRegular() }
