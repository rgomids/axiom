//go:build linux || darwin

package main

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/cli"
	"github.com/rgomids/axiom/internal/workflow"
	"github.com/rgomids/axiom/internal/workflowdefinition"
)

func i404DAGDefinition() workflowdefinition.Definition {
	d := criterionWorkflow(false)
	d.WorkflowID, d.Name = "i404-dag", "Synthetic Stage capacity"
	d.Stages = d.Stages[:1]
	s := &d.Stages[0]
	s.Mode, s.Concurrency = "dag", 2
	s.Outputs = []workflowdefinition.Output{{ID: "result", Kind: "result", Required: true}}
	seed := s.Agents[0]
	s.Agents = nil
	for _, id := range []string{"a", "b", "third", "owner"} {
		a := seed
		a.ID, a.ProfileRef = id, ""
		a.Inputs = append([]string(nil), seed.Inputs...)
		a.DependsOn = []string{}
		a.Outputs, a.EffectCeilings = []string{"result"}, []string{"read"}
		a.RuntimeConstraints = []string{"codex"}
		a.Invocation = &workflowdefinition.AgentInvocation{Runtime: "codex", Model: "stub-model", Effort: "runtime-default"}
		a.IntegrationOwner, a.ValidationOwner = id == "owner", id == "owner"
		if id == "owner" {
			a.DependsOn = []string{"a", "b", "third"}
		}
		s.Agents = append(s.Agents, a)
	}
	return d
}

func TestI404MapperPreservesDeclaredCapacityAndEdges(t *testing.T) {
	for _, mode := range []string{"dag2", "dag1", "sequential1"} {
		t.Run(mode, func(t *testing.T) {
			d := i404DAGDefinition()
			if mode != "dag2" {
				d.Stages[0].Concurrency = 1
			}
			if mode == "sequential1" {
				d.Stages[0].Mode = "sequential"
				for i := 1; i < len(d.Stages[0].Agents); i++ {
					d.Stages[0].Agents[i].DependsOn = []string{d.Stages[0].Agents[i-1].ID}
				}
			}
			second := d.Stages[0]
			second.ID, second.Concurrency = "second", 1
			second.Phase = "specification"
			d.Stages = append(d.Stages, second)
			_, _, stages, err := mapWorkflowDefinition(d)
			if err != nil || len(stages) != 2 {
				t.Fatal("valid declared capacity refused", err)
			}
			for i, stage := range stages {
				if stage.Concurrency == nil || *stage.Concurrency != d.Stages[i].Concurrency {
					t.Fatal("capacity lost or narrowed", stage.Concurrency)
				}
				for j, agent := range stage.Agents {
					if !reflect.DeepEqual(agent.DependsOn, d.Stages[i].Agents[j].DependsOn) && !(len(agent.DependsOn) == 0 && len(d.Stages[i].Agents[j].DependsOn) == 0) {
						t.Fatal("synthetic or lost DAG edge", agent)
					}
				}
			}
			if stages[0].Concurrency == stages[1].Concurrency || len(stages[0].DependsOn) != 0 || !reflect.DeepEqual(stages[1].DependsOn, []string{stages[0].ID}) {
				t.Fatal("stage order/independent pointer mapping changed")
			}
			*stages[0].Concurrency = 3
			if *stages[1].Concurrency != 1 || d.Stages[0].Concurrency == 3 {
				t.Fatal("mapped pointers alias another stage or definition")
			}
		})
	}
}

func TestI404MapperStillRejectsInvalidDefinitions(t *testing.T) {
	if _, diagnostics := workflowdefinition.Encode(i404DAGDefinition()); len(diagnostics) != 0 {
		t.Fatal("base definition invalid", diagnostics)
	}
	for name, mutate := range map[string]func(*workflowdefinition.Stage){
		"zero":          func(s *workflowdefinition.Stage) { s.Concurrency = 0 },
		"negative":      func(s *workflowdefinition.Stage) { s.Concurrency = -1 },
		"over-schema":   func(s *workflowdefinition.Stage) { s.Concurrency = 33 },
		"over-agents":   func(s *workflowdefinition.Stage) { s.Concurrency = 5 },
		"sequential2":   func(s *workflowdefinition.Stage) { s.Mode = "sequential" },
		"cycle":         func(s *workflowdefinition.Stage) { s.Agents[0].DependsOn = []string{"owner"} },
		"dangling":      func(s *workflowdefinition.Stage) { s.Agents[0].DependsOn = []string{"missing"} },
		"missing-owner": func(s *workflowdefinition.Stage) { s.Agents[3].IntegrationOwner = false },
		"invalid-owner": func(s *workflowdefinition.Stage) { s.Agents[3].DependsOn = []string{"a"} },
	} {
		t.Run(name, func(t *testing.T) {
			d := i404DAGDefinition()
			mutate(&d.Stages[0])
			if _, diagnostics := workflowdefinition.Encode(d); len(diagnostics) == 0 {
				t.Fatal("invalid fixture passed definition validation")
			}
			if _, _, stages, err := mapWorkflowDefinition(d); !errors.Is(err, errWorkflowUnmappable) || len(stages) != 0 {
				t.Fatal("invalid definition mapped", err)
			}
		})
	}
}

