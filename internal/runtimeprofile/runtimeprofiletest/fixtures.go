// Package runtimeprofiletest supplies deterministic synthetic canonical
// format-3 Runtime/Profile configurations for tests. It is imported only by
// tests. Runtime versions and executable digests are synthetic: no fixture
// claims production Runtime support, carries a credential or proves a
// capability, and nothing here executes a Runtime.
package runtimeprofiletest

import (
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/runtimeprofile"
)

const (
	CodexRuntimeID          = "codex"
	ClaudeRuntimeID         = "claude"
	CodexRuntimeVersion     = "synthetic-codex-1"
	ClaudeRuntimeVersion    = "synthetic-claude-1"
	CodexDeclarationID      = "codex-effort-sources"
	ClaudeDeclarationID     = "claude-effort-sources"
	InvocationMode          = "noninteractive"
	CodexHighProfileID      = "codex-high"
	CodexDefaultProfileID   = "codex-default"
	CodexUnlistedProfileID  = "codex-unlisted"
	ClaudeLowProfileID      = "claude-low"
	UnknownRuntimeID        = "unknown-runtime"
	UnknownProfileID        = "unknown-profile"
	InitialRevision         = uint64(1)
	SyntheticCapability     = "code"
	SyntheticPreferenceRole = "review"
)

var (
	CodexExecutableDigest  = strings.Repeat("1", 64)
	ClaudeExecutableDigest = strings.Repeat("2", 64)
)

func ExplicitEffort(value string) runtimeprofile.Effort {
	return runtimeprofile.Effort{Mode: runtimeprofile.EffortExplicit, Value: value}
}

func RuntimeDefaultEffort() runtimeprofile.Effort {
	return runtimeprofile.Effort{Mode: runtimeprofile.EffortRuntimeDefault, Value: "default"}
}

func Profile(id, runtimeID, model string, effort runtimeprofile.Effort) runtimeprofile.ModelProfile {
	return runtimeprofile.ModelProfile{ID: id, RuntimeID: runtimeID, Model: model, Effort: effort, Capabilities: []string{SyntheticCapability}}
}

// CodexDeclaration is a valid synthetic Codex EffortDeclaration.
func CodexDeclaration() runtimeprofile.EffortDeclaration {
	return runtimeprofile.EffortDeclaration{DeclarationID: CodexDeclarationID, DeclarationVersion: runtimeprofile.EffortDeclarationVersion, RuntimeVersion: CodexRuntimeVersion, ExecutableDigest: CodexExecutableDigest, InvocationMode: InvocationMode, PrecedenceEstablished: true, JSONKeys: []string{}, EnvironmentKeys: []string{}, Sources: []runtimeprofile.EffortSourceDeclaration{
		{Scope: "user", Anchor: runtimeprofile.EffortAnchorUser, Path: ".codex/config.toml", Format: "toml"},
		{Scope: "project", Anchor: runtimeprofile.EffortAnchorProject, Path: ".codex/config.toml", Format: "toml"},
		{Scope: "managed", Anchor: runtimeprofile.EffortAnchorManaged, Path: "managed_config.toml", Format: "toml"},
	}}
}

// ClaudeDeclaration is a valid synthetic Claude EffortDeclaration; its JSON
// key is synthetic and invents no Claude settings key.
func ClaudeDeclaration() runtimeprofile.EffortDeclaration {
	return runtimeprofile.EffortDeclaration{DeclarationID: ClaudeDeclarationID, DeclarationVersion: runtimeprofile.EffortDeclarationVersion, RuntimeVersion: ClaudeRuntimeVersion, ExecutableDigest: ClaudeExecutableDigest, InvocationMode: InvocationMode, PrecedenceEstablished: true, JSONKeys: []string{"fixtureEffortKey"}, EnvironmentKeys: []string{"CLAUDE_CODE_EFFORT_LEVEL"}, Sources: []runtimeprofile.EffortSourceDeclaration{
		{Scope: "user", Anchor: runtimeprofile.EffortAnchorUser, Path: ".claude/settings.json", Format: "json"},
		{Scope: "project", Anchor: runtimeprofile.EffortAnchorProject, Path: ".claude/settings.json", Format: "json"},
		{Scope: "project-local", Anchor: runtimeprofile.EffortAnchorProject, Path: ".claude/settings.local.json", Format: "json"},
		{Scope: "managed", Anchor: runtimeprofile.EffortAnchorManaged, Path: "managed-settings.json", Format: "json"},
		{Scope: "managed-drop-in", Anchor: runtimeprofile.EffortAnchorManaged, Path: "managed-settings.d", Format: "json", Directory: true},
	}}
}

// CodexOnly is one enabled Codex Runtime allowlisting an explicit-effort and
// a runtime-default Profile, with one applicable declaration.
func CodexOnly() runtimeprofile.ConfigurationV3 {
	return runtimeprofile.ConfigurationV3{
		FormatVersion: runtimeprofile.FormatVersionV3, Revision: InitialRevision,
		Runtimes: []runtimeprofile.RuntimeV3{{ID: CodexRuntimeID, Adapter: CodexRuntimeID, Enabled: true, AllowlistedProfileIDs: []string{CodexHighProfileID, CodexDefaultProfileID}, EffortDeclarations: []runtimeprofile.EffortDeclaration{CodexDeclaration()}}},
		ModelProfiles: []runtimeprofile.ModelProfile{
			Profile(CodexHighProfileID, CodexRuntimeID, "synthetic-codex-model", ExplicitEffort("high")),
			Profile(CodexDefaultProfileID, CodexRuntimeID, "synthetic-codex-model", RuntimeDefaultEffort()),
		},
	}
}

