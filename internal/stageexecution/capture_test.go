//go:build linux || darwin

package stageexecution

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rgomids/axiom/internal/local"
	"github.com/rgomids/axiom/internal/workflow"
)

// All capture faults use real repositories, objects, index and ExecutionStore.
// The fixture stops after a validated enrolled attempt, before principal capture.
func captureFixture(t *testing.T, checks ...workflow.CommandCheck) (*integrationFixture, workflow.RunnerFence, workflow.Execution, IntegrationRequest) {
	t.Helper()
	f := newIntegrationFixture(t, nil)
	if len(checks) > 0 {
		f.checks = checks
	}
	effects := []string{"workspace_write", "validation_command"}
	x, err := workflow.NewExecution(workflow.NewExecutionInput{ExecutionID: engineExecutionID, ProjectID: "proj", WorkItem: workflow.WorkItemLink{Provider: "linear", ID: "AXM-12", RepositoryKey: "app"}, Repositories: []workflow.RepositoryBinding{{Key: "app", Identity: integrationIdentity}}, WorkflowID: "capture", Revision: "r1", Now: time.Now(), EnrollIntegration: true, Stages: []workflow.StageSpec{{ID: "work", TimeoutSeconds: 30, Agents: []workflow.AgentSpec{{ID: "a", Effects: effects}, {ID: "b", DependsOn: []string{"a"}, Effects: effects}}, Commands: f.checks}}})
	if err != nil {
		t.Fatal(err)
	}
	if err = f.store.Create(context.Background(), x); err != nil {
		t.Fatal(err)
	}
	lease, x, err := f.engine.Claim(context.Background(), "proj", engineExecutionID)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { lease.Close() })
	obs, err := f.git.PrepareAgentWorktree(context.Background(), "agent-a", "refs/heads/axiom/agent-a", f.base)
	if err != nil {
		t.Fatal(err)
	}
	gen := lease.Generation()
	now := time.Now()
	transitions := []func(*workflow.Execution) error{
		func(x *workflow.Execution) error {
			return x.PrepareAttempt(workflow.PrepareInput{Ref: workflow.TaskRef{StageID: "work", AgentID: "a"}, AttemptID: "att-a", Generation: gen, Invocation: stubInvocation(), Authority: workflow.AttemptAuthority{RepositoryKey: "app", Effects: effects, Actor: "user:local"}, Workspace: &workflow.AttemptWorkspace{RepositoryKey: "app", RepositoryIdentity: integrationIdentity, BaseSHA: f.base, InitialTargetHead: f.base, SourceBranch: obs.Branch, TargetRef: integrationTarget, Worktree: "agent-a"}, Now: now})
		},
		func(x *workflow.Execution) error {
			return x.MarkStarted("att-a", gen, workflow.ProcessIdentity{PID: 777, StartIdentity: "fixture", GroupIdentity: "777"}, now)
		},
		func(x *workflow.Execution) error { return x.PrepareCommand("att-a", gen, f.checks[0].ID, now) },
		func(x *workflow.Execution) error {
			return x.StartCommand("att-a", gen, f.checks[0].ID, workflow.ProcessIdentity{PID: 778, StartIdentity: "check", GroupIdentity: "778"}, now)
		},
		func(x *workflow.Execution) error { return x.FinishCommand("att-a", gen, f.checks[0].ID, 0, now) },
		func(x *workflow.Execution) error {
			a, _, _ := x.FindAttempt("att-a")
			return x.CompleteAttempt("att-a", gen, workflow.AttemptResult{Commands: a.Commands, Now: now})
		},
	}
	for _, transition := range transitions {
		x, err = f.store.UpdateFenced(context.Background(), "proj", engineExecutionID, lease, transition)
		if err != nil {
			t.Fatal(err)
		}
	}
	write(t, filepath.Join(obs.Path, "staged.txt"), "staged\n")
	gitT(t, obs.Path, "add", "staged.txt")
	write(t, filepath.Join(obs.Path, "a.txt"), "unstaged\n")
	write(t, filepath.Join(obs.Path, "untracked.txt"), "untracked\n")
	return f, lease, x, f.request(x, "a", "int-a", f.base)
}

