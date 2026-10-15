package capabilityproof

import (
	"context"
	"github.com/rgomids/axiom/internal/runtimeprofile"
	"testing"
)

type fakeObserver struct{ observation runtimeprofile.Observation }

func (f fakeObserver) Observe(context.Context, string) (runtimeprofile.Observation, error) {
	return f.observation, nil
}
func TestObserverVariantAndFreshness(t *testing.T) {
	for _, variant := range []string{ReadOnlyVariant, RepositoryWriteVariant} {
		t.Run(variant, func(t *testing.T) {
			s, store, source, target := proofFixture(variant)
			base := fakeObserver{runtimeprofile.Observation{RuntimeID: "codex", Installed: true, Available: true, ExecutableDigest: source.current.Binding.ExecutableDigest, Version: source.current.Binding.Version, Revision: 1}}
			observer := Observer{Base: base, Service: s, Targets: []Target{target}}
			observed, err := observer.Observe(context.Background(), "codex")
			if err != nil || observed.CapabilityStatus[Read] != runtimeprofile.CapabilityProven || len(observed.ProfileProofs) != 1 || observed.ProfileProofs[0].Reference != store.ref {
				t.Fatal(observed, err)
			}
			if variant == ReadOnlyVariant && observed.CapabilityStatus[RepositoryWrite] == runtimeprofile.CapabilityProven {
				t.Fatal("read-only write proof")
			}
			source.current.Binding.Version = "changed"
			observed, err = observer.Observe(context.Background(), "codex")
			if err != nil || observed.CapabilityStatus[Read] == runtimeprofile.CapabilityProven || len(observed.ProfileProofs) != 0 {
				t.Fatal("drift admitted", observed, err)
			}
		})
	}
}

func TestObserverRejectsMixedVariants(t *testing.T) {
	s, _, source, target := proofFixture(ReadOnlyVariant)
	base := fakeObserver{runtimeprofile.Observation{RuntimeID: "codex", Installed: true, Available: true, ExecutableDigest: source.current.Binding.ExecutableDigest, Version: source.current.Binding.Version, Revision: 1}}
	other := target
	other.Variant = RepositoryWriteVariant
	observer := Observer{Base: base, Service: s, Targets: []Target{target, other}}
	if _, err := observer.Observe(context.Background(), "codex"); err == nil {
		t.Fatal("mixed variants admitted")
	}
}

func TestConfiguredSourceReflectsProfileState(t *testing.T) {
	s, store, source, target := proofFixture(ReadOnlyVariant)
	p := runtimeprofile.ModelProfile{ID: "profile", RuntimeID: "codex", Model: "model", Effort: runtimeprofile.Effort{Mode: "explicit", Value: "high"}, Capabilities: []string{Read}, Complexities: []string{"high"}}
	pd, err := runtimeprofile.ProfileDigest(p)
	if err != nil {
		t.Fatal(err)
	}
	store.evidence.Model.ProfileDigest = pd
	source.current.Model.ProfileDigest = pd
	store.ref, _ = Digest(store.evidence)
	source.current.EvidenceRef = store.ref
	cfg := runtimeprofile.ConfigurationV3{FormatVersion: runtimeprofile.FormatVersionV3, Revision: 1, Runtimes: []runtimeprofile.RuntimeV3{{ID: "codex", Adapter: "codex", Enabled: true, AllowlistedProfileIDs: []string{"profile"}}}, ModelProfiles: []runtimeprofile.ModelProfile{p}}
	consumer := func() runtimeprofile.ConsumerV3 {
		c, err := runtimeprofile.NewConsumerV3(runtimeprofile.RecordActiveV3, &cfg)
		if err != nil {
			t.Fatal(err)
		}
		return c
	}
	s.Source = ConfiguredSource{source, consumer()}
	base := fakeObserver{runtimeprofile.Observation{RuntimeID: "codex", Adapter: "codex", Installed: true, Available: true, ExecutableDigest: source.current.Binding.ExecutableDigest, Version: source.current.Binding.Version, Revision: 1, ObservedAt: s.now()}}
	observation, observeErr := Observer{Base: base, Service: s, Targets: []Target{target}}.Observe(context.Background(), "codex")
	choice, code := consumer().ObservedChoice("codex", "profile", observation, []string{Read})
	if observeErr != nil || code != "" || choice.ResolvedProfile.CapabilityProofReference != store.ref {
		t.Fatal(choice, code, observeErr)
	}
	view, err := s.Show(context.Background(), target)
	if err != nil || view.States[Read] != Proven {
		t.Fatal(view, err)
	}
	cfg.ModelProfiles[0].Model = "other"
	s.Source = ConfiguredSource{source, consumer()}
	view, err = s.Show(context.Background(), target)
	if err != nil || view.States[Read] != Unknown || view.Blockers[Read] != "capability_stale" {
		t.Fatal(view, err)
	}
	cfg.ModelProfiles[0] = p
	cfg.Runtimes[0].Enabled = false
	s.Source = ConfiguredSource{source, consumer()}
	view, err = s.Show(context.Background(), target)
	if err != nil || view.States[Read] != Unknown || view.Blockers[Read] != "capability_unknown" {
		t.Fatal(view, err)
	}
	// A configuration without the Profile has no valid consumer at all.
	s.Source = ConfiguredSource{source, runtimeprofile.ConsumerV3{}}
	view, err = s.Show(context.Background(), target)
	if err != nil || view.States[Read] != Unknown {
		t.Fatal(view, err)
	}
}

func TestObserverRejectsMixedProcessModes(t *testing.T) {
	s, _, source, target := proofFixture(ReadOnlyVariant)
	base := fakeObserver{runtimeprofile.Observation{RuntimeID: "codex", Installed: true, Available: true, ExecutableDigest: source.current.Binding.ExecutableDigest, Version: source.current.Binding.Version, Revision: 1}}
	other := target
	other.RuntimeID, other.Process = "claude", !target.Process
	if _, err := (Observer{Base: base, Service: s, Targets: []Target{target, other}}).Observe(context.Background(), "codex"); err != ErrBlocked {
		t.Fatalf("mixed Process modes observed: %v", err)
	}
}
