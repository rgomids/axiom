package runtimeapplication

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/rgomids/axiom/internal/project"
	"github.com/rgomids/axiom/internal/runtimeadapter"
	"github.com/rgomids/axiom/internal/runtimeprofile"
)

const projectID = "123e4567-e89b-42d3-a456-426614174000"

type sourceFake struct {
	snapshot Snapshot
	config   runtimeprofile.ConfigurationV4
	err      error
	loads    int
}

func (s *sourceFake) Load(context.Context, string) (Snapshot, error) {
	s.loads++
	snapshot := s.snapshot
	snapshot.Consumer, _ = runtimeprofile.NewConsumerV4(s.config)
	return snapshot, s.err
}

type observerFake struct {
	runtimes map[string]runtimeprofile.OperationalObservation
	failure  error
}

func (o *observerFake) ObserveExecutable(_ context.Context, id string) (runtimeprofile.OperationalObservation, error) {
	v, ok := o.runtimes[id]
	if !ok {
		return v, runtimeprofile.ErrExecutableUnavailable
	}
	v.EffortSourceDigest = ""
	return v, nil
}

func (o *observerFake) ObserveOperational(ctx context.Context, id, _ string) (runtimeprofile.OperationalObservation, error) {
	if o.failure != nil {
		return runtimeprofile.OperationalObservation{}, o.failure
	}
	v, ok := o.runtimes[id]
	if !ok {
		return v, runtimeprofile.ErrExecutableUnavailable
	}
	return v, nil
}

func (o *observerFake) edit(id string, change func(*runtimeprofile.OperationalObservation)) {
	v := o.runtimes[id]
	change(&v)
	o.runtimes[id] = v
}

func setup(t *testing.T, runtimes ...string) (*sourceFake, *observerFake) {
	t.Helper()
	state := project.State{SchemaVersion: 2, ID: projectID, Slug: "example", Name: "Example"}
	rs := []project.Runtime{}
	ps := []project.ModelProfile{}
	cfg := runtimeprofile.ConfigurationV4{FormatVersion: runtimeprofile.FormatVersionV4, Revision: 7}
	obs := &observerFake{runtimes: map[string]runtimeprofile.OperationalObservation{}}
	for _, id := range runtimes {
		rs = append(rs, project.Runtime{ID: id})
		ps = append(ps, project.ModelProfile{Key: id + "-profile", RuntimeRef: project.Configured(id), Model: project.Configured(id + "-model")})
		profile := runtimeprofile.ModelProfile{ID: id + "-profile", RuntimeID: id, Model: id + "-model", Effort: runtimeprofile.Effort{Mode: "explicit", Value: "high"}, Capabilities: []string{"go"}, Complexities: []string{"high"}}
		cfg.Runtimes = append(cfg.Runtimes, runtimeprofile.RuntimeV4{ID: id, Adapter: id, Enabled: true, AllowlistedProfileIDs: []string{id + "-profile"}})
		cfg.ModelProfiles = append(cfg.ModelProfiles, profile)
		obs.runtimes[id] = runtimeprofile.OperationalObservation{RuntimeID: id, Adapter: id, Installed: true, Available: true, ExecutableDigest: strings.Repeat("a", 64), InvocationMode: "noninteractive", Revision: 7, EffortSourceDigest: strings.Repeat("b", 64), ObservedAt: time.Unix(1, 0)}
	}
	state.Runtimes = project.Configured(rs)
	state.ModelProfiles = project.Configured(ps)
	p, issues := project.New(state)
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	return &sourceFake{snapshot: Snapshot{Project: p, Observer: obs}, config: cfg}, obs
}
func request() Request {
	return Request{Role: "implementation", Complexity: "high", Capabilities: []string{"go"}}
}
func TestProjectResolutionBothExplicitRuntimes(t *testing.T) {
	for _, id := range []string{"codex", "claude"} {
		t.Run(id, func(t *testing.T) {
			source, _ := setup(t, id)
			service := New(source)
			preview, err := service.Preview(context.Background(), projectID, request())
			if err != nil || preview.Choice == nil || preview.Choice.RuntimeID != id || preview.Choice.Model != id+"-model" {
				t.Fatalf("%+v %v", preview, err)
			}
			again, err := service.Preview(context.Background(), projectID, request())
			if err != nil || !reflect.DeepEqual(again, preview) {
				t.Fatal("nondeterministic preview")
			}
			binding, err := service.Check(context.Background(), preview)
			if err != nil || !reflect.DeepEqual(binding.Choice, *preview.Choice) || binding.CredentialReference != "" {
				t.Fatal("preview mismatch", err)
			}
			if wire, _ := json.Marshal(binding); strings.Contains(string(wire), "PRIVATE") {
				t.Fatal("binding serialized its credential reference")
			}
			if strings.Contains(preview.Digest(), "PRIVATE") || strings.Contains(wireString(preview), "PRIVATE") {
				t.Fatal("credential leak")
			}
		})
	}
}

