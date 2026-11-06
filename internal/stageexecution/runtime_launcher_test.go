//go:build linux || darwin

package stageexecution

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rgomids/axiom/internal/executiongraph"
	"github.com/rgomids/axiom/internal/runtimeadapter"
	"github.com/rgomids/axiom/internal/workflow"
)

// boundedHang (the cmd/lingo stub's hang) waits without spinning while the
// shell keeps the parent it started with, for at most ~120s, so a stand-in
// process in its own group ends by itself once the test binary is gone, even
// by SIGKILL with no cleanup.
const boundedHang = `alive() { if [ -r /proc/$$/stat ]; then read -r s < /proc/$$/stat; set -- ${s##*") "}; [ "$2" = "$PPID" ]; else kill -0 "$PPID" 2>/dev/null; fi; }
n=0; while [ "$n" -lt 600 ] && alive; do /bin/sleep 0.2; n=$((n+1)); done; exit 9`

func runtimeLauncherFixture(t *testing.T, runtimeID, effort string) (RuntimeLauncher, LaunchRequest) {
	t.Helper()
	root := realTempDir(t)
	for _, dir := range []string{"repo", "inbox", "home", "bin", "managed"} {
		if err := os.Mkdir(filepath.Join(root, dir), 0700); err != nil {
			t.Fatal(err)
		}
	}
	executable := filepath.Join(root, "bin", runtimeID)
	// Build hashes the stub but never executes it. Even an accidental invocation
	// cannot call a real vendor CLI or use ambient credentials.
	if err := os.WriteFile(executable, []byte("#!/bin/sh\nexit 91\n"), 0700); err != nil {
		t.Fatal(err)
	}
	axiom := filepath.Join(root, "bin", "lingo")
	if err := os.WriteFile(axiom, []byte("stub"), 0700); err != nil {
		t.Fatal(err)
	}
	l := RuntimeLauncher{Workspace: filepath.Join(root, "repo"), InboxRoot: filepath.Join(root, "inbox"), AxiomPath: axiom, Environment: []string{"HOME=" + filepath.Join(root, "home"), "PATH=/usr/bin", "PRIVATE_TOKEN=canary"}}
	l.selectorRoots = func(_ context.Context, inv executiongraph.Invocation, _ bool) (runtimeadapter.EffortSourceRoots, error) {
		return runtimeadapter.EffortSourceRoots{User: filepath.Join(root, "home"), Project: inv.CWD, Managed: filepath.Join(root, "managed")}, nil
	}
	inv := workflow.Invocation{Runtime: runtimeID, Executable: executable, Model: "stub-model", Effort: effort, Instructions: "do scoped work"}
	inv.InstructionsDigest = workflow.DigestText([]byte(inv.Instructions))
	r := LaunchRequest{Execution: workflow.Execution{ExecutionID: engineExecutionID, ProjectID: "proj"}, Ref: workflow.TaskRef{StageID: "work", AgentID: "builder"}, Stage: workflow.StageSpec{Outputs: []string{"result"}}, Attempt: workflow.Attempt{AttemptID: "att-1", RunnerGeneration: 1, Invocation: inv, Authority: workflow.AttemptAuthority{RepositoryKey: "app", Effects: []string{"workspace_write"}}}}
	return l, r
}

func TestRuntimeLauncherBuildsFrozenSelectorsWithoutVendorCalls(t *testing.T) {
	for _, runtimeID := range []string{"codex", "claude"} {
		for _, effort := range []string{"runtime-default", "high"} {
			t.Run(runtimeID+"/"+effort, func(t *testing.T) {
				l, r := runtimeLauncherFixture(t, runtimeID, effort)
				inv, err := l.Build(context.Background(), r)
				if err != nil {
					t.Fatal(err)
				}
				if inv.Argv[0] != r.Attempt.Invocation.Executable || inv.Argv[3] != "stub-model" || !strings.Contains(string(inv.Stdin), r.Attempt.AttemptID) {
					t.Fatalf("wrong frozen tuple: %#v", inv.Argv)
				}
				environment := executiongraph.EffectiveInvocationEnvironment(inv)
				for _, env := range environment {
					if strings.Contains(env, "canary") {
						t.Fatal("credential inherited")
					}
				}
				// The child-context markers deny CLI human authority to the child.
				for _, want := range []string{"AXIOM_ATTEMPT_ID=att-1", "AXIOM_EXECUTION_ID=" + engineExecutionID} {
					if !slices.Contains(environment, want) {
						t.Fatalf("child context %q missing from %v", want, environment)
					}
				}
				if _, err := l.Build(context.Background(), r); err == nil {
					t.Fatal("reused attempt inbox accepted")
				}
			})
		}
	}
}

