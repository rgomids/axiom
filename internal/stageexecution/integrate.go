package stageexecution

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/rgomids/axiom/internal/executiongraph"
	"github.com/rgomids/axiom/internal/gitworkspace"
	"github.com/rgomids/axiom/internal/workflow"
	"github.com/rgomids/axiom/internal/workflowcompiler"
)

var (
	// ErrIntegrationRecorded reports an integration whose terminal outcome is
	// already projected; nothing remains to apply.
	ErrIntegrationRecorded = errors.New("integration_already_recorded")
	// ErrIntegrationReconcile reports an unknown or running integration attempt:
	// its effect may exist, so it is never repeated before reconciliation.
	ErrIntegrationReconcile = errors.New("integration_reconciliation_required")
	ErrValidatorPolicy      = errors.New("validator_policy_unavailable")
	// ErrProjectionPartial is a confirmed integration effect whose ledger
	// projection failed: partial / confirmed_effect_reconciliation_required.
	ErrProjectionPartial = errors.New("confirmed_effect_reconciliation_required")
)

// IntegrateStore reads stage state and persists integration attempts under
// the run lease (ADR-0007 order: lease, then graph CAS). Receipts are
// protected, content-addressed and bound to the exact graph revision.
type IntegrateStore interface {
	StatusSource
	Records(context.Context, executiongraph.SuccessorGraph) ([]executiongraph.AttemptRecord, error)
	Acquire(context.Context, executiongraph.SuccessorGraph) (SupervisorLease, error)
	Save(context.Context, executiongraph.SuccessorGraph, SupervisorLease) (executiongraph.SuccessorGraph, error)
	CommitIntegrationReceipt(context.Context, executiongraph.SuccessorGraph, executiongraph.IntegrationReceipt) (string, error)
	LoadIntegrationReceipt(context.Context, StatusRequest, string) (executiongraph.IntegrationReceipt, error)
}

// IntegrationWorkspace is the existing digest-bound Git integration boundary,
// projected over successor state. It never decides eligibility.
type IntegrationWorkspace interface {
	Observe(context.Context, executiongraph.SuccessorGraph, map[string]string) (executiongraph.IntegrationObservation, []executiongraph.ChildResult, error)
	Apply(context.Context, executiongraph.SuccessorGraph, executiongraph.IntegrationPreview) (executiongraph.AppliedIntegration, error)
	Validate(context.Context, executiongraph.SuccessorGraph, string, []gitworkspace.ValidationCommand) ([]executiongraph.ValidationResult, error)
}

// ValidatorCommands resolves only a reviewed, operator-authored policy of the
// Project by exact digest (I276-T32); never agent records or Plan documents.
type ValidatorCommands interface {
	Commands(context.Context, string, string) ([]gitworkspace.ValidationCommand, error)
}

// StageOutcomeLedger projects terminal outcomes beside the Execution ledger.
type StageOutcomeLedger interface {
	RecordStageOutcome(context.Context, workflow.State, executiongraph.StageOutcome) (string, error)
	LoadStageOutcomes(context.Context, string, string) (map[string]executiongraph.StageOutcome, error)
}

type IntegrateRequest struct {
	Target                StatusRequest
	Execution             workflowcompiler.ExecutionRef
	StageID               string
	ValidatorPolicyDigest string
	OptionalWaivers       map[string]string
}

type IntegratePreview struct {
	Request         IntegrateRequest
	Revision        uint64
	StorageRevision [32]byte
	LedgerDigest    string
	// ProjectionOnly repairs a missing projection of an already terminal
	// integration attempt. It never applies, validates or repeats effects.
	ProjectionOnly bool
	AttemptID      string
	Integration    *executiongraph.IntegrationPreview
	Commands       []string
	Digest         string
}

type IntegrateApply struct {
	Preview        IntegratePreview
	AuthorizeLocal bool
	Actor          string
}

type IntegrateResult struct {
	Status, Category, Next string
	ReceiptRef             string
	Rollup                 *executiongraph.StageRollup
	Outcome                *executiongraph.StageOutcome
	OutcomeRef             string
	ResultTree             string
	Validations            []executiongraph.ValidationResult
}

