package stageexecution

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"slices"

	"github.com/rgomids/axiom/internal/executiongraph"
	"github.com/rgomids/axiom/internal/runtimeadapter"
	"github.com/rgomids/axiom/internal/workflow"
)

// ErrRuntimeLaunch contains no executable paths, child output or credentials.
var ErrRuntimeLaunch = errors.New("runtime_launch_refused: check configured executable, frozen selectors and local paths")
var ErrRuntimeOutput = errors.New("runtime_output_unsafe: declared output could not be read safely")

const maxRuntimeArtifactBytes = 1 << 20

// RuntimeLauncher uses the existing OS-process contract for both supported
// runtimes. All paths and environment are machine-local inputs; selectors come
// exclusively from the persisted attempt. InboxRoot must be a private, empty
// output area outside the repository. Each attempt gets a new directory.
// No vendor command, authentication probe or support prediction runs in Build.
type RuntimeLauncher struct {
	Workspace, InboxRoot, AxiomPath string
	Environment                     []string
	// Tests replace only the machine-local root lookup, keeping the real
	// bounded reader and selector checks while avoiding host configuration.
	selectorRoots func(context.Context, executiongraph.Invocation, bool) (runtimeadapter.EffortSourceRoots, error)
}

func (l RuntimeLauncher) inbox(r LaunchRequest) (string, error) {
	if !workflow.ValidToken(r.Attempt.AttemptID) || filepath.Base(r.Attempt.AttemptID) != r.Attempt.AttemptID {
		return "", ErrRuntimeLaunch
	}
	return filepath.Join(l.InboxRoot, r.Attempt.AttemptID), nil
}

func (l RuntimeLauncher) Build(ctx context.Context, r LaunchRequest) (executiongraph.Invocation, error) {
	fail := func() (executiongraph.Invocation, error) { return executiongraph.Invocation{}, ErrRuntimeLaunch }
	frozen := r.Attempt.Invocation
	if !runtimeOutputSupported || ctx.Err() != nil || workflow.DigestText([]byte(frozen.Instructions)) != frozen.InstructionsDigest {
		return fail()
	}
	inbox, err := l.inbox(r)
	if err != nil {
		return fail()
	}
	// Reject symlinks in every machine-local path before constructing argv.
	for _, path := range []string{l.Workspace, l.InboxRoot, l.AxiomPath} {
		if !cleanRuntimePath(path) {
			return fail()
		}
	}
	info, err := os.Lstat(l.InboxRoot)
	if err != nil || !info.IsDir() || info.Mode().Perm()&0o077 != 0 {
		return fail()
	}
	if !filepath.IsAbs(frozen.Executable) {
		return fail()
	}
	root, err := os.OpenRoot(l.InboxRoot)
	if err != nil {
		return fail()
	}
	defer root.Close()
	if err := root.Mkdir(r.Attempt.AttemptID, 0o700); err != nil {
		return fail()
	}
	inv, err := runtimeadapter.BuildStageInvocation(runtimeadapter.StageInvocationRequest{
		Profile:   runtimeadapter.CommandProfile{RuntimeID: frozen.Runtime, Executable: frozen.Executable, Model: frozen.Model},
		Workspace: l.Workspace, Inbox: inbox, AxiomPath: l.AxiomPath,
		RepositoryWrite: slices.Contains(r.Attempt.Authority.Effects, "workspace_write"),
		Process:         slices.Contains(r.Attempt.Authority.Effects, "process"), ParentEnvironment: l.Environment,
	})
	if err != nil {
		return fail()
	}
	environment := executiongraph.EffectiveInvocationEnvironment(inv)
	if frozen.Effort != runtimeadapter.RuntimeDefaultEffort {
		switch frozen.Runtime {
		case "codex":
			inv.Argv = append(inv.Argv, "--config", "model_reasoning_effort="+frozen.Effort)
		case "claude":
			environment = append(environment, "CLAUDE_CODE_EFFORT_LEVEL="+frozen.Effort)
		default:
			return fail()
		}
	}
	inv = executiongraph.WithExactEnvironment(inv, environment)
	resolveRoots := (runtimeadapter.StageEffortRoots{Repository: l.Workspace}).ResolveOperational
	if l.selectorRoots != nil {
		resolveRoots = l.selectorRoots
	}
	roots, err := resolveRoots(ctx, inv, slices.Contains(r.Attempt.Authority.Effects, "workspace_write"))
	if err != nil {
		return fail()
	}
	if err := runtimeadapter.CheckFrozenSelectors(ctx, frozen.Runtime, frozen.Model, frozen.Effort, roots, inv); err != nil {
		return fail()
	}
	// JSON is data on stdin, never shell syntax or a source of authority. Child
	// output is accepted only as bounded files with declared names in this inbox.
	packet := struct {
		ExecutionID, AttemptID, StageID, AgentID, Instructions, OutputDirectory string
		RunnerGeneration                                                        uint64
		Outputs                                                                 []string
	}{r.Execution.ExecutionID, r.Attempt.AttemptID, r.Ref.StageID, r.Ref.AgentID, frozen.Instructions, inbox, r.Attempt.RunnerGeneration, attemptOutputs(r)}
	inv.Stdin, err = json.Marshal(packet)
	if err != nil || len(inv.Stdin) > executiongraph.MaxStdinBytes {
		return fail()
	}
	return inv, nil
}