func TestRuntimeLauncherRefusesConflictingConfigAndRedacts(t *testing.T) {
	for _, runtimeID := range []string{"codex", "claude"} {
		t.Run(runtimeID, func(t *testing.T) {
			l, r := runtimeLauncherFixture(t, runtimeID, "runtime-default")
			config, content := ".codex/config.toml", `model_reasoning_effort = ["private-canary"`
			if runtimeID == "claude" {
				config, content = ".claude/settings.json", `{"effortLevel":"private-canary"`
			}
			path := filepath.Join(l.Workspace, config)
			if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte(content), 0600); err != nil {
				t.Fatal(err)
			}
			_, err := l.Build(context.Background(), r)
			if err == nil || strings.Contains(err.Error(), "private-canary") || strings.Contains(err.Error(), l.Workspace) {
				t.Fatalf("unsafe diagnostic: %v", err)
			}
		})
	}
}

func TestRuntimeLauncherCollectsOnlyDeclaredBoundedRegularFiles(t *testing.T) {
	for _, mode := range []string{"present", "missing", "symlink", "directory", "oversize", "failed"} {
		t.Run(mode, func(t *testing.T) {
			l, r := runtimeLauncherFixture(t, "codex", "runtime-default")
			if _, err := l.Build(context.Background(), r); err != nil {
				t.Fatal(err)
			}
			inbox, _ := l.inbox(r)
			path := filepath.Join(inbox, "result")
			var err error
			switch mode {
			case "present", "failed":
				err = os.WriteFile(path, []byte("result bytes"), 0600)
			case "symlink":
				err = os.Symlink(r.Attempt.Invocation.Executable, path)
			case "directory":
				err = os.Mkdir(path, 0700)
			case "oversize":
				err = os.WriteFile(path, make([]byte, maxRuntimeArtifactBytes+1), 0600)
			}
			if err != nil {
				t.Fatal(err)
			}
			p := executiongraph.ProcessResult{Output: []byte("private-auth-canary")}
			if mode == "failed" {
				p.ExitCode = 1
			}
			out, err := l.Collect(context.Background(), r, p)
			unsafe := mode == "symlink" || mode == "directory" || mode == "oversize"
			if (err != nil) != unsafe {
				t.Fatalf("collect %s: %v", mode, err)
			}
			if mode == "present" && (len(out.Artifacts) != 1 || out.Artifacts[0].Digest != workflow.DigestText([]byte("result bytes"))) {
				t.Fatalf("output: %#v", out)
			}
			if (mode == "missing" || mode == "failed") && len(out.Artifacts) != 0 {
				t.Fatal("fabricated output")
			}
			if strings.Contains(out.Diagnostic, "canary") {
				t.Fatal("raw vendor diagnostic leaked")
			}
		})
	}
}

