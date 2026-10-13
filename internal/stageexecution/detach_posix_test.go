//go:build linux || darwin

package stageexecution

import (
	"os/exec"
	"testing"
)

func TestDetachmentPlatformAttributes(t *testing.T) {
	// Inspect an inert command; never start an executable.
	cmd := &exec.Cmd{}
	if e := configureDetached(cmd); e != nil || cmd.SysProcAttr == nil || !cmd.SysProcAttr.Setsid {
		t.Fatal("supervisor shares initiating session", e)
	}
}