type captureFaultGit struct {
	IntegrationGit
	point     string
	stop      func()
	mutations int
}

func (g *captureFaultGit) CapturePending(ctx context.Context, name, branch, base, message string, intent func(string) error, candidate func(string) error, before func() error) error {
	err := g.IntegrationGit.CapturePending(ctx, name, branch, base, message, func(path string) error {
		if g.point == "intent" {
			return errors.New("intent refused")
		}
		return intent(path)
	}, func(sha string) error {
		if g.point == "object" {
			g.stop()
			return errors.New("crash after object before candidate record")
		}
		if err := candidate(sha); err != nil {
			return err
		}
		if g.point == "candidate" {
			g.stop()
			return errors.New("crash after candidate record")
		}
		return nil
	}, func() error {
		g.mutations++
		if g.point == "before-staging" && g.mutations == 1 || g.point == "staging" && g.mutations == 2 || g.point == "tree" && g.mutations == 3 {
			g.stop()
		}
		return before()
	})
	if err == nil && g.point == "source-cas" {
		g.stop()
		return errors.New("lost source CAS response")
	}
	return err
}
func (g *captureFaultGit) ObserveTarget(ctx context.Context, ref string) (string, error) {
	if g.point == "target" {
		return "", errors.New("target unavailable")
	}
	return g.IntegrationGit.ObserveTarget(ctx, ref)
}

func TestCaptureCrashFenceRecoveryMatrix(t *testing.T) {
	for _, kind := range []string{"cancel", "lost-fence"} {
		for _, point := range []string{"before-staging", "staging", "tree", "object", "candidate", "source-cas"} {
			t.Run(kind+"/"+point, func(t *testing.T) {
				f, lease, x, req := captureFixture(t)
				path := filepath.Join(f.root, "agent-a")
				beforeFiles := map[string]string{}
				for _, n := range []string{"staged.txt", "a.txt", "untracked.txt"} {
					b, _ := os.ReadFile(filepath.Join(path, n))
					beforeFiles[n] = string(b)
				}
				stop := func() {
					if kind == "lost-fence" {
						lease.Close()
					} else {
						_, err := f.store.UpdateFenced(context.Background(), "proj", engineExecutionID, nil, func(x *workflow.Execution) error { return x.RequestCancel("user:stop", time.Now()) })
						if err != nil {
							t.Fatal(err)
						}
					}
				}
				g := &captureFaultGit{IntegrationGit: f.git, point: point, stop: stop}
				if _, err := f.engine.integrateHeld(context.Background(), lease, x, g, req); err == nil {
					t.Fatal("injected interruption succeeded")
				}
				beforeHead := gitT(t, f.repo, "rev-parse", req.SourceBranch)
				beforeIndex := gitT(t, path, "ls-files", "--stage")
				beforeObjects := gitT(t, f.repo, "count-objects", "-v")
				if point != "source-cas" && beforeHead != f.base {
					t.Fatal("source advanced before persisted candidate CAS")
				}
				if f.target() != f.base {
					t.Fatal("target moved")
				}
				// Reopen the canonical store, acquire a new generation, observe only.
				lease.Close()
				reopened, err := local.NewExecutionStore(filepath.Join(filepath.Dir(f.repo), "state"))
				if err != nil {
					t.Fatal(err)
				}
				engine := f.engine
				engine.Store = reopened
				engine.Workspaces = func(context.Context, workflow.Execution, string) (WorkspaceGit, error) { return f.git, nil }
				next, current, err := engine.Claim(context.Background(), "proj", engineExecutionID)
				if err != nil {
					t.Fatal(err)
				}
				defer next.Close()
				a, _, _ := current.FindAttempt(req.SourceAttemptID)
				if a.Capture == nil || a.Capture.Observation == nil || current.Status != workflow.RunRecoveryRequired {
					t.Fatalf("missing durable recovery: %+v %s", a.Capture, current.Status)
				}
				want := "refused"
				if point == "source-cas" {
					want = "captured"
				}
				if a.Capture.Observation.Outcome != want || a.Capture.Path != path || a.Capture.RunnerGeneration != lease.Generation() {
					t.Fatalf("wrong correlation/outcome: %+v", a.Capture)
				}
				if point == "candidate" || point == "source-cas" {
					if a.Capture.CandidateSHA == "" {
						t.Fatal("candidate lost")
					}
				}
				if point == "object" && a.Capture.CandidateSHA != "" {
					t.Fatal("unrecorded object adopted")
				}
				if _, err := engine.integrateHeld(context.Background(), next, current, g, req); err == nil {
					t.Fatal("capture replay allowed")
				}
				if gitT(t, f.repo, "rev-parse", req.SourceBranch) != beforeHead || gitT(t, path, "ls-files", "--stage") != beforeIndex || gitT(t, f.repo, "count-objects", "-v") != beforeObjects {
					t.Fatal("recovery/reentry mutated Git")
				}
				if len(current.Ready()) != 0 || len(current.Integrations) != 0 {
					t.Fatal("uncertain source unblocked dependents")
				}
				for n, want := range beforeFiles {
					b, _ := os.ReadFile(filepath.Join(path, n))
					if string(b) != want {
						t.Fatal("file changed", n)
					}
				}
				if b, _ := os.ReadFile(filepath.Join(filepath.Dir(f.repo), "user-worktree", "user-dirty.txt")); string(b) != "user work in progress\n" {
					t.Fatal("operator worktree changed")
				}
			})
		}
	}
}

