package stageexecution

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/rgomids/axiom/internal/executiongraph"
	"github.com/rgomids/axiom/internal/runtimeadapter"
	"github.com/rgomids/axiom/internal/workflow"
)

// ErrRuntimeLaunch contains no executable paths, child output or credentials.
var ErrRuntimeLaunch = errors.New("runtime_launch_refused: check configured executable, frozen selectors and local paths")
var ErrRuntimeOutput = errors.New("runtime_output_unsafe: declared output could not be read safely")

// ErrInvalidRuntimeOutput is a declared output that was read but fails its check.
var ErrInvalidRuntimeOutput = errors.New("runtime_output_invalid: declared output failed its check")

// A started validation command whose terminal result was not committed reports
// either a confirmed end (exit observed or group stop confirmed) or an
// unconfirmed stop. The engine treats anything else as unconfirmed.
var (
	ErrCommandTerminated     = errors.New("validation_command_terminated_uncommitted")
	ErrCommandOutcomeUnknown = errors.New("validation_command_stop_unconfirmed")
)

const maxRuntimeArtifactBytes = 1 << 20

// Fixed identity gate for configured validation commands; see Collect.
const (
	commandGateShell  = "/bin/sh"
	commandGateScript = `IFS= read -r gate && [ "$gate" = axiom-command-start ] || exit 125; exec "$@"`
)

// RuntimeLauncher uses the existing OS-process contract for both supported
// runtimes. All paths and environment are machine-local inputs; selectors come
// exclusively from the persisted attempt. InboxRoot must be a private, empty
// output area outside the repository. Each attempt gets a new directory.
// No vendor command, authentication probe or support prediction runs in Build.
type RuntimeLauncher struct {
	// ValidateOutput resolves a bounded retained-artifact reference using the
	// frozen contract. Nil refuses configured validation, never skips it.
	ValidateOutput func(context.Context, workflow.OutputSubject, []byte) (workflow.ArtifactRef, error)
	// RetainOutput retains delivered content for a checked output when it is
	// not already a typed reference, then validates it with the same check.
	RetainOutput func(context.Context, workflow.OutputSubject, []byte) (workflow.ArtifactRef, error)
	// ReadInput returns a predecessor output's exact bytes for the packet.
	ReadInput                       func(context.Context, workflow.Execution, string, workflow.ArtifactRef) ([]byte, error)
	Workspace, InboxRoot, AxiomPath string
	// WorktreeRoots binds enrolled attempts to their immutable confined name.
	WorktreeRoots map[string]string
	// Workspaces maps every unenrolled Repository key to its original root.
	Workspaces  map[string]string
	Environment []string
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
	if _, err := l.Inspect(ctx, r); err != nil {
		return executiongraph.Invocation{}, err
	}
	root, err := os.OpenRoot(l.InboxRoot)
	if err != nil {
		return refuseLaunch(ctx, err, "invocation_build", "")
	}
	defer root.Close()
	if err = root.Mkdir(r.Attempt.AttemptID, 0700); err != nil {
		return refuseLaunch(ctx, err, "invocation_build", "")
	}
	for _, area := range []string{sendArea, receiveArea} {
		if err = root.Mkdir(filepath.Join(r.Attempt.AttemptID, area), 0700); err != nil {
			return refuseLaunch(ctx, err, "invocation_build", "")
		}
	}
	// Inspect again against materialized paths; a preview is never admission.
	inv, err := l.Inspect(ctx, r)
	if err != nil {
		var typed *launchError
		if errors.As(err, &typed) {
			typed.materialization = "prepared"
		}
	}
	return inv, err
}

// Inspect constructs and checks one planned invocation without materializing it.
// Absent output directories are not permission-tested or certified as writable.
// forAttempt binds the workspace to the attempt's authorized Repository; an
// unbound key leaves no workspace, so every later path check refuses.
func (l RuntimeLauncher) forAttempt(r LaunchRequest) RuntimeLauncher {
	if l.Workspaces != nil {
		l.Workspace = l.Workspaces[r.Attempt.Authority.RepositoryKey]
	}
	if w := r.Attempt.Workspace; w != nil {
		root := l.WorktreeRoots[w.RepositoryKey]
		l.Workspace = ""
		if filepath.IsAbs(root) && workflow.ValidWorktreeName(w.Worktree) {
			l.Workspace = filepath.Join(root, w.Worktree)
		}
	}
	return l
}

