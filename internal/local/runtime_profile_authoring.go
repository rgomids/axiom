package local

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/rgomids/axiom/internal/runtimeprofile"
)

// RuntimeProfileSnapshot is an internal authoring input. Configuration must
// never be serialized directly into a public diagnostic; Token is opaque.
type RuntimeProfileSnapshot struct {
	State         string
	Revision      uint64
	Configuration runtimeprofile.Configuration
	Token         string
	wire          []byte
}

const runtimeProfileObject = "configuration.json"

// runtimeProfileFacts binds every visible ancestor, including the nearest
// existing ancestor of an absent state root, and exact file bytes and identity.
func (s RuntimeProfileStore) runtimeProfileFacts() (string, error) {
	target := filepath.Join(s.root, "runtime-profiles", "v1", runtimeProfileObject)
	var facts strings.Builder
	facts.WriteString(target)
	current := volumeRoot(target)
	parts := pathComponents(target)
	for index, part := range parts {
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if os.IsNotExist(err) {
			fmt.Fprintf(&facts, "\x00absent:%s", current)
			break
		}
		if err != nil || info.Mode()&os.ModeSymlink != 0 {
			return "", ErrUnsafe
		}
		var identity string
		if index == len(parts)-1 {
			identity, err = FileIdentity(current)
		} else {
			identity, err = DirectoryIdentity(current)
		}
		if err != nil {
			return "", ErrUnsafe
		}
		facts.WriteByte(0)
		facts.WriteString(identity)
	}
	return facts.String(), nil
}

// openObserved opens only existing objects and never creates a lock file. The
// nearest existing ancestor serializes first creation, including absent roots.
func (s RuntimeProfileStore) openObserved() ([]*os.Root, error) {
	nearest := s.root
	for {
		_, err := os.Lstat(nearest)
		if err == nil {
			break
		}
		if !os.IsNotExist(err) {
			return nil, ErrUnsafe
		}
		next := filepath.Dir(nearest)
		if next == nearest {
			return nil, ErrUnsafe
		}
		nearest = next
	}
	root, err := os.OpenRoot(nearest)
	if err != nil {
		return nil, ErrUnsafe
	}
	chain := []*os.Root{root}
	relative, err := filepath.Rel(nearest, filepath.Join(s.root, "runtime-profiles", "v1"))
	if err != nil {
		root.Close()
		return nil, ErrUnsafe
	}
	if relative != "." {
		for _, part := range strings.Split(relative, string(filepath.Separator)) {
			child, err := existingPrivateChild(chain[len(chain)-1], part)
			if errors.Is(err, ErrNotFound) {
				return chain, nil
			}
			if err != nil {
				closeRootChain(chain)
				return nil, err
			}
			chain = append(chain, child)
		}
	}
	return chain, nil
}
func closeRootChain(chain []*os.Root) {
	for i := len(chain) - 1; i >= 0; i-- {
		chain[i].Close()
	}
}

func (s RuntimeProfileStore) inspectLocked() (RuntimeProfileSnapshot, error) {
	snapshot := RuntimeProfileSnapshot{State: "unsafe"}
	facts, err := s.runtimeProfileFacts()
	if err != nil {
		return snapshot, err
	}
	root, profiles, version, err := s.open(false)
	if errors.Is(err, ErrNotFound) {
		snapshot.State = "fresh"
		snapshot.Token = hexDigest([]byte(facts + "\x00fresh"))
		return snapshot, nil
	}
	if err != nil {
		return snapshot, err
	}
	defer root.Close()
	defer profiles.Close()
	defer version.Close()
	names, err := readDirectoryNamesBounded(version, maxLocalDirectoryEntries)
	if err != nil {
		return snapshot, ErrUnsafe
	}
	for _, name := range names {
		if protocolName(name) {
			snapshot.State = "pending"
			return snapshot, ErrRecoveryRequired
		}
		if name != runtimeProfileObject {
			return snapshot, ErrUnsafe
		}
	}
	wire, err := readPublishedFile(version, runtimeProfileObject)
	if err != nil {
		if errors.Is(err, ErrRecoveryRequired) {
			snapshot.State = "pending"
		}
		return snapshot, err
	}
	snapshot.wire = wire
	cfg, cfgErr := runtimeprofile.Decode(wire)
	if cfgErr == nil {
		snapshot.Configuration = cfg
		snapshot.Revision = cfg.Revision
		snapshot.State = "active-v1"
		if cfg.FormatVersion == runtimeprofile.FormatVersionV2 {
			snapshot.State = "active-v2"
		}
	} else if tombstone, err := runtimeprofile.DecodeTombstone(wire); err == nil {
		snapshot.Revision = tombstone.Revision
		snapshot.State = "retired"
	} else if errors.Is(cfgErr, runtimeprofile.ErrUnsupportedConfigurationFormat) {
		// v2-only reader: a newer format is refused explicitly, never read as v2.
		return snapshot, fmt.Errorf("%w: %w", ErrUnsafe, cfgErr)
	} else {
		return snapshot, ErrUnsafe
	}
	after, err := s.runtimeProfileFacts()
	if err != nil || after != facts {
		return RuntimeProfileSnapshot{State: "unsafe"}, ErrConflict
	}
	snapshot.Token = hexDigest(append([]byte(facts+"\x00"+snapshot.State+"\x00"), wire...))
	return snapshot, nil
}