func TestCaptureAdmissionRefusalHasNoGitEffects(t *testing.T) {
	for _, point := range []string{"intent", "target"} {
		t.Run(point, func(t *testing.T) {
			f, lease, x, req := captureFixture(t)
			path := filepath.Join(f.root, "agent-a")
			refs := gitT(t, f.repo, "show-ref")
			index := gitT(t, path, "ls-files", "--stage")
			objects := gitT(t, f.repo, "count-objects", "-v")
			g := &captureFaultGit{IntegrationGit: f.git, point: point}
			if _, err := f.engine.integrateHeld(context.Background(), lease, x, g, req); err == nil {
				t.Fatal("fault ignored")
			}
			if gitT(t, f.repo, "show-ref") != refs || gitT(t, path, "ls-files", "--stage") != index || gitT(t, f.repo, "count-objects", "-v") != objects {
				t.Fatal("pre-intent refusal mutated Git")
			}
			loaded := f.load(engineExecutionID)
			a, _, _ := loaded.FindAttempt("att-a")
			if a.Capture != nil {
				t.Fatal("uncommitted intent adopted")
			}
		})
	}
}

func TestCaptureHappyPathPersistsExactCandidateBeforeCAS(t *testing.T) {
	f, lease, x, req := captureFixture(t)
	got, err := f.engine.integrateHeld(context.Background(), lease, x, f.git, req)
	if err != nil {
		t.Fatal(err)
	}
	a, _, _ := got.FindAttempt("att-a")
	if a.Capture == nil || a.Capture.Observation.Outcome != "captured" || a.Capture.CandidateSHA != got.Integrations[0].CandidateSHA || got.Integrations[0].Status != workflow.IntegrationIntegrated {
		t.Fatal("capture/integration correlation lost")
	}
}

type captureStoreFault struct {
	local.ExecutionStore
	point string
}

func (s captureStoreFault) UpdateFenced(ctx context.Context, p, id string, f workflow.RunnerFence, mutate func(*workflow.Execution) error) (workflow.Execution, error) {
	return s.ExecutionStore.UpdateFenced(ctx, p, id, f, func(x *workflow.Execution) error {
		a, _, _ := x.FindAttempt("att-a")
		var before *workflow.SourceCapture
		if a != nil && a.Capture != nil {
			c := *a.Capture
			before = &c
		}
		count := len(x.Integrations)
		if err := mutate(x); err != nil {
			return err
		}
		a, _, _ = x.FindAttempt("att-a")
		if a != nil && a.Capture != nil {
			c := a.Capture
			if s.point == "intent" && before == nil || s.point == "candidate" && before != nil && before.CandidateSHA == "" && c.CandidateSHA != "" || s.point == "observation" && c.Observation != nil && (before == nil || before.Observation == nil) || s.point == "prepared" && len(x.Integrations) > count {
				return errors.New("injected store publication refusal: " + s.point)
			}
		}
		return nil
	})
}

