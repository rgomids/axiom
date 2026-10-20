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
func TestNoPromotionWithoutProbe(t *testing.T) {
	for _, facts := range []ProbeFacts{{}, {AuthenticationProven: true}, {InterfaceAbsent: true}, {Verified: true}, {BehavioralProbe: true, Verified: true}, {BehavioralProbe: true, AuthenticationProven: true, Verified: true, Inconclusive: true}} {
		e := testEvidence()
		e.Capabilities[Read] = Result{Proven, facts}
		if _, err := EncodeEvidence(e); err == nil {
			t.Fatalf("accepted %+v", facts)
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
