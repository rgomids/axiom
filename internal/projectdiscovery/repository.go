// Package projectdiscovery derives bootstrap proposals from explicit local
// Repository locations. It is read-only: it never executes a process (no os/exec),
// never uses the network, never reads global/system Git configuration, never
// follows links, and never reads repository file contents except bounded Git
// metadata (.git file/dir, commondir, config).
package projectdiscovery

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/rgomids/axiom/internal/project"
)

// GitState classifies the .git metadata of a Repository location.
type GitState string

const (
	GitAbsent     GitState = "absent"
	GitRepository GitState = "repository"
	GitUnsafe     GitState = "unsafe"
	GitUnreadable GitState = "unreadable"
)

// RemoteStatus classifies the remote identity candidates of a Repository.
type RemoteStatus string

const (
	RemoteLocalOnly  RemoteStatus = "local_only"
	RemoteSingle     RemoteStatus = "single"
	RemoteAmbiguous  RemoteStatus = "ambiguous"
	RemoteIncomplete RemoteStatus = "incomplete"
)

// RemoteCandidate is one distinct normalized locator and the remote names that
// map to it.
type RemoteCandidate struct {
	Locator string   `json:"locator"`
	Names   []string `json:"names"`
}

// Repository is the read-only discovery result. Raw URL text of unsupported
// values is never retained.
type Repository struct {
	Git         GitState          `json:"git"`
	Remote      RemoteStatus      `json:"remote"`
	Candidates  []RemoteCandidate `json:"candidates"`
	Unsupported int               `json:"unsupported"`
}

// ErrInvalidLocation reports a path that is not an absolute, existing,
// non-link directory.
var ErrInvalidLocation = errors.New("repository location is invalid")

const (
	maxGitFile   = 4096
	maxConfig    = 65536
	maxRemotes   = 32
	maxURLValues = 64
)

var (
	remoteName = regexp.MustCompile(`^[A-Za-z0-9._-]{1,64}$`)
	keyPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)
)

// InspectRepository inspects Git metadata of an explicit Repository location.
// Unsafe or unreadable metadata is a classification, not an error.
func InspectRepository(path string) (Repository, error) {
	root, err := validateLocation(path)
	if err != nil {
		return Repository{}, err
	}
	state, configPath := locateConfig(root)
	switch state {
	case GitAbsent:
		return Repository{Git: GitAbsent, Remote: RemoteLocalOnly, Candidates: []RemoteCandidate{}}, nil
	case GitUnsafe, GitUnreadable:
		return Repository{Git: state, Remote: RemoteIncomplete, Candidates: []RemoteCandidate{}}, nil
	}
	repo := Repository{Git: GitRepository, Remote: RemoteLocalOnly, Candidates: []RemoteCandidate{}}
	if configPath == "" {
		return repo, nil
	}
	data, state := readRegular(configPath, maxConfig)
	if state != GitRepository {
		return Repository{Git: state, Remote: RemoteIncomplete, Candidates: []RemoteCandidate{}}, nil
	}
	if !utf8.Valid(data) {
		return Repository{Git: GitUnreadable, Remote: RemoteIncomplete, Candidates: []RemoteCandidate{}}, nil
	}
	cfg, ok := parseConfig(string(data))
	if !ok {
		return Repository{Git: GitUnreadable, Remote: RemoteIncomplete, Candidates: []RemoteCandidate{}}, nil
	}
	repo.Candidates, repo.Unsupported = cfg.candidates()
	switch {
	case cfg.incomplete:
		repo.Remote = RemoteIncomplete
	case len(repo.Candidates) == 1:
		repo.Remote = RemoteSingle
	case len(repo.Candidates) > 1:
		repo.Remote = RemoteAmbiguous
	}
	return repo, nil
}

// DeriveRepositoryKey proposes a Repository key from the directory base name.
func DeriveRepositoryKey(path string) (string, bool) {
	base := strings.ToLower(filepath.Base(filepath.Clean(path)))
	var b strings.Builder
	for _, r := range base {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case b.Len() > 0 && !strings.HasSuffix(b.String(), "-"):
			b.WriteByte('-')
		}
	}
	key := strings.Trim(b.String(), "-")
	if len(key) > 63 {
		key = strings.TrimRight(key[:63], "-")
	}
	if key == "" || !keyPattern.MatchString(key) {
		return "", false
	}
	return key, true
}

func validateLocation(path string) (string, error) {
	if !filepath.IsAbs(path) {
		return "", ErrInvalidLocation
	}
	root := filepath.Clean(path)
	info, err := os.Lstat(root)
	if err != nil || !info.Mode().IsDir() {
		return "", ErrInvalidLocation
	}
	return root, nil
}