func TestCaptureStorePublicationFailureRecovery(t *testing.T) {
	for _, point := range []string{"intent", "candidate", "observation", "prepared"} {
		t.Run(point, func(t *testing.T) {
			f, lease, x, req := captureFixture(t)
			path := filepath.Join(f.root, req.Worktree)
			initialIndex := gitT(t, path, "ls-files", "--stage")
			f.engine.Store = captureStoreFault{ExecutionStore: f.store, point: point}
			if _, err := f.engine.integrateHeld(context.Background(), lease, x, f.git, req); err == nil {
				t.Fatal("store failure ignored")
			}
			beforeHead := gitT(t, f.repo, "rev-parse", req.SourceBranch)
			beforeIndex := gitT(t, path, "ls-files", "--stage")
			beforeObjects := gitT(t, f.repo, "count-objects", "-v")
			loaded := f.load(engineExecutionID)
			a, _, _ := loaded.FindAttempt("att-a")
			if point == "intent" {
				if a.Capture != nil || beforeHead != f.base || beforeIndex != initialIndex {
					t.Fatal("rejected intent mutated Git")
				}
			} else if a.Capture == nil {
				t.Fatal("intent lost")
			}
			if point == "candidate" && (a.Capture.CandidateSHA != "" || beforeHead != f.base) {
				t.Fatal("unpersisted candidate advanced")
			}
			lease.Close()
			fresh, err := local.NewExecutionStore(filepath.Join(filepath.Dir(f.repo), "state"))
			if err != nil {
				t.Fatal(err)
			}
			engine := f.engine
			engine.Store = fresh
			engine.Workspaces = func(context.Context, workflow.Execution, string) (WorkspaceGit, error) { return f.git, nil }
			next, loaded, err := engine.Claim(context.Background(), "proj", engineExecutionID)
			if err != nil {
				t.Fatal(err)
			}
			defer next.Close()
			if loaded.Status != workflow.RunRecoveryRequired || len(loaded.Integrations) != 0 {
				t.Fatal("incomplete integration unblocked work")
			}
			a, _, _ = loaded.FindAttempt("att-a")
			if point != "intent" && (a.Capture.Observation == nil || a.Capture.Observation.Outcome == "unknown") {
				t.Fatalf("lost observable outcome: %+v", a.Capture)
			}
			if _, err := engine.integrateHeld(context.Background(), next, loaded, f.git, req); err == nil {
				t.Fatal("replay allowed")
			}
			if beforeHead != gitT(t, f.repo, "rev-parse", req.SourceBranch) || beforeIndex != gitT(t, path, "ls-files", "--stage") || beforeObjects != gitT(t, f.repo, "count-objects", "-v") {
				t.Fatal("recovery mutated Git")
			}
		})
	}
}

