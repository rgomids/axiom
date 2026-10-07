package projectapp_test

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/rgomids/axiom/internal/manifest"
	"github.com/rgomids/axiom/internal/project"
	"github.com/rgomids/axiom/internal/projectapp"
)

const bootstrapID = "123e4567-e89b-42d3-a456-426614174000"

var bootstrapObservation = projectapp.SetupObservation{PortableDestination: "/portable/sample", LocalDestination: "/state/projects/id", PortableRevision: "absent", LocalRevision: "absent"}

func discovered(status string, locators ...string) *projectapp.RepositoryDiscovery {
	d := &projectapp.RepositoryDiscovery{Git: "repository", RemoteStatus: status}
	for i, locator := range locators {
		d.Candidates = append(d.Candidates, projectapp.DiscoveredRemote{Locator: locator, Names: []string{[]string{"upstream", "origin", "fork"}[i%3]}})
	}
	return d
}

func bootstrapInput(repositories ...projectapp.SetupRepository) projectapp.SetupInput {
	return projectapp.SetupInput{ProjectID: bootstrapID, Slug: "sample", Name: "Sample", Repositories: repositories}
}

func prepare(t *testing.T, input projectapp.SetupInput) projectapp.SetupProposal {
	t.Helper()
	proposal, issues := projectapp.PrepareSetup(manifest.Codec{}, input, bootstrapObservation)
	if len(issues) != 0 {
		t.Fatalf("unexpected issues %v", issues)
	}
	return proposal
}

func rejectSetup(t *testing.T, input projectapp.SetupInput, code projectapp.Code) {
	t.Helper()
	proposal, issues := projectapp.PrepareSetup(manifest.Codec{}, input, bootstrapObservation)
	if len(issues) == 0 || proposal.Valid() || issues[0].Code != code {
		t.Fatalf("want %s, got %v", code, issues)
	}
}

func blockers(p projectapp.SetupProposal) string {
	codes := []string{}
	for _, blocker := range p.Preview().Blockers {
		codes = append(codes, blocker.Code+":"+blocker.Subject)
	}
	return strings.Join(codes, ",")
}

func TestBootstrapMinimalIncompleteProjectIsV3AndPublishable(t *testing.T) {
	p := prepare(t, bootstrapInput(projectapp.SetupRepository{Key: "core", Path: "/work/core", Revision: "fs:core"}))
	if p.Project().State().SchemaVersion != 3 || !p.Publishable() {
		t.Fatalf("minimal bootstrap not publishable v3: %s", blockers(p))
	}
	preview := p.Preview()
	if preview.RuntimePolicy.Status != "absent" || preview.Capability.Readiness != projectapp.CapabilityMissing {
		t.Fatal("missing intent was defaulted")
	}
	state := p.Project().State()
	if state.Runtimes.Form() != project.Absent || state.ModelProfiles.Form() != project.Absent {
		t.Fatal("runtime policy invented")
	}
	if strings.Contains(string(p.Manifest()), "/work/core") || strings.Contains(string(p.Manifest()), "codex") {
		t.Fatalf("portable manifest leaked local state or default: %s", p.Manifest())
	}
}