type IntegrateService struct {
	Store      IntegrateStore
	Workspace  IntegrationWorkspace
	Validators ValidatorCommands
	Ledger     StageOutcomeLedger
	AllocateID func() (string, error)
	Now        func() time.Time
	// Self observes the integrating process for the running attempt's
	// correlation; integration has no child process of its own.
	Self func() (*executiongraph.ProcessCorrelation, error)
}

func (s IntegrateService) now() time.Time {
	if s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}

type integrateSnapshot struct {
	contract Contract
	graph    executiongraph.SuccessorGraph
}

func (s IntegrateService) snapshot(ctx context.Context, r IntegrateRequest) (integrateSnapshot, error) {
	if s.Store == nil || s.Workspace == nil || s.Validators == nil || s.Ledger == nil {
		return integrateSnapshot{}, ErrPreview
	}
	c, err := s.Store.StageSnapshot(ctx, r.Target)
	if err != nil {
		return integrateSnapshot{}, err
	}
	if _, err = Admit(StageIntegrate, c); err != nil {
		return integrateSnapshot{}, err
	}
	if c.Graph == nil || c.Graph.Successor == nil {
		return integrateSnapshot{}, executiongraph.ErrUnsupportedGraphFormat
	}
	g := *c.Graph.Successor
	if g.Parent.ExecutionID != r.Target.ParentID || g.Children[0].Envelope.Scope.ProjectID != r.Target.ProjectID || r.Execution.ExecutionID != r.Target.ParentID || c.Execution.ExecutionID != g.Parent.ExecutionID || c.Execution.ProjectID != r.Target.ProjectID {
		return integrateSnapshot{}, ErrAdmission
	}
	// The graph was published for exactly this Execution generation and stage;
	// an advanced or different ledger cannot receive its outcome.
	if c.Execution.Revision != r.Execution.Revision || c.Execution.Revision != g.Parent.GraphRevision || string(c.Execution.Stage) != r.StageID || c.Execution.StorageRevision == ([32]byte{}) {
		return integrateSnapshot{}, ErrStalePreview
	}
	return integrateSnapshot{contract: c, graph: g}, nil
}

func integrationChildOf(g executiongraph.SuccessorGraph) (executiongraph.SuccessorChild, bool) {
	for _, c := range g.Children {
		if c.ExecutionID == g.Parent.IntegrationChild && c.Envelope.IntegrationOwner {
			return c, true
		}
	}
	return executiongraph.SuccessorChild{}, false
}

func outcomeFor(outcomes map[string]executiongraph.StageOutcome, attemptID string) (string, *executiongraph.StageOutcome) {
	for ref, o := range outcomes {
		if o.Rollup.IntegrationAttemptID == attemptID {
			copy := o
			return ref, &copy
		}
	}
	return "", nil
}

