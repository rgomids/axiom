package workitem

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/completion"
)

// Lifecycle fakes. The fake document is one "name: content" line per section;
// byte-exact GitHub revision is covered by the adapter tests.

func (p *fakeCapability) current() External {
	state := p.state
	if state == "" {
		state = OpenState
	}
	return External{ID: "7", URL: "https://github.com/owner/repo/issues/7", State: state}
}

func (p *fakeCapability) ReadDocument(context.Context, string, string) (External, ProviderDocument, error) {
	p.reads++
	if p.readErr != nil {
		return External{}, ProviderDocument{}, p.readErr
	}
	title, body := p.title, p.body
	if title == "" {
		title = "Original title"
	}
	if body == "" {
		body = "problem: Original problem\nscope: Original scope\n"
	}
	return p.current(), ProviderDocument{Title: title, Body: body}, nil
}

func (*fakeCapability) ReviseDocument(current ProviderDocument, revision DocumentRevision) (ProviderDocument, error) {
	revised := ProviderDocument{Title: current.Title, Body: current.Body}
	if revision.Title != "" {
		revised.Title = revision.Title
	}
	lines := strings.Split(revised.Body, "\n")
	for _, section := range revision.Sections {
		found := false
		for index, line := range lines {
			if strings.HasPrefix(line, section.Name+": ") {
				lines[index], found = section.Name+": "+section.Content, true
			}
		}
		if !found {
			return ProviderDocument{}, ErrDocumentUnrecognized
		}
	}
	revised.Body = strings.Join(lines, "\n")
	return revised, nil
}

func (p *fakeCapability) UpdateDocument(_ context.Context, _, _ string, change DocumentChange) (External, error) {
	p.mutations = append(p.mutations, "update:"+change.Title+"|"+change.Body)
	if p.mutationErr != nil {
		return External{}, p.mutationErr
	}
	if change.Title != "" {
		p.title = change.Title
	}
	if change.Body != "" {
		p.body = change.Body
	}
	return p.current(), nil
}

func (p *fakeCapability) SetState(_ context.Context, _, _, state string) (External, error) {
	p.mutations = append(p.mutations, "state:"+state)
	if p.mutationErr != nil {
		return External{}, p.mutationErr
	}
	p.state = state
	return p.current(), nil
}

func (p *fakeCapability) Comment(_ context.Context, _, _, message string) error {
	p.mutations = append(p.mutations, "comment:"+message)
	return p.mutationErr
}

const testProjectID = "123e4567-e89b-42d3-a456-426614174000"

func lifecycleTarget() Target {
	return Target{ProjectSelector: "sample", RepositoryKey: "main", ProviderResource: "owner/repo"}
}

func seedLink(store *fakeStore, repository, resource, id, state string) Link {
	link := Link{ProjectID: testProjectID, RepositoryKey: repository, Provider: "github", Resource: resource, ExternalID: id, URL: "https://github.com/" + resource + "/issues/" + id, State: state, Revision: [32]byte{9}}
	store.links["github:"+resource+":"+id] = link
	return link
}

func providerCalls(provider *fakeCapability) int {
	return provider.reads + provider.creates + provider.reconciles + len(provider.mutations)
}

