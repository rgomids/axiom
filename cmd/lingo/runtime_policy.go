package main

import (
	"context"
	"errors"
	"os"
	"time"

	"github.com/rgomids/axiom/internal/cli"
	"github.com/rgomids/axiom/internal/completion"
	"github.com/rgomids/axiom/internal/local"
	"github.com/rgomids/axiom/internal/runtimeadapter"
	"github.com/rgomids/axiom/internal/runtimeapplication"
	"github.com/rgomids/axiom/internal/runtimebootstrap"
	"github.com/rgomids/axiom/internal/runtimeprofile"
	"github.com/rgomids/axiom/internal/stageexecution"
)

var errRuntimePolicyInput = errors.New("runtime policy input unavailable")

var _ cli.RuntimeProfilePreviewService = lifecycleService{}

type runtimePolicySource struct {
	installation local.InstallationStore
	portable     local.PortableStore
	stateRoot    string
	runtimes     runtimeObservationSources
}

type runtimeObservationSources struct {
	lookPath   runtimebootstrap.LookPath
	repository string
	consented  bool
}

func (r runtimeObservationSources) scoped(repository string, consented bool) runtimeObservationSources {
	r.repository, r.consented = repository, consented
	return r
}

func (s lifecycleService) runtimeObservationSources() runtimeObservationSources {
	return runtimeObservationSources{lookPath: s.runtimes.lookPath}
}

func (s runtimePolicySource) Load(ctx context.Context, id string) (runtimeapplication.Snapshot, error) {
	if err := ctx.Err(); err != nil {
		return runtimeapplication.Snapshot{}, err
	}
	resolved := s.installation.Resolve(ctx, id)
	if resolved.Status != local.ResolutionFound {
		return runtimeapplication.Snapshot{}, errRuntimePolicyInput
	}
	portable, err := s.portable.InspectRecordedSource(ctx, resolved.Project.Source, resolved.Project.Slug)
	if err != nil || !portable.Exists {
		return runtimeapplication.Snapshot{}, errRuntimePolicyInput
	}
	state := portable.Snapshot.Project().State()
	if state.ID != resolved.Project.ID || state.Slug != resolved.Project.Slug || portable.Snapshot.Revision() != resolved.Project.PortableRevision {
		return runtimeapplication.Snapshot{}, errRuntimePolicyInput
	}
	store, err := local.NewRuntimeProfileStore(s.stateRoot)
	if err != nil {
		return runtimeapplication.Snapshot{}, errRuntimePolicyInput
	}
	localState, err := store.Inspect(ctx)
	if err != nil {
		return runtimeapplication.Snapshot{}, runtimeapplication.LocalStateError{State: localState.State}
	}
	consumer, code := runtimeapplication.ConsumerFromRecord(localState.State, localState.V4)
	if code != "" {
		if localState.State == "active-v4" {
			return runtimeapplication.Snapshot{}, errRuntimePolicyInput
		}
		return runtimeapplication.Snapshot{}, runtimeapplication.LocalStateError{State: localState.State}
	}
	return runtimeapplication.Snapshot{Project: portable.Snapshot.Project(), Consumer: consumer, Observer: s.runtimes.observer(consumer)}, nil
}

func (r runtimeObservationSources) observer(consumer runtimeprofile.ConsumerV4) runtimeadapter.OperationalSource {
	return runtimeadapter.OperationalSource{Configuration: consumer.Configuration(), LookPath: r.lookPath, Repository: r.repository, Environment: os.Environ(), Consented: r.consented, ObservedAt: time.Now()}
}

func (s lifecycleService) runtimePolicyPreview(ctx context.Context, input cli.RuntimeProfilePreviewInput) (runtimeapplication.Preview, runtimeapplication.Service, error) {
	return s.scopedRuntimePolicyPreview(ctx, input, s.runtimeObservationSources())
}

