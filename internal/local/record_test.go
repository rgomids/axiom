package local_test

import (
	"bytes"
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/local"
	"github.com/rgomids/axiom/internal/projectapp"
)

func fixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile("testdata/" + name + ".json")
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func rejected(t *testing.T, b []byte) []local.Issue {
	t.Helper()
	r, issues := local.DecodeRecord(b)
	if len(issues) == 0 || !reflect.DeepEqual(r, local.Record{}) {
		t.Fatal("invalid input returned reusable record")
	}
	if out, errs := local.EncodeRecord(r); out != nil || len(errs) == 0 {
		t.Fatal("invalid record encoded")
	}
	_, again := local.DecodeRecord(b)
	if !reflect.DeepEqual(issues, again) {
		t.Fatal("unstable diagnostics")
	}
	return issues
}

func TestVersionMatrix(t *testing.T) {
	b := fixture(t, "minimal")
	for _, v := range []string{"null", "true", "false", `"1"`, "1.5", "1.0", "1e0", "0", "-1", "2", "999999999999999999999999"} {
		t.Run(v, func(t *testing.T) {
			issues := rejected(t, bytes.Replace(b, []byte(`"formatVersion": 1`), []byte(`"formatVersion": `+v), 1))
			want := "invalid_local_state"
			if v == "0" || v == "-1" || v == "2" || strings.HasPrefix(v, "999") {
				want = "unsupported_local_format"
			}
			if issues[0].Category() != want {
				t.Fatalf("category %s, want %s", issues[0].Category(), want)
			}
		})
	}
	rejected(t, bytes.Replace(b, []byte(`"formatVersion": 1,`), nil, 1))
	rejected(t, bytes.Replace(b, []byte("formatVersion"), []byte("schemaVersion"), 1))
}

func TestRecordRoundTrips(t *testing.T) {
	for _, name := range []string{"minimal", "configured"} {
		t.Run(name, func(t *testing.T) {
			r, issues := local.DecodeRecord(fixture(t, name))
			if len(issues) > 0 {
				t.Fatal(issues)
			}
			out, issues := local.EncodeRecord(r)
			if len(issues) > 0 {
				t.Fatal(issues)
			}
			second, issues := local.DecodeRecord(out)
			if len(issues) > 0 {
				t.Fatal(issues)
			}
			if !reflect.DeepEqual(r.State(), second.State()) {
				t.Fatal("round trip changed metadata")
			}
			out2, _ := local.EncodeRecord(second)
			if !bytes.Equal(out, out2) {
				t.Fatal("unstable encode")
			}
			state := r.State()
			if len(state.Credentials) > 0 {
				state.Credentials[0].ItemReference = "changed"
				if reflect.DeepEqual(state, r.State()) {
					t.Fatal("aliased record")
				}
			}
		})
	}
}

func TestStrictStructures(t *testing.T) {
	b := fixture(t, "configured")
	for _, mutation := range []struct{ old, new string }{
		{`"formatVersion": 1`, `"formatVersion": 1, "formatVersion": 1`},
		{`"formatVersion": 1`, `"formatVersion": 1, "format\u0056ersion": 1`},
		{`"formatVersion": 1`, `"FormatVersion": 1`},
		{`"formatVersion": 1`, `"formatVersion": 1, "unknown": "SYNTHETIC_REJECTED"`},
		{`"projectId": "123e4567-e89b-42d3-a456-426614174000"`, `"projectId": "bad"`},
		{`"observedSlug": "demo"`, `"observedSlug": "../demo"`},
		{`"sourceLocation": "/synthetic/projects/demo"`, `"sourceLocation": "relative"`},
		{`"localRevision": "revision-1"`, `"localRevision": 1`},
		{`"referenceKey": "auth"`, `"referenceKey": "auth", "referenceKey": "other"`},
		{`"sourceKind": "environment"`, `"sourceKind": null`},
		{`"itemReference": "EXAMPLE_REF"`, `"itemReference": {"value":"SYNTHETIC_REJECTED"}`},
		{`"availability": "unverified"`, `"availability": "ready"`},
		{`"basis": "not_checked"`, `"basis": false`},
		{`"at": "2026-09-16T12:00:00Z"`, `"at": "yesterday"`},
		{`"observedAt": "2026-09-16T12:00:00Z"`, `"observedAt": "yesterday"`},
		{`"name": "context/readme.md"`, `"name": "../body"`},
	} {
		t.Run(mutation.new, func(t *testing.T) {
			if !bytes.Contains(b, []byte(mutation.old)) {
				t.Fatal("mutation not applied")
			}
			rejected(t, bytes.ReplaceAll(b, []byte(mutation.old), []byte(mutation.new)))
		})
	}
	for _, bad := range [][]byte{nil, {}, []byte("null"), []byte("[]"), []byte("{"), append(append([]byte{}, b...), []byte(" {}")...), bytes.Replace(b, []byte("demo"), []byte{0xff}, 1)} {
		rejected(t, bad)
	}
}

