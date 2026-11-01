//go:build linux || darwin

package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/rgomids/axiom/internal/cli"
	"github.com/rgomids/axiom/internal/executiongraph"
	"github.com/rgomids/axiom/internal/stageexecution"
	"github.com/rgomids/axiom/internal/workflow"
)

// IR-05 (CR-S1-07): a symlinked command executable (python3, alternatives,
// merged /usr) freezes to its symlink-free target before any Execution exists,
// and that frozen path starts through the launcher's unchanged containment
// check: no command that could never start becomes an unknown effect.
func TestFreezeCommandsResolvesSymlinkedExecutableBeforeExecution(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, dir := range []string{"real", "links", "repo", "inbox"} {
		if err := os.Mkdir(filepath.Join(root, dir), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	target := filepath.Join(root, "real", "check-1.0")
	if err := os.WriteFile(target, []byte("#!/bin/sh\nprintf ran > ran\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(root, "links", "check")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(root, "real", "missing"), filepath.Join(root, "links", "dangling")); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", filepath.Join(root, "links"))
	stages := func(argv0 string) []workflow.StageSpec {
		return []workflow.StageSpec{{ID: "work", Commands: []workflow.CommandCheck{{ID: "check", Argv: []string{argv0, "arg"}, TimeoutSeconds: 5, Effects: []string{"validation_command"}}}}}
	}
	if err := freezeCommands(stages("dangling"), true); !errors.Is(err, errWorkflowUnmappable) {
		t.Fatalf("dangling executable link: %v", err)
	}
	frozen := stages("check")
	if err := freezeCommands(frozen, true); err != nil {
		t.Fatal(err)
	}
	argv := frozen[0].Commands[0].Argv
	if argv[0] != target || argv[1] != "arg" || !stageexecution.CleanRuntimePath(argv[0]) {
		t.Fatalf("frozen argv %q", argv)
	}
	// The frozen command passes the unchanged Collect containment and runs.
	launcher := stageexecution.RuntimeLauncher{Workspace: filepath.Join(root, "repo"), InboxRoot: filepath.Join(root, "inbox"), Environment: []string{"PATH=/usr/bin"}}
	attempt := workflow.Attempt{AttemptID: "att-1", Invocation: workflow.Invocation{Runtime: "codex"}, Authority: workflow.AttemptAuthority{RepositoryKey: "app", Effects: []string{"validation_command"}}}
	if err := os.Mkdir(filepath.Join(root, "inbox", attempt.AttemptID), 0o700); err != nil {
		t.Fatal(err)
	}
	var record workflow.CommandResult
	request := stageexecution.LaunchRequest{
		Execution: workflow.Execution{ExecutionID: "00000000-0000-4000-8000-0000000000f5", ProjectID: "proj"}, Ref: workflow.TaskRef{StageID: "work", AgentID: "agent"}, Stage: frozen[0], Attempt: attempt,
		PrepareCommand: func(id string) error { record = workflow.CommandResult{ID: id, Status: "prepared"}; return nil },
		StartCommand: func(id string, p workflow.ProcessIdentity) error {
			record.Process, record.Status = &p, "running"
			return nil
		},
		FinishCommand: func(id string, code int) (workflow.CommandResult, error) {
			record.ExitCode, record.Status = code, "completed"
			return record, nil
		},
	}
	outcome, err := launcher.Collect(context.Background(), request, executiongraph.ProcessResult{ExitCode: 0})
	if err != nil || len(outcome.Commands) != 1 || outcome.Commands[0].ExitCode != 0 {
		t.Fatalf("frozen symlink target did not start: %+v %v", outcome, err)
	}
	if _, err := os.Stat(filepath.Join(root, "repo", "ran")); err != nil {
		t.Fatal(err)
	}
}

// IR-04 (CR-S1-07): either Runtime-child marker, as now also carried by
// validation commands, refuses human authority before any read or write.
func TestRuntimeChildMarkersRefuseHumanAuthority(t *testing.T) {
	f := newExecutionRunFixture(t)
	for _, marker := range []string{"AXIOM_ATTEMPT_ID", "AXIOM_EXECUTION_ID"} {
		t.Setenv(marker, "00000000-0000-4000-8000-0000000000aa")
		got := f.run(t, cli.ExitFailure, "workflow", "stage", "reconcile", "--project", "external", "--execution", "00000000-0000-4000-8000-0000000000aa", "--abandon", "attempt", "--authorize-human")
		if got.Category != "human_authority_required" || len(f.executions(t)) != 0 {
			t.Fatalf("%s: %s", marker, got.Raw)
		}
		t.Setenv(marker, "")
	}
}
