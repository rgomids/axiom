package local

import (
	"context"
	"errors"
	"github.com/rgomids/axiom/internal/executiongraph"
	"github.com/rgomids/axiom/internal/workflow"
	"os"
	"strings"
	"testing"
	"time"
)

func localDispatch(t *testing.T) (executiongraph.SuccessorGraph, executiongraph.DispatchAuthority) {
	g := localSuccessor(t, "running")
	for i := range g.Children {
		g.Children[i].Attempts = nil
	}
	a := executiongraph.DispatchAuthority{FormatVersion: 1, ProjectID: g.Children[0].Envelope.Scope.ProjectID, PreviewDigest: strings.Repeat("a", 64), Scope: "graph-first-attempts", ParentID: g.Parent.ExecutionID, Revision: g.Parent.GraphRevision, ProposalDigest: g.Parent.ProposalDigest, ActorKind: "human-operator", Actor: "operator", RecordedAt: g.Parent.CreatedAt}
	for _, c := range g.Children {
		a.Children = append(a.Children, executiongraph.DispatchChildAuthority{ChildID: c.ExecutionID, Effects: c.Envelope.AllowedEffects})
	}
	d, e := executiongraph.DispatchAuthorityDigest(a)
	if e != nil {
		t.Fatal(e)
	}
	g.DispatchAuthorityRef = d
	for i := range g.Children {
		g.Children[i].Envelope.AuthorityReference = d
	}
	return g, a
}

func TestDispatchCompanionFaultWindows(t *testing.T) {
	for _, family := range []string{"graphs", "executions"} {
		for _, fault := range []FaultStage{FaultF0, FaultF1, FaultF2, FaultF3, FaultF4, FaultF5, FaultF6, FaultF7, FaultF8} {
			t.Run(family+"/"+string(fault), func(t *testing.T) {
				root, e := os.OpenRoot(t.TempDir())
				if e != nil {
					t.Fatal(e)
				}
				defer root.Close()
				g, a := localDispatch(t)
				graph, _ := executiongraph.EncodeSuccessorGraph(g)
				limit := executiongraph.MaxGraphBytes
				name := graphName(a.ParentID)
				if family == "executions" {
					wire, e := os.ReadFile("../workflowcompiler/testdata/issue306/execution-v3.json")
					if e != nil {
						t.Fatal(e)
					}
					state, e := decodeExecution(wire)
					if e != nil {
						t.Fatal(e)
					}
					graph, e = encodeExecution(state)
					if e != nil {
						t.Fatal(e)
					}
					a.Scope = "single-first-attempt"
					a.ParentID = state.ExecutionID
					a.ProjectID = state.ProjectID
					a.Revision = state.Revision
					a.Children = []executiongraph.DispatchChildAuthority{{ChildID: state.ExecutionID, Effects: []executiongraph.Effect{}}}
					a.SingleAttemptID = "00000000-0000-4000-8000-000000000050"
					a.SingleStageID = string(state.Stage)
					a.SingleStageOrdinal = state.StageOrdinal
					a.LedgerDigest = hexDigest(graph)
					g.DispatchAuthorityRef, e = executiongraph.DispatchAuthorityDigest(a)
					if e != nil {
						t.Fatal(e)
					}
					limit = MaxExecutionBytes
					name = executionName(state.RepositoryKey, state.WorkItem)
				}
				receipt, _ := executiongraph.EncodeDispatchAuthority(a)
				hooks := publicationHooks{contentLimit: limit, companion: g.DispatchAuthorityRef + ".json", companionWire: receipt, fault: func(stage FaultStage) error {
					if stage == fault {
						return ErrSimulatedInterruption
					}
					return nil
				}}
				e = publishFile(context.Background(), root, name, nil, graph, true, hooks)
				var pe *PublicationError
				if !errors.As(e, &pe) {
					t.Fatalf("fault not reported: %v", e)
				}
				committed := fault == FaultF6 || fault == FaultF7 || fault == FaultF8
				if pe.Committed != committed {
					t.Fatalf("commit truth %v", pe)
				}
				if committed {
					if e := confirmDispatch(root, name, graph, hooks.companion, receipt, limit); e != nil {
						t.Fatal(e)
					}
				}
				if fault == FaultF0 || fault == FaultF1 || fault == FaultF2 {
					plan, present, e := planRecovery(root, RecoveryScopeState, family+"/v1/"+a.ProjectID, "")
					if e != nil {
						t.Fatal(e)
					}
					if fault == FaultF2 && (!present || plan.Action != PreservedReview) {
						t.Fatalf("unmarked stage lost %+v", plan)
					}
					if fault != FaultF2 && present {
						t.Fatalf("preparation reported authority %+v", plan)
					}
				}
				if fault == FaultF3 || fault == FaultF4 || fault == FaultF5 || committed {
					names, e := readDirectoryNamesBounded(root, maxLocalDirectoryEntries)
					if e != nil {
						t.Fatal(e)
					}
					found := false
					for _, n := range names {
						if strings.HasPrefix(n, ".axiom-recovery-") {
							m, e := readProtocolMarker(root, n)
							if e != nil || m.Companion != hooks.companion || m.CompanionRevision != hexDigest(receipt) {
								t.Fatalf("recovery does not bind receipt: %+v %v", m, e)
							}
							found = true
						}
					}
					if !found {
						t.Fatal("recovery marker absent")
					}
					plan, present, e := planRecovery(root, RecoveryScopeState, family+"/v1/"+a.ProjectID, "")
					if e != nil || !present {
						t.Fatalf("recovery missing: %v", e)
					}
					expected := RestorePrior
					if committed {
						expected = FinalizeCommitted
					}
					if plan.Action != expected {
						t.Fatalf("recovery action %+v", plan)
					}
					if committed {
						if e := root.Remove(hooks.companion); e != nil {
							t.Fatal(e)
						}
						plan, _, e = planRecovery(root, RecoveryScopeState, family+"/v1/"+a.ProjectID, "")
						if e != nil || plan.Action != PreservedReview {
							t.Fatalf("partial commit accepted %+v %v", plan, e)
						}
					}

					if _, e := readSuccessorPublishedFile(root, name); !errors.Is(e, ErrRecoveryRequired) {
						t.Fatalf("partial graph exposed: %v", e)
					}
				}
			})
		}
	}
}

