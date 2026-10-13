package local

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/rgomids/axiom/internal/coordination"
)

// AttemptInbox is bound by the trusted launcher to exactly one running attempt.
// Its private directory is outside canonical coordination storage. Candidates
// and their sequence numbers are created exclusively under a directory lock;
// ingestion must take the same lock before reading them.
type AttemptInbox struct {
	binding         coordination.ChildContext
	workspace       os.FileInfo
	workspaceAnchor anchor
	mu              sync.Mutex
	identity        os.FileInfo
	inboxAnchor     anchor
}

func NewAttemptInbox(binding coordination.ChildContext) (*AttemptInbox, error) {
	if !coordination.ValidChildContext(binding) {
		return nil, coordination.ErrChildContext
	}
	canonical, err := trustedCanonical(binding.Workspace)
	if err != nil || canonical != binding.Workspace {
		return nil, ErrUnsafe
	}
	workspace, err := os.Lstat(binding.Workspace)
	if err != nil || !workspace.IsDir() || workspace.Mode()&os.ModeSymlink != 0 {
		return nil, ErrUnsafe
	}
	workspaceRoot, chain, err := anchoredRoot(binding.Workspace, false)
	if err != nil {
		return nil, err
	}
	workspaceRoot.Close()
	canonical, err = trustedCanonical(binding.Inbox)
	if err != nil || canonical != binding.Inbox {
		return nil, ErrUnsafe
	}
	adapter := &AttemptInbox{binding: binding, workspace: workspace, workspaceAnchor: anchor{path: binding.Workspace, chain: chain}}
	if info, err := os.Lstat(binding.Inbox); err == nil {
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return nil, ErrUnsafe
		}
		root, identity, err := existingPrivateRootAnchored(binding.Inbox)
		if err != nil {
			return nil, err
		}
		root.Close()
		adapter.identity = info
		adapter.inboxAnchor = identity
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	return adapter, nil
}

func (s *AttemptInbox) validate(ctx context.Context, binding coordination.ChildContext) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || binding != s.binding || !coordination.ValidChildContext(binding) {
		return coordination.ErrChildContext
	}
	canonical, err := trustedCanonical(binding.Workspace)
	if err != nil || canonical != binding.Workspace {
		return ErrUnsafe
	}
	info, err := os.Lstat(binding.Workspace)
	if err != nil || !os.SameFile(info, s.workspace) {
		return coordination.ErrChildContext
	}
	root, err := os.OpenRoot(binding.Workspace)
	if err != nil {
		return err
	}
	defer root.Close()
	if err := s.workspaceAnchor.verify(root); err != nil {
		return err
	}
	return nil
}

func (s *AttemptInbox) Create(ctx context.Context, binding coordination.ChildContext) error {
	if err := s.validate(ctx, binding); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	parent, parentAnchor, err := existingPrivateRootAnchored(filepath.Dir(binding.Inbox))
	if err != nil {
		return err
	}
	defer parent.Close()
	name := filepath.Base(binding.Inbox)
	// Unlike privateChild, an existing attempt directory is never accepted.
	if err := mkdirPrivate(parent, name); err != nil {
		return err
	}
	root, err := existingPrivateChild(parent, name)
	if err != nil {
		return err
	}
	defer root.Close()
	s.identity, err = root.Stat(".")
	if err != nil {
		return err
	}
	s.inboxAnchor = anchor{path: binding.Inbox, chain: append(append([]os.FileInfo(nil), parentAnchor.chain...), s.identity)}
	wire, err := json.Marshal(binding)
	if err != nil {
		return err
	}
	if err := writePrivateFile(root, "binding.json", wire); err != nil {
		return err
	}
	if err := syncRoot(root); err != nil {
		return err
	}
	return s.inboxAnchor.verify(root)
}

