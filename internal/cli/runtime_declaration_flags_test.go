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

// Specification 007 BI-3.1/PM-1: exactly one declaration flag, only on
// binding edit; no format-transition flag exists.
func TestRuntimeDeclarationFlagSurface(t *testing.T) {
	probe := &declarationProbe{}
	var out bytes.Buffer
	Run(context.Background(), []string{"runtime", "binding", "edit", "--runtime=codex", "--effort-declaration=declaration.json", "--expected-revision", "token", "--preview-digest", "digest", "--authorize-local"}, probe, completionProvenance(t), &out)
	if len(probe.got) != 1 {
		t.Fatalf("edit not parsed: %s", out.String())
	}
	if r := probe.got[0]; r.Kind != "binding" || r.Operation != "edit" || r.RuntimeID != "codex" || r.EffortDeclaration != "declaration.json" || !r.AuthorizeLocal || r.ExpectedRevision != "token" || r.PreviewDigest != "digest" {
		t.Fatalf("edit: %+v", r)
	}
	refused := map[string][]string{
		"binding-migrate-declaration": {"runtime", "binding", "migrate", "--runtime", "codex", "--target-format", "3", "--effort-declaration", "declaration.json"},
		"binding-migrate-target":      {"runtime", "binding", "migrate", "--runtime", "codex", "--target-format", "3"},
		"migrate-declaration":         {"runtime", "binding", "migrate", "--runtime", "codex", "--effort-declaration", "declaration.json"},
		"profile-migrate":             {"runtime", "profile", "migrate", "--runtime", "codex", "--effort-declaration", "declaration.json"},
		"profile-edit":                {"runtime", "profile", "edit", "--profile", "codex-high", "--effort-declaration", "declaration.json"},
		"binding-create":              {"runtime", "binding", "create", "--runtime", "codex", "--effort-declaration", "declaration.json"},
		"binding-show":                {"runtime", "binding", "show", "--runtime", "codex", "--effort-declaration", "declaration.json"},
		"edit-target-format":          {"runtime", "binding", "edit", "--runtime", "codex", "--target-format", "3"},
		"profile-target":              {"runtime", "profile", "migrate", "--target-format", "3"},
		"duplicate-file":              {"runtime", "binding", "edit", "--runtime", "codex", "--effort-declaration", "a.json", "--effort-declaration", "b.json"},
	}
	for name, args := range refused {
		probe := &declarationProbe{}
		var out bytes.Buffer
		code := Run(context.Background(), args, probe, completionProvenance(t), &out)
		if len(probe.got) != 0 || code == ExitSuccess {
			t.Fatalf("%s accepted (%d): %s", name, code, out.String())
		}
	}
}
