package githubissues

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/provenance"
	"github.com/rgomids/axiom/internal/testfs"
	"github.com/rgomids/axiom/internal/workitem"
)

// lifecycleStub records every gh invocation's arguments and stdin and answers
// with the fixture response in $AXIOM_TEST_DIR/response.
type lifecycleStub struct {
	adapter   Adapter
	directory string
}

func newLifecycleStub(t *testing.T) lifecycleStub {
	t.Helper()
	testfs.POSIXShell(t)
	directory := t.TempDir()
	gh := filepath.Join(directory, "gh")
	script := `#!/bin/sh
n=$(cat "$AXIOM_TEST_DIR/count" 2>/dev/null || echo 0)
n=$((n + 1))
echo "$n" > "$AXIOM_TEST_DIR/count"
printf '%s\n' "$@" > "$AXIOM_TEST_DIR/args-$n"
cat > "$AXIOM_TEST_DIR/stdin-$n"
cat "$AXIOM_TEST_DIR/response"
exit "$(cat "$AXIOM_TEST_DIR/exit" 2>/dev/null || echo 0)"
`
	if err := os.WriteFile(gh, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AXIOM_TEST_DIR", directory)
	adapter, err := New(gh)
	if err != nil {
		t.Fatal(err)
	}
	return lifecycleStub{adapter: adapter, directory: directory}
}

func (s lifecycleStub) respond(t *testing.T, response string, exit int) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(s.directory, "response"), []byte(response), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(s.directory, "exit"), []byte(strconv.Itoa(exit)), 0o600); err != nil {
		t.Fatal(err)
	}
}

func (s lifecycleStub) calls() int {
	wire, err := os.ReadFile(filepath.Join(s.directory, "count"))
	if err != nil {
		return 0
	}
	count, _ := strconv.Atoi(strings.TrimSpace(string(wire)))
	return count
}

func (s lifecycleStub) call(t *testing.T, number int) (string, string) {
	t.Helper()
	arguments, err := os.ReadFile(filepath.Join(s.directory, "args-"+strconv.Itoa(number)))
	if err != nil {
		t.Fatal(err)
	}
	stdin, err := os.ReadFile(filepath.Join(s.directory, "stdin-"+strconv.Itoa(number)))
	if err != nil {
		t.Fatal(err)
	}
	return strings.Join(strings.Fields(string(arguments)), " "), string(stdin)
}

const (
	openIssue   = `{"number":7,"html_url":"https://github.com/owner/repo/issues/7","state":"open","title":"Current title","body":"Current body"}`
	closedIssue = `{"number":7,"html_url":"https://github.com/owner/repo/issues/7","state":"closed","title":"Current title","body":"Current body"}`
)

