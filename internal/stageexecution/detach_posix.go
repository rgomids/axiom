//go:build linux || darwin

package stageexecution

import (
	"github.com/rgomids/axiom/internal/executiongraph"
	"os/exec"
	"syscall"
)

func configureDetached(cmd *exec.Cmd) error {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	return nil
}

func observeDetached(pid int64) (*executiongraph.ProcessCorrelation, error) {
	return executiongraph.ObserveProcessCorrelation(pid)
}
