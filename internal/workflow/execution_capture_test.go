package workflow

import (
	"bytes"
	"testing"
)

func TestCaptureCodecAndCorrelation(t *testing.T) {
	e := enrolledExecution(t, false)
	succeedEnrolled(t, &e, "a")
	in := intent(e, "int-a", "att-a")
	if err := e.PrepareCapture(in, "/private/agent-att-a"); err != nil {
		t.Fatal(err)
	}
	a, _, _ := e.FindAttempt("att-a")
	if err := e.PrepareIntegration(in); err == nil {
		t.Fatal("prepared without captured candidate")
	}
	if err := e.RecordCaptureCandidate("att-a", 1, sha("c"), t0); err != nil {
		t.Fatal(err)
	}
	if err := e.RecordCaptureCandidate("att-a", 1, sha("d"), t0); err == nil {
		t.Fatal("candidate rewritten")
	}
	if err := e.RecordCaptureCandidate("att-a", 2, sha("d"), t0); err == nil {
		t.Fatal("stale fence accepted")
	}
	o := CaptureObservation{Outcome: "captured", HeadSHA: sha("c"), IndexDigest: DigestText(nil), WorktreeDigest: DigestText(nil), ObservedAt: t0}
	if err := e.ObserveCapture("att-a", 1, o); err != nil {
		t.Fatal(err)
	}
	if err := e.PrepareCapture(in, "/private/agent-att-a"); err == nil {
		t.Fatal("capture replayed")
	}
	wire, err := EncodeExecution(e)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := DecodeExecution(wire)
	if err != nil {
		t.Fatal(err)
	}
	again, _ := EncodeExecution(decoded)
	if !bytes.Equal(wire, again) {
		t.Fatal("codec changed capture")
	}
	if err := e.PrepareIntegration(in); err != nil {
		t.Fatal(err)
	}
	if a.Capture.CandidateSHA != e.Integrations[0].CandidateSHA {
		t.Fatal("candidate identity lost")
	}
}

func TestCaptureAdmissionAndImmutableTransitions(t *testing.T) {
	for _, mode := range []string{"path", "branch", "base", "repo", "attempt", "cancel", "gate", "unenrolled"} {
		t.Run(mode, func(t *testing.T) {
			e := enrolledExecution(t, mode == "gate")
			succeedEnrolled(t, &e, "a")
			in := intent(e, "int-a", "att-a")
			path := "/private/agent-att-a"
			switch mode {
			case "path":
				path = "/private/other"
			case "branch":
				in.SourceBranch = "refs/heads/other"
			case "base":
				in.BaseSHA = sha("a")
			case "repo":
				in.RepositoryIdentity = "other"
			case "attempt":
				in.SourceAttemptID = "missing"
			case "cancel":
				if err := e.RequestCancel("user:cancel", t0); err != nil {
					t.Fatal(err)
				}
			case "unenrolled":
				e = integrationExecution(t, false)
				succeedWithCheck(t, &e, "a", "att-a", "workspace_write", "validation_command")
			}
			if err := e.PrepareCapture(in, path); err == nil {
				t.Fatal("unauthorized capture accepted")
			}
		})
	}
	c := &SourceCapture{Version: 1, Path: "/private/agent", RunnerGeneration: 1, PreparedAt: t0}
	for _, mutate := range []func(*SourceCapture){func(n *SourceCapture) { n.Path = "/other/agent" }, func(n *SourceCapture) { n.RunnerGeneration++ }, func(n *SourceCapture) { n.Version++ }} {
		n := *c
		mutate(&n)
		if PreservesCaptureHistory(c, &n) {
			t.Fatal("intent rewritten")
		}
	}
	n := *c
	n.CandidateSHA = sha("c")
	if PreservesCaptureHistory(nil, &n) {
		t.Fatal("intent and candidate published together")
	}
	if !PreservesCaptureHistory(c, &n) {
		t.Fatal("candidate append refused")
	}
	rewritten := n
	rewritten.CandidateSHA = sha("d")
	if PreservesCaptureHistory(&n, &rewritten) {
		t.Fatal("candidate rewritten")
	}
}

func TestCapturePreservesHumanResultReviewDigest(t *testing.T) {
	template := enrolledExecution(t, false)
	stage := template.Workflow.Stages[0]
	stage.Agents = append([]AgentSpec(nil), stage.Agents[:2]...)
	stage.Agents[1].DependsOn = nil
	stage.ReviewRequired = true
	e, err := NewExecution(NewExecutionInput{ExecutionID: template.ExecutionID, ProjectID: template.ProjectID, WorkItem: template.WorkItem, Repositories: template.Repositories, WorkflowID: "review-capture", Revision: "r1", Stages: []StageSpec{stage}, EnrollIntegration: true, Now: t0})
	if err != nil {
		t.Fatal(err)
	}
	if err := e.ClaimRunner(1, t0); err != nil {
		t.Fatal(err)
	}
	succeedEnrolled(t, &e, "a")
	succeedEnrolled(t, &e, "b")
	before, err := e.StageResultDigest("work")
	if err != nil {
		t.Fatal(err)
	}
	if err := e.ReviewStage("work", "human:reviewer", before, ArtifactRef{Name: "review", RetainedID: "review", RepositoryKey: "app", Digest: DigestText([]byte("review"))}, t0); err != nil {
		t.Fatal(err)
	}
	in := intent(e, "int-a", "att-a")
	if err := e.PrepareCapture(in, "/private/agent-att-a"); err != nil {
		t.Fatal(err)
	}
	if err := e.RecordCaptureCandidate("att-a", 1, sha("c"), t0); err != nil {
		t.Fatal(err)
	}
	if err := e.ObserveCapture("att-a", 1, CaptureObservation{Outcome: "captured", HeadSHA: sha("c"), IndexDigest: DigestText(nil), WorktreeDigest: DigestText(nil), ObservedAt: t0}); err != nil {
		t.Fatal(err)
	}
	after, _ := e.StageResultDigest("work")
	if after != before {
		t.Fatal("principal capture invalidated human writer-result review")
	}
	if err := e.PrepareIntegration(in); err != nil {
		t.Fatal(err)
	}
}
