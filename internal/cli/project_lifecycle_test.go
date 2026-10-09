package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/completion"
	"github.com/rgomids/axiom/internal/projectapp"
)

// lifecycleRecorder adds the optional #230 Project lifecycle service to the
// shared recording service.
type lifecycleRecorder struct {
	recordingService
}

func (s *lifecycleRecorder) ProjectArchive(_ context.Context, input ProjectLifecycleInput) Result {
	s.call = "archive:" + input.Project + ":" + input.PreviewDigest + ":" + boolText(input.AuthorizeLocal)
	return Result{Status: Succeeded, Category: "applied"}
}

func (s *lifecycleRecorder) ProjectReactivate(_ context.Context, input ProjectLifecycleInput) Result {
	s.call = "reactivate:" + input.Project + ":" + input.PreviewDigest + ":" + boolText(input.AuthorizeLocal)
	return Result{Status: Succeeded, Category: "applied"}
}

func (s *lifecycleRecorder) ProjectListFiltered(_ context.Context, input ProjectListInput) Result {
	s.call = "list:" + boolText(input.IncludeArchived)
	return Result{Status: Succeeded, Category: "applied", Projects: []ProjectListView{}}
}

func (s *lifecycleRecorder) ProjectValidateInstalled(_ context.Context, input ProjectLifecycleInput) Result {
	s.call = "validate:" + input.Project
	return Result{Status: Succeeded, Category: "applied"}
}

func boolText(value bool) string {
	if value {
		return "true"
	}
	return "false"
}

func TestProjectLifecycleParserDispatchesExactInputs(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{"archive preview", []string{"project", "archive", "--project", "alpha"}, "archive:alpha::false"},
		{"archive authorized", []string{"project", "archive", "--project", "alpha", "--preview-digest", "abc", "--authorize-local"}, "archive:alpha:abc:true"},
		{"archive digest without authority", []string{"project", "archive", "--project", "alpha", "--preview-digest", "abc"}, "archive:alpha:abc:false"},
		{"reactivate authorized", []string{"project", "reactivate", "--project=beta", "--preview-digest=abc", "--authorize-local"}, "reactivate:beta:abc:true"},
		{"list include archived", []string{"project", "list", "--include-archived"}, "list:true"},
		{"list explicit false", []string{"project", "list", "--include-archived=false"}, "list:false"},
		{"validate by project", []string{"project", "validate", "--project", "alpha"}, "validate:alpha"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service := &lifecycleRecorder{}
			var output bytes.Buffer
			RunInteractive(context.Background(), append([]string{"--json"}, test.args...), service, completionProvenance(t), nil, &output, io.Discard)
			if service.call != test.want {
				t.Fatalf("call = %q, want %q; output=%s", service.call, test.want, output.String())
			}
		})
	}
}

func TestProjectLifecycleParserLeavesEstablishedFormsToTheEstablishedParser(t *testing.T) {
	service := &lifecycleRecorder{}
	for args, want := range map[string]string{
		"project list":                  "list",
		"project validate --slug alpha": "validate:alpha",
		"project show --selector alpha": "resolve:alpha",
	} {
		service.call = ""
		RunInteractive(context.Background(), append([]string{"--json"}, strings.Fields(args)...), service, completionProvenance(t), nil, io.Discard, io.Discard)
		if service.call != want {
			t.Fatalf("%s: call = %q, want %q", args, service.call, want)
		}
	}
}

