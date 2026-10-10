package workflow

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/workflowdefinition"
)

type configuredDefinitions struct {
	document workflowdefinition.Document
	category string
	calls    int
}

func (d *configuredDefinitions) ResolveDefinition(context.Context, string) (DefinitionObservation, string) {
	d.calls++
	return DefinitionObservation{ProjectID: "123e4567-e89b-42d3-a456-426614174000", Document: d.document, Source: "project", Digest: d.document.Digest, Contexts: map[string]ContextObservation{}, ProjectRevision: d.document.Digest, LocalRevision: d.document.Digest}, d.category
}

type configuredValidator struct {
	denied bool
	calls  int
}

func (v *configuredValidator) ValidateStageOutput(context.Context, State, string, workflowdefinition.Stage, workflowdefinition.Output, workflowdefinition.Validator, Reference) error {
	v.calls++
	if v.denied {
		return errors.New("validator rejected")
	}
	return nil
}

type configuredReferences struct{}

func (configuredReferences) Validate(context.Context, string, string, Reference) error { return nil }

func newConfigured(t *testing.T) (Service, *s4Store, *configuredDefinitions, *configuredValidator) {
	t.Helper()
	s, store, _ := newS4Service(t)
	d := &configuredDefinitions{document: workflowdefinition.Builtin()}
	v := &configuredValidator{}
	s.references = configuredReferences{}
	return s.WithDefinitions(d, v), store, d, v
}
func configuredTarget() Target {
	target := s4Target()
	target.RuntimePreview = strings.Repeat("a", 64)
	return target
}
func resultFor(state State) StageResult {
	stage := CurrentStage(state)
	r := StageResult{Inputs: map[string]Reference{}, Outputs: map[string]Reference{}}
	for _, input := range stage.Inputs {
		if input.Kind == "stage-output" {
			r.Inputs[input.ID], _ = priorOutput(state, input.Source)
		}
	}
	for _, output := range stage.Outputs {
		r.Outputs[output.ID] = Reference{Kind: "artifact", ID: stage.ID + "-" + output.ID, Digest: strings.Repeat("b", 64)}
	}
	return r
}
func advanceConfigured(t *testing.T, s Service, state State) State {
	t.Helper()
	value := resultFor(state)
	out := s.Transition(context.Background(), configuredTarget(), TransitionInput{ExpectedRevision: state.Revision, Stage: state.Stage, Outcome: OutcomePassed, StageResult: &value})
	if out.Status != Succeeded {
		t.Fatalf("advance %s: %s", state.Stage, out.Category)
	}
	return out.State
}

func TestConfiguredRevisionIsolationAndOfflineResume(t *testing.T) {
	s, _, d, _ := newConfigured(t)
	ctx := context.Background()
	target := configuredTarget()
	a := s.Start(ctx, target)
	if a.Status != Succeeded || a.State.Binding.Definition.Revision != 1 {
		t.Fatalf("start %s", a.Category)
	}
	r2 := d.document.Definition
	r2.Revision = 2
	r2.Stages[0].Instructions = "Changed R2 instructions"
	doc, issues := workflowdefinition.Encode(r2)
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	d.document = doc
	stopped := s.Transition(ctx, target, TransitionInput{ExpectedRevision: 1, Stage: a.State.Stage, Outcome: OutcomeFailed})
	if stopped.Status != Interrupted {
		t.Fatal(stopped.Category)
	}
	calls := d.calls
	d.category = "workflow_selection_required"
	resumed := s.Resume(ctx, target, stopped.State.Revision)
	if resumed.Status != Succeeded || d.calls != calls || resumed.State.Binding.Definition != a.State.Binding.Definition || CurrentStage(resumed.State).Instructions != CurrentStage(a.State).Instructions {
		t.Fatalf("resume switched contract: %s", resumed.Category)
	}
	b, _, _, _ := newConfigured(t)
	b = b.WithDefinitions(&configuredDefinitions{document: doc}, &configuredValidator{})
	fresh := b.Start(ctx, target)
	if fresh.Status != Succeeded || fresh.State.Binding.Definition.Revision != 2 {
		t.Fatalf("new execution %s", fresh.Category)
	}
}

