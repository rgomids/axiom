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
	"sort"
	"strings"
	"sync"
	"time"
)

const MaxCapturedOutputBytes = 256 << 10

// MaxStdinBytes bounds the in-memory task packet.
const MaxStdinBytes = 256 << 10

var ErrInvalidInvocation = errors.New("invalid runtime invocation")

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

type InvocationResolver interface {
	ResolveInvocation(context.Context, ChildExecution) (Invocation, error)
}

type ProcessResult struct {
	ExitCode        int
	Output          []byte
	ResultReference string
	EffectAmbiguous bool
	Stopped         bool
	OutcomeUnknown  bool
	TimedOut        bool
}

type ProcessRunner interface {
	Run(context.Context, Invocation) ProcessResult
}

type WorkspaceValidator interface {
	ValidateWorkspace(context.Context, ChildExecution) error
}

type GraphAttemptStore interface {
	Save(context.Context, Graph) (Graph, error)
}

type DispatchRequest struct {
	Graph           Graph
	RetryChildIDs   map[string]bool
	CancelRequested bool
}

type DispatchRecord struct {
	ChildID, AttemptID string
	Status             AttemptStatus
	StartedAt, EndedAt time.Time
}

type DispatchResult struct {
	Graph   Graph
	Records []DispatchRecord
	Blocked map[string]string
}

// AttemptCompletionHook runs after process exit and before the terminal save.
// It may drain an inbox and qualify the attempt; errors leave it running.
type AttemptCompletionHook func(context.Context, ChildExecution, ProcessResult, Attempt) (Attempt, error)

type Scheduler struct {
	completionHook AttemptCompletionHook
	invocations    InvocationResolver
	runner         ProcessRunner
	workspaces     WorkspaceValidator
	store          GraphAttemptStore
	allocateID     func() (string, error)
	now            func() time.Time
}

func NewScheduler(invocations InvocationResolver, runner ProcessRunner, workspaces WorkspaceValidator, store GraphAttemptStore, allocateID func() (string, error), now func() time.Time) Scheduler {
	if allocateID == nil {
		allocateID = randomOpaqueID
	}
	if now == nil {
		now = time.Now
	}
	return Scheduler{invocations: invocations, runner: runner, workspaces: workspaces, store: store, allocateID: allocateID, now: now}
}

// WithCompletionHook returns a scheduler with an ordered terminal hook.
func (s Scheduler) WithCompletionHook(hook AttemptCompletionHook) Scheduler {
	s.completionHook = hook
	return s
}

