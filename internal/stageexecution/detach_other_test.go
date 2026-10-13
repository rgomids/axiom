//go:build !linux && !darwin && !windows

package stageexecution

import (
	"errors"
	"os/exec"
	"testing"
)

func TestDetachmentPlatformAttributes(t *testing.T) {
	if e := configureDetached(&exec.Cmd{}); !errors.Is(e, ErrDetachmentUnsupported) {
		t.Fatal(e)
	}
}
