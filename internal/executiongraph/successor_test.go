package executiongraph

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func successorVector(t *testing.T, name string) SuccessorGraph {
	t.Helper()
	b, e := os.ReadFile("testdata/successor/valid/" + name + ".json")
	if e != nil {
		t.Fatal(e)
	}
	g, e := DecodeSuccessorGraph(b)
	if e != nil {
		t.Fatal(name, e)
	}
	return g
}
func TestSuccessorCanonicalVectors(t *testing.T) {
	files, e := filepath.Glob("testdata/successor/valid/*.json")
	if e != nil || len(files) < 16 {
		t.Fatal(files, e)
	}
	for _, name := range files {
		t.Run(filepath.Base(name), func(t *testing.T) {
			b, e := os.ReadFile(name)
			if e != nil {
				t.Fatal(e)
			}
			g, e := DecodeSuccessorGraph(b)
			if e != nil {
				t.Fatal(e)
			}
			encoded, e := EncodeSuccessorGraph(g)
			if e != nil || !bytes.Equal(encoded, b) {
				t.Fatal("noncanonical vector", e)
			}
			if g.StorageRevision != sha256.Sum256(b) {
				t.Fatal("wire revision")
			}
			if _, e := DecodeGraph(b); e == nil {
				t.Fatal("historical reader admitted successor")
			}
			d, e := DecodeGraphDocument(b)
			if e != nil || d.Successor == nil || d.Historical != nil {
				t.Fatal("dispatch", e)
			}
			reencoded, e := EncodeGraphDocument(d)
			if e != nil || !bytes.Equal(b, reencoded) {
				t.Fatal(e)
			}
		})
	}
}
func TestSuccessorInvalidVectors(t *testing.T) {
	files, _ := filepath.Glob("testdata/successor/invalid/*.json")
	if len(files) < 10 {
		t.Fatal(files)
	}
	for _, name := range files {
		t.Run(filepath.Base(name), func(t *testing.T) {
			b, e := os.ReadFile(name)
			if e != nil {
				t.Fatal(e)
			}
			if _, e := DecodeSuccessorGraph(b); e == nil {
				t.Fatal("accepted invalid vector")
			}
			if _, e := DecodeGraphDocument(b); e == nil {
				t.Fatal("dispatcher accepted invalid vector")
			}
		})
	}
	g := successorVector(t, "result_recorded")
	g.Children[0].Attempts[0].ClassificationReason = "invented"
	if _, e := EncodeSuccessorGraph(g); e == nil {
		t.Fatal("writer accepted invalid domain")
	}
}
func TestSuccessorHistoricalBytesAndDigests(t *testing.T) {
	b, e := os.ReadFile("testdata/successor/historical-digests.json")
	if e != nil {
		t.Fatal(e)
	}
	var digests map[string]string
	if e = json.Unmarshal(b, &digests); e != nil {
		t.Fatal(e)
	}
	for name, expected := range digests {
		t.Run(name, func(t *testing.T) {
			wire, e := os.ReadFile(filepath.Join("../..", name))
			if e != nil {
				t.Fatal(e)
			}
			sum := sha256.Sum256(wire)
			if hex.EncodeToString(sum[:]) != expected {
				t.Fatal("historical digest changed")
			}
			d, e := DecodeGraphDocument(wire)
			if e != nil || d.Historical == nil || d.Successor != nil {
				t.Fatal(e)
			}
			out, e := EncodeGraphDocument(d)
			if e != nil || !bytes.Equal(wire, out) {
				t.Fatal("historical bytes changed", e)
			}
			if _, e := DecodeSuccessorGraph(wire); e == nil {
				t.Fatal("historical bytes admitted as successor")
			}
		})
	}
	for _, wire := range []string{`{"formatVersion":99}`, `{"formatVersion":3,"formatVersion":2}`, `{}`, `null`} {
		if _, e := DecodeGraphDocument([]byte(wire)); e == nil {
			t.Fatal(wire)
		}
	}
}
func TestSuccessorAppendOnlyAndEpoch(t *testing.T) {
	old := successorVector(t, "process_outcome_unknown")
	next := successorVector(t, "reconciliation")
	if e := ValidateSuccessorUpdate(old, next); e != nil {
		t.Fatal("append", e)
	}
	newer := successorVector(t, "reconciliation")
	f := newer.Children[0].Attempts[0].ReconciliationFacts[0]
	f.FactID = "00000000-0000-4000-8000-000000000031"
	f.Classification = "recovery_required"
	newer.Children[0].Attempts[0].ReconciliationFacts = append(newer.Children[0].Attempts[0].ReconciliationFacts, f)
	if e := ValidateSuccessorUpdate(next, newer); e != nil {
		t.Fatal("contradictory facts must remain visible", e)
	}
	for _, change := range []func(*SuccessorGraph){
		func(g *SuccessorGraph) {
			g.Children[0].Attempts[0].ReconciliationFacts[0].ObservationsDigest = strings.Repeat("f", 64)
		},
		func(g *SuccessorGraph) { g.Children[0].Attempts[0].ReconciliationFacts = nil },
		func(g *SuccessorGraph) { g.Children[0].Attempts[0].ExitCode = 2 },
		func(g *SuccessorGraph) { g.StageConcurrency++ },
		func(g *SuccessorGraph) { g.Children[0].Attempts[0].LeaseEpoch++ },
	} {
		g := successorVector(t, "reconciliation")
		change(&g)
		if e := ValidateSuccessorUpdate(next, g); e == nil {
			t.Fatal("rewritten persisted state")
		}
	}
	progression := successorVector(t, "epoch_progression")
	if e := ValidateSuccessorGraph(progression); e != nil {
		t.Fatal(e)
	}
	progression.Children[0].Attempts[1].LeaseEpoch = 0
	if e := ValidateSuccessorGraph(progression); e == nil {
		t.Fatal("decreasing epoch")
	}
	running := successorVector(t, "running")
	finished := successorVector(t, "result_recorded")
	if e := ValidateSuccessorUpdate(running, finished); e != nil {
		t.Fatal("terminal save", e)
	}
}
func TestSuccessorQualifyingRecords(t *testing.T) {
	for _, name := range []string{"result_recorded", "blocker_published", "result_not_successful"} {
		g := successorVector(t, name)
		c := g.Children[0]
		a := c.Attempts[0]
		r := AttemptRecord{ParentID: g.Parent.ExecutionID, ChildID: c.ExecutionID, AttemptID: a.AttemptID, GraphRevision: g.Parent.GraphRevision, Revision: 1, Digest: a.QualifyingRecordRef, Accepted: true, Kind: "result", Outcome: "succeeded"}
		if name == "blocker_published" {
			r.Kind = "blocker"
		}
		if name == "result_not_successful" {
			r.Outcome = "failed"
		}
		if e := ValidateSuccessorRecords(g, []AttemptRecord{r}); e != nil {
			t.Fatal(name, e)
		}
		for _, change := range []func(*AttemptRecord){func(r *AttemptRecord) { r.AttemptID = "foreign" }, func(r *AttemptRecord) { r.GraphRevision++ }, func(r *AttemptRecord) { r.Accepted = false }, func(r *AttemptRecord) { r.Digest = strings.Repeat("f", 64) }, func(r *AttemptRecord) { r.Kind = "question" }} {
			bad := r
			change(&bad)
			if e := ValidateSuccessorRecords(g, []AttemptRecord{bad}); e == nil {
				t.Fatal("foreign qualification", name)
			}
		}
		if e := ValidateSuccessorRecords(g, nil); e == nil {
			t.Fatal("unresolved record")
		}
	}
}

