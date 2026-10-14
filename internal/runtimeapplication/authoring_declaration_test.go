package runtimeapplication

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/runtimeprofile"
)

const (
	declarationDigest = "1111111111111111111111111111111111111111111111111111111111111111"
	declarationInput  = `{"declarationId":"codex-effort-sources","declarationVersion":1,"runtimeVersion":"synthetic-codex-1","executableDigest":"` + declarationDigest + `","invocationMode":"noninteractive","precedenceEstablished":true,"jsonKeys":[],"environmentKeys":[],"sources":[{"scope":"user","anchor":"user","path":".codex/config.toml","format":"toml","directory":false},{"scope":"project","anchor":"project","path":".codex/config.toml","format":"toml","directory":false},{"scope":"managed","anchor":"managed","path":"managed_config.toml","format":"toml","directory":false}]}`
)

// memoryStore is a synthetic canonical store: the token binds state and exact
// wire, Publish requires that token, and every write is counted.
type memoryStore struct {
	wire    []byte
	state   string
	writes  int
	refuse  bool
	pending bool
}

func (m *memoryStore) Inspect(context.Context) (LocalSnapshot, error) {
	if m.pending {
		return LocalSnapshot{State: "pending"}, nil
	}
	snapshot := LocalSnapshot{State: m.state}
	if m.state == "fresh" {
		snapshot.Token = "fresh"
		return snapshot, nil
	}
	v, err := runtimeprofile.DecodeVersioned(m.wire)
	if err != nil {
		return LocalSnapshot{State: "unsafe"}, err
	}
	switch {
	case v.V3 != nil:
		snapshot.State, snapshot.Revision, snapshot.V3 = "active-v3", v.V3.Revision, v.V3
	case v.Configuration.FormatVersion == runtimeprofile.FormatVersionV2:
		snapshot.State, snapshot.Revision, snapshot.Configuration = "active-v2", v.Configuration.Revision, *v.Configuration
	default:
		snapshot.State, snapshot.Revision, snapshot.Configuration = "active-v1", v.Configuration.Revision, *v.Configuration
	}
	sum := sha256.Sum256(append([]byte(snapshot.State+"\x00"), m.wire...))
	snapshot.Token = hex.EncodeToString(sum[:])
	return snapshot, nil
}

func (m *memoryStore) Publish(ctx context.Context, token string, wire []byte) (LocalSnapshot, error) {
	current, _ := m.Inspect(ctx)
	if token != current.Token {
		return current, errors.New("conflict")
	}
	if m.refuse {
		// An interrupted publication leaves recovery-required state.
		m.pending = true
		return LocalSnapshot{State: "pending"}, errors.New("publication unconfirmed")
	}
	m.writes++
	m.wire = append([]byte(nil), wire...)
	return m.Inspect(ctx)
}

func declarationSource() runtimeprofile.Configuration {
	return runtimeprofile.Configuration{FormatVersion: runtimeprofile.FormatVersionV2, Revision: 4,
		Runtimes:      []runtimeprofile.Runtime{{ID: "codex", Adapter: "codex", Enabled: true, AllowlistedProfileIDs: []string{"codex-high"}}, {ID: "claude", Adapter: "claude", Enabled: true, AllowlistedProfileIDs: []string{"claude-low"}}},
		ModelProfiles: []runtimeprofile.ModelProfile{{ID: "codex-high", RuntimeID: "codex", Model: "local-codex-model", Capabilities: []string{"go"}, Complexities: []string{"high"}, Effort: runtimeprofile.Effort{Mode: "explicit", Value: "high"}}, {ID: "claude-low", RuntimeID: "claude", Model: "local-claude-model", Capabilities: []string{"go"}, Complexities: []string{"low"}, Effort: runtimeprofile.Effort{Mode: "runtime-default", Value: "default"}}},
		Preferences:   []runtimeprofile.Preference{{Role: "implementation", Complexity: "high", ModelProfileID: "codex-high"}},
	}
}

func v2Store(t *testing.T) *memoryStore {
	t.Helper()
	wire, err := runtimeprofile.Encode(declarationSource())
	if err != nil {
		t.Fatal(err)
	}
	return &memoryStore{wire: wire}
}

