package runtimeadapter

import (
	"path/filepath"
	"sort"
	"strings"

	"github.com/rgomids/axiom/internal/executiongraph"
)

// StageInvocationRequest is an internal #276 seam, deliberately separate from
// historical configured invocations. Version must be the observed binding,
// not a user-selected compatibility override. This builder proves no capability.
type StageInvocationRequest struct {
	Profile           CommandProfile
	Version           string
	Workspace, Inbox  string
	AxiomPath         string
	Process           bool
	RepositoryWrite   bool
	ParentEnvironment []string
}

// BuildStageInvocation owns every option and output bound. Caller configuration
// may select the reviewed executable/model, but cannot supply argv or credentials.
func BuildStageInvocation(request StageInvocationRequest) (executiongraph.Invocation, error) {
	p := request.Profile
	if !stagePath(p.Executable) || strings.TrimSuffix(strings.ToLower(filepath.Base(p.Executable)), ".exe") != p.RuntimeID || p.Model == "" || strings.HasPrefix(p.Model, "-") || strings.ContainsAny(p.Model, "\x00\r\n") || len(p.Arguments) != 0 || len(p.Environment) != 0 || p.CredentialReference != "" || !stagePath(request.Workspace) || !stagePath(request.Inbox) || !stagePath(request.AxiomPath) || stageContains(request.Workspace, request.Inbox) || stageContains(request.Inbox, request.Workspace) || strings.ContainsAny(request.AxiomPath, " ,():*;\"'`$|&<>\\") {
		return executiongraph.Invocation{}, ErrInvalidAdapterConfiguration
	}
	environment, err := StageEnvironment(p.RuntimeID, request.ParentEnvironment)
	if err != nil {
		return executiongraph.Invocation{}, err
	}
	argv := []string{p.Executable}
	cwd := request.Workspace
	switch p.RuntimeID {
	case "codex":
		if request.Version != "0.159.1" {
			return executiongraph.Invocation{}, ErrInvalidAdapterConfiguration
		}
		argv = append(argv, "exec", "--model", p.Model, "--ignore-user-config", "--ephemeral", "-")
		if !request.RepositoryWrite {
			cwd = request.Inbox
		}
		argv = append(argv, "--sandbox", "workspace-write", "-C", cwd)
		if request.RepositoryWrite {
			argv = append(argv, "--add-dir", request.Inbox)
		}
	case "claude":
		if request.Version != "2.1.296" {
			return executiongraph.Invocation{}, ErrInvalidAdapterConfiguration
		}
		tools := "Read,Glob,Grep,Write"
		if request.RepositoryWrite {
			tools += ",Edit"
		} else {
			cwd = request.Inbox
		}
		tools += ",Bash(" + request.AxiomPath + ":*)"
		if request.Process {
			tools = strings.TrimSuffix(tools, ",Bash("+request.AxiomPath+":*)") + ",Bash"
		}
		argv = append(argv, "--print", "--model", p.Model, "--input-format", "text", "--permission-prompts", "none", "--permission-mode", "dontAsk", "--allowedTools", tools, "--add-dir", request.Inbox)
	default:
		return executiongraph.Invocation{}, ErrInvalidAdapterConfiguration
	}
	return executiongraph.WithExactEnvironment(executiongraph.Invocation{RuntimeID: p.RuntimeID, Argv: argv, CWD: cwd, OutputMax: executiongraph.MaxCapturedOutputBytes}, environment), nil
}

// StageEnvironment drops provider credentials and all unlisted variables.
// Correlation variables are never inherited: a value in the supervisor's own
// environment belongs to no attempt. Only the attempt context may set them.
func StageEnvironment(runtimeID string, parent []string) ([]string, error) {
	if runtimeID != "codex" && runtimeID != "claude" {
		return nil, ErrInvalidAdapterConfiguration
	}
	values := map[string]string{}
	for _, entry := range parent {
		key, _, ok := strings.Cut(entry, "=")
		if !ok || !stageEnvironmentKey(key) || strings.ContainsRune(entry, '\x00') {
			return nil, ErrInvalidAdapterConfiguration
		}
		allowed := key == "HOME" || key == "USER" || key == "LOGNAME" || key == "PATH" || key == "SHELL" || key == "TMPDIR" || key == "LANG" || strings.HasPrefix(key, "LC_") || runtimeID == "codex" && key == "CODEX_HOME" || runtimeID == "claude" && key == "CLAUDE_CONFIG_DIR"
		if !allowed {
			continue
		}
		if _, exists := values[key]; exists {
			return nil, ErrInvalidAdapterConfiguration
		}
		values[key] = entry
	}
	result := make([]string, 0, len(values))
	for _, entry := range values {
		result = append(result, entry)
	}
	sort.Strings(result)
	return result, nil
}

func stagePath(path string) bool {
	if !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return false
	}
	for _, char := range path {
		if char < 32 || char == 127 {
			return false
		}
	}
	return true
}
func stageContains(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	return err != nil || rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}
func stageEnvironmentKey(key string) bool {
	if key == "" {
		return false
	}
	for i, char := range key {
		if !(char == '_' || char >= 'A' && char <= 'Z' || char >= 'a' && char <= 'z' || i > 0 && char >= '0' && char <= '9') {
			return false
		}
	}
	return true
}
