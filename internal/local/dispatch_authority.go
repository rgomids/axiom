package local

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"github.com/rgomids/axiom/internal/executiongraph"
	"github.com/rgomids/axiom/internal/workflow"
	"os"
)

// CommitDispatch publishes an inert immutable receipt before the graph's single
// commit point. The owned recovery marker names both exact generations. Readers
// refuse all intermediate states; the receipt alone never authorizes dispatch.
func (s GraphStore) CommitDispatch(ctx context.Context, g executiongraph.SuccessorGraph, a executiongraph.DispatchAuthority, preview string, lease *RunLease) error {
	if executiongraph.ValidateGraphDispatchAuthority(g, a, preview) != nil {
		return executiongraph.ErrInvalidGraph
	}
	for _, c := range g.Children {
		if len(c.Attempts) != 0 {
			return executiongraph.ErrInvalidGraph
		}
	}
	if lease == nil {
		return ErrConflict
	}
	lease.mu.Lock()
	defer lease.mu.Unlock()
	if lease.validateLocked() != nil || lease.stateRoot != s.root || lease.projectID != a.ProjectID || lease.parentID != a.ParentID {
		return ErrConflict
	}
	wire, err := executiongraph.EncodeSuccessorGraph(g)
	if err != nil {
		return err
	}
	receipt, err := executiongraph.EncodeDispatchAuthority(a)
	if err != nil {
		return err
	}
	root, graphs, version, project, err := s.openProject(a.ProjectID, true)
	if err != nil {
		return err
	}
	defer root.Close()
	defer graphs.Close()
	defer version.Close()
	defer project.Close()
	locks, err := lockRoots(true, root, graphs, version, project)
	if err != nil {
		return err
	}
	defer closeFiles(locks)
	hooks := s.hooks
	hooks.contentLimit = executiongraph.MaxGraphBytes
	hooks.companion = g.DispatchAuthorityRef + ".json"
	hooks.companionWire = receipt
	hooks.beforeCommit = lease.validateLocked
	if err := publishFile(ctx, project, graphName(a.ParentID), nil, wire, true, hooks); err != nil {
		return err
	}
	return confirmDispatch(project, graphName(a.ParentID), wire, hooks.companion, receipt, executiongraph.MaxGraphBytes)
}

func confirmDispatch(root *os.Root, name string, wire []byte, authorityName string, receipt []byte, limit int) error {
	actual, err := readPrivateFileBounded(root, name, limit)
	if err != nil || sha256.Sum256(actual) != sha256.Sum256(wire) {
		return &PublicationError{Stage: FaultF8, Committed: true, Err: ErrRecoveryRequired}
	}
	actual, err = readPrivateFileBounded(root, authorityName, executiongraph.MaxDispatchAuthorityBytes)
	if err != nil || sha256.Sum256(actual) != sha256.Sum256(receipt) {
		return &PublicationError{Stage: FaultF8, Committed: true, Err: ErrRecoveryRequired}
	}
	return nil
}

func (s GraphStore) LoadDispatchAuthority(ctx context.Context, projectID, parentID, preview string) (executiongraph.DispatchAuthority, error) {
	g, err := s.LoadSuccessor(ctx, projectID, parentID)
	if err != nil {
		return executiongraph.DispatchAuthority{}, err
	}
	root, graphs, version, project, err := s.openProject(projectID, false)
	if err != nil {
		return executiongraph.DispatchAuthority{}, err
	}
	defer root.Close()
	defer graphs.Close()
	defer version.Close()
	defer project.Close()
	locks, err := lockRoots(false, root, graphs, version, project)
	if err != nil {
		return executiongraph.DispatchAuthority{}, err
	}
	defer closeFiles(locks)
	wire, err := readPublishedFile(project, g.DispatchAuthorityRef+".json")
	if err != nil {
		return executiongraph.DispatchAuthority{}, err
	}
	a, err := executiongraph.DecodeDispatchAuthority(wire)
	if err != nil || hexDigest(wire) != g.DispatchAuthorityRef || executiongraph.ValidateGraphDispatchAuthority(g, a, preview) != nil {
		return executiongraph.DispatchAuthority{}, ErrUnsafe
	}
	return a, nil
}

