//go:build linux

package executiongraph

import (
	"io/fs"
	"os"
	"strings"
	"syscall"
	"testing"
)

func TestProcEntryVanishedOnlyForReapedProcesses(t *testing.T) {
	for _, err := range []error{
		&fs.PathError{Op: "read", Path: "/proc/1/stat", Err: syscall.ESRCH},
		&fs.PathError{Op: "open", Path: "/proc/1/stat", Err: syscall.ENOENT},
		os.ErrNotExist,
	} {
		if !procEntryVanished(err) {
			t.Fatalf("reaped process entry not classified as vanished: %v", err)
		}
	}
	for _, err := range []error{
		nil,
		&fs.PathError{Op: "read", Path: "/proc/1/stat", Err: syscall.EACCES},
		&fs.PathError{Op: "read", Path: "/proc/1/stat", Err: syscall.EIO},
	} {
		if procEntryVanished(err) {
			t.Fatalf("observation failure classified as vanished: %v", err)
		}
	}
}

func TestProcessIdentityKernelTicks(t *testing.T) {
	fields := make([]string, 21)
	for i := range fields {
		fields[i] = "0"
	}
	fields[0], fields[1], fields[2], fields[19] = "S", "1", "123", "456"
	stat := "123 (name with ) parens) " + strings.Join(fields, " ")
	p, err := parseProcessCorrelation(123, stat, "boot-one")
	if err != nil {
		t.Fatal(err)
	}
	if p.StartIdentity != "boot-one:456" || p.GroupIdentity != "boot-one:123" {
		t.Fatal(p)
	}
	fields[19] = "789"
	reused, err := parseProcessCorrelation(123, "123 (other) "+strings.Join(fields, " "), "boot-one")
	if err != nil || *p == *reused {
		t.Fatal(reused, err)
	}

	if _, err := parseProcessCorrelation(123, "malformed", "boot-one"); err == nil {
		t.Fatal("accepted malformed stat")
	}
}