func TestConfiguredPrerequisitesValidatorsAndReplay(t *testing.T) {
	s, store, _, validator := newConfigured(t)
	target := configuredTarget()
	ctx := context.Background()
	state := s.Start(ctx, target).State
	missing := s.Transition(ctx, target, TransitionInput{ExpectedRevision: 1, Stage: state.Stage, Outcome: OutcomePassed})
	if missing.Category != "stage_prerequisite_missing" || store.saves != 0 {
		t.Fatal(missing.Category)
	}
	value := resultFor(state)
	input := TransitionInput{ExpectedRevision: 1, Stage: state.Stage, Outcome: OutcomePassed, StageResult: &value}
	validator.denied = true
	rejected := s.Transition(ctx, target, input)
	if rejected.Category != "stage_validator_failed" || store.saves != 0 {
		t.Fatal(rejected.Category)
	}
	validator.denied = false
	advanced := s.Transition(ctx, target, input)
	if advanced.Status != Succeeded {
		t.Fatal(advanced.Category)
	}
	if replay := s.Transition(ctx, target, input); replay.Category != "workflow_transition_replayed" {
		t.Fatal(replay.Category)
	}
	second := resultFor(advanced.State)
	second.Inputs["source"] = Reference{Kind: "artifact", ID: "foreign", Digest: strings.Repeat("c", 64)}
	rejected = s.Transition(ctx, target, TransitionInput{ExpectedRevision: advanced.State.Revision, Stage: advanced.State.Stage, Outcome: OutcomePassed, StageResult: &second})
	if rejected.Category != "stage_prerequisite_missing" || store.saves != 1 {
		t.Fatal(rejected.Category)
	}
}

func TestConfiguredCustomStagesProjectLifecycleAndDenyUnboundAcceptance(t *testing.T) {
	s, store, d, _ := newConfigured(t)
	definition := d.document.Definition
	for i := range definition.Stages {
		old := definition.Stages[i].ID
		definition.Stages[i].ID = "custom-" + old
		for j := range definition.Stages {
			for k := range definition.Stages[j].Inputs {
				source := definition.Stages[j].Inputs[k].Source
				if strings.HasPrefix(source, old+"/") {
					definition.Stages[j].Inputs[k].Source = "custom-" + source
				}
			}
		}
	}
	doc, issues := workflowdefinition.Encode(definition)
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	d.document = doc
	state := s.Start(context.Background(), configuredTarget()).State
	seen := map[WorkItemLifecycleStage]bool{}
	record := func() {
		p, err := DeriveLifecycle(state)
		if err != nil {
			t.Fatal(err)
		}
		seen[p.Stage] = true
	}
	record()
	for state.Status != ExecutionCompleted {
		stage := CurrentStage(state)
		if kind := configuredPhaseGate(*stage); kind != "" {
			value := resultFor(state)
			denied := s.Transition(context.Background(), configuredTarget(), TransitionInput{ExpectedRevision: state.Revision, Stage: state.Stage, Outcome: OutcomePassed, StageResult: &value})
			if denied.Category != "workflow_authority_required" {
				t.Fatal(denied.Category)
			}
			refKind := map[LifecycleFactKind]string{FactPlanningAuthority: "specification", FactImplementationAuthority: "plan", FactReviewStarted: "evidence"}[kind]
			input := LifecycleFactInput{ExpectedRevision: state.Revision, Kind: kind, Active: true, Actor: "maintainer", Reference: Reference{Kind: refKind, ID: "decision", Digest: strings.Repeat("b", 64)}}
			if denied := s.RecordLifecycleFact(context.Background(), configuredTarget(), input, false); denied.Status != Denied {
				t.Fatal("agent message granted authority")
			}
			accepted := s.RecordLifecycleFact(context.Background(), configuredTarget(), input, true)
			if accepted.Status != Succeeded {
				t.Fatal(accepted.Category)
			}
			state = accepted.State
			record()
		}
		state = advanceConfigured(t, s, state)
		record()
	}
	before := cloneState(store.state)
	// Even authorized, valid Evidence or the exact final technical output is
	// insufficient without #277's correlated delivery packet.
	finalResult := state.Transitions[len(state.Transitions)-1].Ledger.Result
	for _, reference := range []Reference{{Kind: "evidence", ID: "unrelated", Digest: strings.Repeat("b", 64)}, finalResult.Outputs["result"]} {
		accepted := s.RecordLifecycleFact(context.Background(), configuredTarget(), LifecycleFactInput{ExpectedRevision: state.Revision, Kind: FactHumanAcceptance, Actor: "maintainer", Active: true, Reference: reference, StageResult: &finalResult}, true)
		if accepted.Status != Denied || accepted.Category != "delivery_packet_required" || !reflect.DeepEqual(before, store.state) {
			t.Fatalf("unbound acceptance changed state: %s", accepted.Category)
		}
	}
	if NextGateAction(state) != nil {
		t.Fatal("unsupported acceptance advertised")
	}
	// A forged acceptance event cannot become readable canonical truth either.
	forged := cloneState(state)
	event := forged.Transitions[len(forged.Transitions)-1]
	event.Revision++
	event.From = state.Stage
	event.To = state.Stage
	event.Outcome = OutcomeFact
	event.Ledger = nil
	event.Fact = &LifecycleFact{Kind: FactHumanAcceptance, Active: true, Actor: "maintainer", ScopeDigest: executionScopeDigest(state), Reference: Reference{Kind: "evidence", ID: "unrelated", Digest: strings.Repeat("b", 64)}}
	forged.Transitions = append(forged.Transitions, event)
	forged.Revision++
	if ValidState(forged) {
		t.Fatal("forged unbound acceptance accepted")
	}
	for _, stage := range LifecycleStages() {
		if stage == LifecycleAccepted {
			continue
		} // Reserved for packet-bound #277.
		if !seen[stage] {
			t.Errorf("missing lifecycle %s", stage)
		}
	}
	for _, stage := range definition.Stages {
		if ValidDesiredProjectionLabel("axiom:stage:" + stage.ID) {
			t.Fatal("custom label accepted")
		}
	}
}