func (l RuntimeLauncher) Inspect(ctx context.Context, r LaunchRequest) (result executiongraph.Invocation, resultErr error) {
	l = l.forAttempt(r)
	defer func() {
		var typed *launchError
		if errors.As(resultErr, &typed) {
			typed.deterministic = typed.failure.Valid() && typed.failure.Reason != "builder_failed"
			typed.materialization = "none"
		}
	}()
	fail := func() (executiongraph.Invocation, error) {
		return refuseLaunch(ctx, ErrRuntimeLaunch, "invocation_build", "builder_failed")
	}
	frozen := r.Attempt.Invocation
	if ctx.Err() != nil {
		return executiongraph.Invocation{}, ctx.Err()
	}
	if !runtimeOutputSupported || workflow.DigestText([]byte(frozen.Instructions)) != frozen.InstructionsDigest {
		return refuseLaunch(ctx, ErrRuntimeLaunch, "invocation_build", "identity_unsafe")
	}
	inbox, err := l.inbox(r)
	if err != nil {
		return fail()
	}
	// Existing inputs require exact identities; planned outputs allow absent leaves.
	for _, path := range []string{l.Workspace, l.AxiomPath} {
		if !cleanRuntimePath(path) {
			return refuseLaunch(ctx, ErrRuntimeLaunch, "invocation_build", "path_unsafe")
		}
	}
	if !plannedRuntimePath(l.InboxRoot) || !plannedRuntimePath(inbox) || !filepath.IsAbs(frozen.Executable) {
		return refuseLaunch(ctx, ErrRuntimeLaunch, "invocation_build", "path_unsafe")
	}
	if info, err := os.Lstat(l.InboxRoot); err == nil {
		if !info.IsDir() || info.Mode().Perm()&0077 != 0 {
			return refuseLaunch(ctx, ErrRuntimeLaunch, "invocation_build", "path_unsafe")
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return refuseLaunch(ctx, err, "invocation_build", "")
	}

	inv, err := runtimeadapter.BuildStageInvocation(runtimeadapter.StageInvocationRequest{
		Profile:   runtimeadapter.CommandProfile{RuntimeID: frozen.Runtime, Executable: frozen.Executable, Model: frozen.Model},
		Workspace: l.Workspace, Inbox: inbox, AxiomPath: l.AxiomPath,
		RepositoryWrite: slices.Contains(r.Attempt.Authority.Effects, "workspace_write"),
		// A writable attempt may already change the repository, so it also gets a
		// shell to build, test and wait; "process" is not an attempt effect kind.
		Process:           slices.Contains(r.Attempt.Authority.Effects, "workspace_write"),
		ParentEnvironment: l.Environment,
		// The child's own attempt context marks it as a Runtime child, so it
		// cannot record human authority through the Axiom CLI it may invoke.
		Correlation: attemptCorrelation(r),
	})
	if err != nil {
		return refuseLaunch(ctx, err, "invocation_build", "")
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
		return refuseLaunch(ctx, err, "source_resolution", "")
	}
	if err := runtimeadapter.CheckFrozenSelectors(ctx, frozen.Runtime, frozen.Model, frozen.Effort, roots, inv); err != nil {
		return refuseLaunch(ctx, err, "selector_inspection", "")
	}
	// JSON is data on stdin, never shell syntax or a source of authority. Child
	// output is accepted only as bounded files with declared names in this inbox.
	type inputValue struct {
		Provenance *workflow.InputProvenance `json:"provenance,omitempty"`
		Content    string                    `json:"content,omitempty"`
		Digest     string                    `json:"digest,omitempty"`
		WorkItem   *workflow.WorkItemLink    `json:"workItem,omitempty"`
		Artifact   *workflow.ArtifactRef     `json:"artifact,omitempty"`
		Project    *workflow.ProjectInput    `json:"project,omitempty"`
		Source     string                    `json:"source,omitempty"`
	}
	inputs := map[string]inputValue{}
	for _, name := range r.Agent.Inputs {
		found := false
		for _, target := range r.Stage.WorkItemInputs {
			if target == name {
				inputs[name] = inputValue{WorkItem: &r.Execution.WorkItem}
				found = true
			}
		}
		for _, input := range r.Stage.StageOutputInputs {
			if input.Name != name {
				continue
			}
			found = true
			var selected *workflow.ArtifactRef
			var provenance *workflow.InputProvenance
			for _, stage := range r.Execution.Stages {
				if stage.StageID != input.StageID {
					continue
				}
				if stage.Status != workflow.StageCompleted {
					return fail()
				}
				for _, task := range stage.Tasks {
					if len(task.Attempts) == 0 {
						continue
					}
					attempt := task.Attempts[len(task.Attempts)-1]
					if attempt.Status != workflow.AttemptSucceeded {
						return fail()
					}
					for _, artifact := range attempt.Artifacts {
						if artifact.Name != input.Output {
							continue
						}
						// Outputs are Execution-scoped; a Repository-bound agent may read
						// what an earlier stage produced for another bound Repository.
						if selected != nil || !executionRepository(r.Execution, artifact.RepositoryKey) {
							return fail()
						}
						copy := artifact
						selected = &copy
						provenance = &workflow.InputProvenance{ExecutionID: r.Execution.ExecutionID, StageID: stage.StageID, AttemptID: attempt.AttemptID}
						if !provenance.Valid() {
							return fail()
						}
					}
				}
			}
			if selected == nil && input.Required {
				return fail()
			}
			if selected != nil {
				value := inputValue{Artifact: selected, Provenance: provenance}
				if l.ReadInput != nil {
					content, err := l.ReadInput(ctx, r.Execution, provenance.AttemptID, *selected)
					if err != nil {
						return fail()
					}
					value.Content = string(content)
				}
				inputs[name] = value
			}
		}
		for _, input := range r.Stage.FrozenInputs {
			if input.Name == name {
				found = true
				if input.Required && input.Project == nil && input.Artifact == nil && input.Content == "" {
					return fail()
				}
				if input.Artifact != nil && (input.Provenance == nil || !input.Provenance.Valid()) {
					return fail()
				}
				if input.Project != nil || input.Artifact != nil || input.Content != "" {
					value := inputValue{Provenance: input.Provenance, Project: input.Project, Artifact: input.Artifact, Source: input.Source, Content: input.Content, Digest: input.Digest}
					if input.Kind == "work-item" {
						value.WorkItem = &r.Execution.WorkItem
					}
					inputs[name] = value
				}
			}
		}
		if !found {
			return fail()
		}
	}
	// A dependent agent receives the outputs its same-stage dependencies
	// produced, as "<agent>/<output>"; dependsOn already orders the attempts.
	for _, dependency := range r.Agent.DependsOn {
		for _, stage := range r.Execution.Stages {
			if stage.StageID != r.Ref.StageID {
				continue
			}
			for _, task := range stage.Tasks {
				if task.AgentID != dependency || len(task.Attempts) == 0 {
					continue
				}
				attempt := task.Attempts[len(task.Attempts)-1]
				if attempt.Status != workflow.AttemptSucceeded {
					return fail()
				}
				for _, artifact := range attempt.Artifacts {
					value := inputValue{Artifact: &artifact, Provenance: &workflow.InputProvenance{ExecutionID: r.Execution.ExecutionID, StageID: stage.StageID, AttemptID: attempt.AttemptID}}
					if l.ReadInput != nil {
						content, err := l.ReadInput(ctx, r.Execution, attempt.AttemptID, artifact)
						if err != nil {
							return fail()
						}
						value.Content = string(content)
					}
					inputs[dependency+"/"+artifact.Name] = value
				}
			}
		}
	}
	repositoryWritable := slices.Contains(r.Attempt.Authority.Effects, "workspace_write")
	contract := stageContract
	if r.Attempt.Workspace != nil && repositoryWritable {
		contract += " Make repository changes only in RepositoryRoot, on your assigned branch. Commit them there if your sandbox allows it; if committing is refused, leave them uncommitted and Axiom records them as one commit on your branch after you exit. Never stop because a commit was refused: always write the declared outputs."
	}
	// Message areas are the same for both Runtimes. Their contents are data:
	// a message never completes a task, satisfies a gate or grants authority.
	// Contract is the engine's delivery protocol, read first by the child. Without
	// it a model can answer on stdout and exit 0, which is missing_output.
	packet := struct {
		Contract                                                                string
		ExecutionID, AttemptID, StageID, AgentID, Instructions, OutputDirectory string
		// RepositoryKey is a Project label, not a directory name. RepositoryRoot
		// is the bound repository; writable only when RepositoryWritable.
		RepositoryKey, RepositoryRoot                 string
		RepositoryWritable                            bool
		MessageSendDirectory, MessageReceiveDirectory string
		RunnerGeneration                              uint64
		Outputs                                       []string
		DefinitionDigest                              string
		OutputChecks                                  []workflow.OutputCheck
		Inputs                                        map[string]inputValue
	}{contract, r.Execution.ExecutionID, r.Attempt.AttemptID, r.Ref.StageID, r.Ref.AgentID, frozen.Instructions, inbox, r.Attempt.Authority.RepositoryKey, l.Workspace, repositoryWritable, filepath.Join(inbox, sendArea), filepath.Join(inbox, receiveArea), r.Attempt.RunnerGeneration, attemptOutputs(r), r.Stage.DefinitionDigest, r.Stage.OutputChecks, inputs}
	inv.Stdin, err = json.Marshal(packet)
	if err != nil || len(inv.Stdin) > executiongraph.MaxStdinBytes {
		return fail()
	}
	return inv, nil
}

// Collect never copies raw stdout/stderr into diagnostics. Nonzero vendor
// exits (including auth/option errors) are observed failures without fallback.
func (l RuntimeLauncher) Collect(ctx context.Context, r LaunchRequest, p executiongraph.ProcessResult) (Outcome, error) {
	l = l.forAttempt(r)
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
		artifact := workflow.ArtifactRef{Name: name, RepositoryKey: r.Attempt.Authority.RepositoryKey, Digest: workflow.DigestText(data), Size: int64(len(data))}
		for _, check := range r.Stage.OutputChecks {
			if check.Name != name {
				continue
			}
			if l.ValidateOutput == nil {
				return Outcome{}, ErrRuntimeOutput
			}
			subject := workflow.OutputSubject{Isolated: r.Attempt.Workspace != nil, ExecutionID: r.Execution.ExecutionID, AttemptID: r.Attempt.AttemptID, StageID: r.Ref.StageID, DefinitionDigest: r.Stage.DefinitionDigest, RepositoryKey: r.Attempt.Authority.RepositoryKey, Check: check}
			artifact, err = l.ValidateOutput(ctx, subject, data)
			// A reference-shaped file must validate as a reference: retaining it as
			// content would launder a forged or stale reference.
			if err != nil && l.RetainOutput != nil && !artifactReferenceShaped(data) {
				artifact, err = l.RetainOutput(ctx, subject, data)
			}
			if err != nil || artifact.Name != name || artifact.RepositoryKey != r.Attempt.Authority.RepositoryKey || !workflow.ValidToken(artifact.RetainedID) || artifact.Size < 0 {
				// The file exists but fails its check: a deterministic attempt
				// failure, not an unknown effect.
				return Outcome{}, ErrInvalidRuntimeOutput
			}
		}
		out.Artifacts = append(out.Artifacts, artifact)
	}
	for _, check := range r.Stage.Commands {
		for _, effect := range check.Effects {
			if !slices.Contains(r.Attempt.Authority.Effects, effect) {
				return Outcome{}, ErrRuntimeOutput
			}
		}
		if !slices.Contains(check.Effects, "validation_command") || len(check.Argv) == 0 || !cleanRuntimePath(l.Workspace) {
			return Outcome{}, ErrRuntimeOutput
		}
		executable := check.Argv[0]
		if !filepath.IsAbs(executable) || !cleanRuntimePath(executable) {
			return Outcome{}, ErrRuntimeOutput
		}
		if r.PrepareCommand == nil || r.StartCommand == nil || r.FinishCommand == nil {
			return Outcome{}, ErrRuntimeOutput
		}
		if err := r.PrepareCommand(check.ID); err != nil {
			return Outcome{}, err
		}
		commandCtx, cancel := context.WithTimeout(ctx, time.Duration(check.TimeoutSeconds)*time.Second)
		// A fast command could exit before its identity is observed and
		// would then be an unknown effect. The fixed gate blocks on stdin,
		// which the runner releases only after StartCommand is durable, then
		// execs in place (same PID, start time and group). Argv stays
		// positional: no shell expansion of configured arguments.
		gate := []string{commandGateShell, "-c", commandGateScript, "axiom-command-gate", executable}
		inv := executiongraph.Invocation{Argv: append(gate, check.Argv[1:]...), CWD: l.Workspace, Stdin: []byte("axiom-command-start\n"), OutputMax: 4096}
		if _, err := runtimeadapter.StageEnvironment(r.Attempt.Invocation.Runtime, l.Environment); err != nil {
			cancel()
			return Outcome{}, ErrRuntimeOutput
		}
		safeEnvironment, err := runtimeadapter.CommandEnvironment(l.Environment)
		if err != nil {
			cancel()
			return Outcome{}, ErrRuntimeOutput
		}
		// Workspace code the agent could edit runs here, so the command carries
		// the same Runtime-child markers as the agent (advisory guard, D-01).
		correlation, err := attemptCorrelation(r).Environment()
		if err != nil {
			cancel()
			return Outcome{}, ErrRuntimeOutput
		}
		inv = executiongraph.WithExactEnvironment(inv, append(safeEnvironment, correlation...))
		result := (executiongraph.OSProcessRunner{}).RunObserved(commandCtx, inv, func(p *executiongraph.ProcessCorrelation) error {
			return r.StartCommand(check.ID, workflow.ProcessIdentity{PID: p.PID, StartIdentity: p.StartIdentity, GroupIdentity: p.GroupIdentity})
		})
		cancel()
		if result.OutcomeUnknown || result.EffectAmbiguous {
			return Outcome{}, ErrCommandOutcomeUnknown
		}
		if result.SpawnFailed || result.TimedOut || ctx.Err() != nil {
			return Outcome{}, errors.Join(ErrCommandTerminated, ErrRuntimeOutput)
		}
		record, err := r.FinishCommand(check.ID, result.ExitCode)
		if err != nil {
			// The process ended observably; only its terminal record is missing.
			return Outcome{}, errors.Join(ErrCommandTerminated, err)
		}
		out.Commands = append(out.Commands, record)
		if result.ExitCode != 0 {
			out.Diagnostic = "command_validation_failed"
			break
		}
	}
	return out, nil
}

