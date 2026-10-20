package executiongraph

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/rgomids/axiom/internal/runtimeprofile"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestOperationalLedgerGoldenAndHistoricalIsolation(t *testing.T) {
	var digests strings.Builder
	for _, name := range []string{"graph-v4-process_failed.json", "graph-v4-process_outcome_unknown.json", "graph-v4-reconciliation.json", "graph-v4-running.json", "graph-v4.json", "single-run-v2-unknown.json", "single-run-v2.json"} {
		t.Run(name, func(t *testing.T) {
			wire := mustRead(t, "testdata/axm36/"+name)
			var encoded []byte
			var err error
			if strings.HasPrefix(name, "graph-v4") {
				g, e := DecodeSuccessorGraph(wire)
				if e != nil || g.StorageRevision != sha256.Sum256(wire) {
					t.Fatalf("decode graph: %v", e)
				}
				encoded, err = EncodeSuccessorGraph(g)
				if g.FormatVersion != OperationalGraphVersion {
					t.Fatal("new graph decoded as historical generation")
				}
			} else {
				r, e := DecodeSingleStageRun(wire)
				if e != nil || r.StorageRevision != sha256.Sum256(wire) {
					t.Fatalf("decode single: %v", e)
				}
				encoded, err = EncodeSingleStageRun(r)
				if r.FormatVersion != OperationalSingleStageRunVersion {
					t.Fatal("new run decoded as historical generation")
				}
			}
			if err != nil || !bytes.Equal(wire, encoded) {
				t.Fatalf("unstable encoding: %v", err)
			}
			for _, retired := range []string{"runtimeVersion", "capabilityProofReference", "capabilityEvidenceRef"} {
				if bytes.Contains(encoded, []byte(retired)) {
					t.Fatalf("retired field %s", retired)
				}
			}
			fmt.Fprintf(&digests, "%s %x\n", name, sha256.Sum256(encoded))
		})
	}
	if digests.String() != string(mustRead(t, "testdata/axm36/digests.txt")) {
		t.Fatal("successor digest vectors changed")
	}
	historical, err := DecodeSuccessorGraph(mustRead(t, "testdata/successor/valid/publication.json"))
	if err != nil || historical.FormatVersion != SuccessorGraphVersion || historical.Children[0].Envelope.Resolution.FrozenProfile != nil {
		t.Fatal("historical graph lost its generation", err)
	}
	run, err := DecodeSingleStageRun(mustRead(t, "testdata/single-stage/run-v1.json"))
	if err != nil || run.FormatVersion != SingleStageRunVersion || run.Binding.Resolution.FrozenProfile != nil {
		t.Fatal("historical run lost its generation", err)
	}
	for _, name := range []string{"graph-v4.json", "single-run-v2.json"} {
		wire := mustRead(t, "testdata/axm36/"+name)
		for _, older := range [][2]string{{`"formatVersion":4`, `"formatVersion":3`}, {`"formatVersion":2`, `"formatVersion":1`}} {
			downgraded := bytes.Replace(wire, []byte(older[0]), []byte(older[1]), 1)
			if bytes.Equal(downgraded, wire) {
				continue
			}
			if _, err := DecodeSuccessorGraph(downgraded); err == nil {
				t.Fatalf("%s: proof-free tuple accepted under historical generation", name)
			}
			if _, err := DecodeSingleStageRun(downgraded); err == nil {
				t.Fatalf("%s: proof-free tuple accepted under historical generation", name)
			}
		}
	}
}

func TestOperationalLedgerStrictShape(t *testing.T) {
	for _, name := range []string{"graph-v4.json", "single-run-v2.json", "single-run-v2-unknown.json"} {
		t.Run(name, func(t *testing.T) {
			wire := mustRead(t, "testdata/axm36/"+name)
			version := 2
			if strings.HasPrefix(name, "graph-v4") {
				version = 4
			}
			decode := func(b []byte) error {
				if strings.HasPrefix(name, "graph-v4") {
					_, err := DecodeSuccessorGraph(b)
					return err
				}
				_, err := DecodeSingleStageRun(b)
				return err
			}
			if decode(wire) != nil {
				t.Fatal("invalid positive fixture")
			}
			for label, bad := range map[string][]byte{
				"future":      bytes.Replace(wire, []byte(`"formatVersion":`+fmt.Sprint(version)), []byte(`"formatVersion":99`), 1),
				"duplicate":   append([]byte(`{"formatVersion":4,`), wire[1:]...),
				"alias":       bytes.Replace(wire, []byte(`"formatVersion"`), []byte(`"FormatVersion"`), 1),
				"unknown":     append([]byte(`{"extra":false,`), wire[1:]...),
				"truncated":   wire[:len(wire)/2],
				"oversize":    bytes.Repeat([]byte(" "), MaxGraphBytes+1),
				"trailing":    append(append([]byte(nil), wire...), []byte(`{}`)...),
				"old-proof":   bytes.Replace(wire, []byte(`"profileId":`), []byte(`"capabilityProofReference":"retired","profileId":`), 1),
				"old-version": bytes.Replace(wire, []byte(`"profileId":`), []byte(`"runtimeVersion":"99.1","profileId":`), 1),
			} {
				if decode(bad) == nil {
					t.Errorf("admitted %s", label)
				}
			}
			var object any
			if json.Unmarshal(wire, &object) != nil {
				t.Fatal("bad fixture")
			}
			var visit func(any, string)
			visit = func(v any, path string) {
				switch value := v.(type) {
				case map[string]any:
					for key, child := range value {
						if key != "diagnostic" && key != "process" && key != "cancellationSeen" {
							delete(value, key)
							bad, _ := json.Marshal(object)
							if decode(bad) == nil {
								t.Errorf("missing required %s/%s accepted", path, key)
							}
							value[key] = child
						}
						visit(child, path+"/"+key)
					}
				case []any:
					for i, child := range value {
						visit(child, fmt.Sprintf("%s/%d", path, i))
					}
				}
			}
			visit(object, "")
		})
	}
}

