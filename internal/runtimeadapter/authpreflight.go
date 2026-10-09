package runtimeadapter

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"time"
)

// AuthStatus is the bounded outcome of the machine-local CLI subscription
// authentication preflight (Issue #272 / AXM-4). Only
// AuthSubscriptionObserved permits dispatch, and it still proves neither
// usability nor billing: a vendor status surface can report a cached login
// whose token is expired or whose plan cannot serve the request.
type AuthStatus string

const (
	// AuthSubscriptionObserved: the vendor status surface reports the
	// subscription login, and the effective child environment and the
	// inspected configuration select no API/provider override.
	AuthSubscriptionObserved AuthStatus = "subscription_observed"
	// AuthUnproven: the intended path cannot be established from observable
	// evidence (unrecognized, ambiguous or unreadable state).
	AuthUnproven AuthStatus = "unproven"
	// AuthIncompatible: an API-key, provider, helper or configuration
	// override would select another authentication or billing path.
	AuthIncompatible AuthStatus = "incompatible"
	// AuthUnavailable: no login, or no runnable executable.
	AuthUnavailable AuthStatus = "unavailable"
	// AuthUnsupported: the installed CLI exposes no recognized status surface.
	AuthUnsupported AuthStatus = "unsupported"
)

// Evidence kinds distinguish how a report was produced. A local observation
// runs only vendor status/version commands; it is never inference and never a
// real Runtime probe, which requires separate authorization.
const (
	EvidenceLocalObservation = "local_observation"
	EvidenceFake             = "fake"
)

const (
	// UsabilityUnproven is the only usability the preflight can report: only
	// an authorized real dispatch or probe exercises the login.
	UsabilityUnproven = "unproven_until_real_dispatch"
	// RevalidateBeforeDispatch: a report is a point-in-time observation and
	// is repeated at every child dispatch boundary.
	RevalidateBeforeDispatch = "before_each_dispatch"
)

const (
	statusTimeout      = 10 * time.Second
	maxStatusOutput    = 8 << 10
	maxConfigFileBytes = 1 << 20
	maxReportOverrides = 16
)

// AuthReport is presentation-safe Evidence. Every field is an enum, a
// validated version token, a digest, or an environment/configuration key
// name. Raw vendor output, configuration values, paths, credentials and
// account identifiers never enter it.
type AuthReport struct {
	RuntimeID        string     `json:"runtimeId"`
	Status           AuthStatus `json:"status"`
	Reason           string     `json:"reason,omitempty"`
	Method           string     `json:"method"`
	Version          string     `json:"version"`
	ExecutableDigest string     `json:"executableDigest,omitempty"`
	Overrides        []string   `json:"overrides,omitempty"`
	EvidenceKind     string     `json:"evidenceKind"`
	Usability        string     `json:"usability"`
	Revalidation     string     `json:"revalidation"`
}

// DispatchAllowed reports whether the subscription scenario may dispatch.
func (r AuthReport) DispatchAllowed() bool { return r.Status == AuthSubscriptionObserved }

// AuthTarget is the effective child invocation to inspect: the exact
// executable, environment and working directory the child process would get.
type AuthTarget struct {
	RuntimeID        string
	Executable       string
	ExpectedDigest   string
	WorkingDirectory string
	Environment      []string
}

type StatusCommand struct {
	Argv []string
	Env  []string
	CWD  string
}

type StatusResult struct {
	Started  bool
	TimedOut bool
	ExitCode int
	Output   []byte
}

// StatusRunner runs one read-only vendor status or version command.
type StatusRunner interface {
	RunStatus(context.Context, StatusCommand) StatusResult
	EvidenceKind() string
}

// AuthPreflight inspects vendor status surfaces and the effective invocation
// configuration. It never writes, logs in, refreshes or copies credentials,
// and never changes global shell or vendor configuration.
type AuthPreflight struct {
	runner  StatusRunner
	managed ManagedConfiguration
}

// ManagedConfiguration names the system-wide configuration files inspected in
// addition to the user and project files; absent files are fine.
type ManagedConfiguration struct {
	ClaudeSettings []string
	CodexConfig    []string
}

