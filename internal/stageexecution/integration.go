package stageexecution

import (
	"context"
	"errors"
	"path/filepath"
	"time"

	"github.com/rgomids/axiom/internal/executiongraph"
	"github.com/rgomids/axiom/internal/workflow"
)

// IntegrationGit is the principal's bounded local Git boundary
// (gitworkspace.Integrator). Only Advance moves a ref, by compare-and-swap.
type IntegrationGit interface {
	ObserveCandidate(ctx context.Context, worktree, branch, base string) (string, error)
	ObserveTarget(ctx context.Context, ref string) (string, error)
	Merge(ctx context.Context, head, candidate, message string) (string, error)
	PrepareCheckWorktree(ctx context.Context, name, merge string) (string, error)
	ObserveCheckWorktree(ctx context.Context, path, merge string) error
	Advance(ctx context.Context, ref, expected, merge string) (string, error)
}

// IntegrationRequest names one succeeded attempt's isolated branch and the
// principal-owned target it may be integrated into, with the base and target
// head the principal expects.
type IntegrationRequest struct {
	IntegrationID      string
	SourceAttemptID    string
	RepositoryKey      string
	RepositoryIdentity string
	Worktree           string
	SourceBranch       string
	BaseSHA            string
	TargetRef          string
	ExpectedTargetHead string
}

var (
	// ErrIntegrationRefused: observation before any intent refused; nothing was
	// recorded and no Git effect happened.
	ErrIntegrationRefused = errors.New("integration_refused")
	// ErrNotIntegrated: the intent and its outcome are recorded; the target was
	// not advanced by this integration.
	ErrNotIntegrated = errors.New("not_integrated")
	// ErrIntegrationUnknown: a check process/effect or target effect is unknown. It is
	// recorded unknown and never retried automatically.
	ErrIntegrationUnknown = errors.New("integration_outcome_unknown")
)

// Integrate is the principal's exclusive integration operation (Spec 015
// "Parallel work and integration"). It holds the Execution's runner lease, so
// it is serialized with the run loop and with every other integration of the
// Execution; agent attempts hold no lease and cannot call it, and messages
// never reach it. Concurrent integrations of other Executions into the same
// target are serialized by the target compare-and-swap.
//
// Order: observe candidate and target (refusal records nothing), persist the
// intent, compute the merge without touching refs, run every source-stage
// check on the merged tree in a new principal worktree, verify the checks
// left it unchanged and no cancel arrived, then compare-and-swap the target
// and record what was observed. Nothing is reset, removed or retried.
func (e Engine) Integrate(ctx context.Context, projectID, executionID string, git IntegrationGit, req IntegrationRequest) (workflow.Execution, error) {
	lease, current, err := e.Claim(ctx, projectID, executionID)
	if err != nil {
		return current, err
	}
	defer lease.Close()
	gen := lease.Generation()
	if req.IntegrationID == "" {
		if req.IntegrationID, err = e.NewID(); err != nil {
			return current, err
		}
	}
	candidate, err := git.ObserveCandidate(ctx, req.Worktree, req.SourceBranch, req.BaseSHA)
	if err != nil {
		return current, errors.Join(ErrIntegrationRefused, err)
	}
	head, err := git.ObserveTarget(ctx, req.TargetRef)
	if err != nil {
		return current, errors.Join(ErrIntegrationRefused, err)
	}
	if head != req.ExpectedTargetHead {
		return current, errors.Join(ErrIntegrationRefused, workflow.ErrIntegrationStale)
	}
	update := e.commit(ctx, projectID, executionID, lease)
	prepared, err := update(func(x *workflow.Execution) error {
		return x.PrepareIntegration(workflow.IntegrationInput{
			IntegrationID: req.IntegrationID, Generation: gen, SourceAttemptID: req.SourceAttemptID,
			RepositoryKey: req.RepositoryKey, RepositoryIdentity: req.RepositoryIdentity,
			SourceBranch: req.SourceBranch, Worktree: req.Worktree, BaseSHA: req.BaseSHA,
			TargetRef: req.TargetRef, ExpectedTargetHead: head, CandidateSHA: candidate, Now: e.now(),
		})
	})
	if err != nil {
		return current, errors.Join(ErrIntegrationRefused, err)
	}
	record := workflow.IntegrationResult{}
	finish := func(status workflow.IntegrationStatus, diagnostic string) (workflow.Execution, error) {
		record.Status, record.Diagnostic, record.Now = status, diagnostic, e.now()
		if status == workflow.IntegrationIntegrated {
			record.FinalSHA = record.MergeSHA
		}
		if record.ObservedTargetHead == "" && status == workflow.IntegrationNotIntegrated {
			// Before Advance the target was never written by this integration;
			// the observation is forensic only.
			record.ObservedTargetHead, _ = git.ObserveTarget(context.WithoutCancel(ctx), req.TargetRef)
		}
		x, err := update(func(x *workflow.Execution) error { return x.FinishIntegration(req.IntegrationID, gen, record) })
		switch {
		case err != nil:
			// The outcome stays prepared; the next runner records it unknown.
			return prepared, err
		case status == workflow.IntegrationNotIntegrated:
			return x, errors.Join(ErrNotIntegrated, errors.New(diagnostic))
		case status == workflow.IntegrationUnknown:
			return x, errors.Join(ErrIntegrationUnknown, errors.New(diagnostic))
		}
		return x, nil
	}
	merge, err := git.Merge(ctx, head, candidate, "axiom: integrate "+req.SourceBranch+" ("+req.SourceAttemptID+")")
	if err != nil {
		if errors.Is(err, workflow.ErrIntegrationConflict) {
			return finish(workflow.IntegrationNotIntegrated, "merge_conflict")
		}
		return finish(workflow.IntegrationNotIntegrated, "merge_failed")
	}
	record.MergeSHA = merge
	path, err := git.PrepareCheckWorktree(ctx, "integration-"+req.IntegrationID, merge)
	if err != nil {
		return finish(workflow.IntegrationNotIntegrated, "check_worktree_unavailable")
	}
	_, ref, _ := prepared.FindAttempt(req.SourceAttemptID)
	var stage workflow.StageSpec
	for _, s := range prepared.Workflow.Stages {
		if s.ID == ref.StageID {
			stage = s
		}
	}
	checkCtx, stopChecks := context.WithCancel(ctx)
	joinCancel := e.watchCancel(checkCtx, stopChecks, projectID, executionID)
	defer joinCancel()
	beforeStart := func() error {
		if checkCtx.Err() != nil || e.cancelPersisted(ctx, projectID, executionID) {
			return workflow.ErrCancelled
		}
		return nil
	}
	for _, check := range stage.Commands {
		if beforeStart() != nil {
			return finish(workflow.IntegrationNotIntegrated, "cancelled_before_check")
		}
		result := e.runIntegrationCheck(checkCtx, path, check, beforeStart)
		record.Checks = append(record.Checks, result)
		if result.Status == workflow.CheckUnknown {
			return finish(workflow.IntegrationUnknown, "post_merge_check_unknown")
		}
		if result.Status != workflow.CheckPassed {
			return finish(workflow.IntegrationNotIntegrated, "post_merge_check_"+result.Status)
		}
	}
	if err := git.ObserveCheckWorktree(ctx, path, merge); err != nil {
		return finish(workflow.IntegrationNotIntegrated, "check_mutated_worktree")
	}
	if ctx.Err() != nil {
		return finish(workflow.IntegrationNotIntegrated, "interrupted_before_effect")
	}
	if e.cancelPersisted(ctx, projectID, executionID) {
		return finish(workflow.IntegrationNotIntegrated, "cancelled_before_effect")
	}
	observed, err := git.Advance(ctx, req.TargetRef, head, merge)
	record.ObservedTargetHead = observed
	switch {
	case err == nil && observed == merge:
		return finish(workflow.IntegrationIntegrated, "")
	case err == nil:
		return finish(workflow.IntegrationUnknown, "target_observation_mismatch_after_advance")
	case errors.Is(err, workflow.ErrIntegrationTargetCheckedOut):
		return finish(workflow.IntegrationNotIntegrated, "target_checked_out")
	case observed == "":
		return finish(workflow.IntegrationUnknown, "advance_outcome_unknown")
	case observed == merge:
		return finish(workflow.IntegrationUnknown, "advance_outcome_ambiguous")
	case observed == head:
		return finish(workflow.IntegrationNotIntegrated, "target_advance_refused")
	}
	return finish(workflow.IntegrationNotIntegrated, "target_moved_concurrently")
}

