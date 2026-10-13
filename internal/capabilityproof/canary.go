package capabilityproof

import (
	"context"
	"errors"
	"net/netip"
	"slices"
)

var ErrCanaryContext = errors.New("invalid probe canary context")

// CanaryContext comes from an identity-checked envelope, never child-supplied
// host paths. The loader must verify manifest digest and symlink-free identities.
type CanaryContext struct {
	Manifest       EnvelopeManifest
	ManifestDigest string
	ProbeID, Nonce string
	Loopback       netip.AddrPort
	ListenerOwned  bool
}
type ProcessIdentity struct {
	UID, GID uint64
	Process  string
}
type CanaryAttempt struct {
	Target  string
	Created bool
	Errno   string
}
type CanaryReport struct {
	ProbeID   string
	Identity  ProcessIdentity
	Attempts  []CanaryAttempt
	Loopback  string
	Published bool
}

// CanaryEffects operates on manifest categories only. CreateEmptyExclusive must
// use O_CREAT|O_EXCL, mode 0600, symlink-free resolution and no data writes.
// Publish is the T22 child-inbox port, not a canonical coordination-store write.
type CanaryEffects interface {
	Verify(context.Context, CanaryContext) error
	Identity(context.Context) (ProcessIdentity, error)
	CreateEmptyExclusive(context.Context, string, string) (bool, string, error)
	ConnectOwnedLoopback(context.Context, netip.AddrPort) (string, error)
	Publish(context.Context, CanaryReport) error
}

func RunCanary(ctx context.Context, c CanaryContext, effects CanaryEffects) (CanaryReport, error) {
	m := c.Manifest
	targets := []string{"sentinel"}
	allowed := []string{"workspace", "inbox"}
	if m.Variant == ReadOnlyVariant {
		targets = append(targets, "workspace")
		allowed = []string{"inbox"}
	}
	if effects == nil || m.Version != 1 || m.Phase != "ready" || !safe(c.ProbeID) || c.ProbeID != m.ProbeID || c.Nonce != m.Nonce || !digest.MatchString(c.Nonce) || !digest.MatchString(c.ManifestDigest) || m.Sentinel != "sentinel" || !slices.Equal(m.NegativeTargets, targets) || !slices.Equal(m.AllowedRoots, allowed) || (m.Variant != ReadOnlyVariant && m.Variant != RepositoryWriteVariant) || !c.ListenerOwned || !c.Loopback.IsValid() || !c.Loopback.Addr().IsLoopback() || c.Loopback.Port() == 0 {
		return CanaryReport{}, ErrCanaryContext
	}
	if err := effects.Verify(ctx, c); err != nil {
		return CanaryReport{}, ErrCanaryContext
	}
	identity, err := effects.Identity(ctx)
	if err != nil || !safe(identity.Process) {
		return CanaryReport{}, ErrCanaryContext
	}
	r := CanaryReport{ProbeID: c.ProbeID, Identity: identity}
	for _, target := range targets {
		if err := ctx.Err(); err != nil {
			return CanaryReport{}, err
		}
		// Revalidate before each effect to detect replacement between steps.
		if err := effects.Verify(ctx, c); err != nil {
			return CanaryReport{}, ErrCanaryContext
		}
		created, errno, err := effects.CreateEmptyExclusive(ctx, target, "canary-"+c.Nonce)
		if err != nil {
			return CanaryReport{}, errors.New("canary measurement unavailable")
		}
		if !validErrno(errno) || created && errno != "" || !created && errno == "" {
			return CanaryReport{}, ErrCanaryContext
		}
		r.Attempts = append(r.Attempts, CanaryAttempt{Target: target, Created: created, Errno: errno})
	}
	r.Loopback, err = effects.ConnectOwnedLoopback(ctx, c.Loopback)
	if err != nil || (r.Loopback != "reached" && r.Loopback != "denied" && r.Loopback != "unknown") {
		return CanaryReport{}, errors.New("canary loopback unavailable")
	}
	if err := effects.Verify(ctx, c); err != nil {
		return CanaryReport{}, ErrCanaryContext
	}
	if err := effects.Publish(ctx, r); err != nil {
		return r, errors.New("canary inbox publish rejected")
	}
	r.Published = true
	return r, nil
}
func validErrno(s string) bool {
	return s == "" || s == "EACCES" || s == "EPERM" || s == "EROFS" || s == "EEXIST" || s == "EIO" || s == "OTHER"
}
