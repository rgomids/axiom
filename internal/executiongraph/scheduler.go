package executiongraph

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const MaxCapturedOutputBytes = 256 << 10

// MaxStdinBytes bounds the in-memory task packet.
const MaxStdinBytes = 256 << 10

type Invocation struct {
	RuntimeID string
	Argv      []string
	CWD       string
	Env       []string
	OutputMax int
	Stdin     []byte
	// Captured only by WithInheritedEnvironment; separate from bounded explicit
	// overrides so ordinary OS inheritance keeps its existing semantics.
	inheritedEnvironment []string
	exactEnvironment     bool
}

type ProcessResult struct {
	ExitCode        int
	Output          []byte
	ResultReference string
	EffectAmbiguous bool
	Stopped         bool
	OutcomeUnknown  bool
	TimedOut        bool
	SpawnFailed     bool
	FailureCode     string
}

func childEffectsConflict(left, right ChildExecution) bool {
	leftNode := ProposedNode{Scope: left.Envelope.Scope, Effects: left.Envelope.AllowedEffects}
	rightNode := ProposedNode{Scope: right.Envelope.Scope, Effects: right.Envelope.AllowedEffects}
	return effectsConflict(leftNode, rightNode)
}

func validInvocation(child ChildExecution, invocation Invocation) bool {
	if len(invocation.Stdin) > MaxStdinBytes || invocation.RuntimeID != child.Envelope.Resolution.RuntimeID || len(invocation.Argv) == 0 || len(invocation.Argv) > 64 || invocation.CWD != child.Envelope.Workspace || !filepath.IsAbs(invocation.CWD) || invocation.OutputMax <= 0 || invocation.OutputMax > MaxCapturedOutputBytes || len(invocation.Env) > 64 {
		return false
	}
	base := strings.TrimSuffix(strings.ToLower(filepath.Base(invocation.Argv[0])), ".exe")
	if !filepath.IsAbs(invocation.Argv[0]) || base == "sh" || base == "bash" || base == "zsh" || base == "fish" || base == "cmd" || base == "powershell" || base == "pwsh" {
		return false
	}
	for _, argument := range invocation.Argv {
		if argument == "" || strings.ContainsRune(argument, '\x00') {
			return false
		}
	}
	seen := map[string]bool{}
	for _, item := range invocation.inheritedEnvironment {
		key, _, _ := strings.Cut(item, "=")
		seen[key] = true
	}
	for _, item := range invocation.Env {
		key, _, ok := strings.Cut(item, "=")
		if !ok || !validEnvironmentKey(key) || seen[key] || strings.ContainsRune(item, '\x00') {
			return false
		}
		seen[key] = true
	}
	return true
}

func validEnvironmentKey(value string) bool {
	if value == "" || len(value) > 128 {
		return false
	}
	for index, char := range value {
		if !((char >= 'A' && char <= 'Z') || char == '_' || (index > 0 && char >= '0' && char <= '9')) {
			return false
		}
	}
	return true
}

// ErrAuthenticationBlocked marks an invocation refused by the Runtime
// authentication preflight; the child is blocked before any attempt starts.
var ErrAuthenticationBlocked = errors.New("runtime authentication preflight blocked dispatch")

// EffectiveEnvironment is the environment OSProcessRunner gives the child: an
// invocation without environment inherits this process environment. The
// authentication preflight inspects exactly this value.
func EffectiveEnvironment(environment []string) []string {
	if len(environment) == 0 {
		return os.Environ()
	}
	return append([]string(nil), environment...)
}

// WithInheritedEnvironment captures the parent environment once, so preflight
// and dispatch use identical bytes even if the parent environment later changes.
// Explicit overrides remain subject to the ordinary invocation validation.
func WithInheritedEnvironment(invocation Invocation) Invocation {
	invocation.inheritedEnvironment = append([]string{}, os.Environ()...)
	return invocation
}

// EffectiveInvocationEnvironment is shared by authentication and process start.
func EffectiveInvocationEnvironment(invocation Invocation) []string {
	if invocation.exactEnvironment {
		return append([]string{}, invocation.Env...)
	}
	if invocation.inheritedEnvironment != nil {
		return append(append([]string(nil), invocation.inheritedEnvironment...), invocation.Env...)
	}
	return EffectiveEnvironment(invocation.Env)
}

