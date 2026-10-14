package capabilityproof

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

type fakeScanner struct {
	records []StoredEvidence
	err     error
	scans   int
}

func (f *fakeScanner) Scan(ctx context.Context, visit func(StoredEvidence) error) error {
	f.scans++
	for _, r := range f.records {
		if err := visit(r); err != nil {
			return err
		}
	}
	return f.err
}

func stored(e Evidence) StoredEvidence {
	original := e
	original.InvalidatedReason = ""
	ref, _ := Digest(original)
	return StoredEvidence{Ref: ref, Evidence: e}
}

func selectionModel() *ModelBinding {
	return &ModelBinding{"profile", strings.Repeat("b", 64), "model", "explicit", "high", 1, strings.Repeat("c", 64)}
}

func profileEvidence(variant string) Evidence {
	e := testEvidence()
	e.Binding.Variant = variant
	e.Model = selectionModel()
	e.Capabilities[ModelInvocation] = e.Capabilities[Read]
	return e
}

func selector(records ...StoredEvidence) (Selector, *fakeScanner) {
	scanner := &fakeScanner{records: records}
	now := testEvidence().ObservedAt.Add(time.Hour)
	return Selector{Evidence: scanner, Now: func() time.Time { return now }}, scanner
}

func TestEvidenceSelectionUniqueRuntimeLevel(t *testing.T) {
	e := testEvidence()
	s, _ := selector(stored(e))
	ref, code, err := s.Select(context.Background(), e.Binding, nil)
	want, _ := Digest(e)
	if err != nil || code != "" || ref != want {
		t.Fatalf("%q %q %v", ref, code, err)
	}
	// A Runtime-level request never selects Profile-level evidence and vice versa.
	if _, code, err = s.Select(context.Background(), e.Binding, selectionModel()); !errors.Is(err, ErrEvidenceMissing) || code != "capability_unknown" {
		t.Fatal(code, err)
	}
}

func TestEvidenceSelectionUniqueProfileLevel(t *testing.T) {
	runtimeLevel := testEvidence()
	profileLevel := profileEvidence(ReadOnlyVariant)
	s, _ := selector(stored(runtimeLevel), stored(profileLevel))
	ref, code, err := s.Select(context.Background(), profileLevel.Binding, selectionModel())
	if err != nil || code != "" || ref != stored(profileLevel).Ref {
		t.Fatalf("%q %q %v", ref, code, err)
	}
	ref, _, err = s.Select(context.Background(), runtimeLevel.Binding, nil)
	if err != nil || ref != stored(runtimeLevel).Ref {
		t.Fatal(ref, err)
	}
}

func TestEvidenceSelectionPerVariant(t *testing.T) {
	readOnly := profileEvidence(ReadOnlyVariant)
	write := profileEvidence(RepositoryWriteVariant)
	write.Capabilities[RepositoryWrite] = write.Capabilities[Read]
	s, _ := selector(stored(readOnly), stored(write))
	for _, e := range []Evidence{readOnly, write} {
		t.Run(e.Binding.Variant, func(t *testing.T) {
			ref, code, err := s.Select(context.Background(), e.Binding, selectionModel())
			if err != nil || code != "" || ref != stored(e).Ref {
				t.Fatalf("%q %q %v", ref, code, err)
			}
		})
	}
}

func TestEvidenceSelectionOrderIndependent(t *testing.T) {
	match := profileEvidence(ReadOnlyVariant)
	other := profileEvidence(RepositoryWriteVariant)
	expired := profileEvidence(ReadOnlyVariant)
	expired.ObservedAt = match.ObservedAt.Add(-MaxAge - time.Hour)
	drift := profileEvidence(ReadOnlyVariant)
	drift.Binding.Version = "2.0"
	records := []StoredEvidence{stored(match), stored(other), stored(expired), stored(drift)}
	orders := [][]int{{0, 1, 2, 3}, {3, 2, 1, 0}, {1, 3, 0, 2}, {2, 0, 3, 1}}
	for _, order := range orders {
		permuted := make([]StoredEvidence, 0, len(order))
		for _, i := range order {
			permuted = append(permuted, records[i])
		}
		s, _ := selector(permuted...)
		ref, code, err := s.Select(context.Background(), match.Binding, selectionModel())
		if err != nil || code != "" || ref != records[0].Ref {
			t.Fatalf("order %v: %q %q %v", order, ref, code, err)
		}
	}
}

