package executiongraph

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/rgomids/axiom/internal/runtimeprofile"
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
				g, e := DecodeOperationalGraph(wire)
				if e != nil || g.StorageRevision != sha256.Sum256(wire) {
					t.Fatalf("decode graph: %v", e)
				}
				encoded, err = EncodeOperationalGraph(g)
				if _, e := DecodeSuccessorGraph(wire); e == nil {
					t.Fatal("new graph admitted by historical decoder")
				}
			} else {
				r, e := DecodeOperationalSingleStageRun(wire)
				if e != nil || r.StorageRevision != sha256.Sum256(wire) {
					t.Fatalf("decode single: %v", e)
				}
				encoded, err = EncodeOperationalSingleStageRun(r)
				if _, e := DecodeSingleStageRun(wire); e == nil {
					t.Fatal("new run admitted by historical decoder")
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
	if _, err := DecodeOperationalGraph(mustRead(t, "testdata/successor/valid/publication.json")); err == nil {
		t.Fatal("historical graph accepted as current")
	}
	if _, err := DecodeOperationalSingleStageRun(mustRead(t, "testdata/single-stage/run-v1.json")); err == nil {
		t.Fatal("historical run accepted as current")
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
					_, err := DecodeOperationalGraph(b)
					return err
				}
				_, err := DecodeOperationalSingleStageRun(b)
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

func TestOperationalResolutionAndControls(t *testing.T) {
	g, err := DecodeOperationalGraph(mustRead(t, "testdata/axm36/graph-v4.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range g.Children {
		r := c.Envelope.Resolution
		for name, mutate := range map[string]func(*OperationalResolution){
			"runtime":       func(r *OperationalResolution) { r.RuntimeID = "other" },
			"profile":       func(r *OperationalResolution) { r.ModelProfileID = "other" },
			"configuration": func(r *OperationalResolution) { r.ConfigurationRevision++ },
			"observation":   func(r *OperationalResolution) { r.ObservationRevision++ },
			"digest":        func(r *OperationalResolution) { r.ResolvedProfile.ExecutableDigest = "bad" },
		} {
			t.Run(r.RuntimeID+"/"+name, func(t *testing.T) {
				bad := r
				mutate(&bad)
				if ValidateOperationalResolution(bad) == nil {
					t.Fatal("mismatched tuple accepted")
				}
			})
		}
		controls := c.Envelope.Controls
		controls.ReasoningEffort = "other"
		if ValidOperationalProfileControls(r.ResolvedProfile, controls, c.Envelope.Capability) {
			t.Fatal("competing controls accepted")
		}
	}
}

func TestOperationalGraphRetainsSafetyChecks(t *testing.T) {
	wire := mustRead(t, "testdata/axm36/graph-v4.json")
	for name, mutate := range map[string]func(*OperationalGraph){
		"authority digest": func(g *OperationalGraph) { g.Parent.AuthorityDigest = strings.Repeat("f", 64) },
		"effect ceiling": func(g *OperationalGraph) {
			g.Children[0].Envelope.AllowedEffects = []Effect{{Kind: "repository-write", Target: "src"}}
		},
		"cycle":           func(g *OperationalGraph) { g.Children[0].Envelope.Dependencies = []string{g.Children[1].ExecutionID} },
		"duplicate child": func(g *OperationalGraph) { g.Children[0].ExecutionID = g.Children[1].ExecutionID },
		"profile drift":   func(g *OperationalGraph) { g.Children[0].Envelope.Resolution.ResolvedProfile.ProfileID = "other" },
		"concurrency":     func(g *OperationalGraph) { g.StageConcurrency = 0 },
		"integration":     func(g *OperationalGraph) { g.Parent.IntegrationChild = g.Children[0].ExecutionID },
		"cancel authority": func(g *OperationalGraph) {
			g.CancelRequest = &CancelRequest{RequestID: "00000000-0000-4000-8000-000000000099", ParentID: g.Parent.ExecutionID, GraphRevision: g.Parent.GraphRevision, ActorKind: "agent", Actor: "synthetic", RequestedAt: g.Parent.CreatedAt, AuthorityRef: g.DispatchAuthorityRef}
		},
	} {
		t.Run(name, func(t *testing.T) {
			g, err := DecodeOperationalGraph(wire)
			if err != nil {
				t.Fatal(err)
			}
			mutate(&g)
			if _, err := EncodeOperationalGraph(g); err == nil {
				t.Fatal("unsafe graph accepted")
			}
		})
	}
}

func TestOperationalSingleRetainsSafetyChecks(t *testing.T) {
	wire := mustRead(t, "testdata/axm36/single-run-v2.json")
	for name, mutate := range map[string]func(*OperationalSingleStageRun){
		"epoch":               func(r *OperationalSingleStageRun) { r.RunEpoch = 0 },
		"authority":           func(r *OperationalSingleStageRun) { r.Authorities = nil },
		"duplicate authority": func(r *OperationalSingleStageRun) { r.Authorities = append(r.Authorities, r.Authorities[0]) },
		"scope":               func(r *OperationalSingleStageRun) { r.Scope.StageID = "other" },
		"paths":               func(r *OperationalSingleStageRun) { r.Binding.Inbox = r.Binding.Workspace },
		"attempt limit":       func(r *OperationalSingleStageRun) { r.Binding.Controls.MaximumAttempts = 0 },
		"timeout":             func(r *OperationalSingleStageRun) { r.Binding.Controls.Timeout = 25 * time.Hour },
		"effort":              func(r *OperationalSingleStageRun) { r.Binding.Controls.ReasoningEffort = "high" },
		"historical version":  func(r *OperationalSingleStageRun) { r.FormatVersion = 1 },
	} {
		t.Run(name, func(t *testing.T) {
			r, err := DecodeOperationalSingleStageRun(wire)
			if err != nil {
				t.Fatal(err)
			}
			mutate(&r)
			if _, err := EncodeOperationalSingleStageRun(r); err == nil {
				t.Fatal("unsafe run accepted")
			}
		})
	}
}

func operationalGraphVector(t *testing.T, name string) OperationalGraph {
	t.Helper()
	g, err := DecodeOperationalGraph(mustRead(t, "testdata/axm36/graph-v4-"+name+".json"))
	if err != nil {
		t.Fatal(err)
	}
	return g
}

func TestOperationalGraphTerminalPublicationAndReconciliation(t *testing.T) {
	running := operationalGraphVector(t, "running")
	finished := operationalGraphVector(t, "process_failed")
	if ValidateOperationalGraphUpdate(running, finished) != nil {
		t.Fatal("atomic terminal state and diagnostic publication refused")
	}
	old := operationalGraphVector(t, "process_outcome_unknown")
	next := operationalGraphVector(t, "reconciliation")
	if ValidateOperationalGraphUpdate(old, next) != nil {
		t.Fatal("append-only reconciliation refused")
	}
	for name, mutate := range map[string]func(*OperationalGraph){
		"remove diagnostic": func(g *OperationalGraph) { g.Children[0].Attempts[0].Diagnostic = nil },
		"rewrite diagnostic": func(g *OperationalGraph) {
			g.Children[0].Attempts[0].Diagnostic.Cause = "authentication_unavailable"
			g.Children[0].Attempts[0].Diagnostic.CauseConfidence = "confirmed"
		},
		"rewrite fact": func(g *OperationalGraph) {
			g.Children[0].Attempts[0].ReconciliationFacts[0].ObservationsDigest = strings.Repeat("f", 64)
		},
		"remove fact": func(g *OperationalGraph) { g.Children[0].Attempts[0].ReconciliationFacts = nil },
		"rewrite binding": func(g *OperationalGraph) {
			g.Children[0].Envelope.Resolution.ResolvedProfile.ExecutableDigest = strings.Repeat("f", 64)
		},
		"rewrite exit": func(g *OperationalGraph) { g.Children[0].Attempts[0].ExitCode++ },
	} {
		t.Run(name, func(t *testing.T) {
			bad := operationalGraphVector(t, "reconciliation")
			mutate(&bad)
			if ValidateOperationalGraphUpdate(next, bad) == nil {
				t.Fatal("immutable publication rewritten")
			}
		})
	}
	withoutDiagnostic := operationalGraphVector(t, "process_failed")
	withoutDiagnostic.Children[0].Attempts[0].Diagnostic = nil
	if ValidateOperationalGraphUpdate(withoutDiagnostic, finished) == nil {
		t.Fatal("historical cause backfilled")
	}
}

func TestOperationalSingleBothAdaptersAndIntent(t *testing.T) {
	for _, runtimeID := range []string{"codex", "claude"} {
		for _, effort := range []runtimeprofile.Effort{{Mode: "runtime-default", Value: "default"}, {Mode: "explicit", Value: "new-safe-token"}} {
			t.Run(runtimeID+"/"+effort.Mode, func(t *testing.T) {
				r, err := DecodeOperationalSingleStageRun(mustRead(t, "testdata/axm36/single-run-v2.json"))
				if err != nil {
					t.Fatal(err)
				}
				resolution := &r.Binding.Resolution
				resolution.RuntimeID = runtimeID
				resolution.ResolvedProfile.RuntimeID = runtimeID
				resolution.ResolvedProfile.Model = "new-safe-model"
				resolution.ResolvedProfile.Effort = effort
				if effort.Mode == "explicit" {
					r.Binding.Controls.ReasoningEffort = effort.Value
				}
				wire, err := EncodeOperationalSingleStageRun(r)
				if err != nil {
					t.Fatal("safe new intent rejected", err)
				}
				roundtrip, err := DecodeOperationalSingleStageRun(wire)
				if err != nil || roundtrip.Binding.Resolution != r.Binding.Resolution {
					t.Fatal("intent lost", err)
				}
			})
		}
	}
}
