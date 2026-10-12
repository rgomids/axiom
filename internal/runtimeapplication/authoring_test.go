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
	if cfg, err := runtimeprofile.Decode(w); err == nil {
		s.snapshot = LocalSnapshot{State: "active-v2", Revision: cfg.Revision, Configuration: cfg, Token: fmt.Sprintf("snapshot-%d", cfg.Revision)}
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
	if s.snapshot.Revision != 2 || len(s.snapshot.Configuration.ModelProfiles) != 2 {
		t.Fatal("profile not created")
	}
	remove := AuthoringRequest{Kind: "binding", Operation: "remove", RuntimeID: "codex", Cascade: true}
	retired := applyReviewed(t, a, remove)
	if retired.State != "retired" || retired.Revision != 3 {
		t.Fatal("retirement lost generation")
	}
	ts, err := runtimeprofile.DecodeTombstone(s.wire)
	if err != nil || ts.Revision != 3 {
		t.Fatal("not tombstone")
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
func TestAuthoringMigrationRequiresEveryExplicitEffortAndLegacyConsent(t *testing.T) {
	a, s := authoringSetup()
	src, _ := setup(t, "codex", "claude")
	s.snapshot = LocalSnapshot{State: "active-v1", Revision: 7, Configuration: src.snapshot.Configuration, Token: "legacy-token"}
	before, _ := runtimeprofile.Encode(s.snapshot.Configuration)
	r := AuthoringRequest{Kind: "profile", Operation: "migrate", MigrationEfforts: map[string]runtimeprofile.Effort{"codex-profile": {Mode: "explicit", Value: "high"}}}
	if got, err := a.Run(context.Background(), r); err == nil || got.Category != "migration_mapping_required" {
		t.Fatal("partial migration accepted")
	}
	r.MigrationEfforts["claude-profile"] = runtimeprofile.Effort{Mode: "runtime-default", Value: "default"}
	if got, err := a.Run(context.Background(), r); err == nil || got.Category != "legacy_reference_removal_required" {
		t.Fatal("legacy reference dropped without consent")
	}
	r.ClearLegacyReferences = true
	p, err := a.Run(context.Background(), r)
	if err != nil || s.writes != 0 {
		t.Fatal("migration preview failed")
	}
	after, _ := runtimeprofile.Encode(s.snapshot.Configuration)
	if string(before) != string(after) {
		t.Fatal("migration preview rewrote v1")
	}
	r.AuthorizeLocal = true
	r.ExpectedRevision = p.ExpectedRevision
	r.PreviewDigest = p.PreviewDigest
	got, err := a.Run(context.Background(), r)
	if err != nil || got.Revision != 8 || s.snapshot.Configuration.FormatVersion != 2 {
		t.Fatal("migration not one revision")
	}
	for _, rt := range s.snapshot.Configuration.Runtimes {
		if rt.CredentialReference != "" {
			t.Fatal("credential reference survived")
		}
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
func TestAuthoringReadLegacyUnsafeModelRedacted(t *testing.T) {
	a, s := authoringSetup()
	src, _ := setup(t, "codex")
	src.snapshot.Configuration.ModelProfiles[0].Model = "token=synthetic"
	s.snapshot = LocalSnapshot{State: "active-v1", Revision: 7, Configuration: src.snapshot.Configuration, Token: "legacy-token"}
	r, err := a.Run(context.Background(), AuthoringRequest{Kind: "profile", Operation: "list"})
	w, _ := json.Marshal(r)
	if err != nil || strings.Contains(string(w), "synthetic") || len(r.Profiles) != 1 || r.Profiles[0].Model != "redacted" || s.writes != 0 {
		t.Fatal("unsafe historical read exposed data")
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
	src, _ := setup(t, "codex")
	cfg := src.snapshot.Configuration
	cfg.ModelProfiles[0].ID = "sk-synthetic-id"
	cfg.Runtimes[0].AllowlistedProfileIDs = []string{"sk-synthetic-id"}
	cfg.ModelProfiles[0].Model = "sk-synthetic-model"
	cfg.ModelProfiles[0].Effort = runtimeprofile.Effort{Mode: "explicit", Value: "token=synthetic-effort"}
	s.snapshot = LocalSnapshot{State: "active-v1", Revision: 7, Configuration: cfg, Token: "legacy-token"}
	got, err := a.Run(context.Background(), AuthoringRequest{Kind: "profile", Operation: "list"})
	wire, _ := json.Marshal(got)
	if err != nil || strings.Contains(string(wire), "synthetic") {
		t.Fatal("legacy DTO canary exposed")
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
	if err != nil || len(got.Profiles) != 1 || got.Profiles[0].Model != "edited-model" || s.snapshot.Configuration.ModelProfiles[0].Model != "operator-model" || s.writes != 1 {
		t.Fatal("candidate validation ignored edit or wrote state")
	}
}
