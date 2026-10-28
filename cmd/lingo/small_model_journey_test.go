package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/cli"
	"github.com/rgomids/axiom/internal/workitem"
)

// Deterministic replay of the Go VSM incident (Axiom 0.22.3): a Runtime built a
// Work Item create from the skill alone, used unsupported flags, omitted a
// section, then hit an undiagnosed `failure`. Every refusal must now carry a
// stable code in `details`, the scoped inspection must expose only the create
// contract, and the reviewed preview/digest/consent/read-back path must stay
// intact with exactly one Provider create. This proves CLI recoverability only;
// it is not evidence that a smaller model completes the journey.

type journeyEvent struct {
	Status    string                 `json:"status"`
	Result    string                 `json:"result"`
	Next      string                 `json:"next"`
	Details   string                 `json:"details"`
	Draft     *workitem.DraftPreview `json:"draft"`
	Questions []workitem.Question    `json:"questions"`
	WorkItem  *cli.WorkItemView      `json:"workItem"`
	WorkItems []cli.WorkItemView     `json:"workItems"`
	Projects  []cli.ProjectListView  `json:"projects"`
	Skill     *struct {
		Operations []struct {
			Name string `json:"name"`
		} `json:"operations"`
		Commands []struct {
			Command   string `json:"command"`
			Arguments []struct {
				Name         string `json:"name"`
				Required     bool   `json:"required"`
				RequiredWhen string `json:"requiredWhen"`
			} `json:"arguments"`
		} `json:"commands"`
	} `json:"skill"`
}