// declarationHarness supplies the synthetic declaration file and the passive
// observation; it counts reads and never executes anything.
type declarationHarness struct {
	input    string
	observed runtimeprofile.EffortApplicability
	reads    int
	observe  func() runtimeprofile.EffortApplicability
}

func newHarness() *declarationHarness {
	return &declarationHarness{input: declarationInput, observed: runtimeprofile.EffortApplicability{RuntimeID: "codex", RuntimeVersion: "synthetic-codex-1", ExecutableDigest: declarationDigest, InvocationMode: "noninteractive"}}
}

func (h *declarationHarness) authoring(store AuthoringStore) Authoring {
	return Authoring{Store: store,
		ReadDeclaration: func(path string) ([]byte, error) {
			h.reads++
			if path != "declaration.json" || h.input == "" {
				return nil, errors.New("unreadable")
			}
			return []byte(h.input), nil
		},
		Applicability: func(_ context.Context, id string) (runtimeprofile.EffortApplicability, error) {
			if h.observe != nil {
				return h.observe(), nil
			}
			if id != h.observed.RuntimeID {
				return runtimeprofile.EffortApplicability{}, errors.New("unobserved")
			}
			return h.observed, nil
		}}
}

func migrateRequest() AuthoringRequest {
	return AuthoringRequest{Kind: "binding", Operation: "migrate", RuntimeID: "codex", TargetFormat: 3, EffortDeclaration: "declaration.json", MigrationEfforts: map[string]runtimeprofile.Effort{}}
}

func confirmed(req AuthoringRequest, preview AuthoringReport) AuthoringRequest {
	req.AuthorizeLocal, req.ExpectedRevision, req.PreviewDigest = true, preview.ExpectedRevision, preview.PreviewDigest
	return req
}

// BI-3.2: explicit preview (zero writes), exact confirmation, revision+1,
// carried Runtimes/Profiles/Preferences and canonical v3 bytes.
func TestDeclarationMigrationPreviewAndApply(t *testing.T) {
	ctx := context.Background()
	store, h := v2Store(t), newHarness()
	source := declarationSource()
	sourceWire := append([]byte(nil), store.wire...)
	preview, err := h.authoring(store).Run(ctx, migrateRequest())
	if err != nil || preview.Category != "preview" || store.writes != 0 || !bytes.Equal(store.wire, sourceWire) || h.reads != 1 {
		t.Fatalf("%+v %v writes=%d reads=%d", preview, err, store.writes, h.reads)
	}
	effects := strings.Join(preview.Effects, "\n")
	for _, want := range []string{"migrate local format 2 to 3", "source format 2 revision 4 digest ", "add effort declaration codex-effort-sources", "Runtime codex effort declaration codex-effort-sources applies to observed version synthetic-codex-1 executable " + declarationDigest + " mode noninteractive"} {
		if !strings.Contains(effects, want) {
			t.Fatalf("effect %q missing from %s", want, effects)
		}
	}
	if preview.Revision != 4 || preview.CandidateDigest == "" || preview.PreviewDigest == "" || preview.EffortSupport != "unproven" || preview.ModelSupport != "unproven" {
		t.Fatalf("%+v", preview)
	}
	applied, err := h.authoring(store).Run(ctx, confirmed(migrateRequest(), preview))
	if err != nil || applied.Category != "applied" || applied.State != "active-v3" || applied.Revision != 5 || store.writes != 1 {
		t.Fatalf("%+v %v", applied, err)
	}
	// Declaration is never capability proof.
	if applied.EffortSupport != "unproven" || applied.ModelSupport != "unproven" || applied.Authentication != "unknown" {
		t.Fatalf("proof inferred: %+v", applied)
	}
	v, err := runtimeprofile.DecodeVersioned(store.wire)
	if err != nil || v.V3 == nil {
		t.Fatal(err)
	}
	if canonical, _ := runtimeprofile.EncodeV3(*v.V3); !bytes.Equal(canonical, store.wire) || hexDigest(store.wire) != preview.CandidateDigest {
		t.Fatal("published bytes are not the previewed canonical candidate")
	}
	for i, p := range v.V3.ModelProfiles {
		want, _ := runtimeprofile.ProfileDigest(source.ModelProfiles[(i+1)%2])
		if got, _ := runtimeprofile.ProfileDigest(p); got != want {
			t.Fatalf("Profile %s digest changed", p.ID)
		}
	}
	for _, r := range v.V3.Runtimes {
		if r.ID == "claude" && r.EffortDeclarations != nil || r.ID == "codex" && len(r.EffortDeclarations) != 1 {
			t.Fatalf("declarations: %+v", r)
		}
	}
	if len(v.V3.Preferences) != 1 || v.V3.Preferences[0] != source.Preferences[0] {
		t.Fatal("Preferences not carried")
	}
	// The migrated record is listable and declarations stay unproven elsewhere.
	listed, err := h.authoring(store).Run(ctx, AuthoringRequest{Kind: "binding", Operation: "list"})
	if err != nil || listed.State != "active-v3" || len(listed.Runtimes) != 2 || len(listed.Profiles) != 2 {
		t.Fatalf("%+v %v", listed, err)
	}
}

