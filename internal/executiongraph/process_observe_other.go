//go:build !linux && !darwin

package executiongraph

import "errors"

func observeProcesses() ([]processObservation, error) {
	return nil, errors.New("process observation unsupported on this platform")
}