func (s IntegrateService) preview(ctx context.Context, r IntegrateRequest) (IntegratePreview, integrateSnapshot, error) {
	snap, err := s.snapshot(ctx, r)
	if err != nil {
		return IntegratePreview{}, snap, err
	}
	g := snap.graph
	commands, err := s.Validators.Commands(ctx, r.Target.ProjectID, r.ValidatorPolicyDigest)
	if err != nil || len(commands) == 0 {
		return IntegratePreview{}, snap, errors.Join(ErrValidatorPolicy, err)
	}
	p := IntegratePreview{Request: r, Revision: g.Parent.GraphRevision, StorageRevision: g.StorageRevision, LedgerDigest: hex.EncodeToString(snap.contract.Execution.StorageRevision[:])}
	for _, c := range commands {
		p.Commands = append(p.Commands, c.Reference)
	}
	sort.Strings(p.Commands)
	integration, ok := integrationChildOf(g)
	if !ok {
		return IntegratePreview{}, snap, executiongraph.ErrInvalidGraph
	}
	if n := len(integration.Attempts); n > 0 {
		last := integration.Attempts[n-1]
		outcomes, err := s.Ledger.LoadStageOutcomes(ctx, r.Target.ProjectID, g.Parent.ExecutionID)
		if err != nil {
			return IntegratePreview{}, snap, err
		}
		_, recorded := outcomeFor(outcomes, last.AttemptID)
		switch {
		case last.Status == executiongraph.AttemptRunning:
			return IntegratePreview{}, snap, ErrIntegrationReconcile
		case (last.Status == executiongraph.AttemptSucceeded || last.Status == executiongraph.AttemptFailed) && recorded == nil:
			p.ProjectionOnly, p.AttemptID = true, last.AttemptID
		case last.Status == executiongraph.AttemptSucceeded:
			return IntegratePreview{}, snap, ErrIntegrationRecorded
		}
	}
	if !p.ProjectionOnly {
		available, err := executiongraph.SuccessorIntegrationAttemptAvailable(g)
		if err != nil {
			return IntegratePreview{}, snap, err
		}
		if !available {
			if n := len(integration.Attempts); n > 0 && (integration.Attempts[n-1].Status == executiongraph.AttemptUnknown || integration.Attempts[n-1].AmbiguousEffect) {
				return IntegratePreview{}, snap, ErrIntegrationReconcile
			}
			return IntegratePreview{}, snap, executiongraph.ErrIntegrationBlocked
		}
		records, err := s.Store.Records(ctx, g)
		if err != nil {
			return IntegratePreview{}, snap, err
		}
		observation, results, err := s.Workspace.Observe(ctx, g, r.OptionalWaivers)
		if err != nil {
			return IntegratePreview{}, snap, err
		}
		canonical, err := qualifyingResults(g, records, results)
		if err != nil {
			return IntegratePreview{}, snap, err
		}
		consumer, err := executiongraph.LoadSuccessorConsumer(executiongraph.GraphDocument{Successor: &g}, records)
		if err != nil {
			return IntegratePreview{}, snap, err
		}
		ip, err := consumer.IntegrationPreview(observation, results, canonical)
		if err != nil {
			return IntegratePreview{}, snap, err
		}
		p.Integration = &ip
	}
	p.Digest, err = canonicalDigest(struct {
		Operation Operation
		Preview   IntegratePreview
		Graph     executiongraph.SuccessorGraph
	}{StageIntegrate, p, g})
	return p, snap, err
}

// qualifyingResults pairs each observed child result with the accepted
// canonical record its persisted attempt names. Results of any other attempt,
// or without a resolvable qualifying record, are refused by the consumer.
func qualifyingResults(g executiongraph.SuccessorGraph, records []executiongraph.AttemptRecord, results []executiongraph.ChildResult) ([]executiongraph.CanonicalChildResult, error) {
	var canonical []executiongraph.CanonicalChildResult
	for _, result := range results {
		for _, c := range g.Children {
			if c.ExecutionID != result.ChildID || len(c.Attempts) == 0 {
				continue
			}
			a := c.Attempts[len(c.Attempts)-1]
			if a.AttemptID != result.AttemptID || a.QualifyingRecordRef == "" {
				return nil, executiongraph.ErrIntegrationBlocked
			}
			for _, record := range records {
				if record.Digest == a.QualifyingRecordRef {
					canonical = append(canonical, executiongraph.CanonicalChildResult{Record: record, Result: result})
				}
			}
		}
	}
	return canonical, nil
}

func (s IntegrateService) Preview(ctx context.Context, r IntegrateRequest) (IntegratePreview, error) {
	p, _, err := s.preview(ctx, r)
	return p, err
}

