package graphapplication

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/rgomids/axiom/internal/executiongraph"
	"github.com/rgomids/axiom/internal/gitworkspace"
	"github.com/rgomids/axiom/internal/project"
	"github.com/rgomids/axiom/internal/runtimeadapter"
	"github.com/rgomids/axiom/internal/runtimeapplication"
	"github.com/rgomids/axiom/internal/runtimeprofile"
)

type policySource struct{ snapshot runtimeapplication.Snapshot }

func (s *policySource) Load(context.Context, string) (runtimeapplication.Snapshot, error) {
	return s.snapshot, nil
}

// policyFixture reviews previews against the real executable that the command
// profiles name and the credential reference the local configuration binds.
func policyFixture(t *testing.T, graph *executiongraph.Graph, runtimeID, executable, credential string) (*runtimeapplication.Service, map[string]runtimeapplication.Preview, *policySource) {
	t.Helper()
	const projectID = "12345678-1234-4abc-8def-123456789abc"
	state := project.State{SchemaVersion: 2, ID: projectID, Slug: "sample", Name: "Sample", Runtimes: project.Configured([]project.Runtime{{ID: runtimeID}})}
	portableProfiles := []project.ModelProfile{}
	preferences := []project.RuntimePreference{}
	cfg := runtimeprofile.Configuration{FormatVersion: 1, Revision: 1}
	runtime := runtimeprofile.Runtime{ID: runtimeID, Adapter: runtimeID, Enabled: true, CredentialReference: credential}
	for i := range graph.Children {
		child := &graph.Children[i]
		child.Envelope.Scope.ProjectID = projectID
		child.Envelope.Resolution.RuntimeID = runtimeID
		id := child.Envelope.Resolution.ModelProfileID
		portableProfiles = append(portableProfiles, project.ModelProfile{Key: id, RuntimeRef: project.Configured(runtimeID), Model: project.Configured("local-test-profile")})
		preferences = append(preferences, project.RuntimePreference{Role: child.Envelope.Capability.Role, Complexity: "low", ModelProfileRef: id})
		runtime.AllowlistedProfileIDs = append(runtime.AllowlistedProfileIDs, id)
		cfg.ModelProfiles = append(cfg.ModelProfiles, runtimeprofile.ModelProfile{ID: id, RuntimeID: runtimeID, Model: "local-test-profile", Capabilities: []string{"go"}, Complexities: []string{"low"}})
	}
	state.ModelProfiles = project.Configured(portableProfiles)
	state.RuntimePreferences = project.Configured(preferences)
	p, issues := project.New(state)
	if len(issues) > 0 {
		t.Fatal(issues)
	}
	cfg.Runtimes = []runtimeprofile.Runtime{runtime}
	identity, err := runtimeadapter.ExecutableIdentity(executable)
	if err != nil {
		t.Fatal(err)
	}
	inventory, err := runtimeadapter.NewInventory([]runtimeprofile.Observation{{RuntimeID: runtimeID, Adapter: runtimeID, Installed: true, Available: true, ExecutableDigest: identity, Revision: 1, ObservedAt: time.Unix(10, 0).UTC(), CapabilityStatus: map[string]runtimeprofile.CapabilityStatus{"go": runtimeprofile.CapabilityProven}}})
	if err != nil {
		t.Fatal(err)
	}
	source := &policySource{snapshot: runtimeapplication.Snapshot{Project: p, Configuration: cfg, Observer: inventory}}
	policy := runtimeapplication.New(source)
	previews := map[string]runtimeapplication.Preview{}
	for _, child := range graph.Children {
		capability := child.Envelope.Capability
		preview, err := policy.Preview(context.Background(), projectID, runtimeapplication.Request{Role: capability.Role, Complexity: capability.Complexity, Capabilities: capability.Capabilities, RuntimeID: runtimeID})
		if err != nil || preview.Choice == nil {
			t.Fatalf("preview=%+v err=%v", preview, err)
		}
		previews[child.ExecutionID] = preview
	}
	return &policy, previews, source
}

