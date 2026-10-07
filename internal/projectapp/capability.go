package projectapp

import (
	"sort"

	"github.com/rgomids/axiom/internal/project"
)

// ProviderCatalog declares which Provider implementations support which
// capabilities in this build. It is composition data, not Project intent.
type ProviderCatalog map[string][]string

// SupportedProviders is the MVP implementation catalog: GitHub is the only
// Work Item Provider implementation. It is not a domain requirement.
var SupportedProviders = ProviderCatalog{"github": {WorkItemCapability}}

func (c ProviderCatalog) supports(providerID, capability string) bool {
	for _, supported := range c[providerID] {
		if supported == capability {
			return true
		}
	}
	return false
}

// CapabilityResolution is the deterministic capability -> Integration ->
// Provider mapping for one requested capability. It implies no Transport,
// performs no Provider call and grants no authority.
type CapabilityResolution struct {
	Capability    string              `json:"capability"`
	Integration   string              `json:"integration,omitempty"`
	Provider      string              `json:"provider,omitempty"`
	Readiness     CapabilityReadiness `json:"status"`
	CredentialRef string              `json:"-"`
}

// CapabilityAmbiguous reports more than one Integration declaring the
// capability; nothing is chosen by order.
const CapabilityAmbiguous CapabilityReadiness = "ambiguous"

// ResolveCapability is the single capability-mapping algorithm consumed by
// bootstrap preview, EDIT preview, readiness and Work Item preflight.
func ResolveCapability(state project.State, capability string, catalog ProviderCatalog) CapabilityResolution {
	result := CapabilityResolution{Capability: capability, Readiness: CapabilityMissing}
	providers := map[string]string{}
	declared, _ := state.Providers.Value()
	for _, provider := range declared {
		providers[provider.Key] = provider.ID
	}
	integrations, _ := state.Integrations.Value()
	matches := make([]project.Integration, 0, 1)
	for _, integration := range integrations {
		capabilities, ok := integration.Capabilities.Value()
		ref, mapped := integration.ProviderRef.Value()
		if ok && mapped && containsText(capabilities, capability) && providers[ref] != "" {
			matches = append(matches, integration)
		}
	}
	sort.Slice(matches, func(i, j int) bool { return matches[i].Key < matches[j].Key })
	// Compatibility: Lingo's own setup writes the conventional work-items
	// Integration; when several Integrations declare the capability, that
	// explicitly named one is the mapping (the pre-#231 rule). Order never
	// chooses; without the convention, multiplicity is ambiguous.
	if len(matches) > 1 {
		for _, match := range matches {
			if ref, _ := match.ProviderRef.Value(); match.Key == workItemsKey && ref == workItemsKey {
				matches = []project.Integration{match}
			}
		}
	}
	if len(matches) > 1 {
		result.Readiness = CapabilityAmbiguous
		return result
	}
	if len(matches) == 0 {
		return result
	}
	ref, _ := matches[0].ProviderRef.Value()
	result.Integration, result.Provider = matches[0].Key, providers[ref]
	result.CredentialRef, _ = matches[0].CredentialRef.Value()
	result.Readiness = CapabilityUnsupported
	if catalog.supports(result.Provider, capability) {
		result.Readiness = CapabilityReady
	}
	return result
}
