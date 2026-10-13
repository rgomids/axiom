package capabilityproof

import (
	"context"
	"crypto/sha256"
	"errors"
	"reflect"
	"sort"
	"time"
)

var ErrBlocked = errors.New("capability proof blocked")
var ErrStalePreview = errors.New("capability preview stale")
var ErrAuthority = errors.New("local invalidation authority required")

// Store consumes the existing content-addressed evidence store. Invalidation
// conditionally updates the exact reviewed wire revision.
type Store interface {
	Load(context.Context, string) (Evidence, error)
	Invalidate(context.Context, string, string, [sha256.Size]byte) error
}
type Target struct{ RuntimeID, ProfileID, Variant string }
type Current struct {
	Binding       Binding
	Model         *ModelBinding
	EvidenceRef   string
	ProfileActive bool
}

// BindingSource must inspect current executable, adapter, pinned axiom, mode,
// Profile and effort-source identities; evidence is never a source of identity.
type BindingSource interface {
	Current(context.Context, Target) (Current, error)
}
type Service struct {
	Store  Store
	Source BindingSource
	Now    func() time.Time
}
type View struct {
	Target      Target
	Binding     Binding
	Model       *ModelBinding
	EvidenceRef string
	States      map[string]State
	Blockers    map[string]string
	Confinement Confinement
}

func (s Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}
func (s Service) read(ctx context.Context, t Target) (Current, Evidence, error) {
	if err := ctx.Err(); err != nil {
		return Current{}, Evidence{}, err
	}
	if s.Source == nil || s.Store == nil {
		return Current{}, Evidence{}, ErrBlocked
	}
	c, err := s.Source.Current(ctx, t)
	if err != nil {
		return c, Evidence{}, err
	}
	if c.Binding.RuntimeID != t.RuntimeID || c.Binding.Variant != t.Variant || !ValidBinding(c.Binding) || (t.ProfileID != "" && (c.Model == nil || c.Model.ProfileID != t.ProfileID)) {
		return c, Evidence{}, ErrBlocked
	}
	e, err := s.Store.Load(ctx, c.EvidenceRef)
	if err != nil {
		return c, Evidence{}, err
	}
	original := e
	original.InvalidatedReason = ""
	ref, err := Digest(original)
	if err != nil || ref != c.EvidenceRef {
		return c, Evidence{}, ErrInvalidEvidence
	}
	return c, e, nil
}
func state(c Current, e Evidence, cap string, now time.Time) (State, string) {
	if !c.ProfileActive {
		return Unknown, "capability_unknown"
	}
	return Evaluate(e, c.Binding, c.Model, cap, now)
}
func (s Service) Show(ctx context.Context, t Target) (View, error) {
	c, e, err := s.read(ctx, t)
	v := View{Target: t, Binding: c.Binding, Model: c.Model, EvidenceRef: c.EvidenceRef, States: map[string]State{}, Blockers: map[string]string{}, Confinement: Confinement{Level: "unknown", NotTested: []string{"home", "vendor-home", "tmpdir", "system-temp", "external-network"}}}
	caps := []string{Read, RepositoryWrite, Process, Noninteractive, AxiomCoordination, ModelInvocation}
	if c.Model != nil {
		caps = append(caps, "reasoning-effort-"+c.Model.EffortToken)
	}
	for cap := range e.Capabilities {
		caps = append(caps, cap)
	}
	for _, cap := range caps {
		v.States[cap], v.Blockers[cap] = state(c, e, cap, s.now())
	}
	if err == nil && e.Binding == c.Binding && reflect.DeepEqual(e.Model, c.Model) && c.ProfileActive && e.InvalidatedReason == "" && !s.now().Before(e.ObservedAt) && s.now().Sub(e.ObservedAt) <= MaxAge {
		v.Confinement = e.Confinement
	}
	return v, err
}

// Check is called at preview and immediately before every dispatch. Implicit
// and model checks stay outside the Profile declaration resolver.
func (s Service) Check(ctx context.Context, t Target, declared []string, write, process bool, expectedRef string) (string, string, error) {
	c, e, err := s.read(ctx, t)
	if err != nil {
		if errors.Is(err, ErrStalePreview) {
			return "", "capability_stale", err
		}
		return "", "capability_unknown", ErrBlocked
	}
	if expectedRef != "" && c.EvidenceRef != expectedRef {
		return "", "capability_stale", ErrStalePreview
	}
	caps := append([]string(nil), declared...)
	caps = append(caps, Noninteractive, AxiomCoordination, ModelInvocation)
	if write && t.Variant != RepositoryWriteVariant || !write && t.Variant != ReadOnlyVariant {
		return "", "capability_stale", ErrBlocked
	}
	if write {
		caps = append(caps, RepositoryWrite)
	}
	if process {
		caps = append(caps, Process)
	}
	if c.Model == nil {
		return "", "capability_unknown", ErrBlocked
	}
	if c.Model.EffortMode == "explicit" {
		caps = append(caps, "reasoning-effort-"+c.Model.EffortToken)
	}
	sort.Strings(caps)
	for _, cap := range caps {
		if !ValidCapability(cap) {
			return "", "capability_unknown", ErrBlocked
		}
		st, code := state(c, e, cap, s.now())
		if st != Proven {
			if expectedRef != "" && code == "capability_stale" {
				return "", code, ErrStalePreview
			}
			return "", code, ErrBlocked
		}
	}
	return c.EvidenceRef, "", nil
}

type InvalidationPreview struct {
	Target      Target
	EvidenceRef string
	Revision    [sha256.Size]byte
	Current     Current
}

func (s Service) PreviewInvalidate(ctx context.Context, t Target) (InvalidationPreview, error) {
	c, e, err := s.read(ctx, t)
	if err != nil {
		return InvalidationPreview{}, err
	}
	if e.Binding != c.Binding || !reflect.DeepEqual(e.Model, c.Model) {
		return InvalidationPreview{}, ErrStalePreview
	}
	wire, err := EncodeEvidence(e)
	if err != nil {
		return InvalidationPreview{}, err
	}
	if c.Model != nil {
		model := *c.Model
		c.Model = &model
	}
	return InvalidationPreview{t, c.EvidenceRef, sha256.Sum256(wire), c}, nil
}

// LocalAuthority binds the operator confirmation to the exact preview. It is
// not dispatch authority and authorizes no probe or provider effect.
type LocalAuthority struct {
	Actor   string
	Preview InvalidationPreview
}

func (s Service) Invalidate(ctx context.Context, p InvalidationPreview, a LocalAuthority) error {
	if !safe(a.Actor) || !reflect.DeepEqual(a.Preview, p) {
		return ErrAuthority
	}
	fresh, err := s.PreviewInvalidate(ctx, p.Target)
	if err != nil || !reflect.DeepEqual(fresh, p) {
		return ErrStalePreview
	}
	return s.Store.Invalidate(ctx, p.EvidenceRef, "explicit", p.Revision)
}
