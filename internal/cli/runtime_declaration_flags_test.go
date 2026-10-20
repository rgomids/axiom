package cli

import (
	"bytes"
	"context"
	"testing"

	"github.com/rgomids/axiom/internal/runtimeapplication"
)

// declarationProbe captures the parsed authoring request without effects.
type declarationProbe struct {
	authorityProbe
	got []runtimeapplication.AuthoringRequest
}

func (p *declarationProbe) RuntimeAuthoring(_ context.Context, r runtimeapplication.AuthoringRequest) Result {
	p.got = append(p.got, r)
	return Result{Status: Succeeded, Category: "preview"}
}

// AXM-36 D1: the retired declaration and migration flags are ordinary
// invalid input everywhere; nothing reaches the authoring service.
func TestRuntimeDeclarationFlagSurfaceRetired(t *testing.T) {
	refused := map[string][]string{
		"binding-edit-declaration":    {"runtime", "binding", "edit", "--runtime=codex", "--effort-declaration=declaration.json", "--expected-revision", "token", "--preview-digest", "digest", "--authorize-local"},
		"binding-migrate-declaration": {"runtime", "binding", "migrate", "--runtime", "codex", "--target-format", "3", "--effort-declaration", "declaration.json"},
		"binding-migrate-target":      {"runtime", "binding", "migrate", "--runtime", "codex", "--target-format", "3"},
		"migrate-declaration":         {"runtime", "binding", "migrate", "--runtime", "codex", "--effort-declaration", "declaration.json"},
		"profile-migrate":             {"runtime", "profile", "migrate", "--runtime", "codex", "--effort-declaration", "declaration.json"},
		"profile-edit":                {"runtime", "profile", "edit", "--profile", "codex-high", "--effort-declaration", "declaration.json"},
		"binding-create":              {"runtime", "binding", "create", "--runtime", "codex", "--effort-declaration", "declaration.json"},
		"binding-show":                {"runtime", "binding", "show", "--runtime", "codex", "--effort-declaration", "declaration.json"},
		"edit-target-format":          {"runtime", "binding", "edit", "--runtime", "codex", "--target-format", "4"},
		"profile-target":              {"runtime", "profile", "migrate", "--target-format", "4"},
	}
	for name, args := range refused {
		probe := &declarationProbe{}
		var out bytes.Buffer
		code := Run(context.Background(), args, probe, completionProvenance(t), &out)
		if len(probe.got) != 0 || code == ExitSuccess {
			t.Fatalf("%s accepted (%d): %s", name, code, out.String())
		}
	}
	probe := &declarationProbe{}
	var out bytes.Buffer
	Run(context.Background(), []string{"runtime", "binding", "edit", "--runtime=codex", "--enabled=false"}, probe, completionProvenance(t), &out)
	if len(probe.got) != 1 || probe.got[0].Kind != "binding" || probe.got[0].Operation != "edit" {
		t.Fatalf("ordinary binding edit lost: %s", out.String())
	}
}
