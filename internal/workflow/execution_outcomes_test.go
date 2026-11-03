package workflow

import (
	"reflect"
	"testing"
)

func TestRepositoryOutcomesPreserveProvenanceAndCodec(t *testing.T) {
	e := twoStageExecution(t, false)
	e.Repositories = append(e.Repositories, RepositoryBinding{"docs", "github.com/example/docs"})
	run(t, &e, TaskRef{"work", "builder"}, "build", "workspace_write")
	if err := e.CompleteAttempt("build", 1, AttemptResult{Artifacts: []ArtifactRef{artifact("result")}, Now: t0}); err != nil {
		t.Fatal(err)
	}
	if err := e.PrepareAttempt(PrepareInput{Ref: TaskRef{"review", "reviewer"}, AttemptID: "review", Generation: 1, Invocation: testInvocation("review"), Authority: AttemptAuthority{"docs", []string{}, "user:local"}, Now: t0}); err != nil {
		t.Fatal(err)
	}
	if err := e.MarkStarted("review", 1, ProcessIdentity{PID: 4243, StartIdentity: "s", GroupIdentity: "4243"}, t0); err != nil {
		t.Fatal(err)
	}
	a := artifact("review")
	a.RepositoryKey = "docs"
	if err := e.CompleteAttempt("review", 1, AttemptResult{Artifacts: []ArtifactRef{a}, Now: t0}); err != nil {
		t.Fatal(err)
	}
	outcomes := e.RepositoryOutcomes()
	if len(outcomes) != 2 || outcomes[0].RepositoryKey != "app" || outcomes[1].RepositoryKey != "docs" || outcomes[0].Results[0].AttemptID != "build" || outcomes[1].Results[0].AttemptID != "review" || len(outcomes[0].Blockers) != 0 || len(outcomes[1].Blockers) != 0 {
		t.Fatalf("outcomes: %+v", outcomes)
	}
	if outcomes[0].Results[0].Artifacts[0].RepositoryKey != "app" || outcomes[1].Results[0].Artifacts[0].RepositoryKey != "docs" {
		t.Fatal("mixed artifacts")
	}
	wire, err := EncodeExecution(e)
	if err != nil {
		t.Fatal(err)
	}
	back, err := DecodeExecution(wire)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(outcomes, back.RepositoryOutcomes()) {
		t.Fatal("codec changed provenance")
	}
	outcomes[0].Results[0].Artifacts[0].Name = "mutated"
	if e.Stages[0].Tasks[0].Attempts[0].Artifacts[0].Name != "result" {
		t.Fatal("view aliases canonical artifacts")
	}
	if err := e.RequestCancel("user:cancel", t0); err != nil {
		t.Fatal(err)
	}
	for _, o := range e.RepositoryOutcomes() {
		if len(o.Blockers) == 0 {
			t.Fatal("cancelled execution ready")
		}
	}
}

func TestRepositoryOutcomesRefuseFabricatedSuccess(t *testing.T) {
	e := twoStageExecution(t, false)
	run(t, &e, TaskRef{"work", "builder"}, "build", "workspace_write")
	if err := e.CompleteAttempt("build", 1, AttemptResult{Artifacts: []ArtifactRef{artifact("result")}, Now: t0}); err != nil {
		t.Fatal(err)
	}
	run(t, &e, TaskRef{"review", "reviewer"}, "review")
	if err := e.CompleteAttempt("review", 1, AttemptResult{Now: t0}); err != nil {
		t.Fatal(err)
	}
	e.Stages[0].Tasks[0].Attempts[0].Artifacts = nil
	if _, err := EncodeExecution(e); err == nil {
		t.Fatal("codec accepted fabricated success without required output")
	}
	for _, o := range e.RepositoryOutcomes() {
		if len(o.Blockers) == 0 {
			t.Fatal("fabricated success with missing required output ready")
		}
	}
}

func TestRepositoryOutcomesUnknownAndMissingResult(t *testing.T) {
	e := twoStageExecution(t, false)
	if o := e.RepositoryOutcomes()[0]; o.Ready || len(o.Blockers) == 0 {
		t.Fatal("missing result ready")
	}
	run(t, &e, TaskRef{"work", "builder"}, "build", "workspace_write")
	if err := e.MarkUnknown("build", 1, "lost_result", t0); err != nil {
		t.Fatal(err)
	}
	o := e.RepositoryOutcomes()[0]
	if o.Ready || len(o.Blockers) == 0 || len(o.Results) != 1 || o.Results[0].Status != AttemptUnknown {
		t.Fatalf("unknown result: %+v", o)
	}
}
