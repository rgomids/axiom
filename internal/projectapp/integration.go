package projectapp

import (
	"context"
	"sort"

	"github.com/rgomids/axiom/internal/project"
)

// Issue #230 Integration lifecycle (I230-T04). This file owns the Integration
// inventory, the static validation report and the Project/Integration
// resolution that disable and enable need. Everything here is read-only and
// static: no Provider or Transport call is made, no credential is resolved and
// no authority is granted. Disable/enable transitions belong to
// ApplyOperational; portable removal belongs to the EDIT engine.

// IntegrationNotFound is the stable category for a key that the Project does
// not declare (and, for enable, that is not a stale local disable either).
const IntegrationNotFound = "integration_not_found"

// Stable validation finding codes. Capability ambiguity and Provider support
// reuse the readiness codes so one capability fault has one name.
const (
	FindingIntegrationProviderMissing     = "integration_provider_missing"
	FindingIntegrationCredentialMissing   = "integration_credential_reference_missing"
	FindingIntegrationCapabilitiesMissing = "integration_capabilities_missing"
	FindingIntegrationDisabledLocally     = "integration_disabled_locally"
	FindingIntegrationStaleDisabled       = "integration_disabled_stale"
	FindingIntegrationsNotDeclared        = "integrations_not_declared"
)

// MaxIntegrationFindings bounds one validation report.
const MaxIntegrationFindings = 256

const (
	IntegrationEnabled  = "enabled"
	IntegrationDisabled = "disabled"

	FindingError   = "error"
	FindingWarning = "warning"

	IntegrationValid   = "valid"
	IntegrationInvalid = "invalid"
)

// IntegrationView is the safe projection of one declared Integration plus its
// local eligibility. It carries logical names only: the Transport kind (never
// its reference), the CredentialReference name (never a value or location).
type IntegrationView struct {
	Key           string   `json:"key"`
	ProviderRef   string   `json:"providerRef,omitempty"`
	Provider      string   `json:"provider,omitempty"`
	Capabilities  []string `json:"capabilities"`
	Transport     string   `json:"transport,omitempty"`
	CredentialRef string   `json:"credentialRef,omitempty"`
	Local         string   `json:"local"`
}

type IntegrationFinding struct {
	Code        string `json:"code"`
	Severity    string `json:"severity"`
	Integration string `json:"integration,omitempty"`
	Capability  string `json:"capability,omitempty"`
}

type IntegrationValidation struct {
	Status    string               `json:"status"`
	Findings  []IntegrationFinding `json:"findings"`
	Truncated bool                 `json:"truncated,omitempty"`
}

// IntegrationReport is the bounded result of list, show and validate.
// StaleDisabled names keys that are disabled on this machine but no longer
// declared; they are inert and cleared only by a local enable (F-04).
type IntegrationReport struct {
	ProjectID     string                 `json:"projectId"`
	Integrations  []IntegrationView      `json:"integrations"`
	StaleDisabled []string               `json:"staleDisabled,omitempty"`
	Validation    *IntegrationValidation `json:"validation,omitempty"`
}

func declaredIntegrations(state project.State) []project.Integration {
	declared, _ := state.Integrations.Value()
	result := append([]project.Integration(nil), declared...)
	sort.Slice(result, func(i, j int) bool { return result[i].Key < result[j].Key })
	return result
}

func providerIDs(state project.State) map[string]string {
	providers := map[string]string{}
	declared, _ := state.Providers.Value()
	for _, provider := range declared {
		providers[provider.Key] = provider.ID
	}
	return providers
}

func integrationView(integration project.Integration, providers map[string]string, operational OperationalState) IntegrationView {
	view := IntegrationView{Key: integration.Key, Capabilities: []string{}, Local: IntegrationEnabled}
	if operational.IntegrationDisabled(integration.Key) {
		view.Local = IntegrationDisabled
	}
	if ref, ok := integration.ProviderRef.Value(); ok {
		view.ProviderRef, view.Provider = ref, providers[ref]
	}
	if capabilities, ok := integration.Capabilities.Value(); ok {
		view.Capabilities = append(view.Capabilities, capabilities...)
	}
	if transport, ok := integration.Transport.Value(); ok {
		view.Transport = transport.ID
	}
	view.CredentialRef, _ = integration.CredentialRef.Value()
	return view
}

// StaleDisabledIntegrations returns the locally disabled keys the Project no
// longer declares, sorted.
func StaleDisabledIntegrations(state project.State, operational OperationalState) []string {
	declared := map[string]bool{}
	for _, integration := range declaredIntegrations(state) {
		declared[integration.Key] = true
	}
	stale := []string{}
	for _, key := range operational.DisabledIntegrations {
		if !declared[key] {
			stale = append(stale, key)
		}
	}
	return stale
}

