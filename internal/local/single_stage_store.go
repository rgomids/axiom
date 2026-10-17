package local

import (
	"context"
	"crypto/sha256"
	"errors"
	"os"
	"path/filepath"
	"reflect"

	"github.com/rgomids/axiom/internal/executiongraph"
	"github.com/rgomids/axiom/internal/workflow"
)

// SingleStageStore persists ADR-0027 single-stage operational ledgers at
// single-stage-runs/v1/<project>/<scope-digest>.json with their immutable
// version-2 authority receipts and retained previews beside them. The run
// lease (keyed by Execution) is always acquired before these CAS locks.
type SingleStageStore struct {
	root  string
	hooks publicationHooks
}

func NewSingleStageStore(root string) (SingleStageStore, error) {
	if !filepath.IsAbs(root) || filepath.Clean(root) == string(filepath.Separator) {
		return SingleStageStore{}, ErrUnsafe
	}
	canonical, err := trustedCanonical(root)
	if err != nil {
		return SingleStageStore{}, err
	}
	return SingleStageStore{root: canonical}, nil
}

func singleStageName(scopeDigest string) string { return scopeDigest + ".json" }

func (s SingleStageStore) openProject(projectID string, create bool) (*os.Root, *os.Root, *os.Root, *os.Root, error) {
	if !validGraphToken(projectID) || projectID == "." || projectID == ".." || !platformEntryName(projectID) {
		return nil, nil, nil, nil, ErrUnsafe
	}
	openRoot, openChild := existingPrivateRoot, existingPrivateChild
	if create {
		openRoot, openChild = privateRoot, privateChild
	}
	root, err := openRoot(s.root)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	runs, err := openChild(root, "single-stage-runs")
	if err != nil {
		root.Close()
		return nil, nil, nil, nil, err
	}
	version, err := openChild(runs, "v1")
	if err != nil {
		runs.Close()
		root.Close()
		return nil, nil, nil, nil, err
	}
	project, err := openChild(version, projectID)
	if err != nil {
		version.Close()
		runs.Close()
		root.Close()
		return nil, nil, nil, nil, err
	}
	return root, runs, version, project, nil
}

type singleStageHandle struct {
	roots [4]*os.Root
	locks []*os.File
}

func (h singleStageHandle) project() *os.Root { return h.roots[3] }
func (h singleStageHandle) Close() {
	closeFiles(h.locks)
	for i := len(h.roots) - 1; i >= 0; i-- {
		h.roots[i].Close()
	}
}

func (s SingleStageStore) lock(projectID string, exclusive, create bool) (singleStageHandle, error) {
	root, runs, version, project, err := s.openProject(projectID, create)
	if err != nil {
		return singleStageHandle{}, err
	}
	h := singleStageHandle{roots: [4]*os.Root{root, runs, version, project}}
	h.locks, err = lockRoots(exclusive, root, runs, version, project)
	if err != nil {
		h.locks = nil
		h.Close()
		return singleStageHandle{}, err
	}
	return h, nil
}

func readSingleStageFile(root *os.Root, name string) ([]byte, error) {
	pending, err := protocolStatePresent(root)
	if err != nil {
		return nil, err
	}
	if pending {
		return nil, ErrRecoveryRequired
	}
	return readPrivateFileBounded(root, name, executiongraph.MaxSingleStageRunBytes)
}

func decodeStoredSingleStage(wire []byte, projectID, scopeDigest string) (executiongraph.SingleStageRun, error) {
	r, err := executiongraph.DecodeSingleStageRun(wire)
	if err != nil {
		return executiongraph.SingleStageRun{}, err
	}
	if r.Scope.ProjectID != projectID || r.ScopeDigest != scopeDigest {
		return executiongraph.SingleStageRun{}, ErrUnsafe
	}
	return r, nil
}

func (s SingleStageStore) Load(ctx context.Context, projectID, scopeDigest string) (executiongraph.SingleStageRun, error) {
	if err := ctx.Err(); err != nil {
		return executiongraph.SingleStageRun{}, err
	}
	if !hexDigestName.MatchString(singleStageName(scopeDigest)) {
		return executiongraph.SingleStageRun{}, ErrUnsafe
	}
	h, err := s.lock(projectID, false, false)
	if err != nil {
		return executiongraph.SingleStageRun{}, err
	}
	defer h.Close()
	wire, err := readSingleStageFile(h.project(), singleStageName(scopeDigest))
	if err != nil {
		return executiongraph.SingleStageRun{}, err
	}
	return decodeStoredSingleStage(wire, projectID, scopeDigest)
}