func TestLifecycleAdapterRequestAndResponseFixtures(t *testing.T) {
	stub := newLifecycleStub(t)
	ctx := context.Background()
	patch := "api --include --method PATCH repos/owner/repo/issues/7 --input -"
	tests := []struct {
		name, response, args, stdin string
		run                         func() (workitem.External, error)
		state                       string
	}{
		{"read document", openIssue, "api --include repos/owner/repo/issues/7", "", func() (workitem.External, error) {
			external, document, err := stub.adapter.ReadDocument(ctx, "owner/repo", "7")
			if err == nil && (document.Title != "Current title" || document.Body != "Current body") {
				t.Fatalf("document = %#v", document)
			}
			return external, err
		}, workitem.OpenState},
		{"patch title", openIssue, patch, `{"title":"New title"}`, func() (workitem.External, error) {
			return stub.adapter.UpdateDocument(ctx, "owner/repo", "7", workitem.DocumentChange{Title: "New title"})
		}, workitem.OpenState},
		{"patch body", openIssue, patch, `{"body":"New $(body) ` + "`x`" + `"}`, func() (workitem.External, error) {
			return stub.adapter.UpdateDocument(ctx, "owner/repo", "7", workitem.DocumentChange{Body: "New $(body) `x`"})
		}, workitem.OpenState},
		{"patch title and body", openIssue, patch, `{"title":"T","body":"B"}`, func() (workitem.External, error) {
			return stub.adapter.UpdateDocument(ctx, "owner/repo", "7", workitem.DocumentChange{Title: "T", Body: "B"})
		}, workitem.OpenState},
		{"patch state closed", closedIssue, patch, `{"state":"closed"}`, func() (workitem.External, error) {
			return stub.adapter.SetState(ctx, "owner/repo", "7", workitem.ClosedState)
		}, workitem.ClosedState},
		{"patch state open", openIssue, patch, `{"state":"open"}`, func() (workitem.External, error) {
			return stub.adapter.SetState(ctx, "owner/repo", "7", workitem.OpenState)
		}, workitem.OpenState},
		{"post comment", `{"id":1}`, "api --include --method POST repos/owner/repo/issues/7/comments --input -", `{"body":"Evidence"}`, func() (workitem.External, error) {
			return workitem.External{}, stub.adapter.Comment(ctx, "owner/repo", "7", "Evidence")
		}, ""},
	}
	for index, test := range tests {
		stub.respond(t, test.response+"\n", 0)
		external, err := test.run()
		if err != nil || external.State != test.state || test.state != "" && (external.ID != "7" || external.URL != "https://github.com/owner/repo/issues/7") {
			t.Fatalf("%s = %#v, %v", test.name, external, err)
		}
		arguments, stdin := stub.call(t, index+1)
		if arguments != test.args || strings.TrimSpace(stdin) != test.stdin {
			t.Fatalf("%s request args=%q stdin=%q", test.name, arguments, stdin)
		}
	}
}

func TestLifecycleAdapterFailsClosedOnUnexpectedResponsesAndInputs(t *testing.T) {
	stub := newLifecycleStub(t)
	ctx := context.Background()
	stub.respond(t, openIssue+"\n", 0)
	// A PATCH whose response does not confirm the requested state is not a
	// confirmed effect.
	if _, err := stub.adapter.SetState(ctx, "owner/repo", "7", workitem.ClosedState); !isKind(err, workitem.ProviderInvalidResponse) {
		t.Fatalf("unconfirmed state = %v", err)
	}
	stub.respond(t, `{"number":8,"html_url":"https://github.com/owner/repo/issues/8","state":"open","title":"x"}`+"\n", 0)
	if _, _, err := stub.adapter.ReadDocument(ctx, "owner/repo", "7"); !isKind(err, workitem.ProviderInvalidResponse) {
		t.Fatalf("mismatched read = %v", err)
	}
	stub.respond(t, `{"number":7,"html_url":"https://github.com/owner/repo/issues/7","state":"open","title":""}`+"\n", 0)
	if _, _, err := stub.adapter.ReadDocument(ctx, "owner/repo", "7"); !isKind(err, workitem.ProviderInvalidResponse) {
		t.Fatalf("untitled read = %v", err)
	}
	stub.respond(t, "HTTP/2.0 503 unavailable\r\nContent-Type: application/json\r\n\r\n{}\n", 1)
	_, err := stub.adapter.SetState(ctx, "owner/repo", "7", workitem.ClosedState)
	var provider *workitem.ProviderError
	if !errors.As(err, &provider) || provider.Kind != workitem.ProviderUnavailable || !provider.Retryable {
		t.Fatalf("transient PATCH = %#v", err)
	}
	calls := stub.calls()
	for name, run := range map[string]func() error{
		"merged state": func() error { _, err := stub.adapter.SetState(ctx, "owner/repo", "7", "MERGED"); return err },
		"empty change": func() error {
			_, err := stub.adapter.UpdateDocument(ctx, "owner/repo", "7", workitem.DocumentChange{})
			return err
		},
		"multiline title": func() error {
			_, err := stub.adapter.UpdateDocument(ctx, "owner/repo", "7", workitem.DocumentChange{Title: "a\nb"})
			return err
		},
		"oversized body": func() error {
			_, err := stub.adapter.UpdateDocument(ctx, "owner/repo", "7", workitem.DocumentChange{Body: strings.Repeat("x", bodyLimit+1)})
			return err
		},
		"unsafe selector":   func() error { _, err := stub.adapter.SetState(ctx, "owner/repo", "07", workitem.OpenState); return err },
		"unsafe repository": func() error { return stub.adapter.Comment(ctx, "owner/../repo", "7", "x") },
		"empty comment":     func() error { return stub.adapter.Comment(ctx, "owner/repo", "7", "") },
	} {
		err := run()
		if !errors.As(err, &provider) || !provider.EffectNotCommitted {
			t.Fatalf("%s = %#v", name, err)
		}
	}
	if stub.calls() != calls {
		t.Fatal("invalid lifecycle input reached the Provider")
	}
}