func TestConfiguredCorruptionAndPreviewDriftFailClosed(t *testing.T) {
	s, store, d, _ := newConfigured(t)
	ctx := context.Background()
	target := configuredTarget()
	reviewed, failed := s.ValidateTarget(ctx, target)
	if failed.Category != "" {
		t.Fatal(failed.Category)
	}
	r2 := d.document.Definition
	r2.Revision = 2
	d.document, _ = workflowdefinition.Encode(r2)
	target.ReviewedTarget = &reviewed
	if stale := s.Start(ctx, target); stale.Category != "stale_execution_target" || store.created {
		t.Fatal(stale.Category)
	}
	target.ReviewedTarget = nil
	state := s.Start(ctx, target).State
	for name, mutate := range map[string]func(*State){
		"missing":  func(s *State) { s.Binding.Snapshot = nil },
		"tampered": func(s *State) { s.Binding.Snapshot = append(s.Binding.Snapshot, 'x') },
		"future":   func(s *State) { s.FormatVersion = 3 },
		"identity": func(s *State) { s.Binding.Definition.Revision++ },
		"ordinal":  func(s *State) { s.StageOrdinal++ },
		"scope":    func(s *State) { s.Binding.ProjectID = "foreign" },
	} {
		t.Run(name, func(t *testing.T) {
			copy := cloneState(state)
			mutate(&copy)
			if ValidState(copy) {
				t.Fatal("corrupt binding accepted")
			}
		})
	}
}

func TestConfiguredHumanReviewBindsExactResultAndActor(t *testing.T) {
	s, store, d, _ := newConfigured(t)
	definition := d.document.Definition
	definition.Stages[0].Validators[0].Kind = "human-review"
	var issues []workflowdefinition.Diagnostic
	d.document, issues = workflowdefinition.Encode(definition)
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	ctx := context.Background()
	target := configuredTarget()
	state := s.Start(ctx, target).State
	value := resultFor(state)
	input := TransitionInput{ExpectedRevision: 1, Stage: state.Stage, Outcome: OutcomePassed, StageResult: &value}
	if r := s.Transition(ctx, target, input); r.Category != "stage_human_review_required" || store.saves != 0 {
		t.Fatal(r.Category)
	}
	fact := LifecycleFactInput{ExpectedRevision: 1, Kind: FactStageReview, Active: true, Reference: Reference{Kind: "evidence", ID: "review", Digest: strings.Repeat("c", 64)}, StageResult: &value}
	if r := s.RecordLifecycleFact(ctx, target, fact, true); r.Category != "human_actor_required" {
		t.Fatal(r.Category)
	}
	fact.Actor = "maintainer"
	if r := s.RecordLifecycleFact(ctx, target, fact, false); r.Status != Denied {
		t.Fatal("unapproved message granted review")
	}
	review := s.RecordLifecycleFact(ctx, target, fact, true)
	if review.Status != Succeeded {
		t.Fatal(review.Category)
	}
	changed := resultFor(state)
	changed.Outputs["result"] = Reference{Kind: "artifact", ID: "changed", Digest: strings.Repeat("d", 64)}
	input.ExpectedRevision = review.State.Revision
	input.StageResult = &changed
	if r := s.Transition(ctx, target, input); r.Category != "stage_human_review_required" {
		t.Fatal("review transferred to another result")
	}
	input.StageResult = &value
	if r := s.Transition(ctx, target, input); r.Status != Succeeded {
		t.Fatal(r.Category)
	}
	corrupt := cloneState(store.state)
	corrupt.Transitions[0].Fact.Actor = ""
	if ValidState(corrupt) {
		t.Fatal("missing human actor accepted")
	}
}

