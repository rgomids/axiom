package windowsfs

import (
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/sys/windows"
)

func TestMkdirPrivateAndExclusive(t *testing.T) {
	path := t.TempDir()
	root, err := os.OpenRoot(path)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	for _, name := range []string{"private", `private\nested`, "private/slash"} {
		if err := Mkdir(root, name); err != nil {
			t.Fatal(err)
		}
		f, err := root.Open(name)
		if err != nil {
			t.Fatal(err)
		}
		if err := Check(f, true); err != nil {
			t.Fatal(err)
		}
		sd, err := windows.GetSecurityInfo(windows.Handle(f.Fd()), windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION)
		if err != nil {
			t.Fatal(err)
		}
		control, _, err := sd.Control()
		if err != nil || control&windows.SE_DACL_PROTECTED == 0 {
			t.Fatalf("unprotected DACL: %v", err)
		}
		f.Close()
		if err := Mkdir(root, name); !os.IsExist(err) {
			t.Fatalf("existing directory: %v", err)
		}
	}
	for _, name := range []string{`..\escape`, `private\..\escape`, filepath.Join(path, "absolute"), `private:stream`} {
		if err := Mkdir(root, name); err == nil {
			t.Fatalf("accepted %q", name)
		}
	}
}