func TestProjectResolutionHonorsExactModelProfile(t *testing.T) {
	source, _ := setup(t, "codex", "claude")
	req := request()
	req.ModelProfileID = "claude-profile"
	preview, err := New(source).Preview(context.Background(), projectID, req)
	if err != nil || preview.Choice == nil || preview.Choice.RuntimeID != "claude" || preview.Choice.ModelProfileID != "claude-profile" {
		t.Fatalf("preview=%+v err=%v", preview, err)
	}
	req.ModelProfileID = "not-allowlisted"
	if _, err := New(source).Preview(context.Background(), projectID, req); !errors.Is(err, ErrBlocked) {
		t.Fatalf("unavailable exact profile err=%v", err)
	}
}

func TestExactProfileDoesNotUseUnavailableDefaultPreference(t *testing.T) {
	source, _ := setup(t, "codex", "claude")
	state := source.snapshot.Project.State()
	state.RuntimePreferences = project.Configured([]project.RuntimePreference{{Role: "implementation", Complexity: "high", ModelProfileRef: "codex-profile"}})
	p, diagnostics := project.New(state)
	if len(diagnostics) != 0 {
		t.Fatal(diagnostics)
	}
	source.snapshot.Project = p
	// Retain the portable preferred profile but remove it from local admission.
	source.config.ModelProfiles[0].Model = "locally-incompatible-model"
	req := request()
	req.ModelProfileID = "claude-profile"
	if preview, err := New(source).Preview(context.Background(), projectID, req); err != nil || preview.Choice == nil || preview.Choice.ModelProfileID != "claude-profile" {
		t.Fatalf("explicit allowed profile blocked by unused default: %+v %v", preview, err)
	}
	if _, err := New(source).Preview(context.Background(), projectID, request()); !errors.Is(err, ErrBlocked) {
		t.Fatalf("default unexpectedly fell back: %v", err)
	}
}

// Keep JSON leak assertions on the real public representation.

