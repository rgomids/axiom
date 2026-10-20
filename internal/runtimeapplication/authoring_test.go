package runtimeapplication

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/runtimeprofile"
)

type memoryAuthoringStore struct {
	snapshot      LocalSnapshot
	reads, writes int
	wire          []byte
	publishErr    error
}

func (s *memoryAuthoringStore) Inspect(context.Context) (LocalSnapshot, error) {
	s.reads++
	return s.snapshot, nil
}
func (s *memoryAuthoringStore) Publish(_ context.Context, token string, w []byte) (LocalSnapshot, error) {
	s.writes++
	if s.publishErr != nil {
		return LocalSnapshot{}, s.publishErr
	}
	if token != s.snapshot.Token {
		return LocalSnapshot{}, errors.New("stale")
	}
	s.wire = append([]byte(nil), w...)
	if v, err := runtimeprofile.DecodeVersioned(w); err == nil && v.V4 != nil {
		s.snapshot = LocalSnapshot{State: "active-v4", Revision: v.V4.Revision, V4: v.V4, Token: fmt.Sprintf("snapshot-%d", v.V4.Revision)}
	} else if ts, err := runtimeprofile.DecodeTombstone(w); err == nil {
		s.snapshot = LocalSnapshot{State: "retired", Revision: ts.Revision, Token: fmt.Sprintf("snapshot-%d", ts.Revision)}
	} else {
		return LocalSnapshot{}, errors.New("invalid publication")
	}
	return s.snapshot, nil
}
func createRequest() AuthoringRequest {
	return AuthoringRequest{Kind: "binding", Operation: "create", RuntimeID: "codex", ProfileID: "operator-review", Model: "operator-model", Effort: runtimeprofile.Effort{Mode: "explicit", Value: "high"}, Capabilities: []string{"go"}, Complexities: []string{"high"}}
}
func authoringSetup() (Authoring, *memoryAuthoringStore) {
	store := &memoryAuthoringStore{snapshot: LocalSnapshot{State: "fresh", Token: "fresh-token"}}
	return Authoring{Store: store, Installed: func(string) bool { return true }}, store
}
func applyReviewed(t *testing.T, a Authoring, r AuthoringRequest) AuthoringReport {
	t.Helper()
	p, err := a.Run(context.Background(), r)
	if err != nil || p.Category != "preview" {
		t.Fatalf("preview: %+v %v", p, err)
	}
	r.AuthorizeLocal = true
	r.ExpectedRevision = p.ExpectedRevision
	r.PreviewDigest = p.PreviewDigest
	got, err := a.Run(context.Background(), r)
	if err != nil || got.Category != "applied" {
		t.Fatalf("apply: %+v %v", got, err)
	}
	return got
}
func TestAuthoringConfirmedLifecycleReplayAndMonotonicRetirement(t *testing.T) {
	a, s := authoringSetup()
	r := createRequest()
	preview, err := a.Run(context.Background(), r)
	if err != nil || preview.Category != "preview" || s.writes != 0 {
		t.Fatal("preview mutated", err)
	}
	confirmed := r
	confirmed.AuthorizeLocal = true
	confirmed.ExpectedRevision = preview.ExpectedRevision
	confirmed.PreviewDigest = preview.PreviewDigest
	applied, err := a.Run(context.Background(), confirmed)
	if err != nil || applied.Revision != 1 || s.writes != 1 {
		t.Fatal("first create failed", err)
	}
	if _, err := a.Run(context.Background(), confirmed); err == nil || s.writes != 1 {
		t.Fatal("replayed creation wrote")
	}
	profile := AuthoringRequest{Kind: "profile", Operation: "create", RuntimeID: "codex", ProfileID: "second", Model: "operator-model", Effort: runtimeprofile.Effort{Mode: "runtime-default", Value: "default"}, Capabilities: []string{"go"}}
	applyReviewed(t, a, profile)
	if s.snapshot.Revision != 2 || len(s.snapshot.V4.ModelProfiles) != 2 {
		t.Fatal("profile not created")
	}
	lastDigest, _ := runtimeprofile.DigestV4(*s.snapshot.V4)
	remove := AuthoringRequest{Kind: "binding", Operation: "remove", RuntimeID: "codex", Cascade: true}
	retired := applyReviewed(t, a, remove)
	if retired.State != "retired" || retired.Revision != 3 {
		t.Fatal("retirement lost generation")
	}
	ts, err := runtimeprofile.DecodeTombstone(s.wire)
	if err != nil || ts.Revision != 3 || ts.FormatVersion != 1 {
		t.Fatal("not tombstone")
	}
	if ts.PriorConfigurationDigest != lastDigest {
		t.Fatal("retirement not bound to DigestV4 of the exact prior record")
	}
	recreated := applyReviewed(t, a, r)
	if recreated.Revision != 4 {
		t.Fatal("recreation reset generation")
	}
}
func TestAuthoringStaleCandidateAndNoOpZeroWrites(t *testing.T) {
	a, s := authoringSetup()
	applyReviewed(t, a, createRequest())
	r := AuthoringRequest{Kind: "profile", Operation: "edit", ProfileID: "operator-review", Model: "new-model"}
	p, err := a.Run(context.Background(), r)
	if err != nil {
		t.Fatal(err)
	}
	r.AuthorizeLocal = true
	r.ExpectedRevision = p.ExpectedRevision
	r.PreviewDigest = p.PreviewDigest
	r.Model = "other-model"
	if got, err := a.Run(context.Background(), r); err == nil || got.Category != "stale_preview" || s.writes != 1 {
		t.Fatal("changed candidate approved")
	}
	r = AuthoringRequest{Kind: "profile", Operation: "edit", ProfileID: "operator-review", Model: "operator-model"}
	got, err := a.Run(context.Background(), r)
	if err != nil || got.Category != "unchanged" || s.writes != 1 || s.snapshot.Revision != 1 {
		t.Fatal("no-op wrote or incremented")
	}
	r = AuthoringRequest{Kind: "profile", Operation: "remove", ProfileID: "operator-review"}
	got, err = a.Run(context.Background(), r)
	if err == nil || got.Category != "compound_removal_required" || s.writes != 1 {
		t.Fatal("dangling final removal allowed")
	}
}

