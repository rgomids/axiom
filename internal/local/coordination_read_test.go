package local

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/coordination"
)

// The retired format-1 stream is read only for cutover inventory. Its reader
// still returns detached canonical records and refuses tampered bytes.
func TestCoordinationReadStreamCanonicalOnlyAndTamper(t *testing.T) {
	graph := localGraphFixture(t)
	parentID, childID := graph.Parent.ExecutionID, graph.Children[0].ExecutionID
	store, err := NewCoordinationStore(filepath.Join(privateTestRoot(t), "state"), "project-1")
	if err != nil {
		t.Fatal(err)
	}
	service := coordination.New(store, nil, nil)
	first, err := service.Publish(context.Background(), graph, coordination.Input{Kind: coordination.Progress, ParentID: parentID, ChildID: childID, GraphRevision: 1, Fields: []coordination.Field{{Name: "phase", Value: "testing"}}, Provenance: inboxProvenance()})
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.Publish(context.Background(), graph, coordination.Input{Kind: coordination.Blocker, ParentID: parentID, ChildID: childID, GraphRevision: 1, ExpectedRevision: 1, PreviousDigest: first.Digest, Fields: []coordination.Field{{Name: "code", Value: "missing_input"}}, Provenance: inboxProvenance()})
	if err != nil {
		t.Fatal(err)
	}
	stream, found, err := store.ReadStream(context.Background(), parentID, childID)
	if err != nil || !found || len(stream.Records) != 2 || stream.Records[1].Digest != second.Digest {
		t.Fatalf("stream=%+v found=%t err=%v", stream, found, err)
	}
	stream.Records[0].Fields[0].Value = "caller mutation"
	again, _, err := store.ReadStream(context.Background(), parentID, childID)
	if err != nil || again.Records[0].Digest != first.Digest || again.Records[0].Fields[0].Value != "testing" {
		t.Fatalf("mutated stored stream=%+v %v", again, err)
	}
	path := filepath.Join(store.root, "coordination", "v1", "project-1", coordinationName(parentID, childID))
	wire, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	wire = bytes.Replace(wire, []byte(first.Digest), bytes.Repeat([]byte("0"), 64), 1)
	if err := os.WriteFile(path, wire, 0600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.ReadStream(context.Background(), parentID, childID); !errors.Is(err, ErrUnsafe) {
		t.Fatalf("tamper read=%v", err)
	}
}

func inboxProvenance() coordination.Provenance {
	return coordination.Provenance{Product: "fake-child", Version: "dev", Revision: "test", SourceState: "clean"}
}

// canonicalHistory lets Service assign chain fields without changing the local
// publication protocol while constructing a valid stream beyond its old read cap.
type canonicalHistory struct{ records []coordination.Record }

func (s *canonicalHistory) Latest(context.Context, string, string) (coordination.Record, bool, error) {
	if len(s.records) == 0 {
		return coordination.Record{}, false, nil
	}
	return s.records[len(s.records)-1], true, nil
}
func (s *canonicalHistory) Publish(_ context.Context, r coordination.Record) error {
	s.records = append(s.records, r)
	return nil
}

func TestReadStreamSupportsCompleteCodecByteWindow(t *testing.T) {
	graph := localGraphFixture(t)
	store, err := NewCoordinationStore(filepath.Join(privateTestRoot(t), "state"), "project-1")
	if err != nil {
		t.Fatal(err)
	}
	seed := coordinationRecord(graph)
	if err := store.Publish(context.Background(), seed); err != nil {
		t.Fatal(err)
	}
	history := &canonicalHistory{}
	service := coordination.New(history, nil, nil)
	previous := ""
	fields := []coordination.Field{}
	for _, name := range []string{"summary", "reference", "digest", "status", "contract", "scope"} {
		fields = append(fields, coordination.Field{Name: name, Value: strings.Repeat("<", 1400)})
	}
	for n := 0; n < 6; n++ {
		record, err := service.Publish(context.Background(), graph, coordination.Input{ParentID: graph.Parent.ExecutionID, ChildID: graph.Children[0].ExecutionID, GraphRevision: 1, Kind: coordination.ContractProposal, ExpectedRevision: uint64(n), PreviousDigest: previous, Provenance: inboxProvenance(), Fields: fields})
		if err != nil {
			t.Fatal(err)
		}
		previous = record.Digest
	}
	stream := coordination.Stream{FormatVersion: 1, ParentID: graph.Parent.ExecutionID, ChildID: graph.Children[0].ExecutionID, Records: history.records}
	wire, err := coordination.EncodeStream(stream)
	if err != nil {
		t.Fatal(err)
	}
	if len(wire) <= MaxRecordBytes {
		t.Fatal("fixture does not exceed old local byte cap")
	}
	path := filepath.Join(store.root, "coordination", "v1", "project-1", coordinationName(stream.ParentID, stream.ChildID))
	if err := os.WriteFile(path, wire, 0600); err != nil {
		t.Fatal(err)
	}
	loaded, found, err := store.ReadStream(context.Background(), stream.ParentID, stream.ChildID)
	if err != nil || !found || len(loaded.Records) != 6 {
		t.Fatalf("large stream records=%d found=%t err=%v", len(loaded.Records), found, err)
	}
}
