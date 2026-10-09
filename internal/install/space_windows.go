package install

import (
	"github.com/rgomids/axiom/internal/windowsfs"
	"golang.org/x/sys/windows"
)

const binaryName = "axiom.exe"

func supportedWindowsHost() bool {
	return supportedWindowsProduct(windows.RtlGetVersion().ProductType)
}

// supportedWindowsProduct accepts workstation and member Server regardless of
// numeric OS version. Domain controllers and unknown products fail closed.
func supportedWindowsProduct(productType byte) bool {
	return productType == 1 || productType == 3
}

func statfsAvailable(path string) (uint64, error) { return windowsfs.Available(path) }
