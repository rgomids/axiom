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
	"github.com/rgomids/axiom/internal/project"
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

// CR-392-F1: a foreign-owned ancestor of the portable source reaches the
// project list command as the bounded refusal, never as invalid state.
func TestProjectListPortableSourceAncestorOwner(t *testing.T) {
	refusal := &local.AncestorOwnerError{PathClass: "ancestor", UID: 65534, UIDMapping: "unmapped"}
	for _, tc := range []struct {
		err             error
		wanted, refused []string
	}{
		{refusal, []string{"untrusted_ancestor_owner", "65534", "pathClass", "unmapped", "troubleshooting.md"}, []string{"Repair", "reconfigure", "invalid", "canary-private"}},
		{projectapp.ErrUnsafe, []string{"Repair protected Project state"}, []string{"untrusted_ancestor_owner"}},
	} {
		service := lifecycleService{provenance: currentProvenance()}
		result := service.projectListResult(projectapp.NewProjectCatalog(ancestorInstalled{}, ancestorDefinitions{tc.err}).List(context.Background()))
		for _, args := range [][]string{{"project", "list"}, {"--json", "project", "list"}} {
			var out bytes.Buffer
			if code := cli.RunInteractive(context.Background(), args, projectListPresentation{result: result}, currentProvenance(), nil, &out, io.Discard); code == 0 {
				t.Fatalf("%v succeeded: %s", args, out.String())
			}
			for _, wanted := range tc.wanted {
				if !strings.Contains(out.String(), wanted) {
					t.Fatalf("%v missing %s: %s", args, wanted, out.String())
				}
			}
			for _, forbidden := range tc.refused {
				if strings.Contains(out.String(), forbidden) {
					t.Fatalf("%v leaked %s: %s", args, forbidden, out.String())
				}
			}
		}
	}
}

type ancestorInstalled struct{}

func (ancestorInstalled) ListInstalled(context.Context) ([]projectapp.InstalledProject, error) {
	return []projectapp.InstalledProject{{ID: "123e4567-e89b-42d3-a456-426614174000", Slug: "sample", Source: "/canary-private/source"}}, nil
}

type ancestorDefinitions struct{ err error }

func (d ancestorDefinitions) ReadInstalledProject(context.Context, projectapp.InstalledProject) (project.Project, error) {
	return project.Project{}, d.err
}

type projectListPresentation struct {
	cli.Service
	result cli.Result
}

func (s projectListPresentation) List(context.Context) cli.Result { return s.result }

// Exercise the real presentation path for each application result mapping.
type ancestorPresentation struct {
	cli.Service
	result cli.Result
}

func (s ancestorPresentation) Show(context.Context, cli.ResolveInput) cli.Result { return s.result }