// Inspect is zero-write: shared locks use existing directory descriptors.
func (s RuntimeProfileStore) Inspect(ctx context.Context) (RuntimeProfileSnapshot, error) {
	if err := ctx.Err(); err != nil {
		return RuntimeProfileSnapshot{}, err
	}
	chain, err := s.openObserved()
	if err != nil {
		return RuntimeProfileSnapshot{State: "unsafe"}, err
	}
	defer closeRootChain(chain)
	locks, err := lockRoots(false, chain...)
	if err != nil {
		return RuntimeProfileSnapshot{}, err
	}
	defer closeFiles(locks)
	return s.inspectLocked()
}

// Publish consumes an exact reviewed snapshot. It only accepts active v2 or a
// tombstone and never down-converts, deletes the history, or retries a conflict.
func (s RuntimeProfileStore) Publish(ctx context.Context, expectedToken string, next []byte) (RuntimeProfileSnapshot, error) {
	if err := ctx.Err(); err != nil {
		return RuntimeProfileSnapshot{}, err
	}
	chain, err := s.openObserved()
	if err != nil {
		return RuntimeProfileSnapshot{}, err
	}
	defer closeRootChain(chain)
	locks, err := lockRoots(true, chain...)
	if err != nil {
		return RuntimeProfileSnapshot{}, err
	}
	defer closeFiles(locks)
	prior, err := s.inspectLocked()
	if err != nil {
		return prior, err
	}
	if expectedToken == "" || prior.Token != expectedToken || prior.Revision == math.MaxUint64 {
		return prior, ErrConflict
	}
	cfg, cfgErr := runtimeprofile.Decode(next)
	tombstone, tombErr := runtimeprofile.DecodeTombstone(next)
	switch {
	case cfgErr == nil && cfg.FormatVersion == runtimeprofile.FormatVersionV2:
		if cfg.Revision != prior.Revision+1 {
			return prior, ErrConflict
		}
	case tombErr == nil && (prior.State == "active-v1" || prior.State == "active-v2"):
		digest, _ := runtimeprofile.Digest(prior.Configuration)
		if tombstone.Revision != prior.Revision+1 || tombstone.PriorConfigurationDigest != digest {
			return prior, ErrConflict
		}
	case errors.Is(cfgErr, runtimeprofile.ErrUnsupportedConfigurationFormat):
		// Format-3 publication is not authorized (Specification 007 AXM-22 D5).
		return prior, cfgErr
	default:
		return prior, runtimeprofile.ErrInvalidConfiguration
	}
	if bytes.Equal(next, prior.wire) {
		return prior, ErrConflict
	}
	originalFacts, err := s.runtimeProfileFacts()
	if err != nil {
		return prior, err
	}
	root, profiles, version, err := s.open(prior.State == "fresh")
	if err != nil {
		return prior, err
	}
	defer root.Close()
	defer profiles.Close()
	defer version.Close()
	// Existing chain descriptors are already locked; lock newly created objects.
	for _, opened := range []*os.Root{root, profiles, version} {
		info, _ := opened.Stat(".")
		found := false
		for _, existing := range chain {
			old, _ := existing.Stat(".")
			if info != nil && old != nil && os.SameFile(info, old) {
				found = true
				break
			}
		}
		if !found {
			added, e := lockRoots(true, opened)
			if e != nil {
				return prior, e
			}
			defer closeFiles(added)
		}
	}
	hooks := s.hooks
	before := hooks.beforeCommit
	hooks.beforeCommit = func() error {
		if before != nil {
			if err := before(); err != nil {
				return err
			}
		}
		for _, item := range []struct {
			root *os.Root
			path string
		}{{root, s.root}, {profiles, filepath.Join(s.root, "runtime-profiles")}, {version, filepath.Join(s.root, "runtime-profiles", "v1")}} {
			if err := stillAtPath(item.root, item.path); err != nil {
				return err
			}
		}
		facts, e := s.runtimeProfileFacts()
		bound := strings.Split(originalFacts, "\x00absent:")[0]
		if e != nil || !strings.HasPrefix(facts, bound) {
			return ErrConflict
		}
		if prior.State != "fresh" {
			if facts != originalFacts {
				return ErrConflict
			}
		}
		return nil
	}
	publishedFacts, err := s.runtimeProfileFacts()
	if err != nil {
		return prior, err
	}
	ancestorFacts := strings.Split(strings.Split(publishedFacts, "\x00file:")[0], "\x00absent:")[0]
	var committedIdentity string
	afterCommit := hooks.afterCommit
	hooks.afterCommit = func() {
		committedIdentity, _ = FileIdentity(filepath.Join(s.root, "runtime-profiles", "v1", runtimeProfileObject))
		if afterCommit != nil {
			afterCommit()
		}
	}
	hooks.contentLimit = runtimeprofile.MaxConfigurationBytes
	err = publishFile(ctx, version, runtimeProfileObject, prior.wire, next, prior.State == "fresh", hooks)
	if err != nil {
		return prior, err
	}
	observed, err := s.inspectLocked()
	visibleFacts, factsErr := s.runtimeProfileFacts()
	visibleIdentity, identityErr := FileIdentity(filepath.Join(s.root, "runtime-profiles", "v1", runtimeProfileObject))
	if err != nil || factsErr != nil || identityErr != nil || committedIdentity == "" || visibleIdentity != committedIdentity || !strings.HasPrefix(visibleFacts, ancestorFacts) || !bytes.Equal(observed.wire, next) {
		return observed, publicationFailure(FaultF6, true, ErrRecoveryRequired)
	}
	return observed, nil
}

