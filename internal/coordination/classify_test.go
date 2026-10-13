package coordination

import (
	"context"
	"errors"
	"testing"
)

func TestResultOutcomeEnum(t *testing.T) {
	for _, outcome := range []string{"succeeded", "failed", "success", "", "free text"} {
		t.Run(outcome, func(t *testing.T) {
			graph := coordinationGraph(t)
			child := graph.Children[0]
			fields := []Field{{Name: "outcome", Value: outcome}}
			if outcome == "" {
				fields = []Field{{Name: "summary", Value: "missing outcome"}}
			}
			record, err := New(&memoryStore{}, nil, nil).Publish(context.Background(), graph, Input{Kind: Result, ParentID: graph.Parent.ExecutionID, ChildID: child.ExecutionID, GraphRevision: graph.Parent.GraphRevision, Provenance: testProvenance(), Fields: fields})
			valid := outcome == "succeeded" || outcome == "failed"
			if (err == nil) != valid {
				t.Fatalf("record=%+v err=%v", record, err)
			}
			if valid {
				if !ClassifierRecord(record).Accepted {
					t.Fatal("canonical record rejected")
				}
				record.Fields[0].Value = "free text"
				record.Digest, _ = recordDigest(record)
				if ValidRecord(record) || ClassifierRecord(record).Accepted {
					t.Fatal("invalid outcome accepted")
				}
			}
		})
	}
}

func TestAttemptCapacityReserve(t *testing.T) {
	for _, maximum := range []uint32{0, 1, 2, 3, 10, 11} {
		t.Run(string(rune('A'+maximum)), func(t *testing.T) {
			capacity, err := AttemptCapacity(maximum)
			if maximum == 0 || maximum > 10 {
				if !errors.Is(err, ErrCapacity) {
					t.Fatal(err)
				}
				return
			}
			if err != nil || capacity != 64/int(maximum) {
				t.Fatalf("capacity=%d err=%v", capacity, err)
			}
			for _, tt := range []struct {
				count   int
				kind    Kind
				allowed bool
			}{
				{capacity - 2, Progress, true}, {capacity - 1, Progress, false}, {capacity - 1, Result, true}, {capacity - 1, Blocker, true}, {capacity, Result, false}, {capacity, Blocker, false}, {-1, Progress, false},
			} {
				if (CheckAttemptCapacity(maximum, tt.count, tt.kind) == nil) != tt.allowed {
					t.Fatalf("case=%+v", tt)
				}
			}
		})
	}
}
