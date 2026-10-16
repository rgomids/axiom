package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/rgomids/axiom/internal/capabilityproof"
	"github.com/rgomids/axiom/internal/cli"
	"github.com/rgomids/axiom/internal/executiongraph"
	"github.com/rgomids/axiom/internal/local"
	"github.com/rgomids/axiom/internal/projectapp"
	"github.com/rgomids/axiom/internal/runtimeadapter"
	"github.com/rgomids/axiom/internal/runtimeprofile"
	"github.com/rgomids/axiom/internal/runtimeprofile/runtimeprofiletest"
	"github.com/rgomids/axiom/internal/stageexecution"
	"github.com/rgomids/axiom/internal/validatorpolicy"
	"github.com/rgomids/axiom/internal/workflowdefinition"
)

const stageJourneyDecision = "Explicit synthetic human gate decision\n"

var stageJourneyVersions = map[string]string{"codex": "0.159.1", "claude": "2.1.296"}

type stageJourney struct {
	env                            workItemSourceEnvironment
	axiom, distributed, bin, home  string
	workspace, repository, overlay string
	digests                        map[string]string
	managed                        map[string]string
	policy                         string
	environment                    []string
}

type stageJourneyEvent struct {
	Status        string          `json:"status"`
	Category      string          `json:"category"`
	PreviewDigest string          `json:"previewDigest"`
	References    []string        `json:"references"`
	Stage         json.RawMessage `json:"stage"`
	Capability    json.RawMessage `json:"capability"`
	Payload       json.RawMessage `json:"payload"`
}

