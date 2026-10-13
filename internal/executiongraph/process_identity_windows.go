//go:build windows

package executiongraph

import (
	"errors"
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Windows process creation FILETIME is a kernel identity, not an inferred age.
// The job leader identity can be persisted without retaining a process handle.
func ObserveProcessCorrelation(pid int64) (*ProcessCorrelation, error) {
	if pid <= 0 || pid > int64(^uint32(0)) {
		return nil, ErrInvalidGraph
	}
	handle, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if errors.Is(err, windows.ERROR_INVALID_PARAMETER) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer windows.CloseHandle(handle)
	var creation, exit, kernel, user windows.Filetime
	if err := windows.GetProcessTimes(handle, &creation, &exit, &kernel, &user); err != nil {
		return nil, err
	}
	if exit.HighDateTime != 0 || exit.LowDateTime != 0 {
		return nil, nil
	}
	// A leader reference represents a T43 job only while kernel job membership
	// is observable. An uncontained PID never establishes group correlation.
	var inJob uint32
	checkJob := windows.NewLazySystemDLL("kernel32.dll").NewProc("IsProcessInJob")
	ok, _, callErr := checkJob.Call(uintptr(handle), 0, uintptr(unsafe.Pointer(&inJob)))
	if ok == 0 {
		return nil, fmt.Errorf("observe process job membership: %w", callErr)
	}
	if inJob == 0 {
		return nil, errors.New("process has no observable job membership")
	}
	identity := fmt.Sprintf("%08x%08x", creation.HighDateTime, creation.LowDateTime)
	return &ProcessCorrelation{PID: pid, StartIdentity: identity, GroupIdentity: fmt.Sprintf("job-leader:%d:%s", pid, identity)}, nil
}

func ObserveRecordedProcess(recorded ProcessCorrelation) (*ProcessCorrelation, error) {
	current, err := ObserveProcessCorrelation(recorded.PID)
	if err != nil {
		return nil, err
	}
	if current != nil && *current == recorded {
		return current, nil
	}
	// The T43 job is unnamed. After its leader disappears a PID lookup cannot
	// prove every job member stopped. Only the held job observer can prove that;
	// restart observation remains unknown rather than authorizing unsafe retry.
	return current, errors.New("complete recorded job liveness unavailable without job handle")
}