func TestListEnumeratesLocalLinksOnlyInDeterministicOrder(t *testing.T) {
	provider, store := &fakeCapability{}, newFakeStore()
	for _, id := range []string{"10", "2", "7"} {
		seedLink(store, "main", "owner/repo", id, OpenState)
	}
	seedLink(store, "detached", "owner/repo", "3", OpenState)
	service := testService(provider, store)
	listed := service.List(context.Background(), ListTarget{ProjectSelector: "sample"})
	var ids []string
	for _, link := range listed.Links {
		ids = append(ids, link.RepositoryKey+"#"+link.ExternalID)
	}
	if listed.Status != completion.Success || listed.Category != "work_items_listed" || !reflect.DeepEqual(ids, []string{"main#2", "main#7", "main#10"}) {
		t.Fatalf("list = %#v ids=%v", listed, ids)
	}
	if again := service.List(context.Background(), ListTarget{ProjectSelector: "sample", RepositoryKey: "main"}); !reflect.DeepEqual(again.Links, listed.Links) {
		t.Fatalf("filtered list = %#v", again.Links)
	}
	if empty := testService(provider, newFakeStore()).List(context.Background(), ListTarget{ProjectSelector: "sample"}); empty.Status != completion.Success || empty.Links == nil || len(empty.Links) != 0 {
		t.Fatalf("empty list = %#v", empty)
	}
	if unknown := service.List(context.Background(), ListTarget{ProjectSelector: "sample", RepositoryKey: "other"}); unknown.Category != "repository_not_configured" {
		t.Fatalf("unknown repository = %#v", unknown)
	}
	store.listErr = errors.Join(errors.New("marker present"), ErrRecoveryRequired)
	if failed := service.List(context.Background(), ListTarget{ProjectSelector: "sample"}); failed.Status != completion.Failure || failed.Category != "recovery_required" || failed.Links != nil {
		t.Fatalf("recovery list = %#v", failed)
	}
	store.listErr = nil
	foreign := seedLink(store, "main", "owner/repo", "11", OpenState)
	foreign.ProjectID = "223e4567-e89b-42d3-a456-426614174000"
	store.links["github:owner/repo:11"] = foreign
	if failed := service.List(context.Background(), ListTarget{ProjectSelector: "sample"}); failed.Category != "local_work_item_read_failed" {
		t.Fatalf("foreign link list = %#v", failed)
	}
	if providerCalls(provider) != 0 {
		t.Fatalf("list called the Provider: %+v", provider)
	}
}

func TestListFiltersByRepositoryAndOrdersAcrossRepositories(t *testing.T) {
	provider, store := &fakeCapability{}, newFakeStore()
	seedLink(store, "main", "owner/repo", "1", OpenState)
	seedLink(store, "api", "owner/api", "9", OpenState)
	seedLink(store, "api", "owner/api", "10", OpenState)
	service := New(fakeResolver{repositories: []Repository{{Key: "main", Path: "/unused"}, {Key: "api", Path: "/unused"}}}, provider, provider, store, testProvenance())
	keys := func(result Result) []string {
		values := []string{}
		for _, link := range result.Links {
			values = append(values, link.RepositoryKey+"#"+link.ExternalID)
		}
		return values
	}
	if all := service.List(context.Background(), ListTarget{ProjectSelector: "sample"}); !reflect.DeepEqual(keys(all), []string{"api#9", "api#10", "main#1"}) {
		t.Fatalf("unfiltered order = %v", keys(all))
	}
	if filtered := service.List(context.Background(), ListTarget{ProjectSelector: "sample", RepositoryKey: "main"}); !reflect.DeepEqual(keys(filtered), []string{"main#1"}) {
		t.Fatalf("filtered = %v", keys(filtered))
	}
	if filtered := service.List(context.Background(), ListTarget{ProjectSelector: "sample", RepositoryKey: "api"}); !reflect.DeepEqual(keys(filtered), []string{"api#9", "api#10"}) {
		t.Fatalf("filtered api = %v", keys(filtered))
	}
	if providerCalls(provider) != 0 {
		t.Fatal("list called the Provider")
	}
}