// Apply needs local authority only: integration is a local repository effect
// and never invokes a Runtime. Order: receipt, running attempt (commit before
// effect), apply, combined validation, terminal attempt (commit point), then
// the ledger projection. A failed projection after a confirmed terminal
// attempt is partial, never success, and is repaired without new effects.
func (s IntegrateService) Apply(ctx context.Context, r IntegrateApply) (IntegrateResult, error) {
	result := IntegrateResult{Status: "denied_authority", Category: "local_authority_required"}
	if !r.AuthorizeLocal || r.Actor == "" {
		return result, ErrDispatchAuthority
	}
	fresh, snap, err := s.preview(ctx, r.Preview.Request)
	if err != nil {
		result.Status, result.Category = "failure", ""
		return result, err
	}
	if fresh.Digest != r.Preview.Digest {
		result.Category = ErrStalePreview.Error()
		return result, ErrStalePreview
	}
	if fresh.ProjectionOnly {
		return s.project(ctx, snap, fresh.AttemptID, result)
	}
	if s.AllocateID == nil || s.Self == nil {
		result.Status, result.Category = "failure", ""
		return result, ErrPreview
	}
	g := snap.graph
	integration, _ := integrationChildOf(g)
	waivers := append([]executiongraph.OptionalWaiver{}, fresh.Integration.OptionalWaivers...)
	receipt := executiongraph.IntegrationReceipt{FormatVersion: 1, ProjectID: r.Preview.Request.Target.ProjectID, ParentID: g.Parent.ExecutionID, Revision: g.Parent.GraphRevision, GraphDigest: hex.EncodeToString(g.StorageRevision[:]), PreviewDigest: fresh.Digest, IntegrationPreviewDigest: fresh.Integration.Digest, ValidatorPolicyDigest: fresh.Request.ValidatorPolicyDigest, IntegrationChildID: integration.ExecutionID, AttemptNumber: uint32(len(integration.Attempts)) + 1, OptionalWaivers: waivers, Effects: append([]executiongraph.Effect{}, fresh.Integration.Effects...), ExecutionRevision: snap.contract.Execution.Revision, LedgerDigest: fresh.LedgerDigest, StageID: fresh.Request.StageID, StageOrdinal: snap.contract.Execution.StageOrdinal, ActorKind: "human-operator", Actor: r.Actor, RecordedAt: s.now()}
	result.Status, result.Category = "failure", ""
	lease, err := s.Store.Acquire(ctx, g)
	if err != nil {
		result.Category = "temporarily_unavailable"
		return result, err
	}
	if lease == nil || lease.Epoch() == 0 {
		return result, ErrSupervisorRecovery
	}
	defer lease.Close()
	// Re-read under the lease: no supervisor or other integration may have
	// changed the graph since the preview.
	again, _, err := s.preview(ctx, r.Preview.Request)
	if err != nil {
		return result, err
	}
	if again.Digest != fresh.Digest {
		result.Status, result.Category = "denied_authority", ErrStalePreview.Error()
		return result, ErrStalePreview
	}
	if err = executiongraph.CheckIntegrationDispatch(g, receipt); err != nil {
		return result, err
	}
	commands, err := s.Validators.Commands(ctx, receipt.ProjectID, receipt.ValidatorPolicyDigest)
	if err != nil {
		return result, errors.Join(ErrValidatorPolicy, err)
	}
	ref, err := s.Store.CommitIntegrationReceipt(ctx, g, receipt)
	if err != nil {
		return result, err
	}
	if digest, e := executiongraph.IntegrationReceiptDigest(receipt); e != nil || digest != ref {
		result.Status, result.Next = "recovery_required", "confirm the protected integration receipt"
		return result, errors.Join(ErrSupervisorRecovery, e)
	}
	result.ReceiptRef = ref
	attemptID, err := s.AllocateID()
	if err != nil {
		return result, err
	}
	self, err := s.Self()
	if err != nil || self == nil {
		return result, errors.Join(ErrSupervisorRecovery, err)
	}
	started := s.now()
	running := executiongraph.SuccessorAttempt{AttemptID: attemptID, Number: receipt.AttemptNumber, Status: executiongraph.AttemptRunning, StartedAt: &started, LeaseEpoch: lease.Epoch(), Process: self, PacketDigest: ref, ReconciliationFacts: []executiongraph.ReconciliationFact{}}
	g, err = s.saveIntegrationAttempt(ctx, g, integration.ExecutionID, running, lease)
	if err != nil {
		// No effect has been attempted; the receipt alone authorizes nothing new.
		return result, err
	}
	// The running attempt is canonical before any repository effect.
	attemptCtx, cancel := context.WithTimeout(ctx, integration.Envelope.Controls.Timeout)
	defer cancel()
	final := running
	applied, applyErr := s.Workspace.Apply(attemptCtx, g, *fresh.Integration)
	switch {
	case applyErr != nil || !applied.Confirmed || applied.ResultTree == "" || !sameIntegrationEffects(applied.AppliedEffects, fresh.Integration.Effects):
		// The effect may be partial; it is never repeated before reconciliation.
		final.Status, final.ClassificationReason, final.AmbiguousEffect = executiongraph.AttemptUnknown, executiongraph.IntegrationUnknown, true
	case attemptCtx.Err() != nil:
		final.Status, final.ClassificationReason, final.AmbiguousEffect = executiongraph.AttemptUnknown, executiongraph.IntegrationUnknown, true
		final.ResultReference = executiongraph.IntegrationResultDigest(applied.ResultTree)
	default:
		result.ResultTree = applied.ResultTree
		final.ResultReference = executiongraph.IntegrationResultDigest(applied.ResultTree)
		validations, validateErr := s.Workspace.Validate(attemptCtx, g, applied.ResultTree, commands)
		result.Validations = validations
		if wire, e := json.Marshal(validations); e == nil {
			d := sha256.Sum256(wire)
			final.OutputDigest = hex.EncodeToString(d[:])
		}
		switch {
		case attemptCtx.Err() != nil:
			// Interrupted validation commands have unobservable side effects.
			final.Status, final.ClassificationReason, final.AmbiguousEffect = executiongraph.AttemptUnknown, executiongraph.IntegrationUnknown, true
		case validateErr != nil || !validationsPassed(validations, commands):
			final.Status, final.ClassificationReason, final.ExitCode = executiongraph.AttemptFailed, executiongraph.IntegrationFailed, failingExitCode(validations)
		default:
			final.Status, final.ClassificationReason = executiongraph.AttemptSucceeded, executiongraph.IntegrationApplied
		}
	}
	finished := s.now()
	if finished.Before(started) {
		finished = started
	}
	final.FinishedAt = &finished
	// The terminal outcome is recorded even after cancellation.
	g, err = s.saveIntegrationAttempt(context.WithoutCancel(ctx), g, integration.ExecutionID, final, lease)
	if err != nil {
		result.Status, result.Category, result.Next = "recovery_required", "", "reconcile the running integration attempt before any new integration"
		return result, errors.Join(ErrSupervisorRecovery, err)
	}
	rollup, err := executiongraph.RollupSuccessor(g, waivers)
	if err != nil {
		return result, err
	}
	result.Rollup = &rollup
	if final.Status == executiongraph.AttemptUnknown {
		result.Status, result.Category, result.Next = "recovery_required", ErrIntegrationReconcile.Error(), "stage reconcile the integration attempt; its effect is not repeated"
		return result, ErrIntegrationReconcile
	}
	snap.graph = g
	return s.project(context.WithoutCancel(ctx), snap, final.AttemptID, result)
}

