package local

import (
	"context"
	"crypto/sha256"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/rgomids/axiom/internal/capabilityproof"
)

type CapabilityStore struct {
	root  string
	hooks publicationHooks
}

func NewCapabilityStore(root string) (CapabilityStore, error) {
	if !filepath.IsAbs(root) || filepath.Clean(root) == string(filepath.Separator) {
		return CapabilityStore{}, ErrUnsafe
	}
	canonical, err := trustedCanonical(root)
	if err != nil {
		return CapabilityStore{}, err
	}
	return CapabilityStore{root: canonical}, nil
}
func (s CapabilityStore) open(create bool) (*os.Root, *os.Root, *os.Root, error) {
	openRoot, openChild := existingPrivateRoot, existingPrivateChild
	if create {
		openRoot, openChild = privateRoot, privateChild
	}
	root, err := openRoot(s.root)
	if err != nil {
		return nil, nil, nil, err
	}
	parent, err := openChild(root, "capability-evidence")
	if err != nil {
		root.Close()
		return nil, nil, nil, err
	}
	version, err := openChild(parent, "v1")
	if err != nil {
		parent.Close()
		root.Close()
		return nil, nil, nil, err
	}
	return root, parent, version, nil
}
func (s CapabilityStore) Create(ctx context.Context, e capabilityproof.Evidence) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if e.InvalidatedReason != "" {
		return "", capabilityproof.ErrInvalidEvidence
	}
	wire, err := capabilityproof.EncodeEvidence(e)
	if err != nil {
		return "", err
	}
	id, _ := capabilityproof.Digest(e)
	root, parent, version, err := s.open(true)
	if err != nil {
		return "", err
	}
	defer root.Close()
	defer parent.Close()
	defer version.Close()
	locks, err := lockRoots(true, root, parent, version)
	if err != nil {
		return "", err
	}
	defer closeFiles(locks)
	if err := publishFile(ctx, version, id+".json", nil, wire, true, s.hooks); err != nil {
		return "", err
	}
	return id, nil
}
func (s CapabilityStore) Load(ctx context.Context, id string) (capabilityproof.Evidence, error) {
	if err := ctx.Err(); err != nil {
		return capabilityproof.Evidence{}, err
	}
	if !hexDigestName.MatchString(id + ".json") {
		return capabilityproof.Evidence{}, ErrUnsafe
	}
	root, parent, version, err := s.open(false)
	if err != nil {
		return capabilityproof.Evidence{}, err
	}
	defer root.Close()
	defer parent.Close()
	defer version.Close()
	locks, err := lockRoots(false, root, parent, version)
	if err != nil {
		return capabilityproof.Evidence{}, err
	}
	defer closeFiles(locks)
	wire, err := readPublishedFile(version, id+".json")
	if err != nil {
		return capabilityproof.Evidence{}, err
	}
	e, err := capabilityproof.DecodeEvidence(wire)
	if err != nil {
		return e, ErrUnsafe
	}
	// Invalidations change the record but preserve its original content address.
	original := e
	original.InvalidatedReason = ""
	actual, err := capabilityproof.Digest(original)
	if err != nil || actual != id {
		return capabilityproof.Evidence{}, ErrUnsafe
	}
	return e, nil
}

// Scan visits every verified record read-only under a shared lock. It never
// creates directories or writes; an absent store has no records. Pending
// publication state, unexpected names and records that fail decoding or their
// content address abort the scan instead of being skipped.
func (s CapabilityStore) Scan(ctx context.Context, visit func(capabilityproof.StoredEvidence) error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	root, parent, version, err := s.open(false)
	if errors.Is(err, ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	defer root.Close()
	defer parent.Close()
	defer version.Close()
	locks, err := lockRoots(false, root, parent, version)
	if err != nil {
		return err
	}
	defer closeFiles(locks)
	if pending, err := protocolStatePresent(version); err != nil || pending {
		return ErrRecoveryRequired
	}
	names, err := readDirectoryNamesBounded(version, maxLocalDirectoryEntries)
	if err != nil {
		return err
	}
	for _, name := range names {
		if err := ctx.Err(); err != nil {
			return err
		}
		if !hexDigestName.MatchString(name) {
			return ErrUnsafe
		}
		wire, err := readPrivateFile(version, name)
		if err != nil {
			return ErrUnsafe
		}
		e, err := capabilityproof.DecodeEvidence(wire)
		if err != nil {
			return ErrUnsafe
		}
		original := e
		original.InvalidatedReason = ""
		id := strings.TrimSuffix(name, ".json")
		if actual, err := capabilityproof.Digest(original); err != nil || actual != id {
			return ErrUnsafe
		}
		if err := visit(capabilityproof.StoredEvidence{Ref: id, Evidence: e}); err != nil {
			return err
		}
	}
	return nil
}

// Invalidate uses the loaded wire revision as a conditional update; evidence
// content addresses continue to name the original verified record.
func (s CapabilityStore) Invalidate(ctx context.Context, id, reason string, expected [sha256.Size]byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if !hexDigestName.MatchString(id + ".json") {
		return ErrUnsafe
	}
	root, parent, version, err := s.open(false)
	if err != nil {
		return err
	}
	defer root.Close()
	defer parent.Close()
	defer version.Close()
	locks, err := lockRoots(true, root, parent, version)
	if err != nil {
		return err
	}
	defer closeFiles(locks)
	old, err := readPublishedFile(version, id+".json")
	if err != nil {
		return err
	}
	if sha256.Sum256(old) != expected {
		return ErrConflict
	}
	e, err := capabilityproof.DecodeEvidence(old)
	if err != nil {
		return ErrUnsafe
	}
	original := e
	original.InvalidatedReason = ""
	actual, _ := capabilityproof.Digest(original)
	if actual != id {
		return ErrUnsafe
	}
	e.InvalidatedReason = reason
	wire, err := capabilityproof.EncodeEvidence(e)
	if err != nil {
		return err
	}
	return publishFile(ctx, version, id+".json", old, wire, false, s.hooks)
}
