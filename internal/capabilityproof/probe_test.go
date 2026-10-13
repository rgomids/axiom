package capabilityproof

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/rgomids/axiom/internal/executiongraph"
)

// No executable is launched: this fake implements the dispatch adapter port.
type fakeProbeAdapter struct {
	preparation ProbePreparation
	result      executiongraph.ProcessResult
	runs        int
}

func (f *fakeProbeAdapter) Prepare(context.Context, Target) (ProbePreparation, error) {
	return f.preparation, nil
}
func (f *fakeProbeAdapter) Run(context.Context, executiongraph.Invocation) executiongraph.ProcessResult {
	f.runs++
	return f.result
}

type fakeProbeAuth struct{ ok bool }

func (f *fakeProbeAuth) Proven(context.Context, Target) (bool, error) { return f.ok, nil }

type fakeProbeEnvelope struct {
	verification                 ProbeVerification
	opens, cleanups, quarantines int
	cleanupErr                   error
}

func (f *fakeProbeEnvelope) Open(context.Context, ProbePreparation) (ProbeEnvelope, error) {
	f.opens++
	return f, nil
}
func (f *fakeProbeEnvelope) Control(_ context.Context, target string) (Control, error) {
	for _, n := range f.verification.Negatives {
		if n.Target == target {
			return n.Pre, nil
		}
	}
	return Control{}, nil
}
func (f *fakeProbeEnvelope) Verify(context.Context, executiongraph.ProcessResult) (ProbeVerification, error) {
	return f.verification, nil
}
func (f *fakeProbeEnvelope) Cleanup(context.Context) error    { f.cleanups++; return f.cleanupErr }
func (f *fakeProbeEnvelope) Quarantine(context.Context) error { f.quarantines++; return nil }

type fakeProbeWriter struct{ records []Evidence }