func TestProjectLifecycleParserRejectsWithoutEchoOrApplicationCall(t *testing.T) {
	const sentinel = "do-not-render-this-value"
	tests := []struct {
		name       string
		args       []string
		wantResult string
	}{
		{"archive unknown flag", []string{"project", "archive", "--project", "alpha", "--unknown", sentinel}, "Project archive input is invalid"},
		{"archive duplicate project", []string{"project", "archive", "--project", "alpha", "--project", sentinel}, "Project archive input is invalid"},
		{"archive duplicate authority", []string{"project", "archive", "--project", "alpha", "--authorize-local", "--authorize-local"}, "Project archive input is invalid"},
		{"archive positional", []string{"project", "archive", "--project", "alpha", sentinel}, "Project archive input is invalid"},
		{"archive single hyphen", []string{"project", "archive", "-project", sentinel}, "Project archive input is invalid"},
		{"archive separator", []string{"project", "archive", "--", sentinel}, "Project archive input is invalid"},
		{"archive missing selector", []string{"project", "archive"}, "Project archive input is incomplete"},
		{"archive external authority", []string{"project", "archive", "--project", "alpha", "--authorize-external"}, "Project archive input is invalid"},
		{"reactivate missing selector", []string{"project", "reactivate", "--preview-digest", sentinel}, "Project reactivation input is incomplete"},
		{"reactivate slug flag", []string{"project", "reactivate", "--slug", sentinel}, "Project reactivation input is invalid"},
		{"list unknown flag", []string{"project", "list", "--unknown", sentinel}, "Project listing input is invalid"},
		{"list duplicate", []string{"project", "list", "--include-archived", "--include-archived"}, "Project listing input is invalid"},
		{"list positional", []string{"project", "list", sentinel}, "Project listing input is invalid"},
		{"validate both selectors", []string{"project", "validate", "--slug", "alpha", "--project", sentinel}, "Project validation input is invalid"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service := &lifecycleRecorder{}
			for _, mode := range []string{"--json", "--human"} {
				var output bytes.Buffer
				code := RunInteractive(context.Background(), append([]string{mode}, test.args...), service, completionProvenance(t), nil, &output, io.Discard)
				if code != ExitFailure || service.call != "" {
					t.Fatalf("%s: exit=%d call=%q output=%s", mode, code, service.call, output.String())
				}
				if strings.Contains(output.String(), sentinel) {
					t.Fatalf("%s exposed rejected input: %s", mode, output.String())
				}
				if mode == "--json" {
					var event completionEvent
					if err := json.Unmarshal(output.Bytes(), &event); err != nil || event.Status != completion.ValidationFailure || event.Result != test.wantResult {
						t.Fatalf("event = %+v, %v; output=%s", event, err, output.String())
					}
				}
			}
		})
	}
}

func TestProjectLifecycleWithoutOptionalServiceIsUnavailable(t *testing.T) {
	for _, args := range [][]string{
		{"project", "archive", "--project", "alpha"},
		{"project", "reactivate", "--project", "alpha"},
		{"project", "list", "--include-archived"},
		{"project", "validate", "--project", "alpha"},
	} {
		service := &recordingService{}
		var output bytes.Buffer
		if code := Run(context.Background(), args, service, completionProvenance(t), &output); code != ExitFailure || service.call != "" || !strings.Contains(output.String(), `"category":"application_unavailable"`) {
			t.Fatalf("%v: exit=%d call=%q output=%s", args, code, service.call, output.String())
		}
	}
}

func TestProjectLifecycleHumanRendersStatusAndAvailability(t *testing.T) {
	service := &renderingService{t: t}
	var list bytes.Buffer
	if code := RunInteractive(context.Background(), []string{"--human", "project", "list", "--include-archived"}, service, completionProvenance(t), nil, &list, io.Discard); code != ExitSuccess {
		t.Fatalf("list exit=%d output=%s", code, list.String())
	}
	if !strings.Contains(list.String(), "id `123e4567-e89b-42d3-a456-426614174000` · slug `alpha` · name `Alpha` · status `archived`") {
		t.Fatalf("list human = %s", list.String())
	}
	var show bytes.Buffer
	if code := RunInteractive(context.Background(), []string{"--human", "project", "show", "--selector", "alpha"}, service, completionProvenance(t), nil, &show, io.Discard); code != ExitSuccess {
		t.Fatalf("show exit=%d output=%s", code, show.String())
	}
	for _, expected := range []string{"- **status:** `archived`", "- **disabledIntegrations:**\n      - `work-items`", "key `main` · path `/tmp/alpha` · availability `unavailable`"} {
		if !strings.Contains(show.String(), expected) {
			t.Fatalf("%q absent from %s", expected, show.String())
		}
	}
}

type renderingService struct {
	lifecycleRecorder
	t *testing.T
}

func (s *renderingService) ProjectListFiltered(context.Context, ProjectListInput) Result {
	canonical := canonicalResult(s.t, completion.Success, nil, "", completionProvenance(s.t))
	return Result{Completion: &canonical, Projects: []ProjectListView{{ID: "123e4567-e89b-42d3-a456-426614174000", Slug: "alpha", Name: "Alpha", Status: "archived"}}}
}

func (s *renderingService) Show(context.Context, ResolveInput) Result {
	canonical := canonicalResult(s.t, completion.Success, nil, "", completionProvenance(s.t))
	view := projectapp.ViewOperational(projectapp.OperationalState{ProjectStatus: projectapp.ProjectArchived, DisabledIntegrations: []string{"work-items"}})
	return Result{Completion: &canonical, Project: &ProjectView{ID: "123e4567-e89b-42d3-a456-426614174000", Slug: "alpha", Repositories: []RepositoryView{{Key: "main", Path: "/tmp/alpha", Availability: "unavailable"}}, State: &ProjectStateView{Status: "archived", Operational: &view}}}
}
