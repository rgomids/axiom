package cli

import (
	"flag"
	"io"
)

const (
	windowsPermissionsRestoreAction = "windows_permissions_restore"
	windowsBackupFlag               = "backup"
	windowsApproveFlag              = "approve"
)

func windowsPermissionsCommand() commandDefinition {
	return group("windows-permissions", "Windows permission repair recovery",
		leaf("restore", "Restore captured permissions with exact approval (Windows only)", windowsPermissionsRestoreAction))
}

func windowsPermissionsFlagSet() *flag.FlagSet {
	set := flag.NewFlagSet(windowsPermissionsRestoreAction, flag.ContinueOnError)
	set.SetOutput(io.Discard)
	set.String(windowsBackupFlag, "", "Printed private repair backup; `<absolute-json-path>`. Must precede --approve.")
	set.String(windowsApproveFlag, "", "Original repair approval digest; `<backup-digest>`. Must follow --backup. This command accepts separate values only, in this exact order.")
	return set
}

// ParseWindowsPermissionsRestore preserves the existing fixed-order syntax.
// Filesystem validation and permission restoration stay in the Windows adapter.
// Flag names are shared with help; Go flag's broader syntax is not adopted.
func ParseWindowsPermissionsRestore(args []string) (backup, approval string, ok bool) {
	command := windowsPermissionsCommand()
	if len(args) < 2 || args[0] != command.name || args[1] != command.children[0].name || !windowsPermissionsSyntax(args[2:], true) {
		return "", "", false
	}

	return args[3], args[5], true
}

func windowsPermissionsSyntax(args []string, complete bool) bool {
	if len(args) > 4 || len(args)%2 != 0 || complete && len(args) != 4 {
		return false
	}
	names := []string{windowsBackupFlag, windowsApproveFlag}
	for index := 0; index < len(args); index += 2 {
		if args[index] != "--"+names[index/2] {
			return false
		}
	}
	return true
}