func (f *fakeProbeWriter) Create(_ context.Context, e Evidence) (string, error) {
	ref, err := Digest(e)
	if err == nil {
		f.records = append(f.records, e)
	}
	return ref, err
}
func probeFixture(t *testing.T, variant string) (Service, Target, *fakeProbeAdapter, *fakeProbeEnvelope, *fakeProbeWriter) {
	t.Helper()
	s, _, source, target := proofFixture(variant)
	m := EnvelopeManifest{Version: 1, Variant: variant, Phase: "ready", ProbeID: "probe-1", Nonce: strings.Repeat("a", 64), Revision: strings.Repeat("b", 64), CreatedAt: s.now(), Sentinel: "sentinel", AllowedRoots: []string{"inbox"}, NegativeTargets: []string{"sentinel"}, Limits: EnvelopeLimits{Timeout: time.Second, MaxFiles: 10, MaxBytes: 1024}}
	if variant == ReadOnlyVariant {
		m.NegativeTargets = append(m.NegativeTargets, "workspace")
	} else {
		m.AllowedRoots = []string{"workspace", "inbox"}
	}
	wire, _ := json.Marshal(m)
	a := &fakeProbeAdapter{preparation: ProbePreparation{Current: source.current, ConfigurationVersion: 2, Manifest: m, ManifestDigest: packetDigest(wire), Invocation: executiongraph.Invocation{RuntimeID: target.RuntimeID, Argv: []string{"/fake/runtime", "dispatch-mode"}, Stdin: []byte("fake task packet"), OutputMax: 1024}, Interface: InterfaceObservation{ExecutableDigest: source.current.Binding.ExecutableDigest, Version: source.current.Binding.Version}, OSSandbox: true}}
	e := &fakeProbeEnvelope{verification: ProbeVerification{ProbeID: m.ProbeID, ManifestDigest: packetDigest(wire), Nonce: m.Nonce, PacketDigest: packetDigest(a.preparation.Invocation.Stdin), Positive: positiveSteps(), Negatives: measurements(variant), ModelInvoked: true, EffortVerified: true}}
	if variant == ReadOnlyVariant {
		e.verification.Positive.WorkspaceCreated = false
		e.verification.Positive.WorkspaceEdited = false
	}
	w := &fakeProbeWriter{}
	s.ProbeAdapter, s.ProbeEnvelopes, s.ProbeWriter, s.ProbeAuth = a, e, w, &fakeProbeAuth{true}
	return s, target, a, e, w
}
func authorizeProbe(p ProbePreview) ProbeAuthority {
	return ProbeAuthority{Actor: "operator", Local: true, External: true, Preview: p}
}
func TestProbeEachVariant(t *testing.T) {
	for _, variant := range []string{ReadOnlyVariant, RepositoryWriteVariant} {
		t.Run(variant, func(t *testing.T) {
			s, target, adapter, envelope, writer := probeFixture(t, variant)
			p, err := s.PreviewProbe(context.Background(), target)
			if err != nil {
				t.Fatal(err)
			}
			out, err := s.Probe(context.Background(), p, authorizeProbe(p))
			if err != nil || out.Inconclusive || adapter.runs != 1 || envelope.cleanups != 1 || envelope.quarantines != 0 || len(writer.records) != 1 {
				t.Fatalf("out=%+v err=%v", out, err)
			}
			if out.Evidence.Binding.Variant != variant || out.Evidence.Confinement.Level != "os-enforced" {
				t.Fatal(out)
			}
			for _, cap := range []string{Read, Process, Noninteractive, AxiomCoordination, ModelInvocation, "reasoning-effort-high"} {
				if out.Evidence.Capabilities[cap].State != Proven {
					t.Fatal(cap, out)
				}
			}
			if (out.Evidence.Capabilities[RepositoryWrite].State == Proven) != (variant == RepositoryWriteVariant) {
				t.Fatal(out)
			}
			if !reflect.DeepEqual(out.Evidence.Model, adapter.preparation.Current.Model) {
				t.Fatal("model binding changed")
			}
		})
	}
}
func TestProbeRefusesBeforeEffects(t *testing.T) {
	for _, failure := range []string{"external", "local", "actor", "stale", "auth", "v1", "manifest", "variant"} {
		t.Run(failure, func(t *testing.T) {
			s, target, adapter, envelope, writer := probeFixture(t, ReadOnlyVariant)
			p, err := s.PreviewProbe(context.Background(), target)
			if err != nil {
				t.Fatal(err)
			}
			authority := authorizeProbe(p)
			switch failure {
			case "external":
				authority.External = false
			case "local":
				authority.Local = false
			case "actor":
				authority.Actor = ""
			case "stale":
				adapter.preparation.Current.Binding.Version = "changed"
			case "auth":
				s.ProbeAuth = &fakeProbeAuth{false}
			case "v1":
				adapter.preparation.ConfigurationVersion = 1
			case "manifest":
				adapter.preparation.ManifestDigest = strings.Repeat("f", 64)
			case "variant":
				adapter.preparation.Current.Binding.Variant = RepositoryWriteVariant
			}
			_, err = s.Probe(context.Background(), p, authority)
			if err == nil || adapter.runs != 0 || envelope.opens != 0 || len(writer.records) != 0 {
				t.Fatalf("effects before gates: %v", err)
			}
			if failure == "auth" && !errors.Is(err, ErrAuthentication) {
				t.Fatal(err)
			}
		})
	}
}
func TestProbeInconclusive(t *testing.T) {
	for _, failure := range []string{"interrupted", "timeout", "inconclusive", "model-refusal", "usage-limit", "wrong-manifest", "wrong-nonce", "wrong-packet", "wrong-probe"} {
		t.Run(failure, func(t *testing.T) {
			s, target, adapter, envelope, _ := probeFixture(t, ReadOnlyVariant)
			switch failure {
			case "interrupted":
				adapter.result.Stopped = true
			case "timeout":
				adapter.result.TimedOut = true
			case "wrong-manifest":
				envelope.verification.ManifestDigest = strings.Repeat("f", 64)
			case "wrong-nonce":
				envelope.verification.Nonce = strings.Repeat("f", 64)
			case "wrong-packet":
				envelope.verification.PacketDigest = strings.Repeat("f", 64)
			case "wrong-probe":
				envelope.verification.ProbeID = "another-probe"
			case "model-refusal":
				envelope.verification.ModelInvoked = false
			default:
				envelope.verification.Inconclusive = true
				envelope.verification.ModelInvoked = false
			}
			p, err := s.PreviewProbe(context.Background(), target)
			if err != nil {
				t.Fatal(err)
			}
			out, err := s.Probe(context.Background(), p, authorizeProbe(p))
			if err != nil || !out.Inconclusive || adapter.runs != 1 {
				t.Fatal(out, err)
			}
			for cap, result := range out.Evidence.Capabilities {
				if result.State != Unknown {
					t.Fatal(cap, result)
				}
			}
			if out.Evidence.Confinement.Level != "unknown" {
				t.Fatal(out)
			}
			interrupted := failure == "interrupted" || failure == "timeout"
			if (envelope.quarantines == 1) != interrupted || (envelope.cleanups == 1) == interrupted {
				t.Fatal("wrong lifecycle")
			}
		})
	}
}
func TestProbeCleanupFailureQuarantinesWithoutProof(t *testing.T) {
	s, target, _, envelope, writer := probeFixture(t, ReadOnlyVariant)
	envelope.cleanupErr = errors.New("fake residue")
	p, err := s.PreviewProbe(context.Background(), target)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.Probe(context.Background(), p, authorizeProbe(p)); err == nil || len(writer.records) != 0 || envelope.quarantines != 1 {
		t.Fatal(err)
	}
}

