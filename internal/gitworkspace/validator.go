package gitworkspace

import (
	"path/filepath"
	"strings"

	"github.com/rgomids/axiom/internal/executiongraph"
)

type ValidationCommand struct {
	Reference string
	Argv      []string
	Env       []string
	OutputMax int
}

// ValidValidationCommand checks the argv-only integration command boundary.
// HostAbsolutePlaceholder returns a host-absolute path ending in name, for
// structural validation of commands persisted on another platform.
func HostAbsolutePlaceholder(name string) string {
	root, _ := filepath.Abs(string(filepath.Separator))
	return filepath.Join(root, name)
}

func ValidValidationCommand(command ValidationCommand) bool {
	if command.Reference == "" || len(command.Reference) > 512 || strings.ContainsAny(command.Reference, "\x00\r\n") || len(command.Argv) == 0 || len(command.Argv) > 64 || !filepath.IsAbs(command.Argv[0]) || command.OutputMax <= 0 || command.OutputMax > executiongraph.MaxCapturedOutputBytes || len(command.Env) > 64 {
		return false
	}
	base := strings.TrimSuffix(strings.ToLower(filepath.Base(command.Argv[0])), ".exe")
	if base == "sh" || base == "bash" || base == "zsh" || base == "fish" || base == "cmd" || base == "powershell" || base == "pwsh" || base == "dash" || base == "ksh" || base == "csh" || base == "tcsh" || base == "ash" || base == "busybox" || base == "env" {
		return false
	}
	for _, argument := range command.Argv {
		if argument == "" || strings.ContainsRune(argument, '\x00') {
			return false
		}
	}
	seen := map[string]bool{}
	for _, item := range command.Env {
		key, _, ok := strings.Cut(item, "=")
		if !ok || !validEnvironmentKey(key) || seen[key] || strings.ContainsRune(item, '\x00') {
			return false
		}
		seen[key] = true
	}
	return true
}

func validEnvironmentKey(value string) bool {
	if value == "" || len(value) > 128 {
		return false
	}
	for index, char := range value {
		if !((char >= 'A' && char <= 'Z') || char == '_' || (index > 0 && char >= '0' && char <= '9')) {
			return false
		}
	}
	return true
}
