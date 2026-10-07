package projectdiscovery

import (
	"errors"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/rgomids/axiom/internal/project"
)

// TechnologyFact is a name-based technology signal with its Evidence path.
type TechnologyFact struct {
	Key      string `json:"key"`
	Value    string `json:"value"`
	Path     string `json:"path"`
	Count    int    `json:"count"`
	Conflict string `json:"conflict,omitempty"`
}

const (
	maxDirs    = 256
	maxEntries = 4096
	maxDepth   = 2
)

var rootSignals = map[string][2]string{
	"go.mod":              {"language.go", "go"},
	"package.json":        {"language.javascript", "javascript"},
	"tsconfig.json":       {"language.typescript", "typescript"},
	"pnpm-lock.yaml":      {"package-manager.pnpm", "pnpm"},
	"package-lock.json":   {"package-manager.npm", "npm"},
	"yarn.lock":           {"package-manager.yarn", "yarn"},
	"bun.lock":            {"package-manager.bun", "bun"},
	"bun.lockb":           {"package-manager.bun", "bun"},
	"Cargo.toml":          {"language.rust", "rust"},
	"pyproject.toml":      {"language.python", "python"},
	"requirements.txt":    {"language.python", "python"},
	"poetry.lock":         {"package-manager.poetry", "poetry"},
	"uv.lock":             {"package-manager.uv", "uv"},
	"Gemfile":             {"language.ruby", "ruby"},
	"pom.xml":             {"build.maven", "maven"},
	"build.gradle":        {"build.gradle", "gradle"},
	"build.gradle.kts":    {"build.gradle", "gradle"},
	"Dockerfile":          {"container.docker", "docker"},
	"Containerfile":       {"container.docker", "docker"},
	"compose.yaml":        {"container.compose", "compose"},
	"compose.yml":         {"container.compose", "compose"},
	"docker-compose.yaml": {"container.compose", "compose"},
	"docker-compose.yml":  {"container.compose", "compose"},
	".gitlab-ci.yml":      {"workflow.gitlab-ci", "gitlab-ci"},
}

type detector struct {
	root  string
	dirs  int
	facts map[string]*TechnologyFact
}

// DetectTechnology inspects file names only, never opening files or following
// links, within the documented depth, entry and directory bounds. Within a
// directory over the entry bound only the entries returned first are considered.
func DetectTechnology(repoPath string) ([]TechnologyFact, error) {
	root, err := validateLocation(repoPath)
	if err != nil {
		return nil, err
	}
	d := &detector{root: root, facts: map[string]*TechnologyFact{}}
	if !d.walk("", 0) {
		return nil, ErrInvalidLocation
	}
	out := make([]TechnologyFact, 0, len(d.facts))
	managers := 0
	for _, f := range d.facts {
		if strings.HasPrefix(f.Key, "package-manager.") {
			managers++
		}
		out = append(out, *f)
	}
	for i := range out {
		if managers > 1 && strings.HasPrefix(out[i].Key, "package-manager.") {
			out[i].Conflict = "package-manager"
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out, nil
}

// walk visits one directory and returns false only when it cannot be read.
func (d *detector) walk(rel string, depth int) bool {
	d.dirs++
	entries, ok := d.readDir(rel)
	if !ok {
		return false
	}
	for _, e := range entries {
		name, typ := e.Name(), e.Type()
		child := path.Join(rel, name)
		switch {
		case typ&os.ModeSymlink != 0:
		case typ.IsRegular():
			d.file(rel, name, child, depth)
		case typ.IsDir() && depth < maxDepth && d.descend(rel, name) && d.dirs < maxDirs:
			d.walk(child, depth+1)
		}
	}
	return true
}

func (d *detector) descend(rel, name string) bool {
	if rel == ".github" {
		return name == "workflows"
	}
	switch name {
	case ".git", "node_modules", "vendor":
		return false
	}
	return !strings.HasPrefix(name, ".") || name == ".github"
}

func (d *detector) file(rel, name, child string, depth int) {
	// Evidence must be a portable repository-relative path; other names are
	// never proposed, so one unusual file cannot block a bootstrap.
	if !project.RepositoryRelativePath(child) {
		return
	}
	switch {
	case rel == ".github/workflows":
		if strings.HasSuffix(name, ".yml") || strings.HasSuffix(name, ".yaml") {
			d.add("workflow.github-actions", "github-actions", child)
		}
		return
	case rel == ".github":
		return
	}
	if depth == 0 {
		if s, ok := rootSignals[name]; ok {
			d.add(s[0], s[1], child)
		}
	}
	if strings.HasSuffix(name, ".tf") {
		d.add("infrastructure.terraform", "terraform", child)
	}
}

func (d *detector) add(key, value, p string) {
	f := d.facts[key]
	if f == nil {
		d.facts[key] = &TechnologyFact{Key: key, Value: value, Path: p, Count: 1}
		return
	}
	f.Count++
	if p < f.Path {
		f.Path = p
	}
}

// readDir reads at most maxEntries entries, sorted by name.
func (d *detector) readDir(rel string) ([]os.DirEntry, bool) {
	f, err := os.Open(filepath.Join(d.root, filepath.FromSlash(rel)))
	if err != nil {
		return nil, false
	}
	defer f.Close()
	entries, err := f.ReadDir(maxEntries)
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, false
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	return entries, true
}
