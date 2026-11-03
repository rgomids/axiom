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
	return g, a
}

func TestRetryAuthorityHistoricalCodec(t *testing.T) {
	_, a := retryAuthorityFixture(t, "process_failed")
	wire, err := EncodeRetryAuthority(a)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := DecodeRetryAuthority(wire)
	if err != nil || decoded.AttemptNumber != a.AttemptNumber {
		t.Fatal(err)
	}
	if _, err = DecodeRetryAuthority(append([]byte(`{"unknown":1,`), wire[1:]...)); err == nil {
		t.Fatal("unknown field accepted")
	}
	for _, mutate := range []func(*RetryAuthority){
		func(a *RetryAuthority) { a.AttemptNumber = 1 },
		func(a *RetryAuthority) { a.Disposition = "wipe" },
		func(a *RetryAuthority) { a.ActorKind = "agent" },
		func(a *RetryAuthority) { a.GraphDigest = "invalid" },
	} {
		bad := a
		mutate(&bad)
		if _, err := EncodeRetryAuthority(bad); err == nil {
			t.Fatal("malformed historical authority accepted")
		}
	}
}