func TestBootstrapRemoteDiscoveryRules(t *testing.T) {
	single := prepare(t, bootstrapInput(projectapp.SetupRepository{Key: "core", Path: "/w/core", Revision: "r", Discovery: discovered("single", "https://github.com/acme/core.git")}))
	if repos, _ := single.Project().State().Repositories.Value(); repos[0].Remote.Form() != project.Present || !single.Publishable() {
		t.Fatal("single distinct locator not proposed")
	}
	if remote := single.Preview().Repositories[0].Remote; remote.Source != "discovered" || remote.Locator != "https://github.com/acme/core.git" {
		t.Fatalf("remote provenance %+v", remote)
	}
	// origin is never preferred: two distinct locators block until chosen.
	ambiguous := bootstrapInput(projectapp.SetupRepository{Key: "core", Path: "/w/core", Revision: "r", Discovery: discovered("ambiguous", "git@github.com:acme/core.git", "https://github.com/acme/core.git")})
	blocked := prepare(t, ambiguous)
	if blocked.Publishable() || blockers(blocked) != "repository_remote_ambiguous:core" {
		t.Fatalf("ambiguity not blocking: %s", blockers(blocked))
	}
	if repos, _ := blocked.Project().State().Repositories.Value(); repos[0].Remote.Form() != project.Absent {
		t.Fatal("ambiguous remote silently chosen")
	}
	ambiguous.RepositoryRemotes = map[string]string{"core": "https://github.com/acme/core.git"}
	chosen := prepare(t, ambiguous)
	if !chosen.Publishable() || chosen.Preview().Repositories[0].Remote.Source != "operator" {
		t.Fatal("explicit choice not applied")
	}
	ambiguous.RepositoryRemotes = map[string]string{"core": "none"}
	if local := prepare(t, ambiguous); !local.Publishable() {
		t.Fatal("explicit local-only not applied")
	}
	incomplete := prepare(t, bootstrapInput(projectapp.SetupRepository{Key: "core", Path: "/w/core", Revision: "r", Discovery: discovered("incomplete", "https://github.com/acme/core.git")}))
	if incomplete.Publishable() {
		t.Fatal("incomplete Git configuration silently proposed")
	}
	unsafe := &projectapp.RepositoryDiscovery{Git: "unsafe", RemoteStatus: "incomplete"}
	if p := prepare(t, bootstrapInput(projectapp.SetupRepository{Key: "core", Path: "/w/core", Revision: "r", Discovery: unsafe})); !strings.Contains(blockers(p), "repository_unsafe:core") {
		t.Fatal("unsafe Git metadata not blocking")
	}
	for _, remote := range []string{"/srv/git/core.git", "https://user:secret@example.com/r.git", "https://example.com/r.git?token=x", "relative/path"} {
		input := bootstrapInput(projectapp.SetupRepository{Key: "core", Path: "/w/core", Revision: "r"})
		input.RepositoryRemotes = map[string]string{"core": remote}
		rejectSetup(t, input, projectapp.InvalidRepositoryInput)
	}
	input := bootstrapInput(projectapp.SetupRepository{Key: "core", Path: "/w/core", Revision: "r"})
	input.RepositoryRemotes = map[string]string{"other": "none"}
	rejectSetup(t, input, projectapp.InvalidRepositoryInput)
}

func TestBootstrapMultiRepositoryAndDerivedKeys(t *testing.T) {
	p := prepare(t, bootstrapInput(
		projectapp.SetupRepository{Path: "/w/Web App", Revision: "r1", Discovery: &projectapp.RepositoryDiscovery{Git: "absent", RemoteStatus: "local_only", DerivedKey: "web-app"}},
		projectapp.SetupRepository{Key: "api", Path: "/w/api", Revision: "r2", Discovery: discovered("single", "https://github.com/acme/api.git")},
	))
	repos, _ := p.Project().State().Repositories.Value()
	if len(repos) != 2 || repos[0].Key != "api" || repos[1].Key != "web-app" || p.Preview().Repositories[1].KeySource != "derived" || !p.Publishable() {
		t.Fatalf("multi-repository bootstrap wrong: %+v %s", repos, blockers(p))
	}
	clash := prepare(t, bootstrapInput(
		projectapp.SetupRepository{Path: "/a/core", Revision: "r1", Discovery: &projectapp.RepositoryDiscovery{DerivedKey: "core"}},
		projectapp.SetupRepository{Path: "/b/core", Revision: "r2", Discovery: &projectapp.RepositoryDiscovery{DerivedKey: "core"}},
	))
	if clash.Publishable() || !strings.Contains(blockers(clash), "repository_key_conflict:core") {
		t.Fatal("derived key clash not blocking")
	}
	missing := prepare(t, bootstrapInput(projectapp.SetupRepository{Path: "/w/___", Revision: "r", Discovery: &projectapp.RepositoryDiscovery{}}))
	if missing.Publishable() || !strings.Contains(blockers(missing), "repository_key_required") {
		t.Fatal("unusable derived key not blocking")
	}
}

func runtimeCandidates() []projectapp.RuntimeCandidate {
	return []projectapp.RuntimeCandidate{
		{ID: "codex", Profiles: []projectapp.ProfileCandidate{{Key: "worker", Model: "approved-model"}}},
		{ID: "claude", Profiles: []projectapp.ProfileCandidate{{Key: "careful", Model: "careful-model"}}},
	}
}

