package install

import (
	"github.com/rgomids/axiom/internal/windowsfs"
	"golang.org/x/sys/windows"
)

const binaryName = "axiom.exe"

func supportedWindowsHost() bool {
	version := windows.RtlGetVersion()
	return supportedWindowsVersion(version.MajorVersion, version.BuildNumber, version.ProductType)
}

func supportedWindowsVersion(major, build uint32, productType byte) bool {
	// VER_NT_WORKSTATION excludes domain controllers and Windows Server.
	return major == 10 && build >= 17763 && productType == 1
}

func statfsAvailable(path string) (uint64, error) { return windowsfs.Available(path) }
