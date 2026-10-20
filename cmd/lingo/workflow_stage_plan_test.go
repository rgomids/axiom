package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/rgomids/axiom/internal/cli"
	"github.com/rgomids/axiom/internal/executiongraph"
	"github.com/rgomids/axiom/internal/local"
	"github.com/rgomids/axiom/internal/manifest"
	"github.com/rgomids/axiom/internal/project"
	"github.com/rgomids/axiom/internal/projectapp"
	"github.com/rgomids/axiom/internal/runtimeapplication"
	"github.com/rgomids/axiom/internal/runtimeprofile"
	"github.com/rgomids/axiom/internal/workflowcompiler"
	"github.com/rgomids/axiom/internal/workflowdefinition"
)

type stagePlanEvent struct {
	Status           string                         `json:"status"`
	Result           string                         `json:"result"`
	Next             string                         `json:"next"`
	Category         string                         `json:"category"`
	ConfirmedEffects []string                       `json:"confirmedEffects"`
	Execution        *workflowcompiler.ExecutionRef `json:"executionRef"`
	Conditions       []string                       `json:"conditions"`
	Plan             *workflowcompiler.StagePlan    `json:"plan"`
	Workflow         *cli.WorkflowView              `json:"workflow"`
	PreviewDigest    string                         `json:"previewDigest"`
}

