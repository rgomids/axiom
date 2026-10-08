package projectapp_test

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/project"
	"github.com/rgomids/axiom/internal/projectapp"
)

// Issue #230 I230-T04 Evidence: Integration inventory, static validation and
// portable removal are pure application rules over portable intent and
// machine-local operational state.

func fixtureState(t *testing.T) project.State {
	t.Helper()
	return newEditFixture(t).selection.Portable.Project().State()
}

func operational(disabled ...string) projectapp.OperationalState {
	state := projectapp.DefaultOperationalState()
	state.DisabledIntegrations = append(state.DisabledIntegrations, disabled...)
	return state
}

func TestIntegrationInventoryNamesDeclarationsAndLocalEligibilityOnly(t *testing.T) {
	report := projectapp.InventoryIntegrations(fixtureState(t), operational("chat", "gone"))
	if report.ProjectID != editProjectID || len(report.Integrations) != 2 || !reflect.DeepEqual(report.StaleDisabled, []string{"gone"}) {
		t.Fatalf("report = %+v", report)
	}
	chat, workItems := report.Integrations[0], report.Integrations[1]
	if chat.Key != "chat" || chat.Provider != "slack" || chat.ProviderRef != "chat" || chat.CredentialRef != "chat-token" || chat.Local != projectapp.IntegrationDisabled || len(chat.Capabilities) != 0 {
		t.Fatalf("chat = %+v", chat)
	}
	if workItems.Key != "work-items" || workItems.Provider != "github" || !reflect.DeepEqual(workItems.Capabilities, []string{"work-item"}) || workItems.Local != projectapp.IntegrationEnabled {
		t.Fatalf("work-items = %+v", workItems)
	}
	wire, err := json.Marshal(report)
	if err != nil || strings.Contains(string(wire), "keychain") || strings.Contains(string(wire), "axiom/chat-item") {
		t.Fatalf("report leaks credential material or fails to encode: %v %s", err, wire)
	}
	if shown, found := projectapp.ShowIntegration(fixtureState(t), operational("gone"), "work-items"); !found || len(shown.Integrations) != 1 || shown.StaleDisabled != nil {
		t.Fatalf("show = %+v %v", shown, found)
	}
	if _, found := projectapp.ShowIntegration(fixtureState(t), operational("gone"), "gone"); found {
		t.Fatal("a stale disabled key was shown as declared")
	}
	if empty := projectapp.InventoryIntegrations(project.State{ID: editProjectID}, operational()); len(empty.Integrations) != 0 || empty.Integrations == nil {
		t.Fatalf("unconfigured inventory = %+v", empty)
	}
}

func findingCodes(report projectapp.IntegrationReport) []string {
	codes := []string{}
	for _, finding := range report.Validation.Findings {
		codes = append(codes, finding.Severity+":"+finding.Code+":"+finding.Integration+":"+finding.Capability)
	}
	return codes
}