func (s Scheduler) DispatchReady(ctx context.Context, request DispatchRequest) (DispatchResult, error) {
	result := DispatchResult{Graph: request.Graph, Blocked: map[string]string{}}
	// Own the mutable slices so failed saves and concurrent callers cannot alter
	// the supplied graph or previously saved snapshots.
	result.Graph.Children = append([]ChildExecution(nil), request.Graph.Children...)
	for i := range result.Graph.Children {
		result.Graph.Children[i].Attempts = append([]Attempt(nil), request.Graph.Children[i].Attempts...)
	}
	if !ValidGraph(result.Graph) || s.invocations == nil || s.runner == nil || s.workspaces == nil {
		return result, ErrInvalidGraph
	}
	if request.CancelRequested || ctx.Err() != nil {
		for _, child := range result.Graph.Children {
			if len(child.Attempts) == 0 {
				result.Blocked[child.ExecutionID] = "cancelled_before_dispatch"
			}
		}
		return result, nil
	}
	ready := readyChildren(result.Graph, request.RetryChildIDs, result.Blocked)
	batch := nonConflictingBatch(result.Graph, ready, result.Blocked)
	if len(batch) == 0 {
		return result, nil
	}
	type completed struct {
		index   int
		attempt Attempt
		record  DispatchRecord
		err     error
	}
	type planned struct {
		index      int
		child      ChildExecution
		invocation Invocation
		attemptID  string
		number     uint32
		started    time.Time
	}
	plans := make([]planned, 0, len(batch))
	for _, index := range batch {
		child := result.Graph.Children[index]
		if err := s.workspaces.ValidateWorkspace(ctx, child); err != nil {
			result.Blocked[child.ExecutionID] = "invalid_workspace"
			continue
		}
		invocation, err := s.invocations.ResolveInvocation(ctx, child)
		if errors.Is(err, ErrAuthenticationBlocked) {
			result.Blocked[child.ExecutionID] = "authentication_blocked"
			continue
		}
		if err != nil || !validInvocation(child, invocation) {
			result.Blocked[child.ExecutionID] = "invalid_invocation"
			continue
		}
		attemptID, err := s.allocateID()
		if err != nil || !validOpaqueID(attemptID) {
			return result, ErrInvalidGraph
		}
		started := s.now().UTC()
		attemptNumber := uint32(len(child.Attempts) + 1)
		running := Attempt{AttemptID: attemptID, Number: attemptNumber, Status: AttemptRunning, StartedAt: &started}
		result.Graph.Children[index].Attempts = append(result.Graph.Children[index].Attempts, running)
		plans = append(plans, planned{index: index, child: child, invocation: invocation, attemptID: attemptID, number: attemptNumber, started: started})
	}
	if len(plans) == 0 {
		return result, nil
	}
	if s.store != nil {
		persisted, err := s.store.Save(ctx, result.Graph)
		if err != nil {
			return result, err
		}
		result.Graph = persisted
	}
	completedCh := make(chan completed, len(plans))
	var wg sync.WaitGroup
	for _, plan := range plans {
		wg.Add(1)
		go func(index int, child ChildExecution, invocation Invocation, attemptID string, attemptNumber uint32, started time.Time) {
			defer wg.Done()
			attemptCtx, cancel := context.WithTimeout(ctx, child.Envelope.Controls.Timeout)
			defer cancel()
			process := s.runner.Run(attemptCtx, invocation)
			ended := s.now().UTC()
			status := AttemptSucceeded
			if attemptCtx.Err() != nil {
				if process.Stopped {
					status = AttemptCancelled
				} else {
					status = AttemptUnknown
				}
			} else if process.ExitCode != 0 {
				status = AttemptFailed
			}
			digest := sha256.Sum256(process.Output)
			attempt := Attempt{AttemptID: attemptID, Number: attemptNumber, Status: status, StartedAt: &started, FinishedAt: &ended, ResultReference: process.ResultReference, OutputDigest: hex.EncodeToString(digest[:]), ExitCode: process.ExitCode, AmbiguousEffect: process.EffectAmbiguous || status == AttemptUnknown, CancellationSeen: attemptCtx.Err() != nil}
			var hookErr error
			if s.completionHook != nil {
				attempt, hookErr = s.completionHook(ctx, child, process, attempt)
			}
			completedCh <- completed{index: index, attempt: attempt, err: hookErr, record: DispatchRecord{ChildID: child.ExecutionID, AttemptID: attemptID, Status: attempt.Status, StartedAt: started, EndedAt: ended}}
		}(plan.index, plan.child, plan.invocation, plan.attemptID, plan.number, plan.started)
	}
	go func() { wg.Wait(); close(completedCh) }()
	var completionErr error
	persistenceFailed := false
	for item := range completedCh {
		if persistenceFailed {
			continue
		}
		if item.err != nil {
			completionErr = item.err
			continue
		}
		candidate := result.Graph
		candidate.Children = append([]ChildExecution(nil), result.Graph.Children...)
		child := &candidate.Children[item.index]
		child.Attempts = append([]Attempt(nil), child.Attempts...)
		child.Attempts[len(child.Attempts)-1] = item.attempt
		if s.store != nil {
			persisted, err := s.store.Save(ctx, candidate)
			if err != nil {
				completionErr = err
				persistenceFailed = true
				continue
			}
			candidate = persisted
		}
		result.Graph = candidate
		result.Records = append(result.Records, item.record)
	}
	if completionErr != nil {
		return result, completionErr
	}
	sort.Slice(result.Records, func(i, j int) bool { return result.Records[i].ChildID < result.Records[j].ChildID })
	return result, nil
}