func TestEvidenceSelectionExactBindingDrift(t *testing.T) {
	d := strings.Repeat("e", 64)
	cases := map[string]func(*Binding){
		"runtime":         func(b *Binding) { b.RuntimeID = "claude" },
		"executable":      func(b *Binding) { b.ExecutableDigest = d },
		"version":         func(b *Binding) { b.Version = "2.0" },
		"mode":            func(b *Binding) { b.ModeDigest = d },
		"variant":         func(b *Binding) { b.Variant = RepositoryWriteVariant },
		"adapter-version": func(b *Binding) { b.AdapterVersion = "2" },
		"axiom":           func(b *Binding) { b.AxiomDigest = d },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			e := profileEvidence(ReadOnlyVariant)
			s, _ := selector(stored(e))
			current := e.Binding
			mutate(&current)
			if current == e.Binding {
				t.Fatal("mutation had no effect")
			}
			ref, code, err := s.Select(context.Background(), current, selectionModel())
			if ref != "" || code != "capability_unknown" || !errors.Is(err, ErrEvidenceMissing) {
				t.Fatalf("%q %q %v", ref, code, err)
			}
		})
	}
}

func TestEvidenceSelectionExactModelDrift(t *testing.T) {
	d := strings.Repeat("e", 64)
	cases := map[string]func(*ModelBinding){
		"profile-id":             func(m *ModelBinding) { m.ProfileID = "other" },
		"profile-digest":         func(m *ModelBinding) { m.ProfileDigest = d },
		"model":                  func(m *ModelBinding) { m.Model = "other-model" },
		"effort-mode":            func(m *ModelBinding) { m.EffortMode, m.EffortToken = "runtime-default", "default" },
		"effort-token":           func(m *ModelBinding) { m.EffortToken = "low" },
		"configuration-revision": func(m *ModelBinding) { m.ConfigurationRevision = 2 },
		"effort-source":          func(m *ModelBinding) { m.EffortSourceDigest = d },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			e := profileEvidence(ReadOnlyVariant)
			s, _ := selector(stored(e))
			current := selectionModel()
			mutate(current)
			ref, code, err := s.Select(context.Background(), e.Binding, current)
			if ref != "" || code != "capability_unknown" || !errors.Is(err, ErrEvidenceMissing) {
				t.Fatalf("%q %q %v", ref, code, err)
			}
		})
	}
}

func TestEvidenceSelectionNoMatch(t *testing.T) {
	e := testEvidence()
	s, scanner := selector()
	ref, code, err := s.Select(context.Background(), e.Binding, nil)
	if ref != "" || code != "capability_unknown" || !errors.Is(err, ErrEvidenceMissing) || !errors.Is(err, ErrBlocked) || scanner.scans != 1 {
		t.Fatalf("%q %q %v", ref, code, err)
	}
}

func TestEvidenceSelectionAmbiguousFailsClosed(t *testing.T) {
	older := profileEvidence(ReadOnlyVariant)
	newer := profileEvidence(ReadOnlyVariant)
	newer.ObservedAt = older.ObservedAt.Add(time.Minute)
	for _, records := range [][]StoredEvidence{{stored(older), stored(newer)}, {stored(newer), stored(older)}} {
		s, _ := selector(records...)
		ref, code, err := s.Select(context.Background(), older.Binding, selectionModel())
		if ref != "" || code != "capability_unknown" || !errors.Is(err, ErrEvidenceAmbiguous) || errors.Is(err, ErrEvidenceMissing) {
			t.Fatalf("%q %q %v", ref, code, err)
		}
	}
}