func TestIntegrationValidationIsStaticAndStable(t *testing.T) {
	state := fixtureState(t)
	report := projectapp.ValidateIntegrations(state, operational("chat", "gone"), projectapp.SupportedProviders, "")
	want := []string{
		"warning:integration_disabled_locally:chat:",
		"warning:integration_capabilities_missing:chat:",
		"warning:integration_disabled_stale:gone:",
	}
	got := findingCodes(report)
	if report.Validation.Status != projectapp.IntegrationValid || len(got) != len(want) {
		t.Fatalf("validation = %+v", report.Validation)
	}
	for _, expected := range want {
		found := false
		for _, code := range got {
			found = found || code == expected
		}
		if !found {
			t.Fatalf("%s missing from %v", expected, got)
		}
	}
	filtered := projectapp.ValidateIntegrations(state, operational("chat", "gone"), projectapp.SupportedProviders, "work-items")
	if len(filtered.Integrations) != 1 || len(filtered.Validation.Findings) != 0 || filtered.Validation.Status != projectapp.IntegrationValid || filtered.StaleDisabled != nil {
		t.Fatalf("filtered = %+v", filtered)
	}

	// An unsupported Provider for a declared capability is an error.
	unsupported := state
	unsupported.Providers = project.Configured([]project.Provider{{Key: "chat", ID: "slack"}, {Key: "work-items", ID: "gitlab"}})
	report = projectapp.ValidateIntegrations(unsupported, operational(), projectapp.SupportedProviders, "")
	if report.Validation.Status != projectapp.IntegrationInvalid || !contains(findingCodes(report), "error:provider_unsupported:work-items:work-item") {
		t.Fatalf("unsupported = %+v", report.Validation)
	}

	// Two non-conventional Integrations declaring one capability are ambiguous
	// for both; order never chooses.
	ambiguous := state
	ambiguous.Integrations = project.Configured([]project.Integration{
		{Key: "primary", ProviderRef: project.Configured("work-items"), Capabilities: project.Configured([]string{"work-item"})},
		{Key: "secondary", ProviderRef: project.Configured("work-items"), Capabilities: project.Configured([]string{"work-item"})},
	})
	ambiguous.CredentialReferences = project.Unconfigured[[]project.CredentialReference]()
	report = projectapp.ValidateIntegrations(ambiguous, operational(), projectapp.SupportedProviders, "")
	if report.Validation.Status != projectapp.IntegrationInvalid || !contains(findingCodes(report), "error:capability_mapping_ambiguous:primary:work-item") || !contains(findingCodes(report), "error:capability_mapping_ambiguous:secondary:work-item") {
		t.Fatalf("ambiguous = %v", findingCodes(report))
	}
	only := projectapp.ValidateIntegrations(ambiguous, operational(), projectapp.SupportedProviders, "primary")
	if len(only.Validation.Findings) != 1 || only.Validation.Findings[0].Integration != "primary" {
		t.Fatalf("filtered ambiguity = %+v", only.Validation)
	}

	// No declaration: valid with an explicit warning.
	none := projectapp.ValidateIntegrations(project.State{ID: editProjectID}, operational(), projectapp.SupportedProviders, "")
	if none.Validation.Status != projectapp.IntegrationValid || !reflect.DeepEqual(findingCodes(none), []string{"warning:integrations_not_declared::"}) {
		t.Fatalf("none = %+v", none.Validation)
	}
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func TestRemovingNonCanonicalIntegrationKeepsProviderAndCredentialDeclarations(t *testing.T) {
	f := newEditFixture(t)
	proposal := mustPreview(t, f, projectapp.EditIntent{IntegrationRemovals: []string{"chat"}})
	if got := effectCodes(proposal.Preview().Effects); !reflect.DeepEqual(got, []string{"portable:update_portable_project", "portable:remove_portable_integration:chat", "portable:preserve_portable_credential:chat-token", "portable:preserve_portable_provider:chat", "local:update_local_record"}) {
		t.Fatalf("effects = %v", got)
	}
	state := proposal.Project().State()
	providers, _ := state.Providers.Value()
	credentials, _ := state.CredentialReferences.Value()
	integrations, _ := state.Integrations.Value()
	if len(providers) != 2 || len(credentials) != 1 || len(integrations) != 1 || integrations[0].Key != "work-items" {
		t.Fatalf("candidate providers=%v credentials=%v integrations=%v", providers, credentials, integrations)
	}
	if !strings.Contains(proposal.Preview().PortableManifest, "Unrelated portable context") || !strings.Contains(proposal.Preview().PortableManifest, "chat-token") {
		t.Fatal("unrelated portable values were not preserved")
	}
	if proposal.Preview().Capability.Readiness != projectapp.CapabilityReady {
		t.Fatalf("capability = %+v", proposal.Preview().Capability)
	}
	// Only the recorded portable revision follows; bindings and credential
	// bindings are untouched, and no operational state exists in the proposal.
	if !reflect.DeepEqual(proposal.Local().Repositories, f.selection.Local.Repositories) || !reflect.DeepEqual(proposal.Local().Credentials, f.selection.Local.Credentials) {
		t.Fatal("portable Integration removal changed local bindings")
	}
}

func TestRemovingWorkItemsIntegrationRemovesItsPairedProvider(t *testing.T) {
	f := newEditFixture(t)
	proposal := mustPreview(t, f, projectapp.EditIntent{IntegrationRemovals: []string{"work-items"}})
	if got := effectCodes(proposal.Preview().Effects); !reflect.DeepEqual(got, []string{"portable:update_portable_project", "portable:remove_portable_integration:work-items", "portable:remove_portable_provider:work-items", "local:update_local_record"}) {
		t.Fatalf("effects = %v", got)
	}
	if _, configured := workItemProvider(t, proposal.Project()); configured {
		t.Fatal("paired work-items Provider survived")
	}
	if proposal.Preview().Capability.Readiness != projectapp.CapabilityMissing {
		t.Fatalf("capability = %+v", proposal.Preview().Capability)
	}
	// Equivalent to the preview-only predecessor for the same pair.
	predecessor := mustPreview(t, newEditFixture(t), projectapp.EditIntent{RemoveWorkItemProvider: true})
	if proposal.Preview().PortableManifest != predecessor.Preview().PortableManifest {
		t.Fatal("generalized removal differs from removeWorkItemProvider for the work-items pair")
	}

	// Removing every Integration uses the unconfigured form, like CREATE.
	both := mustPreview(t, newEditFixture(t), projectapp.EditIntent{IntegrationRemovals: []string{"work-items", "chat"}})
	if both.Project().State().Integrations.Form() != project.NotConfigured {
		t.Fatalf("integrations form = %v", both.Project().State().Integrations.Form())
	}
	if got := effectCodes(both.Preview().Effects); !reflect.DeepEqual(got, []string{"portable:update_portable_project", "portable:remove_portable_integration:chat", "portable:remove_portable_integration:work-items", "portable:remove_portable_provider:work-items", "portable:preserve_portable_credential:chat-token", "portable:preserve_portable_provider:chat", "local:update_local_record"}) {
		t.Fatalf("effects = %v", got)
	}
}

func TestEveryDeclarableIntegrationKeyIsRemovable(t *testing.T) {
	// Portable Integration keys use the token grammar ([a-z0-9._-]), wider
	// than the setup key grammar; removal must accept every declarable key.
	intent := projectapp.EditIntent{Selector: "sample", IntegrationRemovals: []string{"chat_v2.bot"}}
	if failure := projectapp.ValidateEditIntent(intent); failure != projectapp.EditOK {
		t.Fatalf("declarable key rejected: %v", failure)
	}
	if _, failure := newEditFixture(t).preview(t, intent); failure != projectapp.EditUnknownIntegration {
		t.Fatalf("undeclared token key = %v", failure)
	}
}

func TestRemovingIntegrationFailsClosed(t *testing.T) {
	f := newEditFixture(t)
	if _, failure := f.preview(t, projectapp.EditIntent{IntegrationRemovals: []string{"nope"}}); failure != projectapp.EditUnknownIntegration {
		t.Fatalf("unknown failure = %d", failure)
	}
	for name, intent := range map[string]projectapp.EditIntent{
		"duplicate":    {IntegrationRemovals: []string{"chat", "chat"}},
		"invalid key":  {IntegrationRemovals: []string{"Bad Key"}},
		"empty key":    {IntegrationRemovals: []string{""}},
		"with set":     {IntegrationRemovals: []string{"chat"}, WorkItemProvider: set("github")},
		"with remove":  {IntegrationRemovals: []string{"chat"}, RemoveWorkItemProvider: true},
		"too many":     {IntegrationRemovals: make([]string, projectapp.MaxDisabledIntegrations+1)},
		"path-like":    {IntegrationRemovals: []string{"../chat"}},
		"oversize key": {IntegrationRemovals: []string{strings.Repeat("a", 4096)}},
	} {
		if projectapp.ValidateEditIntent(withSelector(intent)) != projectapp.EditInvalidIntent {
			t.Fatalf("%s accepted", name)
		}
	}
	// A dangling reference is never left behind: another Integration still
	// pointing at providers[work-items] makes the complete candidate invalid.
	state := fixtureState(t)
	state.Integrations = project.Configured([]project.Integration{
		{Key: "work-items", ProviderRef: project.Configured("work-items"), Capabilities: project.Configured([]string{"work-item"})},
		{Key: "mirror", ProviderRef: project.Configured("work-items"), Capabilities: project.Configured([]string{"mirror"})},
	})
	dangling := newEditFixture(t)
	dangling.replacePortable(t, state)
	if _, failure := dangling.preview(t, projectapp.EditIntent{IntegrationRemovals: []string{"work-items"}}); failure != projectapp.EditCandidateInvalid {
		t.Fatalf("dangling failure = %d", failure)
	}
	// Removing only the dependent keeps the shared Provider.
	if _, failure := dangling.preview(t, projectapp.EditIntent{IntegrationRemovals: []string{"mirror"}}); failure != projectapp.EditOK {
		t.Fatalf("mirror removal failure = %d", failure)
	}
}

func withSelector(intent projectapp.EditIntent) projectapp.EditIntent {
	intent.Selector = "sample"
	return intent
}

type stubSelection struct{ id, category string }

func (s stubSelection) SelectProjectID(context.Context, string) (string, string) {
	return s.id, s.category
}

type stubProjects struct {
	project project.Project
	code    string
}

func (s stubProjects) LoadReadiness(context.Context, string) (projectapp.ReadinessSubject, string) {
	return projectapp.ReadinessSubject{Project: s.project}, s.code
}

type stubOperational struct{ state projectapp.OperationalState }

func (s stubOperational) InspectOperational(context.Context, string) (projectapp.OperationalObservation, error) {
	return projectapp.OperationalObservation{Exists: true, Revision: "r", State: s.state}, nil
}

func (stubOperational) CommitOperational(context.Context, string, string, projectapp.OperationalState) error {
	return nil
}

func TestIntegrationTargetResolution(t *testing.T) {
	portable := newEditFixture(t).selection.Portable.Project()
	inspector := func(ops projectapp.OperationalState, code string) projectapp.IntegrationInspector {
		return projectapp.IntegrationInspector{Selection: stubSelection{id: editProjectID}, Projects: stubProjects{project: portable, code: code}, Operational: stubOperational{ops}, Catalog: projectapp.SupportedProviders}
	}
	ctx := context.Background()
	for name, test := range map[string]struct {
		ops       projectapp.OperationalState
		code      string
		operation projectapp.OperationalOperation
		key       string
		want      string
	}{
		"disable declared":                {operational(), "", projectapp.DisableIntegration, "chat", ""},
		"disable undeclared":              {operational(), "", projectapp.DisableIntegration, "gone", projectapp.IntegrationNotFound},
		"disable stale is undeclared":     {operational("gone"), "", projectapp.DisableIntegration, "gone", projectapp.IntegrationNotFound},
		"enable declared":                 {operational("chat"), "", projectapp.EnableIntegration, "chat", ""},
		"enable declared already enabled": {operational(), "", projectapp.EnableIntegration, "chat", ""},
		"enable stale clears":             {operational("gone"), "", projectapp.EnableIntegration, "gone", ""},
		"enable stale needs no portable":  {operational("gone"), projectapp.BlockerProjectSourceUnavailable, projectapp.EnableIntegration, "gone", ""},
		"enable unknown":                  {operational(), "", projectapp.EnableIntegration, "gone", projectapp.IntegrationNotFound},
		"disable needs portable":          {operational(), projectapp.BlockerProjectSourceUnavailable, projectapp.DisableIntegration, "chat", projectapp.BlockerProjectSourceUnavailable},
		"invalid key":                     {operational(), "", projectapp.DisableIntegration, "Bad Key", projectapp.OperationalInvalidInput},
		"wrong operation":                 {operational(), "", projectapp.ArchiveProject, "chat", projectapp.OperationalInvalidInput},
	} {
		id, category := inspector(test.ops, test.code).Target(ctx, "sample", test.operation, test.key)
		if category != test.want || (category == "" && id != editProjectID) {
			t.Fatalf("%s: id=%q category=%q want %q", name, id, category, test.want)
		}
	}
	if _, category := (projectapp.IntegrationInspector{}).Target(ctx, "sample", projectapp.DisableIntegration, "chat"); category != projectapp.OperationalStorageFailure {
		t.Fatalf("uncomposed category = %q", category)
	}
	missing := projectapp.IntegrationInspector{Selection: stubSelection{category: "project_not_found"}, Projects: stubProjects{project: portable}, Operational: stubOperational{operational()}}
	if _, category := missing.List(ctx, "sample"); category != "project_not_found" {
		t.Fatalf("selection category = %q", category)
	}
	if _, category := inspector(operational(), "").Show(ctx, "sample", "gone"); category != projectapp.IntegrationNotFound {
		t.Fatalf("show category = %q", category)
	}
	if report, category := inspector(operational("chat"), "").Validate(ctx, "sample", "chat"); category != "" || len(report.Integrations) != 1 || report.Validation == nil {
		t.Fatalf("validate = %+v %q", report, category)
	}
}
