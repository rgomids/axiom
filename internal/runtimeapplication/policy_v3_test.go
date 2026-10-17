package runtimeapplication

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/project"
	"github.com/rgomids/axiom/internal/runtimeprofile"
	fx "github.com/rgomids/axiom/internal/runtimeprofile/runtimeprofiletest"
)

func portableProfile(id, runtimeID, model string) project.ModelProfile {
	return project.ModelProfile{Key: id, RuntimeRef: project.Configured(runtimeID), Model: project.Configured(model)}
}

// v3Project allows both synthetic Runtimes and every fixture Profile with the
// exact Runtime/model relationship unless profiles overrides it.
func v3Project(t *testing.T, preferences []project.RuntimePreference, profiles ...project.ModelProfile) project.Project {
	t.Helper()
	if profiles == nil {
		profiles = []project.ModelProfile{
			portableProfile(fx.CodexHighProfileID, fx.CodexRuntimeID, "synthetic-codex-model"),
			portableProfile(fx.CodexDefaultProfileID, fx.CodexRuntimeID, "synthetic-codex-model"),
			portableProfile(fx.CodexUnlistedProfileID, fx.CodexRuntimeID, "synthetic-codex-model"),
			portableProfile(fx.ClaudeLowProfileID, fx.ClaudeRuntimeID, "synthetic-claude-model"),
		}
	}
	state := project.State{SchemaVersion: 2, ID: projectID, Slug: "example", Name: "Example",
		Runtimes:      project.Configured([]project.Runtime{{ID: fx.CodexRuntimeID}, {ID: fx.ClaudeRuntimeID}}),
		ModelProfiles: project.Configured(profiles)}
	if preferences != nil {
		state.RuntimePreferences = project.Configured(preferences)
	}
	p, issues := project.New(state)
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	return p
}

func v3Request(profileID string) Request {
	return Request{Role: "implementation", Complexity: "high", Capabilities: []string{fx.SyntheticCapability}, ModelProfileID: profileID}
}

func TestPolicyV3ExactProfileIntentAndIdentity(t *testing.T) {
	cfg := fx.CodexOnly()
	consumer := fx.Consumer(t, cfg)
	for _, tc := range []struct {
		id     string
		effort runtimeprofile.Effort
	}{{fx.CodexHighProfileID, fx.ExplicitEffort("high")}, {fx.CodexDefaultProfileID, fx.RuntimeDefaultEffort()}} {
		candidate, blocker := EvaluatePolicyV3(v3Project(t, nil), consumer, v3Request(tc.id))
		if blocker != nil {
			t.Fatalf("%s refused: %+v", tc.id, blocker)
		}
		got := candidate.Profile
		var profile runtimeprofile.ModelProfile
		for _, p := range cfg.ModelProfiles {
			if p.ID == tc.id {
				profile = p
			}
		}
		profileDigest, err := runtimeprofile.ProfileDigest(profile)
		if err != nil {
			t.Fatal(err)
		}
		if got.RuntimeID != fx.CodexRuntimeID || got.ProfileID != tc.id || got.Model != "synthetic-codex-model" || got.Effort != tc.effort ||
			got.ConfigurationDigest != fx.Digest(t, cfg) || got.ConfigurationRevision != fx.InitialRevision || got.ProfileDigest != profileDigest {
			t.Fatalf("intent did not bind exact v3 identity: %+v", got)
		}
		if err := CheckPolicyV3(v3Project(t, nil), consumer, candidate); err != nil {
			t.Fatal("exact candidate not reproduced", err)
		}
	}
}

func TestPolicyV3NoImplicitRuntimeSelection(t *testing.T) {
	consumer := fx.Consumer(t, fx.MultiRuntime())
	p := v3Project(t, nil)
	request := v3Request("")
	_, blocker := EvaluatePolicyV3(p, consumer, request)
	if blocker == nil || blocker.Code != "ambiguous_match" || !reflect.DeepEqual(blocker.ProfileIDs, []string{fx.ClaudeLowProfileID, fx.CodexDefaultProfileID, fx.CodexHighProfileID}) {
		t.Fatalf("multiple valid Runtimes selected implicitly: %+v", blocker)
	}
	request.RuntimeID = fx.ClaudeRuntimeID
	first, blocker := EvaluatePolicyV3(p, consumer, request)
	second, _ := EvaluatePolicyV3(p, consumer, request)
	if blocker != nil || first.Profile.ProfileID != fx.ClaudeLowProfileID || !reflect.DeepEqual(first, second) {
		t.Fatalf("explicit Runtime narrowing not deterministic: %+v %+v", first, blocker)
	}
	preferred := []project.RuntimePreference{{Role: "implementation", Complexity: "high", ModelProfileRef: fx.CodexHighProfileID}}
	if c, b := EvaluatePolicyV3(v3Project(t, preferred), consumer, v3Request("")); b != nil || c.Profile.ProfileID != fx.CodexHighProfileID {
		t.Fatalf("portable preference not honored: %+v %+v", c, b)
	}
	unavailable := []project.RuntimePreference{{Role: "implementation", Complexity: "high", ModelProfileRef: fx.CodexUnlistedProfileID}}
	if _, b := EvaluatePolicyV3(v3Project(t, unavailable), consumer, v3Request("")); b == nil || b.Code != "no_allowed_match" {
		t.Fatalf("unavailable preferred Profile substituted: %+v", b)
	}
}