func (s lifecycleService) scopedRuntimePolicyPreview(ctx context.Context, input cli.RuntimeProfilePreviewInput, runtimes runtimeObservationSources) (runtimeapplication.Preview, runtimeapplication.Service, error) {
	if !cli.ValidRuntimePreviewInput(input) {
		return runtimeapplication.Preview{Blocker: &runtimeprofile.Blocker{Code: "invalid_request"}}, runtimeapplication.Service{}, errRuntimePolicyInput
	}
	source := s.RuntimePolicySource
	id := input.Project
	if source != nil {
		resolved := s.installation.Resolve(ctx, input.Project)
		if resolved.Status == local.ResolutionFound {
			id = resolved.Project.ID
		}
	}
	if source == nil {
		resolved := s.installation.Resolve(ctx, input.Project)
		if resolved.Status != local.ResolutionFound {
			return runtimeapplication.Preview{Blocker: &runtimeprofile.Blocker{Code: "policy_unavailable"}}, runtimeapplication.Service{}, errRuntimePolicyInput
		}
		id = resolved.Project.ID
		source = runtimePolicySource{installation: s.installation, portable: s.portable, stateRoot: s.stateRoot, runtimes: runtimes}
	}
	service := runtimeapplication.New(source)
	preview, err := service.Preview(ctx, id, input.Request())
	return preview, service, err
}

func (s lifecycleService) RuntimeProfilePreview(ctx context.Context, input cli.RuntimeProfilePreviewInput) cli.Result {
	if input.Project == "" {
		id, failure := s.EffectiveProject(ctx, "")
		if id == "" {
			return failure
		}
		input.Project = id
	}
	preview, _, err := s.runtimePolicyPreview(ctx, input)
	return s.runtimeResolutionResult(preview, err == nil)
}

func (s lifecycleService) runtimeResolutionResult(preview runtimeapplication.Preview, ready bool) cli.Result {
	next := "Correct the Project policy or local Runtime state and request a fresh preview"
	if preview.Blocker != nil {
		switch preview.Blocker.Code {
		case "configuration_missing", "configuration_retired", "profile_effort_unconfigured":
			next = "Use axiom-runtime to inspect and prepare explicit local setup; request a fresh preview afterwards"
		case "unsupported_configuration_format":
			next = "Pre-stable local configuration is not used: retire it explicitly with axiom-runtime (binding remove --runtime <id> --cascade), create canonical configuration, then request a fresh preview"
		case "configuration_recovery_required":
			next = "Use axiom-runtime scoped recovery; preserve the current state and review exact recovery authority"
		default:
			if action := operationalNext(preview.Blocker.Code); action != "" {
				next = action
			}
		}
	}
	response := canonicalCompletion(completion.Facts{ValidationFailed: !ready, Completed: ready}, "Project Runtime policy resolution blocked", nil, next, s.provenance)
	if ready {
		response = canonicalCompletion(completion.Facts{Completed: true}, "Project Runtime resolution preview ready", nil, "Review the Runtime resolution and repeat workflow start with --runtime-preview", s.provenance)
	}
	if preview.Blocker != nil {
		if d := stageexecution.PreSpawnDiagnostic(*preview.Blocker, "workflow-preview"); d != nil {
			response.Diagnostic = diagnosticBase("runtime profile preview", d.Code, "runtime profile list")
			response.Diagnostic.Attempt = d
		}
	}
	response.RuntimeResolution = &preview
	response.PreviewDigest = preview.Digest()
	return response
}

func (s lifecycleService) runtimePolicyFailure(code string) cli.Result {
	return s.runtimeResolutionResult(runtimeapplication.Preview{Blocker: &runtimeprofile.Blocker{Code: code}}, false)
}

func operationalNext(code string) string {
	switch code {
	case "executable_unavailable":
		return "Install the Runtime CLI or expose it on PATH under its own name, then request a fresh preview"
	case "inspection_consent_required":
		return "Use workflow start or workflow stage plan/run with --authorize-local over the registered repository to inspect its Runtime effort sources; runtime profile preview has no inspection scope"
	case "effort_override_conflict":
		return "Remove the competing effort setting (argument, environment or settings file) so the Profile alone owns effort, then request a fresh preview"
	case "configuration_unsafe", "configuration_invalid":
		return "Inspect local Runtime configuration: an unsafe Axiom Runtime configuration store needs axiom-runtime scoped recovery; Runtime effort sources must not be relocated (CODEX_HOME, CLAUDE_CONFIG_DIR), symlinked, foreign-owned or malformed. Then request a fresh preview"
	}
	return ""
}
