package stageexecution

import (
	"context"
	"errors"
	"github.com/rgomids/axiom/internal/executiongraph"
	"os"
	"os/exec"
	"path/filepath"
)

var ErrDetachmentUnsupported = errors.New("supervisor_detachment_unsupported")

type DetachedStart interface {
	Start(context.Context, []string) (executiongraph.ProcessCorrelation, error)
}

// Launch checks committed authority before even asking the detacher to spawn.
// T26 supplies this port after its protected publication; no authority flags
// are accepted by the hidden process.
func Launch(ctx context.Context, r SupervisorRequest, store SupervisorStore, start DetachedStart) (executiongraph.ProcessCorrelation, error) {
	if e := ctx.Err(); e != nil {
		return executiongraph.ProcessCorrelation{}, e
	}
	if store == nil || start == nil {
		return executiongraph.ProcessCorrelation{}, ErrSupervisorAuthority
	}
	g, a, e := store.Load(ctx, r)
	if e != nil {
		return executiongraph.ProcessCorrelation{}, e
	}
	if e = verifySupervisor(r, g, a); e != nil {
		return executiongraph.ProcessCorrelation{}, e
	}
	if e = admitSupervisor(ctx, store, r, g); e != nil {
		return executiongraph.ProcessCorrelation{}, e
	}
	return start.Start(ctx, []string{"__stage-supervisor", r.ProjectID, r.ParentID, r.PreviewDigest, r.AuthorityRef})
}

// SameBinaryDetacher never accepts an executable path from graph/CLI input.
type SameBinaryDetacher struct{}

func (SameBinaryDetacher) Start(ctx context.Context, args []string) (executiongraph.ProcessCorrelation, error) {
	if e := ctx.Err(); e != nil {
		return executiongraph.ProcessCorrelation{}, e
	}
	binary, e := os.Executable()
	if e != nil || !filepath.IsAbs(binary) {
		return executiongraph.ProcessCorrelation{}, ErrDetachmentUnsupported
	}
	cmd := exec.Command(binary, args...)
	// Disconnect all inherited pipes from the initiating tool call.
	null, e := os.OpenFile(os.DevNull, os.O_RDWR, 0)
	if e != nil {
		return executiongraph.ProcessCorrelation{}, e
	}
	defer null.Close()
	cmd.Stdin = null
	cmd.Stdout = null
	cmd.Stderr = null
	if e = configureDetached(cmd); e != nil {
		return executiongraph.ProcessCorrelation{}, e
	}
	if e = cmd.Start(); e != nil {
		return executiongraph.ProcessCorrelation{}, e
	}
	identity, e := observeDetached(int64(cmd.Process.Pid))
	if e != nil || identity == nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return executiongraph.ProcessCorrelation{}, ErrDetachmentUnsupported
	}
	if e = cmd.Process.Release(); e != nil {
		return executiongraph.ProcessCorrelation{}, e
	}
	return *identity, nil
}
