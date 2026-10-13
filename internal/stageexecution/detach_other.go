//go:build !linux && !darwin && !windows

package stageexecution

import (
	"github.com/rgomids/axiom/internal/executiongraph"
	"os/exec"
)

func configureDetached(*exec.Cmd) error { return ErrDetachmentUnsupported }

func observeDetached(int64) (*executiongraph.ProcessCorrelation, error) {
	return nil, ErrDetachmentUnsupported
}
