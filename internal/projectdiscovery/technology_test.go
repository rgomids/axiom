package projectdiscovery

import (
	"errors"
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

func touchAll(t *testing.T, root string, rels ...string) {
	t.Helper()
	for _, rel := range rels {
		writeFile(t, filepath.Join(root, filepath.FromSlash(rel)), "")
	}
}

func summary(facts []TechnologyFact) []string {
	out := []string{}
	for _, f := range facts {
		s := fmt.Sprintf("%s=%s@%s#%d", f.Key, f.Value, f.Path, f.Count)
		if f.Conflict != "" {
			s += "!" + f.Conflict
		}
		out = append(out, s)
	}
	return out
}

func TestDetectTechnology(t *testing.T) {
	tests := []struct {
		name  string
		files []string
		want  []string
	}{
		{"empty", nil, []string{}},
		{"go", []string{"go.mod", "main.go"}, []string{"language.go=go@go.mod#1"}},
		{"node ts pnpm", []string{"package.json", "tsconfig.json", "pnpm-lock.yaml"},
			[]string{"language.javascript=javascript@package.json#1", "language.typescript=typescript@tsconfig.json#1", "package-manager.pnpm=pnpm@pnpm-lock.yaml#1"}},
		{"conflicting lockfiles", []string{"pnpm-lock.yaml", "package-lock.json"},
			[]string{"package-manager.npm=npm@package-lock.json#1!package-manager", "package-manager.pnpm=pnpm@pnpm-lock.yaml#1!package-manager"}},
		{"bun variants one fact", []string{"bun.lock", "bun.lockb"}, []string{"package-manager.bun=bun@bun.lock#2"}},
		{"docker compose", []string{"Dockerfile", "docker-compose.yml", "compose.yaml"},
			[]string{"container.compose=compose@compose.yaml#2", "container.docker=docker@Dockerfile#1"}},
		{"terraform depth 2", []string{"infra/prod/main.tf", "infra/prod/b.tf", "a.tf"},
			[]string{"infrastructure.terraform=terraform@a.tf#3"}},
		{"terraform depth 3 ignored", []string{"a/b/c/main.tf"}, []string{}},
		{"github workflows", []string{".github/workflows/ci.yml", ".github/workflows/b.yaml", ".github/workflows/notes.md", ".github/other.yml", ".github/workflows/sub/x.yml"},
			[]string{"workflow.github-actions=github-actions@.github/workflows/b.yaml#2"}},
		{"gitlab", []string{".gitlab-ci.yml"}, []string{"workflow.gitlab-ci=gitlab-ci@.gitlab-ci.yml#1"}},
		{"mixed", []string{"go.mod", "pyproject.toml", "requirements.txt", "uv.lock", "Cargo.toml", "Gemfile", "pom.xml", "build.gradle.kts", "poetry.lock"},
			[]string{"build.gradle=gradle@build.gradle.kts#1", "build.maven=maven@pom.xml#1", "language.go=go@go.mod#1", "language.python=python@pyproject.toml#2", "language.ruby=ruby@Gemfile#1", "language.rust=rust@Cargo.toml#1", "package-manager.poetry=poetry@poetry.lock#1!package-manager", "package-manager.uv=uv@uv.lock#1!package-manager"}},
		{"root signals only at root", []string{"sub/go.mod", "sub/package.json"}, []string{}},
		{"skipped directories", []string{"node_modules/x.tf", "vendor/x.tf", ".hidden/x.tf", ".git/x.tf", "src/ok.tf"},
			[]string{"infrastructure.terraform=terraform@src/ok.tf#1"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			touchAll(t, root, tc.files...)
			got, err := DetectTechnology(root)
			if err != nil {
				t.Fatal(err)
			}
			if got == nil {
				t.Fatal("facts must not be nil")
			}
			if s := summary(got); !reflect.DeepEqual(s, tc.want) {
				t.Fatalf("got %v\nwant %v", s, tc.want)
			}
		})
	}
}

func TestDetectTechnologyIgnoresLinksAndDirectories(t *testing.T) {
	skipWindows(t)
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "go.mod"), 0o755); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "package.json")
	writeFile(t, outside, "")
	if err := os.Symlink(outside, filepath.Join(root, "package.json")); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "x.tf"), "")
	if err := os.Symlink(dir, filepath.Join(root, "linked")); err != nil {
		t.Fatal(err)
	}
	got, err := DetectTechnology(root)
	if err != nil || len(got) != 0 {
		t.Fatalf("got %v, %v", summary(got), err)
	}
}

