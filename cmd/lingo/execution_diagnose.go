package main

import (
	"context"

	"github.com/rgomids/axiom/internal/cli"
	"github.com/rgomids/axiom/internal/runtimeprofile"
	"github.com/rgomids/axiom/internal/stageexecution"
	"github.com/rgomids/axiom/internal/workflow"
)

type invocationDiagnosis struct {
	StructuralReadiness             string                  `json:"structuralReadiness"`
	ProfileConfigurationEligibility string                  `json:"profileConfigurationEligibility"`
	InvocationAdmissibility         string                  `json:"invocationAdmissibility"`
	Authentication                  string                  `json:"authentication"`
	Inference                       string                  `json:"inference"`
	SelectionMode                   string                  `json:"selectionMode"`
	OutputDirectoryMaterialization  string                  `json:"outputDirectoryMaterialization"`
	LaunchFailure                   *workflow.LaunchFailure `json:"launchFailure,omitempty"`
	NextAction                      string                  `json:"nextAction"`
}

func (c executionComposition) diagnose(ctx context.Context, in cli.WorkflowStageInput) cli.WorkflowStageResponse {
	d := invocationDiagnosis{StructuralReadiness: "not_evaluated", ProfileConfigurationEligibility: "not_evaluated", InvocationAdmissibility: "not_evaluated", Authentication: "not_evaluated", Inference: "not_run", SelectionMode: "frozen", OutputDirectoryMaterialization: "not_evaluated", NextAction: "Inspect missing local invocation context; diagnosis grants no retry authority."}
	report := c.s.readiness().Evaluate(ctx, c.project.ID)
	d.StructuralReadiness = report.Structure
	x, err := c.store.Load(ctx, c.project.ID, in.Execution)
	if err != nil {
		return c.failure(err, nil)
	}
	finish := func() cli.WorkflowStageResponse {
		return c.s.stageSuccess("invocation_"+d.InvocationAdmissibility, "Local point-in-time diagnosis; authentication not evaluated; inference not run", []string{"execution:" + x.ExecutionID}, d.NextAction, d)
	}
	if len(in.Selections) > 0 && !in.NewSelection {
		return c.failure(errSelectionsFrozen, &x)
	}
	var stage workflow.StageSpec
	var agent workflow.AgentSpec
	var task *workflow.TaskRun
	for n, s := range x.Workflow.Stages {
		if s.ID == in.Stage {
			stage = s
			for j, a := range s.Agents {
				if a.ID == in.Agent {
					agent = a
					task = &x.Stages[n].Tasks[j]
				}
			}
		}
	}
	if task == nil {
		return c.failure(workflow.ErrTransition, &x)
	}
	ref := workflow.TaskRef{StageID: in.Stage, AgentID: in.Agent}
	attempt := workflow.Attempt{AttemptID: "diagnose-planned", RunnerGeneration: x.RunnerGeneration}
	if len(task.Attempts) > 0 && !in.NewSelection {
		attempt = task.Attempts[len(task.Attempts)-1]
	} else {
		if in.NewSelection {
			d.SelectionMode = "prospective_preview"
			selections, e := parseSelections(in.Selections)
			if e != nil {
				return c.failure(e, &x)
			}
			c.retrySelections, c.retryRef = selections, &ref
		}
		attempt.Invocation, attempt.Authority, err = c.resolve(ctx, x, ref)
		if err != nil {
			return finish()
		}
	}
	if config, e := c.runtimeConfiguration(ctx); e == nil && config != nil && attempt.Invocation.Profile != "" {
		if consumer, e := runtimeprofile.NewConsumerV4(*config); e == nil {
			d.ProfileConfigurationEligibility = "ineligible"
			if _, e = consumer.AllowedProfile(attempt.Invocation.Runtime, attempt.Invocation.Profile); e == nil {
				d.ProfileConfigurationEligibility = "eligible"
			}
		}
	}
	if err = c.admit(ctx, x); err != nil {
		return finish()
	}
	launcher, err := c.plannedLauncher(x.ExecutionID, x.WorkItem.RepositoryKey)
	if err != nil {
		return finish()
	}
	if _, err = launcher.Inspect(ctx, stageexecution.LaunchRequest{Execution: x, Ref: ref, Stage: stage, Agent: agent, Attempt: attempt}); err != nil {
		if ctx.Err() != nil {
			return finish()
		}
		failure := stageexecution.InspectLaunchFailure(err)
		d.InvocationAdmissibility = "refused"
		d.LaunchFailure = &failure
		d.NextAction = failure.NextAction() + " Diagnosis grants no retry authority."
	} else {
		d.InvocationAdmissibility = "admissible"
		d.NextAction = "Local checks passed only; output directory creation/permissions, authentication and inference remain unevaluated. Dispatch rechecks; use an explicitly authorized retry if eligible."
	}
	return finish()
}
