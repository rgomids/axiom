package main

import (
	"github.com/rgomids/axiom/internal/local"
	"github.com/rgomids/axiom/internal/runtimeapplication"
)

// runtimeProfileValidityV3 is the isolated format-3 profile validation
// evaluator (Specification 007 AXM-22 BLD-001B.3c-2b.2) over one
// local.RuntimeProfileStore.InspectVersioned result. It returns "" only for an
// active, valid format-3 record (runtimeprofile.ValidateV3 through
// NewConsumerV3); v1/v2, invalid, missing, retired, unsafe and
// recovery-required states are refused with their established codes. A stored
// EffortDeclaration is never read as current executable support. Runtime
// profile validate keeps its current behavior until coordinated v3-only
// activation.
func runtimeProfileValidityV3(snapshot local.RuntimeProfileSnapshot, err error) string {
	if err != nil {
		if snapshot.State == "pending" {
			return "configuration_recovery_required"
		}
		return "configuration_unsafe"
	}
	_, code := runtimeapplication.ConsumerFromRecordV3(snapshot.State, snapshot.V3)
	return code
}