// LoadAuthority returns a receipt only when it is content-addressed, listed by
// the ledger and bound to its scope, Execution3 generation and preview.
func (s SingleStageStore) LoadAuthority(ctx context.Context, r executiongraph.SingleStageRun, ref string) (executiongraph.SingleStageAuthority, error) {
	if err := ctx.Err(); err != nil {
		return executiongraph.SingleStageAuthority{}, err
	}
	listed := false
	for _, known := range r.Authorities {
		listed = listed || known == ref
	}
	if !listed || !hexDigestName.MatchString(ref+".json") {
		return executiongraph.SingleStageAuthority{}, ErrUnsafe
	}
	h, err := s.lock(r.Scope.ProjectID, false, false)
	if err != nil {
		return executiongraph.SingleStageAuthority{}, err
	}
	defer h.Close()
	wire, err := readSingleStageFile(h.project(), ref+".json")
	if err != nil {
		return executiongraph.SingleStageAuthority{}, err
	}
	a, err := executiongraph.DecodeSingleStageAuthority(wire)
	if err != nil {
		return executiongraph.SingleStageAuthority{}, err
	}
	if hexDigest(wire) != ref || executiongraph.ValidateSingleStageBindingAuthority(r, a) != nil {
		return executiongraph.SingleStageAuthority{}, ErrUnsafe
	}
	return a, nil
}

// executionGeneration rereads the bound Execution3 through the state root
// handle whose exclusive lock is already held; every Execution3 writer takes
// that lock first, so the generation cannot change before the commit point.
func executionGeneration(root *os.Root, state workflow.State) (string, error) {
	current := root
	for _, part := range []string{"executions", "v1", state.ProjectID} {
		next, err := existingPrivateChild(current, part)
		if current != root {
			current.Close()
		}
		if err != nil {
			return "", err
		}
		current = next
	}
	defer current.Close()
	wire, err := readExecutionFile(current, executionName(state.RepositoryKey, state.WorkItem))
	if err != nil {
		return "", err
	}
	return hexDigest(wire), nil
}

// CommitAuthority is the single commit point for a first-attempt or retry
// receipt. The receipt is published as the protected companion of the ledger
// generation that lists it, then both are read back. A first attempt creates
// the ledger; a retry consumes exactly the reviewed ledger generation.
func (s SingleStageStore) CommitAuthority(ctx context.Context, next executiongraph.SingleStageRun, a executiongraph.SingleStageAuthority, state workflow.State, lease *RunLease) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	ref, err := executiongraph.SingleStageAuthorityDigest(a)
	if err != nil || executiongraph.ValidateSingleStageBindingAuthority(next, a) != nil || len(next.Authorities) != int(a.AttemptNumber) || next.Authorities[len(next.Authorities)-1] != ref || len(next.Attempts) != int(a.AttemptNumber)-1 || next.CancelRequest != nil {
		return executiongraph.ErrInvalidGraph
	}
	if state.ProjectID != next.Scope.ProjectID || state.ExecutionID != next.Scope.ExecutionID || state.Revision != next.Binding.WorkflowRevision || string(state.Stage) != next.Scope.StageID || state.StageOrdinal != next.Scope.StageOrdinal || hexDigestRevision(state.StorageRevision) != a.ExecutionGeneration {
		return ErrConflict
	}
	wire, err := executiongraph.EncodeSingleStageRun(next)
	if err != nil {
		return err
	}
	receipt, err := executiongraph.EncodeSingleStageAuthority(a)
	if err != nil {
		return err
	}
	if lease == nil {
		return ErrConflict
	}
	lease.mu.Lock()
	defer lease.mu.Unlock()
	if lease.validateLocked() != nil || lease.stateRoot != s.root || lease.projectID != next.Scope.ProjectID || lease.parentID != next.Scope.ExecutionID || lease.Epoch < next.RunEpoch {
		return ErrConflict
	}
	h, err := s.lock(next.Scope.ProjectID, true, true)
	if err != nil {
		return err
	}
	defer h.Close()
	generation, err := executionGeneration(h.roots[0], state)
	if err != nil {
		return err
	}
	if generation != a.ExecutionGeneration {
		return ErrConflict
	}
	name := singleStageName(next.ScopeDigest)
	var expected []byte
	create := a.AttemptNumber == 1
	if create {
		if _, err := readPrivateFileBounded(h.project(), name, executiongraph.MaxSingleStageRunBytes); err == nil {
			return ErrConflict
		} else if !errors.Is(err, os.ErrNotExist) && !errors.Is(err, ErrNotFound) {
			return err
		}
	} else {
		expected, err = readSingleStageFile(h.project(), name)
		if err != nil {
			return err
		}
		if hexDigest(expected) != a.LedgerGeneration {
			return ErrConflict
		}
		prior, err := decodeStoredSingleStage(expected, next.Scope.ProjectID, next.ScopeDigest)
		if err != nil {
			return err
		}
		if executiongraph.CheckSingleStageRetryAuthority(prior, a) != nil || executiongraph.ValidateSingleStageUpdate(prior, next) != nil || !reflect.DeepEqual(prior.Attempts, next.Attempts) || prior.RunEpoch != next.RunEpoch {
			return executiongraph.ErrInvalidGraph
		}
	}
	hooks := s.hooks
	hooks.contentLimit = executiongraph.MaxSingleStageRunBytes
	hooks.companion = ref + ".json"
	hooks.companionWire = receipt
	beforeCommit := hooks.beforeCommit
	hooks.beforeCommit = func() error {
		if beforeCommit != nil {
			if err := beforeCommit(); err != nil {
				return err
			}
		}
		return lease.validateLocked()
	}
	if err := publishFile(ctx, h.project(), name, expected, wire, create, hooks); err != nil {
		return err
	}
	return confirmDispatch(h.project(), name, wire, hooks.companion, receipt, executiongraph.MaxSingleStageRunBytes)
}

