package stageexecution

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/rgomids/axiom/internal/executiongraph"
	"github.com/rgomids/axiom/internal/gitworkspace"
	"github.com/rgomids/axiom/internal/workflow"
	"github.com/rgomids/axiom/internal/workflowcompiler"
)

// fakeIntegrateStore keeps exact-byte storage revisions and enforces the
// successor update rules, like the protected local store.
type fakeIntegrateStore struct {
	contract    Contract
	graph       executiongraph.SuccessorGraph
	records     []executiongraph.AttemptRecord
	receipts    map[string]executiongraph.IntegrationReceipt
	leaseHeld   bool
	saves       int
	failSave    int // fail the Nth save (1-based); 0 never
	events      *[]string
	receiptLoad error
}

type fakeIntegrateLease struct{ epoch uint64 }

func (l fakeIntegrateLease) Epoch() uint64   { return l.epoch }
func (l fakeIntegrateLease) Validate() error { return nil }
func (l fakeIntegrateLease) Close() error    { return nil }

func (f *fakeIntegrateStore) StageSnapshot(context.Context, StatusRequest) (Contract, error) {
	c := f.contract
	g := f.graph
	c.Graph = &executiongraph.GraphDocument{Successor: &g}
	return c, nil
}
func (f *fakeIntegrateStore) Records(context.Context, executiongraph.SuccessorGraph) ([]executiongraph.AttemptRecord, error) {
	return append([]executiongraph.AttemptRecord(nil), f.records...), nil
}
func (f *fakeIntegrateStore) Acquire(context.Context, executiongraph.SuccessorGraph) (SupervisorLease, error) {
	if f.leaseHeld {
		return nil, errors.New("lease held")
	}
	var epoch uint64
	for _, c := range f.graph.Children {
		for _, a := range c.Attempts {
			if a.LeaseEpoch > epoch {
				epoch = a.LeaseEpoch
			}
		}
	}
	return fakeIntegrateLease{epoch + 1}, nil
}
func (f *fakeIntegrateStore) Save(_ context.Context, g executiongraph.SuccessorGraph, _ SupervisorLease) (executiongraph.SuccessorGraph, error) {
	f.saves++
	if f.events != nil {
		*f.events = append(*f.events, fmt.Sprintf("save%d", f.saves))
	}
	if f.saves == f.failSave {
		return g, errors.New("injected save failure")
	}
	if g.StorageRevision != f.graph.StorageRevision || executiongraph.ValidateSuccessorUpdate(f.graph, g) != nil {
		return g, errors.New("cas or update refused")
	}
	wire, err := executiongraph.EncodeSuccessorGraph(g)
	if err != nil {
		return g, err
	}
	g.StorageRevision = sha256.Sum256(wire)
	f.graph = g
	return g, nil
}
func (f *fakeIntegrateStore) CommitIntegrationReceipt(_ context.Context, g executiongraph.SuccessorGraph, a executiongraph.IntegrationReceipt) (string, error) {
	if g.StorageRevision != f.graph.StorageRevision {
		return "", errors.New("stale")
	}
	if err := executiongraph.CheckIntegrationDispatch(g, a); err != nil {
		return "", err
	}
	ref, err := executiongraph.IntegrationReceiptDigest(a)
	if err != nil {
		return "", err
	}
	if f.events != nil {
		*f.events = append(*f.events, "receipt")
	}
	f.receipts[ref] = a
	return ref, nil
}
func (f *fakeIntegrateStore) LoadIntegrationReceipt(_ context.Context, _ StatusRequest, ref string) (executiongraph.IntegrationReceipt, error) {
	if f.receiptLoad != nil {
		return executiongraph.IntegrationReceipt{}, f.receiptLoad
	}
	a, ok := f.receipts[ref]
	if !ok {
		return a, errors.New("missing receipt")
	}
	return a, nil
}

type fakeIntegrationWorkspace struct {
	observation executiongraph.IntegrationObservation
	results     []executiongraph.ChildResult
	applyErr    error
	unconfirmed bool
	validate    func([]gitworkspace.ValidationCommand) ([]executiongraph.ValidationResult, error)
	applies     int
	events      *[]string
}