// InventoryIntegrations is the pure list view: every declared Integration in
// key order with its local eligibility, plus inert stale local disables.
func InventoryIntegrations(state project.State, operational OperationalState) IntegrationReport {
	report := IntegrationReport{ProjectID: state.ID, Integrations: []IntegrationView{}, StaleDisabled: StaleDisabledIntegrations(state, operational)}
	providers := providerIDs(state)
	for _, integration := range declaredIntegrations(state) {
		report.Integrations = append(report.Integrations, integrationView(integration, providers, operational))
	}
	return report
}

// ShowIntegration is the pure single-key view; an undeclared key (including a
// stale local disable) is not found.
func ShowIntegration(state project.State, operational OperationalState, key string) (IntegrationReport, bool) {
	report := InventoryIntegrations(state, operational)
	for _, view := range report.Integrations {
		if view.Key == key {
			return IntegrationReport{ProjectID: state.ID, Integrations: []IntegrationView{view}}, true
		}
	}
	return IntegrationReport{}, false
}

// ValidateIntegrations is the static validation: declaration references
// resolve, the capability mapping is unambiguous, the Provider supports each
// declared capability in this build, and local eligibility is reported. It
// calls no Provider or Transport and resolves no credential. A non-empty key
// restricts the report to that declared Integration.
func ValidateIntegrations(state project.State, operational OperationalState, catalog ProviderCatalog, key string) IntegrationReport {
	report := InventoryIntegrations(state, operational)
	providers := providerIDs(state)
	credentials := map[string]bool{}
	declaredCredentials, _ := state.CredentialReferences.Value()
	for _, credential := range declaredCredentials {
		credentials[credential.Key] = true
	}
	findings := []IntegrationFinding{}
	add := func(code, severity, integration, capability string) {
		findings = append(findings, IntegrationFinding{Code: code, Severity: severity, Integration: integration, Capability: capability})
	}
	selected := []IntegrationView{}
	for _, integration := range declaredIntegrations(state) {
		if key != "" && integration.Key != key {
			continue
		}
		view := integrationView(integration, providers, operational)
		selected = append(selected, view)
		ref, hasRef := integration.ProviderRef.Value()
		if !hasRef || providers[ref] == "" {
			add(FindingIntegrationProviderMissing, FindingError, integration.Key, "")
		}
		if credential, ok := integration.CredentialRef.Value(); ok && !credentials[credential] {
			add(FindingIntegrationCredentialMissing, FindingError, integration.Key, "")
		}
		if len(view.Capabilities) == 0 {
			add(FindingIntegrationCapabilitiesMissing, FindingWarning, integration.Key, "")
		}
		if view.Provider != "" {
			for _, capability := range view.Capabilities {
				if !catalog.supports(view.Provider, capability) {
					add(BlockerProviderUnsupported, FindingError, integration.Key, capability)
				}
			}
		}
		if view.Local == IntegrationDisabled {
			add(FindingIntegrationDisabledLocally, FindingWarning, integration.Key, "")
		}
	}
	// Capability mapping is one algorithm (ResolveCapability); ambiguity is
	// attributed to every Integration that declares the contested capability.
	declaring := map[string][]string{}
	for _, integration := range declaredIntegrations(state) {
		declared, _ := integration.Capabilities.Value()
		for _, capability := range declared {
			if !containsText(declaring[capability], integration.Key) {
				declaring[capability] = append(declaring[capability], integration.Key)
			}
		}
	}
	capabilities := make([]string, 0, len(declaring))
	for capability := range declaring {
		capabilities = append(capabilities, capability)
	}
	sort.Strings(capabilities)
	for _, capability := range capabilities {
		if ResolveCapability(state, capability, catalog).Readiness != CapabilityAmbiguous {
			continue
		}
		for _, integration := range declaring[capability] {
			if key == "" || integration == key {
				add(BlockerCapabilityAmbiguous, FindingError, integration, capability)
			}
		}
	}
	if key == "" {
		for _, stale := range report.StaleDisabled {
			add(FindingIntegrationStaleDisabled, FindingWarning, stale, "")
		}
		if len(report.Integrations) == 0 {
			add(FindingIntegrationsNotDeclared, FindingWarning, "", "")
		}
	}
	sort.SliceStable(findings, func(i, j int) bool {
		if findings[i].Integration != findings[j].Integration {
			return findings[i].Integration < findings[j].Integration
		}
		return findings[i].Code+findings[i].Capability < findings[j].Code+findings[j].Capability
	})
	validation := &IntegrationValidation{Status: IntegrationValid, Findings: findings}
	for _, finding := range findings {
		if finding.Severity == FindingError {
			validation.Status = IntegrationInvalid
		}
	}
	if len(findings) > MaxIntegrationFindings {
		validation.Findings, validation.Truncated = findings[:MaxIntegrationFindings], true
	}
	report.Integrations, report.Validation = selected, validation
	if key != "" {
		report.StaleDisabled = nil
	}
	return report
}

