package capabilityproof_test

import (
	"context"
	"crypto/sha256"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rgomids/axiom/internal/capabilityproof"
	"github.com/rgomids/axiom/internal/runtimeadapter"
)

// Process-mode identity is asserted against the canonical adapter ModeDigest,
// never against artificially altered hashes (AXM-22).

var observedAt = time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)

func fixturePath(path string) string {
	return filepath.Join(string(filepath.Separator), filepath.FromSlash(strings.TrimPrefix(path, "/")))
}

func canonicalBinding(t *testing.T, runtime, version string, process bool) capabilityproof.Binding {
	t.Helper()
	request := runtimeadapter.StageInvocationRequest{Profile: runtimeadapter.CommandProfile{RuntimeID: runtime, Executable: fixturePath("/bin/" + runtime), Model: "model"}, Version: version, Workspace: fixturePath("/work"), Inbox: fixturePath("/inbox"), AxiomPath: fixturePath("/bin/axiom"), Process: process}
	mode, err := runtimeadapter.ModeDigest(request)
	if err != nil {
		t.Fatal(err)
	}
	adapter, err := runtimeadapter.AdapterVersion(runtime)
	if err != nil {
		t.Fatal(err)
	}
	d := strings.Repeat("a", 64)
	return capabilityproof.Binding{RuntimeID: runtime, ExecutableDigest: d, Version: version, ModeDigest: mode, Variant: capabilityproof.ReadOnlyVariant, AdapterVersion: adapter, AxiomDigest: d}
}

func proven() capabilityproof.Result {
	return capabilityproof.Result{State: capabilityproof.Proven, Facts: capabilityproof.ProbeFacts{AuthenticationProven: true, BehavioralProbe: true, Verified: true}}
}

func modeEvidence(b capabilityproof.Binding) capabilityproof.Evidence {
	caps := map[string]capabilityproof.Result{}
	for _, c := range []string{capabilityproof.Read, capabilityproof.Process, capabilityproof.Noninteractive, capabilityproof.AxiomCoordination, capabilityproof.ModelInvocation} {
		caps[c] = proven()
	}
	caps["reasoning-effort-high"] = proven()
	return capabilityproof.Evidence{SchemaVersion: 1, Binding: b, Model: modeModel(), ObservedAt: observedAt, ProbePacketDigest: strings.Repeat("a", 64), Capabilities: caps, Confinement: capabilityproof.Confinement{Level: "unknown", NotTested: []string{"home", "vendor-home", "tmpdir", "system-temp", "external-network"}}}
}

func storedEvidence(t *testing.T, e capabilityproof.Evidence) capabilityproof.StoredEvidence {
	t.Helper()
	ref, err := capabilityproof.Digest(e)
	if err != nil {
		t.Fatal(err)
	}
	return capabilityproof.StoredEvidence{Ref: ref, Evidence: e}
}

type scanner []capabilityproof.StoredEvidence

func (s scanner) Scan(_ context.Context, visit func(capabilityproof.StoredEvidence) error) error {
	for _, r := range s {
		if err := visit(r); err != nil {
			return err
		}
	}
	return nil
}

func modeSelector(records ...capabilityproof.StoredEvidence) capabilityproof.Selector {
	return capabilityproof.Selector{Evidence: scanner(records), Now: func() time.Time { return observedAt.Add(time.Hour) }}
}

func TestClaudeProcessModesHaveDistinctCanonicalIdentity(t *testing.T) {
	limited := canonicalBinding(t, "claude", "2.1.296", false)
	process := canonicalBinding(t, "claude", "2.1.296", true)
	if limited.ModeDigest == process.ModeDigest {
		t.Fatal("Claude Process modes collapsed to one ModeDigest")
	}
	// Identical effective requests produce identical digests.
	if again := canonicalBinding(t, "claude", "2.1.296", true); again != process {
		t.Fatal("identical effective requests drifted")
	}
	// Codex 0.159.1 emits the same invocation for both modes, so the canonical
	// identity is shared and the Process capability alone gates the request.
	if codexFalse, codexTrue := canonicalBinding(t, "codex", "0.159.1", false), canonicalBinding(t, "codex", "0.159.1", true); codexFalse != codexTrue {
		t.Fatal("Codex effective invocation unexpectedly depends on Process")
	}
}