// WithExactEnvironment freezes an explicit environment, including an empty
// one. Stage dispatch uses this seam so authentication and spawn inspect the
// same bytes without falling back to historical ambient inheritance.
func WithExactEnvironment(invocation Invocation, environment []string) Invocation {
	invocation.Env = append([]string{}, environment...)
	invocation.inheritedEnvironment = nil
	invocation.exactEnvironment = true
	return invocation
}

type OSProcessRunner struct{}

func (r OSProcessRunner) Run(ctx context.Context, invocation Invocation) ProcessResult {
	return r.RunObserved(ctx, invocation, nil)
}

// RunObserved uses the existing containment and stop-confirmation boundary.
// Stdin is released only after started confirms durable attempt publication.
func (OSProcessRunner) RunObserved(ctx context.Context, invocation Invocation, started func(*ProcessCorrelation) error) ProcessResult {
	if len(invocation.Stdin) > MaxStdinBytes || len(invocation.Argv) == 0 {
		return ProcessResult{ExitCode: -1, OutcomeUnknown: true}
	}
	command := exec.Command(invocation.Argv[0], invocation.Argv[1:]...)
	inputContext, cancelInput := context.WithCancel(ctx)
	defer cancelInput()
	inputReady := make(chan struct{})
	command.Stdin = &authorizedStdin{ctx: inputContext, ready: inputReady, reader: bytes.NewReader(invocation.Stdin)}
	command.Dir = invocation.CWD
	command.Env = EffectiveInvocationEnvironment(invocation)
	output := &limitedBuffer{remaining: invocation.OutputMax}
	command.Stdout = output
	command.Stderr = output
	group, err := newProcessContainment()
	if err != nil {
		return ProcessResult{ExitCode: -1, OutcomeUnknown: true}
	}
	command.WaitDelay = processStopBound
	if err = ctx.Err(); err == nil {
		err = group.start(command)
	}
	startFailed := err != nil
	failureCode := ""
	if startFailed {
		failureCode = "spawn_failed"
		if errors.Is(err, context.DeadlineExceeded) {
			failureCode = "attempt_timeout"
		}
		if errors.Is(err, context.Canceled) {
			failureCode = "attempt_cancelled"
		}
		if errors.Is(err, os.ErrPermission) {
			failureCode = "permission_denied"
		}
		if errors.Is(err, os.ErrNotExist) {
			failureCode = "executable_unavailable"
		}
	}
	stopped := false
	if err == nil && started != nil {
		identity, observationErr := ObserveProcessCorrelation(int64(command.Process.Pid))
		if observationErr == nil && identity != nil {
			observationErr = started(identity)
		} else if observationErr == nil {
			observationErr = ErrInvalidGraph
		}
		if observationErr != nil {
			cancelInput()
			stopCtx, cancel := context.WithTimeout(context.Background(), processStopBound)
			stopped = confirmProcessStop(stopCtx, group)
			cancel()
			wait := make(chan error, 1)
			go func() { wait <- command.Wait() }()
			exitCode := -1
			select {
			case waitErr := <-wait:
				exitCode = processExitCode(waitErr)
			case <-time.After(processStopBound):
			}
			captured := output.Bytes()
			digest := sha256.Sum256(captured)
			return ProcessResult{ExitCode: exitCode, Output: captured, ResultReference: "process-output:" + hex.EncodeToString(digest[:]), OutcomeUnknown: true, Stopped: stopped, TimedOut: errors.Is(ctx.Err(), context.DeadlineExceeded)}
		}
	}
	if err == nil {
		close(inputReady)
	}
	interrupted, completionUnknown := false, false
	if err == nil {
		wait := make(chan error, 1)
		go func() { wait <- command.Wait() }()
		select {
		case err = <-wait:
			gone, observationErr := group.gone()
			if !gone || observationErr != nil {
				completionUnknown = true
				stopCtx, cancel := context.WithTimeout(context.Background(), processStopBound)
				stopped = confirmProcessStop(stopCtx, group)
				cancel()
			}
		case <-ctx.Done():
			interrupted = true
			stopCtx, cancel := context.WithTimeout(context.Background(), processStopBound)
			stopped = confirmProcessStop(stopCtx, group)
			cancel()
			select {
			case err = <-wait:
			case <-time.After(processStopBound):
				err = ctx.Err()
			}
		}
	}
	exitCode := processExitCode(err)
	// Platform observation confirms group/job termination; observed escapes
	// or observation failures veto confirmation. POSIX groups remain advisory.
	captured := output.Bytes()
	digest := sha256.Sum256(captured)
	return ProcessResult{ExitCode: exitCode, Output: captured, ResultReference: "process-output:" + hex.EncodeToString(digest[:]), Stopped: stopped, EffectAmbiguous: completionUnknown, OutcomeUnknown: interrupted && !stopped || completionUnknown, TimedOut: errors.Is(ctx.Err(), context.DeadlineExceeded), SpawnFailed: startFailed, FailureCode: failureCode}
}

