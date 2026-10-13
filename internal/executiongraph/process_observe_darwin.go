//go:build darwin

package executiongraph

import "golang.org/x/sys/unix"

func observeProcesses() ([]processObservation, error) {
	processes, err := unix.SysctlKinfoProcSlice("kern.proc.all")
	if err != nil {
		return nil, err
	}
	result := make([]processObservation, 0, len(processes))
	for _, p := range processes {
		if p.Proc.P_stat == 5 {
			continue
		}
		result = append(result, processObservation{int(p.Proc.P_pid), int(p.Eproc.Ppid), int(p.Eproc.Pgid)})
	}
	return result, nil
}