// legacySnapshot is a recognized pre-stable record as the store reports it.
func legacySnapshot(t *testing.T, format int) LocalSnapshot {
	t.Helper()
	cfg := runtimeprofile.Configuration{FormatVersion: format, Revision: 7,
		Runtimes:      []runtimeprofile.Runtime{{ID: "codex", Adapter: "codex", Enabled: true, AllowlistedProfileIDs: []string{"codex-profile"}}, {ID: "claude", Adapter: "claude", Enabled: false, AllowlistedProfileIDs: []string{"claude-profile"}}},
		ModelProfiles: []runtimeprofile.ModelProfile{{ID: "codex-profile", RuntimeID: "codex", Model: "codex-model", Capabilities: []string{"go"}}, {ID: "claude-profile", RuntimeID: "claude", Model: "claude-model", Capabilities: []string{"go"}}},
		Preferences:   []runtimeprofile.Preference{{Role: "review", Complexity: "high", ModelProfileID: "claude-profile"}},
	}
	if format == runtimeprofile.FormatVersionV2 {
		for i := range cfg.ModelProfiles {
			cfg.ModelProfiles[i].Effort = runtimeprofile.Effort{Mode: "runtime-default", Value: "default"}
		}
	}
	if format == runtimeprofile.FormatVersionV3 {
		v3 := runtimeprofile.ConfigurationV3{FormatVersion: format, Revision: cfg.Revision, ModelProfiles: cfg.ModelProfiles, Preferences: cfg.Preferences}
		for i := range v3.ModelProfiles {
			v3.ModelProfiles[i].Effort = runtimeprofile.Effort{Mode: "runtime-default", Value: "default"}
		}
		for _, r := range cfg.Runtimes {
			v3.Runtimes = append(v3.Runtimes, runtimeprofile.RuntimeV3{ID: r.ID, Adapter: r.Adapter, Enabled: r.Enabled, AllowlistedProfileIDs: r.AllowlistedProfileIDs})
		}
		if runtimeprofile.ValidateV3(v3) != nil {
			t.Fatal("invalid historical v3 fixture")
		}
		return LocalSnapshot{State: "active-v3", Revision: 7, V3: &v3, Token: "legacy-token"}
	}
	if runtimeprofile.Validate(cfg) != nil {
		t.Fatal("invalid legacy fixture")
	}
	return LocalSnapshot{State: fmt.Sprintf("active-v%d", format), Revision: 7, Legacy: cfg, Token: "legacy-token"}
}

