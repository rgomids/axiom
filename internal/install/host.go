package install

import (
	"bufio"
	"bytes"
	"os"
	"regexp"
	"runtime"
	"strings"
)

var productVersionPattern = regexp.MustCompile(`<key>ProductVersion</key>\s*<string>([0-9.]+)</string>`)

// hostRow reports the approved release row of the running host without
// executing any command. A host outside the approved rows yields "".
var hostRow = func() string {
	switch runtime.GOOS + "/" + runtime.GOARCH {
	case "darwin/arm64":
		wire, err := os.ReadFile("/System/Library/CoreServices/SystemVersion.plist")
		if match := productVersionPattern.FindSubmatch(wire); err == nil && match != nil && string(match[1]) == "27.0" {
			return "macos-27:darwin:arm64"
		}
	case "linux/amd64", "linux/arm64":
		wire, err := os.ReadFile("/etc/os-release")
		if err != nil {
			return ""
		}
		values := map[string]string{}
		scanner := bufio.NewScanner(bytes.NewReader(wire))
		for scanner.Scan() {
			if key, value, ok := strings.Cut(scanner.Text(), "="); ok {
				values[key] = strings.Trim(value, `"`)
			}
		}
		if values["ID"] == "ubuntu" && values["VERSION_ID"] == "26.04" {
			return "ubuntu-26.04:linux:" + runtime.GOARCH
		}
	}
	return ""
}