func TestCaptureRecoveryMovedSourceIsUnknown(t *testing.T) {
	f, lease, x, req := captureFixture(t)
	g := &captureFaultGit{IntegrationGit: f.git, point: "candidate", stop: func() { lease.Close() }}
	if _, err := f.engine.integrateHeld(context.Background(), lease, x, g, req); err == nil {
		t.Fatal("fault ignored")
	}
	loaded := f.load(engineExecutionID)
	a, _, _ := loaded.FindAttempt("att-a")
	candidate := a.Capture.CandidateSHA
	foreign := gitT(t, f.repo, "commit-tree", gitT(t, f.repo, "rev-parse", f.base+"^{tree}"), "-p", f.base, "-m", "foreign")
	gitT(t, f.repo, "update-ref", req.SourceBranch, foreign, f.base)
	f.engine.Workspaces = func(context.Context, workflow.Execution, string) (WorkspaceGit, error) { return f.git, nil }
	next, got, err := f.engine.Claim(context.Background(), "proj", engineExecutionID)
	if err != nil {
		t.Fatal(err)
	}
	defer next.Close()
	a, _, _ = got.FindAttempt("att-a")
	if a.Capture.Observation.Outcome != "unknown" || a.Capture.CandidateSHA != candidate || a.Capture.Observation.HeadSHA != foreign {
		t.Fatal("foreign commit adopted")
	}
	if got.Status != workflow.RunRecoveryRequired || f.target() != f.base {
		t.Fatal("unknown unblocked integration")
	}
}

func TestCaptureStoreHistoryAndSnapshotIsolation(t *testing.T) {
	f, lease, x, req := captureFixture(t)
	in := workflow.IntegrationInput{IntegrationID: req.IntegrationID, Generation: lease.Generation(), SourceAttemptID: req.SourceAttemptID, RepositoryKey: req.RepositoryKey, RepositoryIdentity: req.RepositoryIdentity, SourceBranch: req.SourceBranch, Worktree: req.Worktree, BaseSHA: req.BaseSHA, TargetRef: req.TargetRef, ExpectedTargetHead: req.ExpectedTargetHead, Now: time.Now()}
	path := filepath.Join(f.root, req.Worktree)
	clone := func(x workflow.Execution) workflow.Execution {
		wire, err := workflow.EncodeExecution(x)
		if err != nil {
			t.Fatal(err)
		}
		next, err := workflow.DecodeExecution(wire)
		if err != nil {
			t.Fatal(err)
		}
		return next
	}
	intent := clone(x)
	if err := intent.PrepareCapture(in, path); err != nil {
		t.Fatal(err)
	}
	if err := f.store.Save(context.Background(), intent, x.Revision, nil); !errors.Is(err, workflow.ErrStaleRunner) {
		t.Fatal("unfenced intent accepted", err)
	}
	a, _, _ := intent.FindAttempt("att-a")
	a.Capture.CandidateSHA = f.base[:39] + "1"
	if err := f.store.Save(context.Background(), intent, x.Revision, lease.(*local.RunLease)); err == nil {
		t.Fatal("intent+candidate saved together")
	}
	var err error
	x, err = f.store.UpdateFenced(context.Background(), "proj", engineExecutionID, lease, func(x *workflow.Execution) error { return x.PrepareCapture(in, path) })
	if err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"path", "generation", "remove"} {
		next := clone(x)
		a, _, _ := next.FindAttempt("att-a")
		switch mode {
		case "path":
			a.Capture.Path = filepath.Join(filepath.Dir(f.root), "other", req.Worktree)
		case "generation":
			a.Capture.RunnerGeneration++
		case "remove":
			a.Capture = nil
		}
		next.Revision++
		if err := f.store.Save(context.Background(), next, x.Revision, lease.(*local.RunLease)); err == nil {
			t.Fatal("history rewrite", mode)
		}
	}
	before, _ := workflow.EncodeExecution(x)
	refusal := errors.New("refused")
	got, err := f.store.UpdateFenced(context.Background(), "proj", engineExecutionID, lease, func(x *workflow.Execution) error {
		a, _, _ := x.FindAttempt("att-a")
		a.Capture.Path = "/bad/agent-a"
		return refusal
	})
	after, _ := workflow.EncodeExecution(got)
	if !errors.Is(err, refusal) || string(before) != string(after) {
		t.Fatal("failed update aliased capture")
	}
}

type captureMutationGit struct {
	IntegrationGit
	afterCandidate func(string)
}

func (g captureMutationGit) CapturePending(ctx context.Context, n, b, base, m string, intent func(string) error, candidate func(string) error, before func() error) error {
	return g.IntegrationGit.CapturePending(ctx, n, b, base, m, intent, func(sha string) error {
		if err := candidate(sha); err != nil {
			return err
		}
		g.afterCandidate(sha)
		return nil
	}, before)
}