func historicalDigest(s LocalSnapshot) string {
	if s.V3 != nil {
		d, _ := runtimeprofile.DigestV3(*s.V3)
		return d
	}
	d, _ := runtimeprofile.Digest(s.Legacy)
	return d
}

// Specification 007 PM-11 A1: `binding remove --runtime <id> --cascade` is the
// only operation over a pre-stable record and retires the ENTIRE record through
// the existing tombstone bound to the historical Digest, after which canonical
// v3 is created explicitly. Nothing is migrated or partially rewritten.
func TestAuthoringLegacyFullRetirement(t *testing.T) {
	for _, format := range []int{runtimeprofile.FormatVersion, runtimeprofile.FormatVersionV2, runtimeprofile.FormatVersionV3} {
		t.Run(fmt.Sprintf("v%d", format), func(t *testing.T) {
			a, s := authoringSetup()
			s.snapshot = legacySnapshot(t, format)
			historical := historicalDigest(s.snapshot)
			retire := AuthoringRequest{Kind: "binding", Operation: "remove", RuntimeID: "codex", Cascade: true}
			for name, tc := range map[string]struct {
				r    AuthoringRequest
				want string
			}{
				"missing cascade":   {AuthoringRequest{Kind: "binding", Operation: "remove", RuntimeID: "codex"}, "compound_removal_required"},
				"absent runtime":    {AuthoringRequest{Kind: "binding", Operation: "remove", RuntimeID: "other", Cascade: true}, "binding_missing"},
				"no runtime":        {AuthoringRequest{Kind: "binding", Operation: "remove", Cascade: true}, "binding_missing"},
				"profile remove":    {AuthoringRequest{Kind: "profile", Operation: "remove", ProfileID: "codex-profile", Cascade: true}, "unsupported_configuration_format"},
				"legacy create":     {createRequest(), "unsupported_configuration_format"},
				"legacy edit":       {AuthoringRequest{Kind: "binding", Operation: "edit", RuntimeID: "codex", Enabled: new(bool)}, "unsupported_configuration_format"},
				"legacy enable":     {AuthoringRequest{Kind: "profile", Operation: "enable", ProfileID: "claude-profile"}, "unsupported_configuration_format"},
				"legacy disable":    {AuthoringRequest{Kind: "binding", Operation: "disable", RuntimeID: "codex"}, "unsupported_configuration_format"},
				"legacy validate":   {AuthoringRequest{Kind: "profile", Operation: "validate-candidate", ProfileID: "codex-profile", Model: "m"}, "unsupported_configuration_format"},
				"migrate operation": {AuthoringRequest{Kind: "profile", Operation: "migrate"}, "invalid_input"},
			} {
				if got, err := a.Run(context.Background(), tc.r); err == nil || got.Category != tc.want {
					t.Fatalf("%s: %+v %v", name, got, err)
				}
			}
			if listed, err := a.Run(context.Background(), AuthoringRequest{Kind: "profile", Operation: "list"}); err != nil || listed.Category != "inspected" || len(listed.Profiles) != 0 || !strings.Contains(listed.Next, "--cascade") {
				t.Fatalf("legacy list: %+v %v", listed, err)
			}
			preview, err := a.Run(context.Background(), retire)
			if err != nil || preview.Category != "preview" || s.writes != 0 || s.reads == 0 {
				t.Fatalf("preview: %+v %v", preview, err)
			}
			effects := strings.Join(preview.Effects, "\n")
			for _, want := range []string{"retire ENTIRE historical local configuration format", "not only Runtime codex", "retire Runtime codex enabled=true allowlist=codex-profile", "retire Runtime claude enabled=false allowlist=claude-profile", "retire Profile codex-profile runtime=codex", "retire Profile claude-profile runtime=claude", "retire local preference review/high referring to Profile claude-profile", historical, "nothing is migrated"} {
				if !strings.Contains(effects, want) {
					t.Fatalf("preview scope lacks %q: %s", want, effects)
				}
			}
			if len(preview.Runtimes) != 2 || len(preview.Profiles) != 2 || preview.ExpectedRevision != "legacy-token" || preview.State != s.snapshot.State {
				t.Fatalf("preview scope %+v", preview)
			}
			confirm := retire
			confirm.AuthorizeLocal, confirm.ExpectedRevision, confirm.PreviewDigest = true, preview.ExpectedRevision, preview.PreviewDigest
			for name, r := range map[string]AuthoringRequest{
				"no authorization":  {Kind: "binding", Operation: "remove", RuntimeID: "codex", Cascade: true, ExpectedRevision: preview.ExpectedRevision, PreviewDigest: preview.PreviewDigest},
				"wrong revision":    {Kind: "binding", Operation: "remove", RuntimeID: "codex", Cascade: true, AuthorizeLocal: true, ExpectedRevision: "other", PreviewDigest: preview.PreviewDigest},
				"wrong digest":      {Kind: "binding", Operation: "remove", RuntimeID: "codex", Cascade: true, AuthorizeLocal: true, ExpectedRevision: preview.ExpectedRevision, PreviewDigest: strings.Repeat("0", 64)},
				"other runtime arg": {Kind: "binding", Operation: "remove", RuntimeID: "claude", Cascade: true, AuthorizeLocal: true, ExpectedRevision: preview.ExpectedRevision, PreviewDigest: preview.PreviewDigest},
			} {
				if got, err := a.Run(context.Background(), r); err == nil || got.Category == "applied" || s.writes != 0 {
					t.Fatalf("%s: %+v %v", name, got, err)
				}
			}
			// Source drift since the preview: the token and digest no longer match.
			drifted := legacySnapshot(t, format)
			if drifted.V3 != nil {
				drifted.V3.ModelProfiles[0].Model = "changed-model"
			} else {
				drifted.Legacy.ModelProfiles[0].Model = "changed-model"
			}
			drifted.Token = "drifted-token"
			original := s.snapshot
			s.snapshot = drifted
			if got, err := a.Run(context.Background(), confirm); err == nil || got.Category != "stale_preview" || s.writes != 0 {
				t.Fatalf("drift: %+v %v", got, err)
			}
			s.snapshot = original
			applied, err := a.Run(context.Background(), confirm)
			if err != nil || applied.Category != "applied" || s.writes != 1 || applied.State != "retired" || applied.Revision != 8 {
				t.Fatalf("apply: %+v %v", applied, err)
			}
			tomb, err := runtimeprofile.DecodeTombstone(s.wire)
			if err != nil || tomb.FormatVersion != 1 || tomb.Kind != runtimeprofile.TombstoneKind || tomb.Revision != 8 || tomb.PriorConfigurationDigest != historical {
				t.Fatalf("tombstone %+v %v", tomb, err)
			}
			if s.snapshot.V4 != nil {
				t.Fatal("retirement created canonical configuration")
			}
			created := applyReviewed(t, a, createRequest())
			if created.State != "active-v4" || created.Revision != 9 || s.snapshot.V4 == nil || len(s.snapshot.V4.Runtimes) != 1 || len(s.snapshot.V4.ModelProfiles) != 1 || s.snapshot.V4.ModelProfiles[0].ID != "operator-review" {
				t.Fatalf("explicit create after retirement: %+v %+v", created, s.snapshot.V4)
			}
		})
	}
}