func TestEvidenceSelectionExcludesInvalidatedAndExpired(t *testing.T) {
	invalidated := profileEvidence(ReadOnlyVariant)
	invalidated.InvalidatedReason = "explicit"
	expired := profileEvidence(ReadOnlyVariant)
	expired.ObservedAt = invalidated.ObservedAt.Add(-MaxAge - time.Hour)
	future := profileEvidence(ReadOnlyVariant)
	future.ObservedAt = invalidated.ObservedAt.Add(2 * time.Hour)
	for name, e := range map[string]Evidence{"invalidated": invalidated, "expired": expired, "future": future} {
		t.Run(name, func(t *testing.T) {
			record := stored(e)
			s, _ := selector(record)
			ref, code, err := s.Select(context.Background(), e.Binding, selectionModel())
			if ref != "" || code != "capability_unknown" || !errors.Is(err, ErrEvidenceMissing) {
				t.Fatalf("%q %q %v", ref, code, err)
			}
			// Selection neither revives nor alters the excluded record.
			if record.Evidence.InvalidatedReason != e.InvalidatedReason || !record.Evidence.ObservedAt.Equal(e.ObservedAt) {
				t.Fatal("record mutated")
			}
		})
	}
	// An excluded record never makes a valid match ambiguous.
	match := profileEvidence(ReadOnlyVariant)
	match.ProbePacketDigest = strings.Repeat("f", 64)
	s, _ := selector(stored(invalidated), stored(expired), stored(match))
	if ref, _, err := s.Select(context.Background(), match.Binding, selectionModel()); err != nil || ref != stored(match).Ref {
		t.Fatal(ref, err)
	}
}

func TestEvidenceSelectionIntegrityFailsClosed(t *testing.T) {
	e := profileEvidence(ReadOnlyVariant)
	good := stored(e)
	wrongRef := good
	wrongRef.Ref = strings.Repeat("0", 64)
	tampered := good
	tampered.Evidence.Binding.Version = "9.9"
	unsupported := good
	unsupported.Evidence.SchemaVersion = 2
	cases := map[string]*fakeScanner{
		"invalid-content-address": {records: []StoredEvidence{good, wrongRef}},
		"corrupted-content":       {records: []StoredEvidence{tampered}},
		"unsupported-schema":      {records: []StoredEvidence{unsupported, good}},
		"duplicate-entry":         {records: []StoredEvidence{good, good}},
		"unreadable-candidate":    {records: []StoredEvidence{good}, err: errors.New("/private/state/capability-evidence: permission denied")},
	}
	for name, scanner := range cases {
		t.Run(name, func(t *testing.T) {
			s := Selector{Evidence: scanner, Now: func() time.Time { return e.ObservedAt.Add(time.Hour) }}
			ref, code, err := s.Select(context.Background(), e.Binding, selectionModel())
			// The generic blocker hides storage details and is neither a miss nor ambiguity.
			if ref != "" || code != "capability_unknown" || err != ErrBlocked {
				t.Fatalf("%q %q %v", ref, code, err)
			}
		})
	}
}