// Save writes attempt state under the held lease. New attempts carry the held
// epoch; only attempts of that epoch may change; the run epoch advances to it.
func (s SingleStageStore) Save(ctx context.Context, r executiongraph.SingleStageRun, lease *RunLease) (executiongraph.SingleStageRun, error) {
	return s.save(ctx, r, lease, false)
}

func (s SingleStageStore) save(ctx context.Context, r executiongraph.SingleStageRun, lease *RunLease, reconciliation bool) (executiongraph.SingleStageRun, error) {
	if err := ctx.Err(); err != nil {
		return r, err
	}
	if lease == nil || r.StorageRevision == ([32]byte{}) {
		return r, ErrConflict
	}
	lease.mu.Lock()
	defer lease.mu.Unlock()
	if err := lease.validateLocked(); err != nil {
		return r, err
	}
	if lease.stateRoot != s.root || lease.projectID != r.Scope.ProjectID || lease.parentID != r.Scope.ExecutionID || lease.Epoch < r.RunEpoch {
		return r, ErrConflict
	}
	next := executiongraph.CloneSingleStageRun(r)
	next.RunEpoch = lease.Epoch
	wire, err := executiongraph.EncodeSingleStageRun(next)
	if err != nil {
		return r, err
	}
	h, err := s.lock(r.Scope.ProjectID, true, false)
	if err != nil {
		return r, err
	}
	defer h.Close()
	name := singleStageName(r.ScopeDigest)
	expected, err := readSingleStageFile(h.project(), name)
	if err != nil {
		return r, err
	}
	if sha256.Sum256(expected) != r.StorageRevision {
		return r, ErrConflict
	}
	old, err := decodeStoredSingleStage(expected, r.Scope.ProjectID, r.ScopeDigest)
	if err != nil || executiongraph.ValidateSingleStageUpdate(old, next) != nil || !reflect.DeepEqual(old.Authorities, next.Authorities) || !reflect.DeepEqual(old.CancelRequest, next.CancelRequest) {
		return r, executiongraph.ErrInvalidGraph
	}
	if reconciliation && len(next.Attempts) != len(old.Attempts) {
		return r, ErrConflict
	}
	for j, a := range next.Attempts {
		if j >= len(old.Attempts) {
			if a.LeaseEpoch != lease.Epoch || len(a.ReconciliationFacts) != 0 || j >= len(next.Authorities) {
				return r, ErrConflict
			}
			continue
		}
		before, after := old.Attempts[j], a
		before.ReconciliationFacts, after.ReconciliationFacts = nil, nil
		if !reflect.DeepEqual(before, after) && (reconciliation || a.LeaseEpoch != lease.Epoch) {
			return r, ErrConflict
		}
		if !reconciliation && !reflect.DeepEqual(old.Attempts[j].ReconciliationFacts, a.ReconciliationFacts) {
			return r, ErrConflict
		}
	}
	if err := lease.validateLocked(); err != nil {
		return r, err
	}
	hooks := s.hooks
	hooks.contentLimit = executiongraph.MaxSingleStageRunBytes
	beforeCommit := hooks.beforeCommit
	hooks.beforeCommit = func() error {
		if beforeCommit != nil {
			if err := beforeCommit(); err != nil {
				return err
			}
		}
		return lease.validateLocked()
	}
	if err := publishFile(ctx, h.project(), name, expected, wire, false, hooks); err != nil {
		return r, err
	}
	next.StorageRevision = sha256.Sum256(wire)
	return next, nil
}

