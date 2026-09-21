package workitem

import (
	"context"
	"errors"
	"testing"
)

func TestCreateRequiresAuthorityBeforeProviderMutation(t *testing.T) {
	provider := &fakeProvider{}
	service := New(fakeResolver{}, fakeLocator{}, provider, newFakeStore())
	result := service.Create(context.Background(), Target{"sample", "main"}, "Title", "Body", false)
	if result.Category != "external_mutation_denied" || provider.creates != 0 {
		t.Fatalf("denied create = %#v, calls=%d", result, provider.creates)
	}
}

func TestCreateSelectCommentAndComplete(t *testing.T) {
	provider := &fakeProvider{}
	store := newFakeStore()
	service := New(fakeResolver{}, fakeLocator{}, provider, store)
	target := Target{"sample", "main"}
	created := service.Create(context.Background(), target, "Title", "Body", true)
	if created.Status != Succeeded || created.Link.Number != 7 {
		t.Fatalf("create = %#v", created)
	}
	selected := service.Select(context.Background(), target, 7)
	if selected.Status != Succeeded {
		t.Fatalf("select = %#v", selected)
	}
	commented := service.Comment(context.Background(), target, 7, "Evidence", true)
	if commented.Category != "work_item_commented" || provider.comments != 1 {
		t.Fatalf("comment = %#v, calls=%d", commented, provider.comments)
	}
	completed := service.Complete(context.Background(), target, 7, true)
	if completed.Category != "work_item_completed" || completed.Link.State != "CLOSED" {
		t.Fatalf("complete = %#v", completed)
	}
}

func TestSelectReconcilesExistingLinkWithObservedRevision(t *testing.T) {
	provider := &fakeProvider{readState: "CLOSED"}
	store := newFakeStore()
	expected := [32]byte{1}
	store.links["main"] = Link{
		ProjectID:          "123e4567-e89b-42d3-a456-426614174000",
		RepositoryKey:      "main",
		ProviderRepository: "owner/repo",
		Number:             7,
		URL:                "https://github.com/owner/repo/issues/7",
		State:              "OPEN",
		Revision:           expected,
	}
	result := New(fakeResolver{}, fakeLocator{}, provider, store).Select(context.Background(), Target{"sample", "main"}, 7)
	if result.Status != Succeeded || result.Link.State != "CLOSED" || result.Link.Revision != expected {
		t.Fatalf("reconciled select = %#v", result)
	}
	if store.saved.Revision != expected {
		t.Fatalf("save revision = %x want %x", store.saved.Revision, expected)
	}
}

func TestCompleteReportsProviderCommitWhenLocalSaveFails(t *testing.T) {
	provider := &fakeProvider{}
	store := newFakeStore()
	service := New(fakeResolver{}, fakeLocator{}, provider, store)
	target := Target{"sample", "main"}
	if result := service.Select(context.Background(), target, 7); result.Status != Succeeded {
		t.Fatal(result)
	}
	store.failSave = true
	result := service.Complete(context.Background(), target, 7, true)
	want := Link{
		ProjectID:          "123e4567-e89b-42d3-a456-426614174000",
		RepositoryKey:      "main",
		ProviderRepository: "owner/repo",
		Number:             7,
		URL:                "https://github.com/owner/repo/issues/7",
		State:              "CLOSED",
	}
	if result.Category != "provider_committed_local_failed" || result.Link != want || provider.closes != 1 {
		t.Fatalf("complete = %#v, closes=%d", result, provider.closes)
	}
}

func TestCreateReportsProviderCommitWhenLocalSaveFails(t *testing.T) {
	provider := &fakeProvider{}
	store := newFakeStore()
	store.failSave = true
	service := New(fakeResolver{}, fakeLocator{}, provider, store)
	result := service.Create(context.Background(), Target{"sample", "main"}, "Title", "Body", true)
	want := Link{
		ProjectID:          "123e4567-e89b-42d3-a456-426614174000",
		RepositoryKey:      "main",
		ProviderRepository: "owner/repo",
		Number:             7,
		URL:                "https://github.com/owner/repo/issues/7",
		State:              "OPEN",
	}
	if result.Category != "provider_committed_local_failed" || result.Link != want || provider.creates != 1 {
		t.Fatalf("create = %#v, calls=%d", result, provider.creates)
	}
}

func TestRecoveryRequiredRemainsDistinctFromMissingAndGenericFailure(t *testing.T) {
	provider := &fakeProvider{}
	store := newFakeStore()
	store.recovery = true
	service := New(fakeResolver{}, fakeLocator{}, provider, store)
	created := service.Create(context.Background(), Target{"sample", "main"}, "Title", "Body", true)
	if created.Category != "provider_committed_local_recovery_required" || provider.creates != 1 {
		t.Fatalf("create = %#v", created)
	}
	shown := service.Show(context.Background(), Target{"sample", "main"}, 7)
	if shown.Category != "recovery_required" {
		t.Fatalf("show = %#v", shown)
	}
}

type fakeResolver struct{}

func (fakeResolver) Resolve(context.Context, string) (Project, string) {
	return Project{ID: "123e4567-e89b-42d3-a456-426614174000", Repositories: []Repository{{Key: "main", Path: "/repo"}}}, ""
}

type fakeLocator struct{}

func (fakeLocator) GitHubRepository(context.Context, string) (string, error) {
	return "owner/repo", nil
}

type fakeProvider struct {
	creates, comments, closes int
	readState                 string
}

func (p *fakeProvider) Create(context.Context, string, string, string) (External, error) {
	p.creates++
	return External{7, "https://github.com/owner/repo/issues/7", "OPEN"}, nil
}
func (p *fakeProvider) Read(context.Context, string, int) (External, error) {
	state := p.readState
	if state == "" {
		state = "OPEN"
	}
	return External{7, "https://github.com/owner/repo/issues/7", state}, nil
}
func (p *fakeProvider) Comment(context.Context, string, int, string) error { p.comments++; return nil }
func (p *fakeProvider) Close(context.Context, string, int) (External, error) {
	p.closes++
	return External{7, "https://github.com/owner/repo/issues/7", "CLOSED"}, nil
}

type fakeStore struct {
	links    map[string]Link
	saved    Link
	failSave bool
	recovery bool
}

func newFakeStore() *fakeStore { return &fakeStore{links: map[string]Link{}} }
func (s *fakeStore) Save(_ context.Context, link Link) error {
	if s.recovery {
		return ErrRecoveryRequired
	}
	if s.failSave {
		return errors.New("write failed")
	}
	s.saved = link
	s.links[link.RepositoryKey] = link
	return nil
}
func (s *fakeStore) Load(context.Context, string, string, int) (Link, error) {
	if s.recovery {
		return Link{}, ErrRecoveryRequired
	}
	link, ok := s.links["main"]
	if !ok {
		return Link{}, ErrNotFound
	}
	return link, nil
}