func NewAuthPreflight(runner StatusRunner, managed ManagedConfiguration) (AuthPreflight, error) {
	if runner == nil {
		return AuthPreflight{}, ErrInvalidAdapterConfiguration
	}
	for _, path := range append(append([]string(nil), managed.ClaudeSettings...), managed.CodexConfig...) {
		if !filepath.IsAbs(path) {
			return AuthPreflight{}, ErrInvalidAdapterConfiguration
		}
	}
	managed.ClaudeSettings = append([]string(nil), managed.ClaudeSettings...)
	managed.CodexConfig = append([]string(nil), managed.CodexConfig...)
	return AuthPreflight{runner: runner, managed: managed}, nil
}

// DefaultManagedConfiguration returns the documented system-wide files for
// this platform.
func DefaultManagedConfiguration() ManagedConfiguration {
	switch runtime.GOOS {
	case "darwin":
		return ManagedConfiguration{ClaudeSettings: []string{"/Library/Application Support/ClaudeCode/managed-settings.json"}, CodexConfig: []string{"/etc/codex/config.toml", "/etc/codex/managed_config.toml"}}
	case "linux":
		return ManagedConfiguration{ClaudeSettings: []string{"/etc/claude-code/managed-settings.json"}, CodexConfig: []string{"/etc/codex/config.toml", "/etc/codex/managed_config.toml"}}
	case "windows":
		return ManagedConfiguration{ClaudeSettings: []string{`C:\Program Files\ClaudeCode\managed-settings.json`}}
	}
	return ManagedConfiguration{}
}

func (p AuthPreflight) Check(ctx context.Context, target AuthTarget) AuthReport {
	report := AuthReport{RuntimeID: target.RuntimeID, Method: "unknown", Version: "unknown", EvidenceKind: EvidenceFake, Usability: UsabilityUnproven, Revalidation: RevalidateBeforeDispatch}
	if p.runner != nil {
		report.EvidenceKind = p.runner.EvidenceKind()
	}
	if target.RuntimeID != "codex" && target.RuntimeID != "claude" {
		report.RuntimeID = ""
		return report.with(AuthUnsupported, "runtime_unsupported")
	}
	if p.runner == nil || !filepath.IsAbs(target.WorkingDirectory) || ctx.Err() != nil {
		return report.with(AuthUnproven, "preflight_unavailable")
	}
	identity, err := ExecutableIdentity(target.Executable)
	if err != nil {
		return report.with(AuthUnavailable, "executable_unavailable")
	}
	if strings.TrimSuffix(strings.ToLower(filepath.Base(target.Executable)), ".exe") != target.RuntimeID {
		return report.with(AuthIncompatible, "executable_mismatch")
	}
	report.ExecutableDigest = identity
	if target.ExpectedDigest != "" && identity != target.ExpectedDigest {
		return report.with(AuthUnproven, "executable_identity_changed")
	}
	environment, ok := environmentMap(target.Environment)
	if !ok {
		return report.with(AuthUnproven, "environment_ambiguous")
	}
	if overrides := environmentOverrides(target.RuntimeID, environment, "env:"); len(overrides) != 0 {
		report.Overrides = bounded(overrides)
		return report.with(AuthIncompatible, "environment_override")
	}
	overrides, reason := p.configurationOverrides(target.RuntimeID, environment, target.WorkingDirectory)
	if reason != "" {
		return report.with(AuthUnproven, reason)
	}
	if len(overrides) != 0 {
		report.Overrides = bounded(overrides)
		return report.with(AuthIncompatible, "configuration_override")
	}
	command := StatusCommand{Env: append([]string{}, target.Environment...), CWD: target.WorkingDirectory}
	command.Argv = []string{target.Executable, "--version"}
	version := p.runner.RunStatus(ctx, command)
	if !version.Started {
		return report.with(AuthUnavailable, "executable_unavailable")
	}
	if version.TimedOut || version.ExitCode != 0 {
		return report.with(AuthUnproven, "version_unavailable")
	}
	if report.Version = versionToken(version.Output); report.Version == "unknown" {
		return report.with(AuthUnproven, "version_unrecognized")
	}
	if target.RuntimeID == "codex" {
		command.Argv = []string{target.Executable, "login", "status"}
	} else {
		command.Argv = []string{target.Executable, "auth", "status", "--json"}
	}
	status := p.runner.RunStatus(ctx, command)
	if !status.Started {
		return report.with(AuthUnavailable, "executable_unavailable")
	}
	if status.TimedOut {
		return report.with(AuthUnproven, "status_timeout")
	}
	var outcome AuthStatus
	if target.RuntimeID == "codex" {
		report.Method, outcome, reason = classifyCodexStatus(status)
	} else {
		report.Method, outcome, reason = classifyClaudeStatus(status)
	}
	// The status commands ran the file; it must still be the inspected one.
	if after, err := ExecutableIdentity(target.Executable); err != nil || after != identity {
		return report.with(AuthUnproven, "executable_identity_changed")
	}
	return report.with(outcome, reason)
}

