package projectapp_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/rgomids/axiom/internal/manifest"
	"github.com/rgomids/axiom/internal/projectapp"
	"github.com/rgomids/axiom/internal/testfs"
)

// I230-T03 application Evidence: authorized EDIT publication rebuilds the
// candidate from fresh observations and binds exact authority.

type publisherSpy struct {
	requests []projectapp.EditPublicationRequest
	result   projectapp.EditPublicationResult
}

func (p *publisherSpy) PublishEdit(_ context.Context, request projectapp.EditPublicationRequest) projectapp.EditPublicationResult {
	p.requests = append(p.requests, request)
	return p.result
}

func (f *editFixture) apply(intent projectapp.EditIntent, authority projectapp.EditAuthority, publisher projectapp.EditPublisher) projectapp.EditResult {
	if intent.Selector == "" {
		intent.Selector = "sample"
	}
	return projectapp.ApplyEdit(context.Background(), projectapp.EditPorts{Source: f, Checkouts: &checkoutFixture{}, Manifest: manifest.Codec{}, Local: localCodecFixture{}}, publisher, intent, authority)
}

func authorityFor(preview projectapp.EditPreview) projectapp.EditAuthority {
	return projectapp.EditAuthority{ProjectID: preview.ProjectID, PreviewDigest: preview.Digest, AuthorizeLocal: true}
}

func TestApplyEditRejectsPartialAuthorityBeforeAnyRead(t *testing.T) {
	for name, authority := range map[string]projectapp.EditAuthority{
		"id only":            {ProjectID: editProjectID},
		"digest only":        {PreviewDigest: "digest"},
		"authority only":     {AuthorizeLocal: true},
		"missing authority":  {ProjectID: editProjectID, PreviewDigest: "digest"},
		"missing digest":     {ProjectID: editProjectID, AuthorizeLocal: true},
		"missing project id": {PreviewDigest: "digest", AuthorizeLocal: true},
	} {
		t.Run(name, func(t *testing.T) {
			f := newEditFixture(t)
			spy := &publisherSpy{}
			result := f.apply(projectapp.EditIntent{Name: set("Renamed")}, authority, spy)
			if result.Outcome != projectapp.EditRejected || result.Failure != projectapp.EditIncompleteAuthority || len(f.selectors) != 0 || len(spy.requests) != 0 {
				t.Fatalf("result=%+v selectors=%v requests=%d", result, f.selectors, len(spy.requests))
			}
		})
	}
}

func TestApplyEditPreviewsThenPublishesOnlyExactAuthority(t *testing.T) {
	f := newEditFixture(t)
	spy := &publisherSpy{result: projectapp.EditPublicationResult{Status: projectapp.EditPublicationApplied}}
	intent := projectapp.EditIntent{Name: set("Renamed"), RepositoryRemovals: []string{"web"}}
	preview := f.apply(intent, projectapp.EditAuthority{}, spy)
	if preview.Outcome != projectapp.EditPreviewed || len(spy.requests) != 0 {
		t.Fatalf("preview = %+v", preview)
	}
	reviewed := preview.Proposal.Preview()

	stale := authorityFor(reviewed)
	stale.PreviewDigest = "0" + stale.PreviewDigest[1:]
	if denied := f.apply(intent, stale, spy); denied.Outcome != projectapp.EditDenied || len(spy.requests) != 0 {
		t.Fatalf("stale digest = %+v", denied)
	}
	drift := authorityFor(reviewed)
	drift.ProjectID = "123e4567-e89b-42d3-a456-426614174999"
	if denied := f.apply(intent, drift, spy); denied.Outcome != projectapp.EditDenied || len(spy.requests) != 0 {
		t.Fatalf("ID drift = %+v", denied)
	}
	if denied := f.apply(projectapp.EditIntent{Name: set("Other")}, authorityFor(reviewed), spy); denied.Outcome != projectapp.EditDenied || len(spy.requests) != 0 {
		t.Fatalf("other intent = %+v", denied)
	}

	published := f.apply(intent, authorityFor(reviewed), spy)
	if published.Outcome != projectapp.EditPublished || len(spy.requests) != 1 {
		t.Fatalf("publish = %+v", published)
	}
	request := spy.requests[0]
	if request.ProjectID != editProjectID || request.Slug != "sample" || request.PortableDestination != testfs.Path("/portable/sample") ||
		!bytes.Equal(request.PortableExpected, f.selection.Portable.Manifest()) || !bytes.Equal(request.LocalExpected, f.selection.LocalWire) ||
		!bytes.Equal(request.PortableNext, published.Proposal.Manifest()) || !bytes.Equal(request.LocalNext, published.Proposal.LocalWire()) {
		t.Fatalf("publication request does not carry exact observations and candidates: %+v", request)
	}
}

func TestApplyEditLocalOnlyNoOpAndPublicationOutcomes(t *testing.T) {
	f := newEditFixture(t)
	spy := &publisherSpy{result: projectapp.EditPublicationResult{Status: projectapp.EditPublicationApplied}}
	localOnly := projectapp.EditIntent{RepositoryUpserts: []projectapp.RepositoryUpsert{{Key: "web", Path: testfs.Path("/work/web-fixed")}}}
	preview := f.apply(localOnly, projectapp.EditAuthority{}, spy).Proposal.Preview()
	if result := f.apply(localOnly, authorityFor(preview), spy); result.Outcome != projectapp.EditPublished || len(spy.requests[0].PortableNext) != 0 {
		t.Fatalf("local-only publication must not republish the portable Project: %+v", spy.requests)
	}

	noop := f.apply(projectapp.EditIntent{}, projectapp.EditAuthority{}, spy).Proposal.Preview()
	if result := f.apply(projectapp.EditIntent{}, authorityFor(noop), spy); result.Outcome != projectapp.EditUnchanged || len(spy.requests) != 1 {
		t.Fatalf("no-op replay = %+v", result)
	}

	intent := projectapp.EditIntent{Name: set("Renamed")}
	reviewed := f.apply(intent, projectapp.EditAuthority{}, spy).Proposal.Preview()
	for status, want := range map[projectapp.EditPublicationStatus]projectapp.EditOutcome{
		projectapp.EditPublicationConflict:         projectapp.EditConflicted,
		projectapp.EditPublicationPartial:          projectapp.EditPartial,
		projectapp.EditPublicationRecoveryRequired: projectapp.EditRecovery,
		projectapp.EditPublicationFailed:           projectapp.EditFailedPublish,
	} {
		spy.result = projectapp.EditPublicationResult{Status: status}
		if result := f.apply(intent, authorityFor(reviewed), spy); result.Outcome != want {
			t.Fatalf("%s -> %s, want %s", status, result.Outcome, want)
		}
	}
}

func TestEditPreviewDisclosesPreservedRepositoryHistory(t *testing.T) {
	f := newEditFixture(t)
	proposal := mustPreview(t, f, projectapp.EditIntent{RepositoryRemovals: []string{"web", "api"}})
	got := effectCodes(proposal.Preview().Effects)
	want := []string{"local:preserve_repository_history:api", "local:preserve_repository_history:web"}
	if len(got) < 2 || got[len(got)-2] != want[0] || got[len(got)-1] != want[1] {
		t.Fatalf("effects = %v", got)
	}
}
