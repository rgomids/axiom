package install

import "testing"

func TestSupportedWindowsVersion(t *testing.T) {
	for _, tc := range []struct {
		name         string
		major, build uint32
		product      byte
		want         bool
	}{
		{"Windows 10 1809", 10, 17763, 1, true},
		{"Windows 11", 10, 22621, 1, true},
		{"old client", 10, 17134, 1, false},
		{"Server 2022", 10, 20348, 3, false},
		{"domain controller", 10, 20348, 2, false},
		{"unknown product", 10, 22621, 0, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := supportedWindowsVersion(tc.major, tc.build, tc.product); got != tc.want {
				t.Fatalf("supported = %v; want %v", got, tc.want)
			}
		})
	}
}