func TestI404PublicRunCreatesOneCanonicalDAGExecution(t *testing.T) {
	f := newExecutionRunFixture(t)
	d := i404DAGDefinition()
	f.installWorkflowDefinition(t, d)
	got := f.run(t, cli.ExitSuccess, "workflow", "run", "--project", "external", "--repository", "main", "--work-item", "github:owner/repo#7", "--authorize-external")
	if got.Category != "execution_completed" || got.Stage.Status != workflow.RunCompleted || len(f.executions(t)) != 1 {
		t.Fatalf("DAG2 failed canonical public start: %s", got.Raw)
	}
	wire, err := os.ReadFile(f.executions(t)[0])
	if err != nil {
		t.Fatal(err)
	}
	x, err := workflow.DecodeExecution(wire)
	if err != nil || x.Workflow.Stages[0].Concurrency == nil || *x.Workflow.Stages[0].Concurrency != 2 {
		t.Fatal("public run lost capacity", err)
	}
	for _, task := range x.Stages[0].Tasks {
		if len(task.Attempts) != 1 || task.Attempts[0].Status != workflow.AttemptSucceeded || task.Attempts[0].Process == nil {
			t.Fatal("stub did not actually start and complete", task)
		}
	}
	if !reflect.DeepEqual(x.Workflow.Stages[0].Agents[3].DependsOn, []string{"a", "b", "third"}) {
		t.Fatal("public DAG edges changed")
	}
}

func TestI404PublicRunIndependentUnsupportedControlsRefuseBeforeStart(t *testing.T) {
	for _, mode := range []string{"combined-instructions", "required-context"} {
		t.Run(mode, func(t *testing.T) {
			f := newExecutionRunFixture(t)
			d := i404DAGDefinition()
			if mode == "combined-instructions" {
				d.Stages[0].Validators = []workflowdefinition.Validator{{ID: "check", Kind: "artifact-schema", PolicyRef: "builtin-sdd-v1"}}
				d.Stages[0].Instructions = strings.Repeat("a", 9000)
				d.Stages[0].Agents[0].Responsibilities = strings.Repeat("b", 9000)
			} else {
				d.Stages[0].Inputs = append(d.Stages[0].Inputs, workflowdefinition.Input{ID: "missing-context", Kind: "project-context", Source: "business-context", Required: true})
				d.Stages[0].Agents[0].Inputs = []string{"missing-context"}
			}
			f.installWorkflowDefinition(t, d)
			marker := filepath.Join(resolvedTempDir(t), "spawned")
			bin := filepath.Join(os.Getenv("PATH"), "codex")
			if err := os.WriteFile(bin, []byte(strings.Replace(stubRuntimeScript, "#!/bin/sh\n", "#!/bin/sh\nprintf 'started\\n' >> '"+marker+"'\n", 1)), 0700); err != nil {
				t.Fatal(err)
			}
			got := f.run(t, cli.ExitFailure, "workflow", "run", "--project", "external", "--repository", "main", "--work-item", "github:owner/repo#7", "--authorize-external")
			wantCategory := "workflow_unmappable"
			if mode == "required-context" {
				wantCategory = "stage_prerequisite_missing"
			}
			if got.Category != wantCategory || len(f.executions(t)) != 0 {
				t.Fatal("unsupported control reached Execution creation", got.Raw)
			}
			if _, err := os.Stat(marker); !os.IsNotExist(err) {
				t.Fatal("unsupported control spawned stub", err)
			}
		})
	}
}

// Unknown policy references are already refused by Project authoring. Exercise
// the mapper independently so a valid DAG ceiling cannot erase that refusal.
func TestI404MapperUnsupportedValidatorRemainsRefused(t *testing.T) {
	d := i404DAGDefinition()
	d.Stages[0].Validators = []workflowdefinition.Validator{{ID: "unavailable", Kind: "artifact-schema", PolicyRef: "unavailable-policy"}}
	if _, diagnostics := workflowdefinition.Encode(d); len(diagnostics) != 0 {
		t.Fatal(diagnostics)
	}
	if _, _, stages, err := mapWorkflowDefinition(d); !errors.Is(err, errWorkflowUnmappable) || len(stages) != 0 {
		t.Fatal("unsupported validator was silently dropped", err)
	}
}
