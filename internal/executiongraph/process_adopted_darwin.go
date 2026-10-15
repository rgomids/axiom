package executiongraph

import (
	"fmt"
	"os"

	"golang.org/x/sys/unix"
)

// Darwin adopts escaped orphans into launchd instead of this supervisor. A new
// live launchd child cannot be attributed safely, so stop remains unconfirmed.
// Start identity prevents a reused PID from matching the pre-spawn baseline.
func observeAdoptedProcesses(baseline bool) (map[int]string, error) {
	processes, err := unix.SysctlKinfoProcSlice("kern.proc.all")
	if err != nil {
		return nil, err
	}
	result := map[int]string{}
	startedGroups.Lock()
	defer startedGroups.Unlock()
	for _, p := range processes {
		if p.Proc.P_stat == 5 || !baseline && (p.Eproc.Ppid != 1 || startedGroups.leaders[int(p.Eproc.Pgid)]) {
			continue
		}
		if os.Geteuid() != 0 && int(p.Eproc.Pcred.P_ruid) != os.Getuid() && int(p.Eproc.Pcred.P_svuid) != os.Getuid() && int(p.Eproc.Ucred.Uid) != os.Geteuid() {
			continue
		}
		result[int(p.Proc.P_pid)] = fmt.Sprintf("%d:%d", p.Proc.P_starttime.Sec, p.Proc.P_starttime.Usec)
	}
	return result, nil
}
