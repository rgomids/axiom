package projectdiscovery

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

func writeFile(t *testing.T, p, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func gitRepo(t *testing.T, config string) string {
	t.Helper()
	root := t.TempDir()
	writeFile(t, filepath.Join(root, ".git", "config"), config)
	return root
}

func skipWindows(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("symlink semantics differ on Windows")
	}
}

func remote(name, url string) string {
	return fmt.Sprintf("[remote %q]\n\turl = %s\n", name, url)
}

type cand struct {
	locator string
	names   string
}

func TestInspectRepositoryConfig(t *testing.T) {
	many := strings.Builder{}
	for i := 0; i < 33; i++ {
		many.WriteString(remote(fmt.Sprintf("r%d", i), fmt.Sprintf("https://example.com/a/r%d.git", i)))
	}
	manyURLs := "[remote \"r\"]\n"
	for i := 0; i < 65; i++ {
		manyURLs += fmt.Sprintf("\turl = https://example.com/a/u%d.git\n", i)
	}
	tests := []struct {
		name        string
		config      string
		git         GitState
		status      RemoteStatus
		cands       []cand
		unsupported int
	}{
		{"no remotes", "[core]\n\tbare = false\n", GitRepository, RemoteLocalOnly, nil, 0},
		{"empty config", "", GitRepository, RemoteLocalOnly, nil, 0},
		{"one remote", remote("origin", "https://github.com/a/b.git"), GitRepository, RemoteSingle,
			[]cand{{"https://github.com/a/b.git", "origin"}}, 0},
		{"aliases collapse", remote("origin", "https://github.com/a/b.git") + remote("mirror", "https://github.com/a/b.git"),
			GitRepository, RemoteSingle, []cand{{"https://github.com/a/b.git", "mirror,origin"}}, 0},
		{"origin not preferred", remote("origin", "https://github.com/z/z.git") + remote("upstream", "https://github.com/a/a.git"),
			GitRepository, RemoteAmbiguous,
			[]cand{{"https://github.com/a/a.git", "upstream"}, {"https://github.com/z/z.git", "origin"}}, 0},
		{"https vs ssh", remote("origin", "https://github.com/a/b.git") + remote("ssh", "git@github.com:a/b.git"),
			GitRepository, RemoteAmbiguous,
			[]cand{{"git@github.com:a/b.git", "ssh"}, {"https://github.com/a/b.git", "origin"}}, 0},
		{"host case collapses", remote("a", "https://GitHub.com/a/b.git") + remote("b", "https://github.com/a/b.git"),
			GitRepository, RemoteSingle, []cand{{"https://github.com/a/b.git", "a,b"}}, 0},
		{"unsupported counted", remote("pw", "https://user:hunter2@example.com/a/b.git") + remote("local", "/srv/git/b.git"),
			GitRepository, RemoteLocalOnly, nil, 2},
		{"pushurl ignored", "[remote \"origin\"]\n\turl = https://github.com/a/b.git\n\tpushurl = https://github.com/c/d.git\n",
			GitRepository, RemoteSingle, []cand{{"https://github.com/a/b.git", "origin"}}, 0},
		{"case-insensitive keys and legacy header", "[REMOTE.origin]\n\tURL = https://github.com/a/b.git\n",
			GitRepository, RemoteSingle, []cand{{"https://github.com/a/b.git", "origin"}}, 0},
		{"comments and quotes", "# c\n; c\n[remote \"o\"] ; trailing\n\turl = \"https://github.com/a/b.git\" # note\n",
			GitRepository, RemoteSingle, []cand{{"https://github.com/a/b.git", "o"}}, 0},
		{"invalid remote name omitted", remote("bad name!", "https://github.com/a/b.git"),
			GitRepository, RemoteSingle, []cand{{"https://github.com/a/b.git", ""}}, 0},
		{"include", "[include]\n\tpath = other\n" + remote("origin", "https://github.com/a/b.git"),
			GitRepository, RemoteIncomplete, []cand{{"https://github.com/a/b.git", "origin"}}, 0},
		{"includeIf", "[includeIf \"gitdir:/x/\"]\n\tpath = other\n", GitRepository, RemoteIncomplete, nil, 0},
		{"insteadOf", "[url \"git@github.com:\"]\n\tinsteadOf = https://github.com/\n" + remote("origin", "https://github.com/a/b.git"),
			GitRepository, RemoteIncomplete, []cand{{"https://github.com/a/b.git", "origin"}}, 0},
		{"pushInsteadOf", "[url \"x\"]\n\tpushInsteadOf = y\n", GitRepository, RemoteIncomplete, nil, 0},
		{"continuation", "[remote \"origin\"]\n\turl = https://github.com/a/\\\nb.git\n", GitRepository, RemoteIncomplete, nil, 0},
		{"too many remotes", many.String(), GitRepository, RemoteIncomplete, nil, -1},
		{"too many urls", manyURLs, GitRepository, RemoteIncomplete, nil, -1},
		{"malformed header", "[remote \"origin\"\n\turl = x\n", GitUnreadable, RemoteIncomplete, nil, 0},
		{"unterminated subsection", "[remote \"origin]\n", GitUnreadable, RemoteIncomplete, nil, 0},
		{"unterminated quote", "[remote \"o\"]\n\turl = \"abc\n", GitUnreadable, RemoteIncomplete, nil, 0},
		{"invalid utf8", "[core]\n\tx = \xff\n", GitUnreadable, RemoteIncomplete, nil, 0},
		{"oversize", "# " + strings.Repeat("a", 70000) + "\n", GitUnreadable, RemoteIncomplete, nil, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := InspectRepository(gitRepo(t, tc.config))
			if err != nil {
				t.Fatal(err)
			}
			if got.Git != tc.git || got.Remote != tc.status {
				t.Fatalf("got %s/%s, want %s/%s", got.Git, got.Remote, tc.git, tc.status)
			}
			if got.Candidates == nil {
				t.Fatal("Candidates must not be nil")
			}
			if tc.unsupported >= 0 && got.Unsupported != tc.unsupported {
				t.Fatalf("unsupported = %d, want %d", got.Unsupported, tc.unsupported)
			}
			if tc.unsupported < 0 {
				return
			}
			var gotC []cand
			for _, c := range got.Candidates {
				if c.Names == nil {
					t.Fatal("Names must not be nil")
				}
				gotC = append(gotC, cand{c.Locator, strings.Join(c.Names, ",")})
			}
			if !reflect.DeepEqual(gotC, tc.cands) {
				t.Fatalf("candidates = %v, want %v", gotC, tc.cands)
			}
		})
	}
}

