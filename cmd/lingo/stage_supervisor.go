package main

import (
	"context"
	"github.com/rgomids/axiom/internal/stageexecution"
)

type stageSupervisorLoader interface {
	LoadSupervisor(context.Context, stageexecution.SupervisorRequest) (stageexecution.Supervisor, error)
}

// The stage-run composition owns the loader. A shell invocation cannot supply
// a runner, paths, configuration or authority. Absent composition fails closed.
func stageSupervisorCommand(ctx context.Context, args []string, loader stageSupervisorLoader) (bool, int) {
	if len(args) == 0 || args[0] != "__stage-supervisor" {
		return false, 0
	}
	if len(args) != 5 || loader == nil {
		return true, 1
	}
	r := stageexecution.SupervisorRequest{ProjectID: args[1], ParentID: args[2], PreviewDigest: args[3], AuthorityRef: args[4]}
	s, e := loader.LoadSupervisor(ctx, r)
	if e != nil {
		return true, 1
	}
	if e = s.Run(ctx, r); e != nil {
		return true, 1
	}
	return true, 0
}