func TestDispatchCompanionPrecommitReapply(t *testing.T) {
	root, e := os.OpenRoot(t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	defer root.Close()
	g, a := localDispatch(t)
	graph, _ := executiongraph.EncodeSuccessorGraph(g)
	receipt, _ := executiongraph.EncodeDispatchAuthority(a)
	hooks := publicationHooks{contentLimit: executiongraph.MaxGraphBytes, companion: g.DispatchAuthorityRef + ".json", companionWire: receipt, fault: func(stage FaultStage) error {
		if stage == FaultF4 {
			return errors.New("precommit failure")
		}
		return nil
	}}
	if e := publishFile(context.Background(), root, graphName(a.ParentID), nil, graph, true, hooks); e == nil {
		t.Fatal("fault absent")
	}
	hooks.fault = nil
	if e := publishFile(context.Background(), root, graphName(a.ParentID), nil, graph, true, hooks); e != nil {
		t.Fatal(e)
	}
	if e := confirmDispatch(root, graphName(a.ParentID), graph, hooks.companion, receipt, executiongraph.MaxGraphBytes); e != nil {
		t.Fatal(e)
	}
}

func TestDispatchGraphStoreCommitAndReadback(t *testing.T) {
	ctx := context.Background()
	stateRoot := t.TempDir()
	s, e := NewGraphStore(stateRoot)
	if e != nil {
		t.Fatal(e)
	}
	g, a := localDispatch(t)
	leases, e := NewRunLeaseStore(stateRoot)
	if e != nil {
		t.Fatal(e)
	}
	lease, e := leases.Acquire(ctx, a.ProjectID, a.ParentID, 0)
	if e != nil {
		t.Fatal(e)
	}
	defer lease.Close()
	if e := s.CommitDispatch(ctx, g, a, a.PreviewDigest, lease); e != nil {
		t.Fatal(e)
	}
	got, e := s.LoadDispatchAuthority(ctx, a.ProjectID, a.ParentID, a.PreviewDigest)
	if e != nil || got.ParentID != a.ParentID {
		t.Fatalf("readback %+v %v", got, e)
	}
	if e := s.CommitDispatch(ctx, g, a, a.PreviewDigest, lease); !errors.Is(e, ErrConflict) {
		t.Fatalf("reuse %v", e)
	}
}

func TestSingleDispatchStoreCommitAndScope(t *testing.T) {
	ctx := context.Background()
	seed, e := NewWorkflowStore(privateTestRoot(t))
	if e != nil {
		t.Fatal(e)
	}
	state := createLocalOperationalExecution(t, seed)
	state.StorageRevision = [32]byte{}
	ledger, e := encodeExecution(state)
	if e != nil {
		t.Fatal(e)
	}
	a := executiongraph.DispatchAuthority{FormatVersion: 1, ProjectID: state.ProjectID, PreviewDigest: strings.Repeat("a", 64), Scope: "single-first-attempt", ParentID: state.ExecutionID, Revision: state.Revision, ProposalDigest: strings.Repeat("b", 64), Children: []executiongraph.DispatchChildAuthority{{ChildID: state.ExecutionID, Effects: []executiongraph.Effect{}}}, SingleAttemptID: "00000000-0000-4000-8000-000000000050", SingleStageID: string(state.Stage), SingleStageOrdinal: state.StageOrdinal, LedgerDigest: hexDigest(ledger), ActorKind: "human-operator", Actor: "operator", RecordedAt: state.CreatedAt}
	s, e := NewWorkflowStore(t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	if e := s.CommitSingleDispatch(ctx, state, a, a.PreviewDigest); e != nil {
		t.Fatal(e)
	}
	ref, _ := executiongraph.DispatchAuthorityDigest(a)
	if _, e := s.LoadSingleDispatchAuthority(ctx, state, ref, a.PreviewDigest, a.SingleAttemptID); e != nil {
		t.Fatal(e)
	}
	if _, e := s.LoadSingleDispatchAuthority(ctx, state, ref, a.PreviewDigest, "00000000-0000-4000-8000-000000000051"); e == nil {
		t.Fatal("foreign attempt accepted")
	}
	loaded, e := s.Load(ctx, state.ProjectID, state.RepositoryKey, state.WorkItem)
	if e != nil {
		t.Fatal(e)
	}
	// Equal prior/new bytes cannot establish a dispatch commit point.
	if e := s.CommitSingleDispatch(ctx, loaded, a, a.PreviewDigest); !errors.Is(e, ErrConflict) {
		t.Fatalf("no-op ledger accepted: %v", e)
	}
	// Change valid metadata while preserving stage scope. These requests reach
	// the permanent first-attempt scope fence rather than the no-op guard.
	loaded.UpdatedAt = loaded.UpdatedAt.Add(time.Second)
	loaded.Revision++
	loaded.Status = workflow.ExecutionInterrupted
	loaded.Transitions = append(loaded.Transitions, workflow.Transition{Revision: loaded.Revision, From: loaded.Stage, To: loaded.Stage, Outcome: workflow.OutcomeFailed, RequestDigest: strings.Repeat("d", 64), CommittedAt: loaded.UpdatedAt, Provenance: loaded.Provenance})
	updatedWire, e := encodeExecution(loaded)
	if e != nil {
		t.Fatal(e)
	}
	if hexDigest(updatedWire) == a.LedgerDigest {
		t.Fatal("retry fixture did not change ledger generation")
	}
	for _, mutate := range []func(*executiongraph.DispatchAuthority){func(a *executiongraph.DispatchAuthority) { a.SingleAttemptID = "00000000-0000-4000-8000-000000000051" }, func(a *executiongraph.DispatchAuthority) { a.ProposalDigest = strings.Repeat("c", 64) }} {
		next := a
		next.LedgerDigest = hexDigest(updatedWire)
		next.Revision = loaded.Revision
		mutate(&next)
		if e := s.CommitSingleDispatch(ctx, loaded, next, next.PreviewDigest); !errors.Is(e, ErrConflict) {
			t.Fatalf("retry allowed %v", e)
		}
	}
}

func TestDispatchCompanionMismatchRefusesCommit(t *testing.T) {
	root, e := os.OpenRoot(t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	defer root.Close()
	g, a := localDispatch(t)
	graph, _ := executiongraph.EncodeSuccessorGraph(g)
	receipt, _ := executiongraph.EncodeDispatchAuthority(a)
	hooks := publicationHooks{contentLimit: executiongraph.MaxGraphBytes, companion: g.DispatchAuthorityRef + ".json", companionWire: receipt}
	hooks.fault = func(stage FaultStage) error {
		if stage == FaultF5 {
			return root.Remove(hooks.companion)
		}
		return nil
	}
	e = publishFile(context.Background(), root, graphName(a.ParentID), nil, graph, true, hooks)
	var pe *PublicationError
	if !errors.As(e, &pe) || pe.Committed || !errors.Is(e, ErrRecoveryRequired) {
		t.Fatalf("missing receipt committed: %v", e)
	}
	if _, e := root.Lstat(graphName(a.ParentID)); !os.IsNotExist(e) {
		t.Fatalf("partial canonical visible: %v", e)
	}
}

func TestSingleDispatchRetryLedgerGeneration(t *testing.T) {
	wire, e := os.ReadFile("../workflowcompiler/testdata/issue306/execution-v3.json")
	if e != nil {
		t.Fatal(e)
	}
	prior, e := decodeExecution(wire)
	if e != nil {
		t.Fatal(e)
	}
	next := prior
	next.UpdatedAt = next.UpdatedAt.Add(time.Second)
	next.Revision++
	next.Status = workflow.ExecutionInterrupted
	next.Transitions = append(next.Transitions, workflow.Transition{Revision: next.Revision, From: next.Stage, To: next.Stage, Outcome: workflow.OutcomeFailed, RequestDigest: strings.Repeat("d", 64), CommittedAt: next.UpdatedAt, Provenance: next.Provenance})
	changed, e := encodeExecution(next)
	if e != nil {
		t.Fatal(e)
	}
	if hexDigest(changed) == hexDigest(wire) || !sameExecutionBinding(prior, next) {
		t.Fatal("retry fixture fails to preserve scope in distinct valid ledger")
	}
	if _, e := decodeExecution(changed); e != nil {
		t.Fatal(e)
	}
}