func TestInspectRepositoryNeverEchoesUnsupported(t *testing.T) {
	cfg := remote("pw", "https://user:hunter2@example.com/a/b.git") + remote("local", "/srv/secret-path/b.git")
	got, err := InspectRepository(gitRepo(t, cfg))
	if err != nil {
		t.Fatal(err)
	}
	data, _ := json.Marshal(got)
	for _, leak := range []string{"hunter2", "secret-path", "example.com"} {
		if strings.Contains(string(data), leak) {
			t.Fatalf("result leaks %q: %s", leak, data)
		}
	}
	if got.Unsupported != 2 {
		t.Fatalf("unsupported = %d", got.Unsupported)
	}
}

func TestInspectRepositoryAbsentAndMissingConfig(t *testing.T) {
	got, err := InspectRepository(t.TempDir())
	if err != nil || got.Git != GitAbsent || got.Remote != RemoteLocalOnly || got.Candidates == nil {
		t.Fatalf("absent: %+v %v", got, err)
	}
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	got, err = InspectRepository(root)
	if err != nil || got.Git != GitRepository || got.Remote != RemoteLocalOnly {
		t.Fatalf("no config: %+v %v", got, err)
	}
}

func TestInspectRepositoryLinksAndPointers(t *testing.T) {
	skipWindows(t)
	t.Run(".git symlink", func(t *testing.T) {
		root, real := t.TempDir(), gitRepo(t, remote("o", "https://github.com/a/b.git"))
		if err := os.Symlink(filepath.Join(real, ".git"), filepath.Join(root, ".git")); err != nil {
			t.Fatal(err)
		}
		expect(t, root, GitUnsafe, RemoteIncomplete)
	})
	t.Run("config symlink", func(t *testing.T) {
		root := t.TempDir()
		target := filepath.Join(t.TempDir(), "cfg")
		writeFile(t, target, remote("o", "https://github.com/a/b.git"))
		if err := os.MkdirAll(filepath.Join(root, ".git"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(target, filepath.Join(root, ".git", "config")); err != nil {
			t.Fatal(err)
		}
		expect(t, root, GitUnsafe, RemoteIncomplete)
	})
	t.Run("gitdir to symlink", func(t *testing.T) {
		root, real := t.TempDir(), t.TempDir()
		link := filepath.Join(t.TempDir(), "link")
		if err := os.Symlink(real, link); err != nil {
			t.Fatal(err)
		}
		writeFile(t, filepath.Join(root, ".git"), "gitdir: "+link+"\n")
		expect(t, root, GitUnsafe, RemoteIncomplete)
	})
	t.Run("commondir symlink", func(t *testing.T) {
		root := t.TempDir()
		gd := filepath.Join(root, "gd")
		writeFile(t, filepath.Join(root, ".git"), "gitdir: gd\n")
		writeFile(t, filepath.Join(t.TempDir(), "x"), "")
		target := filepath.Join(t.TempDir(), "c")
		writeFile(t, target, "..\n")
		if err := os.MkdirAll(gd, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(target, filepath.Join(gd, "commondir")); err != nil {
			t.Fatal(err)
		}
		expect(t, root, GitUnsafe, RemoteIncomplete)
	})
	t.Run("input symlink", func(t *testing.T) {
		link := filepath.Join(t.TempDir(), "l")
		if err := os.Symlink(t.TempDir(), link); err != nil {
			t.Fatal(err)
		}
		if _, err := InspectRepository(link); !errors.Is(err, ErrInvalidLocation) {
			t.Fatalf("err = %v", err)
		}
		if _, err := DetectTechnology(link); !errors.Is(err, ErrInvalidLocation) {
			t.Fatalf("err = %v", err)
		}
	})
}

func expect(t *testing.T, root string, git GitState, status RemoteStatus) Repository {
	t.Helper()
	got, err := InspectRepository(root)
	if err != nil {
		t.Fatal(err)
	}
	if got.Git != git || got.Remote != status {
		t.Fatalf("got %s/%s, want %s/%s", got.Git, got.Remote, git, status)
	}
	return got
}

func TestInspectRepositoryWorktreeWithCommondir(t *testing.T) {
	main := gitRepo(t, remote("origin", "https://github.com/a/b.git"))
	wt := t.TempDir()
	gd := filepath.Join(main, ".git", "worktrees", "wt")
	writeFile(t, filepath.Join(gd, "commondir"), "../..\n")
	writeFile(t, filepath.Join(wt, ".git"), "gitdir: "+gd+"\r\n")
	got := expect(t, wt, GitRepository, RemoteSingle)
	if got.Candidates[0].Locator != "https://github.com/a/b.git" {
		t.Fatalf("candidates = %+v", got.Candidates)
	}
	// Relative gitdir resolves against the repository path.
	rel := t.TempDir()
	writeFile(t, filepath.Join(rel, ".git"), "gitdir: meta\n")
	writeFile(t, filepath.Join(rel, "meta", "config"), remote("a", "https://github.com/x/y.git"))
	expect(t, rel, GitRepository, RemoteSingle)
}

func TestInspectRepositoryMalformedPointers(t *testing.T) {
	for name, content := range map[string]string{
		"no prefix":  "/somewhere\n",
		"two lines":  "gitdir: a\nb\n",
		"empty path": "gitdir: \n",
		"oversize":   "gitdir: " + strings.Repeat("a", 5000),
		"missing":    "gitdir: /definitely/not/here\n",
	} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			writeFile(t, filepath.Join(root, ".git"), content)
			expect(t, root, GitUnreadable, RemoteIncomplete)
		})
	}
}

