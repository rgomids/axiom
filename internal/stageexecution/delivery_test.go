//go:build linux || darwin

package stageexecution

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/rgomids/axiom/internal/local"
	"github.com/rgomids/axiom/internal/workflow"
)

type deliveryFake struct {
	t                  *testing.T
	f                  *engineFixture
	request            workflow.DeliveryRequest
	reads, pushes, prs int
	lost               bool
	mismatch           bool
	beforeRead         func(int)
	beforeEffect       func()
}

func (d *deliveryFake) ObserveDelivery(_ context.Context, p workflow.DeliveryPreview) (workflow.DeliveryObservation, error) {
	d.t.Helper()
	d.reads++
	if !reflect.DeepEqual(p.Target, d.request.Target) || p.PacketDigest != d.request.Digest() {
		d.t.Fatal("read escaped exact request")
	}
	if d.beforeRead != nil {
		d.beforeRead(d.reads)
	}
	r := d.request
	target := r.Target
	o := workflow.DeliveryObservation{Conclusive: true, BaseCommit: target.BaseCommit, HeadCommit: target.HeadCommit, Tree: target.Tree, DiffDigest: target.DiffDigest, RemoteBase: target.BaseCommit, RemoteHead: r.ExpectedRemoteHead}
	if d.pushes > 0 {
		o.RemoteHead = target.HeadCommit
	}
	if d.prs > 0 {
		o.Candidates = []workflow.DeliveryPR{{ID: "7", URL: "https://github.com/example/app/pull/7", Repository: target.Repository, BaseRef: target.BaseRef, HeadRef: target.HeadRef, BaseCommit: target.BaseCommit, HeadCommit: target.HeadCommit, Title: target.Title, Body: p.PRBody(), Metadata: target.Metadata}}
	}
	if d.mismatch {
		o.RemoteBase = strings.Repeat("f", 40)
	}
	return o, nil
}
func (d *deliveryFake) effect(p workflow.DeliveryPreview) error {
	d.t.Helper()
	x, err := d.f.engine.Store.Load(context.Background(), "proj", engineExecutionID)
	if err != nil || len(x.Deliveries) == 0 || x.Deliveries[len(x.Deliveries)-1].Status != "prepared" || !reflect.DeepEqual(x.Deliveries[len(x.Deliveries)-1].Request, d.request) {
		d.t.Fatalf("effect before durable exact intent: %+v %v", x.Deliveries, err)
	}
	if d.beforeEffect != nil {
		d.beforeEffect()
	}
	if d.lost {
		return errors.New("lost response after fake effect")
	}
	return nil
}
func (d *deliveryFake) PushDelivery(_ context.Context, p workflow.DeliveryPreview) error {
	d.pushes++
	return d.effect(p)
}
func (d *deliveryFake) CreateDeliveryPR(_ context.Context, p workflow.DeliveryPreview) error {
	d.prs++
	return d.effect(p)
}

func deliveryFixture(t *testing.T, pr bool) (*engineFixture, *deliveryFake, workflow.DeliveryRequest, workflow.DeliveryGrant) {
	t.Helper()
	f := newEngineFixture(t, []workflow.StageSpec{{ID: "work", Agents: []workflow.AgentSpec{{ID: "builder"}}, TimeoutSeconds: 5}}, succeed("done"))
	x, err := f.run(t)
	if err != nil || x.Status != workflow.RunCompleted {
		t.Fatalf("run %v %s", err, x.Status)
	}
	r := workflow.DeliveryRequest{ProjectID: x.ProjectID, ExecutionID: x.ExecutionID, Revision: x.Revision, RepositoryKey: "app", RepositoryIdentity: "github.com/example/app", Target: workflow.DeliveryTarget{Kind: "github", Repository: "example/app", Remote: "https://github.com/example/app.git", BaseRef: "refs/heads/main", HeadRef: "refs/heads/work", BaseCommit: strings.Repeat("a", 40), HeadCommit: strings.Repeat("b", 40), Tree: strings.Repeat("c", 40), DiffDigest: strings.Repeat("d", 64), Title: "fixture", Body: "result", Metadata: []byte(`{}`), Push: !pr, PR: pr}}
	if pr {
		r.ExpectedRemoteHead = r.Target.HeadCommit
	}
	a := workflow.DeliveryGrant{Digest: r.Digest(), Actor: "human:reviewer", Push: !pr, PR: pr}
	return f, &deliveryFake{t: t, f: f, request: r}, r, a
}
func allowDelivery(context.Context, workflow.Execution) error { return nil }

