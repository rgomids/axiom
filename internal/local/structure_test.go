package local_test

import (
	"bytes"
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/rgomids/axiom/internal/local"
	"github.com/rgomids/axiom/internal/projectapp"
)

// Walk every object in a valid configured fixture and inject unknown fields,
// nulls, wrong scalar/collection types, and missing required fields in place.
func TestEveryNestedShape(t *testing.T) {
	var root map[string]any
	if err := json.Unmarshal(fixture(t, "configured"), &root); err != nil {
		t.Fatal(err)
	}
	var walk func(any)
	check := func() {
		b, err := json.Marshal(root)
		if err != nil {
			t.Fatal(err)
		}
		rejected(t, b)
	}
	walk = func(value any) {
		switch obj := value.(type) {
		case map[string]any:
			obj["SYNTHETIC_UNKNOWN"] = true
			check()
			delete(obj, "SYNTHETIC_UNKNOWN")
			for key, old := range obj {
				obj[key] = nil
				check()
				obj[key] = true
				check()
				delete(obj, key)
				if key != "runtime" && key != "attempt" {
					check()
				}
				obj[key] = old
				walk(old)
			}
		case []any:
			for _, item := range obj {
				walk(item)
			}
		}
	}
	walk(root)
}

func TestMetadataValidationAndSourceKinds(t *testing.T) {
	r, issues := local.DecodeRecord(fixture(t, "configured"))
	if len(issues) > 0 {
		t.Fatal(issues)
	}
	for _, kind := range []string{"environment", "macos-keychain", "windows-credential-manager", "secret-service", "libsecret", "runtime-managed", "unsupported-source", ""} {
		state := r.State()
		state.Credentials[0].SourceKind = kind
		if kind == "" {
			state.Credentials[0].ItemReference = ""
		}
		record, issues := local.NewRecord(state)
		if len(issues) > 0 {
			t.Fatal(kind, issues)
		}
		data, issues := local.EncodeRecord(record)
		if len(issues) > 0 {
			t.Fatal(issues)
		}
		again, issues := local.DecodeRecord(data)
		if len(issues) > 0 || !reflect.DeepEqual(record.State(), again.State()) {
			t.Fatal("reference metadata changed")
		}
	}
	mutations := []func(*local.RecordState){
		func(s *local.RecordState) { s.PortableRevision = projectapp.PortableRevision{} },
		func(s *local.RecordState) { s.ArtifactDigests = nil },
		func(s *local.RecordState) { s.ArtifactDigests = append(s.ArtifactDigests, s.ArtifactDigests[0]) },
		func(s *local.RecordState) { s.Repositories = append(s.Repositories, s.Repositories[0]) },
		func(s *local.RecordState) { s.Credentials = append(s.Credentials, s.Credentials[0]) },
		func(s *local.RecordState) { s.Repositories[0].Observation.Availability = 99 },
		func(s *local.RecordState) { s.Repositories[0].Observation.Basis = 99 },
		func(s *local.RecordState) { s.Runtime.ExplicitPath = "relative" },
		func(s *local.RecordState) { s.Attempt.Correlation = "token:synthetic" },
		func(s *local.RecordState) { s.Attempt.At = time.Date(10000, 1, 1, 0, 0, 0, 0, time.UTC) },
	}
	for _, mutate := range mutations {
		s := r.State()
		mutate(&s)
		r, issues := local.NewRecord(s)
		if len(issues) == 0 || !reflect.DeepEqual(r, local.Record{}) {
			t.Fatal("invalid metadata accepted")
		}
	}
	for _, bad := range []string{"a", "xx" + strings.Repeat("a", 62)} {
		rejected(t, bytes.Replace(fixture(t, "minimal"), []byte(strings.Repeat("a", 64)), []byte(bad), 1))
	}
}

func TestLimitsAndUnicode(t *testing.T) {
	assertCode(t, bytes.Repeat([]byte(" "), local.MaxRecordBytes+1), "byte_limit")
	assertCode(t, []byte(strings.Repeat("[", local.MaxRecordDepth+1)+strings.Repeat("]", local.MaxRecordDepth+1)), "depth_limit")
	assertCode(t, []byte("["+strings.Repeat("0,", local.MaxRecordNodes)+"0]"), "node_limit")
	for _, s := range []string{`"\ud800"`, `"\udc00"`, `"\ud800x"`} {
		rejected(t, bytes.Replace(fixture(t, "minimal"), []byte(`"revision-1"`), []byte(s), 1))
	}
}

func FuzzRecordRoundTrip(f *testing.F) {
	f.Add([]byte(`{"formatVersion":1}`))
	f.Add([]byte(`null`))
	for _, name := range []string{"minimal", "configured"} {
		b, err := os.ReadFile("testdata/" + name + ".json")
		if err != nil {
			f.Fatal(err)
		}
		f.Add(b)
	}
	f.Fuzz(func(t *testing.T, input []byte) {
		r, issues := local.DecodeRecord(input)
		if len(issues) > 0 {
			if !reflect.DeepEqual(r, local.Record{}) {
				t.Fatal("partial record")
			}
			return
		}
		output, issues := local.EncodeRecord(r)
		if len(issues) > 0 {
			t.Fatal("accepted record cannot encode", issues)
		}
		again, issues := local.DecodeRecord(output)
		if len(issues) > 0 || !reflect.DeepEqual(r.State(), again.State()) {
			t.Fatal("lossy round trip")
		}
	})
}

