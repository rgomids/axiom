package projectapp

import (
	"context"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/project"
)

type selectorFake struct {
	id, category string
	calls        int
}

func (s *selectorFake) SelectProjectID(context.Context, string) (string, string) {
	s.calls++
	return s.id, s.category
}

type projectsFake struct {
	subject ReadinessSubject
	code    string
	calls   int
}

func (p *projectsFake) LoadReadiness(context.Context, string) (ReadinessSubject, string) {
	p.calls++
	return p.subject, p.code
}

func admissionProject(t *testing.T, integrations ...project.Integration) project.Project {
	t.Helper()
	state := project.State{SchemaVersion: 2, ID: operationalTestID, Slug: "sample", Name: "Sample"}
	if len(integrations) != 0 {
		state.Providers = project.Configured([]project.Provider{{Key: "work-items", ID: "github"}})
		state.Integrations = project.Configured(integrations)
	}
	p, issues := project.New(state)
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	return p
}

func workItemsIntegration(key string) project.Integration {
	return project.Integration{Key: key, ProviderRef: project.Configured("work-items"), Capabilities: project.Configured([]string{WorkItemCapability})}
}

type admissionFixture struct {
	selection *selectorFake
	projects  *projectsFake
	store     *operationalSpy
}

func newAdmissionFixture(t *testing.T, state OperationalState) admissionFixture {
	revision := OperationalRevisionAbsent
	if !state.Equal(DefaultOperationalState()) {
		revision = strings.Repeat("e", 64)
	}
	return admissionFixture{
		selection: &selectorFake{id: operationalTestID},
		projects:  &projectsFake{subject: ReadinessSubject{Project: admissionProject(t, workItemsIntegration("work-items"))}},
		store:     &operationalSpy{observation: OperationalObservation{Exists: revision != OperationalRevisionAbsent, Revision: revision, State: state}},
	}
}

func (f admissionFixture) guard() AdmissionGuard {
	return AdmissionGuard{Selection: f.selection, Projects: f.projects, Operational: f.store, Catalog: SupportedProviders}
}

func TestEveryLifecycleOperationHasOneClassification(t *testing.T) {
	want := map[AdmissionClass][]AdmissionOperation{
		AdmissionInspection: {AdmitProjectList, AdmitProjectShow, AdmitProjectResolve, AdmitProjectValidate, AdmitRepositoryList, AdmitRepositoryShow,
			AdmitIntegrationList, AdmitIntegrationShow, AdmitIntegrationValidate, AdmitWorkItemList, AdmitWorkItemShow, AdmitExecutionList, AdmitExecutionStatus, AdmitExecutionEvidence},
		AdmissionAdministrative: {AdmitProjectConfigure, AdmitProjectEdit, AdmitProjectArchive, AdmitProjectReactivate, AdmitRepositoryAttach, AdmitRepositoryUpdate, AdmitRepositoryDetach,
			AdmitIntegrationUpdate, AdmitIntegrationDisable, AdmitIntegrationEnable, AdmitIntegrationRemove},
		AdmissionEvolution: {AdmitWorkItemCreate, AdmitWorkItemSelect, AdmitWorkItemUpdate, AdmitWorkItemComment, AdmitWorkItemClose, AdmitWorkItemReopen, AdmitWorkItemComplete,
			AdmitExecutionStart, AdmitExecutionAdvance, AdmitExecutionFact, AdmitExecutionResume, AdmitExecutionReconcile},
	}
	seen := 0
	for class, operations := range want {
		for _, operation := range operations {
			rule, ok := AdmissionRuleFor(operation)
			if !ok || rule.Class != class {
				t.Errorf("%s classified %+v, want %s", operation, rule, class)
			}
			seen++
		}
	}
	if len(AdmissionOperations()) != seen {
		t.Fatalf("catalog has %d operations, matrix lists %d", len(AdmissionOperations()), seen)
	}
	// Exactly the operations that reach the Provider through the Work Item
	// Integration declare it (I230-T01 F-05).
	for _, operation := range AdmissionOperations() {
		rule, _ := AdmissionRuleFor(operation)
		uses := strings.HasPrefix(string(operation), "work-item.") && rule.Class == AdmissionEvolution || operation == AdmitExecutionReconcile
		if (rule.Capability == WorkItemCapability) != uses {
			t.Errorf("%s capability = %q", operation, rule.Capability)
		}
	}
}

func TestAdmissionMatrixForArchivedProject(t *testing.T) {
	archivedState := OperationalState{ProjectStatus: ProjectArchived, DisabledIntegrations: []string{}}
	for _, operation := range AdmissionOperations() {
		fixture := newAdmissionFixture(t, archivedState)
		decision := fixture.guard().Admit(context.Background(), "sample", operation)
		rule, _ := AdmissionRuleFor(operation)
		if decision.GrantsAuthority {
			t.Fatalf("%s granted authority", operation)
		}
		if rule.Class == AdmissionEvolution {
			if decision.Allowed || decision.Code != AdmissionProjectArchived || decision.ProjectID != operationalTestID {
				t.Errorf("%s: %+v", operation, decision)
			}
			if fixture.projects.calls != 0 {
				t.Errorf("%s loaded portable state after an archive denial", operation)
			}
			continue
		}
		if !decision.Allowed || decision.Code != "" {
			t.Errorf("%s denied on an archived Project: %+v", operation, decision)
		}
	}
}

