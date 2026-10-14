package main

import (
	"github.com/rgomids/axiom/internal/capabilityproof"
	"github.com/rgomids/axiom/internal/local"
)

// productionCapabilityService exposes local evidence storage without claiming
// operational proof. runtimeadapter.BindingSource is not composed until local
// format-3 configuration can be authored and read (AXM-22 BLD-001B.3); the
// canonical service blocks checks while Source is absent. Probe
// dependencies are deliberately unconfigured, so composition launches no effects.
func productionCapabilityService(stateRoot string) (capabilityproof.Service, error) {
	store, err := local.NewCapabilityStore(stateRoot)
	if err != nil {
		return capabilityproof.Service{}, err
	}
	return capabilityproof.Service{Store: store}, nil
}