func TestAuthoringLegacyMalformedRefusesRetirement(t *testing.T) {
	a, s := authoringSetup()
	s.snapshot = legacySnapshot(t, runtimeprofile.FormatVersionV2)
	s.snapshot.Legacy.ModelProfiles[0].Effort = runtimeprofile.Effort{}
	got, err := a.Run(context.Background(), AuthoringRequest{Kind: "binding", Operation: "remove", RuntimeID: "codex", Cascade: true})
	if err == nil || got.Category != "configuration_invalid" || s.writes != 0 {
		t.Fatalf("%+v %v", got, err)
	}
}

func TestAuthoringSafetyAndReadEffects(t *testing.T) {
	for _, canary := range []string{"token=synthetic", "https://example.invalid/token=synthetic", "%2574oken%253Dsynthetic", "/Users/operator/private", "$(command)", "sk-synthetic"} {
		a, s := authoringSetup()
		r := createRequest()
		r.Model = canary
		got, err := a.Run(context.Background(), r)
		w, _ := json.Marshal(got)
		if err == nil || strings.Contains(string(w), canary) || s.writes != 0 {
			t.Fatal("unsafe candidate accepted or leaked")
		}
	}
	a, s := authoringSetup()
	a.Installed = func(string) bool { t.Fatal("passive read invoked installed callback"); return false }
	for _, op := range []string{"list", "show"} {
		got, err := a.Run(context.Background(), AuthoringRequest{Kind: "profile", Operation: op})
		if err != nil || got.Category != "inspected" || s.writes != 0 {
			t.Fatal("read side effect")
		}
	}
}
func TestAuthoringPublicationUncertaintyNotClaimedApplied(t *testing.T) {
	a, s := authoringSetup()
	r := createRequest()
	p, err := a.Run(context.Background(), r)
	if err != nil {
		t.Fatal(err)
	}
	s.publishErr = errors.New("raw private error")
	r.AuthorizeLocal = true
	r.ExpectedRevision = p.ExpectedRevision
	r.PreviewDigest = p.PreviewDigest
	got, err := a.Run(context.Background(), r)
	w, _ := json.Marshal(got)
	if err == nil || got.Category != "publication_unconfirmed" || strings.Contains(string(w), "raw private") {
		t.Fatal("uncertain publication reported applied or leaked")
	}
}

