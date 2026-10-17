package workflow

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
)

type publicationMemory struct {
	records map[string]DeliveryPublication
	record  *DeliveryPublication
	saves   int
	failAt  int
}

func (m *publicationMemory) LoadDeliveryPublication(_ context.Context, _ string, ref string) (DeliveryPublication, error) {
	if m.records != nil {
		if r, ok := m.records[ref]; ok {
			return r, nil
		}
	}
	return DeliveryPublication{}, ErrNotFound
}

func (m *publicationMemory) SaveDeliveryPublication(_ context.Context, _ State, r DeliveryPublication, prior *DeliveryPublication) error {
	m.saves++
	if m.saves == m.failAt {
		return ErrRecoveryRequired
	}
	existing, exists := m.records[r.Preview.Digest()]
	if prior == nil && exists || prior != nil && (!exists || !reflect.DeepEqual(*prior, existing)) {
		return ErrConflict
	}
	copy := r
	m.record = &copy
	if m.records == nil {
		m.records = map[string]DeliveryPublication{}
	}
	m.records[r.Preview.Digest()] = r
	return nil
}

type publicationFake struct {
	observation        DeliveryObservation
	pushes, prs        int
	pushError, prError error
	ambiguous          bool
}

func (f *publicationFake) ObserveDelivery(context.Context, DeliveryPreview) (DeliveryObservation, error) {
	o := f.observation
	if f.ambiguous && len(o.Candidates) > 0 {
		o.Candidates = append(o.Candidates, o.Candidates[0])
	}
	return o, nil
}
func (f *publicationFake) PushDelivery(_ context.Context, p DeliveryPreview) error {
	f.pushes++
	f.observation.RemoteHead = p.Target.HeadCommit
	return f.pushError
}
func (f *publicationFake) CreateDeliveryPR(_ context.Context, p DeliveryPreview) error {
	f.prs++
	t := p.Target
	f.observation.Candidates = []DeliveryPR{{ID: "7", URL: "https://github.com/fixture/repository/pull/7", Repository: t.Repository, BaseRef: t.BaseRef, HeadRef: t.HeadRef, BaseCommit: t.BaseCommit, HeadCommit: t.HeadCommit, Title: t.Title, Body: p.PRBody(), Metadata: t.Metadata}}
	return f.prError
}
func publicationFixture(t *testing.T) (State, DeliveryPublisher, *publicationMemory, *publicationFake, DeliveryPreview) {
	t.Helper()
	s, e := packetFixture(t, false, false)
	p, err := ComposeDeliveryPacket(context.Background(), s, e)
	if err != nil {
		t.Fatal(err)
	}
	tree, base, head, diff := strings.Repeat("a", 40), strings.Repeat("b", 40), strings.Repeat("c", 40), strings.Repeat("d", 64)
	p.Stages[len(p.Stages)-1].ResultTree = tree
	p.Stages[len(p.Stages)-1].DiffDigest = diff
	target := DeliveryTarget{Kind: "github", Repository: "fixture/repository", Remote: "https://github.com/fixture/repository.git", BaseRef: "refs/heads/main", HeadRef: "refs/heads/delivery", BaseCommit: base, HeadCommit: head, Tree: tree, DiffDigest: diff, Title: "Reviewed delivery", Body: "Synthetic reviewed result", Metadata: []byte(`{}`), Push: true, PR: true}
	f := &publicationFake{observation: DeliveryObservation{BaseCommit: base, HeadCommit: head, Tree: tree, DiffDigest: diff, RemoteBase: base, Conclusive: true, Candidates: []DeliveryPR{}}}
	store := &publicationMemory{}
	service := DeliveryPublisher{store, f}
	preview, err := service.Preview(context.Background(), s, p, target)
	if err != nil {
		t.Fatal(err)
	}
	return s, service, store, f, preview
}
func publicationAuthority(p DeliveryPreview) DeliveryAuthority {
	return DeliveryAuthority{p.Digest(), true, true, true}
}
func TestDeliveryPublicationExactEffectsReplayAndLocal(t *testing.T) {
	s, d, _, f, p := publicationFixture(t)
	ctx := context.Background()
	r, err := d.Apply(ctx, s, p, publicationAuthority(p))
	if err != nil || r.Status != "success" || r.Artifact != "https://github.com/fixture/repository/pull/7" {
		t.Fatal(r, err)
	}
	if _, err = d.Apply(ctx, s, p, publicationAuthority(p)); err != nil || f.pushes != 1 || f.prs != 1 {
		t.Fatal("replayed effects", err, f)
	}
	store := &publicationMemory{}
	d = DeliveryPublisher{Store: store}
	p, err = d.Preview(ctx, s, p.Packet, DeliveryTarget{Kind: "local"})
	if err != nil {
		t.Fatal(err)
	}
	r, err = d.Apply(ctx, s, p, DeliveryAuthority{Digest: p.Digest(), Local: true})
	if err != nil || r.Status != "success" || r.Artifact != "delivery-packet:"+p.PacketDigest {
		t.Fatal(r, err)
	}
}
func TestDeliveryPublicationDenialDriftAndStaleHaveZeroEffects(t *testing.T) {
	for _, scenario := range []string{"push_denied", "pr_denied", "local_denied", "digest", "state", "incomplete", "drift"} {
		t.Run(scenario, func(t *testing.T) {
			s, d, store, f, p := publicationFixture(t)
			a := publicationAuthority(p)
			switch scenario {
			case "push_denied":
				a.Push = false
			case "pr_denied":
				a.PR = false
			case "local_denied":
				a.Local = false
			case "digest":
				a.Digest = strings.Repeat("f", 64)
			case "state":
				s.StorageRevision[0] ^= 1
			case "incomplete":
				p.Packet.Ready = false
				p.Packet.Blockers = []string{"missing_review"}
				p.PacketDigest, _ = DeliveryPacketDigest(p.Packet)
				a = publicationAuthority(p)
			case "drift":
				f.observation.RemoteBase = strings.Repeat("e", 40)
			}
			if _, err := d.Apply(context.Background(), s, p, a); err == nil || f.pushes != 0 || f.prs != 0 || store.record != nil {
				t.Fatal("denial produced effects", err)
			}
		})
	}
}
func TestDeliveryPublicationUnknownRequiresReconciliation(t *testing.T) {
	for _, effect := range []string{"push", "pr"} {
		t.Run(effect, func(t *testing.T) {
			s, d, store, f, p := publicationFixture(t)
			if effect == "push" {
				f.pushError = errors.New("lost ack")
			} else {
				f.prError = errors.New("lost ack")
			}
			r, err := d.Apply(context.Background(), s, p, publicationAuthority(p))
			if err == nil || r.Status == "success" {
				t.Fatal("uncertain publication succeeded")
			}
			if _, err = d.Apply(context.Background(), s, p, publicationAuthority(p)); !errors.Is(err, ErrRecoveryRequired) {
				t.Fatal(err)
			}
			if f.pushes != 1 || effect == "pr" && f.prs != 1 {
				t.Fatal("blind retry")
			}
			if effect == "push" {
				if _, err = d.Reconcile(context.Background(), s, *store.record); err == nil {
					t.Fatal("missing PR inferred")
				}
				return
			}
			f.ambiguous = true
			if _, err = d.Reconcile(context.Background(), s, r); err == nil {
				t.Fatal("ambiguous candidates accepted")
			}
			f.ambiguous = false
			r, err = d.Reconcile(context.Background(), s, r)
			if err != nil || r.Status != "success" || f.pushes != 1 || f.prs != 1 {
				t.Fatal(r, err)
			}
		})
	}
}
func TestDeliveryPublicationFenceAndPostEffectPersistenceFailure(t *testing.T) {
	for _, failAt := range []int{1, 2, 3} {
		s, d, store, f, p := publicationFixture(t)
		store.failAt = failAt
		r, err := d.Apply(context.Background(), s, p, publicationAuthority(p))
		if err == nil {
			t.Fatal("fault ignored")
		}
		if failAt == 1 && f.pushes != 0 || failAt == 2 && f.prs != 0 || r.Status == "success" && store.record.Status == "success" {
			t.Fatal("unsafe failure", r)
		}
	}
}

