package gitworkspace_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

type repositoryFixture struct{ repository, revision, tree string }

func initRepository(t *testing.T, repository string) repositoryFixture {
	t.Helper()
	if err := os.MkdirAll(repository, 0o700); err != nil {
		t.Fatal(err)
	}
	git(t, repository, "init", "--initial-branch=main")
	writeFile(t, filepath.Join(repository, "README.md"), "base\n")
	writeFile(t, filepath.Join(repository, "shared.txt"), "base\n")
	git(t, repository, "add", "README.md", "shared.txt")
	git(t, repository, "-c", "user.name=Axiom Test", "-c", "user.email=axiom@example.invalid", "commit", "-m", "base")
	return repositoryFixture{repository: repository, revision: git(t, repository, "rev-parse", "HEAD"), tree: git(t, repository, "rev-parse", "HEAD^{tree}")}
}

func git(t *testing.T, directory string, args ...string) string {
	t.Helper()
	gitPath, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	command := exec.Command(gitPath, args...)
	command.Dir = directory
	command.Env = []string{"LC_ALL=C", "LANG=C", "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null", "GIT_TERMINAL_PROMPT=0", "GIT_AUTHOR_DATE=2000-01-01T00:00:00Z", "GIT_COMMITTER_DATE=2000-01-01T00:00:00Z"}
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v: %s", strings.Join(args, " "), err, output)
	}
	return strings.TrimSpace(string(output))
}

// gitAllowFailure runs Git with the test environment and returns its result
// without failing; control cases use it where a live filter makes Git fail.
func gitAllowFailure(directory string, args ...string) (string, error) {
	gitPath, err := exec.LookPath("git")
	if err != nil {
		return "", err
	}
	command := exec.Command(gitPath, args...)
	command.Dir = directory
	command.Env = []string{"LC_ALL=C", "LANG=C", "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null", "GIT_TERMINAL_PROMPT=0", "PATH=/usr/bin:/bin"}
	output, err := command.CombinedOutput()
	return string(output), err
}

func writeFile(t *testing.T, name, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(name), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(name, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func canonicalTempDir(t *testing.T) string {
	t.Helper()
	value, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return value
}

// fixtureGraph constructs inert integration/codec test data. It has no writer,
// admission, scheduler or process path.
