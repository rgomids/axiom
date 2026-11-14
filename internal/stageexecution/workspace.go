package stageexecution

import (
	"context"
	"crypto/sha256"
	"fmt"
	"slices"

	"github.com/rgomids/axiom/internal/workflow"
)

// WorkspaceGit extends the existing integration boundary with bounded workspace
// preparation. The principal commits its exact binding before calling mutations.
type WorkspaceGit interface {
	IntegrationGit
	ObserveBase(context.Context) (string, error)
	PrepareTarget(context.Context, string, string) error
	MaterializeWorkspace(context.Context, string, string, string) (string, error)
}

func workspaceScope(x workflow.Execution, key string) string {
	return fmt.Sprintf("e-%x", sha256.Sum256([]byte(x.ExecutionID+"\x00"+key)))[:26]
}

func (e Engine) workspaceGit(ctx context.Context, x workflow.Execution, key string) (WorkspaceGit, error) {
	if e.Workspaces == nil {
		return nil, ErrIntegrationRefused
	}
	if e.Admit != nil {
		if err := e.Admit(ctx, x); err != nil {
			return nil, err
		}
	}
	return e.Workspaces(ctx, x, key)
}

// AfterGateIntegrationDependency reports the frozen-plan refusal: an
// after-gated Stage has a same-Stage dependent of an enrolled writer. It is a
// pure function of the persisted Execution, so it can never become runnable.
func AfterGateIntegrationDependency(x workflow.Execution) bool {
	for _, stage := range x.Workflow.Stages {
		if !stage.HumanGate {
			continue
		}
		for _, agent := range stage.Agents {
			for _, dependency := range agent.DependsOn {
				for _, source := range stage.Agents {
					if source.ID == dependency && slices.Contains(source.Effects, "workspace_write") && x.IntegrationEnrolled(workflow.TaskRef{StageID: stage.ID, AgentID: source.ID}) {
						return true
					}
				}
			}
		}
	}
	return false
}

// preflightWorkspaces checks configured writers, including dependent writers,
// before any enrolled writer is launched or any Git intent is materialized.
func (e Engine) preflightWorkspaces(ctx context.Context, x workflow.Execution) error {
	if AfterGateIntegrationDependency(x) {
		return ErrAfterGateIntegrationDependency
	}
	for _, stage := range x.Workflow.Stages {
		for _, agent := range stage.Agents {
			ref := workflow.TaskRef{StageID: stage.ID, AgentID: agent.ID}
			if !x.IntegrationEnrolled(ref) {
				continue
			}
			_, authority, err := e.Resolve(ctx, x, ref)
			if err != nil {
				return err
			}
			if !slices.Contains(authority.Effects, "workspace_write") {
				continue
			}
			if len(stage.Commands) == 0 {
				return workflow.ErrIntegrationChecksUnavailable
			}
			if !slices.Contains(authority.Effects, "validation_command") {
				return workflow.ErrUnauthorizedEffect
			}
			for _, check := range stage.Commands {
				for _, effect := range check.Effects {
					if !slices.Contains(authority.Effects, effect) {
						return workflow.ErrUnauthorizedEffect
					}
				}
			}
		}
	}
	return nil
}

func (e Engine) planWorkspace(ctx context.Context, x workflow.Execution, ref workflow.TaskRef, attemptID string, authority workflow.AttemptAuthority) (*workflow.AttemptWorkspace, bool, error) {
	if !x.IntegrationEnrolled(ref) {
		return nil, false, nil
	}
	git, err := e.workspaceGit(ctx, x, authority.RepositoryKey)
	if err != nil {
		return nil, false, err
	}
	initial, err := git.ObserveBase(ctx)
	if err != nil {
		return nil, false, err
	}
	target := "refs/heads/axiom/" + workspaceScope(x, authority.RepositoryKey) + "/integration"
	first := true
	for _, stage := range x.Stages {
		for _, task := range stage.Tasks {
			for _, attempt := range task.Attempts {
				if w := attempt.Workspace; w != nil && w.RepositoryKey == authority.RepositoryKey {
					first = false
					if initial != w.InitialTargetHead || target != w.TargetRef {
						return nil, false, workflow.ErrIntegrationStale
					}
				}
			}
		}
	}
	base := x.AcceptedTargetHead(authority.RepositoryKey, target)
	if base == "" {
		base = initial
	}
	if !first {
		observed, err := git.ObserveTarget(ctx, target)
		if err != nil {
			return nil, false, err
		}
		if observed != base {
			return nil, false, workflow.ErrIntegrationStale
		}
	}
	identity := ""
	for _, r := range x.Repositories {
		if r.Key == authority.RepositoryKey {
			identity = r.Identity
		}
	}
	name := workspaceScope(x, authority.RepositoryKey) + "-" + attemptID
	return &workflow.AttemptWorkspace{RepositoryKey: authority.RepositoryKey, RepositoryIdentity: identity, BaseSHA: base, SourceBranch: "refs/heads/axiom/" + workspaceScope(x, authority.RepositoryKey) + "/" + attemptID, Worktree: name, TargetRef: target, InitialTargetHead: initial}, first, nil
}