func TestEngineRuntimeLauncherProgressesTwoStagesWithFileOutputs(t *testing.T) {
	l, _ := runtimeLauncherFixture(t, "codex", "runtime-default")
	f, store := startFixture(t, "")
	frozen := stubInvocation()
	frozen.Runtime = "codex"
	frozen.Executable = filepath.Join(filepath.Dir(l.AxiomPath), "codex")
	l.ReadInput = func(_ context.Context, _ workflow.Execution, _ string, artifact workflow.ArtifactRef) ([]byte, error) {
		if artifact.Name != "result" {
			t.Fatal("unexpected input artifact", artifact)
		}
		return []byte("validated local bytes"), nil
	}
	f.engine.Launcher = l
	f.engine.Resolve = func(_ context.Context, _ workflow.Execution, _ workflow.TaskRef) (workflow.Invocation, workflow.AttemptAuthority, error) {
		return frozen, workflow.AttemptAuthority{RepositoryKey: "app", Effects: []string{"workspace_write"}, Actor: "user:local"}, nil
	}
	var calls atomic.Int32
	f.runner.run = func(_ context.Context, inv executiongraph.Invocation, started func(*executiongraph.ProcessCorrelation) error) executiongraph.ProcessResult {
		// The successor receives its predecessor's content, not only a reference.
		if calls.Add(1) == 2 {
			var successor struct {
				Inputs map[string]struct {
					Content string `json:"content"`
				}
			}
			if err := json.Unmarshal(inv.Stdin, &successor); err != nil || successor.Inputs["source"].Content != "validated local bytes" {
				t.Fatalf("predecessor content missing from packet: %v %s", err, inv.Stdin)
			}
		}
		if err := started(&executiongraph.ProcessCorrelation{PID: 777, StartIdentity: "boot:1", GroupIdentity: "777"}); err != nil {
			t.Fatal(err)
		}
		var packet struct {
			Contract, OutputDirectory, RepositoryKey, RepositoryRoot string
			RepositoryWritable                                       bool
		}
		if err := json.Unmarshal(inv.Stdin, &packet); err != nil {
			t.Fatal(err)
		}
		// A writing child is told its repository root; the key is only a label.
		if packet.RepositoryKey != "app" || packet.RepositoryRoot != l.Workspace || !packet.RepositoryWritable || !strings.Contains(packet.Contract, "do not write the output files") {
			t.Fatalf("repository/blocked protocol missing: %+v", packet)
		}
		// The child must be told how to deliver; stdout never counts.
		if !strings.HasPrefix(string(inv.Stdin), `{"Contract":`) || !strings.Contains(packet.Contract, "OutputDirectory") || !strings.Contains(packet.Contract, "stdout") {
			t.Fatal("stage packet lacks the delivery contract")
		}
		if err := os.WriteFile(filepath.Join(packet.OutputDirectory, "result"), []byte("validated local bytes"), 0600); err != nil {
			t.Fatal(err)
		}
		return executiongraph.ProcessResult{}
	}
	req := linkedRequest()
	req.WorkflowID, req.WorkflowRevision, req.Stages = "two-stage", "r1", twoStages(false)
	req.Stages[1].Agents[0].Effects = []string{"workspace_write"}
	req.Stages[1].Agents[0].Inputs = []string{"source"}
	req.Stages[1].StageOutputInputs = []workflow.StageOutputInput{{Name: "source", StageID: "work", Output: "result", Required: true}}
	x, err := f.engine.Start(context.Background(), store, req)
	if err != nil || x.Status != workflow.RunCompleted || f.runner.spawns.Load() != 2 {
		t.Fatalf("two-stage launcher: %v %s %d", err, x.Status, f.runner.spawns.Load())
	}
}

func TestRuntimeLauncherRefusesUnsafeMachineLocalInputs(t *testing.T) {
	for _, mode := range []string{"missing-executable", "symlink-workspace", "public-inbox", "changed-instructions", "parent-attempt"} {
		t.Run(mode, func(t *testing.T) {
			l, r := runtimeLauncherFixture(t, "codex", "runtime-default")
			switch mode {
			case "missing-executable":
				r.Attempt.Invocation.Executable = filepath.Join(filepath.Dir(l.AxiomPath), "missing")
			case "symlink-workspace":
				target := l.Workspace + "-real"
				if err := os.Rename(l.Workspace, target); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(target, l.Workspace); err != nil {
					t.Fatal(err)
				}
			case "public-inbox":
				if err := os.Chmod(l.InboxRoot, 0755); err != nil {
					t.Fatal(err)
				}
			case "changed-instructions":
				r.Attempt.Invocation.Instructions = "tampered"
			case "parent-attempt":
				r.Attempt.AttemptID = "../escape"
			}
			if _, err := l.Build(context.Background(), r); err == nil || strings.Contains(err.Error(), l.Workspace) {
				t.Fatalf("unsafe launch accepted or leaked path: %v", err)
			}
		})
	}
}