func (r AuthReport) with(status AuthStatus, reason string) AuthReport {
	r.Status, r.Reason = status, reason
	return r
}

func bounded(values []string) []string {
	sort.Strings(values)
	values = compactStrings(values)
	if len(values) > maxReportOverrides {
		values = values[:maxReportOverrides]
	}
	return values
}

func compactStrings(values []string) []string {
	result := values[:0]
	for index, value := range values {
		if index == 0 || value != values[index-1] {
			result = append(result, value)
		}
	}
	return result
}

// environmentMap normalizes keys to upper case, matching Windows semantics,
// and rejects duplicate keys whose effective value would be ambiguous.
func environmentMap(environment []string) (map[string]string, bool) {
	result := make(map[string]string, len(environment))
	for _, entry := range environment {
		key, value, ok := strings.Cut(entry, "=")
		if !ok || key == "" || strings.ContainsRune(entry, '\x00') {
			return nil, false
		}
		key = strings.ToUpper(key)
		if _, exists := result[key]; exists {
			return nil, false
		}
		result[key] = value
	}
	return result, true
}

// environmentOverrides names the variables that select API-key, provider,
// gateway or token-based authentication instead of the local subscription
// login. Codex exec honors CODEX_API_KEY even while its status surface
// reports ChatGPT; Claude --print gives ANTHROPIC_API_KEY and cloud
// providers precedence over the claude.ai login.
func environmentOverrides(runtimeID string, environment map[string]string, prefix string) []string {
	var prefixes, exact []string
	if runtimeID == "codex" {
		prefixes = []string{"OPENAI_", "AZURE_OPENAI_"}
		exact = []string{"CODEX_API_KEY"}
	} else {
		prefixes = []string{"ANTHROPIC_", "CLAUDE_CODE_USE_", "CLAUDE_CODE_SKIP_", "CLOUD_ML_", "VERTEX_"}
		exact = []string{"CLAUDE_CODE_OAUTH_TOKEN", "AWS_BEARER_TOKEN_BEDROCK", "CLAUDE_CODE_API_KEY_HELPER_TTL_MS"}
	}
	var overrides []string
	for key := range environment {
		// Token, key and endpoint variables of the vendor's own namespace
		// (for example CLAUDE_CODE_API_KEY_FILE_DESCRIPTOR) also select
		// another authentication path.
		namespace := "CODEX_"
		if runtimeID == "claude" {
			namespace = "CLAUDE_CODE_"
		}
		matched := strings.HasPrefix(key, namespace) && (strings.Contains(key, "TOKEN") || strings.Contains(key, "API_KEY") || strings.Contains(key, "URL"))
		for _, candidate := range prefixes {
			matched = matched || strings.HasPrefix(key, candidate)
		}
		for _, candidate := range exact {
			matched = matched || key == candidate
		}
		if matched {
			overrides = append(overrides, prefix+safeName(key))
		}
	}
	return overrides
}