// InspectRuntimeRecovery and ApplyRuntimeRecovery expose only this canonical
// store's protocol; generic recovery retains its exact authority contract.
func (s RuntimeProfileStore) InspectRuntimeRecovery(ctx context.Context) (RecoveryPlan, bool, error) {
	return inspectRecoveryDirectory(ctx, RecoveryScopeState, s.root, "runtime-profiles/v1", "", false)
}
func (s RuntimeProfileStore) ApplyRuntimeRecovery(ctx context.Context, plan RecoveryPlan, authority RecoveryAuthority) (RecoveryResult, error) {
	if plan.Scope != RecoveryScopeState || plan.Directory != "runtime-profiles/v1" {
		return RecoveryResult{}, ErrUnsafe
	}
	return ApplyRecovery(ctx, RecoveryRoots{State: s.root}, plan, authority)
}

// Recovery identity is included only for the newly authored family, preserving
// historical generic recovery digests for all other stores.
func bindRuntimeRecovery(plan *RecoveryPlan, path string, target *os.Root) error {
	if plan.Scope != RecoveryScopeState || plan.Directory != "runtime-profiles/v1" {
		return nil
	}
	store := RuntimeProfileStore{root: filepath.Clean(path)}
	facts, err := store.runtimeProfileFacts()
	if err != nil {
		return err
	}
	for _, object := range plan.Objects {
		if !object.Present {
			continue
		}
		identity, err := FileIdentity(filepath.Join(target.Name(), object.Name))
		if err != nil {
			return ErrUnsafe
		}
		facts += "\x00" + identity
	}
	plan.SnapshotIdentity = hexDigest([]byte(facts))
	plan.Digest = recoveryDigest(*plan)
	return nil
}
