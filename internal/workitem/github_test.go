package workitem

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestGitHubAdapterUsesBoundedSpecificCommands(t *testing.T) {
	directory := t.TempDir()
	git := filepath.Join(directory, "git")
	gh := filepath.Join(directory, "gh")
	writeExecutable(t, git, "#!/bin/sh\nprintf '%s\\n' 'git@github.com:owner/repo.git'\n")
	writeExecutable(t, gh, `#!/bin/sh
if [ "$1" = issue ] && [ "$2" = create ]; then
  printf '%s\n' 'https://github.com/owner/repo/issues/7'
  exit 0
fi
if [ "$1" = issue ] && [ "$2" = view ]; then
  printf '%s\n' '{"Number":7,"URL":"https://github.com/owner/repo/issues/7","State":"CLOSED"}'
  exit 0
fi
if [ "$1" = issue ] && { [ "$2" = comment ] || [ "$2" = close ]; }; then
  printf '%s\n' 'ok'
  exit 0
fi
exit 1
`)
	adapter, err := NewGitHubAdapter(git, gh)
	if err != nil {
		t.Fatal(err)
	}
	repository, err := adapter.GitHubRepository(context.Background(), directory)
	if err != nil || repository != "owner/repo" {
		t.Fatalf("repository = %q, %v", repository, err)
	}
	created, err := adapter.Create(context.Background(), repository, "Title", "Body")
	if err != nil || created.Number != 7 {
		t.Fatalf("create = %#v, %v", created, err)
	}
	if err := adapter.Comment(context.Background(), repository, 7, "Evidence"); err != nil {
		t.Fatal(err)
	}
	closed, err := adapter.Close(context.Background(), repository, 7)
	if err != nil || closed.State != "CLOSED" {
		t.Fatalf("close = %#v, %v", closed, err)
	}
}

func TestParseGitHubRepositoryRejectsOtherHosts(t *testing.T) {
	for _, value := range []string{"https://example.com/owner/repo.git", "file:///tmp/repo", "git@other:owner/repo.git"} {
		if got, err := parseGitHubRepository(value); err == nil || got != "" {
			t.Fatalf("accepted %q as %q", value, got)
		}
	}
}

func TestExternalReferenceMustMatchRepositoryNumberAndHTTPS(t *testing.T) {
	valid := External{Number: 7, URL: "https://github.com/owner/repo/issues/7", State: "OPEN"}
	if !validExternal("owner/repo", 7, valid) {
		t.Fatal("valid reference rejected")
	}
	for _, candidate := range []External{
		{Number: 7, URL: "https://github.com/other/repo/issues/7", State: "OPEN"},
		{Number: 8, URL: "https://github.com/owner/repo/issues/7", State: "OPEN"},
		{Number: 7, URL: "http://github.com/owner/repo/issues/7", State: "OPEN"},
		{Number: 7, URL: "https://github.com/owner/repo/issues/7?token=sentinel", State: "OPEN"},
	} {
		if validExternal("owner/repo", 7, candidate) {
			t.Fatalf("mismatched reference accepted: %#v", candidate)
		}
	}
}

func writeExecutable(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o700); err != nil {
		t.Fatal(err)
	}
}
