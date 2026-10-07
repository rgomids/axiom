package projectapp

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"regexp"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/rgomids/axiom/internal/project"
)

// Issue #231 guided bootstrap: project configure CREATE. This file is pure.
// Composition performs read-only discovery and passes its results in; this
// layer applies the frozen ambiguity, override and portability rules
// (project-bootstrap-v3.md §3) and builds one digest-bound proposal.

const WorkItemCapability = "work-item"

const (
	maxSetupKeyBytes          = 63
	maxSetupNameBytes         = 256
	maxSetupPathBytes         = 4096
	maxSetupRepositoryCount   = 32
	maxSetupDestinationBytes  = 4096
	maxSetupRevisionBytes     = 128
	maxBusinessContextBytes   = project.MaxBusinessContextBytes
	maxSetupRuntimeSelections = 8
	maxSetupProfiles          = 32
	maxSetupPreferences       = 32
)

// Bootstrap-only blocker codes: a preview carrying any of them is reviewable
// but never publishable.
const (
	BlockerRemoteAmbiguous     = "repository_remote_ambiguous"
	BlockerRepositoryKey       = "repository_key_required"
	BlockerRepositoryKeyClash  = "repository_key_conflict"
	BlockerRepositoryUnsafe    = "repository_unsafe"
	RemoteNone                 = "none"
	remoteSourceDiscovered     = "discovered"
	remoteSourceOperator       = "operator"
	technologySourceDetected   = "detected"
	technologySourceOperator   = "operator"
	repositoryKeySourceDerived = "derived"
)

var setupKeyPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

// DiscoveredRemote is one distinct normalized locator found in local Git
// metadata. Names are local aliases; none of them has semantic priority.
type DiscoveredRemote struct {
	Locator string   `json:"locator"`
	Names   []string `json:"names"`
}

// TechnologyProposal is a detector proposal with repository-relative Evidence.
type TechnologyProposal struct {
	Key, Value, Path string
	Count            int
	Conflict         string
}

// RepositoryDiscovery is the read-only discovery result for one explicit
// Repository location. Git: absent|repository|unsafe|unreadable.
// RemoteStatus: local_only|single|ambiguous|incomplete.
type RepositoryDiscovery struct {
	Git          string
	RemoteStatus string
	Candidates   []DiscoveredRemote
	Unsupported  int
	DerivedKey   string
	Technology   []TechnologyProposal
}

type SetupRepository struct {
	// Key may be empty: the derived key is then proposed for confirmation.
	Key       string
	Path      string
	Revision  string
	Discovery *RepositoryDiscovery
}

type SetupPreference struct{ Role, Complexity, ModelProfile string }

type ProfileCandidate struct {
	Key   string `json:"key"`
	Model string `json:"model"`
}

// RuntimeCandidate is one locally configured, enabled Runtime and its
// allowlisted profiles. Candidates are machine facts, never portable defaults.
type RuntimeCandidate struct {
	ID       string             `json:"id"`
	Profiles []ProfileCandidate `json:"profiles"`
}

// SetupDocumentation is one requested documentation source. For local-file
// sources, composition observes the explicit file and supplies its binding
// (nil when the file is not an existing regular non-link file).
type SetupDocumentation struct {
	Key, Kind, RepositoryRef, Path string
	Binding                        *DocumentationBinding
}

type SetupInput struct {
	ProjectID, Slug, Name string
	Repositories          []SetupRepository
	// RepositoryRemotes maps a Repository key to an explicit locator or "none".
	RepositoryRemotes  map[string]string
	WorkItemProvider   string
	RuntimeCandidates  []RuntimeCandidate
	Runtimes           []string
	ModelProfiles      []string
	RuntimePreferences []SetupPreference
	Technology         []project.TechnologyFact
	RemoveTechnology   []string
	Documentation      []SetupDocumentation
	BusinessContext    string
	ContextSources     []string
	Glossary           []project.GlossaryEntry
}

