package runtimeapplication

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/runtimeprofile"
)

// govsmRequest reproduces the v0.22.0 incident: every field except
// --capabilities, which ModelProfile declaration requires.
func govsmRequest(runtime string) AuthoringRequest {
	return AuthoringRequest{Kind: "binding", Operation: "create", RuntimeID: runtime, ProfileID: runtime + "-default", Model: "gpt-6-luna", Effort: runtimeprofile.Effort{Mode: "explicit", Value: "low"}}
}

func diagnosticFor(r AuthoringReport, field string) (AuthoringDiagnostic, bool) {
	for _, d := range r.Diagnostics {
		if d.Field == field {
			return d, true
		}
	}
	return AuthoringDiagnostic{}, false
}

func TestAuthoringMissingCapabilitiesNamesFieldWithoutAuthentication(t *testing.T) {
	for _, op := range []string{"create", "validate-candidate"} {
		a, s := authoringSetup()
		r := govsmRequest("codex")
		r.Operation = op
		got, err := a.Run(context.Background(), r)
		if err == nil || got.Category != "candidate_invalid" || s.writes != 0 {
			t.Fatalf("%s: %+v %v", op, got, err)
		}
		d, ok := diagnosticFor(got, "--capabilities")
		if !ok || d.Code != "field_required" || !strings.Contains(d.Next, "--capabilities <comma-list>") {
			t.Fatalf("%s: missing specific diagnostic %+v", op, got.Diagnostics)
		}
		if len(got.Diagnostics) != 1 {
			t.Fatalf("%s: unrelated diagnostics %+v", op, got.Diagnostics)
		}
		// Declaration refusals never send the operator to credentials/env.
		wire, _ := json.Marshal(got)
		for _, forbidden := range []string{"OPENAI_API_KEY", "ANTHROPIC", "login", "unset"} {
			if strings.Contains(string(wire), forbidden) {
				t.Fatalf("%s: auth remediation for a schema refusal: %s", op, wire)
			}
		}
		if got.Authentication != "unknown" || got.ModelSupport != "unconfirmed" || got.EffortSupport != "unconfirmed" || !strings.Contains(got.Next, "authentication was not evaluated") {
			t.Fatalf("%s: support/auth claims %+v", op, got)
		}
	}
}

func TestAuthoringCapabilitiesSuppliedProducesPreviewOfUncertifiedModel(t *testing.T) {
	a, s := authoringSetup()
	r := govsmRequest("codex")
	r.Capabilities = []string{"axiom-skills"}
	got, err := a.Run(context.Background(), r)
	if err != nil || got.Category != "preview" || got.PreviewDigest == "" || s.writes != 0 || len(got.Diagnostics) != 0 {
		t.Fatalf("%+v %v", got, err)
	}
	// An unknown model is saved as intent, never as confirmed support.
	if got.ModelSupport != "unconfirmed" || got.EffortSupport != "unconfirmed" || got.Authentication != "unknown" {
		t.Fatal("intent claimed support", got)
	}
	r.Operation = "validate-candidate"
	valid, err := a.Run(context.Background(), r)
	if err != nil || valid.Category != "candidate_valid" || valid.ModelSupport != "unconfirmed" || !strings.Contains(valid.Next, "not vendor certification") {
		t.Fatalf("validate-candidate overclaims: %+v %v", valid, err)
	}
	// Complexities stay optional.
	applied := applyReviewed(t, a, r.withOperation("create"))
	if applied.Revision != 1 || len(applied.Profiles) != 1 || applied.Profiles[0].Model != "gpt-6-luna" {
		t.Fatal(applied)
	}
}

func (r AuthoringRequest) withOperation(op string) AuthoringRequest { r.Operation = op; return r }

