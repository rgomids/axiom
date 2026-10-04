//go:build !windows

package install

import (
	"runtime"
	"testing"
)

// The macOS row must not depend on the OS version (Issue #183): hostRow reads
// nothing from the host beyond GOOS/GOARCH, so any macOS release is eligible.
func TestHostRowDependsOnlyOnOSFamilyAndArchitecture(t *testing.T) {
	want := map[string]string{
		"darwin/arm64": "macos-27:darwin:arm64",
		"linux/amd64":  "linux:linux:amd64",
		"linux/arm64":  "linux:linux:arm64",
	}[runtime.GOOS+"/"+runtime.GOARCH]
	if got := defaultHostRow(); got != want {
		t.Fatalf("hostRow() = %q; want %q", got, want)
	}
}
