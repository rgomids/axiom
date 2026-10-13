package executiongraph

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"testing"
)

func TestIntegrationReceiptHistoricalBytesAndVersionDispatch(t *testing.T) {
	// Immutable B4/T33 receipt: never regenerate it to accommodate a reader.
	const path = "../compatibility/testdata/stable-v1/snapshots/v0.12.0-issue276.b4t33/state/graphs/v1/123e4567-e89b-42d3-a456-426614174000/f1e3274b3d176628c2f77a03254bfe31d6f899668ada48d12e7117fde1cb885b.json"
	wire, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	a, err := DecodeIntegrationReceipt(wire)
	if err != nil || a.FormatVersion != 1 || a.ExecutionID != "" {
		t.Fatalf("historical receipt refused: %v", err)
	}
	encoded, err := EncodeIntegrationReceipt(a)
	if err != nil || !bytes.Equal(encoded, wire) {
		t.Fatalf("historical bytes changed: %v", err)
	}
	sum := sha256.Sum256(wire)
	if hex.EncodeToString(sum[:]) != "f1e3274b3d176628c2f77a03254bfe31d6f899668ada48d12e7117fde1cb885b" {
		t.Fatal("historical digest changed")
	}
	a.ExecutionID = "00000000-0000-4000-8000-0000000005e1"
	if _, err := EncodeIntegrationReceipt(a); err == nil {
		t.Fatal("format 1 meaning extended silently")
	}
	a.FormatVersion = 2
	encoded, err = EncodeIntegrationReceipt(a)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := DecodeIntegrationReceipt(encoded)
	if err != nil || decoded.ExecutionID != a.ExecutionID {
		t.Fatalf("format 2 lost distinct execution binding: %v", err)
	}
	for _, version := range []int{0, 3, 99} {
		a.FormatVersion = version
		if _, err := EncodeIntegrationReceipt(a); err == nil {
			t.Fatalf("unsupported version %d accepted", version)
		}
	}
}