func TestRuntimeLauncherConfiguredValidationCannotBeSkipped(t *testing.T) {
	for _, mode := range []string{"missing-adapter", "refused", "validated"} {
		t.Run(mode, func(t *testing.T) {
			l, r := runtimeLauncherFixture(t, "codex", "runtime-default")
			r.Stage.DefinitionDigest = "sha256:" + strings.Repeat("a", 64)
			r.Stage.OutputChecks = []workflow.OutputCheck{{Name: "result", Kind: "evidence", Evidence: true}}
			if _, err := l.Build(context.Background(), r); err != nil {
				t.Fatal(err)
			}
			inbox, _ := l.inbox(r)
			if err := os.WriteFile(filepath.Join(inbox, "result"), []byte(`{"kind":"artifact"}`), 0600); err != nil {
				t.Fatal(err)
			}
			calls := 0
			if mode != "missing-adapter" {
				l.ValidateOutput = func(_ context.Context, s workflow.OutputSubject, data []byte) (workflow.ArtifactRef, error) {
					calls++
					if s.AttemptID != r.Attempt.AttemptID || s.ExecutionID != r.Execution.ExecutionID || s.RepositoryKey != "app" || s.StageID != "work" || s.DefinitionDigest != r.Stage.DefinitionDigest || !s.Check.Evidence || string(data) != `{"kind":"artifact"}` {
						t.Fatal("uncorrelated subject")
					}
					if mode == "refused" {
						return workflow.ArtifactRef{}, ErrRuntimeOutput
					}
					return workflow.ArtifactRef{Name: "result", RetainedID: "retained", RepositoryKey: "app", Digest: workflow.DigestText([]byte("actual artifact")), Size: 15}, nil
				}
			}
			out, err := l.Collect(context.Background(), r, executiongraph.ProcessResult{ExitCode: 0})
			if mode == "validated" {
				if err != nil || calls != 1 || len(out.Artifacts) != 1 || out.Artifacts[0].RetainedID != "retained" {
					t.Fatalf("%+v %v", out, err)
				}
			} else if err == nil || len(out.Artifacts) != 0 {
				t.Fatal("validator bypassed")
			}
		})
	}
}