func wireString(p Preview) string { wire, _ := json.Marshal(p); return string(wire) }
func TestProjectResolutionBlocksWithoutFallback(t *testing.T) {
	cases := []struct {
		name, code string
		change     func(*sourceFake, *observerFake)
		narrow     bool
	}{
		{"ambiguous", "ambiguous_match", func(*sourceFake, *observerFake) {}, false},
		{"missing runtimes", "policy_unconfigured", func(s *sourceFake, _ *observerFake) {
			state := s.snapshot.Project.State()
			state.Runtimes = project.Declaration[[]project.Runtime]{}
			state.ModelProfiles = project.Declaration[[]project.ModelProfile]{}
			s.snapshot.Project, _ = project.New(state)
		}, false},
		{"unconfigured preferences", "policy_unconfigured", func(s *sourceFake, _ *observerFake) {
			state := s.snapshot.Project.State()
			state.RuntimePreferences = project.Unconfigured[[]project.RuntimePreference]()
			s.snapshot.Project, _ = project.New(state)
		}, false},
		{"unconfigured profiles", "policy_unconfigured", func(s *sourceFake, _ *observerFake) {
			state := s.snapshot.Project.State()
			state.ModelProfiles = project.Unconfigured[[]project.ModelProfile]()
			s.snapshot.Project, _ = project.New(state)
		}, false},
		{"executable unavailable", "executable_unavailable", func(_ *sourceFake, o *observerFake) {
			delete(o.runtimes, "codex")
		}, true},
		{"unavailable is not a selection", "ambiguous_match", func(_ *sourceFake, o *observerFake) {
			delete(o.runtimes, "claude")
		}, false},
		{"capability undeclared by Profile", "no_allowed_match", func(s *sourceFake, _ *observerFake) {
			for i := range s.config.ModelProfiles {
				s.config.ModelProfiles[i].Capabilities = []string{"review"}
			}
		}, true},
		{"inspection consent missing", "inspection_consent_required", func(_ *sourceFake, o *observerFake) { o.failure = runtimeprofile.ErrInspectionConsentRequired }, true},
		{"competing effort selector", "effort_override_conflict", func(_ *sourceFake, o *observerFake) { o.failure = runtimeprofile.ErrEffortOverrideConflict }, true},
		{"invalid effort syntax", "configuration_invalid", func(_ *sourceFake, o *observerFake) { o.failure = runtimeprofile.ErrConfigurationInvalid }, true},
		{"unsafe effort source", "configuration_unsafe", func(_ *sourceFake, o *observerFake) { o.failure = runtimeprofile.ErrConfigurationUnsafe }, true},
		{"opaque observation failure", "invalid_observation", func(_ *sourceFake, o *observerFake) { o.failure = errors.New("PRIVATE_SECRET /home/operator") }, true},
		{"stale observation", "invalid_observation", func(_ *sourceFake, o *observerFake) {
			o.edit("codex", func(v *runtimeprofile.OperationalObservation) { v.Revision-- })
		}, true},
		{"disabled runtime", "no_allowed_match", func(s *sourceFake, _ *observerFake) {
			for i := range s.config.Runtimes {
				s.config.Runtimes[i].Enabled = false
			}
		}, true},
		{"wrong model", "no_allowed_match", func(s *sourceFake, _ *observerFake) {
			for i := range s.config.ModelProfiles {
				s.config.ModelProfiles[i].Model = "wrong-model"
			}
		}, false},
		{"disallowed local codex", "no_allowed_match", func(s *sourceFake, _ *observerFake) {
			state := s.snapshot.Project.State()
			state.Runtimes = project.Configured([]project.Runtime{{ID: "other"}})
			state.ModelProfiles = project.Configured([]project.ModelProfile{{Key: "other", RuntimeRef: project.Configured("other"), Model: project.Configured("model")}})
			s.snapshot.Project, _ = project.New(state)
		}, false},
		{"raw error", "policy_unavailable", func(s *sourceFake, _ *observerFake) { s.err = errors.New("PRIVATE_SECRET") }, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s, o := setup(t, "codex", "claude")
			tc.change(s, o)
			r := request()
			if tc.narrow {
				r.RuntimeID = "codex"
			}
			p, err := New(s).Preview(context.Background(), projectID, r)
			if err == nil || p.Choice != nil || p.Blocker == nil || p.Blocker.Code != tc.code {
				t.Fatalf("%+v %v", p, err)
			}
			if p.ConfigurationRevision != 7 && p.Blocker.Code != "invalid_request" && p.Blocker.Code != "policy_unavailable" {
				t.Fatal("missing configuration revision", p)
			}
			if strings.Contains(wireString(p), "PRIVATE") {
				t.Fatal("leak")
			}
		})
	}
}
func TestProjectPreferenceAndExplicitNarrowing(t *testing.T) {
	s, o := setup(t, "codex", "claude")
	// Local global preferences cannot authorize a Project choice.
	s.config.Preferences = []runtimeprofile.Preference{{Role: "implementation", Complexity: "high", ModelProfileID: "codex-profile"}}
	p, err := New(s).Preview(context.Background(), projectID, request())
	if err == nil || p.Blocker.Code != "ambiguous_match" {
		t.Fatal(p, err)
	}
	state := s.snapshot.Project.State()
	state.RuntimePreferences = project.Configured([]project.RuntimePreference{{Role: "implementation", Complexity: "high", ModelProfileRef: "claude-profile"}})
	s.snapshot.Project, _ = project.New(state)
	p, err = New(s).Preview(context.Background(), projectID, request())
	if err != nil || p.Choice.RuntimeID != "claude" {
		t.Fatal(p, err)
	}
	delete(o.runtimes, "claude")
	p, err = New(s).Preview(context.Background(), projectID, request())
	if err == nil || p.Choice != nil {
		t.Fatal("preferred unavailable fell back", p)
	}
	r := request()
	r.RuntimeID = "codex"
	p, err = New(s).Preview(context.Background(), projectID, r)
	if err == nil {
		t.Fatal("narrowing bypassed preference", p)
	}
}
func TestLegacyV1ProjectionNeverRewritesProject(t *testing.T) {
	s, _ := setup(t, "claude")
	state := s.snapshot.Project.State()
	state.SchemaVersion = 1
	state.Runtime = project.Configured(project.Runtime{ID: "claude"})
	state.Runtimes = project.Declaration[[]project.Runtime]{}
	s.snapshot.Project, _ = project.New(state)
	before := s.snapshot.Project
	p, err := New(s).Preview(context.Background(), projectID, request())
	if err != nil || p.Choice.RuntimeID != "claude" || !before.Equivalent(s.snapshot.Project) || s.snapshot.Project.State().SchemaVersion != 1 {
		t.Fatal(p, err)
	}
}
func TestPreviewRejectsContentAndRevisionDrift(t *testing.T) {
	changes := map[string]func(*sourceFake, *observerFake){
		"config revision": func(s *sourceFake, _ *observerFake) { s.config.Revision++ },
		"same revision config": func(s *sourceFake, _ *observerFake) {
			s.config.ModelProfiles[0].Capabilities = []string{"go", "review"}
		},
		"portable identity": func(s *sourceFake, _ *observerFake) {
			state := s.snapshot.Project.State()
			state.Name = "Changed"
			s.snapshot.Project, _ = project.New(state)
		},
		"observation revision": func(_ *sourceFake, o *observerFake) {
			o.edit("codex", func(v *runtimeprofile.OperationalObservation) { v.Revision++ })
		},
		"executable bytes": func(_ *sourceFake, o *observerFake) {
			o.edit("codex", func(v *runtimeprofile.OperationalObservation) { v.ExecutableDigest = strings.Repeat("c", 64) })
		},
		"effort sources": func(_ *sourceFake, o *observerFake) {
			o.edit("codex", func(v *runtimeprofile.OperationalObservation) { v.EffortSourceDigest = strings.Repeat("d", 64) })
		},
		"profile effort intent": func(s *sourceFake, _ *observerFake) {
			s.config.ModelProfiles[0].Effort = runtimeprofile.Effort{Mode: "explicit", Value: "low"}
		},
	}
	for name, change := range changes {
		t.Run(name, func(t *testing.T) {
			s, o := setup(t, "codex")
			service := New(s)
			p, err := service.Preview(context.Background(), projectID, request())
			if err != nil {
				t.Fatal(err)
			}
			change(s, o)
			if _, err = service.Check(context.Background(), p); !errors.Is(err, ErrStale) {
				t.Fatal(err)
			}
		})
	}
}
func TestInvalidRequestDoesNotReflectRejectedText(t *testing.T) {
	s, _ := setup(t, "codex")
	r := request()
	r.Role = "PRIVATE_SECRET"
	p, err := New(s).Preview(context.Background(), "PRIVATE_SECRET", r)
	if err == nil || s.loads != 0 || strings.Contains(wireString(p), "PRIVATE") {
		t.Fatal(p, err)
	}
}