func TestAuthoringDiagnosticsDistinguishRefusalKinds(t *testing.T) {
	valid := govsmRequest("codex")
	valid.Capabilities = []string{"axiom-skills"}
	mutate := func(f func(*AuthoringRequest)) AuthoringRequest { r := valid; f(&r); return r }
	cases := []struct {
		name, category, field, code string
		req                         AuthoringRequest
	}{
		{"model required", "candidate_invalid", "--model", "field_required", mutate(func(r *AuthoringRequest) { r.Model = "" })},
		{"profile required", "candidate_invalid", "--profile", "field_required", mutate(func(r *AuthoringRequest) { r.ProfileID = "" })},
		{"effort mode required", "candidate_invalid", "--effort-mode", "field_required", mutate(func(r *AuthoringRequest) { r.Effort = runtimeprofile.Effort{} })},
		{"effort value required", "invalid_input", "--effort", "field_required", mutate(func(r *AuthoringRequest) { r.Effort.Value = "" })},
		{"runtime-default needs default", "invalid_input", "--effort", "field_invalid", mutate(func(r *AuthoringRequest) { r.Effort = runtimeprofile.Effort{Mode: "runtime-default", Value: "low"} })},
		{"capability format", "invalid_input", "--capabilities", "field_invalid", mutate(func(r *AuthoringRequest) { r.Capabilities = []string{"Axiom Skills"} })},
		{"capability duplicate", "candidate_invalid", "--capabilities", "field_invalid", mutate(func(r *AuthoringRequest) { r.Capabilities = []string{"go", "go"} })},
		{"capability reserved", "candidate_invalid", "--capabilities", "field_invalid", mutate(func(r *AuthoringRequest) { r.Capabilities = []string{"reasoning-effort-high"} })},
		{"complexity format", "invalid_input", "--complexities", "field_invalid", mutate(func(r *AuthoringRequest) { r.Complexities = []string{"a b"} })},
		{"runtime required", "runtime_unsupported", "--runtime", "field_required", mutate(func(r *AuthoringRequest) { r.RuntimeID = "" })},
		{"runtime unsupported", "runtime_unsupported", "--runtime", "field_invalid", mutate(func(r *AuthoringRequest) { r.RuntimeID = "gemini" })},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			a, s := authoringSetup()
			got, err := a.Run(context.Background(), c.req)
			d, ok := diagnosticFor(got, c.field)
			if err == nil || got.Category != c.category || !ok || d.Code != c.code || d.Rule == "" || d.Next == "" || s.writes != 0 {
				t.Fatalf("%+v %v", got, err)
			}
		})
	}
}

func TestAuthoringBindingMissingAndAbsentRuntimeAreNotModelEvidence(t *testing.T) {
	a, _ := authoringSetup()
	profile := AuthoringRequest{Kind: "profile", Operation: "create", RuntimeID: "claude", ProfileID: "claude-default", Model: "operator-model", Effort: runtimeprofile.Effort{Mode: "runtime-default", Value: "default"}, Capabilities: []string{"axiom-skills"}}
	got, err := a.Run(context.Background(), profile)
	d, ok := diagnosticFor(got, "--runtime")
	if err == nil || got.Category != "binding_missing" || !ok || d.Code != "binding_missing" || !strings.Contains(d.Next, "axiom runtime binding create --runtime claude") {
		t.Fatalf("%+v %v", got, err)
	}
	if got.ModelSupport != "unconfirmed" || len(got.Profiles) != 0 {
		t.Fatal("binding_missing presented as model evidence", got)
	}
	a.Installed = func(string) bool { return false }
	r := govsmRequest("claude")
	r.Capabilities = []string{"axiom-skills"}
	got, err = a.Run(context.Background(), r)
	d, ok = diagnosticFor(got, "--runtime")
	if err == nil || got.Category != "runtime_absent" || !ok || d.Code != "runtime_absent" || !strings.Contains(d.Rule, "not authentication") {
		t.Fatalf("%+v %v", got, err)
	}
}

func TestAuthoringConflictDiagnostics(t *testing.T) {
	a, _ := authoringSetup()
	base := govsmRequest("codex")
	base.Capabilities = []string{"axiom-skills"}
	applyReviewed(t, a, base)
	for name, c := range map[string]struct {
		req             AuthoringRequest
		category, field string
	}{
		"binding exists":      {base, "binding_exists", "--runtime"},
		"profile exists":      {base.withKind("profile"), "profile_exists", "--profile"},
		"profile missing":     {AuthoringRequest{Kind: "profile", Operation: "edit", ProfileID: "absent", Model: "m"}, "profile_missing", "--profile"},
		"binding edit fields": {AuthoringRequest{Kind: "binding", Operation: "edit", RuntimeID: "codex", Model: "m"}, "invalid_binding_candidate", "--model"},
	} {
		t.Run(name, func(t *testing.T) {
			got, err := a.Run(context.Background(), c.req)
			d, ok := diagnosticFor(got, c.field)
			if err == nil || got.Category != c.category || !ok || d.Code != "configuration_conflict" && d.Code != "profile_missing" || d.Next == "" {
				t.Fatalf("%+v %v", got, err)
			}
		})
	}
}

