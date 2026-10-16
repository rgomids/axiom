package cli

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/completion"
)

type capabilityRecorder struct {
	stageRecorder
	requests []RuntimeCapabilityInput
}

func (s *capabilityRecorder) RuntimeCapability(ctx context.Context, in RuntimeCapabilityInput) RuntimeCapabilityResponse {
	s.requests = append(s.requests, in)
	r := s.WorkflowStage(ctx, WorkflowStageInput{Operation: in.Operation})
	return RuntimeCapabilityResponse{Result: r.Result, Payload: map[string]any{"operation": in.Operation}}
}

func TestRuntimeCapabilityRouting(t *testing.T) {
	for _, runtime := range []string{"codex", "claude"} {
		for _, op := range []string{"show", "invalidate"} {
			args := strings.Fields("runtime " + runtime + " capability " + op + " --profile p --variant repository-write --process --repository-root /repo --workspace-root /work --authorize-local")
			if op == "invalidate" {
				args = append(args, "--preview-digest", "reviewed")
			}
			service := &capabilityRecorder{stageRecorder: stageRecorder{status: completion.Facts{Completed: true}, source: completionProvenance(t)}}
			var jsonOut, humanOut bytes.Buffer
			if code := Run(context.Background(), args, service, completionProvenance(t), &jsonOut); code != ExitSuccess {
				t.Fatal(jsonOut.String())
			}
			if code := RunInteractive(context.Background(), args, service, completionProvenance(t), nil, &humanOut, io.Discard); code != ExitSuccess {
				t.Fatal(humanOut.String())
			}
			if len(service.requests) != 2 {
				t.Fatal(service.requests)
			}
			in := service.requests[0]
			if in.RuntimeID != runtime || in.Operation != op || in.ProfileID != "p" || in.Variant != "repository-write" || !in.Process || !in.AuthorizeLocal || in.RepositoryRoot != "/repo" || in.WorkspaceRoot != "/work" {
				t.Fatalf("wrong input: %+v", in)
			}
			for _, word := range []string{"capability", "operation", op} {
				if !strings.Contains(jsonOut.String(), word) || !strings.Contains(humanOut.String(), word) {
					t.Fatalf("parity: json=%s human=%s", &jsonOut, &humanOut)
				}
			}
			var help bytes.Buffer
			if code := Run(context.Background(), append(args[:4:4], "--help"), nil, completionProvenance(t), &help); code != ExitSuccess || !strings.Contains(help.String(), "--variant") {
				t.Fatalf("help: %d %s", code, &help)
			}
		}
	}
}

func TestRuntimeCapabilityParserRefusals(t *testing.T) {
	for _, suffix := range []string{
		"show --variant read-only", "show --profile p", "show --profile p --variant other",
		"show --profile p --variant read-only --authorize-external",
		"show --profile p --variant read-only --preview-digest d",
		"invalidate --profile p --profile q --variant read-only",
		"invalidate --profile p --variant read-only --authorize-local=false --authorize-local",
		"invalidate --profile p --variant read-only trailing",
		"probe --profile p --variant read-only",
		"probe --profile p --variant read-only --probe-id p --manifest-digest invalid",
		"prepare --profile p --variant read-only --probe-id p --authorize-external",
		"probe --profile p --variant read-only --probe-id p --manifest-digest " + strings.Repeat("a", 64) + " --authorize-prepare",
		"probe-extended --profile p --variant read-only --probe-id p --manifest-digest " + strings.Repeat("a", 64),
		"probe --profile p --variant read-only --probe-id p --manifest-digest " + strings.Repeat("a", 64) + " --authorize-extended",
	} {
		service := &capabilityRecorder{}
		var out bytes.Buffer
		code := Run(context.Background(), strings.Fields("runtime codex capability "+suffix), service, completionProvenance(t), &out)
		if code == ExitSuccess || len(service.requests) != 0 {
			t.Fatalf("accepted %q: %d %s", suffix, code, &out)
		}
	}
}

func TestCapabilityProbeHelpAndPreparationAuthority(t *testing.T) {
	for _, runtime := range []string{"codex", "claude"} {
		for _, op := range []string{"prepare", "probe", "probe-extended"} {
			var out bytes.Buffer
			args := []string{"runtime", runtime, "capability", op, "--help"}
			if code := Run(context.Background(), args, nil, completionProvenance(t), &out); code != ExitSuccess {
				t.Fatal(out.String())
			}
			want, absent := "--authorize-prepare", "--authorize-external"
			if op == "probe" {
				want, absent = absent, want
			}
			if op == "probe-extended" {
				want, absent = "--authorize-extended", "--authorize-prepare"
			}
			if !strings.Contains(out.String(), want) || strings.Contains(out.String(), absent) {
				t.Fatalf("authority conflated: %s", &out)
			}
		}
	}
}

func TestCapabilityExcludedFromRuntimeSkill(t *testing.T) {
	for _, op := range skillOperations("axiom-runtime") {
		if capabilityOperation(op) {
			t.Fatalf("capability exposed through runtime skill: %s", op)
		}
	}
}