func newStageJourney(t *testing.T, concurrency ...int) *stageJourney {
	t.Helper()
	temp, err := filepath.EvalSymlinks(os.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("TMPDIR", temp)
	j := &stageJourney{env: newWorkItemSourceEnvironment(t), bin: t.TempDir(), home: t.TempDir(), digests: map[string]string{}, managed: map[string]string{}}
	for _, id := range []string{"codex", "claude"} {
		j.managed[id] = filepath.Join(j.home, "managed-"+id)
		if err := os.Mkdir(j.managed[id], 0o700); err != nil {
			t.Fatal(err)
		}
	}
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	image, err := os.ReadFile(self)
	if err != nil {
		t.Fatal(err)
	}
	for _, runtimeID := range []string{"codex", "claude"} {
		path := filepath.Join(j.bin, runtimeID)
		if err := os.WriteFile(path, append(append([]byte(nil), image...), []byte("\nsynthetic-"+runtimeID+"\n")...), 0o700); err != nil {
			t.Fatal(err)
		}
		if j.digests[runtimeID], err = runtimeadapter.ExecutableIdentity(path); err != nil {
			t.Fatal(err)
		}
	}
	j.overlay = j.writeOverlay(t)
	build := t.TempDir()
	j.axiom, j.distributed = filepath.Join(build, "axiom"), filepath.Join(build, "axiom-distributed")
	if output, err := exec.Command("go", "build", "-overlay", j.overlay, "-o", j.axiom, ".").CombinedOutput(); err != nil {
		t.Fatalf("build synthetic test binary: %v %s", err, output)
	}
	if output, err := exec.Command("go", "build", "-o", j.distributed, ".").CombinedOutput(); err != nil {
		t.Fatalf("build distributed binary: %v %s", err, output)
	}
	installLinkedTarget(t, j.env)
	resolved := j.env.installation.Resolve(context.Background(), "external")
	j.repository = resolved.Project.Repositories[0].Path
	j.initRepository(t)
	installStagePlanPolicy(t, j.env.state, workItemSourceProjectID)
	j.publishProfiles(t)
	j.workspace = filepath.Join(t.TempDir(), "workspace")
	if err := os.Mkdir(j.workspace, 0o700); err != nil {
		t.Fatal(err)
	}
	j.environment = []string{"HOME=" + j.home, "PATH=" + j.bin + string(os.PathListSeparator) + "/usr/bin:/bin", "LINGO_PROJECTS_ROOT=" + j.env.root, "LINGO_STATE_ROOT=" + j.env.state, "AXIOM_CODEX_SKILLS_ROOT=" + filepath.Join(j.home, ".codex", "skills"), "AXIOM_GH_BIN=" + filepath.Join(t.TempDir(), "gh-never-executed"), "LANG=C"}
	j.policy = j.seedValidatorPolicy(t)
	j.selectWorkflow(t, concurrency...)
	return j
}

// selectWorkflow authors and selects a schema-2 copy of the reviewed-delivery
// example in which every agent writes the repository: the read-only effort
// boundary is currently unprovable (PM-13), so read-only children never
// dispatch.
func (j *stageJourney) selectWorkflow(t *testing.T, concurrency ...int) {
	t.Helper()
	wire, err := os.ReadFile("../../docs/specifications/007-configurable-workflows/examples/custom-r1.json")
	if err != nil {
		t.Fatal(err)
	}
	legacy, issues := workflowdefinition.Decode(wire)
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	definition := legacy.Definition
	definition.SchemaVersion = 2
	for i := range definition.Stages {
		stage := &definition.Stages[i]
		if stage.ID == "implementation" && len(concurrency) == 1 {
			stage.Concurrency = concurrency[0]
		}
		for k := range stage.Agents {
			a := &stage.Agents[k]
			a.Effort = workflowdefinition.Effort{}
			a.ProfileRef = "profile-codex"
			if len(a.RuntimeConstraints) == 1 && a.RuntimeConstraints[0] == "claude" {
				a.ProfileRef = "profile-claude"
			}
			if !slices.Contains(a.EffectCeilings, "repository-write") {
				a.EffectCeilings = append(a.EffectCeilings, "repository-write")
				a.Capabilities = append(a.Capabilities, "repository-write")
			}
		}
	}
	current, issues := workflowdefinition.Encode(definition)
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	file := filepath.Join(t.TempDir(), "journey-schema2.json")
	if err = os.WriteFile(file, current.Canonical, 0o600); err != nil {
		t.Fatal(err)
	}
	author := j.env.service.(lifecycleService)
	author.projectsRoot = j.env.elsewhere
	if author.portable, err = local.NewPortableStore(j.env.elsewhere); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	apply := func(input cli.ProjectWorkflowInput) cli.Result {
		t.Helper()
		preview := author.ProjectWorkflow(ctx, input)
		if preview.WorkflowAuthoring == nil || preview.WorkflowAuthoring.PreviewDigest == "" {
			t.Fatalf("authoring %s", preview.Category)
		}
		input.PreviewDigest, input.ExpectedRevision, input.AuthorizeLocal = preview.WorkflowAuthoring.PreviewDigest, preview.WorkflowAuthoring.ProjectRevision, true
		out := author.ProjectWorkflow(ctx, input)
		if out.Category != "applied" {
			t.Fatalf("authoring apply %s", out.Category)
		}
		return out
	}
	created := apply(cli.ProjectWorkflowInput{WorkflowRequest: projectapp.WorkflowRequest{Operation: "create", Project: "external"}, File: file})
	apply(cli.ProjectWorkflowInput{WorkflowRequest: projectapp.WorkflowRequest{Operation: "select", Project: "external", Ref: *created.WorkflowAuthoring.Reference}})
}

func (j *stageJourney) writeOverlay(t *testing.T) string {
	t.Helper()
	type entry struct {
		RuntimeID        string `json:"runtimeId"`
		Version          string `json:"version"`
		ExecutableDigest string `json:"executableDigest"`
	}
	entries := []entry{}
	for runtimeID, digest := range j.digests {
		entries = append(entries, entry{runtimeID, stageJourneyVersions[runtimeID], digest})
	}
	sort.Slice(entries, func(a, b int) bool { return entries[a].RuntimeID < entries[b].RuntimeID })
	wire, err := json.Marshal(map[string]any{"schema": runtimeadapter.VersionRegistrySchema, "entries": entries})
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	registry := filepath.Join(dir, "runtime-versions.json")
	if err := os.WriteFile(registry, wire, 0o600); err != nil {
		t.Fatal(err)
	}
	original, err := filepath.Abs("../../internal/runtimeadapter/runtime-versions.json")
	if err != nil {
		t.Fatal(err)
	}
	managedOriginal, err := filepath.Abs("../../internal/runtimeadapter/authpreflight.go")
	if err != nil {
		t.Fatal(err)
	}
	managedSource, err := os.ReadFile(managedOriginal)
	if err != nil {
		t.Fatal(err)
	}
	for original, replacement := range map[string]string{
		"/etc/codex/config.toml":                                        filepath.Join(j.managed["codex"], "config.toml"),
		"/etc/codex/managed_config.toml":                                filepath.Join(j.managed["codex"], "managed_config.toml"),
		"/etc/claude-code/managed-settings.json":                        filepath.Join(j.managed["claude"], "managed-settings.json"),
		"/Library/Application Support/ClaudeCode/managed-settings.json": filepath.Join(j.managed["claude"], "managed-settings.json"),
	} {
		quoted := strconv.Quote(original)
		if !bytes.Contains(managedSource, []byte(quoted)) {
			t.Fatalf("managed location absent from fixture overlay: %s", original)
		}
		managedSource = bytes.ReplaceAll(managedSource, []byte(quoted), []byte(strconv.Quote(replacement)))
	}
	managedPath := filepath.Join(dir, "authpreflight.go")
	if err := os.WriteFile(managedPath, managedSource, 0o600); err != nil {
		t.Fatal(err)
	}
	overlay, err := json.Marshal(map[string]map[string]string{"Replace": {original: registry, managedOriginal: managedPath}})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "overlay.json")
	if err := os.WriteFile(path, overlay, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func (j *stageJourney) initRepository(t *testing.T) {
	t.Helper()
	for _, args := range [][]string{{"init", "-q", "-b", "main"}, {"add", "-A"}, {"-c", "user.name=fixture", "-c", "user.email=fixture@example.invalid", "commit", "-q", "--allow-empty", "-m", "base"}} {
		for name, content := range map[string]string{"README.md": "synthetic\n", "decision.md": stageJourneyDecision} {
			if err := os.WriteFile(filepath.Join(j.repository, name), []byte(content), 0o600); err != nil {
				t.Fatal(err)
			}
		}
		command := exec.Command("git", args...)
		command.Dir = j.repository
		command.Env = []string{"HOME=" + j.home, "PATH=/usr/bin:/bin", "GIT_CONFIG_NOSYSTEM=1"}
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v %s", args, err, output)
		}
	}
}

// publishProfiles replaces the stage-plan v3 configuration with one carrying
// effort declarations for the fake identities.
func (j *stageJourney) publishProfiles(t *testing.T) {
	t.Helper()
	store, err := local.NewRuntimeProfileStore(j.env.state)
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := store.Inspect(context.Background())
	if err != nil || snapshot.V3 == nil {
		t.Fatalf("stage plan profiles: %v", err)
	}
	cfg := *snapshot.V3
	declarations := map[string]runtimeprofile.EffortDeclaration{"codex": runtimeprofiletest.CodexDeclaration(), "claude": runtimeprofiletest.ClaudeDeclaration()}
	for i := range cfg.Runtimes {
		d := declarations[cfg.Runtimes[i].ID]
		d.ExecutableDigest, d.RuntimeVersion = j.digests[cfg.Runtimes[i].ID], stageJourneyVersions[cfg.Runtimes[i].ID]
		cfg.Runtimes[i].EffortDeclarations = []runtimeprofile.EffortDeclaration{d}
	}
	publishRuntimeProfileV3(t, j.env.state, cfg)
}

func (j *stageJourney) seedValidatorPolicy(t *testing.T) string {
	t.Helper()
	store, err := local.NewValidatorPolicyStore(j.env.state)
	if err != nil {
		t.Fatal(err)
	}
	service := stageexecution.ValidatorPolicyService{Store: store}
	d := validatorpolicy.Definition{FormatVersion: 1, ProjectID: workItemSourceProjectID, Commands: []validatorpolicy.Command{{Reference: "synthetic-check", Argv: []string{"/usr/bin/true"}, Env: []string{"LC_ALL=C"}, OutputMax: 4096}}}
	preview, err := service.Preview(d)
	if err != nil {
		t.Fatal(err)
	}
	id, err := service.Apply(context.Background(), stageexecution.ValidatorPolicyApply{Definition: d, PreviewDigest: preview, AuthorizeLocal: true, Source: "human-operator", Actor: "fixture-operator", AuthorityRef: strings.Repeat("a", 64)})
	if err != nil {
		t.Fatal(err)
	}
	return id
}

// startExecution creates the configured Execution in-process before the
// installed run: production workflow start cannot observe Profile proof.
func (j *stageJourney) startExecution(t *testing.T, workItem string) *cli.WorkflowView {
	t.Helper()
	service := j.env.service.(lifecycleService)
	observations := provenObservations(true)
	for i := range observations {
		observations[i].ExecutableDigest = j.digests[observations[i].RuntimeID]
		observations[i].Version = stageJourneyVersions[observations[i].RuntimeID]
	}
	service.RuntimePolicySource = &controlledObservations{base: runtimePolicySource{installation: service.installation, portable: service.portable, stateRoot: j.env.state, runtimes: service.runtimeObservationSources()}, observations: observations}
	ctx := context.Background()
	input := cli.WorkflowInput{Project: "external", Repository: "main", WorkItem: workItem, Role: "engineer", Complexity: "medium", Capabilities: []string{"read"}, Runtime: "codex"}
	preview := service.WorkflowStart(ctx, input)
	if preview.PreviewDigest == "" {
		t.Fatalf("start preview: %+v", preview)
	}
	input.RuntimePreview = preview.PreviewDigest
	started := service.WorkflowStart(ctx, input)
	if started.Workflow == nil || started.Workflow.Binding == nil {
		t.Fatalf("start: %+v", started)
	}
	return started.Workflow
}

func (j *stageJourney) script(t *testing.T, script fakeStageScript) {
	t.Helper()
	wire, err := json.Marshal(script)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(j.home, fakeStageRuntimeFile), wire, 0o600); err != nil {
		t.Fatal(err)
	}
}