func readyChildren(graph Graph, retries map[string]bool, blocked map[string]string) []int {
	succeeded := map[string]bool{}
	for _, child := range graph.Children {
		if len(child.Attempts) > 0 && child.Attempts[len(child.Attempts)-1].Status == AttemptSucceeded {
			succeeded[child.ExecutionID] = true
		}
	}
	ready := make([]int, 0, len(graph.Children))
	for index, child := range graph.Children {
		if child.Envelope.IntegrationOwner {
			blocked[child.ExecutionID] = "integration_pending"
			continue
		}
		if len(child.Attempts) > 0 && child.Attempts[len(child.Attempts)-1].Status == AttemptRunning {
			blocked[child.ExecutionID] = "attempt_running"
			continue
		}
		if succeeded[child.ExecutionID] {
			continue
		}
		dependenciesReady := true
		for _, dependency := range child.Envelope.Dependencies {
			if !succeeded[dependency] {
				dependenciesReady = false
				break
			}
		}
		if !dependenciesReady {
			blocked[child.ExecutionID] = "dependencies_unsatisfied"
			continue
		}
		if len(child.Attempts) > 0 {
			last := child.Attempts[len(child.Attempts)-1]
			if last.Status == AttemptUnknown {
				blocked[child.ExecutionID] = "unknown_outcome_requires_reconciliation"
				continue
			}
			if last.AmbiguousEffect {
				blocked[child.ExecutionID] = "ambiguous_effect"
				continue
			}
			if uint32(len(child.Attempts)) >= child.Envelope.Controls.MaximumAttempts {
				blocked[child.ExecutionID] = "attempts_exhausted"
				continue
			}
			if !retries[child.ExecutionID] {
				blocked[child.ExecutionID] = "retry_not_authorized"
				continue
			}
		}
		ready = append(ready, index)
	}
	sort.Slice(ready, func(i, j int) bool {
		return graph.Children[ready[i]].ExecutionID < graph.Children[ready[j]].ExecutionID
	})
	return ready
}

func nonConflictingBatch(graph Graph, ready []int, blocked map[string]string) []int {
	batch := make([]int, 0, len(ready))
	for _, candidate := range ready {
		conflict := false
		for _, selected := range batch {
			if childEffectsConflict(graph.Children[candidate], graph.Children[selected]) {
				conflict = true
				break
			}
		}
		if conflict {
			blocked[graph.Children[candidate].ExecutionID] = "effect_conflict_serialized"
			continue
		}
		batch = append(batch, candidate)
	}
	return batch
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
func (OSProcessRunner) RunObserved(ctx context.Context, invocation Invocation, started func(*ProcessCorrelation) error) ProcessResult {
	if len(invocation.Stdin) > MaxStdinBytes || len(invocation.Argv) == 0 {
		return ProcessResult{ExitCode: -1, OutcomeUnknown: true}
	}
	command := exec.Command(invocation.Argv[0], invocation.Argv[1:]...)
	command.Stdin = bytes.NewReader(invocation.Stdin)
	command.Dir = invocation.CWD
	command.Env = EffectiveInvocationEnvironment(invocation)
	output := &limitedBuffer{remaining: invocation.OutputMax}
	command.Stdout = output
	command.Stderr = output
	group, err := newProcessContainment()
	if err != nil {
		return ProcessResult{ExitCode: -1, OutcomeUnknown: true}
	}
	defer group.close()
	command.WaitDelay = processStopBound
	if err = ctx.Err(); err == nil {
		err = group.start(command)
	}
	startFailed := err != nil
	stopped := false
	if err == nil && started != nil {
		identity, observationErr := ObserveProcessCorrelation(int64(command.Process.Pid))
		if observationErr == nil && identity != nil {
			observationErr = started(identity)
		} else if observationErr == nil {
			observationErr = ErrInvalidGraph
		}
		if observationErr != nil {
			stopCtx, cancel := context.WithTimeout(context.Background(), processStopBound)
			stopped = confirmProcessStop(stopCtx, group)
			cancel()
			wait := make(chan error, 1)
			go func() { wait <- command.Wait() }()
			select {
			case <-wait:
			case <-time.After(processStopBound):
			}
			return ProcessResult{ExitCode: -1, OutcomeUnknown: true, Stopped: stopped}
		}
	}
	interrupted := false
	if err == nil {
		wait := make(chan error, 1)
		go func() { wait <- command.Wait() }()
		select {
		case err = <-wait:
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
	exitCode := 0
	if err != nil {
		exitCode = -1
		var exitError *exec.ExitError
		if errors.As(err, &exitError) {
			exitCode = exitError.ExitCode()
		}
	}
	// Platform observation confirms group/job termination; observed escapes
	// or observation failures veto confirmation. POSIX groups remain advisory.
	captured := output.Bytes()
	digest := sha256.Sum256(captured)
	return ProcessResult{ExitCode: exitCode, Output: captured, ResultReference: "process-output:" + hex.EncodeToString(digest[:]), Stopped: stopped, OutcomeUnknown: interrupted && !stopped || startFailed, TimedOut: errors.Is(ctx.Err(), context.DeadlineExceeded)}
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