type SetupObservation struct {
	PortableDestination string
	LocalDestination    string
	PortableRevision    string
	LocalRevision       string
	PortableEquivalent  bool
	LocalEquivalent     bool
}

type CapabilityReadiness string

const (
	CapabilityMissing     CapabilityReadiness = "missing"
	CapabilityReady       CapabilityReadiness = "ready"
	CapabilityUnsupported CapabilityReadiness = "unsupported"
)

type SetupRemotePreview struct {
	Status      string             `json:"status"`
	Locator     string             `json:"locator,omitempty"`
	Source      string             `json:"source,omitempty"`
	Candidates  []DiscoveredRemote `json:"candidates"`
	Unsupported int                `json:"unsupported"`
}

type SetupRepositoryPreview struct {
	Key           string              `json:"key"`
	KeySource     string              `json:"keySource,omitempty"`
	LocalPath     string              `json:"localPath"`
	LocalRevision string              `json:"localRevision"`
	Git           string              `json:"git,omitempty"`
	Remote        *SetupRemotePreview `json:"remote,omitempty"`
}

type SetupCapabilityPreview struct {
	Capability string              `json:"capability"`
	Provider   string              `json:"provider,omitempty"`
	Readiness  CapabilityReadiness `json:"readiness"`
}

type SetupRuntimePolicyPreview struct {
	Status        string                   `json:"status"`
	Runtimes      []string                 `json:"runtimes"`
	ModelProfiles []SetupProfilePreview    `json:"modelProfiles"`
	Preferences   []SetupPreferencePreview `json:"preferences"`
	Candidates    []RuntimeCandidate       `json:"candidates"`
}
type SetupProfilePreview struct {
	Key        string `json:"key"`
	RuntimeRef string `json:"runtimeRef"`
	Model      string `json:"model"`
}
type SetupPreferencePreview struct {
	Role            string `json:"role"`
	Complexity      string `json:"complexity"`
	ModelProfileRef string `json:"modelProfileRef"`
}

type TechnologyEvidence struct {
	Repository string `json:"repository"`
	Path       string `json:"path"`
	Count      int    `json:"count"`
}
type SetupTechnologyPreview struct {
	Key      string               `json:"key"`
	Value    string               `json:"value"`
	Source   string               `json:"source"`
	Evidence []TechnologyEvidence `json:"evidence"`
	Conflict string               `json:"conflict,omitempty"`
}

type SetupDocumentationPreview struct {
	Key           string `json:"key"`
	Kind          string `json:"kind"`
	RepositoryRef string `json:"repositoryRef,omitempty"`
	Path          string `json:"path,omitempty"`
	LocalPath     string `json:"localPath,omitempty"`
}

type SetupContextPreview struct {
	Text       string                  `json:"text,omitempty"`
	SourceRefs []string                `json:"sourceRefs"`
	Glossary   []project.GlossaryEntry `json:"glossary"`
}

type SetupPreview struct {
	ProjectID           string                      `json:"projectId"`
	Slug                string                      `json:"slug"`
	Name                string                      `json:"name"`
	SchemaVersion       int                         `json:"schemaVersion"`
	Repositories        []SetupRepositoryPreview    `json:"repositories"`
	Capability          SetupCapabilityPreview      `json:"capability"`
	RuntimePolicy       SetupRuntimePolicyPreview   `json:"runtimePolicy"`
	Technology          []SetupTechnologyPreview    `json:"technology"`
	Documentation       []SetupDocumentationPreview `json:"documentation"`
	BusinessContext     SetupContextPreview         `json:"businessContext"`
	Authority           AuthorityExpectations       `json:"authority"`
	Blockers            []Finding                   `json:"blockers"`
	PortableDestination string                      `json:"portableDestination"`
	LocalDestination    string                      `json:"localDestination"`
	PortableRevision    string                      `json:"portableRevision"`
	LocalRevision       string                      `json:"localRevision"`
	Effects             []string                    `json:"effects"`
	Digest              string                      `json:"digest"`
}