func (e Engine) materializeWorkspace(ctx context.Context, x workflow.Execution, w *workflow.AttemptWorkspace, first bool) error {
	if w == nil {
		return nil
	}
	git, err := e.workspaceGit(ctx, x, w.RepositoryKey)
	if err != nil {
		return err
	}
	initial, err := git.ObserveBase(ctx)
	if err != nil {
		return err
	}
	if initial != w.InitialTargetHead {
		return workflow.ErrIntegrationStale
	}
	if first {
		if err := git.PrepareTarget(ctx, w.TargetRef, w.InitialTargetHead); err != nil {
			return err
		}
	}
	observed, err := git.ObserveTarget(ctx, w.TargetRef)
	if err != nil {
		return err
	}
	if observed != w.BaseSHA {
		return workflow.ErrIntegrationStale
	}
	_, err = git.MaterializeWorkspace(ctx, w.Worktree, w.SourceBranch, w.BaseSHA)
	return err
}

func (e Engine) integrateReady(ctx context.Context, lease workflow.RunnerFence, x workflow.Execution) (workflow.Execution, error) {
	if x.Status == workflow.RunRecoveryRequired || x.Cancel != nil {
		return x, nil
	}
	for _, stage := range x.Stages {
		for _, task := range stage.Tasks {
			if len(task.Attempts) == 0 {
				continue
			}
			a := task.Attempts[len(task.Attempts)-1]
			if a.Status != workflow.AttemptSucceeded || !x.RequiresIntegration(a.AttemptID) || x.AttemptIntegrated(a.AttemptID) {
				continue
			}
			// Recorded failures and uncertainty are terminal, never automatic retries.
			for _, i := range x.Integrations {
				if i.SourceAttemptID == a.AttemptID {
					return x, ErrNotIntegrated
				}
			}
			waitingGate := false
			if stage.Gate == nil {
				for _, spec := range x.Workflow.Stages {
					waitingGate = waitingGate || spec.ID == stage.StageID && spec.HumanGate
				}
			}
			if waitingGate {
				if stage.Status != workflow.StageAwaitingGate {
					continue
				}
				if !a.WorkspaceGateWait {
					var err error
					x, err = e.commit(ctx, x.ProjectID, x.ExecutionID, lease)(func(current *workflow.Execution) error {
						return current.MarkWorkspaceGateWait(a.AttemptID, lease.Generation(), e.now())
					})
					if err != nil {
						return x, err
					}
				}
				continue
			}
			w := a.Workspace
			if w == nil {
				return x, ErrIntegrationRefused
			}
			git, err := e.workspaceGit(ctx, x, w.RepositoryKey)
			if err != nil {
				return x, err
			}
			initial, err := git.ObserveBase(ctx)
			if err != nil {
				return x, err
			}
			if initial != w.InitialTargetHead {
				return x, workflow.ErrIntegrationStale
			}
			head := x.AcceptedTargetHead(w.RepositoryKey, w.TargetRef)
			if head == "" {
				head = w.InitialTargetHead
			}
			x, err = e.integrateHeld(ctx, lease, x, git, IntegrationRequest{SourceAttemptID: a.AttemptID, RepositoryKey: w.RepositoryKey, RepositoryIdentity: w.RepositoryIdentity, Worktree: w.Worktree, SourceBranch: w.SourceBranch, BaseSHA: w.BaseSHA, TargetRef: w.TargetRef, ExpectedTargetHead: head})
			if err != nil {
				return x, err
			}
		}
	}
	return x, nil
}
