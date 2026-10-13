package executiongraph

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

func reconcileFixture(t *testing.T, name string) (SuccessorGraph, ReconciliationFact, ReconciliationObservations) {
	t.Helper()
	g := successorVector(t, name)
	g.Children[0].Envelope.Controls.MaximumAttempts = 3
	a := g.Children[0].Attempts[0]
	g.Children[0].Attempts[0].ReconciliationFacts = nil
	o := ReconciliationObservations{ParentID: g.Parent.ExecutionID, ChildID: g.Children[0].ExecutionID, GraphRevision: g.Parent.GraphRevision, AttemptID: a.AttemptID, LeaseEpoch: a.LeaseEpoch, WorktreeDigest: strings.Repeat("1", 64), CanonicalRecordsDigest: strings.Repeat("2", 64), QuarantinedInboxDigest: strings.Repeat("3", 64)}
	d, err := o.Digest()
	if err != nil {
		t.Fatal(err)
	}
	f := ReconciliationFact{FactID: "00000000-0000-4000-8000-000000000030", ParentID: o.ParentID, ChildID: o.ChildID, GraphRevision: o.GraphRevision, AttemptID: o.AttemptID, ObservationsDigest: d, Classification: "retry_safe", AuthorityRef: strings.Repeat("d", 64), ActorKind: "human-operator", Actor: "operator", RecordedAt: time.Unix(13, 0).UTC()}
	return g, f, o
}

func TestAttemptHealth(t *testing.T) {
	a := successorVector(t, "running").Children[0].Attempts[0]
	reused := *a.Process
	reused.StartIdentity = "different-start"
	for _, tc := range []struct {
		name     string
		held     bool
		observed *ProcessCorrelation
		err      error
		want     AttemptHealth
	}{
		{"live lease", true, a.Process, nil, AttemptHealth{}},
		{"stale", false, nil, nil, AttemptHealth{Stale: true}},
		{"orphan", false, a.Process, nil, AttemptHealth{Stale: true, Orphan: true}},
		{"PID reused", false, &reused, nil, AttemptHealth{Stale: true}},
		{"unknown", false, nil, errors.New("observation"), AttemptHealth{Stale: true, LivenessUnknown: true}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := ObserveAttemptHealth(a, tc.held, tc.observed, tc.err); got != tc.want {
				t.Fatalf("got %+v want %+v", got, tc.want)
			}
		})
	}
	a.Status = AttemptUnknown
	if got := ObserveAttemptHealth(a, false, a.Process, nil); got != (AttemptHealth{}) {
		t.Fatal(got)
	}
}

func TestReconciliationAppendRetryAndConflict(t *testing.T) {
	g, f, o := reconcileFixture(t, "process_outcome_unknown")
	if ok, err := SuccessorRetryEligible(g, f.ChildID, true); err != nil || ok {
		t.Fatal(ok, err)
	}
	next, err := AppendReconciliationFact(g, f, o)
	if err != nil {
		t.Fatal(err)
	}
	if len(g.Children[0].Attempts[0].ReconciliationFacts) != 0 {
		t.Fatal("aliased caller state")
	}
	a := next.Children[0].Attempts[0]
	if a.Status != AttemptUnknown || !a.AmbiguousEffect || a.ClassificationReason != ProcessOutcomeUnknown {
		t.Fatal("rewrote classification")
	}
	if ok, err := SuccessorRetryEligible(next, f.ChildID, true); err != nil || !ok {
		t.Fatal(ok, err)
	}
	if ok, _ := SuccessorRetryEligible(next, f.ChildID, false); ok {
		t.Fatal("implicit retry")
	}
	f.FactID = "00000000-0000-4000-8000-000000000031"
	duplicate, err := AppendReconciliationFact(next, f, o)
	if err != nil {
		t.Fatal(err)
	}
	f.FactID = "00000000-0000-4000-8000-000000000032"
	f.Classification = "recovery_required"
	conflict, err := AppendReconciliationFact(duplicate, f, o)
	if err != nil {
		t.Fatal(err)
	}
	safe, contradictory := ReconciliationDisposition(conflict.Children[0].Attempts[0])
	if safe || !contradictory || len(conflict.Children[0].Attempts[0].ReconciliationFacts) != 3 {
		t.Fatal("lost contradiction/duplicates")
	}
	if ok, _ := SuccessorRetryEligible(conflict, f.ChildID, true); ok {
		t.Fatal("contradiction authorized retry")
	}
	b, err := EncodeSuccessorGraph(conflict)
	if err != nil {
		t.Fatal(err)
	}
	restarted, err := DecodeSuccessorGraph(b)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(restarted.Children, conflict.Children) {
		t.Fatal("restart lost identity/facts")
	}
	if ok, _ := SuccessorRetryEligible(restarted, f.ChildID, true); ok {
		t.Fatal("restart changed eligibility")
	}
}