func TestResolutionAndControls(t *testing.T) {
	g, err := DecodeSuccessorGraph(mustRead(t, "testdata/axm36/graph-v4.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range g.Children {
		r := c.Envelope.Resolution
		for name, mutate := range map[string]func(*Resolution){
			"runtime":       func(r *Resolution) { r.RuntimeID = "other" },
			"profile":       func(r *Resolution) { r.ModelProfileID = "other" },
			"configuration": func(r *Resolution) { r.ConfigurationRevision++ },
			"observation":   func(r *Resolution) { r.ObservationRevision++ },
			"digest": func(r *Resolution) {
				frozen := *r.FrozenProfile
				frozen.ExecutableDigest = "bad"
				r.FrozenProfile = &frozen
			},
			"historical": func(r *Resolution) { r.ResolvedProfile = &runtimeprofile.ResolvedProfile{} },
		} {
			t.Run(r.RuntimeID+"/"+name, func(t *testing.T) {
				bad := r
				mutate(&bad)
				if validResolution(bad) {
					t.Fatal("mismatched tuple accepted")
				}
			})
		}
		controls := c.Envelope.Controls
		controls.ReasoningEffort = "other"
		if ValidFrozenProfileControls(r.FrozenProfile, controls, c.Envelope.Capability) {
			t.Fatal("competing controls accepted")
		}
	}
}

func TestOperationalGraphRetainsSafetyChecks(t *testing.T) {
	wire := mustRead(t, "testdata/axm36/graph-v4.json")
	for name, mutate := range map[string]func(*SuccessorGraph){
		"authority digest": func(g *SuccessorGraph) { g.Parent.AuthorityDigest = strings.Repeat("f", 64) },
		"effect ceiling": func(g *SuccessorGraph) {
			g.Children[0].Envelope.AllowedEffects = []Effect{{Kind: "repository-write", Target: "src"}}
		},
		"cycle":           func(g *SuccessorGraph) { g.Children[0].Envelope.Dependencies = []string{g.Children[1].ExecutionID} },
		"duplicate child": func(g *SuccessorGraph) { g.Children[0].ExecutionID = g.Children[1].ExecutionID },
		"profile drift":   func(g *SuccessorGraph) { g.Children[0].Envelope.Resolution.FrozenProfile.ProfileID = "other" },
		"concurrency":     func(g *SuccessorGraph) { g.StageConcurrency = 0 },
		"integration":     func(g *SuccessorGraph) { g.Parent.IntegrationChild = g.Children[0].ExecutionID },
		"cancel authority": func(g *SuccessorGraph) {
			g.CancelRequest = &CancelRequest{RequestID: "00000000-0000-4000-8000-000000000099", ParentID: g.Parent.ExecutionID, GraphRevision: g.Parent.GraphRevision, ActorKind: "agent", Actor: "synthetic", RequestedAt: g.Parent.CreatedAt, AuthorityRef: g.DispatchAuthorityRef}
		},
	} {
		t.Run(name, func(t *testing.T) {
			g, err := DecodeSuccessorGraph(wire)
			if err != nil {
				t.Fatal(err)
			}
			mutate(&g)
			if _, err := EncodeSuccessorGraph(g); err == nil {
				t.Fatal("unsafe graph accepted")
			}
		})
	}
}

func TestOperationalSingleRetainsSafetyChecks(t *testing.T) {
	wire := mustRead(t, "testdata/axm36/single-run-v2.json")
	for name, mutate := range map[string]func(*SingleStageRun){
		"epoch":               func(r *SingleStageRun) { r.RunEpoch = 0 },
		"authority":           func(r *SingleStageRun) { r.Authorities = nil },
		"duplicate authority": func(r *SingleStageRun) { r.Authorities = append(r.Authorities, r.Authorities[0]) },
		"scope":               func(r *SingleStageRun) { r.Scope.StageID = "other" },
		"paths":               func(r *SingleStageRun) { r.Binding.Inbox = r.Binding.Workspace },
		"attempt limit":       func(r *SingleStageRun) { r.Binding.Controls.MaximumAttempts = 0 },
		"timeout":             func(r *SingleStageRun) { r.Binding.Controls.Timeout = 25 * time.Hour },
		"effort":              func(r *SingleStageRun) { r.Binding.Controls.ReasoningEffort = "high" },
		"historical version":  func(r *SingleStageRun) { r.FormatVersion = 1 },
	} {
		t.Run(name, func(t *testing.T) {
			r, err := DecodeSingleStageRun(wire)
			if err != nil {
				t.Fatal(err)
			}
			mutate(&r)
			if _, err := EncodeSingleStageRun(r); err == nil {
				t.Fatal("unsafe run accepted")
			}
		})
	}
}

func operationalGraphVector(t *testing.T, name string) SuccessorGraph {
	t.Helper()
	g, err := DecodeSuccessorGraph(mustRead(t, "testdata/axm36/graph-v4-"+name+".json"))
	if err != nil {
		t.Fatal(err)
	}
	return g
}

func TestOperationalGraphTerminalPublicationAndReconciliation(t *testing.T) {
	running := operationalGraphVector(t, "running")
	finished := operationalGraphVector(t, "process_failed")
	if ValidateSuccessorUpdate(running, finished) != nil {
		t.Fatal("atomic terminal state and diagnostic publication refused")
	}
	old := operationalGraphVector(t, "process_outcome_unknown")
	next := operationalGraphVector(t, "reconciliation")
	if ValidateSuccessorUpdate(old, next) != nil {
		t.Fatal("append-only reconciliation refused")
	}
	for name, mutate := range map[string]func(*SuccessorGraph){
		"remove diagnostic": func(g *SuccessorGraph) { g.Children[0].Attempts[0].Diagnostic = nil },
		"rewrite diagnostic": func(g *SuccessorGraph) {
			g.Children[0].Attempts[0].Diagnostic.Cause = "authentication_unavailable"
			g.Children[0].Attempts[0].Diagnostic.CauseConfidence = "confirmed"
		},
		"rewrite fact": func(g *SuccessorGraph) {
			g.Children[0].Attempts[0].ReconciliationFacts[0].ObservationsDigest = strings.Repeat("f", 64)
		},
		"remove fact": func(g *SuccessorGraph) { g.Children[0].Attempts[0].ReconciliationFacts = nil },
		"rewrite binding": func(g *SuccessorGraph) {
			g.Children[0].Envelope.Resolution.FrozenProfile.ExecutableDigest = strings.Repeat("f", 64)
		},
		"rewrite exit": func(g *SuccessorGraph) { g.Children[0].Attempts[0].ExitCode++ },
	} {
		t.Run(name, func(t *testing.T) {
			bad := operationalGraphVector(t, "reconciliation")
			mutate(&bad)
			if ValidateSuccessorUpdate(next, bad) == nil {
				t.Fatal("immutable publication rewritten")
			}
		})
	}
	withoutDiagnostic := operationalGraphVector(t, "process_failed")
	withoutDiagnostic.Children[0].Attempts[0].Diagnostic = nil
	if ValidateSuccessorUpdate(withoutDiagnostic, finished) == nil {
		t.Fatal("historical cause backfilled")
	}
}

func TestOperationalSingleBothAdaptersAndIntent(t *testing.T) {
	for _, runtimeID := range []string{"codex", "claude"} {
		for _, effort := range []runtimeprofile.Effort{{Mode: "runtime-default", Value: "default"}, {Mode: "explicit", Value: "new-safe-token"}} {
			t.Run(runtimeID+"/"+effort.Mode, func(t *testing.T) {
				r, err := DecodeSingleStageRun(mustRead(t, "testdata/axm36/single-run-v2.json"))
				if err != nil {
					t.Fatal(err)
				}
				resolution := &r.Binding.Resolution
				resolution.RuntimeID = runtimeID
				resolution.FrozenProfile.RuntimeID = runtimeID
				resolution.FrozenProfile.Model = "new-safe-model"
				resolution.FrozenProfile.Effort = effort
				if effort.Mode == "explicit" {
					r.Binding.Controls.ReasoningEffort = effort.Value
				}
				wire, err := EncodeSingleStageRun(r)
				if err != nil {
					t.Fatal("safe new intent rejected", err)
				}
				roundtrip, err := DecodeSingleStageRun(wire)
				if err != nil || !reflect.DeepEqual(roundtrip.Binding.Resolution, r.Binding.Resolution) {
					t.Fatal("intent lost", err)
				}
			})
		}
	}
}
