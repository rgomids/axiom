package runtimeprofile

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"
)

func readV4Fixture(t *testing.T, name string) []byte {
	t.Helper()
	wire, err := os.ReadFile("testdata/axm36/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return wire
}

func TestV4CanonicalConfigurationAndFrozenTuple(t *testing.T) {
	goldenDigests := strings.Fields(string(readV4Fixture(t, "digests.txt")))
	if len(goldenDigests) != 4 || goldenDigests[0] != "configuration-v4" || goldenDigests[2] != "resolved-v4" {
		t.Fatal("invalid digest fixture")
	}
	wire := readV4Fixture(t, "configuration-v4.json")
	c, err := DecodeV4(wire)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := EncodeV4(c)
	if err != nil || !bytes.Equal(encoded, wire) {
		t.Fatalf("configuration round trip: %v\n%s", err, encoded)
	}
	digest, err := DigestV4(c)
	if err != nil || digest != goldenDigests[1] {
		t.Fatalf("configuration digest: %s %v", digest, err)
	}
	c.Runtimes[0], c.Runtimes[1] = c.Runtimes[1], c.Runtimes[0]
	c.ModelProfiles[0], c.ModelProfiles[1] = c.ModelProfiles[1], c.ModelProfiles[0]
	if got, _ := EncodeV4(c); !bytes.Equal(got, wire) {
		t.Fatal("set order changed configuration identity")
	}
	versioned, err := DecodeVersioned(wire)
	if err != nil || versioned.FormatVersion != FormatVersionV4 || versioned.V4 == nil || versioned.V3 != nil || versioned.Configuration != nil {
		t.Fatalf("version dispatch: %+v %v", versioned, err)
	}
	if _, err := Decode(wire); !errors.Is(err, ErrUnsupportedConfigurationFormat) {
		t.Fatal("historical configuration decoder admitted v4", err)
	}
	p, err := DecodeResolvedProfileV4(readV4Fixture(t, "resolved-v4.json"))
	if err != nil {
		t.Fatal(err)
	}
	pd, err := ResolvedProfileDigestV4(p)
	if err != nil || pd != goldenDigests[3] {
		t.Fatalf("resolved digest: %s %v", pd, err)
	}
	var historical ResolvedProfile
	encoded, _ = json.Marshal(p)
	if json.Unmarshal(encoded, &historical) != nil || ValidateResolvedProfile(historical) == nil {
		t.Fatal("successor tuple gained historical dispatch authority")
	}
	for _, key := range []string{"runtimeVersion", "capabilityProofReference", "effortDeclarations"} {
		if bytes.Contains(wire, []byte(key)) || bytes.Contains(encoded, []byte(key)) {
			t.Fatal("successor has a certification field", key)
		}
	}
}

func TestV4ClosedSchemasRejectMalformedAndHistoricalAuthority(t *testing.T) {
	for _, name := range []string{"configuration-v4.json", "resolved-v4.json"} {
		t.Run(name, func(t *testing.T) {
			wire := readV4Fixture(t, name)
			decode := func(w []byte) error {
				if name == "configuration-v4.json" {
					_, err := DecodeV4(w)
					return err
				}
				_, err := DecodeResolvedProfileV4(w)
				return err
			}
			for label, bad := range map[string][]byte{
				"truncated": wire[:len(wire)/2],
				"oversize":  bytes.Repeat([]byte(" "), MaxConfigurationBytes+1),
				"trailing":  append(append([]byte(nil), wire...), []byte("{}")...),
				"unknown":   bytes.Replace(wire, []byte("{"), []byte(`{"extra":true,`), 1),
				"proof":     bytes.Replace(wire, []byte("{"), []byte(`{"capabilityProofReference":"old-proof",`), 1),
				"version":   bytes.Replace(wire, []byte("{"), []byte(`{"runtimeVersion":"1.2.3",`), 1),
			} {
				if err := decode(bad); err == nil {
					t.Errorf("accepted %s", label)
				}
			}
			var object map[string]json.RawMessage
			if err := json.Unmarshal(wire, &object); err != nil {
				t.Fatal(err)
			}
			for key, value := range object {
				delete(object, key)
				missing, _ := json.Marshal(object)
				if err := decode(missing); err == nil {
					t.Errorf("accepted missing %s", key)
				}
				object[key] = json.RawMessage("null")
				null, _ := json.Marshal(object)
				if err := decode(null); err == nil {
					t.Errorf("accepted null %s", key)
				}
				object[key] = value
				prefix := []byte(`{"` + key + `":` + string(value) + `,`)
				duplicate := bytes.Replace(wire, []byte("{"), prefix, 1)
				if err := decode(duplicate); err == nil {
					t.Errorf("accepted duplicate %s", key)
				}
				alias := bytes.Replace(wire, []byte(`"`+key+`"`), []byte(`"`+strings.ToUpper(key)+`"`), 1)
				if err := decode(alias); err == nil {
					t.Errorf("accepted alias %s", key)
				}
			}
		})
	}
}

func TestV4ConfigurationRequiredNestedFieldsAndPolicy(t *testing.T) {
	wire := readV4Fixture(t, "configuration-v4.json")
	for label, bad := range map[string][]byte{
		"missing enabled":      bytes.Replace(wire, []byte(`"enabled":true,`), nil, 1),
		"false is present":     bytes.Replace(wire, []byte(`"enabled":true`), []byte(`"enabled":false`), 1),
		"missing effort":       bytes.Replace(wire, []byte(`,"effort":{"mode":"runtime-default","value":"default"}`), nil, 1),
		"missing effort value": bytes.Replace(wire, []byte(`,"value":"default"`), nil, 1),
		"declaration":          bytes.Replace(wire, []byte(`"adapter":"claude"`), []byte(`"adapter":"claude","effortDeclarations":[]`), 1),
		"credential":           bytes.Replace(wire, []byte(`"adapter":"codex"`), []byte(`"adapter":"codex","credentialReference":"local-ref"`), 1),
		"mismatched owner":     bytes.Replace(wire, []byte(`"runtimeId":"claude"`), []byte(`"runtimeId":"codex"`), 1),
		"unknown version":      bytes.Replace(wire, []byte(`"formatVersion":4`), []byte(`"formatVersion":5`), 1),
		"historical version":   bytes.Replace(wire, []byte(`"formatVersion":4`), []byte(`"formatVersion":3`), 1),
		"zero revision":        bytes.Replace(wire, []byte(`"revision":7`), []byte(`"revision":0`), 1),
		"opaque effort":        bytes.Replace(wire, []byte(`"runtime-default"`), []byte(`"explicit"`), 1),
	} {
		_, err := DecodeV4(bad)
		if label == "false is present" || label == "opaque effort" {
			if err != nil {
				t.Errorf("valid %s: %v", label, err)
			}
		} else if err == nil {
			t.Errorf("accepted %s", label)
		}
	}
	for _, version := range []int{1, 2, 3} {
		c, _ := DecodeV4(wire)
		c.FormatVersion = version
		if _, err := EncodeV4(c); !errors.Is(err, ErrUnsupportedConfigurationFormat) {
			t.Fatal("encoded historical configuration", version, err)
		}
	}
}

func TestV4HistoricalConfigurationBytesRemainUnchanged(t *testing.T) {
	for _, c := range []Configuration{testConfiguration(), testV2()} {
		before, _ := Encode(c)
		v, err := DecodeVersioned(before)
		if err != nil || v.V4 != nil || v.Configuration == nil {
			t.Fatal(err)
		}
		after, _ := Encode(*v.Configuration)
		if !bytes.Equal(before, after) {
			t.Fatal("historical bytes changed")
		}
	}
	wire, _ := EncodeV3(testV3())
	v, err := DecodeVersioned(wire)
	if err != nil || v.V3 == nil || v.V4 != nil {
		t.Fatal(err)
	}
	after, _ := EncodeV3(*v.V3)
	if !bytes.Equal(wire, after) || !reflect.DeepEqual(*v.V3, canonicalV3(testV3())) {
		t.Fatal("historical v3 changed")
	}
	sum := sha256.Sum256(wire)
	if digest, _ := DigestV3(*v.V3); digest != hex.EncodeToString(sum[:]) {
		t.Fatal("historical v3 digest changed")
	}
}