func TestExecutionDeliveryDeniedEffectsHaveZeroCalls(t *testing.T) {
	for _, pr := range []bool{false, true} {
		for _, scenario := range []string{"absent", "stale", "agent", "message", "wrong_effect", "target", "sha", "repository", "revision", "revoked", "cancelled"} {
			t.Run(stringPhase(pr)+"/"+scenario, func(t *testing.T) {
				f, d, r, a := deliveryFixture(t, pr)
				check := allowDelivery
				switch scenario {
				case "absent":
					a.Digest = ""
				case "stale":
					a.Digest = strings.Repeat("a", 64)
				case "agent":
					a.Actor = "agent:builder"
				case "message":
					a.Actor = "message:human:reviewer"
				case "wrong_effect":
					a.Push, a.PR = a.PR, a.Push
				case "target":
					r.Target.HeadRef = "refs/heads/other"
				case "sha":
					r.Target.HeadCommit = strings.Repeat("e", 40)
				case "repository":
					r.RepositoryIdentity = "github.com/other/app"
				case "revision":
					r.Revision++
					a.Digest = r.Digest()
				case "revoked":
					check = func(context.Context, workflow.Execution) error { return workflow.ErrUnauthorizedEffect }
				case "cancelled":
					_, err := f.engine.Store.UpdateFenced(context.Background(), "proj", engineExecutionID, nil, func(x *workflow.Execution) error { return x.RequestCancel("human:reviewer", time.Now()) })
					if err != nil {
						t.Fatal(err)
					}
					r.Revision++
					a.Digest = r.Digest()
				}
				_, err := f.engine.Deliver(context.Background(), r, a, d, check)
				if err == nil || d.pushes+d.prs+d.reads != 0 {
					t.Fatalf("denial leaked calls: %v %+v", err, d)
				}
			})
		}
	}
}
func stringPhase(pr bool) string {
	if pr {
		return "pr"
	}
	return "push"
}

func TestExecutionDeliveryIntentCorrelationAndNoReplay(t *testing.T) {
	for _, pr := range []bool{false, true} {
		t.Run(stringPhase(pr), func(t *testing.T) {
			f, d, r, a := deliveryFixture(t, pr)
			x, err := f.engine.Deliver(context.Background(), r, a, d, allowDelivery)
			if err != nil || len(x.Deliveries) != 1 || x.Deliveries[0].Status != "confirmed" || !reflect.DeepEqual(x.Deliveries[0].Request, r) || d.pushes+d.prs != 1 {
				t.Fatalf("confirm: %v %+v", err, x.Deliveries)
			}
			// Even a fresh grant/revision cannot replay an already confirmed exact effect.
			r.Revision = x.Revision
			a.Digest = r.Digest()
			d.request = r
			reads := d.reads
			_, err = f.engine.Deliver(context.Background(), r, a, d, allowDelivery)
			if err == nil || d.pushes+d.prs != 1 || d.reads != reads {
				t.Fatalf("replayed: %v", err)
			}
		})
	}
}

