package capabilityproof

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"
)

func testEvidence() Evidence {
	d := strings.Repeat("a", 64)
	return Evidence{SchemaVersion: 1, Binding: Binding{"codex", d, "1.0", d, ReadOnlyVariant, "1", d}, ObservedAt: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC), ProbePacketDigest: d, Capabilities: map[string]Result{Read: {Proven, ProbeFacts{AuthenticationProven: true, BehavioralProbe: true, Verified: true}}}, Confinement: Confinement{Level: "unknown", NotTested: []string{"home", "vendor-home", "tmpdir", "system-temp", "external-network"}}}
}
func TestFreshnessAndExactBinding(t *testing.T) {
	e := testEvidence()
	now := e.ObservedAt.Add(time.Hour)
	if state, _ := Evaluate(e, e.Binding, nil, Read, now); state != Proven {
		t.Fatal(state)
	}
	v := reflect.ValueOf(e.Binding)
	for i := 0; i < v.NumField(); i++ {
		b := e.Binding
		reflect.ValueOf(&b).Elem().Field(i).SetString("changed")
		state, reason := Evaluate(e, b, nil, Read, now)
		if state != Unknown || reason != "capability_stale" {
			t.Fatalf("binding %s reused", v.Type().Field(i).Name)
		}
	}
	for _, now := range []time.Time{e.ObservedAt.Add(MaxAge + time.Nanosecond), e.ObservedAt.Add(-time.Second)} {
		if state, _ := Evaluate(e, e.Binding, nil, Read, now); state != Unknown {
			t.Fatal("stale proof accepted")
		}
	}
	e.InvalidatedReason = "explicit"
	if state, _ := Evaluate(e, e.Binding, nil, Read, now); state != Unknown {
		t.Fatal("invalidated")
	}
}
func TestNoPromotionWithoutProbe(t *testing.T) {
	for _, facts := range []ProbeFacts{{}, {AuthenticationProven: true}, {InterfaceAbsent: true}, {Verified: true}, {BehavioralProbe: true, Verified: true}, {BehavioralProbe: true, AuthenticationProven: true, Verified: true, Inconclusive: true}} {
		e := testEvidence()
		e.Capabilities[Read] = Result{Proven, facts}
		if _, err := EncodeEvidence(e); err == nil {
			t.Fatalf("accepted %+v", facts)
		}
	}
}
func TestModelEquality(t *testing.T) {
	e := testEvidence()
	d := strings.Repeat("b", 64)
	e.Model = &ModelBinding{"profile", d, "model", "explicit", "high", 1, d}
	e.Capabilities[ModelInvocation] = e.Capabilities[Read]
	if state, _ := Evaluate(e, e.Binding, e.Model, ModelInvocation, e.ObservedAt); state != Proven {
		t.Fatal(state)
	}
	original := *e.Model
	for i := 0; i < reflect.TypeOf(original).NumField(); i++ {
		m := original
		v := reflect.ValueOf(&m).Elem().Field(i)
		if v.Kind() == reflect.String {
			v.SetString("changed")
		} else {
			v.SetUint(2)
		}
		if state, reason := Evaluate(e, e.Binding, &m, ModelInvocation, e.ObservedAt); state != Unknown || reason != "capability_stale" {
			t.Fatal("model element", i)
		}
	}
}
func TestRedactionAndStrictDecode(t *testing.T) {
	for _, value := range []string{"/home/operator/config", "ENV=value", "sk-private-credential", "ghp_privatecredential", "account@example.com"} {
		e := testEvidence()
		e.Binding.Version = value
		if _, err := EncodeEvidence(e); err == nil {
			t.Fatal("unsafe accepted", value)
		}
	}
	e := testEvidence()
	w, err := EncodeEvidence(e)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := DecodeEvidence(w)
	if err != nil || !reflect.DeepEqual(e, decoded) {
		t.Fatal(err)
	}
	var obj map[string]any
	json.Unmarshal(w, &obj)
	obj["transcript"] = "private"
	w, _ = json.Marshal(obj)
	if _, err := DecodeEvidence(w); err == nil {
		t.Fatal("unknown schema field accepted")
	}
	e.Confinement.Level = "os-enforced"
	if _, err := EncodeEvidence(e); err == nil {
		t.Fatal("unverified confinement")
	}
}
func TestEvidenceRejectsDuplicateAndUnsafeMetadata(t *testing.T) {
	e := testEvidence()
	w, _ := EncodeEvidence(e)
	w = append([]byte(`{"schemaVersion":1,`), w[1:]...)
	if _, err := DecodeEvidence(w); err == nil {
		t.Fatal("duplicate field accepted")
	}
	for _, s := range []string{"github_pat_privatecredential", "gho_privatecredential", "AKIAEXAMPLEPRIVATE"} {
		e := testEvidence()
		e.Binding.RuntimeID = s
		if _, err := EncodeEvidence(e); err == nil {
			t.Fatal("credential-like token accepted", s)
		}
	}
	e.Confinement.Scope = []string{"/home/operator"}
	if _, err := EncodeEvidence(e); err == nil {
		t.Fatal("host scope accepted")
	}
	e = testEvidence()
	e.Confinement.NotTested = nil
	if _, err := EncodeEvidence(e); err == nil {
		t.Fatal("missing untested scope accepted")
	}
	e = testEvidence()
	e.Capabilities[RepositoryWrite] = e.Capabilities[Read]
	if _, err := EncodeEvidence(e); err == nil {
		t.Fatal("read-only write capability proven")
	}
}

func TestVariantSelectionAndUnsupported(t *testing.T) {
	e := testEvidence()
	binding := e.Binding
	binding.Variant = RepositoryWriteVariant
	if state, reason := Evaluate(e, binding, nil, Read, e.ObservedAt); state != Unknown || reason != "capability_stale" {
		t.Fatal("cross-variant reuse")
	}
	e.Binding = binding
	if state, _ := Evaluate(e, binding, nil, Read, e.ObservedAt); state != Proven {
		t.Fatal("matching write variant refused")
	}
	e.Capabilities[Process] = Result{State: Unsupported, Facts: ProbeFacts{InterfaceAbsent: true}}
	if state, reason := Evaluate(e, binding, nil, Process, e.ObservedAt); state != Unsupported || reason != "capability_unsupported" {
		t.Fatal("interface absence promoted")
	}
}

func TestConfinementRequiresScopeAndUntestedRoots(t *testing.T) {
	e := testEvidence()
	e.Confinement.NotTested = nil
	if _, err := EncodeEvidence(e); err == nil {
		t.Fatal("untested roots omitted")
	}
	e = testEvidence()
	e.Confinement.Level = "os-enforced"
	e.Confinement.ControlsVerified = true
	e.Confinement.CanaryIdentityMatched = true
	e.Confinement.PositiveWritesVerified = true
	e.Confinement.EveryNegativeDenied = true
	e.Confinement.OSSandboxObserved = true
	e.Confinement.Scope = []string{"sentinel"}
	if _, err := EncodeEvidence(e); err == nil {
		t.Fatal("read-only workspace denial missing")
	}
	e.Confinement.Scope = append(e.Confinement.Scope, "workspace")
	if _, err := EncodeEvidence(e); err != nil {
		t.Fatal(err)
	}
}