func TestCaptureMovedRefsAndExactPersistedCandidate(t *testing.T) {
	for _, mode := range []string{"source", "target", "binding", "happy"} {
		t.Run(mode, func(t *testing.T) {
			f, lease, x, req := captureFixture(t)
			var persisted string
			foreign := gitT(t, f.repo, "commit-tree", gitT(t, f.repo, "rev-parse", f.base+"^{tree}"), "-p", f.base, "-m", "foreign")
			g := captureMutationGit{IntegrationGit: f.git, afterCandidate: func(sha string) {
				loaded := f.load(engineExecutionID)
				a, _, _ := loaded.FindAttempt("att-a")
				if a.Capture == nil || a.Capture.CandidateSHA != sha || a.Capture.Path != filepath.Join(f.root, req.Worktree) || a.Capture.Observation != nil || gitT(t, f.repo, "rev-parse", req.SourceBranch) != f.base {
					t.Fatal("candidate not durably correlated before source CAS")
				}
				persisted = sha
				switch mode {
				case "source":
					gitT(t, f.repo, "update-ref", req.SourceBranch, foreign, f.base)
				case "target":
					gitT(t, f.repo, "update-ref", req.TargetRef, foreign, f.base)
				case "binding":
					gitT(t, filepath.Join(f.root, req.Worktree), "checkout", "--detach", f.base)
				}
			}}
			got, err := f.engine.integrateHeld(context.Background(), lease, x, g, req)
			if persisted == "" {
				t.Fatal("no persisted candidate inspected")
			}
			if mode == "happy" {
				if err != nil || !got.AttemptIntegrated(req.SourceAttemptID) {
					t.Fatal(err)
				}
				return
			}
			if err == nil {
				t.Fatal("moved ref/binding integrated")
			}
			loaded := f.load(engineExecutionID)
			a, _, _ := loaded.FindAttempt("att-a")
			if a.Capture.CandidateSHA != persisted {
				t.Fatal("candidate identity rewritten")
			}
			if mode == "target" {
				if f.target() != foreign || got.Integrations[0].Status != workflow.IntegrationNotIntegrated {
					t.Fatal("target CAS did not preserve foreign advance")
				}
			} else {
				if a.Capture.Observation.Outcome != "unknown" || f.target() != f.base {
					t.Fatal("foreign source adopted")
				}
			}
		})
	}
}

func TestCaptureOwnCommitAndExecutableConfiguration(t *testing.T) {
	for _, mode := range []string{"own-clean", "own-dirty", "filter", "diff", "hooks"} {
		t.Run(mode, func(t *testing.T) {
			f, lease, x, req := captureFixture(t)
			path := filepath.Join(f.root, req.Worktree)
			if mode == "own-clean" || mode == "own-dirty" {
				gitT(t, path, "add", "-A")
				gitT(t, path, "commit", "-q", "-m", "writer-owned")
				if mode == "own-dirty" {
					write(t, filepath.Join(path, "leftover.txt"), "do not adopt\n")
				}
			}
			sentinel := filepath.Join(filepath.Dir(f.repo), "executed")
			switch mode {
			case "filter":
				gitT(t, path, "config", "filter.danger.clean", "touch "+sentinel)
			case "diff":
				gitT(t, path, "config", "diff.external", "touch "+sentinel)
			case "hooks":
				hook := filepath.Join(filepath.Dir(f.repo), "hooks")
				if err := os.Mkdir(hook, 0o700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(hook, "pre-commit"), []byte("#!/bin/sh\ntouch "+sentinel+"\n"), 0o700); err != nil {
					t.Fatal(err)
				}
				gitT(t, path, "config", "core.hooksPath", hook)
			}
			head := gitT(t, f.repo, "rev-parse", req.SourceBranch)
			index := gitT(t, path, "ls-files", "--stage")
			got, err := f.engine.integrateHeld(context.Background(), lease, x, f.git, req)
			if mode == "own-clean" || mode == "hooks" {
				if err != nil || !got.AttemptIntegrated(req.SourceAttemptID) {
					t.Fatal(err)
				}
			} else {
				if err == nil || f.target() != f.base || gitT(t, f.repo, "rev-parse", req.SourceBranch) != head || gitT(t, path, "ls-files", "--stage") != index {
					t.Fatal("refusal changed Git")
				}
			}
			loaded := f.load(engineExecutionID)
			a, _, _ := loaded.FindAttempt("att-a")
			if mode != "hooks" && a.Capture != nil {
				t.Fatal("writer-owned/unsafe work captured")
			}
			if _, err := os.Stat(sentinel); !errors.Is(err, os.ErrNotExist) {
				t.Fatal("Git executable configuration ran")
			}
		})
	}
}

