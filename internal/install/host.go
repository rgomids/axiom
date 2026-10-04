package install

import "runtime"

// hostRow reports the approved release row of the running host without
// executing any command. Eligibility depends only on the OS family and
// architecture, never on the OS version. A host outside the approved rows
// yields "".
var hostRow = func() string {
	switch runtime.GOOS + "/" + runtime.GOARCH {
	case "darwin/arm64":
		// Any macOS version runs the darwin/arm64 build.
		return "macos-27:darwin:arm64"
	case "linux/amd64", "linux/arm64":
		// Any Linux distribution runs the static linux build.
		return "linux:linux:" + runtime.GOARCH
	case "windows/amd64":
		if supportedWindowsHost() {
			return "windows:windows:amd64"
		}
	}
	return ""
}