// writeStagePlanDocument writes one approved Plan document: one bounded work
// unit per declared agent, keyed by agent ID, with explicit scope and effects.
func writeStagePlanDocument(t *testing.T, units []executiongraph.WorkUnit, ceiling []executiongraph.Effect) string {
	t.Helper()
	plan := executiongraph.ApprovedPlan{Approved: true, PlanRevision: "stage-plan-1", PlanDigest: strings.Repeat("c", 64), MaximumNodes: len(units), Work: units}
	wire, err := json.MarshalIndent(workflowcompiler.PlanDocument{FormatVersion: 1, Plan: plan, AuthorityCeiling: ceiling}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "plan.json")
	if err := os.WriteFile(path, wire, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func stagePlanUnit(key string, paths []string, effects ...executiongraph.Effect) executiongraph.WorkUnit {
	return executiongraph.WorkUnit{Key: key, Scope: executiongraph.Scope{ProjectID: workItemSourceProjectID, RepositoryKey: "main", Paths: paths}, Effects: effects, Controls: executiongraph.ExecutionControls{Timeout: 10 * time.Minute, MaximumAttempts: 1}}
}

// The real executable uses the production Runtime observer, which proves only
// Axiom skill integration. Stage planning must therefore fail closed for a
// stage needing `read`, without fallback, mutation or invented capability.
func TestWorkflowStagePlanExecutableFailsClosedAndNeverMutates(t *testing.T) {
	env := newWorkItemSourceEnvironment(t)
	installLinkedTarget(t, env)
	installTestRuntimePolicy(t, env.state, workItemSourceProjectID, "codex")
	binary := filepath.Join(t.TempDir(), "axiom")
	if output, err := exec.Command("go", "build", "-o", binary, ".").CombinedOutput(); err != nil {
		t.Fatalf("build: %v %s", err, output)
	}
	runtimeDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(runtimeDir, "codex"), []byte("#!/bin/sh\nexit 99\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	home := t.TempDir()
	environment := append(os.Environ(), "PATH="+runtimeDir, "AXIOM_CODEX_SKILLS_ROOT="+filepath.Join(t.TempDir(), "skills"), "HOME="+home)
	run := func(code int, args ...string) ([]byte, stagePlanEvent) {
		t.Helper()
		command := exec.Command(binary, args...)
		command.Env = environment
		wire, err := command.Output()
		actual := 0
		if exit, ok := err.(*exec.ExitError); ok {
			actual = exit.ExitCode()
		} else if err != nil {
			t.Fatal(err)
		}
		var value stagePlanEvent
		if actual != code || len(args) > 0 && args[0] == "--json" && json.Unmarshal(bytes.TrimSpace(wire), &value) != nil {
			t.Fatalf("%v code=%d: %s", args, actual, wire)
		}
		t.Logf("r1-evidence {\"scenario\":\"F\",\"canonical\":{\"category\":%q}}", value.Category)
		return wire, value
	}
	run(0, "--json", "runtime", "codex", "install")
	selector := []string{"--project", "external", "--repository", "main", "--work-item", "github:owner/repo#7"}
	start := append(append([]string{"--json", "workflow", "start"}, selector...), "--role", "implementation", "--complexity", "high", "--capabilities", "axiom-skills")
	// Production composition injects no Profile proof, so the public start
	// fails closed before any Execution exists (Specification 007 PM-7).
	_, preview := run(1, start...)
	_, refusedStart := run(1, append(start, "--runtime-preview", preview.PreviewDigest)...)
	if preview.PreviewDigest == "" || refusedStart.Workflow != nil {
		t.Fatalf("unproven current start %+v", refusedStart)
	}
	seeded := seedHistoricalWorkflow(t, env.service.(lifecycleService), cli.WorkflowInput{Project: "external", Repository: "main", Provider: "github", ProviderRepository: "owner/repo", ExternalID: "7", Runtime: "codex", RuntimePreview: preview.PreviewDigest})
	started := stagePlanEvent{Workflow: seeded}
	if started.Workflow == nil || started.Workflow.Binding == nil {
		t.Fatal("missing configured Execution")
	}
	execution := started.Workflow.ExecutionID
	planFile := writeStagePlanDocument(t, []executiongraph.WorkUnit{stagePlanUnit("owner", []string{"docs"})}, nil)
	plan := append(append([]string{"workflow", "stage", "plan"}, selector...), "--execution", execution, "--expected-revision", "1", "--stage", "intake", "--plan", planFile)
	before := snapshotTrees(t, env.root, env.state, env.elsewhere)

	_, refused := run(1, append([]string{"--json"}, plan...)...)
	if refused.Status != "validation_failure" || refused.Category != "legacy_execution_contract" || refused.Plan != nil || len(refused.ConfirmedEffects) != 0 || refused.Execution == nil || refused.Execution.ExecutionID != execution {
		t.Fatalf("production observer claimed an unproven capability: %+v", refused)
	}
	human, _ := run(1, plan...)
	if !bytes.Contains(human, []byte("legacy_execution_contract")) || bytes.HasPrefix(bytes.TrimSpace(human), []byte("{")) {
		t.Fatalf("default output is not the human rendering: %s", human)
	}
	for _, tc := range []struct {
		args     []string
		status   string
		category string
	}{
		{[]string{"--expected-revision", "2"}, "denied_authority", "stale_execution_revision"},
		{[]string{"--stage", "specification"}, "validation_failure", "stage_prerequisite_missing"},
		{[]string{"--stage", "unknown-stage"}, "validation_failure", "stage_not_found"},
		{[]string{"--execution", "00000000-0000-4000-8000-000000000000"}, "validation_failure", "execution_selector_conflict"},
	} {
		args := append([]string{"--json"}, plan...)
		for i := 0; i < len(tc.args); i += 2 {
			for j := range args {
				if args[j] == tc.args[i] {
					args[j+1] = tc.args[i+1]
				}
			}
		}
		_, got := run(1, args...)
		if got.Status != tc.status || got.Category != tc.category || got.Plan != nil {
			t.Fatalf("%v: %+v", tc.args, got)
		}
		if tc.category == "stage_prerequisite_missing" && (len(got.Conditions) != 1 || got.Conditions[0] != "missing_input:source") {
			t.Fatalf("missing input not reported: %+v", got.Conditions)
		}
	}
	invalid := filepath.Join(t.TempDir(), "invalid.json")
	if err := os.WriteFile(invalid, []byte(`{"formatVersion":1,"plan":{},"authorityCeiling":[],"grant":true}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, got := run(1, append(append([]string{"--json"}, plan[:len(plan)-1]...), invalid)...); got.Category != "legacy_execution_contract" {
		t.Fatalf("legacy planning bypassed immutable snapshot guard: %+v", got)
	}
	if _, got := run(1, append([]string{"--json"}, plan[:len(plan)-2]...)...); got.Status != "validation_failure" || got.Result != "Stage plan input is invalid" || !strings.Contains(got.Next, "axiom workflow stage plan --help") {
		t.Fatalf("missing --plan accepted: %+v", got)
	}
	if !bytes.Equal(before, snapshotTrees(t, env.root, env.state, env.elsewhere)) {
		t.Fatal("stage planning mutated local state")
	}
	help, _ := run(0, "workflow", "stage", "plan", "--help")
	for _, want := range []string{"--expected-revision", "--stage", "--plan", "--execution", "never dispatches"} {
		if !bytes.Contains(help, []byte(want)) {
			t.Fatalf("help lacks %s: %s", want, help)
		}
	}
	group, _ := run(0, "workflow", "--help")
	if !bytes.Contains(group, []byte("stage")) {
		t.Fatalf("workflow help does not discover stage planning: %s", group)
	}
}

// installStagePlanPolicy authorizes Codex and Claude with one Profile each,
// deterministic preferences for agents allowed on both Runtimes, and pinned
// business context. Observations are supplied separately by the test source.
func installStagePlanPolicy(t *testing.T, stateRoot, id string) {
	t.Helper()
	recordPath := filepath.Join(stateRoot, "projects", id, "installation.json")
	wire, err := os.ReadFile(recordPath)
	if err != nil {
		t.Fatal(err)
	}
	record, issues := local.DecodeRecord(wire)
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	recordState := record.State()
	source := filepath.Join(recordState.SourceLocation, "axiom.yaml")
	if wire, err = os.ReadFile(source); err != nil {
		t.Fatal(err)
	}
	value, manifestIssues := manifest.Decode(wire)
	if len(manifestIssues) != 0 {
		t.Fatal(manifestIssues)
	}
	state := value.State()
	if state.SchemaVersion < 2 {
		t.Fatal("fixture requires a Runtime allowlist schema")
	}
	state.Runtimes = project.Configured([]project.Runtime{{ID: "codex"}, {ID: "claude"}})
	state.ModelProfiles = project.Configured([]project.ModelProfile{{Key: "profile-codex", RuntimeRef: project.Configured("codex"), Model: project.Configured("local-codex-model")}, {Key: "profile-claude", RuntimeRef: project.Configured("claude"), Model: project.Configured("local-claude-model")}})
	state.RuntimePreferences = project.Configured([]project.RuntimePreference{{Role: "engineer", Complexity: "medium", ModelProfileRef: "profile-codex"}, {Role: "integrator", Complexity: "medium", ModelProfileRef: "profile-codex"}})
	state.BusinessContext = project.Configured(project.BusinessContext{Text: project.Configured("Synthetic bounded business context")})
	if value, manifestIssues = project.New(state); len(manifestIssues) != 0 {
		t.Fatal(manifestIssues)
	}
	if wire, manifestIssues = manifest.Encode(value); len(manifestIssues) != 0 {
		t.Fatal(manifestIssues)
	}
	if err := os.WriteFile(source, wire, 0o600); err != nil {
		t.Fatal(err)
	}
	snapshot, readIssues := projectapp.ReadSnapshot(manifest.Codec{}, wire, nil)
	if len(readIssues) != 0 {
		t.Fatal(readIssues)
	}
	recordState.PortableRevision, recordState.ArtifactDigests = snapshot.Revision(), snapshot.Digests()
	if record, issues = local.NewRecord(recordState); len(issues) != 0 {
		t.Fatal(issues)
	}
	if wire, issues = local.EncodeRecord(record); len(issues) != 0 {
		t.Fatal(issues)
	}
	if err := os.WriteFile(recordPath, wire, 0o600); err != nil {
		t.Fatal(err)
	}
	capabilities := []string{"axiom-skills", "read", "repository-write"}
	cfg := runtimeprofile.ConfigurationV3{FormatVersion: runtimeprofile.FormatVersionV3, Revision: 1}
	for _, runtimeID := range []string{"codex", "claude"} {
		cfg.Runtimes = append(cfg.Runtimes, runtimeprofile.RuntimeV3{ID: runtimeID, Adapter: runtimeID, Enabled: true, AllowlistedProfileIDs: []string{"profile-" + runtimeID}})
		cfg.ModelProfiles = append(cfg.ModelProfiles, runtimeprofile.ModelProfile{ID: "profile-" + runtimeID, RuntimeID: runtimeID, Model: "local-" + runtimeID + "-model", Effort: runtimeprofile.Effort{Mode: runtimeprofile.EffortRuntimeDefault, Value: "default"}, Capabilities: capabilities, Complexities: []string{"medium", "high"}})
	}
	publishRuntimeProfileV3(t, stateRoot, cfg)
}

type controlledObservations struct {
	base         runtimeapplication.Source
	observations []runtimeprofile.OperationalObservation
}

func (s *controlledObservations) Load(ctx context.Context, id string) (runtimeapplication.Snapshot, error) {
	snapshot, err := s.base.Load(ctx, id)
	if err != nil {
		return snapshot, err
	}
	snapshot.Observer = s
	return snapshot, nil
}
func (s *controlledObservations) ObserveExecutable(ctx context.Context, id string) (runtimeprofile.OperationalObservation, error) {
	for _, o := range s.observations {
		if o.RuntimeID == id {
			return o, nil
		}
	}
	return runtimeprofile.OperationalObservation{}, runtimeprofile.ErrExecutableUnavailable
}
func (s *controlledObservations) ObserveOperational(ctx context.Context, id, profile string) (runtimeprofile.OperationalObservation, error) {
	return s.ObserveExecutable(ctx, id)
}
func operationalObservations() []runtimeprofile.OperationalObservation {
	var observations []runtimeprofile.OperationalObservation
	for _, runtimeID := range []string{"codex", "claude"} {
		hash := sha256.Sum256([]byte(runtimeID))
		observations = append(observations, runtimeprofile.OperationalObservation{RuntimeID: runtimeID, Adapter: runtimeID, Installed: true, Available: true, ExecutableDigest: hex.EncodeToString(hash[:]), Revision: 1, ObservedAt: time.Unix(1, 0).UTC(), InvocationMode: "noninteractive", EffortSourceDigest: strings.Repeat("a", 64)})
	}
	return observations
}

// The public CLI path composes the real Execution binding, Project policy and
// compiler. Only Runtime observations are controlled; nothing is invoked.
func TestWorkflowStagePlanPublicJourneySingleAndMixedRuntimeGraph(t *testing.T) {
	env := newWorkItemSourceEnvironment(t)
	installLinkedTarget(t, env)
	installStagePlanPolicy(t, env.state, workItemSourceProjectID)
	service := currentProfileStartFixture(t, env.service.(lifecycleService), env.state)
	observed := &controlledObservations{base: runtimePolicySource{installation: service.installation, portable: service.portable, stateRoot: env.state, runtimes: service.runtimeObservationSources()}, observations: operationalObservations()}
	service.RuntimePolicySource = observed
	ctx := context.Background()
	invoke := func(code int, args ...string) []byte {
		t.Helper()
		var output bytes.Buffer
		if actual := cli.RunInteractive(ctx, args, service, currentProvenance(), nil, &output, io.Discard); actual != code {
			t.Fatalf("%v code=%d: %s", args, actual, output.String())
		}
		return output.Bytes()
	}
	run := func(code int, args ...string) stagePlanEvent {
		t.Helper()
		var value stagePlanEvent
		wire := invoke(code, append([]string{"--json"}, args...)...)
		if err := json.Unmarshal(bytes.TrimSpace(wire), &value); err != nil {
			t.Fatalf("%v: %s", args, wire)
		}
		t.Logf("r1-evidence {\"scenario\":\"A\",\"canonical\":{\"status\":%q,\"plan\":%s}}", value.Status, func() []byte { decoded, _ := json.Marshal(value.Plan); return decoded }())
		t.Logf("r1-evidence {\"scenario\":\"F\",\"canonical\":{\"category\":%q}}", value.Category)
		return value
	}
	startExecution := func(workItem string) *cli.WorkflowView {
		t.Helper()
		start := []string{"workflow", "start", "--project", "external", "--repository", "main", "--work-item", workItem, "--role", "engineer", "--complexity", "medium", "--capabilities", "read", "--runtime", "codex"}
		preview := run(0, start...)
		started := run(0, append(start, "--runtime-preview", preview.PreviewDigest)...)
		if started.Workflow == nil || started.Workflow.Binding == nil {
			t.Fatalf("start %s", started.Category)
		}
		return started.Workflow
	}
	planArgs := func(view *cli.WorkflowView, stage, file string) []string {
		return []string{"workflow", "stage", "plan", "--project", "external", "--repository", "main", "--work-item", "github:" + view.WorkItem.Resource + "#" + view.WorkItem.ExternalID, "--execution", view.ExecutionID, "--expected-revision", strconv.FormatUint(view.Revision, 10), "--stage", stage, "--plan", file}
	}

	// A: single-agent intake on the built-in revision.
	a := startExecution("github:owner/repo#7")
	before := snapshotTrees(t, env.root, env.state, env.elsewhere)
	single := writeStagePlanDocument(t, []executiongraph.WorkUnit{stagePlanUnit("owner", []string{"docs"})}, nil)
	first := run(0, planArgs(a, "intake", single)...)
	if first.Status != "success" || first.Category != "stage_plan_ready" || first.Plan == nil || len(first.ConfirmedEffects) != 0 {
		t.Fatalf("single-agent plan: %+v", first)
	}
	p := first.Plan
	if p.ExecutionKind != "single" || p.GraphProposal != nil || len(p.Resolutions) != 1 || p.Resolutions[0].RuntimeID != "codex" || p.Execution.ExecutionID != a.ExecutionID || p.Execution.Revision != a.Revision || p.Workflow != a.Binding.Definition || p.ProjectID != workItemSourceProjectID || p.RepositoryKey != "main" {
		t.Fatalf("single-agent plan identity: %+v", p)
	}
	if len(p.Compilation.Agents) != 1 || p.Compilation.Agents[0].Input.Instructions == "" || len(p.Compilation.Agents[0].Input.Inputs) != 1 || p.Compilation.Agents[0].Input.Inputs[0].InputID != "source" {
		t.Fatal("child input lost stage instructions or bound Work Item reference")
	}
	if again := run(0, planArgs(a, "intake", single)...); again.Plan.Digest != p.Digest {
		t.Fatal("identical inputs produced a different stage plan")
	}
	human := invoke(0, planArgs(a, "intake", single)...)
	for _, want := range []string{p.Digest, "stage_plan_ready", "single", p.Resolutions[0].ModelProfileID} {
		if !bytes.Contains(human, []byte(want)) {
			t.Fatalf("human rendering lacks %s", want)
		}
	}
	if !bytes.Equal(before, snapshotTrees(t, env.root, env.state, env.elsewhere)) {
		t.Fatal("stage planning mutated local state")
	}
	observed.observations[0].ExecutableDigest = strings.Repeat("e", 64)
	if drifted := run(0, planArgs(a, "intake", single)...); drifted.Plan.Digest == p.Digest {
		t.Fatal("Runtime observation drift did not require a new proposal")
	}
	observed.observations = operationalObservations()

	// B: new schema-2 copy with an explicit Profile per agent and independent Codex/Claude DAG.
	historicalWire, err := os.ReadFile("../../docs/specifications/007-configurable-workflows/examples/custom-r1.json")
	if err != nil {
		t.Fatal(err)
	}
	legacy, issues := workflowdefinition.Decode(historicalWire)
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	definition := legacy.Definition
	definition.SchemaVersion = 2
	for i := range definition.Stages {
		for j := range definition.Stages[i].Agents {
			a := &definition.Stages[i].Agents[j]
			a.Effort = workflowdefinition.Effort{}
			a.ProfileRef = "profile-codex"
			if len(a.RuntimeConstraints) == 1 && a.RuntimeConstraints[0] == "claude" {
				a.ProfileRef = "profile-claude"
			}
		}
	}
	current, issues := workflowdefinition.Encode(definition)
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	file := filepath.Join(t.TempDir(), "explicit-schema2.json")
	if err = os.WriteFile(file, current.Canonical, 0600); err != nil {
		t.Fatal(err)
	}
	request := cli.ProjectWorkflowInput{WorkflowRequest: projectapp.WorkflowRequest{Operation: "create", Project: "external"}, File: file}
	author := service
	author.projectsRoot = env.elsewhere
	if author.portable, err = local.NewPortableStore(env.elsewhere); err != nil {
		t.Fatal(err)
	}
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
	created := apply(request)
	apply(cli.ProjectWorkflowInput{WorkflowRequest: projectapp.WorkflowRequest{Operation: "select", Project: "external", Ref: *created.WorkflowAuthoring.Reference}})
	b := startExecution("github:owner/repo#8")
	if b.Binding.Definition.WorkflowID != "reviewed-delivery" {
		t.Fatalf("B bound %+v", b.Binding.Definition)
	}
	if early := run(1, planArgs(b, "implementation", single)...); early.Category != "stage_prerequisite_missing" {
		t.Fatalf("future stage planned without predecessor outputs: %+v", early)
	}
	resolved := env.installation.Resolve(ctx, "external")
	decision := []byte("Explicit synthetic human gate decision\n")
	if err := os.WriteFile(filepath.Join(resolved.Project.Repositories[0].Path, "decision.md"), decision, 0o600); err != nil {
		t.Fatal(err)
	}
	hash := sha256.Sum256(decision)
	selector := []string{"--project", "external", "--repository", "main", "--work-item", "github:owner/repo#8", "--execution", b.ExecutionID}
	for count := 0; b.CurrentGate != "implementation" && count < 16; count++ {
		if action := b.GateAction; action != nil && action.Operation == "fact" {
			b = run(0, append(append([]string{"workflow", "fact"}, selector...), "--expected-revision", strconv.FormatUint(b.Revision, 10), "--fact", strings.ReplaceAll(string(action.Fact), "_", "-"), "--active", "--actor", "maintainer", "--reference", "specification:decision.md:"+hex.EncodeToString(hash[:]), "--authorize-local")...).Workflow
		}
		file := publishConfiguredOutput(t, env.state, b)
		b = run(0, append(append([]string{"workflow", "advance"}, selector...), "--expected-revision", strconv.FormatUint(b.Revision, 10), "--gate", b.CurrentGate, "--outcome", "pass", "--stage-result", file)...).Workflow
	}
	if b.CurrentGate != "implementation" {
		t.Fatalf("B did not reach implementation: %s", b.CurrentGate)
	}
	write := executiongraph.Effect{Kind: "repository-write", Target: "src/feature.go"}
	graphPlan := writeStagePlanDocument(t, []executiongraph.WorkUnit{
		stagePlanUnit("implementer", []string{"src"}, write),
		stagePlanUnit("reviewer", []string{"docs"}),
		stagePlanUnit("integrator", []string{"src"}, write),
	}, []executiongraph.Effect{write})
	before = snapshotTrees(t, env.root, env.state, env.elsewhere)
	graph := run(0, planArgs(b, "implementation", graphPlan)...)
	g := graph.Plan
	if g == nil || g.ExecutionKind != "graph" || g.GraphProposal == nil || g.Compilation.Proposal == nil || len(g.Compilation.Proposal.Nodes) != 3 {
		t.Fatalf("multi-agent stage did not produce an inspectable graph: %+v", graph)
	}
	runtimes, efforts := map[string]string{}, map[string]string{}
	for _, resolution := range g.Resolutions {
		runtimes[resolution.AgentID], efforts[resolution.AgentID] = resolution.RuntimeID, resolution.EffectiveEffort
	}
	if runtimes["implementer"] != "codex" || runtimes["reviewer"] != "claude" || runtimes["integrator"] != "codex" || efforts["implementer"] != "default" || efforts["reviewer"] != "default" || efforts["integrator"] != "default" {
		t.Fatalf("Runtime/effort resolution: %v %v", runtimes, efforts)
	}
	for _, node := range g.Compilation.Proposal.Nodes {
		if node.Key == "implementation:integrator" && (!node.IntegrationOwner || len(node.Dependencies) != 2) {
			t.Fatalf("integration owner lost dependencies: %+v", node)
		}
	}
	for _, agent := range g.Compilation.Agents {
		if agent.Input.Workflow != b.Binding.Definition || agent.Input.StageID != "implementation" || len(agent.Input.Inputs) != 2 || len(agent.Input.Validators) == 0 || len(agent.Input.HumanGates) != 1 {
			t.Fatalf("child envelope lost workflow, context or validation obligations: %+v", agent.Input)
		}
	}
	if len(g.Blockers) != 1 || g.Blockers[0] != "human_gate_pending:implementation_authority" {
		t.Fatalf("pending human gate not reported: %v", g.Blockers)
	}
	if !bytes.Equal(before, snapshotTrees(t, env.root, env.state, env.elsewhere)) {
		t.Fatal("graph planning mutated local state or satisfied a gate")
	}

	// Fail-closed resolution and authority, with no implicit fallback.
	escalated := writeStagePlanDocument(t, []executiongraph.WorkUnit{
		stagePlanUnit("implementer", []string{"src"}, write),
		stagePlanUnit("reviewer", []string{"docs"}),
		stagePlanUnit("integrator", []string{"src"}, write),
	}, nil)
	if denied := run(1, planArgs(b, "implementation", escalated)...); denied.Status != "denied_authority" || denied.Category != "authority_denied" || denied.Plan != nil {
		t.Fatalf("authority escalation accepted: %+v", denied)
	}
	for _, target := range []string{"..", "C:/outside", "C:outside"} {
		escape := executiongraph.Effect{Kind: "repository-write", Target: target}
		outside := writeStagePlanDocument(t, []executiongraph.WorkUnit{
			stagePlanUnit("implementer", []string{target}, escape),
			stagePlanUnit("reviewer", []string{"docs"}),
			stagePlanUnit("integrator", []string{"src"}, write),
		}, []executiongraph.Effect{escape, write})
		if escaped := run(1, planArgs(b, "implementation", outside)...); escaped.Status != "denied_authority" || escaped.Category != "authority_denied" || escaped.Plan != nil || len(escaped.ConfirmedEffects) != 0 {
			t.Fatalf("scope %q outside the Repository accepted: %+v", target, escaped)
		}
	}
	observed.observations[1].Available = false
	if unavailable := run(1, planArgs(b, "implementation", graphPlan)...); unavailable.Category != "runtime_unresolvable" {
		t.Fatalf("unavailable Claude fell back: %+v", unavailable)
	}
	if !bytes.Equal(before, snapshotTrees(t, env.root, env.state, env.elsewhere)) {
		t.Fatal("refused planning mutated local state")
	}
	if status := run(0, append([]string{"workflow", "status"}, selector...)...).Workflow; status.Revision != b.Revision || status.CurrentGate != "implementation" {
		t.Fatal("planning advanced the Execution")
	}
}
