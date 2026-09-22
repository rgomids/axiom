package projectapp_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/manifest"
	"github.com/rgomids/axiom/internal/projectapp"
)

func TestSetupProposalNormalizesGuidedAndCompleteFacts(t *testing.T) {
	observation := projectapp.SetupObservation{PortableDestination: "/portable/sample", LocalDestination: "/state/projects/id", PortableRevision: "absent", LocalRevision: "absent"}
	first, issues := projectapp.PrepareSetup(manifest.Codec{}, projectapp.SetupInput{
		ProjectID: "123e4567-e89b-42d3-a456-426614174000", Slug: "sample", Name: "Sample",
		Repositories: []projectapp.SetupRepository{{Key: "web", Path: "/work/web", Revision: "web-revision"}, {Key: "api", Path: "/work/api", Revision: "api-revision"}}, WorkItemProvider: "github",
	}, observation)
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	second, issues := projectapp.PrepareSetup(manifest.Codec{}, projectapp.SetupInput{
		ProjectID: "123e4567-e89b-42d3-a456-426614174000", Slug: "sample", Name: "Sample",
		Repositories: []projectapp.SetupRepository{{Key: "api", Path: "/work/api", Revision: "api-revision"}, {Key: "web", Path: "/work/web", Revision: "web-revision"}}, WorkItemProvider: "github",
	}, observation)
	if len(issues) != 0 || !reflect.DeepEqual(first.Preview(), second.Preview()) || string(first.Manifest()) != string(second.Manifest()) {
		t.Fatal("same facts did not normalize to one proposal", issues)
	}
	state := first.Project().State()
	providers, ok := state.Providers.Value()
	if !ok || len(providers) != 1 || providers[0].ID != "github" {
		t.Fatal("provider declaration missing")
	}
	for _, repository := range first.Bindings() {
		if repository.ExplicitPath == "" || repository.CanonicalIdentity == "" {
			t.Fatal("local binding missing")
		}
	}
	portableRepositories, _ := state.Repositories.Value()
	for _, repository := range portableRepositories {
		if _, present := repository.Remote.Value(); present {
			t.Fatal("local path leaked into portable repository")
		}
	}
}

func TestSetupProposalCapabilityAndEffectsAreExplicit(t *testing.T) {
	base := projectapp.SetupInput{ProjectID: "123e4567-e89b-42d3-a456-426614174000", Slug: "sample", Name: "Sample", Repositories: []projectapp.SetupRepository{{Key: "repo", Path: "/work/repo", Revision: "repo-revision"}}}
	observation := projectapp.SetupObservation{PortableDestination: "/portable/sample", LocalDestination: "/state/projects/id", PortableRevision: "absent", LocalRevision: "absent"}
	missing, issues := projectapp.PrepareSetup(manifest.Codec{}, base, observation)
	if len(issues) != 0 || missing.Preview().Capability.Readiness != projectapp.CapabilityMissing {
		t.Fatal("missing capability not explicit", issues)
	}
	base.WorkItemProvider = "linear"
	unsupported, issues := projectapp.PrepareSetup(manifest.Codec{}, base, observation)
	if len(issues) != 0 || unsupported.Preview().Capability.Readiness != projectapp.CapabilityUnsupported {
		t.Fatal("unsupported capability silently changed", issues)
	}
	base.WorkItemProvider = "github"
	ready, issues := projectapp.PrepareSetup(manifest.Codec{}, base, observation)
	if len(issues) != 0 || ready.Preview().Capability.Readiness != projectapp.CapabilityReady || !reflect.DeepEqual(ready.Preview().Effects, []string{"publish_portable_project", "publish_local_bindings"}) {
		t.Fatal("ready/effects mismatch", issues, ready.Preview())
	}
	if !ready.MatchesDigest(ready.Preview().Digest) || ready.MatchesDigest("stale") || !ready.Valid() {
		t.Fatal("preview digest binding failed")
	}
}

func TestSetupProposalRejectsInvalidOrDuplicateInputs(t *testing.T) {
	observation := projectapp.SetupObservation{PortableDestination: "/portable/sample", LocalDestination: "/state/projects/id", PortableRevision: "absent", LocalRevision: "absent"}
	cases := []projectapp.SetupInput{
		{},
		{ProjectID: "123e4567-e89b-42d3-a456-426614174000", Slug: "sample", Name: "Sample", Repositories: []projectapp.SetupRepository{{Key: "repo", Path: "/a", Revision: "a"}, {Key: "repo", Path: "/b", Revision: "b"}}},
		{ProjectID: "123e4567-e89b-42d3-a456-426614174000", Slug: "sample", Name: "Sample", Repositories: []projectapp.SetupRepository{{Key: "repo", Path: "/a", Revision: "a"}}, WorkItemProvider: "$(unsafe)"},
		{ProjectID: "123e4567-e89b-42d3-a456-426614174000", Slug: "sample", Name: "unsafe\nname", Repositories: []projectapp.SetupRepository{{Key: "repo", Path: "/a", Revision: "a"}}},
		{ProjectID: "123e4567-e89b-42d3-a456-426614174000", Slug: "sample", Name: strings.Repeat("x", 257), Repositories: []projectapp.SetupRepository{{Key: "repo", Path: "/a", Revision: "a"}}},
		{ProjectID: "123e4567-e89b-42d3-a456-426614174000", Slug: "sample", Name: "Sample", Repositories: []projectapp.SetupRepository{{Key: "repo", Path: "/unsafe\npath", Revision: "a"}}},
	}
	for _, input := range cases {
		if proposal, issues := projectapp.PrepareSetup(manifest.Codec{}, input, observation); len(issues) == 0 || proposal.Valid() {
			t.Fatal("invalid setup accepted")
		}
	}
	many := make([]projectapp.SetupRepository, 33)
	for index := range many {
		many[index] = projectapp.SetupRepository{Key: "repo-" + string(rune('a'+index%26)) + string(rune('a'+index/26)), Path: "/work/repository", Revision: "revision"}
	}
	input := projectapp.SetupInput{ProjectID: "123e4567-e89b-42d3-a456-426614174000", Slug: "sample", Name: "Sample", Repositories: many}
	if proposal, issues := projectapp.PrepareSetup(manifest.Codec{}, input, observation); len(issues) == 0 || proposal.Valid() {
		t.Fatal("unbounded setup accepted")
	}
}