func TestUpdatePreviewsCurrentProviderDocumentAndRequiresReviewedAuthority(t *testing.T) {
	provider, store := &fakeCapability{}, newFakeStore()
	seedLink(store, "main", "owner/repo", "7", OpenState)
	service := testService(provider, store)
	input := UpdateInput{Target: lifecycleTarget(), Selector: "7", Scope: SectionInput{Supplied: "Revised scope"}}
	preview := service.Update(context.Background(), input, "", false)
	change := preview.Change
	if preview.Status != completion.Success || preview.Category != "work_item_update_ready" || change == nil || change.Document == nil {
		t.Fatalf("preview = %#v", preview)
	}
	if change.Document.Title != "" || change.Document.Body != "problem: Original problem\nscope: Revised scope\n" || !reflect.DeepEqual(change.Fields, []string{"scope"}) || !reflect.DeepEqual(change.Effects, []string{"update_provider_work_item_document"}) || change.CurrentDocument == "" {
		t.Fatalf("preview change = %#v document=%#v", change, change.Document)
	}
	if again := service.Update(context.Background(), input, "", false); again.Change.Digest != change.Digest {
		t.Fatal("preview digest is not deterministic")
	}
	for name, attempt := range map[string]Result{
		"single-step authority":    service.Update(context.Background(), input, "", true),
		"digest without authority": service.Update(context.Background(), input, change.Digest, false),
		"stale digest":             service.Update(context.Background(), input, strings.Repeat("0", 64), true),
	} {
		if attempt.Status != completion.DeniedAuthority || attempt.Category != "external_authority_denied" || attempt.Change == nil || attempt.Change.Digest != change.Digest {
			t.Fatalf("%s = %#v", name, attempt)
		}
	}
	if len(provider.mutations) != 0 {
		t.Fatalf("denied update mutated the Provider: %v", provider.mutations)
	}
	updated := service.Update(context.Background(), input, change.Digest, true)
	if updated.Status != completion.Success || updated.Category != "work_item_updated" || !reflect.DeepEqual(provider.mutations, []string{"update:|problem: Original problem\nscope: Revised scope\n"}) || store.saves != 0 {
		t.Fatalf("update = %#v mutations=%q saves=%d", updated, provider.mutations, store.saves)
	}
	// Replaying the now-applied change is a deterministic no-op.
	replayed := service.Update(context.Background(), input, change.Digest, true)
	if replayed.Status != completion.Success || replayed.Category != "work_item_unchanged" || len(replayed.Change.Effects) != 0 || len(provider.mutations) != 1 {
		t.Fatalf("replay = %#v mutations=%q", replayed, provider.mutations)
	}
	titled := service.Update(context.Background(), UpdateInput{Target: lifecycleTarget(), Selector: "7", Title: "Clearer title"}, "", false)
	if titled.Change == nil || titled.Change.Document.Title != "Clearer title" || titled.Change.Document.Body != "" || !reflect.DeepEqual(titled.Change.Fields, []string{"title"}) {
		t.Fatalf("title preview = %#v", titled.Change)
	}
}

func TestUpdateIsStaleWhenTheProviderDocumentChangesAfterReview(t *testing.T) {
	provider, store := &fakeCapability{}, newFakeStore()
	seedLink(store, "main", "owner/repo", "7", OpenState)
	service := testService(provider, store)
	input := UpdateInput{Target: lifecycleTarget(), Selector: "7", Title: "Reviewed title"}
	preview := service.Update(context.Background(), input, "", false)
	provider.body = "problem: Edited by someone else\nscope: Original scope\n"
	stale := service.Update(context.Background(), input, preview.Change.Digest, true)
	if stale.Status != completion.DeniedAuthority || stale.Category != "external_authority_denied" || len(provider.mutations) != 0 {
		t.Fatalf("stale update = %#v mutations=%v", stale, provider.mutations)
	}
}

func TestUpdateRejectsInvalidEmptyAndUnrecognizedInputWithoutMutation(t *testing.T) {
	provider, store := &fakeCapability{body: "free-form Issue body\n"}, newFakeStore()
	seedLink(store, "main", "owner/repo", "7", OpenState)
	service := testService(provider, store)
	for name, test := range map[string]struct {
		input    UpdateInput
		category string
	}{
		"empty":           {UpdateInput{}, "work_item_update_empty"},
		"multiline title": {UpdateInput{Title: "two\nlines"}, "invalid_work_item_title"},
		"secret title":    {UpdateInput{Title: "token=SYNTHETIC_REJECTED"}, "secret_rejected"},
		"secret section":  {UpdateInput{Scope: SectionInput{Supplied: "password: SYNTHETIC"}}, "secret_rejected"},
		"both authorship": {UpdateInput{Scope: SectionInput{Supplied: "a", Elaborated: "b"}}, "invalid_draft_input"},
		"unrecognized":    {UpdateInput{Scope: SectionInput{Supplied: "Revised scope"}}, "work_item_document_unrecognized"},
	} {
		test.input.Target, test.input.Selector = lifecycleTarget(), "7"
		if got := service.Update(context.Background(), test.input, strings.Repeat("0", 64), true); got.Category != test.category || got.Status == completion.Success {
			t.Fatalf("%s = %#v", name, got)
		}
	}
	if len(provider.mutations) != 0 {
		t.Fatalf("invalid update mutated the Provider: %v", provider.mutations)
	}
}

