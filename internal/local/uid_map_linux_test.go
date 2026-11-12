//go:build linux

package local

import (
	"strings"
	"testing"
)

func TestUIDMapDiagnostic(t *testing.T) {
	for _, tc := range []struct{ name, data, want string }{
		{"initial", "0 0 4294967295\n", "mapped"},
		{"sandbox", "1000 1000 1\n", "unmapped"},
		{"mapped-overflow", "65534 1000 1\n", "mapped"},
		{"multiple", "0 1000 1\n1000 100000 65534\n", "mapped"},
		{"empty", "", "unknown"}, {"malformed", "1000 x 1", "unknown"},
		{"short", "1000 1000", "unknown"}, {"zero", "1000 1000 0", "unknown"},
		{"overflow", "4294967295 1 2", "unknown"},
		{"overlap", "0 1000 2\n1 2000 3", "unknown"},
		{"too-many-ranges", strings.Repeat("1 1 1\n", 341), "unknown"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := uidMapping([]byte(tc.data), 65534); got != tc.want {
				t.Fatalf("got %v want %v", got, tc.want)
			}
		})
	}
}
