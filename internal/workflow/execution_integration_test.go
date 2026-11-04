package workflow

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func sha(c string) string { return strings.Repeat(c, 40) }

func integrationExecution(t *testing.T, gate bool) Execution {
	t.Helper()
	effects := []string{"workspace_write", "validation_command"}
	e, err := NewExecution(NewExecutionInput{
		ExecutionID: testExecutionID, ProjectID: "proj", WorkItem: WorkItemLink{"linear", "AXM-42", "app"},
		Repositories: []RepositoryBinding{{"app", "github.com/example/app"}, {"lib", "github.com/example/lib"}},
		WorkflowID:   "parallel", Revision: "r1", Now: t0,
		Stages: []StageSpec{{
			ID: "work", HumanGate: gate, TimeoutSeconds: 60,
			Agents:   []AgentSpec{{ID: "a", Effects: effects}, {ID: "b", Effects: effects}, {ID: "c", Effects: []string{"workspace_write"}}},
			Commands: []CommandCheck{{ID: "check", Argv: []string{"/bin/true"}, TimeoutSeconds: 10, Effects: []string{"validation_command"}}},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := e.ClaimRunner(1, t0); err != nil {
		t.Fatal(err)
	}
	return e
}

func succeedWithCheck(t *testing.T, e *Execution, agent, id string, effects ...string) {
	t.Helper()
	ref := TaskRef{"work", agent}
	run(t, e, ref, id, effects...)
	gen := e.RunnerGeneration
	if err := e.PrepareCommand(id, gen, "check", t0); err != nil {
		t.Fatal(err)
	}
	if err := e.StartCommand(id, gen, "check", ProcessIdentity{PID: 5151, StartIdentity: "s", GroupIdentity: "5151"}, t0); err != nil {
		t.Fatal(err)
	}
	if err := e.FinishCommand(id, gen, "check", 0, t0); err != nil {
		t.Fatal(err)
	}
	a, _, _ := e.FindAttempt(id)
	if err := e.CompleteAttempt(id, gen, AttemptResult{Commands: append([]CommandResult(nil), a.Commands...), Now: t0}); err != nil {
		t.Fatal(err)
	}
	if a, _, _ := e.FindAttempt(id); a.Status != AttemptSucceeded {
		t.Fatalf("fixture attempt %s is %s", id, a.Status)
	}
}

func intent(e Execution, id, attempt string) IntegrationInput {
	return IntegrationInput{
		IntegrationID: id, Generation: e.RunnerGeneration, SourceAttemptID: attempt,
		RepositoryKey: "app", RepositoryIdentity: "github.com/example/app",
		SourceBranch: "refs/heads/axiom/agent-" + attempt, Worktree: "agent-" + attempt,
		BaseSHA: sha("b"), TargetRef: "refs/heads/axiom/integration", ExpectedTargetHead: sha("b"), CandidateSHA: sha("c"), Now: t0,
	}
}

func passed() []IntegrationCheck { return []IntegrationCheck{{ID: "check", Status: CheckPassed}} }

func TestIntegrationRecordLifecycleAndCodec(t *testing.T) {
	e := integrationExecution(t, false)
	succeedWithCheck(t, &e, "a", "att-a", "workspace_write", "validation_command")
	succeedWithCheck(t, &e, "b", "att-b", "workspace_write", "validation_command")
	if err := e.PrepareIntegration(intent(e, "int-a", "att-a")); err != nil {
		t.Fatal(err)
	}
	// One prepared integration at a time: the principal serializes.
	second := intent(e, "int-b", "att-b")
	second.CandidateSHA = sha("d")
	if err := e.PrepareIntegration(second); !errors.Is(err, ErrIntegrationBusy) {
		t.Fatalf("second concurrent intent accepted: %v", err)
	}
	// integrated requires passed checks and the target observed at FinalSHA.
	for _, bad := range []IntegrationResult{
		{Status: IntegrationIntegrated, MergeSHA: sha("e"), FinalSHA: sha("e"), ObservedTargetHead: sha("e"), Now: t0},
		{Status: IntegrationIntegrated, MergeSHA: sha("e"), FinalSHA: sha("e"), ObservedTargetHead: sha("e"), Checks: []IntegrationCheck{{ID: "check", Status: CheckFailed, ExitCode: 1}}, Now: t0},
		{Status: IntegrationIntegrated, MergeSHA: sha("e"), FinalSHA: sha("e"), ObservedTargetHead: sha("b"), Checks: passed(), Now: t0},
		{Status: IntegrationIntegrated, MergeSHA: sha("e"), FinalSHA: sha("f"), ObservedTargetHead: sha("f"), Checks: passed(), Now: t0},
		{Status: IntegrationNotIntegrated, MergeSHA: sha("e"), FinalSHA: sha("e"), Diagnostic: "x", Now: t0},
		{Status: IntegrationNotIntegrated, Now: t0},
		{Status: IntegrationPrepared, Diagnostic: "x", Now: t0},
	} {
		probe := e
		probe.Integrations = append([]Integration(nil), e.Integrations...)
		if err := probe.FinishIntegration("int-a", e.RunnerGeneration, bad); err == nil {
			t.Fatalf("invalid outcome accepted: %+v", bad)
		}
	}
	if err := e.FinishIntegration("int-a", e.RunnerGeneration, IntegrationResult{Status: IntegrationIntegrated, MergeSHA: sha("e"), FinalSHA: sha("e"), ObservedTargetHead: sha("e"), Checks: passed(), Now: t0}); err != nil {
		t.Fatal(err)
	}
	if got := e.AcceptedTargetHead("app", "refs/heads/axiom/integration"); got != sha("e") {
		t.Fatalf("accepted head %s", got)
	}
	// The recorded branch/worktree/attempt association cannot be reused for
	// another attempt.
	for _, mismatch := range []func(*IntegrationInput){
		func(in *IntegrationInput) { in.SourceBranch = "refs/heads/axiom/agent-att-a" },
		func(in *IntegrationInput) { in.Worktree = "agent-att-a" },
	} {
		in := second
		in.ExpectedTargetHead = sha("e")
		mismatch(&in)
		if err := e.PrepareIntegration(in); !errors.Is(err, ErrTransition) {
			t.Fatalf("association mismatch accepted: %v", err)
		}
	}
	// Lineage: the next integration must start from the recorded final SHA.
	if err := e.PrepareIntegration(second); !errors.Is(err, ErrIntegrationStale) {
		t.Fatalf("stale expected head accepted: %v", err)
	}
	second.ExpectedTargetHead = sha("e")
	if err := e.PrepareIntegration(second); err != nil {
		t.Fatal(err)
	}
	if err := e.FinishIntegration("int-b", e.RunnerGeneration, IntegrationResult{Status: IntegrationNotIntegrated, MergeSHA: sha("f"), ObservedTargetHead: sha("e"), Checks: []IntegrationCheck{{ID: "check", Status: CheckFailed, ExitCode: 3}}, Diagnostic: "post_merge_check_failed", Now: t0}); err != nil {
		t.Fatal(err)
	}
	if e.Status == RunRecoveryRequired || e.AcceptedTargetHead("app", "refs/heads/axiom/integration") != sha("e") {
		t.Fatal("failed check changed the accepted lineage")
	}
	wire, err := EncodeExecution(e)
	if err != nil {
		t.Fatal(err)
	}
	back, err := DecodeExecution(wire)
	if err != nil {
		t.Fatal(err)
	}
	if again, _ := EncodeExecution(back); !bytes.Equal(wire, again) {
		t.Fatal("integration codec is not canonical")
	}
	if tampered := bytes.Replace(wire, []byte(`"status":"not_integrated"`), []byte(`"status":"integrated"`), 1); bytes.Equal(tampered, wire) {
		t.Fatal("fixture lacks the not_integrated record")
	} else if _, err := DecodeExecution(tampered); !errors.Is(err, ErrExecutionInvalid) {
		t.Fatalf("forged integrated status decoded: %v", err)
	}
	// Executions without integrations keep their exact bytes.
	plain := integrationExecution(t, false)
	if wire, _ := EncodeExecution(plain); bytes.Contains(wire, []byte("integrations")) {
		t.Fatal("empty integrations are encoded")
	}
}

func TestIntegrationIntentRefusals(t *testing.T) {
	e := integrationExecution(t, false)
	succeedWithCheck(t, &e, "a", "att-a", "workspace_write", "validation_command")
	run(t, &e, TaskRef{"work", "c"}, "att-c", "workspace_write")
	cases := map[string]struct {
		mutate func(*IntegrationInput)
		want   error
	}{
		"stale fence":            {func(in *IntegrationInput) { in.Generation = 7 }, ErrStaleRunner},
		"child without fence":    {func(in *IntegrationInput) { in.Generation = 0 }, ErrStaleRunner},
		"unknown attempt":        {func(in *IntegrationInput) { in.SourceAttemptID = "att-x" }, ErrTransition},
		"running attempt":        {func(in *IntegrationInput) { in.SourceAttemptID = "att-c" }, ErrTransition},
		"wrong repository key":   {func(in *IntegrationInput) { in.RepositoryKey, in.RepositoryIdentity = "lib", "github.com/example/lib" }, ErrUnauthorizedEffect},
		"changed identity":       {func(in *IntegrationInput) { in.RepositoryIdentity = "github.com/evil/app" }, ErrTransition},
		"missing candidate":      {func(in *IntegrationInput) { in.CandidateSHA = "" }, ErrTransition},
		"abbreviated candidate":  {func(in *IntegrationInput) { in.CandidateSHA = "cccccc" }, ErrTransition},
		"uppercase candidate":    {func(in *IntegrationInput) { in.CandidateSHA = strings.Repeat("C", 40) }, ErrTransition},
		"candidate equals base":  {func(in *IntegrationInput) { in.CandidateSHA = in.BaseSHA }, ErrTransition},
		"target is source":       {func(in *IntegrationInput) { in.TargetRef = in.SourceBranch }, ErrTransition},
		"tag target":             {func(in *IntegrationInput) { in.TargetRef = "refs/tags/v1" }, ErrTransition},
		"traversing ref":         {func(in *IntegrationInput) { in.TargetRef = "refs/heads/../x" }, ErrTransition},
		"lock ref":               {func(in *IntegrationInput) { in.TargetRef = "refs/heads/main.lock" }, ErrTransition},
		"escaping worktree":      {func(in *IntegrationInput) { in.Worktree = "../agent" }, ErrTransition},
		"hidden worktree":        {func(in *IntegrationInput) { in.Worktree = ".git" }, ErrTransition},
		"nested worktree":        {func(in *IntegrationInput) { in.Worktree = "a/b" }, ErrTransition},
		"invalid integration id": {func(in *IntegrationInput) { in.IntegrationID = "a/b" }, ErrTransition},
	}
	for name, c := range cases {
		in := intent(e, "int-1", "att-a")
		c.mutate(&in)
		before, _ := EncodeExecution(e)
		if err := e.PrepareIntegration(in); !errors.Is(err, c.want) {
			t.Errorf("%s: got %v, want %v", name, err, c.want)
		}
		if after, _ := EncodeExecution(e); !bytes.Equal(before, after) {
			t.Fatalf("%s: refusal changed the Execution", name)
		}
	}
	// A source without validation authority cannot run its stage check, so it
	// never has a validated result to integrate.
	if err := e.PrepareCommand("att-c", e.RunnerGeneration, "check", t0); !errors.Is(err, ErrUnauthorizedEffect) {
		t.Fatalf("unauthorized check: %v", err)
	}
}

func TestIntegrationGateCancelUnknownAndChecksProhibitEffect(t *testing.T) {
	gated := integrationExecution(t, true)
	succeedWithCheck(t, &gated, "a", "att-a", "workspace_write", "validation_command")
	if err := gated.PrepareIntegration(intent(gated, "int-a", "att-a")); !errors.Is(err, ErrTransition) {
		t.Fatalf("pending human gate permitted integration: %v", err)
	}
	if err := gated.ApproveGate("work", "agent:a", t0); err == nil {
		t.Fatal("agent approved a human gate")
	}

	cancelled := integrationExecution(t, false)
	succeedWithCheck(t, &cancelled, "a", "att-a", "workspace_write", "validation_command")
	if err := cancelled.RequestCancel("human:op", t0); err != nil {
		t.Fatal(err)
	}
	if err := cancelled.PrepareIntegration(intent(cancelled, "int-a", "att-a")); !errors.Is(err, ErrCancelled) {
		t.Fatalf("cancel permitted integration: %v", err)
	}

	e := integrationExecution(t, false)
	succeedWithCheck(t, &e, "a", "att-a", "workspace_write", "validation_command")
	succeedWithCheck(t, &e, "b", "att-b", "workspace_write", "validation_command")
	if err := e.PrepareIntegration(intent(e, "int-a", "att-a")); err != nil {
		t.Fatal(err)
	}
	// A lost runner leaves the intent prepared; the next fence records unknown.
	if err := e.ClaimRunner(2, t0); err != nil {
		t.Fatal(err)
	}
	if err := e.FinishIntegration("int-a", 2, IntegrationResult{Status: IntegrationIntegrated, MergeSHA: sha("e"), FinalSHA: sha("e"), ObservedTargetHead: sha("e"), Checks: passed(), Now: t0}); !errors.Is(err, ErrTransition) {
		t.Fatalf("new runner finished an older runner's integration: %v", err)
	}
	if err := e.FinishIntegration("int-a", 1, IntegrationResult{Status: IntegrationIntegrated, MergeSHA: sha("e"), FinalSHA: sha("e"), ObservedTargetHead: sha("e"), Checks: passed(), Now: t0}); !errors.Is(err, ErrStaleRunner) {
		t.Fatalf("stale runner finished: %v", err)
	}
	if err := e.MarkIntegrationUnknown("int-a", 2, "runner_lost_before_integration_outcome", t0); err != nil {
		t.Fatal(err)
	}
	if e.Status != RunRecoveryRequired || len(e.Ready()) != 0 {
		t.Fatalf("unknown integration did not block: %s", e.Status)
	}
	next := intent(e, "int-b", "att-b")
	if err := e.PrepareIntegration(next); !errors.Is(err, ErrUnknownOutstanding) {
		t.Fatalf("integration after unknown outcome: %v", err)
	}
	if err := e.ResolveIntegration("int-a", "agent:a", t0); !errors.Is(err, ErrTransition) {
		t.Fatalf("agent resolved an unknown integration: %v", err)
	}
	if err := e.ResolveIntegration("int-a", "human:op", t0); err != nil {
		t.Fatal(err)
	}
	if i := e.integration("int-a"); i.Status != IntegrationUnknown || i.FinalSHA != "" {
		t.Fatal("abandonment rewrote the uncertain outcome")
	}
	next.Generation = e.RunnerGeneration
	if err := e.PrepareIntegration(next); err != nil {
		t.Fatalf("explicit new integration after human disposition: %v", err)
	}

	// No post-merge check declared: no check Evidence, so no integration.
	n, err := NewExecution(NewExecutionInput{
		ExecutionID: testExecutionID, ProjectID: "proj", WorkItem: WorkItemLink{"linear", "AXM-42", "app"},
		Repositories: []RepositoryBinding{{"app", "github.com/example/app"}}, WorkflowID: "w", Revision: "r1", Now: t0,
		Stages: []StageSpec{{ID: "work", TimeoutSeconds: 60, Agents: []AgentSpec{{ID: "a", Effects: []string{"workspace_write", "validation_command"}}}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	_ = n.ClaimRunner(1, t0)
	run(t, &n, TaskRef{"work", "a"}, "att-a", "workspace_write", "validation_command")
	if err := n.CompleteAttempt("att-a", 1, AttemptResult{Now: t0}); err != nil {
		t.Fatal(err)
	}
	if err := n.PrepareIntegration(intent(n, "int-a", "att-a")); !errors.Is(err, ErrIntegrationChecksUnavailable) {
		t.Fatalf("integration without checks: %v", err)
	}
}

func TestIntegrationMessagesGrantNothing(t *testing.T) {
	e := integrationExecution(t, false)
	succeedWithCheck(t, &e, "a", "att-a", "workspace_write", "validation_command")
	run(t, &e, TaskRef{"work", "b"}, "att-b", "workspace_write", "validation_command")
	in := IngressContext{testExecutionID, "att-b", "b", "stub", e.RunnerGeneration}
	// A child result message claiming an integration is only a message.
	if err := e.AcceptMessage(in, MessageCandidate{ID: "m1", Recipient: "a", Kind: MessageResult, Content: `{"integrated":"` + sha("e") + `","status":"integrated"}`}, t0); err != nil {
		t.Fatal(err)
	}
	if len(e.Integrations) != 0 || e.AcceptedTargetHead("app", "refs/heads/axiom/integration") != "" {
		t.Fatal("message created integration state")
	}
}

func TestIntegrationUnknownCheckCodecAndDisposition(t *testing.T) {
	for _, code := range []int{-1, 0} {
		e := integrationExecution(t, false)
		succeedWithCheck(t, &e, "a", "att-a", "workspace_write", "validation_command")
		if err := e.PrepareIntegration(intent(e, "int-a", "att-a")); err != nil {
			t.Fatal(err)
		}
		if err := e.FinishIntegration("int-a", e.RunnerGeneration, IntegrationResult{Status: IntegrationUnknown, MergeSHA: sha("e"), Checks: []IntegrationCheck{{ID: "check", Status: CheckUnknown, ExitCode: code}}, Diagnostic: "post_merge_check_unknown", Now: t0}); err != nil {
			t.Fatal(err)
		}
		wire, err := EncodeExecution(e)
		if err != nil {
			t.Fatal(err)
		}
		got, err := DecodeExecution(wire)
		if err != nil {
			t.Fatal(err)
		}
		if got.Status != RunRecoveryRequired || got.Integrations[0].Checks[0].ExitCode != code {
			t.Fatal("unknown check evidence lost")
		}
		if err := got.PrepareIntegration(intent(got, "int-b", "att-a")); !errors.Is(err, ErrUnknownOutstanding) {
			t.Fatalf("unknown did not block: %v", err)
		}
		if err := got.ResolveIntegration("int-a", "human:maintainer", t0); err != nil {
			t.Fatal(err)
		}
		if got.Integrations[0].Status != IntegrationUnknown || got.Integrations[0].Checks[0].Status != CheckUnknown {
			t.Fatal("disposition rewrote evidence")
		}
		got.Integrations[0].Status = IntegrationNotIntegrated
		if err := got.Validate(); err == nil {
			t.Fatal("unknown check downgraded to known refusal")
		}
	}
}
