package main

import (
	"context"
	"errors"
	"os"

	"github.com/rgomids/axiom/internal/executiongraph"
	"github.com/rgomids/axiom/internal/local"
	"github.com/rgomids/axiom/internal/workflow"
)

// loadStageSingle returns the pre-cutover single-stage ledger of the
// Execution's current stage, or nil when that stage has none. It survives only
// as a read-only input of the not-yet-replaced delivery (S3) and coordination
// (S2) paths; no operational writer, supervisor or dispatch remains. Corruption
// is never treated as absent.
func loadStageSingle(ctx context.Context, singles local.SingleStageStore, state workflow.State) (*executiongraph.SingleStageRun, error) {
	item := state.WorkItem
	scope := executiongraph.SingleStageScope{ProjectID: state.ProjectID, RepositoryKey: state.RepositoryKey, WorkItem: item.Provider + ":" + item.Resource + "#" + item.ExternalID, ExecutionID: state.ExecutionID, StageID: string(state.Stage), StageOrdinal: state.StageOrdinal}
	digest, err := executiongraph.SingleStageScopeDigest(scope)
	if err != nil {
		return nil, nil
	}
	run, err := singles.Load(ctx, state.ProjectID, digest)
	if errors.Is(err, local.ErrNotFound) || errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &run, nil
}