func (f *fakeIntegrationWorkspace) Observe(context.Context, executiongraph.SuccessorGraph, map[string]string) (executiongraph.IntegrationObservation, []executiongraph.ChildResult, error) {
	return f.observation, append([]executiongraph.ChildResult(nil), f.results...), nil
}
func (f *fakeIntegrationWorkspace) Apply(_ context.Context, _ executiongraph.SuccessorGraph, p executiongraph.IntegrationPreview) (executiongraph.AppliedIntegration, error) {
	f.applies++
	if f.events != nil {
		*f.events = append(*f.events, "apply")
	}
	if f.applyErr != nil {
		return executiongraph.AppliedIntegration{}, f.applyErr
	}
	return executiongraph.AppliedIntegration{Confirmed: !f.unconfirmed, ResultTree: strings.Repeat("e", 40), AppliedEffects: p.Effects}, nil
}
func (f *fakeIntegrationWorkspace) Validate(_ context.Context, _ executiongraph.SuccessorGraph, _ string, commands []gitworkspace.ValidationCommand) ([]executiongraph.ValidationResult, error) {
	if f.events != nil {
		*f.events = append(*f.events, "validate")
	}
	if f.validate != nil {
		return f.validate(commands)
	}
	var out []executiongraph.ValidationResult
	for _, c := range commands {
		out = append(out, executiongraph.ValidationResult{CommandReference: c.Reference, ExitCode: 0, OutputDigest: strings.Repeat("f", 64)})
	}
	return out, nil
}

type fakeValidatorCommands struct{ digest string }

func (f fakeValidatorCommands) Commands(_ context.Context, project, digest string) ([]gitworkspace.ValidationCommand, error) {
	if digest != f.digest {
		return nil, errors.New("validator policy digest drift")
	}
	return []gitworkspace.ValidationCommand{{Reference: "go-test", Argv: []string{"/usr/bin/go", "test"}, OutputMax: 1024}}, nil
}

type fakeOutcomeLedger struct {
	outcomes map[string]executiongraph.StageOutcome
	fail     error
	records  int
}

func (f *fakeOutcomeLedger) RecordStageOutcome(_ context.Context, state workflow.State, o executiongraph.StageOutcome) (string, error) {
	if f.fail != nil {
		return "", f.fail
	}
	if o.LedgerDigest != hex.EncodeToString(state.StorageRevision[:]) || o.ExecutionRevision != state.Revision {
		return "", errors.New("ledger generation changed")
	}
	ref, err := executiongraph.StageOutcomeDigest(o)
	if err != nil {
		return "", err
	}
	f.records++
	f.outcomes[ref] = o
	return ref, nil
}
func (f *fakeOutcomeLedger) LoadStageOutcomes(context.Context, string, string) (map[string]executiongraph.StageOutcome, error) {
	out := map[string]executiongraph.StageOutcome{}
	for k, v := range f.outcomes {
		out[k] = v
	}
	return out, nil
}

const integratePolicy = "7482565a0f63b03e16e0324d948a53421b6c03f0a7d1e4f250c7dd952edcaefb"

func recordDigest(child string) string {
	d := sha256.Sum256([]byte("record:" + child))
	return hex.EncodeToString(d[:])
}