func TestRuntimeLauncherResolvesOnlyCompletedUnambiguousStageInputs(t *testing.T) {
	for _, mode := range []string{"valid", "pending", "unknown", "missing", "wrong-repository", "duplicate", "unconfigured", "missing-producer-attempt"} {
		t.Run(mode, func(t *testing.T) {
			l, r := runtimeLauncherFixture(t, "codex", "runtime-default")
			r.Agent.Inputs = []string{"source"}
			r.Stage.StageOutputInputs = []workflow.StageOutputInput{{Name: "source", StageID: "prior", Output: "result", Required: true}}
			artifact := workflow.ArtifactRef{Name: "result", RepositoryKey: "app", Digest: workflow.DigestText([]byte("retained result")), Size: 15}
			attempt := workflow.Attempt{AttemptID: "producer-attempt", Status: workflow.AttemptSucceeded, Artifacts: []workflow.ArtifactRef{artifact}}
			r.Execution.Stages = []workflow.StageRun{{StageID: "prior", Status: workflow.StageCompleted, Tasks: []workflow.TaskRun{{AgentID: "producer", Attempts: []workflow.Attempt{attempt}}}}}
			switch mode {
			case "pending":
				r.Execution.Stages[0].Status = workflow.StagePending
			case "unknown":
				r.Execution.Stages[0].Tasks[0].Attempts[0].Status = workflow.AttemptUnknown
			case "missing":
				r.Execution.Stages[0].Tasks[0].Attempts[0].Artifacts = nil
			case "wrong-repository":
				r.Execution.Stages[0].Tasks[0].Attempts[0].Artifacts[0].RepositoryKey = "other"
			case "duplicate":
				r.Execution.Stages[0].Tasks = append(r.Execution.Stages[0].Tasks, r.Execution.Stages[0].Tasks[0])
			case "unconfigured":
				r.Stage.StageOutputInputs = nil
			case "missing-producer-attempt":
				r.Execution.Stages[0].Tasks[0].Attempts[0].AttemptID = ""
			}
			inv, err := l.Build(context.Background(), r)
			if mode != "valid" {
				if err == nil {
					t.Fatal("invalid input supplied to child")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			var packet struct {
				Inputs map[string]struct {
					Artifact   workflow.ArtifactRef     `json:"artifact"`
					Provenance workflow.InputProvenance `json:"provenance"`
				}
			}
			if err := json.Unmarshal(inv.Stdin, &packet); err != nil {
				t.Fatal(err)
			}
			if packet.Inputs["source"].Artifact.Digest != artifact.Digest || packet.Inputs["source"].Provenance != (workflow.InputProvenance{ExecutionID: r.Execution.ExecutionID, StageID: "prior", AttemptID: "producer-attempt"}) {
				t.Fatal("lost input correlation")
			}
		})
	}
}

func TestRuntimeLauncherAuthorizedCommandArgvAndDenial(t *testing.T) {
	for _, authorized := range []bool{false, true} {
		t.Run(map[bool]string{false: "denied", true: "authorized"}[authorized], func(t *testing.T) {
			l, r := runtimeLauncherFixture(t, "codex", "runtime-default")
			marker := filepath.Join(l.Workspace, "checked")
			command := filepath.Join(filepath.Dir(r.Attempt.Invocation.Executable), "local-check")
			if err := os.WriteFile(command, []byte("#!/bin/sh\nprintf '%s' \"$1\" > checked\n"), 0700); err != nil {
				t.Fatal(err)
			}
			r.Stage.Commands = []workflow.CommandCheck{{ID: "check", Argv: []string{command, "literal;$argument"}, TimeoutSeconds: 5, Effects: []string{"validation_command", "workspace_write"}}}
			var record workflow.CommandResult
			r.PrepareCommand = func(id string) error { record = workflow.CommandResult{ID: id, Status: "prepared"}; return nil }
			r.StartCommand = func(id string, p workflow.ProcessIdentity) error {
				record.Process = &p
				record.Status = "running"
				return nil
			}
			r.FinishCommand = func(id string, code int) (workflow.CommandResult, error) {
				record.ExitCode = code
				record.Status = "completed"
				return record, nil
			}
			if authorized {
				r.Attempt.Authority.Effects = append(r.Attempt.Authority.Effects, "validation_command")
			}
			if _, err := l.Build(context.Background(), r); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(l.InboxRoot, r.Attempt.AttemptID, "result"), []byte("result"), 0600); err != nil {
				t.Fatal(err)
			}
			outcome, err := l.Collect(context.Background(), r, executiongraph.ProcessResult{ExitCode: 0})
			if !authorized {
				if err == nil {
					t.Fatal("unauthorized command accepted")
				}
				if _, err := os.Stat(marker); !os.IsNotExist(err) {
					t.Fatal("unauthorized command ran")
				}
				return
			}
			if err != nil || len(outcome.Commands) != 1 || outcome.Commands[0].ExitCode != 0 {
				t.Fatalf("authorized command: %+v %v", outcome, err)
			}
			if wire, err := os.ReadFile(marker); err != nil || string(wire) != "literal;$argument" {
				t.Fatalf("configured argv changed: %q %v", wire, err)
			}
		})
	}
}

func TestEngineCommandProcessFactsCancellationAndFailure(t *testing.T) {
	for _, mode := range []string{"success", "failure", "cancel"} {
		t.Run(mode, func(t *testing.T) {
			l, _ := runtimeLauncherFixture(t, "codex", "runtime-default")
			f, store := startFixture(t, "")
			command := filepath.Join(filepath.Dir(l.AxiomPath), "check")
			script := "#!/bin/sh\nprintf checked > checked\nexit 0\n"
			if mode == "failure" {
				script = "#!/bin/sh\nexit 7\n"
			}
			if mode == "cancel" {
				script = "#!/bin/sh\n" + boundedHang + "\n"
			}
			if err := os.WriteFile(command, []byte(script), 0700); err != nil {
				t.Fatal(err)
			}
			frozen := stubInvocation()
			frozen.Runtime = "codex"
			frozen.Executable = filepath.Join(filepath.Dir(l.AxiomPath), "codex")
			f.engine.Launcher = l
			f.engine.Resolve = func(context.Context, workflow.Execution, workflow.TaskRef) (workflow.Invocation, workflow.AttemptAuthority, error) {
				return frozen, workflow.AttemptAuthority{RepositoryKey: "app", Effects: []string{"workspace_write", "validation_command"}, Actor: "user:local"}, nil
			}
			f.runner.run = func(_ context.Context, inv executiongraph.Invocation, started func(*executiongraph.ProcessCorrelation) error) executiongraph.ProcessResult {
				if err := started(&executiongraph.ProcessCorrelation{PID: 777, StartIdentity: "boot:1", GroupIdentity: "777"}); err != nil {
					t.Fatal(err)
				}
				var packet struct{ OutputDirectory string }
				if err := json.Unmarshal(inv.Stdin, &packet); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(packet.OutputDirectory, "result"), []byte("validated bytes"), 0600); err != nil {
					t.Fatal(err)
				}
				return executiongraph.ProcessResult{}
			}
			req := linkedRequest()
			req.WorkflowID, req.WorkflowRevision, req.Stages = "command-stub", "r1", twoStages(false)
			for i := range req.Stages {
				req.Stages[i].Agents[0].Effects = []string{"workspace_write", "validation_command"}
			}
			req.Stages[0].Commands = []workflow.CommandCheck{{ID: "check", Argv: []string{command}, TimeoutSeconds: 10, Effects: []string{"validation_command"}}}
			cancelResult := make(chan error, 1)
			if mode == "cancel" {
				go func() {
					deadline := time.Now().Add(5 * time.Second)
					for time.Now().Before(deadline) {
						x, err := f.engine.Store.Load(context.Background(), req.ProjectID, req.ExecutionID)
						if err == nil && len(x.Stages[0].Tasks[0].Attempts) > 0 {
							a := x.Stages[0].Tasks[0].Attempts[0]
							if len(a.Commands) > 0 && a.Commands[0].Process != nil {
								_, err = f.engine.Cancel(context.Background(), req.ProjectID, req.ExecutionID, "user:local")
								cancelResult <- err
								return
							}
						}
						time.Sleep(5 * time.Millisecond)
					}
					cancelResult <- errors.New("command identity never persisted")
				}()
			}
			x, err := f.engine.Start(context.Background(), store, req)
			if err != nil {
				t.Fatal(err)
			}
			if mode == "cancel" {
				if err := <-cancelResult; err != nil {
					t.Fatal(err)
				}
				if x.Status != workflow.RunCancelled && x.Status != workflow.RunRecoveryRequired {
					t.Fatalf("cancel: %s", x.Status)
				}
			} else if mode == "failure" {
				if x.Status != workflow.RunFailed || f.runner.spawns.Load() != 1 {
					t.Fatalf("failure advanced: %s", x.Status)
				}
			} else if x.Status != workflow.RunCompleted {
				a := x.Stages[0].Tasks[0].Attempts[0]
				t.Fatalf("success: %s diag=%q effects=%+v commands=%+v", x.Status, a.Diagnostic, a.Effects, a.Commands)
			}
			a := x.Stages[0].Tasks[0].Attempts[0]
			if len(a.Commands) != 1 || a.Commands[0].Process == nil || a.Commands[0].Process.PID == a.Process.PID || a.Commands[0].StartedAt == nil {
				t.Fatalf("command identity lost: %+v", a.Commands)
			}
			if mode != "cancel" && (a.Commands[0].Status != "completed" || a.Commands[0].EndedAt == nil) {
				t.Fatal("terminal command evidence missing")
			}
		})
	}
}