func TestMissingDiffersFromInvalid(t *testing.T) {
	r, revision, issues := local.DecodeObservedRecord(nil, false)
	if len(issues) > 0 || revision != projectapp.MissingLocalRevision() || !reflect.DeepEqual(r, local.Record{}) {
		t.Fatal("missing classification")
	}
	for _, data := range [][]byte{nil, {}, []byte("{}"), []byte(`{"schemaVersion":1}`)} {
		r, revision, issues = local.DecodeObservedRecord(data, true)
		if len(issues) == 0 || revision == projectapp.MissingLocalRevision() || !reflect.DeepEqual(r, local.Record{}) {
			t.Fatal("corrupt record became absent or reusable")
		}
	}
	data := fixture(t, "configured")
	_, revision, issues = local.DecodeObservedRecord(data, true)
	if len(issues) > 0 || revision != projectapp.ObserveLocalRevision(data) {
		t.Fatal("lost exact observed revision")
	}
	_, _, issues = local.DecodeObservedRecord(data, false)
	if len(issues) == 0 {
		t.Fatal("contradictory absence")
	}
}

func TestSecurityClosedFieldsAndReferenceValues(t *testing.T) {
	b := fixture(t, "configured")
	var obj map[string]any
	if err := json.Unmarshal(b, &obj); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"secret", "token", "password", "apiKey", "body", "content", "schemaVersion", "project", "providers", "modelProfiles", "businessContext", "policies", "credentialReferences"} {
		obj[key] = "SYNTHETIC_REJECTED"
		bad, _ := json.Marshal(obj)
		issues := rejected(t, bad)
		if strings.Contains(fmtIssues(issues), "SYNTHETIC_REJECTED") || strings.Contains(fmtIssues(issues), key) {
			t.Fatal("diagnostic leaked input")
		}
		delete(obj, key)
	}
	for _, val := range []string{"TOKEN=SYNTHETIC_REJECTED", "password:SYNTHETIC_REJECTED", "https://user:SYNTHETIC_REJECTED@example.invalid", "https://example.invalid?token=SYNTHETIC_REJECTED", "%2574oken%253DSYNTHETIC_REJECTED", "${EXAMPLE_REF}", `{"secret":"SYNTHETIC_REJECTED"}`, "line\nbreak"} {
		value, _ := json.Marshal(val)
		issues := rejected(t, bytes.Replace(b, []byte(`"EXAMPLE_REF"`), value, 1))
		if strings.Contains(fmtIssues(issues), "SYNTHETIC_REJECTED") {
			t.Fatal("leaked value")
		}
	}
}
func fmtIssues(issues []local.Issue) string { b, _ := json.Marshal(issues); return string(b) }

func TestEncodeValidatesProposals(t *testing.T) {
	r, issues := local.DecodeRecord(fixture(t, "configured"))
	if len(issues) > 0 {
		t.Fatal(issues)
	}
	state := r.State()
	state.Credentials[0].ItemReference = "TOKEN=SYNTHETIC_REJECTED"
	invalid, issues := local.NewRecord(state)
	if len(issues) == 0 || !reflect.DeepEqual(invalid, local.Record{}) {
		t.Fatal("unsafe proposal accepted")
	}
	if out, issues := local.EncodeRecord(invalid); out != nil || len(issues) == 0 {
		t.Fatal("unsafe output")
	}
}
