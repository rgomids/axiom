package local

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/rgomids/axiom/internal/coordination"
	"github.com/rgomids/axiom/internal/executiongraph"
)

type fakeIngestionFence struct {
	mu       sync.Mutex
	epoch    uint64
	terminal bool
}

func (f *fakeIngestionFence) WithAttempt(_ context.Context, c coordination.ChildContext, fn func() error) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.terminal || c.Epoch != f.epoch {
		return coordination.ErrEpochFenced
	}
	return fn()
}
func ingestionFixture(t *testing.T) (*AttemptInbox, coordination.ChildContext, coordination.InboxPublisher, coordination.Ingestor, *fakeIngestionFence, CoordinationStore) {
	t.Helper()
	inbox, c, publisher := inboxFixture(t)
	// Start the synthetic current attempt at epoch 2 so epoch 1 is a valid
	// previous-epoch candidate, rather than a malformed zero epoch.
	c.Epoch = 2
	inbox.binding = c
	bindingWire, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(c.Inbox, "binding.json"), bindingWire, 0600); err != nil {
		t.Fatal(err)
	}
	publisher.Resolver = fakeChild{binding: c}
	store, err := NewCoordinationStore(filepath.Join(filepath.Dir(c.Inbox), "state"), "project-1")
	if err != nil {
		t.Fatal(err)
	}
	fence := &fakeIngestionFence{epoch: c.Epoch}
	ingestor := coordination.Ingestor{Service: coordination.New(store, nil, nil), Reader: store, Inbox: inbox, Fence: fence, MaximumAttempts: 2, ReferenceChildren: []string{c.ChildID}}
	return inbox, c, publisher, ingestor, fence, store
}
func TestIngestCrossRuntimeQuestionAnswerAndBlocker(t *testing.T) {
	_, c, publisher, ingestor, _, store := ingestionFixture(t)
	question, err := publisher.Publish(context.Background(), c.Workspace, coordination.QuestionRequest, inboxFields(coordination.QuestionRequest), nil, coordination.Provenance{Product: "fake-codex", Version: "test", Revision: "test", SourceState: "clean"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ingestor.Drain(context.Background(), c, false); err != nil {
		t.Fatal(err)
	}
	base := privateTestRoot(t)
	base, err = trustedCanonical(base)
	if err != nil {
		t.Fatal(err)
	}
	workspace := filepath.Join(base, "workspace")
	if err := os.Mkdir(workspace, 0700); err != nil {
		t.Fatal(err)
	}
	other := c
	other.ChildID = opaqueID(4)
	other.AttemptID = opaqueID(5)
	other.Workspace = workspace
	other.Inbox = filepath.Join(base, "inbox")
	inbox, err := NewAttemptInbox(other)
	if err != nil {
		t.Fatal(err)
	}
	child := coordination.InboxPublisher{Resolver: fakeChild{binding: other}, Inbox: inbox}
	if err := child.Create(context.Background(), other); err != nil {
		t.Fatal(err)
	}
	_, err = child.Publish(context.Background(), other.Workspace, coordination.Answer, []coordination.Field{{Name: "answer", Value: "Use v1"}, {Name: "question_reference", Value: question.Record.RecordID}}, nil, coordination.Provenance{Product: "fake-claude", Version: "test", Revision: "test", SourceState: "clean"})
	if err != nil {
		t.Fatal(err)
	}
	otherIngestor := ingestor
	otherIngestor.Inbox = inbox
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for _, work := range []struct {
		i coordination.Ingestor
		c coordination.ChildContext
	}{{ingestor, c}, {otherIngestor, other}} {
		wg.Add(1)
		go func() { defer wg.Done(); _, err := work.i.Drain(context.Background(), work.c, false); errs <- err }()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	records, err := (coordination.Reader{Store: store}).List(context.Background(), coordination.Filter{ParentID: c.ParentID, ChildID: other.ChildID})
	if err != nil || len(records) != 1 || records[0].Kind != coordination.Answer {
		t.Fatal(records, err)
	}
	_, err = publisher.Publish(context.Background(), c.Workspace, coordination.Blocker, []coordination.Field{{Name: "code", Value: "missing_input"}, {Name: "required_action", Value: "Supply contract"}}, nil, inboxProvenance())
	if err != nil {
		t.Fatal(err)
	}
	status, reason, ref, err := ingestor.Classify(context.Background(), c, executiongraph.ProcessResult{ExitCode: 0}, false)
	if err != nil || status != executiongraph.AttemptBlocked || reason != executiongraph.BlockerPublished || ref == "" {
		t.Fatal(status, reason, ref, err)
	}
	drained, err := ingestor.Drain(context.Background(), c, true)
	if err != nil || len(drained) != 2 || drained[1].Digest != ref {
		t.Fatal("drained set differs", drained, err)
	}
	if _, err := publisher.Publish(context.Background(), c.Workspace, coordination.Result, inboxFields(coordination.Result), nil, inboxProvenance()); !errors.Is(err, coordination.ErrAttemptSealed) {
		t.Fatal("late publish", err)
	}
}
func TestIngestRejectsCandidatesAndReportsSafeCategories(t *testing.T) {
	for _, name := range []string{"forged-lineage", "stale-graph-revision", "stale-revision", "previous-epoch", "missing-question", "missing-contract", "after-terminal"} {
		t.Run(name, func(t *testing.T) {
			_, c, publisher, ingestor, fence, store := ingestionFixture(t)
			kind := coordination.Progress
			fields := inboxFields(kind)
			if name == "missing-question" {
				kind = coordination.Answer
				fields = []coordination.Field{{Name: "answer", Value: "v1"}, {Name: "question_reference", Value: opaqueID(99)}}
			}
			if name == "missing-contract" {
				kind = coordination.ContractAcceptance
				fields = []coordination.Field{{Name: "decision", Value: "accepted"}, {Name: "contract_reference", Value: opaqueID(99)}}
			}
			candidate, err := publisher.Publish(context.Background(), c.Workspace, kind, fields, nil, inboxProvenance())
			if err != nil {
				t.Fatal(err)
			}
			switch name {
			case "forged-lineage":
				candidate.Record.ParentID = opaqueID(99)
			case "stale-graph-revision":
				candidate.Record.GraphRevision++
			case "stale-revision":
				candidate.Record.Revision = 2
				candidate.Record.PreviousDigest = candidate.Record.Digest
			case "previous-epoch":
				candidate.Epoch = c.Epoch - 1
			case "after-terminal":
				fence.terminal = true
			}
			// Forge exact valid record bytes through the existing Service for the two
			// lineage/chain cases, avoiding an invalid digest masking those checks.
			if name == "forged-lineage" || name == "stale-revision" || name == "stale-graph-revision" {
				candidate.Record.Digest = coordinationTestDigest(candidate.Record)
			}
			payload, _ := json.Marshal(candidate)
			if err := os.WriteFile(filepath.Join(c.Inbox, fmtCandidate(1)), payload, 0600); err != nil {
				t.Fatal(err)
			}
			records, err := ingestor.Drain(context.Background(), c, true)
			if name == "after-terminal" {
				if !errors.Is(err, coordination.ErrEpochFenced) {
					t.Fatal(err)
				}
			} else if err != nil {
				t.Fatal(err)
			}
			if len(records) != 0 {
				t.Fatal("rejected candidate classified", records)
			}
			_, found, err := store.ReadStream(context.Background(), c.ParentID, c.ChildID)
			if err != nil || found {
				t.Fatal("rejected candidate canonical", err)
			}
			if name != "after-terminal" {
				root, err := existingPrivateRoot(c.Inbox)
				if err != nil {
					t.Fatal(err)
				}
				defer root.Close()
				state, _, _, err := readInboxIngestion(root)
				if err != nil || len(state.Diagnostics) != 1 {
					t.Fatal(state, err)
				}
				want := map[string]string{"forged-lineage": "invalid_lineage", "stale-graph-revision": "invalid_lineage", "stale-revision": "stale_revision", "previous-epoch": "quarantined_epoch", "missing-question": "missing_reference", "missing-contract": "missing_reference"}[name]
				if state.Diagnostics[0].Category != want {
					t.Fatal(state.Diagnostics)
				}
			}
		})
	}
}
func TestIngestCapacityReserveAndCrashReplay(t *testing.T) {
	inbox, c, publisher, ingestor, _, store := ingestionFixture(t)
	ingestor.MaximumAttempts = 10 // six per attempt, one reserved terminal slot
	for n := 0; n < 7; n++ {
		if _, err := publisher.Publish(context.Background(), c.Workspace, coordination.Progress, inboxFields(coordination.Progress), nil, inboxProvenance()); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := publisher.Publish(context.Background(), c.Workspace, coordination.Result, inboxFields(coordination.Result), nil, inboxProvenance()); err != nil {
		t.Fatal(err)
	}
	records, err := ingestor.Drain(context.Background(), c, false)
	if err != nil || len(records) != 6 {
		t.Fatal(records, err)
	}
	root, err := existingPrivateRoot(c.Inbox)
	if err != nil {
		t.Fatal(err)
	}
	state, _, _, err := readInboxIngestion(root)
	root.Close()
	if err != nil || len(state.Diagnostics) != 2 {
		t.Fatal(state, err)
	}
	// Simulate crash before receipt persistence; canonical publication survives.
	if err := os.Remove(filepath.Join(c.Inbox, "ingestion.json")); err != nil {
		t.Fatal(err)
	}
	reopened, err := NewAttemptInbox(c)
	if err != nil {
		t.Fatal(err)
	}
	ingestor.Inbox = reopened
	records, err = ingestor.Drain(context.Background(), c, true)
	if err != nil || len(records) != 6 {
		t.Fatal("replay duplicated", records, err)
	}
	stream, _, err := store.ReadStream(context.Background(), c.ParentID, c.ChildID)
	if err != nil || len(stream.Records) != 6 {
		t.Fatal(stream, err)
	}
	publisher.Inbox = reopened
	if _, err := publisher.Publish(context.Background(), c.Workspace, coordination.Progress, inboxFields(coordination.Progress), nil, inboxProvenance()); !errors.Is(err, coordination.ErrAttemptSealed) {
		t.Fatal("reopened seal lost", err)
	}
	status, reason, ref, err := ingestor.Classify(context.Background(), c, executiongraph.ProcessResult{}, false)
	if err != nil || status != executiongraph.AttemptSucceeded || reason != executiongraph.ResultRecorded || ref != records[len(records)-1].Digest {
		t.Fatal(status, reason, ref, err)
	}
	if err := inbox.Consume(context.Background(), c, true, func(coordination.Candidate) (string, error) { t.Fatal("sealed inbox callback"); return "", nil }); err != nil {
		t.Fatal(err)
	}
}

func coordinationTestDigest(r coordination.Record) string {
	r.Digest = ""
	wire, _ := json.Marshal(r)
	digest := sha256.Sum256(wire)
	return hex.EncodeToString(digest[:])
}

func TestCoordinationFenceLiveLeaseAndTerminalState(t *testing.T) {
	_, c, publisher, ingestor, _, _ := ingestionFixture(t)
	leaseStore, err := NewRunLeaseStore(filepath.Join(filepath.Dir(c.Inbox), "leases"))
	if err != nil {
		t.Fatal(err)
	}
	lease, err := leaseStore.Acquire(context.Background(), "project-1", c.ParentID, c.Epoch-1)
	if err != nil {
		t.Fatal(err)
	}
	defer lease.Close()
	current := c
	fence := CoordinationFence{Lease: lease, Current: func(context.Context) (coordination.ChildContext, error) { return current, nil }}
	ingestor.Fence = fence
	if _, err := publisher.Publish(context.Background(), c.Workspace, coordination.Blocker, inboxFields(coordination.Blocker), nil, inboxProvenance()); err != nil {
		t.Fatal(err)
	}
	status, reason, _, err := ingestor.Classify(context.Background(), c, executiongraph.ProcessResult{}, false)
	if err != nil || status != executiongraph.AttemptBlocked || reason != executiongraph.BlockerPublished {
		t.Fatal(status, reason, err)
	}
	current.Running = false // the terminal CAS has committed
	if _, err := ingestor.Drain(context.Background(), c, true); !errors.Is(err, coordination.ErrEpochFenced) {
		t.Fatal("terminal state accepted", err)
	}
	current = c
	wrong := c
	wrong.Epoch++
	if err := fence.WithAttempt(context.Background(), wrong, func() error { t.Fatal("wrong epoch callback"); return nil }); !errors.Is(err, coordination.ErrEpochFenced) {
		t.Fatal(err)
	}
	if err := lease.Close(); err != nil {
		t.Fatal(err)
	}
	if err := fence.WithAttempt(context.Background(), c, func() error { t.Fatal("closed lease callback"); return nil }); err == nil {
		t.Fatal("closed lease admitted")
	}
}

func TestFinalDrainRacesPublishWithoutLateCanonicalRecord(t *testing.T) {
	_, c, publisher, ingestor, _, store := ingestionFixture(t)
	start := make(chan struct{})
	published := make(chan error, 1)
	go func() {
		<-start
		_, err := publisher.Publish(context.Background(), c.Workspace, coordination.Result, inboxFields(coordination.Result), nil, inboxProvenance())
		published <- err
	}()
	close(start)
	records, err := ingestor.Drain(context.Background(), c, true)
	if err != nil {
		t.Fatal(err)
	}
	publishErr := <-published
	if publishErr != nil && !errors.Is(publishErr, coordination.ErrAttemptSealed) {
		t.Fatal(publishErr)
	}
	stream, found, err := store.ReadStream(context.Background(), c.ParentID, c.ChildID)
	if err != nil {
		t.Fatal(err)
	}
	if publishErr == nil {
		if !found || len(records) != 1 || len(stream.Records) != 1 || records[0].Digest != stream.Records[0].Digest {
			t.Fatal("accepted write escaped final drain")
		}
	} else if found || len(records) != 0 {
		t.Fatal("sealed write became canonical")
	}
}
