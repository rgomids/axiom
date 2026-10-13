//go:build windows

package stageexecution

import (
	"os/exec"
	"testing"
)

func TestDetachmentPlatformAttributes(t *testing.T) {
	cmd := &exec.Cmd{}
	if e := configureDetached(cmd); e != nil || cmd.SysProcAttr == nil || cmd.SysProcAttr.CreationFlags&0x208 != 0x208 {
		t.Fatal("supervisor shares initiating console", e)
	}
}
