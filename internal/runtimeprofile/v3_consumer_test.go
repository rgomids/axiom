package runtimeprofile_test

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/runtimeprofile"
	fx "github.com/rgomids/axiom/internal/runtimeprofile/runtimeprofiletest"
)

func syntheticV3Fixtures() map[string]runtimeprofile.ConfigurationV3 {
	return map[string]runtimeprofile.ConfigurationV3{
		"codex only":            fx.CodexOnly(),
		"claude only":           fx.ClaudeOnly(),
		"multiple runtimes":     fx.MultiRuntime(),
		"nonempty preferences":  fx.WithPreferences(fx.MultiRuntime()),
		"revision changed":      fx.WithRevision(fx.MultiRuntime(), 7),
		"disabled runtime":      fx.WithRuntimeDisabled(fx.MultiRuntime(), fx.ClaudeRuntimeID),
		"no applicable effort":  fx.WithoutDeclarations(fx.MultiRuntime(), fx.CodexRuntimeID),
		"all declarations gone": fx.WithoutDeclarations(fx.WithoutDeclarations(fx.MultiRuntime(), fx.CodexRuntimeID), fx.ClaudeRuntimeID),
	}
}

func TestV3FixturesAreCanonicalAndRoundTrip(t *testing.T) {
	for name, cfg := range syntheticV3Fixtures() {
		wire := fx.Encode(t, cfg)
		versioned, err := runtimeprofile.DecodeVersioned(wire)
		if err != nil || versioned.FormatVersion != runtimeprofile.FormatVersionV3 || versioned.Configuration != nil {
			t.Fatalf("%s: decode %v", name, err)
		}
		consumer, err := runtimeprofile.ConsumeVersioned(versioned)
		if err != nil {
			t.Fatalf("%s: consume %v", name, err)
		}
		if again := fx.Encode(t, consumer.Configuration()); string(again) != string(wire) {
			t.Fatalf("%s: consumer changed canonical bytes", name)
		}
		if consumer.Revision() != cfg.Revision {
			t.Fatalf("%s: revision %d", name, consumer.Revision())
		}
	}
	// Builders are deterministic and never share mutable state.
	a, b := fx.MultiRuntime(), fx.MultiRuntime()
	a.Runtimes[0].EffortDeclarations[0].Sources[0].Path = "changed/config.toml"
	a.ModelProfiles[0].Capabilities[0] = "changed"
	if !reflect.DeepEqual(b, fx.MultiRuntime()) {
		t.Fatal("fixture builders share state")
	}
}

func TestV3ConsumerRecordStates(t *testing.T) {
	valid := fx.MultiRuntime()
	invalid := fx.WithAllowlisted(fx.CodexOnly(), fx.CodexRuntimeID, fx.UnknownProfileID)
	foreign := fx.WithAllowlisted(fx.MultiRuntime(), fx.ClaudeRuntimeID, fx.CodexHighProfileID)
	v2Format := fx.CodexOnly()
	v2Format.FormatVersion = runtimeprofile.FormatVersionV2
	zeroRevision := fx.WithRevision(fx.CodexOnly(), 0)
	duplicateRuntime := fx.MultiRuntime()
	duplicateRuntime.Runtimes = append(duplicateRuntime.Runtimes, fx.ClaudeOnly().Runtimes[0])
	for _, c := range []struct {
		name  string
		state runtimeprofile.RecordStateV3
		cfg   *runtimeprofile.ConfigurationV3
		want  error
	}{
		{"missing", runtimeprofile.RecordMissingV3, nil, runtimeprofile.ErrConfigurationMissingV3},
		{"retired", runtimeprofile.RecordRetiredV3, nil, runtimeprofile.ErrConfigurationRetiredV3},
		{"missing with record", runtimeprofile.RecordMissingV3, &valid, runtimeprofile.ErrInvalidConfiguration},
		{"retired with record", runtimeprofile.RecordRetiredV3, &valid, runtimeprofile.ErrInvalidConfiguration},
		{"active without record", runtimeprofile.RecordActiveV3, nil, runtimeprofile.ErrInvalidConfiguration},
		{"unknown state", 0, &valid, runtimeprofile.ErrInvalidConfiguration},
		{"unknown allowlisted profile", runtimeprofile.RecordActiveV3, &invalid, runtimeprofile.ErrInvalidConfiguration},
		{"foreign allowlisted profile", runtimeprofile.RecordActiveV3, &foreign, runtimeprofile.ErrInvalidConfiguration},
		{"duplicate runtime", runtimeprofile.RecordActiveV3, &duplicateRuntime, runtimeprofile.ErrInvalidConfiguration},
		{"format 2 in v3 type", runtimeprofile.RecordActiveV3, &v2Format, runtimeprofile.ErrInvalidConfiguration},
		{"zero revision", runtimeprofile.RecordActiveV3, &zeroRevision, runtimeprofile.ErrInvalidConfiguration},
	} {
		if _, err := runtimeprofile.NewConsumerV3(c.state, c.cfg); !errors.Is(err, c.want) {
			t.Fatalf("%s: got %v want %v", c.name, err, c.want)
		}
	}
	if _, err := runtimeprofile.NewConsumerV3(runtimeprofile.RecordActiveV3, &valid); err != nil {
		t.Fatal(err)
	}
	// The zero consumer answers nothing.
	if _, err := (runtimeprofile.ConsumerV3{}).EnabledRuntime(fx.CodexRuntimeID); !errors.Is(err, runtimeprofile.ErrInvalidConfiguration) {
		t.Fatalf("zero consumer: %v", err)
	}
}

