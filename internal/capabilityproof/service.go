package capabilityproof

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"reflect"
	"sort"
	"time"

	"github.com/rgomids/axiom/internal/executiongraph"
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
	ProbeAdapter   ProbeAdapter
	ProbeEnvelopes ProbeEnvelopes
	ProbeWriter    EvidenceWriter
	ProbeAuth      ProbeAuthentication
	Store          Store
	Source         BindingSource
	Now            func() time.Time
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

// ProbePreparation describes a single dispatch-identical invocation. Prepare
// must freshly inspect the v2 Profile and PE-05a effort sources without creating
// an envelope or launching inference. Host paths remain only in this preview.
type ProbePreparation struct {
	Current              Current
	ConfigurationVersion int
	Manifest             EnvelopeManifest
	ManifestDigest       string
	Invocation           executiongraph.Invocation
	Interface            InterfaceObservation
	OSSandbox            bool
}

type ProbeAdapter interface {
	Prepare(context.Context, Target) (ProbePreparation, error)
	Run(context.Context, executiongraph.Invocation) executiongraph.ProcessResult
}

// ProbeEnvelope owns verified manifest loading, same-identity controls,
// deterministic filesystem/inbox verification, and exact-object cleanup.
// Verify must correlate nonce, manifest, packet and attempt before accepting
// any canary record; it must never consume Runtime output as proof.
type ProbeEnvelope interface {
	Control(context.Context, string) (Control, error)
	Verify(context.Context, executiongraph.ProcessResult) (ProbeVerification, error)
	Cleanup(context.Context) error
	Quarantine(context.Context) error
}
type ProbeEnvelopes interface {
	Open(context.Context, ProbePreparation) (ProbeEnvelope, error)
}
type ProbeVerification struct {
	ProbeID, ManifestDigest, Nonce, PacketDigest string
	Positive                                     PositiveSteps
	Negatives                                    []NegativeMeasurement
	ModelInvoked, EffortVerified                 bool
	Inconclusive                                 bool
}
type EvidenceWriter interface {
	Create(context.Context, Evidence) (string, error)
}
type ProbeAuthentication interface {
	Proven(context.Context, Target) (bool, error)
}

type ProbePreview struct {
	Target      Target
	Preparation ProbePreparation
	Risk, Cost  string
}

// ProbeAuthority is separate from invalidation and extended-temp authority.
type ProbeAuthority struct {
	Actor           string
	Local, External bool
	Preview         ProbePreview
}
type ProbeOutcome struct {
	Evidence     Evidence
	EvidenceRef  string
	Inconclusive bool
}

var ErrAuthentication = errors.New("authentication_blocked")