func (j *stageJourney) invoke(t *testing.T, binary string, args ...string) (int, stageJourneyEvent, []byte) {
	t.Helper()
	command := exec.Command(binary, append([]string{"--json"}, args...)...)
	command.Env = j.environment
	command.Dir = j.home
	output, err := command.Output()
	code := 0
	if exit, ok := err.(*exec.ExitError); ok {
		code = exit.ExitCode()
	} else if err != nil {
		t.Fatalf("%v: %v", args, err)
	}
	var event stageJourneyEvent
	if err := json.Unmarshal(bytes.TrimSpace(output), &event); err != nil {
		t.Fatalf("%v: invalid JSON %q", args, output)
	}
	return code, event, output
}

func (e stageJourneyEvent) reference(t *testing.T, prefix string) string {
	t.Helper()
	for _, r := range e.References {
		if value, ok := strings.CutPrefix(r, prefix); ok {
			return value
		}
	}
	t.Fatalf("no %s reference in %+v", prefix, e.References)
	return ""
}

func (j *stageJourney) run(t *testing.T, want int, args ...string) stageJourneyEvent {
	t.Helper()
	code, event, output := j.invoke(t, j.axiom, args...)
	if code != want {
		t.Fatalf("%v: code=%d %s", args, code, output)
	}
	return event
}

// seedEvidence stores synthetic capability Evidence for the binding the
// installed binary itself reports, through the canonical Evidence store. It
// is classified synthetic: no Runtime was invoked to produce it.
func (j *stageJourney) seedEvidence(t *testing.T, runtimeID, profileID, variant string, caps ...string) string {
	t.Helper()
	profiles, err := local.NewRuntimeProfileStore(j.env.state)
	if err != nil {
		t.Fatal(err)
	}
	source := runtimeadapter.BindingSource{
		Configuration: canonicalV3Configuration(profiles),
		LookPath:      func(name string) (string, error) { return filepath.Join(j.bin, name), nil },
		Version: func(_ context.Context, id, digest string) (string, error) {
			if j.digests[id] != digest {
				return "", capabilityproof.ErrBlocked
			}
			return stageJourneyVersions[id], nil
		},
		AxiomPath: j.axiom,
		Stage: func(context.Context, capabilityproof.Target) (runtimeadapter.StageContext, error) {
			return runtimeadapter.StageContext{Workspace: filepath.Join(j.workspace, "worktrees"), Inbox: filepath.Join(j.workspace, "inboxes"), ParentEnvironment: j.environment}, nil
		},
		EffortRoots: func(ctx context.Context, target capabilityproof.Target, environment []string) (runtimeadapter.EffortSourceRoots, error) {
			roots, err := (runtimeadapter.StageEffortRoots{Repository: j.repository}).Resolve(ctx, target, environment)
			roots.Managed = j.managed[target.RuntimeID]
			return roots, err
		},
		EffortConsented: true,
	}
	prepared, err := source.PrepareProbe(context.Background(), capabilityproof.Target{RuntimeID: runtimeID, ProfileID: profileID, Variant: variant})
	if err != nil {
		t.Fatalf("observe %s binding: %v", runtimeID, err)
	}
	view := prepared.Current
	store, err := local.NewCapabilityStore(j.env.state)
	if err != nil {
		t.Fatal(err)
	}
	proven := capabilityproof.Result{State: capabilityproof.Proven, Facts: capabilityproof.ProbeFacts{AuthenticationProven: true, BehavioralProbe: true, Verified: true}}
	results := map[string]capabilityproof.Result{}
	for _, c := range caps {
		results[c] = proven
	}
	ref, err := store.Create(context.Background(), capabilityproof.Evidence{SchemaVersion: 1, Binding: view.Binding, Model: view.Model, ObservedAt: time.Now().Add(-time.Minute), ProbePacketDigest: strings.Repeat("5", 64), Capabilities: results, Confinement: capabilityproof.Confinement{Level: "advisory", CanaryIdentityMatched: true, Scope: []string{"workspace", "inbox"}, NotTested: []string{"home", "vendor-home", "tmpdir", "system-temp", "external-network"}}})
	if err != nil {
		t.Fatal(err)
	}
	return ref
}

