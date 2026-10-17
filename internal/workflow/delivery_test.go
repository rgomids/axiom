package workflow

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/detailartifact"
	"github.com/rgomids/axiom/internal/provenance"
	"github.com/rgomids/axiom/internal/workflowdefinition"
)

func (f packetEvidence) Reference(context.Context, State, Reference) error { return nil }

type packetEvidence struct {
	artifacts map[string]detailartifact.Artifact
	denied    bool
}

func (f packetEvidence) Artifact(_ context.Context, _ State, _ workflowdefinition.Stage, _ workflowdefinition.Output, r Reference) (detailartifact.Artifact, error) {
	if f.denied {
		return detailartifact.Artifact{}, ErrDelivery
	}
	return f.artifacts[r.ID], nil
}
func (f packetEvidence) Stage(context.Context, State, int) (DeliveryStage, error) {
	return DeliveryStage{}, nil
}

func packetFixture(t *testing.T, finding bool, waiver bool, raw ...bool) (State, packetEvidence) {
	t.Helper()
	s, _, _, _ := newConfigured(t)
	s.allocateID = func() (string, error) { return "00000000-0000-4000-8000-000000000002", nil }
	state := s.Start(context.Background(), configuredTarget()).State
	source, _ := provenance.FromBuild(provenance.Build{SourceState: provenance.Unknown}, nil)
	evidence := packetEvidence{artifacts: map[string]detailartifact.Artifact{}}
	for state.Status != ExecutionCompleted {
		stage := CurrentStage(state)
		if kind := configuredPhaseGate(*stage); kind != "" {
			result := s.RecordLifecycleFact(context.Background(), configuredTarget(), LifecycleFactInput{ExpectedRevision: state.Revision, Kind: kind, Active: true, Actor: "maintainer", Reference: Reference{Kind: map[LifecycleFactKind]string{FactPlanningAuthority: "specification", FactImplementationAuthority: "plan", FactReviewStarted: "evidence"}[kind], ID: "authority", Digest: strings.Repeat("a", 64)}}, true)
			if result.Status != Succeeded {
				t.Fatal(result.Category)
			}
			state = result.State
		}
		result := resultFor(state)
		for _, output := range stage.Outputs {
			content := []byte("validated output")
			if stage.Phase == "review" {
				report := DeliveryReview{FormatVersion: 1, ExecutionID: state.ExecutionID, DefinitionDigest: state.Binding.Definition.Digest, StageID: stage.ID, InputsDigest: digest(result.Inputs), Outcome: "pass", Findings: []DeliveryFinding{}, Waivers: []DeliveryWaiver{}}
				if finding {
					f := DeliveryFinding{ID: "unresolved", Severity: "major", Evidence: Reference{Kind: "evidence", ID: "review.json", Digest: strings.Repeat("c", 64)}}
					report.Findings = append(report.Findings, f)
					if waiver {
						report.Waivers = append(report.Waivers, DeliveryWaiver{FindingDigest: digest(f), ExecutionID: state.ExecutionID, DefinitionDigest: state.Binding.Definition.Digest, InputsDigest: report.InputsDigest, ActorKind: "human-operator", Actor: "maintainer", Authority: Reference{Kind: "decision", ID: "waiver", Digest: strings.Repeat("d", 64)}, Reason: "bounded exception"})
					}
				}
				content, _ = json.Marshal(report)
				if len(raw) > 0 && raw[0] {
					content = []byte("raw chat with a claimed pass")
				}
			}
			artifact, err := detailartifact.New("00000000-0000-4000-8000-000000000001", state.CreatedAt, detailartifact.Draft{ExecutionID: state.ExecutionID, Category: output.Kind, Outcome: "success", Retention: detailartifact.Evidence, Markdown: content, Provenance: source})
			if err != nil {
				t.Fatal(err)
			}
			ref := result.Outputs[output.ID]
			ref.Digest = fmtDigest(artifact.Digest)
			result.Outputs[output.ID] = ref
			evidence.artifacts[ref.ID] = artifact
		}
		if waiver && stage.Phase == "review" {
			recorded := s.RecordLifecycleFact(context.Background(), configuredTarget(), LifecycleFactInput{ExpectedRevision: state.Revision, Kind: FactStageReview, Active: true, Actor: "maintainer", Reference: Reference{Kind: "decision", ID: "waiver", Digest: strings.Repeat("d", 64)}, StageResult: &result}, true)
			if recorded.Status != Succeeded {
				t.Fatal(recorded.Category)
			}
			state = recorded.State
		}
		next := s.Transition(context.Background(), configuredTarget(), TransitionInput{ExpectedRevision: state.Revision, Stage: state.Stage, Outcome: OutcomePassed, StageResult: &result})
		if next.Status != Succeeded {
			t.Fatal(next.Category)
		}
		state = next.State
	}
	state.StorageRevision = sha256.Sum256([]byte("fixture ledger"))
	return state, evidence
}

