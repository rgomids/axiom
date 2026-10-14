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

// Specification 007 BI-3.1: exactly two flags, only on binding migrate/edit;
// every other command keeps its existing flag surface.
func TestRuntimeDeclarationFlagSurface(t *testing.T) {
	accepted := map[string][]string{
		"migrate": {"runtime", "binding", "migrate", "--runtime", "codex", "--target-format", "3", "--effort-declaration", "declaration.json"},
		"edit":    {"runtime", "binding", "edit", "--runtime=codex", "--effort-declaration=declaration.json", "--expected-revision", "token", "--preview-digest", "digest", "--authorize-local"},
	}
	for name, args := range accepted {
		probe := &declarationProbe{}
		var out bytes.Buffer
		Run(context.Background(), args, probe, completionProvenance(t), &out)
		if len(probe.got) != 1 {
			t.Fatalf("%s not parsed: %s", name, out.String())
		}
		r := probe.got[0]
		if r.Kind != "binding" || r.RuntimeID != "codex" || r.EffortDeclaration != "declaration.json" {
			t.Fatalf("%s: %+v", name, r)
		}
		if name == "migrate" && r.TargetFormat != 3 || name == "edit" && (r.TargetFormat != 0 || !r.AuthorizeLocal || r.ExpectedRevision != "token" || r.PreviewDigest != "digest") {
			t.Fatalf("%s: %+v", name, r)
		}
	}
	refused := map[string][]string{
		"profile-migrate":    {"runtime", "profile", "migrate", "--runtime", "codex", "--effort-declaration", "declaration.json"},
		"profile-edit":       {"runtime", "profile", "edit", "--profile", "codex-high", "--effort-declaration", "declaration.json"},
		"binding-create":     {"runtime", "binding", "create", "--runtime", "codex", "--effort-declaration", "declaration.json"},
		"binding-show":       {"runtime", "binding", "show", "--runtime", "codex", "--effort-declaration", "declaration.json"},
		"edit-target-format": {"runtime", "binding", "edit", "--runtime", "codex", "--target-format", "3"},
		"profile-target":     {"runtime", "profile", "migrate", "--target-format", "3"},
		"non-integer":        {"runtime", "binding", "migrate", "--target-format", "three"},
		"duplicate-target":   {"runtime", "binding", "migrate", "--target-format", "3", "--target-format", "3"},
		"duplicate-file":     {"runtime", "binding", "edit", "--runtime", "codex", "--effort-declaration", "a.json", "--effort-declaration", "b.json"},
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