type SetupProposal struct {
	project       project.Project
	manifest      []byte
	bindings      []RepositoryBinding
	documentation []DocumentationBinding
	preview       SetupPreview
}

func setupIssue(code Code) []Issue {
	return []Issue{{Phase: ValidationPhase, Field: ProjectField, Code: code}}
}

func PrepareSetup(codec ManifestCodec, input SetupInput, observation SetupObservation) (SetupProposal, []Issue) {
	if codec == nil || input.ProjectID == "" || !boundedSetupText(input.Name, maxSetupNameBytes) || !boundedSetupKey(input.Slug) || len(input.Repositories) == 0 || len(input.Repositories) > maxSetupRepositoryCount {
		return SetupProposal{}, problem(InvalidPreview)
	}
	if !boundedSetupText(observation.PortableDestination, maxSetupDestinationBytes) || !boundedSetupText(observation.LocalDestination, maxSetupDestinationBytes) || !boundedSetupText(observation.PortableRevision, maxSetupRevisionBytes) || !boundedSetupText(observation.LocalRevision, maxSetupRevisionBytes) {
		return SetupProposal{}, problem(InvalidPreview)
	}
	preview := SetupPreview{
		ProjectID: input.ProjectID, Slug: input.Slug, Name: input.Name, SchemaVersion: 3,
		PortableDestination: observation.PortableDestination, LocalDestination: observation.LocalDestination,
		PortableRevision: observation.PortableRevision, LocalRevision: observation.LocalRevision,
		Authority: readinessAuthority, Blockers: []Finding{}, Effects: []string{},
	}
	state := project.State{SchemaVersion: 3, ID: input.ProjectID, Slug: input.Slug, Name: input.Name}
	repositories, bindings, keys, issues := setupRepositories(input, &preview)
	if len(issues) != 0 {
		return SetupProposal{}, issues
	}
	state.Repositories = project.Configured(repositories)
	if issues := setupCapability(input.WorkItemProvider, &state, &preview); len(issues) != 0 {
		return SetupProposal{}, issues
	}
	if issues := setupRuntimePolicy(input, &state, &preview); len(issues) != 0 {
		return SetupProposal{}, issues
	}
	if issues := setupTechnology(input, &state, &preview); len(issues) != 0 {
		return SetupProposal{}, issues
	}
	documentation, issues := setupDocumentation(input, keys, &state, &preview)
	if len(issues) != 0 {
		return SetupProposal{}, issues
	}
	if issues := setupContext(input, &state, &preview); len(issues) != 0 {
		return SetupProposal{}, issues
	}
	configured, domainIssues := project.New(state)
	if len(domainIssues) != 0 {
		return SetupProposal{}, setupIssue(domainIssueCode(domainIssues))
	}
	manifest, codecIssues := codec.Encode(configured)
	if len(codecIssues) != 0 {
		return SetupProposal{}, problem(InvalidPreview)
	}
	preview.Blockers = sortedFindings(preview.Blockers)
	if !observation.PortableEquivalent {
		preview.Effects = append(preview.Effects, "publish_portable_project")
	}
	if !observation.LocalEquivalent {
		preview.Effects = append(preview.Effects, "publish_local_bindings")
	}
	preview.Digest = setupPreviewDigest(preview)
	return SetupProposal{project: configured, manifest: manifest, bindings: bindings, documentation: documentation, preview: preview}, nil
}

// domainIssueCode maps a domain rejection to the bootstrap input family it
// came from without echoing rejected values.
func domainIssueCode(issues []project.Issue) Code {
	for _, issue := range issues {
		switch {
		case strings.HasPrefix(issue.Field, "repositories"):
			return InvalidRepositoryInput
		case strings.HasPrefix(issue.Field, "runtime"), strings.HasPrefix(issue.Field, "modelProfiles"):
			return InvalidRuntimePolicyInput
		case strings.HasPrefix(issue.Field, "technologyContext"):
			return InvalidTechnologyInput
		case strings.HasPrefix(issue.Field, "documentationSources"):
			return InvalidDocumentationInput
		case strings.HasPrefix(issue.Field, "businessContext"):
			return InvalidContextInput
		}
	}
	return InvalidPreview
}

