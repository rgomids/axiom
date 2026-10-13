//go:build windows

package executiongraph

import (
	"fmt"
	"os/exec"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

type windowsProcessJob struct{ job windows.Handle }

func newProcessContainment() (processContainment, error) {
	job, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return nil, err
	}
	limits := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{}
	limits.BasicLimitInformation.LimitFlags = windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE
	_, err = windows.SetInformationJobObject(job, windows.JobObjectExtendedLimitInformation, uintptr(unsafe.Pointer(&limits)), uint32(unsafe.Sizeof(limits)))
	if err != nil {
		windows.CloseHandle(job)
		return nil, err
	}
	return &windowsProcessJob{job: job}, nil
}
func (g *windowsProcessJob) start(cmd *exec.Cmd) error {
	// os/exec closes its primary thread handle. Start suspended and recover that
	// thread through Toolhelp; no child code runs before job assignment.
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: windows.CREATE_SUSPENDED}
	if err := cmd.Start(); err != nil {
		return err
	}
	fail := func(err error) error { _ = cmd.Process.Kill(); _ = cmd.Wait(); return err }
	process, err := windows.OpenProcess(windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE, false, uint32(cmd.Process.Pid))
	if err != nil {
		return fail(err)
	}
	defer windows.CloseHandle(process)
	if err = windows.AssignProcessToJobObject(g.job, process); err != nil {
		return fail(err)
	}
	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPTHREAD, 0)
	if err != nil {
		return fail(err)
	}
	defer windows.CloseHandle(snapshot)
	entry := windows.ThreadEntry32{Size: uint32(unsafe.Sizeof(windows.ThreadEntry32{}))}
	for err = windows.Thread32First(snapshot, &entry); err == nil; err = windows.Thread32Next(snapshot, &entry) {
		if entry.OwnerProcessID != uint32(cmd.Process.Pid) {
			continue
		}
		thread, openErr := windows.OpenThread(windows.THREAD_SUSPEND_RESUME, false, entry.ThreadID)
		if openErr != nil {
			return fail(openErr)
		}
		_, resumeErr := windows.ResumeThread(thread)
		windows.CloseHandle(thread)
		if resumeErr != nil {
			return fail(resumeErr)
		}
		return nil
	}
	return fail(fmt.Errorf("locate suspended process thread: %w", err))
}
func (g *windowsProcessJob) stop() error { return windows.TerminateJobObject(g.job, 1) }
func (g *windowsProcessJob) gone() (bool, error) {
	// JOBOBJECT_BASIC_ACCOUNTING_INFORMATION has four LARGE_INTEGER values
	// followed by four DWORD counters; ActiveProcesses is the third counter.
	accounting := struct {
		Times                                                                          [4]int64
		TotalPageFaultCount, TotalProcesses, ActiveProcesses, TotalTerminatedProcesses uint32
	}{}
	err := windows.QueryInformationJobObject(g.job, windows.JobObjectBasicAccountingInformation, uintptr(unsafe.Pointer(&accounting)), uint32(unsafe.Sizeof(accounting)), nil)
	return err == nil && accounting.ActiveProcesses == 0, err
}
func (g *windowsProcessJob) close() { _ = windows.CloseHandle(g.job) }