func TestAdmissionMatrixForDisabledIntegration(t *testing.T) {
	disabled := OperationalState{ProjectStatus: ProjectActive, DisabledIntegrations: []string{"work-items"}}
	for _, operation := range AdmissionOperations() {
		fixture := newAdmissionFixture(t, disabled)
		decision := fixture.guard().Admit(context.Background(), "sample", operation)
		rule, _ := AdmissionRuleFor(operation)
		if rule.Capability != "" {
			if decision.Allowed || decision.Code != AdmissionIntegrationDisabled || decision.Integration != "work-items" {
				t.Errorf("%s: %+v", operation, decision)
			}
			continue
		}
		if !decision.Allowed {
			t.Errorf("%s denied by an Integration it does not use: %+v", operation, decision)
		}
	}
}

func TestAdmissionTargetsTheIntegrationThatProvidesTheCapability(t *testing.T) {
	fixture := newAdmissionFixture(t, OperationalState{ProjectStatus: ProjectActive, DisabledIntegrations: []string{"work-items"}})
	fixture.projects.subject = ReadinessSubject{Project: admissionProject(t, workItemsIntegration("tracker"))}
	if decision := fixture.guard().Admit(context.Background(), "sample", AdmitWorkItemCreate); !decision.Allowed || decision.Integration != "tracker" {
		t.Fatalf("disabled unrelated key denied the tracker Integration: %+v", decision)
	}
	fixture.store.observation.State.DisabledIntegrations = []string{"tracker"}
	if decision := fixture.guard().Admit(context.Background(), "sample", AdmitWorkItemCreate); decision.Allowed || decision.Code != AdmissionIntegrationDisabled || decision.Integration != "tracker" {
		t.Fatalf("tracker Integration not denied: %+v", decision)
	}
}

func TestAdmissionFailsClosedWhenTheIntegrationCannotBeAttributed(t *testing.T) {
	for name, test := range map[string]struct {
		subject ReadinessSubject
		code    string
		want    string
	}{
		"no integration":      {subject: ReadinessSubject{Project: admissionProjectNoIntegration(t)}, want: BlockerCapabilityMissing},
		"ambiguous":           {subject: ReadinessSubject{Project: admissionProjectAmbiguous(t)}, want: BlockerCapabilityAmbiguous},
		"stale installation":  {code: BlockerInstallationStale, want: BlockerInstallationStale},
		"source unavailable":  {code: BlockerProjectSourceUnavailable, want: BlockerProjectSourceUnavailable},
		"selection drift":     {subject: ReadinessSubject{Project: admissionProjectOtherID(t)}, want: BlockerProjectStateInvalid},
		"portable recovering": {code: BlockerRecoveryRequired, want: BlockerRecoveryRequired},
	} {
		t.Run(name, func(t *testing.T) {
			fixture := newAdmissionFixture(t, DefaultOperationalState())
			fixture.projects.subject, fixture.projects.code = test.subject, test.code
			if decision := fixture.guard().Admit(context.Background(), "sample", AdmitExecutionReconcile); decision.Allowed || decision.Code != test.want {
				t.Fatalf("decision = %+v", decision)
			}
			// With a readiness requirement, the #231 preflight reports the
			// same code; admission defers instead of duplicating it.
			if decision := fixture.guard().Admit(context.Background(), "sample", AdmitWorkItemCreate); !decision.Allowed || decision.Deferred != test.want {
				t.Fatalf("readiness-gated decision = %+v", decision)
			}
		})
	}
}

func admissionProjectNoIntegration(t *testing.T) project.Project { return admissionProject(t) }

func admissionProjectAmbiguous(t *testing.T) project.Project {
	return admissionProject(t, workItemsIntegration("alpha"), workItemsIntegration("beta"))
}

func admissionProjectOtherID(t *testing.T) project.Project {
	t.Helper()
	p, issues := project.New(project.State{SchemaVersion: 2, ID: "123e4567-e89b-42d3-a456-426614174999", Slug: "sample", Name: "Sample"})
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	return p
}

func TestAdmissionStateFailuresAndSelection(t *testing.T) {
	for name, test := range map[string]struct {
		inspectErr error
		want       string
	}{
		"malformed":   {ErrUnsafe, OperationalStateInvalid},
		"interrupted": {ErrRecoveryRequired, OperationalRecoveryRequired},
	} {
		fixture := newAdmissionFixture(t, DefaultOperationalState())
		fixture.store.inspectErr = test.inspectErr
		if decision := fixture.guard().Admit(context.Background(), "sample", AdmitExecutionAdvance); decision.Allowed || decision.Code != test.want {
			t.Errorf("%s evolution: %+v", name, decision)
		}
		// Inspection and administration never consult local operational state.
		for _, operation := range []AdmissionOperation{AdmitExecutionStatus, AdmitWorkItemList, AdmitProjectEdit, AdmitIntegrationEnable} {
			if decision := fixture.guard().Admit(context.Background(), "sample", operation); !decision.Allowed {
				t.Errorf("%s %s: %+v", name, operation, decision)
			}
		}
	}
	fixture := newAdmissionFixture(t, OperationalState{ProjectStatus: ProjectArchived, DisabledIntegrations: []string{}})
	fixture.selection.category = "project_ambiguous"
	if decision := fixture.guard().Admit(context.Background(), "sample", AdmitWorkItemCreate); !decision.Allowed || decision.Deferred != "project_ambiguous" || len(fixture.store.commits) != 0 {
		t.Fatalf("selection failure must defer to the operation's own exact selection: %+v", decision)
	}
	if decision := (AdmissionGuard{}).Admit(context.Background(), "sample", AdmissionOperation("execution.cancel")); decision.Allowed || decision.Code != AdmissionOperationUnclassified {
		t.Fatalf("unclassified operation admitted: %+v", decision)
	}
	if decision := (AdmissionGuard{}).Admit(context.Background(), "sample", AdmitWorkItemCreate); decision.Allowed {
		t.Fatalf("uncomposed guard admitted evolution: %+v", decision)
	}
}
