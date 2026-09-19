// Package codexruntime installs thin Axiom entrypoints at Codex user scope.
package codexruntime

import (
	"context"
	"embed"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

//go:embed skills/*/SKILL.md
var skillFiles embed.FS

var skillNames = []string{
	"axiom-project-configure",
	"axiom-project-show",
	"axiom-work-item-create",
	"axiom-work-item-run",
	"axiom-work-item-status",
}

type Status string

const (
	Applied   Status = "applied"
	Unchanged Status = "unchanged"
	Ready     Status = "ready"
	Missing   Status = "missing"
	Failed    Status = "failed"
)

type Result struct {
	Status   Status
	Category string
}

type Service struct{ root string }

func New(root string) (Service, error) {
	if !filepath.IsAbs(root) || filepath.Clean(root) == string(filepath.Separator) {
		return Service{}, errors.New("unsafe Codex skill root")
	}
	return Service{root: filepath.Clean(root)}, nil
}

func (s Service) Install(ctx context.Context) Result {
	if err := ctx.Err(); err != nil {
		return Result{Status: Failed, Category: "cancelled"}
	}
	if err := ensureRoot(s.root); err != nil {
		return Result{Status: Failed, Category: "codex_skill_root_unavailable"}
	}
	created := make([]string, 0, len(skillNames))
	changed := false
	for _, name := range skillNames {
		if err := ctx.Err(); err != nil {
			rollback(created)
			return Result{Status: Failed, Category: "cancelled"}
		}
		content, err := fs.ReadFile(skillFiles, "skills/"+name+"/SKILL.md")
		if err != nil {
			rollback(created)
			return Result{Status: Failed, Category: "codex_skill_package_invalid"}
		}
		made, err := installOne(s.root, name, content)
		if err != nil {
			rollback(created)
			return Result{Status: Failed, Category: "codex_skill_conflict"}
		}
		if made {
			created = append(created, filepath.Join(s.root, name))
			changed = true
		}
	}
	if !changed {
		return Result{Status: Unchanged, Category: "codex_already_configured"}
	}
	return Result{Status: Applied, Category: "codex_configured"}
}

func (s Service) Inspect(ctx context.Context) Result {
	if err := ctx.Err(); err != nil {
		return Result{Status: Failed, Category: "cancelled"}
	}
	info, err := os.Lstat(s.root)
	if os.IsNotExist(err) {
		return Result{Status: Missing, Category: "codex_not_configured"}
	}
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return Result{Status: Failed, Category: "codex_skill_root_unavailable"}
	}
	for _, name := range skillNames {
		content, readErr := fs.ReadFile(skillFiles, "skills/"+name+"/SKILL.md")
		if readErr != nil || !matchesInstalled(s.root, name, content) {
			return Result{Status: Missing, Category: "codex_skills_missing_or_changed"}
		}
	}
	return Result{Status: Ready, Category: "codex_ready"}
}

func ensureRoot(root string) error {
	info, err := os.Lstat(root)
	if os.IsNotExist(err) {
		return os.MkdirAll(root, 0o700)
	}
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return errors.New("invalid root")
	}
	return nil
}

func installOne(root, name string, content []byte) (bool, error) {
	directory := filepath.Join(root, name)
	info, err := os.Lstat(directory)
	if err == nil {
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() || !matchesInstalled(root, name, content) {
			return false, errors.New("skill conflict")
		}
		return false, nil
	}
	if !os.IsNotExist(err) {
		return false, err
	}
	if err := os.Mkdir(directory, 0o700); err != nil {
		return false, err
	}
	file, err := os.OpenFile(filepath.Join(directory, "SKILL.md"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		_ = os.Remove(directory)
		return false, err
	}
	_, writeErr := file.Write(content)
	closeErr := file.Close()
	if writeErr != nil || closeErr != nil {
		_ = os.RemoveAll(directory)
		return false, errors.New("skill write failed")
	}
	return true, nil
}

func matchesInstalled(root, name string, expected []byte) bool {
	directory := filepath.Join(root, name)
	entries, err := os.ReadDir(directory)
	if err != nil || len(entries) != 1 || entries[0].Name() != "SKILL.md" || entries[0].Type()&os.ModeSymlink != 0 {
		return false
	}
	actual, err := os.ReadFile(filepath.Join(directory, "SKILL.md"))
	return err == nil && string(actual) == string(expected)
}

func rollback(paths []string) {
	for index := len(paths) - 1; index >= 0; index-- {
		_ = os.RemoveAll(paths[index])
	}
}
