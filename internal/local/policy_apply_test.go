package local

import (
	"bytes"
	"context"
	"errors"
	"github.com/rgomids/axiom/internal/manifest"
	"github.com/rgomids/axiom/internal/project"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/rgomids/axiom/internal/projectapp"
	"github.com/rgomids/axiom/internal/runtimeprofile"
	fx "github.com/rgomids/axiom/internal/runtimeprofile/runtimeprofiletest"
)

func reviewedPolicy(t *testing.T, f editPublicationFixture, intent projectapp.EditIntent) projectapp.EditAuthority {
	t.Helper()
	proposal, failure := projectapp.PreviewPolicyEdit(context.Background(), f.policyPorts(t), intent)
	if failure != projectapp.EditOK {
		t.Fatal(failure)
	}
	preview := proposal.Preview()
	return projectapp.EditAuthority{ProjectID: preview.ProjectID, PreviewDigest: preview.Digest, AuthorizeLocal: true}
}

func TestPolicyApplyExactReplayAndNoop(t *testing.T) {
	for _, name := range []string{"preview", "partial", "wrong_id", "wrong_digest", "no_authority", "apply", "noop", "cancelled"} {
		t.Run(name, func(t *testing.T) {
			f := newPolicyEditFixture(t)
			intent := policyIntent(t, `{"runtimes":{"mode":"members","add":[{"id":"another"}]}}`)
			if name == "noop" {
				intent = policyIntent(t, `{}`)
			}
			authority := reviewedPolicy(t, f, intent)
			want := projectapp.EditPublished
			ctx := context.Background()
			switch name {
			case "preview":
				authority = projectapp.EditAuthority{}
				want = projectapp.EditPreviewed
			case "partial":
				authority.ProjectID = ""
				want = projectapp.EditRejected
			case "wrong_id":
				authority.ProjectID = "123e4567-e89b-42d3-a456-426614174001"
				want = projectapp.EditDenied
			case "wrong_digest":
				authority.PreviewDigest = "wrong"
				want = projectapp.EditDenied
			case "no_authority":
				authority.AuthorizeLocal = false
				want = projectapp.EditRejected
			case "noop":
				want = projectapp.EditUnchanged
			case "cancelled":
				cancelled, cancel := context.WithCancel(ctx)
				cancel()
				ctx = cancelled
				want = projectapp.EditRejected
			}
			beforeManifest, beforeLocal := fileBytes(t, f.manifestPath()), fileBytes(t, f.recordPath())
			result := projectapp.ApplyPolicyEdit(ctx, f.policyPorts(t), f.publisher(nil), intent, authority)
			if result.Outcome != want {
				t.Fatal(result.Outcome, result.Failure, result.Publication)
			}
			if name != "apply" && (!bytes.Equal(beforeManifest, fileBytes(t, f.manifestPath())) || !bytes.Equal(beforeLocal, fileBytes(t, f.recordPath())) || len(f.editStates(t)) != 0) {
				t.Fatal("unauthorized or unchanged request wrote state")
			}
			if name == "apply" {
				repeated := projectapp.ApplyPolicyEdit(ctx, f.policyPorts(t), f.publisher(nil), intent, authority)
				if repeated.Outcome == projectapp.EditPublished {
					t.Fatal("stale replay republished")
				}
			}
		})
	}
}

func TestPolicyApplyTruthfulRecovery(t *testing.T) {
	for _, scenario := range []struct {
		name    string
		stage   EditStage
		failure error
		want    projectapp.EditOutcome
	}{
		{"before_portable", EditStageRecorded, errors.New("synthetic failure"), projectapp.EditFailedPublish},
		{"interrupted", EditStageRecorded, ErrSimulatedInterruption, projectapp.EditRecovery},
		{"after_portable", EditStagePortableCommitted, errors.New("synthetic failure"), projectapp.EditPartial},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			f := newPolicyEditFixture(t)
			intent := policyIntent(t, `{"runtimes":{"mode":"members","add":[{"id":"another"}]}}`)
			authority := reviewedPolicy(t, f, intent)
			before := fileBytes(t, f.manifestPath())
			result := projectapp.ApplyPolicyEdit(context.Background(), f.policyPorts(t), f.publisher(func(stage EditStage) error {
				if stage == scenario.stage {
					return scenario.failure
				}
				return nil
			}), intent, authority)
			if result.Outcome != scenario.want {
				t.Fatal(result.Outcome, result.Publication)
			}
			changed := !bytes.Equal(before, fileBytes(t, f.manifestPath()))
			if changed != (scenario.want == projectapp.EditPartial) {
				t.Fatal("incorrect portable commit truth")
			}
			if scenario.want == projectapp.EditRecovery {
				f.applyPlan(t, RestorePrior)
			}
			if scenario.want == projectapp.EditPartial {
				f.applyPlan(t, FinalizeCommitted)
			}
		})
	}
}

type policyPublisher func(context.Context, projectapp.EditPublicationRequest) projectapp.EditPublicationResult

func (p policyPublisher) PublishEdit(ctx context.Context, r projectapp.EditPublicationRequest) projectapp.EditPublicationResult {
	return p(ctx, r)
}

type policyReadFailure struct {
	base  projectapp.EditSource
	calls int
}

func (s *policyReadFailure) SelectForEdit(ctx context.Context, selector string) (projectapp.EditSelection, projectapp.EditFailure) {
	s.calls++
	if s.calls > 1 {
		return projectapp.EditSelection{}, projectapp.EditUnavailable
	}
	return s.base.SelectForEdit(ctx, selector)
}

