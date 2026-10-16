//go:build linux

package executiongraph

import (
	"errors"
	"os"
	"strconv"
	"strings"
	"syscall"
)

// procEntryVanished reports a /proc entry whose process was reaped between
// enumeration and read; the kernel answers ENOENT or ESRCH for it.
func procEntryVanished(err error) bool {
	return errors.Is(err, os.ErrNotExist) || errors.Is(err, syscall.ESRCH)
}

func observeProcesses() ([]processObservation, error) {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil, err
	}
	var result []processObservation
	for _, entry := range entries {
		pid, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue
		}
		data, err := os.ReadFile("/proc/" + entry.Name() + "/stat")
		if procEntryVanished(err) {
			continue
		}
		if err != nil {
			return nil, err
		}
		end := strings.LastIndex(string(data), ")")
		if end < 0 {
			return nil, errors.New("invalid process stat")
		}
		fields := strings.Fields(string(data[end+1:]))
		if len(fields) < 3 {
			return nil, errors.New("short process stat")
		}
		if fields[0] == "Z" {
			continue
		}
		parent, err := strconv.Atoi(fields[1])
		if err != nil {
			return nil, err
		}
		group, err := strconv.Atoi(fields[2])
		if err != nil {
			return nil, err
		}
		result = append(result, processObservation{pid, parent, group})
	}
	return result, nil
}