var stageJourneyCaps = []string{capabilityproof.Read, capabilityproof.RepositoryWrite, capabilityproof.Noninteractive, capabilityproof.AxiomCoordination, capabilityproof.ModelInvocation}

func stageSelector(view *cli.WorkflowView) []string {
	return []string{"--project", "external", "--repository", "main", "--work-item", "github:" + view.WorkItem.Resource + "#" + view.WorkItem.ExternalID, "--execution", view.ExecutionID}
}

func stageRunArgs(view *cli.WorkflowView, stage, plan, policy, workspace string) []string {
	return append(append([]string{"workflow", "stage", "run"}, stageSelector(view)...), "--expected-revision", strconv.FormatUint(view.Revision, 10), "--stage", stage, "--plan", plan, "--validator-policy", policy, "--workspace-root", workspace)
}

func (j *stageJourney) waitStatus(t *testing.T, view *cli.WorkflowView, done func(json.RawMessage) bool) json.RawMessage {
	t.Helper()
	deadline := time.Now().Add(20 * time.Second)
	for {
		_, event, output := j.invoke(t, j.axiom, append([]string{"workflow", "stage", "status"}, stageSelector(view)...)...)
		if done(event.Stage) {
			return event.Stage
		}
		if time.Now().After(deadline) {
			log, _ := os.ReadFile(filepath.Join(j.home, "fake-stage-invocations.jsonl"))
			t.Fatalf("stage status did not converge: %s\nfake log: %s", output, log)
		}
		time.Sleep(200 * time.Millisecond)
	}
}

func writeUnit(key string, paths []string, target string) executiongraph.WorkUnit {
	return stagePlanUnit(key, paths, executiongraph.Effect{Kind: "repository-write", Target: target})
}

func (j *stageJourney) invocations(t *testing.T) []fakeStageInvocation {
	t.Helper()
	wire, err := os.ReadFile(filepath.Join(j.home, "fake-stage-invocations.jsonl"))
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	var out []fakeStageInvocation
	for _, line := range bytes.Split(bytes.TrimSpace(wire), []byte("\n")) {
		var r fakeStageInvocation
		if len(line) > 0 && json.Unmarshal(line, &r) == nil {
			out = append(out, r)
		}
	}
	return out
}

type singleStatus struct {
	Single struct {
		Binding struct {
			Workspace string `json:"workspace"`
		} `json:"binding"`
		Attempts []struct {
			AttemptID            string `json:"attemptId"`
			Number               int    `json:"number"`
			Status               string `json:"status"`
			ClassificationReason string `json:"classificationReason"`
			QualifyingRecordRef  string `json:"qualifyingRecordRef"`
			ExitCode             int    `json:"exitCode"`
		} `json:"attempts"`
	} `json:"single"`
	Next string `json:"next"`
}

func decodeSingle(t *testing.T, wire json.RawMessage) singleStatus {
	t.Helper()
	var v singleStatus
	if err := json.Unmarshal(wire, &v); err != nil {
		t.Fatalf("single status %s: %v", wire, err)
	}
	return v
}

func latestSettled(stage json.RawMessage) bool {
	var v singleStatus
	if json.Unmarshal(stage, &v) != nil || len(v.Single.Attempts) == 0 {
		return false
	}
	status := v.Single.Attempts[len(v.Single.Attempts)-1].Status
	return status != "running" && status != ""
}

