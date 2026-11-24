package workflow

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// These expected bytes come from the pinned pre-amendment writer, never this
// package's current encoder. See testdata/i404/PROVENANCE.md.
func TestI404FrozenFormat4(t *testing.T) {
	manifestWire, err := os.ReadFile("testdata/i404/manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	manifestSum := sha256.Sum256(manifestWire)
	if hex.EncodeToString(manifestSum[:]) != "bea909faa01160daf0256baee9ad123def0ce9b232bdbfa5778662e150cd5ac8" {
		t.Fatal("frozen manifest changed")
	}
	var manifest map[string]string
	if err = json.Unmarshal(manifestWire, &manifest); err != nil {
		t.Fatal(err)
	}
	for name, hash := range manifest {
		wire, err := os.ReadFile("testdata/i404/" + name)
		if err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(wire)
		if hex.EncodeToString(sum[:]) != hash {
			t.Fatalf("frozen bytes changed: %s", name)
		}
	}
	for _, name := range []string{"single", "dag", "terminal", "prepared", "running", "multi-active"} {
		t.Run(name, func(t *testing.T) {
			wire, err := os.ReadFile("testdata/i404/" + name + ".json")
			if err != nil {
				t.Fatal(err)
			}
			e, err := DecodeExecution(wire)
			if err != nil {
				t.Fatal(err)
			}
			input, err := os.ReadFile("testdata/i404/" + name + ".digest-input.json")
			if err != nil {
				t.Fatal(err)
			}
			sum := sha256.Sum256(input)
			if e.Workflow.Digest != "sha256:"+hex.EncodeToString(sum[:]) {
				t.Fatal("independent digest mismatch")
			}
			if got, err := workflowDigest(WorkflowSnapshot{ID: e.Workflow.ID, Revision: e.Workflow.Revision, Stages: cloneStages(e.Workflow.Stages)}); err != nil || got != e.Workflow.Digest {
				t.Fatal("clone changed frozen digest", err)
			}
			encoded, err := EncodeExecution(e)
			if err != nil || !bytes.Equal(encoded, wire) {
				t.Fatal("frozen roundtrip changed", err)
			}
			if strings.Contains(string(encoded), `"concurrency"`) {
				t.Fatal("historical absence materialized")
			}
		})
	}
}
