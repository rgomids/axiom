//go:build linux || darwin

package stageexecution

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/executiongraph"
	"github.com/rgomids/axiom/internal/runtimeadapter"
	"github.com/rgomids/axiom/internal/workflow"
)

func runtimeLauncherFixture(t *testing.T, runtimeID, effort string) (RuntimeLauncher, LaunchRequest) {
	t.Helper()
	root := t.TempDir()
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
	r := LaunchRequest{Execution: workflow.Execution{ExecutionID: engineExecutionID}, Ref: workflow.TaskRef{StageID: "work", AgentID: "builder"}, Stage: workflow.StageSpec{Outputs: []string{"result"}}, Attempt: workflow.Attempt{AttemptID: "att-1", RunnerGeneration: 1, Invocation: inv, Authority: workflow.AttemptAuthority{RepositoryKey: "app", Effects: []string{"workspace_write"}}}}
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
				for _, env := range executiongraph.EffectiveInvocationEnvironment(inv) {
					if strings.Contains(env, "canary") {
						t.Fatal("credential inherited")
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
			config, content := ".codex/config.toml", `model_reasoning_effort = "private-canary"`
			if runtimeID == "claude" {
				config, content = ".claude/settings.json", `{"effortLevel":"private-canary"}`
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
	f.engine.Launcher = l
	f.engine.Resolve = func(_ context.Context, _ workflow.Execution, _ workflow.TaskRef) (workflow.Invocation, workflow.AttemptAuthority, error) {
		return frozen, workflow.AttemptAuthority{RepositoryKey: "app", Effects: []string{"workspace_write"}, Actor: "user:local"}, nil
	}
	f.runner.run = func(_ context.Context, inv executiongraph.Invocation, started func(*executiongraph.ProcessCorrelation) error) executiongraph.ProcessResult {
		if err := started(&executiongraph.ProcessCorrelation{PID: 777, StartIdentity: "boot:1", GroupIdentity: "777"}); err != nil {
			t.Fatal(err)
		}
		var packet struct{ OutputDirectory string }
		if err := json.Unmarshal(inv.Stdin, &packet); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(packet.OutputDirectory, "result"), []byte("validated local bytes"), 0600); err != nil {
			t.Fatal(err)
		}
		return executiongraph.ProcessResult{}
	}
	req := linkedRequest()
	req.WorkflowID, req.WorkflowRevision, req.Stages = "two-stage", "r1", twoStages(false)
	req.Stages[1].Agents[0].Effects = []string{"workspace_write"}
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