// T36 single-agent journey on the installed CLI: dispatch preview, both
// authorities, detached supervisor, fake Codex child, ingested result and
// persisted inspection. Every scenario is synthetic.
func TestInstalledSyntheticSingleStageJourney(t *testing.T) {
	j := newStageJourney(t)
	j.run(t, 0, "runtime", "codex", "install")
	j.run(t, 0, "runtime", "claude", "install")
	view := j.startExecution(t, "github:owner/repo#7")
	write := executiongraph.Effect{Kind: "repository-write", Target: "docs/intake.md"}
	plan := writeStagePlanDocument(t, []executiongraph.WorkUnit{writeUnit("owner", []string{"docs"}, "docs/intake.md")}, []executiongraph.Effect{write})
	args := stageRunArgs(view, "intake", plan, j.policy, j.workspace)

	if refused := j.run(t, 1, append(args, "--authorize-local")...); refused.Category != "runtime_unresolvable" {
		t.Fatalf("dispatch preview without Evidence: %+v", refused)
	}
	j.seedEvidence(t, "codex", "profile-codex", capabilityproof.RepositoryWriteVariant, stageJourneyCaps...)
	if refused := j.run(t, 1, args...); refused.Category != "runtime_unresolvable" {
		t.Fatalf("preview without effort-inspection consent: %+v", refused)
	}
	escalated := writeStagePlanDocument(t, []executiongraph.WorkUnit{writeUnit("owner", []string{"docs"}, "docs/intake.md")}, nil)
	if denied := j.run(t, 1, append(stageRunArgs(view, "intake", escalated, j.policy, j.workspace), "--authorize-local")...); denied.Status != "denied_authority" || denied.Category != "authority_denied" {
		t.Fatalf("authority escalation: %+v", denied)
	}
	preview := j.run(t, 0, append(args, "--authorize-local")...)
	digest := preview.reference(t, "stage-preview:")
	if code, distributed, _ := j.invoke(t, j.distributed, append(args, "--authorize-local")...); code == 0 || len(distributed.References) != 0 || distributed.Category != "runtime_unresolvable" {
		t.Fatalf("distributed binary admitted synthetic identities: %+v", distributed)
	}
	if code, distributed, _ := j.invoke(t, j.distributed, "runtime", "codex", "capability", "show", "--profile", "profile-codex", "--variant", "repository-write", "--repository-root", j.repository, "--workspace-root", j.workspace, "--authorize-local"); code == 0 || distributed.Category != "capability_unknown" {
		t.Fatalf("distributed binary observed synthetic Evidence: %+v", distributed)
	}
	j.script(t, fakeStageScript{Agents: map[string][]fakeStageStep{"owner": {
		{Op: "write", Path: "docs/intake.md", Content: "synthetic intake\n"},
		{Op: "publish", Kind: "progress", Fields: []string{"phase=writing", "percent=50"}},
		{Op: "publish", Kind: "result", Fields: []string{"outcome=succeeded", "summary=synthetic"}},
	}}})
	for _, refused := range [][]string{
		append(args, "--preview-digest", digest, "--authorize-local"),
		append(args, "--preview-digest", digest, "--authorize-external"),
		append(args, "--preview-digest", strings.Repeat("0", 64), "--authorize-local", "--authorize-external"),
	} {
		if event := j.run(t, 1, refused...); event.Status != "denied_authority" && event.Category != "runtime_unresolvable" {
			t.Fatalf("%v: %+v", refused[len(refused)-3:], event)
		}
	}
	if len(j.invocations(t)) != 0 {
		t.Fatal("refused dispatch started a Runtime")
	}
	dispatched := j.run(t, 0, append(args, "--preview-digest", digest, "--authorize-local", "--authorize-external")...)
	if dispatched.Category != "stage_dispatched" || len(dispatched.References) == 0 {
		t.Fatalf("dispatch %+v", dispatched)
	}
	status := decodeSingle(t, j.waitStatus(t, view, latestSettled))
	if len(status.Single.Attempts) != 1 || status.Single.Attempts[0].Status != "succeeded" || status.Single.Attempts[0].ClassificationReason != "result_recorded" || status.Single.Attempts[0].QualifyingRecordRef == "" || status.Next != "advance_execution" {
		t.Fatalf("synthetic single status %+v", status)
	}
	calls := j.invocations(t)
	if len(calls) != 1 || calls[0].Runtime != "codex" || calls[0].Agent != "owner" || calls[0].Attempt != status.Single.Attempts[0].AttemptID || calls[0].CWD != status.Single.Binding.Workspace || !slices.Contains(calls[0].Argv, "local-codex-model") {
		t.Fatalf("fake invocation %+v", calls)
	}
	for _, key := range calls[0].Environment {
		if key == "AXIOM_GH_BIN" || key == "LANG_UNUSED" {
			t.Fatalf("child inherited %s", key)
		}
	}
	if !slices.Contains(calls[0].Environment, "AXIOM_ATTEMPT_ID") {
		t.Fatalf("child correlation missing: %v", calls[0].Environment)
	}
	if wire, err := os.ReadFile(filepath.Join(status.Single.Binding.Workspace, "docs", "intake.md")); err != nil || string(wire) != "synthetic intake\n" {
		t.Fatalf("child workspace effect: %q %v", wire, err)
	}
	if _, err := os.Lstat(filepath.Join(j.repository, "docs")); !os.IsNotExist(err) {
		t.Fatal("child wrote the registered repository")
	}
	records := j.run(t, 0, append(append([]string{"workflow", "coordination", "list"}, stageSelector(view)...), "--child", view.ExecutionID)...)
	if !bytes.Contains(records.Stage, []byte(`"progress"`)) || !bytes.Contains(records.Stage, []byte(`"result"`)) {
		t.Fatalf("canonical records %s", records.Stage)
	}
	integrate := append(append([]string{"workflow", "stage", "integrate"}, stageSelector(view)...), "--expected-revision", strconv.FormatUint(view.Revision, 10), "--stage", "intake", "--validator-policy", j.policy)
	if refused := j.run(t, 1, integrate...); refused.Category != "single_stage_operation_unsupported" {
		t.Fatalf("single integrate %+v", refused)
	}
	if again := j.run(t, 1, append(args, "--preview-digest", digest, "--authorize-local", "--authorize-external")...); again.Category == "stage_dispatched" {
		t.Fatalf("succeeded single stage dispatched twice: %+v", again)
	}
	if len(j.invocations(t)) != 1 {
		t.Fatal("re-run started another Runtime")
	}
}

