package local_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/rgomids/axiom/internal/local"
	"github.com/rgomids/axiom/internal/manifest"
	"github.com/rgomids/axiom/internal/project"
	"github.com/rgomids/axiom/internal/projectapp"
)

const artifactManifest = "schemaVersion: 1\nproject: {id: 123e4567-e89b-42d3-a456-426614174000, slug: demo, name: Demo}\n"

func documentManifest(t *testing.T, name string) []byte {
	t.Helper()
	quoted, err := json.Marshal(name)
	if err != nil {
		t.Fatal(err)
	}
	return []byte(artifactManifest + "businessContext:\n  documents: [" + string(quoted) + "]\npolicies: [policies/review.md]\n")
}

func TestPortableArtifactDigestRecordRoundTrip(t *testing.T) {
	for _, name := range []string{
		"context/api#legacy.md", "context/api?legacy.md", "context/api%23legacy.md",
		"context/ API legacy.md ", "context/ação.md", "context/ac\u0327a\u0303o.md",
		"context/api\tlegacy.md", "context/api\u0001legacy.md", "context/api\ufffdlegacy.md",
	} {
		t.Run(name, func(t *testing.T) {
			wire := documentManifest(t, name)
			p, issues := manifest.Decode(wire)
			if len(issues) != 0 || !p.Equivalent(p) {
				t.Fatal("manifest did not produce a valid Project", issues)
			}
			context, _ := p.State().BusinessContext.Value()
			names, _ := context.Documents.Value()
			if !reflect.DeepEqual(names, []string{name}) {
				t.Fatal("Project normalized document name", names)
			}
			docs := []projectapp.Document{{Name: "policies/review.md", Content: []byte("Synthetic policy")}, {Name: name, Content: []byte("Synthetic context")}}
			snapshot, errs := projectapp.ReadSnapshot(manifest.Codec{}, wire, docs)
			if len(errs) != 0 || !snapshot.Project().Equivalent(p) {
				t.Fatal("valid Project did not produce a valid snapshot", errs)
			}
			want := []projectapp.ArtifactDigest{
				{Name: "axiom.yaml", Digest: sha256.Sum256(wire)},
				{Name: name, Digest: sha256.Sum256(docs[1].Content)},
				{Name: docs[0].Name, Digest: sha256.Sum256(docs[0].Content)},
			}
			if !reflect.DeepEqual(snapshot.Digests(), want) {
				t.Fatal("snapshot changed names or content digests")
			}
			base, localIssues := local.DecodeRecord(fixture(t, "configured"))
			if len(localIssues) != 0 {
				t.Fatal(localIssues)
			}
			state := base.State()
			state.ProjectID, state.ObservedSlug = p.State().ID, p.State().Slug
			state.PortableRevision, state.ArtifactDigests = snapshot.Revision(), snapshot.Digests()
			record, localIssues := local.NewRecord(state)
			if len(localIssues) != 0 {
				t.Fatal("T04 rejected T02 digests", localIssues)
			}
			encoded, localIssues := local.EncodeRecord(record)
			if len(localIssues) != 0 {
				t.Fatal("T04 could not encode T02 digests", localIssues)
			}
			again, localIssues := local.DecodeRecord(encoded)
			if len(localIssues) != 0 || !reflect.DeepEqual(again.State(), state) {
				t.Fatal("round trip changed metadata", localIssues)
			}
			reencoded, localIssues := local.EncodeRecord(again)
			if len(localIssues) != 0 || !bytes.Equal(encoded, reencoded) {
				t.Fatal("unstable encoding", localIssues)
			}
		})
	}
}

func TestReadSnapshotRejectsInvalidUTF8BeforeLocalRecord(t *testing.T) {
	name := string([]byte{'c', 'o', 'n', 't', 'e', 'x', 't', '/', 0xff, '.', 'm', 'd'})
	// Keep the manifest valid: malformed bytes enter through Document.Name,
	// without any JSON/YAML encoding that could replace them first.
	snapshot, issues := projectapp.ReadSnapshot(manifest.Codec{}, []byte(artifactManifest), []projectapp.Document{{Name: name, Content: []byte("content")}})
	if len(issues) == 0 {
		t.Error("invalid UTF-8 document name accepted")
	}
	if snapshot.Digests() != nil {
		t.Error("invalid document produced reusable digests before T04")
	}
}