// IntegrationInspector composes the read-only observations the Integration
// operations need. It reads machine-local operational state and the recorded
// portable intent the readiness adapter already confines; nothing else.
type IntegrationInspector struct {
	Selection   AdmissionSelector
	Projects    ReadinessProjects
	Operational OperationalStore
	Catalog     ProviderCatalog
}

// IntegrationSnapshot is one consistent observation of a Project's declared
// Integrations and local operational state.
type IntegrationSnapshot struct {
	ProjectID   string
	State       project.State
	Operational OperationalState
}

func (i IntegrationInspector) composed() bool {
	return i.Selection != nil && i.Projects != nil && i.Operational != nil
}

// Load selects the Project, observes local operational state and loads the
// recorded portable intent. The returned category is a Select category, a
// readiness structural blocker code or an operational category.
func (i IntegrationInspector) Load(ctx context.Context, selector string) (IntegrationSnapshot, string) {
	projectID, operational, category := i.observe(ctx, selector)
	if category != "" {
		return IntegrationSnapshot{}, category
	}
	return i.portable(ctx, selector, projectID, operational)
}

func (i IntegrationInspector) observe(ctx context.Context, selector string) (string, OperationalState, string) {
	if !i.composed() {
		return "", OperationalState{}, OperationalStorageFailure
	}
	projectID, category := i.Selection.SelectProjectID(ctx, selector)
	if category != "" {
		return "", OperationalState{}, category
	}
	observation, category := InspectOperational(ctx, i.Operational, projectID)
	if category != "" {
		return "", OperationalState{}, category
	}
	return projectID, observation.State, ""
}

func (i IntegrationInspector) portable(ctx context.Context, selector, projectID string, operational OperationalState) (IntegrationSnapshot, string) {
	subject, code := i.Projects.LoadReadiness(ctx, selector)
	if code != "" {
		return IntegrationSnapshot{}, code
	}
	state := subject.Project.State()
	if state.ID != projectID {
		return IntegrationSnapshot{}, BlockerProjectStateInvalid
	}
	return IntegrationSnapshot{ProjectID: projectID, State: state, Operational: operational}, ""
}

func (i IntegrationInspector) List(ctx context.Context, selector string) (IntegrationReport, string) {
	snapshot, category := i.Load(ctx, selector)
	if category != "" {
		return IntegrationReport{}, category
	}
	return InventoryIntegrations(snapshot.State, snapshot.Operational), ""
}

func (i IntegrationInspector) Show(ctx context.Context, selector, key string) (IntegrationReport, string) {
	if !ValidIntegrationKey(key) {
		return IntegrationReport{}, OperationalInvalidInput
	}
	snapshot, category := i.Load(ctx, selector)
	if category != "" {
		return IntegrationReport{}, category
	}
	report, found := ShowIntegration(snapshot.State, snapshot.Operational, key)
	if !found {
		return IntegrationReport{}, IntegrationNotFound
	}
	return report, ""
}

// Validate reports static validity; an empty key validates every declaration.
func (i IntegrationInspector) Validate(ctx context.Context, selector, key string) (IntegrationReport, string) {
	if key != "" && !ValidIntegrationKey(key) {
		return IntegrationReport{}, OperationalInvalidInput
	}
	snapshot, category := i.Load(ctx, selector)
	if category != "" {
		return IntegrationReport{}, category
	}
	if key != "" {
		if _, found := ShowIntegration(snapshot.State, snapshot.Operational, key); !found {
			return IntegrationReport{}, IntegrationNotFound
		}
	}
	return ValidateIntegrations(snapshot.State, snapshot.Operational, i.Catalog, key), ""
}

// Target resolves the exact Project for a local disable or enable and checks
// the key. Disable needs a declared key. Enable also accepts a key that is
// currently disabled locally, declared or stale, so a stale entry can always
// be cleared without reading portable intent.
func (i IntegrationInspector) Target(ctx context.Context, selector string, operation OperationalOperation, key string) (string, string) {
	if !ValidIntegrationKey(key) || operation != DisableIntegration && operation != EnableIntegration {
		return "", OperationalInvalidInput
	}
	projectID, operational, category := i.observe(ctx, selector)
	if category != "" {
		return "", category
	}
	if operation == EnableIntegration && operational.IntegrationDisabled(key) {
		return projectID, ""
	}
	snapshot, category := i.portable(ctx, selector, projectID, operational)
	if category != "" {
		return "", category
	}
	if _, found := ShowIntegration(snapshot.State, snapshot.Operational, key); !found {
		return "", IntegrationNotFound
	}
	return projectID, ""
}
