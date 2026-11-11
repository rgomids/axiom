package workflow

import (
	"reflect"
	"slices"
	"time"
)

// IntegrationEnrollment freezes the new-run decision to isolate a stage's
// same-repository writers. Its absence preserves historical run semantics.
type IntegrationEnrollment struct {
	StageID       string `json:"stageId"`
	RepositoryKey string `json:"repositoryKey"`
}

// AttemptWorkspace is principal-owned intent, persisted before creating any
// refs or worktree. Machine-local roots remain outside the portable aggregate.
type AttemptWorkspace struct {
	RepositoryKey      string `json:"repositoryKey"`
	RepositoryIdentity string `json:"repositoryIdentity"`
	BaseSHA            string `json:"baseSha"`
	SourceBranch       string `json:"sourceBranch"`
	Worktree           string `json:"worktree"`
	TargetRef          string `json:"targetRef"`
	InitialTargetHead  string `json:"initialTargetHead"`
}

func cloneAttemptWorkspace(w *AttemptWorkspace) *AttemptWorkspace {
	if w == nil {
		return nil
	}
	copied := *w
	return &copied
}

func (e Execution) agentRepository(agent AgentSpec) string {
	if agent.Repository != "" {
		return agent.Repository
	}
	return e.WorkItem.RepositoryKey
}

func (e Execution) integrationEnrollments() []IntegrationEnrollment {
	var out []IntegrationEnrollment
	for _, stage := range e.Workflow.Stages {
		counts := map[string]int{}
		for _, agent := range stage.Agents {
			if slices.Contains(agent.Effects, "workspace_write") {
				counts[e.agentRepository(agent)]++
			}
		}
		// Repository binding order gives a deterministic encoding.
		for _, repo := range e.Repositories {
			if counts[repo.Key] >= 2 {
				out = append(out, IntegrationEnrollment{stage.ID, repo.Key})
			}
		}
	}
	return out
}

// IntegrationEnrolled includes repository consumers in downstream stages, so
// they observe the accepted tree rather than the operator's original checkout.
func (e Execution) IntegrationEnrolled(ref TaskRef) bool {
	_, agent, _, err := e.task(ref)
	if err != nil {
		return false
	}
	repository := e.agentRepository(agent)
	visited := map[string]bool{}
	var follows func(string, string) bool
	follows = func(stage, ancestor string) bool {
		if stage == ancestor {
			return true
		}
		if visited[stage] {
			return false
		}
		visited[stage] = true
		i := e.stageIndex(stage)
		if i < 0 {
			return false
		}
		for _, dep := range e.Workflow.Stages[i].DependsOn {
			if follows(dep, ancestor) {
				return true
			}
		}
		return false
	}
	for _, enrollment := range e.Enrollments {
		clear(visited)
		if enrollment.RepositoryKey == repository && follows(ref.StageID, enrollment.StageID) {
			return true
		}
	}
	return false
}

func (e Execution) RequiresIntegration(attemptID string) bool {
	a, ref, ok := e.FindAttempt(attemptID)
	return ok && e.IntegrationEnrolled(ref) && slices.Contains(a.Authority.Effects, "workspace_write")
}

func (e Execution) AttemptIntegrated(attemptID string) bool {
	for _, i := range e.Integrations {
		if i.SourceAttemptID == attemptID && i.Status == IntegrationIntegrated {
			return true
		}
	}
	return false
}

func (e Execution) integrationFailed(attemptID string) bool {
	for _, i := range e.Integrations {
		if i.SourceAttemptID == attemptID && i.Status == IntegrationNotIntegrated {
			return true
		}
	}
	return false
}