// integrationFixture builds a successor graph whose two work children
// succeeded with qualifying result records, and an integration child.
func integrationFixture(t *testing.T) (IntegrateService, IntegrateRequest, *fakeIntegrateStore, *fakeIntegrationWorkspace, *fakeOutcomeLedger) {
	t.Helper()
	g := graphFixture(t)
	started, finished := time.Unix(11, 0).UTC(), time.Unix(12, 0).UTC()
	var records []executiongraph.AttemptRecord
	var results []executiongraph.ChildResult
	for i, c := range g.Children {
		if c.Envelope.IntegrationOwner {
			g.Children[i].Envelope.Controls.MaximumAttempts = 3
			continue
		}
		attempt := fmt.Sprintf("00000000-0000-4000-8000-00000000010%d", i)
		ref := recordDigest(c.ExecutionID)
		g.Children[i].Attempts = []executiongraph.SuccessorAttempt{{AttemptID: attempt, Number: 1, Status: executiongraph.AttemptSucceeded, StartedAt: &started, FinishedAt: &finished, ClassificationReason: executiongraph.ResultRecorded, QualifyingRecordRef: ref, LeaseEpoch: 1, Process: &executiongraph.ProcessCorrelation{PID: 10 + int64(i), StartIdentity: "s", GroupIdentity: "g"}, PacketDigest: strings.Repeat("b", 64), ReconciliationFacts: []executiongraph.ReconciliationFact{}}}
		records = append(records, executiongraph.AttemptRecord{ParentID: g.Parent.ExecutionID, ChildID: c.ExecutionID, AttemptID: attempt, GraphRevision: g.Parent.GraphRevision, Revision: uint64(i + 1), Kind: "result", Outcome: "succeeded", Digest: ref, Accepted: true})
		results = append(results, executiongraph.ChildResult{ParentID: g.Parent.ExecutionID, ChildID: c.ExecutionID, AttemptID: attempt, GraphRevision: g.Parent.GraphRevision, EnvelopeDigest: executiongraph.EnvelopeDigest(g.Children[i].Envelope), BaseRevision: strings.Repeat("a", 40), ResultTree: strings.Repeat(fmt.Sprint(i), 40)})
	}
	wire, err := executiongraph.EncodeSuccessorGraph(g)
	if err != nil {
		t.Fatal(err)
	}
	g, err = executiongraph.DecodeSuccessorGraph(wire)
	if err != nil {
		t.Fatal(err)
	}
	c := currentContract(t)
	// The Execution and the allocated graph parent are distinct identities.
	c.Execution.ExecutionID = "00000000-0000-4000-8000-0000000005e1"
	c.Execution.ProjectID = g.Children[0].Envelope.Scope.ProjectID
	c.Execution.Revision = g.Parent.GraphRevision
	c.Execution.Stage = "implementation"
	c.Execution.StageOrdinal = 2
	c.Execution.StorageRevision = sha256.Sum256([]byte("execution-ledger-generation"))
	store := &fakeIntegrateStore{contract: c, graph: g, records: records, receipts: map[string]executiongraph.IntegrationReceipt{}}
	workspace := &fakeIntegrationWorkspace{observation: executiongraph.IntegrationObservation{TargetRevision: strings.Repeat("a", 40), TargetTree: strings.Repeat("c", 40)}, results: results}
	ledger := &fakeOutcomeLedger{outcomes: map[string]executiongraph.StageOutcome{}}
	n := 0
	s := IntegrateService{Store: store, Workspace: workspace, Validators: fakeValidatorCommands{integratePolicy}, Ledger: ledger, AllocateID: func() (string, error) { n++; return fmt.Sprintf("00000000-0000-4000-8000-0000000002%02d", n), nil }, Now: func() time.Time { return time.Unix(100, 0).UTC() }, Self: func() (*executiongraph.ProcessCorrelation, error) {
		return &executiongraph.ProcessCorrelation{PID: 77, StartIdentity: "integrator-start", GroupIdentity: "integrator-group"}, nil
	}}
	r := IntegrateRequest{Target: StatusRequest{ProjectID: c.Execution.ProjectID, ParentID: g.Parent.ExecutionID}, Execution: workflowcompiler.ExecutionRef{ExecutionID: c.Execution.ExecutionID, Revision: g.Parent.GraphRevision}, StageID: "implementation", ValidatorPolicyDigest: integratePolicy}
	return s, r, store, workspace, ledger
}

func integrationAttempt(g executiongraph.SuccessorGraph) executiongraph.SuccessorAttempt {
	c, _ := integrationChildOf(g)
	return c.Attempts[len(c.Attempts)-1]
}

func TestIntegrateTwoChildrenWithValidators(t *testing.T) {
	s, r, store, workspace, ledger := integrationFixture(t)
	events := []string{}
	store.events, workspace.events = &events, &events
	p, err := s.Preview(context.Background(), r)
	if err != nil {
		t.Fatal(err)
	}
	again, err := s.Preview(context.Background(), r)
	if err != nil || again.Digest != p.Digest {
		t.Fatal("integrate preview is not deterministic")
	}
	if p.ProjectionOnly || p.Integration == nil || len(p.Integration.Sources) != 2 || store.saves != 0 {
		t.Fatalf("preview = %+v, saves = %d", p, store.saves)
	}
	res, err := s.Apply(context.Background(), IntegrateApply{Preview: p, AuthorizeLocal: true, Actor: "operator"})
	if err != nil || res.Status != "success" {
		t.Fatalf("apply = %+v, %v", res, err)
	}
	// Receipt, then running attempt, before any effect; terminal save last.
	if strings.Join(events, ",") != "receipt,save1,apply,validate,save2" {
		t.Fatalf("order = %v", events)
	}
	a := integrationAttempt(store.graph)
	if a.Status != executiongraph.AttemptSucceeded || a.ClassificationReason != executiongraph.IntegrationApplied || a.PacketDigest != res.ReceiptRef || a.ResultReference != executiongraph.IntegrationResultDigest(strings.Repeat("e", 40)) {
		t.Fatalf("integration attempt = %+v", a)
	}
	if res.Rollup == nil || res.Rollup.Status != "success" || res.Outcome == nil || res.Outcome.Rollup.Status != "success" || ledger.records != 1 {
		t.Fatalf("rollup = %+v outcome = %+v", res.Rollup, res.Outcome)
	}
	// Roll-up is a pure function of persisted state: deterministic.
	again2, _ := executiongraph.RollupSuccessor(store.graph, nil)
	if fmt.Sprint(again2) != fmt.Sprint(*res.Rollup) {
		t.Fatal("roll-up not deterministic from persisted graph")
	}
	if _, err := s.Preview(context.Background(), r); !errors.Is(err, ErrIntegrationRecorded) {
		t.Fatalf("second integrate preview = %v", err)
	}
}

