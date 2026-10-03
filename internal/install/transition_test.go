package install

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rgomids/axiom/internal/compatibility"
	"github.com/rgomids/axiom/internal/detailartifact"
	"github.com/rgomids/axiom/internal/local"
	"github.com/rgomids/axiom/internal/provenance"
)

const (
	pocFixture   = "../compatibility/testdata/poc-v0.1.0-poc.1"
	pocProjectID = "46f9e9bf-9da2-4769-b9b7-f058f0ab6e90"
)

// withPOCState reproduces the historical POC state and portable trees in the
// installation's state roots with the private modes the POC wrote.
func (i installation) withPOCState(t *testing.T) {
	t.Helper()
	for tree, target := range map[string]string{"state": i.target.State.State, "projects": i.target.State.Projects} {
		source := filepath.Join(pocFixture, tree)
		err := filepath.WalkDir(source, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			relative, _ := filepath.Rel(source, path)
			destination := filepath.Join(target, relative)
			if entry.IsDir() {
				return os.Mkdir(destination, 0o700)
			}
			wire, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			return os.WriteFile(destination, wire, 0o600)
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}

// withV1State is the POC tree without its workflow history: v1-readable
// installation, work-item and portable records.
func (i installation) withV1State(t *testing.T) {
	t.Helper()
	i.withPOCState(t)
	if err := os.RemoveAll(filepath.Join(i.target.State.State, "workflows")); err != nil {
		t.Fatal(err)
	}
}

func (i installation) workItem() string {
	return filepath.Join(i.target.State.State, "work-items", pocProjectID, "main-7.json")
}

func createV1Artifact(t *testing.T, state string) {
	t.Helper()
	store, err := local.NewArtifactStore(state)
	if err != nil {
		t.Fatal(err)
	}
	source, err := provenance.FromBuild(provenance.Build{Version: provenance.Development, Revision: "123456789abc", SourceState: provenance.Clean}, nil)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if _, err := store.Create(ctx, detailartifact.Draft{CorrelationID: "123e4567-e89b-42d3-a456-426614174000", Category: "diagnostic", Outcome: "failure", Retention: detailartifact.Diagnostic, Markdown: []byte("# Detail\n"), Provenance: source}); err != nil {
		t.Fatal(err)
	}
}

func replaceIn(t *testing.T, path, old, new string) {
	t.Helper()
	wire := read(t, path)
	if !strings.Contains(wire, old) {
		t.Fatalf("%s does not contain %q", path, old)
	}
	writeFile(t, path, []byte(strings.Replace(wire, old, new, 1)), 0o600)
}

// The installer consumes the centralized compatibility policy: every observed
// persisted state resolves to exactly one strategy, and every strategy this
// release cannot execute stops before any effect and without authority.
func TestUpgradeResolvesForwardTransitionPolicy(t *testing.T) {
	tests := []struct {
		name           string
		version        string
		state          func(installation, *testing.T)
		classification compatibility.Classification
		strategy       compatibility.Strategy
		want           string
	}{
		{"absent state newer release", "1.1.0", func(installation, *testing.T) {}, compatibility.AbsentV1, compatibility.StrategyDirect, ""},
		{"current v1 newer release", "1.1.0", installation.withV1State, compatibility.ValidV1, compatibility.StrategyDirect, ""},
		{"recognized POC", "1.1.0", installation.withPOCState, compatibility.RecognizedPOC, compatibility.StrategyPreserveRebuildReconfigure, "state_transition_unavailable"},
		{"recognized POC same release", "1.0.0", installation.withPOCState, compatibility.RecognizedPOC, compatibility.StrategyPreserveRebuildReconfigure, "state_transition_unavailable"},
		{"foreign entry", "1.1.0", func(i installation, t *testing.T) {
			i.withV1State(t)
			writeFile(t, filepath.Join(i.target.State.State, "notes.txt"), []byte("foreign\n"), 0o600)
		}, compatibility.Malformed, compatibility.StrategyRefuse, "state_unsafe"},
		{"modified POC workflow", "1.1.0", func(i installation, t *testing.T) {
			i.withPOCState(t)
			replaceIn(t, filepath.Join(i.target.State.State, "workflows", pocProjectID, "main-7.json"), `"gate":"plan"`, `"gate":"planning"`)
		}, compatibility.Malformed, compatibility.StrategyRefuse, "state_unsafe"},
		{"unsafe symlink entry", "1.1.0", func(i installation, t *testing.T) {
			i.withV1State(t)
			if err := os.Symlink(t.TempDir(), filepath.Join(i.target.State.State, "work-items", pocProjectID, "link.json")); err != nil {
				t.Fatal(err)
			}
		}, compatibility.Malformed, compatibility.StrategyRefuse, "state_unsafe"},
		{"unsafe permissive mode", "1.1.0", func(i installation, t *testing.T) {
			i.withV1State(t)
			if err := os.Chmod(i.workItem(), 0o644); err != nil {
				t.Fatal(err)
			}
		}, compatibility.Malformed, compatibility.StrategyRefuse, "state_unsafe"},
		{"ambiguous mixed POC and v1", "1.1.0", func(i installation, t *testing.T) {
			i.withPOCState(t)
			createV1Artifact(t, i.target.State.State)
		}, compatibility.Malformed, compatibility.StrategyRefuse, "state_unsafe"},
		{"corrupt record", "1.1.0", func(i installation, t *testing.T) {
			i.withV1State(t)
			writeFile(t, i.workItem(), []byte("{\"formatVersion\":1,"), 0o600)
		}, compatibility.Malformed, compatibility.StrategyRefuse, "state_unsafe"},
		{"unsupported newer format", "1.1.0", func(i installation, t *testing.T) {
			i.withV1State(t)
			replaceIn(t, i.workItem(), `"formatVersion":1`, `"formatVersion":2`)
		}, compatibility.UnsupportedNewer, compatibility.StrategyRefuse, "state_unsupported"},
		{"older format outside window", "1.1.0", func(i installation, t *testing.T) {
			i.withV1State(t)
			replaceIn(t, filepath.Join(i.target.State.State, "projects", pocProjectID, "installation.json"), `"formatVersion":1`, `"formatVersion":0`)
		}, compatibility.UnsupportedOlder, compatibility.StrategyRefuse, "state_unsupported"},
		{"interrupted protocol state", "1.1.0", func(i installation, t *testing.T) {
			i.withV1State(t)
			writeFile(t, filepath.Join(i.target.State.State, "work-items", pocProjectID, ".axiom-recovery-0001"), []byte("interrupted\n"), 0o600)
		}, compatibility.RecoveryRequired, compatibility.StrategyRefuse, "state_recovery_required"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			installed := install(t, newBundle("1.0.0", []byte("old-binary\n")))
			test.state(installed, t)
			binary := []byte("new-binary\n")
			if test.version == "1.0.0" {
				binary = installed.current.binary
			}
			candidate := installed.candidate(t, newBundle(test.version, binary))
			before := snapshot(t, filepath.Dir(installed.target.BinaryDir))
			service := NewService()
			preview, err := service.Preview(context.Background(), installed.target, candidate)
			if category(err) != test.want {
				t.Fatalf("error=%v want %q", err, test.want)
			}
			decision := preview.Transition
			if decision.Strategy != test.strategy || decision.Classification != test.classification || preview.State != test.classification || decision.StateDigest == "" || decision.StateDigest != preview.StateDigest {
				t.Fatalf("decision=%+v state=%s want %s/%s", decision, preview.State, test.strategy, test.classification)
			}
			if test.want != "" {
				if preview.Digest != "" || len(preview.Effects) != 0 {
					t.Fatalf("refused transition produced authorizable preview: %+v", preview)
				}
				if _, err := Authorize(preview, preview.Digest); category(err) != "authority_denied" {
					t.Fatalf("refused transition was authorizable: %v", err)
				}
				if after := snapshot(t, filepath.Dir(installed.target.BinaryDir)); after != before {
					t.Fatal("refused transition changed installation or state")
				}
				return
			}
			authority, err := Authorize(preview, preview.Digest)
			if err != nil {
				t.Fatal(err)
			}
			result, err := service.Apply(context.Background(), preview, authority)
			if err != nil || result.Status != "success" {
				t.Fatalf("result=%+v err=%v", result, err)
			}
			if read(t, filepath.Join(installed.target.BinaryDir, binaryName)) != string(binary) {
				t.Fatal("direct strategy did not upgrade the binary")
			}
		})
	}
}

func TestUpgradeSameReleaseWithCurrentStateIsNoOp(t *testing.T) {
	installed := install(t, newBundle("1.0.0", []byte("old-binary\n")))
	installed.withV1State(t)
	before := snapshot(t, filepath.Dir(installed.target.BinaryDir))
	preview, err := NewService().Preview(context.Background(), installed.target, installed.candidate(t, installed.current))
	if err != nil || len(preview.Effects) != 0 || preview.Resume || preview.Transition.Strategy != compatibility.StrategyDirect {
		t.Fatalf("preview=%+v err=%v", preview, err)
	}
	if _, err := Authorize(preview, preview.Digest); category(err) != "authority_denied" {
		t.Fatalf("no-op preview granted authority: %v", err)
	}
	if after := snapshot(t, filepath.Dir(installed.target.BinaryDir)); after != before {
		t.Fatal("same-release preview changed state")
	}
}

func TestUpgradeDowngradeWithCurrentStateHasZeroEffects(t *testing.T) {
	installed := install(t, newBundle("1.1.0", []byte("current-binary\n")))
	installed.withV1State(t)
	before := snapshot(t, filepath.Dir(installed.target.BinaryDir))
	if _, err := NewService().Preview(context.Background(), installed.target, installed.candidate(t, newBundle("1.0.0", []byte("older-binary\n")))); category(err) != "downgrade_refused" {
		t.Fatalf("error=%v want downgrade_refused", err)
	}
	if after := snapshot(t, filepath.Dir(installed.target.BinaryDir)); after != before {
		t.Fatal("refused downgrade changed state")
	}
}

// The authorized digest binds the transition decision to the exact inspected
// state: any later change of that state denies the old authority.
func TestUpgradeTransitionAuthorityIsStaleWhenStateChanges(t *testing.T) {
	for _, test := range []struct {
		name   string
		change func(*testing.T, installation)
	}{
		{"current v1 content changes", func(t *testing.T, i installation) {
			if err := os.RemoveAll(i.target.State.Projects); err != nil {
				t.Fatal(err)
			}
		}},
		{"state becomes recognized POC", func(t *testing.T, i installation) { i.addPOCWorkflow(t) }},
		{"state becomes unsupported newer", func(t *testing.T, i installation) {
			replaceIn(t, i.workItem(), `"formatVersion":1`, `"formatVersion":2`)
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			installed := install(t, newBundle("1.0.0", []byte("old-binary\n")))
			installed.withV1State(t)
			service := NewService()
			preview, err := service.Preview(context.Background(), installed.target, installed.candidate(t, newBundle("1.1.0", []byte("new-binary\n"))))
			if err != nil || preview.Transition.Strategy != compatibility.StrategyDirect {
				t.Fatalf("preview=%+v err=%v", preview, err)
			}
			authority, err := Authorize(preview, preview.Digest)
			if err != nil {
				t.Fatal(err)
			}
			test.change(t, installed)
			before := snapshot(t, filepath.Dir(installed.target.BinaryDir))
			if result, err := service.Apply(context.Background(), preview, authority); category(err) != "authority_denied" || len(result.Ledger) != 0 {
				t.Fatalf("stale state authority result=%+v err=%v", result, err)
			}
			if after := snapshot(t, filepath.Dir(installed.target.BinaryDir)); after != before {
				t.Fatal("stale state authority changed installation or state")
			}
		})
	}
}

func TestUpgradeTransitionAuthorityIsStaleWhenTargetReleaseChanges(t *testing.T) {
	installed := install(t, newBundle("1.0.0", []byte("old-binary\n")))
	installed.withV1State(t)
	service := NewService()
	reviewed, err := service.Preview(context.Background(), installed.target, installed.candidate(t, newBundle("1.1.0", []byte("reviewed-binary\n"))))
	if err != nil {
		t.Fatal(err)
	}
	authority, err := Authorize(reviewed, reviewed.Digest)
	if err != nil {
		t.Fatal(err)
	}
	for _, retargeted := range []*bundle{newBundle("1.2.0", []byte("other-binary\n")), newBundle("1.1.0", []byte("rebuilt-binary\n"))} {
		changed, err := service.Preview(context.Background(), installed.target, installed.candidate(t, retargeted))
		if err != nil || changed.Transition != reviewed.Transition || changed.Digest == reviewed.Digest {
			t.Fatalf("changed target preview=%+v err=%v", changed, err)
		}
		before := snapshot(t, filepath.Dir(installed.target.BinaryDir))
		if _, err := Authorize(changed, reviewed.Digest); category(err) != "authority_denied" {
			t.Fatalf("reviewed digest authorized a different target release: %v", err)
		}
		if result, err := service.Apply(context.Background(), changed, authority); category(err) != "authority_denied" || len(result.Ledger) != 0 {
			t.Fatalf("stale target authority result=%+v err=%v", result, err)
		}
		if after := snapshot(t, filepath.Dir(installed.target.BinaryDir)); after != before {
			t.Fatal("stale target authority changed installation or state")
		}
	}
}

func (i installation) addPOCWorkflow(t *testing.T) {
	t.Helper()
	workflows := privateDirectory(t, privateDirectory(t, i.target.State.State, "workflows"), pocProjectID)
	wire, err := os.ReadFile(filepath.Join(pocFixture, "state", "workflows", pocProjectID, "main-7.json"))
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(workflows, "main-7.json"), wire, 0o600)
}

