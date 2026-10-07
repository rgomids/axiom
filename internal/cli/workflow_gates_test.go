package cli

import (
	"bytes"
	"context"
	"reflect"
	"testing"
)

type gateRecordingService struct {
	recordingService
	input WorkflowInput
}

func (s *gateRecordingService) WorkflowAdvance(_ context.Context, input WorkflowInput) Result {
	s.input = input
	s.call = "advance"
	return Result{Status: Succeeded, Category: "applied"}
}

func TestAutomaticGateCLIForwardsExactInputAndRejectsConflicts(t *testing.T) {
	base := []string{"workflow", "advance", "--project", "sample", "--repository", "main", "--work-item", "github:owner/repo#7", "--execution", "018f4a44-7c31-7dd4-9d00-111111111111", "--expected-revision", "1", "--automatic"}
	service := &gateRecordingService{}
	var output bytes.Buffer
	if code := Run(context.Background(), base, service, completionProvenance(t), &output); code != ExitSuccess {
		t.Fatalf("code=%d output=%s", code, &output)
	}
	want := WorkflowInput{Automatic: true, Project: "sample", Repository: "main", WorkItem: "github:owner/repo#7", Provider: "github", ProviderRepository: "owner/repo", ExternalID: "7", Execution: "018f4a44-7c31-7dd4-9d00-111111111111", ExpectedRevision: 1}
	if !reflect.DeepEqual(service.input, want) {
		t.Fatalf("input = %#v", service.input)
	}
	for _, extra := range [][]string{{"--gate", "intake"}, {"--outcome", "pass"}, {"--reference", "evidence:path:hash"}, {"--next", "continue"}, {"--gate="}, {"--outcome="}, {"--reference="}, {"--next="}, {"--automatic"}, {"-automatic"}, {"--automatic=true"}} {
		service := &gateRecordingService{}
		output.Reset()
		args := append(append([]string{}, base...), extra...)
		if code := Run(context.Background(), args, service, completionProvenance(t), &output); code == ExitSuccess || service.call != "" {
			t.Fatalf("invalid %v dispatched: %d %s", extra, code, &output)
		}
	}
}

func TestWorkflowGateArgumentDiscoveryMatchesAutomaticAndExplicitModes(t *testing.T) {
	inspection, ok := inspectSkill("axiom-work-item-run")
	if !ok {
		t.Fatal("run skill is undiscoverable")
	}
	for _, command := range inspection.Commands {
		if command.Command != "axiom workflow advance" {
			continue
		}
		foundAutomatic, conditional := false, 0
		for _, arg := range command.Arguments {
			if arg.Name == "--automatic" {
				foundAutomatic = !arg.Required && arg.Description != ""
			}
			if arg.Name == "--gate" || arg.Name == "--outcome" {
				if arg.Required || arg.RequiredWhen != "--automatic is false or absent" {
					t.Fatalf("conditional argument = %#v", arg)
				}
				conditional++
			}
		}
		if !foundAutomatic || conditional != 2 {
			t.Fatalf("discovery = %#v", command)
		}
		if missingRequiredInputs(workflowAdvanceAction, requestInput{automatic: true}, "gate", "outcome") || !missingRequiredInputs(workflowAdvanceAction, requestInput{}, "gate", "outcome") {
			t.Fatal("discovery requirements diverge from automatic mode")
		}
		return
	}
	t.Fatal("advance command missing from discovery")
}
