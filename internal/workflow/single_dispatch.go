package workflow

import (
	"context"
	"errors"

	"github.com/rgomids/axiom/internal/executiongraph"
	"github.com/rgomids/axiom/internal/runtimeprofile"
)

var ErrSingleDispatch = errors.New("invalid_single_dispatch")

// SingleDispatch extends the sequential execution seam without a graph parent
// or an integration child. The immutable receipt owns the first attempt ID;
// Execution3's closed wire format is preserved.
type SingleDispatch struct {
	State           State
	AttemptID       string
	ResolvedProfile *runtimeprofile.ResolvedProfile
	InputDigest     string
	Workspace       string
	Inbox           string
	AxiomDigest     string
}

// SequentialStageSupervisor uses the same frozen invocation adapter, stdin
// packet, capability/auth checks, final drain and ClassifyAttempt boundary as
// the graph supervisor. It starts one bounded attempt, never another child.
type SequentialStageSupervisor interface {
	StartSingle(context.Context, SingleDispatch, executiongraph.DispatchAuthority) (executiongraph.ProcessCorrelation, error)
}

func StartSingleStage(ctx context.Context, s SingleDispatch, a executiongraph.DispatchAuthority, supervisor SequentialStageSupervisor) (executiongraph.ProcessCorrelation, error) {
	if supervisor == nil || s.State.FormatVersion != ProfileConfiguredFormatVersion || s.AttemptID == "" || s.ResolvedProfile == nil || s.State.Binding == nil || s.State.Binding.ResolvedProfile == nil || *s.ResolvedProfile != *s.State.Binding.ResolvedProfile || a.Scope != "single-first-attempt" || a.SingleAttemptID != s.AttemptID || a.ParentID != s.State.ExecutionID || a.Revision != s.State.Revision || a.SingleStageID != string(s.State.Stage) || executiongraph.ValidateDispatchAuthority(a) != nil {
		return executiongraph.ProcessCorrelation{}, ErrSingleDispatch
	}
	return supervisor.StartSingle(ctx, s, a)
}
