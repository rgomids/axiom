package local

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/rgomids/axiom/internal/projectapp"
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