// stubRuntime writes an executable that would leave a marker if it ever ran.
func stubRuntime(t *testing.T, directory, runtimeID string) (string, string) {
	t.Helper()
	if err := os.MkdirAll(directory, 0o700); err != nil {
		t.Fatal(err)
	}
	executable := filepath.Join(directory, runtimeID)
	marker := filepath.Join(directory, runtimeID+"-started")
	if err := os.WriteFile(executable, []byte("#!/bin/sh\ntouch "+marker+"\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	return executable, marker
}

type countedCredentials struct{ calls int }

func (c *countedCredentials) ResolveEnvironment(context.Context, string) ([]string, error) {
	c.calls++
	return []string{"AXIOM_SYNTHETIC_CREDENTIAL=placeholder"}, nil
}

func TestPolicyInvocationChecksExactSelectionBeforeCredentials(t *testing.T) {
	for _, runtimeID := range []string{"codex", "claude"} {
		t.Run(runtimeID, func(t *testing.T) {
			graph := buildGraph(t, canonicalTempDir(t))
			executable, _ := stubRuntime(t, t.TempDir(), runtimeID)
			policy, previews, source := policyFixture(t, &graph, runtimeID, executable, "synthetic:credential")
			credentials := &countedCredentials{}
			profiles := []runtimeadapter.CommandProfile{}
			for _, child := range graph.Children {
				profiles = append(profiles, runtimeadapter.CommandProfile{RuntimeID: runtimeID, ModelProfileID: child.Envelope.Resolution.ModelProfileID, Executable: executable, Model: "local-test-profile", CredentialReference: "synthetic:credential", OutputMax: 4096})
			}
			adapter, err := runtimeadapter.NewInvocationResolver(profiles, credentials)
			if err != nil {
				t.Fatal(err)
			}
			guard, err := newPolicyInvocations(LocalConfiguration{Graph: graph, RuntimePolicy: policy, RuntimePreviews: previews, RuntimeProfiles: profiles}, adapter)
			if err != nil {
				t.Fatal(err)
			}
			invocation, err := guard.ResolveInvocation(context.Background(), graph.Children[0])
			if err != nil || credentials.calls != 1 || !strings.Contains(strings.Join(invocation.Argv, " "), "--model local-test-profile") {
				t.Fatalf("invocation=%+v err=%v calls=%d", invocation, err, credentials.calls)
			}
			before := credentials.calls
			for _, change := range []func(*executiongraph.ChildExecution){
				func(c *executiongraph.ChildExecution) { c.Envelope.Scope.ProjectID = "other-project" },
				func(c *executiongraph.ChildExecution) { c.Envelope.Capability.Role = "review" },
				func(c *executiongraph.ChildExecution) { c.Envelope.Capability.Complexity = "high" },
				func(c *executiongraph.ChildExecution) { c.Envelope.Capability.Capabilities = []string{"shell"} },
				func(c *executiongraph.ChildExecution) { c.Envelope.Resolution.ModelProfileID = "other-profile" },
				func(c *executiongraph.ChildExecution) { c.Envelope.Resolution.ObservationRevision++ },
			} {
				child := graph.Children[0]
				change(&child)
				if _, err := guard.ResolveInvocation(context.Background(), child); err == nil || credentials.calls != before {
					t.Fatalf("binding mismatch err=%v credentials=%d", err, credentials.calls)
				}
			}
			source.snapshot.Configuration.ModelProfiles[0].Model = "changed-model"
			if _, err := guard.ResolveInvocation(context.Background(), graph.Children[0]); err == nil || credentials.calls != before {
				t.Fatalf("drift err=%v credentials=%d", err, credentials.calls)
			}
			source.snapshot.Configuration.ModelProfiles[0].Model = "local-test-profile"
			state := source.snapshot.Project.State()
			state.Name = "Changed Project"
			changed, issues := project.New(state)
			if len(issues) != 0 {
				t.Fatal(issues)
			}
			original := source.snapshot.Project
			source.snapshot.Project = changed
			if _, err := guard.ResolveInvocation(context.Background(), graph.Children[0]); err == nil || credentials.calls != before {
				t.Fatalf("portable drift err=%v credentials=%d", err, credentials.calls)
			}
			source.snapshot.Project = original
			source.snapshot.Observer = nil
			if _, err := guard.ResolveInvocation(context.Background(), graph.Children[0]); err == nil || credentials.calls != before {
				t.Fatalf("observer unavailable err=%v credentials=%d", err, credentials.calls)
			}
		})
	}
}

func TestPolicyInvocationRejectsMissingMismatchedAndOverrideBindings(t *testing.T) {
	graph := buildGraph(t, canonicalTempDir(t))
	executable, _ := stubRuntime(t, t.TempDir(), "codex")
	policy, previews, _ := policyFixture(t, &graph, "codex", executable, "")
	profiles := []runtimeadapter.CommandProfile{}
	for _, child := range graph.Children {
		profiles = append(profiles, runtimeadapter.CommandProfile{RuntimeID: "codex", ModelProfileID: child.Envelope.Resolution.ModelProfileID, Executable: executable, Model: "local-test-profile", OutputMax: 4096})
	}
	for _, mutation := range []func(*LocalConfiguration){
		func(c *LocalConfiguration) { c.RuntimePreviews = nil },
		func(c *LocalConfiguration) { c.RuntimeProfiles[0].Model = "different" },
		func(c *LocalConfiguration) { c.RuntimeProfiles[0].Arguments = []string{"--model", "different"} },
		func(c *LocalConfiguration) { c.RuntimeProfiles[0].Arguments = []string{"-m=different"} },
		func(c *LocalConfiguration) { c.RuntimeProfiles[0].Arguments = []string{"--config", "model=different"} },
		func(c *LocalConfiguration) { c.RuntimeProfiles[0].Arguments = []string{"--model=different"} },
		func(c *LocalConfiguration) { c.RuntimeProfiles[0].Arguments = []string{"-pm", "different"} },
		func(c *LocalConfiguration) { c.RuntimeProfiles[0].Arguments = []string{"--settings=other.json"} },
		func(c *LocalConfiguration) { c.RuntimeProfiles[0].Arguments = []string{"resume"} },
		func(c *LocalConfiguration) { c.RuntimeProfiles[0].Arguments = []string{"--"} },
		func(c *LocalConfiguration) { c.RuntimeProfiles[0].Environment = []string{"OPENAI_API_KEY=other"} },
		func(c *LocalConfiguration) { c.RuntimeProfiles[0].Environment = []string{"ANTHROPIC_MODEL=other"} },
		func(c *LocalConfiguration) { c.RuntimeProfiles[0].Environment = []string{"codex_home=/other"} },
		func(c *LocalConfiguration) { c.RuntimeProfiles[0].Environment = []string{"CLAUDE_CONFIG_DIR=/other"} },
		func(c *LocalConfiguration) { c.RuntimeProfiles[0].Environment = []string{"GH_TOKEN=other"} },
		func(c *LocalConfiguration) { c.RuntimeProfiles[0].Environment = []string{"AWS_PROFILE=other"} },
		func(c *LocalConfiguration) {
			preview := c.RuntimePreviews[c.Graph.Children[0].ExecutionID]
			choice := *preview.Choice
			choice.ExecutableDigest = ""
			preview.Choice = &choice
			c.RuntimePreviews = map[string]runtimeapplication.Preview{c.Graph.Children[0].ExecutionID: preview, c.Graph.Children[1].ExecutionID: c.RuntimePreviews[c.Graph.Children[1].ExecutionID], c.Graph.Children[2].ExecutionID: c.RuntimePreviews[c.Graph.Children[2].ExecutionID]}
		},
	} {
		cfg := LocalConfiguration{Graph: graph, RuntimePolicy: policy, RuntimePreviews: previews, RuntimeProfiles: append([]runtimeadapter.CommandProfile(nil), profiles...)}
		mutation(&cfg)
		if _, err := newPolicyInvocations(cfg, helperInvocations{}); !errors.Is(err, ErrInvalidComposition) {
			t.Fatalf("err=%v", err)
		}
	}
	if _, err := NewLocalService(context.Background(), LocalConfiguration{Graph: graph}); !errors.Is(err, ErrInvalidComposition) {
		t.Fatalf("missing policy err=%v", err)
	}
}

func TestSupplementalArgumentsPreservePromptAndPermissionOptions(t *testing.T) {
	for _, runtimeID := range []string{"codex", "claude"} {
		if !selectionSafeArguments(runtimeID, []string{"--permission-mode", "plan", "--prompt", "Review the approved change"}) {
			t.Fatal("ordinary arguments rejected")
		}
	}
	if !selectionSafeArguments("claude", []string{"-p", "Review this change", "--permission-mode", "plan"}) {
		t.Fatal("Claude print arguments rejected")
	}
}

func TestAlternateModelSelectionRejectedBeforeCredentials(t *testing.T) {
	for runtimeID, options := range map[string][]string{
		"claude": {"--fallback-model", "--agent", "--agents", "--from-pr", "--teleport", "--fork-session", "-r", "-pr"},
		"codex":  {"--profile", "--oss", "--local-provider", "-p", "-jp"},
	} {
		t.Run(runtimeID, func(t *testing.T) {
			graph := buildGraph(t, canonicalTempDir(t))
			executable, _ := stubRuntime(t, t.TempDir(), runtimeID)
			policy, previews, _ := policyFixture(t, &graph, runtimeID, executable, "synthetic:credential")
			credentials := &countedCredentials{}
			profiles := []runtimeadapter.CommandProfile{}
			for _, child := range graph.Children {
				profiles = append(profiles, runtimeadapter.CommandProfile{RuntimeID: runtimeID, ModelProfileID: child.Envelope.Resolution.ModelProfileID, Executable: executable, Model: "local-test-profile", CredentialReference: "synthetic:credential", OutputMax: 4096})
			}
			for _, option := range options {
				for _, arguments := range [][]string{{option, "other-model"}, {option + "=other-model"}} {
					t.Run(strings.Join(arguments, " "), func(t *testing.T) {
						candidate := append([]runtimeadapter.CommandProfile(nil), profiles...)
						candidate[0].Arguments = arguments
						adapter, err := runtimeadapter.NewInvocationResolver(candidate, credentials)
						if err != nil {
							t.Fatal(err)
						}
						_, err = newPolicyInvocations(LocalConfiguration{Graph: graph, RuntimeProfiles: candidate, RuntimePolicy: policy, RuntimePreviews: previews}, adapter)
						if !errors.Is(err, ErrInvalidComposition) || credentials.calls != 0 {
							t.Fatalf("err=%v credentials=%d", err, credentials.calls)
						}
					})
				}
			}
		})
	}
}

func TestProductionDispatchStalePolicyHasNoAttemptsOrCredentials(t *testing.T) {
	for _, runtimeID := range []string{"codex", "claude"} {
		t.Run(runtimeID, func(t *testing.T) {
			root := canonicalTempDir(t)
			repository, revision := initRepository(t, filepath.Join(root, "repository"))
			graph := buildGraph(t, filepath.Join(root, "workspaces"))
			executable, marker := stubRuntime(t, filepath.Join(root, "bin"), runtimeID)
			policy, previews, source := policyFixture(t, &graph, runtimeID, executable, "synthetic:credential")
			credentials := &countedCredentials{}
			profiles := []runtimeadapter.CommandProfile{}
			for _, child := range graph.Children {
				profiles = append(profiles, runtimeadapter.CommandProfile{RuntimeID: runtimeID, ModelProfileID: child.Envelope.Resolution.ModelProfileID, Executable: executable, Model: "local-test-profile", CredentialReference: "synthetic:credential", OutputMax: 4096})
			}
			gitPath, err := exec.LookPath("git")
			if err != nil {
				t.Fatal(err)
			}
			store := &memoryGraphStore{wire: mustEncodeGraph(t, graph)}
			before := string(store.wire)
			allocations := 0
			service, err := NewLocalService(context.Background(), LocalConfiguration{
				Repository: repository, WorkspaceRoot: filepath.Join(root, "workspaces"), BaseRevision: revision,
				Graph: graph, GraphStore: store, RuntimeProfiles: profiles, Credentials: credentials, RuntimePolicy: policy, RuntimePreviews: previews,
				Validators:        []gitworkspace.ValidationCommand{{Reference: "diff-check", Argv: []string{gitPath, "diff", "--check"}, OutputMax: 4096}},
				AllocateAttemptID: func() (string, error) { allocations++; return "attempt-1", nil },
			})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := service.PrepareWorkspaces(context.Background()); err != nil {
				t.Fatal(err)
			}
			source.snapshot.Configuration.Revision++
			result, err := service.DispatchReady(context.Background(), nil, false)
			if err != nil || len(result.Records) != 0 || allocations != 0 || credentials.calls != 0 || string(store.wire) != before {
				t.Fatalf("records=%v allocations=%d credentials=%d persistedChanged=%t err=%v", result.Records, allocations, credentials.calls, string(store.wire) != before, err)
			}
			for _, child := range result.Graph.Children {
				if len(child.Attempts) != 0 {
					t.Fatal("blocked dispatch created attempts")
				}
			}
			if len(result.Blocked) != len(graph.Children) {
				t.Fatalf("blocked=%v", result.Blocked)
			}
			if _, err := os.Lstat(marker); !os.IsNotExist(err) {
				t.Fatal("stale dispatch started the Runtime process")
			}
		})
	}
}

func TestProductionDispatchUsesPreviewedModelBothAdapters(t *testing.T) {
	root := canonicalTempDir(t)
	// This isolated executable only validates adapter argv. It never invokes a
	// vendor runtime, reads credentials or performs repository writes.
	sourcePath := filepath.Join(root, "runtime-helper.go")
	source := `package main
import ("fmt"; "os"; "path/filepath"; "strings")
func main() {
 name := strings.TrimSuffix(filepath.Base(os.Args[0]), ".exe")
 prefix := "exec"
 if name == "claude" { prefix = "--print" }
 if len(os.Args) != 4 || os.Args[1] != prefix || os.Args[2] != "--model" || os.Args[3] != "local-test-profile" { fmt.Fprintln(os.Stderr, "unexpected adapter selection"); os.Exit(1) }
 fmt.Println("synthetic-runtime-ok")
}`
	if err := os.WriteFile(sourcePath, []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	for _, runtimeID := range []string{"codex", "claude"} {
		t.Run(runtimeID, func(t *testing.T) {
			binaryName := runtimeID
			if runtime.GOOS == "windows" {
				binaryName += ".exe"
			}
			executable := filepath.Join(root, binaryName)
			if output, err := exec.Command("go", "build", "-o", executable, sourcePath).CombinedOutput(); err != nil {
				t.Fatalf("build synthetic runtime: %v: %s", err, output)
			}
			caseRoot := filepath.Join(root, runtimeID+"-case")
			repository, revision := initRepository(t, filepath.Join(caseRoot, "repository"))
			graph := buildGraph(t, filepath.Join(caseRoot, "workspaces"))
			policy, previews, _ := policyFixture(t, &graph, runtimeID, executable, "")
			profiles := []runtimeadapter.CommandProfile{}
			for _, child := range graph.Children {
				profiles = append(profiles, runtimeadapter.CommandProfile{RuntimeID: runtimeID, ModelProfileID: child.Envelope.Resolution.ModelProfileID, Executable: executable, Model: "local-test-profile", OutputMax: 4096})
			}
			gitPath, err := exec.LookPath("git")
			if err != nil {
				t.Fatal(err)
			}
			store := &memoryGraphStore{wire: mustEncodeGraph(t, graph)}
			service, err := NewLocalService(context.Background(), LocalConfiguration{
				Repository: repository, WorkspaceRoot: filepath.Join(caseRoot, "workspaces"), BaseRevision: revision,
				Graph: graph, GraphStore: store, RuntimeProfiles: profiles, RuntimePolicy: policy, RuntimePreviews: previews,
				Validators: []gitworkspace.ValidationCommand{{Reference: "diff-check", Argv: []string{gitPath, "diff", "--check"}, OutputMax: 4096}},
			})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := service.PrepareWorkspaces(context.Background()); err != nil {
				t.Fatal(err)
			}
			result, err := service.DispatchReady(context.Background(), nil, false)
			if err != nil || len(result.Records) != 2 {
				t.Fatalf("result=%+v err=%v", result, err)
			}
			for _, record := range result.Records {
				if record.Status != executiongraph.AttemptSucceeded {
					t.Fatalf("record=%+v", record)
				}
			}
			persisted, err := store.Load(context.Background(), "", "")
			if err != nil {
				t.Fatal(err)
			}
			for _, child := range persisted.Children {
				choice := previews[child.ExecutionID].Choice
				if child.Envelope.Resolution.RuntimeID != choice.RuntimeID || child.Envelope.Resolution.ModelProfileID != choice.ModelProfileID {
					t.Fatalf("persisted resolution=%+v choice=%+v", child.Envelope.Resolution, choice)
				}
				if child.Envelope.IntegrationOwner {
					if len(child.Attempts) != 0 {
						t.Fatal("integration owner dispatched")
					}
					if len(result.Blocked) != 1 || result.Blocked[child.ExecutionID] != "integration_pending" {
						t.Fatalf("unexpected blockers: %v", result.Blocked)
					}
					continue
				}
				if len(child.Attempts) != 1 || child.Attempts[0].Status != executiongraph.AttemptSucceeded {
					t.Fatalf("attempts=%+v", child.Attempts)
				}
				if _, blocked := result.Blocked[child.ExecutionID]; blocked {
					t.Fatalf("successful child blocked: %v", result.Blocked)
				}
			}
		})
	}
}

// CR-002/CR-003: the reviewed preview binds the concrete executable and the
// configured credential reference. Any divergence between that binding and the
// command profile, or any Runtime drift after preview, blocks before credential
// resolution, attempt allocation, process start or persisted graph change.
func TestProductionDispatchBlocksUnreviewedBindingWithZeroEffects(t *testing.T) {
	scenarios := []struct {
		name   string
		mutate func(t *testing.T, executable string, profiles []runtimeadapter.CommandProfile, source *policySource) []string
	}{
		{"credential reference differs from reviewed configuration", func(_ *testing.T, _ string, profiles []runtimeadapter.CommandProfile, _ *policySource) []string {
			for index := range profiles {
				profiles[index].CredentialReference = "synthetic:credential-b"
			}
			return nil
		}},
		{"executable replaced after preview", func(t *testing.T, executable string, _ []runtimeadapter.CommandProfile, _ *policySource) []string {
			if err := os.WriteFile(executable, []byte("#!/bin/sh\n# replaced\n"), 0o700); err != nil {
				t.Fatal(err)
			}
			return nil
		}},
		{"executable removed after preview", func(t *testing.T, executable string, _ []runtimeadapter.CommandProfile, _ *policySource) []string {
			if err := os.Remove(executable); err != nil {
				t.Fatal(err)
			}
			return nil
		}},
		{"command profile names another executable", func(t *testing.T, executable string, profiles []runtimeadapter.CommandProfile, _ *policySource) []string {
			other, marker := stubRuntime(t, filepath.Join(filepath.Dir(executable), "other"), filepath.Base(executable))
			for index := range profiles {
				profiles[index].Executable = other
			}
			return []string{marker}
		}},
		{"Runtime unavailable after preview", func(t *testing.T, _ string, _ []runtimeadapter.CommandProfile, source *policySource) []string {
			observation, err := source.snapshot.Observer.Observe(context.Background(), source.snapshot.Configuration.Runtimes[0].ID)
			if err != nil {
				t.Fatal(err)
			}
			observation.Available = false
			inventory, err := runtimeadapter.NewInventory([]runtimeprofile.Observation{observation})
			if err != nil {
				t.Fatal(err)
			}
			source.snapshot.Observer = inventory
			return nil
		}},
	}
	for _, runtimeID := range []string{"codex", "claude"} {
		for _, scenario := range scenarios {
			t.Run(runtimeID+"/"+scenario.name, func(t *testing.T) {
				root := canonicalTempDir(t)
				repository, revision := initRepository(t, filepath.Join(root, "repository"))
				graph := buildGraph(t, filepath.Join(root, "workspaces"))
				executable, marker := stubRuntime(t, filepath.Join(root, "bin"), runtimeID)
				policy, previews, source := policyFixture(t, &graph, runtimeID, executable, "synthetic:credential-a")
				profiles := []runtimeadapter.CommandProfile{}
				for _, child := range graph.Children {
					profiles = append(profiles, runtimeadapter.CommandProfile{RuntimeID: runtimeID, ModelProfileID: child.Envelope.Resolution.ModelProfileID, Executable: executable, Model: "local-test-profile", CredentialReference: "synthetic:credential-a", OutputMax: 4096})
				}
				markers := append([]string{marker}, scenario.mutate(t, executable, profiles, source)...)
				gitPath, err := exec.LookPath("git")
				if err != nil {
					t.Fatal(err)
				}
				credentials := &countedCredentials{}
				store := &memoryGraphStore{wire: mustEncodeGraph(t, graph)}
				before := string(store.wire)
				allocations := 0
				service, err := NewLocalService(context.Background(), LocalConfiguration{
					Repository: repository, WorkspaceRoot: filepath.Join(root, "workspaces"), BaseRevision: revision,
					Graph: graph, GraphStore: store, RuntimeProfiles: profiles, Credentials: credentials, RuntimePolicy: policy, RuntimePreviews: previews,
					Validators:        []gitworkspace.ValidationCommand{{Reference: "diff-check", Argv: []string{gitPath, "diff", "--check"}, OutputMax: 4096}},
					AllocateAttemptID: func() (string, error) { allocations++; return "attempt-1", nil },
				})
				if err != nil {
					t.Fatal(err)
				}
				if _, err := service.PrepareWorkspaces(context.Background()); err != nil {
					t.Fatal(err)
				}
				result, err := service.DispatchReady(context.Background(), nil, false)
				if err != nil || len(result.Records) != 0 || allocations != 0 || credentials.calls != 0 || string(store.wire) != before || len(result.Blocked) != len(graph.Children) {
					t.Fatalf("records=%v blocked=%v allocations=%d credentials=%d persistedChanged=%t err=%v", result.Records, result.Blocked, allocations, credentials.calls, string(store.wire) != before, err)
				}
				for _, child := range result.Graph.Children {
					if len(child.Attempts) != 0 {
						t.Fatal("blocked dispatch created attempts")
					}
				}
				for _, marker := range markers {
					if _, err := os.Lstat(marker); !os.IsNotExist(err) {
						t.Fatal("blocked dispatch started a Runtime process")
					}
				}
			})
		}
	}
}

// The reviewed binding itself still dispatches, so the blocks above come from
// divergence rather than a guard that rejects every credential reference.
func TestPolicyInvocationResolvesOnlyTheReviewedCredentialReference(t *testing.T) {
	graph := buildGraph(t, canonicalTempDir(t))
	executable, _ := stubRuntime(t, t.TempDir(), "claude")
	policy, previews, _ := policyFixture(t, &graph, "claude", executable, "synthetic:credential-a")
	for reference, want := range map[string]int{"synthetic:credential-a": 1, "synthetic:credential-b": 0, "": 0} {
		credentials := &countedCredentials{}
		profiles := []runtimeadapter.CommandProfile{}
		for _, child := range graph.Children {
			profiles = append(profiles, runtimeadapter.CommandProfile{RuntimeID: "claude", ModelProfileID: child.Envelope.Resolution.ModelProfileID, Executable: executable, Model: "local-test-profile", CredentialReference: reference, OutputMax: 4096})
		}
		adapter, err := runtimeadapter.NewInvocationResolver(profiles, credentials)
		if err != nil {
			t.Fatal(err)
		}
		guard, err := newPolicyInvocations(LocalConfiguration{Graph: graph, RuntimePolicy: policy, RuntimePreviews: previews, RuntimeProfiles: profiles}, adapter)
		if err != nil {
			t.Fatal(err)
		}
		_, err = guard.ResolveInvocation(context.Background(), graph.Children[0])
		if (err == nil) != (want == 1) || credentials.calls != want {
			t.Fatalf("reference=%q err=%v credentials=%d", reference, err, credentials.calls)
		}
	}
}