func hexDigest(wire []byte) string { sum := sha256.Sum256(wire); return hex.EncodeToString(sum[:]) }

// BI-3.1: add or replace by declarationId on active-v3, never implicitly.
func TestDeclarationEditOnV3(t *testing.T) {
	ctx := context.Background()
	store, h := v2Store(t), newHarness()
	preview, _ := h.authoring(store).Run(ctx, migrateRequest())
	if _, err := h.authoring(store).Run(ctx, confirmed(migrateRequest(), preview)); err != nil {
		t.Fatal(err)
	}
	edit := AuthoringRequest{Kind: "binding", Operation: "edit", RuntimeID: "codex", EffortDeclaration: "declaration.json"}
	same, err := h.authoring(store).Run(ctx, edit)
	if err != nil || same.Category != "unchanged" {
		t.Fatalf("identical declaration: %+v %v", same, err)
	}
	h.input = strings.Replace(declarationInput, `"environmentKeys":[]`, `"environmentKeys":["CODEX_EFFORT"]`, 1)
	replace, err := h.authoring(store).Run(ctx, edit)
	if err != nil || replace.Category != "preview" || !strings.Contains(strings.Join(replace.Effects, "\n"), "replace effort declaration codex-effort-sources") {
		t.Fatalf("%+v %v", replace, err)
	}
	applied, err := h.authoring(store).Run(ctx, confirmed(edit, replace))
	if err != nil || applied.Revision != 6 || store.writes != 2 {
		t.Fatalf("%+v %v", applied, err)
	}
	// A different declarationId for the same applicability tuple is ambiguous.
	h.input = strings.Replace(declarationInput, "codex-effort-sources", "codex-alias", 1)
	if got, err := h.authoring(store).Run(ctx, edit); err == nil || got.Category != "candidate_invalid" || store.writes != 2 {
		t.Fatalf("ambiguous: %+v %v", got, err)
	}
	// Ordinary v2 mutations never touch format 3 (no downgrade).
	enabled := false
	if got, err := h.authoring(store).Run(ctx, AuthoringRequest{Kind: "binding", Operation: "edit", RuntimeID: "codex", Enabled: &enabled}); err == nil || got.Category != "configuration_unavailable" || store.writes != 2 {
		t.Fatalf("v2 mutation over v3: %+v %v", got, err)
	}
	if got, err := h.authoring(store).Run(ctx, migrateRequest()); err == nil || got.Category != "migration_mapping_required" {
		t.Fatalf("re-migration: %+v %v", got, err)
	}
}