func setupRepositories(input SetupInput, preview *SetupPreview) ([]project.Repository, []RepositoryBinding, map[string]bool, []Issue) {
	type candidate struct {
		repository SetupRepository
		key        string
		source     string
	}
	candidates := make([]candidate, 0, len(input.Repositories))
	operatorKeys, derivedCount := map[string]bool{}, map[string]int{}
	for _, repository := range input.Repositories {
		if !boundedSetupText(repository.Path, maxSetupPathBytes) || !boundedSetupText(repository.Revision, maxSetupRevisionBytes) {
			return nil, nil, nil, problem(InvalidPreview)
		}
		current := candidate{repository: repository, key: repository.Key}
		if repository.Key != "" {
			if !boundedSetupKey(repository.Key) || operatorKeys[repository.Key] {
				return nil, nil, nil, problem(InvalidPreview)
			}
			operatorKeys[repository.Key] = true
		} else if repository.Discovery != nil && boundedSetupKey(repository.Discovery.DerivedKey) {
			current.key, current.source = repository.Discovery.DerivedKey, repositoryKeySourceDerived
			derivedCount[current.key]++
		} else {
			preview.Blockers = append(preview.Blockers, Finding{Code: BlockerRepositoryKey})
			continue
		}
		candidates = append(candidates, current)
	}
	for _, current := range candidates {
		if current.source == repositoryKeySourceDerived && (derivedCount[current.key] > 1 || operatorKeys[current.key]) {
			preview.Blockers = append(preview.Blockers, Finding{Code: BlockerRepositoryKeyClash, Subject: current.key})
		}
	}
	keys := map[string]bool{}
	for _, current := range candidates {
		keys[current.key] = true
	}
	for key, remote := range input.RepositoryRemotes {
		if !keys[key] || remote != RemoteNone && !portableLocator(remote) {
			return nil, nil, nil, setupIssue(InvalidRepositoryInput)
		}
	}
	sort.SliceStable(candidates, func(i, j int) bool { return candidates[i].key < candidates[j].key })
	portable := make([]project.Repository, 0, len(candidates))
	bindings := make([]RepositoryBinding, 0, len(candidates))
	seen := map[string]bool{}
	for _, current := range candidates {
		if seen[current.key] {
			continue // a derived-key clash is already a blocker
		}
		seen[current.key] = true
		repository := project.Repository{Key: current.key}
		view := SetupRepositoryPreview{Key: current.key, KeySource: current.source, LocalPath: current.repository.Path, LocalRevision: current.repository.Revision}
		if remote, ok := setupRemote(current.key, current.repository.Discovery, input.RepositoryRemotes, &view, preview); ok {
			repository.Remote = project.Configured(remote)
		}
		portable = append(portable, repository)
		bindings = append(bindings, RepositoryBinding{RepositoryKey: current.key, ExplicitPath: current.repository.Path, CanonicalIdentity: current.repository.Revision, Observation: Observation{Availability: Unverified, Basis: NotChecked}})
		preview.Repositories = append(preview.Repositories, view)
	}
	return portable, bindings, keys, nil
}