func (r AuthoringRequest) withKind(k string) AuthoringRequest { r.Kind = k; return r }

// The explanation layer never relaxes or contradicts runtimeprofile rules.
func TestProfileViolationsAgreeWithValidation(t *testing.T) {
	good := runtimeprofile.ModelProfile{ID: "p", RuntimeID: "codex", Model: "m", Effort: runtimeprofile.Effort{Mode: "explicit", Value: "low"}, Capabilities: []string{"go"}}
	variants := []func(*runtimeprofile.ModelProfile){
		func(p *runtimeprofile.ModelProfile) {},
		func(p *runtimeprofile.ModelProfile) { p.Capabilities = nil },
		func(p *runtimeprofile.ModelProfile) { p.Capabilities = []string{"go", "go"} },
		func(p *runtimeprofile.ModelProfile) { p.Capabilities = []string{"reasoning-effort-x"} },
		func(p *runtimeprofile.ModelProfile) { p.Complexities = []string{"high", "high"} },
		func(p *runtimeprofile.ModelProfile) { p.Model = "" },
		func(p *runtimeprofile.ModelProfile) { p.ID = "" },
		func(p *runtimeprofile.ModelProfile) { p.Effort = runtimeprofile.Effort{} },
		func(p *runtimeprofile.ModelProfile) { p.Complexities = []string{"high"} },
	}
	for i, v := range variants {
		p := good
		v(&p)
		cfg := runtimeprofile.ConfigurationV4{FormatVersion: runtimeprofile.FormatVersionV4, Revision: 1, Runtimes: []runtimeprofile.RuntimeV4{{ID: "codex", Adapter: "codex", Enabled: true, AllowlistedProfileIDs: []string{"p"}}}, ModelProfiles: []runtimeprofile.ModelProfile{p}}
		if p.ID == "" {
			cfg.Runtimes[0].AllowlistedProfileIDs = []string{""}
		}
		invalid := runtimeprofile.ValidateV4(cfg) != nil
		if invalid != (len(runtimeprofile.ProfileViolations(p)) != 0) {
			t.Fatalf("variant %d: validate=%v violations=%v", i, invalid, runtimeprofile.ProfileViolations(p))
		}
	}
}

// CR-356-02: --enabled on a Profile operation names the field instead of the
// generic fallback, for every operation the parser accepts it on.
func TestAuthoringProfileEnabledNamesField(t *testing.T) {
	enabled := true
	for _, op := range []string{"create", "edit", "validate-candidate"} {
		a, s := authoringSetup()
		base := govsmRequest("codex")
		base.Capabilities = []string{"read"}
		applyReviewed(t, a, base)
		writes := s.writes
		r := AuthoringRequest{Kind: "profile", Operation: op, RuntimeID: "codex", ProfileID: "p", Model: "m", Effort: runtimeprofile.Effort{Mode: "explicit", Value: "low"}, Capabilities: []string{"read"}, Enabled: &enabled}
		if op == "edit" {
			r.ProfileID = "codex-default"
		}
		got, err := a.Run(context.Background(), r)
		d, ok := diagnosticFor(got, "--enabled")
		if err == nil || got.Category != "invalid_input" || !ok || d.Code != "field_invalid" || len(got.Diagnostics) != 1 || !strings.Contains(d.Rule, "Runtime binding eligibility") || !strings.Contains(d.Next, "Remove --enabled") || s.writes != writes {
			t.Fatalf("%s: %+v %v", op, got, err)
		}
		if got.Authentication != "unknown" || got.ModelSupport != "unconfirmed" || got.EffortSupport != "unconfirmed" || !strings.HasPrefix(got.Next, d.Next) {
			t.Fatalf("%s: semantics changed %+v", op, got)
		}
	}
	// Binding operations keep accepting --enabled.
	a, _ := authoringSetup()
	r := govsmRequest("codex")
	r.Capabilities = []string{"read"}
	r.Enabled = &enabled
	if got, err := a.Run(context.Background(), r); err != nil || got.Category != "preview" {
		t.Fatalf("binding --enabled regressed: %+v %v", got, err)
	}
}
