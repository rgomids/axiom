//go:build !linux && !darwin

package executiongraph

import "errors"

// Unsupported identity observation remains unknown; it never proves stop.
func ObserveProcessCorrelation(pid int64) (*ProcessCorrelation, error) {
	return nil, errors.New("process start/group identity observation unsupported on this platform")
}
func ObserveRecordedProcess(recorded ProcessCorrelation) (*ProcessCorrelation, error) {
	return ObserveProcessCorrelation(recorded.PID)
}