func TestReconciliationStaleAndBindingFailures(t *testing.T) {
	g, f, o := reconcileFixture(t, "running")
	next, err := AppendReconciliationFact(g, f, o)
	if err != nil {
		t.Fatal(err)
	}
	if next.Children[0].Attempts[0].Status != AttemptRunning {
		t.Fatal("stale status rewritten")
	}
	if ok, _ := SuccessorRetryEligible(next, f.ChildID, true); ok {
		t.Fatal("redispatched running")
	}
	for _, tc := range []struct {
		name   string
		change func(*ReconciliationFact, *ReconciliationObservations)
	}{
		{"live lease", func(f *ReconciliationFact, o *ReconciliationObservations) {
			o.LeaseHeld = true
			f.ObservationsDigest, _ = o.Digest()
		}},
		{"orphan retry", func(f *ReconciliationFact, o *ReconciliationObservations) {
			o.ProcessAlive = true
			f.ObservationsDigest, _ = o.Digest()
		}},
		{"unknown process", func(f *ReconciliationFact, o *ReconciliationObservations) {
			o.ProcessUnknown = true
			f.ObservationsDigest, _ = o.Digest()
		}},
		{"wrong graph revision", func(f *ReconciliationFact, o *ReconciliationObservations) { f.GraphRevision++ }},
		{"wrong parent", func(f *ReconciliationFact, o *ReconciliationObservations) { f.ParentID = f.ChildID }},
		{"wrong attempt", func(f *ReconciliationFact, o *ReconciliationObservations) { f.AttemptID = f.ChildID }},
		{"wrong epoch", func(f *ReconciliationFact, o *ReconciliationObservations) {
			o.LeaseEpoch++
			f.ObservationsDigest, _ = o.Digest()
		}},
		{"stale observations", func(f *ReconciliationFact, o *ReconciliationObservations) { o.WorktreeDigest = strings.Repeat("4", 64) }},
		{"dispatch authority reused", func(f *ReconciliationFact, o *ReconciliationObservations) { f.AuthorityRef = g.DispatchAuthorityRef }},
		{"missing authority", func(f *ReconciliationFact, o *ReconciliationObservations) { f.AuthorityRef = "" }},
		{"wrong actor", func(f *ReconciliationFact, o *ReconciliationObservations) { f.ActorKind = "agent" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			badF, badO := f, o
			tc.change(&badF, &badO)
			if _, err := AppendReconciliationFact(g, badF, badO); err == nil {
				t.Fatal("accepted")
			}
		})
	}
	terminal, f, o := reconcileFixture(t, "process_outcome_unknown")
	reconciled, err := AppendReconciliationFact(terminal, f, o)
	if err != nil {
		t.Fatal(err)
	}
	reconciled.Children[0].Attempts[0].Status = AttemptSucceeded
	reconciled.Children[0].Attempts[0].AmbiguousEffect = false
	reconciled.Children[0].Attempts[0].ClassificationReason = ResultRecorded
	reconciled.Children[0].Attempts[0].QualifyingRecordRef = strings.Repeat("a", 64)
	if ValidateSuccessorUpdate(terminal, reconciled) == nil {
		t.Fatal("rewrote terminal status")
	}
	if ValidateLeaseEpoch(terminal, 0) == nil {
		t.Fatal("accepted lower epoch")
	}
	terminal.Children[0].Attempts[0].LeaseEpoch = 2
	if ValidateLeaseEpoch(terminal, 1) == nil {
		t.Fatal("accepted stale epoch")
	}
}