func TestPolicyReadBackTruthAndDrift(t *testing.T) {
	for _, name := range []string{"v2", "v3", "noop", "reader_failure", "later_writer", "runtime_changed"} {
		t.Run(name, func(t *testing.T) {
			version := 3
			if name == "v2" {
				version = 2
			}
			f := newPolicyEditVersionFixture(t, version)
			intent := policyIntent(t, `{"runtimes":{"mode":"members","add":[{"id":"another"}]}}`)
			if name == "noop" {
				intent = policyIntent(t, `{}`)
			}
			authority := reviewedPolicy(t, f, intent)
			ports := f.policyPorts(t)
			if name == "reader_failure" {
				ports.Edit.Source = &policyReadFailure{base: ports.Edit.Source}
			}
			publisher := policyPublisher(func(ctx context.Context, r projectapp.EditPublicationRequest) projectapp.EditPublicationResult {
				result := f.publisher(nil).PublishEdit(ctx, r)
				if name == "later_writer" {
					later := policyIntent(t, `{"runtimes":{"mode":"members","add":[{"id":"later"}]}}`)
					applied := projectapp.ApplyPolicyEdit(ctx, f.policyPorts(t), f.publisher(nil), later, reviewedPolicy(t, f, later))
					if applied.Outcome != projectapp.EditPublished {
						t.Fatal(applied.Outcome)
					}
				}
				if name == "runtime_changed" {
					store, err := NewRuntimeProfileStore(f.state)
					if err != nil {
						t.Fatal(err)
					}
					fresh, err := store.Inspect(ctx)
					if err != nil {
						t.Fatal(err)
					}
					wire, err := runtimeprofile.EncodeV3(fx.CodexOnly())
					if err != nil {
						t.Fatal(err)
					}
					if _, err := store.Publish(ctx, fresh.Token, wire); err != nil {
						t.Fatal(err)
					}
				}
				return result
			})
			result := projectapp.ApplyPolicyEdit(context.Background(), ports, publisher, intent, authority)
			want := projectapp.EditPublished
			if name == "noop" {
				want = projectapp.EditUnchanged
			}
			if result.Outcome != want || result.ReadBack == nil {
				t.Fatal(result.Outcome, result.Failure)
			}
			status := "confirmed"
			if name == "reader_failure" {
				status = "unverified"
			}
			if name == "later_writer" {
				status = "drift"
			}
			if result.ReadBack.Status != status || result.ReadBack.ReadinessChanged != (name == "runtime_changed") {
				t.Fatal(result.ReadBack)
			}
			if result.ReadBack.Observed != nil {
				observed := result.ReadBack.Observed
				if observed.SchemaVersion != version || observed.ProjectID != authority.ProjectID {
					t.Fatal("identity/schema changed")
				}
				if status == "confirmed" && (observed.PortableRevision != result.ReadBack.ExpectedPortableRevision || observed.LocalRevision != result.ReadBack.ExpectedLocalRevision) {
					t.Fatal("read-back revisions differ")
				}
			}
		})
	}
}

func TestPolicyRemovalPreservesOtherProjectRegistryAndBindings(t *testing.T) {
	f := newPolicyEditFixture(t)
	ctx := context.Background()
	ports := f.policyPorts(t)
	selection, failure := ports.Edit.Source.SelectForEdit(ctx, "sample")
	if failure != projectapp.EditOK {
		t.Fatal(failure)
	}
	state := selection.Portable.Project().State()
	state.ID = "123e4567-e89b-42d3-a456-426614174001"
	state.Slug = "other"
	p, issues := project.New(state)
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	wire, issues := manifest.Encode(p)
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	if err := f.portable.Create(ctx, "other", wire); err != nil {
		t.Fatal(err)
	}
	otherSource := filepath.Join(f.projects, "other")
	if result := f.installation.Install(ctx, otherSource); result.Status != InstallationApplied {
		t.Fatal(result)
	}
	store, err := NewRuntimeProfileStore(f.state)
	if err != nil {
		t.Fatal(err)
	}
	fresh, err := store.Inspect(ctx)
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := runtimeprofile.EncodeV3(fx.MultiRuntime())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Publish(ctx, fresh.Token, cfg); err != nil {
		t.Fatal(err)
	}
	paths := []string{filepath.Join(otherSource, manifestName), filepath.Join(f.state, "projects", state.ID, installationRecord), filepath.Join(f.state, "runtime-profiles", "v1", runtimeProfileObject)}
	before := map[string][]byte{}
	for _, path := range paths {
		before[path] = fileBytes(t, path)
	}
	intent := policyIntent(t, `{"runtimes":{"mode":"replace","values":[]},"modelProfiles":{"mode":"replace","values":[]},"runtimePreferences":{"mode":"replace","values":[]}}`)
	result := projectapp.ApplyPolicyEdit(ctx, ports, f.publisher(nil), intent, reviewedPolicy(t, f, intent))
	if result.Outcome != projectapp.EditPublished || result.ReadBack.Status != "confirmed" {
		t.Fatal(result.Outcome, result.ReadBack)
	}
	for path, original := range before {
		if !bytes.Equal(original, fileBytes(t, path)) {
			t.Fatal("unrelated state changed", filepath.Base(path))
		}
	}
	observed, code := f.installation.Inspect(ctx, editTestProjectID)
	if code != "" {
		t.Fatal(code)
	}
	a, b := selection.Local, ApplicationRecord(observed.Record)
	b.PortableRevision, b.ArtifactDigests = a.PortableRevision, a.ArtifactDigests
	if !reflect.DeepEqual(a, b) {
		t.Fatal("local bindings changed")
	}
	other, err := f.portable.Inspect(ctx, "other")
	if err != nil || !other.Snapshot.Project().Equivalent(p) {
		t.Fatal("other Project read-back changed", err)
	}
}
