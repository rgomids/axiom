package main

import (
	"bytes"
	"os"
	"sync"
	"testing"

	"golang.org/x/sys/windows"
)

// Windows chmod cannot revoke reads. Deny FILE_READ_DATA to the current token
// while retaining READ_CONTROL and file attributes for the no-write snapshot.
func workflow303DenyRead(t *testing.T, path string) {
	t.Helper()
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	original, err := windows.GetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION)
	if err != nil {
		t.Fatal(err)
	}
	originalACL, _, err := original.DACL()
	if err != nil {
		t.Fatal(err)
	}
	control, _, err := original.Control()
	if err != nil {
		t.Fatal(err)
	}
	restoreFlags := windows.SECURITY_INFORMATION(windows.DACL_SECURITY_INFORMATION | windows.UNPROTECTED_DACL_SECURITY_INFORMATION)
	if control&windows.SE_DACL_PROTECTED != 0 {
		restoreFlags = windows.DACL_SECURITY_INFORMATION | windows.PROTECTED_DACL_SECURITY_INFORMATION
	}
	var once sync.Once
	t.Cleanup(func() {
		once.Do(func() {
			if err := windows.SetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, restoreFlags, nil, nil, originalACL, nil); err != nil {
				t.Error(err)
				return
			}
			after, err := os.ReadFile(path)
			if err != nil || !bytes.Equal(before, after) {
				t.Errorf("unreadable fixture bytes changed: %v", err)
			}
		})
	})
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		t.Fatal(err)
	}
	sid := user.User.Sid.String()
	denied, err := windows.SecurityDescriptorFromString("D:P(D;;0x1;;;" + sid + ")(A;;FA;;;" + sid + ")(A;;FA;;;SY)(A;;FA;;;BA)")
	if err != nil {
		t.Fatal(err)
	}
	acl, _, err := denied.DACL()
	if err != nil {
		t.Fatal(err)
	}
	if err := windows.SetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION, nil, nil, acl, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := os.ReadFile(path); !os.IsPermission(err) {
		t.Fatalf("fixture must deny actual read: %v", err)
	}
}

// FileMode on Windows does not encode ACLs, so capture the DACL explicitly.
func workflow303AccessState(t *testing.T, path string) string {
	t.Helper()
	sd, err := windows.GetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION)
	if err != nil {
		t.Fatal(err)
	}
	return sd.String()
}
