package stageexecution

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/rgomids/axiom/internal/executiongraph"
	"github.com/rgomids/axiom/internal/workflow"
)

type fakeRunPorts struct {
	events       []string
	fail         string
	publication  Publication
	readMismatch bool
	onBuild      func()
}

func (f *fakeRunPorts) event(s string) error {
	f.events = append(f.events, s)
	if f.fail == s {
		return errInjected
	}
	return nil
}
func (f *fakeRunPorts) Build(_ context.Context, p DispatchPreview, actor string, at time.Time) (Publication, error) {
	if e := f.event("build"); e != nil {
		return Publication{}, e
	}
	if f.onBuild != nil {
		f.onBuild()
	}
	// The publication port is a fake: no filesystem, Runtime or Provider effects.
	f.publication = Publication{Authority: executiongraph.DispatchAuthority{PreviewDigest: p.Digest, Actor: actor, RecordedAt: at}}
	if p.Publication == nil {
		f.publication.Single = &SinglePublication{Execution: workflow.State{ExecutionID: p.Plan.Execution.ExecutionID}}
	} else {
		f.publication.Graph = &executiongraph.SuccessorGraph{Parent: executiongraph.ParentExecution{ExecutionID: p.Plan.Execution.ExecutionID}}
	}
	return f.publication, nil
}
func (f *fakeRunPorts) Commit(_ context.Context, p Publication) error {
	f.publication = p
	if f.fail == "committed" {
		f.events = append(f.events, "commit")
		return fakeCommittedError{}
	}
	return f.event("commit")
}
func (f *fakeRunPorts) ReadBack(_ context.Context, p Publication) (Publication, error) {
	e := f.event("readback")
	if f.readMismatch {
		p.Authority.Actor = "foreign"
	}
	return p, e
}
func (f *fakeRunPorts) Prepare(context.Context, Publication) error { return f.event("workspaces") }
func (f *fakeRunPorts) Project(context.Context, Publication) error { return f.event("projection") }
func (f *fakeRunPorts) Start(context.Context, Publication) (executiongraph.ProcessCorrelation, error) {
	return executiongraph.ProcessCorrelation{PID: 123, StartIdentity: "fake-start", GroupIdentity: "fake-group"}, f.event("start")
}
func runFixture(t *testing.T, single bool) (RunService, RunRequest, *fakeRunPorts, *fakePreviewSource) {
	t.Helper()
	preview, source, compiler := previewFixture(t, single)
	p, e := preview.Preview(context.Background(), compiler.plan.Execution, compiler.plan.StageID)
	if e != nil {
		t.Fatal(e)
	}
	ports := &fakeRunPorts{}
	service := RunService{Preview: preview, Publisher: ports, Workspaces: ports, Starter: ports, Projection: ports, Now: func() time.Time { return time.Unix(20, 0).UTC() }}
	r := RunRequest{Execution: compiler.plan.Execution, StageID: compiler.plan.StageID, PreviewDigest: p.Digest, ExpectedRevision: compiler.plan.Execution.Revision, AuthorizeLocal: true, AuthorizeExternal: true, Actor: "operator"}
	return service, r, ports, source
}
func TestRunGraphAndSingleCommitBeforeStart(t *testing.T) {
	for _, single := range []bool{false, true} {
		t.Run(map[bool]string{false: "graph", true: "single"}[single], func(t *testing.T) {
			s, r, f, _ := runFixture(t, single)
			result, e := s.Apply(context.Background(), r)
			if e != nil || result.Status != "success" || result.Process == nil {
				t.Fatal(result, e)
			}
			if !reflect.DeepEqual(f.events, []string{"build", "commit", "readback", "projection", "workspaces", "start"}) {
				t.Fatal(f.events)
			}
			if (result.Publication.Single != nil) != single || (result.Publication.Graph != nil) == single {
				t.Fatal("fabricated graph or single", result)
			}
		})
	}
}
func TestRunAuthorityAndDriftRefuseBeforeCommit(t *testing.T) {
	for _, name := range []string{"external authority", "local authority", "revision", "preview digest", "policy drift", "identity drift", "drift during allocation"} {
		t.Run(name, func(t *testing.T) {
			s, r, f, source := runFixture(t, false)
			want := ErrStalePreview
			switch name {
			case "external authority":
				r.AuthorizeExternal = false
				want = ErrDispatchAuthority
			case "local authority":
				r.AuthorizeLocal = false
				want = ErrDispatchAuthority
			case "revision":
				r.ExpectedRevision++
			case "preview digest":
				r.PreviewDigest = strings.Repeat("f", 64)
			case "policy drift":
				source.o.PolicyDigest = strings.Repeat("f", 64)
			case "identity drift":
				changed := *s.Preview.Compiler.(*fakePlanCompiler).plan.Compilation.Agents[0].Input.FrozenProfile
				changed.ExecutableDigest = strings.Repeat("f", 64)
				s.Preview.Compiler.(*fakePlanCompiler).plan.Compilation.Agents[0].Input.FrozenProfile = &changed
			case "drift during allocation":
				f.onBuild = func() { source.o.PolicyDigest = strings.Repeat("f", 64) }
			}
			result, e := s.Apply(context.Background(), r)
			if !errors.Is(e, want) || result.Status != "denied_authority" {
				t.Fatal(result, e)
			}
			for _, event := range f.events {
				if event != "build" {
					t.Fatalf("effect before authority: %s", event)
				}
			}
		})
	}
}
func TestRunPublicationAndStartFailures(t *testing.T) {
	cases := []struct {
		failure, status string
		events          []string
	}{
		{"commit", "failure", []string{"build", "commit"}},
		{"readback", "recovery_required", []string{"build", "commit", "readback"}},
		{"projection", "partial", []string{"build", "commit", "readback", "projection"}},
		{"workspaces", "recovery_required", []string{"build", "commit", "readback", "projection", "workspaces"}},
		{"start", "recovery_required", []string{"build", "commit", "readback", "projection", "workspaces", "start"}},
	}
	for _, tc := range cases {
		for _, single := range []bool{false, true} {
			t.Run(tc.failure+map[bool]string{false: "/graph", true: "/single"}[single], func(t *testing.T) {
				s, r, f, _ := runFixture(t, single)
				f.fail = tc.failure
				result, e := s.Apply(context.Background(), r)
				if !errors.Is(e, errInjected) || result.Status != tc.status || result.Recovery == "" || !reflect.DeepEqual(f.events, tc.events) {
					t.Fatal(result, e, f.events)
				}
			})
		}
	}
	s, r, f, _ := runFixture(t, false)
	f.readMismatch = true
	result, e := s.Apply(context.Background(), r)
	if !errors.Is(e, ErrSupervisorRecovery) || result.Status != "recovery_required" || !reflect.DeepEqual(f.events, []string{"build", "commit", "readback"}) {
		t.Fatal(result, e, f.events)
	}
}

type fakeCommittedError struct{}

func (fakeCommittedError) Error() string         { return "injected confirmed commit failure" }
func (fakeCommittedError) EffectCommitted() bool { return true }
func TestRunConfirmedCommitFailureIsPartial(t *testing.T) {
	s, r, f, _ := runFixture(t, false)
	f.fail = "committed"
	result, e := s.Apply(context.Background(), r)
	if e == nil || result.Status != "partial" || result.Publication.Graph == nil || result.Recovery == "" || !reflect.DeepEqual(f.events, []string{"build", "commit"}) {
		t.Fatal(result, e, f.events)
	}
}
