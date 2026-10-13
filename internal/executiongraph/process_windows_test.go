//go:build windows

package executiongraph

import (
	"context"
	"os"
	"os/exec"
	"testing"
	"time"
)

func TestWindowsJobStopsDescendants(t *testing.T) {
	if os.Getenv("AXIOM_JOB_HELPER") == "descendant" {
		time.Sleep(time.Hour)
		return
	}
	if os.Getenv("AXIOM_JOB_HELPER") == "parent" {
		child := exec.Command(os.Args[0], "-test.run=^TestWindowsJobStopsDescendants$")
		child.Env = append(os.Environ(), "AXIOM_JOB_HELPER=descendant")
		if err := child.Start(); err != nil {
			os.Exit(2)
		}
		if err := os.WriteFile(os.Getenv("AXIOM_JOB_READY"), []byte("ready"), 0600); err != nil {
			os.Exit(3)
		}
		time.Sleep(time.Hour)
		return
	}
	ready := t.TempDir() + "/ready"
	command := exec.Command(os.Args[0], "-test.run=^TestWindowsJobStopsDescendants$")
	command.Env = append(os.Environ(), "AXIOM_JOB_HELPER=parent", "AXIOM_JOB_READY="+ready)
	group, err := newProcessContainment()
	if err != nil {
		t.Fatal(err)
	}
	defer group.close()
	if err := group.start(command); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = group.stop(); _ = command.Wait() }()
	deadline := time.Now().Add(3 * time.Second)
	for {
		if _, err := os.Stat(ready); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("descendant did not start")
		}
		time.Sleep(10 * time.Millisecond)
	}
	ctx, cancel := context.WithTimeout(context.Background(), processStopBound)
	defer cancel()
	if !confirmProcessStop(ctx, group) {
		t.Fatal("job stop not confirmed")
	}
	if !confirmProcessStop(ctx, group) {
		t.Fatal("already empty job not confirmed")
	}
}