func TestExecutionDeliveryLostResponseInspectOnlyExactTarget(t *testing.T) {
	for _, pr := range []bool{false, true} {
		t.Run(stringPhase(pr), func(t *testing.T) {
			f, d, r, a := deliveryFixture(t, pr)
			d.lost = true
			x, err := f.engine.Deliver(context.Background(), r, a, d, allowDelivery)
			if !errors.Is(err, workflow.ErrUnknownOutstanding) || x.Status != workflow.RunRecoveryRequired || x.Deliveries[0].Status != "unknown" {
				t.Fatalf("lost fabricated success: %v %+v", err, x.Deliveries)
			}
			// Wrong authority cannot even inspect a remote.
			bad := a
			bad.Digest = strings.Repeat("e", 64)
			reads := d.reads
			_, err = f.engine.InspectDelivery(context.Background(), "proj", engineExecutionID, 0, bad, d, allowDelivery)
			if err == nil || d.reads != reads {
				t.Fatal("unauthorized inspect")
			}
			x, err = f.engine.InspectDelivery(context.Background(), "proj", engineExecutionID, 0, a, d, allowDelivery)
			if err != nil || x.Deliveries[0].Status != "unknown" || len(x.Deliveries[0].Observations) != 1 || d.reads != reads+1 || d.pushes+d.prs != 1 {
				t.Fatalf("inspect changed outcome/replayed: %v %+v", err, x.Deliveries)
			}
			r.Revision = x.Revision
			a.Digest = r.Digest()
			d.request = r
			_, err = f.engine.Deliver(context.Background(), r, a, d, allowDelivery)
			if err == nil || d.pushes+d.prs != 1 {
				t.Fatal("unknown replayed")
			}
		})
	}
}

func TestExecutionDeliveryDriftAndRevocationBeforeEffect(t *testing.T) {
	for _, scenario := range []string{"base", "head", "tree", "remote_head", "binding", "authority", "path", "ref", "cancel"} {
		t.Run(scenario, func(t *testing.T) {
			f, d, r, a := deliveryFixture(t, false)
			revoked := false
			d.beforeRead = func(n int) {
				if n == 2 {
					switch scenario {
					case "base":
						d.mismatch = true
					case "head":
						d.request.Target.HeadCommit = strings.Repeat("e", 40)
					case "tree":
						d.request.Target.Tree = strings.Repeat("e", 40)
					case "remote_head":
						d.request.ExpectedRemoteHead = strings.Repeat("e", 40)
					case "binding", "authority", "path", "ref":
						revoked = true
					case "cancel":
						_, err := f.engine.Store.UpdateFenced(context.Background(), "proj", engineExecutionID, nil, func(x *workflow.Execution) error { return x.RequestCancel("human:reviewer", time.Now()) })
						if err != nil {
							t.Fatal(err)
						}
					}
				}
			}
			check := func(context.Context, workflow.Execution) error {
				if revoked {
					return workflow.ErrUnauthorizedEffect
				}
				return nil
			}
			x, err := f.engine.Deliver(context.Background(), r, a, d, check)
			if err == nil || d.pushes+d.prs != 0 || len(x.Deliveries) != 1 || x.Deliveries[0].Status != "refused" || x.Status == workflow.RunRecoveryRequired {
				t.Fatalf("drift effect: %v %+v", err, x.Deliveries)
			}
		})
	}
}