// runIntegrationCheck re-runs one authorized stage command on the merged tree
// through the ADR-0028 observed process boundary with an exact minimal
// environment. Unconfirmed process/effect outcomes remain unknown.
func (e Engine) runIntegrationCheck(ctx context.Context, cwd string, check workflow.CommandCheck, beforeStart func() error) workflow.IntegrationCheck {
	terminated := workflow.IntegrationCheck{ID: check.ID, Status: workflow.CheckTerminated, ExitCode: -1}
	if len(check.Argv) == 0 || !filepath.IsAbs(check.Argv[0]) || !cleanRuntimePath(check.Argv[0]) || !cleanRuntimePath(cwd) {
		return terminated
	}
	checkCtx, cancel := context.WithTimeout(ctx, time.Duration(check.TimeoutSeconds)*time.Second)
	defer cancel()
	gate := []string{commandGateShell, "-c", commandGateScript, "axiom-command-gate", check.Argv[0]}
	inv := executiongraph.Invocation{Argv: append(gate, check.Argv[1:]...), CWD: cwd, Stdin: []byte("axiom-command-start\n"), OutputMax: 4096}
	inv = executiongraph.WithExactEnvironment(inv, []string{"PATH=/usr/bin:/bin", "LC_ALL=C"})
	if beforeStart() != nil {
		return terminated
	}
	result := e.Runner.RunObserved(checkCtx, inv, func(*executiongraph.ProcessCorrelation) error {
		// The command gate still holds stdin while persisted cancellation is checked.
		return beforeStart()
	})
	if result.OutcomeUnknown || result.EffectAmbiguous {
		return workflow.IntegrationCheck{ID: check.ID, Status: workflow.CheckUnknown, ExitCode: result.ExitCode}
	}
	if result.SpawnFailed || result.Stopped || result.TimedOut || checkCtx.Err() != nil {
		return terminated
	}
	if result.ExitCode != 0 {
		return workflow.IntegrationCheck{ID: check.ID, Status: workflow.CheckFailed, ExitCode: result.ExitCode}
	}
	return workflow.IntegrationCheck{ID: check.ID, Status: workflow.CheckPassed}
}