// RequestCancel is the only ledger write without the run lease: a write-once
// control message, serialized with attempt saves by the ledger CAS.
func (s SingleStageStore) RequestCancel(ctx context.Context, r executiongraph.SingleStageRun, request executiongraph.CancelRequest) (executiongraph.SingleStageRun, error) {
	if err := ctx.Err(); err != nil {
		return r, err
	}
	if r.StorageRevision == ([32]byte{}) {
		return r, ErrConflict
	}
	next := executiongraph.CloneSingleStageRun(r)
	if r.CancelRequest != nil {
		if *r.CancelRequest != request {
			return r, ErrConflict
		}
		return r, nil
	}
	next.CancelRequest = &request
	if executiongraph.ValidateSingleStageUpdate(r, next) != nil {
		return r, executiongraph.ErrInvalidGraph
	}
	wire, err := executiongraph.EncodeSingleStageRun(next)
	if err != nil {
		return r, err
	}
	h, err := s.lock(r.Scope.ProjectID, true, false)
	if err != nil {
		return r, err
	}
	defer h.Close()
	name := singleStageName(r.ScopeDigest)
	expected, err := readSingleStageFile(h.project(), name)
	if err != nil {
		return r, err
	}
	if sha256.Sum256(expected) != r.StorageRevision {
		return r, ErrConflict
	}
	hooks := s.hooks
	hooks.contentLimit = executiongraph.MaxSingleStageRunBytes
	if err := publishFile(ctx, h.project(), name, expected, wire, false, hooks); err != nil {
		return r, err
	}
	next.StorageRevision = sha256.Sum256(wire)
	return next, nil
}

// AppendReconciliation acquires the run lease, so no supervisor can start
// between the apply-time observation and the fact publication.
func (s SingleStageStore) AppendReconciliation(ctx context.Context, r executiongraph.SingleStageRun, fact executiongraph.ReconciliationFact, observe func(context.Context, executiongraph.SingleStageRun, string) (executiongraph.ReconciliationObservations, error)) (executiongraph.SingleStageRun, error) {
	if observe == nil || executiongraph.ValidateSingleStageRun(r) != nil {
		return r, executiongraph.ErrInvalidGraph
	}
	leases, err := NewRunLeaseStore(s.root)
	if err != nil {
		return r, err
	}
	lease, err := leases.Acquire(ctx, r.Scope.ProjectID, r.Scope.ExecutionID, r.RunEpoch)
	if err != nil {
		return r, err
	}
	defer lease.Close()
	observations, err := observe(ctx, r, fact.AttemptID)
	if err != nil {
		return r, err
	}
	observations.LeaseHeld = false
	next, err := executiongraph.AppendSingleStageReconciliation(r, fact, observations)
	if err != nil {
		return r, err
	}
	return s.save(ctx, next, lease, true)
}

// CommitDispatchInputs retains the exact reviewed preview, inert until the
// ledger generation that lists its authority is committed.
func (s SingleStageStore) CommitDispatchInputs(ctx context.Context, d executiongraph.DispatchInputs) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	wire, err := executiongraph.EncodeDispatchInputs(d)
	if err != nil {
		return "", err
	}
	h, err := s.lock(d.ProjectID, true, true)
	if err != nil {
		return "", err
	}
	defer h.Close()
	return commitInertInputs(ctx, h.project(), wire, s.hooks)
}

func (s SingleStageStore) LoadDispatchInputs(ctx context.Context, projectID, executionID, previewDigest string) (executiongraph.DispatchInputs, error) {
	if err := ctx.Err(); err != nil {
		return executiongraph.DispatchInputs{}, err
	}
	if !validGraphAddress(projectID, executionID) {
		return executiongraph.DispatchInputs{}, ErrUnsafe
	}
	h, err := s.lock(projectID, false, false)
	if err != nil {
		return executiongraph.DispatchInputs{}, err
	}
	defer h.Close()
	return findDispatchInputs(h.project(), projectID, executionID, previewDigest)
}

func classifySingleStage(projectID, file string, wire []byte) InventoryKind {
	if a, err := executiongraph.DecodeSingleStageAuthority(wire); err == nil {
		if a.ProjectID != projectID || hexDigest(wire)+".json" != file {
			return InventoryMalformed
		}
		return InventorySingleStageAuthority
	} else if errors.Is(err, executiongraph.ErrUnsupportedSingleStageFormat) {
		return versionedFailureWithin(wire, executiongraph.SingleStageAuthorityVersion)
	}
	if kind := classifyDispatchInputs(projectID, file, wire); kind != "" {
		return kind
	}
	r, err := executiongraph.DecodeSingleStageRun(wire)
	if errors.Is(err, executiongraph.ErrUnsupportedSingleStageFormat) {
		return versionedFailureWithin(wire, executiongraph.SingleStageRunVersion)
	}
	if err != nil {
		return InventoryMalformed
	}
	if r.Scope.ProjectID != projectID || singleStageName(r.ScopeDigest) != file {
		return InventoryMalformed
	}
	return InventorySingleStageRun
}
