package workflow

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"
)

var t0 = time.Date(2026, 10, 31, 12, 0, 0, 0, time.UTC)

const testExecutionID = "00000000-0000-4000-8000-000000000042"

func testInvocation(instructions string) Invocation {
	prov := []FieldProvenance{{"runtime", "local_default", ""}, {"model", "workflow_agent", ""}, {"effort", "run_override", ""}, {"instructions", "workflow_agent", ""}}
	return Invocation{Runtime: "stub", Executable: "stub-bin", Model: "m1", Effort: "runtime-default", Instructions: instructions, InstructionsDigest: DigestText([]byte(instructions)), Provenance: prov}
}

func twoStageExecution(t *testing.T, gate bool) Execution {
	t.Helper()
	e, err := NewExecution(NewExecutionInput{
		ExecutionID: testExecutionID, ProjectID: "proj", WorkItem: WorkItemLink{"linear", "AXM-42", "app"},
		Repositories: []RepositoryBinding{{"app", "github.com/example/app"}},
		WorkflowID:   "minimal", Revision: "r1", Now: t0,
		Stages: []StageSpec{
			{ID: "work", Agents: []AgentSpec{{ID: "builder", Effects: []string{"workspace_write"}}}, Outputs: []string{"result"}, HumanGate: gate, TimeoutSeconds: 60},
			{ID: "review", DependsOn: []string{"work"}, Agents: []AgentSpec{{ID: "reviewer"}}, TimeoutSeconds: 60},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := e.ClaimRunner(1, t0); err != nil {
		t.Fatal(err)
	}
	return e
}

func prepare(t *testing.T, e *Execution, ref TaskRef, id string, effects ...string) {
	t.Helper()
	if effects == nil {
		effects = []string{}
	}
	if err := e.PrepareAttempt(PrepareInput{Ref: ref, AttemptID: id, Generation: e.RunnerGeneration, Invocation: testInvocation("do"), Authority: AttemptAuthority{"app", effects, "user:local"}, Now: t0}); err != nil {
		t.Fatal(err)
	}
}

func run(t *testing.T, e *Execution, ref TaskRef, id string, effects ...string) {
	t.Helper()
	prepare(t, e, ref, id, effects...)
	if err := e.MarkStarted(id, e.RunnerGeneration, ProcessIdentity{PID: 4242, StartIdentity: "s", GroupIdentity: "4242"}, t0); err != nil {
		t.Fatal(err)
	}
}

func artifact(name string) ArtifactRef {
	return ArtifactRef{Name: name, RepositoryKey: "app", Digest: DigestText([]byte(name)), Size: 1}
}

func TestExecutionCodecRoundTripAndStrictness(t *testing.T) {
	e := twoStageExecution(t, false)
	wire, err := EncodeExecution(e)
	if err != nil {
		t.Fatal(err)
	}
	back, err := DecodeExecution(wire)
	if err != nil {
		t.Fatal(err)
	}
	again, _ := EncodeExecution(back)
	if !bytes.Equal(wire, again) {
		t.Fatal("codec is not canonical")
	}
	unknown := bytes.Replace(wire, []byte(`"revision":`), []byte(`"extra":1,"revision":`), 1)
	if _, err := DecodeExecution(unknown); !errors.Is(err, ErrExecutionInvalid) {
		t.Fatalf("unknown member accepted: %v", err)
	}
	alias := bytes.Replace(wire, []byte(`"projectId"`), []byte(`"ProjectID"`), 1)
	if _, err := DecodeExecution(alias); !errors.Is(err, ErrExecutionInvalid) {
		t.Fatalf("case alias accepted: %v", err)
	}
	if _, err := DecodeExecution(append(wire, bytes.Repeat([]byte(" "), MaxExecutionBytes)...)); !errors.Is(err, ErrExecutionInvalid) {
		t.Fatalf("oversize accepted: %v", err)
	}
	tampered := bytes.Replace(wire, []byte(`"timeoutSeconds":60`), []byte(`"timeoutSeconds":61`), 1)
	if _, err := DecodeExecution(tampered); !errors.Is(err, ErrExecutionInvalid) {
		t.Fatalf("workflow edit without digest accepted: %v", err)
	}
	forgedStatus := bytes.Replace(wire, []byte(`"status":"ready"`), []byte(`"status":"completed"`), 1)
	if _, err := DecodeExecution(forgedStatus); !errors.Is(err, ErrExecutionInvalid) {
		t.Fatalf("status not derived from facts accepted: %v", err)
	}
}

func TestExecutionRefusesLegacyFormats(t *testing.T) {
	for _, wire := range []string{
		`{"formatVersion":3,"executionId":"x"}`,
		`{"formatVersion":5,"executionId":"x"}`,
		`{"kind":"single_stage_run","formatVersion":2}`,
		`{"kind":"graph","formatVersion":4}`,
		`{"kind":"execution","formatVersion":1}`,
	} {
		if _, err := DecodeExecution([]byte(wire)); !errors.Is(err, ErrLegacyExecution) {
			t.Fatalf("%s: got %v, want legacy refusal", wire, err)
		}
	}
	if _, err := DecodeExecution([]byte(`{"kind":"execution","formatVersion":5}`)); !errors.Is(err, ErrExecutionInvalid) || errors.Is(err, ErrLegacyExecution) {
		t.Fatalf("future version: %v", err)
	}
}

func TestExecutionRefusesInvalidWorkflowBeforeAnyAttempt(t *testing.T) {
	base := NewExecutionInput{ExecutionID: testExecutionID, ProjectID: "p", WorkItem: WorkItemLink{"linear", "W-1", "app"}, Repositories: []RepositoryBinding{{"app", "id"}}, WorkflowID: "w", Revision: "r", Now: t0}
	cases := map[string][]StageSpec{
		"cycle":       {{ID: "a", DependsOn: []string{"b"}, Agents: []AgentSpec{{ID: "x"}}, TimeoutSeconds: 1}, {ID: "b", DependsOn: []string{"a"}, Agents: []AgentSpec{{ID: "y"}}, TimeoutSeconds: 1}},
		"unknown dep": {{ID: "a", DependsOn: []string{"zzz"}, Agents: []AgentSpec{{ID: "x"}}, TimeoutSeconds: 1}},
		"no agents":   {{ID: "a", TimeoutSeconds: 1}},
		"bad effect":  {{ID: "a", Agents: []AgentSpec{{ID: "x", Effects: []string{"merge"}}}, TimeoutSeconds: 1}},
		"timeout":     {{ID: "a", Agents: []AgentSpec{{ID: "x"}}, TimeoutSeconds: 0}},
	}
	for name, stages := range cases {
		in := base
		in.Stages = stages
		if _, err := NewExecution(in); !errors.Is(err, ErrExecutionInvalid) {
			t.Errorf("%s: %v", name, err)
		}
	}
	in := base
	in.WorkItem.RepositoryKey = "other"
	in.Stages = []StageSpec{{ID: "a", Agents: []AgentSpec{{ID: "x"}}, TimeoutSeconds: 1}}
	if _, err := NewExecution(in); !errors.Is(err, ErrExecutionInvalid) {
		t.Errorf("work item linked to an unbound repository accepted: %v", err)
	}
}

func TestTwoSequentialStagesProgressAutomatically(t *testing.T) {
	e := twoStageExecution(t, false)
	if got := e.Ready(); len(got) != 1 || got[0] != (TaskRef{"work", "builder"}) {
		t.Fatalf("ready = %v", got)
	}
	run(t, &e, TaskRef{"work", "builder"}, "a1", "workspace_write")
	if len(e.Ready()) != 0 || e.Status != RunRunning {
		t.Fatalf("successor eligible while work runs: %v %s", e.Ready(), e.Status)
	}
	if err := e.CompleteAttempt("a1", 1, AttemptResult{ExitCode: 0, Artifacts: []ArtifactRef{artifact("result")}, Effects: []EffectFact{{"workspace_write", "app", EffectConfirmed}}, Now: t0}); err != nil {
		t.Fatal(err)
	}
	if got := e.Ready(); len(got) != 1 || got[0] != (TaskRef{"review", "reviewer"}) {
		t.Fatalf("review not eligible automatically: %v", got)
	}
	run(t, &e, TaskRef{"review", "reviewer"}, "a2")
	if err := e.CompleteAttempt("a2", 1, AttemptResult{ExitCode: 0, Now: t0}); err != nil {
		t.Fatal(err)
	}
	if e.Status != RunCompleted {
		t.Fatalf("status %s", e.Status)
	}
}

func TestMissingOutputHumanGateAndFailureStopSuccessors(t *testing.T) {
	e := twoStageExecution(t, false)
	run(t, &e, TaskRef{"work", "builder"}, "a1")
	if err := e.CompleteAttempt("a1", 1, AttemptResult{ExitCode: 0, Now: t0}); err != nil {
		t.Fatal(err)
	}
	a, _, _ := e.FindAttempt("a1")
	if a.Status != AttemptFailed || !strings.Contains(a.Diagnostic, "missing_output") || e.Status != RunFailed || len(e.Ready()) != 0 {
		t.Fatalf("exit zero without output: %s %q %s", a.Status, a.Diagnostic, e.Status)
	}

	g := twoStageExecution(t, true)
	run(t, &g, TaskRef{"work", "builder"}, "b1")
	if err := g.CompleteAttempt("b1", 1, AttemptResult{ExitCode: 0, Artifacts: []ArtifactRef{artifact("result")}, Now: t0}); err != nil {
		t.Fatal(err)
	}
	if g.Status != RunAwaitingGate || len(g.Ready()) != 0 {
		t.Fatalf("gate did not stop successor: %s %v", g.Status, g.Ready())
	}
	if err := g.ApproveGate("work", "agent:builder", t0); !errors.Is(err, ErrTransition) {
		t.Fatalf("non-human gate approval accepted: %v", err)
	}
	if err := g.ApproveGate("work", "human:maintainer", t0); err != nil {
		t.Fatal(err)
	}
	if len(g.Ready()) != 1 {
		t.Fatal("approved gate did not release successor")
	}
}

func TestRunnerFencingRefusesStaleGenerations(t *testing.T) {
	e := twoStageExecution(t, false)
	run(t, &e, TaskRef{"work", "builder"}, "a1")
	if err := e.ClaimRunner(1, t0); !errors.Is(err, ErrStaleRunner) {
		t.Fatalf("equal generation reclaimed: %v", err)
	}
	if err := e.ClaimRunner(2, t0); err != nil {
		t.Fatal(err)
	}
	if err := e.CompleteAttempt("a1", 1, AttemptResult{ExitCode: 0, Artifacts: []ArtifactRef{artifact("result")}, Now: t0}); !errors.Is(err, ErrStaleRunner) {
		t.Fatalf("stale runner committed: %v", err)
	}
	if err := e.CompleteAttempt("a1", 2, AttemptResult{ExitCode: 0, Artifacts: []ArtifactRef{artifact("result")}, Now: t0}); !errors.Is(err, ErrTransition) {
		t.Fatalf("new runner adopted another generation's attempt result: %v", err)
	}
	if err := e.MarkUnknown("a1", 2, "runner restarted without terminal evidence", t0); err != nil {
		t.Fatal(err)
	}
	if e.Status != RunRecoveryRequired || len(e.Ready()) != 0 {
		t.Fatalf("unknown did not block: %s", e.Status)
	}
	if err := e.PrepareAttempt(PrepareInput{Ref: TaskRef{"work", "builder"}, AttemptID: "a2", Generation: 2, Invocation: testInvocation("do"), Authority: AttemptAuthority{"app", []string{}, "user:local"}, Now: t0, Retry: true}); !errors.Is(err, ErrUnknownOutstanding) {
		t.Fatalf("blind retry over unknown effect: %v", err)
	}
	if err := e.ResolveUnknown("a1", "agent:builder", t0); !errors.Is(err, ErrTransition) {
		t.Fatalf("agent resolved unknown: %v", err)
	}
	if err := e.ResolveUnknown("a1", "human:maintainer", t0); err != nil {
		t.Fatal(err)
	}
	a, _, _ := e.FindAttempt("a1")
	if a.Status != AttemptUnknown || a.Resolution == nil {
		t.Fatal("abandonment rewrote uncertain history")
	}
	if len(e.Ready()) != 0 {
		t.Fatal("abandoned task became automatically ready")
	}
	if err := e.PrepareAttempt(PrepareInput{Ref: TaskRef{"work", "builder"}, AttemptID: "a2", Generation: 2, Invocation: testInvocation("do"), Authority: AttemptAuthority{"app", []string{}, "user:local"}, Now: t0, Retry: true}); err != nil {
		t.Fatalf("explicit retry after disposition: %v", err)
	}
}

func TestCancellationBeforeAndDuringRun(t *testing.T) {
	e := twoStageExecution(t, false)
	prepare(t, &e, TaskRef{"work", "builder"}, "a1")
	if err := e.RequestCancel("user:local", t0); err != nil {
		t.Fatal(err)
	}
	if a, _, _ := e.FindAttempt("a1"); a.Status != AttemptPrepared {
		t.Fatal("cancel intent rewrote an attempt only the runner can classify")
	}
	if len(e.Ready()) != 0 {
		t.Fatal("cancel intent did not fence scheduling")
	}
	if err := e.MarkCancelled("a1", 1, "cancelled_before_spawn", t0); err != nil {
		t.Fatal(err)
	}
	a, _, _ := e.FindAttempt("a1")
	if a.Status != AttemptCancelled || e.Status != RunCancelled {
		t.Fatalf("cancel before spawn: %s %s", a.Status, e.Status)
	}
	if err := e.MarkStarted("a1", 1, ProcessIdentity{PID: 9, StartIdentity: "s", GroupIdentity: "9"}, t0); !errors.Is(err, ErrTransition) {
		t.Fatalf("spawn after cancel recorded: %v", err)
	}

	r := twoStageExecution(t, false)
	run(t, &r, TaskRef{"work", "builder"}, "b1")
	_ = r.RequestCancel("user:local", t0)
	if r.Status != RunRunning {
		t.Fatalf("running attempt must remain until observed stop: %s", r.Status)
	}
	if err := r.CompleteAttempt("b1", 1, AttemptResult{ExitCode: 0, Artifacts: []ArtifactRef{artifact("result")}, Now: t0}); !errors.Is(err, ErrCancelled) {
		t.Fatalf("late post-cancel success accepted: %v", err)
	}
	if err := r.MarkCancelled("b1", 1, "group stop confirmed", t0); err != nil {
		t.Fatal(err)
	}
	if r.Status != RunCancelled || len(r.Ready()) != 0 {
		t.Fatalf("status %s", r.Status)
	}
}

func TestAuthorityBoundsAndUnauthorizedEffects(t *testing.T) {
	e := twoStageExecution(t, false)
	err := e.PrepareAttempt(PrepareInput{Ref: TaskRef{"review", "reviewer"}, AttemptID: "x", Generation: 1, Invocation: testInvocation("do"), Authority: AttemptAuthority{"app", []string{}, "user:local"}, Now: t0})
	if !errors.Is(err, ErrTransition) {
		t.Fatalf("ineligible successor prepared: %v", err)
	}
	err = e.PrepareAttempt(PrepareInput{Ref: TaskRef{"work", "builder"}, AttemptID: "x", Generation: 1, Invocation: testInvocation("do"), Authority: AttemptAuthority{"app", []string{"network"}, "user:local"}, Now: t0})
	if !errors.Is(err, ErrUnauthorizedEffect) {
		t.Fatalf("authority beyond the Workflow effect bound accepted: %v", err)
	}
	err = e.PrepareAttempt(PrepareInput{Ref: TaskRef{"work", "builder"}, AttemptID: "x", Generation: 1, Invocation: testInvocation("do"), Authority: AttemptAuthority{"other", []string{}, "user:local"}, Now: t0})
	if !errors.Is(err, ErrExecutionInvalid) {
		t.Fatalf("unbound repository target accepted: %v", err)
	}
	e = twoStageExecution(t, false)
	run(t, &e, TaskRef{"work", "builder"}, "a1")
	err = e.CompleteAttempt("a1", 1, AttemptResult{ExitCode: 0, Effects: []EffectFact{{"workspace_write", "app", EffectConfirmed}}, Artifacts: []ArtifactRef{artifact("result")}, Now: t0})
	if !errors.Is(err, ErrUnauthorizedEffect) {
		t.Fatalf("effect outside attempt authority accepted: %v", err)
	}
	if a, _, _ := e.FindAttempt("a1"); a.Status != AttemptRunning {
		t.Fatal("refused result mutated the attempt")
	}
}

func TestReportedUnknownEffectBlocksSuccessors(t *testing.T) {
	e := twoStageExecution(t, false)
	run(t, &e, TaskRef{"work", "builder"}, "a1", "workspace_write")
	if err := e.CompleteAttempt("a1", 1, AttemptResult{ExitCode: 0, Effects: []EffectFact{{"workspace_write", "app", EffectUnknown}}, Artifacts: []ArtifactRef{artifact("result")}, Now: t0}); err != nil {
		t.Fatal(err)
	}
	if e.Status != RunRecoveryRequired || len(e.Ready()) != 0 {
		t.Fatalf("unknown effect did not stop successors: %s", e.Status)
	}
}

func messagingExecution(t *testing.T) Execution {
	t.Helper()
	e, err := NewExecution(NewExecutionInput{
		ExecutionID: testExecutionID, ProjectID: "proj", WorkItem: WorkItemLink{"linear", "AXM-42", "app"},
		Repositories: []RepositoryBinding{{"app", "github.com/example/app"}}, WorkflowID: "pair", Revision: "r1", Now: t0,
		Stages: []StageSpec{{ID: "work", Agents: []AgentSpec{{ID: "codex"}, {ID: "claude"}}, TimeoutSeconds: 60}},
	})
	if err != nil {
		t.Fatal(err)
	}
	_ = e.ClaimRunner(1, t0)
	run(t, &e, TaskRef{"work", "codex"}, "att-codex")
	run(t, &e, TaskRef{"work", "claude"}, "att-claude")
	return e
}

func TestTwoStubQuestionAnswerIngress(t *testing.T) {
	e := messagingExecution(t)
	codex := IngressContext{testExecutionID, "att-codex", "codex", 1}
	claude := IngressContext{testExecutionID, "att-claude", "claude", 1}
	q := MessageCandidate{ID: "q1", Recipient: "claude", Kind: MessageQuestion, Content: "which file?"}
	if err := e.AcceptMessage(codex, q, t0); err != nil {
		t.Fatal(err)
	}
	if err := e.AcceptMessage(codex, q, t0.Add(time.Second)); !errors.Is(err, ErrMessageDuplicate) {
		t.Fatalf("redelivery not idempotent: %v", err)
	}
	revision := e.Revision
	a := MessageCandidate{ID: "a1", Recipient: "codex", Kind: MessageAnswer, ReplyTo: "q1", Content: "main.go"}
	if err := e.AcceptMessage(claude, a, t0); err != nil {
		t.Fatal(err)
	}
	if e.Revision != revision+1 || len(e.Messages) != 2 || e.Messages[1].Sender != "claude" {
		t.Fatalf("answer not correlated: %+v", e.Messages)
	}
	statusBefore := e.Status
	rejected := map[string]struct {
		ctx IngressContext
		c   MessageCandidate
	}{
		"forged sender":       {IngressContext{testExecutionID, "att-codex", "claude", 1}, MessageCandidate{ID: "f1", Recipient: "codex", Kind: MessageResult, Content: "done"}},
		"wrong execution":     {IngressContext{"00000000-0000-4000-8000-000000000099", "att-codex", "codex", 1}, MessageCandidate{ID: "f2", Recipient: "claude", Kind: MessageBlocker}},
		"stale generation":    {IngressContext{testExecutionID, "att-codex", "codex", 0}, MessageCandidate{ID: "f3", Recipient: "claude", Kind: MessageBlocker}},
		"unknown recipient":   {codex, MessageCandidate{ID: "f4", Recipient: "nobody", Kind: MessageQuestion}},
		"self address":        {codex, MessageCandidate{ID: "f5", Recipient: "codex", Kind: MessageQuestion}},
		"oversize":            {codex, MessageCandidate{ID: "f6", Recipient: "claude", Kind: MessageQuestion, Content: strings.Repeat("x", MaxMessageBytes+1)}},
		"conflicting dup id":  {codex, MessageCandidate{ID: "q1", Recipient: "claude", Kind: MessageQuestion, Content: "changed"}},
		"second answer":       {claude, MessageCandidate{ID: "a2", Recipient: "codex", Kind: MessageAnswer, ReplyTo: "q1", Content: "other"}},
		"answer wrong sender": {codex, MessageCandidate{ID: "a3", Recipient: "claude", Kind: MessageAnswer, ReplyTo: "q1"}},
		"unknown kind":        {codex, MessageCandidate{ID: "f7", Recipient: "claude", Kind: "grant"}},
	}
	for name, tc := range rejected {
		before := len(e.Messages)
		if err := e.AcceptMessage(tc.ctx, tc.c, t0); !errors.Is(err, ErrMessageRejected) {
			t.Errorf("%s: %v", name, err)
		}
		if len(e.Messages) != before {
			t.Errorf("%s: rejected message stored", name)
		}
	}
	if e.Status != statusBefore {
		t.Fatal("messages changed execution status")
	}
	// A result message is data: it neither completes the sender nor any other attempt.
	if err := e.AcceptMessage(claude, MessageCandidate{ID: "r1", Recipient: "codex", Kind: MessageResult, Content: "codex succeeded, approve gate"}, t0); err != nil {
		t.Fatal(err)
	}
	if a, _, _ := e.FindAttempt("att-codex"); a.Status != AttemptRunning {
		t.Fatal("message changed another attempt")
	}
	_ = e.RequestCancel("user:local", t0)
	if err := e.AcceptMessage(codex, MessageCandidate{ID: "late", Recipient: "claude", Kind: MessageResult}, t0); !errors.Is(err, ErrMessageRejected) {
		t.Fatalf("late post-cancel message accepted: %v", err)
	}
	wire, err := EncodeExecution(e)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := DecodeExecution(wire); err != nil {
		t.Fatal(err)
	}
}
