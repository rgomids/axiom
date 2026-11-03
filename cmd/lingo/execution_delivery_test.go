//go:build linux || darwin

package main

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/cli"
	"github.com/rgomids/axiom/internal/local"
	"github.com/rgomids/axiom/internal/workflow"
)

type cliDeliveryFake struct {
	reads, pushes, prs int

	request workflow.DeliveryRequest
}

func (d *cliDeliveryFake) ObserveDelivery(_ context.Context, p workflow.DeliveryPreview) (workflow.DeliveryObservation, error) {
	d.reads++
	t := p.Target
	o := workflow.DeliveryObservation{Conclusive: true, BaseCommit: t.BaseCommit, HeadCommit: t.HeadCommit, Tree: t.Tree, DiffDigest: t.DiffDigest, RemoteBase: t.BaseCommit, RemoteHead: d.request.ExpectedRemoteHead}
	if d.pushes > 0 {
		o.RemoteHead = t.HeadCommit
	}
	return o, nil
}
func (d *cliDeliveryFake) PushDelivery(context.Context, workflow.DeliveryPreview) error {
	d.pushes++
	return nil
}
func (d *cliDeliveryFake) CreateDeliveryPR(context.Context, workflow.DeliveryPreview) error {
	d.prs++
	return nil
}

func TestPublicExecutionDeliveryExactGrantAndChildDenial(t *testing.T) {
	f := newExecutionRunFixture(t)
	got := f.run(t, cli.ExitSuccess, minimalRun("--authorize-external")...)
	store, err := local.NewExecutionStore(f.env.state)
	if err != nil {
		t.Fatal(err)
	}
	x, err := store.Load(context.Background(), got.Stage.ProjectID, got.Stage.ExecutionID)
	if err != nil {
		t.Fatal(err)
	}
	r := workflow.DeliveryRequest{ProjectID: x.ProjectID, ExecutionID: x.ExecutionID, Revision: x.Revision, RepositoryKey: "main", RepositoryIdentity: x.Repositories[0].Identity, Target: workflow.DeliveryTarget{Kind: "github", Repository: "owner/repo", BaseRef: "refs/heads/main", HeadRef: "refs/heads/work", BaseCommit: strings.Repeat("a", 40), HeadCommit: strings.Repeat("b", 40), Tree: strings.Repeat("c", 40), DiffDigest: strings.Repeat("d", 64), Title: "fixture", Body: "result", Metadata: []byte(`{}`), Push: true}}
	r.Target.Remote = "https://github.com/" + r.Target.Repository + ".git"
	d := &cliDeliveryFake{request: r}
	factories := 0
	service := f.env.service.(lifecycleService)
	service.deliveryEffects = func(context.Context, string) (workflow.DeliveryEffects, error) { factories++; return d, nil }
	f.env.service = service
	wire, _ := json.Marshal(r)
	base := []string{"workflow", "delivery", "advance", "--project", "external", "--execution", x.ExecutionID, "--request", string(wire)}
	preview := f.run(t, cli.ExitSuccess, base...)
	if !strings.Contains(preview.Raw, r.Digest()) || factories != 0 {
		t.Fatal("preview had effects")
	}
	for _, flags := range [][]string{{"--grant", r.Digest()}, {"--grant", r.Digest(), "--authorize-local", "--authorize-human", "--authorize-pr"}, {"--grant", strings.Repeat("e", 64), "--authorize-local", "--authorize-human", "--authorize-push"}} {
		f.run(t, cli.ExitFailure, append(append([]string{}, base...), flags...)...)
	}
	authorized := append(append([]string{}, base...), "--grant", r.Digest(), "--authorize-local", "--authorize-human", "--authorize-push")
	t.Setenv("AXIOM_ATTEMPT_ID", "forged-human-claim")
	f.run(t, cli.ExitFailure, authorized...)
	t.Setenv("AXIOM_ATTEMPT_ID", "")
	if factories+d.reads+d.pushes+d.prs != 0 {
		t.Fatal("denial reached provider")
	}
	result := f.run(t, cli.ExitSuccess, authorized...)
	if d.pushes != 1 || d.prs != 0 || !strings.Contains(result.Raw, `"status":"confirmed"`) {
		t.Fatalf("unconfirmed %s", result.Raw)
	}
	// Read-only status retains the exact intent and confirmation without paths.
	status := f.run(t, cli.ExitSuccess, "workflow", "stage", "status", "--project", "external", "--execution", x.ExecutionID)
	if len(status.Stage.Deliveries) != 1 || status.Stage.Deliveries[0].Request.Digest() != r.Digest() || strings.Contains(result.Raw, f.env.state) {
		t.Fatalf("status %s", status.Raw)
	}
}
