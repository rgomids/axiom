package workflow

import (
	"reflect"
	"testing"
)

func rejectionFixture(t *testing.T) (Execution, string, ArtifactRef) {
	t.Helper()
	e := twoStageExecution(t, true)
	run(t, &e, TaskRef{"work", "builder"}, "a1", "workspace_write")
	if err := e.CompleteAttempt("a1", 1, AttemptResult{ExitCode: 0, Artifacts: []ArtifactRef{artifact("result")}, Now: t0}); err != nil {
		t.Fatal(err)
	}
	digest, err := e.StageResultDigest("work")
	if err != nil {
		t.Fatal(err)
	}
	evidence := ArtifactRef{Name: "human-review", RetainedID: "evidence-1", RepositoryKey: "app", Digest: DigestText([]byte("review")), Size: 1}
	return e, digest, evidence
}

func TestRejectedResultPreservesFactsAndStopsEveryProgression(t *testing.T) {
	e, digest, evidence := rejectionFixture(t)
	before, _ := EncodeExecution(e)
	for _, actor := range []string{"agent:codex", "human:", ""} {
		if e.RejectResult("work", actor, digest, evidence, t0) == nil {
			t.Fatalf("accepted actor %q", actor)
		}
	}
	if after, _ := EncodeExecution(e); !reflect.DeepEqual(before, after) {
		t.Fatal("denial changed facts")
	}
	if err := e.RejectResult("work", "human:operator", digest, evidence, t0); err != nil {
		t.Fatal(err)
	}
	if e.Status != RunRejected || len(e.Ready()) != 0 || e.Stages[0].Tasks[0].Attempts[0].Status != AttemptSucceeded {
		t.Fatal("rejection rewrote exit-zero facts or progressed")
	}
	if e.ApproveGate("work", "human:operator", t0) == nil || e.ReviewStage("work", "human:operator", digest, evidence, t0) == nil || e.RejectResult("work", "human:operator", digest, evidence, t0) == nil {
		t.Fatal("rejected result accepted or rewritten")
	}
	if e.PrepareAttempt(PrepareInput{Ref: TaskRef{"review", "reviewer"}, AttemptID: "a2", Generation: 1, Invocation: testInvocation("do"), Authority: AttemptAuthority{"app", nil, "user:local"}, Now: t0}) == nil {
		t.Fatal("rejected source spawned successor")
	}
	wire, err := EncodeExecution(e)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := DecodeExecution(wire); err != nil {
		t.Fatal(err)
	}
}

func TestRejectedResultRefusesWrongEvidenceAndUncertainEffects(t *testing.T) {
	for _, change := range []func(*Execution, *string, *ArtifactRef){
		func(e *Execution, d *string, a *ArtifactRef) { *d = DigestText([]byte("stale-attempt")) },
		func(e *Execution, d *string, a *ArtifactRef) { a.RepositoryKey = "other" },
		func(e *Execution, d *string, a *ArtifactRef) { a.RetainedID = "" },
		func(e *Execution, d *string, a *ArtifactRef) {
			e.Stages[0].Tasks[0].Attempts[0].Status = AttemptUnknown
		},
	} {
		e, d, a := rejectionFixture(t)
		change(&e, &d, &a)
		if e.RejectResult("work", "human:operator", d, a, t0) == nil {
			t.Fatal("invalid rejection accepted")
		}
	}
}