func TestDeliveryPublicationRetryNeedsConclusiveAbsenceAndNewExactAuthority(t *testing.T) {
	s, d, _, f, p := publicationFixture(t)
	ctx := context.Background()
	f.prError = errors.New("lost ack")
	r, err := d.Apply(ctx, s, p, publicationAuthority(p))
	if err == nil || r.Status != "partial" {
		t.Fatal(r, err)
	}
	target := p.Target
	target.RetryOf = p.Digest()
	target.Push = false
	if _, err = d.Preview(ctx, s, p.Packet, target); err == nil {
		t.Fatal("existing candidate admitted retry")
	}
	f.observation.Candidates = []DeliveryPR{}
	f.observation.Conclusive = false
	if _, err = d.Preview(ctx, s, p.Packet, target); err == nil {
		t.Fatal("inconclusive absence admitted retry")
	}
	f.observation.Conclusive = true
	retry, err := d.Preview(ctx, s, p.Packet, target)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = d.Apply(ctx, s, retry, publicationAuthority(p)); err == nil || f.prs != 1 || f.pushes != 1 {
		t.Fatal("old authority reused", err)
	}
	f.prError = nil
	r, err = d.Apply(ctx, s, retry, publicationAuthority(retry))
	if err != nil || r.Status != "success" || f.prs != 2 || f.pushes != 1 {
		t.Fatal(r, err, f.prs, f.pushes)
	}
}