// project records the ledger projection of one terminal integration attempt
// from persisted state and its receipt. It is idempotent per attempt.
func (s IntegrateService) project(ctx context.Context, snap integrateSnapshot, attemptID string, result IntegrateResult) (IntegrateResult, error) {
	g := snap.graph
	integration, _ := integrationChildOf(g)
	if len(integration.Attempts) == 0 || integration.Attempts[len(integration.Attempts)-1].AttemptID != attemptID {
		result.Status = "failure"
		return result, executiongraph.ErrInvalidGraph
	}
	attempt := integration.Attempts[len(integration.Attempts)-1]
	receipt, err := s.Store.LoadIntegrationReceipt(ctx, StatusRequest{ProjectID: g.Children[0].Envelope.Scope.ProjectID, ParentID: g.Parent.ExecutionID}, attempt.PacketDigest)
	if err != nil {
		result.Status, result.Category, result.Next = "partial", ErrProjectionPartial.Error(), "repair the projection with stage integrate after the receipt is readable"
		return result, errors.Join(ErrProjectionPartial, err)
	}
	result.ReceiptRef = attempt.PacketDigest
	rollup, err := executiongraph.RollupSuccessor(g, receipt.OptionalWaivers)
	if err != nil {
		return result, err
	}
	result.Rollup = &rollup
	outcome, err := executiongraph.BuildStageOutcome(g, attempt.PacketDigest, receipt, s.now())
	if err == nil && receipt.LedgerDigest != hex.EncodeToString(snap.contract.Execution.StorageRevision[:]) {
		err = ErrStalePreview
	}
	var ref string
	if err == nil {
		ref, err = s.Ledger.RecordStageOutcome(ctx, snap.contract.Execution, outcome)
	}
	if err != nil {
		// The integration attempt is canonical: its effect is confirmed and is
		// never repeated. Only the projection is missing.
		result.Status, result.Category, result.Next = "partial", ErrProjectionPartial.Error(), "repair the stage outcome projection with stage integrate (projection only)"
		return result, errors.Join(ErrProjectionPartial, err)
	}
	result.Outcome, result.OutcomeRef = &outcome, ref
	result.Status, result.Category, result.Next = "success", "", ""
	if rollup.Status == "failure" {
		result.Status, result.Category, result.Next = "failure", "integration_failed", "inspect validator results; a new integration attempt needs its own authority"
		return result, executiongraph.ErrIntegrationFailed
	}
	return result, nil
}