func TestBootstrapRuntimePolicyIsExplicitAndLocal(t *testing.T) {
	input := bootstrapInput(projectapp.SetupRepository{Key: "core", Path: "/w/core", Revision: "r"})
	input.RuntimeCandidates = runtimeCandidates()
	none := prepare(t, input)
	if none.Preview().RuntimePolicy.Status != "absent" || len(none.Preview().RuntimePolicy.Candidates) != 2 || none.Project().State().Runtimes.Form() != project.Absent {
		t.Fatal("candidates were treated as a default policy")
	}
	input.Runtimes, input.ModelProfiles = []string{"claude"}, []string{"careful"}
	input.RuntimePreferences = []projectapp.SetupPreference{{Role: "implementation", Complexity: "high", ModelProfile: "careful"}}
	chosen := prepare(t, input)
	state := chosen.Project().State()
	runtimes, _ := state.Runtimes.Value()
	profiles, _ := state.ModelProfiles.Value()
	model, _ := profiles[0].Model.Value()
	if len(runtimes) != 1 || runtimes[0].ID != "claude" || model != "careful-model" || state.RuntimePreferences.Form() != project.Present {
		t.Fatalf("runtime policy not authored: %+v %+v", runtimes, profiles)
	}
	for name, edit := range map[string]func(*projectapp.SetupInput){
		"unknown runtime":          func(i *projectapp.SetupInput) { i.Runtimes = []string{"other"} },
		"profile other runtime":    func(i *projectapp.SetupInput) { i.ModelProfiles = []string{"worker"} },
		"unknown profile":          func(i *projectapp.SetupInput) { i.ModelProfiles = []string{"missing"} },
		"preference unselected":    func(i *projectapp.SetupInput) { i.RuntimePreferences[0].ModelProfile = "worker" },
		"profiles without runtime": func(i *projectapp.SetupInput) { i.Runtimes = nil },
		"duplicate runtime":        func(i *projectapp.SetupInput) { i.Runtimes = []string{"claude", "claude"} },
		"no local candidates":      func(i *projectapp.SetupInput) { i.RuntimeCandidates = nil },
	} {
		copy := input
		copy.RuntimePreferences = append([]projectapp.SetupPreference(nil), input.RuntimePreferences...)
		edit(&copy)
		t.Run(name, func(t *testing.T) { rejectSetup(t, copy, projectapp.InvalidRuntimePolicyInput) })
	}
}

func TestBootstrapTechnologyProposalsAreEditable(t *testing.T) {
	core := &projectapp.RepositoryDiscovery{Git: "absent", RemoteStatus: "local_only", Technology: []projectapp.TechnologyProposal{
		{Key: "language.go", Value: "go", Path: "go.mod", Count: 1},
		{Key: "package-manager.npm", Value: "npm", Path: "package-lock.json", Count: 1, Conflict: "package-manager"},
		{Key: "package-manager.pnpm", Value: "pnpm", Path: "pnpm-lock.yaml", Count: 1, Conflict: "package-manager"},
	}}
	web := &projectapp.RepositoryDiscovery{Git: "absent", RemoteStatus: "local_only", Technology: []projectapp.TechnologyProposal{{Key: "language.go", Value: "go", Path: "tools/go.mod", Count: 2}}}
	input := bootstrapInput(projectapp.SetupRepository{Key: "web", Path: "/w/web", Revision: "r2", Discovery: web}, projectapp.SetupRepository{Key: "core", Path: "/w/core", Revision: "r1", Discovery: core})
	detected := prepare(t, input)
	technology := detected.Preview().Technology
	if len(technology) != 3 || technology[0].Key != "language.go" || len(technology[0].Evidence) != 2 || technology[0].Evidence[0].Repository != "core" || technology[1].Conflict != "package-manager" {
		t.Fatalf("detected technology not reviewable: %+v", technology)
	}
	input.RemoveTechnology = []string{"package-manager.npm"}
	input.Technology = []project.TechnologyFact{{Key: "language.go", Value: "go1.26"}, {Key: "cloud.aws", Value: "aws"}}
	edited := prepare(t, input)
	facts, _ := edited.Project().State().TechnologyContext.Value()
	if len(facts) != 3 || facts[0].Key != "cloud.aws" || facts[1].Value != "go1.26" || facts[2].Key != "package-manager.pnpm" {
		t.Fatalf("technology edit not applied: %+v", facts)
	}
	if edited.Preview().Digest == detected.Preview().Digest {
		t.Fatal("digest does not cover technology edits")
	}
	for _, edit := range []func(*projectapp.SetupInput){
		func(i *projectapp.SetupInput) { i.RemoveTechnology = []string{"language.rust"} },
		func(i *projectapp.SetupInput) { i.Technology = []project.TechnologyFact{{Key: "Bad", Value: "x"}} },
		func(i *projectapp.SetupInput) {
			i.Technology = []project.TechnologyFact{{Key: "a", Value: "/usr/bin/go"}}
		},
	} {
		copy := input
		edit(&copy)
		if _, issues := projectapp.PrepareSetup(manifest.Codec{}, copy, bootstrapObservation); len(issues) == 0 {
			t.Fatal("invalid technology edit accepted")
		}
	}
}