// setupRemote applies the ambiguity rules: an explicit choice always wins;
// otherwise only one distinct discovered locator may be proposed. No remote
// name, including origin, has priority.
func setupRemote(key string, discovery *RepositoryDiscovery, choices map[string]string, view *SetupRepositoryPreview, preview *SetupPreview) (string, bool) {
	remote := &SetupRemotePreview{Status: "local_only", Candidates: []DiscoveredRemote{}}
	if discovery != nil {
		view.Git = discovery.Git
		remote.Status, remote.Unsupported = discovery.RemoteStatus, discovery.Unsupported
		remote.Candidates = append(remote.Candidates, discovery.Candidates...)
		sort.Slice(remote.Candidates, func(i, j int) bool { return remote.Candidates[i].Locator < remote.Candidates[j].Locator })
		if discovery.Git == "unsafe" || discovery.Git == "unreadable" {
			preview.Blockers = append(preview.Blockers, Finding{Code: BlockerRepositoryUnsafe, Subject: key})
		}
	}
	view.Remote = remote
	if choice, ok := choices[key]; ok {
		remote.Source = remoteSourceOperator
		if choice == RemoteNone {
			return "", false
		}
		remote.Locator, _ = project.NormalizeLocator(choice)
		return remote.Locator, true
	}
	switch remote.Status {
	case "single":
		if len(remote.Candidates) == 1 {
			remote.Locator, remote.Source = remote.Candidates[0].Locator, remoteSourceDiscovered
			return remote.Locator, true
		}
		preview.Blockers = append(preview.Blockers, Finding{Code: BlockerRemoteAmbiguous, Subject: key})
	case "ambiguous", "incomplete":
		preview.Blockers = append(preview.Blockers, Finding{Code: BlockerRemoteAmbiguous, Subject: key})
	}
	return "", false
}

func portableLocator(value string) bool {
	locator, issues := project.NormalizeLocator(value)
	return len(issues) == 0 && !strings.ContainsAny(locator, "?#") && len(value) <= maxSetupPathBytes
}

func setupCapability(provider string, state *project.State, preview *SetupPreview) []Issue {
	if provider == "" || provider == RemoteNone {
		state.Providers = project.Unconfigured[[]project.Provider]()
		state.Integrations = project.Unconfigured[[]project.Integration]()
	} else {
		if !boundedSetupKey(provider) {
			return problem(InvalidPreview)
		}
		state.Providers = project.Configured([]project.Provider{{Key: workItemsKey, ID: provider}})
		state.Integrations = project.Configured([]project.Integration{{Key: workItemsKey, ProviderRef: project.Configured(workItemsKey), Capabilities: project.Configured([]string{WorkItemCapability})}})
	}
	resolution := ResolveCapability(*state, WorkItemCapability, SupportedProviders)
	preview.Capability = SetupCapabilityPreview{Capability: WorkItemCapability, Provider: resolution.Provider, Readiness: resolution.Readiness}
	return nil
}