// A publication refusal must never release the validation command, including
// the EOF produced when the runner cancels the blocked stdin copy.
func TestCommandGateRefusesPublicationFailureWithoutEffects(t *testing.T) {
	for i := 0; i < 10; i++ {
		l, r := runtimeLauncherFixture(t, "codex", "runtime-default")
		command := filepath.Join(filepath.Dir(l.AxiomPath), "check")
		marker := filepath.Join(l.Workspace, "unauthorized-effect")
		if err := os.WriteFile(command, []byte("#!/bin/sh\nprintf effect > unauthorized-effect\n"), 0700); err != nil {
			t.Fatal(err)
		}
		r.Stage.Commands = []workflow.CommandCheck{{ID: "check", Argv: []string{command}, TimeoutSeconds: 5, Effects: []string{"validation_command", "workspace_write"}}}
		r.Attempt.Authority.Effects = append(r.Attempt.Authority.Effects, "validation_command")
		r.PrepareCommand = func(string) error { return nil }
		r.StartCommand = func(string, workflow.ProcessIdentity) error { return errors.New("synthetic publication refusal") }
		r.FinishCommand = func(string, int) (workflow.CommandResult, error) {
			t.Fatal("refused command finished")
			return workflow.CommandResult{}, nil
		}
		if _, err := l.Build(context.Background(), r); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(l.InboxRoot, r.Attempt.AttemptID, "result"), []byte("result"), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := l.Collect(context.Background(), r, executiongraph.ProcessResult{}); err == nil {
			t.Fatal("publication refusal lost")
		}
		if _, err := os.Stat(marker); !os.IsNotExist(err) {
			t.Fatalf("refused command ran: %v", err)
		}
	}
}