// A failed single-agent attempt is retried only with an explicit workspace
// disposition; reset is applied by the fenced supervisor before the new
// attempt, so the failed attempt's file is gone.
func TestInstalledSyntheticSingleStageRetryReset(t *testing.T) {
	j := newStageJourney(t)
	j.run(t, 0, "runtime", "codex", "install")
	view := j.startExecution(t, "github:owner/repo#8")
	unit := writeUnit("owner", []string{"docs"}, "docs/intake.md")
	unit.Controls.MaximumAttempts = 2
	plan := writeStagePlanDocument(t, []executiongraph.WorkUnit{unit}, []executiongraph.Effect{{Kind: "repository-write", Target: "docs/intake.md"}})
	args := stageRunArgs(view, "intake", plan, j.policy, j.workspace)
	j.seedEvidence(t, "codex", "profile-codex", capabilityproof.RepositoryWriteVariant, stageJourneyCaps...)
	j.script(t, fakeStageScript{
		Agents:   map[string][]fakeStageStep{"owner": {{Op: "write", Path: "docs/intake.md", Content: "second\n"}, {Op: "publish", Kind: "result", Fields: []string{"outcome=succeeded"}}}},
		Attempts: map[string][]fakeStageStep{"owner#1": {{Op: "write", Path: "docs/stale.md", Content: "first\n"}, {Op: "exit", Code: 3}}},
	})
	digest := j.run(t, 0, append(args, "--authorize-local")...).reference(t, "stage-preview:")
	j.run(t, 0, append(args, "--preview-digest", digest, "--authorize-local", "--authorize-external")...)
	first := decodeSingle(t, j.waitStatus(t, view, latestSettled))
	if len(first.Single.Attempts) != 1 || first.Single.Attempts[0].Status != "failed" || first.Single.Attempts[0].ClassificationReason != "process_failed" || first.Single.Attempts[0].ExitCode != 3 {
		t.Fatalf("failed attempt %+v", first)
	}
	workspace := first.Single.Binding.Workspace
	if _, err := os.Stat(filepath.Join(workspace, "docs", "stale.md")); err != nil {
		t.Fatalf("failed attempt effect: %v", err)
	}
	retry := append(append([]string{"workflow", "stage", "retry"}, stageSelector(view)...), "--expected-revision", strconv.FormatUint(view.Revision, 10), "--stage", "intake", "--plan", plan, "--validator-policy", j.policy, "--child", view.ExecutionID, "--authorize-local")
	retryDigest := j.run(t, 0, retry...).reference(t, "stage-preview:")
	if refused := j.run(t, 1, append(retry, "--preview-digest", retryDigest, "--authorize-external")...); refused.Category != "workspace_disposition_required" {
		t.Fatalf("retry without disposition %+v", refused)
	}
	if refused := j.run(t, 1, append(retry, "--preview-digest", retryDigest, "--disposition", "reset")...); refused.Status != "denied_authority" {
		t.Fatalf("retry without external authority %+v", refused)
	}
	if len(j.invocations(t)) != 1 {
		t.Fatal("refused retry started a Runtime")
	}
	j.run(t, 0, append(retry, "--preview-digest", retryDigest, "--authorize-external", "--disposition", "reset")...)
	second := decodeSingle(t, j.waitStatus(t, view, func(stage json.RawMessage) bool {
		return latestSettled(stage) && len(decodeSingle(t, stage).Single.Attempts) == 2
	}))
	if last := second.Single.Attempts[1]; last.Status != "succeeded" || last.Number != 2 {
		t.Fatalf("retried attempt %+v", second)
	}
	if _, err := os.Lstat(filepath.Join(workspace, "docs", "stale.md")); !os.IsNotExist(err) {
		t.Fatalf("reset preserved the failed attempt's file: %v", err)
	}
	if wire, err := os.ReadFile(filepath.Join(workspace, "docs", "intake.md")); err != nil || string(wire) != "second\n" {
		t.Fatalf("retried effect %q %v", wire, err)
	}
	if again := j.run(t, 1, append(retry, "--preview-digest", retryDigest, "--authorize-external", "--disposition", "reset")...); again.Category == "stage_retry_dispatched" {
		t.Fatal("retry receipt reused")
	}
}

type workflowEvent struct {
	Workflow *cli.WorkflowView `json:"workflow"`
}

func (j *stageJourney) advanceTo(t *testing.T, view *cli.WorkflowView, stage string) *cli.WorkflowView {
	t.Helper()
	sum := sha256.Sum256([]byte(stageJourneyDecision))
	digest := hex.EncodeToString(sum[:])
	call := func(args ...string) *cli.WorkflowView {
		t.Helper()
		code, _, output := j.invoke(t, j.axiom, args...)
		var event workflowEvent
		if code != 0 || json.Unmarshal(bytes.TrimSpace(output), &event) != nil || event.Workflow == nil {
			t.Fatalf("%v: code=%d %s", args, code, output)
		}
		return event.Workflow
	}
	selector := stageSelector(view)
	fact := func(v *cli.WorkflowView) *cli.WorkflowView {
		if action := v.GateAction; action != nil && action.Operation == "fact" {
			reference := "specification:decision.md:" + digest
			if action.Fact == "implementation_authority" {
				reference = "plan:decision.md:" + digest
			}
			return call(append(append([]string{"workflow", "fact"}, selector...), "--expected-revision", strconv.FormatUint(v.Revision, 10), "--fact", strings.ReplaceAll(string(action.Fact), "_", "-"), "--active", "--actor", "maintainer", "--reference", reference, "--authorize-local")...)
		}
		return v
	}
	for count := 0; view.CurrentGate != stage && count < 16; count++ {
		view = fact(view)
		file := publishConfiguredOutput(t, j.env.state, view)
		view = call(append(append([]string{"workflow", "advance"}, selector...), "--expected-revision", strconv.FormatUint(view.Revision, 10), "--gate", view.CurrentGate, "--outcome", "pass", "--stage-result", file)...)
	}
	if view.CurrentGate != stage {
		t.Fatalf("did not reach %s: %s", stage, view.CurrentGate)
	}
	return fact(view)
}

func graphSettled(stage json.RawMessage) bool {
	return bytes.Count(stage, []byte(`"status":"succeeded"`)) >= 2 || bytes.Contains(stage, []byte(`"status":"failed"`)) || bytes.Contains(stage, []byte(`"status":"unknown"`)) || bytes.Contains(stage, []byte(`"status":"blocked"`))
}