func TestExecutionDeliveryConcurrentLeaseStaleFenceAndHistory(t *testing.T) {
	f, d, r, a := deliveryFixture(t, false)
	d.beforeEffect = func() {
		// A second runner cannot acquire the lease while the effect is in flight.
		current, loadErr := f.engine.Store.Load(context.Background(), "proj", engineExecutionID)
		if loadErr != nil {
			t.Fatal(loadErr)
		}
		concurrent := r
		concurrent.Revision = current.Revision
		grant := a
		grant.Digest = concurrent.Digest()
		_, err := f.engine.Deliver(context.Background(), concurrent, grant, d, allowDelivery)
		if !errors.Is(err, workflow.ErrRunnerHeld) {
			t.Fatalf("concurrent runner was not fenced: %v", err)
		}
	}
	x, err := f.engine.Deliver(context.Background(), r, a, d, allowDelivery)
	if err != nil {
		t.Fatal(err)
	}
	store := f.engine.Store.(local.ExecutionStore)
	// Lease-free edits and terminal/history rewrites cannot win at CAS.
	_, err = store.UpdateFenced(context.Background(), "proj", engineExecutionID, nil, func(x *workflow.Execution) error { x.Deliveries = nil; x.Revision++; return nil })
	if err == nil {
		t.Fatal("deleted delivery history")
	}
	oldGeneration := x.RunnerGeneration
	lease, _, err := f.engine.Claim(context.Background(), "proj", engineExecutionID)
	if err != nil {
		t.Fatal(err)
	}
	// Even a current lease holder cannot rewrite valid terminal intent.
	_, err = store.UpdateFenced(context.Background(), "proj", engineExecutionID, lease, func(x *workflow.Execution) error {
		x.Deliveries[0].Request.Target.Title = "rewritten"
		x.Deliveries[0].Grant.Digest = x.Deliveries[0].Request.Digest()
		x.Revision++
		return nil
	})
	if !errors.Is(err, local.ErrConflict) {
		t.Fatalf("terminal intent rewritten: %v", err)
	}
	lease.Close()
	_, err = store.UpdateFenced(context.Background(), "proj", engineExecutionID, lease, func(x *workflow.Execution) error { x.Revision++; return nil })
	if !errors.Is(err, workflow.ErrStaleRunner) {
		t.Fatalf("closed fence committed: %v", err)
	}
	if err = x.FinishDelivery(0, oldGeneration+1, "confirmed", nil, time.Now()); !errors.Is(err, workflow.ErrStaleRunner) {
		t.Fatalf("stale transition %v", err)
	}
}

func TestExecutionDeliveryCrashAfterIntentNeverReeffects(t *testing.T) {
	f, d, r, a := deliveryFixture(t, false)
	lease, x, err := f.engine.Claim(context.Background(), "proj", engineExecutionID)
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.engine.Store.UpdateFenced(context.Background(), "proj", engineExecutionID, lease, func(x *workflow.Execution) error { return x.PrepareDelivery(r, a, lease.Generation(), time.Now()) })
	if err != nil {
		t.Fatal(err)
	}
	lease.Close()
	lease, x, err = f.engine.Claim(context.Background(), "proj", engineExecutionID)
	if err != nil {
		t.Fatal(err)
	}
	lease.Close()
	if x.Status != workflow.RunRecoveryRequired || x.Deliveries[0].Status != "unknown" {
		t.Fatal("crash not unknown")
	}
	r.Revision = x.Revision
	a.Digest = r.Digest()
	_, err = f.engine.Deliver(context.Background(), r, a, d, allowDelivery)
	if err == nil || d.pushes+d.prs+d.reads != 0 {
		t.Fatal("crash replay")
	}
}

func TestExecutionDeliveryWrongObservedPRNeverConfirms(t *testing.T) {
	for _, field := range []string{"repository", "base", "head", "sha", "body", "metadata", "ambiguous", "inconclusive"} {
		t.Run(field, func(t *testing.T) {
			f, d, r, a := deliveryFixture(t, true)
			boundary := deliveryObservationFault{deliveryFake: d, field: field}
			x, err := f.engine.Deliver(context.Background(), r, a, boundary, allowDelivery)
			if !errors.Is(err, workflow.ErrUnknownOutstanding) || x.Deliveries[0].Status != "unknown" || d.prs != 1 {
				t.Fatalf("forged PR confirmed: %v %+v", err, x.Deliveries)
			}
		})
	}
}

type deliveryObservationFault struct {
	*deliveryFake
	field string
}