func safeName(name string) string {
	if name == "" || len(name) > 128 {
		return "unnamed"
	}
	for _, char := range name {
		if !(char >= 'A' && char <= 'Z' || char >= 'a' && char <= 'z' || char >= '0' && char <= '9' || char == '_' || char == '-' || char == '.') {
			return "unnamed"
		}
	}
	return name
}

func (p AuthPreflight) configurationOverrides(runtimeID string, environment map[string]string, cwd string) ([]string, string) {
	home := environment["HOME"]
	if home == "" {
		home = environment["USERPROFILE"]
	}
	var overrides []string
	if runtimeID == "codex" {
		root := environment["CODEX_HOME"]
		if root == "" && filepath.IsAbs(home) {
			root = filepath.Join(home, ".codex")
		}
		if !filepath.IsAbs(root) {
			return nil, "configuration_root_unresolved"
		}
		files := []struct{ scope, path string }{{"user", filepath.Join(root, "config.toml")}, {"managed", filepath.Join(root, "managed_config.toml")}, {"project", filepath.Join(cwd, ".codex", "config.toml")}}
		for _, path := range p.managed.CodexConfig {
			files = append(files, struct{ scope, path string }{"managed", path})
		}
		for _, file := range files {
			content, found, err := readBoundedFile(file.path)
			if err != nil {
				return nil, "configuration_unreadable"
			}
			if found {
				overrides = append(overrides, codexConfigOverrides(content, "codex_config:"+file.scope+":")...)
			}
		}
		return overrides, ""
	}
	root := environment["CLAUDE_CONFIG_DIR"]
	if root == "" && filepath.IsAbs(home) {
		root = filepath.Join(home, ".claude")
	}
	if !filepath.IsAbs(root) {
		return nil, "configuration_root_unresolved"
	}
	files := []struct{ scope, path string }{
		{"user", filepath.Join(root, "settings.json")},
		{"project", filepath.Join(cwd, ".claude", "settings.json")},
		{"project_local", filepath.Join(cwd, ".claude", "settings.local.json")},
	}
	for _, path := range p.managed.ClaudeSettings {
		files = append(files, struct{ scope, path string }{"managed", path})
		dropIns, err := filepath.Glob(filepath.Join(filepath.Dir(path), "managed-settings.d", "*.json"))
		if err != nil {
			return nil, "configuration_unreadable"
		}
		for _, dropIn := range dropIns {
			files = append(files, struct{ scope, path string }{"managed", dropIn})
		}
	}
	for _, file := range files {
		content, found, err := readBoundedFile(file.path)
		if err != nil {
			return nil, "configuration_unreadable"
		}
		if !found {
			continue
		}
		settings, ok := claudeSettingsOverrides(content, "claude_settings:"+file.scope+":")
		if !ok {
			return nil, "configuration_unreadable"
		}
		overrides = append(overrides, settings...)
	}
	return overrides, ""
}