func TestInstalledSyntheticGraphStageJourney(t *testing.T) {
	j := newStageJourney(t)
	j.run(t, 0, "runtime", "codex", "install")
	j.run(t, 0, "runtime", "claude", "install")
	saveTargetLink(t, j.env.state, workItemSourceProjectID, "9", "owner/repo")
	view := j.advanceTo(t, j.startExecution(t, "github:owner/repo#9"), "implementation")
	impl := executiongraph.Effect{Kind: "repository-write", Target: "src/impl.go"}
	review := executiongraph.Effect{Kind: "repository-write", Target: "docs/review.md"}
	integrated := executiongraph.Effect{Kind: "integration", Target: "src/impl.go"}
	integratedReview := executiongraph.Effect{Kind: "integration", Target: "docs/review.md"}
	plan := writeStagePlanDocument(t, []executiongraph.WorkUnit{
		stagePlanUnit("implementer", []string{"src"}, impl),
		stagePlanUnit("reviewer", []string{"docs"}, review),
		stagePlanUnit("integrator", []string{"src", "docs"}, impl, integrated, integratedReview),
	}, []executiongraph.Effect{impl, review, integrated, integratedReview})
	args := stageRunArgs(view, "implementation", plan, j.policy, j.workspace)
	j.seedEvidence(t, "codex", "profile-codex", capabilityproof.RepositoryWriteVariant, stageJourneyCaps...)
	if refused := j.run(t, 1, append(args, "--authorize-local")...); refused.Category != "runtime_unresolvable" {
		t.Fatalf("graph preview without Claude Evidence: %+v", refused)
	}
	j.seedEvidence(t, "claude", "profile-claude", capabilityproof.RepositoryWriteVariant, stageJourneyCaps...)
	selectors := stageSelector(view)
	j.script(t, fakeStageScript{Selectors: selectors, Agents: map[string][]fakeStageStep{
		"implementer": {
			{Op: "write", Path: "src/impl.go", Content: "package src\n"},
			{Op: "publish", Kind: "question_request", Fields: []string{"question=Is the API stable?", "response_contract=yes-or-no"}},
			{Op: "publish", Kind: "contract_proposal", Fields: []string{"contract=api-v1", "scope=src"}},
			{Op: "await", Kind: "answer", Millis: 20000},
			{Op: "publish", Kind: "progress", Fields: []string{"phase=implemented", "percent=100"}},
			{Op: "publish", Kind: "artifact_publication", Fields: []string{"artifact_id=impl", "artifact_digest=" + strings.Repeat("b", 64), "media_type=text/plain"}},
			{Op: "publish", Kind: "result", Fields: []string{"outcome=succeeded", "summary=implemented"}},
		},
		"reviewer": {
			{Op: "await", Kind: "question_request", Millis: 20000},
			{Op: "publish", Kind: "answer", Fields: []string{"answer=yes", "question_reference={captured}"}},
			{Op: "await", Kind: "contract_proposal", Millis: 20000},
			{Op: "publish", Kind: "contract_acceptance", Fields: []string{"decision=accepted", "contract_reference={captured}"}},
			{Op: "publish", Kind: "blocker", Fields: []string{"code=review_pending", "required_action=finish-review"}},
			{Op: "write", Path: "docs/review.md", Content: "reviewed\n"},
			{Op: "publish", Kind: "dependency_resolution", Fields: []string{"dependency=api", "resolution=stable"}},
			{Op: "publish", Kind: "result", Fields: []string{"outcome=succeeded", "summary=reviewed"}},
		},
	}})
	digest := j.run(t, 0, append(args, "--authorize-local")...).reference(t, "stage-preview:")
	j.run(t, 0, append(args, "--preview-digest", digest, "--authorize-local", "--authorize-external")...)
	status := j.waitStatus(t, view, graphSettled)
	var persisted stageexecution.StatusResult
	if err := json.Unmarshal(status, &persisted); err != nil || persisted.Graph.Successor == nil {
		t.Fatalf("graph status: %s %v", status, err)
	}
	g := persisted.Graph.Successor
	var completed []executiongraph.SuccessorAttempt
	for _, child := range g.Children {
		if child.Envelope.IntegrationOwner {
			if len(child.Attempts) != 0 || persisted.Blockers[child.ExecutionID] != "integration_pending" {
				t.Fatalf("integration ran before operator authority: %+v", child)
			}
			continue
		}
		if len(child.Attempts) != 1 || child.Attempts[0].Status != "succeeded" || child.Attempts[0].QualifyingRecordRef == "" {
			t.Fatalf("graph child did not succeed: %+v", child)
		}
		completed = append(completed, child.Attempts[0])
	}
	if len(completed) != 2 || !completed[0].StartedAt.Before(*completed[1].FinishedAt) || !completed[1].StartedAt.Before(*completed[0].FinishedAt) {
		t.Fatalf("independent children did not overlap: %+v", completed)
	}
	calls := j.invocations(t)
	if len(calls) != 2 || calls[0].Runtime == calls[1].Runtime {
		t.Fatalf("expected two cross-Runtime FAKE calls: %+v", calls)
	}
	records := j.run(t, 0, append([]string{"workflow", "coordination", "list"}, selectors...)...)
	for _, kind := range []string{"question_request", "answer", "progress", "contract_proposal", "artifact_publication", "dependency_resolution", "contract_acceptance", "blocker", "result"} {
		if !bytes.Contains(records.Stage, []byte(`"kind":"`+kind+`"`)) {
			t.Fatalf("missing canonical %s: %s", kind, records.Stage)
		}
	}
	integrate := append(append([]string{"workflow", "stage", "integrate"}, selectors...), "--expected-revision", strconv.FormatUint(view.Revision, 10), "--stage", "implementation", "--validator-policy", j.policy)
	idigest := j.run(t, 0, integrate...).reference(t, "stage-preview:")
	applied := j.run(t, 0, append(integrate, "--preview-digest", idigest, "--authorize-local")...)
	var result stageexecution.IntegrateResult
	if err := json.Unmarshal(applied.Stage, &result); err != nil || result.Rollup == nil || result.Rollup.Status != "success" || result.OutcomeRef == "" || result.ResultTree == "" || len(result.Validations) != 1 || result.Validations[0].ExitCode != 0 {
		t.Fatalf("integration did not persist validated outcome: %s %v", applied.Stage, err)
	}
	for _, child := range g.Children {
		if !child.Envelope.IntegrationOwner {
			continue
		}
		for path, want := range map[string]string{"src/impl.go": "package src\n", "docs/review.md": "reviewed\n"} {
			if wire, err := os.ReadFile(filepath.Join(child.Envelope.Workspace, path)); err != nil || string(wire) != want {
				t.Fatalf("integrated %s: %q %v", path, wire, err)
			}
		}
	}
	if again := j.run(t, 1, append(integrate, "--preview-digest", idigest, "--authorize-local")...); again.Category == "stage_integrated" {
		t.Fatal("integration repeated its effects")
	}
}

