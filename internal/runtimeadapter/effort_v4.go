package runtimeadapter

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"reflect"

	"github.com/rgomids/axiom/internal/executiongraph"
	"github.com/rgomids/axiom/internal/runtimeprofile"
)

var ErrEffortConfigurationUnsafe = runtimeprofile.ErrConfigurationUnsafe
var ErrEffortConfigurationInvalid = errors.Join(runtimeprofile.ErrConfigurationInvalid, ErrEffortConfigurationUnsafe)

type EffortSourceRequestV4 struct {
	Consented        bool
	Configuration    runtimeprofile.ConfigurationV4
	ProfileID        string
	ExecutableDigest string
	InvocationMode   string
	Roots            EffortSourceRoots
	Invocation       executiongraph.Invocation
}

type effortProjectionV4 struct {
	AdapterRule      string                `json:"adapterRule"`
	RuntimeID        string                `json:"runtimeId"`
	Model            string                `json:"model"`
	Effort           runtimeprofile.Effort `json:"effort"`
	ExecutableDigest string                `json:"executableDigest"`
	InvocationMode   string                `json:"invocationMode"`
	Facts            []effortSourceFact    `json:"facts"`
}

func (r StageEffortRoots) ResolveOperational(ctx context.Context, inv executiongraph.Invocation, repositoryWrite bool) (EffortSourceRoots, error) {
	if ctx.Err() != nil || !stagePath(inv.CWD) || inv.CWD == string(filepath.Separator) || !stagePath(r.Repository) || r.Repository == string(filepath.Separator) {
		return EffortSourceRoots{}, ErrEffortConfigurationUnsafe
	}
	home, ok := effectiveHome(executiongraph.EffectiveInvocationEnvironment(inv))
	if !ok {
		return EffortSourceRoots{}, ErrEffortConfigurationUnsafe
	}
	managed, ok := managedEffortRoot(inv.RuntimeID)
	if !ok {
		return EffortSourceRoots{}, ErrEffortConfigurationUnsafe
	}
	project := inv.CWD
	if repositoryWrite {
		project = r.Repository
	}
	return EffortSourceRoots{User: home, Project: project, Managed: managed}, nil
}

func InspectEffortSourcesV4(ctx context.Context, r EffortSourceRequestV4) (string, error) {
	projection, _, err := inspectEffortV4(ctx, r)
	if err != nil {
		return "", err
	}
	wire, err := json.Marshal(projection)
	if err != nil {
		return "", ErrEffortConfigurationUnsafe
	}
	return effortHash(wire), nil
}

func inspectEffortV4(ctx context.Context, r EffortSourceRequestV4) (effortProjectionV4, effortRules, error) {
	var none effortProjectionV4
	if ctx.Err() != nil || !r.Consented || r.InvocationMode != "noninteractive" || len(r.Invocation.Argv) == 0 {
		return none, effortRules{}, ErrEffortConfigurationUnsafe
	}
	consumer, err := runtimeprofile.NewConsumerV4(r.Configuration)
	if err != nil {
		return none, effortRules{}, ErrEffortConfigurationUnsafe
	}
	profile, err := consumer.AllowedProfile(r.Invocation.RuntimeID, r.ProfileID)
	if err != nil {
		return none, effortRules{}, ErrEffortConfigurationUnsafe
	}
	home, ok := effectiveHome(executiongraph.EffectiveInvocationEnvironment(r.Invocation))
	if !ok || home != r.Roots.User || r.Roots.User == r.Roots.Project || r.Roots.User == r.Roots.Managed || r.Roots.Project == r.Roots.Managed {
		return none, effortRules{}, ErrEffortConfigurationUnsafe
	}
	anchors := map[string]string{runtimeprofile.EffortAnchorUser: r.Roots.User, runtimeprofile.EffortAnchorProject: r.Roots.Project, runtimeprofile.EffortAnchorManaged: r.Roots.Managed}
	bindings := map[string]string{}
	for anchor, root := range anchors {
		binding, err := trustedEffortAnchor(anchor, root)
		if err != nil {
			return none, effortRules{}, ErrEffortConfigurationUnsafe
		}
		bindings[anchor] = binding
	}
	rules := fixedEffortRules(profile.RuntimeID, r.Roots)
	intent := effortIntent{profile.RuntimeID, r.InvocationMode, r.ExecutableDigest, profile.Model, profile.Effort}
	facts, err := observeEffortFacts(ctx, intent, rules, r.Invocation)
	if err != nil {
		return none, effortRules{}, operationalEffortError(err)
	}
	for anchor, root := range anchors {
		after, err := trustedEffortAnchor(anchor, root)
		if err != nil || after != bindings[anchor] {
			return none, effortRules{}, ErrEffortConfigurationUnsafe
		}
	}
	if after, err := ExecutableIdentity(r.Invocation.Argv[0]); err != nil || after != r.ExecutableDigest || ctx.Err() != nil {
		return none, effortRules{}, ErrEffortConfigurationUnsafe
	}
	return effortProjectionV4{AdapterRule: "axiom-effort-source/v2/" + profile.RuntimeID, RuntimeID: profile.RuntimeID, Model: profile.Model, Effort: profile.Effort, ExecutableDigest: r.ExecutableDigest, InvocationMode: r.InvocationMode, Facts: facts}, rules, nil
}

