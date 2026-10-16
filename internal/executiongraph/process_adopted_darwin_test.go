package executiongraph

import (
	"testing"

	"golang.org/x/sys/unix"
)

func TestDarwinAdoptionDoesNotAssumeDescendantCredentials(t *testing.T) {
	processes := make([]unix.KinfoProc, 3)
	for i := range processes {
		processes[i].Proc.P_pid = int32(999990 + i)
		processes[i].Proc.P_stat = 1
		processes[i].Proc.P_starttime = unix.Timeval{Sec: 1, Usec: int32(i)}
		processes[i].Eproc.Ppid = 1
		processes[i].Eproc.Pgid = int32(999990 + i)
	}
	processes[0].Eproc.Pcred.P_ruid = 501
	processes[0].Eproc.Pcred.P_svuid = 501
	processes[0].Eproc.Ucred.Uid = 501
	processes[1].Eproc.Pcred.P_ruid = 0
	processes[1].Eproc.Pcred.P_svuid = 0
	processes[1].Eproc.Ucred.Uid = 0
	processes[2].Eproc.Ppid = 999989
	baseline := adoptedSnapshot(processes, true, 0)
	if len(baseline) != 3 {
		t.Fatal("baseline excluded a live process identity", baseline)
	}
	adopted := adoptedSnapshot(processes, false, 0)
	if len(adopted) != 2 || adopted[999990] == "" || adopted[999991] == "" {
		t.Fatal("an un-attributable orphan was discarded by credentials", adopted)
	}
	processes[1].Proc.P_starttime.Usec++
	if adoptedSnapshot(processes, false, 0)[999991] == adopted[999991] {
		t.Fatal("PID reuse lost start identity")
	}
}