func TestInspectRepositoryInvalidLocation(t *testing.T) {
	file := filepath.Join(t.TempDir(), "f")
	writeFile(t, file, "x")
	for _, p := range []string{"", "relative/dir", ".", file, filepath.Join(t.TempDir(), "missing")} {
		if _, err := InspectRepository(p); !errors.Is(err, ErrInvalidLocation) {
			t.Errorf("InspectRepository(%q) err = %v", p, err)
		}
	}
}

func TestInspectRepositoryDeterministicAndIndependent(t *testing.T) {
	a := gitRepo(t, remote("x", "https://github.com/a/a.git")+remote("y", "https://github.com/b/b.git"))
	b := gitRepo(t, remote("y", "https://github.com/b/b.git")+remote("x", "https://github.com/a/a.git"))
	c := t.TempDir()
	first, _ := InspectRepository(a)
	for i := 0; i < 3; i++ {
		again, _ := InspectRepository(a)
		if !reflect.DeepEqual(first, again) {
			t.Fatal("repeated calls differ")
		}
	}
	reordered, _ := InspectRepository(b)
	if !reflect.DeepEqual(first, reordered) {
		t.Fatalf("config order changed output: %+v vs %+v", first, reordered)
	}
	if got, _ := InspectRepository(c); got.Git != GitAbsent || got.Remote != RemoteLocalOnly {
		t.Fatalf("independent repo affected: %+v", got)
	}
}

func TestDeriveRepositoryKey(t *testing.T) {
	tests := []struct {
		path string
		key  string
		ok   bool
	}{
		{"/work/My_Repo.Name", "my-repo-name", true},
		{"/work/--a__b--", "a-b", true},
		{"/work/Axiom", "axiom", true},
		{"/work/api/", "api", true},
		{"/work/" + strings.Repeat("a", 62) + "-b", strings.Repeat("a", 62), true},
		{"/work/" + strings.Repeat("a", 80), strings.Repeat("a", 63), true},
		{"/work/___", "", false},
		{"/work/日本語", "", false},
		{"/", "", false},
	}
	for _, tc := range tests {
		key, ok := DeriveRepositoryKey(tc.path)
		if key != tc.key || ok != tc.ok {
			t.Errorf("DeriveRepositoryKey(%q) = %q,%v want %q,%v", tc.path, key, ok, tc.key, tc.ok)
		}
	}
}

func TestQueryOrFragmentLocatorIsUnsupported(t *testing.T) {
	root := gitRepo(t, remote("a", "https://example.com/r.git?token=synthetic-secret")+remote("b", `"https://example.com/r.git#frag"`))
	got, err := InspectRepository(root)
	if err != nil || got.Remote != RemoteLocalOnly || got.Unsupported != 2 {
		t.Fatalf("%+v %v", got, err)
	}
}
