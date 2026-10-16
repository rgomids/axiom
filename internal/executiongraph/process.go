package executiongraph

import (
	"context"
	"os/exec"
	"time"
)

const processStopBound = 2 * time.Second

// processContainment keeps spawn, stop and observation in the same platform boundary.
type processContainment interface {
	start(*exec.Cmd) error
	stop() error
	gone() (bool, error)
}

// confirmProcessStop never turns a signal delivery or an observation error into
// evidence of termination. The observer must cover the complete containment.
func confirmProcessStop(ctx context.Context, group processContainment) bool {
	if err := group.stop(); err != nil {
		return false
	}
	for {
		gone, err := group.gone()
		if err != nil {
			return false
		}
		if gone {
			return true
		}
		select {
		case <-ctx.Done():
			return false
		case <-time.After(10 * time.Millisecond):
		}
	}
}
