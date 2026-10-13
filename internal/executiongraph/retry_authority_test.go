package executiongraph

import (
	"crypto/sha256"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

func retryAuthorityFixture(t *testing.T, name string) (SuccessorGraph, RetryAuthority) {
	t.Helper()
	wire, err := os.ReadFile("testdata/successor/valid/" + name + ".json")
	if err != nil {
		t.Fatal(err)
	}
	g, err := DecodeSuccessorGraph(wire)
	if err != nil {
		t.Fatal(err)
	}
	g.Children[0].Envelope.Controls.MaximumAttempts = 3
	wire, err = EncodeSuccessorGraph(g)
	if err != nil {
		t.Fatal(err)
	}
	g.StorageRevision = sha256.Sum256(wire)
	c := g.Children[0]
	a := RetryAuthority{FormatVersion: 1, ProjectID: c.Envelope.Scope.ProjectID, ParentID: g.Parent.ExecutionID, Revision: g.Parent.GraphRevision, GraphDigest: fmt.Sprintf("%x", g.StorageRevision), PreviewDigest: strings.Repeat("a", 64), DispatchDigest: strings.Repeat("b", 64), ChildID: c.ExecutionID, PriorAttemptID: c.Attempts[len(c.Attempts)-1].AttemptID, AttemptNumber: uint32(len(c.Attempts)) + 1, Effects: c.Envelope.AllowedEffects, Disposition: DispositionNotRequired, ActorKind: "human-operator", Actor: "operator", RecordedAt: time.Unix(90, 0).UTC()}
	if RetryDispositionRequired(c) {
		a.Disposition = DispositionPreserve
	}
	return g, a
}

func TestRetryAuthorityCodecAndDispatchCheck(t *testing.T) {
	g, a := retryAuthorityFixture(t, "process_failed")
	wire, err := EncodeRetryAuthority(a)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := DecodeRetryAuthority(wire)
	if err != nil || decoded.AttemptNumber != a.AttemptNumber {
		t.Fatal(err)
	}
	if n, err := CheckRetryDispatch(g, a); err != nil || n != 2 {
		t.Fatal(n, err)
	}
	if _, err = DecodeRetryAuthority(append([]byte(`{"unknown":1,`), wire[1:]...)); err == nil {
		t.Fatal("unknown field accepted")
	}
	mutations := map[string]func(*SuccessorGraph, *RetryAuthority){
		"first-attempt-number": func(_ *SuccessorGraph, a *RetryAuthority) { a.AttemptNumber = 1 },
		"counter-reset":        func(_ *SuccessorGraph, a *RetryAuthority) { a.AttemptNumber = 3 },
		"stale-graph":          func(g *SuccessorGraph, _ *RetryAuthority) { g.StorageRevision[0] ^= 1 },
		"foreign-prior":        func(_ *SuccessorGraph, a *RetryAuthority) { a.PriorAttemptID = "00000000-0000-4000-8000-000000000999" },
		"disposition":          func(_ *SuccessorGraph, a *RetryAuthority) { a.Disposition = DispositionReset },
		"unknown-disposition":  func(_ *SuccessorGraph, a *RetryAuthority) { a.Disposition = "wipe" },
		"exhausted": func(g *SuccessorGraph, a *RetryAuthority) {
			g.Children[0].Envelope.Controls.MaximumAttempts = 1
			wire, _ := EncodeSuccessorGraph(*g)
			g.StorageRevision = sha256.Sum256(wire)
			a.GraphDigest = fmt.Sprintf("%x", g.StorageRevision)
		},
		"cancelled-graph": func(g *SuccessorGraph, _ *RetryAuthority) {
			g.CancelRequest = &CancelRequest{RequestID: "00000000-0000-4000-8000-000000000998", ParentID: g.Parent.ExecutionID, GraphRevision: 1, ActorKind: "human-operator", Actor: "operator", RequestedAt: time.Unix(95, 0).UTC(), AuthorityRef: strings.Repeat("c", 64)}
		},
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			g, a := retryAuthorityFixture(t, "process_failed")
			mutate(&g, &a)
			if _, err := CheckRetryDispatch(g, a); err == nil {
				t.Fatal("retry authority accepted", name)
			}
		})
	}
	for _, name := range []string{"result_recorded", "running", "process_outcome_unknown"} {
		t.Run("ineligible-"+name, func(t *testing.T) {
			g, a := retryAuthorityFixture(t, name)
			if _, err := CheckRetryDispatch(g, a); err == nil {
				t.Fatal("ineligible status retried", name)
			}
		})
	}
}