func TestPolicyV3FailsClosed(t *testing.T) {
	multi := fx.MultiRuntime()
	mismatched := []project.ModelProfile{portableProfile(fx.CodexHighProfileID, fx.ClaudeRuntimeID, "synthetic-codex-model")}
	otherModel := []project.ModelProfile{portableProfile(fx.CodexHighProfileID, fx.CodexRuntimeID, "other-model")}
	for _, tc := range []struct {
		name     string
		cfg      runtimeprofile.ConfigurationV3
		profiles []project.ModelProfile
		request  Request
		want     string
	}{
		{"disabled runtime", fx.WithRuntimeDisabled(fx.CodexOnly(), fx.CodexRuntimeID), nil, v3Request(fx.CodexHighProfileID), "no_allowed_match"},
		{"missing runtime", multi, nil, Request{Role: "implementation", Complexity: "high", Capabilities: []string{fx.SyntheticCapability}, RuntimeID: fx.UnknownRuntimeID}, "no_allowed_match"},
		{"missing profile", multi, nil, v3Request(fx.UnknownProfileID), "no_allowed_match"},
		{"profile runtime mismatch", multi, mismatched, v3Request(fx.CodexHighProfileID), "no_allowed_match"},
		{"profile model drift", multi, otherModel, v3Request(fx.CodexHighProfileID), "no_allowed_match"},
		{"not allowlisted", multi, nil, v3Request(fx.CodexUnlistedProfileID), "no_allowed_match"},
		{"unknown capability", multi, nil, Request{Role: "implementation", Complexity: "high", Capabilities: []string{"rust"}, ModelProfileID: fx.CodexHighProfileID}, "no_allowed_match"},
		{"invalid request", multi, nil, Request{Role: "Implementation", Complexity: "high", Capabilities: []string{fx.SyntheticCapability}}, "invalid_request"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, blocker := EvaluatePolicyV3(v3Project(t, nil, tc.profiles...), fx.Consumer(t, tc.cfg), tc.request)
			if blocker == nil || blocker.Code != tc.want {
				t.Fatalf("got %+v want %s", blocker, tc.want)
			}
		})
	}
	if _, blocker := EvaluatePolicyV3(v3Project(t, nil), runtimeprofile.ConsumerV3{}, v3Request(fx.CodexHighProfileID)); blocker == nil || blocker.Code != "invalid_configuration" {
		t.Fatalf("unconsumed configuration admitted: %+v", blocker)
	}
}

func TestPolicyV3DeclarationIsNotCapabilityProof(t *testing.T) {
	with := fx.CodexOnly()
	without := fx.WithoutDeclarations(fx.CodexOnly(), fx.CodexRuntimeID)
	a, ba := EvaluatePolicyV3(v3Project(t, nil), fx.Consumer(t, with), v3Request(fx.CodexHighProfileID))
	b, bb := EvaluatePolicyV3(v3Project(t, nil), fx.Consumer(t, without), v3Request(fx.CodexHighProfileID))
	if ba != nil || bb != nil {
		t.Fatal("declaration presence changed admissibility", ba, bb)
	}
	// Declarations participate in DigestV3 only; selection is otherwise equal.
	if a.Profile.ConfigurationDigest == b.Profile.ConfigurationDigest {
		t.Fatal("EffortDeclarations did not participate in DigestV3")
	}
	a.Profile.ConfigurationDigest, b.Profile.ConfigurationDigest = "", ""
	if !reflect.DeepEqual(a, b) {
		t.Fatalf("declaration changed selection: %+v %+v", a, b)
	}
	wire, _ := json.Marshal(a)
	for _, forbidden := range []string{"proof", "resolvedProfile", "declaration", "observation", "credential"} {
		if strings.Contains(strings.ToLower(string(wire)), strings.ToLower(forbidden)) {
			t.Fatalf("candidate claims %s: %s", forbidden, wire)
		}
	}
	// Effort is never a capability: a Profile cannot declare one and a request
	// for one is not satisfied by the Runtime's EffortDeclaration.
	declared := fx.CodexOnly()
	declared.ModelProfiles[0].Capabilities = append(declared.ModelProfiles[0].Capabilities, "reasoning-effort-high")
	if _, err := runtimeprofile.NewConsumerV3(runtimeprofile.RecordActiveV3, &declared); !errors.Is(err, runtimeprofile.ErrInvalidConfiguration) {
		t.Fatal("effort capability declaration accepted", err)
	}
	request := v3Request(fx.CodexHighProfileID)
	request.Capabilities = []string{"reasoning-effort-high"}
	if _, blocker := EvaluatePolicyV3(v3Project(t, nil), fx.Consumer(t, with), request); blocker == nil || blocker.Code != "no_allowed_match" {
		t.Fatalf("effort capability inferred from declaration: %+v", blocker)
	}
}

