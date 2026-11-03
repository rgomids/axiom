package executiongraph

import "testing"

func TestHistoricalGraphCodecRoundTrip(t *testing.T) {
	const path = "../compatibility/testdata/stable-v1/snapshots/v0.12.0-issue276/state/graphs/v1/project-1/11e594f481958c10e3015d0bf0447a22f068a8a647f475df15ce2c7ab4b8f3f1.json"
	wire := mustRead(t, path)
	decoded, err := DecodeGraph(wire)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := EncodeGraph(decoded)
	if err != nil || string(encoded) != string(wire) {
		t.Fatalf("historical bytes changed: %v", err)
	}
}