func (s IntegrateService) saveIntegrationAttempt(ctx context.Context, g executiongraph.SuccessorGraph, childID string, a executiongraph.SuccessorAttempt, lease SupervisorLease) (executiongraph.SuccessorGraph, error) {
	next := cloneSupervisorGraph(g)
	for i, c := range next.Children {
		if c.ExecutionID != childID {
			continue
		}
		attempts := next.Children[i].Attempts
		if n := len(attempts); n > 0 && attempts[n-1].AttemptID == a.AttemptID {
			attempts[n-1] = a
		} else {
			attempts = append(attempts, a)
		}
		next.Children[i].Attempts = attempts
		saved, err := s.Store.Save(ctx, next, lease)
		if err != nil {
			return g, err
		}
		for _, sc := range saved.Children {
			if sc.ExecutionID == childID {
				if n := len(sc.Attempts); n == 0 || sc.Attempts[n-1].AttemptID != a.AttemptID || sc.Attempts[n-1].Status != a.Status {
					return g, ErrSupervisorRecovery
				}
			}
		}
		return saved, nil
	}
	return g, executiongraph.ErrInvalidGraph
}

func sameIntegrationEffects(a, b []executiongraph.Effect) bool {
	if len(a) != len(b) {
		return false
	}
	key := func(e executiongraph.Effect) string { return e.Kind + "\x00" + e.Target }
	x, y := make([]string, len(a)), make([]string, len(b))
	for i := range a {
		x[i], y[i] = key(a[i]), key(b[i])
	}
	sort.Strings(x)
	sort.Strings(y)
	for i := range x {
		if x[i] != y[i] {
			return false
		}
	}
	return true
}

// validationsPassed requires one passing result per reviewed command.
func validationsPassed(results []executiongraph.ValidationResult, commands []gitworkspace.ValidationCommand) bool {
	if len(results) != len(commands) || len(results) == 0 {
		return false
	}
	want := map[string]bool{}
	for _, c := range commands {
		want[c.Reference] = true
	}
	for _, r := range results {
		if !want[r.CommandReference] || r.ExitCode != 0 || len(r.OutputDigest) != 64 {
			return false
		}
		delete(want, r.CommandReference)
	}
	return len(want) == 0
}

func failingExitCode(results []executiongraph.ValidationResult) int {
	for _, r := range results {
		if r.ExitCode != 0 {
			return r.ExitCode
		}
	}
	return 1
}

// CurrentProcess observes this process for integration attempt correlation.
func CurrentProcess() (*executiongraph.ProcessCorrelation, error) {
	p, err := executiongraph.ObserveProcessCorrelation(int64(os.Getpid()))
	if err != nil {
		return nil, fmt.Errorf("observe integrating process: %w", err)
	}
	return p, nil
}