// locateConfig resolves the Git config path. An empty path with GitRepository
// means the config file does not exist.
func locateConfig(root string) (GitState, string) {
	gitPath := filepath.Join(root, ".git")
	info, err := os.Lstat(gitPath)
	if os.IsNotExist(err) {
		return GitAbsent, ""
	}
	if err != nil {
		return GitUnreadable, ""
	}
	gitDir := gitPath
	switch {
	case info.Mode().IsDir():
	case info.Mode().IsRegular():
		target, state := readGitLine(gitPath, "gitdir: ")
		if state != GitRepository {
			return state, ""
		}
		gitDir = resolve(root, target)
		if state := plainDir(gitDir); state != GitRepository {
			return state, ""
		}
	default:
		return GitUnsafe, ""
	}
	configDir := gitDir
	commonPath := filepath.Join(gitDir, "commondir")
	switch _, err := os.Lstat(commonPath); {
	case err == nil:
		target, state := readGitLine(commonPath, "")
		if state != GitRepository {
			return state, ""
		}
		configDir = resolve(gitDir, target)
		if state := plainDir(configDir); state != GitRepository {
			return state, ""
		}
	case !os.IsNotExist(err):
		return GitUnreadable, ""
	}
	configPath := filepath.Join(configDir, "config")
	switch info, err := os.Lstat(configPath); {
	case os.IsNotExist(err):
		return GitRepository, ""
	case err != nil:
		return GitUnreadable, ""
	case !info.Mode().IsRegular():
		return GitUnsafe, ""
	}
	return GitRepository, configPath
}

func resolve(base, p string) string {
	if filepath.IsAbs(p) {
		return filepath.Clean(p)
	}
	return filepath.Join(base, p)
}

// plainDir reports GitRepository for a non-link directory.
func plainDir(p string) GitState {
	info, err := os.Lstat(p)
	switch {
	case err != nil:
		return GitUnreadable
	case info.Mode()&os.ModeSymlink != 0:
		return GitUnsafe
	case !info.Mode().IsDir():
		return GitUnreadable
	}
	return GitRepository
}

// readRegular reads a bounded regular file; GitRepository means success.
func readRegular(p string, limit int64) ([]byte, GitState) {
	info, err := os.Lstat(p)
	switch {
	case err != nil:
		return nil, GitUnreadable
	case !info.Mode().IsRegular():
		return nil, GitUnsafe
	case info.Size() > limit:
		return nil, GitUnreadable
	}
	f, err := os.Open(p)
	if err != nil {
		return nil, GitUnreadable
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil || int64(len(data)) > limit {
		return nil, GitUnreadable
	}
	return data, GitRepository
}

// readGitLine reads a one-line pointer file (.git or commondir) and returns
// its payload after the optional prefix.
func readGitLine(p, prefix string) (string, GitState) {
	data, state := readRegular(p, maxGitFile)
	if state != GitRepository || !utf8.Valid(data) {
		if state == GitRepository {
			state = GitUnreadable
		}
		return "", state
	}
	line := strings.TrimSuffix(strings.TrimSuffix(string(data), "\n"), "\r")
	line, ok := strings.CutPrefix(line, prefix)
	if !ok || line == "" || strings.ContainsAny(line, "\r\n\x00") {
		return "", GitUnreadable
	}
	return line, GitRepository
}

type urlEntry struct{ name, raw string }

func (c config) candidates() ([]RemoteCandidate, int) {
	byLocator := map[string]map[string]struct{}{}
	unsupported := 0
	for _, e := range c.urls {
		// Query and fragment parts can carry credentials or tokens and are never
		// portable Repository identity, so such locators are unsupported.
		locator, issues := project.NormalizeLocator(e.raw)
		if len(issues) > 0 || strings.ContainsAny(locator, "?#") {
			unsupported++
			continue
		}
		names := byLocator[locator]
		if names == nil {
			names = map[string]struct{}{}
			byLocator[locator] = names
		}
		if remoteName.MatchString(e.name) {
			names[e.name] = struct{}{}
		}
	}
	out := make([]RemoteCandidate, 0, len(byLocator))
	for locator, set := range byLocator {
		names := make([]string, 0, len(set))
		for n := range set {
			names = append(names, n)
		}
		sort.Strings(names)
		out = append(out, RemoteCandidate{Locator: locator, Names: names})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Locator < out[j].Locator })
	return out, unsupported
}
