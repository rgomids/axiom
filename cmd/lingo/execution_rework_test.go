//go:build linux || darwin

package main

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/cli"
	"github.com/rgomids/axiom/internal/detailartifact"
	"github.com/rgomids/axiom/internal/local"
	"github.com/rgomids/axiom/internal/workflow"
)

func rejectionEvidence(t *testing.T, f *executionRunFixture, x workflow.Execution, change func(*detailartifact.Draft)) string {
	t.Helper()
	digest, err := x.StageResultDigest(workflow.DefaultStageID)
	if err != nil {
		t.Fatal(err)
	}
	draft := detailartifact.Draft{ExecutionID: x.ExecutionID, Category: "evidence", Outcome: "success", Retention: detailartifact.Evidence, Markdown: []byte("Human rejects this exact result; repair required."), Provenance: currentProvenance(), References: []detailartifact.Reference{{Kind: "workflow-stage", Value: workflow.DefaultStageID}, {Kind: "workflow-definition", Value: x.Workflow.Digest}, {Kind: "workflow-result", Value: digest}, {Kind: "workflow-repository", Value: x.WorkItem.RepositoryKey}}}
	if change != nil {
		change(&draft)
	}
	store, err := local.NewArtifactStore(f.env.state)
	if err != nil {
		t.Fatal(err)
	}
	a, err := store.Create(context.Background(), draft)
	if err != nil {
		t.Fatal(err)
	}
	return a.ID
}

func TestPublicRejectedResultAndLinkedReworkUseCanonicalLifecycle(t *testing.T) {
	f := newExecutionRunFixture(t)
	done := f.run(t, cli.ExitSuccess, minimalRun("--authorize-external")...)
	store, err := local.NewExecutionStore(f.env.state)
	if err != nil {
		t.Fatal(err)
	}
	source, err := store.Load(context.Background(), done.Stage.ProjectID, done.Stage.ExecutionID)
	if err != nil {
		t.Fatal(err)
	}
	digest, _ := source.StageResultDigest(workflow.DefaultStageID)
	evidence := rejectionEvidence(t, f, source, nil)
	args := []string{"workflow", "stage", "reject", "--project", "external", "--execution", source.ExecutionID, "--stage", workflow.DefaultStageID, "--result-digest", digest, "--evidence", evidence}
	f.run(t, cli.ExitFailure, args...)
	t.Setenv("AXIOM_ATTEMPT_ID", "agent-forging-human")
	f.run(t, cli.ExitFailure, append(args, "--authorize-human", "--authorize-local")...)
	t.Setenv("AXIOM_ATTEMPT_ID", "")
	// CLI has no actor input; even a claimed human identity cannot bypass ingress.
	f.run(t, cli.ExitFailure, append(args, "--actor", "human:operator", "--authorize-human", "--authorize-local")...)
	for _, change := range []func(*detailartifact.Draft){
		func(d *detailartifact.Draft) { d.ExecutionID = "00000000-0000-4000-8000-000000000099" },
		func(d *detailartifact.Draft) { d.References[0].Value = "wrong-stage" },
		func(d *detailartifact.Draft) { d.References[2].Value = workflow.DigestText([]byte("stale attempt")) },
		func(d *detailartifact.Draft) { d.References[3].Value = "wrong-repo" },
	} {
		bad := append([]string(nil), args...)
		bad[len(bad)-1] = rejectionEvidence(t, f, source, change)
		f.run(t, cli.ExitFailure, append(bad, "--authorize-human", "--authorize-local")...)
	}
	rejected := f.run(t, cli.ExitSuccess, append(args, "--authorize-human", "--authorize-local")...)
	if rejected.Stage.Status != workflow.RunRejected || rejected.Stage.Rejection == nil || latestAttempt(t, rejected.Stage, workflow.DefaultStageID).Status != workflow.AttemptSucceeded {
		t.Fatal(rejected.Raw)
	}
	frozen, err := store.Load(context.Background(), source.ProjectID, source.ExecutionID)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(frozen.Stages, source.Stages) {
		t.Fatal("source facts changed")
	}
	f.run(t, cli.ExitFailure, append(args, "--authorize-human", "--authorize-local")...)
	f.run(t, cli.ExitFailure, "workflow", "run", "--project", "external", "--execution", source.ExecutionID, "--authorize-external")
	// A new run cannot inherit authorization or the old invocation selection.
	f.run(t, cli.ExitFailure, minimalRun("--rework-of", source.ExecutionID)...)
	f.run(t, cli.ExitFailure, "workflow", "run", "--project", "external", "--repository", "main", "--work-item", "github:owner/repo#7", "--rework-of", source.ExecutionID, "--authorize-external")
	rework := f.run(t, cli.ExitSuccess, minimalRun("--rework-of", source.ExecutionID, "--authorize-external")...)
	if rework.Stage.ExecutionID == source.ExecutionID || rework.Stage.ReworkOf == nil || rework.Stage.ReworkOf.ExecutionID != source.ExecutionID || rework.Stage.Rejection != nil || len(rework.Stage.Deliveries) != 0 || rework.Stage.Status != workflow.RunCompleted {
		t.Fatal(rework.Raw)
	}
	newAttempt := latestAttempt(t, rework.Stage, workflow.DefaultStageID)
	if newAttempt.AttemptID == latestAttempt(t, done.Stage, workflow.DefaultStageID).AttemptID {
		t.Fatal("attempt identity inherited")
	}
	after, err := store.Load(context.Background(), source.ProjectID, source.ExecutionID)
	if err != nil || !reflect.DeepEqual(after, frozen) {
		t.Fatal("rework mutated source", err)
	}
	status := f.run(t, cli.ExitSuccess, "workflow", "stage", "status", "--project", "external", "--execution", source.ExecutionID)
	if status.Stage.Status != workflow.RunRejected || !strings.Contains(status.Stage.NextAction, "--rework-of") {
		t.Fatal(status.Raw)
	}
	packet := f.run(t, cli.ExitSuccess, "workflow", "delivery", "packet", "--project", "external", "--repository", "main", "--work-item", "github:owner/repo#7", "--execution", source.ExecutionID)
	if strings.Contains(packet.Raw, `"ready":true`) {
		t.Fatal("rejection authorized delivery", packet.Raw)
	}
}

