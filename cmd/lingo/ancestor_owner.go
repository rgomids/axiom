package main

import (
	"fmt"

	"github.com/rgomids/axiom/internal/cli"
	"github.com/rgomids/axiom/internal/completion"
	"github.com/rgomids/axiom/internal/local"
	"github.com/rgomids/axiom/internal/provenance"
)

func ancestorOwnerFailure(category string, source provenance.Value) (cli.Result, bool) {
	refusal, ok := local.ParseAncestorOwnerCategory(category)
	if !ok {
		return cli.Result{}, false
	}
	message := fmt.Sprintf("Local storage ancestor owner is untrusted: %s, uid=%d", refusal.PathClass, refusal.UID)
	if refusal.UIDMapping == "unmapped" {
		message += "; user namespace has unmapped owners (host owner identity is unknown)"
	}
	result := canonicalCompletion(completion.Facts{ValidationFailed: true}, message, nil, "Preserve state; see troubleshooting.md: Untrusted ancestor owner in the installed operational documentation. Compatibility requires a reviewed isolation-preserving solution", source)
	result.Category = "untrusted_ancestor_owner"
	result.Diagnostic = &cli.Diagnostic{
		SchemaVersion: 1, Category: result.Category,
		Target:           map[string]string{"pathClass": refusal.PathClass},
		Observed:         []string{fmt.Sprintf("owner_uid=%d", refusal.UID), "uid_mapping=" + refusal.UIDMapping},
		NotObserved:      []string{"Host owner identity", "Local state integrity"},
		Confinement:      "Ancestor trust check refused access",
		ConfirmedEffects: []string{}, UncertainEffects: []string{},
	}
	return result, true
}
