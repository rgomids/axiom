package workitem

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/completion"
)

func TestTypesAndStoryDeliveryValue(t *testing.T) {
	for _, itemType := range []Type{"", Task, Bug, Story, "epic"} {
		t.Run(string(itemType), func(t *testing.T) {
			input := completeDraft()
			input.Type = itemType
			service := testService(&fakeCapability{}, newFakeStore())
			preview := service.Prepare(context.Background(), input)
			if itemType == "epic" {
				if preview.Category != "invalid_work_item_type" {
					t.Fatalf("result=%+v", preview)
				}
				return
			}
			if itemType == Story {
				// Implementation sections alone cannot constitute a story.
				if preview.Category != "draft_incomplete" || len(preview.Questions) != 2 || preview.Questions[0].Field != "beneficiary" || preview.Questions[1].Field != "value" {
					t.Fatalf("result=%+v", preview)
				}
				input.Beneficiary = SectionInput{Supplied: "Repository maintainers"}
				input.Value = SectionInput{Supplied: "Prioritize product delivery from a backlog with visible intent"}
				preview = service.Prepare(context.Background(), input)
				if preview.Draft == nil || preview.Draft.Draft.Value.Content != input.Value.Supplied {
					t.Fatalf("result=%+v", preview)
				}
			}
			if preview.Status != completion.Success || !preview.Draft.Draft.Type.Valid() {
				t.Fatalf("result=%+v", preview)
			}
			if itemType == "" && preview.Draft.Draft.Type != Task {
				t.Fatal("legacy input must normalize to explicit task")
			}
		})
	}
}

type classifiedCapability struct {
	*fakeCapability
	metadata        json.RawMessage
	verified        bool
	verifyErr       error
	classifications []string
}

func (p *classifiedCapability) ClassifyDocument(_ context.Context, _ Draft, _ DraftTarget, document ProviderDocument, explicit []string) (ProviderDocument, error) {
	p.classifications = append([]string(nil), explicit...)
	document.Metadata = append(json.RawMessage(nil), p.metadata...)
	return document, nil
}

func (p *classifiedCapability) VerifyDocument(context.Context, CreateRequest, External) (bool, error) {
	return p.verified, p.verifyErr
}

func TestClassificationAndTypeBindExactAuthority(t *testing.T) {
	provider := &classifiedCapability{fakeCapability: &fakeCapability{}, metadata: json.RawMessage(`{"classification":["technical"]}`), verified: true}
	service := New(fakeResolver{}, provider, provider, newFakeStore(), testProvenance())
	input := completeDraft()
	preview := service.Prepare(context.Background(), input)
	provider.metadata = json.RawMessage(`{"classification":["delivery"]}`)
	changed := service.Prepare(context.Background(), input)
	if changed.Draft.Digest == preview.Draft.Digest {
		t.Fatal("adapter metadata must participate in digest")
	}
	if denied := service.Create(context.Background(), input, preview.Draft.Digest, true); denied.Status != completion.DeniedAuthority || provider.creates != 0 {
		t.Fatalf("denied=%+v", denied)
	}
	input.Type = Bug
	if denied := service.Create(context.Background(), input, changed.Draft.Digest, true); denied.Status != completion.DeniedAuthority {
		t.Fatalf("type change=%+v", denied)
	}
	input.Type = Task
	input.Classification = []string{"technical"}
	first := service.Prepare(context.Background(), input)
	input.Classification = []string{"delivery"}
	second := service.Prepare(context.Background(), input)
	if first.Draft.Correlation == second.Draft.Correlation {
		t.Fatal("classification intention must participate in recovery identity")
	}
}

func TestMetadataDegradationSurvivesConfirmedRetryWithoutDuplicate(t *testing.T) {
	provider := &classifiedCapability{fakeCapability: &fakeCapability{}, metadata: json.RawMessage(`{"classification":["technical"]}`)}
	service := New(fakeResolver{}, provider, provider, newFakeStore(), testProvenance())
	input := completeDraft()
	preview := service.Prepare(context.Background(), input)
	for attempt := 0; attempt < 2; attempt++ {
		created := service.Create(context.Background(), input, preview.Draft.Digest, true)
		if created.Status != completion.Partial || created.Category != "provider_metadata_incomplete" || created.Link.ExternalID != "7" || provider.creates != 1 {
			t.Fatalf("attempt=%d result=%+v calls=%d", attempt, created, provider.creates)
		}
	}
	provider.verified = true
	if result := service.Create(context.Background(), input, preview.Draft.Digest, true); result.Status != completion.Success || provider.creates != 1 {
		t.Fatalf("result=%+v", result)
	}
}