func (d deliveryObservationFault) ObserveDelivery(ctx context.Context, p workflow.DeliveryPreview) (workflow.DeliveryObservation, error) {
	o, err := d.deliveryFake.ObserveDelivery(ctx, p)
	if d.prs == 0 {
		return o, err
	}
	switch d.field {
	case "repository":
		o.Candidates[0].Repository = "other/app"
	case "base":
		o.Candidates[0].BaseRef = "refs/heads/other"
	case "head":
		o.Candidates[0].HeadRef = "refs/heads/other"
	case "sha":
		o.Candidates[0].HeadCommit = strings.Repeat("f", 40)
	case "body":
		o.Candidates[0].Body = "forged success"
	case "metadata":
		o.Candidates[0].Metadata = []byte(`{"changed":true}`)
	case "ambiguous":
		o.Candidates = append(o.Candidates, o.Candidates[0])
	case "inconclusive":
		o.Conclusive = false
	}
	return o, err
}

func TestExecutionDeliveryCodecRefusesFabricatedConfirmation(t *testing.T) {
	f, d, r, a := deliveryFixture(t, false)
	x, err := f.engine.Deliver(context.Background(), r, a, d, allowDelivery)
	if err != nil {
		t.Fatal(err)
	}
	wire, err := workflow.EncodeExecution(x)
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"grant", "request", "observation", "missing", "generation"} {
		t.Run(field, func(t *testing.T) {
			var forged workflow.Execution
			if err := json.Unmarshal(wire, &forged); err != nil {
				t.Fatal(err)
			}
			record := &forged.Deliveries[0]
			switch field {
			case "grant":
				record.Grant.Actor = "agent:builder"
			case "request":
				record.Request.Target.HeadCommit = strings.Repeat("e", 40)
			case "observation":
				record.Observations[0].RemoteHead = strings.Repeat("e", 40)
			case "missing":
				record.Observations = nil
			case "generation":
				record.RunnerGeneration = forged.RunnerGeneration + 1
			}
			raw, _ := json.Marshal(forged)
			if _, err := workflow.DecodeExecution(raw); err == nil {
				t.Fatal("codec accepted forged confirmation")
			}
		})
	}
}

func TestExecutionDeliveryPushThenPRUsesSameExecution(t *testing.T) {
	f, d, r, a := deliveryFixture(t, false)
	x, err := f.engine.Deliver(context.Background(), r, a, d, allowDelivery)
	if err != nil {
		t.Fatal(err)
	}
	r.Revision = x.Revision
	r.Target.Push = false
	r.Target.PR = true
	r.ExpectedRemoteHead = r.Target.HeadCommit
	a.Push = false
	a.PR = true
	a.Digest = r.Digest()
	d.request = r
	x, err = f.engine.Deliver(context.Background(), r, a, d, allowDelivery)
	if err != nil || x.ExecutionID != r.ExecutionID || len(x.Deliveries) != 2 || x.Deliveries[0].Status != "confirmed" || x.Deliveries[1].Status != "confirmed" || d.pushes != 1 || d.prs != 1 {
		t.Fatalf("two phases: %v %+v", err, x.Deliveries)
	}
}

func TestExecutionDeliveryPushMustCreate(t *testing.T) {
	for _, head := range []string{strings.Repeat("b", 40), strings.Repeat("e", 40)} {
		t.Run(head[:1], func(t *testing.T) {
			f, d, r, a := deliveryFixture(t, false)
			r.ExpectedRemoteHead = head
			d.request = r
			a.Digest = r.Digest()
			if r.Valid() {
				t.Error("canonical validation accepted existing push head")
			}
			x, err := f.engine.Deliver(context.Background(), r, a, d, allowDelivery)
			if !errors.Is(err, workflow.ErrUnauthorizedEffect) || d.reads+d.pushes+d.prs != 0 || len(x.Deliveries) != 0 || x.Status == workflow.RunRecoveryRequired {
				t.Fatalf("invalid push reached intent/effect: %v %+v", err, x)
			}
		})
	}
}

