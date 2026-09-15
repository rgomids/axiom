package projectapp_test

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/rgomids/axiom/internal/project"
	"github.com/rgomids/axiom/internal/projectapp"
)

const identity = "12345678-1234-4abc-8def-123456789abc"

// Synthetic codec substitutes only the boundary; it does not pretend to parse YAML.
type fixtureCodec struct{ definition project.Project }

func (f fixtureCodec) Decode([]byte) (project.Project, []projectapp.Issue) { return f.definition, nil }
func (f fixtureCodec) Encode(project.Project) ([]byte, []projectapp.Issue) {
	return []byte("synthetic manifest"), nil
}
func definition(t *testing.T) project.Project {
	t.Helper()
	p, issues := project.New(project.State{SchemaVersion: 1, ID: identity, Slug: "sample", Name: "Sample"})
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	return p
}
func snapshot(t *testing.T, content string) projectapp.ArtifactSnapshot {
	t.Helper()
	s, issues := projectapp.ReadSnapshot(fixtureCodec{definition(t)}, []byte(content), nil)
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	return s
}
func preview(t *testing.T) projectapp.Preview {
	t.Helper()
	p, issues := projectapp.PreviewPortable(projectapp.Create, projectapp.Destination{}, projectapp.NewDestination(), projectapp.ArtifactSnapshot{}, snapshot(t, "synthetic manifest"))
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	return p
}
func approved(t *testing.T, p projectapp.Preview) projectapp.Authority {
	t.Helper()
	a, issues := projectapp.Confirm(p, projectapp.Human)
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	return a
}

type portableSpy struct {
	calls    int
	result   projectapp.MutationResult
	received projectapp.AuthorizedPortable
}

func (s *portableSpy) CommitPortable(_ context.Context, request projectapp.AuthorizedPortable) projectapp.MutationResult {
	s.calls++
	s.received = request
	return s.result
}

type localSpy struct {
	calls  int
	result projectapp.MutationResult
}

func (s *localSpy) CommitLocal(context.Context, projectapp.AuthorizedLocal) projectapp.MutationResult {
	s.calls++
	return s.result
}

func TestAuthorityDenialCallsNoStore(t *testing.T) {
	for _, name := range []string{"missing", "revoked", "stale", "portable revision", "local revision", "zero preview"} {
		t.Run(name, func(t *testing.T) {
			p := preview(t)
			a := approved(t, p)
			current := p.Expected()
			switch name {
			case "missing":
				a = projectapp.Authority{}
			case "revoked":
				a.Revoke()
			case "stale":
				p = preview(t) // Same bytes, different explicit destination/preview.
			case "portable revision":
				current.Portable = snapshot(t, "changed").Revision()
			case "local revision":
				current.Local = projectapp.ObserveLocalRevision([]byte("changed"))
			case "zero preview":
				p = projectapp.Preview{}
			}
			store := &portableSpy{}
			result := projectapp.ApplyPortable(context.Background(), store, p, a, current)
			if store.calls != 0 || result.Commit != projectapp.NotCommitted || result.Status != projectapp.Denied {
				t.Fatalf("denial: %+v calls=%d", result, store.calls)
			}
		})
	}
}

func TestAuthorityBindsCompleteWriteSetIntentOperationAndDestinations(t *testing.T) {
	old := snapshot(t, "old")
	next := snapshot(t, "next")
	from, to := projectapp.NewDestination(), projectapp.NewDestination()
	p, issues := projectapp.PreviewPortable(projectapp.Update, from, from, old, next)
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	a := approved(t, p)
	changed, issues := projectapp.PreviewPortable(projectapp.Update, from, from, old, snapshot(t, "other complete bytes"))
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	for _, candidate := range []projectapp.Preview{changed, preview(t)} {
		store := &portableSpy{}
		if result := projectapp.ApplyPortable(context.Background(), store, candidate, a, candidate.Expected()); result.Status != projectapp.Denied || store.calls != 0 {
			t.Fatal("changed proposal authorized")
		}
	}
	// An operation/destination mismatch is invalid even before confirmation.
	if _, issues := projectapp.PreviewPortable(projectapp.Update, from, to, old, next); len(issues) == 0 {
		t.Fatal("update gained move authority")
	}
	store := &portableSpy{result: projectapp.MutationResult{Commit: projectapp.Committed, Status: projectapp.Applied}}
	result := projectapp.ApplyPortable(context.Background(), store, p, a, p.Expected())
	if store.calls != 1 || result.Commit != projectapp.Committed {
		t.Fatal("exact approval denied")
	}
	if !store.received.Valid() || !reflect.DeepEqual(store.received.Snapshot().Manifest(), next.Manifest()) {
		t.Fatal("store did not receive complete authorized state")
	}
	if !reflect.DeepEqual(p.Writes(), []projectapp.Write{{Destination: from, Name: "axiom.yaml", Action: projectapp.Replace}}) {
		t.Fatal("incorrect exact write set")
	}
}

