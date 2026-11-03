package executiongraph

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"testing"
)

func TestEffectiveEnvironmentMatchesProcessInheritance(t *testing.T) {
	t.Setenv("AXM_EFFECTIVE_ENV_PROBE", "inherited")
	inherited := strings.Join(EffectiveEnvironment(nil), "\n")
	if !strings.Contains(inherited, "AXM_EFFECTIVE_ENV_PROBE=inherited") {
		t.Fatal("empty invocation environment must report the inherited process environment")
	}
	explicit := EffectiveEnvironment([]string{"PATH=/usr/bin"})
	if len(explicit) != 1 || explicit[0] != "PATH=/usr/bin" {
		t.Fatalf("explicit environment=%v", explicit)
	}
}

func TestOSProcessRunnerUsesCapturedEnvironment(t *testing.T) {
	if os.Getenv("AXM_CAPTURED_ENV_HELPER") == "1" {
		if os.Getenv("AXM_CAPTURED_ENV_PROBE") != "captured" || os.Getenv("CLAUDE_CODE_EFFORT_LEVEL") != "high" {
			os.Exit(1)
		}
		os.Exit(0)
	}
	t.Setenv("AXM_CAPTURED_ENV_PROBE", "captured")
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	invocation := WithInheritedEnvironment(Invocation{Argv: []string{executable, "-test.run=^TestOSProcessRunnerUsesCapturedEnvironment$"}, CWD: t.TempDir(), Env: []string{"AXM_CAPTURED_ENV_HELPER=1", "CLAUDE_CODE_EFFORT_LEVEL=high"}, OutputMax: 1024})
	t.Setenv("AXM_CAPTURED_ENV_PROBE", "changed")
	result := (OSProcessRunner{}).Run(context.Background(), invocation)
	if result.ExitCode != 0 {
		t.Fatal("process runner did not use captured environment and override")
	}
}

func TestOSProcessRunnerStdinBytes(t *testing.T) {
	if os.Getenv("AXM_STDIN_HELPER") == "1" {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			os.Exit(2)
		}
		_, _ = os.Stdout.Write(data)
		os.Exit(0)
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	for _, stdin := range [][]byte{nil, []byte("packet\x00\r\nUnicode: é"), bytes.Repeat([]byte("x"), MaxStdinBytes), bytes.Repeat([]byte("x"), MaxStdinBytes+1)} {
		t.Run(fmt.Sprint(len(stdin)), func(t *testing.T) {
			result := (OSProcessRunner{}).Run(context.Background(), Invocation{Argv: []string{executable, "-test.run=^TestOSProcessRunnerStdinBytes$"}, CWD: t.TempDir(), Env: append(os.Environ(), "AXM_STDIN_HELPER=1"), Stdin: stdin, OutputMax: MaxCapturedOutputBytes})
			if len(stdin) > MaxStdinBytes {
				if result.ExitCode != -1 {
					t.Fatal("oversize accepted")
				}
				return
			}
			if result.ExitCode != 0 || !bytes.Equal(result.Output, stdin) {
				t.Fatalf("exit=%d output length=%d", result.ExitCode, len(result.Output))
			}
		})
	}
}