func TestIntegrateRefusals(t *testing.T) {
	ctx := context.Background()
	t.Run("missing child result", func(t *testing.T) {
		s, r, _, workspace, _ := integrationFixture(t)
		workspace.results = workspace.results[:1]
		if _, err := s.Preview(ctx, r); !errors.Is(err, executiongraph.ErrIntegrationBlocked) {
			t.Fatalf("missing result = %v", err)
		}
	})
	t.Run("result_missing classification refused", func(t *testing.T) {
		s, r, store, _, _ := integrationFixture(t)
		g := store.graph
		g.Children[0].Attempts[0].Status = executiongraph.AttemptFailed
		g.Children[0].Attempts[0].ClassificationReason = executiongraph.ResultMissing
		g.Children[0].Attempts[0].QualifyingRecordRef = ""
		wire, _ := executiongraph.EncodeSuccessorGraph(g)
		store.graph, _ = executiongraph.DecodeSuccessorGraph(wire)
		if _, err := s.Preview(ctx, r); err == nil {
			t.Fatal("integration of a result_missing child previewed")
		}
	})
	t.Run("caller result for another attempt", func(t *testing.T) {
		s, r, _, workspace, _ := integrationFixture(t)
		workspace.results[0].AttemptID = "00000000-0000-4000-8000-0000000009ff"
		if _, err := s.Preview(ctx, r); !errors.Is(err, executiongraph.ErrIntegrationBlocked) {
			t.Fatalf("foreign attempt result = %v", err)
		}
	})
	t.Run("conflicting results", func(t *testing.T) {
		s, r, _, workspace, _ := integrationFixture(t)
		workspace.observation.Conflicts = []string{"child_result_conflict"}
		if _, err := s.Preview(ctx, r); !errors.Is(err, executiongraph.ErrIntegrationBlocked) {
			t.Fatalf("conflict = %v", err)
		}
	})
	t.Run("validator policy drift", func(t *testing.T) {
		s, r, _, _, _ := integrationFixture(t)
		r.ValidatorPolicyDigest = strings.Repeat("9", 64)
		if _, err := s.Preview(ctx, r); !errors.Is(err, ErrValidatorPolicy) {
			t.Fatalf("policy drift = %v", err)
		}
	})
	t.Run("no local authority", func(t *testing.T) {
		s, r, store, workspace, _ := integrationFixture(t)
		p, _ := s.Preview(ctx, r)
		if res, err := s.Apply(ctx, IntegrateApply{Preview: p, Actor: "operator"}); !errors.Is(err, ErrDispatchAuthority) || res.Status != "denied_authority" || store.saves != 0 || workspace.applies != 0 {
			t.Fatalf("apply without authority = %+v %v", res, err)
		}
	})
	t.Run("stale preview", func(t *testing.T) {
		s, r, store, workspace, _ := integrationFixture(t)
		p, _ := s.Preview(ctx, r)
		store.contract.Execution.StorageRevision[0] ^= 1
		if _, err := s.Apply(ctx, IntegrateApply{Preview: p, AuthorizeLocal: true, Actor: "operator"}); !errors.Is(err, ErrStalePreview) || workspace.applies != 0 {
			t.Fatalf("stale apply = %v", err)
		}
	})
	t.Run("lease held by supervisor", func(t *testing.T) {
		s, r, store, workspace, _ := integrationFixture(t)
		p, _ := s.Preview(ctx, r)
		store.leaseHeld = true
		if res, err := s.Apply(ctx, IntegrateApply{Preview: p, AuthorizeLocal: true, Actor: "operator"}); err == nil || res.Category != "temporarily_unavailable" || workspace.applies != 0 {
			t.Fatalf("held lease = %+v %v", res, err)
		}
	})
	t.Run("cancel requested", func(t *testing.T) {
		s, r, store, _, _ := integrationFixture(t)
		g := store.graph
		g.CancelRequest = &executiongraph.CancelRequest{RequestID: "00000000-0000-4000-8000-0000000003ff", ParentID: g.Parent.ExecutionID, GraphRevision: g.Parent.GraphRevision, ActorKind: "human-operator", Actor: "operator", RequestedAt: time.Unix(50, 0).UTC(), AuthorityRef: strings.Repeat("7", 64)}
		wire, _ := executiongraph.EncodeSuccessorGraph(g)
		store.graph, _ = executiongraph.DecodeSuccessorGraph(wire)
		if _, err := s.Preview(ctx, r); !errors.Is(err, executiongraph.ErrIntegrationBlocked) {
			t.Fatalf("cancelled graph = %v", err)
		}
	})
	t.Run("historical graph refused", func(t *testing.T) {
		s, r, store, _, _ := integrationFixture(t)
		hist := historicalGraph(t, 2)
		c := store.contract
		c.Graph = &hist
		if _, err := Admit(StageIntegrate, c); !errors.Is(err, executiongraph.ErrUnsupportedGraphFormat) {
			t.Fatalf("historical admission = %v", err)
		}
		_ = s
		_ = r
	})
}

