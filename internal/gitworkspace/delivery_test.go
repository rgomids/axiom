package gitworkspace

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/workflow"
	"github.com/rgomids/axiom/internal/workflowdefinition"
)

func TestDeliveryGitLocalBareConfirmationAndNoForce(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	repo, remote := filepath.Join(dir, "repo"), filepath.Join(dir, "remote.git")
	run := func(cwd string, args ...string) string {
		t.Helper()
		c := exec.Command("git", args...)
		c.Dir = cwd
		c.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1")
		out, err := c.CombinedOutput()
		if err != nil {
			t.Fatal(string(out), err)
		}
		return strings.TrimSpace(string(out))
	}
	run(dir, "init", "--bare", remote)
	run(dir, "init", "-b", "main", repo)
	run(repo, "-c", "user.name=Fixture", "-c", "user.email=fixture@example.invalid", "commit", "--allow-empty", "-m", "initial")
	base := run(repo, "rev-parse", "HEAD")
	run(repo, "push", remote, "main")
	g, err := NewDeliveryGit(ctx, repo)
	if err != nil {
		t.Fatal(err)
	}
	g.allowFile = true
	tree := run(repo, "rev-parse", "HEAD^{tree}")
	packet := workflow.DeliveryPacket{FormatVersion: 1, Kind: "delivery_packet", ProjectID: "fixture-project", RepositoryKey: "main", WorkItem: workflow.WorkItem{Provider: "github", Resource: "fixture/repository", ExternalID: "1", URL: "https://github.com/fixture/repository/issues/1", State: "OPEN"}, ExecutionID: "00000000-0000-4000-8000-000000000001", ExecutionRevision: 1, LedgerDigest: strings.Repeat("a", 64), BindingDigest: strings.Repeat("b", 64), Definition: workflowdefinition.Builtin().Ref("builtin"), Stages: []workflow.DeliveryStage{{Ordinal: 0, StageID: "implementation", ResultTree: tree}}, Blockers: []string{}, Ready: true, Next: "preview", Provenance: workflow.Identity{Product: "Axiom", Version: "fixture", Revision: "fixture", SourceState: "unknown"}}
	proposal, err := g.PreviewDeliveryCommit(ctx, packet, workflow.DeliveryTarget{Kind: "github", Tree: tree, BaseCommit: base, BaseRef: "refs/heads/main", HeadRef: "refs/heads/delivery"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = g.ApplyDeliveryCommit(ctx, packet, proposal, proposal.Digest(), false); err == nil {
		t.Fatal("commit without authority")
	}
	prepared, err := g.ApplyDeliveryCommit(ctx, packet, proposal, proposal.Digest(), true)
	if err != nil {
		t.Fatal(err)
	}
	head := prepared.HeadCommit
	p := workflow.DeliveryPreview{Target: workflow.DeliveryTarget{Repository: "fixture/repository", Remote: remote, BaseRef: "refs/heads/main", HeadRef: "refs/heads/delivery", BaseCommit: base, HeadCommit: head}}
	observed, err := g.ObserveDeliveryGit(ctx, p)
	if err != nil || observed.RemoteBase != base || observed.RemoteHead != "" || observed.Tree != tree {
		t.Fatal(observed, err)
	}
	p.Observation = observed
	run(remote, "update-ref", p.Target.HeadRef, base)
	if err = g.PushDelivery(ctx, p); err == nil {
		t.Fatal("remote drift after observation was overwritten")
	}
	if tip := run(remote, "rev-parse", p.Target.HeadRef); tip != base {
		t.Fatal("unexpected remote effect")
	}
	run(remote, "update-ref", "-d", p.Target.HeadRef, base)
	if err = g.PushDelivery(ctx, p); err != nil {
		t.Fatal(err)
	}
	observed, err = g.ObserveDeliveryGit(ctx, p)
	if err != nil || observed.RemoteHead != head {
		t.Fatal(observed, err)
	}
	g.allowFile = false
	if err = g.PushDelivery(ctx, p); err == nil {
		t.Fatal("product path accepted file transport")
	}
}

func TestDeliveryGitCredentialIsTransientAndBoundToGitHub(t *testing.T) {
	g, err := (DeliveryGit{}).WithGitHubCredential("fixture-only")
	if err != nil || len(g.authentication) != 3 || g.authentication[1] != "GIT_CONFIG_KEY_0=http.https://github.com/.extraheader" {
		t.Fatal("missing scoped transient credential")
	}
	for _, invalid := range []string{"", "line\nseparator", "argument separator"} {
		if _, err = g.WithGitHubCredential(invalid); err == nil {
			t.Fatal("invalid credential")
		}
	}
}