// A historical record is refused by the v3 consumer, never upgraded.
func TestV3ConsumerRefusesHistoricalFormats(t *testing.T) {
	v2 := runtimeprofile.Configuration{FormatVersion: runtimeprofile.FormatVersionV2, Revision: 1,
		Runtimes:      []runtimeprofile.Runtime{{ID: fx.CodexRuntimeID, Adapter: fx.CodexRuntimeID, Enabled: true, AllowlistedProfileIDs: []string{fx.CodexHighProfileID}}},
		ModelProfiles: []runtimeprofile.ModelProfile{fx.Profile(fx.CodexHighProfileID, fx.CodexRuntimeID, "synthetic-codex-model", fx.ExplicitEffort("high"))},
	}
	wire, err := runtimeprofile.Encode(v2)
	if err != nil {
		t.Fatal(err)
	}
	versioned, err := runtimeprofile.DecodeVersioned(wire)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := runtimeprofile.ConsumeVersioned(versioned); !errors.Is(err, runtimeprofile.ErrUnsupportedConfigurationFormat) {
		t.Fatalf("v2: %v", err)
	}
	v3 := fx.CodexOnly()
	for name, v := range map[string]runtimeprofile.VersionedConfiguration{
		"v3 tag over v2 body":   {FormatVersion: runtimeprofile.FormatVersionV3, Configuration: &v2},
		"v3 tag with both":      {FormatVersion: runtimeprofile.FormatVersionV3, Configuration: &v2, V3: &v3},
		"v3 tag without record": {FormatVersion: runtimeprofile.FormatVersionV3},
		"future format":         {FormatVersion: runtimeprofile.FormatVersionV3 + 1, V3: &v3},
	} {
		if _, err := runtimeprofile.ConsumeVersioned(v); !errors.Is(err, runtimeprofile.ErrInvalidConfiguration) {
			t.Fatalf("%s: %v", name, err)
		}
	}
}

func TestV3ConsumerRuntimeAndProfileLookup(t *testing.T) {
	consumer := fx.Consumer(t, fx.WithRuntimeDisabled(fx.MultiRuntime(), fx.ClaudeRuntimeID))
	runtime, err := consumer.EnabledRuntime(fx.CodexRuntimeID)
	if err != nil || runtime.ID != fx.CodexRuntimeID || !runtime.Enabled || len(runtime.EffortDeclarations) != 1 || runtime.EffortDeclarations[0].DeclarationID != fx.CodexDeclarationID {
		t.Fatalf("enabled runtime: %+v %v", runtime, err)
	}
	// Returned values are independent of the consumer.
	runtime.EffortDeclarations[0].Sources[0].Path = "changed/config.toml"
	runtime.AllowlistedProfileIDs[0] = fx.CodexUnlistedProfileID
	if again, _ := consumer.EnabledRuntime(fx.CodexRuntimeID); again.EffortDeclarations[0].Sources[0].Path == "changed/config.toml" || contains(again.AllowlistedProfileIDs, fx.CodexUnlistedProfileID) {
		t.Fatal("consumer state escaped")
	}
	for _, c := range []struct {
		name, runtimeID, profileID string
		want                       error
	}{
		{"disabled runtime", fx.ClaudeRuntimeID, fx.ClaudeLowProfileID, runtimeprofile.ErrRuntimeDisabled},
		{"unknown runtime", fx.UnknownRuntimeID, fx.CodexHighProfileID, runtimeprofile.ErrRuntimeAbsent},
		{"no implicit runtime", "", fx.CodexHighProfileID, runtimeprofile.ErrRuntimeAbsent},
		{"unknown profile", fx.CodexRuntimeID, fx.UnknownProfileID, runtimeprofile.ErrProfileMissing},
		{"no implicit profile", fx.CodexRuntimeID, "", runtimeprofile.ErrProfileMissing},
		{"profile of another runtime", fx.CodexRuntimeID, fx.ClaudeLowProfileID, runtimeprofile.ErrProfileRuntimeMismatch},
		{"owned but not allowlisted", fx.CodexRuntimeID, fx.CodexUnlistedProfileID, runtimeprofile.ErrProfileNotAllowlisted},
	} {
		if _, err := consumer.AllowedProfile(c.runtimeID, c.profileID); !errors.Is(err, c.want) {
			t.Fatalf("%s: got %v want %v", c.name, err, c.want)
		}
		if _, err := consumer.ProfileIntent(c.runtimeID, c.profileID); !errors.Is(err, c.want) {
			t.Fatalf("%s intent: got %v want %v", c.name, err, c.want)
		}
	}
	// The same Profile is allowed once its Runtime is enabled: no alternative
	// Runtime or Profile was ever substituted above.
	enabled := fx.Consumer(t, fx.MultiRuntime())
	if p, err := enabled.AllowedProfile(fx.ClaudeRuntimeID, fx.ClaudeLowProfileID); err != nil || p.ID != fx.ClaudeLowProfileID || p.RuntimeID != fx.ClaudeRuntimeID {
		t.Fatalf("enabled claude: %+v %v", p, err)
	}
}