// Collect never copies raw stdout/stderr into diagnostics. Nonzero vendor
// exits (including auth/option errors) are observed failures without fallback.
func (l RuntimeLauncher) Collect(ctx context.Context, r LaunchRequest, p executiongraph.ProcessResult) (Outcome, error) {
	if p.ExitCode != 0 {
		return Outcome{Diagnostic: "runtime_exit_nonzero: inspect local Runtime authentication/options; no fallback attempted"}, nil
	}
	inbox, err := l.inbox(r)
	if !runtimeOutputSupported || err != nil || !cleanRuntimePath(inbox) {
		return Outcome{}, ErrRuntimeOutput
	}
	root, err := os.OpenRoot(inbox)
	if err != nil {
		return Outcome{}, ErrRuntimeOutput
	}
	defer root.Close()
	var out Outcome
	for _, name := range attemptOutputs(r) {
		if ctx.Err() != nil {
			return Outcome{}, ctx.Err()
		}
		if !workflow.ValidToken(name) || filepath.Base(name) != name {
			return Outcome{}, ErrRuntimeOutput
		}
		info, err := root.Lstat(name)
		if errors.Is(err, os.ErrNotExist) {
			continue
		} // aggregate reports missing_output
		if err != nil || !info.Mode().IsRegular() || info.Size() > maxRuntimeArtifactBytes {
			return Outcome{}, ErrRuntimeOutput
		}
		f, err := root.OpenFile(name, os.O_RDONLY|noFollowRuntimeOutput, 0)
		if err != nil {
			return Outcome{}, ErrRuntimeOutput
		}
		opened, statErr := f.Stat()
		if statErr != nil || !os.SameFile(info, opened) {
			f.Close()
			return Outcome{}, ErrRuntimeOutput
		}
		data, readErr := io.ReadAll(io.LimitReader(f, maxRuntimeArtifactBytes+1))
		closeErr := f.Close()
		if readErr != nil || closeErr != nil || len(data) > maxRuntimeArtifactBytes {
			return Outcome{}, ErrRuntimeOutput
		}
		out.Artifacts = append(out.Artifacts, workflow.ArtifactRef{Name: name, RepositoryKey: r.Attempt.Authority.RepositoryKey, Digest: workflow.DigestText(data), Size: int64(len(data))})
	}
	return out, nil
}

func cleanRuntimePath(path string) bool {
	if !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return false
	}
	for current := path; ; current = filepath.Dir(current) {
		info, err := os.Lstat(current)
		if err != nil || info.Mode()&os.ModeSymlink != 0 {
			return false
		}
		if current == filepath.Dir(current) {
			return true
		}
	}
}

func attemptOutputs(r LaunchRequest) []string {
	if r.Agent.OutputScope {
		return r.Agent.Outputs
	}
	return r.Stage.Outputs
}
