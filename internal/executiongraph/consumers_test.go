package executiongraph

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"
)

func consumerRecords(g SuccessorGraph) []AttemptRecord {
	var records []AttemptRecord
	for _, c := range g.Children {
		for _, a := range c.Attempts {
			if a.QualifyingRecordRef == "" {
				continue
			}
			r := AttemptRecord{ParentID: g.Parent.ExecutionID, ChildID: c.ExecutionID, AttemptID: a.AttemptID, GraphRevision: g.Parent.GraphRevision, Revision: 1, Digest: a.QualifyingRecordRef, Accepted: true, Kind: "result", Outcome: "succeeded"}
			if a.ClassificationReason == BlockerPublished {
				r.Kind = "blocker"
				r.Outcome = ""
			}
			if a.ClassificationReason == ResultNotSuccessful {
				r.Outcome = "failed"
			}
			records = append(records, r)
		}
	}
	return records
}
func loadConsumer(t *testing.T, g SuccessorGraph) SuccessorConsumer {
	t.Helper()
	c, e := LoadSuccessorConsumer(GraphDocument{Successor: &g}, consumerRecords(g))
	if e != nil {
		t.Fatal(e)
	}
	return c
}

func TestConsumerPersistedStateMatrix(t *testing.T) {
	for _, name := range []string{"result_recorded", "blocker_published", "result_missing", "result_not_successful", "process_failed", "process_outcome_unknown", "cancel_confirmed", "timeout_stopped", "running"} {
		t.Run(name, func(t *testing.T) {
			g := successorVector(t, name)
			g.Children[0].Envelope.Controls.MaximumAttempts = 3
			// The reviewer is made dependent on the implementation child.
			g.Children[2].Envelope.Dependencies = []string{g.Children[0].ExecutionID}
			c := loadConsumer(t, g)
			id := g.Children[0].ExecutionID
			ready, blocked, e := c.Ready(map[string]bool{id: true})
			if e != nil {
				t.Fatal(e)
			}
			a := g.Children[0].Attempts[0]
			if g.CancelRequest == nil && a.Status != AttemptSucceeded && blocked[g.Children[2].ExecutionID] != "dependencies_unsatisfied" {
				t.Fatal(ready, blocked)
			}
			retry, e := SuccessorRetryEligible(g, id, true)
			if e != nil {
				t.Fatal(e)
			}
			found := false
			for _, r := range ready {
				found = found || r == id
			}
			if found != retry {
				t.Fatal("retry and readiness disagree")
			}
			status, e := c.Status()
			if e != nil || !reflect.DeepEqual(status.Children, g.Children) {
				t.Fatal("status changed persisted facts", e)
			}
			wire, e := EncodeSuccessorGraph(g)
			if e != nil {
				t.Fatal(e)
			}
			restart, e := DecodeSuccessorGraph(wire)
			if e != nil {
				t.Fatal(e)
			}
			after := loadConsumer(t, restart)
			rr, bb, e := after.Ready(map[string]bool{id: true})
			if e != nil || !reflect.DeepEqual(rr, ready) || !reflect.DeepEqual(bb, blocked) {
				t.Fatal("restart changed consumer state", e)
			}
		})
	}
}

func TestConsumerLoadInvariants(t *testing.T) {
	g := successorVector(t, "result_recorded")
	records := consumerRecords(g)
	for _, change := range []func(*SuccessorGraph){func(g *SuccessorGraph) { g.Children[0].Attempts[0].ClassificationReason = "" }, func(g *SuccessorGraph) { g.Children[0].Attempts[0].QualifyingRecordRef = "" }} {
		bad := successorVector(t, "result_recorded")
		change(&bad)
		if _, e := LoadSuccessorConsumer(GraphDocument{Successor: &bad}, records); e == nil {
			t.Fatal("accepted I1/I2 violation")
		}
	}
	for _, name := range []string{"result_recorded", "blocker_published"} {
		bad := successorVector(t, name)
		if _, e := LoadSuccessorConsumer(GraphDocument{Successor: &bad}, nil); e == nil {
			t.Fatal("unresolved I2/I3")
		}
		for _, change := range []func(*AttemptRecord){func(r *AttemptRecord) { r.AttemptID = r.ChildID }, func(r *AttemptRecord) { r.GraphRevision++ }, func(r *AttemptRecord) { r.ParentID = r.ChildID }, func(r *AttemptRecord) { r.Digest = strings.Repeat("e", 64) }} {
			rs := consumerRecords(bad)
			change(&rs[0])
			if _, e := LoadSuccessorConsumer(GraphDocument{Successor: &bad}, rs); e == nil {
				t.Fatal("foreign qualifying record")
			}
		}
	}
	if _, e := LoadSuccessorConsumer(GraphDocument{Historical: &Graph{}}, nil); !errors.Is(e, ErrUnsupportedGraphFormat) {
		t.Fatal(e)
	}
	g.ClassificationPolicy = ""
	if _, e := LoadSuccessorConsumer(GraphDocument{Successor: &g}, records); !errors.Is(e, ErrUnsupportedGraphFormat) {
		t.Fatal(e)
	}
}