func TestV3ConsumerProfileIntent(t *testing.T) {
	cfg := fx.WithRevision(fx.MultiRuntime(), 9)
	consumer := fx.Consumer(t, cfg)
	for _, c := range []struct {
		runtimeID, profileID, model string
		effort                      runtimeprofile.Effort
	}{
		{fx.CodexRuntimeID, fx.CodexHighProfileID, "synthetic-codex-model", fx.ExplicitEffort("high")},
		{fx.CodexRuntimeID, fx.CodexDefaultProfileID, "synthetic-codex-model", fx.RuntimeDefaultEffort()},
		{fx.ClaudeRuntimeID, fx.ClaudeLowProfileID, "synthetic-claude-model", fx.ExplicitEffort("low")},
	} {
		intent, err := consumer.ProfileIntent(c.runtimeID, c.profileID)
		if err != nil {
			t.Fatal(c.profileID, err)
		}
		profile, _ := consumer.AllowedProfile(c.runtimeID, c.profileID)
		profileDigest, err := runtimeprofile.ProfileDigest(profile)
		if err != nil {
			t.Fatal(err)
		}
		want := runtimeprofile.ProfileIntentV3{RuntimeID: c.runtimeID, Adapter: c.runtimeID, ProfileID: c.profileID, Model: c.model, Effort: c.effort, ProfileDigest: profileDigest, ConfigurationRevision: 9, ConfigurationDigest: fx.Digest(t, cfg)}
		if intent != want {
			t.Fatalf("intent %+v want %+v", intent, want)
		}
	}
	// Intent carries neither the Runtime's declaration nor any proof.
	intentType := reflect.TypeOf(runtimeprofile.ProfileIntentV3{})
	for i := 0; i < intentType.NumField(); i++ {
		name := intentType.Field(i).Name
		if strings.Contains(name, "Declaration") || strings.Contains(name, "Proof") || strings.Contains(name, "Capability") || strings.Contains(name, "Observation") {
			t.Fatalf("intent field %s implies proof or declaration ownership", name)
		}
	}
}

func TestV3ConsumerEffortDeclarationOwnership(t *testing.T) {
	consumer := fx.Consumer(t, fx.MultiRuntime())
	for _, a := range []runtimeprofile.EffortApplicability{fx.CodexApplicability(), fx.ClaudeApplicability()} {
		d, err := consumer.EffortDeclaration(a)
		if err != nil || d.RuntimeVersion != a.RuntimeVersion || d.ExecutableDigest != a.ExecutableDigest || d.ProofReference() != d.DeclarationID {
			t.Fatalf("%s: %+v %v", a.RuntimeID, d, err)
		}
	}
	// A declaration is owned by its enclosing Runtime only.
	crossed := fx.ClaudeApplicability()
	crossed.RuntimeID = fx.CodexRuntimeID
	otherVersion := fx.CodexApplicability()
	otherVersion.RuntimeVersion = "synthetic-codex-2"
	otherExecutable := fx.CodexApplicability()
	otherExecutable.ExecutableDigest = strings.Repeat("3", 64)
	interactive := fx.CodexApplicability()
	interactive.InvocationMode = "interactive"
	for name, a := range map[string]runtimeprofile.EffortApplicability{"crossed runtime": crossed, "other version": otherVersion, "other executable": otherExecutable, "other mode": interactive} {
		if _, err := consumer.EffortDeclaration(a); !errors.Is(err, runtimeprofile.ErrEffortDeclarationUnproven) {
			t.Fatalf("%s: %v", name, err)
		}
	}
	// No applicable declaration is not evidence of support.
	none := fx.Consumer(t, fx.WithoutDeclarations(fx.MultiRuntime(), fx.CodexRuntimeID))
	if _, err := none.EffortDeclaration(fx.CodexApplicability()); !errors.Is(err, runtimeprofile.ErrEffortDeclarationUnproven) {
		t.Fatalf("missing declaration: %v", err)
	}
	if _, err := none.ProfileIntent(fx.CodexRuntimeID, fx.CodexHighProfileID); err != nil {
		t.Fatalf("intent does not depend on a declaration: %v", err)
	}
	disabled := fx.Consumer(t, fx.WithRuntimeDisabled(fx.MultiRuntime(), fx.CodexRuntimeID))
	if _, err := disabled.EffortDeclaration(fx.CodexApplicability()); !errors.Is(err, runtimeprofile.ErrRuntimeDisabled) {
		t.Fatalf("disabled: %v", err)
	}
	if _, err := consumer.EffortDeclaration(runtimeprofile.EffortApplicability{RuntimeID: fx.UnknownRuntimeID}); !errors.Is(err, runtimeprofile.ErrRuntimeAbsent) {
		t.Fatalf("unknown: %v", err)
	}
}

