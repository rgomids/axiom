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
	"github.com/rgomids/axiom/internal/provenance"
	"github.com/rgomids/axiom/internal/workitem"
)

// Exercise the Runtime -> CLI -> Lingo -> Provider renderer path with no live
// Provider mutation. Natural-language reasoning is supplied by the Runtime.
func TestMinimalIntentInterviewProducesReusableDraftWithoutMutation(t *testing.T) {
	root, state, repository := filepath.Join(t.TempDir(), "projects"), filepath.Join(t.TempDir(), "state"), t.TempDir()
	t.Setenv("LINGO_PROJECTS_ROOT", root)
	t.Setenv("LINGO_STATE_ROOT", state)
	fakeProviderState := filepath.Join(t.TempDir(), "provider.json")
	if err := os.WriteFile(fakeProviderState, []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}
	gh, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("AXIOM_GH_BIN", gh)
	t.Setenv(fakeGitHubStateVariable, fakeProviderState)
	service := compose()
	configureProject(t, service, "interview", "Interview", "main="+repository, "github")
	before := snapshotTrees(t, root, state)
	base := []string{"work-item", "create", "--project", "interview", "--repository", "main", "--provider-repository", "owner/repo", "--intent", "Export fails when I select multiple rows."}
	run := func(args []string, want int) struct {
		Draft     *workitem.DraftPreview
		Questions []workitem.Question
	} {
		t.Helper()
		var output bytes.Buffer
		if code := cli.Run(context.Background(), args, service, currentProvenance(), &output); code != want {
			t.Fatalf("code=%d output=%s", code, output.String())
		}
		var value struct {
			Draft     *workitem.DraftPreview
			Questions []workitem.Question
		}
		if err := json.Unmarshal(output.Bytes(), &value); err != nil {
			t.Fatal(err)
		}
		return value
	}
	incomplete := run(base, cli.ExitFailure)
	if len(incomplete.Questions) != 6 || incomplete.Draft != nil {
		t.Fatalf("incomplete=%#v", incomplete)
	}
	for _, question := range incomplete.Questions {
		if question.Field == "problem" {
			t.Fatal("asked for supplied problem again")
		}
	}
	// Conversation answer supplies an observable outcome and a concrete check;
	// the Runtime proposes bounded scope and explicitly identifies unknown facts.
	args := append(append([]string(nil), base...), "--desired-outcome", "Export all selected rows into a file.", "--acceptance", "Select two rows, export, and verify both rows in the saved file.")
	for _, section := range []string{"context=No additional environment or background supplied.", "scope=Restore export of multiple selected rows.", "constraints=No additional constraints supplied; preserve unrelated behavior.", "non_goals=No redesign or unrelated export features."} {
		args = append(args, "--elaborated-section", section)
	}
	preview := run(args, cli.ExitSuccess).Draft
	if preview == nil || len(preview.Draft.Sections) != 7 || preview.Digest == "" || preview.Target.Resource != "owner/repo" {
		t.Fatalf("preview=%#v", preview)
	}
	for index, section := range preview.Draft.Sections {
		want := provenance.AxiomAuthored
		if index == 0 || index == 1 || index == 6 {
			want = provenance.UserAuthored
		}
		if section.Authorship != want || !strings.Contains(preview.ProviderDocument.Body, section.Content) {
			t.Fatalf("section=%#v body=%s", section, preview.ProviderDocument.Body)
		}
	}
	if preview.Draft.Sections[0].Content != "Export fails when I select multiple rows." {
		t.Fatal("original problem lost")
	}
	corrected := append(append([]string(nil), args...), "--problem", "Export fails only when more than ten rows are selected.")
	fresh := run(corrected, cli.ExitSuccess).Draft
	if fresh == nil || fresh.Digest == preview.Digest {
		t.Fatal("correction did not change preview")
	}
	run(append(corrected, "--preview-digest", preview.Digest, "--authorize-external"), cli.ExitFailure)
	// One supplied section cannot silently shadow a different Runtime proposal.
	run(append(append([]string(nil), args...), "--elaborated-section", "desired_outcome=Another outcome"), cli.ExitFailure)
	if after := snapshotTrees(t, root, state); !bytes.Equal(before, after) {
		t.Fatal("preview or stale authority mutated local state")
	}
	wire, err := os.ReadFile(fakeProviderState)
	if err != nil {
		t.Fatal(err)
	}
	var provider fakeGitHub
	if err := json.Unmarshal(wire, &provider); err != nil {
		t.Fatal(err)
	}
	if provider.IssueCreated || len(provider.Mutations) != 0 {
		t.Fatalf("Provider mutated: %+v", provider)
	}
}