func TestConfiguredContextIsPinnedAndUnsupportedAdapterBlocks(t *testing.T) {
	s, store, _, validator := newConfigured(t)
	ctx := context.Background()
	target := configuredTarget()
	state := s.Start(ctx, target).State
	// An unsupported installed validator cannot be replaced by an agent's pass.
	s.validators = nil
	value := resultFor(state)
	if r := s.Transition(ctx, target, TransitionInput{ExpectedRevision: 1, Stage: state.Stage, Outcome: OutcomePassed, StageResult: &value}); r.Category != "stage_validator_failed" || store.saves != 0 {
		t.Fatal(r.Category)
	}
	s.validators = validator
	// Test the retained context boundary directly with a valid custom definition.
	doc, _ := bindingDocument(state)
	definition := doc.Definition
	definition.Stages[0].Inputs = []workflowdefinition.Input{{ID: "source", Kind: "project-context", Source: "business-context", Required: true}}
	doc, issues := workflowdefinition.Encode(definition)
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	state.Binding.Snapshot = doc.Canonical
	state.Binding.Definition = doc.Ref("project")
	content := []byte("Pinned business context")
	state.Binding.Contexts = map[string]ContextObservation{"business-context": {Source: "business-context", Digest: digestBytes(content), Content: content}}
	if !ValidState(state) {
		t.Fatal("valid context snapshot rejected")
	}
	missing := cloneState(state)
	delete(missing.Binding.Contexts, "business-context")
	if ValidState(missing) {
		t.Fatal("missing required retained context accepted")
	}
	value = resultFor(state)
	value.Inputs = StageInputs(state)
	if !validStageResult(state, *CurrentStage(state), value) {
		t.Fatal("pinned context rejected")
	}
	value.Inputs["source"] = Reference{Kind: "context", ID: "business-context", Digest: strings.Repeat("e", 64)}
	if validStageResult(state, *CurrentStage(state), value) {
		t.Fatal("changed context admitted")
	}
	state.Binding.Contexts["business-context"] = ContextObservation{Source: "business-context", Digest: digestBytes(content), Content: []byte("tampered")}
	if ValidState(state) {
		t.Fatal("tampered retained context accepted")
	}
}

func TestConfiguredProviderProjectionCannotAdvanceCanonicalState(t *testing.T) {
	s, store, _, _ := newConfigured(t)
	ctx := context.Background()
	state := advanceConfigured(t, s, s.Start(ctx, configuredTarget()).State)
	provider := s.projection.(*s4Projection)
	provider.observation.IssueLabels = []string{"axiom:stage:accepted"}
	provider.observation.IssueState = "CLOSED"
	before := cloneState(store.state)
	preview := s.PrepareProjection(ctx, configuredTarget(), state.Revision)
	if preview.Status != Succeeded || !reflect.DeepEqual(before, store.state) {
		t.Fatalf("provider rewrote canonical state: %s", preview.Category)
	}
	if preview.Preview == nil || preview.Preview.Label != "axiom:stage:specifying" {
		t.Fatalf("projection followed provider: %#v", preview.Preview)
	}
	projection, err := DeriveLifecycle(store.state)
	if err != nil || projection.Stage != LifecycleSpecifying || store.state.Revision != state.Revision {
		t.Fatal("provider acceptance advanced lifecycle")
	}
}