func TestGoVSMIncidentJourneyIsRecoverableWithoutCorrectiveIntervention(t *testing.T) {
	root, state, repository := filepath.Join(t.TempDir(), "projects"), filepath.Join(t.TempDir(), "state"), t.TempDir()
	t.Setenv("LINGO_PROJECTS_ROOT", root)
	t.Setenv("LINGO_STATE_ROOT", state)
	providerState := filepath.Join(t.TempDir(), "provider.json")
	if err := os.WriteFile(providerState, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	gh, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("AXIOM_GH_BIN", gh)
	t.Setenv(fakeGitHubStateVariable, providerState)
	service := compose()
	configureProject(t, service, "govsm", "Go VSM", "core="+repository, "github")
	provider := func() fakeGitHub {
		t.Helper()
		wire, err := os.ReadFile(providerState)
		if err != nil {
			t.Fatal(err)
		}
		var value fakeGitHub
		if err := json.Unmarshal(wire, &value); err != nil {
			t.Fatal(err)
		}
		return value
	}
	run := func(wantCode int, args ...string) journeyEvent {
		t.Helper()
		var output bytes.Buffer
		code := cli.Run(context.Background(), args, service, currentProvenance(), &output)
		var event journeyEvent
		if err := json.Unmarshal(output.Bytes(), &event); err != nil || code != wantCode {
			t.Fatalf("%v: code=%d err=%v output=%s", args, code, err, output.String())
		}
		return event
	}

	// Discovery: representative read-only operations need no authority.
	if listed := run(cli.ExitSuccess, "project", "list"); len(listed.Projects) != 1 || listed.Projects[0].Slug != "govsm" {
		t.Fatalf("project list = %+v", listed)
	}
	if validated := run(cli.ExitSuccess, "project", "validate", "--project", "govsm"); validated.Status != "success" {
		t.Fatalf("project validate = %+v", validated)
	}

	// Operation-scoped inspection returns only the selected contract.
	var whole, scoped bytes.Buffer
	if handled, code := cli.InspectSkill([]string{"--json", "skill", "inspect", "axiom-work-item"}, currentProvenance(), &whole); !handled || code != cli.ExitSuccess {
		t.Fatalf("inspect = %d", code)
	}
	if handled, code := cli.InspectSkill([]string{"--json", "skill", "inspect", "axiom-work-item", "--operation", "create"}, currentProvenance(), &scoped); !handled || code != cli.ExitSuccess {
		t.Fatalf("scoped inspect = %d %s", code, scoped.String())
	}
	if scoped.Len()*5 > whole.Len() {
		t.Fatalf("scoped inspection is not materially smaller: %d of %d bytes", scoped.Len(), whole.Len())
	}
	var inspected journeyEvent
	if err := json.Unmarshal(scoped.Bytes(), &inspected); err != nil || inspected.Skill == nil || len(inspected.Skill.Operations) != 1 || inspected.Skill.Operations[0].Name != "create" {
		t.Fatalf("scoped inspection = %s", scoped.String())
	}
	arguments := map[string]string{}
	for _, command := range inspected.Skill.Commands {
		if command.Command != "axiom work-item create" && command.Command != "axiom work-item select" {
			t.Fatalf("scoped inspection leaked %s", command.Command)
		}
		if command.Command != "axiom work-item create" {
			continue
		}
		for _, argument := range command.Arguments {
			arguments[argument.Name] = argument.RequiredWhen
		}
	}
	if _, ok := arguments["--title"]; ok {
		t.Fatal("create advertises --title")
	}
	if !strings.Contains(arguments["--desired-outcome"], "--elaborated-section desired_outcome=") || !strings.Contains(arguments["--problem"], "--intent") {
		t.Fatalf("section conditions = %+v", arguments)
	}

	base := []string{"work-item", "create", "--project", "govsm", "--repository", "core", "--provider-repository", "owner/repo"}
	with := func(extra ...string) []string { return append(append([]string(nil), base...), extra...) }
	before := snapshotTrees(t, root, state)

	// Attempt 1: unsupported flags are named, not a generic selector failure.
	if first := run(cli.ExitFailure, with("--title", "Update README", "--body", "x")...); first.Status != "validation_failure" || first.Details != "unknown_input:--title" {
		t.Fatalf("attempt 1 = %+v", first)
	}
	// A missing target selector is named too.
	if missing := run(cli.ExitFailure, "work-item", "create", "--project", "govsm", "--repository", "core", "--intent", "x"); missing.Details != "missing_required_input:--provider-repository" {
		t.Fatalf("missing target = %+v", missing)
	}
	// Attempt 2: missing sections come back as questions with a stable code.
	second := run(cli.ExitFailure, with("--type", "task", "--intent", "Update the README to Go open source conventions")...)
	if second.Details != "draft_incomplete" || len(second.Questions) != 6 || second.Questions[0].Field != "desired_outcome" {
		t.Fatalf("attempt 2 = %+v", second)
	}
	// Supplying one section both verbatim and elaborated is a named conflict.
	if conflict := run(cli.ExitFailure, with("--problem", "P", "--elaborated-section", "problem=Q")...); conflict.Details != "invalid_draft_input" || conflict.Result == "Work Item operation did not complete" {
		t.Fatalf("section conflict = %+v", conflict)
	}
	// Provider failure is a distinct code with an actionable next step.
	unreachable := run(cli.ExitFailure, "work-item", "create", "--project", "govsm", "--repository", "core", "--provider-repository", "RGomides/GOVSM", "--intent", "P", "--desired-outcome", "D", "--context", "C", "--scope", "S", "--constraints", "K", "--non-goals", "N", "--acceptance", "A")
	if !strings.HasPrefix(unreachable.Details, "provider_") || unreachable.Result == "Work Item operation did not complete" || unreachable.Next == "" {
		t.Fatalf("provider failure = %+v", unreachable)
	}
	if after := snapshotTrees(t, root, state); !bytes.Equal(before, after) || len(provider().Mutations) != 0 {
		t.Fatal("refused attempts changed state or reached a Provider mutation")
	}

	// Attempt 3: --intent with seven elaborated sections previews; the
	// confirmed identifiers survive unchanged into the reviewed target.
	sections := []string{"--type", "task", "--intent", "Update the README to Go open source conventions"}
	for _, section := range []string{"problem=README lacks Go conventions", "desired_outcome=README follows Go open source conventions", "context=Public Go module", "scope=README only", "constraints=No code changes", "non_goals=No new docs site", "acceptance_expectations=README has install, usage, contributing and license sections"} {
		sections = append(sections, "--elaborated-section", section)
	}
	preview := run(cli.ExitSuccess, with(sections...)...)
	if preview.Draft == nil || preview.Draft.Digest == "" || preview.Draft.Target.RepositoryKey != "core" || preview.Draft.Target.Resource != "owner/repo" || len(provider().Mutations) != 0 {
		t.Fatalf("preview = %+v", preview)
	}
	// Authority without the reviewed digest, or with a different one, is refused.
	if denied := run(cli.ExitFailure, with(append(sections, "--authorize-external")...)...); denied.Status == "success" || denied.Details == "" {
		t.Fatalf("undigested create = %+v", denied)
	}
	if stale := run(cli.ExitFailure, with(append(sections, "--preview-digest", strings.Repeat("0", 64), "--authorize-external")...)...); stale.Status == "success" || len(provider().Mutations) != 0 {
		t.Fatalf("stale digest create = %+v", stale)
	}
	authorized := with(append(sections, "--preview-digest", preview.Draft.Digest, "--authorize-external")...)
	created := run(cli.ExitSuccess, authorized...)
	if created.WorkItem == nil || created.WorkItem.Resource != "owner/repo" || created.WorkItem.RepositoryKey != "core" || created.Details != "" {
		t.Fatalf("create = %+v", created)
	}
	// Read-back and duplicate protection: repeating the same authority does not
	// create another Issue.
	run(cli.ExitSuccess, authorized...)
	if listed := run(cli.ExitSuccess, "work-item", "list", "--project", "govsm"); len(listed.WorkItems) != 1 || listed.WorkItems[0].ExternalID != created.WorkItem.ExternalID {
		t.Fatalf("read-back = %+v", listed)
	}
	if applied := provider(); !applied.IssueCreated || len(applied.Mutations) != 1 {
		t.Fatalf("Provider mutations = %+v", applied)
	}
}

func TestEmbeddedWorkItemSkillRoutesThroughScopedInspection(t *testing.T) {
	source, err := os.ReadFile(filepath.Join("..", "..", "internal", "codexruntime", "skills", "axiom-work-item", "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"`axiom --json skill inspect axiom-work-item --operation <operation>`", "`details`", "`--elaborated-section <name>=<text>`", "There is no\n`--title` or `--body`", "reuse those exact\nvalues"} {
		if !strings.Contains(string(source), want) {
			t.Fatalf("axiom-work-item skill lacks %q", want)
		}
	}
}