func attemptCorrelation(r LaunchRequest) runtimeadapter.AttemptCorrelation {
	return runtimeadapter.AttemptCorrelation{ProjectID: r.Execution.ProjectID, ParentID: r.Execution.ExecutionID, ChildID: r.Execution.ExecutionID, AttemptID: r.Attempt.AttemptID}
}

// CleanRuntimePath reports an absolute, clean path with no symlinked
// component, the containment every machine-local launcher path must meet.
func CleanRuntimePath(path string) bool { return cleanRuntimePath(path) }

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
	return append(append([]string(nil), r.Stage.Outputs...), r.Stage.OptionalOutputs...)
}

func plannedRuntimePath(path string) bool {
	if !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return false
	}
	for current := path; ; current = filepath.Dir(current) {
		info, err := os.Lstat(current)
		if err == nil {
			return info.IsDir() && cleanRuntimePath(current)
		}
		if !errors.Is(err, os.ErrNotExist) || current == filepath.Dir(current) {
			return false
		}
	}
}

const stageContract = "You are an Axiom stage agent running non-interactively. Do the work in Instructions using Inputs. " +
	"Deliver it as files: for each name in Outputs, write one regular file with exactly that name directly inside OutputDirectory " +
	"(no extension, no subdirectory, no symlink). The stage completes only when every declared output file exists; " +
	"text printed to stdout is discarded and does not count. Write the content itself (Markdown or text) for every output, " +
	"including those in OutputChecks: Axiom retains and checks it; never invent artifact references. " +
	"RepositoryKey is the Project's label for the bound repository, not a directory name: never search for it. " +
	"RepositoryRoot is the repository root. When RepositoryWritable is true it is your working directory and you make repository changes there; " +
	"otherwise it is read-only: read and verify it, never modify it. " +
	"If you cannot complete the stage, do not write the output files: explain why on stdout and exit, so the attempt fails instead of passing. " +
	"Use only the tools you are given; a check you cannot run with them (for example a shell command) is reported in the output as not run and does not block the stage. " +
	"MessageSendDirectory and MessageReceiveDirectory are optional coordination areas. " +
	"To send a message write one file MessageSendDirectory/ID.json holding exactly the JSON object with fields id, recipient, kind and content, " +
	"plus replyTo only on an answer: id equals ID (lowercase letters, digits and hyphens), recipient is another agent ID of this Workflow, " +
	"kind is question, answer, blocker or result, content is text under 4 KiB, and replyTo is the question id. Any other shape is discarded unread. " +
	"Delivered messages appear as MessageReceiveDirectory/ID.json with id, sender, kind, optional replyTo and content. " +
	"An answer is accepted only while its asker still runs, so an agent that asks keeps polling for the answer before it exits. " +
	"Before you exit, read MessageReceiveDirectory again and answer every question addressed to you; when Instructions say a peer will ask you something, keep polling until it arrives and you answered it. Exit when the files are written."

func artifactReferenceShaped(data []byte) bool {
	var ref struct {
		Kind string `json:"kind"`
	}
	return json.Unmarshal(data, &ref) == nil && ref.Kind == "artifact"
}

func executionRepository(x workflow.Execution, key string) bool {
	for _, r := range x.Repositories {
		if r.Key == key {
			return true
		}
	}
	return false
}
