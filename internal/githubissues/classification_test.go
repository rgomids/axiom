package githubissues

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/testfs"
	"github.com/rgomids/axiom/internal/workitem"
)

func classificationAdapter(t *testing.T, catalog, issue string) (Adapter, string, string, string) {
	t.Helper()
	testfs.POSIXShell(t)
	directory := t.TempDir()
	gh, labels, response, log, payload := filepath.Join(directory, "gh"), filepath.Join(directory, "labels"), filepath.Join(directory, "issue"), filepath.Join(directory, "log"), filepath.Join(directory, "payload")
	for path, content := range map[string]string{labels: catalog, response: issue, gh: `#!/bin/sh
printf '%s\n' "$*" >> "$AXIOM_CLASS_LOG"
case "$*" in
  *labels?per_page=100*) cat "$AXIOM_CLASS_LABELS" ;;
  *POST*) cat > "$AXIOM_CLASS_PAYLOAD"; cat "$AXIOM_CLASS_ISSUE" ;;
  *issues/7*) cat "$AXIOM_CLASS_ISSUE" ;;
  *) exit 1 ;;
esac
`} {
		if err := os.WriteFile(path, []byte(content), 0700); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("AXIOM_CLASS_LOG", log)
	t.Setenv("AXIOM_CLASS_LABELS", labels)
	t.Setenv("AXIOM_CLASS_PAYLOAD", payload)
	t.Setenv("AXIOM_CLASS_ISSUE", response)
	adapter, err := New(gh)
	if err != nil {
		t.Fatal(err)
	}
	return adapter, labels, log, payload
}

const classifiedIssue = `{"number":7,"html_url":"https://github.com/owner/repo/issues/7","state":"open","labels":[{"name":"bug"},{"name":"area:cli"}]}`

func TestClassifySelectsOnlyObservedLabelsAndMakesDegradationExplicit(t *testing.T) {
	adapter, labels, _, _ := classificationAdapter(t, `[{"name":"bug"},{"name":"type:bug"},{"name":"enhancement"},{"name":"area:cli"}]`, classifiedIssue)
	for _, test := range []struct {
		name           string
		kind           workitem.Type
		explicit, want []string
		unsupported    bool
	}{
		{"bug", workitem.Bug, nil, []string{"type:bug"}, false},
		{"story fallback", workitem.Story, nil, []string{"enhancement"}, false},
		{"explicit areas", workitem.Bug, []string{"bug", "area:cli", "bug"}, []string{"area:cli", "bug"}, false},
		{"unknown", workitem.Bug, []string{"invented"}, nil, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			document, err := adapter.ClassifyDocument(context.Background(), workitem.Draft{Type: test.kind}, workitem.DraftTarget{Resource: "owner/repo"}, workitem.ProviderDocument{}, test.explicit)
			if test.unsupported {
				var provider *workitem.ProviderError
				if !errors.As(err, &provider) || provider.Kind != workitem.ProviderClassificationUnsupported {
					t.Fatalf("err=%v", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			metadata, err := decodeCreateMetadata(document.Metadata)
			if err != nil || !reflect.DeepEqual(metadata.Labels, test.want) {
				t.Fatalf("metadata=%+v err=%v", metadata, err)
			}
		})
	}
	if err := os.WriteFile(labels, []byte(`[]`), 0600); err != nil {
		t.Fatal(err)
	}
	document, err := adapter.ClassifyDocument(context.Background(), workitem.Draft{Type: workitem.Task}, workitem.DraftTarget{Resource: "owner/repo"}, workitem.ProviderDocument{}, nil)
	if err != nil || string(document.Metadata) != `{"labels":[]}` || !reflect.DeepEqual(document.Notices, []string{"no_existing_label_for_item_type"}) {
		t.Fatalf("document=%+v err=%v", document, err)
	}
}

func TestCreateAppliesReviewedLabelsViaStdinAndDetectsDroppedLabels(t *testing.T) {
	for _, dropped := range []bool{false, true} {
		t.Run(fmt.Sprint(dropped), func(t *testing.T) {
			response := classifiedIssue
			if dropped {
				response = `{"number":7,"html_url":"https://github.com/owner/repo/issues/7","state":"open","labels":[]}`
			}
			adapter, _, log, payload := classificationAdapter(t, `[{"name":"bug"},{"name":"area:cli"}]`, response)
			request := workitem.CreateRequest{Resource: "owner/repo", Correlation: strings.Repeat("a", 64), Document: workitem.ProviderDocument{Title: "Bounded change", Body: "Reviewed body", Metadata: json.RawMessage(`{"labels":["area:cli","bug"]}`)}}
			external, err := adapter.Create(context.Background(), request)
			if err != nil || external.MetadataIncomplete != dropped {
				t.Fatalf("external=%+v err=%v", external, err)
			}
			wire, err := os.ReadFile(payload)
			if err != nil {
				t.Fatal(err)
			}
			var sent struct {
				Labels []string `json:"labels"`
			}
			if err := json.Unmarshal(wire, &sent); err != nil || !reflect.DeepEqual(sent.Labels, []string{"area:cli", "bug"}) {
				t.Fatalf("payload=%s err=%v", wire, err)
			}
			args, _ := os.ReadFile(log)
			if strings.Contains(string(args), "area:cli") || strings.Count(string(args), "POST") != 1 {
				t.Fatalf("args=%s", args)
			}
			verified, err := adapter.VerifyDocument(context.Background(), request, external)
			if err != nil || verified == dropped {
				t.Fatalf("verified=%v err=%v", verified, err)
			}
		})
	}
}

func TestMissingOrInvalidLabelsCannotCausePost(t *testing.T) {
	adapter, _, log, _ := classificationAdapter(t, `[]`, classifiedIssue)
	for _, metadata := range []string{`{"labels":["bug"]}`, `{"labels":["bug","bug"]}`, `{"labels":["bad\nlabel"]}`, `{"labels":[],"assignees":["unexpected"]}`} {
		_, err := adapter.Create(context.Background(), workitem.CreateRequest{Resource: "owner/repo", Correlation: strings.Repeat("a", 64), Document: workitem.ProviderDocument{Title: "Reviewed", Body: "Body", Metadata: json.RawMessage(metadata)}})
		var provider *workitem.ProviderError
		if !errors.As(err, &provider) || !provider.EffectNotCommitted {
			t.Fatalf("metadata=%s err=%v", metadata, err)
		}
	}
	args, _ := os.ReadFile(log)
	if strings.Contains(string(args), "POST") {
		t.Fatalf("mutation=%s", args)
	}
}

func TestCatalogPaginationFailsClosedOnMalformedOrRepeatedPages(t *testing.T) {
	for _, catalog := range []string{`null`, `{}`, `[{"name":""}]`, `[{"name":"bug"},{"name":"bug"}]`, `[{"name":"bad\nlabel"}]`} {
		adapter, _, _, _ := classificationAdapter(t, catalog, classifiedIssue)
		if _, err := adapter.creationLabels(context.Background(), "owner/repo"); err == nil {
			t.Fatalf("accepted=%s", catalog)
		}
	}
	rows := make([]map[string]string, 100)
	for i := range rows {
		rows[i] = map[string]string{"name": fmt.Sprintf("label-%03d", i)}
	}
	wire, _ := json.Marshal(rows)
	adapter, _, log, _ := classificationAdapter(t, string(wire), classifiedIssue)
	if _, err := adapter.creationLabels(context.Background(), "owner/repo"); err == nil {
		t.Fatal("repeated full pages accepted")
	}
	args, _ := os.ReadFile(log)
	if !strings.Contains(string(args), "page=2") {
		t.Fatalf("pagination=%s", args)
	}
}

func TestCatalogSelectionCanReachSecondPageAndRemovedLabelPreventsCreation(t *testing.T) {
	rows := make([]map[string]string, 100)
	for i := range rows {
		rows[i] = map[string]string{"name": fmt.Sprintf("area:%03d", i)}
	}
	wire, _ := json.Marshal(rows)
	adapter, catalog, log, _ := classificationAdapter(t, string(wire), classifiedIssue)
	script := `#!/bin/sh
printf '%s\n' "$*" >> "$AXIOM_CLASS_LOG"
case "$*" in
  *page=2*) printf '%s\n' '[{"name":"type:bug"}]' ;;
  *labels?per_page=100*) cat "$AXIOM_CLASS_LABELS" ;;
  *) exit 1 ;;
esac
`
	if err := os.WriteFile(adapter.gh, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	document, err := adapter.ClassifyDocument(context.Background(), workitem.Draft{Type: workitem.Bug}, workitem.DraftTarget{Resource: "owner/repo"}, workitem.ProviderDocument{Title: "Reviewed bug", Body: "Reviewed outcome"}, nil)
	if err != nil || string(document.Metadata) != `{"labels":["type:bug"]}` {
		t.Fatalf("document=%+v err=%v", document, err)
	}
	if err := os.WriteFile(catalog, []byte(`[]`), 0600); err != nil {
		t.Fatal(err)
	}
	_, err = adapter.Create(context.Background(), workitem.CreateRequest{Resource: "owner/repo", Correlation: strings.Repeat("a", 64), Document: document})
	var provider *workitem.ProviderError
	if !errors.As(err, &provider) || provider.Kind != workitem.ProviderClassificationUnsupported || !provider.EffectNotCommitted {
		t.Fatalf("err=%v", err)
	}
	args, _ := os.ReadFile(log)
	if strings.Contains(string(args), "POST") {
		t.Fatalf("mutation=%s", args)
	}
}

func TestStoryDocumentContainsReviewedTypeBeneficiaryAndValue(t *testing.T) {
	draft := workitem.Draft{Type: workitem.Story, Beneficiary: &workitem.DraftSection{Name: "beneficiary", Content: "Maintainers", Authorship: "user"}, Value: &workitem.DraftSection{Name: "value", Content: "Prioritize work by meaningful delivery outcome", Authorship: "user"}}
	for _, name := range []string{"problem", "desired_outcome", "context", "scope", "constraints", "non_goals", "acceptance_expectations"} {
		draft.Sections = append(draft.Sections, workitem.DraftSection{Name: name, Content: "Reviewed content", Authorship: "user"})
	}
	document, err := (Adapter{}).Render(draft, workitem.DraftTarget{}, strings.Repeat("a", 64), testSource())
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"Work Item type: `story`", "## Story beneficiary", "    Maintainers", "## Story value", "    Prioritize work by meaningful delivery outcome"} {
		if !strings.Contains(document.Body, expected) {
			t.Fatalf("missing %q body=%s", expected, document.Body)
		}
	}
	draft.Type = "epic"
	if _, err := (Adapter{}).Render(draft, workitem.DraftTarget{}, strings.Repeat("a", 64), testSource()); err == nil {
		t.Fatal("unsupported type rendered")
	}
}

// Issue label taxonomy contract (PR #195): on a repository whose catalog governs type,
// area and status label families, `work-item create --classification
// type:<type> --classification area:<area>` sends exactly one type and one area
// label. The repository Issue label policy then adds status:planned
// (scripts/test-issue-label-policy.py AxiomAuthoredTest). Area is never
// inferred: without explicit area intent no area label is proposed.
func TestExplicitClassificationMeetsIssueLabelTaxonomy(t *testing.T) {
	catalog := `[{"name":"type:epic"},{"name":"type:story"},{"name":"type:task"},{"name":"type:bug"},{"name":"type:research"},` +
		`{"name":"area:cli"},{"name":"area:work-item"},{"name":"area:installer"},` +
		`{"name":"status:planned"},{"name":"status:active"},{"name":"status:blocked"},{"name":"platform:windows"},{"name":"axiom:stage:specifying"}]`
	adapter, _, _, _ := classificationAdapter(t, catalog, classifiedIssue)
	family := func(labels []string, prefix string) []string {
		var selected []string
		for _, label := range labels {
			if strings.HasPrefix(label, prefix) {
				selected = append(selected, label)
			}
		}
		return selected
	}
	classify := func(kind workitem.Type, explicit []string) []string {
		t.Helper()
		document, err := adapter.ClassifyDocument(context.Background(), workitem.Draft{Type: kind}, workitem.DraftTarget{Resource: "owner/repo"}, workitem.ProviderDocument{}, explicit)
		if err != nil {
			t.Fatalf("%s: %v", kind, err)
		}
		metadata, err := decodeCreateMetadata(document.Metadata)
		if err != nil {
			t.Fatal(err)
		}
		return metadata.Labels
	}
	for _, test := range []struct {
		kind workitem.Type
		area string
	}{{workitem.Story, "area:work-item"}, {workitem.Task, "area:cli"}, {workitem.Bug, "area:installer"}} {
		labels := classify(test.kind, []string{"type:" + string(test.kind), test.area})
		if !reflect.DeepEqual(family(labels, "type:"), []string{"type:" + string(test.kind)}) || !reflect.DeepEqual(family(labels, "area:"), []string{test.area}) ||
			len(family(labels, "status:")) != 0 || len(family(labels, "axiom:")) != 0 || len(labels) != 2 {
			t.Fatalf("%s labels=%v", test.kind, labels)
		}
		if inferred := classify(test.kind, nil); !reflect.DeepEqual(inferred, []string{"type:" + string(test.kind)}) {
			t.Fatalf("%s inferred=%v", test.kind, inferred)
		}
	}
}
