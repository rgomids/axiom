//go:build linux

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/rgomids/axiom/internal/cli"
	"github.com/rgomids/axiom/internal/executiongraph"
	"github.com/rgomids/axiom/internal/local"
	"github.com/rgomids/axiom/internal/workflow"
)

// IR-01 (CR-S1-07): a human abandonment is refused while a recorded process of
// the Execution is alive; the refusal names the bounded next action and writes
// nothing. After the orphan group is gone the same abandonment succeeds.
func TestPublicReconcileAbandonRefusesLiveOrphanWithoutWriting(t *testing.T) {
	f := newExecutionRunFixture(t)
	failed := f.run(t, cli.ExitFailure, minimalRun("--select", "agent.instructions=mode:fail", "--authorize-external")...)
	id := failed.Stage.ExecutionID
	orphan := exec.Command("/bin/sh", "-c", "while :; do read -t 1 x; done")
	orphan.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := orphan.Start(); err != nil {
		t.Fatal(err)
	}
	reaped := false
	kill := func() {
		if !reaped {
			reaped = true
			_ = syscall.Kill(-orphan.Process.Pid, syscall.SIGKILL)
			_ = orphan.Wait()
		}
	}
	t.Cleanup(kill)
	identity, err := executiongraph.ObserveProcessCorrelation(int64(orphan.Process.Pid))
	if err != nil || identity == nil {
		t.Fatalf("observe orphan: %v", err)
	}
	store, err := local.NewExecutionStore(f.env.state)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	// A runner started the orphan, published its identity and was killed.
	lease, err := store.AcquireFence(ctx, workItemSourceProjectID, id)
	if err != nil {
		t.Fatal(err)
	}
	gen := lease.Generation()
	if _, err := store.UpdateFenced(ctx, workItemSourceProjectID, id, lease, func(x *workflow.Execution) error {
		if err := x.ClaimRunner(gen, time.Now()); err != nil {
			return err
		}
		last := x.Stages[0].Tasks[0].Attempts[0]
		if err := x.PrepareAttempt(workflow.PrepareInput{Ref: workflow.TaskRef{StageID: "work", AgentID: "agent"}, AttemptID: "orphan-attempt", Generation: gen, Invocation: last.Invocation, Authority: last.Authority, Now: time.Now(), Retry: true, WorkspaceDisposition: "preserve"}); err != nil {
			return err
		}
		return x.MarkStarted("orphan-attempt", gen, workflow.ProcessIdentity{PID: identity.PID, StartIdentity: identity.StartIdentity, GroupIdentity: identity.GroupIdentity}, time.Now())
	}); err != nil {
		t.Fatal(err)
	}
	lease.Close()
	reconciled := f.run(t, cli.ExitSuccess, "workflow", "stage", "reconcile", "--project", "external", "--execution", id, "--authorize-local")
	if !strings.Contains(reconciled.Raw, "process_alive_effects_unknown") || !strings.Contains(reconciled.Stage.NextAction, "stop any process it reports alive") {
		t.Fatalf("reconcile: %s", reconciled.Raw)
	}
	record := f.executions(t)
	if len(record) != 1 {
		t.Fatalf("records: %v", record)
	}
	before, err := os.ReadFile(record[0])
	if err != nil {
		t.Fatal(err)
	}
	abandon := []string{"workflow", "stage", "reconcile", "--project", "external", "--execution", id, "--abandon", "orphan-attempt", "--authorize-human"}
	refused := f.run(t, cli.ExitFailure, abandon...)
	if refused.Category != "stop_orphan_then_abandon" || !strings.Contains(refused.Next, "Stop the process") || !strings.Contains(refused.Raw, "process_alive_effects_unknown") {
		t.Fatalf("abandon beside a live orphan: %s", refused.Raw)
	}
	if after, err := os.ReadFile(record[0]); err != nil || !bytes.Equal(before, after) {
		t.Fatalf("refused abandon wrote the Execution: %v", err)
	}
	if err := syscall.Kill(orphan.Process.Pid, 0); err != nil {
		t.Fatalf("abandon signalled the orphan: %v", err)
	}
	retry := []string{"workflow", "stage", "retry", "--project", "external", "--execution", id, "--stage", "work", "--agent", "agent", "--disposition", "preserve", "--authorize-external"}
	if got := f.run(t, cli.ExitFailure, retry...); got.Category != "recovery_required" || len(got.Stage.Stages[0].Tasks[0].Attempts) != 2 {
		t.Fatalf("retry beside a live orphan: %s", got.Raw)
	}
	kill()
	abandoned := f.run(t, cli.ExitSuccess, abandon...)
	if a := latestAttempt(t, abandoned.Stage, "work"); a.Resolution == nil || abandoned.Category != "attempt_abandoned" {
		t.Fatalf("abandon after stop: %s", abandoned.Raw)
	}
	again := f.run(t, cli.ExitFailure, retry...)
	if a := latestAttempt(t, again.Stage, "work"); a.Number != 3 {
		t.Fatalf("one retry after abandon: %+v", a)
	}
}

