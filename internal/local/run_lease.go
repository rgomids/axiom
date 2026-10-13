package local

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"os"
	"path/filepath"
	"sync"

	"github.com/rgomids/axiom/internal/strictjson"
)

// RunLeaseStore is separate from graph CAS locks. Its per-graph OS lock must
// always be acquired first (ADR-0007 / ADR-0025); it is never waited upon.
type RunLeaseStore struct {
	root  string
	hooks publicationHooks
}

type runLeaseRecord struct {
	FormatVersion int    `json:"formatVersion"`
	ProjectID     string `json:"projectId"`
	ParentID      string `json:"parentId"`
	Epoch         uint64 `json:"epoch"`
}

// RunLease holds the directory object containing the epoch record, not a
// replaceable epoch file. Anchors detect directory/lock replacement and links.
type RunLease struct {
	mu                  sync.Mutex
	root                *os.Root
	lock                *os.File
	identity            anchor
	Epoch               uint64
	projectID, parentID string
	epochIdentity       os.FileInfo
	stateRoot           string
}

func NewRunLeaseStore(root string) (RunLeaseStore, error) {
	if !filepath.IsAbs(root) || filepath.Clean(root) == string(filepath.Separator) {
		return RunLeaseStore{}, ErrUnsafe
	}
	canonical, err := trustedCanonical(root)
	if err != nil {
		return RunLeaseStore{}, err
	}
	return RunLeaseStore{root: canonical}, nil
}

func (s RunLeaseStore) open(projectID, parentID string, create bool) (*os.Root, anchor, error) {
	if !validGraphAddress(projectID, parentID) || projectID == "." || projectID == ".." || !platformEntryName(projectID) {
		return nil, anchor{}, ErrUnsafe
	}
	path := filepath.Join(s.root, "run-leases", "v1", projectID, graphName(parentID))
	if create {
		return privateRootAnchored(path)
	}
	return existingPrivateRootAnchored(path)
}

func decodeRunLease(wire []byte) (runLeaseRecord, error) {
	var r runLeaseRecord
	if strictjson.Decode(wire, &r, MaxRecordBytes) != nil || r.FormatVersion != 1 || !validGraphAddress(r.ProjectID, r.ParentID) || r.Epoch == 0 {
		return r, ErrUnsafe
	}
	return r, nil
}

func (s RunLeaseStore) Acquire(ctx context.Context, projectID, parentID string, minimumEpoch uint64) (*RunLease, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	root, identity, err := s.open(projectID, parentID, true)
	if err != nil {
		return nil, err
	}
	lock, err := lockDirectory(root, true)
	if err != nil {
		root.Close()
		return nil, err
	}
	lease := &RunLease{root: root, lock: lock, identity: identity, projectID: projectID, parentID: parentID, stateRoot: s.root}
	fail := func(err error) (*RunLease, error) { lease.Close(); return nil, err }
	if err := identity.verify(root); err != nil {
		return fail(err)
	}
	prior, err := readPublishedFile(root, "epoch.json")
	epoch := minimumEpoch
	create := errors.Is(err, ErrNotFound) || errors.Is(err, os.ErrNotExist)
	if err != nil && !create {
		return fail(err)
	}
	if !create {
		r, e := decodeRunLease(prior)
		if e != nil || r.ProjectID != projectID || r.ParentID != parentID {
			return fail(ErrUnsafe)
		}
		if r.Epoch < minimumEpoch {
			return fail(ErrConflict)
		}
		if r.Epoch > epoch {
			epoch = r.Epoch
		}
	}
	if epoch == math.MaxUint64 {
		return fail(ErrConflict)
	}
	epoch++
	wire, err := json.Marshal(runLeaseRecord{1, projectID, parentID, epoch})
	if err != nil {
		return fail(err)
	}
	wire = append(wire, '\n')
	if err := publishFile(ctx, root, "epoch.json", prior, wire, create, s.hooks); err != nil {
		return fail(err)
	}
	lease.Epoch = epoch
	lease.epochIdentity, err = root.Lstat("epoch.json")
	if err != nil {
		return fail(err)
	}
	if err := lease.Validate(); err != nil {
		return fail(err)
	}
	return lease, nil
}

// Probe takes the same exclusive lock without changing the epoch. Missing state
// is free; unsafe/interrupted publication is an error, never proof of freedom.
func (s RunLeaseStore) Probe(ctx context.Context, projectID, parentID string) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	root, identity, err := s.open(projectID, parentID, false)
	if errors.Is(err, ErrNotFound) || errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	defer root.Close()
	if err := identity.verify(root); err != nil {
		return false, err
	}
	lock, err := lockDirectory(root, true)
	if errors.Is(err, ErrConflict) {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	defer lock.Close()
	if err := identity.verify(root); err != nil {
		return false, err
	}
	wire, err := readPublishedFile(root, "epoch.json")
	if err != nil {
		return false, err
	}
	r, err := decodeRunLease(wire)
	if err != nil || r.ProjectID != projectID || r.ParentID != parentID {
		return false, ErrUnsafe
	}
	return false, nil
}

func (l *RunLease) Validate() error {
	if l == nil {
		return ErrConflict
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.validateLocked()
}

// validateLocked is used while a save pins this handle through its commit point,
// preventing concurrent Close from releasing the lease between fence and rename.
func (l *RunLease) validateLocked() error {
	if l.lock == nil {
		return ErrConflict
	}
	if err := l.identity.verify(l.root); err != nil {
		return err
	}
	info, err := l.root.Lstat("epoch.json")
	if err != nil || l.epochIdentity == nil || !os.SameFile(info, l.epochIdentity) {
		return ErrReplaced
	}
	wire, err := readPublishedFile(l.root, "epoch.json")
	if err != nil {
		return err
	}
	r, err := decodeRunLease(wire)
	if err != nil || r.Epoch != l.Epoch || r.ProjectID != l.projectID || r.ParentID != l.parentID {
		return ErrConflict
	}
	return nil
}

func (l *RunLease) Close() error {
	if l == nil {
		return nil
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.lock == nil {
		return nil
	}
	err := l.lock.Close()
	l.lock = nil
	return errors.Join(err, l.root.Close())
}