// readBoundedFile reads one regular configuration file, checking the opened
// file itself. Absence is not an error; any other failure is unreadable.
func readBoundedFile(path string) ([]byte, bool, error) {
	file, err := os.Open(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, ErrInvalidAdapterConfiguration
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() > maxConfigFileBytes {
		return nil, false, ErrInvalidAdapterConfiguration
	}
	content, err := io.ReadAll(io.LimitReader(file, maxConfigFileBytes+1))
	if err != nil || len(content) > maxConfigFileBytes {
		return nil, false, ErrInvalidAdapterConfiguration
	}
	return content, true, nil
}

// codexConfigOverrides is a conservative line scan of config.toml for the
// keys that select another provider or the API-key login. It reports key
// names only and never retains values.
func codexConfigOverrides(content []byte, prefix string) []string {
	var overrides []string
	for _, raw := range strings.Split(string(content), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") {
			if first, _ := tomlKeySegments(strings.Trim(line, "[] \t")); first == "model_providers" {
				overrides = append(overrides, prefix+"model_providers")
			}
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		first, last := tomlKeySegments(key)
		if inline := strings.TrimSpace(value); strings.HasPrefix(inline, "{") || strings.HasPrefix(inline, "[") {
			// Inline tables/arrays are not interpreted: any watched key inside
			// them is an override.
			for _, watched := range []string{"model_provider", "base_url", "preferred_auth_method", "forced_login_method"} {
				if strings.Contains(inline, watched) {
					overrides = append(overrides, prefix+safeName(last))
					break
				}
			}
		}
		value = tomlScalar(value)
		switch {
		case strings.HasSuffix(last, "base_url"):
			overrides = append(overrides, prefix+safeName(last))
		case first == "model_providers" || last == "model_providers":
			overrides = append(overrides, prefix+"model_providers")
		case last == "model_provider" && value != "openai":
			overrides = append(overrides, prefix+"model_provider")
		case last == "preferred_auth_method" && value != "chatgpt":
			overrides = append(overrides, prefix+"preferred_auth_method")
		case last == "forced_login_method" && value != "chatgpt":
			overrides = append(overrides, prefix+"forced_login_method")
		}
	}
	return overrides
}

func tomlKeySegments(key string) (string, string) {
	parts := strings.Split(strings.TrimSpace(key), ".")
	for index := range parts {
		parts[index] = strings.Trim(strings.TrimSpace(parts[index]), `"'`)
	}
	return parts[0], parts[len(parts)-1]
}

func tomlScalar(value string) string {
	value = strings.TrimSpace(value)
	for _, quote := range []string{`"`, `'`} {
		if strings.HasPrefix(value, quote) {
			if end := strings.Index(value[1:], quote); end >= 0 {
				return value[1 : end+1]
			}
			return value
		}
	}
	value, _, _ = strings.Cut(value, "#")
	return strings.TrimSpace(value)
}

// claudeSettingsOverrides reports settings that select API-key helpers,
// cloud-provider credentials, the Console (API) login or an environment
// override. Unparseable settings are not assumed safe.
func claudeSettingsOverrides(content []byte, prefix string) ([]string, bool) {
	var settings map[string]json.RawMessage
	if err := json.Unmarshal(content, &settings); err != nil {
		return nil, false
	}
	var overrides []string
	for _, key := range []string{"apiKeyHelper", "awsCredentialExport", "awsAuthRefresh", "gcpAuthRefresh"} {
		if _, exists := settings[key]; exists {
			overrides = append(overrides, prefix+key)
		}
	}
	if raw, exists := settings["forceLoginMethod"]; exists {
		var method string
		if json.Unmarshal(raw, &method) != nil || method != "claudeai" {
			overrides = append(overrides, prefix+"forceLoginMethod")
		}
	}
	if raw, exists := settings["env"]; exists {
		var environment map[string]json.RawMessage
		if json.Unmarshal(raw, &environment) != nil {
			return nil, false
		}
		keys := make(map[string]string, len(environment))
		for key := range environment {
			keys[strings.ToUpper(key)] = ""
		}
		overrides = append(overrides, environmentOverrides("claude", keys, prefix+"env:")...)
	}
	return overrides, true
}

var versionPattern = regexp.MustCompile(`\b(\d{1,6}\.\d{1,6}\.\d{1,6}(?:-(?:alpha|beta|rc)(?:\.\d{1,4})?)?)\b`)

func versionToken(output []byte) string {
	if match := versionPattern.FindSubmatch(output); match != nil {
		return string(match[1])
	}
	return "unknown"
}

// classifyCodexStatus recognizes only the exact `codex login status` lines.
// Warnings and other lines are ignored; their content never leaves here.
func classifyCodexStatus(result StatusResult) (string, AuthStatus, string) {
	methods := map[string]bool{}
	for _, line := range strings.Split(string(result.Output), "\n") {
		line = strings.TrimSpace(line)
		switch {
		case line == "Logged in using ChatGPT":
			methods["chatgpt"] = true
		case strings.HasPrefix(line, "Logged in using an API key"):
			methods["api_key"] = true
		case line == "Not logged in":
			methods["none"] = true
		}
	}
	if len(methods) > 1 {
		return "unknown", AuthUnproven, "status_ambiguous"
	}
	switch {
	case methods["chatgpt"] && result.ExitCode == 0:
		return "chatgpt", AuthSubscriptionObserved, ""
	case methods["chatgpt"]:
		return "unknown", AuthUnproven, "status_ambiguous"
	case methods["api_key"]:
		return "api_key", AuthIncompatible, "api_key_login"
	case methods["none"]:
		return "none", AuthUnavailable, "not_logged_in"
	case result.ExitCode != 0:
		return "unknown", AuthUnsupported, "status_surface_unsupported"
	}
	return "unknown", AuthUnproven, "status_unrecognized"
}

// classifyClaudeStatus decodes only loggedIn, authMethod and apiProvider from
// `claude auth status --json`; identity fields are never decoded.
func classifyClaudeStatus(result StatusResult) (string, AuthStatus, string) {
	var status struct {
		LoggedIn    *bool   `json:"loggedIn"`
		AuthMethod  *string `json:"authMethod"`
		APIProvider *string `json:"apiProvider"`
	}
	output := result.Output
	start, end := bytes.IndexByte(output, '{'), bytes.LastIndexByte(output, '}')
	if start < 0 || end < start || json.Unmarshal(output[start:end+1], &status) != nil || status.LoggedIn == nil || status.AuthMethod == nil {
		if result.ExitCode != 0 {
			return "unknown", AuthUnsupported, "status_surface_unsupported"
		}
		return "unknown", AuthUnproven, "status_unrecognized"
	}
	method := strings.ToLower(*status.AuthMethod)
	switch {
	case !*status.LoggedIn:
		return "none", AuthUnavailable, "not_logged_in"
	case status.APIProvider != nil && *status.APIProvider != "firstParty" || method == "third_party":
		return "third_party", AuthIncompatible, "provider_override"
	case method == "claude.ai" && result.ExitCode == 0:
		return "claude.ai", AuthSubscriptionObserved, ""
	case method == "claude.ai":
		return "unknown", AuthUnproven, "status_ambiguous"
	case strings.Contains(method, "api") || strings.Contains(method, "console"):
		return "api_key", AuthIncompatible, "api_key_login"
	}
	return "unknown", AuthUnproven, "auth_method_unrecognized"
}

// OSStatusRunner runs vendor status/version commands with exactly the given
// environment, no stdin, a timeout and bounded output.
type OSStatusRunner struct{}

func (OSStatusRunner) EvidenceKind() string { return EvidenceLocalObservation }

func (OSStatusRunner) RunStatus(ctx context.Context, command StatusCommand) StatusResult {
	if len(command.Argv) == 0 || !filepath.IsAbs(command.Argv[0]) {
		return StatusResult{}
	}
	ctx, cancel := context.WithTimeout(ctx, statusTimeout)
	defer cancel()
	process := exec.CommandContext(ctx, command.Argv[0], command.Argv[1:]...)
	process.Dir = command.CWD
	process.Env = append([]string{}, command.Env...)
	output := &boundedOutput{remaining: maxStatusOutput}
	process.Stdout, process.Stderr = output, output
	// A descendant holding the output pipe cannot hold Wait past the timeout.
	process.WaitDelay = time.Second
	if err := process.Start(); err != nil {
		return StatusResult{}
	}
	err := process.Wait()
	result := StatusResult{Started: true, Output: output.Bytes(), TimedOut: errors.Is(ctx.Err(), context.DeadlineExceeded)}
	var exitError *exec.ExitError
	if errors.As(err, &exitError) {
		result.ExitCode = exitError.ExitCode()
	} else if err != nil {
		result.ExitCode = -1
	}
	return result
}

type boundedOutput struct {
	bytes.Buffer
	remaining int
}

func (b *boundedOutput) Write(value []byte) (int, error) {
	if b.remaining > 0 {
		keep := value
		if len(keep) > b.remaining {
			keep = keep[:b.remaining]
		}
		b.Buffer.Write(keep)
		b.remaining -= len(keep)
	}
	return len(value), nil
}