func TestPolicyV3IdentityDriftIsStale(t *testing.T) {
	p := v3Project(t, nil)
	candidate, blocker := EvaluatePolicyV3(p, fx.Consumer(t, fx.CodexOnly()), v3Request(fx.CodexHighProfileID))
	if blocker != nil {
		t.Fatal(blocker)
	}
	changedEffort := fx.CodexOnly()
	changedEffort.ModelProfiles[0].Effort = fx.ExplicitEffort("low")
	for name, cfg := range map[string]runtimeprofile.ConfigurationV3{
		"revision":    fx.WithRevision(fx.CodexOnly(), 2),
		"declaration": fx.WithoutDeclarations(fx.CodexOnly(), fx.CodexRuntimeID),
		"effort":      changedEffort,
		"disabled":    fx.WithRuntimeDisabled(fx.CodexOnly(), fx.CodexRuntimeID),
	} {
		if err := CheckPolicyV3(p, fx.Consumer(t, cfg), candidate); !errors.Is(err, ErrStale) {
			t.Fatalf("%s drift accepted: %v", name, err)
		}
	}
}

func TestConsumerFromRecordV3States(t *testing.T) {
	valid := fx.CodexOnly()
	invalid := fx.CodexOnly()
	invalid.ModelProfiles[0].Effort = runtimeprofile.Effort{}
	for _, tc := range []struct {
		state string
		cfg   *runtimeprofile.ConfigurationV3
		want  string
	}{
		{"active-v3", &valid, ""},
		{"fresh", nil, "configuration_missing"},
		{"retired", nil, "configuration_retired"},
		{"active-v3", &invalid, "invalid_configuration"},
		{"active-v3", nil, "invalid_configuration"},
		{"active-v1", nil, "unsupported_configuration_format"},
		{"active-v2", nil, "unsupported_configuration_format"},
		{"pending", nil, "configuration_recovery_required"},
		{"unsafe", nil, "configuration_unsafe"},
		{"", nil, "configuration_unsafe"},
	} {
		consumer, code := ConsumerFromRecordV3(tc.state, tc.cfg)
		if code != tc.want || (code == "") != (consumer.Digest() != "") {
			t.Fatalf("%s: got %q want %q", tc.state, code, tc.want)
		}
	}
}

func TestReadinessCandidatesV3(t *testing.T) {
	cfg := fx.WithRuntimeDisabled(fx.MultiRuntime(), fx.ClaudeRuntimeID)
	before := fx.Encode(t, cfg)
	candidates, code := ReadinessCandidatesV3("active-v3", &cfg)
	if code != "" || len(candidates) != 1 || candidates[0].RuntimeID != fx.CodexRuntimeID || len(candidates[0].Profiles) != 2 {
		t.Fatalf("disabled Runtime or unlisted Profile disclosed: %+v %s", candidates, code)
	}
	for _, intent := range candidates[0].Profiles {
		if intent.ProfileID == fx.CodexUnlistedProfileID || intent.Model != "synthetic-codex-model" || !runtimeprofile.ValidEffort(intent.Effort) || intent.ConfigurationDigest != fx.Digest(t, cfg) {
			t.Fatalf("candidate lacks exact intent: %+v", intent)
		}
	}
	if string(fx.Encode(t, cfg)) != string(before) {
		t.Fatal("projection mutated its input")
	}
	multi := fx.MultiRuntime()
	all, code := ReadinessCandidatesV3("active-v3", &multi)
	if code != "" || len(all) != 2 {
		t.Fatalf("enabled Runtimes not all disclosed: %+v", all)
	}
	invalid := fx.CodexOnly()
	invalid.Runtimes[0].AllowlistedProfileIDs = append(invalid.Runtimes[0].AllowlistedProfileIDs, fx.UnknownProfileID)
	for state, want := range map[string]string{"fresh": "configuration_missing", "retired": "configuration_retired", "active-v2": "unsupported_configuration_format"} {
		if got, code := ReadinessCandidatesV3(state, nil); got != nil || code != want {
			t.Fatalf("%s: got %q want %q", state, code, want)
		}
	}
	if got, code := ReadinessCandidatesV3("active-v3", &invalid); got != nil || code != "invalid_configuration" {
		t.Fatalf("invalid v3 disclosed: %q", code)
	}
}
