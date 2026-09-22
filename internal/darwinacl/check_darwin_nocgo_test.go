//go:build darwin && !cgo

package darwinacl

import (
	"errors"
	"os"
	"os/exec"
	"testing"
)

func TestCheckPrivateAcceptsNoACLAndDetectsExtendedACL(t *testing.T) {
	directory, err := os.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer directory.Close()
	if err := CheckPrivate(directory); err != nil {
		t.Fatalf("ACL-free directory rejected: %v", err)
	}
	command := exec.Command("/bin/chmod", "+a", "everyone allow read,search", directory.Name())
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("set synthetic ACL: %v: %s", err, output)
	}
	if err := CheckPrivate(directory); !errors.Is(err, errPresent) {
		t.Fatalf("extended ACL not detected: %v", err)
	}
}

func TestCheckPrivateFailsClosedWhenStateCannotBeRead(t *testing.T) {
	directory, err := os.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := directory.Close(); err != nil {
		t.Fatal(err)
	}
	if err := CheckPrivate(directory); err == nil {
		t.Fatal("closed descriptor ACL state accepted")
	}
}
