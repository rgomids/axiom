package local

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"os"

	"github.com/rgomids/axiom/internal/executiongraph"
	"github.com/rgomids/axiom/internal/workflow"
)

// RecordStageOutcome projects one terminal integration outcome beside the
// Execution ledger. Execution3 bytes are never rewritten: the immutable record
// binds the exact ledger generation, which must still be canonical under the
// exclusive Execution locks. One integration attempt has at most one outcome.
func (s WorkflowStore) RecordStageOutcome(ctx context.Context, state workflow.State, o executiongraph.StageOutcome) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	wire, err := executiongraph.EncodeStageOutcome(o)
	if err != nil {
		return "", err
	}
	ref := hexDigest(wire)
	if o.ProjectID != state.ProjectID || o.ExecutionID != state.ExecutionID || o.ExecutionRevision != state.Revision || o.LedgerDigest != hexDigestRevision(state.StorageRevision) || state.FormatVersion != workflow.ProfileConfiguredFormatVersion {
		return "", ErrUnsafe
	}
	root, executions, version, project, err := s.openProject(state.ProjectID, false)
	if err != nil {
		return "", err
	}
	defer root.Close()
	defer executions.Close()
	defer version.Close()
	defer project.Close()
	locks, err := lockRoots(true, root, executions, version, project)
	if err != nil {
		return "", err
	}
	defer closeFiles(locks)
	canonical, err := readExecutionFile(project, executionName(state.RepositoryKey, state.WorkItem))
	if err != nil {
		return "", err
	}
	if sha256.Sum256(canonical) != state.StorageRevision {
		return "", ErrConflict
	}
	existing, err := s.stageOutcomes(project, o.ExecutionID)
	if err != nil {
		return "", err
	}
	for name, prior := range existing {
		if prior.Rollup.IntegrationAttemptID != o.Rollup.IntegrationAttemptID {
			continue
		}
		if name != ref {
			return "", ErrConflict
		}
		// Content addressing makes a repeated projection a read-back.
		return ref, nil
	}
	hooks := s.hooks
	hooks.contentLimit = executiongraph.MaxStageOutcomeBytes
	if err := publishFile(ctx, project, ref+".json", nil, wire, true, hooks); err != nil {
		return "", err
	}
	confirmed, err := readPrivateFileBounded(project, ref+".json", executiongraph.MaxStageOutcomeBytes)
	if err != nil || !bytes.Equal(confirmed, wire) {
		return "", &PublicationError{Stage: FaultF8, Committed: true, Err: ErrRecoveryRequired}
	}
	return ref, nil
}

// LoadStageOutcomes returns the projected outcomes of one Execution keyed by
// reference. Records for other Executions in the Project are ignored.
func (s WorkflowStore) LoadStageOutcomes(ctx context.Context, projectID, executionID string) (map[string]executiongraph.StageOutcome, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	root, executions, version, project, err := s.openProject(projectID, false)
	if errors.Is(err, ErrNotFound) || errors.Is(err, os.ErrNotExist) {
		return map[string]executiongraph.StageOutcome{}, nil
	}
	if err != nil {
		return nil, err
	}
	defer root.Close()
	defer executions.Close()
	defer version.Close()
	defer project.Close()
	locks, err := lockRoots(false, root, executions, version, project)
	if err != nil {
		return nil, err
	}
	defer closeFiles(locks)
	pending, err := protocolStatePresent(project)
	if err != nil {
		return nil, err
	}
	if pending {
		return nil, ErrRecoveryRequired
	}
	return s.stageOutcomes(project, executionID)
}

func (s WorkflowStore) stageOutcomes(project *os.Root, executionID string) (map[string]executiongraph.StageOutcome, error) {
	names, err := readDirectoryNamesBounded(project, maxLocalDirectoryEntries)
	if err != nil {
		return nil, err
	}
	result := map[string]executiongraph.StageOutcome{}
	for _, name := range names {
		if !hexDigestName.MatchString(name) {
			continue
		}
		// Execution ledgers and receipts share this directory; read within the
		// directory's own bound and keep only decodable stage outcomes.
		wire, err := readPrivateFileBounded(project, name, MaxExecutionBytes)
		if err != nil {
			return nil, err
		}
		o, err := executiongraph.DecodeStageOutcome(wire)
		if err != nil {
			continue
		}
		if hexDigest(wire)+".json" != name {
			return nil, ErrUnsafe
		}
		if o.ExecutionID == executionID {
			result[name[:len(name)-len(".json")]] = o
		}
	}
	return result, nil
}

func classifyStageOutcome(projectID, file string, wire []byte) InventoryKind {
	o, err := executiongraph.DecodeStageOutcome(wire)
	if err != nil {
		return ""
	}
	if o.ProjectID != projectID || hexDigest(wire)+".json" != file {
		return InventoryMalformed
	}
	return InventoryStageOutcome
}