// CommitSingleDispatch preserves Execution3 bytes: the protected receipt binds
// the exact ledger generation and first-attempt identity outside its codec.
func (s WorkflowStore) CommitSingleDispatch(ctx context.Context, state workflow.State, a executiongraph.DispatchAuthority, preview string) error {
	wire, err := encodeExecution(state)
	if err != nil {
		return err
	}
	if executiongraph.ValidateDispatchAuthority(a) != nil || a.Scope != "single-first-attempt" || a.PreviewDigest != preview || a.ProjectID != state.ProjectID || a.ParentID != state.ExecutionID || a.Revision != state.Revision || a.SingleStageID != string(state.Stage) || a.SingleStageOrdinal != state.StageOrdinal || a.LedgerDigest != hexDigest(wire) || state.FormatVersion != workflow.ProfileConfiguredFormatVersion || a.RecordedAt.After(state.UpdatedAt) {
		return ErrUnsafe
	}
	receipt, _ := executiongraph.EncodeDispatchAuthority(a)
	digest, _ := executiongraph.DispatchAuthorityDigest(a)
	root, executions, version, project, err := s.openProject(state.ProjectID, true)
	if err != nil {
		return err
	}
	defer root.Close()
	defer executions.Close()
	defer version.Close()
	defer project.Close()
	locks, err := lockRoots(true, root, executions, version, project)
	if err != nil {
		return err
	}
	defer closeFiles(locks)
	name := executionName(state.RepositoryKey, state.WorkItem)
	var expected []byte
	create := state.StorageRevision == ([32]byte{})
	if !create {
		expected, err = readExecutionFile(project, name)
		if err != nil {
			return err
		}
		if sha256.Sum256(expected) != state.StorageRevision {
			return ErrConflict
		}
		if bytes.Equal(expected, wire) {
			return ErrConflict
		}
		prior, err := decodeExecution(expected)
		if err != nil || !sameExecutionBinding(prior, state) {
			return ErrUnsafe
		}

	}
	// A single stage's first-attempt scope is consumed by its exact canonical
	// ledger generation; a later authority cannot widen it into retry authority.
	names, e := readDirectoryNamesBounded(project, maxLocalDirectoryEntries)
	if e != nil {
		return e
	}
	for _, name := range names {
		if !hexDigestName.MatchString(name) {
			continue
		}
		b, e := readPrivateFileBounded(project, name, MaxExecutionBytes)
		if e != nil {
			return e
		}
		prior, e := executiongraph.DecodeDispatchAuthority(b)
		if e != nil || prior.ParentID != a.ParentID || prior.SingleStageID != a.SingleStageID || prior.SingleStageOrdinal != a.SingleStageOrdinal {
			continue
		}
		if prior.Scope == "single-first-attempt" && (name != digest+".json" || len(expected) != 0 && prior.LedgerDigest == hexDigest(expected)) {
			return ErrConflict
		}
		if prior.SingleAttemptID == a.SingleAttemptID && name != digest+".json" {
			return ErrConflict
		}
	}
	hooks := s.hooks
	hooks.contentLimit = MaxExecutionBytes
	hooks.companion = digest + ".json"
	hooks.companionWire = receipt
	if err := publishFile(ctx, project, name, expected, wire, create, hooks); err != nil {
		return err
	}
	return confirmDispatch(project, name, wire, hooks.companion, receipt, MaxExecutionBytes)
}

func (s WorkflowStore) LoadSingleDispatchAuthority(ctx context.Context, state workflow.State, reference, preview, attemptID string) (executiongraph.DispatchAuthority, error) {
	if !hexDigestName.MatchString(reference + ".json") {
		return executiongraph.DispatchAuthority{}, ErrUnsafe
	}
	current, err := s.Load(ctx, state.ProjectID, state.RepositoryKey, state.WorkItem)
	if err != nil {
		return executiongraph.DispatchAuthority{}, err
	}
	root, executions, version, project, err := s.openProject(state.ProjectID, false)
	if err != nil {
		return executiongraph.DispatchAuthority{}, err
	}
	defer root.Close()
	defer executions.Close()
	defer version.Close()
	defer project.Close()
	locks, err := lockRoots(false, root, executions, version, project)
	if err != nil {
		return executiongraph.DispatchAuthority{}, err
	}
	defer closeFiles(locks)
	canonical, e := readExecutionFile(project, executionName(state.RepositoryKey, state.WorkItem))
	if e != nil {
		return executiongraph.DispatchAuthority{}, e
	}
	current, e = decodeExecution(canonical)
	if e != nil {
		return executiongraph.DispatchAuthority{}, e
	}
	current.StorageRevision = sha256.Sum256(canonical)
	wire, err := readPublishedFile(project, reference+".json")
	if err != nil {
		return executiongraph.DispatchAuthority{}, err
	}
	a, err := executiongraph.DecodeDispatchAuthority(wire)
	digest, _ := executiongraph.DispatchAuthorityDigest(a)
	if err != nil || hexDigest(wire) != reference || digest != reference || a.Scope != "single-first-attempt" || a.ParentID != current.ExecutionID || a.ProjectID != current.ProjectID || a.Revision != current.Revision || a.SingleStageID != string(current.Stage) || a.SingleStageOrdinal != current.StageOrdinal || a.PreviewDigest != preview || a.SingleAttemptID != attemptID || a.LedgerDigest != hexDigestRevision(current.StorageRevision) {
		return executiongraph.DispatchAuthority{}, ErrUnsafe
	}
	return a, nil
}
func hexDigestRevision(revision [32]byte) string { return fmt.Sprintf("%x", revision) }
