package executiongraph

import (
	"strings"
	"testing"
)

func boundAuthority(t *testing.T) (SuccessorGraph, DispatchAuthority) {
	g := successorVector(t, "running")
	for i := range g.Children {
		g.Children[i].Attempts = nil
	}
	a := DispatchAuthority{FormatVersion: 1, ProjectID: g.Children[0].Envelope.Scope.ProjectID, PreviewDigest: strings.Repeat("a", 64), Scope: "graph-first-attempts", ParentID: g.Parent.ExecutionID, Revision: g.Parent.GraphRevision, ProposalDigest: g.Parent.ProposalDigest, ActorKind: "human-operator", Actor: "operator", RecordedAt: g.Parent.CreatedAt}
	for _, c := range g.Children {
		a.Children = append(a.Children, DispatchChildAuthority{ChildID: c.ExecutionID, Effects: c.Envelope.AllowedEffects})
	}
	d, err := DispatchAuthorityDigest(a)
	if err != nil {
		t.Fatal(err)
	}
	g.DispatchAuthorityRef = d
	for i := range g.Children {
		g.Children[i].Envelope.AuthorityReference = d
	}
	return g, a
}

func TestDispatchAuthorityClosedCodec(t *testing.T) {
	_, a := boundAuthority(t)
	b, _ := EncodeDispatchAuthority(a)
	for _, wire := range [][]byte{append([]byte(`{"unknown":true,`), b[1:]...), append([]byte(`{"formatVersion":1,`), b[1:]...)} {
		if _, err := DecodeDispatchAuthority(wire); err == nil {
			t.Fatal("untrusted field accepted")
		}
	}
	a.Children[0].Effects = append(a.Children[0].Effects, Effect{Kind: "unapproved", Target: "x"})
	if ValidateDispatchAuthority(a) == nil {
		t.Fatal("unknown effect accepted")
	}
}