func TestCloseAndReopenWhenAlreadyThereAreNoOpsWithoutMutation(t *testing.T) {
	for _, test := range []struct {
		state, category string
		run             func(Service) Result
	}{
		{ClosedState, "work_item_already_closed", func(s Service) Result { return s.Close(context.Background(), lifecycleTarget(), "7", "any", true) }},
		{ClosedState, "work_item_already_closed", func(s Service) Result { return s.Complete(context.Background(), lifecycleTarget(), "7", "", false) }},
		{OpenState, "work_item_already_open", func(s Service) Result { return s.Reopen(context.Background(), lifecycleTarget(), "7", "any", true) }},
	} {
		provider, store := &fakeCapability{state: test.state}, newFakeStore()
		seedLink(store, "main", "owner/repo", "7", test.state)
		got := test.run(testService(provider, store))
		if got.Status != completion.Success || got.Category != test.category || got.Change == nil || len(got.Change.Effects) != 0 || len(provider.mutations) != 0 || store.saves != 0 {
			t.Fatalf("%s no-op = %#v mutations=%v saves=%d", test.category, got, provider.mutations, store.saves)
		}
	}
}

func TestCloseAndReopenRepairStaleLocalLinkWithoutProviderMutation(t *testing.T) {
	for _, test := range []struct {
		name, providerState, localState, category string
		run                                       func(Service, string, bool) Result
	}{
		{"close", ClosedState, OpenState, "work_item_closed", func(s Service, digest string, authorized bool) Result {
			return s.Close(context.Background(), lifecycleTarget(), "7", digest, authorized)
		}},
		{"reopen", OpenState, ClosedState, "work_item_reopened", func(s Service, digest string, authorized bool) Result {
			return s.Reopen(context.Background(), lifecycleTarget(), "7", digest, authorized)
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			provider, store := &fakeCapability{state: test.providerState}, newFakeStore()
			seedLink(store, "main", "owner/repo", "7", test.localState)
			service := testService(provider, store)
			preview := test.run(service, "", false)
			if preview.Category != "work_item_"+test.name+"_ready" || preview.Change == nil || !reflect.DeepEqual(preview.Change.Effects, []string{"update_local_work_item_link"}) || store.saves != 0 {
				t.Fatalf("repair preview = %#v saves=%d", preview, store.saves)
			}
			if denied := test.run(service, "", true); denied.Category != "external_authority_denied" || store.saves != 0 {
				t.Fatalf("unreviewed repair = %#v saves=%d", denied, store.saves)
			}
			if denied := test.run(service, preview.Change.Digest, false); denied.Category != "external_authority_denied" || store.saves != 0 {
				t.Fatalf("unauthorized repair = %#v saves=%d", denied, store.saves)
			}
			if denied := test.run(service, "stale", true); denied.Category != "external_authority_denied" || store.saves != 0 {
				t.Fatalf("stale repair = %#v saves=%d", denied, store.saves)
			}
			repaired := test.run(service, preview.Change.Digest, true)
			if repaired.Status != completion.Success || repaired.Category != test.category || repaired.Link.State != test.providerState || store.links["github:owner/repo:7"].State != test.providerState || store.saves != 1 || len(provider.mutations) != 0 {
				t.Fatalf("repair = %#v saves=%d mutations=%v", repaired, store.saves, provider.mutations)
			}
			if again := test.run(service, "", false); again.Status != completion.Success || len(again.Change.Effects) != 0 || store.saves != 1 {
				t.Fatalf("converged = %#v saves=%d", again, store.saves)
			}
		})
	}
}

