package local

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/rgomids/axiom/internal/coordination"
	"github.com/rgomids/axiom/internal/strictjson"
)

type InboxDiagnostic struct {
	Sequence uint64 `json:"sequence"`
	Category string `json:"category"`
}
type inboxIngestion struct {
	Sequence    uint64            `json:"sequence"`
	Sealed      bool              `json:"sealed"`
	Diagnostics []InboxDiagnostic `json:"diagnostics"`
}

func readInboxIngestion(root *os.Root) (inboxIngestion, []byte, bool, error) {
	wire, err := readPublishedFile(root, "ingestion.json")
	if errors.Is(err, os.ErrNotExist) || errors.Is(err, ErrNotFound) {
		return inboxIngestion{}, nil, true, nil
	}
	if err != nil {
		return inboxIngestion{}, nil, false, err
	}
	var state inboxIngestion
	if strictjson.Decode(wire, &state, 16<<10) != nil || len(state.Diagnostics) > 64 || state.Sequence >= maxLocalDirectoryEntries {
		return state, nil, false, ErrUnsafe
	}
	for _, d := range state.Diagnostics {
		if d.Sequence == 0 || d.Sequence > state.Sequence || !ingestionCategory(d.Category) {
			return state, nil, false, ErrUnsafe
		}
	}
	return state, wire, false, nil
}
func ingestionCategory(c string) bool {
	switch c {
	case "quarantined_epoch", "invalid_lineage", "stale_revision", "duplicate_record", "capacity_exceeded", "missing_reference", "invalid_record":
		return true
	}
	return false
}

// Consume pins the inbox object and directory lock also used by Append. Receipt
// writes use the existing recoverable publication protocol. No rejected content
// is copied to diagnostics; stale-epoch files remain in this sealed/private inbox.
func (s *AttemptInbox) Consume(ctx context.Context, binding coordination.ChildContext, final bool, accept func(coordination.Candidate) (string, error)) error {
	if accept == nil {
		return coordination.ErrChildContext
	}
	if err := s.validate(ctx, binding); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	root, err := existingPrivateRoot(binding.Inbox)
	if err != nil {
		return err
	}
	defer root.Close()
	info, err := root.Stat(".")
	if err != nil || s.identity == nil || !os.SameFile(info, s.identity) {
		return ErrUnsafe
	}
	if err := s.inboxAnchor.verify(root); err != nil {
		return err
	}
	lock, err := lockInbox(ctx, root)
	if err != nil {
		return err
	}
	defer lock.Close()
	expectedBinding, _ := json.Marshal(binding)
	actual, err := readPrivateFileBounded(root, "binding.json", 4096)
	if err != nil || !bytes.Equal(expectedBinding, actual) {
		return coordination.ErrChildContext
	}
	state, expected, create, err := readInboxIngestion(root)
	if err != nil {
		return err
	}
	if state.Sealed {
		return nil
	}
	names, err := readDirectoryNamesBounded(root, maxLocalDirectoryEntries)
	if err != nil {
		return err
	}
	sort.Strings(names)
	sequence := uint64(1)
	for _, name := range names {
		if name == "binding.json" || name == "ingestion.json" {
			continue
		}
		if name != fmt.Sprintf("candidate-%020d.json", sequence) {
			return ErrUnsafe
		}
		current := sequence
		sequence++
		if current <= state.Sequence {
			continue
		}
		wire, err := readPrivateFileBounded(root, name, coordination.MaxRecordBytes+1024)
		if err != nil {
			return err
		}
		var candidate coordination.Candidate
		if strictjson.Decode(wire, &candidate, coordination.MaxRecordBytes+1024) != nil || candidate.Sequence != current {
			candidate = coordination.Candidate{Sequence: current}
		}
		category, err := accept(candidate)
		if err != nil {
			return err
		}
		if category != "accepted" && !ingestionCategory(category) {
			return ErrUnsafe
		}
		state.Sequence = current
		if category != "accepted" {
			if len(state.Diagnostics) == 64 {
				state.Diagnostics = state.Diagnostics[1:]
			}
			state.Diagnostics = append(state.Diagnostics, InboxDiagnostic{current, category})
		}
		next, _ := json.Marshal(state)
		if err := publishFile(ctx, root, "ingestion.json", expected, next, create, publicationHooks{}); err != nil {
			return err
		}
		expected, create = next, false
	}
	if state.Sequence >= sequence {
		return ErrUnsafe
	}
	if final {
		state.Sealed = true
		wire, _ := json.Marshal(state)
		if err := publishFile(ctx, root, "ingestion.json", expected, wire, create, publicationHooks{}); err != nil {
			return err
		}
	}
	if err := s.validate(ctx, binding); err != nil {
		return err
	}
	return s.inboxAnchor.verify(root)
}

// CoordinationFence composes the existing live lease with a freshly loaded
// attempt binding. Current must read canonical graph/ledger state and must not
// acquire this lease mutex again. The supervisor shares the lease for its saves.
type CoordinationFence struct {
	Lease   *RunLease
	Current func(context.Context) (coordination.ChildContext, error)
}

func (f CoordinationFence) WithAttempt(ctx context.Context, c coordination.ChildContext, drain func() error) error {
	if f.Lease == nil || f.Current == nil || drain == nil {
		return coordination.ErrEpochFenced
	}
	f.Lease.mu.Lock()
	defer f.Lease.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := f.Lease.validateLocked(); err != nil {
		return err
	}
	if f.Lease.parentID != c.ParentID || f.Lease.Epoch != c.Epoch {
		return coordination.ErrEpochFenced
	}
	current, err := f.Current(ctx)
	if err != nil {
		return err
	}
	if current != c || !coordination.ValidChildContext(current) {
		return coordination.ErrEpochFenced
	}
	if err := drain(); err != nil {
		return err
	}
	return f.Lease.validateLocked()
}

// Child publishers and the supervisor use separate adapter handles/processes;
// transient contention is waited on only within the caller's context.
func lockInbox(ctx context.Context, root *os.Root) (*os.File, error) {
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		lock, err := lockDirectory(root, true)
		if !errors.Is(err, ErrConflict) {
			return lock, err
		}
		timer := time.NewTimer(time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
}
