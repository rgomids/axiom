package workflow

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
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

func TestI404ExplicitCapacityCodec(t *testing.T) {
	for _, limit := range []int{1, 2} {
		t.Run(fmt.Sprint(limit), func(t *testing.T) {
			wire, err := os.ReadFile("testdata/i404/dag.json")
			if err != nil {
				t.Fatal(err)
			}
			e, err := DecodeExecution(wire)
			if err != nil {
				t.Fatal(err)
			}
			e.Workflow.Stages[0].Concurrency = &limit
			input, err := os.ReadFile(fmt.Sprintf("testdata/i404/explicit-%d.digest-input.json", limit))
			if err != nil {
				t.Fatal(err)
			}
			sum := sha256.Sum256(input)
			e.Workflow.Digest = "sha256:" + hex.EncodeToString(sum[:])
			got, err := workflowDigest(e.Workflow)
			if err != nil || got != e.Workflow.Digest {
				t.Fatal("explicit independent digest mismatch", got, err)
			}
			wire, err = EncodeExecution(e)
			if err != nil {
				t.Fatal(err)
			}
			back, err := DecodeExecution(wire)
			if err != nil {
				t.Fatal(err)
			}
			if back.Workflow.Stages[0].Concurrency == nil || *back.Workflow.Stages[0].Concurrency != limit {
				t.Fatal("lost explicit capacity")
			}
			again, err := EncodeExecution(back)
			if err != nil || !bytes.Equal(wire, again) {
				t.Fatal("explicit bytes changed", err)
			}
			cloned := cloneStages(e.Workflow.Stages)
			*cloned[0].Concurrency = 3
			if *e.Workflow.Stages[0].Concurrency != limit {
				t.Fatal("clone pointer alias")
			}
			for _, value := range []string{"0", "-1", "1.5", "null", "33", "4", "99999999999999999999999999", `"2"`, "true", `{"value":2}`} {
				bad := bytes.Replace(wire, []byte(fmt.Sprintf(`"concurrency":%d`, limit)), []byte(`"concurrency":`+value), 1)
				if _, err := DecodeExecution(bad); err == nil {
					t.Fatalf("invalid capacity accepted: %s", value)
				}
			}
			for _, replacement := range []string{fmt.Sprintf(`"Concurrency":%d`, limit), fmt.Sprintf(`"concurrency":%d,"concurrency":%d`, limit, limit), fmt.Sprintf(`"concurrency":%d,"extra":1`, limit)} {
				bad := bytes.Replace(wire, []byte(fmt.Sprintf(`"concurrency":%d`, limit)), []byte(replacement), 1)
				if _, err := DecodeExecution(bad); err == nil {
					t.Fatal("invalid shape accepted", replacement)
				}
			}
			other := 3 - limit
			tampered := bytes.Replace(wire, []byte(fmt.Sprintf(`"concurrency":%d`, limit)), []byte(fmt.Sprintf(`"concurrency":%d`, other)), 1)
			if _, err := DecodeExecution(tampered); err == nil {
				t.Fatal("capacity tampering without digest accepted")
			}
			bad := bytes.Replace(wire, []byte(fmt.Sprintf(`,"concurrency":%d`, limit)), nil, 1)
			if _, err := DecodeExecution(bad); err == nil {
				t.Fatal("removed capacity without digest accepted")
			}
			bad = bytes.Replace(wire, []byte(`"timeoutSeconds":60,`), nil, 1)
			if _, err := DecodeExecution(bad); err == nil {
				t.Fatal("required member removed")
			}
		})
	}
	for _, limit := range []int{0, -1, 4, 33} {
		n := limit
		_, err := NewExecution(NewExecutionInput{ExecutionID: testExecutionID, ProjectID: "proj", WorkItem: WorkItemLink{"local", "synthetic", "app"}, Repositories: []RepositoryBinding{{"app", "example/app"}}, WorkflowID: "i404", Revision: "r1", Now: t0, Stages: []StageSpec{{ID: "work", TimeoutSeconds: 60, Concurrency: &n, Agents: []AgentSpec{{ID: "a"}, {ID: "b"}, {ID: "c"}}}}})
		if err == nil {
			t.Fatal("constructor accepted invalid bound", limit)
		}
	}
	a, b := DefaultWorkflowStages(), DefaultWorkflowStages()
	if a[0].Concurrency == nil || *a[0].Concurrency != 1 {
		t.Fatal("new default must explicitly declare 1")
	}
	*a[0].Concurrency = 2
	if *b[0].Concurrency != 1 {
		t.Fatal("defaults alias")
	}
}

func TestI404ConstructorCapacityPointersIndependent(t *testing.T) {
	n := 1
	e, err := NewExecution(NewExecutionInput{ExecutionID: testExecutionID, ProjectID: "proj", WorkItem: WorkItemLink{"local", "synthetic", "app"}, Repositories: []RepositoryBinding{{"app", "example/app"}}, WorkflowID: "i404", Revision: "r1", Now: t0, Stages: []StageSpec{{ID: "a", TimeoutSeconds: 60, Concurrency: &n, Agents: []AgentSpec{{ID: "a"}}}, {ID: "b", TimeoutSeconds: 60, Concurrency: &n, Agents: []AgentSpec{{ID: "b"}}}}})
	if err != nil {
		t.Fatal(err)
	}
	n = 2
	if *e.Workflow.Stages[0].Concurrency != 1 {
		t.Fatal("constructor aliases input")
	}
	*e.Workflow.Stages[0].Concurrency = 2
	if *e.Workflow.Stages[1].Concurrency != 1 {
		t.Fatal("stages alias")
	}
}
