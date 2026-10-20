package runtimeprofile

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func testV2() Configuration {
	c := testConfiguration()
	c.FormatVersion = FormatVersionV2
	for i := range c.Runtimes {
		c.Runtimes[i].CredentialReference = ""
	}
	for i := range c.ModelProfiles {
		c.ModelProfiles[i].Effort = Effort{EffortExplicit, "high"}
	}
	return c
}
func TestV2ClosedCodecAndHistoricalBytes(t *testing.T) {
	c := testV2()
	w, err := Encode(c)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := Decode(w)
	if err != nil || !reflect.DeepEqual(c, decoded) {
		t.Fatalf("round trip: %v", err)
	}
	for _, candidate := range []string{
		strings.Replace(string(w), `"effort":{"mode":"explicit","value":"high"}`, `"effort":null`, 1),
		strings.Replace(string(w), `"effort":{"mode":"explicit","value":"high"}`, `"effort":{"mode":"explicit","value":"high","unknown":true}`, 1),
		strings.Replace(string(w), `"mode":"explicit"`, `"mode":"explicit","mode":"runtime-default"`, 1),
		strings.Replace(string(w), `"formatVersion":2`, `"formatVersion":2,"formatVersion":1`, 1),
		strings.Replace(string(w), `"adapter":"codex"`, `"adapter":"codex","credentialReference":""`, 1),
		strings.Replace(string(w), `"revision":7`, `"revision":7.5`, 1),
		strings.Replace(string(w), `"revision":7`, `"revision":18446744073709551616`, 1),
		strings.Replace(string(w), `"model":"local-codex-model"`, `"model":"/Users/operator/private"`, 1),
		strings.Replace(string(w), `"value":"high"`, `"value":"token=synthetic"`, 1),
		strings.Replace(string(w), `"capabilities":["go","repository-write"]`, `"capabilities":["reasoning-effort-high"]`, 1),
	} {
		if _, err := Decode([]byte(candidate)); !errors.Is(err, ErrInvalidConfiguration) {
			t.Fatal("invalid v2 accepted")
		}
	}
	legacy := testConfiguration()
	before, _ := Encode(legacy)
	if strings.Contains(string(before), `"effort"`) {
		t.Fatal("historical effort invented")
	}
	after, err := Decode(before)
	again, _ := Encode(after)
	if err != nil || string(before) != string(again) {
		t.Fatal("historical bytes changed")
	}
	mixed := strings.Replace(string(before), `"model":"local-codex-model"`, `"model":"local-codex-model","effort":null`, 1)
	if _, err := Decode([]byte(mixed)); err == nil {
		t.Fatal("v1 effort accepted")
	}
}
func TestTombstoneClosedMetadata(t *testing.T) {
	tstone := Tombstone{TombstoneKind, 1, 8, strings.Repeat("a", 64)}
	w, err := EncodeTombstone(tstone)
	if err != nil {
		t.Fatal(err)
	}
	got, err := DecodeTombstone(w)
	if err != nil || got != tstone {
		t.Fatal("tombstone roundtrip failed")
	}
	for _, candidate := range []string{strings.Replace(string(w), `"revision":8`, `"revision":0`, 1), strings.Replace(string(w), `"formatVersion":1`, `"formatVersion":2`, 1), strings.Replace(string(w), `"kind":`, `"configuration":{},"kind":`, 1), strings.Replace(string(w), strings.Repeat("a", 64), strings.Repeat("A", 64), 1)} {
		if _, err := DecodeTombstone([]byte(candidate)); err == nil {
			t.Fatal("invalid tombstone accepted")
		}
	}
	if _, err := Decode(w); err == nil {
		t.Fatal("retirement decoded as active")
	}
}
func TestV2CanonicalProfileEffortAndBounds(t *testing.T) {
	c := testV2()
	d, _ := Digest(c)
	c.ModelProfiles[0].Effort = Effort{EffortRuntimeDefault, "default"}
	next, _ := Digest(c)
	if d == next {
		t.Fatal("effort not bound")
	}
	p := c.ModelProfiles[0]
	pd, _ := ProfileDigest(p)
	p.Capabilities = []string{"repository-write", "go"}
	again, _ := ProfileDigest(p)
	if pd != again {
		t.Fatal("capability order changed profile digest")
	}
	c.Runtimes[0].AllowlistedProfileIDs = nil
	if Validate(c) == nil {
		t.Fatal("empty allowlist accepted")
	}
	c = testV2()
	for i := 0; i < 31; i++ {
		p := c.ModelProfiles[0]
		p.ID = "p" + strings.Repeat("a", i+1)
		c.ModelProfiles = append(c.ModelProfiles, p)
		c.Runtimes[0].AllowlistedProfileIDs = append(c.Runtimes[0].AllowlistedProfileIDs, p.ID)
	}
	if Validate(c) == nil {
		t.Fatal("33 profiles accepted")
	}
}
func TestV2UnsafePayloadNeverEchoed(t *testing.T) {
	for _, v := range []string{"https://example.invalid/token=synthetic", "%2574oken%253Dsynthetic", "C:\\Users\\operator\\private", "$(command)", "token\n=synthetic", "sk-synthetic"} {
		c := testV2()
		c.ModelProfiles[0].Model = v
		_, err := Encode(c)
		if err == nil || strings.Contains(err.Error(), v) {
			t.Fatal("unsafe payload accepted/reflected")
		}
	}
	p := ResolvedProfile{}
	if ValidateResolvedProfile(p) == nil {
		t.Fatal("empty snapshot accepted")
	}
}

func TestExactV2AndTombstoneMemberNames(t *testing.T) {
	cfg := testV2()
	wire, err := Encode(cfg)
	if err != nil {
		t.Fatal(err)
	}
	for _, change := range [][2]string{{`"formatVersion"`, `"FormatVersion"`}, {`"runtimes"`, `"Runtimes"`}, {`"effort"`, `"Effort"`}, {`"mode"`, `"Mode"`}, {`"value"`, `"Value"`}, {`"adapter":"codex"`, `"adapter":"codex","CredentialReference":""`}, {`"model":"local-codex-model"`, `"model":"local-codex-model","Model":"other"`}} {
		if _, err := Decode([]byte(strings.Replace(string(wire), change[0], change[1], 1))); err == nil {
			t.Fatal("noncanonical v2 member accepted")
		}
	}
	ts, _ := EncodeTombstone(Tombstone{TombstoneKind, 1, 8, strings.Repeat("a", 64)})
	for _, key := range []string{"kind", "formatVersion", "revision", "priorConfigurationDigest"} {
		alias := strings.ToUpper(key[:1]) + key[1:]
		if _, err := DecodeTombstone([]byte(strings.Replace(string(ts), `"`+key+`"`, `"`+alias+`"`, 1))); err == nil {
			t.Fatal("noncanonical tombstone member accepted")
		}
	}
	legacy, _ := Encode(testConfiguration())
	for _, effort := range []string{`"Effort":null`, `"EFFORT":{}`} {
		mixed := strings.Replace(string(legacy), `"model":"local-codex-model"`, `"model":"local-codex-model",`+effort, 1)
		if _, err := Decode([]byte(mixed)); err == nil {
			t.Fatal("legacy mixed ownership accepted")
		}
	}
}
