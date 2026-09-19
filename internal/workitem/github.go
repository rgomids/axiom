package workitem

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const commandOutputLimit = 64 * 1024

type GitHubAdapter struct {
	git, gh string
	timeout time.Duration
}

func NewGitHubAdapter(gitBinary, ghBinary string) (GitHubAdapter, error) {
	if gitBinary == "" {
		var err error
		gitBinary, err = exec.LookPath("git")
		if err != nil {
			return GitHubAdapter{}, err
		}
	}
	if ghBinary == "" {
		var err error
		ghBinary, err = exec.LookPath("gh")
		if err != nil {
			return GitHubAdapter{}, err
		}
	}
	if !filepath.IsAbs(gitBinary) || !filepath.IsAbs(ghBinary) {
		return GitHubAdapter{}, errors.New("provider binaries must be absolute")
	}
	return GitHubAdapter{git: gitBinary, gh: ghBinary, timeout: 15 * time.Second}, nil
}

func (a GitHubAdapter) GitHubRepository(ctx context.Context, repositoryPath string) (string, error) {
	output, err := a.run(ctx, a.git, "-C", repositoryPath, "remote", "get-url", "origin")
	if err != nil {
		return "", err
	}
	return parseGitHubRepository(strings.TrimSpace(string(output)))
}

func (a GitHubAdapter) Create(ctx context.Context, repository, title, body string) (External, error) {
	if repository == "" || strings.TrimSpace(title) == "" {
		return External{}, errors.New("invalid create input")
	}
	output, err := a.run(ctx, a.gh, "issue", "create", "--repo", repository, "--title", title, "--body", body)
	if err != nil {
		return External{}, err
	}
	issueURL := strings.TrimSpace(string(output))
	issueRepository, number, err := issueReferenceFromURL(issueURL)
	if err != nil {
		return External{}, err
	}
	if issueRepository != repository {
		return External{}, errors.New("GitHub repository mismatch")
	}
	return External{Number: number, URL: issueURL, State: "OPEN"}, nil
}

func (a GitHubAdapter) Read(ctx context.Context, repository string, number int) (External, error) {
	if repository == "" || number <= 0 {
		return External{}, errors.New("invalid read input")
	}
	output, err := a.run(ctx, a.gh, "issue", "view", strconv.Itoa(number), "--repo", repository, "--json", "number,url,state")
	if err != nil {
		return External{}, err
	}
	var external External
	if err := json.Unmarshal(output, &external); err != nil || !validExternal(repository, number, external) {
		return External{}, errors.New("invalid GitHub response")
	}
	return external, nil
}

func (a GitHubAdapter) Comment(ctx context.Context, repository string, number int, message string) error {
	if strings.TrimSpace(message) == "" {
		return errors.New("empty comment")
	}
	_, err := a.run(ctx, a.gh, "issue", "comment", strconv.Itoa(number), "--repo", repository, "--body", message)
	return err
}

func (a GitHubAdapter) Close(ctx context.Context, repository string, number int) (External, error) {
	if _, err := a.run(ctx, a.gh, "issue", "close", strconv.Itoa(number), "--repo", repository); err != nil {
		return External{}, err
	}
	return a.Read(ctx, repository, number)
}

func (a GitHubAdapter) run(parent context.Context, binary string, arguments ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(parent, a.timeout)
	defer cancel()
	var output boundedBuffer
	command := exec.CommandContext(ctx, binary, arguments...)
	command.Stdout = &output
	command.Stderr = &output
	if err := command.Run(); err != nil {
		return nil, errors.New("provider command failed")
	}
	return output.Bytes(), nil
}

type boundedBuffer struct{ bytes.Buffer }

func (b *boundedBuffer) Write(input []byte) (int, error) {
	remaining := commandOutputLimit - b.Len()
	if remaining <= 0 {
		return 0, errors.New("provider output exceeded limit")
	}
	if len(input) > remaining {
		_, _ = b.Buffer.Write(input[:remaining])
		return remaining, errors.New("provider output exceeded limit")
	}
	return b.Buffer.Write(input)
}

func parseGitHubRepository(raw string) (string, error) {
	path := ""
	if strings.HasPrefix(raw, "git@github.com:") {
		path = strings.TrimPrefix(raw, "git@github.com:")
	} else {
		parsed, err := url.Parse(raw)
		if err != nil || !strings.EqualFold(parsed.Hostname(), "github.com") || parsed.User != nil {
			return "", errors.New("not a GitHub repository")
		}
		path = strings.TrimPrefix(parsed.Path, "/")
	}
	path = strings.TrimSuffix(path, ".git")
	if !ValidGitHubRepository(path) {
		return "", errors.New("invalid GitHub repository")
	}
	parts := strings.Split(path, "/")
	return fmt.Sprintf("%s/%s", parts[0], parts[1]), nil
}

var githubSegment = regexp.MustCompile(`^[A-Za-z0-9_.-]+$`)

func ValidGitHubRepository(value string) bool {
	parts := strings.Split(value, "/")
	return len(parts) == 2 && parts[0] != "." && parts[0] != ".." && parts[1] != "." && parts[1] != ".." && githubSegment.MatchString(parts[0]) && githubSegment.MatchString(parts[1])
}

func issueReferenceFromURL(raw string) (string, int, error) {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme != "https" || !strings.EqualFold(parsed.Hostname(), "github.com") || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", 0, errors.New("invalid issue URL")
	}
	parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	if len(parts) != 4 || parts[2] != "issues" || !ValidGitHubRepository(parts[0]+"/"+parts[1]) {
		return "", 0, errors.New("invalid issue URL")
	}
	number, err := strconv.Atoi(parts[3])
	if err != nil || number <= 0 {
		return "", 0, errors.New("invalid issue number")
	}
	return parts[0] + "/" + parts[1], number, nil
}

func validExternal(repository string, number int, external External) bool {
	issueRepository, issueNumber, err := issueReferenceFromURL(external.URL)
	return err == nil && issueRepository == repository && issueNumber == number && external.Number == number && (external.State == "OPEN" || external.State == "CLOSED")
}