func validateWorkspaces(e Execution) error {
	if len(e.Enrollments) > MaxExecutionStages*len(e.Repositories) {
		return invalid("integration enrollment bounds")
	}
	if len(e.Enrollments) > 0 && !reflect.DeepEqual(e.Enrollments, e.integrationEnrollments()) {
		return invalid("integration enrollment")
	}
	targets := map[string]AttemptWorkspace{}
	sources, worktrees := map[string]bool{}, map[string]bool{}
	for _, stage := range e.Stages {
		for _, task := range stage.Tasks {
			for _, attempt := range task.Attempts {
				ref := TaskRef{stage.StageID, task.AgentID}
				enrolled := e.IntegrationEnrolled(ref)
				if attempt.WorkspaceGateWait && (attempt.Status != AttemptSucceeded || !e.RequiresIntegration(attempt.AttemptID) || !e.Workflow.Stages[e.stageIndex(stage.StageID)].HumanGate) {
					return invalid("workspace gate wait")
				}
				if attempt.WorkspaceGateWait {
					if _, err := e.StageResultDigest(stage.StageID); err != nil {
						return invalid("workspace gate wait before stage completion")
					}
				}
				if attempt.WorkspaceUncertain && (attempt.Status != AttemptSucceeded || !e.RequiresIntegration(attempt.AttemptID) || hasIntegration(e, attempt.AttemptID)) {
					return invalid("workspace uncertainty")
				}
				w := attempt.Workspace
				if enrolled != (w != nil) {
					return invalid("attempt workspace enrollment")
				}
				if w == nil {
					continue
				}
				_, agent, _, _ := e.task(ref)
				if w.RepositoryKey != e.agentRepository(agent) || w.RepositoryKey != attempt.Authority.RepositoryKey || !ValidObjectID(w.BaseSHA) || !ValidObjectID(w.InitialTargetHead) || !ValidBranchRef(w.SourceBranch) || !ValidBranchRef(w.TargetRef) || w.SourceBranch == w.TargetRef || !ValidWorktreeName(w.Worktree) {
					return invalid("attempt workspace identity")
				}
				bound := false
				for _, repo := range e.Repositories {
					bound = bound || repo.Key == w.RepositoryKey && repo.Identity == w.RepositoryIdentity
				}
				if !bound {
					return invalid("attempt workspace repository")
				}
				key := w.RepositoryKey + "\x00"
				if sources[key+w.SourceBranch] || worktrees[key+w.Worktree] {
					return invalid("attempt workspace collision")
				}
				sources[key+w.SourceBranch], worktrees[key+w.Worktree] = true, true
				if previous, ok := targets[w.RepositoryKey]; ok && (previous.TargetRef != w.TargetRef || previous.InitialTargetHead != w.InitialTargetHead) {
					return invalid("attempt workspace target")
				}
				knownBase := w.BaseSHA == w.InitialTargetHead
				for _, integration := range e.Integrations {
					knownBase = knownBase || integration.RepositoryKey == w.RepositoryKey && integration.TargetRef == w.TargetRef && integration.Status == IntegrationIntegrated && integration.FinalSHA == w.BaseSHA && integration.EndedAt != nil && !attempt.PreparedAt.Before(*integration.EndedAt)
				}
				if !knownBase {
					return invalid("attempt workspace base lineage")
				}
				targets[w.RepositoryKey] = *w
			}
		}
	}
	return nil
}

func hasIntegration(e Execution, attemptID string) bool {
	for _, integration := range e.Integrations {
		if integration.SourceAttemptID == attemptID {
			return true
		}
	}
	return false
}

// MarkWorkspaceUnknown preserves a validated terminal result when a lost runner
// left no integration observation. The missing outcome never authorizes replay.
func (e *Execution) MarkWorkspaceUnknown(id string, generation uint64, now time.Time) error {
	if err := e.fence(generation); err != nil {
		return err
	}
	a, _, ok := e.FindAttempt(id)
	if !ok || a.RunnerGeneration >= generation || a.Status != AttemptSucceeded || !e.RequiresIntegration(id) || hasIntegration(*e, id) {
		return ErrTransition
	}
	a.WorkspaceUncertain = true
	e.touch(now)
	return e.Validate()
}

// MarkWorkspaceGateWait records a known safe stop before human approval. It is
// not integration authority: the gate still has to be approved by a human.
func (e *Execution) MarkWorkspaceGateWait(id string, generation uint64, now time.Time) error {
	if err := e.fence(generation); err != nil {
		return err
	}
	a, ref, ok := e.FindAttempt(id)
	if !ok || a.RunnerGeneration != generation || a.Status != AttemptSucceeded || !e.RequiresIntegration(id) || hasIntegration(*e, id) {
		return ErrTransition
	}
	index := e.stageIndex(ref.StageID)
	if !e.Workflow.Stages[index].HumanGate || e.Stages[index].Status != StageAwaitingGate || e.Stages[index].Gate != nil || a.WorkspaceUncertain {
		return ErrTransition
	}
	a.WorkspaceGateWait = true
	e.touch(now)
	return e.Validate()
}