func TestChangedProviderMetadataReusesConfirmedItem(t *testing.T) {
	provider := &classifiedCapability{fakeCapability: &fakeCapability{}, metadata: json.RawMessage(`{"classification":["technical"]}`), verified: true}
	service := New(fakeResolver{}, provider, provider, newFakeStore(), testProvenance())
	input := completeDraft()
	preview := service.Prepare(context.Background(), input)
	if created := service.Create(context.Background(), input, preview.Draft.Digest, true); created.Status != completion.Success {
		t.Fatalf("created=%+v", created)
	}
	provider.metadata = json.RawMessage(`{"classification":["delivery"]}`)
	provider.verified = false
	updated := service.Prepare(context.Background(), input)
	if updated.Draft.Digest == preview.Draft.Digest || updated.Draft.Correlation != preview.Draft.Correlation {
		t.Fatal("provider mapping changed delivery identity")
	}
	result := service.Create(context.Background(), input, updated.Draft.Digest, true)
	if result.Status != completion.Partial || result.Category != "provider_metadata_incomplete" || provider.creates != 1 || result.Link.ExternalID != "7" {
		t.Fatalf("result=%+v creates=%d", result, provider.creates)
	}
}

func TestReconciledMetadataFailurePreservesExistingReference(t *testing.T) {
	provider := &classifiedCapability{fakeCapability: &fakeCapability{reconcileSequence: [][]External{{{ID: "7", URL: "https://github.com/owner/repo/issues/7", State: "OPEN"}}}}, metadata: json.RawMessage(`{"classification":["technical"]}`), verifyErr: errors.New("verification unavailable")}
	service := New(fakeResolver{}, provider, provider, newFakeStore(), testProvenance())
	input := completeDraft()
	preview := service.Prepare(context.Background(), input)
	result := service.Create(context.Background(), input, preview.Draft.Digest, true)
	if result.Status != completion.Partial || result.Category != "provider_metadata_unverified" || result.Link.ExternalID != "7" || provider.creates != 0 {
		t.Fatalf("result=%+v creates=%d", result, provider.creates)
	}
}

func TestStoryValueAndClassificationsUseDraftSecurityLimits(t *testing.T) {
	for _, field := range []string{"beneficiary", "value", "classification"} {
		input := completeDraft()
		input.Type, input.Beneficiary, input.Value = Story, SectionInput{Supplied: "Maintainers"}, SectionInput{Supplied: "Understand delivery value"}
		synthetic := "password=SYNTHETIC_REJECTED"
		switch field {
		case "beneficiary":
			input.Beneficiary.Supplied = synthetic
		case "value":
			input.Value.Supplied = synthetic
		case "classification":
			input.Classification = []string{synthetic}
		}
		result := testService(&fakeCapability{}, newFakeStore()).Prepare(context.Background(), input)
		if result.Category != "secret_rejected" {
			t.Fatalf("%s=%+v", field, result)
		}
	}
	input := completeDraft()
	input.Classification = []string{strings.Repeat("x", 257)}
	if result := testService(&fakeCapability{}, newFakeStore()).Prepare(context.Background(), input); result.Category != "invalid_classification" {
		t.Fatalf("result=%+v", result)
	}
	input = completeDraft()
	input.Value = SectionInput{Supplied: "Beneficial outcome"}
	if result := testService(&fakeCapability{}, newFakeStore()).Prepare(context.Background(), input); result.Category != "story_value_requires_story" {
		t.Fatalf("result=%+v", result)
	}
}

func TestEquivalentClassificationsKeepIdentityAndCreateOnlyOnce(t *testing.T) {
	provider := &classifiedCapability{fakeCapability: &fakeCapability{}, metadata: json.RawMessage(`{"classification":["area:cli","bug"]}`), verified: true}
	service := New(fakeResolver{}, provider, provider, newFakeStore(), testProvenance())
	variants := [][]string{{"bug", "area:cli"}, {"area:cli", "bug"}, {"bug", "area:cli", "bug"}}
	canonical := []string{"area:cli", "bug"}
	var correlation, digest string
	for _, values := range variants {
		input := completeDraft()
		input.Classification = append([]string(nil), values...)
		preview := service.Prepare(context.Background(), input)
		if preview.Status != completion.Success || preview.Draft == nil {
			t.Fatalf("preview=%+v", preview)
		}
		if correlation == "" {
			correlation, digest = preview.Draft.Correlation, preview.Draft.Digest
		}
		if preview.Draft.Correlation != correlation || preview.Draft.Digest != digest {
			t.Fatalf("equivalent input %v changed identity or authority digest", values)
		}
		if !reflect.DeepEqual(preview.Draft.Draft.Classification, canonical) || !reflect.DeepEqual(provider.classifications, canonical) {
			t.Fatalf("draft=%v adapter=%v", preview.Draft.Draft.Classification, provider.classifications)
		}
		if !reflect.DeepEqual(input.Classification, values) {
			t.Fatal("normalization mutated caller input")
		}
		created := service.Create(context.Background(), input, digest, true)
		if created.Status != completion.Success || created.Link.ExternalID != "7" || provider.creates != 1 {
			t.Fatalf("input=%v result=%+v creates=%d", values, created, provider.creates)
		}
	}
}