func TestObservationAndGapRoundTrips(t *testing.T) {
	r, issues := local.DecodeRecord(fixture(t, "configured"))
	if len(issues) > 0 {
		t.Fatal(issues)
	}
	for _, a := range []projectapp.Availability{projectapp.Unverified, projectapp.Available, projectapp.Unavailable} {
		for _, basis := range []projectapp.ObservationBasis{projectapp.NotChecked, projectapp.PresentMetadata, projectapp.MissingMetadata, projectapp.UnsupportedCheck, projectapp.UncertainPermissions, projectapp.MismatchedMetadata} {
			state := r.State()
			state.Runtime.Observation.Availability = a
			state.Runtime.Observation.Basis = basis
			record, issues := local.NewRecord(state)
			if len(issues) > 0 {
				t.Fatal(issues)
			}
			output, issues := local.EncodeRecord(record)
			if len(issues) > 0 {
				t.Fatal(issues)
			}
			again, issues := local.DecodeRecord(output)
			if len(issues) > 0 || !reflect.DeepEqual(record.State(), again.State()) {
				t.Fatal("changed observation")
			}
		}
	}
	state := r.State()
	state.Repositories[0].ExplicitPath = ""
	state.Repositories[0].CanonicalIdentity = ""
	state.Repositories[0].Observation = projectapp.Observation{}
	state.Credentials[0].SourceKind = ""
	state.Credentials[0].ItemReference = ""
	state.Runtime.ExplicitPath = ""
	state.Runtime.Observation = projectapp.Observation{}
	state.Attempt = projectapp.AttemptMetadata{}
	record, issues := local.NewRecord(state)
	if len(issues) > 0 {
		t.Fatal(issues)
	}
	output, issues := local.EncodeRecord(record)
	if len(issues) > 0 {
		t.Fatal(issues)
	}
	again, issues := local.DecodeRecord(output)
	if len(issues) > 0 || !reflect.DeepEqual(record.State(), again.State()) {
		t.Fatal("unresolved metadata changed")
	}
}

func TestAllReferenceSlotsRejectPayload(t *testing.T) {
	record, issues := local.DecodeRecord(fixture(t, "configured"))
	if len(issues) > 0 {
		t.Fatal(issues)
	}
	for _, mutate := range []func(*local.RecordState){
		func(s *local.RecordState) { s.LocalRevision = "token:synthetic" },
		func(s *local.RecordState) { s.Credentials[0].ReferenceKey = "token:synthetic" },
		func(s *local.RecordState) { s.Credentials[0].SourceKind = "token:synthetic" },
		func(s *local.RecordState) { s.Repositories[0].RepositoryKey = "token:synthetic" },
		func(s *local.RecordState) { s.Repositories[0].CanonicalIdentity = "token:synthetic" },
		func(s *local.RecordState) { s.Runtime.RuntimeID = "token:synthetic" },
		func(s *local.RecordState) { s.Attempt.Correlation = "token:synthetic" },
	} {
		s := record.State()
		mutate(&s)
		got, issues := local.NewRecord(s)
		if len(issues) == 0 || !reflect.DeepEqual(got, local.Record{}) {
			t.Fatal("payload accepted")
		}
	}
}

func TestNestedSecretAndBodyFields(t *testing.T) {
	data := fixture(t, "configured")
	for _, slot := range []string{`"referenceKey": "auth"`, `"name": "context/readme.md"`, `"availability": "unverified"`, `"runtimeId": "example-runtime"`, `"correlation": "attempt-1"`} {
		for _, forbidden := range []string{"secret", "token", "password", "apiKey", "body", "content"} {
			bad := bytes.Replace(data, []byte(slot), []byte(slot+`, "`+forbidden+`": "SYNTHETIC_REJECTED"`), 1)
			if bytes.Equal(data, bad) {
				t.Fatal("mutation not applied")
			}
			issues := rejected(t, bad)
			if strings.Contains(fmtIssues(issues), "SYNTHETIC_REJECTED") {
				t.Fatal("diagnostic leaked payload")
			}
		}
	}
}

func assertCode(t *testing.T, input []byte, code string) {
	t.Helper()
	issues := rejected(t, input)
	if issues[0].Code != code {
		t.Fatalf("got %s, want %s", issues[0].Code, code)
	}
}
func TestByteBoundaryAndEncodeLimit(t *testing.T) {
	input := fixture(t, "minimal")
	input = append(input, bytes.Repeat([]byte(" "), local.MaxRecordBytes-len(input))...)
	r, issues := local.DecodeRecord(input)
	if len(issues) > 0 {
		t.Fatal(issues)
	}
	assertCode(t, append(input, ' '), "byte_limit")
	s := r.State()
	s.LocalRevision = strings.Repeat("a", local.MaxRecordBytes)
	r, issues = local.NewRecord(s)
	if len(issues) > 0 {
		t.Fatal(issues)
	}
	output, issues := local.EncodeRecord(r)
	if output != nil || len(issues) == 0 || issues[0].Code != "byte_limit" {
		t.Fatal("oversized encoding returned")
	}
}
