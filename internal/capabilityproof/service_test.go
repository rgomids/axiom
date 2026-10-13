package capabilityproof

import (
	"context"
	"crypto/sha256"
	"errors"
	"strings"
	"testing"
	"time"
)

type fakeProofStore struct {
	evidence Evidence
	ref      string
	writes   int
}

func (f *fakeProofStore) Load(_ context.Context, ref string) (Evidence, error) {
	if ref != f.ref {
		return Evidence{}, ErrInvalidEvidence
	}
	return f.evidence, nil
}
func (f *fakeProofStore) Invalidate(_ context.Context, ref, reason string, revision [sha256.Size]byte) error {
	wire, _ := EncodeEvidence(f.evidence)
	if ref != f.ref || sha256.Sum256(wire) != revision {
		return ErrStalePreview
	}
	f.evidence.InvalidatedReason = reason
	f.writes++
	return nil
}

type fakeBindingSource struct{ current Current }

func (f *fakeBindingSource) Current(context.Context, Target) (Current, error) { return f.current, nil }
func proofFixture(variant string) (Service, *fakeProofStore, *fakeBindingSource, Target) {
	e := testEvidence()
	e.Binding.Variant = variant
	e.Model = &ModelBinding{"profile", strings.Repeat("b", 64), "model", "explicit", "high", 1, strings.Repeat("c", 64)}
	for _, c := range []string{Process, Noninteractive, AxiomCoordination, ModelInvocation, "reasoning-effort-high"} {
		e.Capabilities[c] = e.Capabilities[Read]
	}
	if variant == RepositoryWriteVariant {
		e.Capabilities[RepositoryWrite] = e.Capabilities[Read]
	}
	ref, _ := Digest(e)
	store := &fakeProofStore{evidence: e, ref: ref}
	model := *e.Model
	source := &fakeBindingSource{Current{e.Binding, &model, ref, true}}
	return Service{Store: store, Source: source, Now: func() time.Time { return e.ObservedAt.Add(time.Hour) }}, store, source, Target{"codex", "profile", variant}
}
func TestCheckMatchingPerVariant(t *testing.T) {
	for _, variant := range []string{ReadOnlyVariant, RepositoryWriteVariant} {
		t.Run(variant, func(t *testing.T) {
			s, store, _, target := proofFixture(variant)
			ref, code, err := s.Check(context.Background(), target, []string{Read}, variant == RepositoryWriteVariant, true, "")
			if err != nil || code != "" || ref != store.ref {
				t.Fatalf("%s %s %v", ref, code, err)
			}
			if _, _, err = s.Check(context.Background(), target, []string{Read}, variant == RepositoryWriteVariant, true, ref); err != nil {
				t.Fatal(err)
			}
		})
	}
}
func TestCheckBindingDrift(t *testing.T) {
	changes := map[string]func(*fakeBindingSource){
		"executable":    func(f *fakeBindingSource) { f.current.Binding.ExecutableDigest = strings.Repeat("d", 64) },
		"version":       func(f *fakeBindingSource) { f.current.Binding.Version = "2.0" },
		"mode":          func(f *fakeBindingSource) { f.current.Binding.ModeDigest = strings.Repeat("d", 64) },
		"adapter":       func(f *fakeBindingSource) { f.current.Binding.AdapterVersion = "2" },
		"axiom":         func(f *fakeBindingSource) { f.current.Binding.AxiomDigest = strings.Repeat("d", 64) },
		"profile":       func(f *fakeBindingSource) { f.current.Model.ProfileDigest = strings.Repeat("d", 64) },
		"effort-source": func(f *fakeBindingSource) { f.current.Model.EffortSourceDigest = strings.Repeat("d", 64) },
		"model":         func(f *fakeBindingSource) { f.current.Model.Model = "other" },
		"effort":        func(f *fakeBindingSource) { f.current.Model.EffortToken = "low" },
		"revision":      func(f *fakeBindingSource) { f.current.Model.ConfigurationRevision++ },
	}
	for name, change := range changes {
		t.Run(name, func(t *testing.T) {
			s, store, source, target := proofFixture(ReadOnlyVariant)
			change(source)
			_, code, err := s.Check(context.Background(), target, []string{Read}, false, true, "")
			if code != "capability_stale" || err == nil {
				t.Fatalf("%s %v", code, err)
			}
			_, code, err = s.Check(context.Background(), target, []string{Read}, false, true, store.ref)
			if code != "capability_stale" || !errors.Is(err, ErrStalePreview) {
				t.Fatalf("dispatch drift: %s %v", code, err)
			}
		})
	}
}
func TestCheckExpiredAndWrongVariant(t *testing.T) {
	s, store, source, target := proofFixture(ReadOnlyVariant)
	s.Now = func() time.Time { return store.evidence.ObservedAt.Add(MaxAge + time.Second) }
	if _, code, err := s.Check(context.Background(), target, nil, false, false, ""); err == nil || code != "capability_stale" {
		t.Fatal(code, err)
	}
	s.Now = func() time.Time { return store.evidence.ObservedAt }
	source.current.Binding.Variant = RepositoryWriteVariant
	target.Variant = RepositoryWriteVariant
	if _, code, err := s.Check(context.Background(), target, nil, true, false, ""); err == nil || code != "capability_stale" {
		t.Fatal(code, err)
	}
}
func TestShowStatesAndProfileRemoval(t *testing.T) {
	for _, kind := range []string{"fresh", "expired", "invalidated", "disabled", "removed", "other-binding"} {
		t.Run(kind, func(t *testing.T) {
			s, store, source, target := proofFixture(ReadOnlyVariant)
			switch kind {
			case "expired":
				s.Now = func() time.Time { return store.evidence.ObservedAt.Add(MaxAge + time.Second) }
			case "invalidated":
				store.evidence.InvalidatedReason = "explicit"
			case "disabled", "removed":
				source.current.ProfileActive = false
			case "other-binding":
				source.current.Binding.Version = "2.0"
			}
			view, err := s.Show(context.Background(), target)
			if err != nil {
				t.Fatal(err)
			}
			want := Unknown
			if kind == "fresh" {
				want = Proven
			}
			if view.States[Read] != want || len(view.Confinement.NotTested) != 5 {
				t.Fatalf("%+v", view)
			}
		})
	}
}
func TestInvalidateAuthorityAndPreview(t *testing.T) {
	s, store, source, target := proofFixture(ReadOnlyVariant)
	ctx := context.Background()
	p, err := s.PreviewInvalidate(ctx, target)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.Invalidate(ctx, p, LocalAuthority{}); !errors.Is(err, ErrAuthority) || store.writes != 0 {
		t.Fatal(err)
	}
	authority := LocalAuthority{Actor: "operator", Preview: p}
	source.current.Binding.Version = "2.0"
	if err = s.Invalidate(ctx, p, authority); !errors.Is(err, ErrStalePreview) || store.writes != 0 {
		t.Fatal(err)
	}
	if _, err = s.PreviewInvalidate(ctx, target); !errors.Is(err, ErrStalePreview) {
		t.Fatal("another binding preview", err)
	}
	source.current.Binding.Version = p.Current.Binding.Version
	if err = s.Invalidate(ctx, p, authority); err != nil || store.writes != 1 {
		t.Fatal(err)
	}
	view, err := s.Show(ctx, target)
	if err != nil || view.States[Read] != Unknown || store.evidence.InvalidatedReason != "explicit" {
		t.Fatal(view, err)
	}
	if err = s.Invalidate(ctx, p, authority); !errors.Is(err, ErrStalePreview) {
		t.Fatal("stale wire preview", err)
	}
}
func TestRequiredNonProvenBlocks(t *testing.T) {
	for _, status := range []State{Unknown, Unsupported} {
		s, store, source, target := proofFixture(ReadOnlyVariant)
		store.evidence.Capabilities[Noninteractive] = Result{State: status, Facts: ProbeFacts{InterfaceAbsent: status == Unsupported}}
		store.ref, _ = Digest(store.evidence)
		source.current.EvidenceRef = store.ref
		if _, _, err := s.Check(context.Background(), target, nil, false, false, ""); err == nil {
			t.Fatal("non-proven admitted")
		}
	}
}

