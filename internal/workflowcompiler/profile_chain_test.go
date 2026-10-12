package workflowcompiler

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/rgomids/axiom/internal/executiongraph"
	"github.com/rgomids/axiom/internal/local"
	"github.com/rgomids/axiom/internal/project"
	"github.com/rgomids/axiom/internal/runtimeapplication"
	"github.com/rgomids/axiom/internal/runtimeprofile"
	"github.com/rgomids/axiom/internal/workflow"
	"github.com/rgomids/axiom/internal/workflowdefinition"
)

// This chain exercises Axiom's actual protected stores and application/compiler
// boundaries. Only positive capability evidence is injected; no vendor runs.
type chainAuthoringStore struct{ store local.RuntimeProfileStore }

func chainSnapshot(s local.RuntimeProfileSnapshot) runtimeapplication.LocalSnapshot {
	return runtimeapplication.LocalSnapshot{State: s.State, Revision: s.Revision, Configuration: s.Configuration, Token: s.Token}
}
func (s chainAuthoringStore) Inspect(ctx context.Context) (runtimeapplication.LocalSnapshot, error) {
	v, e := s.store.Inspect(ctx)
	return chainSnapshot(v), e
}
func (s chainAuthoringStore) Publish(ctx context.Context, token string, w []byte) (runtimeapplication.LocalSnapshot, error) {
	v, e := s.store.Publish(ctx, token, w)
	return chainSnapshot(v), e
}

type chainObserver map[string]runtimeprofile.Observation

func (o chainObserver) Observe(_ context.Context, id string) (runtimeprofile.Observation, error) {
	v, ok := o[id]
	if !ok {
		return v, fmt.Errorf("observation unavailable")
	}
	return v, nil
}

type chainPolicySource struct {
	store    local.RuntimeProfileStore
	project  project.Project
	observer chainObserver
}

func (s *chainPolicySource) Load(ctx context.Context, _ string) (runtimeapplication.Snapshot, error) {
	cfg, err := s.store.Load(ctx)
	return runtimeapplication.Snapshot{Project: s.project, Configuration: cfg, Observer: s.observer}, err
}

