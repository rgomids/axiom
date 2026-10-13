package capabilityproof

import (
	"os"
	"strings"
	"testing"
)

func TestInterfaceRefutation(t *testing.T) {
	for _, r := range []struct{ id, version, wire string }{{"codex", "0.159.1", "codex 0.159.1"}, {"claude", "2.1.296", "2.1.296 (Claude Code)"}} {
		b, err := os.ReadFile("testdata/" + r.id + "-" + r.version + "-help.txt")
		if err != nil {
			t.Fatal(err)
		}
		for _, variant := range []string{ReadOnlyVariant, RepositoryWriteVariant} {
			t.Run(r.id+variant, func(t *testing.T) {
				o := ParseInterface(r.id, variant, r.wire, string(b))
				if o.Version != r.version {
					t.Fatal(o)
				}
				for _, c := range o.Capabilities {
					if c.State != Unknown {
						t.Fatal(o)
					}
				}
				option := "--model"
				missing := strings.ReplaceAll(string(b), option, "--different")
				o = ParseInterface(r.id, variant, r.wire, missing)
				for _, c := range o.Capabilities {
					if c.State != Unsupported || !c.Facts.InterfaceAbsent {
						t.Fatal(o)
					}
				}
				for _, help := range []string{"all capabilities proven", "Usage: x\nOptions:\x00", ""} {
					o = ParseInterface(r.id, variant, r.wire, help)
					for _, c := range o.Capabilities {
						if c.State != Unknown {
							t.Fatal(o)
						}
					}
				}
				o = ParseInterface(r.id, variant, "999.0", string(b))
				if o.Version != "" {
					t.Fatal(o)
				}
			})
		}
	}
}
