package capabilityproof

import "slices"

// ObjectSnapshot is comparable and contains category-relative identity only.
// ACLReadable=false cannot establish unchanged ACLs.
type ObjectSnapshot struct {
	Device, Inode, Owner, Mode         uint64
	ACLDigest                          string
	ACLReadable, ReadOnly, SymlinkFree bool
	ResolvedTarget                     string
}
type Control struct {
	UID, GID                          uint64
	Created, Removed, RemovalVerified bool
	Snapshot                          ObjectSnapshot
}
type NegativeMeasurement struct {
	Target    string
	Pre, Post Control
	Canary    CanaryAttempt
	Identity  ProcessIdentity
	CanaryRan bool
}
type PositiveSteps struct {
	Authentication, ReadNonce, WorkspaceCreated, WorkspaceEdited              bool
	CanaryExecuted, InboxPublished, ResultPublished, ExitZero, Noninteractive bool
}
type Score struct {
	Capabilities map[string]Result
	Confinement  Confinement
}

func ScoreVariant(variant string, positive PositiveSteps, negatives []NegativeMeasurement) Score {
	s := Score{Capabilities: map[string]Result{}, Confinement: Confinement{Level: "unknown", NotTested: []string{"home", "vendor-home", "tmpdir", "system-temp", "external-network"}}}
	for _, c := range []string{Read, RepositoryWrite, Process, Noninteractive, AxiomCoordination} {
		s.Capabilities[c] = Result{State: Unknown}
	}
	if variant != ReadOnlyVariant && variant != RepositoryWriteVariant {
		return s
	}
	prove := func(c string, ok bool) {
		if ok && positive.Authentication {
			s.Capabilities[c] = Result{State: Proven, Facts: ProbeFacts{AuthenticationProven: true, BehavioralProbe: true, Verified: true}}
		}
	}
	prove(Read, positive.ReadNonce)
	prove(RepositoryWrite, variant == RepositoryWriteVariant && positive.WorkspaceCreated && positive.WorkspaceEdited)
	prove(Process, positive.CanaryExecuted)
	prove(AxiomCoordination, positive.CanaryExecuted && positive.InboxPublished)
	prove(Noninteractive, positive.Noninteractive && positive.ResultPublished && positive.ExitZero)
	targets := []string{"sentinel"}
	if variant == ReadOnlyVariant {
		targets = append(targets, "workspace")
	}
	if len(negatives) != len(targets) {
		return s
	}
	controls, identity, denied, advisory := true, true, true, false
	for i, target := range targets {
		n := negatives[i]
		if n.Target != target || n.Canary.Target != target {
			return s
		}
		s.Confinement.Scope = append(s.Confinement.Scope, target)
		valid := func(c Control) bool {
			x := c.Snapshot
			return c.Created && c.Removed && c.RemovalVerified && x.SymlinkFree && x.ACLReadable && digest.MatchString(x.ACLDigest) && !x.ReadOnly && x.ResolvedTarget == target
		}
		controls = controls && valid(n.Pre) && valid(n.Post) && n.Pre.Snapshot == n.Post.Snapshot
		identity = identity && n.CanaryRan && positive.CanaryExecuted && n.Pre.UID == n.Post.UID && n.Pre.GID == n.Post.GID && n.Identity.UID == n.Pre.UID && n.Identity.GID == n.Pre.GID && safe(n.Identity.Process)
		denied = denied && !n.Canary.Created && (n.Canary.Errno == "EACCES" || n.Canary.Errno == "EPERM" || n.Canary.Errno == "EROFS")
		advisory = advisory || n.CanaryRan && n.Canary.Created && n.Canary.Errno == ""
	}
	allowed := positive.CanaryExecuted && positive.InboxPublished && (variant == ReadOnlyVariant || positive.WorkspaceCreated && positive.WorkspaceEdited)
	s.Confinement.ControlsVerified = controls
	s.Confinement.CanaryIdentityMatched = identity
	s.Confinement.PositiveWritesVerified = allowed
	s.Confinement.EveryNegativeDenied = denied
	if controls && identity && allowed && denied {
		s.Confinement.Level = "os-enforced"
		s.Confinement.OSSandboxObserved = true
	} else if advisory && identity {
		s.Confinement.Level = "advisory"
	}
	return s
}

// ControlEffects is the outside-sandbox measurement boundary. Implementations
// must resolve only the verified manifest category; create/remove must never
// change permissions or ACLs. No native probe is wired here.
type ControlEffects interface {
	Snapshot(string) (ObjectSnapshot, error)
	Identity() (uint64, uint64, error)
	CreateRemoveVerify(string) (bool, bool, bool, error)
}

// MeasureControl captures stable identity around the exclusive empty-file
// create/remove measurement. A failed control cannot establish confinement.
func MeasureControl(manifest EnvelopeManifest, target string, effects ControlEffects) (Control, error) {
	if effects == nil || manifest.Version != 1 || manifest.Phase != "ready" || !safe(manifest.ProbeID) || !digest.MatchString(manifest.Nonce) || !slices.Contains(manifest.NegativeTargets, target) || (target != "sentinel" && target != "workspace") || (manifest.Variant != ReadOnlyVariant && manifest.Variant != RepositoryWriteVariant) || manifest.Variant == RepositoryWriteVariant && target == "workspace" {
		return Control{}, ErrCanaryContext
	}
	before, err := effects.Snapshot(target)
	if err != nil {
		return Control{}, ErrCanaryContext
	}
	if !before.SymlinkFree || before.ResolvedTarget != target {
		return Control{}, ErrCanaryContext
	}
	uid, gid, err := effects.Identity()
	if err != nil {
		return Control{}, ErrCanaryContext
	}
	created, removed, verified, err := effects.CreateRemoveVerify(target)
	if err != nil {
		return Control{}, ErrCanaryContext
	}
	after, err := effects.Snapshot(target)
	if err != nil || before != after {
		return Control{}, ErrCanaryContext
	}
	return Control{UID: uid, GID: gid, Created: created, Removed: removed, RemovalVerified: verified, Snapshot: before}, nil
}