// AXM-12: a human can reject through the public CLI alone. --note is retained
// by Axiom as the correlated Evidence; refusals name the failed precondition
// and never retain orphan Evidence.
func TestPublicRejectWithHumanNoteRetainsCorrelatedEvidence(t *testing.T) {
	f := newExecutionRunFixture(t)
	done := f.run(t, cli.ExitSuccess, minimalRun("--authorize-external")...)
	x := done.Stage
	digest := ""
	store, err := local.NewExecutionStore(f.env.state)
	if err != nil {
		t.Fatal(err)
	}
	source, err := store.Load(context.Background(), x.ProjectID, x.ExecutionID)
	if err != nil {
		t.Fatal(err)
	}
	if digest, err = source.StageResultDigest(workflow.DefaultStageID); err != nil {
		t.Fatal(err)
	}
	args := []string{"workflow", "stage", "reject", "--project", "external", "--execution", x.ExecutionID, "--stage", workflow.DefaultStageID, "--result-digest", digest, "--note", "Rejected: tests missing.", "--authorize-human", "--authorize-local"}
	before := artifactCount(t, f)
	for _, c := range []struct {
		change   func([]string) []string
		category string
	}{
		// Neither --evidence nor --note: refused by the CLI input contract.
		{func(a []string) []string { return append(a[:11], a[13:]...) }, ""},
		{func(a []string) []string { return append(a, "--evidence", "x") }, "invalid_review"},
		{func(a []string) []string { a[10] = workflow.DigestText([]byte("stale")); return a }, "result_digest_mismatch"},
		{func(a []string) []string { a[8] = "unknown-stage"; return a }, "result_digest_mismatch"},
		{func(a []string) []string { a[12] = "password: hunter2"; return a }, "review_evidence_invalid"},
	} {
		got := f.run(t, cli.ExitFailure, c.change(append([]string(nil), args...))...)
		if got.Category != c.category || artifactCount(t, f) != before {
			t.Fatal(c.category, got.Raw)
		}
	}
	rejected := f.run(t, cli.ExitSuccess, args...)
	if rejected.Stage.Status != workflow.RunRejected || rejected.Stage.Rejection == nil {
		t.Fatal(rejected.Raw)
	}
	artifacts, err := local.NewArtifactStore(f.env.state)
	if err != nil {
		t.Fatal(err)
	}
	a, err := artifacts.Read(context.Background(), rejected.Stage.Rejection.Evidence.RetainedID)
	want := []detailartifact.Reference{{Kind: "workflow-stage", Value: workflow.DefaultStageID}, {Kind: "workflow-definition", Value: source.Workflow.Digest}, {Kind: "workflow-result", Value: digest}, {Kind: "workflow-repository", Value: source.WorkItem.RepositoryKey}}
	if err != nil || a.ExecutionID != x.ExecutionID || string(a.Markdown) != "Rejected: tests missing." || a.Category != "evidence" || a.Retention != detailartifact.Evidence || a.Outcome != "success" || !reflect.DeepEqual(a.References, want) {
		t.Fatal(a, err)
	}
	// Already rejected: a state refusal that retains nothing.
	if got := f.run(t, cli.ExitFailure, args...); got.Category != "transition_refused" || !strings.Contains(got.Raw, "Evidence and digest are valid") || artifactCount(t, f) != before+1 {
		t.Fatal(got.Raw)
	}
	rework := f.run(t, cli.ExitSuccess, minimalRun("--rework-of", x.ExecutionID, "--authorize-external")...)
	if rework.Stage.ReworkOf == nil || rework.Stage.ReworkOf.ExecutionID != x.ExecutionID {
		t.Fatal(rework.Raw)
	}
}