type executableObserver struct {
	id, path string
}

func (o executableObserver) ObserveExecutable(ctx context.Context, id string) (runtimeprofile.OperationalObservation, error) {
	observation := runtimeprofile.OperationalObservation{RuntimeID: id, Adapter: id, Revision: 7, ObservedAt: time.Unix(1, 0), InvocationMode: "noninteractive"}
	if id != o.id {
		return observation, runtimeprofile.ErrExecutableUnavailable
	}
	identity, err := runtimeadapter.ExecutableIdentity(o.path)
	if err != nil {
		return observation, runtimeadapter.ErrExecutableUnavailable
	}
	observation.Installed, observation.Available, observation.ExecutableDigest = true, true, identity
	return observation, ctx.Err()
}

func (o executableObserver) ObserveOperational(ctx context.Context, id, _ string) (runtimeprofile.OperationalObservation, error) {
	observation, err := o.ObserveExecutable(ctx, id)
	observation.EffortSourceDigest = strings.Repeat("b", 64)
	return observation, err
}

func TestExecutableIdentityBindsReviewedRuntime(t *testing.T) {
	for _, id := range []string{"codex", "claude"} {
		t.Run(id, func(t *testing.T) {
			source, _ := setup(t, id)
			executable := filepath.Join(t.TempDir(), id)
			writeExecutable(t, executable, "original")
			source.snapshot.Observer = executableObserver{id: id, path: executable}
			service := New(source)
			preview, err := service.Preview(context.Background(), projectID, request())
			if err != nil || preview.Choice == nil || preview.Choice.ExecutableDigest == "" || preview.Choice.RuntimeVersion != "" || preview.Choice.FrozenProfile == nil || preview.Choice.ResolvedProfile != nil {
				t.Fatalf("preview=%+v err=%v", preview, err)
			}
			if strings.Contains(wireString(preview), executable) {
				t.Fatal("preview exposed the executable path")
			}
			if _, err := service.Check(context.Background(), preview); err != nil {
				t.Fatalf("unchanged machine state rejected: %v", err)
			}
			writeExecutable(t, executable, "replaced")
			if _, err := service.Check(context.Background(), preview); !errors.Is(err, ErrStale) {
				t.Fatalf("replaced executable accepted: %v", err)
			}
			if err := os.Remove(executable); err != nil {
				t.Fatal(err)
			}
			if _, err := service.Check(context.Background(), preview); !errors.Is(err, ErrStale) {
				t.Fatalf("removed executable accepted: %v", err)
			}
			if blocked, err := service.Preview(context.Background(), projectID, request()); err == nil || blocked.Blocker.Code != "executable_unavailable" {
				t.Fatalf("missing executable resolved: %+v", blocked)
			}
		})
	}
}

func writeExecutable(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte("#!/bin/sh\n# "+content+"\nexit 99\n"), 0o700); err != nil {
		t.Fatal(err)
	}
}
