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

func TestCoordinationReadStreamCanonicalOnlyAndTamper(t *testing.T) {
	_, binding, publisher := inboxFixture(t)
	store, err := NewCoordinationStore(filepath.Join(filepath.Dir(binding.Inbox), "state"), "project-1")
	if err != nil {
		t.Fatal(err)
	}
	candidate, err := publisher.Publish(context.Background(), binding.Workspace, coordination.Progress, inboxFields(coordination.Progress), nil, inboxProvenance())
	if err != nil {
		t.Fatal(err)
	}
	graph := localGraphFixture(t)
	service := coordination.New(store, nil, nil)
	first, err := service.Publish(context.Background(), graph, coordination.Input{Kind: coordination.Progress, ParentID: binding.ParentID, ChildID: binding.ChildID, GraphRevision: 1, Fields: inboxFields(coordination.Progress), Provenance: inboxProvenance()})
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.Publish(context.Background(), graph, coordination.Input{Kind: coordination.Blocker, ParentID: binding.ParentID, ChildID: binding.ChildID, GraphRevision: 1, ExpectedRevision: 1, PreviousDigest: first.Digest, Fields: inboxFields(coordination.Blocker), Provenance: inboxProvenance()})
	if err != nil {
		t.Fatal(err)
	}
	stream, found, err := store.ReadStream(context.Background(), binding.ParentID, binding.ChildID)
	if err != nil || !found || len(stream.Records) != 2 || stream.Records[1].Digest != second.Digest {
		t.Fatalf("stream=%+v found=%t err=%v", stream, found, err)
	}
	reader := coordination.Reader{Store: store}
	filter := coordination.Filter{ParentID: binding.ParentID, ChildID: binding.ChildID}
	if _, err := reader.Show(context.Background(), filter, candidate.Record.RecordID); !errors.Is(err, coordination.ErrRecordNotFound) {
		t.Fatalf("candidate visible: %v", err)
	}
	stream.Records[0].Fields[0].Value = "caller mutation"
	shown, err := reader.Show(context.Background(), filter, first.RecordID)
	if err != nil || shown.Digest != first.Digest {
		t.Fatalf("mutated stored stream=%+v %v", shown, err)
	}
	path := filepath.Join(store.root, "coordination", "v1", "project-1", coordinationName(binding.ParentID, binding.ChildID))
	wire, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	wire = bytes.Replace(wire, []byte(first.Digest), bytes.Repeat([]byte("0"), 64), 1)
	if err := os.WriteFile(path, wire, 0600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.ReadStream(context.Background(), binding.ParentID, binding.ChildID); !errors.Is(err, ErrUnsafe) {
		t.Fatalf("tamper read=%v", err)
	}
	if _, err := reader.List(context.Background(), filter); !errors.Is(err, ErrUnsafe) {
		t.Fatalf("tamper list=%v", err)
	}
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
