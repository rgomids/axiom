package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/rgomids/axiom/internal/cli"
	"github.com/rgomids/axiom/internal/install"
	"github.com/rgomids/axiom/internal/local"
	"github.com/rgomids/axiom/internal/windowsfs"
)

// prepareWindowsOnboarding validates every required root before binary
// publication, and repairs standard Runtime directories only after consent.
func prepareWindowsOnboarding(target install.Target, input io.Reader, output io.Writer) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	var targets []windowsfs.RepairTarget
	standardCodex := filepath.Join(home, ".agents", "skills")
	if strings.EqualFold(filepath.Clean(target.SkillsRoot), standardCodex) {
		targets = append(targets, windowsfs.RepairTarget{Path: filepath.Dir(standardCodex)}, windowsfs.RepairTarget{Path: standardCodex, Private: true})
	}
	roots := discoverRuntimeRoots()
	if path, err := exec.LookPath("claude"); err == nil && filepath.IsAbs(path) && roots.claudeSkillsError == nil && strings.EqualFold(roots.claudeSkillsRoot, filepath.Join(home, ".claude", "skills")) {
		targets = append(targets, windowsfs.RepairTarget{Path: filepath.Join(home, ".claude")}, windowsfs.RepairTarget{Path: roots.claudeSkillsRoot, Private: true})
	}
	plan, err := windowsfs.PreviewRepair(home, targets)
	if err != nil {
		return err
	}
	defer plan.Close()
	if len(plan.Changes) != 0 {
		fmt.Fprintln(output, "O Axiom precisa ajustar as permissões destas pastas para instalar suas skills:")
		for _, change := range plan.Changes {
			fmt.Fprintf(output, "  - %s\n", change.Path)
		}
		fmt.Fprintln(output, "O ajuste restringe o acesso de outras contas a essas pastas. Suas skills existentes serão preservadas.")
		fmt.Fprintln(output, "Será criado um backup das permissões atuais para desfazer o ajuste, se necessário.")
		reader := bufio.NewReader(io.LimitReader(input, 2048))
		for {
			fmt.Fprintln(output, "Permitir o ajuste e continuar? [S/n] (D: detalhes)")
			line, err := reader.ReadString('\n')
			// EOF is cancellation, never unattended approval of the default.
			if err != nil {
				return fmt.Errorf("permission_repair_declined: instalação cancelada; nenhuma permissão ou binário foi alterado")
			}
			answer := strings.ToLower(strings.TrimSpace(line))
			if answer == "d" {
				for _, change := range plan.Changes {
					fmt.Fprintf(output, "permission_target=%s\npermission_before=%s\npermission_after=%s\n", change.Path, change.Before, change.After)
				}
				fmt.Fprintf(output, "permission_plan=%s\n", plan.Digest)
				continue
			}
			if answer == "" || answer == "s" || answer == "sim" || answer == "y" || answer == "yes" {
				break
			}
			return fmt.Errorf("permission_repair_declined: instalação cancelada; nenhuma permissão ou binário foi alterado")
		}
		backupPath := filepath.Join(home, ".axiom", "windows", "permission-backups")
		backup, err := local.CreateOwnedDirectory(backupPath)
		if err != nil {
			return err
		}
		defer backup.Close()
		name := plan.Digest + ".json"
		err = plan.Apply(plan.Digest, func(wire []byte) error {
			// Exclusive backup publication: an older backup is never replaced.
			stage, err := backup.Stage(".repair-backup-", wire, 0o600)
			if err != nil {
				return err
			}
			defer backup.Remove(stage)
			if err := backup.PublishNew(stage, name); err != nil {
				return err
			}
			if err := backup.Sync(); err != nil {
				return err
			}
			fmt.Fprintf(output, "permission_backup=%s\n", filepath.Join(backupPath, name))
			return nil
		})
		if err != nil {
			return fmt.Errorf("permission_repair_failed: preserve the backup for recovery: %w", err)
		}
		fmt.Fprintln(output, "permission_repair=verified")
	}
	// Release pinned handles before ordinary directory coordination. The next
	// opens revalidate the filesystem boundary rather than trusting the preview.
	plan.Close()
	paths := []string{target.BinaryDir, target.ReceiptDir, target.State.Projects, target.State.State, target.SkillsRoot}
	if path, err := exec.LookPath("claude"); err == nil && filepath.IsAbs(path) {
		if roots.claudeSkillsError != nil {
			return roots.claudeSkillsError
		}
		paths = append(paths, roots.claudeSkillsRoot)
	}
	for _, path := range paths {
		if path == "" {
			continue
		}
		directory, err := local.CreateOwnedDirectory(path)
		if err != nil {
			return fmt.Errorf("onboarding_root=%q: %w", path, err)
		}
		directory.Close()
	}
	return nil
}

func restoreWindowsPermissions(args []string, output io.Writer) error {
	backup, approval, ok := cli.ParseWindowsPermissionsRestore(args)
	if !ok || !filepath.IsAbs(backup) {
		return fmt.Errorf("usage: axiom windows-permissions restore --backup <absolute-json-path> --approve <backup-digest>; see %s", cli.HelpCommand(args))
	}
	path := filepath.Clean(backup)
	directory, err := local.OpenOwnedDirectory(filepath.Dir(path))
	if err != nil {
		return err
	}
	defer directory.Close()
	wire, err := directory.ReadFile(filepath.Base(path), 65536)
	if err != nil {
		return err
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	if err := windowsfs.RestoreRepair(home, wire, approval); err != nil {
		return err
	}
	fmt.Fprintln(output, "permission_restore=verified; onboarding may require permission repair again")
	return nil
}
