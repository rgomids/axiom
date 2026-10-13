package stageexecution

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/executiongraph"
)

// Recompile a valid, newly reviewed proposal without rebinding the persisted
// graph. Both compiler and reviewed input agree; only persisted authority is
// stale. This is a synthetic test and has no Runtime or Provider effects.
func replaceReviewedProposal(t *testing.T, s PreviewService, source *fakePreviewSource) {
	t.Helper()
	compiler := s.Compiler.(*fakePlanCompiler)
	p := *compiler.plan.Compilation.Proposal
	p.PlanRevision += "-changed"
	p.Digest = ""
	wire, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(wire)
	p.Digest = hex.EncodeToString(sum[:])
	compiler.plan.Compilation.Proposal = &p
	ref := *compiler.plan.GraphProposal
	ref.PlanRevision, ref.Digest = p.PlanRevision, p.Digest
	compiler.plan.GraphProposal = &ref
	source.o.Reviewed = compiler.plan
	if _, err := s.Preview(context.Background(), compiler.plan.Execution, compiler.plan.StageID); err != nil {
		t.Fatalf("changed proposal must itself be a valid fresh preview: %v", err)
	}
}

func TestRetryRejectsRecompiledProposalDifferentFromPersistedGraph(t *testing.T) {
	h := retryFixture(t, "failed", 3)
	p, err := h.service.Preview(context.Background(), h.request)
	if err != nil {
		t.Fatal(err)
	}
	before := cloneSupervisorGraph(h.store.g)
	replaceReviewedProposal(t, h.service.Dispatch, h.source)
	if _, err = h.service.Preview(context.Background(), h.request); !errors.Is(err, ErrStalePreview) {
		t.Fatalf("fresh but foreign proposal preview accepted: %v", err)
	}
	if _, err = h.service.Apply(context.Background(), RetryApply{Preview: p, AuthorizeLocal: true, AuthorizeExternal: true, Actor: "synthetic-operator"}); !errors.Is(err, ErrStalePreview) {
		t.Fatalf("stale digest accepted: %v", err)
	}
	if !reflect.DeepEqual(before, h.store.g) || h.store.commits != 0 || len(h.starter.refs) != 0 || len(h.disposer.dispositions) != 0 {
		t.Fatal("foreign proposal changed state, authority, or attempted an effect")
	}
}

func TestResumeRejectsRecompiledProposalDifferentFromPersistedGraph(t *testing.T) {
	s, r, _, store, ports, source := resumeFixture(t)
	store.g = cloneSupervisorGraph(store.g)
	before := cloneSupervisorGraph(store.g)
	replaceReviewedProposal(t, s.Preview, source)
	if _, err := s.Apply(context.Background(), r); !errors.Is(err, ErrStalePreview) {
		t.Fatalf("foreign proposal accepted on resume: %v", err)
	}
	if !reflect.DeepEqual(before, store.g) || len(ports.events) != 0 {
		t.Fatal("resume rebound graph or attempted unauthorized effects")
	}
}

// Even an exact preview-authority match must not bypass the persisted Graph's
// proposal binding. All other admission checks succeed before Apply refuses.
func TestStageRunRejectsGraphProposalMismatchWithMatchingPreviewAuthority(t *testing.T) {
	s, r, _, store, ports, _ := resumeFixture(t)
	ctx := context.Background()
	fresh, err := s.Preview.Preview(ctx, r.Execution, r.StageID)
	if err != nil || fresh.Publication == nil || fresh.Digest != r.PreviewDigest {
		t.Fatalf("valid recompiled stage-run preview required: %v", err)
	}
	store.g = cloneSupervisorGraph(store.g)
	store.g.Parent.ProposalDigest = strings.Repeat("f", 64)
	store.a.ProposalDigest = store.g.Parent.ProposalDigest
	ref, err := executiongraph.DispatchAuthorityDigest(store.a)
	if err != nil {
		t.Fatal(err)
	}
	store.g.DispatchAuthorityRef = ref
	for i := range store.g.Children {
		store.g.Children[i].Envelope.AuthorityReference = ref
	}
	r.Committed.AuthorityRef = ref
	wire, err := executiongraph.EncodeSuccessorGraph(store.g)
	if err != nil {
		t.Fatal(err)
	}
	store.g.StorageRevision = sha256.Sum256(wire)
	if err = verifySupervisor(*r.Committed, store.g, store.a); err != nil {
		t.Fatalf("must reach proposal comparison, not authority refusal: %v", err)
	}
	if err = admitSupervisor(ctx, store, *r.Committed, store.g); err != nil {
		t.Fatalf("must reach proposal comparison, not admission refusal: %v", err)
	}
	if fresh.Publication.ExpectedDigest == store.g.Parent.ProposalDigest {
		t.Fatal("fixture did not produce a Graph mismatch")
	}
	if err = executiongraph.CheckFirstDispatch(store.g, store.a, r.PreviewDigest, store.g.Children[0].ExecutionID, 1); err != nil {
		t.Fatalf("remaining first attempt required: %v", err)
	}
	before, authority, saves := cloneSupervisorGraph(store.g), store.a, store.saves
	result, err := s.Apply(ctx, r)
	if !errors.Is(err, ErrStalePreview) || result.Status != "denied_authority" || result.Process != nil {
		t.Fatalf("Graph mismatch accepted: %+v, %v", result, err)
	}
	if !reflect.DeepEqual(before, store.g) || !reflect.DeepEqual(authority, store.a) || store.saves != saves || len(ports.events) != 0 {
		t.Fatal("Graph mismatch mutated state or authority, allocated an Execution, or dispatched")
	}
}
