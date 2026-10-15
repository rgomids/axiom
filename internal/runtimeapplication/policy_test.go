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
	"github.com/rgomids/axiom/internal/runtimeprofile/runtimeprofiletest"
)

const projectID = "123e4567-e89b-42d3-a456-426614174000"

// sourceFake re-reads config on every Load, as the production source re-reads
// the canonical v3 record; an invalid config yields no consumer.
type sourceFake struct {
	snapshot Snapshot
	config   runtimeprofile.ConfigurationV3
	err      error
	loads    int
}

func (s *sourceFake) Load(context.Context, string) (Snapshot, error) {
	s.loads++
	snapshot := s.snapshot
	snapshot.Consumer, _ = runtimeprofile.NewConsumerV3(runtimeprofile.RecordActiveV3, &s.config)
	return snapshot, s.err
}

// observerFake injects authoritative observations. Its proofs bind the exact
// Profile intent, executable and version; nothing here acquires proof.
type observerFake map[string]runtimeprofile.Observation

func proven(id string, p runtimeprofile.ModelProfile, o runtimeprofile.Observation) runtimeprofile.ProfileProof {
	return runtimeprofile.ProfileProof{RuntimeID: id, Model: p.Model, Effort: p.Effort, ExecutableDigest: o.ExecutableDigest, RuntimeVersion: o.Version, InvocationMode: "noninteractive", ObservationRevision: o.Revision, Reference: "injected-proof", EffortSourceDigest: strings.Repeat("b", 64), Authentication: runtimeprofile.CapabilityProven, ModelAvailability: runtimeprofile.CapabilityProven, EffortSupport: runtimeprofile.CapabilityProven, NoOverride: runtimeprofile.CapabilityProven, CapabilityStatus: map[string]runtimeprofile.CapabilityStatus{"go": runtimeprofile.CapabilityProven}}
}

