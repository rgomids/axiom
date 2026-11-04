// Package gitworkspace implements the concrete local Git boundary for S8.
package gitworkspace

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

var (
	ErrInvalidConfiguration = errors.New("invalid Git workspace configuration")
	ErrWorkspaceCollision   = errors.New("Git workspace collision")
	ErrWorkspaceInvalid     = errors.New("Git workspace validation failed")
	ErrForeignMutation      = errors.New("foreign Git workspace mutation")
	ErrIntegrationConflict  = errors.New("Git integration conflict")
	ErrNoResult             = errors.New("child has no successful result")
	// ErrExecutableGitConfiguration reports effective repository configuration
	// that would let Git start an implicit command during a workspace operation.
	ErrExecutableGitConfiguration = errors.New("executable Git configuration rejected")
)

// effectiveUID is the identity that must own every control path.
var effectiveUID = os.Geteuid

// filterSensitiveCommands are the Git commands this adapter runs that can
// convert working-tree content (checkout, staging, patch application or
// working-tree comparison) and therefore consult filter or diff drivers.
var filterSensitiveCommands = map[string]bool{"worktree": true, "add": true, "apply": true, "diff": true}

// Manager holds the shared hardened Git boundary used by integration and delivery.
type Manager struct {
	git, repository, commonDirectory, root string
}

func (m *Manager) confinedWorkspace(value string, mustExist bool) (string, error) {
	if !filepath.IsAbs(value) || filepath.Clean(value) != value || strings.ContainsAny(value, "\x00\r\n") {
		return "", ErrWorkspaceInvalid
	}
	relative, err := filepath.Rel(m.root, value)
	if err != nil || relative == "." || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", ErrWorkspaceInvalid
	}
	current := m.root
	parts := strings.Split(relative, string(filepath.Separator))
	limit := len(parts)
	if !mustExist {
		limit--
	}
	for _, part := range parts[:limit] {
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if err != nil {
			return "", ErrWorkspaceInvalid
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return "", ErrWorkspaceInvalid
		}
	}
	if mustExist {
		resolved, err := filepath.EvalSymlinks(value)
		if err != nil || !samePath(resolved, value) || !isWithin(m.root, resolved) {
			return "", ErrWorkspaceInvalid
		}
	}
	return value, nil
}

func (m *Manager) changedPaths(ctx context.Context, workspace string) ([]string, error) {
	tracked, err := m.gitOutput(ctx, workspace, nil, nil, "diff", "--no-ext-diff", "--no-textconv", "--name-only", "-z", "HEAD", "--")
	if err != nil {
		return nil, err
	}
	untracked, err := m.gitOutput(ctx, workspace, nil, nil, "ls-files", "--others", "--exclude-standard", "-z", "--")
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	for _, value := range append(splitNUL(tracked), splitNUL(untracked)...) {
		value = path.Clean(filepath.ToSlash(value))
		if value == "." || strings.HasPrefix(value, "../") || path.IsAbs(value) {
			return nil, ErrForeignMutation
		}
		seen[value] = true
	}
	result := make([]string, 0, len(seen))
	for value := range seen {
		result = append(result, value)
	}
	sort.Strings(result)
	return result, nil
}

func (m *Manager) diffTrees(ctx context.Context, base, result string) ([]byte, error) {
	if !validObjectID(base) || !validObjectID(result) {
		return nil, ErrWorkspaceInvalid
	}
	return m.gitOutput(ctx, m.repository, nil, nil, "diff", "--no-ext-diff", "--no-textconv", "--binary", "--full-index", base, result, "--")
}

func (m *Manager) revision(ctx context.Context, cwd, revision string) (string, error) {
	return m.gitObject(ctx, cwd, "rev-parse", "--verify", revision+"^{commit}")
}

func (m *Manager) tree(ctx context.Context, cwd, revision string) (string, error) {
	return m.gitObject(ctx, cwd, "rev-parse", "--verify", revision+"^{tree}")
}

func (m *Manager) gitObject(ctx context.Context, cwd string, args ...string) (string, error) {
	return m.gitObjectWithEnv(ctx, cwd, nil, args...)
}

func (m *Manager) gitObjectWithEnv(ctx context.Context, cwd string, env []string, args ...string) (string, error) {
	output, err := m.gitOutput(ctx, cwd, nil, env, args...)
	value := strings.ToLower(strings.TrimSpace(string(output)))
	if err != nil || !validObjectID(value) {
		return "", ErrWorkspaceInvalid
	}
	return value, nil
}

func (m *Manager) gitPath(ctx context.Context, cwd string, args ...string) (string, error) {
	output, err := m.gitOutput(ctx, cwd, nil, nil, args...)
	if err != nil {
		return "", err
	}
	value := filepath.Clean(strings.TrimSpace(string(output)))
	if !filepath.IsAbs(value) {
		value = filepath.Join(cwd, value)
	}
	return filepath.EvalSymlinks(value)
}

