package local

import (
	"context"
	"errors"
	"os"
	"strings"

	"github.com/rgomids/axiom/internal/workflow"
)

// List inspects only the canonical namespace. A damaged, unexpected or pending
// entry fails the entire observation; no old record is converted or resumed.
func (s ExecutionStore) List(ctx context.Context, projectID string) ([]workflow.Execution, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !workflow.ValidToken(projectID) || !platformEntryName(projectID) {
		return nil, ErrUnsafe
	}
	roots, err := s.open(projectID, false)
	if errors.Is(err, os.ErrNotExist) || errors.Is(err, ErrNotFound) {
		return []workflow.Execution{}, nil
	}
	if err != nil {
		return nil, err
	}
	defer roots.Close()
	locks, err := lockRoots(false, roots.chain...)
	if err != nil {
		return nil, err
	}
	defer closeFiles(locks)
	if pending, err := protocolStatePresent(roots.project()); err != nil {
		return nil, err
	} else if pending {
		return nil, ErrRecoveryRequired
	}
	names, err := childNames(roots.project())
	if err != nil {
		return nil, err
	}
	out := make([]workflow.Execution, 0, len(names))
	for _, name := range names {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if name == ".lock" {
			continue
		}
		id, ok := strings.CutSuffix(name, ".json")
		if !ok || !validRunAddress(projectID, id) {
			return nil, ErrUnsafe
		}
		x, _, err := s.read(roots.project(), projectID, id)
		if err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, nil
}