func processExitCode(err error) int {
	if err == nil {
		return 0
	}
	var exitError *exec.ExitError
	if errors.As(err, &exitError) {
		return exitError.ExitCode()
	}
	return -1
}

type authorizedStdin struct {
	ctx    context.Context
	ready  <-chan struct{}
	reader *bytes.Reader
}

func (r *authorizedStdin) Read(p []byte) (int, error) {
	select {
	case <-r.ctx.Done():
		return 0, r.ctx.Err()
	case <-r.ready:
		return r.reader.Read(p)
	}
}

type limitedBuffer struct {
	mu sync.Mutex
	bytes.Buffer
	remaining int
}

func (b *limitedBuffer) Write(value []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	original := len(value)
	if len(value) > b.remaining {
		value = value[:b.remaining]
	}
	if len(value) > 0 {
		_, _ = b.Buffer.Write(value)
		b.remaining -= len(value)
	}
	return original, nil
}

func (b *limitedBuffer) Bytes() []byte {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]byte(nil), b.Buffer.Bytes()...)
}

// DispatchBatch checks a caller's complete proposed batch before dispatch.
// Running attempts consume capacity even when they are not ready themselves.
func (c SuccessorConsumer) DispatchBatch(ids []string, retries map[string]bool) error {
	ready, _, err := c.Ready(retries)
	if err != nil {
		return err
	}
	allowed := map[string]bool{}
	for _, id := range ready {
		allowed[id] = true
	}
	running := 0
	for _, child := range c.graph.Children {
		if len(child.Attempts) > 0 && child.Attempts[len(child.Attempts)-1].Status == AttemptRunning {
			running++
		}
	}
	if len(ids)+running > int(c.graph.StageConcurrency) {
		return ErrStageConcurrency
	}
	selected := []ChildExecution{}
	seen := map[string]bool{}
	for _, id := range ids {
		if seen[id] || !allowed[id] {
			return ErrInvalidGraph
		}
		seen[id] = true
		for _, child := range c.graph.Children {
			if child.ExecutionID != id {
				continue
			}
			candidate := ChildExecution{Envelope: child.Envelope}
			for _, other := range selected {
				if childEffectsConflict(candidate, other) {
					return ErrInvalidGraph
				}
			}
			// Concurrent running effects also conflict with a newly proposed child.
			for _, other := range c.graph.Children {
				if len(other.Attempts) > 0 && other.Attempts[len(other.Attempts)-1].Status == AttemptRunning && childEffectsConflict(candidate, ChildExecution{Envelope: other.Envelope}) {
					return ErrInvalidGraph
				}
			}
			selected = append(selected, candidate)
		}
	}
	return nil
}

// GraphAttemptStore is retained by the independent integration support contract;
// it has no Graph dispatch owner or production store implementation.
type GraphAttemptStore interface {
	Save(context.Context, Graph) (Graph, error)
}

// DispatchRecord remains an inert acceptance input for integration support.
type DispatchRecord struct {
	ChildID, AttemptID string
	Status             AttemptStatus
	StartedAt, EndedAt time.Time
}