// IR-02 (CR-S1-07): black-box. An operator signal to the foreground runner
// stops the agent's own process group, which the terminal signal never
// reaches, and the runner commits the observed stop before exiting. The agent
// is a local /bin/sh stub; no vendor CLI runs, so vendor SIGPIPE behavior is
// not exercised here.
func TestPublicWorkflowRunSignalStopsOwnedGroupAndCommitsStop(t *testing.T) {
	binary := filepath.Join(t.TempDir(), "lingo")
	if output, err := exec.Command("go", "build", "-o", binary, ".").CombinedOutput(); err != nil {
		t.Fatalf("build executable: %v: %s", err, output)
	}
	for _, signal := range []syscall.Signal{syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP} {
		t.Run(signal.String(), func(t *testing.T) {
			f := newExecutionRunFixture(t)
			var stdout bytes.Buffer
			runner := exec.Command(binary, append([]string{"--json"}, minimalRun("--select", "agent.instructions=mode:hang", "--authorize-external")...)...)
			runner.Env = os.Environ()
			runner.Stdout = &stdout
			if err := runner.Start(); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = runner.Process.Kill(); _, _ = runner.Process.Wait() })
			id, pid := "", int64(0)
			for deadline := time.Now().Add(20 * time.Second); pid == 0; time.Sleep(20 * time.Millisecond) {
				if time.Now().After(deadline) {
					t.Fatal("no running attempt observed")
				}
				files := f.executions(t)
				if len(files) != 1 {
					continue
				}
				id = strings.TrimSuffix(filepath.Base(files[0]), ".json")
				status := f.run(t, cli.ExitSuccess, "workflow", "stage", "status", "--project", "external", "--execution", id)
				if a := status.Stage.Stages[0].Tasks[0].Attempts; len(a) == 1 && a[0].Status == workflow.AttemptRunning && a[0].Process != nil {
					pid = a[0].Process.PID
				}
			}
			// Even when the assertion below fails, never leak the stub group:
			// stop it only if the leader identity is still the recorded one.
			leader, err := executiongraph.ObserveProcessCorrelation(pid)
			if err != nil || leader == nil {
				t.Fatalf("observe stub agent: %v", err)
			}
			t.Cleanup(func() {
				if current, err := executiongraph.ObserveProcessCorrelation(pid); err == nil && current != nil && *current == *leader {
					_ = syscall.Kill(-int(pid), syscall.SIGKILL)
				}
			})
			if err := runner.Process.Signal(signal); err != nil {
				t.Fatal(err)
			}
			exited := make(chan error, 1)
			go func() { exited <- runner.Wait() }()
			select {
			case err := <-exited:
				var exit *exec.ExitError
				if !errors.As(err, &exit) || exit.ExitCode() != cli.ExitCancelled {
					t.Fatalf("runner exit: %v %s", err, stdout.String())
				}
			case <-time.After(15 * time.Second):
				t.Fatal("runner did not exit after the signal")
			}
			var event executionRunEvent
			if err := json.Unmarshal(bytes.TrimSpace(stdout.Bytes()), &event); err != nil || event.Category != "workflow_cancelled" {
				t.Fatalf("runner result: %v %s", err, stdout.String())
			}
			if err := syscall.Kill(-int(pid), 0); !errors.Is(err, syscall.ESRCH) {
				t.Fatalf("agent process group survived the runner: %v", err)
			}
			status := f.run(t, cli.ExitSuccess, "workflow", "stage", "status", "--project", "external", "--execution", id)
			a := latestAttempt(t, status.Stage, "work")
			if a.Status != workflow.AttemptCancelled || a.Diagnostic != "group_stop_confirmed" || status.Stage.Status != workflow.RunCancelled || status.Stage.Cancel == nil || status.Stage.Cancel.Actor != "runner:interrupted" {
				t.Fatalf("committed stop: %s", status.Raw)
			}
		})
	}
}