func TestIntegrateValidatorFailureRecordsFailureOutcome(t *testing.T) {
	s, r, store, workspace, ledger := integrationFixture(t)
	workspace.validate = func(commands []gitworkspace.ValidationCommand) ([]executiongraph.ValidationResult, error) {
		return []executiongraph.ValidationResult{{CommandReference: commands[0].Reference, ExitCode: 2, OutputDigest: strings.Repeat("f", 64)}}, nil
	}
	p, err := s.Preview(context.Background(), r)
	if err != nil {
		t.Fatal(err)
	}
	res, err := s.Apply(context.Background(), IntegrateApply{Preview: p, AuthorizeLocal: true, Actor: "operator"})
	if !errors.Is(err, executiongraph.ErrIntegrationFailed) || res.Status != "failure" || res.Rollup.Status != "failure" {
		t.Fatalf("validator failure = %+v %v", res, err)
	}
	a := integrationAttempt(store.graph)
	if a.Status != executiongraph.AttemptFailed || a.ClassificationReason != executiongraph.IntegrationFailed || a.ExitCode != 2 || ledger.records != 1 {
		t.Fatalf("attempt = %+v records = %d", a, ledger.records)
	}
	// A failed integration may be attempted again only under new authority.
	workspace.validate = nil
	p2, err := s.Preview(context.Background(), r)
	if err != nil || p2.ProjectionOnly {
		t.Fatalf("re-preview after failure = %+v %v", p2, err)
	}
	if res, err := s.Apply(context.Background(), IntegrateApply{Preview: p2, AuthorizeLocal: true, Actor: "operator"}); err != nil || res.Status != "success" {
		t.Fatalf("second integration = %+v %v", res, err)
	}
	c, _ := integrationChildOf(store.graph)
	if len(c.Attempts) != 2 || ledger.records != 2 {
		t.Fatalf("attempts = %d, outcomes = %d", len(c.Attempts), ledger.records)
	}
}

func TestIntegrateUnknownEffectBlocksRerun(t *testing.T) {
	for _, mode := range []string{"apply error", "unconfirmed apply"} {
		t.Run(mode, func(t *testing.T) {
			s, r, store, workspace, ledger := integrationFixture(t)
			if mode == "apply error" {
				workspace.applyErr = errors.New("interrupted merge")
			} else {
				workspace.unconfirmed = true
			}
			p, _ := s.Preview(context.Background(), r)
			res, err := s.Apply(context.Background(), IntegrateApply{Preview: p, AuthorizeLocal: true, Actor: "operator"})
			if !errors.Is(err, ErrIntegrationReconcile) || res.Status != "recovery_required" || ledger.records != 0 {
				t.Fatalf("unknown = %+v %v", res, err)
			}
			a := integrationAttempt(store.graph)
			if a.Status != executiongraph.AttemptUnknown || a.ClassificationReason != executiongraph.IntegrationUnknown || !a.AmbiguousEffect {
				t.Fatalf("attempt = %+v", a)
			}
			applies := workspace.applies
			workspace.applyErr, workspace.unconfirmed = nil, false
			if _, err := s.Preview(context.Background(), r); !errors.Is(err, ErrIntegrationReconcile) || workspace.applies != applies {
				t.Fatalf("re-run after unknown = %v", err)
			}
		})
	}
}

