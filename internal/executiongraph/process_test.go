package executiongraph

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

type fakeProcessContainment struct {
	alive, escaped      bool
	stopErr, observeErr error
	descendants         int
}

func (*fakeProcessContainment) start(*exec.Cmd) error { return nil }
func (g *fakeProcessContainment) stop() error {
	if g.stopErr != nil {
		return g.stopErr
	}
	g.alive = false
	g.descendants = 0
	return nil
}
func (g *fakeProcessContainment) gone() (bool, error) {
	return !g.alive && g.descendants == 0 && !g.escaped, g.observeErr
}
func TestConfirmedProcessGroupStop(t *testing.T) {
	for _, tc := range []struct {
		name  string
		group fakeProcessContainment
		want  bool
	}{
		{"descendants", fakeProcessContainment{alive: true, descendants: 3}, true},
		{"escaped descendant", fakeProcessContainment{alive: true, descendants: 3, escaped: true}, false},
		{"already gone", fakeProcessContainment{}, true},
		{"permission denied", fakeProcessContainment{alive: true, stopErr: errors.New("permission denied")}, false},
		{"observation denied", fakeProcessContainment{observeErr: errors.New("permission denied")}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			if got := confirmProcessStop(ctx, &tc.group); got != tc.want {
				t.Fatalf("stopped=%v want %v", got, tc.want)
			}
		})
	}
}

func TestOSProcessRunnerConfirmsFailureBeforeSpawn(t *testing.T) {
	root := t.TempDir()
	denied := filepath.Join(root, "denied")
	if err := os.WriteFile(denied, []byte("#!/bin/sh\nexit 0\n"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ path, code string }{{denied, "permission_denied"}, {filepath.Join(root, "missing"), "executable_unavailable"}} {
		calls := 0
		p := (OSProcessRunner{}).RunObserved(context.Background(), Invocation{Argv: []string{tc.path}, CWD: root, OutputMax: 1024}, func(*ProcessCorrelation) error { calls++; return nil })
		if calls != 0 || !p.SpawnFailed || p.FailureCode != tc.code || p.OutcomeUnknown || p.ExitCode == 0 {
			t.Fatalf("%+v calls=%d", p, calls)
		}
	}
}

func TestOSProcessRunnerHoldsPacketUntilAttemptPublication(t *testing.T) {
	if os.Getenv("AXM_STDIN_PUBLICATION_HELPER") == "1" {
		payload, _ := io.ReadAll(os.Stdin)
		if len(payload) > 0 {
			if err := os.WriteFile(os.Getenv("AXM_STDIN_MARKER"), payload, 0600); err != nil {
				os.Exit(2)
			}
		}
		os.Exit(0)
	}
	for _, refuse := range []bool{false, true} {
		marker := filepath.Join(t.TempDir(), "delivered")
		executable, err := os.Executable()
		if err != nil {
			t.Fatal(err)
		}
		inv := WithExactEnvironment(Invocation{Argv: []string{executable, "-test.run=^TestOSProcessRunnerHoldsPacketUntilAttemptPublication$"}, CWD: filepath.Dir(marker), Stdin: []byte("authorized-packet"), OutputMax: 1024}, []string{"AXM_STDIN_PUBLICATION_HELPER=1", "AXM_STDIN_MARKER=" + marker})
		calls := 0
		result := (OSProcessRunner{}).RunObserved(context.Background(), inv, func(*ProcessCorrelation) error {
			calls++
			time.Sleep(50 * time.Millisecond)
			if _, err := os.Stat(marker); !os.IsNotExist(err) {
				t.Error("packet delivered before durable attempt publication")
			}
			if refuse {
				return errors.New("synthetic publication refusal")
			}
			return nil
		})
		payload, err := os.ReadFile(marker)
		if calls != 1 {
			t.Fatalf("callback count %d", calls)
		}
		if refuse {
			if !os.IsNotExist(err) || !result.OutcomeUnknown {
				t.Fatal("failed callback delivered packet or erased uncertainty")
			}
			continue
		}
		if err != nil || string(payload) != "authorized-packet" || result.ExitCode != 0 {
			t.Fatalf("result=%+v err=%v", result, err)
		}
	}
}

func TestOSProcessRunnerRetainsFastRejectionBeforeCorrelation(t *testing.T) {
	root := t.TempDir()
	executable := filepath.Join(root, "reject-option")
	if err := os.WriteFile(executable, []byte("#!/bin/sh\nprintf \"error: unknown option '--synthetic'\\n\" >&2\nexit 7\n"), 0700); err != nil {
		t.Fatal(err)
	}
	inv := WithExactEnvironment(Invocation{Argv: []string{executable}, CWD: root, Stdin: []byte("packet"), OutputMax: 1024}, nil)
	result := (OSProcessRunner{}).RunObserved(context.Background(), inv, func(*ProcessCorrelation) error {
		time.Sleep(100 * time.Millisecond)
		return errors.New("synthetic publication failure")
	})
	if result.ExitCode != 7 || !bytes.Contains(result.Output, []byte("error: unknown option")) {
		t.Fatalf("lost actual process result: %+v", result)
	}
	a := SuccessorAttempt{AttemptID: "00000000-0000-4000-8000-000000000001", Number: 1, Status: AttemptUnknown, ClassificationReason: ProcessOutcomeUnknown}
	d := DiagnoseAttempt(a, result)
	if d.EffectCertainty != "unknown" || d.Cause != "invocation_incompatible" || d.CauseConfidence != "hypothesis" {
		t.Fatalf("%+v", d)
	}
}