func TestInstalledSyntheticGraphConflictRetryReset(t *testing.T) {
	j := newStageJourney(t, 1)
	j.run(t, 0, "runtime", "codex", "install")
	j.run(t, 0, "runtime", "claude", "install")
	saveTargetLink(t, j.env.state, workItemSourceProjectID, "10", "owner/repo")
	view := j.advanceTo(t, j.startExecution(t, "github:owner/repo#10"), "implementation")
	write := executiongraph.Effect{Kind: "repository-write", Target: "src/shared.go"}
	integration := executiongraph.Effect{Kind: "integration", Target: "src/shared.go"}
	a, b := stagePlanUnit("implementer", []string{"src"}, write), stagePlanUnit("reviewer", []string{"src"}, write)
	a.Controls.MaximumAttempts = 2
	plan := writeStagePlanDocument(t, []executiongraph.WorkUnit{a, b, stagePlanUnit("integrator", []string{"src"}, write, integration)}, []executiongraph.Effect{write, integration})
	args := stageRunArgs(view, "implementation", plan, j.policy, j.workspace)
	j.seedEvidence(t, "codex", "profile-codex", capabilityproof.RepositoryWriteVariant, stageJourneyCaps...)
	j.seedEvidence(t, "claude", "profile-claude", capabilityproof.RepositoryWriteVariant, stageJourneyCaps...)
	j.script(t, fakeStageScript{
		Agents: map[string][]fakeStageStep{
			"implementer": {{Op: "write", Path: "src/shared.go", Content: "implementation\n"}, {Op: "publish", Kind: "result", Fields: []string{"outcome=succeeded"}}},
			"reviewer":    {{Op: "write", Path: "src/shared.go", Content: "review\n"}, {Op: "publish", Kind: "result", Fields: []string{"outcome=succeeded"}}},
		},
		Attempts: map[string][]fakeStageStep{"implementer#1": {{Op: "write", Path: "src/stale.go", Content: "stale\n"}, {Op: "sleep", Millis: 300}, {Op: "exit", Code: 7}}},
	})
	digest := j.run(t, 0, append(args, "--authorize-local")...).reference(t, "stage-preview:")
	j.run(t, 0, append(args, "--preview-digest", digest, "--authorize-local", "--authorize-external")...)
	settled := func(stage json.RawMessage) bool {
		var status stageexecution.StatusResult
		if json.Unmarshal(stage, &status) != nil || status.Graph.Successor == nil {
			return false
		}
		for _, child := range status.Graph.Successor.Children {
			if child.Envelope.IntegrationOwner {
				continue
			}
			if len(child.Attempts) == 0 || child.Attempts[len(child.Attempts)-1].Status == "running" {
				return false
			}
		}
		return true
	}
	var first stageexecution.StatusResult
	if err := json.Unmarshal(j.waitStatus(t, view, graphSettled), &first); err != nil {
		t.Fatal(err)
	}
	var failed executiongraph.SuccessorChild
	var attempts []executiongraph.SuccessorAttempt
	for _, child := range first.Graph.Successor.Children {
		if child.Envelope.IntegrationOwner {
			continue
		}
		if len(child.Attempts) != 0 {
			attempts = append(attempts, child.Attempts[0])
		}
		if child.NodeKey == "implementation:implementer" {
			failed = child
		}
	}
	if len(attempts) != 1 {
		t.Fatalf("dependent child ran after failed prerequisite: %+v", attempts)
	}
	if failed.ExecutionID == "" || failed.Attempts[0].Status != "failed" {
		t.Fatalf("missing failed child: %+v", failed)
	}
	retry := append(append([]string{"workflow", "stage", "retry"}, stageSelector(view)...), "--expected-revision", strconv.FormatUint(view.Revision, 10), "--stage", "implementation", "--plan", plan, "--validator-policy", j.policy, "--child", failed.ExecutionID, "--authorize-local")
	rdigest := j.run(t, 0, retry...).reference(t, "stage-preview:")
	j.run(t, 0, append(retry, "--preview-digest", rdigest, "--authorize-external", "--disposition", "reset")...)
	var after stageexecution.StatusResult
	if err := json.Unmarshal(j.waitStatus(t, view, func(stage json.RawMessage) bool {
		var status stageexecution.StatusResult
		if json.Unmarshal(stage, &status) != nil || status.Graph.Successor == nil {
			return false
		}
		for _, child := range status.Graph.Successor.Children {
			if child.ExecutionID == failed.ExecutionID {
				return len(child.Attempts) == 2 && child.Attempts[1].Status != "running"
			}
		}
		return false
	}), &after); err != nil {
		t.Fatal(err)
	}
	for _, child := range after.Graph.Successor.Children {
		if child.ExecutionID == failed.ExecutionID && (len(child.Attempts) != 2 || child.Attempts[1].Status != "succeeded") {
			t.Fatalf("retry failed: %+v", child)
		}
	}
	resumeDigest := j.run(t, 0, append(args, "--authorize-local")...).reference(t, "stage-preview:")
	j.run(t, 0, append(args, "--preview-digest", resumeDigest, "--authorize-local", "--authorize-external")...)
	if err := json.Unmarshal(j.waitStatus(t, view, settled), &after); err != nil {
		t.Fatal(err)
	}
	var implemented, reviewed executiongraph.SuccessorAttempt
	for _, child := range after.Graph.Successor.Children {
		if child.NodeKey == "implementation:implementer" {
			implemented = child.Attempts[1]
		}
		if child.NodeKey == "implementation:reviewer" {
			reviewed = child.Attempts[0]
		}
	}
	if implemented.FinishedAt == nil || reviewed.StartedAt == nil || reviewed.StartedAt.Before(*implemented.FinishedAt) {
		t.Fatalf("conflicting children overlapped: %+v %+v", implemented, reviewed)
	}
	if _, err := os.Lstat(filepath.Join(failed.Envelope.Workspace, "src/stale.go")); !os.IsNotExist(err) {
		t.Fatalf("graph reset preserved stale file: %v", err)
	}
	integrate := append(append([]string{"workflow", "stage", "integrate"}, stageSelector(view)...), "--expected-revision", strconv.FormatUint(view.Revision, 10), "--stage", "implementation", "--validator-policy", j.policy)
	if refused := j.run(t, 1, integrate...); refused.Category != "integration_blocked" {
		t.Fatalf("conflicting result trees admitted: %+v", refused)
	}
	if len(j.invocations(t)) != 3 {
		t.Fatalf("unexpected calls: %+v", j.invocations(t))
	}
}
