package main

import (
	"bytes"
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/rgomids/axiom/internal/cli"
	"github.com/rgomids/axiom/internal/provenance"
	"github.com/rgomids/axiom/internal/workflow"
)

func gateService(t *testing.T) (lifecycleService, *gateStore) {
	t.Helper()
	source, err := provenance.FromBuild(provenance.Build{Version: provenance.Development, Revision: "abc123", SourceState: provenance.Clean}, nil)
	if err != nil {
		t.Fatal(err)
	}
	store := &gateStore{}
	s := workflow.New(gateResolver{}, gateWorkItems{}, store, nil, gateReferences{}, source, func() (string, error) { return "018f4a44-7c31-7dd4-9d00-111111111111", nil }, func() time.Time { return time.Date(2026, 10, 6, 12, 0, 0, 0, time.UTC) })
	return lifecycleService{workflows: s, provenance: source}, store
}

// Seed an existing Execution; runtime policy tests cover reviewed start.
func startGateFixture(service lifecycleService, input cli.WorkflowInput) cli.Result {
	return workflowResult(service.workflows.Start(context.Background(), workflowTarget(input)), service.provenance)
}

func TestWorkflowGateCommandConvergesAcrossCLIAndRuntime(t *testing.T) {
	for _, runtime := range []string{"codex", "claude"} {
		t.Run(runtime, func(t *testing.T) {
			cliService, cliStore := gateService(t)
			runtimeService, runtimeStore := gateService(t)
			input := cli.WorkflowInput{Project: "sample", Repository: "main", Provider: "github", ProviderRepository: "owner/repo", ExternalID: "7", Runtime: runtime}
			started := startGateFixture(cliService, input)
			startGateFixture(runtimeService, input)
			if started.Workflow.GateAction == nil || !started.Workflow.GateAction.Automatic || len(started.Workflow.GateCommand) == 0 {
				t.Fatalf("start = %#v", started)
			}
			var output bytes.Buffer
			command := started.Workflow.GateCommand
			if code := cli.RunInteractive(context.Background(), command[1:], cliService, cliService.provenance, nil, &output, nil); code != cli.ExitSuccess {
				t.Fatalf("CLI = %d %s", code, &output)
			}
			input.Runtime = ""
			input.Automatic = true
			input.Execution = started.Workflow.ExecutionID
			input.ExpectedRevision = 1
			direct := runtimeService.WorkflowAdvance(context.Background(), input)
			var rendered struct {
				Workflow *cli.WorkflowView `json:"workflow"`
			}
			if err := json.Unmarshal(output.Bytes(), &rendered); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(rendered.Workflow, direct.Workflow) || !reflect.DeepEqual(cliStore.state, runtimeStore.state) {
				t.Fatalf("CLI and Runtime diverged: %s %#v", &output, direct)
			}
			if direct.Workflow.CurrentGate != "specification" || direct.Workflow.GateAction.Automatic || direct.Workflow.Revision != 2 {
				t.Fatalf("unexpected automatic progression = %#v", direct.Workflow)
			}
		})
	}
}

func TestWorkflowFactCommandNamesExplicitAuthorityAndCondition(t *testing.T) {
	state := workflow.State{ProjectID: "project-id", RepositoryKey: "main", ExecutionID: "execution-id", Revision: 7, WorkItem: workflow.WorkItem{Provider: "github", Resource: "owner/repo", ExternalID: "7"}}
	for _, active := range []bool{true, false} {
		action := &workflow.GateAction{Operation: "fact", Fact: workflow.FactPlanningAuthority, Active: active, AuthorityRequired: true, ReferenceRequired: true}
		args := workflowGateCommand(state, action)
		wantActive := "--active=false"
		if active {
			wantActive = "--active=true"
		}
		for _, value := range []string{"planning-authority", wantActive, "--authorize-local", "<kind>:<reference>:<sha256>", "--execution", "execution-id", "--expected-revision", "7"} {
			found := false
			for _, arg := range args {
				if arg == value {
					found = true
				}
			}
			if !found {
				t.Fatalf("command missing %s: %v", value, args)
			}
		}
	}
	if workflowGateCommand(state, nil) != nil {
		t.Fatal("no action produced command")
	}
}

type gateResolver struct{}

type gateReferences struct{}

func (gateReferences) Validate(context.Context, string, string, workflow.Reference) error { return nil }

func TestReturnedGateCommandsCarryWholeJourneyAndRequireAuthority(t *testing.T) {
	service, store := gateService(t)
	input := cli.WorkflowInput{Project: "sample", Repository: "main", Provider: "github", ProviderRepository: "owner/repo", ExternalID: "7", Runtime: "codex"}
	result := startGateFixture(service, input)
	for step := 0; step < 20; step++ {
		action := result.Workflow.GateAction
		if action == nil {
			if result.Workflow.LifecycleStage != "accepted" || store.state.Status != workflow.ExecutionCompleted {
				t.Fatalf("journey stopped early: %#v", result.Workflow)
			}
			return
		}
		command := append([]string(nil), result.Workflow.GateCommand...)
		kind := "evidence"
		if action.Fact == workflow.FactPlanningAuthority {
			kind = "specification"
		}
		if action.Fact == workflow.FactImplementationAuthority {
			kind = "plan"
		}
		for index, arg := range command {
			if arg == "<pass-or-fail>" {
				command[index] = "pass"
			}
			if arg == "<kind>:<reference>:<sha256>" {
				command[index] = kind + ":docs/result.md:" + strings.Repeat("0", 64)
			}
		}
		var output bytes.Buffer
		if action.AuthorityRequired {
			before := store.state
			denied := append([]string(nil), command[:len(command)-1]...)
			if code := cli.RunInteractive(context.Background(), denied[1:], service, service.provenance, nil, &output, nil); code == cli.ExitSuccess || !reflect.DeepEqual(before, store.state) {
				t.Fatalf("authority bypass: %d %s", code, &output)
			}
			output.Reset()
		}
		if code := cli.RunInteractive(context.Background(), command[1:], service, service.provenance, nil, &output, nil); code != cli.ExitSuccess {
			t.Fatalf("step %d = %d %s", step, code, &output)
		}
		input.Runtime = ""
		input.Execution = store.state.ExecutionID
		before := store.state
		result = service.WorkflowStatus(context.Background(), input)
		if !reflect.DeepEqual(before, store.state) {
			t.Fatal("status mutated state")
		}
	}
	t.Fatal("gate journey did not terminate")
}

func (gateResolver) Resolve(context.Context, string) (workflow.Project, string) {
	return workflow.Project{ID: "123e4567-e89b-42d3-a456-426614174000", Repositories: []workflow.Repository{{Key: "main", Path: "/repository"}}}, ""
}

type gateWorkItems struct{}

func (gateWorkItems) Load(context.Context, string, string, string, string, string) (workflow.WorkItem, error) {
	return workflow.WorkItem{Provider: "github", Resource: "owner/repo", ExternalID: "7", URL: "https://github.com/owner/repo/issues/7", State: "OPEN"}, nil
}

type gateStore struct {
	state  workflow.State
	exists bool
}

func (s *gateStore) Create(_ context.Context, state workflow.State) error {
	s.state = state
	s.exists = true
	return nil
}
func (s *gateStore) Load(context.Context, string, string, workflow.WorkItem) (workflow.State, error) {
	if !s.exists {
		return workflow.State{}, workflow.ErrNotFound
	}
	return s.state, nil
}
func (s *gateStore) Save(_ context.Context, state workflow.State) error { s.state = state; return nil }
