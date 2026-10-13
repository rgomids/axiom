package validatorpolicy

import "testing"

func TestPortableAbsoluteIsHostIndependent(t *testing.T) {
	for _, p := range []string{"/synthetic/validator", `C:\tools\go.exe`, "c:/tools/go", `\\server\share\go`} {
		if !portableAbsolute(p) {
			t.Errorf("%q should be portable-absolute", p)
		}
	}
	for _, p := range []string{"go", "./go", `tools\go`, "C:go", ""} {
		if portableAbsolute(p) {
			t.Errorf("%q should not be portable-absolute", p)
		}
	}
	for _, argv0 := range []string{"/bin/sh", `C:\Windows\System32\cmd.exe`} {
		if ValidateDefinition(Definition{FormatVersion: 1, ProjectID: "p", Commands: []Command{{Reference: "r", Argv: []string{argv0}, OutputMax: 1}}}) == nil {
			t.Errorf("shell %q must stay rejected", argv0)
		}
	}
}
