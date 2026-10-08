package windowsfs

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/sys/windows"
)

func repairFixture(t *testing.T) (string, []RepairTarget, string) {
	t.Helper()
	home := t.TempDir()
	root := filepath.Join(home, ".agents")
	skills := filepath.Join(root, "skills")
	other := filepath.Join(skills, "unrelated", "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(other), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(other, []byte("unchanged unrelated skill"), 0o600); err != nil {
		t.Fatal(err)
	}
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		t.Fatal(err)
	}
	sd, err := windows.SecurityDescriptorFromString("O:" + user.User.Sid.String() + "D:P(A;OICI;FA;;;" + user.User.Sid.String() + ")(A;OICI;FA;;;SY)(A;OICI;FA;;;BA)(A;OICI;FA;;;WD)")
	if err != nil {
		t.Fatal(err)
	}
	acl, _, err := sd.DACL()
	if err != nil {
		t.Fatal(err)
	}
	if err := windows.SetNamedSecurityInfo(root, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION, nil, nil, acl, nil); err != nil {
		t.Fatal(err)
	}
	return home, []RepairTarget{{root, false}, {skills, true}}, other
}

func fileSD(t *testing.T, path string) string {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	sd, err := repairSecurity(f)
	if err != nil {
		t.Fatal(err)
	}
	return sd.String()
}

func TestRepairRequiresAuthorityAndPreservesDescendants(t *testing.T) {
	home, targets, other := repairFixture(t)
	before := fileSD(t, other)
	plan, err := PreviewRepair(home, targets)
	if err != nil {
		t.Fatal(err)
	}
	defer plan.Close()
	if len(plan.Changes) != 2 {
		t.Fatalf("changes: %+v", plan.Changes)
	}
	called := false
	if err := plan.Apply("wrong", func([]byte) error { called = true; return nil }); err == nil || called {
		t.Fatal("wrong authority permitted effects")
	}
	var backup []byte
	if err := plan.Apply(plan.Digest, func(wire []byte) error { backup = append([]byte(nil), wire...); return nil }); err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(backup, []byte(plan.Digest)) || !bytes.Contains(backup, []byte("before")) {
		t.Fatal("missing recovery backup")
	}
	if after := fileSD(t, other); after != before {
		t.Fatalf("unrelated descendant ACL changed\nbefore=%s\nafter=%s", before, after)
	}
	wire, err := os.ReadFile(other)
	if err != nil || string(wire) != "unchanged unrelated skill" {
		t.Fatal("unrelated content changed")
	}
	plan.Close()
	second, err := PreviewRepair(home, targets)
	if err != nil {
		t.Fatal(err)
	}
	defer second.Close()
	if len(second.Changes) != 0 {
		t.Fatal("repair not idempotent")
	}
	second.Close()
	if err := RestoreRepair(home, backup, "wrong"); err == nil {
		t.Fatal("restored without authority")
	}
	if err := RestoreRepair(home, backup, plan.Digest); err != nil {
		t.Fatal(err)
	}
	for _, change := range plan.Changes {
		if got := fileSD(t, change.Path); !equivalentRepairDescriptor(got, change.Before) {
			t.Fatalf("original ACL not restored: %s", change.Path)
		}
	}
	if after := fileSD(t, other); after != before {
		t.Fatal("restore propagated to unrelated child")
	}
}

func TestRepairBackupFailureHasNoEffects(t *testing.T) {
	home, targets, _ := repairFixture(t)
	before := fileSD(t, targets[0].Path)
	plan, err := PreviewRepair(home, targets)
	if err != nil {
		t.Fatal(err)
	}
	defer plan.Close()
	err = plan.Apply(plan.Digest, func([]byte) error { return errors.New("disk full") })
	plan.Close()
	if err == nil || fileSD(t, targets[0].Path) != before {
		t.Fatal("backup failure changed ACL")
	}
}

func TestRepairRejectsUnboundedTargets(t *testing.T) {
	home := t.TempDir()
	for _, path := range []string{home, filepath.Join(home, "AppData"), filepath.Join(home, ".agents", "skills", "unrelated"), t.TempDir()} {
		if plan, err := PreviewRepair(home, []RepairTarget{{path, false}}); err == nil {
			plan.Close()
			t.Fatalf("accepted %q", path)
		}
	}
}

func TestRepairRejectsDescriptorDriftBeforeBackup(t *testing.T) {
	home, targets, _ := repairFixture(t)
	plan, err := PreviewRepair(home, targets)
	if err != nil {
		t.Fatal(err)
	}
	defer plan.Close()
	change := plan.Changes[0]
	if err := setRepairDescriptor(plan.files[change.Path], change.After); err != nil {
		t.Fatal(err)
	}
	called := false
	if err := plan.Apply(plan.Digest, func([]byte) error { called = true; return nil }); err == nil || called {
		t.Fatal("descriptor drift was authorized")
	}
}

func TestRepairRefusesReparseTarget(t *testing.T) {
	home := t.TempDir()
	if err := os.Symlink(t.TempDir(), filepath.Join(home, ".agents")); err != nil {
		t.Skipf("symlink privilege unavailable: %v", err)
	}
	if plan, err := PreviewRepair(home, []RepairTarget{{filepath.Join(home, ".agents"), false}}); err == nil {
		plan.Close()
		t.Fatal("reparse target accepted")
	}
}

func TestRepairRollsBackEarlierObjectOnLaterDrift(t *testing.T) {
	home, targets, _ := repairFixture(t)
	plan, err := PreviewRepair(home, targets)
	if err != nil {
		t.Fatal(err)
	}
	defer plan.Close()
	if len(plan.Changes) != 2 {
		t.Fatal("fixture must need two repairs")
	}
	first, second := plan.Changes[0], plan.Changes[1]
	err = plan.Apply(plan.Digest, func([]byte) error {
		return setRepairDescriptor(plan.files[second.Path], second.After)
	})
	if err == nil {
		t.Fatal("later drift was accepted")
	}
	sd, readErr := repairSecurity(plan.files[first.Path])
	if readErr != nil || !equivalentRepairDescriptor(sd.String(), first.Before) {
		t.Fatal("earlier object was not rolled back")
	}
}