func setupRuntimePolicy(input SetupInput, state *project.State, preview *SetupPreview) []Issue {
	policy := SetupRuntimePolicyPreview{Status: "absent", Runtimes: []string{}, ModelProfiles: []SetupProfilePreview{}, Preferences: []SetupPreferencePreview{}, Candidates: []RuntimeCandidate{}}
	profiles := map[string]SetupProfilePreview{}
	runtimes := map[string]bool{}
	for _, candidate := range input.RuntimeCandidates {
		if !runtimeToken(candidate.ID) || runtimes[candidate.ID] {
			return setupIssue(InvalidRuntimePolicyInput)
		}
		runtimes[candidate.ID] = true
		copied := RuntimeCandidate{ID: candidate.ID, Profiles: append([]ProfileCandidate{}, candidate.Profiles...)}
		sort.Slice(copied.Profiles, func(i, j int) bool { return copied.Profiles[i].Key < copied.Profiles[j].Key })
		for _, profile := range copied.Profiles {
			if _, duplicate := profiles[profile.Key]; duplicate || profile.Key == "" || profile.Model == "" {
				return setupIssue(InvalidRuntimePolicyInput)
			}
			profiles[profile.Key] = SetupProfilePreview{Key: profile.Key, RuntimeRef: candidate.ID, Model: profile.Model}
		}
		policy.Candidates = append(policy.Candidates, copied)
	}
	sort.Slice(policy.Candidates, func(i, j int) bool { return policy.Candidates[i].ID < policy.Candidates[j].ID })
	if len(input.Runtimes) > maxSetupRuntimeSelections || len(input.ModelProfiles) > maxSetupProfiles || len(input.RuntimePreferences) > maxSetupPreferences {
		return setupIssue(InvalidRuntimePolicyInput)
	}
	if len(input.Runtimes) == 0 {
		if len(input.ModelProfiles) != 0 || len(input.RuntimePreferences) != 0 {
			return setupIssue(InvalidRuntimePolicyInput)
		}
		preview.RuntimePolicy = policy
		return nil
	}
	chosen := map[string]bool{}
	declared := make([]project.Runtime, 0, len(input.Runtimes))
	for _, id := range input.Runtimes {
		// Only locally known candidates may be allowed; nothing is defaulted.
		if !runtimes[id] || chosen[id] {
			return setupIssue(InvalidRuntimePolicyInput)
		}
		chosen[id] = true
		declared = append(declared, project.Runtime{ID: id})
		policy.Runtimes = append(policy.Runtimes, id)
	}
	selected := map[string]bool{}
	portableProfiles := make([]project.ModelProfile, 0, len(input.ModelProfiles))
	for _, key := range input.ModelProfiles {
		profile, ok := profiles[key]
		if !ok || selected[key] || !chosen[profile.RuntimeRef] {
			return setupIssue(InvalidRuntimePolicyInput)
		}
		selected[key] = true
		portableProfiles = append(portableProfiles, project.ModelProfile{Key: key, RuntimeRef: project.Configured(profile.RuntimeRef), Model: project.Configured(profile.Model)})
		policy.ModelProfiles = append(policy.ModelProfiles, profile)
	}
	preferences := make([]project.RuntimePreference, 0, len(input.RuntimePreferences))
	for _, preference := range input.RuntimePreferences {
		if !selected[preference.ModelProfile] {
			return setupIssue(InvalidRuntimePolicyInput)
		}
		preferences = append(preferences, project.RuntimePreference{Role: preference.Role, Complexity: preference.Complexity, ModelProfileRef: preference.ModelProfile})
		policy.Preferences = append(policy.Preferences, SetupPreferencePreview{Role: preference.Role, Complexity: preference.Complexity, ModelProfileRef: preference.ModelProfile})
	}
	sort.Strings(policy.Runtimes)
	sort.Slice(policy.ModelProfiles, func(i, j int) bool { return policy.ModelProfiles[i].Key < policy.ModelProfiles[j].Key })
	sort.Slice(policy.Preferences, func(i, j int) bool {
		if policy.Preferences[i].Role != policy.Preferences[j].Role {
			return policy.Preferences[i].Role < policy.Preferences[j].Role
		}
		return policy.Preferences[i].Complexity < policy.Preferences[j].Complexity
	})
	state.Runtimes = project.Configured(declared)
	state.ModelProfiles = project.Configured(portableProfiles)
	if len(preferences) != 0 {
		state.RuntimePreferences = project.Configured(preferences)
	}
	policy.Status = "configured"
	preview.RuntimePolicy = policy
	return nil
}

func runtimeToken(value string) bool {
	if value == "" || len(value) > 256 {
		return false
	}
	return strings.IndexFunc(value, func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-' || r == '_' || r == '.')
	}) < 0
}

