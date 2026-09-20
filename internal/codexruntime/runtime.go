// Package codexruntime installs thin Axiom entrypoints at Codex user scope.
package codexruntime

import (
	"context"
	"crypto/sha256"
	"embed"
	"encoding/hex"
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

var legacySkillDigests = map[string]string{
	"axiom-project-configure": "87dc55d4a459d4abf70bb53c7f91695da9cbae18da3a5afd6112f964062b5b9c",
	"axiom-project-show":      "594fc02985f5884c780b2c584c6774424bb63c5234ee2f32e5800002cd3c5c02",
	"axiom-work-item-create":  "556fff5e6b38d204bd4acd6a37f74a23da33c88409a7fbc91c9ccfaf3d70c493",
	"axiom-work-item-run":     "35bf4d66efa1a182479579f882a408f9b394c32e5b0e02d7dfbf8ef9d129c59b",
	"axiom-work-item-status":  "009ab0f59c2992c79ca7732a2d451f2b652e4afd94697afd02572bb75ec3db0b",
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
	for _, name := range skillNames {
		content, err := fs.ReadFile(skillFiles, "skills/"+name+"/SKILL.md")
		if err != nil {
			return Result{Status: Failed, Category: "codex_skill_package_invalid"}
		}
		if !installableOne(s.root, name, content) {
			return Result{Status: Failed, Category: "codex_skill_conflict"}
		}
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
		changedOne, createdOne, err := installOne(s.root, name, content)
		if err != nil {
			rollback(created)
			return Result{Status: Failed, Category: "codex_skill_conflict"}
		}
		if createdOne {
			created = append(created, filepath.Join(s.root, name))
		}
		if changedOne {
			changed = true
		}
	}
	if !changed {
		return Result{Status: Unchanged, Category: "codex_already_configured"}
	}
	return Result{Status: Applied, Category: "codex_configured"}
}

func installableOne(root, name string, content []byte) bool {
	info, err := os.Lstat(filepath.Join(root, name))
	if os.IsNotExist(err) {
		return true
	}
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return false
	}
	return matchesInstalled(root, name, content) || matchesLegacyInstalled(root, name)
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

func installOne(root, name string, content []byte) (bool, bool, error) {
	directory := filepath.Join(root, name)
	info, err := os.Lstat(directory)
	if err == nil {
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return false, false, errors.New("skill conflict")
		}
		if matchesInstalled(root, name, content) {
			return false, false, nil
		}
		if !matchesLegacyInstalled(root, name) || replaceKnownSkill(directory, name, content) != nil {
			return false, false, errors.New("skill conflict")
		}
		return true, false, nil
	}
	if !os.IsNotExist(err) {
		return false, false, err
	}
	if err := os.Mkdir(directory, 0o700); err != nil {
		return false, false, err
	}
	file, err := os.OpenFile(filepath.Join(directory, "SKILL.md"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		_ = os.Remove(directory)
		return false, false, err
	}
	_, writeErr := file.Write(content)
	closeErr := file.Close()
	if writeErr != nil || closeErr != nil {
		_ = os.RemoveAll(directory)
		return false, false, errors.New("skill write failed")
	}
	return true, true, nil
}

func matchesLegacyInstalled(root, name string) bool {
	content, ok := singleSkillContent(filepath.Join(root, name))
	if !ok {
		return false
	}
	digest := sha256.Sum256(content)
	return hex.EncodeToString(digest[:]) == legacySkillDigests[name]
}

func replaceKnownSkill(directory, name string, content []byte) error {
	root, err := os.OpenRoot(directory)
	if err != nil {
		return err
	}
	defer root.Close()
	const temporary = ".axiom-skill-update"
	file, err := root.OpenFile(temporary, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	_, writeErr := file.Write(content)
	syncErr := file.Sync()
	closeErr := file.Close()
	if writeErr != nil || syncErr != nil || closeErr != nil {
		_ = root.Remove(temporary)
		return errors.New("skill update write failed")
	}
	defer root.Remove(temporary)
	current, err := root.ReadFile("SKILL.md")
	if err != nil {
		return err
	}
	digest := sha256.Sum256(current)
	if hex.EncodeToString(digest[:]) != legacySkillDigests[name] {
		return errors.New("skill changed during update")
	}
	return root.Rename(temporary, "SKILL.md")
}

func singleSkillContent(directory string) ([]byte, bool) {
	entries, err := os.ReadDir(directory)
	if err != nil || len(entries) != 1 || entries[0].Name() != "SKILL.md" || entries[0].Type()&os.ModeSymlink != 0 {
		return nil, false
	}
	content, err := os.ReadFile(filepath.Join(directory, "SKILL.md"))
	return content, err == nil
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