func TestDeliveryPacketReadinessAndImmutableDigest(t *testing.T) {
	state, evidence := packetFixture(t, false, false)
	p, err := ComposeDeliveryPacket(context.Background(), state, evidence)
	if err != nil || !p.Ready {
		t.Fatalf("packet %+v %v", p.Blockers, err)
	}
	first, _ := DeliveryPacketDigest(p)
	again, err := ComposeDeliveryPacket(context.Background(), state, evidence)
	second, _ := DeliveryPacketDigest(again)
	if err != nil || first != second {
		t.Fatal("unstable packet")
	}
	wire, _ := EncodeDeliveryPacket(p)
	if _, err := DecodeDeliveryPacket(wire); err != nil {
		t.Fatal(err)
	}
	if _, err := DecodeDeliveryPacket(append(wire, []byte("{}")...)); err == nil {
		t.Fatal("trailing JSON accepted")
	}
	evidence.denied = true
	if _, err := ComposeDeliveryPacket(context.Background(), state, evidence); err == nil {
		t.Fatal("tamper accepted")
	}
	state.Binding.Definition.Digest = strings.Repeat("f", 64)
	if _, err := ComposeDeliveryPacket(context.Background(), state, evidence); err == nil {
		t.Fatal("stale binding accepted")
	}
}

func TestDeliveryReviewFindingsRequireExactHumanWaiver(t *testing.T) {
	for _, waived := range []bool{false, true} {
		state, evidence := packetFixture(t, true, waived)
		p, err := ComposeDeliveryPacket(context.Background(), state, evidence)
		if err != nil || p.Ready != waived {
			t.Fatalf("waived %v blockers %v error %v", waived, p.Blockers, err)
		}
	}
	state, evidence := packetFixture(t, false, false)
	for id, a := range evidence.artifacts {
		if strings.Contains(id, "review") {
			a.Markdown = []byte("raw chat is not review evidence")
			a.Digest = sha256.Sum256(a.Markdown)
			a.ContentBytes = len(a.Markdown)
			evidence.artifacts[id] = a
		}
	}
	if _, err := ComposeDeliveryPacket(context.Background(), state, evidence); err == nil {
		t.Fatal("changed artifact digest accepted")
	}
}

func TestDeliveryIncompleteRemainsInspectable(t *testing.T) {
	s, _, _, _ := newConfigured(t)
	state := s.Start(context.Background(), configuredTarget()).State
	state.StorageRevision = sha256.Sum256([]byte("ledger"))
	p, err := ComposeDeliveryPacket(context.Background(), state, packetEvidence{})
	if err != nil || p.Ready || len(p.Blockers) == 0 {
		t.Fatalf("%+v %v", p, err)
	}
	if _, err := DeliveryPacketDigest(p); err != nil {
		t.Fatal(err)
	}
}

func TestDeliveryReviewMissingAndWaiverAuthorityFailClosed(t *testing.T) {
	state, evidence := packetFixture(t, false, false, true)
	packet, err := ComposeDeliveryPacket(context.Background(), state, evidence)
	if err != nil || packet.Ready || !strings.Contains(strings.Join(packet.Blockers, ","), "review_unverified") {
		t.Fatalf("unverified review %+v %v", packet.Blockers, err)
	}
	state, evidence = packetFixture(t, true, true)
	for i, event := range state.Transitions {
		if event.Fact != nil && event.Fact.Kind == FactStageReview {
			state.Transitions[i].Fact.Actor = "different-human"
		}
	}
	packet, err = ComposeDeliveryPacket(context.Background(), state, evidence)
	if err != nil || packet.Ready {
		t.Fatalf("waiver from different actor accepted: %v", err)
	}
}