// The consumer reads and returns format 3 only; declarations are never dropped.
func TestV3ConsumerNoDowngrade(t *testing.T) {
	cfg := fx.WithPreferences(fx.MultiRuntime())
	out := fx.Consumer(t, cfg).Configuration()
	if out.FormatVersion != runtimeprofile.FormatVersionV3 || !reflect.DeepEqual(fx.Encode(t, out), fx.Encode(t, cfg)) {
		t.Fatal("consumer did not preserve the canonical v3 record")
	}
	declarations := 0
	for _, r := range out.Runtimes {
		declarations += len(r.EffortDeclarations)
	}
	if declarations != 2 {
		t.Fatalf("declarations retained: %d", declarations)
	}
	consumerType := reflect.TypeOf(runtimeprofile.ConsumerV3{})
	for i := 0; i < consumerType.NumMethod(); i++ {
		m := consumerType.Method(i)
		for j := 0; j < m.Type.NumOut(); j++ {
			if m.Type.Out(j) == reflect.TypeOf(runtimeprofile.Configuration{}) || m.Type.Out(j) == reflect.TypeOf(runtimeprofile.Runtime{}) {
				t.Fatalf("%s exposes a format-1/2 projection", m.Name)
			}
		}
	}
}

func TestV3ConsumerDigestIdentity(t *testing.T) {
	for name, cfg := range syntheticV3Fixtures() {
		sum := sha256.Sum256(fx.Encode(t, cfg))
		want := hex.EncodeToString(sum[:])
		if got := fx.Consumer(t, cfg).Digest(); got != want || fx.Digest(t, cfg) != want {
			t.Fatalf("%s: consumer digest %s want SHA256(EncodeV3) %s", name, got, want)
		}
	}
	base := fx.MultiRuntime()
	drifted := fx.MultiRuntime()
	drifted.Runtimes[0].EffortDeclarations[0].Sources[0].Path = ".codex/other/config.toml"
	revised := fx.WithRevision(fx.MultiRuntime(), 2)
	preferred := fx.WithPreferences(fx.MultiRuntime())
	baseDigest := fx.Consumer(t, base).Digest()
	for name, cfg := range map[string]runtimeprofile.ConfigurationV3{"declaration drift": drifted, "revision": revised, "preferences": preferred, "declaration removed": fx.WithoutDeclarations(base, fx.ClaudeRuntimeID)} {
		if fx.Consumer(t, cfg).Digest() == baseDigest {
			t.Fatalf("%s did not change DigestV3", name)
		}
	}
	// ProfileDigest stays independent of Runtime-owned declarations and revision.
	for _, cfg := range []runtimeprofile.ConfigurationV3{drifted, revised, fx.WithoutDeclarations(base, fx.CodexRuntimeID)} {
		a, _ := fx.Consumer(t, base).ProfileIntent(fx.CodexRuntimeID, fx.CodexHighProfileID)
		b, _ := fx.Consumer(t, cfg).ProfileIntent(fx.CodexRuntimeID, fx.CodexHighProfileID)
		if a.ProfileDigest == "" || a.ProfileDigest != b.ProfileDigest || a.ConfigurationDigest == b.ConfigurationDigest {
			t.Fatalf("profile digest coupling: %+v %+v", a, b)
		}
	}
}

func contains(values []string, value string) bool {
	for _, v := range values {
		if v == value {
			return true
		}
	}
	return false
}
