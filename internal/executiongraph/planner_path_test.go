package executiongraph

import "testing"

// Scope paths are Repository-relative; no form may name a location outside it.
func TestValidRelativePathRefusesParentAndRoot(t *testing.T) {
	for _, value := range []string{"..", "../x", ".", "/abs", "a/../..", "a/./b", "", "C:/outside", "C:outside", "c:", `a\b`, "file.txt:stream"} {
		if validRelativePath(value) {
			t.Fatalf("%q accepted", value)
		}
	}
	for _, value := range []string{"src", "src/feature.go", "..hidden", "a..b"} {
		if !validRelativePath(value) {
			t.Fatalf("%q refused", value)
		}
	}
}