func TestGatherReconciliationObservations(t *testing.T) {
	g, _, o := reconcileFixture(t, "running")
	s := ReconciliationSources{LeaseProbe: func() (bool, error) { return false, nil }, ProcessProbe: func(ProcessCorrelation) (*ProcessCorrelation, error) { return nil, nil }, Worktree: func() (string, error) { return o.WorktreeDigest, nil }, CanonicalRecords: func() (string, error) { return o.CanonicalRecordsDigest, nil }, QuarantinedInbox: func() (string, error) { return o.QuarantinedInboxDigest, nil }}
	got, err := GatherReconciliationObservations(g, o.ChildID, o.AttemptID, s)
	if err != nil || got != o {
		t.Fatal(got, err)
	}
	s.ProcessProbe = func(ProcessCorrelation) (*ProcessCorrelation, error) { return nil, errors.New("denied") }
	got, err = GatherReconciliationObservations(g, o.ChildID, o.AttemptID, s)
	if err != nil || !got.ProcessUnknown {
		t.Fatal(got, err)
	}
	s.Worktree = func() (string, error) { return "", errors.New("unavailable") }
	if _, err = GatherReconciliationObservations(g, o.ChildID, o.AttemptID, s); err == nil {
		t.Fatal("ignored worktree failure")
	}
}

func TestReconciliationSchedulerAndIntegrationReadPaths(t *testing.T) {
	g, f, o := reconcileFixture(t, "process_outcome_unknown")
	ready, blocked, err := ReadySuccessorChildren(g, map[string]bool{f.ChildID: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range ready {
		if id == f.ChildID {
			t.Fatal("unknown dispatched without fact")
		}
	}
	if blocked[f.ChildID] != "retry_not_eligible" {
		t.Fatal(blocked)
	}
	next, err := AppendReconciliationFact(g, f, o)
	if err != nil {
		t.Fatal(err)
	}
	ready, _, err = ReadySuccessorChildren(next, map[string]bool{f.ChildID: true})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, id := range ready {
		found = found || id == f.ChildID
	}
	if !found {
		t.Fatal("persisted fact ignored")
	}
	// Move the same unknown attempt/fact to the integration owner and use its
	// persisted integration reason. No process classifier is called.
	integration := successorVector(t, "integration_unknown")
	for i, c := range integration.Children {
		if c.Envelope.IntegrationOwner {
			integration.Children[i].Envelope.Controls.MaximumAttempts = 3
			a := integration.Children[i].Attempts[0]
			fact := f
			fact.ChildID = c.ExecutionID
			fact.AttemptID = a.AttemptID
			observations := o
			observations.ChildID = c.ExecutionID
			observations.AttemptID = a.AttemptID
			observations.LeaseEpoch = a.LeaseEpoch
			fact.ObservationsDigest, _ = observations.Digest()
			if ok, err := SuccessorIntegrationAttemptAvailable(integration); err != nil || ok {
				t.Fatal(ok, err)
			}
			reconciled, err := AppendReconciliationFact(integration, fact, observations)
			if err != nil {
				t.Fatal(err)
			}
			if ok, err := SuccessorIntegrationAttemptAvailable(reconciled); err != nil || !ok {
				t.Fatal(ok, err)
			}
			if reconciled.Children[i].Attempts[0].Status != AttemptUnknown {
				t.Fatal("fact promoted integration success")
			}
		}
	}
}
