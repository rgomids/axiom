package main

import (
	"context"
	"errors"
	"path/filepath"
	"time"

	"github.com/rgomids/axiom/internal/cli"
	"github.com/rgomids/axiom/internal/codexruntime"
	"github.com/rgomids/axiom/internal/completion"
	"github.com/rgomids/axiom/internal/local"
	"github.com/rgomids/axiom/internal/runtimeadapter"
	"github.com/rgomids/axiom/internal/runtimeapplication"
	"github.com/rgomids/axiom/internal/runtimebootstrap"
	"github.com/rgomids/axiom/internal/runtimeprofile"
)

var errRuntimePolicyInput = errors.New("runtime policy input unavailable")

var _ cli.RuntimeProfilePreviewService = lifecycleService{}

type runtimePolicySource struct {
	installation local.InstallationStore
	portable     local.PortableStore
	stateRoot    string
	runtimes     runtimeObservationSources
}

// runtimeObservationSources are the machine-local facts the production observer
// reads: executables resolved from PATH exactly as first-run discovers them and
// Axiom's own skill integration per Runtime. No caller-supplied inventory exists.
type runtimeObservationSources struct {
	lookPath     runtimebootstrap.LookPath
	integrations map[string]func() (runtimeadapter.IntegrationInspector, error)
}

func (s lifecycleService) runtimeObservationSources() runtimeObservationSources {
	return runtimeObservationSources{lookPath: s.runtimes.lookPath, integrations: map[string]func() (runtimeadapter.IntegrationInspector, error){
		"codex": func() (runtimeadapter.IntegrationInspector, error) { return integrationInspector{s.codex}, nil },
		"claude": func() (runtimeadapter.IntegrationInspector, error) {
			service, err := s.runtimes.claude()
			return integrationInspector{service}, err
		},
	}}
}

type integrationInspector struct{ service codexruntime.Service }

func (i integrationInspector) IntegrationReady(ctx context.Context) bool {
	return i.service.Inspect(ctx).Status == codexruntime.Ready
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
	cfg, err := store.Load(ctx)
	if err != nil {
		return runtimeapplication.Snapshot{}, errRuntimePolicyInput
	}
	observer, err := s.runtimes.observer(cfg)
	if err != nil {
		return runtimeapplication.Snapshot{}, errRuntimePolicyInput
	}
	return runtimeapplication.Snapshot{Project: portable.Snapshot.Project(), Configuration: cfg, Observer: observer}, nil
}

// Each Load observes afresh, so Check compares the reviewed executable
// identity and integration proof with the machine as it is now.
func (r runtimeObservationSources) observer(cfg runtimeprofile.Configuration) (runtimeadapter.ExecutableObserver, error) {
	runtimes := make([]runtimeadapter.ObservedRuntime, 0, len(cfg.Runtimes))
	for _, runtime := range cfg.Runtimes {
		observed := runtimeadapter.ObservedRuntime{ID: runtime.ID}
		// A disabled Runtime is never a candidate; it is not inspected either.
		if r.lookPath != nil && runtime.Enabled {
			if path, err := r.lookPath(runtime.Adapter); err == nil && filepath.IsAbs(path) {
				observed.Executable = path
			}
		}
		if integration, ok := r.integrations[runtime.ID]; ok {
			if inspector, err := integration(); err == nil {
				observed.Integration = inspector
			}
		}
		runtimes = append(runtimes, observed)
	}
	return runtimeadapter.NewExecutableObserver(cfg.Revision, time.Now(), runtimes)
}

func (s lifecycleService) runtimePolicyPreview(ctx context.Context, input cli.RuntimeProfilePreviewInput) (runtimeapplication.Preview, runtimeapplication.Service, error) {
	if !cli.ValidRuntimePreviewInput(input) {
		return runtimeapplication.Preview{Blocker: &runtimeprofile.Blocker{Code: "invalid_request"}}, runtimeapplication.Service{}, errRuntimePolicyInput
	}
	source := s.RuntimePolicySource
	id := input.Project
	if source == nil {
		resolved := s.installation.Resolve(ctx, input.Project)
		if resolved.Status != local.ResolutionFound {
			return runtimeapplication.Preview{Blocker: &runtimeprofile.Blocker{Code: "policy_unavailable"}}, runtimeapplication.Service{}, errRuntimePolicyInput
		}
		id = resolved.Project.ID
		source = runtimePolicySource{installation: s.installation, portable: s.portable, stateRoot: s.stateRoot, runtimes: s.runtimeObservationSources()}
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
	response := canonicalCompletion(completion.Facts{ValidationFailed: !ready, Completed: ready}, "Project Runtime policy resolution blocked", nil, "Correct the Project policy or local Runtime state and request a fresh preview", s.provenance)
	if ready {
		response = canonicalCompletion(completion.Facts{Completed: true}, "Project Runtime resolution preview ready", nil, "Review the Runtime resolution and repeat workflow start with --runtime-preview", s.provenance)
	}
	response.RuntimeResolution = &preview
	response.PreviewDigest = preview.Digest()
	return response
}

func (s lifecycleService) runtimePolicyFailure(code string) cli.Result {
	return s.runtimeResolutionResult(runtimeapplication.Preview{Blocker: &runtimeprofile.Blocker{Code: code}}, false)
}