func TestAuthoringLegacyDTOCanariesRedacted(t *testing.T) {
	a, s := authoringSetup()
	s.snapshot = legacySnapshot(t, runtimeprofile.FormatVersion)
	cfg := &s.snapshot.Legacy
	cfg.ModelProfiles[0].ID = "sk-synthetic-id"
	cfg.Runtimes[0].AllowlistedProfileIDs = []string{"sk-synthetic-id"}
	cfg.ModelProfiles[0].Model = "sk-synthetic-model"
	cfg.Preferences = nil
	for _, r := range []AuthoringRequest{{Kind: "profile", Operation: "list"}, {Kind: "binding", Operation: "remove", RuntimeID: "codex", Cascade: true}} {
		got, err := a.Run(context.Background(), r)
		wire, _ := json.Marshal(got)
		if err != nil || strings.Contains(string(wire), "synthetic") || s.writes != 0 {
			t.Fatalf("legacy DTO canary exposed: %s %v", wire, err)
		}
	}
}
func TestAuthoringValidateCandidateValidatesSuppliedIntent(t *testing.T) {
	a, s := authoringSetup()
	r := createRequest()
	r.Operation = "validate-candidate"
	got, err := a.Run(context.Background(), r)
	if err != nil || got.Category != "candidate_valid" || len(got.Profiles) != 1 || got.Profiles[0].Model != r.Model || s.writes != 0 {
		t.Fatal("candidate validation ignored supplied fresh intent")
	}
	applyReviewed(t, a, createRequest())
	r = AuthoringRequest{Kind: "profile", Operation: "validate-candidate", ProfileID: "operator-review", Model: "edited-model"}
	got, err = a.Run(context.Background(), r)
	if err != nil || len(got.Profiles) != 1 || got.Profiles[0].Model != "edited-model" || s.snapshot.V4.ModelProfiles[0].Model != "operator-model" || s.writes != 1 {
		t.Fatal("candidate validation ignored edit or wrote state")
	}
}