func TestConsumerLateRecordsAndIndependentSnapshot(t *testing.T) {
	g := successorVector(t, "result_recorded")
	records := consumerRecords(g)
	late := records[0]
	late.Revision++
	late.Kind = "blocker"
	late.Outcome = ""
	late.Digest = strings.Repeat("f", 64)
	records = append(records, late)
	c, e := LoadSuccessorConsumer(GraphDocument{Successor: &g}, records)
	if e != nil {
		t.Fatal(e)
	}
	g.Children[0].Attempts[0].Status = AttemptFailed
	records[0].Digest = "bad"
	status, e := c.Status()
	if e != nil || status.Children[0].Attempts[0].Status != AttemptSucceeded {
		t.Fatal("late records or caller mutation rewrote persisted state", e)
	}
	status.Children[0].Attempts[0].Status = AttemptFailed
	next, _ := c.Status()
	if next.Children[0].Attempts[0].Status != AttemptSucceeded {
		t.Fatal("aliased status")
	}
}

func TestConsumerBatchConcurrency(t *testing.T) {
	g := successorVector(t, "result_missing")
	g.Children[0].Attempts = nil
	g.StageConcurrency = 1
	c := loadConsumer(t, g)
	ids := []string{g.Children[0].ExecutionID, g.Children[2].ExecutionID}
	if e := c.DispatchBatch(ids, nil); !errors.Is(e, ErrStageConcurrency) {
		t.Fatal(e)
	}
	if e := c.DispatchBatch(ids[:1], nil); e != nil {
		t.Fatal(e)
	}
	g = successorVector(t, "running")
	g.StageConcurrency = 1
	c = loadConsumer(t, g)
	if e := c.DispatchBatch([]string{g.Children[2].ExecutionID}, nil); !errors.Is(e, ErrStageConcurrency) {
		t.Fatal("running attempt not counted", e)
	}
	g.StageConcurrency = 2
	c = loadConsumer(t, g)
	if e := c.DispatchBatch([]string{g.Children[2].ExecutionID}, nil); e != nil {
		t.Fatal(e)
	}
	if e := c.DispatchBatch([]string{g.Children[0].ExecutionID}, nil); e == nil {
		t.Fatal("running dispatched")
	}
}

func TestConsumerIntegrationCanonicalBinding(t *testing.T) {
	g := successorVector(t, "result_recorded")
	// Give the independent reviewer its own persisted qualifying result.
	a := g.Children[0].Attempts[0]
	a.AttemptID = "00000000-0000-4000-8000-000000000011"
	a.QualifyingRecordRef = strings.Repeat("d", 64)
	g.Children[2].Attempts = []SuccessorAttempt{a}
	records := consumerRecords(g)
	var results []ChildResult
	var canonical []CanonicalChildResult
	base := strings.Repeat("1", 40)
	tree := strings.Repeat("2", 40)
	for _, r := range records {
		for _, child := range g.Children {
			if child.ExecutionID == r.ChildID {
				result := ChildResult{ParentID: r.ParentID, ChildID: r.ChildID, AttemptID: r.AttemptID, GraphRevision: r.GraphRevision, EnvelopeDigest: EnvelopeDigest(child.Envelope), BaseRevision: base, ResultTree: tree, Effects: child.Envelope.AllowedEffects}
				results = append(results, result)
				canonical = append(canonical, CanonicalChildResult{Record: r, Result: result})
			}
		}
	}
	c := loadConsumer(t, g)
	observation := IntegrationObservation{TargetRevision: base, TargetTree: base}
	if _, e := c.IntegrationPreview(observation, results, canonical); e != nil {
		t.Fatal(e)
	}
	bad := append([]ChildResult(nil), results...)
	bad[0].ResultTree = strings.Repeat("3", 40)
	if _, e := c.IntegrationPreview(observation, bad, canonical); e == nil {
		t.Fatal("caller result bypassed qualifying record")
	}
	forged := append([]CanonicalChildResult(nil), canonical...)
	forged[0].Record.Digest = strings.Repeat("e", 64)
	if _, e := c.IntegrationPreview(observation, results, forged); e == nil {
		t.Fatal("digest mismatch accepted")
	}
	missing := successorVector(t, "result_missing")
	c = loadConsumer(t, missing)
	if _, e := c.IntegrationPreview(observation, results, canonical); e == nil {
		t.Fatal("result_missing integrated")
	}
}

func TestConsumerReconciliationRetryWithoutPromotion(t *testing.T) {
	g, f, o := reconcileFixture(t, "process_outcome_unknown")
	g.Children[2].Envelope.Dependencies = []string{f.ChildID}
	c := loadConsumer(t, g)
	if e := c.DispatchBatch([]string{f.ChildID}, map[string]bool{f.ChildID: true}); e == nil {
		t.Fatal("unknown retry without fact")
	}
	next, e := AppendReconciliationFact(g, f, o)
	if e != nil {
		t.Fatal(e)
	}
	c = loadConsumer(t, next)
	if e := c.DispatchBatch([]string{f.ChildID}, map[string]bool{f.ChildID: true}); e != nil {
		t.Fatal(e)
	}
	_, blocked, e := c.Ready(map[string]bool{f.ChildID: true})
	if e != nil || blocked[g.Children[2].ExecutionID] != "dependencies_unsatisfied" {
		t.Fatal("fact promoted success", e)
	}
	// skipped has no permitted terminal classification in the successor family.
	next.Children[0].Attempts[0].Status = AttemptSkipped
	if _, e := LoadSuccessorConsumer(GraphDocument{Successor: &next}, nil); e == nil {
		t.Fatal("skipped successor admitted to retry")
	}
}

