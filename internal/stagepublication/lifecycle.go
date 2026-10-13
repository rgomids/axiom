package stagepublication

import (
	"context"

	"github.com/rgomids/axiom/internal/executiongraph"
	"github.com/rgomids/axiom/internal/local"
	"github.com/rgomids/axiom/internal/stageexecution"
)

// LocalStageState composes protected persistence with the caller's bound
// Execution/Workflow admission loader. Loading never resolves another Profile.
type LocalStageState struct {
	Graphs   local.GraphStore
	Contract func(context.Context, stageexecution.StatusRequest) (stageexecution.Contract, error)
}

func (s LocalStageState) StageSnapshot(ctx context.Context, r stageexecution.StatusRequest) (stageexecution.Contract, error) {
	if s.Contract == nil {
		return stageexecution.Contract{}, stageexecution.ErrAdmission
	}
	c, err := s.Contract(ctx, r)
	if err != nil {
		return c, err
	}
	g, err := s.Graphs.LoadSuccessor(ctx, r.ProjectID, r.ParentID)
	if err != nil {
		return c, err
	}
	c.Graph = &executiongraph.GraphDocument{Successor: &g}
	return c, nil
}
func (s LocalStageState) RequestCancel(ctx context.Context, g executiongraph.SuccessorGraph, r executiongraph.CancelRequest) (executiongraph.SuccessorGraph, error) {
	return s.Graphs.RequestCancel(ctx, g, r)
}