func setupTechnology(input SetupInput, state *project.State, preview *SetupPreview) []Issue {
	facts := map[string]*SetupTechnologyPreview{}
	for _, repository := range input.Repositories {
		if repository.Discovery == nil {
			continue
		}
		key := repository.Key
		if key == "" {
			key = repository.Discovery.DerivedKey
		}
		for _, proposal := range repository.Discovery.Technology {
			if !project.ValidContextKey(proposal.Key) || !project.ContextValue(proposal.Value, project.MaxTechnologyValueBytes, false) || !project.RepositoryRelativePath(proposal.Path) {
				return setupIssue(InvalidTechnologyInput)
			}
			fact := facts[proposal.Key]
			if fact == nil {
				fact = &SetupTechnologyPreview{Key: proposal.Key, Value: proposal.Value, Source: technologySourceDetected, Evidence: []TechnologyEvidence{}}
				facts[proposal.Key] = fact
			}
			fact.Evidence = append(fact.Evidence, TechnologyEvidence{Repository: key, Path: proposal.Path, Count: proposal.Count})
			if proposal.Conflict != "" {
				fact.Conflict = proposal.Conflict
			}
		}
	}
	for _, key := range input.RemoveTechnology {
		if facts[key] == nil {
			return setupIssue(InvalidTechnologyInput)
		}
		delete(facts, key)
	}
	operator := map[string]bool{}
	for _, fact := range input.Technology {
		if !project.ValidContextKey(fact.Key) || !project.ContextValue(fact.Value, project.MaxTechnologyValueBytes, false) || operator[fact.Key] {
			return setupIssue(InvalidTechnologyInput)
		}
		operator[fact.Key] = true
		facts[fact.Key] = &SetupTechnologyPreview{Key: fact.Key, Value: fact.Value, Source: technologySourceOperator, Evidence: []TechnologyEvidence{}}
	}
	preview.Technology = []SetupTechnologyPreview{}
	portable := make([]project.TechnologyFact, 0, len(facts))
	for _, fact := range facts {
		sort.Slice(fact.Evidence, func(i, j int) bool { return fact.Evidence[i].Repository < fact.Evidence[j].Repository })
		preview.Technology = append(preview.Technology, *fact)
		portable = append(portable, project.TechnologyFact{Key: fact.Key, Value: fact.Value})
	}
	sort.Slice(preview.Technology, func(i, j int) bool { return preview.Technology[i].Key < preview.Technology[j].Key })
	if len(portable) > project.MaxTechnologyFacts {
		return setupIssue(InvalidTechnologyInput)
	}
	if len(portable) != 0 {
		state.TechnologyContext = project.Configured(portable)
	}
	return nil
}

func setupDocumentation(input SetupInput, keys map[string]bool, state *project.State, preview *SetupPreview) ([]DocumentationBinding, []Issue) {
	preview.Documentation = []SetupDocumentationPreview{}
	if len(input.Documentation) > project.MaxDocumentationSources {
		return nil, setupIssue(InvalidDocumentationInput)
	}
	sources := make([]project.DocumentationSource, 0, len(input.Documentation))
	bindings := []DocumentationBinding{}
	seen := map[string]bool{}
	for _, requested := range input.Documentation {
		if !project.ValidContextKey(requested.Key) || seen[requested.Key] {
			return nil, setupIssue(InvalidDocumentationInput)
		}
		seen[requested.Key] = true
		view := SetupDocumentationPreview{Key: requested.Key, Kind: requested.Kind}
		switch requested.Kind {
		case project.RepositorySource:
			if !keys[requested.RepositoryRef] || !project.RepositoryRelativePath(requested.Path) || requested.Binding != nil {
				return nil, setupIssue(InvalidDocumentationInput)
			}
			sources = append(sources, project.DocumentationSource{Key: requested.Key, Kind: requested.Kind, RepositoryRef: project.Configured(requested.RepositoryRef), Path: project.Configured(requested.Path)})
			view.RepositoryRef, view.Path = requested.RepositoryRef, requested.Path
		case project.LocalFileSource:
			// The absolute path stays machine-local; the portable source is the key.
			if requested.Binding == nil || requested.Binding.SourceKey != requested.Key || !boundedSetupText(requested.Binding.ExplicitPath, maxSetupPathBytes) {
				return nil, setupIssue(InvalidDocumentationInput)
			}
			sources = append(sources, project.DocumentationSource{Key: requested.Key, Kind: requested.Kind})
			bindings = append(bindings, *requested.Binding)
			view.LocalPath = requested.Binding.ExplicitPath
		default:
			return nil, setupIssue(InvalidDocumentationInput)
		}
		preview.Documentation = append(preview.Documentation, view)
	}
	sort.Slice(preview.Documentation, func(i, j int) bool { return preview.Documentation[i].Key < preview.Documentation[j].Key })
	sort.Slice(bindings, func(i, j int) bool { return bindings[i].SourceKey < bindings[j].SourceKey })
	if len(sources) != 0 {
		state.DocumentationSources = project.Configured(sources)
	}
	return bindings, nil
}

