package workitem

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"sort"
	"strconv"
	"strings"

	"github.com/rgomids/axiom/internal/completion"
	"github.com/rgomids/axiom/internal/provenance"
)

// Issue #230 (I230-T06) Work Item lifecycle: list, update, comment, close and
// reopen. Every external mutation is previewed from current Provider or local
// truth, bound to a digest and executed only with that exact reviewed digest
// and explicit external authority. There is no delete and no arbitrary
// Provider field update; type and classification stay create-time contracts.

var (
	// ErrDocumentUnrecognized means the current Provider document does not have
	// the Axiom structure a section revision needs; nothing is guessed.
	ErrDocumentUnrecognized = errors.New("work item document unrecognized")
	ErrDocumentTooLarge     = errors.New("work item document exceeds provider limit")
)

const (
	maxTitleBytes     = 256
	maxListedWorkItem = 256

	OpenState   = "OPEN"
	ClosedState = "CLOSED"
)

// Lifecycle is the reviewed Provider lifecycle capability. Provider formats
// and transport stay behind it; authority, no-op and outcome semantics stay
// here.
type Lifecycle interface {
	// ReadDocument reads the current Provider title and body (R-X).
	ReadDocument(context.Context, string, string) (External, ProviderDocument, error)
	// ReviseDocument re-renders only the revised fields of the current
	// document and preserves everything else byte-for-byte.
	ReviseDocument(ProviderDocument, DocumentRevision) (ProviderDocument, error)
	// UpdateDocument changes only the non-empty fields of the change.
	UpdateDocument(context.Context, string, string, DocumentChange) (External, error)
	// SetState changes only the Provider open/closed state.
	SetState(context.Context, string, string, string) (External, error)
	Comment(context.Context, string, string, string) error
}

// DocumentRevision names only the explicitly supplied fields of an update.
type DocumentRevision struct {
	Title    string         `json:"title,omitempty"`
	Sections []DraftSection `json:"sections,omitempty"`
}

// DocumentChange is the exact Provider field change; an empty field is not
// sent and therefore not changed.
type DocumentChange struct {
	Title string `json:"title,omitempty"`
	Body  string `json:"body,omitempty"`
}

type ChangeOperation string

const (
	ChangeUpdate  ChangeOperation = "update"
	ChangeComment ChangeOperation = "comment"
	ChangeClose   ChangeOperation = "close"
	ChangeReopen  ChangeOperation = "reopen"
)

// ChangePreview is the reviewed, digest-bound description of one Work Item
// lifecycle effect.
type ChangePreview struct {
	Operation ChangeOperation `json:"operation"`
	Target    DraftTarget     `json:"target"`
	// External is current Provider truth for update, close and reopen, and the
	// local link for comment, which reads no Provider state.
	External          External          `json:"external"`
	Fields            []string          `json:"fields,omitempty"`
	CurrentDocument   string            `json:"currentDocument,omitempty"`
	Document          *DocumentChange   `json:"document,omitempty"`
	Comment           string            `json:"comment,omitempty"`
	DesiredState      string            `json:"desiredState,omitempty"`
	Effects           []string          `json:"effects"`
	ExpectedRevisions ExpectedRevisions `json:"expectedRevisions"`
	Digest            string            `json:"digest"`
}

type ListTarget struct {
	ProjectSelector, RepositoryKey string
}

type UpdateInput struct {
	Target                                  Target
	Selector                                string
	Title                                   string
	Problem, DesiredOutcome, Context, Scope SectionInput
	Constraints, NonGoals, Acceptance       SectionInput
}