func TestCloseAndReopenRepairLocalConflictDoesNotMutateProvider(t *testing.T) {
	for _, test := range []struct {
		name, providerState, localState string
		run                             func(Service, string) Result
	}{
		{"close", ClosedState, OpenState, func(s Service, digest string) Result {
			return s.Close(context.Background(), lifecycleTarget(), "7", digest, true)
		}},
		{"reopen", OpenState, ClosedState, func(s Service, digest string) Result {
			return s.Reopen(context.Background(), lifecycleTarget(), "7", digest, true)
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			provider, store := &fakeCapability{state: test.providerState}, newFakeStore()
			seedLink(store, "main", "owner/repo", "7", test.localState)
			service := testService(provider, store)
			preview := test.run(service, "")
			store.saveErr = ErrConflict
			got := test.run(service, preview.Change.Digest)
			if got.Status != completion.Partial || got.Category != "provider_confirmed_local_conflict" || got.Link.State != test.providerState || store.links["github:owner/repo:7"].State != test.localState || len(provider.mutations) != 0 {
				t.Fatalf("repair conflict = %#v mutations=%v", got, provider.mutations)
			}
		})
	}
}

func TestCloseAndReopenRepairRejectsChangedLocalRevision(t *testing.T) {
	for _, test := range []struct {
		name, providerState, localState string
		run                             func(Service, string) Result
	}{
		{"close", ClosedState, OpenState, func(s Service, digest string) Result {
			return s.Close(context.Background(), lifecycleTarget(), "7", digest, true)
		}},
		{"reopen", OpenState, ClosedState, func(s Service, digest string) Result {
			return s.Reopen(context.Background(), lifecycleTarget(), "7", digest, true)
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			provider, store := &fakeCapability{state: test.providerState}, newFakeStore()
			seedLink(store, "main", "owner/repo", "7", test.localState)
			service := testService(provider, store)
			preview := test.run(service, "")
			link := store.links["github:owner/repo:7"]
			link.Revision = [32]byte{4}
			store.links["github:owner/repo:7"] = link
			if stale := test.run(service, preview.Change.Digest); stale.Category != "external_authority_denied" || store.saves != 0 || len(provider.mutations) != 0 {
				t.Fatalf("stale repair = %#v saves=%d mutations=%v", stale, store.saves, provider.mutations)
			}
		})
	}
}

func TestCloseAndReopenRequireReviewedAuthorityThenRecordLocalState(t *testing.T) {
	provider, store := &fakeCapability{}, newFakeStore()
	seedLink(store, "main", "owner/repo", "7", OpenState)
	service := testService(provider, store)
	preview := service.Close(context.Background(), lifecycleTarget(), "7", "", false)
	if preview.Category != "work_item_close_ready" || preview.Change == nil || preview.Change.DesiredState != ClosedState || !reflect.DeepEqual(preview.Change.Effects, []string{"set_provider_work_item_state", "update_local_work_item_link"}) {
		t.Fatalf("close preview = %#v", preview)
	}
	if denied := service.Close(context.Background(), lifecycleTarget(), "7", "", true); denied.Category != "external_authority_denied" || len(provider.mutations) != 0 {
		t.Fatalf("single-step close = %#v", denied)
	}
	closed := service.Close(context.Background(), lifecycleTarget(), "7", preview.Change.Digest, true)
	if closed.Status != completion.Success || closed.Category != "work_item_closed" || closed.Link.State != ClosedState || store.links["github:owner/repo:7"].State != ClosedState || !reflect.DeepEqual(provider.mutations, []string{"state:CLOSED"}) {
		t.Fatalf("close = %#v mutations=%v", closed, provider.mutations)
	}
	reopenPreview := service.Reopen(context.Background(), lifecycleTarget(), "7", "", false)
	if reopenPreview.Category != "work_item_reopen_ready" || reopenPreview.Change.DesiredState != OpenState {
		t.Fatalf("reopen preview = %#v", reopenPreview)
	}
	reopened := service.Reopen(context.Background(), lifecycleTarget(), "7", reopenPreview.Change.Digest, true)
	if reopened.Category != "work_item_reopened" || store.links["github:owner/repo:7"].State != OpenState || !reflect.DeepEqual(provider.mutations, []string{"state:CLOSED", "state:OPEN"}) {
		t.Fatalf("reopen = %#v mutations=%v", reopened, provider.mutations)
	}
}

