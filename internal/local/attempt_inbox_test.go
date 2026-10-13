package local

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/rgomids/axiom/internal/coordination"
)

type fakeChild struct {
	binding coordination.ChildContext
	err     error
}

func (f fakeChild) ResolveChild(context.Context, string) (coordination.ChildContext, error) {
	return f.binding, f.err
}

func inboxFixture(t *testing.T) (*AttemptInbox, coordination.ChildContext, coordination.InboxPublisher) {
	t.Helper()
	base := privateTestRoot(t)
	base, err := trustedCanonical(base)
	if err != nil {
		t.Fatal(err)
	}
	workspace := filepath.Join(base, "workspace")
	if err := os.Mkdir(workspace, 0700); err != nil {
		t.Fatal(err)
	}
	binding := coordination.ChildContext{Workspace: workspace, Inbox: filepath.Join(base, "attempt"), ParentID: opaqueID(1), ChildID: opaqueID(2), AttemptID: opaqueID(3), GraphRevision: 1, Epoch: 1, Running: true}
	inbox, err := NewAttemptInbox(binding)
	if err != nil {
		t.Fatal(err)
	}
	publisher := coordination.InboxPublisher{Resolver: fakeChild{binding: binding}, Inbox: inbox}
	if err := publisher.Create(context.Background(), binding); err != nil {
		t.Fatal(err)
	}
	return inbox, binding, publisher
}

func inboxFields(kind coordination.Kind) []coordination.Field {
	fields := map[coordination.Kind]coordination.Field{
		coordination.QuestionRequest:      {Name: "question", Value: "Which contract?"},
		coordination.Answer:               {Name: "answer", Value: "Use v1"},
		coordination.ContractProposal:     {Name: "contract", Value: "v1"},
		coordination.ContractAcceptance:   {Name: "decision", Value: "accepted"},
		coordination.Blocker:              {Name: "code", Value: "missing_input"},
		coordination.DependencyResolution: {Name: "resolution", Value: "artifact:1"},
		coordination.ArtifactPublication:  {Name: "artifact_id", Value: "artifact:1"},
		coordination.Progress:             {Name: "phase", Value: "testing"},
		coordination.Result:               {Name: "outcome", Value: "succeeded"},
	}
	return []coordination.Field{fields[kind]}
}
func inboxProvenance() coordination.Provenance {
	return coordination.Provenance{Product: "fake-child", Version: "dev", Revision: "test", SourceState: "clean"}
}

func TestAttemptInboxEveryKindAndExclusiveCreate(t *testing.T) {
	inbox, binding, publisher := inboxFixture(t)
	if err := inbox.Create(context.Background(), binding); !errors.Is(err, os.ErrExist) {
		t.Fatalf("duplicate create: %v", err)
	}
	kinds := []coordination.Kind{coordination.QuestionRequest, coordination.Answer, coordination.ContractProposal, coordination.ContractAcceptance, coordination.Blocker, coordination.DependencyResolution, coordination.ArtifactPublication, coordination.Progress, coordination.Result}
	for i, kind := range kinds {
		t.Run(string(kind), func(t *testing.T) {
			candidate, err := publisher.Publish(context.Background(), binding.Workspace, kind, inboxFields(kind), nil, inboxProvenance())
			if err != nil || candidate.Sequence != uint64(i+1) || candidate.Epoch != binding.Epoch || candidate.Record.AttemptID != binding.AttemptID {
				t.Fatalf("candidate=%+v err=%v", candidate, err)
			}
			wire, err := os.ReadFile(filepath.Join(binding.Inbox, fmtCandidate(uint64(i+1))))
			if err != nil {
				t.Fatal(err)
			}
			var stored coordination.Candidate
			if err := json.Unmarshal(wire, &stored); err != nil || stored.Record.Digest != candidate.Record.Digest {
				t.Fatalf("stored=%+v err=%v", stored, err)
			}
		})
	}
	reopened, err := NewAttemptInbox(binding)
	if err != nil {
		t.Fatal(err)
	}
	publisher.Inbox = reopened
	candidate, err := publisher.Publish(context.Background(), binding.Workspace, coordination.Progress, inboxFields(coordination.Progress), nil, inboxProvenance())
	if err != nil || candidate.Sequence != 10 {
		t.Fatalf("reopened sequence=%d err=%v", candidate.Sequence, err)
	}
	// An inbox is never consulted by canonical read use cases.
	store, err := NewCoordinationStore(filepath.Join(filepath.Dir(binding.Inbox), "state"), "project-1")
	if err != nil {
		t.Fatal(err)
	}
	records, err := (coordination.Reader{Store: store}).List(context.Background(), coordination.Filter{ParentID: binding.ParentID, ChildID: binding.ChildID})
	if err != nil || len(records) != 0 {
		t.Fatalf("inbox leaked into reads: %v %v", records, err)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(binding.Inbox), "state")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("read created canonical state: %v", err)
	}
}

