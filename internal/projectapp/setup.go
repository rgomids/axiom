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

const WorkItemCapability = "work-item"

const (
	maxSetupKeyBytes         = 63
	maxSetupNameBytes        = 256
	maxSetupPathBytes        = 4096
	maxSetupRepositoryCount  = 32
	maxSetupDestinationBytes = 4096
	maxSetupRevisionBytes    = 128
)

var setupKeyPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

type SetupRepository struct {
	Key      string
	Path     string
	Revision string
}

type SetupInput struct {
	ProjectID, Slug, Name string
	Repositories          []SetupRepository
	WorkItemProvider      string
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

type SetupRepositoryPreview struct {
	Key           string `json:"key"`
	LocalPath     string `json:"localPath"`
	LocalRevision string `json:"localRevision"`
}

type SetupCapabilityPreview struct {
	Capability string              `json:"capability"`
	Provider   string              `json:"provider,omitempty"`
	Readiness  CapabilityReadiness `json:"readiness"`
}

type SetupPreview struct {
	ProjectID           string                   `json:"projectId"`
	Slug                string                   `json:"slug"`
	Name                string                   `json:"name"`
	Repositories        []SetupRepositoryPreview `json:"repositories"`
	Capability          SetupCapabilityPreview   `json:"capability"`
	PortableDestination string                   `json:"portableDestination"`
	LocalDestination    string                   `json:"localDestination"`
	PortableRevision    string                   `json:"portableRevision"`
	LocalRevision       string                   `json:"localRevision"`
	Effects             []string                 `json:"effects"`
	Digest              string                   `json:"digest"`
}

type SetupProposal struct {
	project  project.Project
	manifest []byte
	bindings []RepositoryBinding
	preview  SetupPreview
}

func PrepareSetup(codec ManifestCodec, input SetupInput, observation SetupObservation) (SetupProposal, []Issue) {
	if codec == nil || input.ProjectID == "" || !boundedSetupText(input.Name, maxSetupNameBytes) || !boundedSetupKey(input.Slug) || len(input.Repositories) == 0 || len(input.Repositories) > maxSetupRepositoryCount {
		return SetupProposal{}, problem(InvalidPreview)
	}
	if !boundedSetupText(observation.PortableDestination, maxSetupDestinationBytes) || !boundedSetupText(observation.LocalDestination, maxSetupDestinationBytes) || !boundedSetupText(observation.PortableRevision, maxSetupRevisionBytes) || !boundedSetupText(observation.LocalRevision, maxSetupRevisionBytes) {
		return SetupProposal{}, problem(InvalidPreview)
	}
	repositories := append([]SetupRepository(nil), input.Repositories...)
	sort.Slice(repositories, func(i, j int) bool { return repositories[i].Key < repositories[j].Key })
	portableRepositories := make([]project.Repository, 0, len(repositories))
	bindings := make([]RepositoryBinding, 0, len(repositories))
	seen := map[string]bool{}
	for _, repository := range repositories {
		if !boundedSetupKey(repository.Key) || !boundedSetupText(repository.Path, maxSetupPathBytes) || !boundedSetupText(repository.Revision, maxSetupRevisionBytes) || seen[repository.Key] {
			return SetupProposal{}, problem(InvalidPreview)
		}
		seen[repository.Key] = true
		portableRepositories = append(portableRepositories, project.Repository{Key: repository.Key})
		bindings = append(bindings, RepositoryBinding{RepositoryKey: repository.Key, ExplicitPath: repository.Path, CanonicalIdentity: repository.Revision, Observation: Observation{Availability: Unverified, Basis: NotChecked}})
	}
	state := project.State{SchemaVersion: 1, ID: input.ProjectID, Slug: input.Slug, Name: input.Name, Repositories: project.Configured(portableRepositories)}
	capability := SetupCapabilityPreview{Capability: WorkItemCapability, Readiness: CapabilityMissing}
	if input.WorkItemProvider != "" {
		if !boundedSetupKey(input.WorkItemProvider) {
			return SetupProposal{}, problem(InvalidPreview)
		}
		state.Providers = project.Configured([]project.Provider{{Key: "work-items", ID: input.WorkItemProvider}})
		state.Integrations = project.Configured([]project.Integration{{Key: "work-items", ProviderRef: project.Configured("work-items"), Capabilities: project.Configured([]string{WorkItemCapability})}})
		capability.Provider = input.WorkItemProvider
		capability.Readiness = CapabilityUnsupported
		if input.WorkItemProvider == "github" {
			capability.Readiness = CapabilityReady
		}
	} else {
		state.Providers = project.Unconfigured[[]project.Provider]()
		state.Integrations = project.Unconfigured[[]project.Integration]()
	}
	configured, domainIssues := project.New(state)
	if len(domainIssues) != 0 {
		return SetupProposal{}, problem(InvalidPreview)
	}
	manifest, codecIssues := codec.Encode(configured)
	if len(codecIssues) != 0 {
		return SetupProposal{}, problem(InvalidPreview)
	}
	preview := SetupPreview{
		ProjectID: input.ProjectID, Slug: input.Slug, Name: input.Name,
		Repositories: make([]SetupRepositoryPreview, 0, len(repositories)),
		Capability:   capability, PortableDestination: observation.PortableDestination,
		LocalDestination: observation.LocalDestination, PortableRevision: observation.PortableRevision,
		LocalRevision: observation.LocalRevision, Effects: []string{},
	}
	for _, repository := range repositories {
		preview.Repositories = append(preview.Repositories, SetupRepositoryPreview{Key: repository.Key, LocalPath: repository.Path, LocalRevision: repository.Revision})
	}
	if !observation.PortableEquivalent {
		preview.Effects = append(preview.Effects, "publish_portable_project")
	}
	if !observation.LocalEquivalent {
		preview.Effects = append(preview.Effects, "publish_local_bindings")
	}
	preview.Digest = setupPreviewDigest(preview)
	return SetupProposal{project: configured, manifest: manifest, bindings: bindings, preview: preview}, nil
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
func (p SetupProposal) Preview() SetupPreview { return p.preview }
func (p SetupProposal) MatchesDigest(value string) bool {
	return value != "" && value == p.preview.Digest
}
func (p SetupProposal) Valid() bool {
	return p.project.Equivalent(p.project) && p.preview.Digest == setupPreviewDigest(p.preview)
}