func TestExecutionDeliveryRefusalAllowsFreshGrant(t *testing.T) {
	f, d, r, a := deliveryFixture(t, false)
	checks := 0
	check := func(context.Context, workflow.Execution) error {
		checks++
		if checks == 3 {
			return workflow.ErrUnauthorizedEffect
		}
		return nil
	}
	x, err := f.engine.Deliver(context.Background(), r, a, d, check)
	if !errors.Is(err, workflow.ErrDeliveryRefused) || len(x.Deliveries) != 1 || x.Deliveries[0].Status != "refused" || x.Status == workflow.RunRecoveryRequired || d.pushes+d.prs != 0 {
		t.Fatalf("refusal: %v %+v", err, x.Deliveries)
	}
	wire, err := workflow.EncodeExecution(x)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = workflow.DecodeExecution(wire); err != nil {
		t.Fatal(err)
	}
	if err := x.ObserveDelivery(0, x.RunnerGeneration, a, workflow.DeliveryObservation{}, time.Now()); err == nil {
		t.Fatal("inspection changed a refused result")
	}
	prior := x.Deliveries[0]
	r.Revision = x.Revision
	a.Digest = r.Digest()
	d.request = r
	x, err = f.engine.Deliver(context.Background(), r, a, d, allowDelivery)
	if err != nil || len(x.Deliveries) != 2 || !reflect.DeepEqual(prior, x.Deliveries[0]) || x.Deliveries[1].Status != "confirmed" || d.pushes != 1 {
		t.Fatalf("refused blocked new grant: %v %+v", err, x.Deliveries)
	}
	changed := append([]workflow.ExecutionDelivery(nil), x.Deliveries...)
	changed[0].Status = "unknown"
	if workflow.PreservesDeliveryHistory(x.Deliveries, changed) {
		t.Fatal("refusal history rewritten")
	}
}

func TestExecutionDeliveryCallerCancellationBeforeEffect(t *testing.T) {
	for _, at := range []string{"read", "final_check", "persisted_final_check"} {
		t.Run(at, func(t *testing.T) {
			f, d, r, a := deliveryFixture(t, false)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			d.beforeRead = func(n int) {
				if n == 2 && at == "read" {
					cancel()
				}
			}
			checks := 0
			check := func(context.Context, workflow.Execution) error {
				checks++
				if checks == 4 {
					if at == "final_check" {
						cancel()
					}
					if at == "persisted_final_check" {
						if _, err := f.engine.Cancel(context.Background(), r.ProjectID, r.ExecutionID, "human:reviewer"); err != nil {
							t.Fatal(err)
						}
					}
				}
				return nil
			}
			x, err := f.engine.Deliver(ctx, r, a, d, check)
			if !errors.Is(err, workflow.ErrDeliveryRefused) || len(x.Deliveries) != 1 || x.Deliveries[0].Status != "refused" || x.Status == workflow.RunRecoveryRequired || d.pushes+d.prs != 0 {
				t.Fatalf("cancel: %v %+v", err, x.Deliveries)
			}
		})
	}
}

type refusedDeliveryFake struct{ *deliveryFake }

func (d refusedDeliveryFake) PushDelivery(context.Context, workflow.DeliveryPreview) error {
	return workflow.ErrDeliveryRefused
}
func TestExecutionDeliveryAdapterKnownRefusal(t *testing.T) {
	f, d, r, a := deliveryFixture(t, false)
	x, err := f.engine.Deliver(context.Background(), r, a, refusedDeliveryFake{d}, allowDelivery)
	if !errors.Is(err, workflow.ErrDeliveryRefused) || x.Deliveries[0].Status != "refused" || x.Status == workflow.RunRecoveryRequired || d.pushes+d.prs != 0 {
		t.Fatalf("adapter refusal: %v %+v", err, x.Deliveries)
	}
}
