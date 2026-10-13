package graphapplication

import (
	"runtime"
	"testing"
)

// skipPOSIXStagePaths marks tests whose fixtures use POSIX absolute stage
// paths; Windows stage-path support is tracked separately (#276 pending).
func skipPOSIXStagePaths(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("POSIX stage-path fixtures; Windows stage paths pending")
	}
}
