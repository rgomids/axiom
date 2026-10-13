//go:build linux

package executiongraph

import (
	"errors"
	"os"
	"strconv"
	"strings"
)

// ObserveProcessCorrelation reads kernel start ticks, never wall-clock time.
// Group identities include the boot identity to fence a host restart.
func ObserveProcessCorrelation(pid int64) (*ProcessCorrelation, error) {
	if pid <= 0 {
		return nil, ErrInvalidGraph
	}
	wire, err := os.ReadFile("/proc/" + strconv.FormatInt(pid, 10) + "/stat")
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	boot, err := os.ReadFile("/proc/sys/kernel/random/boot_id")
	if err != nil {
		return nil, err
	}
	return parseProcessCorrelation(pid, string(wire), strings.TrimSpace(string(boot)))
}

func parseProcessCorrelation(pid int64, stat, boot string) (*ProcessCorrelation, error) {
	end := strings.LastIndex(stat, ")")
	if end < 0 {
		return nil, ErrInvalidGraph
	}
	fields := strings.Fields(stat[end+1:])
	if len(fields) < 20 || !boundedIdentity(boot) {
		return nil, ErrInvalidGraph
	}
	if fields[0] == "Z" {
		return nil, nil
	}
	group, err := strconv.ParseInt(fields[2], 10, 64)
	if err != nil || group <= 0 {
		return nil, ErrInvalidGraph
	}
	ticks, err := strconv.ParseUint(fields[19], 10, 64)
	if err != nil {
		return nil, ErrInvalidGraph
	}
	return &ProcessCorrelation{PID: pid, StartIdentity: boot + ":" + strconv.FormatUint(ticks, 10), GroupIdentity: boot + ":" + strconv.FormatInt(group, 10)}, nil
}

// ObserveRecordedProcess also checks the complete recorded process group. If
// the leader is gone/reused while group members remain, identity is uncertain;
// it cannot be treated as confirmed stop or safe reconciliation.
func ObserveRecordedProcess(recorded ProcessCorrelation) (*ProcessCorrelation, error) {
	current, err := ObserveProcessCorrelation(recorded.PID)
	if err != nil {
		return nil, err
	}
	if current != nil && *current == recorded {
		return current, nil
	}
	parts := strings.Split(recorded.GroupIdentity, ":")
	if len(parts) != 2 {
		return nil, ErrInvalidGraph
	}
	boot, err := os.ReadFile("/proc/sys/kernel/random/boot_id")
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(string(boot)) != parts[0] {
		return current, nil
	}
	group, err := strconv.Atoi(parts[1])
	if err != nil {
		return nil, err
	}
	processes, err := observeProcesses()
	if err != nil {
		return nil, err
	}
	for _, p := range processes {
		if p.group == group {
			return nil, errors.New("recorded group alive without matching leader identity")
		}
	}
	return current, nil
}
