package main

import (
	"context"
	"path/filepath"
	"time"

	"github.com/rgomids/axiom/internal/cli"
	"github.com/rgomids/axiom/internal/completion"
	"github.com/rgomids/axiom/internal/local"
	"github.com/rgomids/axiom/internal/projectapp"
	"github.com/rgomids/axiom/internal/projectdiscovery"
	"github.com/rgomids/axiom/internal/runtimeapplication"
	"github.com/rgomids/axiom/internal/runtimeprofile"
)

// Issue #231 composition: the canonical readiness evaluator, the pre-effect
// gates that consume its operation projections, and bootstrap discovery.

// readiness composes the single read-only evaluator from local adapters and
// the #140 Runtime availability projection over the same policy source that
// workflow start uses.
func (s lifecycleService) readiness() projectapp.ProjectReadiness {
	return projectapp.ProjectReadiness{
		Projects:      local.ReadinessProjects{Installation: s.installation, Portable: s.portable},
		Repositories:  local.RepositoryProbe{},
		Documentation: local.DocumentationResolver{},
		Runtime:       runtimeReadiness{s},
		Catalog:       projectapp.SupportedProviders,
	}
}

type runtimeReadiness struct{ service lifecycleService }

func (r runtimeReadiness) Availability(ctx context.Context, projectID string) string {
	source := r.service.RuntimePolicySource
	if source == nil {
		source = runtimePolicySource{installation: r.service.installation, portable: r.service.portable, stateRoot: r.service.stateRoot, runtimes: r.service.runtimeObservationSources()}
	}
	return runtimeapplication.New(source).Availability(ctx, projectID)
}

// preflight returns a failure before any effect when a structurally valid
// Project cannot satisfy the operation. Structural failures (not installed,
// recovery, invalid or stale installation) return nil so the operation's own
// resolution reports its established category; it applies the same Select
// and source checks and therefore also fails before effects.
func (s lifecycleService) preflight(ctx context.Context, selector string, operation projectapp.ReadinessOperation) *cli.Result {
	if selector == "" {
		return nil
	}
	report, projection := s.readiness().EvaluateOperation(ctx, selector, operation)
	if report.Structure != "valid" || projection.Ready() {
		return nil
	}
	message := "Project is not ready for Work Item operations"
	if operation == projectapp.OperationExecution {
		message = "Project is not ready for Execution"
	}
	result := canonicalCompletion(completion.Facts{ValidationFailed: true}, message, []string{"project:" + report.ProjectID}, "Resolve the reported readiness blockers, then retry; run project validate for the full report", s.provenance)
	result.Category = projection.Blockers[0].Code
	result.Preflight = &projection
	return &result
}

func runtimeCandidates(ctx context.Context, stateRoot string) []projectapp.RuntimeCandidate {
	store, err := local.NewRuntimeProfileStore(stateRoot)
	if err != nil {
		return nil
	}
	cfg, err := store.Load(ctx)
	if err != nil {
		return nil
	}
	return candidatesFrom(cfg)
}

// candidatesFrom lists enabled Runtimes with their allowlisted profiles. It
// is a disclosure of machine facts; nothing here selects a Runtime.
func candidatesFrom(cfg runtimeprofile.Configuration) []projectapp.RuntimeCandidate {
	profiles := map[string]runtimeprofile.ModelProfile{}
	for _, profile := range cfg.ModelProfiles {
		profiles[profile.ID] = profile
	}
	candidates := []projectapp.RuntimeCandidate{}
	for _, runtime := range cfg.Runtimes {
		if !runtime.Enabled {
			continue
		}
		candidate := projectapp.RuntimeCandidate{ID: runtime.ID, Profiles: []projectapp.ProfileCandidate{}}
		for _, id := range runtime.AllowlistedProfileIDs {
			if profile, ok := profiles[id]; ok && profile.RuntimeID == runtime.ID {
				candidate.Profiles = append(candidate.Profiles, projectapp.ProfileCandidate{Key: profile.ID, Model: profile.Model})
			}
		}
		candidates = append(candidates, candidate)
	}
	return candidates
}

// discoverRepository runs the read-only discovery adapters for one explicit
// Repository location. It never consults the working directory.
func discoverRepository(path string) *projectapp.RepositoryDiscovery {
	discovery := &projectapp.RepositoryDiscovery{}
	if repository, err := projectdiscovery.InspectRepository(path); err == nil {
		discovery.Git, discovery.RemoteStatus, discovery.Unsupported = string(repository.Git), string(repository.Remote), repository.Unsupported
		for _, candidate := range repository.Candidates {
			discovery.Candidates = append(discovery.Candidates, projectapp.DiscoveredRemote{Locator: candidate.Locator, Names: candidate.Names})
		}
	}
	if facts, err := projectdiscovery.DetectTechnology(path); err == nil {
		for _, fact := range facts {
			discovery.Technology = append(discovery.Technology, projectapp.TechnologyProposal{Key: fact.Key, Value: fact.Value, Path: fact.Path, Count: fact.Count, Conflict: fact.Conflict})
		}
	}
	discovery.DerivedKey, _ = projectdiscovery.DeriveRepositoryKey(path)
	return discovery
}

func documentationInputs(input cli.ConfigureInput) []projectapp.SetupDocumentation {
	documentation := make([]projectapp.SetupDocumentation, 0, len(input.Documentation))
	for _, requested := range input.Documentation {
		current := projectapp.SetupDocumentation{Key: requested.Key, Kind: requested.Kind, RepositoryRef: requested.Repository, Path: requested.Path}
		if requested.Kind == "local-file" {
			current.Path = ""
			if binding, ok := local.ObserveDocumentationFile(requested.Key, requested.Path, func() projectapp.Observation {
				return projectapp.Observation{Availability: projectapp.Available, Basis: projectapp.PresentMetadata, ObservedAt: time.Now().UTC().Truncate(time.Second)}
			}); ok {
				current.Binding = &binding
			}
		}
		documentation = append(documentation, current)
	}
	return documentation
}

func setupIssueText(issues []projectapp.Issue) (string, string) {
	switch issues[0].Code {
	case projectapp.InvalidRepositoryInput:
		return "Project setup Repository input is invalid", "Provide existing Repository keys and portable remote locators (no local paths or credentials), or none"
	case projectapp.InvalidRuntimePolicyInput:
		return "Project setup Runtime policy input is invalid", "Choose only locally configured Runtimes and profiles; run runtime profile validate to inspect local configuration"
	case projectapp.InvalidTechnologyInput:
		return "Project setup technology input is invalid", "Use lowercase technology keys with bounded non-path values, and remove only proposed facts"
	case projectapp.InvalidDocumentationInput:
		return "Project setup documentation input is invalid", "Reference a declared Repository with a contained relative path, or an existing absolute regular file"
	case projectapp.InvalidContextInput:
		return "Project setup business context input is invalid", "Keep context bounded, reference declared documentation sources, and use unique glossary keys"
	}
	return "Project setup input is invalid", "Correct Project identity, repositories, or capability declaration"
}

func cleanPath(path string) string {
	if path == "" {
		return ""
	}
	return filepath.Clean(path)
}