func isKind(err error, kind workitem.ProviderErrorKind) bool {
	var provider *workitem.ProviderError
	return errors.As(err, &provider) && provider.Kind == kind
}

const testCorrelation = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

func renderBody(t *testing.T, draft workitem.Draft) string {
	t.Helper()
	document, err := (Adapter{}).Render(draft, workitem.DraftTarget{}, testCorrelation, testSource())
	if err != nil {
		t.Fatal(err)
	}
	return document.Body
}

func withSection(draft workitem.Draft, section workitem.DraftSection) workitem.Draft {
	revised := draft
	revised.Sections = append([]workitem.DraftSection(nil), draft.Sections...)
	for index := range revised.Sections {
		if revised.Sections[index].Name == section.Name {
			revised.Sections[index] = section
		}
	}
	return revised
}

// A section revision of an Axiom-rendered body must equal rendering the
// revised draft: every unspecified byte, the header and the footer survive.
func TestReviseDocumentEqualsRenderingTheRevisedDraft(t *testing.T) {
	cases := renderGoldenCases()
	for _, test := range []struct {
		name     string
		revision workitem.DraftSection
	}{
		{"task", workitem.DraftSection{Name: "scope", Content: "Narrower scope.\n\n- one\n- two", Authorship: provenance.UserAuthored}},
		{"story", workitem.DraftSection{Name: "problem", Content: "Story problem revised", Authorship: provenance.UserAuthored}},
		{"markdown", workitem.DraftSection{Name: "scope", Content: "Run:\n\n```sh\n## still code\n```", Authorship: provenance.UserAuthored}},
		{"structure-spoof", workitem.DraftSection{Name: "acceptance_expectations", Content: "---\n## Problem\n_All section content is user-authored._", Authorship: provenance.UserAuthored}},
		{"structure-spoof", workitem.DraftSection{Name: "constraints", Content: "Plain constraints", Authorship: provenance.UserAuthored}},
		{"mixed-authorship", workitem.DraftSection{Name: "context", Content: "Context now user-authored", Authorship: provenance.UserAuthored}},
		{"task", workitem.DraftSection{Name: "problem", Content: "Axiom elaborated problem", Authorship: provenance.AxiomAuthored}},
		{"unicode", workitem.DraftSection{Name: "desired_outcome", Content: "Resultado 日本語 👩‍💻", Authorship: provenance.UserAuthored}},
	} {
		draft := cases[test.name]
		current := workitem.ProviderDocument{Title: "Human edited title", Body: renderBody(t, draft)}
		revised, err := (Adapter{}).ReviseDocument(current, workitem.DocumentRevision{Sections: []workitem.DraftSection{test.revision}})
		if err != nil {
			t.Fatalf("%s/%s: %v", test.name, test.revision.Name, err)
		}
		if want := renderBody(t, withSection(draft, test.revision)); revised.Body != want || revised.Title != current.Title {
			t.Fatalf("%s/%s revised body:\n%s\nwant:\n%s", test.name, test.revision.Name, revised.Body, want)
		}
		crlf := current
		crlf.Body = strings.ReplaceAll(current.Body, "\n", "\r\n")
		revisedCRLF, err := (Adapter{}).ReviseDocument(crlf, workitem.DocumentRevision{Sections: []workitem.DraftSection{test.revision}})
		if want := strings.ReplaceAll(renderBody(t, withSection(draft, test.revision)), "\n", "\r\n"); err != nil || revisedCRLF.Body != want {
			t.Fatalf("%s/%s CRLF revision = %q, %v", test.name, test.revision.Name, revisedCRLF.Body, err)
		}
	}
}