// BI-3.3: every refusal is fail-closed with no write.
func TestDeclarationRefusals(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		name, category string
		req            func() AuthoringRequest
		setup          func(*memoryStore, *declarationHarness)
	}{
		{"no-target-format", "invalid_input", func() AuthoringRequest { r := migrateRequest(); r.TargetFormat = 0; return r }, nil},
		{"target-format-2", "invalid_input", func() AuthoringRequest { r := migrateRequest(); r.TargetFormat = 2; return r }, nil},
		{"target-format-without-declaration", "invalid_input", func() AuthoringRequest { r := migrateRequest(); r.EffortDeclaration = ""; return r }, nil},
		{"target-format-on-edit", "invalid_input", func() AuthoringRequest { r := migrateRequest(); r.Operation = "edit"; return r }, nil},
		{"declaration-on-profile", "invalid_input", func() AuthoringRequest { r := migrateRequest(); r.Kind = "profile"; return r }, nil},
		{"missing-runtime", "invalid_input", func() AuthoringRequest { r := migrateRequest(); r.RuntimeID = ""; return r }, nil},
		{"mixed-v1-mapping", "invalid_input", func() AuthoringRequest {
			r := migrateRequest()
			r.MigrationEfforts = map[string]runtimeprofile.Effort{"codex-high": {Mode: "explicit", Value: "high"}}
			return r
		}, nil},
		{"declaration-on-v2-edit", "migration_required", func() AuthoringRequest {
			return AuthoringRequest{Kind: "binding", Operation: "edit", RuntimeID: "codex", EffortDeclaration: "declaration.json"}
		}, nil},
		{"missing-file", "invalid_input", migrateRequest, func(_ *memoryStore, h *declarationHarness) { h.input = "" }},
		{"malformed", "invalid_input", migrateRequest, func(_ *memoryStore, h *declarationHarness) { h.input = declarationInput[:50] }},
		{"oversized", "invalid_input", migrateRequest, func(_ *memoryStore, h *declarationHarness) {
			h.input = declarationInput + strings.Repeat(" ", runtimeprofile.MaxConfigurationBytes)
		}},
		{"unknown-field", "invalid_input", migrateRequest, func(_ *memoryStore, h *declarationHarness) {
			h.input = strings.Replace(declarationInput, `"jsonKeys":[]`, `"jsonKeys":[],"capabilityStatus":"proven"`, 1)
		}},
		{"duplicate-field", "invalid_input", migrateRequest, func(_ *memoryStore, h *declarationHarness) {
			h.input = strings.Replace(declarationInput, `"jsonKeys":[]`, `"jsonKeys":[],"jsonKeys":[]`, 1)
		}},
		{"wrong-runtime-rules", "invalid_input", func() AuthoringRequest { r := migrateRequest(); r.RuntimeID = "claude"; return r }, nil},
		{"wrong-executable", "effort_configuration_unproven", migrateRequest, func(_ *memoryStore, h *declarationHarness) {
			h.observed.ExecutableDigest = strings.Repeat("9", 64)
		}},
		{"version-mismatch", "effort_configuration_unproven", migrateRequest, func(_ *memoryStore, h *declarationHarness) {
			h.observed.RuntimeVersion = "synthetic-codex-2"
		}},
		{"unobserved", "effort_configuration_unproven", migrateRequest, func(_ *memoryStore, h *declarationHarness) { h.observed.RuntimeID = "none" }},
		{"recovery-required", "recovery_required", migrateRequest, func(m *memoryStore, _ *declarationHarness) { m.pending = true }},
		{"fresh", "migration_mapping_required", migrateRequest, func(m *memoryStore, _ *declarationHarness) { m.state = "fresh" }},
		{"v1", "migration_required", migrateRequest, func(m *memoryStore, _ *declarationHarness) {
			cfg := declarationSource()
			cfg.FormatVersion = runtimeprofile.FormatVersion
			for i := range cfg.ModelProfiles {
				cfg.ModelProfiles[i].Effort = runtimeprofile.Effort{}
			}
			var err error
			if m.wire, err = runtimeprofile.Encode(cfg); err != nil {
				t.Fatal(err)
			}
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			store, h := v2Store(t), newHarness()
			if c.setup != nil {
				c.setup(store, h)
			}
			before := append([]byte(nil), store.wire...)
			got, err := h.authoring(store).Run(ctx, c.req())
			if err == nil || got.Category != c.category || store.writes != 0 || !bytes.Equal(store.wire, before) {
				t.Fatalf("%+v %v writes=%d", got, err, store.writes)
			}
		})
	}
	// The Runtime must be configured; the declaration is never attached elsewhere.
	store, h := v2Store(t), newHarness()
	cfg := declarationSource()
	cfg.Runtimes, cfg.ModelProfiles, cfg.Preferences = cfg.Runtimes[1:], cfg.ModelProfiles[1:], nil
	store.wire, _ = runtimeprofile.Encode(cfg)
	if got, err := h.authoring(store).Run(ctx, migrateRequest()); err == nil || got.Category != "candidate_invalid" || store.writes != 0 {
		t.Fatalf("unconfigured Runtime: %+v %v", got, err)
	}
}