func TestIntegrateProjectionFailureIsPartialAndRepairable(t *testing.T) {
	s, r, store, workspace, ledger := integrationFixture(t)
	ledger.fail = errors.New("ledger projection failed")
	p, _ := s.Preview(context.Background(), r)
	res, err := s.Apply(context.Background(), IntegrateApply{Preview: p, AuthorizeLocal: true, Actor: "operator"})
	if !errors.Is(err, ErrProjectionPartial) || res.Status != "partial" || res.Category != "confirmed_effect_reconciliation_required" || res.ReceiptRef == "" || res.Next == "" {
		t.Fatalf("projection failure = %+v %v", res, err)
	}
	if a := integrationAttempt(store.graph); a.Status != executiongraph.AttemptSucceeded {
		t.Fatalf("confirmed effect not canonical: %+v", a)
	}
	// Repair: projection only, no new effect, attempt or receipt.
	ledger.fail = nil
	applies, saves, receipts := workspace.applies, store.saves, len(store.receipts)
	repair, err := s.Preview(context.Background(), r)
	if err != nil || !repair.ProjectionOnly || repair.Integration != nil {
		t.Fatalf("repair preview = %+v %v", repair, err)
	}
	if _, err := s.Apply(context.Background(), IntegrateApply{Preview: repair, Actor: "operator"}); !errors.Is(err, ErrDispatchAuthority) {
		t.Fatal("projection repair without local authority")
	}
	fixed, err := s.Apply(context.Background(), IntegrateApply{Preview: repair, AuthorizeLocal: true, Actor: "operator"})
	if err != nil || fixed.Status != "success" || fixed.Outcome == nil || ledger.records != 1 {
		t.Fatalf("repair = %+v %v", fixed, err)
	}
	if workspace.applies != applies || store.saves != saves || len(store.receipts) != receipts {
		t.Fatal("projection repair repeated an effect")
	}
}

func TestIntegrateTerminalSaveFailureRequiresReconciliation(t *testing.T) {
	s, r, store, _, ledger := integrationFixture(t)
	store.failSave = 2
	p, _ := s.Preview(context.Background(), r)
	res, err := s.Apply(context.Background(), IntegrateApply{Preview: p, AuthorizeLocal: true, Actor: "operator"})
	if err == nil || res.Status != "recovery_required" || ledger.records != 0 {
		t.Fatalf("terminal save failure = %+v %v", res, err)
	}
	if a := integrationAttempt(store.graph); a.Status != executiongraph.AttemptRunning {
		t.Fatalf("attempt = %+v", a)
	}
	if _, err := s.Preview(context.Background(), r); !errors.Is(err, ErrIntegrationReconcile) {
		t.Fatalf("running integration re-preview = %v", err)
	}
}

func TestStageOutcomeRejectsIncompleteRollup(t *testing.T) {
	_, _, store, _, _ := integrationFixture(t)
	r, err := executiongraph.RollupSuccessor(store.graph, nil)
	if err != nil || r.Status != "partial" {
		t.Fatalf("rollup before integration = %+v %v", r, err)
	}
	o := executiongraph.StageOutcome{FormatVersion: 1, ProjectID: "p", ExecutionID: store.graph.Parent.ExecutionID, ExecutionRevision: 1, LedgerDigest: strings.Repeat("1", 64), StageID: "implementation", GraphDigest: strings.Repeat("2", 64), IntegrationReceiptRef: strings.Repeat("3", 64), ValidatorPolicyDigest: integratePolicy, Rollup: r, RecordedAt: time.Unix(1, 0).UTC()}
	if executiongraph.ValidateStageOutcome(o) == nil {
		t.Fatal("partial roll-up projected as a stage outcome")
	}
}
