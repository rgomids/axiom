package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/compatibility"
	"github.com/rgomids/axiom/internal/install"
	"github.com/rgomids/axiom/internal/windowsfs"
	"golang.org/x/sys/windows"
)

func onboardingFixture(t *testing.T) (install.Target, string, string) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("USERPROFILE", home)
	t.Setenv("CLAUDE_CONFIG_DIR", filepath.Join(home, "claude-isolated"))
	t.Setenv("PATH", "")
	skills := filepath.Join(home, ".agents", "skills")
	other := filepath.Join(skills, "unrelated", "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(other), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(other, []byte("unrelated"), 0o600); err != nil {
		t.Fatal(err)
	}
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		t.Fatal(err)
	}
	sd, err := windows.SecurityDescriptorFromString("D:P(A;OICI;FA;;;" + user.User.Sid.String() + ")(A;OICI;FA;;;SY)(A;OICI;FA;;;BA)(A;OICI;FA;;;WD)")
	if err != nil {
		t.Fatal(err)
	}
	acl, _, err := sd.DACL()
	if err != nil {
		t.Fatal(err)
	}
	if err := windows.SetNamedSecurityInfo(filepath.Join(home, ".agents"), windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION, nil, nil, acl, nil); err != nil {
		t.Fatal(err)
	}
	target := install.Target{BinaryDir: filepath.Join(home, ".axiom", "windows", "bin"), ReceiptDir: filepath.Join(home, ".axiom", "windows", "install"), SkillsRoot: skills, State: compatibility.Roots{Projects: filepath.Join(home, ".axiom", "projects"), State: filepath.Join(home, ".axiom", "windows", "state"), Skills: skills}}
	plan, err := windowsfs.PreviewRepair(home, []windowsfs.RepairTarget{{Path: filepath.Join(home, ".agents")}, {Path: skills, Private: true}})
	if err != nil {
		t.Fatal(err)
	}
	digest := plan.Digest
	plan.Close()
	return target, digest, other
}

func TestWindowsOnboardingDeclinedRepairPublishesNothing(t *testing.T) {
	target, _, other := onboardingFixture(t)
	var output bytes.Buffer
	err := prepareWindowsOnboarding(target, strings.NewReader("no\n"), &output)
	if err == nil || !strings.Contains(err.Error(), "permission_repair_declined") {
		t.Fatalf("result: %v", err)
	}
	if _, err := os.Lstat(target.BinaryDir); !os.IsNotExist(err) {
		t.Fatal("declined repair created installation")
	}
	if wire, err := os.ReadFile(other); err != nil || string(wire) != "unrelated" {
		t.Fatal("unrelated content changed")
	}
}

func TestWindowsOnboardingApprovedRepairPreparesStandardRoots(t *testing.T) {
	target, digest, other := onboardingFixture(t)
	var output bytes.Buffer
	if err := prepareWindowsOnboarding(target, strings.NewReader("REPAIR "+digest+"\n"), &output); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "permission_repair=verified") || !strings.Contains(output.String(), "permission_backup=") {
		t.Fatal("missing repair result")
	}
	for _, path := range []string{target.BinaryDir, target.ReceiptDir, target.State.Projects, target.State.State, target.SkillsRoot} {
		if info, err := os.Stat(path); err != nil || !info.IsDir() {
			t.Fatalf("missing root %s: %v", path, err)
		}
	}
	if wire, err := os.ReadFile(other); err != nil || string(wire) != "unrelated" {
		t.Fatal("unrelated content changed")
	}
	output.Reset()
	if err := prepareWindowsOnboarding(target, strings.NewReader(""), &output); err != nil || strings.Contains(output.String(), "REPAIR ") {
		t.Fatalf("not idempotent: %v %s", err, output.String())
	}
}