func fixedEffortRules(runtimeID string, roots EffortSourceRoots) effortRules {
	if runtimeID == "codex" {
		return effortRules{NestedProfiles: true, Sources: []EffortSource{
			{Scope: "user", Root: roots.User, Relative: filepath.Join(".codex", "config.toml"), Format: "toml"},
			{Scope: "project", Root: roots.Project, Relative: filepath.Join(".codex", "config.toml"), Format: "toml"},
			{Scope: "managed", Root: roots.Managed, Relative: "config.toml", Format: "toml"},
			{Scope: "managed", Root: roots.Managed, Relative: "managed_config.toml", Format: "toml"},
		}}
	}
	return effortRules{JSONKeys: []string{"effortLevel", "maxEffortLevel", "modelSettings", "CLAUDE_CODE_EFFORT_LEVEL"}, EnvironmentKeys: []string{"CLAUDE_CODE_EFFORT_LEVEL"}, Sources: []EffortSource{
		{Scope: "user", Root: roots.User, Relative: filepath.Join(".claude", "settings.json"), Format: "json"},
		{Scope: "project", Root: roots.Project, Relative: filepath.Join(".claude", "settings.json"), Format: "json"},
		{Scope: "project-local", Root: roots.Project, Relative: filepath.Join(".claude", "settings.local.json"), Format: "json"},
		{Scope: "managed", Root: roots.Managed, Relative: "managed-settings.json", Format: "json"},
		{Scope: "managed-drop-in", Root: roots.Managed, Relative: "managed-settings.d", Format: "json", Directory: true},
	}}
}

func VerifyChildEquivalenceV4(ctx context.Context, r EffortSourceRequestV4, child string) error {
	if !stagePath(child) || child == r.Roots.Project || child == r.Roots.User || child == r.Roots.Managed || r.Invocation.CWD != child {
		return ErrEffortConfigurationUnsafe
	}
	repository, rules, err := inspectEffortV4(ctx, r)
	if err != nil {
		return err
	}
	childRequest := r
	childRequest.Roots.Project = child
	childProjection, _, err := inspectEffortV4(ctx, childRequest)
	if err != nil {
		return err
	}
	projectScopes := map[string]bool{}
	for _, source := range rules.Sources {
		if source.Root != r.Roots.Project {
			continue
		}
		mirror := source
		mirror.Root = child
		repoBytes, repoPresent, _, repoErr := readEffortSource(source)
		childBytes, childPresent, _, childErr := readEffortSource(mirror)
		if source.Directory || repoErr != nil || childErr != nil || repoPresent != childPresent || !bytes.Equal(repoBytes, childBytes) {
			return ErrEffortConfigurationUnsafe
		}
		projectScopes[source.Scope] = true
	}
	for _, projection := range []*effortProjectionV4{&repository, &childProjection} {
		for i := range projection.Facts {
			if projectScopes[projection.Facts[i].Scope] {
				projection.Facts[i].Binding = ""
			}
		}
	}
	if !reflect.DeepEqual(repository, childProjection) || ctx.Err() != nil {
		return ErrEffortConfigurationUnsafe
	}
	return nil
}

func operationalEffortError(err error) error {
	if errors.Is(err, ErrEffortConfigurationInvalid) {
		return ErrEffortConfigurationInvalid
	}
	if errors.Is(err, ErrEffortOverrideConflict) {
		return ErrEffortOverrideConflict
	}
	return ErrEffortConfigurationUnsafe
}
