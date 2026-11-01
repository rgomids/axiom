package executiongraph

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

type invocationFake struct {
	invalid bool
	stdin   []byte
}

func (f invocationFake) ResolveInvocation(_ context.Context, child ChildExecution) (Invocation, error) {
	executable := "/usr/bin/true"
	if f.invalid {
		executable = "/bin/sh"
	}
	return Invocation{RuntimeID: child.Envelope.Resolution.RuntimeID, Argv: []string{executable, "literal;not-shell"}, CWD: child.Envelope.Workspace, Env: []string{"PATH=/usr/bin"}, OutputMax: 1024, Stdin: f.stdin}, nil
}

type attemptStoreFake struct {
	mu    sync.Mutex
	graph Graph
	saves []Graph
}

func (s *attemptStoreFake) Save(_ context.Context, graph Graph) (Graph, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.graph = graph
	s.saves = append(s.saves, graph)
	return graph, nil
}

func sequenceAllocatorFrom(start uint64) func() (string, error) {
	var next atomic.Uint64
	next.Store(start)
	return func() (string, error) {
		return fmt.Sprintf("00000000-0000-4000-8000-%012d", next.Add(1)), nil
	}
}

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

func TestInheritedSnapshotKeepsExplicitEnvironmentValidation(t *testing.T) {
	for i := 0; i < 70; i++ {
		t.Setenv(fmt.Sprintf("AXM_SNAPSHOT_%d", i), "synthetic")
	}
	t.Setenv("lowercase_inherited", "synthetic")
	child := mustGraph(t).Children[0]
	invocation, err := (invocationFake{}).ResolveInvocation(context.Background(), child)
	if err != nil {
		t.Fatal(err)
	}
	invocation.Env = []string{"CLAUDE_CODE_EFFORT_LEVEL=high"}
	invocation = WithInheritedEnvironment(invocation)
	if !validInvocation(child, invocation) {
		t.Fatal("ordinary inherited environment rejected")
	}
	t.Setenv("lowercase_inherited", "changed")
	if !strings.Contains(strings.Join(EffectiveInvocationEnvironment(invocation), "\n"), "lowercase_inherited=synthetic") {
		t.Fatal("environment snapshot changed")
	}
	for _, invalid := range [][]string{
		{"lowercase_explicit=synthetic"},
		{"CLAUDE_CODE_EFFORT_LEVEL=high", "CLAUDE_CODE_EFFORT_LEVEL=low"},
		{"lowercase_inherited=override"},
	} {
		invocation.Env = invalid
		if validInvocation(child, invocation) {
			t.Fatal("invalid explicit override accepted")
		}
	}
	invocation.Env = nil
	for i := 0; i < 65; i++ {
		invocation.Env = append(invocation.Env, fmt.Sprintf("EXPLICIT_%d=synthetic", i))
	}
	if validInvocation(child, invocation) {
		t.Fatal("explicit environment bound weakened")
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

func TestInvocationStdinBound(t *testing.T) {
	child := mustGraph(t).Children[0]
	invocation, _ := (invocationFake{}).ResolveInvocation(context.Background(), child)
	for _, size := range []int{0, MaxStdinBytes, MaxStdinBytes + 1} {
		invocation.Stdin = make([]byte, size)
		if validInvocation(child, invocation) != (size <= MaxStdinBytes) {
			t.Fatalf("size=%d", size)
		}
	}
}
