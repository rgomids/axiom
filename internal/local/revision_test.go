package local_test

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/rgomids/axiom/internal/local"
	"github.com/rgomids/axiom/internal/projectapp"
)

func TestLocalRevisionDerivedOnlyFromObservedBytes(t *testing.T) {
	input := fixture(t, "configured")
	record, revision, issues := local.DecodeObservedRecord(input, true)
	if len(issues) > 0 || revision != projectapp.ObserveLocalRevision(input) {
		t.Fatal("record without persisted revision must decode with exact observed revision", issues)
	}
	// Equal metadata with different JSON whitespace has a different byte revision.
	spaced := append(bytes.Clone(input), ' ')
	again, spacedRevision, issues := local.DecodeObservedRecord(spaced, true)
	if len(issues) > 0 || !reflect.DeepEqual(record.State(), again.State()) || spacedRevision == revision || spacedRevision != projectapp.ObserveLocalRevision(spaced) {
		t.Fatal("local revision followed metadata instead of exact bytes", issues)
	}
	output, issues := local.EncodeRecord(record)
	if len(issues) > 0 {
		t.Fatal(issues)
	}
	var object map[string]json.RawMessage
	if err := json.Unmarshal(output, &object); err != nil {
		t.Fatal(err)
	}
	if _, exists := object["localRevision"]; exists {
		t.Fatal("encoder persisted a local revision")
	}
	if _, exists := object["portableRevision"]; !exists {
		t.Fatal("encoder dropped independent portable revision metadata")
	}
	decoded, encodedRevision, issues := local.DecodeObservedRecord(output, true)
	if len(issues) > 0 || encodedRevision != projectapp.ObserveLocalRevision(output) || !reflect.DeepEqual(record.State(), decoded.State()) {
		t.Fatal("encoded bytes did not round trip with their own observed revision", issues)
	}
	state := record.State()
	state.Repositories[0].ExplicitPath = "/synthetic/relocated#checkout"
	changed, issues := local.NewRecord(state)
	if len(issues) > 0 {
		t.Fatal(issues)
	}
	changedBytes, issues := local.EncodeRecord(changed)
	if len(issues) > 0 {
		t.Fatal(issues)
	}
	changedRecord, changedRevision, issues := local.DecodeObservedRecord(changedBytes, true)
	if len(issues) > 0 || changedRevision == encodedRevision || changedRevision != projectapp.ObserveLocalRevision(changedBytes) || changedRecord.State().PortableRevision != record.State().PortableRevision {
		t.Fatal("local mutation changed portable revision metadata or lost byte revision", issues)
	}
}

func TestPersistedLocalRevisionRejectedWithoutReuse(t *testing.T) {
	input := bytes.Replace(fixture(t, "configured"), []byte(`"formatVersion": 1`), []byte(`"formatVersion": 1, "localRevision": "revision-1"`), 1)
	before := bytes.Clone(input)
	issues := rejected(t, input)
	if !reflect.DeepEqual(issues, []local.Issue{{Field: "installation", Code: "unknown_field"}}) {
		t.Fatal("removed field was not rejected by the closed schema", issues)
	}
	record, revision, issues := local.DecodeObservedRecord(input, true)
	if len(issues) == 0 || !reflect.DeepEqual(record, local.Record{}) || revision != (projectapp.LocalRevision{}) || !bytes.Equal(input, before) {
		t.Fatal("obsolete record reused, treated as missing, or rewritten")
	}
}