func TestBootstrapDocumentationContextAndGlossary(t *testing.T) {
	binding := &projectapp.DocumentationBinding{SourceKey: "product-notes", ExplicitPath: "/home/user/notes/product.md", CanonicalIdentity: "file:" + strings.Repeat("a", 64), Observation: projectapp.Observation{Availability: projectapp.Available, Basis: projectapp.PresentMetadata, ObservedAt: time.Unix(1, 0).UTC()}}
	input := bootstrapInput(projectapp.SetupRepository{Key: "core", Path: "/w/core", Revision: "r"})
	input.Documentation = []projectapp.SetupDocumentation{
		{Key: "product-notes", Kind: project.LocalFileSource, Binding: binding},
		{Key: "architecture", Kind: project.RepositorySource, RepositoryRef: "core", Path: "docs/architecture"},
	}
	input.BusinessContext = "Bounded context.\nSecond line."
	input.ContextSources = []string{"product-notes", "architecture"}
	input.Glossary = []project.GlossaryEntry{{Key: "work-item", Term: "Work Item", Definition: "A bounded unit."}}
	p := prepare(t, input)
	manifestText := string(p.Manifest())
	if strings.Contains(manifestText, "/home/user") || !strings.Contains(manifestText, "kind: local-file") || !strings.Contains(manifestText, "glossary:") {
		t.Fatalf("portable/local separation broken:\n%s", manifestText)
	}
	if docs := p.DocumentationBindings(); len(docs) != 1 || docs[0].ExplicitPath != "/home/user/notes/product.md" {
		t.Fatal("local documentation binding lost")
	}
	if p.Preview().Documentation[1].LocalPath == "" || p.Preview().BusinessContext.SourceRefs[0] != "architecture" {
		t.Fatalf("preview incomplete: %+v", p.Preview().Documentation)
	}
	for name, tc := range map[string]struct {
		edit func(*projectapp.SetupInput)
		code projectapp.Code
	}{
		"unbound local file": {func(i *projectapp.SetupInput) { i.Documentation[0].Binding = nil }, projectapp.InvalidDocumentationInput},
		"traversal":          {func(i *projectapp.SetupInput) { i.Documentation[1].Path = "../secrets" }, projectapp.InvalidDocumentationInput},
		"absolute repo path": {func(i *projectapp.SetupInput) { i.Documentation[1].Path = "/etc" }, projectapp.InvalidDocumentationInput},
		"unknown repository": {func(i *projectapp.SetupInput) { i.Documentation[1].RepositoryRef = "other" }, projectapp.InvalidDocumentationInput},
		"unknown kind":       {func(i *projectapp.SetupInput) { i.Documentation[1].Kind = "notion" }, projectapp.InvalidDocumentationInput},
		"duplicate key":      {func(i *projectapp.SetupInput) { i.Documentation[1].Key = "product-notes" }, projectapp.InvalidDocumentationInput},
		"dangling sourceRef": {func(i *projectapp.SetupInput) { i.ContextSources = []string{"missing"} }, projectapp.InvalidContextInput},
		"long context":       {func(i *projectapp.SetupInput) { i.BusinessContext = strings.Repeat("x", 4097) }, projectapp.InvalidContextInput},
		"duplicate glossary": {func(i *projectapp.SetupInput) { i.Glossary = append(i.Glossary, i.Glossary[0]) }, projectapp.InvalidContextInput},
		"bad glossary term": {func(i *projectapp.SetupInput) {
			i.Glossary = []project.GlossaryEntry{{Key: "k", Term: "", Definition: "d"}}
		}, projectapp.InvalidContextInput},
	} {
		copy := input
		copy.Documentation = append([]projectapp.SetupDocumentation(nil), input.Documentation...)
		tc.edit(&copy)
		t.Run(name, func(t *testing.T) { rejectSetup(t, copy, tc.code) })
	}
}