// List enumerates Axiom-linked Work Items from local links only. It never
// calls the Provider, so unlinked Provider Issues are never listed.
func (s Service) List(ctx context.Context, target ListTarget) Result {
	if s.resolver == nil || s.store == nil || !s.source.Valid() || target.ProjectSelector == "" {
		return result(completion.ValidationFailure, "invalid_work_item_input")
	}
	project, category := s.resolver.Resolve(ctx, target.ProjectSelector)
	if category != "" {
		return result(completion.ValidationFailure, category)
	}
	attached := make(map[string]bool, len(project.Repositories))
	for _, repository := range project.Repositories {
		attached[repository.Key] = true
	}
	if target.RepositoryKey != "" && !attached[target.RepositoryKey] {
		return result(completion.ValidationFailure, "repository_not_configured")
	}
	stored, err := s.store.List(ctx, project.ID)
	if err != nil {
		switch {
		case errors.Is(err, ErrRecoveryRequired):
			return result(completion.Failure, "recovery_required")
		case errors.Is(err, ErrConflict):
			return result(completion.Failure, "local_work_item_conflict")
		}
		return result(completion.Failure, "local_work_item_read_failed")
	}
	links := make([]Link, 0, len(stored))
	for _, link := range stored {
		if link.ProjectID != project.ID {
			return result(completion.Failure, "local_work_item_read_failed")
		}
		// Links of a detached Repository key stay preserved but unreachable
		// until the same key is attached again (#230 F-03).
		if !attached[link.RepositoryKey] || target.RepositoryKey != "" && link.RepositoryKey != target.RepositoryKey {
			continue
		}
		links = append(links, link)
	}
	if len(links) > maxListedWorkItem {
		return result(completion.Failure, "work_item_list_exceeds_limit")
	}
	sort.Slice(links, func(i, j int) bool { return linkLess(links[i], links[j]) })
	return Result{Status: completion.Success, Category: "work_items_listed", Links: links}
}

func linkLess(left, right Link) bool {
	if left.RepositoryKey != right.RepositoryKey {
		return left.RepositoryKey < right.RepositoryKey
	}
	if left.Provider != right.Provider {
		return left.Provider < right.Provider
	}
	if left.Resource != right.Resource {
		return left.Resource < right.Resource
	}
	leftNumber, leftErr := strconv.Atoi(left.ExternalID)
	rightNumber, rightErr := strconv.Atoi(right.ExternalID)
	if leftErr == nil && rightErr == nil && leftNumber != rightNumber {
		return leftNumber < rightNumber
	}
	return left.ExternalID < right.ExternalID
}

// Update changes only the title and the Axiom-authored sections of one
// linked Work Item. The preview reads the current Provider document.
func (s Service) Update(ctx context.Context, input UpdateInput, previewDigest string, authorized bool) Result {
	revision, category := normalizeRevision(input)
	if category != "" {
		return result(completion.ValidationFailure, category)
	}
	link, failure := s.lifecycleLink(ctx, input.Target, input.Selector)
	if failure.Category != "" {
		return failure
	}
	external, current, err := s.lifecycle.ReadDocument(ctx, link.Resource, link.ExternalID)
	if err != nil {
		return providerFailure(err, "provider_read_failed")
	}
	if !s.capability.ValidExternal(link.Resource, link.ExternalID, external) {
		return result(completion.Failure, "invalid_provider_response")
	}
	proposed, err := s.lifecycle.ReviseDocument(current, revision)
	if err != nil {
		if errors.Is(err, ErrDocumentTooLarge) {
			return result(completion.ValidationFailure, "draft_too_large")
		}
		return result(completion.Failure, "work_item_document_unrecognized")
	}
	change := DocumentChange{}
	if proposed.Title != current.Title {
		change.Title = proposed.Title
	}
	if proposed.Body != current.Body {
		change.Body = proposed.Body
	}
	preview := s.changePreview(ChangeUpdate, link, external)
	preview.Fields = revision.fields()
	preview.CurrentDocument = documentDigest(current)
	if change != (DocumentChange{}) {
		preview.Document = &change
		preview.Effects = []string{"update_provider_work_item_document"}
	}
	return s.applyChange(ctx, link, &preview, previewDigest, authorized, "work_item_unchanged", func() Result {
		updated, err := s.lifecycle.UpdateDocument(ctx, link.Resource, link.ExternalID, change)
		if err != nil {
			return providerFailure(err, "provider_mutation_failed")
		}
		if !s.capability.ValidExternal(link.Resource, link.ExternalID, updated) {
			return result(completion.Failure, "invalid_provider_response")
		}
		return Result{Status: completion.Success, Category: "work_item_updated", Link: link}
	})
}