func TestCloseIsStaleWhenTheLocalLinkChangesAfterReview(t *testing.T) {
	provider, store := &fakeCapability{}, newFakeStore()
	seedLink(store, "main", "owner/repo", "7", OpenState)
	service := testService(provider, store)
	preview := service.Close(context.Background(), lifecycleTarget(), "7", "", false)
	link := store.links["github:owner/repo:7"]
	link.Revision = [32]byte{4}
	store.links["github:owner/repo:7"] = link
	if stale := service.Close(context.Background(), lifecycleTarget(), "7", preview.Change.Digest, true); stale.Category != "external_authority_denied" || len(provider.mutations) != 0 {
		t.Fatalf("stale close = %#v", stale)
	}
}

func TestCommentAndCompleteRequireTheReviewedDigest(t *testing.T) {
	provider, store := &fakeCapability{}, newFakeStore()
	seedLink(store, "main", "owner/repo", "7", OpenState)
	service := testService(provider, store)
	single := service.Comment(context.Background(), lifecycleTarget(), "7", "Evidence", "", true)
	if single.Status != completion.DeniedAuthority || single.Category != "external_authority_denied" || single.Change == nil || single.Change.Comment != "Evidence" || provider.reads != 0 {
		t.Fatalf("single-step comment = %#v", single)
	}
	commented := service.Comment(context.Background(), lifecycleTarget(), "7", "Evidence", single.Change.Digest, true)
	if commented.Category != "work_item_commented" || !reflect.DeepEqual(provider.mutations, []string{"comment:Evidence"}) {
		t.Fatalf("comment = %#v mutations=%v", commented, provider.mutations)
	}
	if other := service.Comment(context.Background(), lifecycleTarget(), "7", "Different", single.Change.Digest, true); other.Category != "external_authority_denied" {
		t.Fatalf("comment digest is not bound to the message: %#v", other)
	}
	for _, message := range []string{"", " padded ", "token=SYNTHETIC_REJECTED"} {
		if got := service.Comment(context.Background(), lifecycleTarget(), "7", message, single.Change.Digest, true); got.Status != completion.ValidationFailure {
			t.Fatalf("comment %q = %#v", message, got)
		}
	}
	completeSingle := service.Complete(context.Background(), lifecycleTarget(), "7", "", true)
	if completeSingle.Category != "external_authority_denied" || completeSingle.Change == nil || completeSingle.Change.Operation != ChangeClose {
		t.Fatalf("single-step complete = %#v", completeSingle)
	}
	completed := service.Complete(context.Background(), lifecycleTarget(), "7", completeSingle.Change.Digest, true)
	if completed.Category != "work_item_closed" || !reflect.DeepEqual(provider.mutations, []string{"comment:Evidence", "state:CLOSED"}) {
		t.Fatalf("complete = %#v mutations=%v", completed, provider.mutations)
	}
}