func TestBootstrapPreviewIsDeterministicAndBounded(t *testing.T) {
	input := bootstrapInput(projectapp.SetupRepository{Key: "core", Path: "/w/core", Revision: "r", Discovery: discovered("single", "https://github.com/acme/core.git")})
	input.WorkItemProvider = "github"
	first, second := prepare(t, input), prepare(t, input)
	if first.Preview().Digest != second.Preview().Digest {
		t.Fatal("same inputs produced different digests")
	}
	wire, _ := json.Marshal(first.Preview())
	if !strings.Contains(string(wire), `"grantsAuthority":false`) || first.Preview().Capability.Readiness != projectapp.CapabilityReady {
		t.Fatalf("authority/capability disclosure missing: %s", wire)
	}
	input.WorkItemProvider = "linear"
	if unsupported := prepare(t, input); unsupported.Preview().Capability.Readiness != projectapp.CapabilityUnsupported || !unsupported.Publishable() {
		t.Fatal("unsupported Provider must be declared truthfully and remain publishable")
	}
}

func TestBootstrapRejectsUnsafePortableProse(t *testing.T) {
	for _, field := range []string{"text", "term", "definition"} {
		for _, value := range []string{"[Docs](https://example.com);password=synthetic", "[Docs](https://example.com),/home/user/private", "path:/home/user/private", "location:~/private", `Read path:C:\Users\user\private`, "Use `token=synthetic`", "See `/home/user/private`", "See `file:/private/document`", "Use **password=synthetic**", "See https://example.com/docs?q=a,b&token=synthetic", "See https://example.com/docs?q=a;b&token=synthetic", "token=synthetic", "password=synthetic", "api_key=synthetic", "/Users/user/private", "/home/user/private", `C:\Users\user\private`, "file:/private/document", "file:///tmp/document", "%252Fhome%252Fuser", "https://example.com?token=synthetic"} {
			t.Run(field+"/"+value, func(t *testing.T) {
				input := bootstrapInput(projectapp.SetupRepository{Key: "core", Path: "/w/core", Revision: "r"})
				if field == "text" {
					input.BusinessContext = value
				} else {
					entry := project.GlossaryEntry{Key: "work-item", Term: "Work Item", Definition: "A bounded unit of work."}
					if field == "term" {
						entry.Term = value
					} else {
						entry.Definition = value
					}
					input.Glossary = []project.GlossaryEntry{entry}
				}
				proposal, issues := projectapp.PrepareSetup(manifest.Codec{}, input, bootstrapObservation)
				if len(issues) != 1 || issues[0].Code != projectapp.InvalidContextInput {
					t.Fatalf("want InvalidContextInput: %v", issues)
				}
				if proposal.Valid() || proposal.Publishable() || proposal.Manifest() != nil || !reflect.DeepEqual(proposal.Preview(), projectapp.SetupPreview{}) {
					t.Fatal("rejected context left portable proposal state")
				}
				store := &prosePublicationSpy{}
				lifecycle := projectapp.NewLifecycle(store, manifest.Codec{}, &fakeEntropy{})
				result := lifecycle.PublishConfigured(context.Background(), proposal.Project())
				if result.Status != projectapp.LifecycleFailed || store.calls != 0 {
					t.Fatal("rejected context reached storage")
				}
			})
		}
	}
}

type prosePublicationSpy struct{ calls int }

func (s *prosePublicationSpy) Read(context.Context, string) ([]byte, error) {
	s.calls++
	return nil, projectapp.ErrNotFound
}
func (s *prosePublicationSpy) Create(context.Context, string, []byte) error { s.calls++; return nil }
func (s *prosePublicationSpy) Update(context.Context, string, []byte, []byte) error {
	s.calls++
	return nil
}

func TestBootstrapAcceptsPortableProse(t *testing.T) {
	input := bootstrapInput(projectapp.SetupRepository{Key: "core", Path: "/w/core", Revision: "r"})
	input.BusinessContext = "The checkout domain handles orders.\nRefunds belong to the payments context."
	input.Glossary = []project.GlossaryEntry{{Key: "work-item", Term: "Work Item", Definition: "A bounded unit of work.\nSee https://example.com/docs?q=a,b&next=/orders"}}
	proposal := prepare(t, input)
	preview := proposal.Preview().BusinessContext
	if !proposal.Publishable() || preview.Text != input.BusinessContext || !reflect.DeepEqual(preview.Glossary, input.Glossary) {
		t.Fatal("normal prose changed in bootstrap preview")
	}
	decoded, issues := manifest.Decode(proposal.Manifest())
	if len(issues) != 0 || !decoded.Equivalent(proposal.Project()) {
		t.Fatal("normal prose failed portable round trip")
	}
}
