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
	State    string
	Revision uint64
	// Legacy is the strictly decoded pre-stable format-1/2 record of an
	// active-v1/active-v2 snapshot. It is read only to refuse it and to bind its
	// explicit retirement (Specification 007 PM-11); it is never a current
	// Runtime/Profile configuration.
	Legacy runtimeprofile.Configuration
	// V3 is the strictly decoded canonical record of an active-v3 snapshot.
	V3 *runtimeprofile.ConfigurationV3
	// V4 is the strictly decoded proof-free record of an active-v4 snapshot.
	V4    *runtimeprofile.ConfigurationV4
	Token string
	wire  []byte
	// epoch is the committed authorization epoch bound into Token (PM-6).
	epoch uint64
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

// inspectLocked classifies the canonical object: fresh, retired, active-v3, or
// a recognized pre-stable active-v1/active-v2 record (PM-11), which only
// explicit retirement may replace.
func (s RuntimeProfileStore) inspectLocked() (RuntimeProfileSnapshot, error) {
	snapshot := RuntimeProfileSnapshot{State: "unsafe"}
	facts, err := s.runtimeProfileFacts()
	if err != nil {
		return snapshot, err
	}
	root, profiles, version, err := s.open(false)
	if errors.Is(err, ErrNotFound) {
		snapshot.State = "fresh"
		snapshot.Token = hexDigest([]byte(facts + "\x00fresh" + authorizationEpochFact(0)))
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
	object := false
	for _, name := range names {
		if protocolName(name) {
			snapshot.State = "pending"
			return snapshot, ErrRecoveryRequired
		}
		if name != runtimeProfileObject && name != authorizationEpochObject {
			return snapshot, ErrUnsafe
		}
		object = object || name == runtimeProfileObject
	}
	// The epoch never rolls back with configuration bytes (PM-6). An absent
	// record is epoch 0 only for states that can predate it.
	epoch, epochPresent, err := readAuthorizationEpoch(version)
	if err != nil {
		return snapshot, err
	}
	snapshot.epoch = epoch
	if !object {
		// A first publication interrupted before its commit point leaves only
		// the advanced epoch; the state is still fresh at that epoch.
		after, err := s.runtimeProfileFacts()
		if err != nil || after != facts {
			return RuntimeProfileSnapshot{State: "unsafe"}, ErrConflict
		}
		snapshot.State = "fresh"
		snapshot.Token = hexDigest([]byte(facts + "\x00fresh" + authorizationEpochFact(epoch)))
		return snapshot, nil
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
		snapshot.Legacy = cfg
		snapshot.Revision = cfg.Revision
		snapshot.State = "active-v1"
		if cfg.FormatVersion == runtimeprofile.FormatVersionV2 {
			snapshot.State = "active-v2"
		}
	} else if tombstone, err := runtimeprofile.DecodeTombstone(wire); err == nil {
		snapshot.Revision = tombstone.Revision
		snapshot.State = "retired"
	} else if errors.Is(cfgErr, runtimeprofile.ErrUnsupportedConfigurationFormat) {
		// Format 3 is strictly decoded by its own codec, never read as v2.
		v, err := runtimeprofile.DecodeVersioned(wire)
		switch {
		case err == nil && v.FormatVersion == runtimeprofile.FormatVersionV3 && v.V3 != nil:
			snapshot.V3 = v.V3
			snapshot.Revision = v.V3.Revision
			snapshot.State = "active-v3"
		case err == nil && v.FormatVersion == runtimeprofile.FormatVersionV4 && v.V4 != nil:
			snapshot.V4 = v.V4
			snapshot.Revision = v.V4.Revision
			snapshot.State = "active-v4"
		default:
			return snapshot, fmt.Errorf("%w: %w", ErrUnsafe, runtimeprofile.ErrUnsupportedConfigurationFormat)
		}
	} else {
		return snapshot, ErrUnsafe
	}
	// Every canonical v3 record is published after an epoch advance, so its
	// absence is a missing or deleted record, never an initial state.
	if (snapshot.State == "active-v3" || snapshot.State == "active-v4") && !epochPresent {
		return RuntimeProfileSnapshot{State: "unsafe"}, ErrUnsafe
	}
	after, err := s.runtimeProfileFacts()
	if err != nil || after != facts {
		return RuntimeProfileSnapshot{State: "unsafe"}, ErrConflict
	}
	snapshot.Token = hexDigest(append([]byte(facts+"\x00"+snapshot.State+authorizationEpochFact(epoch)+"\x00"), wire...))
	return snapshot, nil
}

// Inspect is the zero-write reader: shared locks use existing directory
// descriptors. It never migrates, and v1/v2/tombstone classifications and
// tokens are unchanged.
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

// Publish consumes an exact reviewed snapshot. Format 3 (Specification 007
// PM-1) is accepted over fresh, retired or active-v3 state, in canonical bytes
// with revision+1, and never over a format 1/2 record, so nothing is migrated
// or converted by overwrite. The existing tombstone retires an active record
// with revision+1: priorConfigurationDigest is DigestV3 of an active-v3 record
// (PM-9) and the historical Digest of a pre-stable active-v1/v2 record (PM-11).
// No format 1/2 configuration is ever written. It never down-converts v3,
// deletes the history, or retries a conflict.
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
	_, cfgErr := runtimeprofile.Decode(next)
	tombstone, tombErr := runtimeprofile.DecodeTombstone(next)
	switch {
	case cfgErr == nil:
		// No format 1/2 write: no v3 → v2 downgrade and no legacy authoring.
		return prior, runtimeprofile.ErrUnsupportedConfigurationFormat
	case errors.Is(cfgErr, runtimeprofile.ErrUnsupportedConfigurationFormat):
		v, err := runtimeprofile.DecodeVersioned(next)
		switch {
		case err == nil && v.FormatVersion == runtimeprofile.FormatVersionV3 && v.V3 != nil:
			// Format 3 never replaces a v1/v2/v4 record, pending or unsafe state.
			if prior.State != "fresh" && prior.State != "retired" && prior.State != "active-v3" {
				return prior, runtimeprofile.ErrUnsupportedConfigurationFormat
			}
			if canonical, err := runtimeprofile.EncodeV3(*v.V3); err != nil || !bytes.Equal(canonical, next) || v.V3.Revision != prior.Revision+1 {
				return prior, ErrConflict
			}
		case err == nil && v.FormatVersion == runtimeprofile.FormatVersionV4 && v.V4 != nil:
			// Format 4 starts only from fresh or retired state, or replaces format 4:
			// an active historical record is retired explicitly, never converted.
			if prior.State != "fresh" && prior.State != "retired" && prior.State != "active-v4" {
				return prior, runtimeprofile.ErrUnsupportedConfigurationFormat
			}
			if canonical, err := runtimeprofile.EncodeV4(*v.V4); err != nil || !bytes.Equal(canonical, next) || v.V4.Revision != prior.Revision+1 {
				return prior, ErrConflict
			}
		default:
			return prior, runtimeprofile.ErrUnsupportedConfigurationFormat
		}
	case tombErr == nil && prior.State == "active-v4" && prior.V4 != nil:
		digest, err := runtimeprofile.DigestV4(*prior.V4)
		if err != nil || tombstone.Revision != prior.Revision+1 || tombstone.PriorConfigurationDigest != digest {
			return prior, ErrConflict
		}
	case tombErr == nil && prior.State == "active-v3" && prior.V3 != nil:
		digest, err := runtimeprofile.DigestV3(*prior.V3)
		if err != nil || tombstone.Revision != prior.Revision+1 || tombstone.PriorConfigurationDigest != digest {
			return prior, ErrConflict
		}
	case tombErr == nil && (prior.State == "active-v1" || prior.State == "active-v2"):
		digest, err := runtimeprofile.Digest(prior.Legacy)
		if err != nil || tombstone.Revision != prior.Revision+1 || tombstone.PriorConfigurationDigest != digest {
			return prior, ErrConflict
		}
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
	// Advance the epoch before the configuration can reach its commit point:
	// whatever this attempt's outcome (commit, F3 rollback, recovery), the
	// reviewed prior token can never match again (PM-6). The epoch is read
	// from the locked descriptor, so concurrent writers cannot lose an update.
	epoch, _, err := readAuthorizationEpoch(version)
	if err != nil || epoch != prior.epoch {
		return prior, ErrConflict
	}
	if _, err := advanceAuthorizationEpoch(version, epoch, hooks); err != nil {
		return prior, err
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
	// The recovery preview binds the epoch it will advance (PM-6).
	epoch, _, err := readAuthorizationEpoch(target)
	if err != nil {
		return err
	}
	facts += authorizationEpochFact(epoch)
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
