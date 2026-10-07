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
	err      error
	loads    int
}

func (s *sourceFake) Load(context.Context, string) (Snapshot, error) {
	s.loads++
	return s.snapshot, s.err
}

type observerFake map[string]runtimeprofile.Observation

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
	cfg := runtimeprofile.Configuration{FormatVersion: 1, Revision: 7}
	obs := observerFake{}
	for _, id := range runtimes {
		rs = append(rs, project.Runtime{ID: id})
		ps = append(ps, project.ModelProfile{Key: id + "-profile", RuntimeRef: project.Configured(id), Model: project.Configured(id + "-model")})
		cfg.Runtimes = append(cfg.Runtimes, runtimeprofile.Runtime{ID: id, Adapter: id, Enabled: true, AllowlistedProfileIDs: []string{id + "-profile"}, CredentialReference: "env:PRIVATE_TEST_REFERENCE"})
		cfg.ModelProfiles = append(cfg.ModelProfiles, runtimeprofile.ModelProfile{ID: id + "-profile", RuntimeID: id, Model: id + "-model", Capabilities: []string{"go"}, Complexities: []string{"high"}})
		obs[id] = runtimeprofile.Observation{RuntimeID: id, Adapter: id, Installed: true, Available: true, Version: "test-1.0", Revision: 7, ObservedAt: time.Unix(1, 0), CapabilityStatus: map[string]runtimeprofile.CapabilityStatus{"go": runtimeprofile.CapabilityProven}}
	}
	state.Runtimes = project.Configured(rs)
	state.ModelProfiles = project.Configured(ps)
	p, issues := project.New(state)
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	return &sourceFake{snapshot: Snapshot{Project: p, Configuration: cfg, Observer: obs}}, obs
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
			if err != nil || !reflect.DeepEqual(binding.Choice, *preview.Choice) || binding.CredentialReference != "env:PRIVATE_TEST_REFERENCE" {
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

// Keep JSON leak assertions on the real public representation.

func wireString(p Preview) string { wire, _ := json.Marshal(p); return string(wire) }
func TestProjectResolutionBlocksWithoutFallback(t *testing.T) {
	cases := []struct {
		name, code string
		change     func(*sourceFake, observerFake)
	}{
		{"ambiguous", "ambiguous_match", func(*sourceFake, observerFake) {}},
		{"missing runtimes", "policy_unconfigured", func(s *sourceFake, _ observerFake) {
			state := s.snapshot.Project.State()
			state.Runtimes = project.Declaration[[]project.Runtime]{}
			state.ModelProfiles = project.Declaration[[]project.ModelProfile]{}
			s.snapshot.Project, _ = project.New(state)
		}},
		{"unconfigured preferences", "policy_unconfigured", func(s *sourceFake, _ observerFake) {
			state := s.snapshot.Project.State()
			state.RuntimePreferences = project.Unconfigured[[]project.RuntimePreference]()
			s.snapshot.Project, _ = project.New(state)
		}},
		{"unconfigured profiles", "policy_unconfigured", func(s *sourceFake, _ observerFake) {
			state := s.snapshot.Project.State()
			state.ModelProfiles = project.Unconfigured[[]project.ModelProfile]()
			s.snapshot.Project, _ = project.New(state)
		}},
		{"unavailable", "no_allowed_match", func(_ *sourceFake, o observerFake) {
			for id, v := range o {
				v.Available = false
				o[id] = v
			}
		}},
		{"capability declared", "no_allowed_match", func(_ *sourceFake, o observerFake) {
			for id, v := range o {
				v.CapabilityStatus["go"] = runtimeprofile.CapabilityDeclared
				o[id] = v
			}
		}},
		{"stale observation", "no_allowed_match", func(_ *sourceFake, o observerFake) {
			for id, v := range o {
				v.Revision--
				o[id] = v
			}
		}},
		{"wrong model", "no_allowed_match", func(s *sourceFake, _ observerFake) {
			for i := range s.snapshot.Configuration.ModelProfiles {
				s.snapshot.Configuration.ModelProfiles[i].Model = "wrong-model"
			}
		}},
		{"disallowed local codex", "no_allowed_match", func(s *sourceFake, _ observerFake) {
			state := s.snapshot.Project.State()
			state.Runtimes = project.Configured([]project.Runtime{{ID: "other"}})
			state.ModelProfiles = project.Configured([]project.ModelProfile{{Key: "other", RuntimeRef: project.Configured("other"), Model: project.Configured("model")}})
			s.snapshot.Project, _ = project.New(state)
		}},
		{"raw error", "policy_unavailable", func(s *sourceFake, _ observerFake) { s.err = errors.New("PRIVATE_SECRET") }},
		{"unsafe version", "invalid_observation", func(_ *sourceFake, o observerFake) {
			v := o["codex"]
			v.Version = "Bearer PRIVATE_SECRET"
			o["codex"] = v
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s, o := setup(t, "codex", "claude")
			tc.change(s, o)
			p, err := New(s).Preview(context.Background(), projectID, request())
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
	s.snapshot.Configuration.Preferences = []runtimeprofile.Preference{{Role: "implementation", Complexity: "high", ModelProfileID: "codex-profile"}}
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
		"config revision": func(s *sourceFake, _ observerFake) { s.snapshot.Configuration.Revision++ },
		"same revision config": func(s *sourceFake, _ observerFake) {
			s.snapshot.Configuration.Runtimes[0].CredentialReference = "env:NEW_TEST_REFERENCE"
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
			source.snapshot.Configuration.ModelProfiles[0].Capabilities = []string{runtimeadapter.IntegrationCapability, "go"}
			executable := filepath.Join(t.TempDir(), id)
			writeExecutable(t, executable, "original")
			ready := integrationFake(true)
			observe := func() {
				observer, err := runtimeadapter.NewExecutableObserver(7, time.Now(), []runtimeadapter.ObservedRuntime{{ID: id, Executable: executable, Integration: ready}})
				if err != nil {
					t.Fatal(err)
				}
				source.snapshot.Observer = observer
			}
			observe()
			service := New(source)
			proven := Request{Role: "implementation", Complexity: "high", Capabilities: []string{runtimeadapter.IntegrationCapability}}
			preview, err := service.Preview(context.Background(), projectID, proven)
			if err != nil || preview.Choice == nil || preview.Choice.ExecutableDigest == "" || preview.Choice.RuntimeVersion != "" {
				t.Fatalf("preview=%+v err=%v", preview, err)
			}
			if strings.Contains(wireString(preview), executable) {
				t.Fatal("preview exposed the executable path")
			}
			if unproven, err := service.Preview(context.Background(), projectID, request()); err == nil || unproven.Blocker.Code != "no_allowed_match" {
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

func writeExecutable(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte("#!/bin/sh\n# "+content+"\nexit 99\n"), 0o700); err != nil {
		t.Fatal(err)
	}
}
