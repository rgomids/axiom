package local_test

import (
	"bytes"
	"context"
	"reflect"
	"testing"

	"github.com/rgomids/axiom/internal/local"
	"github.com/rgomids/axiom/internal/manifest"
	"github.com/rgomids/axiom/internal/project"
	"github.com/rgomids/axiom/internal/projectapp"
)

// Controlled LocalReader seam, not a filesystem adapter or install use case.
type recordReader struct {
	data   []byte
	exists bool
	reused int
}

var _ projectapp.LocalReader = (*recordReader)(nil)

func (f *recordReader) ReadLocal(_ context.Context, p project.Project) (projectapp.LocalSnapshot, projectapp.LocalRevision, []projectapp.Issue) {
	r, revision, issues := local.DecodeObservedRecord(f.data, f.exists)
	if len(issues) > 0 {
		return projectapp.LocalSnapshot{}, revision, []projectapp.Issue{{Phase: projectapp.LocalPhase, Field: projectapp.InstallationField, Code: projectapp.InvalidSnapshot}}
	}
	if !f.exists {
		return projectapp.LocalSnapshot{}, revision, nil
	}
	s := r.State()
	if s.ProjectID != p.State().ID {
		return projectapp.LocalSnapshot{}, revision, []projectapp.Issue{{Code: projectapp.InvalidSnapshot}}
	}
	snapshot, errs := projectapp.NewLocalSnapshot(p, projectapp.LocalState{Source: projectapp.NewDestination(), PortableRevision: s.PortableRevision, ArtifactDigests: s.ArtifactDigests, Repositories: s.Repositories, Credentials: s.Credentials, Runtime: s.Runtime, Attempt: s.Attempt})
	if len(errs) == 0 {
		f.reused++
	}
	return snapshot, revision, errs
}
func TestLocalReaderFailureNeverReusesBinding(t *testing.T) {
	p, issues := project.New(project.State{SchemaVersion: 1, ID: "123e4567-e89b-42d3-a456-426614174000", Slug: "demo", Name: "Demo"})
	if len(issues) > 0 {
		t.Fatal(issues)
	}
	good := fixture(t, "configured")
	for _, data := range [][]byte{good, bytes.Replace(good, []byte(`"formatVersion": 1`), []byte(`"formatVersion": 1, "localRevision": "revision-1"`), 1), bytes.Replace(good, []byte(`"formatVersion": 1,`), nil, 1), bytes.Replace(good, []byte(`"formatVersion": 1`), []byte(`"formatVersion": 2`), 1), nil} {
		f := &recordReader{data: data, exists: true}
		before := bytes.Clone(data)
		got, revision, issues := f.ReadLocal(context.Background(), p)
		if bytes.Equal(data, good) {
			if len(issues) > 0 || len(got.State().Credentials) != 1 || f.reused != 1 {
				t.Fatal("valid binding not preserved")
			}
			continue
		}
		if len(issues) == 0 || f.reused != 0 || !reflect.DeepEqual(got, projectapp.LocalSnapshot{}) || revision == projectapp.MissingLocalRevision() {
			t.Fatal("failed record reused or absent")
		}
		if !bytes.Equal(data, before) {
			t.Fatal("input changed")
		}
	}
}

func TestPortableCodecRejectsLocalFields(t *testing.T) {
	portable := []byte("schemaVersion: 1\nproject:\n  id: 123e4567-e89b-42d3-a456-426614174000\n  slug: demo\n  name: Demo\n")
	for _, field := range []string{"formatVersion", "projectId", "observedSlug", "sourceLocation", "portableRevision", "localRevision", "artifactDigests", "credentials", "attempt"} {
		p, issues := manifest.Decode(append(bytes.Clone(portable), []byte(field+": local\n")...))
		if len(issues) == 0 || p.Equivalent(p) {
			t.Fatal("portable codec accepted local field")
		}
	}
	for _, nested := range []string{"runtime:\n  id: example\n  explicitPath: /synthetic/bin\n", "repositories:\n  - key: api\n    observation: local\n"} {
		_, issues := manifest.Decode(append(bytes.Clone(portable), []byte(nested)...))
		if len(issues) == 0 {
			t.Fatal("nested local metadata accepted")
		}
	}
	p, issues := manifest.Decode(portable)
	if len(issues) > 0 {
		t.Fatal(issues)
	}
	encoded, issues := manifest.Encode(p)
	if len(issues) > 0 {
		t.Fatal(issues)
	}
	for _, field := range []string{"formatVersion", "sourceLocation", "localRevision", "artifactDigests", "explicitPath", "observation", "attempt"} {
		if bytes.Contains(encoded, []byte(field)) {
			t.Fatal("local state emitted")
		}
	}
}