func TestArtifactNameUnicodeWirePreservation(t *testing.T) {
	for _, input := range []struct{ quoted, name string }{
		{`"context/api�.md"`, "context/api\ufffd.md"},
		{`"context/api\ufffd.md"`, "context/api\ufffd.md"},
		{`"context/api\uFFFD.md"`, "context/api\ufffd.md"},
		{`"context/api\ud83d\udcdd.md"`, "context/api📝.md"},
		{`"context/api\uDBFF\uDFFF.md"`, "context/api\U0010ffff.md"},
		{`"context/api\u0009.md"`, "context/api\t.md"},
	} {
		t.Run(input.quoted, func(t *testing.T) {
			wire := bytes.Replace(fixture(t, "configured"), []byte(`"context/readme.md"`), []byte(input.quoted), 1)
			record, issues := local.DecodeRecord(wire)
			if len(issues) != 0 || record.State().ArtifactDigests[1].Name != input.name {
				t.Fatal("valid Unicode name changed or rejected", issues)
			}
			encoded, issues := local.EncodeRecord(record)
			if len(issues) != 0 {
				t.Fatal(issues)
			}
			again, issues := local.DecodeRecord(encoded)
			if len(issues) != 0 || !reflect.DeepEqual(again.State(), record.State()) {
				t.Fatal("Unicode round trip changed metadata", issues)
			}
		})
	}
	for _, invalid := range []string{`"context/\ud800"`, `"context/\ud800x"`, `"context/\ud800\u0041"`, `"context/\ud800\ud800"`, `"context/\udc00"`, `"context/\udc00\ud800"`, `"context/\udfff"`, `"context/\uZZZZ"`, `"context/\u123"`, `"context/\ud800\uZZZZ"`} {
		rejected(t, bytes.Replace(fixture(t, "configured"), []byte(`"context/readme.md"`), []byte(invalid), 1))
	}
	base, issues := local.DecodeRecord(fixture(t, "configured"))
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	state := base.State()
	state.ArtifactDigests[1].Name = "context/\xff.md"
	if record, issues := local.NewRecord(state); len(issues) == 0 || !reflect.DeepEqual(record, local.Record{}) {
		t.Fatal("malformed UTF-8 accepted for lossy JSON encoding")
	}
	// The six literal characters backslash-u-d-8-0-0 are valid path metadata.
	state = base.State()
	state.SourceLocation = `/synthetic/\ud800`
	record, issues := local.NewRecord(state)
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	encoded, issues := local.EncodeRecord(record)
	if len(issues) != 0 {
		t.Fatal("literal escape text treated as a surrogate", issues)
	}
	again, issues := local.DecodeRecord(encoded)
	if len(issues) != 0 || !reflect.DeepEqual(again.State(), state) {
		t.Fatal("literal escape text changed", issues)
	}
}

func TestInvalidPortableDocumentNamesNeverProduceDigests(t *testing.T) {
	for _, name := range []string{"", "../outside.md", "context/../outside.md", "/absolute.md", "~/.file", "context\\file.md", "file:document", "context/$value", "context/`value`", "context/\x00file", "context/\rfile", "context/\nfile"} {
		t.Run(name, func(t *testing.T) {
			state := project.State{SchemaVersion: 1, ID: "123e4567-e89b-42d3-a456-426614174000", Slug: "demo", Name: "Demo", BusinessContext: project.Configured(project.BusinessContext{Documents: project.Configured([]string{name})})}
			if p, issues := project.New(state); len(issues) == 0 || p.Equivalent(p) {
				t.Fatal("domain accepted invalid document")
			}
			wire := documentManifest(t, name)
			if p, issues := manifest.Decode(wire); len(issues) == 0 || p.Equivalent(p) {
				t.Fatal("manifest accepted invalid document")
			}
			snapshot, issues := projectapp.ReadSnapshot(manifest.Codec{}, wire, []projectapp.Document{{Name: name}, {Name: "policies/review.md"}})
			if len(issues) == 0 || snapshot.Digests() != nil {
				t.Fatal("invalid document produced reusable digests")
			}
		})
	}
}

func TestArtifactNamesKeepSnapshotBoundaryProtections(t *testing.T) {
	base, issues := local.DecodeRecord(fixture(t, "configured"))
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	for _, name := range []string{"", "/absolute", "../outside", "context/../outside", "context/./file", "context//file", "context/", "context\\file", "file:document", "~/.file", "context/$value", "context/`value`", "context/\x00file", "context/\rfile", "context/\nfile", "axiom.yaml"} {
		t.Run(name, func(t *testing.T) {
			// An unreferenced document still crosses T02's artifact-name boundary.
			snapshot, errs := projectapp.ReadSnapshot(manifest.Codec{}, []byte(artifactManifest), []projectapp.Document{{Name: name}})
			if len(errs) == 0 || snapshot.Digests() != nil {
				t.Fatal("snapshot accepted invalid document")
			}
			state := base.State()
			state.ArtifactDigests[1].Name = name
			record, issues := local.NewRecord(state)
			if len(issues) == 0 || !reflect.DeepEqual(record, local.Record{}) {
				t.Fatal("invalid or duplicate artifact accepted")
			}
			quoted, _ := json.Marshal(name)
			rejected(t, bytes.Replace(fixture(t, "configured"), []byte(`"context/readme.md"`), quoted, 1))
		})
	}
}