func (s Service) PreviewProbe(ctx context.Context, t Target) (ProbePreview, error) {
	if err := ctx.Err(); err != nil {
		return ProbePreview{}, err
	}
	if s.ProbeAdapter == nil || s.ProbeEnvelopes == nil || s.ProbeWriter == nil || s.ProbeAuth == nil {
		return ProbePreview{}, ErrBlocked
	}
	p, err := s.ProbeAdapter.Prepare(ctx, t)
	if err != nil {
		return ProbePreview{}, err
	}
	if p.Current.Model != nil {
		model := *p.Current.Model
		p.Current.Model = &model
	}
	p.Manifest.AllowedRoots = append([]string(nil), p.Manifest.AllowedRoots...)
	p.Manifest.NegativeTargets = append([]string(nil), p.Manifest.NegativeTargets...)
	identities := make(map[string]string, len(p.Manifest.Identities))
	for k, v := range p.Manifest.Identities {
		identities[k] = v
	}
	if p.Manifest.Identities != nil {
		p.Manifest.Identities = identities
	}
	capabilities := make(map[string]Result, len(p.Interface.Capabilities))
	for k, v := range p.Interface.Capabilities {
		capabilities[k] = v
	}
	if p.Interface.Capabilities != nil {
		p.Interface.Capabilities = capabilities
	}
	p.Invocation.Argv = append([]string(nil), p.Invocation.Argv...)
	p.Invocation.Env = append([]string(nil), p.Invocation.Env...)
	p.Invocation.Stdin = append([]byte(nil), p.Invocation.Stdin...)
	p.Invocation = executiongraph.WithExactEnvironment(p.Invocation, p.Invocation.Env)
	c := p.Current
	// A valid model binding includes the Profile revision and inspected effort
	// source; declarations, authentication and help alone are never proof.
	check := Evidence{SchemaVersion: 1, Binding: c.Binding, Model: c.Model, Capabilities: map[string]Result{ModelInvocation: {State: Unknown}}, Confinement: Confinement{Level: "unknown", NotTested: []string{"home", "vendor-home", "tmpdir", "system-temp", "external-network"}}, ObservedAt: s.now(), ProbePacketDigest: packetDigest(p.Invocation.Stdin)}
	m := p.Manifest
	wire, _ := json.Marshal(m)
	if p.ConfigurationVersion != 2 || !c.ProfileActive || c.Model == nil || c.Model.ProfileID != t.ProfileID || c.Binding.RuntimeID != t.RuntimeID || c.Binding.Variant != t.Variant || !ValidEvidence(check) || m.Version != 1 || m.Phase != "ready" || m.Variant != t.Variant || !safe(m.ProbeID) || !digest.MatchString(m.Nonce) || !digest.MatchString(m.Revision) || m.CreatedAt.IsZero() || m.Limits.Timeout <= 0 || m.Limits.Timeout > 24*time.Hour || m.Limits.MaxFiles <= 0 || m.Limits.MaxBytes <= 0 || p.ManifestDigest != packetDigest(wire) || p.Interface.ExecutableDigest != c.Binding.ExecutableDigest || p.Interface.Version != c.Binding.Version || p.Invocation.RuntimeID != t.RuntimeID || len(p.Invocation.Argv) == 0 || len(p.Invocation.Stdin) == 0 || p.Invocation.OutputMax <= 0 || p.Invocation.OutputMax > executiongraph.MaxCapturedOutputBytes || len(p.Invocation.Stdin) > executiongraph.MaxStdinBytes {
		return ProbePreview{}, ErrBlocked
	}
	targets := []string{"sentinel"}
	roots := []string{"inbox"}
	if t.Variant == ReadOnlyVariant {
		targets = append(targets, "workspace")
	} else {
		roots = []string{"workspace", "inbox"}
	}
	if !reflect.DeepEqual(m.NegativeTargets, targets) || !reflect.DeepEqual(m.AllowedRoots, roots) || m.Sentinel != "sentinel" {
		return ProbePreview{}, ErrBlocked
	}
	return ProbePreview{Target: t, Preparation: p, Risk: "advisory confinement may permit model effects outside the envelope; reads are not confined", Cost: "one vendor inference invocation; no retries"}, nil
}

func packetDigest(wire []byte) string {
	d := sha256.Sum256(wire)
	return hex.EncodeToString(d[:])
}

