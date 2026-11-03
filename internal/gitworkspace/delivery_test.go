package gitworkspace

import (
	"context"
	"errors"
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
	// The canonical path delivers an existing reviewed commit; no packet commit owner.
	run(repo, "switch", "-c", "delivery")
	run(repo, "-c", "user.name=Fixture", "-c", "user.email=fixture@example.invalid", "commit", "--allow-empty", "-m", "reviewed delivery")
	head := run(repo, "rev-parse", "HEAD")
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
	if err = g.PushDelivery(ctx, p); !errors.Is(err, workflow.ErrDeliveryRefused) {
		t.Fatal("product path failed to classify file transport refusal", err)
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

func TestDeliveryGitPreEffectRefusal(t *testing.T) {
	// A nil manager would panic if validation reached Git.
	p := workflow.DeliveryPreview{Target: workflow.DeliveryTarget{HeadCommit: strings.Repeat("a", 40)}, Observation: workflow.DeliveryObservation{RemoteHead: strings.Repeat("b", 40)}}
	if err := (DeliveryGit{}).PushDelivery(context.Background(), p); !errors.Is(err, workflow.ErrDeliveryRefused) {
		t.Fatal(err)
	}
}
