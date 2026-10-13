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
	p := runtimeprofile.ModelProfile{ID: "profile", RuntimeID: "codex", Model: "model", Effort: runtimeprofile.Effort{Mode: "explicit", Value: "high"}, Capabilities: []string{Read}, Roles: []string{"implementation"}, Complexities: []string{"high"}}
	pd, err := runtimeprofile.ProfileDigest(p)
	if err != nil {
		t.Fatal(err)
	}
	store.evidence.Model.ProfileDigest = pd
	source.current.Model.ProfileDigest = pd
	store.ref, _ = Digest(store.evidence)
	source.current.EvidenceRef = store.ref
	cfg := runtimeprofile.Configuration{FormatVersion: 2, Revision: 1, Runtimes: []runtimeprofile.Runtime{{ID: "codex", Adapter: "codex", Enabled: true, AllowlistedProfileIDs: []string{"profile"}}}, ModelProfiles: []runtimeprofile.ModelProfile{p}}
	s.Source = ConfiguredSource{source, cfg}
	view, err := s.Show(context.Background(), target)
	if err != nil || view.States[Read] != Proven {
		t.Fatal(view, err)
	}
	cfg.ModelProfiles[0].Model = "other"
	s.Source = ConfiguredSource{source, cfg}
	view, err = s.Show(context.Background(), target)
	if err != nil || view.States[Read] != Unknown || view.Blockers[Read] != "capability_stale" {
		t.Fatal(view, err)
	}
	cfg.ModelProfiles[0] = p
	cfg.Runtimes[0].Enabled = false
	s.Source = ConfiguredSource{source, cfg}
	view, err = s.Show(context.Background(), target)
	if err != nil || view.States[Read] != Unknown || view.Blockers[Read] != "capability_unknown" {
		t.Fatal(view, err)
	}
	cfg.Runtimes[0].Enabled = true
	cfg.ModelProfiles = nil
	s.Source = ConfiguredSource{source, cfg}
	view, err = s.Show(context.Background(), target)
	if err != nil || view.States[Read] != Unknown {
		t.Fatal(view, err)
	}
}