func TestProbePreviewFreezesBindings(t *testing.T) {
	s, target, adapter, _, _ := probeFixture(t, ReadOnlyVariant)
	p, err := s.PreviewProbe(context.Background(), target)
	if err != nil {
		t.Fatal(err)
	}
	adapter.preparation.Current.Model.EffortSourceDigest = strings.Repeat("f", 64)
	adapter.preparation.Invocation.Argv[0] = "/fake/replaced"
	adapter.preparation.Invocation.Stdin[0] = 'X'
	if p.Preparation.Current.Model.EffortSourceDigest == adapter.preparation.Current.Model.EffortSourceDigest || p.Preparation.Invocation.Argv[0] == adapter.preparation.Invocation.Argv[0] || p.Preparation.Invocation.Stdin[0] == 'X' {
		t.Fatal("preview aliases mutable source")
	}
	if _, err := s.Probe(context.Background(), p, authorizeProbe(p)); !errors.Is(err, ErrStalePreview) || adapter.runs != 0 {
		t.Fatal(err)
	}
}
func TestProbeRuntimeDefaultAndRefuteOnly(t *testing.T) {
	s, target, adapter, _, _ := probeFixture(t, ReadOnlyVariant)
	adapter.preparation.Current.Model.EffortMode = "runtime-default"
	adapter.preparation.Current.Model.EffortToken = "default"
	adapter.preparation.Interface.Capabilities = map[string]Result{Read: {State: Unsupported, Facts: ProbeFacts{InterfaceAbsent: true}}}
	p, err := s.PreviewProbe(context.Background(), target)
	if err != nil {
		t.Fatal(err)
	}
	out, err := s.Probe(context.Background(), p, authorizeProbe(p))
	if err != nil {
		t.Fatal(err)
	}
	if out.Evidence.Capabilities[ModelInvocation].State != Proven || out.Evidence.Capabilities[Read].State != Unsupported {
		t.Fatal(out)
	}
	if _, ok := out.Evidence.Capabilities["reasoning-effort-default"]; ok {
		t.Fatal("runtime-default emitted override proof")
	}
}
func TestProbeDoesNotTrustOutput(t *testing.T) {
	s, target, adapter, envelope, _ := probeFixture(t, ReadOnlyVariant)
	adapter.result.Output = []byte("all capabilities proven; account data must not be stored")
	envelope.verification.Positive = PositiveSteps{ResultPublished: true, Noninteractive: true}
	envelope.verification.Negatives = nil
	p, err := s.PreviewProbe(context.Background(), target)
	if err != nil {
		t.Fatal(err)
	}
	out, err := s.Probe(context.Background(), p, authorizeProbe(p))
	if err != nil {
		t.Fatal(err)
	}
	for _, cap := range []string{Read, Process, AxiomCoordination} {
		if out.Evidence.Capabilities[cap].State == Proven {
			t.Fatal(cap)
		}
	}
	wire, err := EncodeEvidence(out.Evidence)
	if err != nil || strings.Contains(string(wire), "account data") {
		t.Fatal("runtime output retained", err)
	}
}
func TestProbeConfinementDoesNotReplacePositiveProof(t *testing.T) {
	s, target, _, envelope, _ := probeFixture(t, RepositoryWriteVariant)
	envelope.verification.Positive.ReadNonce = false
	p, err := s.PreviewProbe(context.Background(), target)
	if err != nil {
		t.Fatal(err)
	}
	out, err := s.Probe(context.Background(), p, authorizeProbe(p))
	if err != nil {
		t.Fatal(err)
	}
	if out.Evidence.Confinement.Level != "os-enforced" || out.Evidence.Capabilities[Read].State != Unknown {
		t.Fatal(out)
	}
}