func TestCaptureRecoveryPreservesPriorSuccessAndObservesDrift(t *testing.T) {
	f, lease, x, req := captureFixture(t)
	f.engine.Store = captureStoreFault{ExecutionStore: f.store, point: "prepared"}
	if _, err := f.engine.integrateHeld(context.Background(), lease, x, f.git, req); err == nil {
		t.Fatal("fault ignored")
	}
	loaded := f.load(engineExecutionID)
	a, _, _ := loaded.FindAttempt("att-a")
	candidate := a.Capture.CandidateSHA
	if a.Capture.Observation.Outcome != "captured" {
		t.Fatal("missing prior success")
	}
	foreign := gitT(t, f.repo, "commit-tree", gitT(t, f.repo, "rev-parse", f.base+"^{tree}"), "-p", f.base, "-m", "foreign")
	gitT(t, f.repo, "update-ref", req.SourceBranch, foreign, candidate)
	lease.Close()
	f.engine.Store = f.store
	f.engine.Workspaces = func(context.Context, workflow.Execution, string) (WorkspaceGit, error) { return f.git, nil }
	next, got, err := f.engine.Claim(context.Background(), "proj", engineExecutionID)
	if err != nil {
		t.Fatal(err)
	}
	defer next.Close()
	a, _, _ = got.FindAttempt("att-a")
	if a.Capture.Observation.Outcome != "captured" || a.Capture.Recovery == nil || a.Capture.Recovery.Outcome != "unknown" || a.Capture.Recovery.HeadSHA != foreign || got.Status != workflow.RunRecoveryRequired {
		t.Fatalf("historical/recovery observations conflated: %+v", a.Capture)
	}
}

func TestCaptureLostIntegrationOutcomeDoesNotReplay(t *testing.T) {
	f, lease, x, req := captureFixture(t)
	drop := &lostFinishStore{ExecutionStore: f.store}
	drop.drop.Store(true)
	f.engine.Store = drop
	if _, err := f.engine.integrateHeld(context.Background(), lease, x, f.git, req); err == nil {
		t.Fatal("lost outcome ignored")
	}
	loaded := f.load(engineExecutionID)
	if len(loaded.Integrations) != 1 || loaded.Integrations[0].Status != workflow.IntegrationPrepared {
		t.Fatal("prepared integration lost")
	}
	head := f.target()
	objects := gitT(t, f.repo, "count-objects", "-v")
	lease.Close()
	f.engine.Store = f.store
	f.engine.Workspaces = func(context.Context, workflow.Execution, string) (WorkspaceGit, error) { return f.git, nil }
	next, got, err := f.engine.Claim(context.Background(), "proj", engineExecutionID)
	if err != nil {
		t.Fatal(err)
	}
	defer next.Close()
	if got.Integrations[0].Status != workflow.IntegrationUnknown || got.Status != workflow.RunRecoveryRequired {
		t.Fatal("lost target outcome adopted")
	}
	if _, err := f.engine.integrateHeld(context.Background(), next, got, f.git, req); err == nil {
		t.Fatal("reentry allowed")
	}
	if f.target() != head || gitT(t, f.repo, "count-objects", "-v") != objects {
		t.Fatal("integration replayed")
	}
}