func TestDetectTechnologyBounds(t *testing.T) {
	t.Run("directory count", func(t *testing.T) {
		root := t.TempDir()
		for i := 0; i < 300; i++ {
			touchAll(t, root, fmt.Sprintf("d%03d/main.tf", i))
		}
		got, err := DetectTechnology(root)
		if err != nil || len(got) != 1 {
			t.Fatalf("got %v, %v", summary(got), err)
		}
		if got[0].Count != 255 || got[0].Path != "d000/main.tf" {
			t.Fatalf("fact = %+v", got[0])
		}
	})
	t.Run("entries per directory", func(t *testing.T) {
		root := t.TempDir()
		for i := 0; i < 4200; i++ {
			touchAll(t, root, fmt.Sprintf("f%04d.txt", i))
		}
		touchAll(t, root, "go.mod")
		got, err := DetectTechnology(root)
		if err != nil || len(got) > 1 {
			t.Fatalf("got %v, %v", summary(got), err)
		}
	})
}

func TestDetectTechnologyDeterministic(t *testing.T) {
	root := t.TempDir()
	touchAll(t, root, "go.mod", "b/x.tf", "a/y.tf", "Dockerfile")
	first, _ := DetectTechnology(root)
	for i := 0; i < 3; i++ {
		again, _ := DetectTechnology(root)
		if !reflect.DeepEqual(first, again) {
			t.Fatal("repeated calls differ")
		}
	}
	if first[len(first)-1].Key != "language.go" || first[0].Key != "container.docker" {
		t.Fatalf("not sorted by key: %v", summary(first))
	}
	if first[1].Path != "a/y.tf" {
		t.Fatalf("terraform path = %q", first[1].Path)
	}
}

func TestDetectTechnologyInvalidLocation(t *testing.T) {
	file := filepath.Join(t.TempDir(), "f")
	writeFile(t, file, "")
	for _, p := range []string{"", "rel", file, filepath.Join(t.TempDir(), "missing")} {
		if _, err := DetectTechnology(p); !errors.Is(err, ErrInvalidLocation) {
			t.Errorf("DetectTechnology(%q) err = %v", p, err)
		}
	}
}

func TestPackageDoesNotImportProcessOrNetwork(t *testing.T) {
	forbidden := map[string]bool{"os/exec": true, "net": true, "net/http": true, "syscall": true}
	files, err := filepath.Glob("*.go")
	if err != nil || len(files) == 0 {
		t.Fatalf("glob: %v %v", files, err)
	}
	fset := token.NewFileSet()
	for _, name := range files {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, name, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatal(err)
		}
		for _, imp := range f.Imports {
			path, _ := strconv.Unquote(imp.Path.Value)
			if forbidden[path] {
				t.Errorf("%s imports forbidden package %q", name, path)
			}
		}
	}
}

func TestNonPortableEvidenceNamesAreNeverProposed(t *testing.T) {
	root := t.TempDir()
	touchAll(t, root, "$old.tf", "a:b.tf", " padded.tf")
	facts, err := DetectTechnology(root)
	if err != nil || len(facts) != 0 {
		t.Fatalf("unportable evidence proposed: %v %v", summary(facts), err)
	}
	touchAll(t, root, "main.tf")
	facts, _ = DetectTechnology(root)
	if got := summary(facts); !reflect.DeepEqual(got, []string{"infrastructure.terraform=terraform@main.tf#1"}) {
		t.Fatalf("got %v", got)
	}
}