func setupContext(input SetupInput, state *project.State, preview *SetupPreview) []Issue {
	preview.BusinessContext = SetupContextPreview{SourceRefs: []string{}, Glossary: []project.GlossaryEntry{}}
	if input.BusinessContext == "" && len(input.ContextSources) == 0 && len(input.Glossary) == 0 {
		return nil
	}
	// Validate every prose value before adding any context to preview/state.
	for _, entry := range input.Glossary {
		if !project.PortableContextValue(entry.Term, project.MaxGlossaryTermBytes, false) || !project.PortableContextValue(entry.Definition, project.MaxGlossaryDefinitionLen, true) {
			return setupIssue(InvalidContextInput)
		}
	}
	context := project.BusinessContext{}
	if input.BusinessContext != "" {
		if !project.PortableContextValue(input.BusinessContext, maxBusinessContextBytes, true) {
			return setupIssue(InvalidContextInput)
		}
		context.Text = project.Configured(input.BusinessContext)
		preview.BusinessContext.Text = input.BusinessContext
	}
	if len(input.ContextSources) != 0 {
		refs := append([]string(nil), input.ContextSources...)
		sort.Strings(refs)
		context.SourceRefs = project.Configured(refs)
		preview.BusinessContext.SourceRefs = refs
	}
	if len(input.Glossary) != 0 {
		glossary := append([]project.GlossaryEntry(nil), input.Glossary...)
		sort.Slice(glossary, func(i, j int) bool { return glossary[i].Key < glossary[j].Key })
		context.Glossary = project.Configured(glossary)
		preview.BusinessContext.Glossary = glossary
	}
	state.BusinessContext = project.Configured(context)
	return nil
}

func boundedSetupKey(value string) bool {
	return len(value) <= maxSetupKeyBytes && setupKeyPattern.MatchString(value)
}

func boundedSetupText(value string, maximum int) bool {
	if value == "" || len(value) > maximum || !utf8.ValidString(value) || strings.TrimSpace(value) != value {
		return false
	}
	for _, current := range value {
		if unicode.IsControl(current) {
			return false
		}
	}
	return true
}

func setupPreviewDigest(preview SetupPreview) string {
	preview.Digest = ""
	wire, _ := json.Marshal(preview)
	digest := sha256.Sum256(wire)
	return hex.EncodeToString(digest[:])
}

func (p SetupProposal) Project() project.Project { return p.project }
func (p SetupProposal) Manifest() []byte         { return append([]byte(nil), p.manifest...) }
func (p SetupProposal) Bindings() []RepositoryBinding {
	return append([]RepositoryBinding(nil), p.bindings...)
}
func (p SetupProposal) DocumentationBindings() []DocumentationBinding {
	return append([]DocumentationBinding(nil), p.documentation...)
}
func (p SetupProposal) Preview() SetupPreview { return p.preview }
func (p SetupProposal) MatchesDigest(value string) bool {
	return value != "" && value == p.preview.Digest
}

// Publishable reports whether the reviewed proposal is free of bootstrap
// blockers. Missing optional intent is readiness information, not a blocker.
func (p SetupProposal) Publishable() bool { return p.Valid() && len(p.preview.Blockers) == 0 }
func (p SetupProposal) Valid() bool {
	return p.project.Equivalent(p.project) && p.preview.Digest == setupPreviewDigest(p.preview)
}
