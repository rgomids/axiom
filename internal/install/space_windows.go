package install

import (
	"github.com/rgomids/axiom/internal/windowsfs"
	"golang.org/x/sys/windows"
)

const binaryName = "axiom.exe"

func supportedWindowsHost() bool {
	return supportedWindowsProduct(windows.RtlGetVersion().ProductType)
}

// supportedWindowsProduct accepts Windows client editions regardless of the
// numeric OS version. VER_NT_WORKSTATION excludes domain controllers and
// Windows Server.
func supportedWindowsProduct(productType byte) bool {
	return productType == 1
}

func statfsAvailable(path string) (uint64, error) { return windowsfs.Available(path) }