func TestCommandGateEOFDoesNotExecuteConfiguredArgv(t *testing.T) {
	root := realTempDir(t)
	marker := filepath.Join(root, "effect")
	inv := executiongraph.Invocation{Argv: []string{commandGateShell, "-c", commandGateScript, "axiom-command-gate", "/bin/sh", "-c", "printf effect > effect"}, CWD: root, OutputMax: 1024}
	result := (executiongraph.OSProcessRunner{}).RunObserved(context.Background(), inv, func(*executiongraph.ProcessCorrelation) error { return nil })
	if result.ExitCode == 0 {
		t.Fatal("EOF released command")
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("EOF executed command: %v", err)
	}
}

// IR-04 (CR-S1-07): workspace code run as a validation command carries the
// same Runtime-child markers as the agent, so a nested Axiom CLI call cannot
// record human authority, and still receives no secret-bearing variable.
func TestRuntimeLauncherCommandCarriesAttemptCorrelation(t *testing.T) {
	l, r := runtimeLauncherFixture(t, "claude", "runtime-default")
	command := filepath.Join(filepath.Dir(r.Attempt.Invocation.Executable), "local-check")
	script := "#!/bin/sh\nprintf '%s|%s|%s|%s' \"$AXIOM_ATTEMPT_ID\" \"$AXIOM_EXECUTION_ID\" \"${PRIVATE_TOKEN-unset}\" \"${CLAUDE_CONFIG_DIR-unset}\" > markers\n"
	if err := os.WriteFile(command, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	l.Environment = append(l.Environment, "AXIOM_ATTEMPT_ID=inherited-forged", "CLAUDE_CONFIG_DIR=/private/claude")
	r.Stage.Commands = []workflow.CommandCheck{{ID: "check", Argv: []string{command}, TimeoutSeconds: 5, Effects: []string{"validation_command"}}}
	r.Attempt.Authority.Effects = append(r.Attempt.Authority.Effects, "validation_command")
	var record workflow.CommandResult
	r.PrepareCommand = func(id string) error { record = workflow.CommandResult{ID: id, Status: "prepared"}; return nil }
	r.StartCommand = func(id string, p workflow.ProcessIdentity) error {
		record.Process, record.Status = &p, "running"
		return nil
	}
	r.FinishCommand = func(id string, code int) (workflow.CommandResult, error) {
		record.ExitCode, record.Status = code, "completed"
		return record, nil
	}
	if err := os.MkdirAll(filepath.Join(l.InboxRoot, r.Attempt.AttemptID), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(l.InboxRoot, r.Attempt.AttemptID, "result"), []byte("result"), 0600); err != nil {
		t.Fatal(err)
	}
	if outcome, err := l.Collect(context.Background(), r, executiongraph.ProcessResult{ExitCode: 0}); err != nil || len(outcome.Commands) != 1 {
		t.Fatalf("command: %+v %v", outcome, err)
	}
	wire, err := os.ReadFile(filepath.Join(l.Workspace, "markers"))
	if err != nil {
		t.Fatal(err)
	}
	if want := r.Attempt.AttemptID + "|" + r.Execution.ExecutionID + "|unset|unset"; string(wire) != want {
		t.Fatalf("command environment %q, want %q", wire, want)
	}
}

// A writable attempt needs a shell to build, test and wait for messages; a
// read-only Claude attempt keeps only the Axiom CLI.
func TestRuntimeLauncherShellFollowsWorkspaceWrite(t *testing.T) {
	for _, write := range []bool{true, false} {
		l, r := runtimeLauncherFixture(t, "claude", "runtime-default")
		if !write {
			r.Attempt.Authority.Effects = nil
		}
		inv, err := l.Build(context.Background(), r)
		if err != nil {
			t.Fatal(err)
		}
		i := slices.Index(inv.Argv, "--allowedTools")
		if got := strings.HasSuffix(inv.Argv[i+1], ",Bash"); got != write {
			t.Fatalf("write=%v tools %q", write, inv.Argv[i+1])
		}
	}
}