func (s Service) Probe(ctx context.Context, p ProbePreview, a ProbeAuthority) (out ProbeOutcome, err error) {
	if !safe(a.Actor) || !a.Local || !a.External || !reflect.DeepEqual(a.Preview, p) {
		return out, ErrAuthority
	}
	fresh, err := s.PreviewProbe(ctx, p.Target)
	if err != nil || !reflect.DeepEqual(fresh, p) {
		return out, ErrStalePreview
	}
	ok, err := s.ProbeAuth.Proven(ctx, p.Target)
	if err != nil || !ok {
		return out, ErrAuthentication
	}
	// Revalidate after authentication too: an edited Profile never gets a probe
	// using the old reviewed tuple.
	fresh, err = s.PreviewProbe(ctx, p.Target)
	if err != nil || !reflect.DeepEqual(fresh, p) {
		return out, ErrStalePreview
	}
	envelope, err := s.ProbeEnvelopes.Open(ctx, p.Preparation)
	if err != nil {
		return out, err
	}
	if envelope == nil {
		return out, ErrBlocked
	}
	quarantine := true
	defer func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
		defer cleanupCancel()
		if quarantine {
			err = errors.Join(err, envelope.Quarantine(cleanupCtx))
		}
	}()
	runCtx, cancel := context.WithTimeout(ctx, p.Preparation.Manifest.Limits.Timeout)
	defer cancel()
	pre := map[string]Control{}
	controlsOK := true
	for _, target := range p.Preparation.Manifest.NegativeTargets {
		control, controlErr := envelope.Control(runCtx, target)
		pre[target] = control
		controlsOK = controlsOK && controlErr == nil
	}
	if err := runCtx.Err(); err != nil {
		return out, err
	}
	fresh, err = s.PreviewProbe(runCtx, p.Target)
	if err != nil || !reflect.DeepEqual(fresh, p) {
		return out, ErrStalePreview
	}
	result := s.ProbeAdapter.Run(runCtx, p.Preparation.Invocation) // exactly once
	verification, verifyErr := envelope.Verify(runCtx, result)
	post := map[string]Control{}
	for _, target := range p.Preparation.Manifest.NegativeTargets {
		control, controlErr := envelope.Control(runCtx, target)
		post[target] = control
		controlsOK = controlsOK && controlErr == nil
	}
	for i := range verification.Negatives {
		n := &verification.Negatives[i]
		n.Pre, n.Post = pre[n.Target], post[n.Target]
	}
	interrupted := runCtx.Err() != nil || result.TimedOut || result.OutcomeUnknown || result.Stopped || result.EffectAmbiguous
	correlated := verification.ProbeID == p.Preparation.Manifest.ProbeID && verification.ManifestDigest == p.Preparation.ManifestDigest && verification.Nonce == p.Preparation.Manifest.Nonce && verification.PacketDigest == packetDigest(p.Preparation.Invocation.Stdin)
	inconclusive := !correlated || interrupted || verifyErr != nil || verification.Inconclusive || !verification.ModelInvoked || !positiveResult(verification.Positive, result)
	positive := verification.Positive
	positive.Authentication = true // independent proven preflight
	positive.ExitZero = result.ExitCode == 0 && !interrupted
	score := ScoreVariant(p.Target.Variant, positive, verification.Negatives)
	if (!p.Preparation.OSSandbox || !controlsOK) && score.Confinement.Level == "os-enforced" {
		score.Confinement.Level = "unknown"
		score.Confinement.OSSandboxObserved = false
	}
	facts := ProbeFacts{AuthenticationProven: true, BehavioralProbe: true, Verified: true}
	score.Capabilities[ModelInvocation] = Result{State: Unknown}
	if verification.ModelInvoked && positive.ExitZero {
		score.Capabilities[ModelInvocation] = Result{State: Proven, Facts: facts}
	}
	if p.Preparation.Current.Model.EffortMode == "explicit" {
		cap := "reasoning-effort-" + p.Preparation.Current.Model.EffortToken
		score.Capabilities[cap] = Result{State: Unknown}
		if verification.EffortVerified && score.Capabilities[ModelInvocation].State == Proven {
			score.Capabilities[cap] = Result{State: Proven, Facts: facts}
		}
	} else if !verification.EffortVerified {
		score.Capabilities[ModelInvocation] = Result{State: Unknown}
	}
	if inconclusive {
		for cap := range score.Capabilities {
			score.Capabilities[cap] = Result{State: Unknown, Facts: ProbeFacts{Inconclusive: true}}
		}
		score.Confinement = Confinement{Level: "unknown", NotTested: []string{"home", "vendor-home", "tmpdir", "system-temp", "external-network"}}
	} else {
		for cap, observation := range p.Preparation.Interface.Capabilities {
			if observation.State == Unsupported && observation.Facts.InterfaceAbsent && ValidCapability(cap) {
				score.Capabilities[cap] = observation
			}
		}
	}
	model := *p.Preparation.Current.Model
	out = ProbeOutcome{Evidence: Evidence{SchemaVersion: 1, Binding: p.Preparation.Current.Binding, Model: &model, Capabilities: score.Capabilities, Confinement: score.Confinement, ObservedAt: s.now(), ProbePacketDigest: packetDigest(p.Preparation.Invocation.Stdin)}, Inconclusive: inconclusive}
	if _, err := EncodeEvidence(out.Evidence); err != nil {
		return ProbeOutcome{}, err
	}
	finishCtx, finishCancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
	defer finishCancel()
	if !interrupted {
		if cleanupErr := envelope.Cleanup(finishCtx); cleanupErr != nil {
			return ProbeOutcome{}, cleanupErr
		}
		quarantine = false
	}
	// Store only redacted observations, never process output or host paths.
	out.EvidenceRef, err = s.ProbeWriter.Create(finishCtx, out.Evidence)
	return out, err
}

func positiveResult(p PositiveSteps, r executiongraph.ProcessResult) bool {
	return p.ResultPublished && p.Noninteractive && r.ExitCode == 0
}