func TestSuccessorCancellationAndCorrelationBounds(t *testing.T) {
	old := successorVector(t, "publication")
	next := successorVector(t, "cancel_request")
	if err := ValidateSuccessorUpdate(old, next); err != nil {
		t.Fatal("cancel request append", err)
	}
	changed := successorVector(t, "cancel_request")
	changed.CancelRequest.Actor = "other"
	if err := ValidateSuccessorUpdate(next, changed); err == nil {
		t.Fatal("mutated cancellation")
	}
	if err := ValidateSuccessorUpdate(next, old); err == nil {
		t.Fatal("deleted cancellation")
	}
	for _, change := range []func(*SuccessorGraph){
		func(g *SuccessorGraph) { g.Children[0].Attempts[0].Process.PID = 0 },
		func(g *SuccessorGraph) { g.Children[0].Attempts[0].Process.StartIdentity = "" },
		func(g *SuccessorGraph) { g.Children[0].Attempts[0].Process.GroupIdentity = strings.Repeat("x", 513) },
		func(g *SuccessorGraph) { g.Children[0].Attempts[0].PacketDigest = "" },
		func(g *SuccessorGraph) { g.StageConcurrency = MaxStageConcurrency + 1 },
		func(g *SuccessorGraph) { g.ClassificationPolicy = "unknown" },
	} {
		g := successorVector(t, "running")
		change(&g)
		if ValidateSuccessorGraph(g) == nil {
			t.Fatal("invalid correlation/envelope")
		}
	}
	cancelled := successorVector(t, "cancel_confirmed")
	cancelled.CancelRequest = nil
	if ValidateSuccessorGraph(cancelled) == nil {
		t.Fatal("cancellation without request")
	}
	if _, err := EncodeGraphDocument(GraphDocument{}); err == nil {
		t.Fatal("empty union")
	}
}