func (o observerFake) Observe(_ context.Context, id string) (runtimeprofile.Observation, error) {
	v, ok := o[id]
	if !ok {
		return v, errors.New("private raw error")
	}
	return v, nil
}
func setup(t *testing.T, runtimes ...string) (*sourceFake, observerFake) {
	t.Helper()
	state := project.State{SchemaVersion: 2, ID: projectID, Slug: "example", Name: "Example"}
	rs := []project.Runtime{}
	ps := []project.ModelProfile{}
	cfg := runtimeprofile.ConfigurationV3{FormatVersion: runtimeprofile.FormatVersionV3, Revision: 7}
	obs := observerFake{}
	for _, id := range runtimes {
		rs = append(rs, project.Runtime{ID: id})
		ps = append(ps, project.ModelProfile{Key: id + "-profile", RuntimeRef: project.Configured(id), Model: project.Configured(id + "-model")})
		profile := runtimeprofile.ModelProfile{ID: id + "-profile", RuntimeID: id, Model: id + "-model", Effort: runtimeprofile.Effort{Mode: "explicit", Value: "high"}, Capabilities: []string{"go"}, Complexities: []string{"high"}}
		cfg.Runtimes = append(cfg.Runtimes, runtimeprofile.RuntimeV3{ID: id, Adapter: id, Enabled: true, AllowlistedProfileIDs: []string{id + "-profile"}})
		cfg.ModelProfiles = append(cfg.ModelProfiles, profile)
		o := runtimeprofile.Observation{RuntimeID: id, Adapter: id, Installed: true, Available: true, Version: "test-1.0", ExecutableDigest: strings.Repeat("a", 64), Revision: 7, ObservedAt: time.Unix(1, 0), CapabilityStatus: map[string]runtimeprofile.CapabilityStatus{"go": runtimeprofile.CapabilityProven}}
		o.ProfileProofs = []runtimeprofile.ProfileProof{proven(id, profile, o)}
		obs[id] = o
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
			// Canonical v3 carries no credential reference.
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
	// Observation and proof gates apply after exact selection; narrowed cases
	// request one Runtime so the two configured Runtimes are not ambiguous.
	cases := []struct {
		name, code string
		change     func(*sourceFake, observerFake)
		narrow     bool
	}{
		{"ambiguous", "ambiguous_match", func(*sourceFake, observerFake) {}, false},
		{"missing runtimes", "policy_unconfigured", func(s *sourceFake, _ observerFake) {
			state := s.snapshot.Project.State()
			state.Runtimes = project.Declaration[[]project.Runtime]{}
			state.ModelProfiles = project.Declaration[[]project.ModelProfile]{}
			s.snapshot.Project, _ = project.New(state)
		}, false},
		{"unconfigured preferences", "policy_unconfigured", func(s *sourceFake, _ observerFake) {
			state := s.snapshot.Project.State()
			state.RuntimePreferences = project.Unconfigured[[]project.RuntimePreference]()
			s.snapshot.Project, _ = project.New(state)
		}, false},
		{"unconfigured profiles", "policy_unconfigured", func(s *sourceFake, _ observerFake) {
			state := s.snapshot.Project.State()
			state.ModelProfiles = project.Unconfigured[[]project.ModelProfile]()
			s.snapshot.Project, _ = project.New(state)
		}, false},
		{"unavailable", "no_allowed_match", func(_ *sourceFake, o observerFake) {
			for id, v := range o {
				v.Available = false
				o[id] = v
			}
		}, true},
		{"unavailable is not a selection", "ambiguous_match", func(_ *sourceFake, o observerFake) {
			v := o["claude"]
			v.Available = false
			o["claude"] = v
		}, false},
		{"capability declared", "capability_unproven", func(_ *sourceFake, o observerFake) {
			for id, v := range o {
				v.CapabilityStatus["go"] = runtimeprofile.CapabilityDeclared
				o[id] = v
			}
		}, true},
		{"no proof", "effort_unproven", func(_ *sourceFake, o observerFake) {
			for id, v := range o {
				v.ProfileProofs = nil
				o[id] = v
			}
		}, true},
		{"stale observation", "no_allowed_match", func(_ *sourceFake, o observerFake) {
			for id, v := range o {
				v.Revision--
				o[id] = v
			}
		}, true},
		{"wrong model", "no_allowed_match", func(s *sourceFake, _ observerFake) {
			for i := range s.config.ModelProfiles {
				s.config.ModelProfiles[i].Model = "wrong-model"
			}
		}, false},
		{"disallowed local codex", "no_allowed_match", func(s *sourceFake, _ observerFake) {
			state := s.snapshot.Project.State()
			state.Runtimes = project.Configured([]project.Runtime{{ID: "other"}})
			state.ModelProfiles = project.Configured([]project.ModelProfile{{Key: "other", RuntimeRef: project.Configured("other"), Model: project.Configured("model")}})
			s.snapshot.Project, _ = project.New(state)
		}, false},
		{"raw error", "policy_unavailable", func(s *sourceFake, _ observerFake) { s.err = errors.New("PRIVATE_SECRET") }, false},
		{"unsafe version", "invalid_observation", func(_ *sourceFake, o observerFake) {
			v := o["codex"]
			v.Version = "Bearer PRIVATE_SECRET"
			o["codex"] = v
		}, true},
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
	v := o["claude"]
	v.Available = false
	o["claude"] = v
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
	changes := map[string]func(*sourceFake, observerFake){
		"config revision": func(s *sourceFake, _ observerFake) { s.config.Revision++ },
		"same revision config": func(s *sourceFake, _ observerFake) {
			s.config.ModelProfiles[0].Capabilities = []string{"go", "review"}
		},
		"same revision declaration": func(s *sourceFake, _ observerFake) {
			s.config.Runtimes[0].EffortDeclarations = []runtimeprofile.EffortDeclaration{runtimeprofiletest.CodexDeclaration()}
		},
		"portable identity": func(s *sourceFake, _ observerFake) {
			state := s.snapshot.Project.State()
			state.Name = "Changed"
			s.snapshot.Project, _ = project.New(state)
		},
		"observation revision":  func(_ *sourceFake, o observerFake) { v := o["codex"]; v.Revision++; o["codex"] = v },
		"same revision version": func(_ *sourceFake, o observerFake) { v := o["codex"]; v.Version = "changed"; o["codex"] = v },
		"capability": func(_ *sourceFake, o observerFake) {
			o["codex"].CapabilityStatus["go"] = runtimeprofile.CapabilityDeclared
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

type integrationFake bool

func (i integrationFake) IntegrationReady(context.Context) bool { return bool(i) }

// The production observer is the only machine-local source: removing or
// replacing the reviewed executable after preview blocks Check, and only
// Axiom's own skill integration can make a capability proven.
func TestExecutableObserverBindsReviewedRuntime(t *testing.T) {
	for _, id := range []string{"codex", "claude"} {
		t.Run(id, func(t *testing.T) {
			source, _ := setup(t, id)
			source.config.ModelProfiles[0].Capabilities = []string{runtimeadapter.IntegrationCapability, "go"}
			profile := source.config.ModelProfiles[0]
			executable := filepath.Join(t.TempDir(), id)
			writeExecutable(t, executable, "original")
			ready := integrationFake(true)
			observe := func() {
				observer, err := runtimeadapter.NewExecutableObserver(7, time.Now(), []runtimeadapter.ObservedRuntime{{ID: id, Executable: executable, Integration: ready}})
				if err != nil {
					t.Fatal(err)
				}
				source.snapshot.Observer = injectedProof{observer, id, profile}
			}
			observe()
			service := New(source)
			proven := Request{Role: "implementation", Complexity: "high", Capabilities: []string{runtimeadapter.IntegrationCapability}}
			preview, err := service.Preview(context.Background(), projectID, proven)
			if err != nil || preview.Choice == nil || preview.Choice.ExecutableDigest == "" || preview.Choice.RuntimeVersion != "test-1.0" {
				t.Fatalf("preview=%+v err=%v", preview, err)
			}
			if strings.Contains(wireString(preview), executable) {
				t.Fatal("preview exposed the executable path")
			}
			if unproven, err := service.Preview(context.Background(), projectID, request()); err == nil || unproven.Blocker.Code != "capability_unproven" {
				t.Fatalf("unprovable capability resolved: %+v", unproven)
			}
			if _, err := service.Check(context.Background(), preview); err != nil {
				t.Fatalf("unchanged machine state rejected: %v", err)
			}
			writeExecutable(t, executable, "replaced")
			observe()
			if _, err := service.Check(context.Background(), preview); !errors.Is(err, ErrStale) {
				t.Fatalf("replaced executable accepted: %v", err)
			}
			if err := os.Remove(executable); err != nil {
				t.Fatal(err)
			}
			observe()
			if _, err := service.Check(context.Background(), preview); !errors.Is(err, ErrStale) {
				t.Fatalf("removed executable accepted: %v", err)
			}
			writeExecutable(t, executable, "original")
			ready = false
			observe()
			if _, err := service.Check(context.Background(), preview); !errors.Is(err, ErrStale) {
				t.Fatalf("unverified integration accepted: %v", err)
			}
		})
	}
}

// injectedProof adds a proof bound to whatever executable is observed now, so
// the test isolates executable identity and integration drift.
type injectedProof struct {
	base    runtimeprofile.Observer
	id      string
	profile runtimeprofile.ModelProfile
}

func (o injectedProof) Observe(ctx context.Context, id string) (runtimeprofile.Observation, error) {
	observed, err := o.base.Observe(ctx, id)
	if err == nil && id == o.id && observed.Available {
		// A synthetic version: an unknown version can never resolve a Profile.
		observed.Version = "test-1.0"
		proof := proven(id, o.profile, observed)
		proof.CapabilityStatus = map[string]runtimeprofile.CapabilityStatus{runtimeadapter.IntegrationCapability: runtimeprofile.CapabilityProven}
		observed.ProfileProofs = []runtimeprofile.ProfileProof{proof}
	}
	return observed, err
}

func writeExecutable(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte("#!/bin/sh\n# "+content+"\nexit 99\n"), 0o700); err != nil {
		t.Fatal(err)
	}
}