// Comment adds one reviewed Provider comment. The preview is local: it names
// the linked Work Item and the exact message, and reads no Provider state.
func (s Service) Comment(ctx context.Context, target Target, selector, message, previewDigest string, authorized bool) Result {
	if !validDraftText(message) {
		return result(completion.ValidationFailure, "invalid_work_item_comment")
	}
	if looksSensitive(message) {
		return result(completion.ValidationFailure, "secret_rejected")
	}
	link, failure := s.lifecycleLink(ctx, target, selector)
	if failure.Category != "" {
		return failure
	}
	preview := s.changePreview(ChangeComment, link, External{ID: link.ExternalID, URL: link.URL, State: link.State})
	preview.Comment = message
	preview.Effects = []string{"create_provider_comment"}
	return s.applyChange(ctx, link, &preview, previewDigest, authorized, "", func() Result {
		if err := s.lifecycle.Comment(ctx, link.Resource, link.ExternalID, message); err != nil {
			var provider *ProviderError
			if errors.As(err, &provider) && provider.EffectNotCommitted {
				return providerFailure(err, "provider_mutation_failed")
			}
			// A comment is not idempotent: an uncertain outcome is never
			// reported as retry-safe.
			return result(completion.Failure, "provider_comment_ambiguous")
		}
		return Result{Status: completion.Success, Category: "work_item_commented", Link: link}
	})
}

// Close sets the Provider state to closed and then records it in the local
// link under its revision. It is not workflow progress or acceptance.
func (s Service) Close(ctx context.Context, target Target, selector, previewDigest string, authorized bool) Result {
	return s.setState(ctx, ChangeClose, target, selector, ClosedState, previewDigest, authorized)
}

// Complete is the historical CLI spelling of Close; it has no other meaning.
func (s Service) Complete(ctx context.Context, target Target, selector, previewDigest string, authorized bool) Result {
	return s.Close(ctx, target, selector, previewDigest, authorized)
}

func (s Service) Reopen(ctx context.Context, target Target, selector, previewDigest string, authorized bool) Result {
	return s.setState(ctx, ChangeReopen, target, selector, OpenState, previewDigest, authorized)
}

func (s Service) setState(ctx context.Context, operation ChangeOperation, target Target, selector, desired, previewDigest string, authorized bool) Result {
	link, failure := s.lifecycleLink(ctx, target, selector)
	if failure.Category != "" {
		return failure
	}
	external, err := s.capability.Read(ctx, link.Resource, link.ExternalID)
	if err != nil {
		return providerFailure(err, "provider_read_failed")
	}
	if !s.capability.ValidExternal(link.Resource, link.ExternalID, external) {
		return result(completion.Failure, "invalid_provider_response")
	}
	preview := s.changePreview(operation, link, external)
	preview.DesiredState = desired
	noop := "work_item_already_closed"
	if desired == OpenState {
		noop = "work_item_already_open"
	}
	if external.State != desired {
		preview.Effects = []string{"set_provider_work_item_state", "update_local_work_item_link"}
	}
	return s.applyChange(ctx, link, &preview, previewDigest, authorized, noop, func() Result {
		changed, err := s.lifecycle.SetState(ctx, link.Resource, link.ExternalID, desired)
		if err != nil {
			return providerFailure(err, "provider_mutation_failed")
		}
		if !s.capability.ValidExternal(link.Resource, link.ExternalID, changed) || changed.State != desired {
			return result(completion.Failure, "invalid_provider_response")
		}
		next := linkFrom(DraftTarget{ProjectID: link.ProjectID, RepositoryKey: link.RepositoryKey, Provider: link.Provider, Resource: link.Resource}, changed, link.Revision)
		persisted := s.persistLocal(ctx, next, true)
		if persisted.Status == completion.Success {
			persisted.Category = "work_item_closed"
			if desired == OpenState {
				persisted.Category = "work_item_reopened"
			}
		}
		return persisted
	})
}

// applyChange is the single reviewed-authority gate of every lifecycle
// mutation: a preview without effects is a deterministic no-op; otherwise
// only the exact reviewed digest with explicit external authority executes.
func (s Service) applyChange(ctx context.Context, link Link, preview *ChangePreview, previewDigest string, authorized bool, noop string, execute func() Result) Result {
	if preview.Effects == nil {
		preview.Effects = []string{}
	}
	preview.Digest = ""
	preview.Digest = digestValue(preview)
	if len(preview.Effects) == 0 {
		return Result{Status: completion.Success, Category: noop, Link: link, Change: preview}
	}
	if !authorized && previewDigest == "" {
		return Result{Status: completion.Success, Category: "work_item_" + string(preview.Operation) + "_ready", Link: link, Change: preview}
	}
	if !authorized || previewDigest != preview.Digest {
		return Result{Status: completion.DeniedAuthority, Category: "external_authority_denied", Link: link, Change: preview}
	}
	if ctx.Err() != nil {
		return Result{Status: completion.Interrupted, Category: "work_item_change_cancelled", Change: preview}
	}
	executed := execute()
	executed.Change = preview
	return executed
}