func TestProfileV2ProtectedMigrationCompilerGraphRestartChain(t *testing.T) {
	ctx := context.Background()
	tmp := t.TempDir()
	stateRoot := filepath.Join(tmp, "state")
	store, err := local.NewRuntimeProfileStore(stateRoot)
	if err != nil {
		t.Fatal(err)
	}
	legacy := runtimeprofile.Configuration{FormatVersion: 1, Revision: 1, Runtimes: []runtimeprofile.Runtime{{ID: "codex", Adapter: "codex", Enabled: true, AllowlistedProfileIDs: []string{"codex-explicit", "codex-default"}, CredentialReference: "env:LEGACY_PLACEHOLDER"}, {ID: "claude", Adapter: "claude", Enabled: true, AllowlistedProfileIDs: []string{"claude-explicit"}}}, ModelProfiles: []runtimeprofile.ModelProfile{{ID: "codex-explicit", RuntimeID: "codex", Model: "codex-model", Capabilities: []string{"read", "repository-write"}, Complexities: []string{"medium"}}, {ID: "codex-default", RuntimeID: "codex", Model: "codex-model", Capabilities: []string{"read", "repository-write"}, Complexities: []string{"medium"}}, {ID: "claude-explicit", RuntimeID: "claude", Model: "claude-model", Capabilities: []string{"read", "repository-write"}, Complexities: []string{"medium"}}}}
	legacyWire, err := runtimeprofile.Encode(legacy)
	if err != nil {
		t.Fatal(err)
	}
	legacyFixture := filepath.Join(tmp, "historical-v1.json")
	if err := os.WriteFile(legacyFixture, legacyWire, 0600); err != nil {
		t.Fatal(err)
	}
	if err := store.Create(ctx, legacy); err != nil {
		t.Fatal(err)
	}
	sourcePath := "../../docs/specifications/007-configurable-workflows/examples/custom-r1.json"
	sourceWire, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatal(err)
	}
	historical, issues := workflowdefinition.Decode(sourceWire)
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	historicalDigest := historical.Digest
	mapping := map[string]runtimeprofile.Effort{"codex-explicit": {Mode: "explicit", Value: "medium"}, "codex-default": {Mode: "runtime-default", Value: "default"}, "claude-explicit": {Mode: "explicit", Value: "high"}}
	author := runtimeapplication.Authoring{Store: chainAuthoringStore{store}, Installed: func(string) bool { t.Fatal("migration must not discover/probe vendor"); return false }}
	migration := runtimeapplication.AuthoringRequest{Kind: "profile", Operation: "migrate", MigrationEfforts: mapping, ClearLegacyReferences: true}
	preview, err := author.Run(ctx, migration)
	if err != nil || preview.Category != "preview" {
		t.Fatalf("migration preview %s: %v", preview.Category, err)
	}
	inspected, err := store.Inspect(ctx)
	if err != nil || inspected.Configuration.FormatVersion != 1 || inspected.Revision != 1 {
		t.Fatal("preview changed historical active state")
	}
	migration.AuthorizeLocal = true
	migration.ExpectedRevision = preview.ExpectedRevision
	migration.PreviewDigest = preview.PreviewDigest
	applied, err := author.Run(ctx, migration)
	if err != nil || applied.Revision != 2 {
		t.Fatal("reviewed local migration failed", err)
	}
	restarted, err := local.NewRuntimeProfileStore(stateRoot)
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := restarted.Load(ctx)
	if err != nil || cfg.FormatVersion != 2 || cfg.Revision != 2 {
		t.Fatal("restart did not load v2", err)
	}
	const projectID = "123e4567-e89b-42d3-a456-426614174000"
	ps := project.State{SchemaVersion: 2, ID: projectID, Slug: "profile-chain", Name: "Profile chain", Runtimes: project.Configured([]project.Runtime{{ID: "codex"}, {ID: "claude"}})}
	portable := []project.ModelProfile{}
	for _, p := range cfg.ModelProfiles {
		portable = append(portable, project.ModelProfile{Key: p.ID, RuntimeRef: project.Configured(p.RuntimeID), Model: project.Configured(p.Model)})
	}
	ps.ModelProfiles = project.Configured(portable)
	proj, diagnostics := project.New(ps)
	if len(diagnostics) > 0 {
		t.Fatal(diagnostics)
	}
	observations := chainObserver{}
	for _, r := range cfg.Runtimes {
		observations[r.ID] = runtimeprofile.Observation{RuntimeID: r.ID, Adapter: r.Adapter, Installed: true, Available: true, Version: "injected-1.0", ExecutableDigest: digestForTest("executable-" + r.ID), Revision: cfg.Revision, ObservedAt: time.Unix(1, 0).UTC(), CapabilityStatus: map[string]runtimeprofile.CapabilityStatus{"read": runtimeprofile.CapabilityProven, "repository-write": runtimeprofile.CapabilityProven}}
	}
	for _, p := range cfg.ModelProfiles {
		o := observations[p.RuntimeID]
		o.ProfileProofs = append(o.ProfileProofs, runtimeprofile.ProfileProof{RuntimeID: p.RuntimeID, Model: p.Model, Effort: p.Effort, ExecutableDigest: o.ExecutableDigest, RuntimeVersion: o.Version, InvocationMode: "noninteractive", ObservationRevision: o.Revision, Reference: "injected-" + p.ID, EffortSourceDigest: digestForTest("effort-sources-" + p.ID), Authentication: runtimeprofile.CapabilityProven, ModelAvailability: runtimeprofile.CapabilityProven, EffortSupport: runtimeprofile.CapabilityProven, NoOverride: runtimeprofile.CapabilityProven, CapabilityStatus: map[string]runtimeprofile.CapabilityStatus{"read": runtimeprofile.CapabilityProven, "repository-write": runtimeprofile.CapabilityProven}})
		observations[p.RuntimeID] = o
	}
	source := &chainPolicySource{store: restarted, project: proj, observer: observations}
	policy := runtimeapplication.New(source)
	// Explicit synthetic operator mappings prepare a new schema2 document. Local
	// migration never infers these mappings from historical Workflow requests.
	copied, issues := workflowdefinition.Decode(historical.Canonical)
	if len(issues) > 0 {
		t.Fatal(issues)
	}
	d := copied.Definition
	d.SchemaVersion = 2
	d.Revision = 2
	for i := range d.Stages {
		for j := range d.Stages[i].Agents {
			a := &d.Stages[i].Agents[j]
			a.ProfileRef = "codex-default"
			if a.ID == "implementer" {
				a.ProfileRef = "codex-explicit"
			}
			if a.ID == "reviewer" {
				a.ProfileRef = "claude-explicit"
			}
			a.Effort = workflowdefinition.Effort{}
		}
	}
	mapped, issues := workflowdefinition.Encode(d)
	if len(issues) > 0 {
		t.Fatal(issues)
	}
	req := requestFor(t, mapped, "implementation", []InputBinding{{InputID: "source", Reference: "tasks/result", Digest: digestForTest("prior-stage")}, {InputID: "business-context", Reference: "business-context", Digest: digestForTest("context")}})
	req.ProjectID = projectID
	for i := range req.Plan.Work {
		req.Plan.Work[i].Scope.ProjectID = projectID
	}
	compiler := New(policy, capabilityOK{})
	compiled, err := compiler.CompileV2(ctx, req)
	if err != nil || compiled.FormatVersion != 2 || compiled.Proposal == nil || compiled.Proposal.FormatVersion != 2 {
		t.Fatal("compile v2 chain failed", err)
	}
	planned, err := compiler.PlanStageV2(ctx, stagePlanRequest(t, req, nil))
	if err != nil || planned.FormatVersion != 2 || planned.GraphProposal == nil || planned.Compilation.Proposal == nil || planned.Compilation.Proposal.FormatVersion != 2 {
		t.Fatal("stageplan v2 chain failed", err)
	}
	if planned.Compilation.Digest != compiled.Digest || len(planned.Resolutions) != 3 {
		t.Fatal("stageplan/compile disagree")
	}
	publication := executiongraph.PublicationRequest{Proposal: *compiled.Proposal, ExpectedDigest: compiled.Proposal.Digest, GraphRevision: 1, ParentAuthority: req.ParentAuthority, ChildAuthorities: map[string][]executiongraph.Effect{}, Resolutions: map[string]executiongraph.Resolution{}, Workspaces: map[string]string{}, AuthorityReferences: map[string]string{}}
	for _, a := range compiled.Agents {
		key := a.Input.NodeKey
		publication.ChildAuthorities[key] = a.Effects
		publication.Workspaces[key] = filepath.Join(tmp, "workspace", a.AgentID)
		publication.AuthorityReferences[key] = "reviewed-local-authority-" + a.AgentID
		publication.Resolutions[key] = executiongraph.Resolution{RuntimeID: a.Choice.RuntimeID, ModelProfileID: a.Choice.ModelProfileID, ConfigurationRevision: a.Choice.ConfigurationRevision, ObservationRevision: a.Choice.ObservationRevision, ResolvedProfile: a.Choice.ResolvedProfile}
	}
	graphStore, err := local.NewGraphStore(stateRoot)
	if err != nil {
		t.Fatal(err)
	}
	allocation := 0
	allocate := func() (string, error) {
		allocation++
		return fmt.Sprintf("00000000-0000-4000-8000-%012d", allocation), nil
	}
	graph, err := executiongraph.NewGraphService(graphStore, allocate, func() time.Time { return time.Unix(10, 0).UTC() }).Publish(ctx, publication)
	if err != nil || graph.FormatVersion != 2 {
		t.Fatal("protected graph publication failed", err)
	}
	graphRestart, err := local.NewGraphStore(stateRoot)
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := graphRestart.Load(ctx, projectID, graph.Parent.ExecutionID)
	if err != nil || loaded.FormatVersion != 2 || len(loaded.Children) != 3 {
		t.Fatal("graph restart failed", err)
	}
	beforeGraph, err := executiongraph.EncodeGraph(graph)
	if err != nil {
		t.Fatal(err)
	}
	afterGraph, err := executiongraph.EncodeGraph(loaded)
	if err != nil || !bytes.Equal(beforeGraph, afterGraph) {
		t.Fatal("graph restart changed frozen bytes")
	}
	for _, child := range loaded.Children {
		r := child.Envelope.Resolution.ResolvedProfile
		if r == nil || !executiongraph.ValidProfileControls(r, child.Envelope.Controls, child.Envelope.Capability) {
			t.Fatal("graph lost sole-profile controls")
		}
		if r.ConfigurationRevision != 2 {
			t.Fatal("graph lost migration generation")
		}
	}

	// The configured Execution3 uses the same canonical v2 resolution and retains
	// its complete schema2 snapshot in the protected WorkflowStore.
	startSelection := runtimeapplication.Request{Role: "engineer", Complexity: "medium", Capabilities: []string{"read"}, ModelProfileID: "codex-default"}
	startPreview, err := policy.Preview(ctx, projectID, startSelection)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Unix(20, 0).UTC()
	executionID := "123e4567-e89b-42d3-a456-426614174001"
	execution := workflow.State{ExecutionID: executionID, FormatVersion: workflow.ProfileConfiguredFormatVersion, WorkflowVersion: workflow.ProfileConfiguredWorkflowVersion, ProjectID: projectID, RepositoryKey: "main", WorkItem: workflow.WorkItem{Provider: "github", Resource: "example/public", ExternalID: "306", URL: "https://example.invalid/issues/306", State: "OPEN"}, RuntimeID: "codex", Stage: workflow.Stage(mapped.Definition.Stages[0].ID), Revision: 1, Status: workflow.ExecutionActive, CreatedAt: now, UpdatedAt: now, Provenance: workflow.Identity{Product: "Axiom", Version: "test", Revision: "injected", SourceState: "clean"}}
	content := []byte("Sanitized business context")
	frozenStart := *startPreview.Choice.ResolvedProfile
	execution.Binding = &workflow.WorkflowBinding{ResolvedProfile: &frozenStart, ProjectRevision: startPreview.ProjectDigest, LocalRevision: startPreview.ConfigurationDigest, ProjectID: projectID, Definition: mapped.Ref("project"), SchemaVersion: 2, SnapshotRef: "execution:" + executionID + "#/binding/snapshot", Snapshot: append([]byte(nil), mapped.Canonical...), ObservationDigest: startPreview.ObservationDigest, RuntimePreview: startPreview.Digest(), BoundAt: now, Contexts: map[string]workflow.ContextObservation{"business-context": {Source: "business-context", Content: content, Digest: digestForTest(string(content))}}}
	if !workflow.ValidState(execution) {
		t.Fatal("invalid Execution3 test state")
	}
	executionStore, err := local.NewWorkflowStore(stateRoot)
	if err != nil {
		t.Fatal(err)
	}
	if err := executionStore.Create(ctx, execution); err != nil {
		t.Fatal("protected Execution3 publication", err)
	}
	executionRestart, err := local.NewWorkflowStore(stateRoot)
	if err != nil {
		t.Fatal(err)
	}
	loadedExecution, err := executionRestart.Load(ctx, projectID, "main", execution.WorkItem)
	if err != nil || loadedExecution.FormatVersion != 3 || loadedExecution.Binding.ResolvedProfile == nil || *loadedExecution.Binding.ResolvedProfile != frozenStart || !bytes.Equal(loadedExecution.Binding.Snapshot, mapped.Canonical) {
		t.Fatal("Execution3 restart lost snapshot", err)
	}
	modified := loadedExecution
	bindingCopy := *loadedExecution.Binding
	profileCopy := *loadedExecution.Binding.ResolvedProfile
	profileCopy.Effort = runtimeprofile.Effort{Mode: "explicit", Value: "high"}
	bindingCopy.ResolvedProfile = &profileCopy
	modified.Binding = &bindingCopy
	if !workflow.ValidState(modified) {
		t.Fatal("rebind refusal fixture is not structurally valid")
	}
	if err := executionRestart.Save(ctx, modified); err == nil {
		t.Fatal("protected store rebound existing Execution3")
	}
	retained, err := executionRestart.Load(ctx, projectID, "main", execution.WorkItem)
	if err != nil || *retained.Binding.ResolvedProfile != frozenStart {
		t.Fatal("rebind refusal changed persisted tuple")
	}
	paths, err := filepath.Glob(filepath.Join(stateRoot, "executions", "v1", projectID, "*.json"))
	if err != nil || len(paths) != 1 {
		t.Fatalf("Execution3 record path count %d: %v", len(paths), err)
	}
	recordPath := paths[0]
	originalExecutionWire, err := os.ReadFile(recordPath)
	if err != nil {
		t.Fatal(err)
	}
	// Append-only current-family corpus. Paths are normalized only in this test
	// receipt; actual protected records retain their isolated workspace paths.
	configurationWire, err := runtimeprofile.Encode(cfg)
	if err != nil {
		t.Fatal(err)
	}
	configurationDigest, err := runtimeprofile.Digest(cfg)
	if err != nil {
		t.Fatal(err)
	}
	tombstoneWire, err := runtimeprofile.EncodeTombstone(runtimeprofile.Tombstone{Kind: runtimeprofile.TombstoneKind, FormatVersion: 1, Revision: 3, PriorConfigurationDigest: configurationDigest})
	if err != nil {
		t.Fatal(err)
	}
	stagePlanWire, err := json.Marshal(planned)
	if err != nil {
		t.Fatal(err)
	}
	corpus := map[string][]byte{"local-v2.json": configurationWire, "tombstone-v1.json": tombstoneWire, "workflow-v2.json": mapped.Canonical, "stage-plan-v2.json": stagePlanWire, "execution-v3.json": originalExecutionWire, "graph-v2.json": beforeGraph}
	for name, wire := range corpus {
		for _, agent := range compiled.Agents {
			originalPath, _ := json.Marshal(filepath.Join(tmp, "workspace", agent.AgentID))
			normalizedPath, _ := json.Marshal("/synthetic/issue306/workspace/" + agent.AgentID)
			wire = bytes.ReplaceAll(wire, originalPath, normalizedPath)
		}
		wire = bytes.TrimSpace(bytes.ReplaceAll(wire, []byte(tmp), []byte("/synthetic/issue306")))
		path := filepath.Join("testdata", "issue306", name)
		frozen, err := os.ReadFile(path)
		if err != nil || !bytes.Equal(bytes.TrimSpace(frozen), wire) {
			t.Fatalf("current corpus changed: %s (%v)", name, err)
		}
		hash := sha256.Sum256(bytes.TrimSpace(frozen))
		t.Logf("current-corpus %s sha256=%s", name, hex.EncodeToString(hash[:]))
	}
	for _, poison := range []string{strings.Replace(string(originalExecutionWire), `"formatVersion":3`, `"formatVersion":3,"formatVersion":3`, 1), strings.Replace(string(originalExecutionWire), `"profileId":"codex-default"`, `"profileId":"codex-default","profileId":"codex-default"`, 1)} {
		if poison == string(originalExecutionWire) {
			t.Fatal("duplicate key test did not mutate wire")
		}
		if err := os.WriteFile(recordPath, []byte(poison), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := executionRestart.Load(ctx, projectID, "main", execution.WorkItem); err == nil {
			t.Fatal("Execution3 duplicate keys accepted")
		}
		if err := os.WriteFile(recordPath, originalExecutionWire, 0600); err != nil {
			t.Fatal(err)
		}
	}
	restored, err := executionRestart.Load(ctx, projectID, "main", execution.WorkItem)
	if err != nil || *restored.Binding.ResolvedProfile != frozenStart {
		t.Fatal("Execution3 restored bytes failed restart")
	}

	// A reviewed tuple cannot authorize later proof/source or selected-model drift.
	selection := runtimeapplication.Request{Role: "implementer", Complexity: "medium", Capabilities: []string{"read"}, ModelProfileID: "codex-explicit"}
	resolved, err := policy.Preview(ctx, projectID, selection)
	if err != nil {
		t.Fatal(err)
	}
	o := observations["codex"]
	original := o.ProfileProofs[0]
	o.ProfileProofs[0].EffortSourceDigest = digestForTest("changed-sources")
	observations["codex"] = o
	if _, err := policy.Check(ctx, resolved); err == nil {
		t.Fatal("proof source drift reused review")
	}
	o.ProfileProofs[0] = original
	o.ProfileProofs[0].Model = "other-model"
	observations["codex"] = o
	if _, err := compiler.CompileV2(ctx, req); err == nil {
		t.Fatal("model-mismatched proof compiled")
	}
	o.ProfileProofs[0] = original
	o.ProfileProofs[0].EffortSupport = runtimeprofile.CapabilityDeclared
	observations["codex"] = o
	if _, err := compiler.CompileV2(ctx, req); err == nil {
		t.Fatal("unknown effort compiled")
	}
	// No later refusal mutates existing graph snapshots or source artifacts.
	again, err := graphRestart.Load(ctx, projectID, graph.Parent.ExecutionID)
	if err != nil || !reflect.DeepEqual(again.Children, loaded.Children) {
		t.Fatal("failure rewrote historic graph")
	}
	legacyAfter, err := os.ReadFile(legacyFixture)
	if err != nil || !bytes.Equal(legacyAfter, legacyWire) {
		t.Fatal("v1 fixture changed")
	}
	sourceAfter, err := os.ReadFile(sourcePath)
	if err != nil || !bytes.Equal(sourceAfter, sourceWire) || historical.Digest != historicalDigest {
		t.Fatal("source workflow revision1 changed")
	}
}