func TestRevocationSharedAcrossAuthorityCopies(t *testing.T) {
	p := preview(t)
	a := approved(t, p)
	copy := a
	a.Revoke()
	if _, issues := projectapp.AuthorizePortable(p, copy, p.Expected()); len(issues) == 0 {
		t.Fatal("copy escaped revocation")
	}
	if (projectapp.AuthorizedPortable{}).Valid() || (projectapp.AuthorizedLocal{}).Valid() {
		t.Fatal("zero permit valid")
	}
}

func localPreview(t *testing.T) projectapp.Preview {
	t.Helper()
	state := projectapp.LocalState{Source: projectapp.NewDestination(), PortableRevision: snapshot(t, "source").Revision()}
	s, issues := projectapp.NewLocalSnapshot(definition(t), state)
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	p, issues := projectapp.PreviewLocal(projectapp.Install, projectapp.NewDestination(), projectapp.MissingLocalRevision(), s)
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	return p
}
func TestLocalAuthorityCannotAuthorizePortableAndViceVersa(t *testing.T) {
	p := localPreview(t)
	a := approved(t, p)
	store := &localSpy{result: projectapp.MutationResult{Commit: projectapp.Committed, Status: projectapp.Applied}}
	if got := projectapp.ApplyLocal(context.Background(), store, p, a, p.Expected()); got.Commit != projectapp.Committed || store.calls != 1 {
		t.Fatal("local permit rejected")
	}
	for _, name := range []string{"missing", "revoked", "stale portable", "stale local", "other authority"} {
		t.Run(name, func(t *testing.T) {
			a := approved(t, p)
			current := p.Expected()
			switch name {
			case "missing":
				a = projectapp.Authority{}
			case "revoked":
				a.Revoke()
			case "stale portable":
				current.Portable = snapshot(t, "changed").Revision()
			case "stale local":
				current.Local = projectapp.ObserveLocalRevision([]byte("changed"))
			case "other authority":
				a = approved(t, preview(t))
			}
			denied := &localSpy{}
			if got := projectapp.ApplyLocal(context.Background(), denied, p, a, current); got.Status != projectapp.Denied || denied.calls != 0 {
				t.Fatal("local denial failed")
			}
		})
	}
	if _, issues := projectapp.AuthorizePortable(p, a, p.Expected()); len(issues) == 0 {
		t.Fatal("local grant authorizes portable")
	}
	portable := preview(t)
	if _, issues := projectapp.AuthorizeLocal(portable, approved(t, portable), portable.Expected()); len(issues) == 0 {
		t.Fatal("portable grant authorizes local")
	}
	if _, issues := projectapp.Confirm(portable, projectapp.Approver(99)); len(issues) == 0 {
		t.Fatal("unknown/AI approver accepted")
	}
}

// Fake time/identity and fault seams require no ambient clock or entropy access.
type fakeClock struct{ instant time.Time }

func (f fakeClock) Now() time.Time { return f.instant }

type fakeEntropy struct{ calls int }

func (f *fakeEntropy) NewID() (string, []projectapp.Issue) { f.calls++; return identity, nil }
func TestInjectedMetadataAndFaultResults(t *testing.T) {
	var clock projectapp.Clock = fakeClock{time.Unix(1, 0)}
	entropy := &fakeEntropy{}
	var allocator projectapp.IdentityAllocator = entropy
	id, issues := allocator.NewID()
	if id != identity || len(issues) != 0 || entropy.calls != 1 || clock.Now() != time.Unix(1, 0) {
		t.Fatal("non deterministic fixture")
	}
	p := preview(t)
	for _, fault := range []projectapp.MutationResult{
		{Commit: projectapp.NotCommitted, Status: projectapp.Failed},
		{Commit: projectapp.Committed, Status: projectapp.Failed},
		{Commit: projectapp.CommitUnknown, Status: projectapp.Failed},
	} {
		store := &portableSpy{result: fault}
		got := projectapp.ApplyPortable(context.Background(), store, p, approved(t, p), p.Expected())
		if got.Commit != fault.Commit {
			t.Fatal("fault lost actual commit")
		}
	}
}