func TestReviseDocumentPreservesHumanContentOutsideRevisedSections(t *testing.T) {
	original := renderBody(t, renderDraft(workitem.Task, nil))
	edited := strings.Replace(original, "## Scope\n", "## Human notes\n\nKeep this paragraph.\n\n## Scope\n", 1) + "\nHuman trailer after the footer.\n"
	revision := workitem.DraftSection{Name: "context", Content: "Revised context", Authorship: provenance.UserAuthored}
	revised, err := (Adapter{}).ReviseDocument(workitem.ProviderDocument{Title: "T", Body: edited}, workitem.DocumentRevision{Sections: []workitem.DraftSection{revision}})
	if err != nil {
		t.Fatal(err)
	}
	want := strings.Replace(edited, "## Context\n\nThe GitHub renderer owns Issue presentation.\n", "## Context\n\nRevised context\n", 1)
	if revised.Body != want {
		t.Fatalf("revised:\n%s\nwant:\n%s", revised.Body, want)
	}
	titled, err := (Adapter{}).ReviseDocument(workitem.ProviderDocument{Title: "T", Body: "not an Axiom body"}, workitem.DocumentRevision{Title: "New title"})
	if err != nil || titled.Title != "New title" || titled.Body != "not an Axiom body" {
		t.Fatalf("title-only revision = %#v, %v", titled, err)
	}
}

func TestReviseDocumentRejectsUnrecognizedOrOversizedBodies(t *testing.T) {
	original := renderBody(t, renderDraft(workitem.Task, nil))
	scope := []workitem.DraftSection{{Name: "scope", Content: "x", Authorship: provenance.UserAuthored}}
	for name, body := range map[string]string{
		"free form":          "A human Issue body\n",
		"no footer":          original[:strings.Index(original, "\n---\n")],
		"duplicate heading":  strings.Replace(original, "## Context\n", "## Scope\n\nduplicate\n\n## Context\n", 1),
		"missing section":    strings.Replace(original, "## Scope\n", "## Range\n", 1),
		"unknown authorship": strings.Replace(original, "_All section content is user-authored._", "_Somebody wrote this._", 1),
		"partial authorship": strings.Replace(original, "_All section content is user-authored._", "_User-authored: Problem. Axiom-authored: Scope._", 1),
		"collapsed region":   strings.Replace(original, "## Scope\n\nTitle and body presentation only.\n\n", "## Scope\nTitle and body presentation only.\n", 1),
	} {
		if _, err := (Adapter{}).ReviseDocument(workitem.ProviderDocument{Title: "T", Body: body}, workitem.DocumentRevision{Sections: scope}); !errors.Is(err, workitem.ErrDocumentUnrecognized) {
			t.Fatalf("%s = %v", name, err)
		}
	}
	large := []workitem.DraftSection{{Name: "scope", Content: strings.Repeat("y", bodyLimit), Authorship: provenance.UserAuthored}}
	if _, err := (Adapter{}).ReviseDocument(workitem.ProviderDocument{Title: "T", Body: original}, workitem.DocumentRevision{Sections: large}); !errors.Is(err, workitem.ErrDocumentTooLarge) {
		t.Fatalf("oversized = %v", err)
	}
}
