package compatibility

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/local"
	"github.com/rgomids/axiom/internal/runtimeprofile"
	"github.com/rgomids/axiom/internal/runtimeprofile/runtimeprofiletest"
)

// runtimeProfileRoot is a State root holding only one local Runtime/Profile
// object, written as an earlier release or the current writer left it.
func runtimeProfileRoot(t *testing.T, name string, wire []byte) Roots {
	t.Helper()
	base := privateDir(t, t.TempDir(), "roots")
	state := privateDir(t, base, "state")
	version := privateDir(t, privateDir(t, state, "runtime-profiles"), "v1")
	writePrivate(t, filepath.Join(version, name), wire)
	return Roots{State: state}
}

func legacyRuntimeProfile(t *testing.T, format int) []byte {
	t.Helper()
	cfg := runtimeprofile.Configuration{FormatVersion: format, Revision: 1,
		Runtimes:      []runtimeprofile.Runtime{{ID: "codex", Adapter: "codex", Enabled: true, AllowlistedProfileIDs: []string{"codex-default"}}},
		ModelProfiles: []runtimeprofile.ModelProfile{{ID: "codex-default", RuntimeID: "codex", Model: "local-codex", Capabilities: []string{"go"}}},
	}
	if format == runtimeprofile.FormatVersionV2 {
		cfg.ModelProfiles[0].Effort = runtimeprofile.Effort{Mode: "runtime-default", Value: "default"}
	}
	wire, err := runtimeprofile.Encode(cfg)
	if err != nil {
		t.Fatal(err)
	}
	return wire
}

// Specification 007 PM-11 (B1): application upgrade eligibility is separate
// from Runtime/Profile operational admission. Recognized pre-stable records
// pass the upgrade gate with explicit reconfiguration guidance and unchanged
// bytes; malformed, future and interrupted state keep their strict refusals.
func TestRuntimeProfileUpgradeClassification(t *testing.T) {
	v3 := runtimeprofiletest.Encode(t, runtimeprofiletest.CodexOnly())
	tomb, err := runtimeprofile.EncodeTombstone(runtimeprofile.Tombstone{Kind: runtimeprofile.TombstoneKind, FormatVersion: 1, Revision: 2, PriorConfigurationDigest: runtimeprofiletest.Digest(t, runtimeprofiletest.CodexOnly())})
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, object string
		wire         []byte
		kind         local.InventoryKind
		class        Classification
		strategy     Strategy
		outcome      Outcome
		reconfigure  bool
	}{
		{"valid v1", "configuration.json", legacyRuntimeProfile(t, 1), local.InventoryRuntimeProfilePreStable, ValidV1, StrategyDirect, OutcomeCompatible, true},
		{"valid v2", "configuration.json", legacyRuntimeProfile(t, 2), local.InventoryRuntimeProfilePreStable, ValidV1, StrategyDirect, OutcomeCompatible, true},
		{"active v3", "configuration.json", v3, local.InventoryRuntimeProfile, ValidV1, StrategyDirect, OutcomeCompatible, false},
		{"retired", "configuration.json", tomb, local.InventoryRuntimeProfile, ValidV1, StrategyDirect, OutcomeCompatible, false},
		{"malformed v1", "configuration.json", []byte(`{"formatVersion":1,"revision":1}`), local.InventoryMalformed, Malformed, StrategyRefuse, OutcomeUnsafe, false},
		{"malformed v2", "configuration.json", []byte(`{"formatVersion":2,"revision":1,"runtimes":[]}`), local.InventoryMalformed, Malformed, StrategyRefuse, OutcomeUnsafe, false},
		{"malformed v3", "configuration.json", bytes.Replace(v3, []byte(`"revision":1`), []byte(`"revision":0`), 1), local.InventoryMalformed, Malformed, StrategyRefuse, OutcomeUnsafe, false},
		{"future", "configuration.json", bytes.Replace(v3, []byte(`"formatVersion":3`), []byte(`"formatVersion":99`), 1), local.InventoryNewer, UnsupportedNewer, StrategyRefuse, OutcomeUnsupported, false},
		{"pending", ".axiom-recovery-configuration.json", v3, local.InventoryRecovery, RecoveryRequired, StrategyRefuse, OutcomeRecoveryRequired, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			roots := runtimeProfileRoot(t, tc.object, tc.wire)
			report := inspect(t, roots)
			if len(report.objects) != 1 || report.objects[0].Kind != tc.kind {
				t.Fatalf("objects %+v", report.objects)
			}
			decision := Resolve(Owned, report)
			if report.Classification != tc.class || decision.Strategy != tc.strategy || decision.Outcome != tc.outcome {
				t.Fatalf("classification %s decision %+v", report.Classification, decision)
			}
			if got := strings.Contains(strings.Join(report.Next, "\n"), "binding remove --runtime <id> --cascade"); got != tc.reconfigure {
				t.Fatalf("reconfiguration guidance %v: %v", got, report.Next)
			}
			after, err := os.ReadFile(filepath.Join(roots.State, "runtime-profiles", "v1", tc.object))
			if err != nil || !bytes.Equal(after, tc.wire) {
				t.Fatal("inspection changed the source bytes")
			}
		})
	}
}
