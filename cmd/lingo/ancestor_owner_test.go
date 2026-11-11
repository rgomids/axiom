package main

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/cli"
	"github.com/rgomids/axiom/internal/completion"
	"github.com/rgomids/axiom/internal/local"
	"github.com/rgomids/axiom/internal/projectapp"
)

func TestAncestorOwnerDiagnostic(t *testing.T) {
	for _, mapping := range []string{"unknown", "mapped", "unmapped"} {
		category := local.AncestorOwnerCategory(&local.AncestorOwnerError{PathClass: "filesystem_root", UID: 65534, UIDMapping: mapping})
		service := lifecycleService{provenance: currentProvenance()}
		direct, ok := ancestorOwnerFailure(category, service.provenance)
		if !ok {
			t.Fatal("missing diagnostic")
		}
		for _, result := range []cli.Result{
			direct, projectShowFailure(category, service.provenance), service.operationalSelectionFailure(category),
			service.admissionDenied(projectapp.AdmissionDecision{Code: category}),
			service.stageFailure(completion.ValidationFailure, category, "", "", nil).Result,
		} {
			for _, jsonMode := range []bool{false, true} {
				args := []string{"project", "show", "--selector", "synthetic"}
				if jsonMode {
					args = append([]string{"--json"}, args...)
				}
				var text bytes.Buffer
				code := cli.RunInteractive(context.Background(), args, ancestorPresentation{result: result}, currentProvenance(), nil, &text, io.Discard)
				if code == 0 {
					t.Fatal("refusal succeeded")
				}
				output := text.String()
				for _, wanted := range []string{"65534", "filesystem_root", "untrusted_ancestor_owner", "troubleshooting.md"} {
					if !strings.Contains(output, wanted) {
						t.Fatalf("missing %s: %s", wanted, output)
					}
				}
				for _, forbidden := range []string{"Repair", "reconfigure", "invalid_existing_local_state", "configuration_unavailable"} {
					if strings.Contains(output, forbidden) {
						t.Fatalf("misleading %s: %s", forbidden, output)
					}
				}
				if strings.Contains(output, "user namespace has unmapped owners") != (mapping == "unmapped") {
					t.Fatal(output)
				}
			}
		}
	}
}

// Exercise the real presentation path for each application result mapping.
type ancestorPresentation struct {
	cli.Service
	result cli.Result
}

func (s ancestorPresentation) Show(context.Context, cli.ResolveInput) cli.Result { return s.result }
