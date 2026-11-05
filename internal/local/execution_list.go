package local

import (
	"context"
	"errors"
	"os"
	"strings"

	"github.com/rgomids/axiom/internal/workflow"
)

// ExecutionReferenceNamespace reports whether entry is one of the exact
// canonical Execution namespaces (ADR-0037) the v1 state inventory deliberately
// reports as unsupported_newer (executions/v2) or unknown
// (executions/v2-attempt-outputs), and that namespace is still a private,
// non-symlinked directory below root. Only Workflow-reference observation may
// set these entries aside: List strictly scans executions/v2/<project> itself
// and attempt outputs carry no Workflow reference. compatibility.Inspect also
// reports them as InventoryCanonicalExecution so an owned upgrade between
// releases that write them is not refused; cutover keeps refusing them.
func ExecutionReferenceNamespace(root string, entry InventoryEntry) bool {
	var name string
	switch {
	case entry.Relative == "executions/v2" && entry.Kind == InventoryNewer:
		name = "v2"
	case entry.Relative == "executions/v2-attempt-outputs" && entry.Kind == InventoryUnknown:
		name = "v2-attempt-outputs"
	default:
		return false
	}
	state, err := existingPrivateRoot(root)
	if err != nil {
		return false
	}
	defer state.Close()
	executions, err := existingPrivateChild(state, "executions")
	if err != nil {
		return false
	}
	defer executions.Close()
	namespace, err := existingPrivateChild(executions, name)
	if err != nil {
		return false
	}
	namespace.Close()
	return true
}

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