// A resumable interrupted upgrade still passes through the policy: state that
// no longer resolves to direct blocks the resume without further effects and
// keeps the operation marker for recovery.
func TestUpgradeResumeRefusesNonDirectStateWithoutEffects(t *testing.T) {
	installed := install(t, newBundle("1.0.0", []byte("old-binary\n")))
	installed.withV1State(t)
	candidate := installed.candidate(t, newBundle("1.1.0", []byte("new-binary\n")))
	service := NewService()
	preview, err := service.Preview(context.Background(), installed.target, candidate)
	if err != nil {
		t.Fatal(err)
	}
	authority, _ := Authorize(preview, preview.Digest)
	service.afterEffect = func(kind string) error {
		if kind == "binary" {
			return errors.New("injected interruption")
		}
		return nil
	}
	if result, err := service.Apply(context.Background(), preview, authority); err == nil || result.Status != "partial" {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	installed.addPOCWorkflow(t)
	before := snapshot(t, filepath.Dir(installed.target.BinaryDir))
	resume, err := NewService().Preview(context.Background(), installed.target, candidate)
	if category(err) != "state_transition_unavailable" || !resume.Resume || resume.Digest != "" || resume.Transition.Strategy != compatibility.StrategyPreserveRebuildReconfigure {
		t.Fatalf("resume=%+v err=%v", resume, err)
	}
	if after := snapshot(t, filepath.Dir(installed.target.BinaryDir)); after != before {
		t.Fatal("refused resume changed installation or state")
	}
	if !strings.Contains(read(t, filepath.Join(installed.target.ReceiptDir, markerName)), "stage=binary_committed\n") {
		t.Fatal("refused resume lost the operation marker")
	}
}

// Resulting-state validation uses the same policy: state that stops resolving
// to direct while effects publish is never reported as success.
func TestUpgradeFinalStateMustStillResolveDirect(t *testing.T) {
	installed := install(t, newBundle("1.0.0", []byte("old-binary\n")))
	installed.withV1State(t)
	service := NewService()
	preview, err := service.Preview(context.Background(), installed.target, installed.candidate(t, newBundle("1.1.0", []byte("new-binary\n"))))
	if err != nil {
		t.Fatal(err)
	}
	authority, _ := Authorize(preview, preview.Digest)
	service.afterEffect = func(kind string) error {
		if kind == "receipt" {
			installed.addPOCWorkflow(t)
		}
		return nil
	}
	result, err := service.Apply(context.Background(), preview, authority)
	if category(err) != "final_verification_failed" || result.Status != "partial" || len(result.Ledger) != 2 {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}

// A failed state inspection during resume keeps the resume fact for the
// caller, leaves the operation marker intact and plans no new effect.
func TestUpgradeResumeKeepsResumeWhenStateInspectionFails(t *testing.T) {
	installed := install(t, newBundle("1.0.0", []byte("old-binary\n")))
	installed.withV1State(t)
	candidate := installed.candidate(t, newBundle("1.1.0", []byte("new-binary\n")))
	service := NewService()
	preview, err := service.Preview(context.Background(), installed.target, candidate)
	if err != nil {
		t.Fatal(err)
	}
	authority, _ := Authorize(preview, preview.Digest)
	service.afterEffect = func(kind string) error {
		if kind == "binary" {
			return errors.New("injected interruption")
		}
		return nil
	}
	if result, err := service.Apply(context.Background(), preview, authority); err == nil || result.Status != "partial" {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	marker := filepath.Join(installed.target.ReceiptDir, markerName)
	markerBefore := read(t, marker)
	before := snapshot(t, filepath.Dir(installed.target.BinaryDir))
	uninspectable := installed.target
	uninspectable.State.State = "relative-state-root"
	resume, err := NewService().Preview(context.Background(), uninspectable, candidate)
	if category(err) != "state_inspection_failed" || !resume.Resume || resume.Digest != "" || len(resume.Effects) != 0 {
		t.Fatalf("resume=%+v err=%v", resume, err)
	}
	if _, err := Authorize(resume, resume.Digest); category(err) != "authority_denied" {
		t.Fatalf("failed inspection was authorizable: %v", err)
	}
	if read(t, marker) != markerBefore || !strings.Contains(markerBefore, "stage=binary_committed\n") {
		t.Fatal("failed inspection changed the operation marker")
	}
	if after := snapshot(t, filepath.Dir(installed.target.BinaryDir)); after != before {
		t.Fatal("failed inspection changed installation or state")
	}
}
