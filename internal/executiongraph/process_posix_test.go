//go:build linux || darwin

package executiongraph

import (
	"context"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"syscall"
	"testing"
	"time"
)

func TestPOSIXProcessContainment(t *testing.T) {
	if os.Getenv("AXIOM_GROUP_HELPER") == "descendant" {
		time.Sleep(time.Hour)
		return
	}
	if os.Getenv("AXIOM_GROUP_HELPER") == "parent" {
		child := exec.Command(os.Args[0], "-test.run=^TestPOSIXProcessContainment$")
		child.Env = append(os.Environ(), "AXIOM_GROUP_HELPER=descendant")
		if os.Getenv("AXIOM_GROUP_ESCAPE") == "1" {
			child.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
		}
		if err := child.Start(); err != nil {
			os.Exit(2)
		}
		if err := os.WriteFile(os.Getenv("AXIOM_GROUP_PIDFILE"), []byte(strconv.Itoa(child.Process.Pid)), 0600); err != nil {
			os.Exit(3)
		}
		if os.Getenv("AXIOM_GROUP_ORPHAN") == "1" {
			// Double fork: the intermediate exits, orphaning the escaped child so
			// no ancestry from the group leader remains at stop time.
			os.Exit(0)
		}
		time.Sleep(time.Hour)
		return
	}
	for _, mode := range []string{"descendants", "escaped descendant", "orphaned escaped descendant"} {
		escaped := mode != "descendants"
		orphan := mode == "orphaned escaped descendant"
		t.Run(mode, func(t *testing.T) {
			if orphan && runtime.GOOS != "linux" {
				t.Skip("orphaned double-fork escapes are observable only through the Linux child subreaper; other POSIX platforms record this as a containment limitation")
			}
			pidfile := t.TempDir() + "/pid"
			command := exec.Command(os.Args[0], "-test.run=^TestPOSIXProcessContainment$")
			command.Env = append(os.Environ(), "AXIOM_GROUP_HELPER=parent", "AXIOM_GROUP_PIDFILE="+pidfile)
			if escaped {
				command.Env = append(command.Env, "AXIOM_GROUP_ESCAPE=1")
			}
			if orphan {
				command.Env = append(command.Env, "AXIOM_GROUP_ORPHAN=1")
			}
			group := &posixProcessGroup{}
			if err := group.start(command); err != nil {
				t.Fatal(err)
			}
			defer func() { _ = group.stop(); _ = command.Wait() }()
			deadline := time.Now().Add(3 * time.Second)
			var pid int
			for time.Now().Before(deadline) {
				data, err := os.ReadFile(pidfile)
				if err == nil {
					pid, _ = strconv.Atoi(string(data))
					break
				}
				time.Sleep(10 * time.Millisecond)
			}
			if pid == 0 {
				t.Fatal("descendant did not start")
			}
			if orphan {
				// Let the intermediate exit so only the reparented escapee remains.
				_ = command.Wait()
			}
			defer syscall.Kill(pid, syscall.SIGKILL)
			ctx, cancel := context.WithTimeout(context.Background(), processStopBound)
			defer cancel()
			if got := confirmProcessStop(ctx, group); got == escaped {
				t.Fatalf("Stopped=%v escaped=%v", got, escaped)
			}
			if escaped && syscall.Kill(pid, 0) != nil {
				t.Fatal("escaped descendant should remain alive until explicit cleanup")
			}
		})
	}
}

func TestPOSIXAlreadyExitedGroup(t *testing.T) {
	command := exec.Command(os.Args[0], "-test.run=^$")
	group := &posixProcessGroup{}
	if err := group.start(command); err != nil {
		t.Fatal(err)
	}
	if err := command.Wait(); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), processStopBound)
	defer cancel()
	if !confirmProcessStop(ctx, group) {
		t.Fatal("already exited group not confirmed gone")
	}
}
