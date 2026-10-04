package install

import "testing"

// sameDevice is unknown on Windows; the cross-filesystem case is reported
// unverified there unless a distinct volume is configured explicitly.
func sameDevice(t *testing.T, first, second string) bool {
	t.Helper()
	return first[:2] == second[:2]
}
