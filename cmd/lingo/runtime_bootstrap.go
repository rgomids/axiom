package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/rgomids/axiom/internal/cli"
	"github.com/rgomids/axiom/internal/codexruntime"
	"github.com/rgomids/axiom/internal/completion"
	"github.com/rgomids/axiom/internal/provenance"
	"github.com/rgomids/axiom/internal/runtimeadapter"
	"github.com/rgomids/axiom/internal/runtimebootstrap"
)

// runtimeRoots holds the resolved user-global roots for Runtime discovery and
// integration. An unsafe Claude configuration directory fails only Claude
// operations, never the rest of the CLI.
type runtimeRoots struct {
	lookPath          runtimebootstrap.LookPath
	codexConfigRoot   string
	claudeConfigRoot  string
	claudeSkillsRoot  string
	claudeSkillsError error
}

func discoverRuntimeRoots() runtimeRoots {
	home, err := os.UserHomeDir()
	if err != nil || !filepath.IsAbs(home) {
		home = ""
	}
	roots := runtimeRoots{lookPath: exec.LookPath}
	if home != "" {
		roots.codexConfigRoot = filepath.Join(home, ".codex")
	}
	roots.claudeConfigRoot, _ = runtimebootstrap.ClaudeConfigurationRoot(os.Getenv, home)
	roots.claudeSkillsRoot, roots.claudeSkillsError = runtimebootstrap.ClaudeSkillsRoot(os.Getenv, home)
	return roots
}

func (r runtimeRoots) claude() (codexruntime.Service, error) {
	if r.claudeSkillsError != nil {
		return codexruntime.Service{}, r.claudeSkillsError
	}
	return codexruntime.NewClaude(r.claudeSkillsRoot)
}

func (s lifecycleService) RuntimeClaudeInstall(ctx context.Context) cli.Result {
	service, err := s.runtimes.claude()
	if err != nil {
		return cli.Result{Status: cli.Failed, Category: "claude_skill_root_unavailable"}
	}
	return runtimeResult(service.Install(ctx))
}

func (s lifecycleService) RuntimeClaudeStatus(ctx context.Context) cli.Result {
	service, err := s.runtimes.claude()
	if err != nil {
		return canonicalCompletion(completion.Facts{ValidationFailed: true}, "Claude configuration directory is unsafe", nil, "Set CLAUDE_CONFIG_DIR to an absolute path or unset it", s.provenance)
	}
	return runtimeStatus(service.Inspect(ctx), "Claude", "claude", s.provenance)
}

// firstRunConflictNext is the deterministic next step for a Runtime whose
// integration is blocked by preserved artifacts listed as conflict lines.
const firstRunConflictNext = "Review each reported conflict artifact in that Runtime's skill root; Axiom preserves it and never overwrites it. Keep it, or move it aside if it is not yours, then run axiom first-run again"

// FirstRun discovers Codex and Claude by executable and converges Axiom's
// user-global integration for each one present. Absence is a valid state.
func (s lifecycleService) FirstRun(ctx context.Context) cli.Result {
	runtimes := []runtimebootstrap.Runtime{
		{ID: "codex", Executable: "codex", ConfigurationRoot: s.runtimes.codexConfigRoot, Integration: func() (runtimebootstrap.Integration, error) { return s.codex, nil }},
		{ID: "claude", Executable: "claude", ConfigurationRoot: s.runtimes.claudeConfigRoot, Integration: func() (runtimebootstrap.Integration, error) { return s.runtimes.claude() }},
	}
	report := runtimebootstrap.Run(ctx, s.runtimes.lookPath, runtimes)
	view := bootstrapView(report)
	references := []string{}
	cancelled := false
	for _, runtime := range report.Runtimes {
		if runtime.Present {
			references = append(references, "runtime:"+runtime.ID+":"+string(runtime.State))
		}
		cancelled = cancelled || runtime.Category == "cancelled"
	}
	var response cli.Result
	switch {
	case cancelled:
		response = canonicalCompletion(completion.Facts{WasInterrupted: true}, "First run was interrupted", references, "Run axiom first-run again", s.provenance)
	case report.Detected() == 0:
		response = canonicalCompletion(completion.Facts{Completed: true}, "No supported Runtime is currently available", nil, "Install Codex or Claude yourself if needed, then run axiom first-run again; Project setup does not require a Runtime", s.provenance)
	case report.Failed() == 0:
		response = canonicalCompletion(completion.Facts{Completed: true}, "Axiom integration is configured for every detected Runtime", references, "Run project configure with explicit Project inputs", s.provenance)
	case report.Failed() < report.Detected():
		response = canonicalCompletion(completion.Facts{RequestedEffectConfirmed: true, SecondaryFailure: true}, "Axiom integration is configured for some detected Runtimes", references, firstRunConflictNext, s.provenance)
	default:
		response = canonicalCompletion(completion.Facts{Failed: true}, "Axiom integration could not be configured for any detected Runtime", references, firstRunConflictNext, s.provenance)
	}
	response.Bootstrap = &view
	return response
}