func (s *AttemptInbox) Append(ctx context.Context, binding coordination.ChildContext, record coordination.Record) (coordination.Candidate, error) {
	if err := s.validate(ctx, binding); err != nil {
		return coordination.Candidate{}, err
	}
	wire, err := coordination.Encode(record)
	if err != nil {
		return coordination.Candidate{}, err
	}
	if record.ParentID != binding.ParentID || record.ChildID != binding.ChildID || record.AttemptID != binding.AttemptID || record.GraphRevision != binding.GraphRevision || record.Revision != 1 || record.PreviousDigest != "" {
		return coordination.Candidate{}, coordination.ErrChildContext
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	root, err := existingPrivateRoot(binding.Inbox)
	if err != nil {
		return coordination.Candidate{}, err
	}
	defer root.Close()
	info, err := root.Stat(".")
	if err != nil || s.identity == nil || !os.SameFile(info, s.identity) {
		return coordination.Candidate{}, ErrUnsafe
	}
	if err := s.inboxAnchor.verify(root); err != nil {
		return coordination.Candidate{}, err
	}
	var lock *os.File
	for {
		lock, err = lockDirectory(root, true)
		if !errors.Is(err, ErrConflict) {
			break
		}
		timer := time.NewTimer(time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return coordination.Candidate{}, ctx.Err()
		case <-timer.C:
		}
	}
	if err != nil {
		return coordination.Candidate{}, err
	}
	defer lock.Close()
	expected, _ := json.Marshal(binding)
	actual, err := readPrivateFileBounded(root, "binding.json", 4096)
	if err != nil || !bytes.Equal(actual, expected) {
		return coordination.Candidate{}, coordination.ErrChildContext
	}
	// The files are the sequence journal, so reopening the adapter cannot reuse a
	// sequence. Partial or forged journal entries fail closed instead of skipping.
	names, err := readDirectoryNamesBounded(root, maxLocalDirectoryEntries)
	if err != nil {
		return coordination.Candidate{}, err
	}
	sort.Strings(names)
	sequence := uint64(1)
	for _, name := range names {
		if err := ctx.Err(); err != nil {
			return coordination.Candidate{}, err
		}
		if name == "binding.json" {
			continue
		}
		if name != fmt.Sprintf("candidate-%020d.json", sequence) {
			return coordination.Candidate{}, ErrUnsafe
		}
		previous, err := readPrivateFileBounded(root, name, coordination.MaxRecordBytes+1024)
		if err != nil {
			return coordination.Candidate{}, err
		}
		var candidate coordination.Candidate
		if json.Unmarshal(previous, &candidate) != nil || candidate.Sequence != sequence || candidate.Epoch != binding.Epoch || !coordination.ValidRecord(candidate.Record) || candidate.Record.ParentID != binding.ParentID || candidate.Record.ChildID != binding.ChildID || candidate.Record.AttemptID != binding.AttemptID || candidate.Record.GraphRevision != binding.GraphRevision || candidate.Record.Revision != 1 || candidate.Record.PreviousDigest != "" {
			return coordination.Candidate{}, ErrUnsafe
		}
		canonical, _ := json.Marshal(candidate)
		if !bytes.Equal(canonical, previous) {
			return coordination.Candidate{}, ErrUnsafe
		}
		sequence++
	}
	if sequence >= maxLocalDirectoryEntries {
		return coordination.Candidate{}, coordination.ErrCapacity
	}
	if err := ctx.Err(); err != nil {
		return coordination.Candidate{}, err
	}
	detached, err := coordination.Decode(wire)
	if err != nil {
		return coordination.Candidate{}, err
	}
	candidate := coordination.Candidate{Sequence: sequence, Epoch: binding.Epoch, Record: detached}
	payload, err := json.Marshal(candidate)
	if err != nil {
		return coordination.Candidate{}, err
	}
	if err := writePrivateFile(root, fmt.Sprintf("candidate-%020d.json", sequence), payload); err != nil {
		return coordination.Candidate{}, err
	}
	if err := syncRoot(root); err != nil {
		return coordination.Candidate{}, err
	}
	if err := s.validate(ctx, binding); err != nil {
		return coordination.Candidate{}, err
	}
	if err := s.inboxAnchor.verify(root); err != nil {
		return coordination.Candidate{}, err
	}
	visible, err := os.Lstat(binding.Inbox)
	if err != nil || !os.SameFile(info, visible) {
		return coordination.Candidate{}, ErrUnsafe
	}
	return candidate, nil
}