func TestEvidenceSelectionNeverCrossesProcessModes(t *testing.T) {
	limited := canonicalBinding(t, "claude", "2.1.296", false)
	process := canonicalBinding(t, "claude", "2.1.296", true)
	for name, tc := range map[string]struct{ held, requested capabilityproof.Binding }{
		"false evidence, true request": {limited, process},
		"true evidence, false request": {process, limited},
	} {
		t.Run(name, func(t *testing.T) {
			ref, code, err := modeSelector(storedEvidence(t, modeEvidence(tc.held))).Select(context.Background(), tc.requested, modeModel())
			if !errors.Is(err, capabilityproof.ErrEvidenceMissing) || code != "capability_unknown" || ref != "" {
				t.Fatalf("cross-mode Evidence selected: %q %q %v", ref, code, err)
			}
		})
	}
	// Both modes held: each request selects exactly its own record.
	held := []capabilityproof.StoredEvidence{storedEvidence(t, modeEvidence(limited)), storedEvidence(t, modeEvidence(process))}
	for i, b := range []capabilityproof.Binding{limited, process} {
		if ref, _, err := modeSelector(held...).Select(context.Background(), b, modeModel()); err != nil || ref != held[i].Ref {
			t.Fatalf("mode %d: %q %v", i, ref, err)
		}
	}
}

func modeModel() *capabilityproof.ModelBinding {
	return &capabilityproof.ModelBinding{ProfileID: "profile", ProfileDigest: strings.Repeat("b", 64), Model: "model", EffortMode: "explicit", EffortToken: "high", ConfigurationRevision: 1, EffortSourceDigest: strings.Repeat("c", 64)}
}

// modeSource is a synthetic BindingSource computing the current ModeDigest
// from the requested Target through the canonical adapter. It reads no
// Evidence for identity and launches nothing.
type modeSource struct {
	t   *testing.T
	ref string
}

func (s modeSource) Current(_ context.Context, target capabilityproof.Target) (capabilityproof.Current, error) {
	return capabilityproof.Current{Binding: canonicalBinding(s.t, target.RuntimeID, "2.1.296", target.Process), Model: modeModel(), EvidenceRef: s.ref, ProfileActive: true}, nil
}

type store map[string]capabilityproof.Evidence

func (s store) Load(_ context.Context, ref string) (capabilityproof.Evidence, error) {
	e, ok := s[ref]
	if !ok {
		return e, capabilityproof.ErrInvalidEvidence
	}
	return e, nil
}
func (store) Invalidate(context.Context, string, string, [sha256.Size]byte) error {
	panic("revalidation must not invalidate Evidence")
}

func TestRevalidationRejectsProcessModeDrift(t *testing.T) {
	held := storedEvidence(t, modeEvidence(canonicalBinding(t, "claude", "2.1.296", false)))
	s := capabilityproof.Service{Store: store{held.Ref: held.Evidence}, Source: modeSource{t, held.Ref}, Now: func() time.Time { return observedAt.Add(time.Hour) }}
	target := capabilityproof.Target{RuntimeID: "claude", ProfileID: "profile", Variant: capabilityproof.ReadOnlyVariant}
	if ref, code, err := s.Check(context.Background(), target, []string{capabilityproof.Read}, false, false, held.Ref); err != nil || code != "" || ref != held.Ref {
		t.Fatalf("matching mode: %q %q %v", ref, code, err)
	}
	// Dispatch for Process=true computes a different current ModeDigest; the
	// Process=false Evidence is stale for it, even with Process proven there.
	target.Process = true
	view, err := s.Show(context.Background(), target)
	if err != nil || view.Target != target {
		t.Fatalf("drift view: %+v %v", view, err)
	}
	for _, c := range []string{capabilityproof.Read, capabilityproof.Process} {
		if view.States[c] != capabilityproof.Unknown || view.Blockers[c] != "capability_stale" {
			t.Fatalf("%s admitted across Process modes: %s %s", c, view.States[c], view.Blockers[c])
		}
	}
	for _, expected := range []string{"", held.Ref} {
		ref, code, err := s.Check(context.Background(), target, []string{capabilityproof.Read}, false, true, expected)
		if !errors.Is(err, capabilityproof.ErrBlocked) && !errors.Is(err, capabilityproof.ErrStalePreview) || code != "capability_stale" || ref != "" {
			t.Fatalf("dispatch drift admitted (expected %q): %q %q %v", expected, ref, code, err)
		}
	}
}