func bootstrapView(report runtimebootstrap.Report) cli.BootstrapView {
	view := cli.BootstrapView{Detected: report.Detected(), Failed: report.Failed(), Runtimes: make([]cli.RuntimeBootstrapView, 0, len(report.Runtimes))}
	for _, runtime := range report.Runtimes {
		entry := cli.RuntimeBootstrapView{Runtime: runtime.ID, Executable: runtime.Executable, Present: runtime.Present, ConfigurationWithoutExecutable: runtime.ConfigurationWithoutExecutable, State: string(runtime.State), Reason: runtime.Category}
		if runtime.Result != nil {
			skills := runtimeView(*runtime.Result)
			entry.SkillSetVersion, entry.Skills, entry.Receipt, entry.Conflicts = skills.SkillSetVersion, skills.Skills, skills.Receipt, skills.Conflicts
		}
		view.Runtimes = append(view.Runtimes, entry)
	}
	return view
}

// RuntimeAuth runs the read-only subscription authentication preflight for
// the Runtime executable on PATH, with this shell's environment and working
// directory: what a child without an explicit environment would inherit.
// Child dispatch repeats it against the exact effective invocation.
func (s lifecycleService) RuntimeAuth(ctx context.Context, runtimeID string) cli.Result {
	label := map[string]string{"codex": "Codex", "claude": "Claude"}[runtimeID]
	report := runtimeadapter.AuthReport{RuntimeID: runtimeID, Status: runtimeadapter.AuthUnavailable, Reason: "executable_unavailable", Method: "unknown", Version: "unknown", EvidenceKind: runtimeadapter.EvidenceLocalObservation, Usability: runtimeadapter.UsabilityUnproven, Revalidation: runtimeadapter.RevalidateBeforeDispatch}
	preflight, err := runtimeadapter.NewAuthPreflight(runtimeadapter.OSStatusRunner{}, runtimeadapter.DefaultManagedConfiguration())
	cwd, cwdErr := os.Getwd()
	if s.runtimes.lookPath != nil && err == nil && cwdErr == nil {
		if executable, lookErr := s.runtimes.lookPath(runtimeID); lookErr == nil && filepath.IsAbs(executable) {
			report = preflight.Check(ctx, runtimeadapter.AuthTarget{RuntimeID: runtimeID, Executable: executable, WorkingDirectory: cwd, Environment: os.Environ()})
		}
	}
	return runtimeAuthResult(report, label, s.provenance)
}

func runtimeAuthResult(report runtimeadapter.AuthReport, label string, source provenance.Value) cli.Result {
	references := []string{"runtime-auth:" + report.RuntimeID + ":" + string(report.Status)}
	var response cli.Result
	switch report.Status {
	case runtimeadapter.AuthSubscriptionObserved:
		response = canonicalCompletion(completion.Facts{Completed: true}, label+" subscription login observed; usability stays unproven until a real dispatch", references, "Child dispatch repeats this preflight on its exact invocation; a real probe needs separate Runtime authorization", source)
	case runtimeadapter.AuthIncompatible:
		response = canonicalCompletion(completion.Facts{ValidationFailed: true}, label+" authentication selects an API-key or provider path", references, "Remove the reported overrides from the child environment or configuration yourself, then repeat runtime auth", source)
	case runtimeadapter.AuthUnavailable:
		response = canonicalCompletion(completion.Facts{ValidationFailed: true}, label+" subscription login is unavailable", references, "Log in to "+label+" with its subscription yourself, then repeat runtime auth", source)
	default:
		response = canonicalCompletion(completion.Facts{ValidationFailed: true}, label+" subscription authentication path is unproven", references, "Inspect the reported limitation; dispatch stays blocked until the intended path is observable", source)
	}
	response.RuntimeAuth = &report
	return response
}
