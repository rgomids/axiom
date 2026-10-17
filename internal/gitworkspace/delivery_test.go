package gitworkspace

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/workflow"
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
	head, err := g.PrepareDeliveryCommit(ctx, tree, base, "reviewed local result")
	if err != nil {
		t.Fatal(err)
	}
	run(repo, "update-ref", "refs/heads/delivery", head)
	p := workflow.DeliveryPreview{Target: workflow.DeliveryTarget{Repository: "fixture/repository", Remote: remote, BaseRef: "refs/heads/main", HeadRef: "refs/heads/delivery", BaseCommit: base, HeadCommit: head}}
	observed, err := g.ObserveDeliveryGit(ctx, p)
	if err != nil || observed.RemoteBase != base || observed.RemoteHead != "" || observed.Tree != tree {
		t.Fatal(observed, err)
	}
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
