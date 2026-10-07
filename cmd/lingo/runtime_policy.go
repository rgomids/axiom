package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"

	"github.com/rgomids/axiom/internal/cli"
	"github.com/rgomids/axiom/internal/completion"
	"github.com/rgomids/axiom/internal/local"
	"github.com/rgomids/axiom/internal/runtimeadapter"
	"github.com/rgomids/axiom/internal/runtimeapplication"
	"github.com/rgomids/axiom/internal/runtimeprofile"
)

var errRuntimePolicyInput = errors.New("runtime policy input unavailable")

var _ cli.RuntimeProfilePreviewService = lifecycleService{}

type runtimePolicySource struct {
	installation            local.InstallationStore
	portable                local.PortableStore
	stateRoot, observations string
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
	inventory, err := readRuntimeObservations(s.observations)
	if err != nil {
		return runtimeapplication.Snapshot{}, errRuntimePolicyInput
	}
	return runtimeapplication.Snapshot{Project: portable.Snapshot.Project(), Configuration: cfg, Observer: inventory}, nil
}

// The caller supplies an existing machine inventory. Nothing executes or claims
// capabilities by probing a Runtime; each Load reads this inventory afresh.
func readRuntimeObservations(path string) (runtimeadapter.Inventory, error) {
	if !filepath.IsAbs(path) {
		return runtimeadapter.Inventory{}, errRuntimePolicyInput
	}
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Size() > runtimeprofile.MaxConfigurationBytes {
		return runtimeadapter.Inventory{}, errRuntimePolicyInput
	}
	file, err := os.Open(path)
	if err != nil {
		return runtimeadapter.Inventory{}, errRuntimePolicyInput
	}
	defer file.Close()
	current, err := file.Stat()
	if err != nil || !current.Mode().IsRegular() || !os.SameFile(info, current) {
		return runtimeadapter.Inventory{}, errRuntimePolicyInput
	}
	wire, err := io.ReadAll(io.LimitReader(file, runtimeprofile.MaxConfigurationBytes+1))
	if err != nil || len(wire) > runtimeprofile.MaxConfigurationBytes {
		return runtimeadapter.Inventory{}, errRuntimePolicyInput
	}
	if !uniqueObservationKeys(wire) {
		return runtimeadapter.Inventory{}, errRuntimePolicyInput
	}
	decoder := json.NewDecoder(bytes.NewReader(wire))
	decoder.DisallowUnknownFields()
	var observations []runtimeprofile.Observation
	if err := decoder.Decode(&observations); err != nil || len(observations) == 0 || len(observations) > 8 {
		return runtimeadapter.Inventory{}, errRuntimePolicyInput
	}
	var trailing any
	if decoder.Decode(&trailing) != io.EOF {
		return runtimeadapter.Inventory{}, errRuntimePolicyInput
	}
	for _, observation := range observations {
		if len(observation.CapabilityStatus) > 32 {
			return runtimeadapter.Inventory{}, errRuntimePolicyInput
		}
		for _, status := range observation.CapabilityStatus {
			if status != runtimeprofile.CapabilityDeclared && status != runtimeprofile.CapabilityProven {
				return runtimeadapter.Inventory{}, errRuntimePolicyInput
			}
		}
	}
	return runtimeadapter.NewInventory(observations)
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
		source = runtimePolicySource{installation: s.installation, portable: s.portable, stateRoot: s.stateRoot, observations: input.Observations}
	}
	service := runtimeapplication.New(source)
	preview, err := service.Preview(ctx, id, input.Request())
	return preview, service, err
}

func (s lifecycleService) RuntimeProfilePreview(ctx context.Context, input cli.RuntimeProfilePreviewInput) cli.Result {
	preview, _, err := s.runtimePolicyPreview(ctx, input)
	return s.runtimeResolutionResult(preview, err == nil)
}

func (s lifecycleService) runtimeResolutionResult(preview runtimeapplication.Preview, ready bool) cli.Result {
	response := canonicalCompletion(completion.Facts{ValidationFailed: !ready, Completed: ready}, "Project Runtime policy resolution blocked", nil, "Correct the policy or inventory and request a fresh preview", s.provenance)
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

// Enforce exact closed Observation keys before typed decoding: encoding/json
// accepts case aliases, including Unicode folds that can overwrite declarations.
func uniqueObservationKeys(wire []byte) bool {
	decoder := json.NewDecoder(bytes.NewReader(wire))
	var scan func(int) bool
	scan = func(depth int) bool {
		if depth > 4 {
			return false
		}
		token, err := decoder.Token()
		if err != nil {
			return false
		}
		delimiter, ok := token.(json.Delim)
		if !ok {
			return true
		}
		if delimiter != '[' && delimiter != '{' {
			return false
		}
		seen := map[string]bool{}
		for decoder.More() {
			if delimiter == '{' {
				token, err := decoder.Token()
				if err != nil {
					return false
				}
				key, ok := token.(string)
				if !ok {
					return false
				}
				if depth == 1 && !canonicalObservationKey(key) || depth == 2 && !observationCapabilityToken(key) {
					return false
				}
				if seen[key] {
					return false
				}
				seen[key] = true
			}
			if !scan(depth + 1) {
				return false
			}
		}
		end, err := decoder.Token()
		return err == nil && end == map[json.Delim]json.Delim{'[': ']', '{': '}'}[delimiter]
	}
	if !scan(0) {
		return false
	}
	_, err := decoder.Token()
	return err == io.EOF
}

func canonicalObservationKey(key string) bool {
	switch key {
	case "runtimeId", "adapter", "installed", "available", "version", "revision", "observedAt", "capabilityStatus":
		return true
	default:
		return false
	}
}
func observationCapabilityToken(key string) bool {
	if len(key) == 0 || len(key) > 256 {
		return false
	}
	for _, c := range key {
		if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '-' || c == '_' || c == '.') {
			return false
		}
	}
	return true
}
