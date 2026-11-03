package executiongraph

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func successorVector(t *testing.T, name string) SuccessorGraph {
	t.Helper()
	b, e := os.ReadFile("testdata/successor/valid/" + name + ".json")
	if e != nil {
		t.Fatal(e)
	}
	g, e := DecodeSuccessorGraph(b)
	if e != nil {
		t.Fatal(name, e)
	}
	return g
}
func TestSuccessorCanonicalVectors(t *testing.T) {
	files, e := filepath.Glob("testdata/successor/valid/*.json")
	if e != nil || len(files) < 16 {
		t.Fatal(files, e)
	}
	for _, name := range files {
		t.Run(filepath.Base(name), func(t *testing.T) {
			b, e := os.ReadFile(name)
			if e != nil {
				t.Fatal(e)
			}
			g, e := DecodeSuccessorGraph(b)
			if e != nil {
				t.Fatal(e)
			}
			encoded, e := EncodeSuccessorGraph(g)
			if e != nil || !bytes.Equal(encoded, b) {
				t.Fatal("noncanonical vector", e)
			}
			if g.StorageRevision != sha256.Sum256(b) {
				t.Fatal("wire revision")
			}
			if _, e := DecodeGraph(b); e == nil {
				t.Fatal("historical reader admitted successor")
			}
			d, e := DecodeGraphDocument(b)
			if e != nil || d.Successor == nil || d.Historical != nil {
				t.Fatal("dispatch", e)
			}
			reencoded, e := EncodeGraphDocument(d)
			if e != nil || !bytes.Equal(b, reencoded) {
				t.Fatal(e)
			}
		})
	}
}
func TestSuccessorInvalidVectors(t *testing.T) {
	files, _ := filepath.Glob("testdata/successor/invalid/*.json")
	if len(files) < 10 {
		t.Fatal(files)
	}
	for _, name := range files {
		t.Run(filepath.Base(name), func(t *testing.T) {
			b, e := os.ReadFile(name)
			if e != nil {
				t.Fatal(e)
			}
			if _, e := DecodeSuccessorGraph(b); e == nil {
				t.Fatal("accepted invalid vector")
			}
			if _, e := DecodeGraphDocument(b); e == nil {
				t.Fatal("dispatcher accepted invalid vector")
			}
		})
	}
	g := successorVector(t, "result_recorded")
	g.Children[0].Attempts[0].ClassificationReason = "invented"
	if _, e := EncodeSuccessorGraph(g); e == nil {
		t.Fatal("writer accepted invalid domain")
	}
}
func TestSuccessorHistoricalBytesAndDigests(t *testing.T) {
	b, e := os.ReadFile("testdata/successor/historical-digests.json")
	if e != nil {
		t.Fatal(e)
	}
	var digests map[string]string
	if e = json.Unmarshal(b, &digests); e != nil {
		t.Fatal(e)
	}
	for name, expected := range digests {
		t.Run(name, func(t *testing.T) {
			wire, e := os.ReadFile(filepath.Join("../..", name))
			if e != nil {
				t.Fatal(e)
			}
			sum := sha256.Sum256(wire)
			if hex.EncodeToString(sum[:]) != expected {
				t.Fatal("historical digest changed")
			}
			d, e := DecodeGraphDocument(wire)
			if e != nil || d.Historical == nil || d.Successor != nil {
				t.Fatal(e)
			}
			out, e := EncodeGraphDocument(d)
			if e != nil || !bytes.Equal(wire, out) {
				t.Fatal("historical bytes changed", e)
			}
			if _, e := DecodeSuccessorGraph(wire); e == nil {
				t.Fatal("historical bytes admitted as successor")
			}
		})
	}
	for _, wire := range []string{`{"formatVersion":99}`, `{"formatVersion":3,"formatVersion":2}`, `{}`, `null`} {
		if _, e := DecodeGraphDocument([]byte(wire)); e == nil {
			t.Fatal(wire)
		}
	}
}

func TestSuccessorCancellationAndCorrelationBounds(t *testing.T) {
	for _, change := range []func(*SuccessorGraph){
		func(g *SuccessorGraph) { g.Children[0].Attempts[0].Process.PID = 0 },
		func(g *SuccessorGraph) { g.Children[0].Attempts[0].Process.StartIdentity = "" },
		func(g *SuccessorGraph) { g.Children[0].Attempts[0].Process.GroupIdentity = strings.Repeat("x", 513) },
		func(g *SuccessorGraph) { g.Children[0].Attempts[0].PacketDigest = "" },
		func(g *SuccessorGraph) { g.StageConcurrency = MaxStageConcurrency + 1 },
		func(g *SuccessorGraph) { g.ClassificationPolicy = "unknown" },
	} {
		g := successorVector(t, "running")
		change(&g)
		if ValidateSuccessorGraph(g) == nil {
			t.Fatal("invalid correlation/envelope")
		}
	}
	cancelled := successorVector(t, "cancel_confirmed")
	cancelled.CancelRequest = nil
	if ValidateSuccessorGraph(cancelled) == nil {
		t.Fatal("cancellation without request")
	}
	if _, err := EncodeGraphDocument(GraphDocument{}); err == nil {
		t.Fatal("empty union")
	}
}