func (m *Manager) gitOutput(ctx context.Context, cwd string, stdin []byte, extraEnv []string, args ...string) ([]byte, error) {
	if filterSensitiveCommands[args[0]] {
		if err := m.rejectExecutableConfiguration(ctx, cwd); err != nil {
			return nil, err
		}
	}
	return m.runGit(ctx, cwd, stdin, extraEnv, args...)
}

func (m *Manager) runGit(ctx context.Context, cwd string, stdin []byte, extraEnv []string, args ...string) ([]byte, error) {
	baseArgs := []string{"-c", "core.hooksPath=/dev/null", "-c", "core.fsmonitor=false", "-c", "credential.helper="}
	command := exec.CommandContext(ctx, m.git, append(baseArgs, args...)...)
	command.Dir = cwd
	command.Env = append([]string{"LC_ALL=C", "LANG=C", "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null", "GIT_TERMINAL_PROMPT=0", "GIT_OPTIONAL_LOCKS=0"}, extraEnv...)
	command.Stdin = bytes.NewReader(stdin)
	var stdout, stderr bytes.Buffer
	command.Stdout, command.Stderr = &stdout, &stderr
	if err := command.Run(); err != nil {
		message := strings.TrimSpace(stderr.String())
		if len(message) > 512 {
			message = message[:512]
		}
		return nil, fmt.Errorf("git %s failed: %w: %s", args[0], err, message)
	}
	return stdout.Bytes(), nil
}

// rejectExecutableConfiguration reads the effective configuration Git would
// use in cwd (repository, worktree and included files; system and global are
// already excluded) and fails closed when a filter or diff driver could start
// a command. It never rewrites configuration.
func (m *Manager) rejectExecutableConfiguration(ctx context.Context, cwd string) error {
	output, err := m.runGit(ctx, cwd, nil, nil, "config", "--list", "--includes", "-z")
	if err != nil {
		return fmt.Errorf("%w: %v", ErrExecutableGitConfiguration, err)
	}
	for _, entry := range splitNUL(output) {
		key, value, _ := strings.Cut(entry, "\n")
		if value != "" && executableConfigurationKey(key) {
			return ErrExecutableGitConfiguration
		}
	}
	return nil
}

// executableConfigurationKey matches filter.<driver>.{clean,smudge,process},
// diff.external, diff.<driver>.{command,textconv} and merge.<driver>.driver
// (consulted by the S2.3 merge-tree integration). Section and variable
// names are case-insensitive; the driver subsection may contain dots.
func executableConfigurationKey(key string) bool {
	first := strings.Index(key, ".")
	last := strings.LastIndex(key, ".")
	if first < 0 {
		return false
	}
	section, variable := strings.ToLower(key[:first]), strings.ToLower(key[last+1:])
	switch {
	case section == "filter" && first != last:
		return variable == "clean" || variable == "smudge" || variable == "process"
	case section == "diff" && first == last:
		return variable == "external"
	case section == "diff":
		return variable == "command" || variable == "textconv"
	case section == "merge" && first != last:
		return variable == "driver"
	}
	return false
}

func validObjectID(value string) bool {
	decoded, err := hex.DecodeString(value)
	return err == nil && (len(decoded) == 20 || len(decoded) == sha256.Size)
}

func cleanExistingDirectory(value string) (string, error) {
	absolute, err := filepath.Abs(value)
	if err != nil || filepath.Clean(value) != absolute {
		return "", ErrInvalidConfiguration
	}
	resolved, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return "", ErrInvalidConfiguration
	}
	info, err := os.Stat(resolved)
	if err != nil || !info.IsDir() {
		return "", ErrInvalidConfiguration
	}
	return resolved, nil
}

func cleanAuthorizedRoot(value string) (string, error) {
	absolute, err := filepath.Abs(value)
	if err != nil || filepath.Clean(value) != absolute {
		return "", ErrInvalidConfiguration
	}
	for current := absolute; ; current = filepath.Dir(current) {
		info, statErr := os.Lstat(current)
		if statErr == nil {
			if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
				return "", ErrInvalidConfiguration
			}
			break
		}
		if !errors.Is(statErr, os.ErrNotExist) || filepath.Dir(current) == current {
			return "", ErrInvalidConfiguration
		}
	}
	if err := os.MkdirAll(absolute, 0o700); err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(absolute)
	if err != nil || !samePath(resolved, absolute) || !privateDirectory(absolute) {
		return "", ErrInvalidConfiguration
	}
	return absolute, nil
}

// privateDirectory follows the existing local/codexruntime pattern: a
// non-symlink directory owned by the effective user, without group/other
// permission bits or extended ACL entries.

// privateRegularFile requires a single-link, owner-only regular file owned by
// the effective user without extended ACL entries.

func samePath(left, right string) bool {
	return filepath.Clean(left) == filepath.Clean(right)
}

func isWithin(root, value string) bool {
	relative, err := filepath.Rel(root, value)
	return err == nil && relative != "." && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

func splitNUL(value []byte) []string {
	parts := bytes.Split(value, []byte{0})
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if len(part) > 0 {
			result = append(result, string(part))
		}
	}
	return result
}