func TestInvalidationPreviewFreezesProfile(t *testing.T) {
	s, store, source, target := proofFixture(ReadOnlyVariant)
	p, err := s.PreviewInvalidate(context.Background(), target)
	if err != nil {
		t.Fatal(err)
	}
	authority := LocalAuthority{Actor: "operator", Preview: p}
	source.current.Model.ProfileDigest = strings.Repeat("d", 64)
	if p.Current.Model.ProfileDigest == source.current.Model.ProfileDigest {
		t.Fatal("preview aliased source")
	}
	if err = s.Invalidate(context.Background(), p, authority); !errors.Is(err, ErrStalePreview) || store.writes != 0 {
		t.Fatal(err)
	}
}

func TestEachRequiredCapabilityBlocks(t *testing.T) {
	for _, cap := range []string{Read, RepositoryWrite, Process, Noninteractive, AxiomCoordination, ModelInvocation, "reasoning-effort-high"} {
		t.Run(cap, func(t *testing.T) {
			s, store, source, target := proofFixture(RepositoryWriteVariant)
			delete(store.evidence.Capabilities, cap)
			store.ref, _ = Digest(store.evidence)
			source.current.EvidenceRef = store.ref
			if _, code, err := s.Check(context.Background(), target, []string{Read}, true, true, ""); err == nil || code != "capability_unknown" {
				t.Fatal(code, err)
			}
		})
	}
}