func (s Service) changePreview(operation ChangeOperation, link Link, external External) ChangePreview {
	return ChangePreview{
		Operation:         operation,
		Target:            DraftTarget{ProjectID: link.ProjectID, RepositoryKey: link.RepositoryKey, Provider: link.Provider, Resource: link.Resource},
		External:          external,
		ExpectedRevisions: ExpectedRevisions{Local: hex.EncodeToString(link.Revision[:])},
	}
}

// lifecycleLink resolves one linked Work Item through the same exact local
// selection as Show, after confirming the Project's Work Item Provider is the
// implemented one. Provider mismatch fails before any Provider call.
func (s Service) lifecycleLink(ctx context.Context, target Target, selector string) (Link, Result) {
	if s.resolver == nil || s.store == nil || !s.source.Valid() || target.ProjectSelector == "" || target.RepositoryKey == "" || selector == "" {
		return Link{}, result(completion.ValidationFailure, "invalid_work_item_input")
	}
	project, category := s.resolver.Resolve(ctx, target.ProjectSelector)
	if category != "" {
		return Link{}, result(completion.ValidationFailure, category)
	}
	if s.capability == nil || s.lifecycle == nil || project.Provider != s.capability.ProviderID() {
		return Link{}, result(completion.ValidationFailure, "work_item_capability_unavailable")
	}
	shown := s.Show(ctx, target, selector)
	if shown.Status != completion.Success {
		return Link{}, shown
	}
	if shown.Link.Provider != s.capability.ProviderID() || !s.capability.ValidResource(shown.Link.Resource) {
		return Link{}, result(completion.ValidationFailure, "work_item_capability_unavailable")
	}
	return shown.Link, Result{}
}

func normalizeRevision(input UpdateInput) (DocumentRevision, string) {
	revision := DocumentRevision{}
	total := 0
	if input.Title != "" {
		if !validDraftText(input.Title) || strings.ContainsAny(input.Title, "\n\t") || len(input.Title) > maxTitleBytes {
			return DocumentRevision{}, "invalid_work_item_title"
		}
		if looksSensitive(input.Title) {
			return DocumentRevision{}, "secret_rejected"
		}
		revision.Title = input.Title
		total += len(input.Title)
	}
	for _, field := range []struct {
		name  string
		input SectionInput
	}{
		{"problem", input.Problem}, {"desired_outcome", input.DesiredOutcome}, {"context", input.Context},
		{"scope", input.Scope}, {"constraints", input.Constraints}, {"non_goals", input.NonGoals},
		{"acceptance_expectations", input.Acceptance},
	} {
		if field.input.Supplied != "" && field.input.Elaborated != "" {
			return DocumentRevision{}, "invalid_draft_input"
		}
		content, authorship := field.input.Supplied, provenance.UserAuthored
		if content == "" {
			content, authorship = field.input.Elaborated, provenance.AxiomAuthored
		}
		if content == "" {
			continue
		}
		if !validDraftText(content) {
			return DocumentRevision{}, "invalid_draft_input"
		}
		if looksSensitive(content) {
			return DocumentRevision{}, "secret_rejected"
		}
		total += len(content)
		revision.Sections = append(revision.Sections, DraftSection{Name: field.name, Content: content, Authorship: authorship})
	}
	if revision.Title == "" && len(revision.Sections) == 0 {
		return DocumentRevision{}, "work_item_update_empty"
	}
	if total > maxDraftBytes {
		return DocumentRevision{}, "draft_too_large"
	}
	return revision, ""
}

func (r DocumentRevision) fields() []string {
	fields := make([]string, 0, len(r.Sections)+1)
	if r.Title != "" {
		fields = append(fields, "title")
	}
	for _, section := range r.Sections {
		fields = append(fields, section.Name)
	}
	return fields
}

func documentDigest(document ProviderDocument) string {
	digest := sha256.Sum256([]byte(document.Title + "\x00" + document.Body))
	return hex.EncodeToString(digest[:])
}