// ClaudeOnly is one enabled Claude Runtime with one explicit-effort Profile
// and one applicable declaration.
func ClaudeOnly() runtimeprofile.ConfigurationV3 {
	return runtimeprofile.ConfigurationV3{
		FormatVersion: runtimeprofile.FormatVersionV3, Revision: InitialRevision,
		Runtimes:      []runtimeprofile.RuntimeV3{{ID: ClaudeRuntimeID, Adapter: ClaudeRuntimeID, Enabled: true, AllowlistedProfileIDs: []string{ClaudeLowProfileID}, EffortDeclarations: []runtimeprofile.EffortDeclaration{ClaudeDeclaration()}}},
		ModelProfiles: []runtimeprofile.ModelProfile{Profile(ClaudeLowProfileID, ClaudeRuntimeID, "synthetic-claude-model", ExplicitEffort("low"))},
	}
}

// MultiRuntime combines both Runtimes with different allowlists: Codex owns
// CodexUnlistedProfileID without allowlisting it.
func MultiRuntime() runtimeprofile.ConfigurationV3 {
	c := CodexOnly()
	claude := ClaudeOnly()
	c.Runtimes = append(c.Runtimes, claude.Runtimes...)
	c.ModelProfiles = append(c.ModelProfiles, claude.ModelProfiles...)
	c.ModelProfiles = append(c.ModelProfiles, Profile(CodexUnlistedProfileID, CodexRuntimeID, "synthetic-codex-model", ExplicitEffort("medium")))
	return c
}

// WithPreferences adds a nonempty Preference set.
func WithPreferences(c runtimeprofile.ConfigurationV3) runtimeprofile.ConfigurationV3 {
	c.Preferences = append(append([]runtimeprofile.Preference(nil), c.Preferences...), runtimeprofile.Preference{Role: SyntheticPreferenceRole, Complexity: "low", ModelProfileID: c.ModelProfiles[0].ID})
	return c
}

func WithRevision(c runtimeprofile.ConfigurationV3, revision uint64) runtimeprofile.ConfigurationV3 {
	c.Revision = revision
	return c
}

// WithRuntimeDisabled disables runtimeID, keeping its Profiles and declarations.
func WithRuntimeDisabled(c runtimeprofile.ConfigurationV3, runtimeID string) runtimeprofile.ConfigurationV3 {
	return mapRuntime(c, runtimeID, func(r *runtimeprofile.RuntimeV3) { r.Enabled = false })
}

// WithoutDeclarations leaves runtimeID with no applicable declaration.
func WithoutDeclarations(c runtimeprofile.ConfigurationV3, runtimeID string) runtimeprofile.ConfigurationV3 {
	return mapRuntime(c, runtimeID, func(r *runtimeprofile.RuntimeV3) { r.EffortDeclarations = nil })
}

// WithDeclaration replaces runtimeID's declarations with d.
func WithDeclaration(c runtimeprofile.ConfigurationV3, runtimeID string, d runtimeprofile.EffortDeclaration) runtimeprofile.ConfigurationV3 {
	return mapRuntime(c, runtimeID, func(r *runtimeprofile.RuntimeV3) { r.EffortDeclarations = []runtimeprofile.EffortDeclaration{d} })
}

// WithAllowlisted appends profileID to runtimeID's allowlist without checking
// it, so tests can build inconsistent references.
func WithAllowlisted(c runtimeprofile.ConfigurationV3, runtimeID, profileID string) runtimeprofile.ConfigurationV3 {
	return mapRuntime(c, runtimeID, func(r *runtimeprofile.RuntimeV3) {
		r.AllowlistedProfileIDs = append(append([]string(nil), r.AllowlistedProfileIDs...), profileID)
	})
}

func mapRuntime(c runtimeprofile.ConfigurationV3, runtimeID string, edit func(*runtimeprofile.RuntimeV3)) runtimeprofile.ConfigurationV3 {
	c.Runtimes = append([]runtimeprofile.RuntimeV3(nil), c.Runtimes...)
	for i := range c.Runtimes {
		if c.Runtimes[i].ID == runtimeID {
			edit(&c.Runtimes[i])
		}
	}
	return c
}

// CodexApplicability is the synthetic observed identity CodexDeclaration
// applies to; it is an input for lookups, not an observation.
func CodexApplicability() runtimeprofile.EffortApplicability {
	return runtimeprofile.EffortApplicability{RuntimeID: CodexRuntimeID, RuntimeVersion: CodexRuntimeVersion, ExecutableDigest: CodexExecutableDigest, InvocationMode: InvocationMode}
}

func ClaudeApplicability() runtimeprofile.EffortApplicability {
	return runtimeprofile.EffortApplicability{RuntimeID: ClaudeRuntimeID, RuntimeVersion: ClaudeRuntimeVersion, ExecutableDigest: ClaudeExecutableDigest, InvocationMode: InvocationMode}
}

// Consumer builds the canonical v3 consumer for an active record c.
func Consumer(tb testing.TB, c runtimeprofile.ConfigurationV3) runtimeprofile.ConsumerV3 {
	tb.Helper()
	consumer, err := runtimeprofile.NewConsumerV3(runtimeprofile.RecordActiveV3, &c)
	if err != nil {
		tb.Fatalf("synthetic v3 consumer: %v", err)
	}
	return consumer
}

func Encode(tb testing.TB, c runtimeprofile.ConfigurationV3) []byte {
	tb.Helper()
	wire, err := runtimeprofile.EncodeV3(c)
	if err != nil {
		tb.Fatalf("synthetic v3 encode: %v", err)
	}
	return wire
}

func Digest(tb testing.TB, c runtimeprofile.ConfigurationV3) string {
	tb.Helper()
	digest, err := runtimeprofile.DigestV3(c)
	if err != nil {
		tb.Fatalf("synthetic v3 digest: %v", err)
	}
	return digest
}