func TestEvidenceSelectionRequiresIndependentCurrentIdentity(t *testing.T) {
	e := profileEvidence(ReadOnlyVariant)
	s, scanner := selector(stored(e))
	// Persisted Evidence is never a source of current identity: without an
	// independently observed Binding nothing is enumerated or selected.
	for name, b := range map[string]Binding{"zero": {}, "partial": {RuntimeID: e.Binding.RuntimeID, Variant: e.Binding.Variant}} {
		ref, code, err := s.Select(context.Background(), b, selectionModel())
		if ref != "" || code != "capability_unknown" || err != ErrBlocked {
			t.Fatalf("%s: %q %q %v", name, ref, code, err)
		}
	}
	if scanner.scans != 0 {
		t.Fatal("scanned without current identity", scanner.scans)
	}
	if _, _, err := (Selector{}).Select(context.Background(), e.Binding, nil); err != ErrBlocked {
		t.Fatal("nil scanner", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, _, err := s.Select(ctx, e.Binding, selectionModel()); !errors.Is(err, context.Canceled) || scanner.scans != 0 {
		t.Fatal("cancelled", err)
	}
}

// expiringContext reports a deadline only once the scanner expires it, so the
// test observes expiry during enumeration without wall-clock timing.
type expiringContext struct {
	context.Context
	err error
}

func (c *expiringContext) Err() error { return c.err }

// interruptingScanner visits its first record, interrupts the context and
// aborts as a real scanner does when it observes ctx.Err between entries.
type interruptingScanner struct {
	records   []StoredEvidence
	interrupt func()
	wrap      bool
	visited   int
}

func (f *interruptingScanner) Scan(ctx context.Context, visit func(StoredEvidence) error) error {
	for _, r := range f.records {
		if err := ctx.Err(); err != nil {
			if f.wrap {
				return fmt.Errorf("/private/state/capability-evidence/%s: %w", r.Ref, err)
			}
			return err
		}
		if err := visit(r); err != nil {
			return err
		}
		f.visited++
		f.interrupt()
	}
	return nil
}

func TestEvidenceSelectionPreservesContextErrorsDuringScan(t *testing.T) {
	e := profileEvidence(ReadOnlyVariant)
	expired := profileEvidence(ReadOnlyVariant)
	expired.ObservedAt = e.ObservedAt.Add(-MaxAge - time.Hour)
	now := func() time.Time { return e.ObservedAt.Add(time.Hour) }
	// Uninterrupted, the same enumeration selects e uniquely.
	baseline := &interruptingScanner{records: []StoredEvidence{stored(e), stored(expired)}, interrupt: func() {}}
	if ref, _, err := (Selector{Evidence: baseline, Now: now}).Select(context.Background(), e.Binding, selectionModel()); err != nil || ref != stored(e).Ref {
		t.Fatal("baseline", ref, err)
	}
	for _, wrap := range []bool{false, true} {
		cancelCtx, cancel := context.WithCancel(context.Background())
		expiring := &expiringContext{Context: context.Background()}
		cases := []struct {
			name      string
			ctx       context.Context
			interrupt func()
			want      error
		}{
			{"canceled", cancelCtx, cancel, context.Canceled},
			{"deadline", expiring, func() { expiring.err = context.DeadlineExceeded }, context.DeadlineExceeded},
		}
		for _, c := range cases {
			t.Run(fmt.Sprintf("%s/wrapped=%t", c.name, wrap), func(t *testing.T) {
				records := []StoredEvidence{stored(e), stored(expired)}
				scanner := &interruptingScanner{records: records, interrupt: c.interrupt, wrap: wrap}
				s := Selector{Evidence: scanner, Now: now}
				ref, code, err := s.Select(c.ctx, e.Binding, selectionModel())
				if ref != "" || code != "capability_unknown" || !errors.Is(err, c.want) || scanner.visited != 1 {
					t.Fatalf("%q %q %v visited=%d", ref, code, err, scanner.visited)
				}
				// Neither a miss nor ambiguity, and no storage detail leaks.
				if errors.Is(err, ErrBlocked) || strings.Contains(err.Error(), "/private") {
					t.Fatal("context error lost identity or leaked detail", err)
				}
				if records[0].Evidence.InvalidatedReason != "" || !records[0].Evidence.ObservedAt.Equal(e.ObservedAt) {
					t.Fatal("record mutated")
				}
			})
		}
		cancel()
	}
	// A non-context scanner failure still fails closed with the generic blocker.
	s := Selector{Evidence: &fakeScanner{records: []StoredEvidence{stored(e)}, err: errors.New("i/o")}, Now: now}
	if ref, code, err := s.Select(context.Background(), e.Binding, selectionModel()); ref != "" || code != "capability_unknown" || err != ErrBlocked {
		t.Fatalf("%q %q %v", ref, code, err)
	}
}
