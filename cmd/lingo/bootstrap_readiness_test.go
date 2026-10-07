package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/cli"
	"github.com/rgomids/axiom/internal/completion"
	"github.com/rgomids/axiom/internal/local"
	"github.com/rgomids/axiom/internal/projectapp"
	"github.com/rgomids/axiom/internal/runtimeprofile"
)

// Issue #231 black-box Evidence: explicit-location bootstrap, portable/local
// separation, operation-scoped readiness and pre-effect enforcement.

type bootstrapEnv struct {
	root, state, workspace string
	service                cli.Service
}

func newBootstrapEnv(t *testing.T) bootstrapEnv {
	t.Helper()
	env := bootstrapEnv{root: filepath.Join(t.TempDir(), "projects"), state: filepath.Join(t.TempDir(), "state"), workspace: t.TempDir()}
	t.Setenv("LINGO_PROJECTS_ROOT", env.root)
	t.Setenv("LINGO_STATE_ROOT", env.state)
	t.Setenv("AXIOM_GH_BIN", filepath.Join(t.TempDir(), "gh-never-executed"))
	env.service = compose()
	return env
}

func writeTree(t *testing.T, base string, files map[string]string) string {
	t.Helper()
	for name, content := range files {
		path := filepath.Join(base, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return base
}

type readinessEvent struct {
	Status    string                         `json:"status"`
	Result    string                         `json:"result"`
	Setup     *projectapp.SetupPreview       `json:"setup"`
	Readiness *projectapp.ReadinessReport    `json:"readiness"`
	Preflight *projectapp.OperationReadiness `json:"preflight"`
}

func runEvent(t *testing.T, service cli.Service, wantCode int, args ...string) (readinessEvent, string) {
	t.Helper()
	var output bytes.Buffer
	if code := cli.Run(context.Background(), args, service, currentProvenance(), &output); code != wantCode {
		t.Fatalf("%v: exit %d output=%s", args, code, output.String())
	}
	var event readinessEvent
	if err := json.Unmarshal(output.Bytes(), &event); err != nil {
		t.Fatalf("%v: %v %s", args, err, output.String())
	}
	return event, output.String()
}

func blockerCodes(findings []projectapp.Finding) string {
	codes := []string{}
	for _, finding := range findings {
		codes = append(codes, finding.Code+":"+finding.Subject)
	}
	return strings.Join(codes, ",")
}

func TestBootstrapFromExplicitRepositoriesResolvesAmbiguityAndSeparatesLocalState(t *testing.T) {
	env := newBootstrapEnv(t)
	core := writeTree(t, filepath.Join(env.workspace, "core"), map[string]string{
		".git/config":                "[remote \"origin\"]\n\turl = git@github.com:acme/core.git\n[remote \"upstream\"]\n\turl = https://github.com/acme/core.git\n",
		"go.mod":                     "module synthetic\n",
		"docs/architecture/index.md": "synthetic architecture",
	})
	web := writeTree(t, filepath.Join(env.workspace, "Web"), map[string]string{
		".git/config":  "[remote \"alias-a\"]\n\turl = https://github.com/acme/web.git\n[remote \"alias-b\"]\n\turl = https://GitHub.com/acme/web.git\n",
		"Dockerfile":   "FROM scratch\n",
		"package.json": "{}", "pnpm-lock.yaml": "", "package-lock.json": "",
	})
	notes := writeTree(t, filepath.Join(env.workspace, "private"), map[string]string{"product.md": "SYNTHETIC-PRIVATE-NOTES-BODY"})
	notesPath := filepath.Join(notes, "product.md")
	// The caller's working directory is another checkout; it must never be used.
	cwd := writeTree(t, filepath.Join(env.workspace, "cwd"), map[string]string{".git/config": "[remote \"origin\"]\n\turl = https://github.com/acme/cwd-should-not-appear.git\n"})
	wd, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(wd) })
	if err := os.Chdir(cwd); err != nil {
		t.Fatal(err)
	}
	base := []string{"project", "configure", "--slug", "multi", "--name", "Multi",
		"--repository", "core=" + core, "--repository", web, "--work-item-provider", "github",
		"--documentation", "architecture=repository:core/docs/architecture", "--documentation", "notes=local-file:" + notesPath,
		"--business-context", "Bounded synthetic context.", "--context-source", "architecture",
		"--glossary", "work-item=Work Item:A bounded unit of intent", "--technology", "cloud.aws=aws", "--remove-technology", "package-manager.npm"}
	before := snapshotTrees(t, env.workspace)
	ambiguous, _ := runEvent(t, env.service, cli.ExitSuccess, base...)
	if ambiguous.Setup == nil || blockerCodes(ambiguous.Setup.Blockers) != "repository_remote_ambiguous:core" {
		t.Fatalf("ambiguity not reported: %+v", ambiguous.Setup)
	}
	for _, repository := range ambiguous.Setup.Repositories {
		if repository.Key == "web" && (repository.KeySource != "derived" || repository.Remote.Status != "single" || len(repository.Remote.Candidates) != 1 || len(repository.Remote.Candidates[0].Names) != 2) {
			t.Fatalf("aliases did not collapse: %+v", repository.Remote)
		}
		if repository.Key == "core" && (len(repository.Remote.Candidates) != 2 || repository.Remote.Locator != "") {
			t.Fatalf("ambiguous remote chosen: %+v", repository.Remote)
		}
	}
	// Even with authority, a blocked proposal never publishes.
	refused, _ := runEvent(t, env.service, cli.ExitFailure, append(base, "--project-id", ambiguous.Setup.ProjectID, "--preview-digest", ambiguous.Setup.Digest, "--authorize-local")...)
	if refused.Result != "Project bootstrap has unresolved blockers" {
		t.Fatalf("blocked publish result %q", refused.Result)
	}
	for _, root := range []string{env.root, env.state} {
		if _, err := os.Stat(root); !os.IsNotExist(err) {
			t.Fatalf("blocked bootstrap wrote %s: %v", root, err)
		}
	}
	chosen := append(base, "--repository-remote", "core=https://github.com/acme/core.git")
	preview, _ := runEvent(t, env.service, cli.ExitSuccess, chosen...)
	if len(preview.Setup.Blockers) != 0 || preview.Setup.SchemaVersion != 3 {
		t.Fatalf("explicit remote not applied: %+v", preview.Setup.Blockers)
	}
	published, _ := runEvent(t, env.service, cli.ExitSuccess, append(chosen, "--project-id", preview.Setup.ProjectID, "--preview-digest", preview.Setup.Digest, "--authorize-local")...)
	if published.Result != "Project setup published" {
		t.Fatalf("publish result %q", published.Result)
	}
	if after := snapshotTrees(t, env.workspace); !bytes.Equal(before, after) {
		t.Fatal("bootstrap wrote inside Repository or documentation locations")
	}
	manifest, err := os.ReadFile(filepath.Join(env.root, "multi", "axiom.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{env.workspace, notesPath, "cwd-should-not-appear", "SYNTHETIC-PRIVATE-NOTES-BODY", "package-manager.npm", "codex"} {
		if strings.Contains(string(manifest), forbidden) {
			t.Fatalf("portable manifest contains %q:\n%s", forbidden, manifest)
		}
	}
	for _, required := range []string{"schemaVersion: 3", "remote: https://github.com/acme/core.git", "remote: https://github.com/acme/web.git", "key: language.go", "key: container.docker", "key: cloud.aws", "key: package-manager.pnpm", "kind: local-file", "path: docs/architecture", "glossary:"} {
		if !strings.Contains(string(manifest), required) {
			t.Fatalf("portable manifest lacks %q:\n%s", required, manifest)
		}
	}
	record, err := os.ReadFile(filepath.Join(env.state, "projects", preview.Setup.ProjectID, "installation.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(record, []byte(`"formatVersion":2`)) || !bytes.Contains(record, []byte(`"sourceKey":"notes"`)) || bytes.Contains(record, []byte("SYNTHETIC-PRIVATE-NOTES-BODY")) {
		t.Fatalf("local documentation binding wrong: %s", record)
	}

	// Readiness: Work Item ready, Execution blocked by absent Runtime policy.
	validated, output := runEvent(t, env.service, cli.ExitSuccess, "project", "validate", "--slug", "multi")
	report := validated.Readiness
	if validated.Result != "Project is valid" || report == nil || report.Effective != "partial" || report.SchemaVersion != 3 {
		t.Fatalf("readiness = %+v", report)
	}
	if work, exec := report.Operation(projectapp.OperationWorkItem), report.Operation(projectapp.OperationExecution); !work.Ready() || blockerCodes(exec.Blockers) != "runtime_policy_unavailable:" {
		t.Fatalf("operations = %+v / %+v", work, exec)
	}
	for _, source := range report.Documentation {
		if source.Status != projectapp.DocumentationAvailable {
			t.Fatalf("documentation %s = %s", source.Key, source.Status)
		}
	}
	for _, leak := range []string{env.workspace, "SYNTHETIC-PRIVATE-NOTES-BODY"} {
		if strings.Contains(output, leak) {
			t.Fatalf("readiness leaked %q", leak)
		}
	}
	// A replaced local document warns without blocking unrelated operations.
	if err := os.Remove(notesPath); err != nil {
		t.Fatal(err)
	}
	writeTree(t, filepath.Join(env.workspace, "other"), map[string]string{"x.md": "x"})
	writeTree(t, notes, map[string]string{"product.md": "replacement"})
	stale, _ := runEvent(t, env.service, cli.ExitSuccess, "project", "validate", "--slug", "multi")
	if !strings.Contains(blockerCodes(stale.Readiness.Warnings), "documentation_stale:notes") || !stale.Readiness.Operation(projectapp.OperationWorkItem).Ready() {
		t.Fatalf("stale document not reported as warning: %+v", stale.Readiness.Warnings)
	}
}

func TestBootstrapAuthorsRuntimePolicyFromLocalCandidatesOnly(t *testing.T) {
	env := newBootstrapEnv(t)
	repository := writeTree(t, filepath.Join(env.workspace, "main"), map[string]string{"README.md": "x"})
	store, err := local.NewRuntimeProfileStore(env.state)
	if err != nil {
		t.Fatal(err)
	}
	cfg := runtimeprofile.Configuration{FormatVersion: 1, Revision: 1, Runtimes: []runtimeprofile.Runtime{{ID: "claude", Adapter: "claude", Enabled: true, AllowlistedProfileIDs: []string{"worker"}, CredentialReference: "private-test-reference"}}, ModelProfiles: []runtimeprofile.ModelProfile{{ID: "worker", RuntimeID: "claude", Model: "approved-model", Capabilities: []string{"axiom-skills"}, Complexities: []string{"high"}}}}
	if err := store.Create(context.Background(), cfg); err != nil {
		t.Fatal(err)
	}
	base := []string{"project", "configure", "--slug", "policy", "--name", "Policy", "--repository", "main=" + repository, "--work-item-provider", "github"}
	none, _ := runEvent(t, env.service, cli.ExitSuccess, base...)
	if none.Setup.RuntimePolicy.Status != "absent" || len(none.Setup.RuntimePolicy.Candidates) != 1 || none.Setup.RuntimePolicy.Candidates[0].ID != "claude" {
		t.Fatalf("candidates = %+v", none.Setup.RuntimePolicy)
	}
	// A Runtime that is not locally configured (Codex here) is never accepted.
	invalid, _ := runEvent(t, env.service, cli.ExitFailure, append(base, "--runtime", "codex")...)
	if invalid.Result != "Project setup Runtime policy input is invalid" {
		t.Fatalf("non-local runtime accepted: %q", invalid.Result)
	}
	chosen := append(base, "--runtime", "claude", "--model-profile", "worker", "--runtime-preference", "implementation/high=worker")
	preview, output := runEvent(t, env.service, cli.ExitSuccess, chosen...)
	if strings.Contains(output, "private-test-reference") || preview.Setup.RuntimePolicy.Status != "configured" {
		t.Fatalf("policy preview = %+v", preview.Setup.RuntimePolicy)
	}
	runEvent(t, env.service, cli.ExitSuccess, append(chosen, "--project-id", preview.Setup.ProjectID, "--preview-digest", preview.Setup.Digest, "--authorize-local")...)
	manifest, err := os.ReadFile(filepath.Join(env.root, "policy", "axiom.yaml"))
	if err != nil || !strings.Contains(string(manifest), "runtimes:\n  - id: claude") || !strings.Contains(string(manifest), "model: approved-model") || strings.Contains(string(manifest), "private-test-reference") {
		t.Fatalf("authored policy = %s %v", manifest, err)
	}
	// With the Runtime observable, both operations are ready without editing axiom.yaml.
	service, _ := testRuntime(t, env.service.(lifecycleService), "claude")
	report := service.readiness().Evaluate(context.Background(), "policy")
	if report.Effective != "ready" || report.Runtime.Status != "ready" {
		t.Fatalf("readiness = %+v", report)
	}
	// Removing the executable blocks Execution only; no other Runtime is chosen.
	missing := service
	missing.runtimes.lookPath = func(string) (string, error) { return "", os.ErrNotExist }
	blocked := missing.readiness().Evaluate(context.Background(), "policy")
	if blocked.Effective != "partial" || blocked.Runtime.Detail != "runtime_unavailable" || !blocked.Operation(projectapp.OperationWorkItem).Ready() {
		t.Fatalf("blocked = %+v", blocked)
	}
}

func TestBlockedPreflightPerformsNoEffects(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Provider stub is a POSIX script")
	}
	env := newBootstrapEnv(t)
	calls := filepath.Join(t.TempDir(), "gh-calls")
	gh := filepath.Join(t.TempDir(), "gh")
	if err := os.WriteFile(gh, []byte("#!/bin/sh\necho called >> '"+calls+"'\nexit 1\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AXIOM_GH_BIN", gh)
	env.service = compose()
	repository := writeTree(t, filepath.Join(env.workspace, "main"), map[string]string{"README.md": "x"})
	for _, provider := range []string{"", "linear"} {
		slug := "blocked" + provider
		configureProject(t, env.service, slug, "Blocked", "main="+repository, provider)
		before := snapshotTrees(t, env.root, env.state, env.workspace)
		want := map[string]string{"": "capability_mapping_missing", "linear": "provider_unsupported"}[provider]
		service := env.service.(lifecycleService)
		results := map[string]cli.Result{
			"create":  service.WorkItemCreate(context.Background(), cli.WorkItemInput{Project: slug, Repository: "main", ProviderRepository: "owner/repo", Intent: "x", Problem: "x", DesiredOutcome: "x", Context: "x", Scope: "x", Constraints: "x", NonGoals: "x", Acceptance: "x", PreviewDigest: "any", AuthorizeExternal: true}),
			"select":  service.WorkItemSelect(context.Background(), cli.WorkItemInput{Project: slug, Repository: "main", ProviderRepository: "owner/repo", Number: 1, PreviewDigest: "any", AuthorizeLocal: true}),
			"comment": service.WorkItemComment(context.Background(), cli.WorkItemInput{Project: slug, Repository: "main", Number: 1, Message: "x", AuthorizeExternal: true}),
			"start":   service.WorkflowStart(context.Background(), cli.WorkflowInput{Project: slug, Repository: "main", Number: 1, Role: "implementation", Complexity: "high", Capabilities: []string{"axiom-skills"}, RuntimePreview: "any"}),
			"resume":  service.WorkflowResume(context.Background(), cli.WorkflowInput{Project: slug, Repository: "main", Number: 1, ExpectedRevision: 1}),
		}
		for name, result := range results {
			if result.Completion == nil || result.Completion.Status() != completion.ValidationFailure || result.Category != want || result.Preflight == nil {
				t.Fatalf("%s %s: %+v", provider, name, result)
			}
		}
		// The same blocker code is what project validate reports.
		validated, _ := runEvent(t, env.service, cli.ExitSuccess, "project", "validate", "--slug", slug)
		if validated.Readiness.Operation(projectapp.OperationWorkItem).Blockers[0].Code != want {
			t.Fatalf("validate disagrees with preflight: %+v", validated.Readiness)
		}
		if after := snapshotTrees(t, env.root, env.state, env.workspace); !bytes.Equal(before, after) {
			t.Fatal("blocked operation changed local state")
		}
		if _, err := os.Stat(calls); !os.IsNotExist(err) {
			t.Fatal("blocked operation invoked the Provider")
		}
	}
}

func TestReadinessReadsOlderSchemasWithoutRewriting(t *testing.T) {
	env := newBootstrapEnv(t)
	runCLI(t, env.service, []string{"project", "init", "--slug", "legacy", "--name", "Legacy"}, cli.ExitSuccess, "applied")
	source := filepath.Join(env.root, "legacy")
	runCLI(t, env.service, []string{"project", "install", "--source", source}, cli.ExitSuccess, "installed")
	before := snapshotTrees(t, env.root, env.state)
	validated, _ := runEvent(t, env.service, cli.ExitSuccess, "project", "validate", "--slug", "legacy")
	if validated.Readiness == nil || validated.Readiness.SchemaVersion != 1 || validated.Readiness.Effective != "blocked" || validated.Readiness.Operation(projectapp.OperationWorkItem).Blockers[0].Code != "capability_mapping_missing" {
		t.Fatalf("v1 readiness = %+v", validated.Readiness)
	}
	if after := snapshotTrees(t, env.root, env.state); !bytes.Equal(before, after) {
		t.Fatal("readiness rewrote or migrated v1 state")
	}
	missing, _ := runEvent(t, env.service, cli.ExitFailure, "project", "validate", "--slug", "absent")
	if missing.Readiness != nil {
		t.Fatal("invalid Project reported readiness")
	}
}

func TestGuidedBootstrapAsksOnlyForUnresolvedRemote(t *testing.T) {
	env := newBootstrapEnv(t)
	core := writeTree(t, filepath.Join(env.workspace, "core"), map[string]string{
		".git/config": "[remote \"origin\"]\n\turl = https://github.com/acme/fork.git\n[remote \"upstream\"]\n\turl = https://github.com/acme/core.git\n",
	})
	// Explicit identity and Repository; only the Work Item Provider and the
	// ambiguous remote are unresolved, so only those are asked.
	answers := strings.NewReader("github\n2\nyes\n")
	var output, prompts bytes.Buffer
	code := cli.RunInteractive(context.Background(), []string{"--json", "project", "configure", "--slug", "guided", "--name", "Guided", "--repository", "core=" + core}, env.service, currentProvenance(), answers, &output, &prompts)
	if code != cli.ExitSuccess {
		t.Fatalf("exit %d output=%s prompts=%s", code, output.String(), prompts.String())
	}
	for _, want := range []string{"Work Item provider (github or none): ", "1) https://github.com/acme/core.git", "2) https://github.com/acme/fork.git", "Remote for core (number, locator, or none): ", "Publish this exact proposal? [yes/no]: "} {
		if !strings.Contains(prompts.String(), want) {
			t.Fatalf("prompt %q missing: %s", want, prompts.String())
		}
	}
	for _, unexpected := range []string{"Project slug: ", "Project name: ", "Repository key=absolute-path"} {
		if strings.Contains(prompts.String(), unexpected) {
			t.Fatalf("asked for already supplied intent %q", unexpected)
		}
	}
	manifest, err := os.ReadFile(filepath.Join(env.root, "guided", "axiom.yaml"))
	if err != nil || !strings.Contains(string(manifest), "remote: https://github.com/acme/fork.git") {
		t.Fatalf("operator choice not published: %s %v", manifest, err)
	}
}
