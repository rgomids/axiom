package local_test

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/rgomids/axiom/internal/local"
	"github.com/rgomids/axiom/internal/project"
	"github.com/rgomids/axiom/internal/projectapp"
)

func repositorySource(path string) project.DocumentationSource {
	return project.DocumentationSource{Key: "docs", Kind: project.RepositorySource, RepositoryRef: project.Configured("core"), Path: project.Configured(path)}
}

func mustWrite(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("synthetic private body"), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestDocumentationResolverRepositorySources(t *testing.T) {
	repo := t.TempDir()
	mustWrite(t, filepath.Join(repo, "docs", "architecture", "index.md"))
	mustWrite(t, filepath.Join(repo, "README.md"))
	outside := t.TempDir()
	mustWrite(t, filepath.Join(outside, "secret.md"))
	resolver := local.DocumentationResolver{}
	cases := map[string]projectapp.DocumentationStatus{
		"docs/architecture":          projectapp.DocumentationAvailable,
		"docs/architecture/index.md": projectapp.DocumentationAvailable,
		"README.md":                  projectapp.DocumentationAvailable,
		"docs/missing":               projectapp.DocumentationMissing,
		"README.md/child":            projectapp.DocumentationMissing,
		"../escape":                  projectapp.DocumentationUnsafe,
	}
	if runtime.GOOS != "windows" {
		if err := os.Symlink(outside, filepath.Join(repo, "linked")); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(filepath.Join(outside, "secret.md"), filepath.Join(repo, "docs", "leaf.md")); err != nil {
			t.Fatal(err)
		}
		cases["linked/secret.md"] = projectapp.DocumentationUnsafe
		cases["linked"] = projectapp.DocumentationUnsafe
		cases["docs/leaf.md"] = projectapp.DocumentationUnsafe
	}
	for path, want := range cases {
		got := resolver.ResolveDocumentation(context.Background(), projectapp.DocumentationRequest{Source: repositorySource(path), RepositoryPath: repo})
		if got != want {
			t.Fatalf("%s: %s, want %s", path, got, want)
		}
	}
	if got := resolver.ResolveDocumentation(context.Background(), projectapp.DocumentationRequest{Source: repositorySource("docs")}); got != projectapp.DocumentationRepositoryUnavailable {
		t.Fatalf("unbound Repository resolved: %s", got)
	}
	// The working directory is never a fallback for an unusable Repository.
	wd, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(wd) })
	if err := os.Chdir(repo); err != nil {
		t.Fatal(err)
	}
	if got := resolver.ResolveDocumentation(context.Background(), projectapp.DocumentationRequest{Source: repositorySource("README.md")}); got != projectapp.DocumentationRepositoryUnavailable {
		t.Fatal("CWD used as Repository fallback")
	}
}

func TestDocumentationResolverLocalFileBindings(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "product.md")
	mustWrite(t, file)
	resolver := local.DocumentationResolver{}
	binding, ok := local.ObserveDocumentationFile("notes", file, func() projectapp.Observation {
		return projectapp.Observation{Availability: projectapp.Available, Basis: projectapp.PresentMetadata}
	})
	if !ok || binding.ExplicitPath != file {
		t.Fatal("explicit regular file not observed")
	}
	source := project.DocumentationSource{Key: "notes", Kind: project.LocalFileSource}
	resolve := func(b *projectapp.DocumentationBinding) projectapp.DocumentationStatus {
		return resolver.ResolveDocumentation(context.Background(), projectapp.DocumentationRequest{Source: source, Binding: b})
	}
	if got := resolve(&binding); got != projectapp.DocumentationAvailable {
		t.Fatalf("bound file %s", got)
	}
	if got := resolve(nil); got != projectapp.DocumentationUnbound {
		t.Fatalf("unbound %s", got)
	}
	// Replacing the file changes its identity: stale, never silently accepted.
	if err := os.Remove(file); err != nil {
		t.Fatal(err)
	}
	if got := resolve(&binding); got != projectapp.DocumentationMissing {
		t.Fatalf("missing %s", got)
	}
	keep := filepath.Join(dir, "keep.md")
	mustWrite(t, keep)
	mustWrite(t, file)
	if got := resolve(&binding); got != projectapp.DocumentationStale {
		t.Fatalf("replaced %s", got)
	}
	if _, ok := local.ObserveDocumentationFile("notes", dir, nil); ok {
		t.Fatal("directory accepted as local file")
	}
	if _, ok := local.ObserveDocumentationFile("notes", "relative.md", nil); ok {
		t.Fatal("relative path accepted")
	}
	if runtime.GOOS != "windows" {
		link := filepath.Join(dir, "link.md")
		if err := os.Symlink(keep, link); err != nil {
			t.Fatal(err)
		}
		if _, ok := local.ObserveDocumentationFile("notes", link, nil); ok {
			t.Fatal("link accepted at bootstrap")
		}
		linked := binding
		linked.ExplicitPath = link
		if got := resolve(&linked); got != projectapp.DocumentationUnsafe {
			t.Fatalf("link resolved: %s", got)
		}
	}
}