// BI-3.2 items 8–9: inexact or drifted confirmation is stale_preview with no
// write; an unconfirmed publication is never success.
func TestDeclarationConfirmationDrift(t *testing.T) {
	ctx := context.Background()
	drift := map[string]struct {
		category string
		change   func(*memoryStore, *declarationHarness, *AuthoringRequest)
	}{
		"inexact-digest": {"stale_preview", func(_ *memoryStore, _ *declarationHarness, r *AuthoringRequest) {
			r.PreviewDigest = strings.Repeat("0", 64)
		}},
		"stale-candidate": {"stale_preview", func(_ *memoryStore, _ *declarationHarness, r *AuthoringRequest) { r.PreviewDigest = r.ExpectedRevision }},
		"stale-snapshot": {"stale_preview", func(m *memoryStore, _ *declarationHarness, _ *AuthoringRequest) {
			cfg := declarationSource()
			cfg.Revision = 5
			m.wire, _ = runtimeprofile.Encode(cfg)
		}},
		"changed-declaration": {"stale_preview", func(_ *memoryStore, h *declarationHarness, _ *AuthoringRequest) {
			h.input = strings.Replace(declarationInput, `"environmentKeys":[]`, `"environmentKeys":["CODEX_EFFORT"]`, 1)
		}},
		// A replaced executable no longer matches the declaration at all.
		"changed-executable": {"effort_configuration_unproven", func(_ *memoryStore, h *declarationHarness, _ *AuthoringRequest) {
			h.observed.ExecutableDigest = strings.Repeat("8", 64)
		}},
		// Drift between the apply's observation and the write is caught by the
		// revalidation immediately before publication.
		"executable-drift-during-apply": {"stale_preview", func(_ *memoryStore, h *declarationHarness, _ *AuthoringRequest) {
			calls := 0
			h.observe = func() runtimeprofile.EffortApplicability {
				calls++
				o := h.observed
				if calls > 1 {
					o.ExecutableDigest = strings.Repeat("7", 64)
				}
				return o
			}
		}},
		"inexact-revision": {"stale_preview", func(_ *memoryStore, _ *declarationHarness, r *AuthoringRequest) { r.ExpectedRevision = "other" }},
	}
	for name, c := range drift {
		t.Run(name, func(t *testing.T) {
			store, h := v2Store(t), newHarness()
			preview, err := h.authoring(store).Run(ctx, migrateRequest())
			if err != nil {
				t.Fatal(err)
			}
			req := confirmed(migrateRequest(), preview)
			c.change(store, h, &req)
			before := append([]byte(nil), store.wire...)
			got, err := h.authoring(store).Run(ctx, req)
			if err == nil || got.Category != c.category || store.writes != 0 || !bytes.Equal(store.wire, before) {
				t.Fatalf("%+v %v writes=%d", got, err, store.writes)
			}
		})
	}
	t.Run("authority-without-authorize-local", func(t *testing.T) {
		store, h := v2Store(t), newHarness()
		preview, _ := h.authoring(store).Run(ctx, migrateRequest())
		req := confirmed(migrateRequest(), preview)
		req.AuthorizeLocal = false
		if got, err := h.authoring(store).Run(ctx, req); err == nil || got.Category != "local_authority_required" || store.writes != 0 {
			t.Fatalf("%+v %v", got, err)
		}
	})
	t.Run("interrupted-publication", func(t *testing.T) {
		store, h := v2Store(t), newHarness()
		preview, _ := h.authoring(store).Run(ctx, migrateRequest())
		store.refuse = true
		got, err := h.authoring(store).Run(ctx, confirmed(migrateRequest(), preview))
		if err == nil || got.Category != "publication_unconfirmed" || store.writes != 0 {
			t.Fatalf("%+v %v", got, err)
		}
		// No automatic retry: the next attempt requires scoped recovery first.
		if got, err := h.authoring(store).Run(ctx, confirmed(migrateRequest(), preview)); err == nil || got.Category != "recovery_required" || store.writes != 0 {
			t.Fatalf("%+v %v", got, err)
		}
	})
}
