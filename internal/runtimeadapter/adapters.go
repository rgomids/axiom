// Package runtimeadapter contains the two concrete S8 Runtime boundaries.
package runtimeadapter

import (
	"context"
	"errors"
	"path/filepath"
	"sort"
	"strings"

	"github.com/rgomids/axiom/internal/executiongraph"
)

var ErrInvalidAdapterConfiguration = errors.New("invalid runtime adapter configuration")

type CommandProfile struct {
	RuntimeID, ModelProfileID string
	Executable                string
	Model                     string
	Arguments                 []string
	Environment               []string
	CredentialReference       string
	OutputMax                 int
}

type CredentialResolver interface {
	ResolveEnvironment(context.Context, string) ([]string, error)
}

type InvocationResolver struct {
	profiles    map[string]CommandProfile
	credentials CredentialResolver
}

func NewInvocationResolver(profiles []CommandProfile, credentials CredentialResolver) (InvocationResolver, error) {
	resolver := InvocationResolver{profiles: make(map[string]CommandProfile, len(profiles)), credentials: credentials}
	for _, profile := range profiles {
		if !validCommandProfile(profile) {
			return InvocationResolver{}, ErrInvalidAdapterConfiguration
		}
		key := profile.RuntimeID + "\x00" + profile.ModelProfileID
		if _, exists := resolver.profiles[key]; exists {
			return InvocationResolver{}, ErrInvalidAdapterConfiguration
		}
		profile.Arguments = append([]string(nil), profile.Arguments...)
		profile.Environment = append([]string(nil), profile.Environment...)
		resolver.profiles[key] = profile
	}
	return resolver, nil
}

func (r InvocationResolver) ResolveInvocation(ctx context.Context, child executiongraph.ChildExecution) (executiongraph.Invocation, error) {
	resolution := child.Envelope.Resolution
	profile, exists := r.profiles[resolution.RuntimeID+"\x00"+resolution.ModelProfileID]
	if !exists {
		return executiongraph.Invocation{}, ErrInvalidAdapterConfiguration
	}
	argv := []string{profile.Executable}
	switch profile.RuntimeID {
	case "codex":
		argv = append(argv, "exec", "--model", profile.Model)
	case "claude":
		argv = append(argv, "--print", "--model", profile.Model)
	case "cursor-agent":
		argv = append(argv, "-p", "--model", profile.Model, "--output-format", "text", "--force")
	default:
		return executiongraph.Invocation{}, ErrInvalidAdapterConfiguration
	}
	environment := append([]string(nil), profile.Environment...)
	argv = append(argv, profile.Arguments...)
	if effort := child.Envelope.Controls.ReasoningEffort; effort != "" {
		proven := false
		for _, capability := range child.Envelope.Capability.Capabilities {
			proven = proven || capability == "reasoning-effort-"+effort
		}
		if !proven || len(effort) > 64 || strings.ContainsAny(effort, "\x00\r\n \t") {
			return executiongraph.Invocation{}, ErrInvalidAdapterConfiguration
		}
		switch profile.RuntimeID {
		case "codex":
			argv = append(argv, "--config", "model_reasoning_effort="+effort)
		case "claude":
			environment = append(environment, "CLAUDE_CODE_EFFORT_LEVEL="+effort)
		}
	}
	if profile.CredentialReference != "" {
		if r.credentials == nil {
			return executiongraph.Invocation{}, ErrInvalidAdapterConfiguration
		}
		credentialEnvironment, err := r.credentials.ResolveEnvironment(ctx, profile.CredentialReference)
		if err != nil {
			return executiongraph.Invocation{}, err
		}
		environment = append(environment, credentialEnvironment...)
	}
	sort.Strings(environment)
	invocation := executiongraph.Invocation{RuntimeID: profile.RuntimeID, Argv: argv, CWD: child.Envelope.Workspace, Env: environment, OutputMax: profile.OutputMax}
	if child.Envelope.Controls.ReasoningEffort != "" && profile.RuntimeID == "claude" {
		// Keep inherited process variables separate from validated explicit
		// overrides, preserving profile and credential isolation.
		if len(profile.Environment) == 0 && profile.CredentialReference == "" {
			invocation = executiongraph.WithInheritedEnvironment(invocation)
		}
		count := 0
		for _, value := range executiongraph.EffectiveInvocationEnvironment(invocation) {
			if strings.HasPrefix(value, "CLAUDE_CODE_EFFORT_LEVEL=") {
				count++
			}
		}
		if count != 1 {
			return executiongraph.Invocation{}, ErrInvalidAdapterConfiguration
		}
	}
	return invocation, nil
}

func validCommandProfile(profile CommandProfile) bool {
	if profile.RuntimeID != "codex" && profile.RuntimeID != "claude" && profile.RuntimeID != "cursor-agent" || profile.ModelProfileID == "" || profile.Model == "" || !filepath.IsAbs(profile.Executable) || profile.OutputMax <= 0 || profile.OutputMax > executiongraph.MaxCapturedOutputBytes || len(profile.Arguments) > 32 || len(profile.Environment) > 32 {
		return false
	}
	if filepath.Base(profile.Executable) != profile.RuntimeID {
		return false
	}
	for _, argument := range profile.Arguments {
		if argument == "" || strings.ContainsRune(argument, '\x00') {
			return false
		}
	}
	for _, environment := range profile.Environment {
		if !strings.Contains(environment, "=") || strings.ContainsRune(environment, '\x00') {
			return false
		}
	}
	return profile.CredentialReference == "" || (!strings.ContainsAny(profile.CredentialReference, "\x00\r\n=") && len(profile.CredentialReference) <= 256)
}
