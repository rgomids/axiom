package executiongraph

import (
	"errors"
)

var ErrUnsupportedGraphFormat = errors.New("unsupported_graph_format")
var ErrStageConcurrency = errors.New("recorded stage concurrency exceeded")

// SuccessorConsumer is an immutable snapshot loaded with resolved canonical
// records. No consumer reclassifies attempts or selects a later record.
type SuccessorConsumer struct {
	graph   SuccessorGraph
	records []AttemptRecord
}

func LoadSuccessorConsumer(document GraphDocument, records []AttemptRecord) (SuccessorConsumer, error) {
	if document.Successor == nil || document.Historical != nil || document.Successor.ClassificationPolicy != ClassificationPolicyV1 {
		return SuccessorConsumer{}, ErrUnsupportedGraphFormat
	}
	if err := ValidateSuccessorRecords(*document.Successor, records); err != nil {
		return SuccessorConsumer{}, err
	}
	// The successor codec owns the copy, including all nested envelopes/facts.
	wire, err := EncodeSuccessorGraph(*document.Successor)
	if err != nil {
		return SuccessorConsumer{}, err
	}
	graph, err := DecodeSuccessorGraph(wire)
	if err != nil {
		return SuccessorConsumer{}, err
	}
	graph.StorageRevision = document.Successor.StorageRevision
	return SuccessorConsumer{graph: graph, records: append([]AttemptRecord(nil), records...)}, nil
}

func (c SuccessorConsumer) Ready(retries map[string]bool) ([]string, map[string]string, error) {
	return ReadySuccessorChildren(c.graph, retries)
}

// Status renders canonical persisted outcomes, including reasons and facts.
// Returned state is independent of the loaded snapshot.
func (c SuccessorConsumer) Status() (SuccessorGraph, error) {
	wire, err := EncodeSuccessorGraph(c.graph)
	if err != nil {
		return SuccessorGraph{}, err
	}
	graph, err := DecodeSuccessorGraph(wire)
	graph.StorageRevision = c.graph.StorageRevision
	return graph, err
}

// WorkspaceProjection exposes a successor graph to the existing Git workspace
// manager in its historical shape: envelopes, topology and each work child's
// latest attempt identity/status. It is never persisted, never encoded as a
// Graph2 document and never used for eligibility, which stays successor-only.
func WorkspaceProjection(g SuccessorGraph) (Graph, error) {
	if ValidateSuccessorGraph(g) != nil {
		return Graph{}, ErrInvalidGraph
	}
	historical := Graph{FormatVersion: g.projectionVersion(), Parent: g.Parent}
	for _, child := range g.Children {
		h := ChildExecution{ExecutionID: child.ExecutionID, ParentID: child.ParentID, GraphRevision: child.GraphRevision, NodeKey: child.NodeKey, Envelope: child.Envelope}
		if !child.Envelope.IntegrationOwner && len(child.Attempts) > 0 {
			a := child.Attempts[len(child.Attempts)-1]
			if a.Status != AttemptRunning {
				h.Attempts = []Attempt{{AttemptID: a.AttemptID, Number: 1, Status: a.Status, StartedAt: a.StartedAt, FinishedAt: a.FinishedAt}}
			}
		}
		historical.Children = append(historical.Children, h)
	}
	if !ValidGraph(historical) {
		return Graph{}, ErrInvalidGraph
	}
	return historical, nil
}