func TestAttemptInboxRejectsInvalidPublish(t *testing.T) {
	tests := []string{"no child context", "human shell", "foreign workspace", "empty attempt", "another inbox", "another attempt", "stale epoch", "oversized", "credential", "unknown kind", "unknown field"}
	for _, name := range tests {
		t.Run(name, func(t *testing.T) {
			_, binding, publisher := inboxFixture(t)
			resolved := binding
			workspace := binding.Workspace
			fields := inboxFields(coordination.Progress)
			kind := coordination.Progress
			switch name {
			case "no child context":
				publisher.Resolver = nil
			case "human shell":
				resolved.Running = false
			case "foreign workspace":
				workspace = filepath.Join(filepath.Dir(binding.Workspace), "foreign")
			case "empty attempt":
				resolved.AttemptID = ""
			case "another inbox":
				resolved.Inbox = filepath.Join(filepath.Dir(binding.Inbox), "other")
			case "another attempt":
				resolved.AttemptID = opaqueID(4)
			case "stale epoch":
				resolved.Epoch++
			case "oversized":
				fields[0].Value = strings.Repeat("x", 4097)
			case "credential":
				fields[0].Value = "pass" + "word=synthetic-value"
			case "unknown kind":
				kind = "unknown"
			case "unknown field":
				fields[0].Name = "authority"
			}
			if publisher.Resolver != nil {
				publisher.Resolver = fakeChild{binding: resolved}
			}
			if _, err := publisher.Publish(context.Background(), workspace, kind, fields, nil, inboxProvenance()); err == nil {
				t.Fatal("invalid publish accepted")
			}
			names, err := os.ReadDir(binding.Inbox)
			if err != nil || len(names) != 1 || names[0].Name() != "binding.json" {
				t.Fatalf("rejection wrote files: %v %v", names, err)
			}
		})
	}
}

func TestAttemptInboxConcurrentAdaptersAllocateUniqueSequence(t *testing.T) {
	_, binding, publisher := inboxFixture(t)
	const count = 12
	var wg sync.WaitGroup
	results := make(chan coordination.Candidate, count)
	failures := make(chan error, count)
	for i := 0; i < count; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			adapter, err := NewAttemptInbox(binding)
			if err != nil {
				failures <- err
				return
			}
			p := publisher
			p.Inbox = adapter
			ctx, cancel := context.WithTimeout(context.Background(), testInboxTimeout)
			defer cancel()
			c, err := p.Publish(ctx, binding.Workspace, coordination.Progress, inboxFields(coordination.Progress), nil, inboxProvenance())
			if err != nil {
				failures <- err
				return
			}
			results <- c
		}()
	}
	wg.Wait()
	close(results)
	close(failures)
	for err := range failures {
		t.Error(err)
	}
	seen := map[uint64]bool{}
	for c := range results {
		if seen[c.Sequence] {
			t.Errorf("duplicate %d", c.Sequence)
		}
		seen[c.Sequence] = true
	}
	for n := uint64(1); n <= count; n++ {
		if !seen[n] {
			t.Errorf("missing %d", n)
		}
	}
}

func TestAttemptInboxRejectsReplacementAndTamperedJournal(t *testing.T) {
	for _, target := range []string{"workspace", "inbox", "binding", "journal"} {
		t.Run(target, func(t *testing.T) {
			_, binding, publisher := inboxFixture(t)
			switch target {
			case "workspace", "inbox":
				path := binding.Workspace
				if target == "inbox" {
					path = binding.Inbox
				}
				if err := os.Rename(path, path+"-old"); err != nil {
					t.Fatal(err)
				}
				if err := os.Mkdir(path, 0700); err != nil {
					t.Fatal(err)
				}
			case "binding":
				if err := os.WriteFile(filepath.Join(binding.Inbox, "binding.json"), []byte("{}"), 0600); err != nil {
					t.Fatal(err)
				}
			case "journal":
				if err := os.WriteFile(filepath.Join(binding.Inbox, fmtCandidate(2)), []byte("{}"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := publisher.Publish(context.Background(), binding.Workspace, coordination.Progress, inboxFields(coordination.Progress), nil, inboxProvenance()); err == nil {
				t.Fatal("tampering accepted")
			}
		})
	}
}

const testInboxTimeout = 5 * time.Second

func fmtCandidate(sequence uint64) string { return fmt.Sprintf("candidate-%020d.json", sequence) }

func TestAttemptInboxSequenceIsPerAttempt(t *testing.T) {
	_, binding, first := inboxFixture(t)
	other := binding
	other.AttemptID = opaqueID(4)
	other.Inbox = filepath.Join(filepath.Dir(binding.Inbox), "other-attempt")
	adapter, err := NewAttemptInbox(other)
	if err != nil {
		t.Fatal(err)
	}
	second := coordination.InboxPublisher{Resolver: fakeChild{binding: other}, Inbox: adapter}
	if err := second.Create(context.Background(), other); err != nil {
		t.Fatal(err)
	}
	for index, publisher := range []coordination.InboxPublisher{first, second, first, second} {
		child, _ := publisher.Resolver.ResolveChild(context.Background(), binding.Workspace)
		candidate, err := publisher.Publish(context.Background(), child.Workspace, coordination.Progress, inboxFields(coordination.Progress), nil, inboxProvenance())
		if err != nil {
			t.Fatal(err)
		}
		if candidate.Sequence != uint64(index/2+1) || candidate.Record.AttemptID != child.AttemptID {
			t.Fatalf("wrong attempt sequence: %+v", candidate)
		}
	}
	for _, child := range []coordination.ChildContext{binding, other} {
		entries, err := os.ReadDir(child.Inbox)
		if err != nil || len(entries) != 3 {
			t.Fatalf("attempt entries=%v err=%v", entries, err)
		}
	}
}
