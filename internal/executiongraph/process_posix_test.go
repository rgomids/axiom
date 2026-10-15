//go:build linux || darwin

package executiongraph

import (
	"context"
	"os"
	"os/exec"
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
			defer group.close()
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
	defer group.close()
	if err := command.Wait(); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), processStopBound)
	defer cancel()
	if !confirmProcessStop(ctx, group) {
		t.Fatal("already exited group not confirmed gone")
	}
}

func TestOSProcessRunnerStopsDescendants(t *testing.T) {
	if mode := os.Getenv("AXIOM_RUNNER_GROUP_HELPER"); mode != "" {
		if mode == "descendant" {
			time.Sleep(time.Hour)
			os.Exit(0)
		}
		child := exec.Command(os.Args[0], "-test.run=^TestOSProcessRunnerStopsDescendants$")
		child.Env = append(os.Environ(), "AXIOM_RUNNER_GROUP_HELPER=descendant")
		if err := child.Start(); err != nil {
			os.Exit(2)
		}
		if err := os.WriteFile(os.Getenv("AXIOM_RUNNER_GROUP_PIDFILE"), []byte(strconv.Itoa(child.Process.Pid)), 0600); err != nil {
			os.Exit(3)
		}
		if mode == "completion" {
			deadline := time.Now().Add(10 * time.Second)
			for time.Now().Before(deadline) {
				if _, err := os.Stat(os.Getenv("AXIOM_RUNNER_GROUP_PIDFILE") + ".exit"); err == nil {
					os.Exit(0)
				}
				time.Sleep(10 * time.Millisecond)
			}
			os.Exit(4)
		}
		time.Sleep(time.Hour)
		os.Exit(0)
	}
	for _, mode := range []string{"completion", "timeout", "cancel", "correlation failure"} {
		t.Run(mode, func(t *testing.T) {
			pidfile := t.TempDir() + "/pid"
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if mode == "timeout" {
				var timeoutCancel context.CancelFunc
				ctx, timeoutCancel = context.WithTimeout(ctx, 5*time.Second)
				defer timeoutCancel()
			}
			var descendant *ProcessCorrelation
			started := func(*ProcessCorrelation) error {
				deadline := time.Now().Add(10 * time.Second)
				for time.Now().Before(deadline) {
					data, err := os.ReadFile(pidfile)
					if err == nil {
						pid, parseErr := strconv.ParseInt(string(data), 10, 64)
						if parseErr == nil {
							descendant, err = ObserveProcessCorrelation(pid)
							if err == nil && descendant != nil {
								break
							}
						}
					}
					time.Sleep(10 * time.Millisecond)
				}
				if descendant == nil {
					cancel()
					return ErrInvalidGraph
				}
				switch mode {
				case "completion":
					return os.WriteFile(pidfile+".exit", nil, 0600)
				case "cancel":
					cancel()
				case "correlation failure":
					return ErrInvalidGraph
				}
				return nil
			}
			result := (OSProcessRunner{}).RunObserved(ctx, Invocation{
				Argv: []string{os.Args[0], "-test.run=^TestOSProcessRunnerStopsDescendants$"},
				Env:  append(os.Environ(), "AXIOM_RUNNER_GROUP_HELPER="+mode, "AXIOM_RUNNER_GROUP_PIDFILE="+pidfile),
				CWD:  t.TempDir(), OutputMax: 1024,
			}, started)
			if descendant == nil {
				t.Fatal("descendant did not start")
			}
			t.Cleanup(func() {
				current, err := ObserveProcessCorrelation(descendant.PID)
				if err == nil && current != nil && *current == *descendant {
					_ = syscall.Kill(int(descendant.PID), syscall.SIGKILL)
				}
			})
			if !result.Stopped || result.OutcomeUnknown != (mode == "correlation failure" || mode == "completion") {
				t.Fatalf("mode=%s result=%+v", mode, result)
			}
			if current, err := ObserveProcessCorrelation(descendant.PID); err != nil || current != nil {
				t.Fatalf("descendant remains: %+v, %v", current, err)
			}
			if result.TimedOut != (mode == "timeout") {
				t.Fatalf("mode=%s timedOut=%v", mode, result.TimedOut)
			}
			if mode == "completion" && result.ExitCode != 0 {
				t.Fatalf("completion exit=%d", result.ExitCode)
			}
			if mode == "completion" {
				status, reason, _ := ClassifyAttempt(result, true, AttemptCorrelation{}, nil)
				if status != AttemptUnknown || reason != ProcessOutcomeUnknown {
					t.Fatalf("unobserved descendant effects classified as %s/%s", status, reason)
				}
			}
		})
	}
}

func TestPOSIXUnstartedGroupRefusesStop(t *testing.T) {
	group := &posixProcessGroup{}
	if err := group.stop(); err == nil {
		t.Fatal("unstarted group accepted stop")
	}
	if gone, err := group.gone(); gone || err == nil {
		t.Fatalf("unstarted group gone=%v error=%v", gone, err)
	}
}

func TestOSProcessRunnerNaturalExitPreservesOutcome(t *testing.T) {
	if value := os.Getenv("AXIOM_RUNNER_EXIT_HELPER"); value != "" {
		code, err := strconv.Atoi(value)
		if err != nil {
			os.Exit(2)
		}
		os.Exit(code)
	}
	for _, code := range []int{0, 7} {
		t.Run(strconv.Itoa(code), func(t *testing.T) {
			result := (OSProcessRunner{}).Run(context.Background(), Invocation{
				Argv: []string{os.Args[0], "-test.run=^TestOSProcessRunnerNaturalExitPreservesOutcome$"},
				Env:  append(os.Environ(), "AXIOM_RUNNER_EXIT_HELPER="+strconv.Itoa(code)),
				CWD:  t.TempDir(), OutputMax: 1024,
			})
			if result.ExitCode != code || result.Stopped || result.OutcomeUnknown || result.EffectAmbiguous || result.TimedOut {
				t.Fatalf("exit %d: %+v", code, result)
			}
		})
	}
}