func TestConsumerHistoricalDocumentsRefused(t *testing.T) {
	for _, path := range []string{"../workflowcompiler/testdata/issue306/graph-v2.json", "../compatibility/testdata/stable-v1/snapshots/v0.12.0-issue276/state/graphs/v1/project-1/11e594f481958c10e3015d0bf0447a22f068a8a647f475df15ce2c7ab4b8f3f1.json"} {
		wire, e := os.ReadFile(path)
		if e != nil {
			t.Fatal(e)
		}
		doc, e := DecodeGraphDocument(wire)
		if e != nil || doc.Historical == nil {
			t.Fatal("historical inspection refused", e)
		}
		if _, e := LoadSuccessorConsumer(doc, nil); !errors.Is(e, ErrUnsupportedGraphFormat) {
			t.Fatal("historical mutation admitted", e)
		}
	}
}

func TestConsumerPersistedCancellationRefusesDispatchAndIntegration(t *testing.T) {
	g := successorVector(t, "cancel_request")
	c := loadConsumer(t, g)
	ready, blocked, e := c.Ready(nil)
	if e != nil || len(ready) != 0 {
		t.Fatal("cancelled graph ready", ready, e)
	}
	for _, child := range g.Children {
		if blocked[child.ExecutionID] != "cancel_requested" {
			t.Fatal(blocked)
		}
		if e := c.DispatchBatch([]string{child.ExecutionID}, map[string]bool{child.ExecutionID: true}); e == nil {
			t.Fatal("dispatch despite persisted cancellation")
		}
	}
	if available, e := SuccessorIntegrationAttemptAvailable(g); e != nil || available {
		t.Fatal("integration despite cancellation", e)
	}
	state, e := c.Status()
	if e != nil || !reflect.DeepEqual(state.CancelRequest, g.CancelRequest) {
		t.Fatal("lost persisted cancel request", e)
	}
}

func TestConsumerIntegrationPreviewReconciledAttemptCount(t *testing.T) {
	g := successorVector(t, "integration_unknown")
	// Qualify every ordinary source using independent canonical record projections.
	source := successorVector(t, "result_recorded").Children[0].Attempts[0]
	var integration int
	for i, child := range g.Children {
		if child.Envelope.IntegrationOwner {
			integration = i
			g.Children[i].Envelope.Controls.MaximumAttempts = 3
			continue
		}
		a := source
		a.AttemptID = fmt.Sprintf("00000000-0000-4000-8000-%012d", 20+i)
		a.QualifyingRecordRef = strings.Repeat(string(rune('a'+i)), 64)
		g.Children[i].Attempts = []SuccessorAttempt{a}
	}
	a := g.Children[integration].Attempts[0]
	_, fact, observations := reconcileFixture(t, "process_outcome_unknown")
	fact.ChildID = g.Children[integration].ExecutionID
	fact.AttemptID = a.AttemptID
	observations.ChildID = fact.ChildID
	observations.AttemptID = fact.AttemptID
	observations.LeaseEpoch = a.LeaseEpoch
	fact.ObservationsDigest, _ = observations.Digest()
	next, e := AppendReconciliationFact(g, fact, observations)
	if e != nil {
		t.Fatal(e)
	}
	var results []ChildResult
	var canonical []CanonicalChildResult
	base := strings.Repeat("1", 40)
	for _, r := range consumerRecords(next) {
		for _, child := range next.Children {
			if child.ExecutionID == r.ChildID {
				result := ChildResult{ParentID: r.ParentID, ChildID: r.ChildID, AttemptID: r.AttemptID, GraphRevision: r.GraphRevision, EnvelopeDigest: EnvelopeDigest(child.Envelope), BaseRevision: base, ResultTree: strings.Repeat("2", 40), Effects: child.Envelope.AllowedEffects}
				results = append(results, result)
				canonical = append(canonical, CanonicalChildResult{Record: r, Result: result})
			}
		}
	}
	before := loadConsumer(t, g)
	observation := IntegrationObservation{TargetRevision: base, TargetTree: base}
	if _, e := before.IntegrationPreview(observation, results, canonical); e == nil {
		t.Fatal("unreconciled integration attempt retried")
	}
	after := loadConsumer(t, next)
	preview, e := after.IntegrationPreview(observation, results, canonical)
	if e != nil {
		t.Fatal(e)
	}
	if preview.IntegrationAttempts != 1 {
		t.Fatal("preview lost canonical integration attempt count", preview.IntegrationAttempts)
	}
	digest, e := integrationPreviewDigest(preview)
	if e != nil || digest != preview.Digest {
		t.Fatal("attempt count not bound to digest", e)
	}
}
