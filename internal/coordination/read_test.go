package coordination

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
)

type readableStream struct {
	mu     sync.Mutex
	stream Stream
	reads  chan struct{}
}

func (s *readableStream) ReadStream(ctx context.Context, parent, child string) (Stream, bool, error) {
	if err := ctx.Err(); err != nil {
		return Stream{}, false, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.reads != nil {
		select {
		case s.reads <- struct{}{}:
		default:
		}
	}
	if s.stream.ParentID != parent || s.stream.ChildID != child || len(s.stream.Records) == 0 {
		return Stream{}, false, nil
	}
	copy := s.stream
	copy.Records = append([]Record(nil), s.stream.Records...)
	return copy, true, nil
}
func (s *readableStream) add(t *testing.T, kind Kind, attempt string) Record {
	t.Helper()
	s.mu.Lock()
	defer s.mu.Unlock()
	n := len(s.stream.Records)
	record := Record{FormatVersion: FormatVersion, RecordID: fmt.Sprintf("00000000-0000-4000-8000-%012d", n+100), Kind: kind, ParentID: s.stream.ParentID, ChildID: s.stream.ChildID, AttemptID: attempt, GraphRevision: 1, Revision: uint64(n + 1), Provenance: testProvenance(), Fields: []Field{kindField(kind)}, CreatedAt: time.Unix(20, 0).UTC()}
	if n > 0 {
		record.PreviousDigest = s.stream.Records[n-1].Digest
	}
	var err error
	record.Digest, err = recordDigest(record)
	if err != nil || !ValidRecord(record) {
		t.Fatalf("bad fixture: %v", err)
	}
	s.stream.Records = append(s.stream.Records, record)
	return record
}
func newReadableStream() *readableStream {
	return &readableStream{stream: Stream{FormatVersion: 1, ParentID: "00000000-0000-4000-8000-000000000001", ChildID: "00000000-0000-4000-8000-000000000002"}}
}
func streamFilter(s *readableStream) Filter {
	return Filter{ParentID: s.stream.ParentID, ChildID: s.stream.ChildID}
}

func TestReaderListShowEachKindAndFilters(t *testing.T) {
	store := newReadableStream()
	reader := Reader{Store: store}
	base := streamFilter(store)
	attempt := "00000000-0000-4000-8000-000000000003"
	kinds := []Kind{QuestionRequest, Answer, ContractProposal, ContractAcceptance, Blocker, DependencyResolution, ArtifactPublication, Progress, Result}
	for _, kind := range kinds {
		record := store.add(t, kind, attempt)
		t.Run(string(kind), func(t *testing.T) {
			filter := base
			filter.Kind = kind
			filter.AttemptID = attempt
			records, err := reader.List(context.Background(), filter)
			if err != nil || len(records) != 1 || records[0].Digest != record.Digest {
				t.Fatalf("list=%+v err=%v", records, err)
			}
			shown, err := reader.Show(context.Background(), filter, record.RecordID)
			if err != nil || shown.Digest != record.Digest {
				t.Fatalf("show=%+v err=%v", shown, err)
			}
			records[0].Fields[0].Value = "caller mutation"
			shown, err = reader.Show(context.Background(), filter, record.RecordID)
			if err != nil || shown.Fields[0].Value == "caller mutation" {
				t.Fatalf("aliased records: %+v %v", shown, err)
			}
		})
	}
	all, err := reader.List(context.Background(), base)
	if err != nil || len(all) != 9 {
		t.Fatalf("all: %d %v", len(all), err)
	}
	for _, target := range []string{"parent", "child", "attempt", "kind"} {
		t.Run("filter_"+target, func(t *testing.T) {
			filter := base
			switch target {
			case "parent":
				filter.ParentID = attempt
			case "child":
				filter.ChildID = attempt
			case "attempt":
				filter.AttemptID = base.ParentID
			case "kind":
				filter.Kind = "invalid"
			}
			records, err := reader.List(context.Background(), filter)
			if target == "kind" {
				if !errors.Is(err, ErrInvalidRecord) {
					t.Fatalf("invalid filter err=%v", err)
				}
			} else if err != nil || len(records) != 0 {
				t.Fatalf("foreign filter=%v err=%v", records, err)
			}
		})
	}
	if _, err := reader.Show(context.Background(), base, "00000000-0000-4000-8000-000000000099"); !errors.Is(err, ErrRecordNotFound) {
		t.Fatalf("unknown id=%v", err)
	}
	filter := base
	filter.Kind = Blocker
	if _, err := reader.Show(context.Background(), filter, all[0].RecordID); !errors.Is(err, ErrRecordNotFound) {
		t.Fatalf("show ignored filter=%v", err)
	}
}

func TestReaderWaitNewRecordTimeoutAndCancellation(t *testing.T) {
	store := newReadableStream()
	store.reads = make(chan struct{}, 1)
	reader := Reader{Store: store}
	filter := streamFilter(store)
	first := store.add(t, Progress, "")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	type result struct {
		records []Record
		err     error
	}
	done := make(chan result, 1)
	go func() {
		records, err := reader.Wait(ctx, filter, first.Revision, 2*time.Second)
		done <- result{records, err}
	}()
	select {
	case <-store.reads:
	case <-ctx.Done():
		t.Fatal("wait never read")
	}
	added := store.add(t, Blocker, "")
	select {
	case got := <-done:
		if got.err != nil || len(got.records) != 1 || got.records[0].Digest != added.Digest {
			t.Fatalf("wait=%+v", got)
		}
	case <-ctx.Done():
		t.Fatal("wait did not finish")
	}
	start := time.Now()
	if _, err := reader.Wait(ctx, filter, added.Revision, 30*time.Millisecond); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("timeout=%v", err)
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("unbounded wait: %s", elapsed)
	}
	childCtx, childCancel := context.WithTimeout(ctx, 3*time.Second)
	go func() {
		records, err := reader.Wait(childCtx, filter, added.Revision, time.Second)
		done <- result{records, err}
	}()
	childCancel()
	select {
	case got := <-done:
		if !errors.Is(got.err, context.Canceled) {
			t.Fatalf("cancel=%v", got.err)
		}
	case <-ctx.Done():
		t.Fatal("cancel did not finish")
	}
}
func TestReaderWaitRequiresCallerDeadlineAndShorterBound(t *testing.T) {
	reader := Reader{Store: newReadableStream()}
	filter := streamFilter(newReadableStream())
	if _, err := reader.Wait(context.Background(), filter, 0, time.Second); !errors.Is(err, ErrInvalidWait) {
		t.Fatalf("no deadline=%v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	for _, bound := range []time.Duration{0, -time.Second, time.Second, 2 * time.Second} {
		if _, err := reader.Wait(ctx, filter, 0, bound); !errors.Is(err, ErrInvalidWait) {
			t.Fatalf("bound=%s err=%v", bound, err)
		}
	}
}
func TestReaderRejectsTamperedStream(t *testing.T) {
	store := newReadableStream()
	record := store.add(t, Progress, "")
	store.stream.Records[0].Fields[0].Value = "tampered"
	reader := Reader{Store: store}
	filter := streamFilter(store)
	if _, err := reader.List(context.Background(), filter); !errors.Is(err, ErrInvalidRecord) {
		t.Fatalf("list=%v", err)
	}
	if _, err := reader.Show(context.Background(), filter, record.RecordID); !errors.Is(err, ErrInvalidRecord) {
		t.Fatalf("show=%v", err)
	}
}

func TestReaderRejectsUnencodableRecord(t *testing.T) {
	store := newReadableStream()
	store.add(t, Progress, "")
	record := &store.stream.Records[0]
	record.Kind = ContractProposal
	record.Fields = nil
	// Individually valid fields can exceed the wire-size limit in aggregate.
	for _, name := range []string{"summary", "reference", "digest", "status", "contract", "scope"} {
		record.Fields = append(record.Fields, Field{Name: name, Value: strings.Repeat("<", 4096)})
	}
	record.Digest, _ = recordDigest(*record)
	if !ValidRecord(*record) {
		t.Fatal("fixture does not isolate wire-size validation")
	}
	if _, err := (Reader{Store: store}).List(context.Background(), streamFilter(store)); !errors.Is(err, ErrInvalidRecord) {
		t.Fatalf("oversized wire accepted: %v", err)
	}
}
