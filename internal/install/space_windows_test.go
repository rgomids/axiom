package install

import "testing"

func TestSupportedWindowsProduct(t *testing.T) {
	for _, tc := range []struct {
		name    string
		product byte
		want    bool
	}{
		{"client", 1, true},
		{"server", 3, true},
		{"domain controller", 2, false},
		{"unknown product", 0, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := supportedWindowsProduct(tc.product); got != tc.want {
				t.Fatalf("supported = %v; want %v", got, tc.want)
			}
		})
	}
}
