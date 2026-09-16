package local_test

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/rgomids/axiom/internal/local"
	"github.com/rgomids/axiom/internal/project"
	"github.com/rgomids/axiom/internal/projectapp"
)

// These exercise public codec boundaries with metadata only, never OS paths.
func setLocations(s *local.RecordState, value string) {
	s.SourceLocation = value
	s.Repositories[0].ExplicitPath = value
	s.Runtime.ExplicitPath = value
}

func TestLocalPathsPreserveArbitraryMetadata(t *testing.T) {
	base, issues := local.DecodeRecord(fixture(t, "configured"))
	if len(issues) > 0 {
		t.Fatal(issues)
	}
	for _, value := range []string{
		"/Users/me/work/foo#legacy", "/synthetic/$cache", "/synthetic/what?",
		"/synthetic/`literal`", `/synthetic/back\slash`, `/synthetic/"quoted"`,
		"/synthetic/ação 日本語", " /synthetic/space ", "/synthetic//path/",
		"/synthetic/./path", "/synthetic/../path", "relative/path", "../path",
		"/", " ",
	} {
		t.Run(value, func(t *testing.T) {
			state := base.State()
			setLocations(&state, value)
			record, issues := local.NewRecord(state)
			if len(issues) > 0 {
				t.Fatal(issues)
			}
			wire, issues := local.EncodeRecord(record)
			if len(issues) > 0 {
				t.Fatal(issues)
			}
			before := bytes.Clone(wire)
			again, revision, issues := local.DecodeObservedRecord(wire, true)
			if len(issues) > 0 || !reflect.DeepEqual(again.State(), state) {
				t.Fatal("local path metadata changed", issues)
			}
			if !bytes.Equal(wire, before) || revision != projectapp.ObserveLocalRevision(wire) {
				t.Fatal("input bytes or observed revision changed")
			}
		})
	}
}

func TestLocalPathTextRejectedAtEachBoundary(t *testing.T) {
	base, issues := local.DecodeRecord(fixture(t, "configured"))
	if len(issues) > 0 {
		t.Fatal(issues)
	}
	for _, slot := range []struct {
		name string
		set  func(*local.RecordState, string)
		old  string
	}{
		{"source", func(s *local.RecordState, v string) { s.SourceLocation = v }, "/synthetic/projects/demo"},
		{"repository", func(s *local.RecordState, v string) { s.Repositories[0].ExplicitPath = v }, "/synthetic/checkouts/api"},
		{"runtime", func(s *local.RecordState, v string) { s.Runtime.ExplicitPath = v }, "/synthetic/bin/runtime"},
	} {
		t.Run(slot.name, func(t *testing.T) {
			for _, value := range []string{"/synthetic/\x00x", "/synthetic/\nx", "/synthetic/\rx", "/synthetic/\tx", "/synthetic/\x7fx", "/synthetic/\u0085x", "/synthetic/\ufffdx", "/synthetic/\xffx"} {
				state := base.State()
				slot.set(&state, value)
				record, issues := local.NewRecord(state)
				if len(issues) == 0 || !reflect.DeepEqual(record, local.Record{}) {
					t.Fatal("invalid text returned reusable record")
				}
				if out, errs := local.EncodeRecord(record); out != nil || len(errs) == 0 {
					t.Fatal("invalid path encoded")
				}
				encoded, _ := json.Marshal(value)
				old, _ := json.Marshal(slot.old)
				rejected(t, bytes.Replace(fixture(t, "configured"), old, encoded, 1))
			}
			// Raw malformed UTF-8 and escaped unpaired surrogates must not repair.
			for _, encoded := range [][]byte{{'"', 0xff, '"'}, []byte(`"\ud800"`)} {
				old, _ := json.Marshal(slot.old)
				rejected(t, bytes.Replace(fixture(t, "configured"), old, encoded, 1))
			}
		})
	}
	state := base.State()
	state.SourceLocation = ""
	if _, issues := local.NewRecord(state); len(issues) == 0 {
		t.Fatal("missing required source accepted")
	}
	state = base.State()
	state.Repositories[0].ExplicitPath = ""
	state.Runtime.ExplicitPath = ""
	if _, issues := local.NewRecord(state); len(issues) > 0 {
		t.Fatal("unresolved optional paths rejected", issues)
	}
}

func TestIdentityDiagnosticsUseLocalSchemaWithDomainRules(t *testing.T) {
	base, issues := local.DecodeRecord(fixture(t, "minimal"))
	if len(issues) > 0 {
		t.Fatal(issues)
	}
	valid := base.State()
	for _, input := range []struct{ id, slug string }{
		{valid.ProjectID, valid.ObservedSlug},
		{valid.ProjectID, "a-0"},
		{"123e4567-e89b-42d3-b456-426614174000", "demo"},
		{"", "demo"},
		{"123E4567-e89b-42d3-a456-426614174000", "demo"},
		{"123e4567-e89b-12d3-a456-426614174000", "demo"},
		{"123e4567-e89b-42d3-7456-426614174000", "demo"},
		{valid.ProjectID, ""},
		{valid.ProjectID, "../demo"},
		{valid.ProjectID, "Demo"},
		{valid.ProjectID, "demo--x"},
		{valid.ProjectID, "démø"},
		{"bad", "../bad"},
	} {
		state := base.State()
		state.ProjectID, state.ObservedSlug = input.id, input.slug
		_, constructed := local.NewRecord(state)
		var wire map[string]any
		if err := json.Unmarshal(fixture(t, "minimal"), &wire); err != nil {
			t.Fatal(err)
		}
		wire["projectId"], wire["observedSlug"] = input.id, input.slug
		data, err := json.Marshal(wire)
		if err != nil {
			t.Fatal(err)
		}
		_, decoded := local.DecodeRecord(data)
		domain := project.ValidateIdentity(input.id, input.slug)
		for _, got := range [][]local.Issue{constructed, decoded} {
			if len(domain) == 0 {
				if len(got) > 0 {
					t.Fatal("codec rejected domain-valid identity", got)
				}
				continue
			}
			// The codec retains first-error ordering and local-state remedy/category.
			field := map[string]string{"project.id": "installation.projectId", "project.slug": "installation.observedSlug"}[domain[0].Field]
			want := []local.Issue{{Field: field, Code: domain[0].Code}}
			if !reflect.DeepEqual(got, want) || got[0].Severity() != "error" || got[0].Category() != "invalid_local_state" || got[0].Remedy() != "preserve_record_and_inspect_local_state" {
				t.Fatalf("local identity diagnostics %v, want %v", got, want)
			}
		}
	}
}