func TestLifecycleTargetFailuresHappenBeforeAnyProviderCall(t *testing.T) {
	provider, store := &fakeCapability{}, newFakeStore()
	service := testService(provider, store)
	if missing := service.Close(context.Background(), lifecycleTarget(), "7", "", false); missing.Category != "work_item_not_found" {
		t.Fatalf("missing = %#v", missing)
	}
	seedLink(store, "main", "owner/repo", "7", OpenState)
	seedLink(store, "main", "owner/other", "7", OpenState)
	unqualified := Target{ProjectSelector: "sample", RepositoryKey: "main"}
	if ambiguous := service.Update(context.Background(), UpdateInput{Target: unqualified, Selector: "7", Title: "x"}, "", false); ambiguous.Category != "work_item_ambiguous" {
		t.Fatalf("ambiguous = %#v", ambiguous)
	}
	mismatch := New(fakeResolver{provider: "gitlab"}, provider, provider, store, testProvenance())
	for name, got := range map[string]Result{
		"update":  mismatch.Update(context.Background(), UpdateInput{Target: lifecycleTarget(), Selector: "7", Title: "x"}, "", false),
		"close":   mismatch.Close(context.Background(), lifecycleTarget(), "7", "", false),
		"reopen":  mismatch.Reopen(context.Background(), lifecycleTarget(), "7", "", false),
		"comment": mismatch.Comment(context.Background(), lifecycleTarget(), "7", "x", "", false),
	} {
		if got.Category != "work_item_capability_unavailable" {
			t.Fatalf("%s provider mismatch = %#v", name, got)
		}
	}
	if without := New(fakeResolver{}, provider, nil, store, testProvenance()).Close(context.Background(), lifecycleTarget(), "7", "", false); without.Category != "work_item_capability_unavailable" {
		t.Fatalf("missing lifecycle = %#v", without)
	}
	cancelled, cancel := context.WithCancel(context.Background())
	preview := service.Close(context.Background(), lifecycleTarget(), "7", "", false)
	cancel()
	if got := service.Close(cancelled, lifecycleTarget(), "7", preview.Change.Digest, true); got.Status != completion.Interrupted {
		t.Fatalf("cancelled close = %#v", got)
	}
	if providerCalls(provider) != 2 || len(provider.mutations) != 0 {
		t.Fatalf("target failures reached the Provider: %+v", provider)
	}
}

func TestTransientProviderFailureIsRetryableWithoutLocalWrite(t *testing.T) {
	provider, store := &fakeCapability{}, newFakeStore()
	seedLink(store, "main", "owner/repo", "7", OpenState)
	service := testService(provider, store)
	preview := service.Close(context.Background(), lifecycleTarget(), "7", "", false)
	provider.mutationErr = &ProviderError{Kind: ProviderUnavailable, Retryable: true}
	got := service.Close(context.Background(), lifecycleTarget(), "7", preview.Change.Digest, true)
	if got.Status != completion.RetryableFailure || got.Category != "provider_unavailable" || store.saves != 0 || store.links["github:owner/repo:7"].State != OpenState {
		t.Fatalf("transient close = %#v saves=%d", got, store.saves)
	}
	provider.mutationErr, provider.readErr = nil, &ProviderError{Kind: ProviderRateLimited, Retryable: true, EffectNotCommitted: true}
	if read := service.Reopen(context.Background(), lifecycleTarget(), "7", "", false); read.Status != completion.RetryableFailure || read.Category != "provider_rate_limited" {
		t.Fatalf("transient preview read = %#v", read)
	}
	provider.readErr = nil
	comment := service.Comment(context.Background(), lifecycleTarget(), "7", "Evidence", "", false)
	provider.mutationErr = &ProviderError{Kind: ProviderUnavailable, Retryable: true}
	if got := service.Comment(context.Background(), lifecycleTarget(), "7", "Evidence", comment.Change.Digest, true); got.Status != completion.Failure || got.Category != "provider_comment_ambiguous" {
		t.Fatalf("uncertain comment = %#v", got)
	}
	provider.mutationErr = &ProviderError{Kind: ProviderRateLimited, Retryable: true, EffectNotCommitted: true}
	if got := service.Comment(context.Background(), lifecycleTarget(), "7", "Evidence", comment.Change.Digest, true); got.Status != completion.RetryableFailure {
		t.Fatalf("uncommitted comment = %#v", got)
	}
}

func TestProviderConfirmedCloseWithLocalFailureIsTruthfulPartial(t *testing.T) {
	for _, test := range []struct {
		name     string
		saveErr  error
		category string
	}{
		{"conflict", ErrConflict, "provider_confirmed_local_conflict"},
		{"recovery", ErrRecoveryRequired, "provider_confirmed_local_recovery_required"},
		{"write", errors.New("controlled write failure"), "provider_confirmed_local_failed"},
	} {
		provider, store := &fakeCapability{}, newFakeStore()
		seedLink(store, "main", "owner/repo", "7", OpenState)
		service := testService(provider, store)
		preview := service.Close(context.Background(), lifecycleTarget(), "7", "", false)
		store.saveErr = test.saveErr
		got := service.Close(context.Background(), lifecycleTarget(), "7", preview.Change.Digest, true)
		if got.Status != completion.Partial || got.Category != test.category || got.Link.State != ClosedState || got.Change == nil || !reflect.DeepEqual(provider.mutations, []string{"state:CLOSED"}) {
			t.Fatalf("%s partial = %#v mutations=%v", test.name, got, provider.mutations)
		}
		// The Provider effect is never repeated. A fresh reviewed preview may
		// repair the local link after the write fault is resolved.
		repair := service.Close(context.Background(), lifecycleTarget(), "7", "", false)
		if repair.Category != "work_item_close_ready" || repair.Change == nil || !reflect.DeepEqual(repair.Change.Effects, []string{"update_local_work_item_link"}) {
			t.Fatalf("%s repair preview = %#v", test.name, repair)
		}
		if stale := service.Close(context.Background(), lifecycleTarget(), "7", preview.Change.Digest, true); stale.Category != "external_authority_denied" || len(provider.mutations) != 1 {
			t.Fatalf("%s stale retry = %#v mutations=%v", test.name, stale, provider.mutations)
		}
		store.saveErr = nil
		if again := service.Close(context.Background(), lifecycleTarget(), "7", repair.Change.Digest, true); again.Category != "work_item_closed" || store.links["github:owner/repo:7"].State != ClosedState || len(provider.mutations) != 1 {
			t.Fatalf("%s repair = %#v mutations=%v", test.name, again, provider.mutations)
		}
	}
}

func TestProviderConfirmedReopenWithLocalFailureCanBeRepaired(t *testing.T) {
	provider, store := &fakeCapability{state: ClosedState}, newFakeStore()
	seedLink(store, "main", "owner/repo", "7", ClosedState)
	service := testService(provider, store)
	preview := service.Reopen(context.Background(), lifecycleTarget(), "7", "", false)
	store.saveErr = errors.New("controlled write failure")
	partial := service.Reopen(context.Background(), lifecycleTarget(), "7", preview.Change.Digest, true)
	if partial.Status != completion.Partial || partial.Category != "provider_confirmed_local_failed" || partial.Link.State != OpenState || store.links["github:owner/repo:7"].State != ClosedState || !reflect.DeepEqual(provider.mutations, []string{"state:OPEN"}) {
		t.Fatalf("reopen partial = %#v mutations=%v", partial, provider.mutations)
	}
	repair := service.Reopen(context.Background(), lifecycleTarget(), "7", "", false)
	if repair.Category != "work_item_reopen_ready" || repair.Change == nil || !reflect.DeepEqual(repair.Change.Effects, []string{"update_local_work_item_link"}) {
		t.Fatalf("reopen repair preview = %#v", repair)
	}
	store.saveErr = nil
	reopened := service.Reopen(context.Background(), lifecycleTarget(), "7", repair.Change.Digest, true)
	if reopened.Status != completion.Success || reopened.Category != "work_item_reopened" || store.links["github:owner/repo:7"].State != OpenState || !reflect.DeepEqual(provider.mutations, []string{"state:OPEN"}) {
		t.Fatalf("reopen repair = %#v mutations=%v", reopened, provider.mutations)
	}
}
